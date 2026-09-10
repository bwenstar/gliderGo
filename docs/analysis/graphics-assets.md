# Glider PRO 1.0.4 — Sprite and Background Art: Formats and Extraction Plan

## Scope

This document specifies, for a from-scratch Go re-implementation:

1. Exactly how Glider PRO's art is stored on disk (resource fork, `PICT`, `clut`, `CURS`, `crsr`, `cicn`, `ICON`, `ICN#`/`icl4`/`icl8`, `ics#`/`ics4`/`ics8`, `PAT#`, `acur`).
2. Which QuickDraw picture opcodes are **actually present** in the shipped resources — measured, not assumed — and therefore what a from-scratch PICT decoder must implement and what it may safely refuse.
3. The indexed 8-bit colour model, the exact palette, and the named colour indices the drawing code hard-codes.
4. Every transparency/masking mechanism in the game, with quantitative proof of mask polarity and of the colour key.
5. The complete sprite-sheet layout manifest: every source `Rect`, every sheet's PICT ID, every mask pairing — §6.
6. A concrete PICT → RGBA extraction pipeline, its output naming scheme, its manifest, a per-step risk ranking, and a measured full run that proves it — §7.

Out of scope: sound (`snd `), house file format (BinHex/QuickTime `Houses/`), physics, input, networking, the level editor's UI logic beyond its use of art.

Also deliberately **not** here, because they have their own documents and this one would only duplicate them:

| Topic | Document |
|---|---|
| Resource-fork container format, full resource census, the 152-row `PICT` table | `docs/analysis/resource-fork.md` (see §2 below) |
| Room background storage, the 8-tile room composition model, `DrawLocale` / `DrawRoomBackground` | `docs/analysis/rendering.md:1739` (§6) |
| The compositing pipeline (`backSrcMap` → `workSrcMap` → window), dirty-rect bookkeeping, `RenderFrame` order | `docs/analysis/rendering.md:2984` (§9) and `:3134` (§10) |
| The draw-time dispatcher view of `srcRects` (which helper is called for which `what`) | `docs/analysis/rendering.md:2230` (§7.4-§7.6) |
| Per-frame animated-element renderers (flames, pendulums, sparkles, stars, bands, toast, shreds) | `docs/analysis/rendering.md:3204` (§11) |

Every non-obvious claim is cited as `GliderPRO/Sources/File.c:NNN`, path relative to the repository root, line numbers taken from a CR→LF converted copy of the file (`tr '\r' '\n'`). The originals have CR-only line endings and are read-only; nothing under `GliderPRO/` was modified.

Binary-layout claims in this document were verified by parsing the actual bytes with python3 and are reported with observed values. Where a number is derived rather than observed, it says so.

## Sources read

Read in full (LF-converted copies):

| File | LF lines |
|---|---|
| `GliderPRO/Sources/RoomGraphics.c` | 462 |
| `GliderPRO/Sources/Render.c` | 772 |
| `GliderPRO/Sources/ColorUtils.c` | 224 |
| `GliderPRO/Sources/ObjectDraw.c` | 1406 |
| `GliderPRO/Sources/ObjectDraw2.c` | 1437 |
| `GliderPRO/Sources/ObjectDrawAll.c` | 966 |
| `GliderPRO/Sources/StructuresInit.c` | 724 |
| `GliderPRO/Sources/StructuresInit2.c` | 476 |
| `GliderPRO/Sources/Utilities.c` | 788 |
| `GliderPRO/Sources/Marquee.c` | 511 |
| `GliderPRO/Sources/Transitions.c` | 145 |
| `GliderPRO/Sources/Environ.c` | 734 |
| `GliderPRO/Sources/MainWindow.c` | 601 |
| `GliderPRO/Sources/AnimCursor.c` | 263 |
| `GliderPRO/Headers/GliderDefines.h` | 625 |

Read in the parts that bear on graphics:

| File | Lines read |
|---|---|
| `GliderPRO/Sources/DynamicMaps.c` | 1-200, 300-429 |
| `GliderPRO/Sources/Objects.c` | 1-90 |
| `GliderPRO/Sources/RoomInfo.c` | 28-50, 60-109, 395-439 |
| `GliderPRO/Sources/InterfaceInit.c` | 16-19, 82-106, 180-219 |
| `GliderPRO/Sources/Play.c` | 40-60, 115-136 |
| `GliderPRO/Sources/Player.c` | 1138-1166 |
| `GliderPRO/Headers/Externs.h` | 1-30, 180-200 |
| `GliderPRO/Sources/ObjectRects.c` | 50-80, 200-250 |
| `GliderPRO/Sources/ObjectEdit.c` | 25-35, 1300-1330, 2270-2350, 2520-2540 |
| `GliderPRO/Sources/ObjectAdd.c` | 35-50 |
| `GliderPRO/Sources/ObjectInfo.c` | 50-65, 2300-2350 |
| `GliderPRO/Sources/HouseIO.c` | 560-590 |
| `GliderPRO/Sources/House.c` | 240-255 |
| `GliderPRO/Sources/Room.c` | 35-105 |
| `GliderPRO/Sources/RoomInfo.c` | 395-430, 500-560, 750-790, 826-860 |
| `GliderPRO/Sources/Banner.c` | 15-25, 45-70, 215-230 |
| `GliderPRO/Sources/Input.c` | 15-20, 80-90 |
| `GliderPRO/Sources/GameOver.c` | 20-25, 80-95 |
| `GliderPRO/Sources/HighScores.c` | 60-70 |
| `GliderPRO/Sources/RubberBands.c` | 20-30 |
| `GliderPRO/Sources/Scoreboard.c` | 30-45 |
| `GliderPRO/Sources/Dynamics.c` | 15-25 |
| `GliderPRO/Sources/RectUtils.c` | 190-215 |
| `GliderPRO/Headers/Objects.h` | 1-43 (all) |

Binary/resource sources:

| Path | Size |
|---|---|
| `GliderPRO/Glider PRO.r` | 15,475,666 bytes, 199,843 LF lines (Rez text dump of the whole resource fork) |

Tools written for this analysis (in the repo, not under `GliderPRO/`):

| Path | Purpose |
|---|---|
| `tools/probe_pict.py` | From-scratch stdlib-only PICT v1/v2 parser, PackBits decoder, `clut` parser, PNG writer. 37,019 bytes. |
| `tools/probe_rez.py` | Rez-dump inventory/extractor (pre-existing; used to materialise 538 resources to `/tmp/wf-gfx/res/<TYPE>/<id>.bin`). |
| `tools/extract_art.py` | The §7 pipeline, stages 1-3: the four alpha rules, the atlas crop, the RGBA PNG writer, and the whole §6.2/§6.3 atlas embedded as machine-readable `SHEETS`/`ATLAS` literals. 452 lines, 29,622 bytes. |

---

## 1. The rendering stack at a glance

### 1.1 Mac Toolbox facilities used, and what Go must supply

Glider PRO is a pure QuickDraw game. It never touches a 3D API, never uses `CopyDeepMask`, never uses arithmetic transfer modes, and never uses `QDProcs` bottlenecks. The full list of graphics Toolbox calls used, and their Go replacements:

| Toolbox call | Where | Go replacement |
|---|---|---|
| `NewGWorld(&gw, depth, bounds, nil, nil, useTempMem)` | `GliderPRO/Sources/Utilities.c:270` | Allocate an indexed 8-bit surface: `struct{W,H,Stride int; Pix []uint8; Pal *[256]RGB}` |
| `NewGWorld(..., depth=1, ...)` (always via `CreateOffScreenGWorld`) | `GliderPRO/Sources/ObjectDraw2.c:1283` | 1-bit alpha mask: `[]uint8` bitmap, or expand to `[]bool`/`[]uint8` 0/255 |
| `LockPixels(GetGWorldPixMap(gw))` | `GliderPRO/Sources/Utilities.c:275` | no-op |
| `GetGWorld` / `SetGWorld` | throughout | explicit destination argument on every blit |
| `SetPort((GrafPtr)aGWorld)` | `GliderPRO/Sources/RoomGraphics.c:238` | explicit destination argument |
| `DisposeGWorld` | `GliderPRO/Sources/ObjectDraw2.c:1294-1295` | GC / pool |
| `CopyBits(src, dst, &srcR, &dstR, srcCopy, nil)` | ~60 sites | blit, no scaling needed when rects are congruent (they always are — see §3.12.4) |
| `CopyBits(..., transparent, nil)` | `GliderPRO/Sources/ObjectDraw2.c:1403`, `:1430` | colour-key blit, key = the port's **background** colour = white = palette index 0 |
| `CopyBits(..., srcXor, nil)` | `GliderPRO/Sources/Marquee.c:438` | XOR blit (editor only) |
| `CopyBits(..., srcCopy, aRgn)` | `GliderPRO/Sources/Transitions.c:120` | clip rect (the region is always the window's visRgn) |
| `CopyMask(src, mask, dst, &srcR, &maskR, &dstR)` | ~40 sites | 1-bit stencil blit; bit 1 = copy source |
| `DrawPicture(pic, &bounds)` | `GliderPRO/Sources/Utilities.c:330`, `:347` | decode PICT into the destination surface, scaling `picFrame` → `bounds` |
| `GetPicture(id)` / `ReleaseResource` | `GliderPRO/Sources/Utilities.c:322`, `:332` | look up a decoded image by ID in an asset map |
| `PaintRect(&r)` | `GliderPRO/Sources/RoomGraphics.c:76`, `:187`, `:203` | fill with the current pen pattern in the fore colour (always solid black here) |
| `Index2Color(i, &rgb)` | `GliderPRO/Sources/ColorUtils.c:25`, `:41`, `:57` | `pal[i]` |
| `RGBForeColor` / `GetForeColor` | `GliderPRO/Sources/ColorUtils.c` | current draw colour, saved/restored |
| `GetIndPattern(&pat, listID, i+1)` | `GliderPRO/Sources/Marquee.c:504` | 8-byte 8x8 1-bit pattern from `PAT#` 128 |
| `PenPat` / `PenMode(patXor)` / `PenNormal` | `GliderPRO/Sources/Marquee.c:41-49` | patterned XOR framing (editor marquee) |
| `GetCursor(id)` / `SetCursor` | `GliderPRO/Sources/InterfaceInit.c:82-106` | `CURS` 16x16 1-bit + mask + hotspot |
| `GetCCursor(id)` / `SetCCursor` | `GliderPRO/Sources/AnimCursor.c:87`, `:98` | `crsr` colour cursor |
| `GetCIcon` / `PlotCIcon` / `DisposeCIcon` | `GliderPRO/Sources/Utilities.c:398-404` | `cicn` |
| `GetIconSuite` / `PlotIconSuite` | `GliderPRO/Sources/Utilities.c:384-386` | `ICN#`/`icl4`/`icl8`/`ics#`/`ics4`/`ics8` family by ID |
| `NewRgn` / `RectRgn` / `UnionRgn` / `SetClip` / `GetClip` / `DisposeRgn` | `GliderPRO/Sources/Render.c:746-768`, `:153-183` | a clip region built from a union of rects — a `[]Rect` clip list suffices (only rects are ever unioned) |
| `GetPortVisibleRegion` | `GliderPRO/Sources/Transitions.c:120` | window bounds |
| `GetMainDevice` / `HasDepth` / `SetDepth` | `GliderPRO/Sources/Utilities.c:169` (`GetMainDevice`), `GliderPRO/Sources/Environ.c:265`, `:281`, `:297` (`HasDepth`), `:389` (`SetDepth`) | nothing — the Go port owns its framebuffer |
| `SwapMMUMode(true32b)` | `GliderPRO/Sources/MainWindow.c:530`, `:544` | nothing (it is inside commented-out code) |

### 1.2 The surface (GWorld) inventory

Persistent offscreen surfaces, all created by `CreateOffScreenGWorld` (`GliderPRO/Sources/Utilities.c:266`):

```c
OSErr CreateOffScreenGWorld (GWorldPtr *theGWorld, Rect *bounds, short depth)
{
    OSErr theErr;
    theErr = NewGWorld(theGWorld, depth, bounds, nil, nil, useTempMem);
    if (theErr)
        theErr = NewGWorld(theGWorld, depth, bounds, nil, nil, 0);
    LockPixels(GetGWorldPixMap(*theGWorld));
    return theErr;
}
```

Note it ignores the first error and retries out of the application heap; and it calls `LockPixels` on a possibly-nil pixmap if both allocations failed. A Go port should return the error.

| Surface | Depth | Bounds | Created at | Contents |
|---|---|---|---|---|
| `backSrcMap` | 8 | `backSrcRect` = `houseRect` zeroed | `GliderPRO/Sources/StructuresInit2.c:158` | the static room composite: background tiles + all non-animated objects |
| `workSrcMap` | 8 | `workSrcRect` = `houseRect` zeroed | `GliderPRO/Sources/StructuresInit2.c:154` | the per-frame composite (copy of back + moving sprites) |
| `boardSrcMap` | 8 | `houseRect` width x 20 | `GliderPRO/Sources/StructuresInit.c:59` | scoreboard strip from `PICT` 1997 |
| `badgeSrcMap` | 8 | 32 x 66 | `GliderPRO/Sources/StructuresInit.c:59` | `PICT` 1996 |
| `glidSrcMap` | 8 | 48 x 668 | `GliderPRO/Sources/StructuresInit.c:169` | player-1 glider sheet, `PICT` 3999 (or 3976 when foiled) |
| `glid2SrcMap` | 8 | 48 x 668 | `GliderPRO/Sources/StructuresInit.c:169` | player-2 glider sheet, `PICT` 3974 (or 3963 when foiled); in 1-player mode holds the foil sheet 3976 |
| `glidMaskMap` | **1** | 48 x 668 | `GliderPRO/Sources/StructuresInit.c:169` | `PICT` 4999, shared by all four glider sheets |
| `shadowSrcMap` / `shadowMaskMap` | 8 / 1 | 48 x 18 | `GliderPRO/Sources/StructuresInit.c:169` | `PICT` 3998 / 4998 |
| `bandsSrcMap` / `bandsMaskMap` | 8 / 1 | 16 x 18 | `GliderPRO/Sources/StructuresInit.c:169` | `PICT` 4007 / 5007 |
| `blowerSrcMap` / `blowerMaskMap` | 8 / 1 | 48 x 402 | `GliderPRO/Sources/StructuresInit.c:244` | `PICT` 4000 / 5000 |
| `furnitureSrcMap` / `furnitureMaskMap` | 8 / 1 | 64 x 278 | `GliderPRO/Sources/StructuresInit.c:293` | `PICT` 4001 / 5001 (art is only 221 rows tall — see §3.12.6) |
| `bonusSrcMap` / `bonusMaskMap` | 8 / 1 | 88 x 378 | `GliderPRO/Sources/StructuresInit.c:341` | `PICT` 4002 / 5002 |
| `pointsSrcMap` / `pointsMaskMap` | 8 / 1 | 24 x 120 | `GliderPRO/Sources/StructuresInit.c:341` | `PICT` 4006 / 5006 |
| `transSrcMap` / `transMaskMap` | 8 / 1 | 56 x 32 | `GliderPRO/Sources/StructuresInit.c:425` | `PICT` 4008 / 5008 |
| `switchSrcMap` | 8 | 32 x 104 | `GliderPRO/Sources/StructuresInit.c:447` | `PICT` 4003. **No mask surface exists.** |
| `lightSrcMap` / `lightMaskMap` | 8 / 1 | 72 x 126 | `GliderPRO/Sources/StructuresInit.c:492` | `PICT` 4004 / 5004 |
| `applianceSrcMap` / `applianceMaskMap` | 8 / 1 | 80 x 269 | `GliderPRO/Sources/StructuresInit.c:529` | `PICT` 4005 / 5005 (mask is only 268 rows — see §3.12.5) |
| `toastSrcMap` / `toastMaskMap` | 8 / 1 | 32 x 174 | `GliderPRO/Sources/StructuresInit.c:529` | `PICT` 4009 / 5009 |
| `shredSrcMap` / `shredMaskMap` | 8 / 1 | 40 x 35 | `GliderPRO/Sources/StructuresInit.c:529` | `PICT` 4010 / 5010 |
| `balloonSrcMap` / `balloonMaskMap` | 8 / 1 | 24 x 240 | `GliderPRO/Sources/StructuresInit.c:614` | `PICT` 4011 / 5011 |
| `copterSrcMap` / `copterMaskMap` | 8 / 1 | 32 x 300 | `GliderPRO/Sources/StructuresInit.c:614` | `PICT` 4012 / 5012 |
| `dartSrcMap` / `dartMaskMap` | 8 / 1 | 64 x 76 | `GliderPRO/Sources/StructuresInit.c:614` | `PICT` 4013 / 5013 |
| `ballSrcMap` / `ballMaskMap` | 8 / 1 | 32 x 64 | `GliderPRO/Sources/StructuresInit.c:614` | `PICT` 4014 / 5014 |
| `dripSrcMap` / `dripMaskMap` | 8 / 1 | 16 x 72 | `GliderPRO/Sources/StructuresInit.c:614` | `PICT` 4015 / 5015 |
| `enemySrcMap` / `enemyMaskMap` | 8 / 1 | 36 x 33 | `GliderPRO/Sources/StructuresInit.c:614` | `PICT` 4016 / 5016 |
| `fishSrcMap` / `fishMaskMap` | 8 / 1 | 16 x 128 | `GliderPRO/Sources/StructuresInit.c:614` | `PICT` 4017 / 5017 |
| `clutterSrcMap` / `clutterMaskMap` | 8 / 1 | 128 x 69 | `GliderPRO/Sources/StructuresInit2.c:55` | `PICT` 4018 / 5018 |
| `suppSrcMap` | 8 | 512 x 44 | `GliderPRO/Sources/StructuresInit2.c:98` | `PICT` 1999, floor/ceiling support strip. **No mask.** |
| `angelSrcMap` / `angelMaskMap` | 8 / 1 | 96 x 44 | `GliderPRO/Sources/StructuresInit2.c:119` | `PICT` 1019 / **1020** (+1, not +1000) |
| `tileSrcMap` | 8 | 128 x 80 | `GliderPRO/Sources/RoomInfo.c:407` | editor-only room thumbnail; `nil` until the Room Info dialog opens (`GliderPRO/Sources/StructuresInit2.c:179`) |
| `savedMaps[0..23].map` | 8 | varies | `GliderPRO/Sources/DynamicMaps.c:82` | per-animation background backups / pre-composited filmstrips |
| `toolSrcMap`, `nailSrcMap` | 8 | varies | set `nil` at `GliderPRO/Sources/InterfaceInit.c:191-192` | editor tool palette art (`PICT` 1011 / 1010) |

Throwaway surfaces created and destroyed inside a single draw call: `GliderPRO/Sources/ObjectDraw2.c:173`+`:177` (`DrawMailboxLeft`), `:256`+`:260` (`DrawMailboxRight`), `:654`+`:658` (`DrawTV`), `:748`+`:752` (`DrawVCR`), `:803`+`:807` (`DrawStereo`), `:858`+`:862` (`DrawMicrowave`), `:1279`+`:1283` (`DrawPictWithMaskObject`), `:1397` (`DrawPictSansWhiteObject`, 8-bit only), `:1424` (`DrawCustPictSansWhite`, 8-bit only). Each pair is one `kPreferredDepth` surface plus one 1-bit mask surface; the two colour-key helpers allocate no mask.

### 1.3 Depth model

```c
#define kPreferredDepth     8
```
`GliderPRO/Headers/Externs.h:15`. Every 8-bit GWorld in the table above is created with this constant; every mask GWorld is created with a literal `1`.

`thisMac.isDepth` holds the *screen* depth, read back from the device pixmap:

```c
short WhatsOurDepth (void)      // GliderPRO/Sources/Environ.c:232
{
    if (thisMac.hasColor)
        thisDepth = (**(**thisGDevice).gdPMap).pixelSize;
    else
        thisDepth = 1;
    return thisDepth;
}
```

`HandleDepthSwitching` (`GliderPRO/Sources/Environ.c:505`) dispatches on the user preference:

| Constant | Value | Definition | Effect |
|---|---|---|---|
| `kSwitchIfNeeded` | 0 | `GliderPRO/Headers/GliderDefines.h:43` | switch to 8-bit only if the screen is not already 8-bit |
| `kSwitchTo256Colors` | 1 | `GliderPRO/Headers/GliderDefines.h:44` | `SwitchToDepth(8, true)` |
| `kSwitchTo16Grays` | 2 | `GliderPRO/Headers/GliderDefines.h:45` | `SwitchToDepth(4, false)` |

`SwitchDepthOrAbort` (`GliderPRO/Sources/Environ.c:404`) puts up `ALRT` 130 (`#define kSwitchDepthAlert 130`, `GliderPRO/Sources/Environ.c:18`) with three buttons: 1 → `SwitchToDepth(8, true)`, 2 → `SwitchToDepth(4, false)`, 3 → `ExitToShell()`.

So there are exactly two supported runtime colour modes: **8-bit indexed colour** (the normal case, and the one all shipped art is authored for) and **4-bit, 16 greys**. In 4-bit mode the offscreen GWorlds are *still 8 bits deep* — `kPreferredDepth` is a compile-time 8 — but QuickDraw dithers/maps on the final `CopyBits` to the screen, and the game additionally:

* selects a monochrome text branch: `if (thisMac.isDepth == 4)` at `GliderPRO/Sources/MainWindow.c:68`;
* substitutes small integer colour indices for the 8-bit named ones, e.g. `GliderPRO/Sources/ObjectDraw2.c:1046-1051` uses `brownC = 11; tanC = 9; dkstRedC = 15;` instead of `k8BrownColor`/`k8TanColor`/`k8DkRed2Color`;
* forces even x-coordinates on saved-map backups so that a 4-bit (2 pixels/byte) `CopyBits` stays byte-aligned — `GliderPRO/Sources/DynamicMaps.c:326`, `:409`, `:495`, `:584`, `:671`. The idiom is:

```c
if ((thisMac.isDepth == 4) && ((src.left % 2) == 1))
{
    QOffsetRect(&src, -1, 0);
    if (src.left < 0)
        QOffsetRect(&src, 2, 0);
}
```

A Go port that renders 8-bit indexed into a 32-bit framebuffer does not need the 4-bit mode at all, and does not need the alignment fixes. It should be aware that the fixes shift some flame/star animations by one pixel on real hardware in 16-grey mode.

`CheckMemorySize` (`GliderPRO/Sources/Environ.c:562`) computes the memory budget from the same numbers, and is a useful cross-check on the sheet sizes:

```c
#define kBaseBytesNeeded    614400L   // GliderPRO/Sources/Environ.c:564
#define kPaddingBytes       204800L   // GliderPRO/Sources/Environ.c:565
```
work and back maps each cost `((houseRect.right - houseRect.left) * (houseRect.bottom + 1 - houseRect.top) * thisMac.isDepth) / 8`; the scoreboard costs `(right - left) * 21 * isDepth / 8` plus 6396. The per-sheet byte constants that follow (`GliderPRO/Sources/Environ.c:594-641`) independently confirm the sheet dimensions:

| Sheet | Colour bytes | Mask bytes | Implied 8-bit size | Matches |
|---|---|---|---|---|
| glider | 32112 | 32064 (÷8 = 4008) | 48x668 → 48x669 rows padded | 3999/4999 |
| glider2 | 32112 | — | | 3974 |
| shadow | 912 | 864 | 48x18/19 | 3998/4998 |
| bands | 304 | 288 | 16x18/19 | 4007/5007 |
| blower | 19344 | 19344 | 48x402/403 | 4000/5000 |
| furniture | 17856 | 17792 | 64x278/279 | 4001/5001 |
| prizes | 33264 | 33176 | 88x378 | 4002/5002 |
| points | 2904 | 2880 | 24x120/121 | 4006/5006 |
| transport | 1848 | 1792 | 56x32/33 | 4008/5008 |
| switches | 3360 | **absent** | 32x104/105 | 4003, no mask |
| lights | 9144 | 9072 | 72x126/127 | 4004/5004 |
| appliances | 21600 | 21520 | 80x269/270 | 4005/5005 |
| toast | 5600 | 5568 | 32x174/175 | 4009/5009 |
| shredded | 1440 | 1400 | 40x35/36 | 4010/5010 |
| balloon | 5784 | 5760 | 24x240/241 | 4011/5011 |
| copter | 9632 | 9600 | 32x300/301 | 4012/5012 |
| dart | 4928 | 4864 | 64x76/77 | 4013/5013 |
| ball | 2080 | 2048 | 32x64/65 | 4014/5014 |
| drip | 1168 | 1152 | 16x72/73 | 4015/5015 |
| enemy | 1224 | 1188 | 36x33/34 | 4016/5016 |
| fish | 2064 | 2048 | 16x128/129 | 4017/5017 |
| clutter | 8960 | 8832 | 128x69/70 | 4018/5018 |
| support | 23040 | **absent** | 512x44/45 | 1999, no mask |
| angel | 4320 | 4224 | 96x44/45 | 1019/1020 |

The "switches" and "support" lines having no mask entry is independent confirmation that `PICT` 5003 does not exist and that the floor support is drawn opaquely.

---

## 2. The resource fork as shipped — specified in `resource-fork.md`

This section previously restated the Rez grammar, the 35-type resource census and the 152-row `PICT`
table. All three are the subject of a dedicated document in this directory and are **not** duplicated
here. Read them there:

| What you want | Where it is |
|---|---|
| Rez (`DeRez`) text grammar and the round-trip verifier | `docs/analysis/resource-fork.md` Part 1 |
| Full census: 538 resources, 35 types, 3,168,309 bytes, with per-type ID ranges | `docs/analysis/resource-fork.md:330` (§2.1 Master table) |
| Ten largest resources; purgeable/preload census; resource names; set relationships | `docs/analysis/resource-fork.md:377`, `:392`, `:418`, `:450` |
| Negative results — resource types that are *absent* | `docs/analysis/resource-fork.md:470` (§2.6) |
| `PICT` functional groups with byte totals (10 groups, sums to 152 / 1,919,598) | `docs/analysis/resource-fork.md:493` (§3.1.1) |
| **The complete 152-row `PICT` table** — version, `picFrame`, opcode, depth, length, role | `docs/analysis/resource-fork.md:949` (§3.1.11) |
| Per-group `PICT` detail: about box, dialog banners, Room-Info thumbnails, banner/scoreboard, room backgrounds, room-object art, sprite sheets, masks, `PICT` 10000 | `docs/analysis/resource-fork.md:572`-`:947` (§3.1.3-§3.1.10) |
| Which code loads which `PICT`, and by what call | `docs/analysis/resource-fork.md:1109` (§3.1.12) |

Five facts from that document are load-bearing for the rest of *this* one, so they are restated once
here and nowhere else:

1. **152 `PICT` resources, 1,919,598 bytes.** IDs: 150-151, 153, 1000-1023, 1202, 1211, 1216-1217, 1988-2017, 3903-3904, 3912-3915, 3921, 3927, 3957-4018, 4998-5002, 5004-5018, 10000 (`docs/analysis/resource-fork.md:330`).
2. **There is no `PICT` 5003.** The single gap in the 4998-5018 mask run is the switch sheet's mask, which was never authored. See §5.2 and §6.2.
3. **`picSize` (the int16 at +0) is a truncated 16-bit length** and is therefore wrong for the 9 pictures over 64 KB; a decoder must use the resource's real length (`docs/analysis/resource-fork.md:511`).
4. **`picFrame` origin is (0,0) for 150 of the 152.** The two exceptions are masks 5006 (origin 91,0) and 5010 (origin 0,195); `LoadGraphic` normalises the origin away with `OffsetRect(&bounds, -bounds.left, -bounds.top)` (`GliderPRO/Sources/Utilities.c:327-330`), so a decoder must too.
5. **GWorld height is not derivable from `PICT` height.** The hard-coded `QSetRect` in `StructuresInit*.c` is the authority; `PICT` 4001 is 64x221 inside a 64x278 GWorld, and mask 5005 is 80x268 inside an 80x269 GWorld.

---

## 3. PICT, as actually used by Glider PRO

Everything in this section was measured by parsing all 152 extracted `PICT` resources with `tools/probe_pict.py`. All multi-byte fields are **big-endian**. All coordinates are QuickDraw `Rect` = `{int16 top; int16 left; int16 bottom; int16 right}` — note the **top,left,bottom,right** order, not left,top,right,bottom.

### 3.1-3.2 Resource header and version detection — specified in `resource-fork.md` §3.1.2

The 10-byte header (`int16 picSize`; `Rect picFrame`) and the v1/v2 discrimination, together with
the census of bytes 10..13 across all 152 resources (`0011 02FF` x116, `1101 0100` x4,
`1101 A000` x32), are specified at `docs/analysis/resource-fork.md:511` (§3.1.2 "Verified header
facts"). Not repeated here. The four consequences the rest of §3 depends on:

1. **The opcode stream starts at +10.** `sizeof(Picture)` header = 10 bytes, matching the `(*thePicture)->picFrame` dereference at `GliderPRO/Sources/Utilities.c:327`.
2. **Dispatch on the first *byte* at +10, not the first word.** v2 reads `00 11 02 FF`; v1 reads `11 01`, which as a big-endian 16-bit word is `0x1101` and would be mistaken for a `Clip`-adjacent opcode. Measured: **116 v2 and 36 v1** pictures in the file. Note that "v2" here means only the `VersionOp` payload `02 FF`; the *`HeaderOp`* version is a separate field and both of its values occur — see consequence 4.
3. **v1 opcodes are 1 byte; v2 opcodes are 2 bytes and must start on an even offset** relative to the start of the picture data. `probe_pict.py`'s `Reader.align()` enforces this before every v2 opcode read. Without alignment the stream survives `ShortComment` (2 bytes of data, 4 total) but desynchronises on odd-length `LongComment` or pixel payloads.
4. **The v2 `HeaderOp` payload may be skipped wholesale — but *both* header versions occur.** Every one of the 116 v2 pictures carries `0x0C00` immediately after the `VersionOp`, with a **24-byte** payload in both forms. The first field is `int16 version`, and it selects the layout:
   * `version == -1` (**standard v2**): `int16 version`, `int16 reserved`, `Fixed bounds.left`, `Fixed bounds.top`, `Fixed bounds.right`, `Fixed bounds.bottom`, `int32 reserved` — a 16-byte `Fixed`-precision bounding box. **95 pictures.**
   * `version == -2` (**extended v2**): `int16 version`, `int16 reserved`, `Fixed hRes`, `Fixed vRes`, `Rect srcRect`, `int32 reserved`. **21 pictures**, exactly: 1000, 1001, 1002, 1004, 1006, 1007, 1008, 1010, 1012, 1013, 1014, 1994, 1995, 1997, 2002, 2016, 2017, 3975, 3976, 3997, 3999.

   Both are 24 bytes, so a decoder that consumes 24 and moves on is correct for either. Nothing in Glider PRO reads any of it, and `DrawPicture` takes its scale from the destination rect, so the 24 bytes are inert. **A decoder must not reject `version == -2`**: that set includes the glider sheet 3999, the splash screen 1000, the scoreboard 1997 and three room backgrounds (2002, 2016, 2017). The extended set is also exactly the set of pictures whose embedded `ColorTable`s have `ctFlags == 0x0000` (§7.5), which suggests they were exported by a different tool or tool version than the other 95.

### 3.3 Measured opcode histogram

Across all 152 pictures, **only nine distinct opcodes appear**:

| Opcode | Name | Occurrences | Meaning |
|---|---|---|---|
| 0x0001 | `Clip` | 152 | clipping region; **always** rgnSize 10 (a bare rect) |
| 0x0011 | `VersionOp` | 152 | (v1 form is the byte pair `11 01`) |
| 0x001E | `DefHilite` | 95 | reset highlight colour; no data. Appears exactly once in each of the 95 `version == -1` pictures and in none of the 21 extended-v2 ones or the 36 v1 ones |
| 0x0090 | `BitsRect` | 16 (+1 special) | uncompressed copy-bits with rect source |
| 0x0098 | `PackBitsRect` | 185 | PackBits-compressed copy-bits with rect source |
| 0x0099 | `PackBitsRgn` | 3 | PackBits-compressed copy-bits with region mask |
| 0x00A0 | `ShortComment` | 269 | 2-byte kind |
| 0x00FF | `OpEndPic` | 152 | end of picture |
| 0x0C00 | `HeaderOp` | 116 | v2 header |

`ShortComment` kinds observed: `130` (picDwgBeg) x129, `131` (picDwgEnd) x129, `142` (picBitBeg) x4, `143` (picBitEnd) x4, and `12345` x3 (application-private, only in `PICT` 3968, 3969, 3995 — the three region-masked ones). 129 + 129 + 4 + 4 + 3 = 269, matching the histogram row above. The `142`/`143` pairs occur in `PICT` 150, 151, 1009 and 4999.

**Not present anywhere**: `DirectBitsRect` (0x009A), `DirectBitsRgn` (0x009B), `BitsRgn` (0x0091), `PackBitsRgn` in v1 form, any of the line/rect/oval/arc/poly/region drawing opcodes (0x20-0x8F), any text opcode (0x28-0x2F), `PnPixPat`/`FillPixPat` (0x0012-0x0014), `LongComment` (0x00A1), `CompressedQuickTime` (0x8200), `UncompressedQuickTime` (0x8201), or any opcode >= 0x0100 other than 0x0C00. (The extended-v2 `HeaderOp` form **is** present — 21 pictures — see §3.2 consequence 4.)

Consequence: **a decoder for Glider PRO's art needs to implement exactly seven opcodes with semantics** (`VersionOp`, `HeaderOp`, `Clip`, `BitsRect`, `PackBitsRect`, `PackBitsRgn`, `OpEndPic`) plus skip-by-length for `DefHilite` and `ShortComment` — nine in total. Everything else can be a hard error. That is the single most important simplification in this document, and §7.2/§7.3 turn it into a runnable allow-list.

Nevertheless the length table below is worth implementing in full, because it lets the decoder *skip* an unexpected opcode instead of dying, and because user-supplied house `PICT`s (`kUserBackground 3000`, `GliderPRO/Headers/GliderDefines.h:522`) can contain anything.

### 3.4 Opcode data-length table

`probe_pict.py`'s `op_datalen(op)` returns either a fixed byte count or a marker meaning "length is data-dependent". The table, with the markers used:

| Opcodes | Data length |
|---|---|
| 0x0000 `NOP`, 0x001C `HiliteMode`, 0x001E `DefHilite`, 0x0038-0x003F, 0x0048-0x004F, 0x0058-0x005F, 0x0068-0x006F, 0x0078-0x007F, 0x0088-0x008F, 0x00FF `OpEndPic` | 0 |
| 0x0003 `TxFont`, 0x0004 `TxFace` (1 + 1 pad), 0x0005 `TxMode`, 0x0008 `PnMode`, 0x000D `TxSize`, 0x0011 `VersionOp` (v2: 2), 0x0015 `PnLocHFrac`, 0x0016 `ChExtra`, 0x0023 `ShortLineFrom` (2), 0x002C-0x002E | 2 |
| 0x0006 `SpExtra`, 0x0007 `PnSize`(4), 0x000E `FgColor`, 0x000F `BkColor`, 0x0021 `LineFrom`, 0x0022 `ShortLine`(6→see below) | 4 |
| 0x0009 `PnPat`, 0x000A `FillPat`, 0x000B `OvSize`(4), 0x000C `Origin`(4), 0x0010 `TxRatio`(8), 0x001A `RGBFgCol`(6), 0x001B `RGBBkCol`(6), 0x001D `HiliteColor`(6), 0x001F `OpColor`(6) | 8 / 4 / 6 as noted |
| 0x0020 `Line` | 8 |
| 0x0022 `ShortLine` | 6 |
| 0x0030-0x0037 rect ops, 0x0040-0x0047 rounded-rect, 0x0050-0x0057 oval | 8 |
| 0x0060-0x0067 arc, 0x0068+ | 12 |
| 0x0070-0x0077 poly | `VAR_POLY`: read `uint16 polySize` at the start, total = `polySize` |
| 0x0080-0x0087 region | `VAR_REGION`: read `uint16 rgnSize`, total = `rgnSize` |
| 0x0012 `BkPixPat`, 0x0013 `PnPixPat`, 0x0014 `FillPixPat` | `VAR_PIXPAT` (pattern type + PixMap + ColorTable + PixData) |
| 0x0028 `LongText` | `VAR_TEXT` (Point + `uint8 len` + bytes) |
| 0x0029 `DHText` | `VAR_DHTEXT` (`uint8 dh` + `uint8 len` + bytes) |
| 0x002A `DVText` | `VAR_DVTEXT` |
| 0x002B `DHDVText` | `VAR_DHDVTEXT` |
| 0x0090 `BitsRect`, 0x0098 `PackBitsRect` | `VAR_BITS`, no region |
| 0x0091 `BitsRgn`, 0x0099 `PackBitsRgn` | `VAR_BITS`, with region |
| 0x009A `DirectBitsRect`, 0x009B `DirectBitsRgn` | `VAR_BITS` with 4-byte pixel size / direct PixMap |
| 0x00A0 `ShortComment` | 2 |
| 0x00A1 `LongComment` | `VAR_LONGCOMMENT`: `uint16 kind` + `uint16 size` + `size` bytes |
| 0x00A2-0x00AF, 0x00B0-0x00CF | 0 (reserved, no data) |
| 0x0100-0x01FF | 2 |
| 0x0200-0x02FF | 4 |
| 0x0300-0x0BFF | 8 |
| 0x0C00 `HeaderOp` | 24 |
| 0x0C01-0x7FFF | 24 (0x0C00 block), then 254 for 0x0D00-0x0DFF etc. |
| 0x7F00-0x7FFF | 254 |
| 0x8000-0x80FF | 0 |
| 0x8100-0xFFFF | `VAR_LEN32`: `uint32 size` + `size` bytes |

None of the `VAR_*` markers other than `VAR_BITS` and `VAR_REGION` is ever exercised by the shipped art. They are implemented in the prototype defensively.

### 3.5 `BitsRect` / `PackBitsRect` / `PackBitsRgn` payload layout

This is the only opcode family that matters. Read order, exactly as `probe_pict.py::Picture._read_bits` does it:

```
1.  rawRowBytes := u16()                         # peeked, not consumed by the reader below
2.  isPixMap   := (rawRowBytes & 0x8000) != 0
3.  if isPixMap:
4.      read the 50-byte PixMap record  (baseAddr present? NO for BitsRect family)
5.      read the ColorTable
6.  else:
7.      read the 14-byte old-style BitMap record (rowBytes:2 + bounds:8 -> 10 bytes here,
                                                  baseAddr omitted)
8.      colorTable := nil   (implicitly 1-bit: 0 = white/fg?, see below)
9.  srcRect := Rect()      # 8 bytes, in PixMap/BitMap bounds coordinates
10. dstRect := Rect()      # 8 bytes, in picFrame coordinates
11. mode    := u16()       # transfer mode
12. if opcode is 0x0091 or 0x0099:  maskRgn := Region()   # variable length
13. pixData := rows, one per bounds row (see 3.8/3.9)
```

Important detail that trips up naive decoders: for `BitsRect`/`PackBitsRect`/`BitsRgn`/`PackBitsRgn` the **`baseAddr` field is omitted** from the on-disk PixMap. The record therefore starts at `rowBytes`. (For `DirectBitsRect`/`DirectBitsRgn` a 4-byte pseudo-baseAddr *is* present — irrelevant here since those opcodes never occur.)

`probe_pict.py` handles this via `read_pixmap(r, has_baseaddr=False)`.

For the **old-style BitMap** branch (bit 15 of `rowBytes` clear), the on-disk record is 10 bytes here: `int16 rowBytes` + `Rect bounds`, and there is no colour table — the data is 1 bit per pixel and QuickDraw maps `0` → the port's **foreground** colour and `1` → background, or vice versa depending on mode. Measured: 37 of the 38 1-bit raster ops use this old-style BitMap form. Since every one of them is used as a `CopyMask` stencil or as a 1-bit `CopyBits` source into a fresh GWorld, the practical rule established in §5.2 is what matters: **bit value 1 means "opaque / copy the source pixel"**.

### 3.6 The on-disk PixMap record (50 bytes)

Verified against `crsr` and `cicn` resources by computing total resource size from the parsed fields and comparing with the actual file length: exact match for all 12 `crsr` and all 44 `cicn`.

| Offset | Size | Field | Type | Notes |
|---|---|---|---|---|
| +0 | 4 | `baseAddr` | `Ptr` | **omitted** in the BitsRect family; present in `crsr`/`cicn`/`DirectBits` |
| +4 | 2 | `rowBytes` | `int16` | bit 15 (`0x8000`) set ⇒ this is a PixMap, not a BitMap. Low 15 bits = bytes per row. |
| +6 | 8 | `bounds` | `Rect` | top, left, bottom, right |
| +14 | 2 | `pmVersion` | `int16` | always 0 in this file |
| +16 | 2 | `packType` | `int16` | 0 = default for depth, 1 = unpacked, 2 = drop pad byte (32-bit), 3 = PackBits by words, 4 = PackBits by bytes with plane separation |
| +18 | 4 | `packSize` | `int32` | always 0 in this file |
| +22 | 4 | `hRes` | `Fixed` | always 0x00480000 = 72.0 |
| +26 | 4 | `vRes` | `Fixed` | always 0x00480000 = 72.0 |
| +30 | 2 | `pixelType` | `int16` | 0 = chunky indexed. **Always 0 here.** |
| +32 | 2 | `pixelSize` | `int16` | bits per pixel. Measured values: 1 and 8 only. |
| +34 | 2 | `cmpCount` | `int16` | components per pixel. **Always 1 here.** |
| +36 | 2 | `cmpSize` | `int16` | bits per component. 1 or 8, equal to `pixelSize`. |
| +38 | 4 | `planeBytes` | `int32` | always 0 |
| +42 | 4 | `pmTable` | `CTabHandle` | on disk, a placeholder/offset; ignored by the decoder |
| +46 | 4 | `pmReserved` | `int32` | always 0 |

Offsets +0..+46 give 50 bytes total. When `baseAddr` is omitted the record read is 46 bytes and all subsequent offsets shift down by 4.

### 3.7 The on-disk ColorTable

| Offset | Size | Field | Notes |
|---|---|---|---|
| +0 | 4 | `ctSeed` | identity; see below |
| +4 | 2 | `ctFlags` | bit 15 (`0x8000`) set ⇒ "device" table: the *position* in the array is the pixel index and the `value` field is meaningless. Clear ⇒ use `value` as the index. |
| +6 | 2 | `ctSize` | **number of entries minus one** |
| +8 | 8 x (`ctSize`+1) | `ctTable` | array of `ColorSpec` |

`ColorSpec` is 8 bytes: `uint16 value; uint16 red; uint16 green; uint16 blue`. Channels are 16-bit; to get 8-bit, take the high byte (in this file the low byte always equals the high byte, so `>> 8` and `& 0xFF` agree).

Total table length = `8 + 8 * (ctSize + 1)`.

### 3.8 PackBits

Two variants. Both are byte-oriented control bytes; the difference is the unit that gets repeated.

```
unpackbits_bytes(src, out_len):
 1. while len(out) < out_len:
 2.     n := int8(src[p]); p += 1
 3.     if n >= 0:                       # 0..127
 4.         count := n + 1
 5.         out += src[p : p+count]; p += count
 6.     else if n != -128:               # -1..-127
 7.         count := 257 - (n & 0xFF)    # == 1 - n
 8.         out += src[p] * count; p += 1
 9.     else:                            # n == -128 (0x80)
10.         pass                         # no-op, consume nothing more
```

`unpackbits_words` is identical except that a literal run copies `count` *words* (2 bytes each) and a repeat run repeats a single 2-byte word `count` times. Which variant applies:

* `pixelSize <= 8` → **byte** variant.
* `pixelSize == 16` → **word** variant.
* `pixelSize == 32` → byte variant, but the row is stored as `cmpCount` separate planes of `width` bytes each (`packType == 4`), which must be de-interleaved.

Glider PRO only ever needs the byte variant (`pixelSize` is 1 or 8 everywhere), but the prototype implements all three because user house `PICT`s might not be so tidy.

### 3.9 Scanline framing and the unpacked-row rules

Each row of packed pixel data is preceded by a byte count:

* if `rowBytes > 250` → the count is a **uint16**;
* else → the count is a **uint8**.

Measured `rowBytes` values in this file span 2..1536, so both branches are exercised. (`PICT` 1997 has `rowBytes` 1536; `PICT` 5007 has `rowBytes` 2.)

Three conditions cause the pixel data to be stored **unpacked** (no per-row count, `rowBytes` raw bytes per row):

1. the opcode is `BitsRect`/`BitsRgn` (0x0090/0x0091) — by definition;
2. `rowBytes < 8` — the classic QuickDraw rule; packing tiny rows is counter-productive so QuickDraw does not do it, even under a `PackBits*` opcode;
3. `packType == 1` — explicit "unpacked".

`probe_pict.py::Picture._read_rows` implements exactly:

```python
if (not packed) or rb < 8 or hdr.packType == 1:
    # raw rows
else:
    # per-row count + PackBits
```

Measured evidence that rule 2 is needed and is consistent: all 16 `BitsRect` (0x0090) raster ops have `rowBytes` in 2..6 (all < 8), and all 21 1-bit `PackBitsRect` (0x0098) raster ops have `rowBytes` in 8..42. So for this particular file the two rules never disagree; a decoder that applies rule 2 is correct either way. Rule 3 is exercised exactly once, by `PICT` 1003 (`packType == 1`, `rowBytes == 4`) — where rule 2 also fires. **So on this file, rules 1 and 2 alone suffice; rule 3 is belt-and-braces.**

### 3.10 The on-disk Region

Needed only to *skip* correctly (see §3.12.3), but the prototype decodes it:

| Offset | Size | Field |
|---|---|---|
| +0 | 2 | `rgnSize` — total bytes **including** this field |
| +2 | 8 | `rgnBBox` — `Rect` |
| +10 | `rgnSize - 10` | inversion-point data |

If `rgnSize == 10` the region is exactly its bounding rect. The inversion data is a sequence of `int16 y`, then a run of `int16 x` values, terminated by `0x7FFF`; the whole list is terminated by another `0x7FFF`. Toggling at each `x` on each `y` scanline reconstructs the mask.

### 3.11 Measured raster-op parameter uniformity

206 raster ops across 152 pictures (the inventory file has one header line plus 205 data rows). Grouped by (opcode, pixelSize, packType, ctab presence, ctFlags):

| Count | Opcode | pixelSize | packType | ColorTable | ctFlags |
|---|---|---|---|---|---|
| 143 | 0x0098 | 8 | 0 | ctSize 255 | 0x8000 |
| 21 | 0x0098 | 1 | 0 | none (old BitMap) | — |
| 21 | 0x0098 | 8 | 0 | ctSize 255 | 0x0000 |
| 16 | 0x0090 | 1 | 0 | none (old BitMap) | — |
| 3 | 0x0099 | 8 | 0 | ctSize 255 | 0x8000 |
| 1 | 0x0090 | 1 | **1** | **ctSize 1** | **0x0001** |

Totals: 167 ops at 8 bpp, 38 at 1 bpp, 205. The single outlier is `PICT` 1003 — the only 1-bit resource stored as a real *PixMap* (with a 2-entry colour table) rather than an old-style BitMap.

Invariant across **all** 205: `pixelType == 0`, `cmpCount == 1`, `pmVersion == 0`, `packSize == 0`, `planeBytes == 0`, `pmReserved == 0`, `hRes == vRes == 0x00480000` (72 dpi), `mode == 0` (`srcCopy`). For the 8-bpp ops, `cmpSize == 8`.

**Every transfer mode in every raster op is `srcCopy` (0).** Transparency is never encoded inside a PICT; it is always applied by the caller (§5).

### 3.12 Anomalies a correct extractor must handle

#### 3.12.1 `picSize` wraps

`picSize` is a signed int16, so any picture over 32,767 bytes stores garbage. Measured wrapped cases (19 of them):

| ID | Actual length | Stored `picSize` |
|---|---|---|
| 1000 | 108,482 | -22,590 |
| 1011 | 48,728 | -16,808 |
| 1021 | 51,520 | -14,016 |
| 1993 | 43,936 | -21,600 |
| 2001 | 89,338 | 23,802 |
| 2002 | 52,328 | -13,208 |
| 2003 | 43,158 | -22,378 |
| 2004 | 83,276 | 17,740 |
| 2005 | 70,572 | 5,036 |
| 2006 | 105,140 | -25,932 |
| 2007 | 61,926 | -3,610 |
| 2008 | 82,922 | 17,386 |
| 2009 | 34,856 | -30,680 |
| 2010 | 43,020 | -22,516 |
| 2011 | 81,036 | 15,500 |
| 2012 | 37,790 | -27,746 |
| 2013 | 32,966 | -32,570 |
| 2014 | 75,132 | 9,596 |
| 2016 | 76,176 | 10,640 |

Note `2013` is only 199 bytes over the limit and wraps to -32,570; `2015` at 23,748 does not wrap. **Use the resource length as the authoritative end, and stop at `OpEndPic`.** All 152 pictures had exactly zero trailing bytes after `OpEndPic` when parsed this way.

#### 3.12.2 Multiple raster ops per picture ("banding")

Large pictures are split into several `PackBitsRect`/`BitsRect` ops with **disjoint** `bounds`/`srcRect`/`dstRect`. A decoder that returns "the first bitmap" produces a truncated image.

| ID | Ops | Band layout |
|---|---|---|
| 1011 | 2 | rows 0-141, 142-215 |
| 1021 | 6 | 460 rows split six ways |
| 1993 | 2 | |
| 2000, 2001, 2003-2015 | 4 | rows 0-99, 100-199, 200-299, 300-321 (100/100/100/22) |
| 2002, 2016, 2017 | 1 | single op |
| 4999 | 2 | rows 0-511, 512-667 |
| all others | 1 | |

The 512-row split in `PICT` 4999 and the 100-row banding in the backgrounds strongly suggest the authoring tool had a per-op byte or row ceiling. Whatever the cause, the extractor must **composite all raster ops into one canvas** keyed on `dstRect`.

#### 3.12.3 Region-masked ops are a no-op

Three pictures use `PackBitsRgn` (0x0099): `PICT` 3968 (`kVase2`, `rgnSize` 44), 3969 (`kVase1`, `rgnSize` 52), 3995 (`kFilingCabinet`, `rgnSize` 56). All three regions were decoded and rasterised: **zero pixels inside `picFrame` are excluded by any of them.** They are effectively rectangular covers.

The parser must still consume the region bytes, or the stream desynchronises.

Note that 3968, 3969 and 3995 are exactly the three pictures carrying the private `ShortComment` kind `12345`. Likely the same authoring tool wrote all three.

#### 3.12.4 `picFrame` origin is not always (0,0)

Two masks have a shifted frame:

| ID | `picFrame` (t,l,b,r) | PixMap `bounds` |
|---|---|---|
| 5006 | (0, 91, 120, 115) | (0, 88, 120, 120) |
| 5010 | (195, 0, 230, 40) | (195, 0, 230, 40) |

The game is immune because `LoadGraphic` normalises:

```c
HLock((Handle)thePicture);
bounds = (*thePicture)->picFrame;
HUnlock((Handle)thePicture);
OffsetRect(&bounds, -bounds.left, -bounds.top);
DrawPicture(thePicture, &bounds);
```
`GliderPRO/Sources/Utilities.c:326-330`.

`DrawPicture(pic, dstRect)` maps `picFrame` onto `dstRect`. Since `dstRect` here is `picFrame` translated to the origin, the transform is a pure translation of `(-picFrame.left, -picFrame.top)`. **The extractor must apply the identical translation**, i.e. output canvas = `picFrame` size, and each raster op's `dstRect` is offset by `(-picFrame.left, -picFrame.top)` before writing.

`LoadScaledGraphic` does *not* normalise — it maps `picFrame` straight onto the caller's rect, scaling (`GliderPRO/Sources/Utilities.c:340`). It is used for the editor thumbnails (`GliderPRO/Sources/RoomInfo.c:417`, `:420`) and for `kManholeThruFloor` 3957 (`GliderPRO/Sources/RoomGraphics.c:281`). Those are the only two scaling sites in the whole game.

#### 3.12.5 `bounds` can be wider than `picFrame`

QuickDraw rounds PixMap widths up (usually to a multiple of 16 pixels for 1-bit data). Measured discrepancies:

| ID | `picFrame` width | max `bounds` width |
|---|---|---|
| 3903 | 94 | 96 |
| 3904 | 94 | 96 |
| 3912 | 92 | 96 |
| 3915 | 92 | 96 |
| 3927 | 54 | 56 |
| 5006 | 24 | 32 |
| 5016 | 36 | 40 |
| 1991 | 330 | 336 |
| 1998 | 332 | 336 |

**Output size must come from `picFrame`, never from `bounds`.** The extra columns are padding and must be cropped. `srcRect`/`dstRect` are the authority for which source pixels land where.

#### 3.12.6 Sheet GWorlds larger than their art

Two cases where `NewGWorld` allocates more rows than `LoadGraphic` writes, leaving uninitialised pixels:

* `furnitureSrcRect` is `0,0,64,278` (`GliderPRO/Sources/StructuresInit.c:301`) but `PICT` 4001 and mask 5001 are only 221 rows tall. Rows 221-277 of `furnitureSrcMap`/`furnitureMaskMap` are never written. The highest source rect in use, `srcRects[kStool]` = 48x38 at y=183 (bottom 221), stops exactly at the art's edge, so **nothing reads the garbage**. Harmless.
* `applianceSrcRect` is 80x**269** (`GliderPRO/Sources/StructuresInit.c:538`) but mask `PICT` 5005 is only **268** rows. Row 268 of `applianceMaskMap` is uninitialised — and `tvScreen2` = `(0,220)-(64,269)` (`GliderPRO/Sources/StructuresInit.c:572-573`) **does** reference row 268. This is a genuine shipped off-by-one. In practice `tvScreen2` is used with plain `CopyBits`, not `CopyMask`, so the garbage mask row is never consulted; but a Go port that pre-multiplies alpha into the sheet at load time will bake in whatever it decides row 268's mask is. **Recommendation: treat missing mask rows as fully transparent (0).**

#### 3.12.7 No picture clips itself

All 152 `Clip` opcodes have `rgnSize == 10` and a bbox that contains `picFrame`. `PICT` 5006's clip bbox is `(0,0,720,576)` — the authoring machine's screen. Clip opcodes can be ignored for extraction, but not for a general PICT renderer.

### 3.13 Summary: the minimum viable decoder

```
1.  data := resource bytes
2.  picFrame := Rect at data[2..10]
3.  W := picFrame.right - picFrame.left ; H := picFrame.bottom - picFrame.top
4.  canvas := new uint8[W*H]   (palette indices; init 0 = white)
5.  alpha  := new uint8[W*H]   (0 everywhere; only for 1-bit masks we care about coverage)
6.  detect version (3.2), position p after the header
7.  loop:
8.      if v2: align p to even
9.      op := (v2 ? u16 : u8)
10.     if op == OpEndPic: break
11.     if op in {0x0090,0x0091,0x0098,0x0099}:
12.         parse per 3.5; rows per 3.8/3.9; region per 3.10 if 0x0091/0x0099
13.         for each source row/col in srcRect:
14.             write to canvas at dstRect + (-picFrame.left, -picFrame.top)
15.     else: skip op_datalen(op) bytes (3.4)
16. emit PNG: 8-bit indexed with the clut-128 palette, or 1-bit greyscale for masks
```

---

## 4. The colour model

### 4.1 `clut` 128 and 129 are byte-identical, and are the only palette

Measured:

* `clut` 128 and `clut` 129 are both 2,056 bytes and **byte-for-byte identical**.
* Header: `ctSeed = 0x00000000`, `ctFlags = 0x0000`, `ctSize = 255` (256 entries). 8 + 8*256 = 2,056. ✔
* All 256 `value` fields are sequential 0, 1, 2, ... 255.
* Every 16-bit channel has high byte == low byte (e.g. `0xFFFF`, `0xCCCC`, `0x9999`), so an 8-bit palette is exact.
* The 256 RGB triples are **identical** to the embedded `ColorTable` of every one of the 167 8-bit raster ops in the file (SHA-256 over the concatenated 768 bytes agrees for all 167; prefix `5e107e5a1b7f8563`).

**Conclusion: Glider PRO uses exactly one 256-entry palette everywhere.** No 8-bit `PICT` carries a different table. A Go port can hard-code it.

Interestingly, **no source file in the game reads the `clut` resources.** They exist to be installed by the Palette Manager / to document the intent; the running game relies on the system device colour table (which is the standard Mac OS 8-bit table) plus `Index2Color`. The measured equality above is what makes hard-coding safe.

### 4.2 The palette, dumped

Index, then RRGGBB (8 bits per channel), four per line:

```
  0 FFFFFF     1 FFFFCC     2 FFFF99     3 FFFF66
  4 FFFF33     5 FFFF00     6 FFCCFF     7 FFCCCC
  8 FFCC99     9 FFCC66    10 FFCC33    11 FFCC00
 12 FF99FF    13 FF99CC    14 FF9999    15 FF9966
 16 FF9933    17 FF9900    18 FF66FF    19 FF66CC
 20 FF6699    21 FF6666    22 FF6633    23 FF6600
 24 FF33FF    25 FF33CC    26 FF3399    27 FF3366
 28 FF3333    29 FF3300    30 FF00FF    31 FF00CC
 32 FF0099    33 FF0066    34 FF0033    35 FF0000
 36 CCFFFF    37 CCFFCC    38 CCFF99    39 CCFF66
 40 CCFF33    41 CCFF00    42 CCCCFF    43 CCCCCC
 44 CCCC99    45 CCCC66    46 CCCC33    47 CCCC00
 48 CC99FF    49 CC99CC    50 CC9999    51 CC9966
 52 CC9933    53 CC9900    54 CC66FF    55 CC66CC
 56 CC6699    57 CC6666    58 CC6633    59 CC6600
 60 CC33FF    61 CC33CC    62 CC3399    63 CC3366
 64 CC3333    65 CC3300    66 CC00FF    67 CC00CC
 68 CC0099    69 CC0066    70 CC0033    71 CC0000
 72 99FFFF    73 99FFCC    74 99FF99    75 99FF66
 76 99FF33    77 99FF00    78 99CCFF    79 99CCCC
 80 99CC99    81 99CC66    82 99CC33    83 99CC00
 84 9999FF    85 9999CC    86 999999    87 999966
 88 999933    89 999900    90 9966FF    91 9966CC
 92 996699    93 996666    94 996633    95 996600
 96 9933FF    97 9933CC    98 993399    99 993366
100 993333   101 993300   102 9900FF   103 9900CC
104 990099   105 990066   106 990033   107 990000
108 66FFFF   109 66FFCC   110 66FF99   111 66FF66
112 66FF33   113 66FF00   114 66CCFF   115 66CCCC
116 66CC99   117 66CC66   118 66CC33   119 66CC00
120 6699FF   121 6699CC   122 669999   123 669966
124 669933   125 669900   126 6666FF   127 6666CC
128 666699   129 666666   130 666633   131 666600
132 6633FF   133 6633CC   134 663399   135 663366
136 663333   137 663300   138 6600FF   139 6600CC
140 660099   141 660066   142 660033   143 660000
144 33FFFF   145 33FFCC   146 33FF99   147 33FF66
148 33FF33   149 33FF00   150 33CCFF   151 33CCCC
152 33CC99   153 33CC66   154 33CC33   155 33CC00
156 3399FF   157 3399CC   158 339999   159 339966
160 339933   161 339900   162 3366FF   163 3366CC
164 336699   165 336666   166 336633   167 336600
168 3333FF   169 3333CC   170 333399   171 333366
172 333333   173 333300   174 3300FF   175 3300CC
176 330099   177 330066   178 330033   179 330000
180 00FFFF   181 00FFCC   182 00FF99   183 00FF66
184 00FF33   185 00FF00   186 00CCFF   187 00CCCC
188 00CC99   189 00CC66   190 00CC33   191 00CC00
192 0099FF   193 0099CC   194 009999   195 009966
196 009933   197 009900   198 0066FF   199 0066CC
200 006699   201 006666   202 006633   203 006600
204 0033FF   205 0033CC   206 003399   207 003366
208 003333   209 003300   210 0000FF   211 0000CC
212 000099   213 000066   214 000033   215 EE0000
216 DD0000   217 BB0000   218 AA0000   219 880000
220 770000   221 550000   222 440000   223 220000
224 110000   225 00EE00   226 00DD00   227 00BB00
228 00AA00   229 008800   230 007700   231 005500
232 004400   233 002200   234 001100   235 0000EE
236 0000DD   237 0000BB   238 0000AA   239 000088
240 000077   241 000055   242 000044   243 000022
244 000011   245 EEEEEE   246 DDDDDD   247 BBBBBB
248 AAAAAA   249 888888   250 777777   251 555555
252 444444   253 222222   254 111111   255 000000
```

### 4.3 Palette structure

This is the standard classic Mac OS 8-bit system palette. Its generation rule, which a Go port can use to synthesise it or to assert against:

```
levels := [6]uint8{0xFF, 0xCC, 0x99, 0x66, 0x33, 0x00}   // descending

i := 0
for r := 0; r < 6; r++ {
    for g := 0; g < 6; g++ {
        for b := 0; b < 6; b++ {
            if r == 5 && g == 5 && b == 5 { continue }   // pure black omitted here
            pal[i] = RGB{levels[r], levels[g], levels[b]}
            i++
        }
    }
}
// i == 215 at this point
ramp := [10]uint8{0xEE,0xDD,0xBB,0xAA,0x88,0x77,0x55,0x44,0x22,0x11}
for k := 0; k < 10; k++ { pal[215+k] = RGB{ramp[k], 0, 0} }        // 215..224 red
for k := 0; k < 10; k++ { pal[225+k] = RGB{0, ramp[k], 0} }        // 225..234 green
for k := 0; k < 10; k++ { pal[235+k] = RGB{0, 0, ramp[k]} }        // 235..244 blue
for k := 0; k < 10; k++ { pal[245+k] = RGB{ramp[k], ramp[k], ramp[k]} } // 245..254 grey
pal[255] = RGB{0, 0, 0}
```

Verified: this construction reproduces all 256 measured entries exactly. Key consequences:

* **Index 0 is pure white `#FFFFFF`.** This is the colour key for `transparent`-mode blits (§5.3) and the default fill of a fresh GWorld.
* **Index 255 is pure black `#000000`.**
* Index 5 is `#FFFF00` pure yellow — deliberately excluded (`if (i != 5)`) from the (dead) grey-wash animations at `GliderPRO/Sources/MainWindow.c:481` and `:568`, presumably because the splash text is drawn in it.
* The 6x6x6 cube ordering is **R outermost, then G, then B, all descending from 0xFF**. Getting this backwards is an easy and highly visible porting bug.

### 4.4 Named colour indices hard-coded by the drawing code

`GliderPRO/Sources/ObjectDraw.c:16-48` defines the master set. Every value is a palette index into the table above:

| Constant | Index | RGB | Source line |
|---|---|---|---|
| `k8WhiteColor` | 0 | FFFFFF | `GliderPRO/Sources/ObjectDraw.c:16` |
| `kYellowColor` | 5 | FFFF00 | `GliderPRO/Sources/ObjectDraw.c:17` |
| `kGoldColor` | 11 | FFCC00 | `GliderPRO/Sources/ObjectDraw.c:18` |
| `k8RedColor` | 35 | FF0000 | `GliderPRO/Sources/ObjectDraw.c:19` |
| `kPaleVioletColor` | 42 | CCCCFF | `GliderPRO/Sources/ObjectDraw.c:20` |
| `k8LtstGray3Color` | 43 | CCCCCC | `GliderPRO/Sources/ObjectDraw.c:38` |
| `k8LtTanColor` | 52 | CC9933 | `GliderPRO/Sources/ObjectDraw.c:21` |
| `k8BambooColor` | 53 | CC9900 | `GliderPRO/Sources/ObjectDraw.c:22` |
| `kDarkFleshColor` | 58 | CC6633 | `GliderPRO/Sources/ObjectDraw.c:23` |
| `k8OrangeColor` | 59 | CC6600 | `GliderPRO/Sources/ObjectDraw.c:26` |
| `k8TanColor` | 94 | 996633 | `GliderPRO/Sources/ObjectDraw.c:24` |
| `k8PissYellowColor` | 95 | 996600 | `GliderPRO/Sources/ObjectDraw.c:25` |
| `k8PumpkinColor` | 101 | 993300 | `GliderPRO/Sources/ObjectDraw.c:35` |
| `k8BrownColor` | 137 | 663300 | `GliderPRO/Sources/ObjectDraw.c:27` |
| `k8Red4Color` | 143 | 660000 | `GliderPRO/Sources/ObjectDraw.c:28` |
| `k8SkyColor` | 150 | 33CCFF | `GliderPRO/Sources/ObjectDraw.c:29` |
| `k8EarthBlueColor` | 170 | 333399 | `GliderPRO/Sources/ObjectDraw.c:30` |
| `k8DkGray3Color` | 172 | 333333 | `GliderPRO/Sources/ObjectDraw.c:46` |
| `k8DkRedColor` | 222 | 440000 | `GliderPRO/Sources/ObjectDraw.c:31` |
| `k8DkRed2Color` | 223 | 220000 | `GliderPRO/Sources/ObjectDraw.c:32` |
| `kIntenseGreenColor` | 225 | 00EE00 | `GliderPRO/Sources/ObjectDraw.c:33` |
| `kIntenseBlueColor` | 235 | 0000EE | `GliderPRO/Sources/ObjectDraw.c:34` |
| `k8LtstGrayColor` | 245 | EEEEEE | `GliderPRO/Sources/ObjectDraw.c:36` |
| `k8LtstGray2Color` | 246 | DDDDDD | `GliderPRO/Sources/ObjectDraw.c:37` |
| `k8LtstGray4Color` | 247 | BBBBBB | `GliderPRO/Sources/ObjectDraw.c:39` |
| `k8LtstGray5Color` | 248 | AAAAAA | `GliderPRO/Sources/ObjectDraw.c:40` |
| `k8LtGrayColor` | 249 | 888888 | `GliderPRO/Sources/ObjectDraw.c:41` |
| `k8GrayColor` | 250 | 777777 | `GliderPRO/Sources/ObjectDraw.c:42` |
| `k8Gray2Color` | 251 | 555555 | `GliderPRO/Sources/ObjectDraw.c:43` |
| `k8DkGrayColor` | 252 | 444444 | `GliderPRO/Sources/ObjectDraw.c:44` |
| `k8DkGray2Color` | 253 | 222222 | `GliderPRO/Sources/ObjectDraw.c:45` |
| `k8DkstGrayColor` | 254 | 111111 | `GliderPRO/Sources/ObjectDraw.c:47` |
| `k8BlackColor` | 255 | 000000 | `GliderPRO/Sources/ObjectDraw.c:48` |

`GliderPRO/Sources/ObjectDraw2.c:19-37` re-declares a 19-entry subset with identical values, plus one extra name: `kIntenseYellowColor` = 5 (`GliderPRO/Sources/ObjectDraw2.c:20`) — same index as `kYellowColor`.

One more, in the shared header: `#define kRedOrangeColor8 23   // actually, 18` (`GliderPRO/Headers/GliderDefines.h:542`). Index 23 is `FF6600`; the comment claims it should be 18 (`FF66FF`, magenta) which would be wrong, so treat the code value 23 as authoritative.

These indices are used only for *procedurally drawn* geometry — clock hands, table legs, cabinet shading, window sills, the calendar month text. They are not used for sprite art, which comes from `PICT`s. A Go port that renders in truecolour can look these up in the palette table and forget indices entirely, **except** where the code compares or writes pixel values (it never does).

### 4.5 `ColorUtils.c` — the entire colour-drawing API

`ColorUtils.c` exports **thirteen** functions: nine drawing wrappers, three fore-colour setters and
`RestoreColorsSlam`. Each of the nine drawing wrappers follows the same save/set/draw/restore pattern.
Representative example:

```c
void ColorRect (Rect *theRect, long color)      // GliderPRO/Sources/ColorUtils.c:36
{
    RGBColor    theRGBColor, wasColor;

    GetForeColor(&wasColor);
    Index2Color(color, &theRGBColor);
    RGBForeColor(&theRGBColor);
    PaintRect(theRect);
    RGBForeColor(&wasColor);
}
```

| Function | Line | Draw call |
|---|---|---|
| `ColorText(StringPtr, long)` | `GliderPRO/Sources/ColorUtils.c:20` | `DrawString` |
| `ColorRect(Rect*, long)` | `GliderPRO/Sources/ColorUtils.c:36` | `PaintRect` |
| `ColorOval(Rect*, long)` | `GliderPRO/Sources/ColorUtils.c:52` | `PaintOval` |
| `ColorRegion(RgnHandle, long)` | `GliderPRO/Sources/ColorUtils.c:68` | `PaintRgn` |
| `ColorLine(short,short,short,short,long)` | `GliderPRO/Sources/ColorUtils.c:84` | `MoveTo`+`LineTo` |
| `HiliteRect(Rect*, short, short)` | `GliderPRO/Sources/ColorUtils.c:103` | `PenMode(patXor)` + `PaintRect` with two colours |
| `ColorFrameRect(Rect*, long)` | `GliderPRO/Sources/ColorUtils.c:120` | `FrameRect` |
| `ColorFrameWHRect(Rect*, short, short, long)` | `GliderPRO/Sources/ColorUtils.c:137` | `FrameRect` on a width/height-adjusted rect |
| `ColorFrameOval(Rect*, long)` | `GliderPRO/Sources/ColorUtils.c:153` | `FrameOval` |
| `LtGrayForeColor()` | `GliderPRO/Sources/ColorUtils.c:169` | `RGBForeColor{0xBFFF,0xBFFF,0xBFFF}` |
| `GrayForeColor()` | `GliderPRO/Sources/ColorUtils.c:185` | `RGBForeColor{0x7FFF,0x7FFF,0x7FFF}` |
| `DkGrayForeColor()` | `GliderPRO/Sources/ColorUtils.c:201` | `RGBForeColor{0x3FFF,0x3FFF,0x3FFF}` |
| `RestoreColorsSlam()` | `GliderPRO/Sources/ColorUtils.c:218` | `RestoreDeviceClut(nil)` + `PaintBehind(nil, GetGrayRgn())` |

The three grey helpers pass 16-bit RGB values that are **not** palette entries: `0xBFFF` → 8-bit `0xBF` = 191, `0x7FFF` → `0x7F` = 127, `0x3FFF` → `0x3F` = 63. On an 8-bit device QuickDraw snaps each to the nearest palette entry. Re-derived by scanning all 256 entries of `clut` 128 for the minimum squared RGB distance (not just the 245-254 grey ramp — the 6x6x6 colour cube contributes greys too, and index 43 `CCCCCC` is closer to 191 than index 248 `AAAAAA` is):

* 191 → **247** (`BBBBBB` = 187, Δ=4). Runners-up: 43 `CCCCCC` = 204 (Δ=13), 248 `AAAAAA` = 170 (Δ=21).
* 127 → **250** (`777777` = 119, Δ=8). Runners-up: 249 `888888` = 136 (Δ=9), 129 `666666` = 102 (Δ=25).
* 63 → **252** (`444444` = 68, Δ=5). Runners-up: 172 `333333` = 51 (Δ=12), 251 `555555` = 85 (Δ=22).

A Go port should either snap identically or just use the literal RGB values.

`RestoreColorsSlam` is the only Palette Manager call that survives in live code.

### 4.6 The 16-grey path

There is no separate art for 4-bit mode. The game relies on QuickDraw's dither-on-`CopyBits`. The only source-level differences are:

1. Text colouring: `GliderPRO/Sources/MainWindow.c:68` chooses `ForeColor(blackColor)` style output instead of `ColorText(..., 5L)` / `ColorText(..., 28L)`.
2. Substituted small indices in procedural drawing, e.g. `GliderPRO/Sources/ObjectDraw2.c:1047-1052` (`DrawWallWindow` swaps `k8BrownColor`/`k8TanColor`/`k8DkRed2Color` for 11/9/15).
3. The even-x alignment fixes in `DynamicMaps.c` listed in §1.3.

**Recommendation for the Go port: do not implement 16-grey mode.** It is a 1994 concession to 4-bit displays, adds five special cases, and changes nothing about the asset pipeline.

### 4.7 Dead palette code (do not port)

Three substantial functions in `MainWindow.c` are commented out in the shipped source. They are documented here only so a porter does not mistake them for live behaviour:

* `SetPaletteToGrays` (`GliderPRO/Sources/MainWindow.c:454`) — converts all 256 palette entries to luminance via `longGray = (red*3)/10 + (green*6)/10 + (blue*1)/10`, skipping index 5, then `SetEntries(0, 255, newColors)`.
* `HardDrawMainWindow` (`GliderPRO/Sources/MainWindow.c:504`) — `SwapMMUMode(true32b)` and a hand-rolled 460-row x 160-long screen blit.
* `WashColorIn` (`GliderPRO/Sources/MainWindow.c:552`) — a `kGray2ColorSteps 180`-step colour fade, again skipping index 5.

None is called. The globals they need (`CTabHandle theCTab; PixMapHandle thePMap; ColorSpec *wasColors, *newColors;` at `GliderPRO/Sources/MainWindow.c:27-43`) are still declared.

---

## 5. Transparency and masking

There are exactly **four** compositing paths in Glider PRO. Getting these right is the single biggest correctness risk in an art port, so each is stated with the measurement that proves it.

### 5.1 The four paths

| # | Mechanism | Where | Used for |
|---|---|---|---|
| 1 | `CopyMask(colour, mask, dest, srcR, maskR, destR)` with a permanent 1-bit mask GWorld | ~40 sites in `ObjectDraw.c`, `ObjectDraw2.c`, `Render.c` | every persistent sprite sheet that has a mask |
| 2 | `CopyBits(..., transparent, nil)` — colour key on white / index 0 | `GliderPRO/Sources/ObjectDraw2.c:1401`, `:1430` | 20 object types drawn straight from their own `PICT`, plus user custom pictures |
| 3 | Throwaway 8-bit + 1-bit GWorld pair built, `CopyMask`ed, disposed, per draw call | `GliderPRO/Sources/ObjectDraw2.c:173`, `:210`, `:655`, `:749`, `:803`, `:858`, `:1272` | mailboxes, TV, VCR, stereo, microwave, cobweb, cloud |
| 4 | Opaque `DrawPicture` straight into `backSrcMap` | `GliderPRO/Sources/ObjectDraw2.c:1197` | 7 object types that are full-bleed rectangles |

### 5.2 Path 1: the mask-ID convention, and mask polarity proven

**The convention is `maskPictID = colourPictID + 1000`.** Verified pairs:

| Colour PICT | Mask PICT | Sheet |
|---|---|---|
| 3998 | 4998 | shadow |
| 3999 | 4999 | glider (also serves 3963, 3974, 3976) |
| 4000 | 5000 | blower |
| 4001 | 5001 | furniture |
| 4002 | 5002 | prizes/bonus |
| 4003 | — | switches (**no mask exists**) |
| 4004 | 5004 | lights |
| 4005 | 5005 | appliances |
| 4006 | 5006 | points |
| 4007 | 5007 | rubber bands |
| 4008 | 5008 | transport |
| 4009 | 5009 | toast |
| 4010 | 5010 | shredded |
| 4011 | 5011 | balloon |
| 4012 | 5012 | copter |
| 4013 | 5013 | dart |
| 4014 | 5014 | ball |
| 4015 | 5015 | drip |
| 4016 | 5016 | enemy (spider) |
| 4017 | 5017 | fish |
| 4018 | 5018 | clutter |

Two exceptions:

* **The angel breaks the rule**: `GliderPRO/Sources/StructuresInit2.c:135` loads `LoadGraphic(kAngelPictID + 1)`, i.e. `PICT` **1020**, not 2019. Both 1019 and 1020 exist and are 96x44.
* **Switches have no mask at all.** `InitSwitches` (`GliderPRO/Sources/StructuresInit.c:447`) creates only `switchSrcMap` and every switch draw is a plain opaque `CopyBits` (`GliderPRO/Sources/ObjectDraw2.c:301-371`). `PICT` 5003 is absent from the resource fork and absent from the `Environ.c` memory budget.

Also outside the +1000 family but following the same idea: `PICT` 1989 masks 1990 (game-over pages), 1991 masks 1992 (banner page bottom), 1998 masks 1994 (high scores).

#### Polarity, measured

`CopyMask`'s documented semantics are "where the mask bit is 1, copy the source". To *prove* it against the shipped art rather than assume it, six colour+mask pairs were composited and the predicate `colour != 0` (non-white) was compared against `mask == 1`:

| Pair | Pixels | Agreement | Mask-opaque but colour is white | Mask-clear but colour is non-white |
|---|---|---|---|---|
| 4000 / 5000 (blower) | 19,296 | 99.84% | 31 | **0** |
| 4002 / 5002 (prizes) | 33,264 | 93.37% | 1,942 | 264 |
| 4001 / 5001 (furniture) | 14,144 | 99.22% | 45 | 65 |
| 4011 / 5011 (balloon) | 5,760 | 99.69% | 18 | **0** |
| 4017 / 5017 (fish) | 2,048 | 98.78% | 25 | **0** |
| 1019 / 1020 (angel) | 4,224 | 83.88% | **681** | **0** |

Interpretation:

* The agreement is high but not perfect, which is exactly what you expect if the mask is authoritative and the artist used white *inside* the sprite.
* The angel is decisive: **681 pixels are white but mask-opaque.** Those are the angel's white robe/wings. A colour-key blit would punch 681 holes in it.
* The blower, balloon and fish have **zero** pixels that are non-white outside the mask, i.e. the mask never clips visible art.
* Prizes (4002) and furniture (4001) do have some non-white pixels outside the mask (264 and 65). Those are stray/unused pixels in the sheet's inter-frame gutters; they are correctly suppressed by the mask.

**Conclusion, stated as the porting rule: for every sheet that has a mask, use the mask. Bit value 1 (a black pixel in the 1-bit PICT) means opaque. Never substitute a white colour key for a mask.** The masks carry information the colour data does not.

#### The `CopyMask` idiom

The canonical call, from `GliderPRO/Sources/ObjectDraw.c:59`:

```c
void DrawSimpleBlowers (short what, Rect *theRect)
{
    CopyMask((BitMap *)*GetGWorldPixMap(blowerSrcMap),
             (BitMap *)*GetGWorldPixMap(blowerMaskMap),
             (BitMap *)*GetGWorldPixMap(backSrcMap),
             &srcRects[what], &srcRects[what], theRect);
}
```

Note that the source rect and the mask rect are **the same rect**: the mask sheet is pixel-aligned with the colour sheet, same dimensions, same layout. This holds for every mask pair in the game. That means a Go port can pre-multiply the mask into the colour sheet at load time, producing one RGBA sheet per pair, and reduce path 1 to an ordinary alpha blit. The only caveat is §3.12.6 (mask 5005 is one row short).

### 5.3 Path 2: the white colour key, measured

`GliderPRO/Sources/ObjectDraw2.c:1302`:

```c
void DrawPictSansWhiteObject (short what, Rect *theRect)
{
    Rect        bounds;
    GWorldPtr   tempMap;
    short       pictID;
    CGrafPtr    wasCPort;
    GDHandle    wasWorld;

    GetGWorld(&wasCPort, &wasWorld);
    switch (what) { /* 20 cases, see below */ }
    bounds = srcRects[what];
    CreateOffScreenGWorld(&tempMap, &bounds, kPreferredDepth);
    SetGWorld(tempMap, nil);
    LoadGraphic(pictID);
    CopyBits((BitMap *)*GetGWorldPixMap(tempMap),
             (BitMap *)*GetGWorldPixMap(backSrcMap),
             &srcRects[what], theRect, transparent, nil);
    DisposeGWorld(tempMap);
    SetGWorld(wasCPort, wasWorld);
}
```

QuickDraw's `transparent` transfer mode copies every source pixel **except** those equal to the source port's **background colour**. A freshly created GWorld's background is white, and nothing in the game changes it before these calls, so the key is white = **palette index 0**.

To prove the key is actually used (rather than the art being padded with some other colour), the palette-index histogram of all 20 sans-white pictures was measured. Every one contains index-0 pixels; the fraction of the sprite that is keyed out:

| Object | PICT | % index 0 |
|---|---|---|
| `kHipLamp` | 3994 | 86.8% |
| `kDecoLamp` | 3993 | 84.2% |
| `kGuitar` | 3991 | 57.7% |
| `kVase2` | 3968 | 53.5% |
| `kBBQ` | 3988 | 46.5% |
| `kChimes` | 3961 | 36.8% |
| `kBear` | 3972 | 35.5% |
| `kCinderBlock` | 3960 | 23.5% |
| `kVase1` | 3969 | 17.4% |
| `kDoorInRt` | 3983 | 17.2% |
| `kDoorInLf` | 3984 | 17.2% |
| `kFlowerBox` | 3959 | 12.9% |
| `kWindowInRt` | 3979 | 12.7% |
| `kWindowInLf` | 3980 | 12.7% |
| `kRug` | 3962 | 12.0% |
| `kBooks` | 3964 | 10.1% |
| `kManhole` | 3967 | 9.6% |
| `kTrunk` | 3987 | 7.2% |
| `kFireplace` | 3973 | 4.4% |
| `kUpStairs` | 3997 | 1.9% |

`kHipLamp` at 86.8% is the clearest case: a tall thin lamp in a 72x276 box is mostly background.

The exact `switch` (`GliderPRO/Sources/ObjectDraw2.c:1313-1394`):

| `what` | Constant value | `pictID` |
|---|---|---|
| `kBBQ` | 0x0C | 3988 |
| `kTrunk` | 0x1B | 3987 |
| `kManhole` | 0x1D | 3967 |
| `kBooks` | 0x1E | 3964 |
| `kUpStairs` | 0x31 | 3997 |
| `kDoorInLf` | 0x37 | 3984 |
| `kDoorInRt` | 0x38 | 3983 |
| `kWindowInLf` | 0x3B | 3980 |
| `kWindowInRt` | 0x3C | 3979 |
| `kHipLamp` | 0x54 | 3994 |
| `kDecoLamp` | 0x55 | 3993 |
| `kGuitar` | 0x64 | 3991 |
| `kCinderBlock` | 0x6B | 3960 |
| `kFlowerBox` | 0x6C | 3959 |
| `kFireplace` | 0x84 | 3973 |
| `kBear` | 0x87 | 3972 |
| `kVase1` | 0x89 | 3969 |
| `kVase2` | 0x8A | 3968 |
| `kRug` | 0x8E | 3962 |
| `kChimes` | 0x8F | 3961 |

The user-supplied variant, `GliderPRO/Sources/ObjectDraw2.c:1412`:

```c
void DrawCustPictSansWhite (short pictID, Rect *theRect)
{
    bounds = *theRect;
    ZeroRectCorner(&bounds);
    CreateOffScreenGWorld(&tempMap, &bounds, kPreferredDepth);
    SetGWorld(tempMap, nil);
    LoadGraphic(pictID);
    CopyBits((BitMap *)*GetGWorldPixMap(tempMap),
             (BitMap *)*GetGWorldPixMap(backSrcMap),
             &bounds, theRect, transparent, nil);
    DisposeGWorld(tempMap);
}
```

Called from `GliderPRO/Sources/ObjectDrawAll.c` for `kCustomPict` (0x6E) with `pictID = thisObject.data.g.height` — i.e. the object record's "height" field is overloaded to carry a PICT resource ID, which will be >= `kUserBackground 3000` for house-supplied art. Note the temp GWorld is sized from the *object's* rect and the graphic is loaded unscaled at (0,0), so a user PICT larger than the object rect is silently cropped and a smaller one leaves white (keyed-out) margin.

**Porting rule for path 2: alpha = 0 where the palette index is 0, alpha = 255 otherwise.** No mask resource is involved.

### 5.4 Path 3: throwaway GWorld pairs

`DrawPictWithMaskObject` (`GliderPRO/Sources/ObjectDraw2.c:1253`) is the generic form:

```c
void DrawPictWithMaskObject (short what, Rect *theRect)
{
    switch (what)
    {
        case kCobweb: pictID = kCobwebPictID; maskID = kCobwebMaskID; break;  // 3958 / 3927
        case kCloud:  pictID = kCloudPictID;  maskID = kCloudMaskID;  break;  // 3965 / 3921
    }
    bounds = srcRects[what];
    CreateOffScreenGWorld(&tempMap,  &bounds, kPreferredDepth);
    SetGWorld(tempMap, nil);  LoadGraphic(pictID);
    CreateOffScreenGWorld(&tempMask, &bounds, 1);
    SetGWorld(tempMask, nil); LoadGraphic(maskID);
    CopyMask(tempMap, tempMask, backSrcMap, &srcRects[what], &srcRects[what], theRect);
    DisposeGWorld(tempMap);
    DisposeGWorld(tempMask);
}
```

The specialised siblings do the same thing but also composite an animated sub-rect (screen, LED, clock display) before the `CopyMask`:

| Function | Line | Colour PICT | Mask PICT |
|---|---|---|---|
| `DrawMailboxLeft` | `GliderPRO/Sources/ObjectDraw2.c:112` | 3986 | 3904 |
| `DrawMailboxRight` | `GliderPRO/Sources/ObjectDraw2.c:195` | 3985 | 3903 |
| `DrawTV` | `GliderPRO/Sources/ObjectDraw2.c:640` | 3992 | 3912 |
| `DrawVCR` | `GliderPRO/Sources/ObjectDraw2.c:734` | 3990 | 3913 |
| `DrawStereo` | `GliderPRO/Sources/ObjectDraw2.c:788` | 3989 | 3914 |
| `DrawMicrowave` | `GliderPRO/Sources/ObjectDraw2.c:843` | 3971 | 3915 |
| `DrawPictWithMaskObject` (kCobweb) | `GliderPRO/Sources/ObjectDraw2.c:1253` | 3958 | 3927 |
| `DrawPictWithMaskObject` (kCloud) | `GliderPRO/Sources/ObjectDraw2.c:1253` | 3965 | 3921 |

`kMailboxBase 296` (`GliderPRO/Sources/ObjectDraw2.c:105`) is the baseline y the mailbox art is anchored to.

These eight mask PICTs (3903, 3904, 3912, 3913, 3914, 3915, 3921, 3927) are the **only** mask IDs in the 3900-3927 range that actually exist as resources.

### 5.5 Path 4: the opaque path

`GliderPRO/Sources/ObjectDraw2.c:1197`:

```c
void DrawPictObject (short what, Rect *theRect)
{
    switch (what) { /* 7 cases */ }
    GetGWorld(&wasCPort, &wasWorld);
    SetGWorld(backSrcMap, nil);
    thePicture = GetPicture(pictID);
    bounds = srcRects[what];
    QOffsetRect(&bounds, theRect->left, theRect->top);
    DrawPicture(thePicture, &bounds);
    ReleaseResource((Handle)thePicture);
    SetGWorld(wasCPort, wasWorld);
}
```

Cases (`GliderPRO/Sources/ObjectDraw2.c:1205-1234`):

| `what` | Value | `pictID` |
|---|---|---|
| `kFilingCabinet` | 0x14 | 3995 |
| `kDownStairs` | 0x32 | 3996 |
| `kDoorExRt` | 0x39 | 3982 |
| `kDoorExLf` | 0x3A | 3981 |
| `kWindowExRt` | 0x3D | 3977 |
| `kWindowExLf` | 0x3E | 3978 |
| `kOzma` | 0x81 | 3975 |

Note this path uses `bounds = srcRects[what]` offset to the destination, **not** the object's own rect, and it goes straight into `backSrcMap` with no keying. Every one of these seven pictures' `picFrame` size equals its `srcRects` entry (3995 = 74x107 = `srcRects[kFilingCabinet]`; 3996/3997 = 160x267 = `srcRects[kDownStairs]`/`[kUpStairs]`; 3981/3982 = 16x322; 3977/3978 = 16x170; 3975 = 102x92 = `srcRects[kOzma]`), so no scaling occurs.

Also opaque: every switch draw (`GliderPRO/Sources/ObjectDraw2.c:301-371`), the floor support strip (`GliderPRO/Sources/RoomGraphics.c:270`), and the room background tiles (`GliderPRO/Sources/RoomGraphics.c:249`).

### 5.6 Vestigial mask-ID constants

`GliderPRO/Sources/ObjectDraw2.c:39-66` declares 28 mask-ID constants, of which only 8 correspond to real resources. The full list, with existence marked:

| Constant | ID | Resource exists? |
|---|---|---|
| `kBBQMaskID` | 3900 | no |
| `kUpStairsMaskID` | 3901 | no |
| `kTrunkMaskID` | 3902 | no |
| `kMailboxRightMaskID` | 3903 | **yes** |
| `kMailboxLeftMaskID` | 3904 | **yes** |
| `kDoorInLeftMaskID` | 3905 | no |
| `kDoorInRightMaskID` | 3906 | no |
| `kWindowInLeftMaskID` | 3907 | no |
| `kWindowInRightMaskID` | 3908 | no |
| `kHipLampMaskID` | 3909 | no |
| `kDecoLampMaskID` | 3910 | no |
| `kGuitarMaskID` | 3911 | no |
| `kTVMaskID` | 3912 | **yes** |
| `kVCRMaskID` | 3913 | **yes** |
| `kStereoMaskID` | 3914 | **yes** |
| `kMicrowaveMaskID` | 3915 | **yes** |
| `kFireplaceMaskID` | 3916 | no |
| `kBearMaskID` | 3917 | no |
| `kVase1MaskID` | 3918 | no |
| `kVase2MaskID` | 3919 | no |
| `kManholeMaskID` | 3920 | no |
| `kCloudMaskID` | 3921 | **yes** |
| `kBooksMaskID` | 3922 | no |
| `kRugMaskID` | 3923 | no |
| `kChimesMaskID` | 3924 | no |
| `kCinderMaskID` | 3925 | no |
| `kFlowerBoxMaskID` | 3926 | no |
| `kCobwebMaskID` | 3927 | **yes** |

Compare with the sans-white list in §5.3: the 20 objects whose masks are missing are exactly the 20 that use the white colour key. So this is not an accident — at some point during development Calhoun replaced 20 mask-based draws with colour-key draws and deleted the masks, leaving the `#define`s behind. **A Go port must use the colour key for those 20 and must not go looking for masks 3900-3926.**

---

## 6. The sprite-atlas map

This is the manifest §5 has been building toward: for every object type the game can draw, *which
PICT the pixels come from, where inside it, how many frames, and where the alpha comes from*. It is
the extraction-unit view. The complementary **draw-time dispatcher** view — which helper function
`DrawARoomsObjects` calls for each `what`, and in what order relative to the background — is in
`docs/analysis/rendering.md:2230` (§7.4-§7.6) and is not repeated here.

### 6.1 `srcRects[]`: one array, seven different meanings

```c
srcRects = (Rect *)NewPtr(sizeof(Rect) * kNumSrcRects);   // StructuresInit2.c:271
```

| Fact | Value | Cite |
|---|---|---|
| Array type | `Rect *srcRects` — a heap array of `Rect`, not a fixed global | `GliderPRO/Sources/StructuresInit2.c:270-273` |
| Element size | 8 bytes (`Rect` = 4 x `int16`) | — |
| Length | `kNumSrcRects` = `0x90` = **144** | `GliderPRO/Headers/GliderDefines.h:437` |
| Allocation | `NewPtr`, **not** `NewPtrClear` — untouched slots hold whatever was in the heap | `GliderPRO/Sources/StructuresInit2.c:271` |
| Index | the object's `what` byte, directly; the array is sparse | `GliderPRO/Sources/ObjectRects.c:58` |
| Initialised in | exactly one function, `InitSrcRects` | `GliderPRO/Sources/StructuresInit2.c:306-475` |
| Entries actually assigned | **116** of 144 | measured, below |
| Entries never assigned | **28** | §6.4 |

The 116 assignments were re-derived mechanically rather than read off by eye: a script re-executes
every `QSetRect`/`QOffsetRect` call in `InitSrcRects:306-475` in source order, resolving the two
symbolic arguments (`kTableThick` = 8, `kShelfThick` = 6, `GliderPRO/Headers/GliderDefines.h:439-440`)
from the header. Argument order matters and is easy to get wrong:

```c
void QSetRect (Rect *theRect, short l, short t, short r, short b)   // RectUtils.c:210
void QOffsetRect (Rect *theRect, short h, short v)                  // RectUtils.c:197
```

`QSetRect` takes **(left, top, right, bottom)** while the on-disk and in-memory `Rect` field order is
**(top, left, bottom, right)**. Every rect in this section is printed as `(l,t,r,b)` to match the
`QSetRect` call it came from; `rendering.md` §7.4 prints the same rects as `(t,l,b,r)` to match the
struct. They agree.

Counting the assignments per comment-labelled group in `InitSrcRects`:

| Group | Comment at | Entries |
|---|---|---|
| Blowers | `GliderPRO/Sources/StructuresInit2.c:308` | 16 |
| Furniture | `GliderPRO/Sources/StructuresInit2.c:338` | 15 |
| Prizes | `GliderPRO/Sources/StructuresInit2.c:357` | 15 |
| Transport | `GliderPRO/Sources/StructuresInit2.c:385` | 16 |
| Switch | `GliderPRO/Sources/StructuresInit2.c:404` | 9 |
| Lights | `GliderPRO/Sources/StructuresInit2.c:418` | 8 |
| Appliances | `GliderPRO/Sources/StructuresInit2.c:430` | 14 |
| Enemies | `GliderPRO/Sources/StructuresInit2.c:450` | 9 |
| Clutter | `GliderPRO/Sources/StructuresInit2.c:460` | 14 |
| **Total** | | **116** |

An entry means one of exactly seven things. Which one it is cannot be read off the rect — it is
determined by the draw helper the dispatcher calls. This taxonomy is the load-bearing content of §6,
because it decides whether an extractor should crop a sheet, decode a standalone PICT, or emit
nothing at all:

| Kind | Meaning | Extractor action | Count |
|---|---|---|---|
| `sheet` | a genuine sub-rect of one of the 14 static 8-bit sheet GWorlds; the helper passes `srcRects[what]` straight to `CopyMask`/`CopyBits` as the source rect | crop the sheet's RGBA at this rect | **50** |
| `tmpl` | size/hit template only. The art comes from *named* sub-rect globals of the same sheet (`tableSrc`, `hingeSrc`, `trackLightSrc[i]`, …), not from this rect | crop the named sub-rects (§6.6) instead | **12** |
| `key` | the object has its own `PICT`, blitted with QuickDraw `transparent` mode (white/index 0 keyed out) | decode the PICT, alpha = 0 where index == 0 | **21** |
| `opaq` | the object has its own `PICT`, drawn with no transparency at all | decode the PICT, alpha = 255 everywhere | **9** |
| `pair` | the object has its own colour `PICT` **and** its own 1-bit mask `PICT`, composited through a throwaway GWorld pair | decode both, alpha from the mask | **8** |
| `proc` | drawn procedurally from coloured rects and lines; there are no pixels to extract | emit nothing; port the drawing code | **3** |
| `none` | never drawn at all (invisible objects, triggers, lift areas). The rect is only the default size the editor gives a new one | emit nothing | **13** |
| | | **Total** | **116** |

50 + 12 + 21 + 9 + 8 + 3 + 13 = 116. The `key` count is 21 rather than the 20 of §5.3 because
`kCustomPict` (0x6E) is the 21st colour-keyed type and its `PICT` ID is a runtime value, not a
constant (§7.9).

### 6.2 The 14 static sheet surfaces — extraction units

These are the surfaces that `srcRects` entries index into. Every one is a single 8-bit `PICT` plus
(with one exception) a single same-size 1-bit mask `PICT`, loaded once at startup by `LoadGraphic`
into a GWorld whose bounds come from a hard-coded `QSetRect`. The full GWorld inventory including
the non-atlas surfaces is §1.2; this table adds the *measured* art dimensions and the alpha coverage
produced by running the §7 pipeline over them.

| Sheet key | GWorld pair | GWorld bounds | Art `PICT` | measured art size | Mask `PICT` | measured mask size | opaque px / total (measured) |
|---|---|---|---|---|---|---|---|
| `blower` | `blowerSrcMap` / `blowerMaskMap` | 48 x 402 | **4000** | 48x402 | **5000** | 48x402 | 9,471 / 19,296 = 49.1% |
| `furniture` | `furnitureSrcMap` / `furnitureMaskMap` | 64 x 278 | **4001** | 64x**221** | **5001** | 64x**221** | 8,904 / 14,144 = 63.0% |
| `bonus` | `bonusSrcMap` / `bonusMaskMap` | 88 x 378 | **4002** | 88x378 | **5002** | 88x378 | 12,599 / 33,264 = 37.9% |
| `switch` | `switchSrcMap` — **no mask GWorld** | 32 x 104 | **4003** | 32x104 | **none (5003 absent)** | — | 3,328 / 3,328 = 100% |
| `light` | `lightSrcMap` / `lightMaskMap` | 72 x 126 | **4004** | 72x126 | **5004** | 72x126 | 4,785 / 9,072 = 52.7% |
| `appliance` | `applianceSrcMap` / `applianceMaskMap` | 80 x 269 | **4005** | 80x269 | **5005** | 80x**268** | 15,783 / 21,520 = 73.3% |
| `trans` | `transSrcMap` / `transMaskMap` | 56 x 32 | **4008** | 56x32 | **5008** | 56x32 | 1,530 / 1,792 = 85.4% |
| `balloon` | `balloonSrcMap` / `balloonMaskMap` | 24 x 240 | **4011** | 24x240 | **5011** | 24x240 | 3,811 / 5,760 = 66.2% |
| `copter` | `copterSrcMap` / `copterMaskMap` | 32 x 300 | **4012** | 32x300 | **5012** | 32x300 | 1,739 / 9,600 = 18.1% |
| `dart` | `dartSrcMap` / `dartMaskMap` | 64 x 76 | **4013** | 64x76 | **5013** | 64x76 | 2,433 / 4,864 = 50.0% |
| `ball` | `ballSrcMap` / `ballMaskMap` | 32 x 64 | **4014** | 32x64 | **5014** | 32x64 | 1,524 / 2,048 = 74.4% |
| `drip` | `dripSrcMap` / `dripMaskMap` | 16 x 72 | **4015** | 16x72 | **5015** | 16x72 | 340 / 1,152 = 29.5% |
| `enemy` | `enemySrcMap` / `enemyMaskMap` | 36 x 33 | **4016** | 36x33 | **5016** | 36x33 | 1,010 / 1,188 = 85.0% |
| `clutter` | `clutterSrcMap` / `clutterMaskMap` | 128 x 69 | **4018** | 128x69 | **5018** | 128x69 | 5,043 / 8,832 = 57.1% |

`QSetRect` sites for the bounds: `GliderPRO/Sources/StructuresInit.c:253` (blower), `:301`
(furniture), `:350` (bonus), `:431` (trans), `:455` (switch), `:501` (light), `:538` (appliance),
`:623` (balloon), `:632` (copter), `:641` (dart), `:650` (ball), `:659` (drip), `:668` (enemy),
and `GliderPRO/Sources/StructuresInit2.c:63` (clutter).

Two dimension anomalies, both real and both harmless, but both fatal to a loader that sizes its
surface from the `PICT` instead of from the `QSetRect`:

* **Furniture: 57 unpainted rows.** `furnitureSrcRect` is 64x278 but `PICT` 4001 and mask 5001 are both 64x221. Nothing reads below y=221: the deepest furniture-sheet rect is `srcRects[kStool]` = (0,183,48,221) (`GliderPRO/Sources/StructuresInit2.c:349-350`), whose bottom is exactly 221. So the art is precisely as tall as its deepest consumer and the extra 57 rows of the GWorld are pure slack.
* **Appliance mask: one row short.** `PICT` 4005 is 80x269, mask 5005 is 80x268. The missing row 268 lies inside `tvScreen2` (0,220,64,269), which is blitted with opaque `srcCopy` and never consults the mask (`GliderPRO/Sources/ObjectDraw2.c:640-733`), so the shortfall is invisible. An extractor that pads the mask with 0 (transparent) reproduces the original exactly; one that pads with 1 also does, because nothing samples that row through the mask.

There are exactly **12** further art surfaces that hold frames or strips but are **not** indexed by
`srcRects` at all, so an extractor reaches them through the named globals of §6.6 rather than through
the atlas:

| # | Surface | Art `PICT` | Mask `PICT` | measured size |
|---|---|---|---|---|
| 1 | `glidSrcMap` | 3999 (`kGliderPictID`) | 4999 | 48x668 |
| 2 | `glid2SrcMap` | 3974 (`kGlider2PictID`) | shares `glidMaskMap` (4999) | 48x668 |
| 3 | `shadowSrcMap` | 3998 | 4998 | 48x18 |
| 4 | `bandsSrcMap` | 4007 | 5007 | 16x18 |
| 5 | `pointsSrcMap` | 4006 | 5006 (origin (91,0)) | 24x120 |
| 6 | `toastSrcMap` | 4009 | 5009 | 32x174 |
| 7 | `shredSrcMap` | 4010 | 5010 (origin (0,195)) | 40x35 |
| 8 | `fishSrcMap` | 4017 | 5017 | 16x128 |
| 9 | `suppSrcMap` | 1999 | none — opaque | 512x44 |
| 10 | `angelSrcMap` | 1019 (`kAngelPictID`) | **1020** (`+1`, not `+1000`) | 96x44 |
| 11 | `badgeSrcMap` | 1996 (`kBadgePictID`) | none — opaque | 32x66 |
| 12 | `boardSrcMap` | 1997 | none — opaque | art is **1536x20**, drawn into a `houseRect`-wide x 20 GWorld at a computed h-offset (`GliderPRO/Sources/StructuresInit.c:78-90`) |

Two more 48x668 colour sheets share `glidMaskMap` and are swapped into the two glider surfaces at
runtime rather than getting surfaces of their own: **3976** `kGliderFoilPictID` and **3963**
`kGliderFoil2PictID` (`GliderPRO/Headers/GliderDefines.h:567`, `:565`). The swap matrix, measured
from the two call sites:

| Situation | `glidSrcMap` holds | `glid2SrcMap` holds | Cite |
|---|---|---|---|
| two-player game, start | 3999 | 3974 | `GliderPRO/Sources/Play.c:124-127` |
| one-player game, start | 3999 | **3976** (the foil sheet lives here permanently) | `GliderPRO/Sources/Play.c:132-135` |
| two-player game, `DeckGliderInFoil` | **3976** | **3963** | `GliderPRO/Sources/Player.c:1146-1152` |
| one-player game, `DeckGliderInFoil` | 3999 (unchanged) | 3976 (unchanged) | `GliderPRO/Sources/Player.c:1146` guards on `twoPlayerGame` |

All four sheets are 48x668 with identical frame geometry, so one mask (4999) serves all four. An
extractor emits four RGBA glider sheets from one mask.

### 6.3 All 116 initialised entries, group by group

Column meanings: **rect in sheet** is the computed `srcRects[what]` as `(l,t,r,b)`; entries marked
*template* are not valid sheet coordinates (they were never `QOffsetRect`-ed, or the art comes from
elsewhere) and exist only as a default size. **art PICT** is where the pixels really are. **alpha
source** is the §5 path. **draw helper** is the function `DrawARoomsObjects`
(`GliderPRO/Sources/ObjectDrawAll.c:23`) dispatches to; `(AddDynamicObject)` means the type is not
drawn into `backSrcMap` at all but registered with the per-frame animator instead.

**Blowers** (`InitSrcRects` group comment at `GliderPRO/Sources/StructuresInit2.c:308`)

| hex | constant | rect in sheet (l,t,r,b) | W x H | art PICT | alpha source | draw helper | notes |
|---|---|---|---|---|---|---|---|
| 0x01 | `kFloorVent` | (0,0,48,11) | 48 x 11 | **4000** (`blowerSrcMap` 48x402) | mask **5000** | `DrawSimpleBlowers` |  |
| 0x02 | `kCeilingVent` | (0,11,48,22) | 48 x 11 | **4000** (`blowerSrcMap` 48x402) | mask **5000** | `DrawSimpleBlowers` |  |
| 0x03 | `kFloorBlower` | (0,22,48,37) | 48 x 15 | **4000** (`blowerSrcMap` 48x402) | mask **5000** | `DrawSimpleBlowers` |  |
| 0x04 | `kCeilingBlower` | (0,37,48,52) | 48 x 15 | **4000** (`blowerSrcMap` 48x402) | mask **5000** | `DrawSimpleBlowers` |  |
| 0x05 | `kSewerGrate` | (0,52,48,69) | 48 x 17 | **4000** (`blowerSrcMap` 48x402) | mask **5000** | `DrawSimpleBlowers` |  |
| 0x06 | `kLeftFan` | (0,69,40,124) | 40 x 55 | **4000** (`blowerSrcMap` 48x402) | mask **5000** | `DrawSimpleBlowers` | tikiFlame[] overlays x40-48 |
| 0x07 | `kRightFan` | (0,124,40,179) | 40 x 55 | **4000** (`blowerSrcMap` 48x402) | mask **5000** | `DrawSimpleBlowers` |  |
| 0x08 | `kTaper` | (0,209,20,268) | 20 x 59 | **4000** (`blowerSrcMap` 48x402) | mask **5000** | `DrawSimpleBlowers` | + flame[0..4] 16x15 stride 15 @(32,179) |
| 0x09 | `kCandle` | (0,179,32,209) | 32 x 30 | **4000** (`blowerSrcMap` 48x402) | mask **5000** | `DrawSimpleBlowers` | + flame[0..4] 16x15 stride 15 @(32,179) |
| 0x0A | `kStubby` | (0,268,20,304) | 20 x 36 | **4000** (`blowerSrcMap` 48x402) | mask **5000** | `DrawSimpleBlowers` | + flame[0..4] 16x15 stride 15 @(32,179) |
| 0x0B | `kTiki` | (21,268,48,296) | 27 x 28 | **4000** (`blowerSrcMap` 48x402) | mask **5000** | `DrawTiki` | + tikiFlame[0..4] 8x10 stride 10 @(40,69); pole drawn procedurally |
| 0x0C | `kBBQ` | (0,0,64,33) *template* | 64 x 33 | **3988** (own) | white colour key (index 0) | `DrawPictSansWhiteObject` | + coals[0..3] 32x9 stride 9 @(0,304) from blower sheet |
| 0x0D | `kInvisBlower` | (0,0,24,24) *template* | 24 x 24 | - (never drawn) | n/a | `-` | hit/size template only |
| 0x0E | `kGrecoVent` | (0,340,48,358) | 48 x 18 | **4000** (`blowerSrcMap` 48x402) | mask **5000** | `DrawSimpleBlowers` |  |
| 0x0F | `kSewerBlower` | (0,390,32,402) | 32 x 12 | **4000** (`blowerSrcMap` 48x402) | mask **5000** | `DrawSimpleBlowers` |  |
| 0x10 | `kLiftArea` | (0,0,64,32) *template* | 64 x 32 | - (never drawn) | n/a | `-` | size overridden by data.a.distance/tall |

**Furniture** (`InitSrcRects` group comment at `GliderPRO/Sources/StructuresInit2.c:338`)

| hex | constant | rect in sheet (l,t,r,b) | W x H | art PICT | alpha source | draw helper | notes |
|---|---|---|---|---|---|---|---|
| 0x11 | `kTable` | (0,0,64,8) *template* | 64 x 8 | **4001** (`furnitureSrcMap` 64x278) | mask **5001** | `DrawTable` | art = tableSrc 64x22 @(0,0); rect is kTableThick=8 thick |
| 0x12 | `kShelf` | (0,0,64,6) *template* | 64 x 6 | **4001** (`furnitureSrcMap` 64x278) | mask **5001** | `DrawShelf` | art = shelfSrc 16x21 @(0,22); rect is kShelfThick=6 thick |
| 0x13 | `kCabinet` | (0,0,64,64) *template* | 64 x 64 | **4001** (`furnitureSrcMap` 64x278) | mask **5001** | `DrawCabinet` | art = hingeSrc 4x16 @(16,22), handleSrc 4x21 @(20,22) |
| 0x14 | `kFilingCabinet` | (0,0,74,107) *template* | 74 x 107 | **3995** (own) | none - fully opaque | `DrawPictObject` |  |
| 0x15 | `kWasteBasket` | (0,43,64,104) | 64 x 61 | **4001** (`furnitureSrcMap` 64x278) | mask **5001** | `DrawSimpleFurniture` |  |
| 0x16 | `kMilkCrate` | (0,104,64,162) | 64 x 58 | **4001** (`furnitureSrcMap` 64x278) | mask **5001** | `DrawSimpleFurniture` |  |
| 0x17 | `kCounter` | (0,0,128,64) *template* | 128 x 64 | - (procedural) | n/a | `DrawCounter` | no PICT pixels at all |
| 0x18 | `kDresser` | (0,0,128,64) *template* | 128 x 64 | **4001** (`furnitureSrcMap` 64x278) | mask **5001** | `DrawDresser` | art = knobSrc 8x8 @(24,22) srcCopy, leftFootSrc 16x16 @(32,22), rightFootSrc 16x16 @(48,22) |
| 0x19 | `kDeckTable` | (0,0,64,8) *template* | 64 x 8 | **4001** (`furnitureSrcMap` 64x278) | mask **5001** | `DrawDeckTable` | art = deckSrc 64x21 @(0,162) |
| 0x1A | `kStool` | (0,183,48,221) | 48 x 38 | **4001** (`furnitureSrcMap` 64x278) | mask **5001** | `DrawStool` | + procedural legs to kStoolBase 304 |
| 0x1B | `kTrunk` | (0,0,144,80) *template* | 144 x 80 | **3987** (own) | white colour key (index 0) | `DrawPictSansWhiteObject` |  |
| 0x1C | `kInvisObstacle` | (0,0,64,64) *template* | 64 x 64 | - (never drawn) | n/a | `-` | hit/size template only |
| 0x1D | `kManhole` | (0,0,123,22) *template* | 123 x 22 | **3967** (own) | white colour key (index 0) | `DrawPictSansWhiteObject` |  |
| 0x1E | `kBooks` | (0,0,64,51) *template* | 64 x 51 | **3964** (own) | white colour key (index 0) | `DrawPictSansWhiteObject` |  |
| 0x1F | `kInvisBounce` | (0,0,64,64) *template* | 64 x 64 | - (never drawn) | n/a | `-` | hit/size template only |

**Prizes** (`InitSrcRects` group comment at `GliderPRO/Sources/StructuresInit2.c:357`)

| hex | constant | rect in sheet (l,t,r,b) | W x H | art PICT | alpha source | draw helper | notes |
|---|---|---|---|---|---|---|---|
| 0x21 | `kRedClock` | (0,0,28,17) | 28 x 17 | **4002** (`bonusSrcMap` 88x378) | mask **5002** | `DrawRedClock` | + digits[0..10] 4x6 stride 6 @(28,0) |
| 0x22 | `kBlueClock` | (0,17,28,42) | 28 x 25 | **4002** (`bonusSrcMap` 88x378) | mask **5002** | `DrawBlueClock` | + ColorLine hands |
| 0x23 | `kYellowClock` | (0,42,28,70) | 28 x 28 | **4002** (`bonusSrcMap` 88x378) | mask **5002** | `DrawYellowClock` | + ColorLine hands |
| 0x24 | `kCuckoo` | (0,148,40,228) | 40 x 80 | **4002** (`bonusSrcMap` 88x378) | mask **5002** | `DrawCuckoo` | + pendulumSrc[0..2] 32x28 stride 28 @(56,186) |
| 0x25 | `kPaper` | (0,127,48,148) | 48 x 21 | **4002** (`bonusSrcMap` 88x378) | mask **5002** | `DrawSimplePrizes` |  |
| 0x26 | `kBattery` | (32,0,48,25) | 16 x 25 | **4002** (`bonusSrcMap` 88x378) | mask **5002** | `DrawSimplePrizes` |  |
| 0x27 | `kBands` | (20,70,48,93) | 28 x 23 | **4002** (`bonusSrcMap` 88x378) | mask **5002** | `DrawSimplePrizes` | bandRects[0..2] 16x6 stride 6 in bandsSrcMap 4007/5007 are the in-flight bands |
| 0x28 | `kGreaseRt` | (0,243,32,270) | 32 x 27 | **4002** (`bonusSrcMap` 88x378) | mask **5002** | `DrawGreaseRt` | frames greaseSrcRt[0..3] 32x27 @(0,243),(0,270),(0,297),(32,297) |
| 0x29 | `kGreaseLf` | (0,324,32,351) | 32 x 27 | **4002** (`bonusSrcMap` 88x378) | mask **5002** | `DrawGreaseLf` | frames greaseSrcLf[0..3] 32x27 @(0,324),(32,324),(0,351),(32,351) |
| 0x2A | `kFoil` | (0,228,55,243) | 55 x 15 | **4002** (`bonusSrcMap` 88x378) | mask **5002** | `DrawFoil` |  |
| 0x2B | `kInvisBonus` | (0,0,24,24) *template* | 24 x 24 | - (never drawn) | n/a | `-` | hit/size template only |
| 0x2C | `kStar` | (48,0,80,31) | 32 x 31 | **4002** (`bonusSrcMap` 88x378) | mask **5002** | `DrawSimplePrizes` | starSrc[0..5] 32x31 stride 31 @(48,0) |
| 0x2D | `kSparkle` | (0,70,20,89) | 20 x 19 | **4002** (`bonusSrcMap` 88x378) | mask **5002** | `(AddDynamicObject)` | sparkleSrc[0..4] 20x19; frames 2,3,4 @(0,70),(0,89),(0,108); [0]=[4], [1]=[3] |
| 0x2E | `kHelium` | (32,270,88,286) | 56 x 16 | **4002** (`bonusSrcMap` 88x378) | mask **5002** | `DrawSimplePrizes` |  |
| 0x2F | `kSlider` | (0,0,64,16) *template* | 64 x 16 | - (never drawn) | n/a | `-` | hit/size template only |

**Transport** (`InitSrcRects` group comment at `GliderPRO/Sources/StructuresInit2.c:385`)

| hex | constant | rect in sheet (l,t,r,b) | W x H | art PICT | alpha source | draw helper | notes |
|---|---|---|---|---|---|---|---|
| 0x31 | `kUpStairs` | (0,0,160,267) *template* | 160 x 267 | **3997** (own) | white colour key (index 0) | `DrawPictSansWhiteObject` |  |
| 0x32 | `kDownStairs` | (0,0,160,267) *template* | 160 x 267 | **3996** (own) | none - fully opaque | `DrawPictObject` |  |
| 0x33 | `kMailboxLf` | (0,0,94,80) *template* | 94 x 80 | **3986** (own) | mask **3904** (temp GWorld) | `DrawMailboxLeft` | anchored to kMailboxBase 296 |
| 0x34 | `kMailboxRt` | (0,0,94,80) *template* | 94 x 80 | **3985** (own) | mask **3903** (temp GWorld) | `DrawMailboxRight` | anchored to kMailboxBase 296 |
| 0x35 | `kFloorTrans` | (0,1,56,16) | 56 x 15 | **4008** (`transSrcMap` 56x32) | mask **5008** | `DrawSimpleTransport` |  |
| 0x36 | `kCeilingTrans` | (0,16,56,31) | 56 x 15 | **4008** (`transSrcMap` 56x32) | mask **5008** | `DrawSimpleTransport` |  |
| 0x37 | `kDoorInLf` | (0,0,144,322) *template* | 144 x 322 | **3984** (own) | white colour key (index 0) | `DrawPictSansWhiteObject` |  |
| 0x38 | `kDoorInRt` | (0,0,144,322) *template* | 144 x 322 | **3983** (own) | white colour key (index 0) | `DrawPictSansWhiteObject` |  |
| 0x39 | `kDoorExRt` | (0,0,16,322) *template* | 16 x 322 | **3982** (own) | none - fully opaque | `DrawPictObject` |  |
| 0x3A | `kDoorExLf` | (0,0,16,322) *template* | 16 x 322 | **3981** (own) | none - fully opaque | `DrawPictObject` |  |
| 0x3B | `kWindowInLf` | (0,0,20,170) *template* | 20 x 170 | **3980** (own) | white colour key (index 0) | `DrawPictSansWhiteObject` |  |
| 0x3C | `kWindowInRt` | (0,0,20,170) *template* | 20 x 170 | **3979** (own) | white colour key (index 0) | `DrawPictSansWhiteObject` |  |
| 0x3D | `kWindowExRt` | (0,0,16,170) *template* | 16 x 170 | **3977** (own) | none - fully opaque | `DrawPictObject` |  |
| 0x3E | `kWindowExLf` | (0,0,16,170) *template* | 16 x 170 | **3978** (own) | none - fully opaque | `DrawPictObject` |  |
| 0x3F | `kInvisTrans` | (0,0,64,32) *template* | 64 x 32 | - (never drawn) | n/a | `-` | hit/size template only |
| 0x40 | `kDeluxeTrans` | (0,0,64,64) *template* | 64 x 64 | - (never drawn) | n/a | `-` | hit/size template only |

**Switch** (`InitSrcRects` group comment at `GliderPRO/Sources/StructuresInit2.c:404`)

| hex | constant | rect in sheet (l,t,r,b) | W x H | art PICT | alpha source | draw helper | notes |
|---|---|---|---|---|---|---|---|
| 0x41 | `kLightSwitch` | (0,0,15,24) *template* | 15 x 24 | **4003** (`switchSrcMap` 32x104) | **no mask** (opaque `srcCopy`) | `DrawLightSwitch` | art = lightSwitchSrc[0] 15x24 @(0,0) on / [1] @(16,0) off |
| 0x42 | `kMachineSwitch` | (0,48,16,72) *template* | 16 x 24 | **4003** (`switchSrcMap` 32x104) | **no mask** (opaque `srcCopy`) | `DrawMachineSwitch` | art = machineSwitchSrc[0] 16x24 @(0,24) / [1] @(16,24); rect y-offset 48 disagrees |
| 0x43 | `kThermostat` | (0,48,15,72) *template* | 15 x 24 | **4003** (`switchSrcMap` 32x104) | **no mask** (opaque `srcCopy`) | `DrawThermostat` | art = thermostatSrc[0] 15x24 @(0,48) / [1] @(16,48) |
| 0x44 | `kPowerSwitch` | (0,72,8,80) *template* | 8 x 8 | **4003** (`switchSrcMap` 32x104) | **no mask** (opaque `srcCopy`) | `DrawPowerSwitch` | art = powerSrc[0] 8x8 @(0,72) / [1] @(8,72) |
| 0x45 | `kKnifeSwitch` | (0,80,16,104) *template* | 16 x 24 | **4003** (`switchSrcMap` 32x104) | **no mask** (opaque `srcCopy`) | `DrawKnifeSwitch` | art = knifeSwitchSrc[0] 16x24 @(0,80) / [1] @(16,80) |
| 0x46 | `kInvisSwitch` | (0,0,12,12) *template* | 12 x 12 | - (never drawn) | n/a | `-` | hit/size template only |
| 0x47 | `kTrigger` | (0,0,12,12) *template* | 12 x 12 | - (never drawn) | n/a | `-` | hit/size template only |
| 0x48 | `kLgTrigger` | (0,0,48,48) *template* | 48 x 48 | - (never drawn) | n/a | `-` | hit/size template only |
| 0x49 | `kSoundTrigger` | (0,0,32,32) *template* | 32 x 32 | - (never drawn) | n/a | `-` | hit/size template only |

**Lights** (`InitSrcRects` group comment at `GliderPRO/Sources/StructuresInit2.c:418`)

| hex | constant | rect in sheet (l,t,r,b) | W x H | art PICT | alpha source | draw helper | notes |
|---|---|---|---|---|---|---|---|
| 0x51 | `kCeilingLight` | (0,0,64,20) | 64 x 20 | **4004** (`lightSrcMap` 72x126) | mask **5004** | `DrawSimpleLight` |  |
| 0x52 | `kLightBulb` | (0,20,16,48) | 16 x 28 | **4004** (`lightSrcMap` 72x126) | mask **5004** | `DrawSimpleLight` |  |
| 0x53 | `kTableLamp` | (16,20,64,90) | 48 x 70 | **4004** (`lightSrcMap` 72x126) | mask **5004** | `DrawSimpleLight` |  |
| 0x54 | `kHipLamp` | (0,0,72,276) *template* | 72 x 276 | **3994** (own) | white colour key (index 0) | `DrawPictSansWhiteObject` |  |
| 0x55 | `kDecoLamp` | (0,0,64,212) *template* | 64 x 212 | **3993** (own) | white colour key (index 0) | `DrawPictSansWhiteObject` |  |
| 0x56 | `kFlourescent` | (0,0,64,12) *template* | 64 x 12 | **4004** (`lightSrcMap` 72x126) | mask **5004** | `DrawFlourescent` | art = flourescentSrc1 16x12 @(0,78) off / flourescentSrc2 @(0,90) on, tiled |
| 0x57 | `kTrackLight` | (0,0,64,24) *template* | 64 x 24 | **4004** (`lightSrcMap` 72x126) | mask **5004** | `DrawTrackLight` | art = trackLightSrc[0..2] 24x24 stride 24 in x @(0,102); kTrackLightSpacing 64 |
| 0x58 | `kInvisLight` | (0,0,16,16) *template* | 16 x 16 | - (never drawn) | n/a | `-` | hit/size template only |

**Appliances** (`InitSrcRects` group comment at `GliderPRO/Sources/StructuresInit2.c:430`)

| hex | constant | rect in sheet (l,t,r,b) | W x H | art PICT | alpha source | draw helper | notes |
|---|---|---|---|---|---|---|---|
| 0x61 | `kShredder` | (0,0,73,22) | 73 x 22 | **4005** (`applianceSrcMap` 80x269) | mask **5005** | `DrawSimpleAppliance` | shredSrcMap 4010/5010 40x35 holds the shredded-paper animation |
| 0x62 | `kToaster` | (0,22,48,49) | 48 x 27 | **4005** (`applianceSrcMap` 80x269) | mask **5005** | `DrawSimpleAppliance` | toastSrcMap 4009/5009 32x174 holds breadSrc[0..5] 32x29 stride 29 |
| 0x63 | `kMacPlus` | (0,49,48,107) | 48 x 58 | **4005** (`applianceSrcMap` 80x269) | mask **5005** | `DrawMacPlus` | + plusScreen1 32x22 @(48,127) off / plusScreen2 @(48,149) on, srcCopy at +10,+7 |
| 0x64 | `kGuitar` | (0,0,64,172) *template* | 64 x 172 | **3991** (own) | white colour key (index 0) | `DrawPictSansWhiteObject` |  |
| 0x65 | `kTV` | (0,0,92,77) *template* | 92 x 77 | **3992** (own) | mask **3912** (temp GWorld) | `DrawTV` | + tvScreen1 64x49 @(0,171) off / tvScreen2 @(0,220) on, from applianceSrcMap |
| 0x66 | `kCoffee` | (0,107,43,171) | 43 x 64 | **4005** (`applianceSrcMap` 80x269) | mask **5005** | `DrawCoffee` | + coffeeLight1 8x4 @(72,171) off / coffeeLight2 @(72,175) on, srcCopy at +32,+57 |
| 0x67 | `kOutlet` | (64,22,80,46) | 16 x 24 | **4005** (`applianceSrcMap` 80x269) | mask **5005** | `DrawOutlet` | outletSrc[0..3] 16x24 stride 24 @(64,22) is the spark animation |
| 0x68 | `kVCR` | (0,0,96,22) *template* | 96 x 22 | **3990** (own) | mask **3913** (temp GWorld) | `DrawVCR` | + vcrTime1 16x4 @(64,179) / vcrTime2 @(64,183) from applianceSrcMap |
| 0x69 | `kStereo` | (0,0,128,53) *template* | 128 x 53 | **3989** (own) | mask **3914** (temp GWorld) | `DrawStereo` | + stereoLight1 4x1 @(68,171) / stereoLight2 @(68,172) |
| 0x6A | `kMicrowave` | (0,0,92,59) *template* | 92 x 59 | **3971** (own) | mask **3915** (temp GWorld) | `DrawMicrowave` | + microOff 16x35 @(64,187) / microOn @(64,222) |
| 0x6B | `kCinderBlock` | (0,0,40,62) *template* | 40 x 62 | **3960** (own) | white colour key (index 0) | `DrawPictSansWhiteObject` |  |
| 0x6C | `kFlowerBox` | (0,0,80,32) *template* | 80 x 32 | **3959** (own) | white colour key (index 0) | `DrawPictSansWhiteObject` |  |
| 0x6D | `kCDs` | (48,22,64,52) | 16 x 30 | **4005** (`applianceSrcMap` 80x269) | mask **5005** | `DrawSimpleAppliance` |  |
| 0x6E | `kCustomPict` | (0,0,72,34) *template* | 72 x 34 | **data.g.height** (own) | white colour key (index 0) | `DrawCustPictSansWhite` | default art PICT 10000 is exactly 72x34 |

**Enemies** (`InitSrcRects` group comment at `GliderPRO/Sources/StructuresInit2.c:450`)

| hex | constant | rect in sheet (l,t,r,b) | W x H | art PICT | alpha source | draw helper | notes |
|---|---|---|---|---|---|---|---|
| 0x71 | `kBalloon` | (0,0,24,30) | 24 x 30 | **4011** (`balloonSrcMap` 24x240) | mask **5011** | `(AddDynamicObject)` | balloonSrc[0..7] 24x30 stride 30 |
| 0x72 | `kCopterLf` | (0,0,32,30) | 32 x 30 | **4012** (`copterSrcMap` 32x300) | mask **5012** | `(AddDynamicObject)` | copterSrc[0..9] 32x30 stride 30 |
| 0x73 | `kCopterRt` | (0,0,32,30) | 32 x 30 | **4012** (`copterSrcMap` 32x300) | mask **5012** | `(AddDynamicObject)` | copterSrc[0..9] 32x30 stride 30 |
| 0x74 | `kDartLf` | (0,0,64,19) | 64 x 19 | **4013** (`dartSrcMap` 64x76) | mask **5013** | `(AddDynamicObject)` | dartSrc[0..3] 64x19 stride 19 |
| 0x75 | `kDartRt` | (0,0,64,19) | 64 x 19 | **4013** (`dartSrcMap` 64x76) | mask **5013** | `(AddDynamicObject)` | dartSrc[0..3] 64x19 stride 19 |
| 0x76 | `kBall` | (0,0,32,32) | 32 x 32 | **4014** (`ballSrcMap` 32x64) | mask **5014** | `(AddDynamicObject)` | ballSrc[0..1] 32x32 stride 32 |
| 0x77 | `kDrip` | (0,0,16,12) | 16 x 12 | **4015** (`dripSrcMap` 16x72) | mask **5015** | `DrawDrip` | static draw uses dripSrc[3] @(0,36), NOT srcRects[kDrip]; dripSrc[0..5] 16x12 stride 12 |
| 0x78 | `kFish` | (0,0,36,33) | 36 x 33 | **4016** (`enemySrcMap` 36x33) | mask **5016** | `DrawFish` | static draw reads enemySrcMap 4016/5016; the 8-frame fishSrcMap 4017/5017 16x16 stride 16 is animation-only |
| 0x79 | `kCobweb` | (0,0,54,45) *template* | 54 x 45 | **3958** (own) | mask **3927** (temp GWorld) | `DrawPictWithMaskObject` |  |

**Clutter** (`InitSrcRects` group comment at `GliderPRO/Sources/StructuresInit2.c:460`)

| hex | constant | rect in sheet (l,t,r,b) | W x H | art PICT | alpha source | draw helper | notes |
|---|---|---|---|---|---|---|---|
| 0x81 | `kOzma` | (0,0,102,92) *template* | 102 x 92 | **3975** (own) | none - fully opaque | `DrawPictObject` |  |
| 0x82 | `kMirror` | (0,0,64,64) *template* | 64 x 64 | - (procedural) | n/a | `DrawMirror` | four ColorFrameRect insets, no PICT |
| 0x83 | `kMousehole` | (0,0,10,11) | 10 x 11 | **4018** (`clutterSrcMap` 128x69) | mask **5018** | `DrawSimpleClutter` |  |
| 0x84 | `kFireplace` | (0,0,180,142) *template* | 180 x 142 | **3973** (own) | white colour key (index 0) | `DrawPictSansWhiteObject` |  |
| 0x86 | `kWallWindow` | (0,0,64,80) *template* | 64 x 80 | - (procedural) | n/a | `DrawWallWindow` | kWindowSillThick 7, no PICT |
| 0x87 | `kBear` | (0,0,56,58) *template* | 56 x 58 | **3972** (own) | white colour key (index 0) | `DrawPictSansWhiteObject` |  |
| 0x88 | `kCalendar` | (0,0,63,92) *template* | 63 x 92 | **3970** (own) | none - fully opaque | `DrawCalendar` | + month name from STR# 1005 at +((64-w)/2), +55 |
| 0x89 | `kVase1` | (0,0,36,45) *template* | 36 x 45 | **3969** (own) | white colour key (index 0) | `DrawPictSansWhiteObject` |  |
| 0x8A | `kVase2` | (0,0,35,57) *template* | 35 x 57 | **3968** (own) | white colour key (index 0) | `DrawPictSansWhiteObject` |  |
| 0x8B | `kBulletin` | (0,0,80,58) *template* | 80 x 58 | **3966** (own) | none - fully opaque | `DrawBulletin` |  |
| 0x8C | `kCloud` | (0,0,128,30) *template* | 128 x 30 | **3965** (own) | mask **3921** (temp GWorld) | `DrawPictWithMaskObject` |  |
| 0x8D | `kFaucet` | (0,51,56,69) | 56 x 18 | **4018** (`clutterSrcMap` 128x69) | mask **5018** | `DrawSimpleClutter` |  |
| 0x8E | `kRug` | (0,0,144,18) *template* | 144 x 18 | **3962** (own) | white colour key (index 0) | `DrawPictSansWhiteObject` |  |
| 0x8F | `kChimes` | (0,0,28,74) *template* | 28 x 74 | **3961** (own) | white colour key (index 0) | `DrawPictSansWhiteObject` |  |

### 6.4 The 28 slots `InitSrcRects` never touches

`srcRects` is allocated with `NewPtr`, not `NewPtrClear` (`GliderPRO/Sources/StructuresInit2.c:271`),
so the 28 slots `InitSrcRects` skips contain **uninitialised Mac heap bytes** for the whole run.
They are:

| `what` values | Why unassigned |
|---|---|
| `0x00` | not a valid object code; `kObjectIsEmpty` is `-1`, not 0 (`GliderPRO/Headers/GliderDefines.h:526`), and `what` is a `short` (`GliderPRO/Headers/GliderStructs.h:92`) so `-1` is representable |
| `0x20` | gap between the Furniture block (ends `0x1F`) and Prizes (starts `0x21`) |
| `0x30` | gap between Prizes (ends `0x2F`) and Transport (starts `0x31`) |
| `0x4A` `0x4B` `0x4C` `0x4D` `0x4E` `0x4F` `0x50` | gap between Switch (ends `0x49`) and Lights (starts `0x51`) |
| `0x59` `0x5A` `0x5B` `0x5C` `0x5D` `0x5E` `0x5F` `0x60` | gap between Lights (ends `0x58`) and Appliances (starts `0x61`) |
| `0x6F` `0x70` | gap between Appliances (ends `0x6E`) and Enemies (starts `0x71`) |
| `0x7A` `0x7B` `0x7C` `0x7D` `0x7E` `0x7F` `0x80` | gap between Enemies (ends `0x79`) and Clutter (starts `0x81`) |
| **`0x85` `kFlower`** | **a defined, shipped, drawable object type whose slot is nonetheless skipped** — §6.5 |

27 + 1 = 28, and 116 + 28 = 144 = `kNumSrcRects`. The seven contiguous gaps (`0x00`, `0x20`, `0x30`,
`0x4A`-`0x50`, `0x59`-`0x60`, `0x6F`-`0x70`, `0x7A`-`0x80`) are exactly the undefined `what` codes
catalogued in `docs/analysis/object-taxonomy.md` §5.1 / `docs/analysis/house-format.md:1202-1207`; that
document also establishes by census that **no undefined code occurs in any of the 4070 rooms of the 22
shipped houses**, so the garbage is never read in practice.

Two robustness details a port should replicate deliberately rather than by accident:

* **The play-mode and edit-mode paths disagree about undefined codes.** `GetObjectRect`
  (`GliderPRO/Sources/ObjectRects.c:32-273`) has **no `default:` arm**, so an object with
  `what == 0x20` leaves `*itsRect` completely unwritten — the caller's uninitialised stack rect —
  rather than reading `srcRects[0x20]`. `DrawARoomsObjects`
  (`GliderPRO/Sources/ObjectDrawAll.c:23-965`) likewise has no `default:`, so nothing is drawn. But
  the editor's `GetThisRoomsObjRects` (`GliderPRO/Sources/ObjectEdit.c:2049-2341`) **does** have one,
  at `:2334-2336`: `QSetRect(&roomObjectRects[i], -2, -2, -1, -1)` — a deliberately degenerate
  off-screen rect. Neither `ObjectRects.c`, `ObjectDrawAll.c` nor `ObjectDraw2.c` contains the token
  `default` anywhere. A port should use the editor's behaviour (an explicit empty rect) everywhere.
* `GliderPRO/Sources/ObjectRects.c:68-71` is **unreachable dead code**: four statements sitting between
  the `break` that ends the `kLiftArea` arm (`:66`) and the next `case` label (`:73`). They are a
  copy of the blower arm's body. A compiler with dead-code warnings will flag it; a porter transcribing
  arm-by-arm may accidentally attach them to `kLiftArea` and break lift-area sizing, which must come
  from `data.a.distance` x `data.a.tall * 2` (`:64`).

A Go port should make the array a fixed `[144]image.Rectangle` with an explicit `valid [144]bool`
companion (or a `map[uint8]Rect`), and treat a lookup of any of the 28 as a hard error in debug builds.

### 6.5 The special case: `srcRects[kFlower]` (0x85) is never assigned, and never read

This is the one entry that looks like a bug and is not. Walking the Clutter block of `InitSrcRects`:

```c
QSetRect(&srcRects[kFireplace],  0, 0, 180, 142);   // 0x84   StructuresInit2.c:463
QSetRect(&srcRects[kWallWindow], 0, 0,  64,  80);   // 0x86   StructuresInit2.c:464
```

`kFlower` = `0x85` (`GliderPRO/Headers/GliderDefines.h:425`) is skipped between them. There is no
`QSetRect(&srcRects[kFlower], ...)` anywhere in the program — verified by scanning every one of the
367 `srcRects[` occurrences across all 67 `.c` files in `GliderPRO/Sources/`. Their distribution:
`ObjectAdd.c` 83, `StructuresInit2.c` 159 (the initialiser itself), `ObjectRects.c` 41,
`ObjectDraw2.c` 26, `ObjectEdit.c` 13, `ObjectDraw.c` 10, `HouseLegal.c` 5, everywhere else 0.

It does not matter, because **nothing ever reads `srcRects[kFlower]`**. Flowers are sized and drawn
from `flowerSrc[]` instead. The six sites that would otherwise consult the atlas all route around it:

| Site | Code | What it uses instead |
|---|---|---|
| draw | `DrawFlower(&itsRect, thisObject.data.i.pict)` (`GliderPRO/Sources/ObjectDrawAll.c:926`) → `DrawFlower` (`GliderPRO/Sources/ObjectDraw2.c:1028-1034`) | `&flowerSrc[which]` as **both** the source rect and the mask rect, against `clutterSrcMap` / `clutterMaskMap` |
| runtime size | `GetObjectRect` clutter arm (`GliderPRO/Sources/ObjectRects.c:255-271`) | `*itsRect = who->data.i.bounds` — the rect stored in the house file |
| editor size | `GetThisRoomsObjRects` clutter arm (`GliderPRO/Sources/ObjectEdit.c:2316-2333`) | `data.i.bounds` |
| editor create | `AddNewObject` (`GliderPRO/Sources/ObjectAdd.c:739-747`) | `newRect = flowerSrc[wasFlower]` centred on the click point, then stored in `data.i.bounds`; `data.i.pict = wasFlower` |
| editor resize | flower Info dialog (`GliderPRO/Sources/ObjectInfo.c:2328-2333` and the duplicate at `:2360-2365`) | `bounds.right = bounds.left + RectWide(&flowerSrc[flower])`, `bounds.top = bounds.bottom - RectTall(&flowerSrc[flower])` — width grows rightward, height grows **upward** from a fixed baseline |
| hot spots | `CreateActiveRects` clutter arm (`GliderPRO/Sources/ObjectRects.c:1035-1049`) | bare `break` — flowers have no active rect |

`DrawFlower`'s signature is `void DrawFlower (Rect *theRect, short which)`: no `what` parameter at
all, so the atlas index is not even in scope. `which` is `data.i.pict`, clamped to `0..5` by the
6-radio-button dialog: `kRadioFlower1` = **6** and `kRadioFlower6` = **11**
(`GliderPRO/Sources/ObjectInfo.c:58-59`) are DITL item numbers, and the dialog round-trips through
`flower = data.i.pict + kRadioFlower1` / `flower -= kRadioFlower1`
(`GliderPRO/Sources/ObjectInfo.c:2312-2313`, `:2324`, `:2345-2348`). `kNumFlowers` = **6**
(`GliderPRO/Headers/GliderDefines.h:458`).

The six rects, transcribed verbatim from `GliderPRO/Sources/StructuresInit2.c:72-88` and converted to
`(l,t,r,b)` inside the 128x69 clutter sheet:

| `flowerSrc[i]` | `QSetRect` W x H | `QOffsetRect` | resulting rect (l,t,r,b) | W x H | extracted appearance |
|---|---|---|---|---|---|
| 0 | 10 x 28 | (0, 23) | (0,23,10,51) | 10 x 28 | small yellow bud on a stem |
| 1 | 24 x 35 | (10, 16) | (10,16,34,51) | 24 x 35 | pink tulip |
| 2 | 34 x 35 | (34, 16) | (34,16,68,51) | 34 x 35 | purple iris |
| 3 | 27 x 23 | (68, 14) | (68,14,95,37) | 27 x 23 | cluster of violets |
| 4 | 27 x 14 | (68, 37) | (68,37,95,51) | 27 x 14 | white daisies |
| 5 | 32 x 51 | (95, 0) | (95,0,127,51) | 32 x 51 | sunflower |

Note the widths are **irregular** — this is not a strided frame strip, and `flowerSrc[3]` and
`flowerSrc[4]` are stacked vertically in the same 27-pixel column. Note also that
`flowerSrc[5].right` is 127, one pixel short of the sheet's 128, and that all six sit above y=51 while
the sheet is 69 rows tall: rows 51-68 hold `srcRects[kFaucet]` = (0,51,56,69).

Verification that all six are real art rather than five plus one blank: running the §7 pipeline over
`flowerSrc[0..5]` and stitching the crops 4x-scaled side by side produced six visually distinct
plants, with mask coverage 32.9%, 30.1%, 22.9%, 39.5%, 30.2% and 45.5% of their bounding boxes
respectively (measured against mask PICT 5018). So
`data.i.pict` genuinely selects among six sprites, and a port that only implements one flower will be
visibly wrong.

**What a port must do:** give `kFlower` no atlas entry at all. Size it from
`flowerSrc[data.i.pict]` and, for a *new* flower, seed `data.i.pict` with `RandomInt(kNumFlowers)`
unless shift is held (`GliderPRO/Sources/ObjectAdd.c:740-742` — `wasFlower` is a file-scope global,
`GliderPRO/Sources/ObjectAdd.c:42`, so shift-click repeats the previous choice).

### 6.6 The named sub-rect manifest — 58 arrays, 197 cells

These are the sprite cells the `tmpl`-kind objects and every animation actually read. They are all
plain file-scope globals, and — unlike `srcRects` — they are **fixed-size arrays, not heap
allocations**, so a Go port can make them package-level `[N]image.Rectangle` values. Their
declarations are spread across seven files:

| File | Arrays declared |
|---|---|
| `GliderPRO/Sources/Objects.c:19-72` | all the `*SrcRect` bounds and `*SrcMap`/`*MaskMap` GWorld pointers, plus `flame`, `tikiFlame`, `coals`, `tableSrc`..`deckSrc`, `starSrc`, `sparkleSrc`, `digits`, `pendulumSrc`, `greaseSrcRt/Lf`, the five switch pairs, `flourescentSrc1/2`, `trackLightSrc`, the 12 appliance overlays, `outletSrc`, `balloonSrc`, `copterSrc`, `dartSrc`, `ballSrc`, `dripSrc`, `fishSrc`, `flowerSrc`, and `Rect *srcRects` itself at `:71` |
| `GliderPRO/Sources/Player.c:44-48` | `shadowSrcRect`, `shadowSrc[kNumShadowSrcRects]`, `gliderSrc[kNumGliderSrcRects]` |
| `GliderPRO/Sources/DynamicMaps.c:30` | `pointsSrc[15]` |
| `GliderPRO/Sources/Dynamics.c:20` | `breadSrc[kNumBreadPicts]` |
| `GliderPRO/Sources/RubberBands.c:23` | `bandRects[3]` |
| `GliderPRO/Sources/Scoreboard.c:37-38` | `badgesBlankRects[4]`, `badgesBadgesRects[4]`, `badgesDestRects[4]` |
| `GliderPRO/Sources/ObjectEdit.c:30` | `leftStartGliderSrc`, `rightStartGliderSrc` |

They are initialised by the same `Init*` functions that create the sheets, so their coordinates are
*inside the sheet whose surface the `Init*` function just built*. Cell geometry is printed below as
first-cell origin plus stride so a porter can generate each array with one loop, matching the
original's `for` loops.

`rendering.md` §3.x and §12.2 print the same arrays as fully expanded `(t,l,b,r)` rect lists; this
table is the loop-parameter form, and the two agree.

| array | sheet | N | cell W x H | first cell origin (x,y) | stride | N governed by | cite |
|---|---|---|---|---|---|---|---|
| `gliderSrc[0..20]` | glider 48x668 | 21 | 48 x 20 | (0,0) | (0,+20) | literal `i <= 20` | `GliderPRO/Sources/StructuresInit.c:191-195` |
| `gliderSrc[21..28]` | glider 48x668 | 8 | 48 x 26 | (0,420) | (0,+26) | literal `i <= 28` | `GliderPRO/Sources/StructuresInit.c:196-200` |
| `gliderSrc[29]` | glider 48x668 | 1 | 48 x 20 | (0,628) | — | — | `GliderPRO/Sources/StructuresInit.c:202-203` |
| `gliderSrc[30]` | glider 48x668 | 1 | 48 x 20 | (0,648) | — | — | `GliderPRO/Sources/StructuresInit.c:204-205` |
| `shadowSrc[0..1]` | shadow 48x18 | 2 | 48 x 9 | (0,0) | (0,+9) | `kNumShadowSrcRects` 2 | `GliderPRO/Sources/StructuresInit.c:216-220` |
| `bandRects[0..2]` | bands 16x18 | 3 | 16 x 6 | (0,0) | (0,+6) | literal 3 | `GliderPRO/Sources/StructuresInit.c:231-235` |
| `flame[0..4]` | blower 48x402 | 5 | 16 x 15 | (32,179) | (0,+15) | `kNumCandleFlames` 5 | `GliderPRO/Sources/StructuresInit.c:262-266` |
| `tikiFlame[0..4]` | blower 48x402 | 5 | 8 x 10 | (40,69) | (0,+10) | `kNumTikiFlames` 5 | `GliderPRO/Sources/StructuresInit.c:268-272` |
| `coals[0..3]` | blower 48x402 | 4 | 32 x 9 | (0,304) | (0,+9) | `kNumBBQCoals` 4 | `GliderPRO/Sources/StructuresInit.c:274-278` |
| `leftStartGliderSrc` | blower 48x402 | 1 | 48 x 16 | (0,358) | — | — | `GliderPRO/Sources/StructuresInit.c:280-281` |
| `rightStartGliderSrc` | blower 48x402 | 1 | 48 x 16 | (0,374) | — | — | `GliderPRO/Sources/StructuresInit.c:283-284` |
| `tableSrc` | furniture 64x278 | 1 | 64 x 22 | (0,0) | — | — | `GliderPRO/Sources/StructuresInit.c:310-311` |
| `shelfSrc` | furniture 64x278 | 1 | **16** x 21 | (0,22) | — | — | `GliderPRO/Sources/StructuresInit.c:313-314` |
| `hingeSrc` | furniture 64x278 | 1 | 4 x 16 | (16,22) | — | — | `GliderPRO/Sources/StructuresInit.c:316-317` |
| `handleSrc` | furniture 64x278 | 1 | 4 x 21 | (20,22) | — | — | `GliderPRO/Sources/StructuresInit.c:319-320` |
| `knobSrc` | furniture 64x278 | 1 | 8 x 8 | (24,22) | — | — | `GliderPRO/Sources/StructuresInit.c:322-323` |
| `leftFootSrc` | furniture 64x278 | 1 | 16 x 16 | (32,22) | — | — | `GliderPRO/Sources/StructuresInit.c:325-326` |
| `rightFootSrc` | furniture 64x278 | 1 | 16 x 16 | (48,22) | — | — | `GliderPRO/Sources/StructuresInit.c:328-329` |
| `deckSrc` | furniture 64x278 | 1 | 64 x 21 | (0,162) | — | — | `GliderPRO/Sources/StructuresInit.c:331-332` |
| `digits[0..10]` | bonus 88x378 | 11 | 4 x 6 | (28,0) | (0,+6) | literal 11 | `GliderPRO/Sources/StructuresInit.c:359-363` |
| `pendulumSrc[0..2]` | bonus 88x378 | 3 | 32 x 28 | (56,186) | (0,+28) | `kNumPendulums` 3 | `GliderPRO/Sources/StructuresInit.c:365-369` |
| `greaseSrcRt[0..3]` | bonus 88x378 | 4 | 32 x 27 | **irregular** | — | literal x4 | `GliderPRO/Sources/StructuresInit.c:371-378` |
| `greaseSrcLf[0..3]` | bonus 88x378 | 4 | 32 x 27 | **irregular** | — | literal x4 | `GliderPRO/Sources/StructuresInit.c:380-387` |
| `starSrc[0..5]` | bonus 88x378 | 6 | 32 x 31 | (48,0) | (0,+31) | literal 6 | `GliderPRO/Sources/StructuresInit.c:389-393` |
| `sparkleSrc[2..4]` | bonus 88x378 | 3 | 20 x 19 | (0,70) | (0,+19) | literal 3 | `GliderPRO/Sources/StructuresInit.c:395-399` |
| `sparkleSrc[0..1]` | bonus 88x378 | 2 | 20 x 19 | **aliases** | — | copies of `[4]`, `[3]` | `GliderPRO/Sources/StructuresInit.c:400-401` |
| `pointsSrc[0..14]` | points 24x120 | 15 | 24 x 8 | (0,0) | (0,+8) | literal 15 | `GliderPRO/Sources/StructuresInit.c:412-416` |
| `lightSwitchSrc[0..1]` | switch 32x104 | 2 | 15 x 24 | (0,0) | (+16,0) | literal x2 | `GliderPRO/Sources/StructuresInit.c:460-463` |
| `machineSwitchSrc[0..1]` | switch 32x104 | 2 | 16 x 24 | (0,24) | (+16,0) | literal x2 | `GliderPRO/Sources/StructuresInit.c:465-468` |
| `thermostatSrc[0..1]` | switch 32x104 | 2 | 15 x 24 | (0,48) | (+16,0) | literal x2 | `GliderPRO/Sources/StructuresInit.c:470-473` |
| `powerSrc[0..1]` | switch 32x104 | 2 | 8 x 8 | (0,72) | (+8,0) | literal x2 | `GliderPRO/Sources/StructuresInit.c:475-478` |
| `knifeSwitchSrc[0..1]` | switch 32x104 | 2 | 16 x 24 | (0,80) | (+16,0) | literal x2 | `GliderPRO/Sources/StructuresInit.c:480-483` |
| `flourescentSrc1` | light 72x126 | 1 | 16 x 12 | (0,78) | — | off state | `GliderPRO/Sources/StructuresInit.c:510-511` |
| `flourescentSrc2` | light 72x126 | 1 | 16 x 12 | (0,90) | — | on state | `GliderPRO/Sources/StructuresInit.c:513-514` |
| `trackLightSrc[0..2]` | light 72x126 | 3 | 24 x 24 | (0,102) | (+24,0) | `kNumTrackLights` 3 | `GliderPRO/Sources/StructuresInit.c:516-521` |
| `plusScreen1` | appliance 80x269 | 1 | 32 x 22 | (48,127) | — | off state | `GliderPRO/Sources/StructuresInit.c:565-566` |
| `plusScreen2` | appliance 80x269 | 1 | 32 x 22 | (48,149) | — | on state | `GliderPRO/Sources/StructuresInit.c:567-568` |
| `tvScreen1` | appliance 80x269 | 1 | 64 x 49 | (0,171) | — | off state | `GliderPRO/Sources/StructuresInit.c:570-571` |
| `tvScreen2` | appliance 80x269 | 1 | 64 x 49 | (0,220) | — | on state | `GliderPRO/Sources/StructuresInit.c:572-573` |
| `coffeeLight1` | appliance 80x269 | 1 | 8 x 4 | (72,171) | — | off state | `GliderPRO/Sources/StructuresInit.c:575-576` |
| `coffeeLight2` | appliance 80x269 | 1 | 8 x 4 | (72,175) | — | on state | `GliderPRO/Sources/StructuresInit.c:577-578` |
| `outletSrc[0..3]` | appliance 80x269 | 4 | 16 x 24 | (64,22) | (0,+24) | `kNumOutletPicts` 4 | `GliderPRO/Sources/StructuresInit.c:580-584` |
| `vcrTime1` | appliance 80x269 | 1 | 16 x 4 | (64,179) | — | off state | `GliderPRO/Sources/StructuresInit.c:592-593` |
| `vcrTime2` | appliance 80x269 | 1 | 16 x 4 | (64,183) | — | on state | `GliderPRO/Sources/StructuresInit.c:594-595` |
| `stereoLight1` | appliance 80x269 | 1 | **4 x 1** | (68,171) | — | off state | `GliderPRO/Sources/StructuresInit.c:597-598` |
| `stereoLight2` | appliance 80x269 | 1 | **4 x 1** | (68,172) | — | on state | `GliderPRO/Sources/StructuresInit.c:599-600` |
| `microOn` | appliance 80x269 | 1 | 16 x 35 | (64,222) | — | on state | `GliderPRO/Sources/StructuresInit.c:602-603` |
| `microOff` | appliance 80x269 | 1 | 16 x 35 | (64,187) | — | off state | `GliderPRO/Sources/StructuresInit.c:604-605` |
| `breadSrc[0..5]` | toast 32x174 | 6 | 32 x 29 | (0,0) | (0,+29) | `kNumBreadPicts` 6 | `GliderPRO/Sources/StructuresInit.c:586-590` |
| `balloonSrc[0..7]` | balloon 24x240 | 8 | 24 x 30 | (0,0) | (0,+30) | `kNumBalloonFrames` 8 | `GliderPRO/Sources/StructuresInit.c:686-690` |
| `copterSrc[0..9]` | copter 32x300 | 10 | 32 x 30 | (0,0) | (0,+30) | `kNumCopterFrames` 10 | `GliderPRO/Sources/StructuresInit.c:692-696` |
| `dartSrc[0..3]` | dart 64x76 | 4 | 64 x 19 | (0,0) | (0,+19) | `kNumDartFrames` 4 | `GliderPRO/Sources/StructuresInit.c:698-702` |
| `ballSrc[0..1]` | ball 32x64 | 2 | 32 x 32 | (0,0) | (0,+32) | `kNumBallFrames` 2 | `GliderPRO/Sources/StructuresInit.c:704-708` |
| `dripSrc[0..5]` | drip 16x72 | 6 | 16 x 12 | (0,0) | (0,+12) | `kNumDripFrames` 6 | `GliderPRO/Sources/StructuresInit.c:710-714` |
| `fishSrc[0..7]` | fish 16x128 | 8 | 16 x 16 | (0,0) | (0,+16) | `kNumFishFrames` 8 | `GliderPRO/Sources/StructuresInit.c:716-720` |
| `flowerSrc[0..5]` | clutter 128x69 | 6 | **irregular** | **irregular** | — | `kNumFlowers` 6 | `GliderPRO/Sources/StructuresInit2.c:72-88` |
| `badgesBlankRects[0..3]` | badge 32x66 | 4 | 16 x 16/17 | (0,0) | irregular | literal x4; roles foil, bands, battery, helium | `GliderPRO/Sources/StructuresInit.c:135-142` |
| `badgesBadgesRects[0..3]` | badge 32x66 | 4 | 16 x 16/17 | (16,0) | irregular | literal x4; same four roles | `GliderPRO/Sources/StructuresInit.c:144-151` |

58 arrays, **197 cells** in total.

The five arrays marked *irregular* or *aliases* cannot be generated by a loop and must be transcribed
literally. All as `(l,t,r,b)`:

```
greaseSrcRt[0..3]      (0,243,32,270)  (0,270,32,297)  (0,297,32,324)  (32,297,64,324)
greaseSrcLf[0..3]      (0,324,32,351)  (32,324,64,351) (0,351,32,378)  (32,351,64,378)
sparkleSrc[0..4]       (0,108,20,127)  (0,89,20,108)   (0,70,20,89)    (0,89,20,108)   (0,108,20,127)
flowerSrc[0..5]        (0,23,10,51)    (10,16,34,51)   (34,16,68,51)   (68,14,95,37)   (68,37,95,51)  (95,0,127,51)
badgesBlankRects[0..3] (0,0,16,16)     (0,16,16,32)    (0,32,16,49)    (0,49,16,66)
badgesBadgesRects[0..3](16,0,32,16)    (16,16,32,32)   (16,32,32,49)   (16,49,32,66)
```

Three things in that block are easy to get wrong:

1. **`greaseSrcRt` wraps to a second column but `greaseSrcLf` wraps on a different frame.** Right-side
   grease goes down the x=0 column for frames 0,1,2 and then jumps to x=32 for frame 3; left-side
   grease alternates x=0, x=32, x=0, x=32. Transcribe the eight `QOffsetRect` calls literally
   (`GliderPRO/Sources/StructuresInit.c:371-387`); do not infer a pattern.
2. **`sparkleSrc` is a 5-entry ping-pong built from 3 distinct cells.** The loop fills indices 2,3,4
   at y = 70, 89, 108, then `sparkleSrc[0] = sparkleSrc[4]; sparkleSrc[1] = sparkleSrc[3];`
   (`GliderPRO/Sources/StructuresInit.c:400-401`), giving the y-sequence **108, 89, 70, 89, 108** for
   indices 0..4. Combined with `kStartSparkle` = 4 (`GliderPRO/Headers/GliderDefines.h:545`) and a
   decrementing mode counter, a sparkle therefore plays large→small→large. `kNumSparkleModes` = 5
   (`:252`), `kMaxSparkles` = 3 (`:251`).
3. **The badge rects are 16 or 17 rows tall, not 16.** Rows are 0-16, 16-32, 32-49, 49-66: frames 2
   and 3 are 17 rows. The array is 32x66, i.e. two 16-wide columns ("blank" then "badge") of four
   rows. The four rows are, per the source comments, **foil, rubber bands, battery, helium**
   (`GliderPRO/Sources/StructuresInit.c:135`, `:137`, `:139`, `:141`); the corresponding on-screen
   destinations `badgesDestRects[0..3]` are at `432 + hOffset`, `449 + hOffset`, `467 + hOffset`,
   `467 + hOffset` (`:153-160` — note 2 and 3 share an x, they are alternatives, not both drawn).
   The same rects are tabulated in `rendering.md:4133` / `:4174`.

### 6.7 Frame counts and strides, all in one place

Every animated type's frame count, the constant that governs it, and the stride between frames.
"Stride" is the delta between consecutive cell rects; `(0,+H)` means the strip runs downward with no
gap, which is the case for all but the switch pairs (which run rightward) and the four irregular
arrays.

| Frames | Constant | Value | Cell W x H | Stride | Sheet |
|---|---|---|---|---|---|
| candle / taper / stubby flame | `kNumCandleFlames` | **5** | 16 x 15 | (0,+15) | blower 4000/5000 |
| tiki flame | `kNumTikiFlames` | **5** | 8 x 10 | (0,+10) | blower 4000/5000 |
| BBQ coals | `kNumBBQCoals` | **4** | 32 x 9 | (0,+9) | blower 4000/5000 |
| cuckoo pendulum | `kNumPendulums` | **3** | 32 x 28 | (0,+28) | bonus 4002/5002 |
| toast / bread | `kNumBreadPicts` | **6** | 32 x 29 | (0,+29) | toast 4009/5009 |
| outlet spark | `kNumOutletPicts` | **4** | 16 x 24 | (0,+24) | appliance 4005/5005 |
| track light | `kNumTrackLights` | **3** | 24 x 24 | (+24,0) | light 4004/5004 |
| balloon | `kNumBalloonFrames` | **8** | 24 x 30 | (0,+30) | balloon 4011/5011 |
| copter | `kNumCopterFrames` | **10** | 32 x 30 | (0,+30) | copter 4012/5012 |
| dart | `kNumDartFrames` | **4** | 64 x 19 | (0,+19) | dart 4013/5013 |
| ball | `kNumBallFrames` | **2** | 32 x 32 | (0,+32) | ball 4014/5014 |
| drip | `kNumDripFrames` | **6** | 16 x 12 | (0,+12) | drip 4015/5015 |
| fish (animated) | `kNumFishFrames` | **8** | 16 x 16 | (0,+16) | **fish 4017/5017** |
| flower | `kNumFlowers` | **6** | irregular | irregular | clutter 4018/5018 |
| glider, level | (literal `i <= 20`) | **21** | 48 x 20 | (0,+20) | glider 3999/4999 |
| glider, burning | (literal `i <= 28`) | **8** | 48 x 26 | (0,+26) | glider 3999/4999 |
| glider, extra 2 | — | **2** | 48 x 20 | at y 628, 648 | glider 3999/4999 |
| glider total | `kNumGliderSrcRects` | **31** | — | — | glider 3999/4999 |
| shadow | `kNumShadowSrcRects` | **2** | 48 x 9 | (0,+9) | shadow 3998/4998 |
| rubber band | (literal 3) | **3** | 16 x 6 | (0,+6) | bands 4007/5007 |
| flying points | (literal 15) | **15** | 24 x 8 | (0,+8) | points 4006/5006 |
| clock digits | (literal 11) | **11** | 4 x 6 | (0,+6) | bonus 4002/5002 |
| star | (literal 6) | **6** | 32 x 31 | (0,+31) | bonus 4002/5002 |
| sparkle | (literal 3, aliased to 5) | **5 slots / 3 cells** | 20 x 19 | ping-pong | bonus 4002/5002 |
| grease, right | (literal x4) | **4** | 32 x 27 | irregular | bonus 4002/5002 |
| grease, left | (literal x4) | **4** | 32 x 27 | irregular | bonus 4002/5002 |
| badge | (literal x4) | **4 x 2** | 16 x 16/17 | irregular | badge 1996 |

Constants all from `GliderPRO/Headers/GliderDefines.h:445-458` and `:558-559`; the "literal" rows have
no named constant at all and the loop bound is typed inline, which is a porting hazard (§6.8).

Two-state (not animated) pairs, for completeness: `lightSwitchSrc`, `machineSwitchSrc`,
`thermostatSrc`, `powerSrc`, `knifeSwitchSrc`, `flourescentSrc1/2`, `plusScreen1/2`, `tvScreen1/2`,
`coffeeLight1/2`, `vcrTime1/2`, `stereoLight1/2`, `microOn`/`microOff` — 13 pairs. In every case
index `[0]` / the `...1` name is the state drawn when the boolean is **true** for switches
(`GliderPRO/Sources/ObjectDraw2.c:303-314`) but the **off** state for the appliance overlays; the
naming is not consistent, so read each call site.

### 6.8 Atlas anomalies: eleven places the map and the code disagree

Every one of these was verified against the source; none is a transcription error in this document.

1. **`DrawFish` reads the wrong sheet — deliberately.** `DrawFish (short what, Rect *theRect)`
   (`GliderPRO/Sources/ObjectDraw2.c:964-970`) copies from `enemySrcMap` / `enemyMaskMap`
   (PICT 4016 / 5016, 36x33) using `srcRects[kFish]` = (0,0,36,33). The 8-frame 16x16 `fishSrcMap`
   (4017 / 5017, 16x128) is used **only** by the dynamic-object animator, never by the static draw. So
   `kFish` has two unrelated sprites: a 36x33 "fish in the wall" still and a 16x16 swimming
   animation. An extractor must emit both and must not try to reconcile them.
2. **`DrawDrip` ignores its own atlas entry.** `DrawDrip (Rect *theRect)`
   (`GliderPRO/Sources/ObjectDraw2.c:974-980`) hard-codes `&dripSrc[3]` — the 4th of 6 frames, at
   y=36 — as both source and mask rect. `srcRects[kDrip]` = (0,0,16,12) (frame 0) is used only for
   sizing by `GetObjectRect` (`GliderPRO/Sources/ObjectRects.c:245-253`). Same W x H, different
   pixels: a port that draws frame 0 statically shows the wrong drip shape.
3. **All five switch draws bypass `srcRects` entirely** and use plain `srcCopy`, no mask
   (`GliderPRO/Sources/ObjectDraw2.c:301`, `:319`, `:333`, `:347`, `:361`). Consequences: there is no
   `switchMaskMap` GWorld and no PICT 5003; and `srcRects[kMachineSwitch]` = (0,48,16,72)
   (`GliderPRO/Sources/StructuresInit2.c:405-406`) **disagrees with the art**, which lives at
   `machineSwitchSrc[0]` = (0,24,16,48). The atlas entry duplicates the thermostat's row. It is
   harmless because only its W x H (16 x 24) is ever consulted.
4. **`InitTransports` restores an uninitialised graphics port.** `GliderPRO/Sources/StructuresInit.c:425-441`
   declares `CGrafPtr wasCPort; GDHandle wasWorld;`, never calls `GetGWorld(&wasCPort, &wasWorld)`,
   and then calls `SetGWorld(wasCPort, wasWorld)` at `:440`. Enumerating every `GetGWorld` site in the
   file — `:68`, `:176`, `:251`, `:299`, `:348`, `:453`, `:499`, `:536`, `:621` — confirms there is
   none between 425 and 441. It is benign only because the very next function, `InitSwitches`, calls
   `GetGWorld` at `:453` before drawing. A Go port has no ambient current port and simply drops this.
5. **The furniture sheet has 57 unpainted rows.** GWorld 64x278 (`:301`), art 64x221. Deepest
   consumer `srcRects[kStool]` bottom = 221. Slack, never read. (`rendering.md` states the lowest
   furniture sub-rect is `deckSrc` with bottom y=183; `srcRects[kStool]` reaches 221 — see the
   erratum note at the end of this section.)
6. **The appliance mask is one row short.** Art 4005 = 80x269, mask 5005 = 80x268. Row 268 falls
   inside `tvScreen2` = (0,220,64,269), which is drawn with opaque `srcCopy`, so the mask row is never
   sampled. Extract with the missing row as alpha 0.
7. **`DrawPictSansWhiteObject` has no `default:` arm.** `GliderPRO/Sources/ObjectDraw2.c:1302-1409`:
   a 20-case switch at `:1315-1393` assigns `pictID`, then `:1396` reads `srcRects[what]` and `:1399`
   calls `LoadGraphic(pictID)`. For any `what` outside those 20 cases, `pictID` is an uninitialised
   local and `LoadGraphic` will `RedAlert(kErrFailedGraphicLoad)` or draw garbage. The dispatcher never
   does that, but a port should make the mapping a table with an explicit error.
8. **`kCustomPict` (0x6E) overloads a height field as a resource ID.**
   `thisObject.data.g.height` *is* the PICT ID (`GliderPRO/Sources/ObjectRects.c:220-237`). If
   `GetPicture` returns nil the code **rewrites the house data in memory** to 10000
   (`:224`, and identically at `GliderPRO/Sources/ObjectEdit.c:2285`) and falls back to
   `srcRects[kCustomPict]` = (0,0,72,34), which is exactly PICT 10000's size. Otherwise the object's
   rect comes from the picture's own `picFrame`, origin-normalised by `ZeroRectCorner`.
9. **The one `LoadScaledGraphic` call on object art is a no-op scale.** `DrawFloorSupport`
   (`GliderPRO/Sources/RoomGraphics.c:257`) calls `LoadScaledGraphic(kManholeThruFloor, &tempManholes[i])`
   at six sites, one per drawn room in the neighbour grid — `:282`, `:300`, `:317`, `:335`, `:353`,
   `:371` — with `kManholeThruFloor` = 3957
   (`GliderPRO/Sources/RoomGraphics.c:16`). `AddTempManholeRect`
   (`GliderPRO/Sources/Objects.c:351-362`) builds those rects as `tempRect = *manHole;
   tempRect.bottom = tempRect.top + kFloorSupportTall;` with `kFloorSupportTall` = 44
   (`GliderPRO/Headers/GliderDefines.h:500`), and the manhole is 123 wide — exactly PICT 3957's
   measured 123x44. So no scaling occurs anywhere in the object pipeline.
   Beware: unlike `LoadGraphic`, `LoadScaledGraphic` does **not** origin-normalise `picFrame`
   (`GliderPRO/Sources/Utilities.c:340-349` vs `:317-333`) — it maps `picFrame` straight onto the
   destination rect. For the two pictures whose `picFrame` origin is non-zero (masks 5006, 5010) that
   difference would matter, but both are loaded with `LoadGraphic`
   (`GliderPRO/Sources/StructuresInit.c:410`, `:563`), so it never bites.
10. **No object art is ever scaled, proven exhaustively.** For all **38** object types with their own
    PICT (21 colour-keyed including `kCustomPict`'s default 10000, 9 opaque, 8 mask-paired), the
    `srcRects` entry has origin (0,0) *and* W x H exactly equal to the picture's `picFrame` W x H —
    38/38, zero mismatches, measured. All 8 paired masks likewise match their art's dimensions
    exactly (3903/3904 = 94x80, 3912 = 92x77, 3913 = 96x22, 3914 = 128x53, 3915 = 92x59,
    3921 = 128x30, 3927 = 54x45). A port can therefore blit 1:1 and never implement PICT scaling.
11. **Frame counts with no named constant.** `digits` (11), `starSrc` (6), `bandRects` (3),
    `pointsSrc` (15), `sparkleSrc` (3 cells → 5 slots), the two grease arrays (4 each), the badge
    arrays (4 each), and both glider ranges (21 and 8) are all typed as literals in the `for` bounds.
    There is no `kNumDigits`, `kNumStars` (`kMaxStars` = 4 is the *simultaneous* limit, not the frame
    count), `kNumBands` or `kNumPointsFrames`. Do not invent constants that map to a different
    quantity.

> **Erratum for `docs/analysis/rendering.md`.** That document attributes `kBBQ`'s art to PICT **3958**
> at `docs/analysis/rendering.md:2402` and again at `:2438`. The correct ID is **3988**
> (`kBBQPictID`, `GliderPRO/Sources/ObjectDraw2.c:94`, used at `:1316`); **3958** is `kCobwebPictID`
> (`:67`, used at `:1268`), which is `kCobweb`'s art. The two constants are 30 apart in the same
> `#define` block, and `rendering.md` gets 3958 right elsewhere (`:1344` measures it as 54x45 and
> `:2756` maps `kCobweb` to it), so the error is localised to those two sentences. It is corrected
> here and in the erratum section appended to that file.

---

## 7. The extraction pipeline

Everything above is analysis. This section is the executable conclusion: a four-stage pipeline that
turns `GliderPRO/Glider PRO.r` into a directory of RGBA PNGs plus a machine-readable manifest, with
no third-party dependencies and no Mac Toolbox. It has been run end to end; the measured results are
in §7.11.

### 7.1 Stage map

| Stage | Tool | Input | Output |
|---|---|---|---|
| 0 | `tools/probe_rez.py extract` | `GliderPRO/Glider PRO.r` (15,475,666 bytes, 199,843 LF lines) | `<res>/<TYPE>/<id>.bin` — one file per resource, raw payload, 35 types |
| 1 | `tools/probe_pict.py` (`Picture.parse` → `rasterise`) | one `PICT` payload | `(W, H, RGB bytes, info)` where `info["idxmap"]` is the **palette-index plane** |
| 2 | `tools/extract_art.py` (alpha rules) | index plane + mask/key rule | RGBA byte buffer |
| 3 | `tools/extract_art.py` (crop + PNG) | RGBA + the §6 atlas table | `<out>/sheet/*.png`, `<out>/object/*.png`, `<out>/manifest.json` |

Runnable, exactly as measured:

```sh
python3 tools/probe_rez.py extract /tmp/gp/res          # stage 0
python3 tools/extract_art.py /tmp/gp/res /tmp/gp/art    # stages 1-3
```

`tools/extract_art.py` is self-contained: the whole §6.2 sheet table and all 116 §6.3 atlas rows are
embedded in it as the `SHEETS` dict and the `ATLAS` list, so the atlas is machine-readable and a Go
port can transcode it directly rather than re-deriving it from `InitSrcRects`.

The two things stage 0 has to get right are documented in `docs/analysis/resource-fork.md` Part 2 and
are not restated here. One naming detail matters when you look for files: resource types containing
characters illegal in filenames are sanitised, so `'snd '` becomes the directory `snd_`.

### 7.2 Stage 1: the opcode subset a Glider PRO decoder actually needs

§3.3 measured the histogram; §3.13 gave the loop. The operational statement for an implementer is
this: **implement seven opcodes with semantics, two as skip-by-length, and treat everything else as a
fatal error.**

| Opcode | Name | Must do | Why |
|---|---|---|---|
| 0x0011 | `VersionOp` | detect v1 vs v2; in v2 consume 2 bytes; thereafter align every opcode read to an even offset | v2 byte pattern is `00 11 02 FF` at offset 10 (§3.1.2) |
| 0x0C00 | `HeaderOp` | consume 24 bytes and ignore them, **whatever the version field says** | 116 of 152 pictures have it: 95 standard (`version == -1`), 21 extended (`version == -2`). Both payloads are 24 bytes, so no version-specific handling is needed — and rejecting `-2` would reject 21 shipped pictures including `PICT` 3999, 1000, 1997, 2002, 2016, 2017 (§3.2) |
| 0x0001 | `Clip` | read `uint16 rgnSize`, consume `rgnSize` bytes | always 10 (a bare rect) in all 152; a real region here would need clipping |
| 0x0090 | `BitsRect` | parse per §3.5, rows **unpacked** | 17 occurrences: 16 old-style 1-bit `BitMap`s, plus `PICT` 1003, a real 1-bit *PixMap* with `packType == 1` |
| 0x0098 | `PackBitsRect` | parse per §3.5, rows PackBits-decoded (byte or word variant per §3.8/§3.9) | 185 occurrences — the workhorse |
| 0x0099 | `PackBitsRgn` | as 0x0098, plus read and **ignore** the mask region | 3 occurrences (`PICT` 3968, 3969, 3995); the region equals the picture rect, so ignoring it is measurably correct |
| 0x001E | `DefHilite` | skip 0 bytes | 95 occurrences |
| 0x00A0 | `ShortComment` | skip 2 bytes | 269 occurrences |
| 0x00FF | `OpEndPic` | stop | 152 occurrences |

Total nine opcodes. `tools/extract_art.py` encodes exactly this set as the frozenset `ALLOWED_OPS`
and raises on anything else. Running that guard over **all 152 `PICT`s decodes 152 of 152 with zero
rejections**, which is the evidence that the allow-list is not merely sufficient for the sprites the
extractor happens to touch but for the entire art set.

### 7.3 What a decoder may refuse

A Glider-PRO-specific decoder is allowed to be hostile. Everything in this list is provably absent
from the shipped art (§3.3), so refusing it converts a whole class of silent corruption into a loud
error. Ranked by how likely a general PICT would use it:

| Refuse | Opcode(s) | Rationale |
|---|---|---|
| `DirectBitsRect`, `DirectBitsRgn` | 0x009A, 0x009B | 16/24/32-bit direct-colour rasters. Zero occurrences; the game is 8-bit only (`kPreferredDepth` = 8, `GliderPRO/Headers/Externs.h:15`) |
| `BitsRgn` | 0x0091 | uncompressed + region. Zero occurrences |
| `BkPixPat`, `PnPixPat`, `FillPixPat` | 0x0012-0x0014 | pattern PixMaps; would require a whole second PixMap parser. Zero occurrences |
| all vector ops | 0x0020-0x0027, 0x0030-0x0087 | line/rect/roundrect/oval/arc/poly/region draws. Zero occurrences: **not one pixel of shipped art is drawn by a vector opcode.** (Procedural drawing does exist in Glider PRO, but it is C code in `ObjectDraw*.c`, not PICT opcodes — §7.15) |
| all text ops | 0x0028-0x002B, and the font/face/size/mode setters 0x0003-0x0005, 0x000D | zero occurrences. Text in the game is drawn by C code from `STR#` resources |
| `LongComment` | 0x00A1 | zero occurrences (only `ShortComment`) |
| QuickTime payloads | 0x8200, 0x8201 | zero occurrences; would need a JPEG/RLE codec |
| ~~extended-v2 header~~ | ~~`HeaderOp` with `version == -2`~~ | **Do not refuse this.** 21 of the 116 `HeaderOp`s are the extended form (§3.2 consequence 4). It changes only the interpretation of the 24 skipped payload bytes — `Fixed hRes`/`Fixed vRes`/`Rect srcRect` instead of a `Fixed` bounding box — and the extractor ignores the payload in both cases, so the two forms are indistinguishable downstream |
| any opcode ≥ 0x0100 other than 0x0C00 | — | zero occurrences |
| `packType` other than 0 | PixMap +16 | 204 of 205 raster ops have `packType == 0`; the single exception is `packType == 1` (unpacked) in `PICT` 1003, a UI resource. `packType` 2 (drop pad byte) and 4 (plane separation) never occur |
| `pixelSize` other than 1 or 8 | PixMap +32 | 167 ops at 8 bpp, 38 at 1 bpp, nothing else (§3.11) |
| transfer `mode` other than `srcCopy` (0) | raster op +? | all 205 are `srcCopy`. Transparency is never inside a PICT (§3.11) |

The one place this hostility must be relaxed is user-supplied house art (§7.12): a house `PICT` was
authored in some 1990s paint program and can legitimately be anything. There the right behaviour is
to log and fall back to `PICT` 2000, mirroring `LoadGraphicSpecial`
(`GliderPRO/Sources/RoomGraphics.c:134-158`), not to abort.

### 7.4 Stage 1 output: why the index plane, not the RGB, is the real product

`rasterise` (`tools/probe_pict.py:702`) returns both an RGB canvas and `info["idxmap"]`, a list of
`H` rows of `W` **palette indices** with `None` where the canvas was never painted. Both alpha rules
in §5 are defined on *indices*, not colours:

* the white colour key is "index == 0", not "RGB == (255,255,255)";
* a 1-bit mask is "index == 1".

Those are not interchangeable. Index 0 is `#FFFFFF`, but so is nothing else in the palette — the
palette has exactly one white — so in this particular game the RGB test happens to give the same
answer. **Do not rely on that.** A Go port should carry the index plane, because the same 8-bit
surfaces are also used for the `srcXor` mirror trick and for `Index2Color` lookups elsewhere in the
renderer, and because `None` (never painted) must be distinguished from "painted white" in general
even though here both map to transparent.

`rasterise` initialises its canvas to white (`b"\xff"` per byte) because "QuickDraw ports start
white" — a fresh `GWorld` from `CreateOffScreenGWorld` (`GliderPRO/Sources/Utilities.c:266`) is
white-filled, and `LoadGraphic` draws the picture into it without clearing first
(`GliderPRO/Sources/Utilities.c:317-333`). The index plane's `None` therefore means the same thing as
index 0 for every use in this game.

### 7.5 Stage 2: palette resolution order

Measured facts that set the policy (§4.1, §3.11):

* 168 of 205 raster ops carry an embedded `ColorTable`; **167 of the 168 resolve byte-identically to
  `clut` 128**. The one exception is `PICT` 1003's 2-entry 1-bit table.
* `ctFlags` distribution across those 168: `0x8000` (device table, index == position) x146,
  `0x0000` x21, `0x0001` x1.
* All 21 `ctFlags == 0x0000` pictures — 1000, 1001, 1002, 1004, 1006, 1007, 1008, 1010, 1012, 1013,
  1014, 1994, 1995, 1997, 2002, 2016, 2017, 3975, 3976, 3997, 3999 — nonetheless have identity
  `value` fields 0..255, so the "map value → index" step is a no-op for them too.
* The remaining 37 raster ops are old-style 1-bit `BitMap`s with no table at all.

Resolution order, in this order, no exceptions:

1. **Embedded `ColorTable`, if the op has one.** Honour `ctFlags` bit 15: when set, entry *i* is
   index *i* and the `value` field is meaningless; when clear, `value` is the index. (In this file
   both give the same answer, but honouring the flag is free.)
2. **`clut` 128** (`tools/probe_pict.py:824` parses it; `:417` converts it to a 256-entry RGB list).
   Only needed for the 1-bit `BitMap` ops, and for those it is the *wrong* thing to apply — see 3.
3. **1-bit data is not palette data.** For `pixelSize == 1`, do **not** index a 256-entry table. Treat
   value 1 as opaque and value 0 as transparent (§5.2). A decoder that runs 1-bit masks through
   `clut` 128 gets index 0 = white and index 1 = `#FFFFCC`, which looks plausible and is completely
   meaningless.
4. **Synthetic grey ramp** as a last resort for depths with no table (never exercised by this file;
   `tools/probe_pict.py:728-731` implements it defensively).

Consequence for a Go port: `clut` 128 is a *safety net, not a requirement*. The art is
self-describing. But because all 167 tables agree, hard-coding the 256 triples (dumped in §4.2) and
skipping ColorTable parsing entirely is also valid for the shipped art — just not for house art.

### 7.6 Stage 2: the four alpha rules, unified into RGBA

§5 established four compositing paths. All four collapse to "write an 8-bit alpha channel at load
time", which is what makes a Go port simple. The implementations are the four functions in
`tools/extract_art.py`:

| §5 path | Rule | Function | Applies to |
|---|---|---|---|
| 1 (permanent mask GWorld) and 3 (throwaway GWorld pair) | `alpha = 255 if maskIndex == 1 else 0` | `rgba_from_mask(art, mask)` | 13 of the 14 sheets, 8 mask-paired objects |
| 2 (white colour key) | `alpha = 0 if (artIndex is None or artIndex == 0) else 255` | `rgba_from_key(art)` | 21 colour-keyed objects (20 with constant IDs + `kCustomPict`) |
| 4 (opaque) | `alpha = 255` everywhere | `rgba_opaque(art)` | 9 opaque objects, the `switch` sheet, `suppSrcMap`, `badgeSrcMap`, `boardSrcMap`, all 18 backgrounds |
| — (procedural) | emit nothing | — | `kCounter`, `kMirror`, `kWallWindow` |

Two rules that are easy to get wrong and are load-bearing:

* **Mask and art are pixel-aligned and the same rect.** `CopyMask` is called with the *same* rect for
  source and mask (`GliderPRO/Sources/ObjectDraw.c:61-64`), so no offset arithmetic is needed.
  This is why pre-multiplying alpha into the sheet at load time is valid.
* **Mask 5005 is one row shorter than art 4005** (80x268 vs 80x269, §3.12.6). `rgba_from_mask` treats
  a missing mask row as fully transparent. That reproduces the original exactly, because the only
  consumer of row 268 is `tvScreen2` (0,220,64,269) which is blitted opaque with `srcCopy` and never
  consults the mask (`GliderPRO/Sources/ObjectDraw2.c:681`, rect set at `GliderPRO/Sources/StructuresInit.c:572-573`). Padding with opaque would also be
  correct; crashing on the size mismatch would not.

Straight alpha, not premultiplied: every source pixel is either fully opaque or fully transparent
(there is no partial alpha anywhere in the game), so premultiplication is a no-op and the distinction
does not matter — but the RGB under a transparent pixel is *not* guaranteed to be black, it is
whatever the art had there. A Go port that composites with `draw.Over` is fine; one that assumes
premultiplied and skips the multiply will show colour fringing if it ever filters or scales.

### 7.7 Stage 3: cropping the atlas

For a `kind == "sheet"` entry the whole job is: crop the sheet's RGBA at `srcRects[what]`. The rects
in §6.3 are printed in `QSetRect` order `(l,t,r,b)`; `crop()` takes them in that order. Two
non-obvious points:

* **The sheet GWorld can be bigger than the art** (§3.12.6): `furnitureSrcRect` is 64x278 but `PICT`
  4001 is 64x221. Size the surface from the art and clamp crops, or size it from the `QSetRect` and
  leave the slack transparent. `crop()` clamps; both give identical output for the shipped rects
  because the deepest furniture rect bottoms out at exactly y=221.
* **Origins are already sheet-absolute.** `InitSrcRects` does `QSetRect(...)` then `QOffsetRect(...)`;
  §6.3 prints the post-offset value. Do not offset again.

For `kind == "tmpl"` there is nothing to crop from `srcRects` — the rect is a size/hit template and
the art is in the named sub-rect globals of §6.6. The extractor deliberately emits **no** PNG for
those 12 and records the sheet plus a pointer in the manifest, because emitting the template rect
would produce 12 wrong sprites that look almost right (e.g. `srcRects[kTable]` (0,0,64,8) would crop
the top 8 rows of the furniture sheet instead of `tableSrc` 64x22).

### 7.8 The complete 152-`PICT` accounting, by extractor role

`resource-fork.md` §3.1.1 partitions the 152 `PICT`s by **ID range**. The extractor needs a different
cut: by **role in the pipeline**, because that is what decides which alpha rule and which output
directory each one gets. Both partitions were computed programmatically and cross-checked against
each other; neither has duplicates, gaps or extras.

| Bucket | Count | IDs | Alpha rule | Output |
|---|---|---:|---|---|
| `sheet_art` | 14 | 4000, 4001, 4002, 4003, 4004, 4005, 4008, 4011, 4012, 4013, 4014, 4015, 4016, 4018 | mask (13) / opaque (4003) | `sheet/<name>.png` |
| `sheet_mask` | 13 | 5000, 5001, 5002, 5004, 5005, 5008, 5011, 5012, 5013, 5014, 5015, 5016, 5018 | consumed | — |
| `strip_art` | 14 | 1019, 1996, 1997, 1999, 3963, 3974, 3976, 3998, 3999, 4006, 4007, 4009, 4010, 4017 | mask (8) / opaque (6) | `strip/<name>.png` |
| `strip_mask` | 8 | 1020, 4998, 4999, 5006, 5007, 5009, 5010, 5017 | consumed | — |
| `obj_art` | 38 | 21 keyed (3959, 3960, 3961, 3962, 3964, 3967, 3968, 3969, 3972, 3973, 3979, 3980, 3983, 3984, 3987, 3988, 3991, 3993, 3994, 3997, 10000) + 9 opaque (3966, 3970, 3975, 3977, 3978, 3981, 3982, 3995, 3996) + 8 paired (3958, 3965, 3971, 3985, 3986, 3989, 3990, 3992) | key / opaque / mask | `object/<WHAT>_<constant>.png` |
| `obj_mask` | 8 | 3903, 3904, 3912, 3913, 3914, 3915, 3921, 3927 | consumed | — |
| `bg` | 18 | 2000-2017 | opaque | `bg/<id>.png` |
| `misc` | 1 | 3957 (`kManholeThruFloor`) | opaque | `misc/3957_manhole_thru_floor.png` |
| `ui` | 38 | 150, 151, 153, 1000-1018, 1021, 1022, 1023, 1202, 1211, 1216, 1217, 1988-1995, 1998 | opaque | `ui/<id>.png` |
| **Total** | **152** | | | |

14 + 13 + 14 + 8 + 38 + 8 + 18 + 1 + 38 = 152. ✔

Cross-check against `resource-fork.md` §3.1.1's ID-range groups, which reconciles exactly:

| `resource-fork.md` group | Count | = my buckets |
|---|---:|---|
| Room backgrounds 2000-2017 | 18 | `bg` 18 |
| Dialog art 1000-1023 | 24 | `ui` 22 (1000-1018, 1021-1023) + `strip_art` 1019 + `strip_mask` 1020 |
| Room-object art 3957-3997 | 41 | `obj_art` 37 (excl. 10000) + `misc` 3957 + `strip_art` 3963, 3974, 3976 |
| Game-over / banner / scoreboard 1988-1999 | 12 | `ui` 9 (1988-1995, 1998) + `strip_art` 1996, 1997, 1999 |
| Animated sprite sheets 3998-4018 | 21 | `sheet_art` 14 + `strip_art` 3998, 3999, 4006, 4007, 4009, 4010, 4017 |
| Sprite-sheet masks 4998-5018 (no 5003) | 20 | `sheet_mask` 13 + `strip_mask` 4998, 4999, 5006, 5007, 5009, 5010, 5017 |
| Room Info thumbnails 1202, 1211, 1216, 1217 | 4 | `ui` 4 |
| About-box art 150, 151, 153 | 3 | `ui` 3 |
| Custom-picture placeholder 10000 | 1 | `obj_art` (keyed) |
| Object masks 3903-3927 | 8 | `obj_mask` 8 |

The four Room Info thumbnails are worth a note because their existence looks arbitrary until you find
the code. They are 128x80 (measured), exactly `tileSrcRect`
(`QSetRect(&tileSrcRect, 0, 0, 128, 80)`, `GliderPRO/Sources/StructuresInit2.c:178`), and they exist
only for backgrounds **2002, 2011, 2016, 2017** because those four are hard-coded to load
`tempBack - 800` instead of `tempBack` (`GliderPRO/Sources/RoomInfo.c:416-420`): 2002→1202,
2011→1211, 2016→1216, 2017→1217. The other 14 backgrounds get their full 512x322 art scaled down to
128x80 by `DrawPicture`. Two of the three other call sites in the same function
(`GliderPRO/Sources/RoomInfo.c:514`, `:555`) **omit the −800 substitution**, so re-picking one of
those four backgrounds through the "original artwork" or "bounds" buttons silently switches the
thumbnail to the scaled full-size version. That is a bug in the original; a port should apply the
substitution uniformly.

### 7.9 Output naming scheme

```
<out>/
  sheet/<name>.png            14 files: blower, furniture, bonus, switch, light, appliance,
                              trans, balloon, copter, dart, ball, drip, enemy, clutter
  object/<WHAT>_<constant>.png   e.g. 0B_kTiki.png, 6E_kCustomPict.png
  object/85_kFlower_<i>.png      i = 0..5, the six flowerSrc[] cells
  strip/<name>.png            glider, glider2, gliderFoil, gliderFoil2, shadow, bands, points,
                              toast, shred, fish, supp, angel, badge, board
  bg/<id>.png                 2000..2017
  ui/<id>.png                 the 38 UI pictures
  misc/3957_manhole_thru_floor.png
  manifest.json
```

Rules and their reasons:

1. **`<WHAT>` is the object type byte in uppercase hex, two digits, zero-padded** — `01`..`8F`. It
   sorts in the same order as the `GliderDefines.h` groups and matches the `what` field stored in the
   house file, so a room dump can be joined against the sprite directory with no lookup table.
2. **`<constant>` is the exact C identifier**, `kTiki` not `tiki`, so grepping the original sources
   for a filename works.
3. **Sheets are named by their GWorld prefix**, `blower` from `blowerSrcMap`, not by `PICT` ID. IDs
   change meaning across the house/app resource chain (§7.12); GWorld names do not.
4. **Masks get no output file of their own.** They are consumed. The manifest records which mask each
   sprite used, so the mapping is recoverable without a second directory.
5. **`kCustomPict` (0x6E) gets `object/6E_kCustomPict.png` from `PICT` 10000** — the *placeholder*,
   not the user's art, because the real ID is runtime data (§7.12). This is why §6.1's `key` count is
   21 but only 20 keyed objects have a constant `PICT` ID.
6. **Flowers are the one type with no atlas entry** (§6.5), so they get six numbered files instead of
   one. Naming them `85_kFlower_<i>` rather than `85_kFlower` makes it structurally impossible for a
   consumer to look up "the flower sprite" and get one arbitrary variant.

### 7.10 `manifest.json`

Shape:

```json
{ "sheets": { "<name>": {...} }, "objects": { "<WHAT>_<constant>": {...} }, "stats": {...} }
```

Per-sheet fields: `gworld`, `gworld_size` (the `QSetRect` bounds, as a string like `"64x278"`),
`art_pict`, `mask_pict` (`null` for `switch`), `art_size` `[w,h]` (**measured from the `PICT`, which
is not always `gworld_size`**), `alpha`, `opaque_px`, `total_px`.

Per-object fields: `what` (decimal), `kind` (one of the seven of §6.1), `srcRect` `[l,t,r,b]`, `w`,
`h`, `draw` (the C helper name), `note`, `from` (`"sheet:<name>"` or `"pict:<id>"`), `file` (present
only when a PNG was written), and depending on kind `alpha`, `mask_pict`, `opaque_px`,
`transparent_px`, `total_px`.

`stats` is the §6.1 taxonomy histogram, and reproducing it is the pipeline's own self-test:

```
{"sheet": 50, "key": 21, "none": 13, "tmpl": 12, "opaq": 9, "pair": 8, "proc": 3, "frames": 1}
```

50 + 21 + 13 + 12 + 9 + 8 + 3 = **116** srcRects entries, + 1 `frames` entry for `kFlower` = 117
object records. 87 of the 117 carry a `file` (50 sheet crops + 20 keyed + 9 opaque + 8 paired), and
with the six flower crops that is **93 object PNGs**. The 30 with no file are 13 `none`, 12 `tmpl`,
3 `proc`, `kCustomPict` (its ID is runtime data), and the `85_kFlower` record itself (whose six PNGs
are listed under `frames` rather than `file`) — all of which are *correctly* fileless.

### 7.11 Measured full-run results

`python3 tools/extract_art.py /tmp/wf-gfx/res /tmp/gp-art` produced 14 sheet PNGs and 93 object PNGs.
The sheet alpha coverage it reports, which is the §6.2 table's last column:

```
sheet appliance   80x269 art 4005 mask 5005                      opaque  15783/ 21520 (73.3%)
sheet ball        32x64  art 4014 mask 5014                      opaque   1524/  2048 (74.4%)
sheet balloon     24x240 art 4011 mask 5011                      opaque   3811/  5760 (66.2%)
sheet blower      48x402 art 4000 mask 5000                      opaque   9471/ 19296 (49.1%)
sheet bonus       88x378 art 4002 mask 5002                      opaque  12599/ 33264 (37.9%)
sheet clutter    128x69  art 4018 mask 5018                      opaque   5043/  8832 (57.1%)
sheet copter      32x300 art 4012 mask 5012                      opaque   1739/  9600 (18.1%)
sheet dart        64x76  art 4013 mask 5013                      opaque   2433/  4864 (50.0%)
sheet drip        16x72  art 4015 mask 5015                      opaque    340/  1152 (29.5%)
sheet enemy       36x33  art 4016 mask 5016                      opaque   1010/  1188 (85.0%)
sheet furniture   64x221 art 4001 mask 5001                      opaque   8904/ 14144 (63.0%)
sheet light       72x126 art 4004 mask 5004                      opaque   4785/  9072 (52.7%)
sheet switch      32x104 art 4003 opaque (no mask resource exists) opaque   3328/  3328 (100.0%)
sheet trans       56x32  art 4008 mask 5008                      opaque   1530/  1792 (85.4%)
```

Note `furniture 64x221` and `switch ... 100.0%` — the two structural anomalies of §6.2 falling out of
the run rather than being asserted.

The keyed objects' transparency, computed by the pipeline from the RGBA it wrote, reproduces the
independently-measured §5.3 table to the tenth of a percent for all 20:

```
 86.8% 54_kHipLamp     84.2% 55_kDecoLamp    57.7% 64_kGuitar      53.5% 8A_kVase2
 46.5% 0C_kBBQ         36.8% 8F_kChimes      35.5% 87_kBear        23.5% 6B_kCinderBlock
 17.4% 89_kVase1       17.2% 38_kDoorInRt    17.2% 37_kDoorInLf    12.9% 6C_kFlowerBox
 12.7% 3C_kWindowInRt  12.7% 3B_kWindowInLf  12.0% 8E_kRug         10.1% 1E_kBooks
  9.6% 1D_kManhole      7.2% 1B_kTrunk        4.4% 84_kFireplace    1.9% 31_kUpStairs
```

That agreement is the pipeline's end-to-end validation: §5.3 measured palette-index histograms
directly off the decoded pictures, §7 measured alpha channels off the written PNGs, and the two
independent paths agree.

### 7.12 House-embedded `PICT`s: IDs ≥ 3000

A house file is a Mac document with its own resource fork, and it can carry its own `PICT`s. This is
the one place where the extraction model has to become two-level.

**How the house fork gets into the search chain** (`GliderPRO/Sources/HouseIO.c:567-577`):

```c
void OpenHouseResFork (void)
{
    if (houseResFork == -1)
    {
        houseResFork = FSpOpenResFile(&theHousesSpecs[thisHouseIndex], fsCurPerm);   // :571
        if (houseResFork == -1)
            YellowAlert(kYellowFailedResOpen, ResError());
        else
            UseResFile(houseResFork);                                               // :575
    }
}
```

`UseResFile` pushes the house to the **front** of the resource search chain, so from that moment
every `GetPicture(id)` in the entire program checks the house *first* and the application second.
`GetPicture` searches the whole chain; `Count1Resources`/`Get1Resource` search only the current file.
`HouseHasOriginalPicts` relies on exactly that: `return (Count1Resources('PICT') > 0)` — non-zero
means the *house* has `PICT`s, regardless of the 152 in the app
(`GliderPRO/Sources/House.c:246-252`).

Which IDs a house may use, and the shadowing rules that follow, are specified in
`docs/analysis/original-houses.md` §1.5 (the ID-ownership table, the 16-house shadowing matrix, Fun
House's shadowing of backgrounds 2014 and 2015, and the two-level-lookup port note). That is not
repeated here. The extraction-specific facts are:

1. **The editor's declared range is 3000..3799.** `GliderPRO/Sources/RoomInfo.c:762` accepts a typed
   background ID only if `(longID >= 3000) && (longID < 3800) && PictIDExists((short)longID)`.
   `kUserBackground` = 3000 (`GliderPRO/Headers/GliderDefines.h:522`) is the floor;
   3800 is an exclusive ceiling that appears nowhere else. Note the app's own object art starts at
   3903, above that ceiling — the gap 3800..3902 is unclaimed by anything.
2. **`PictIDExists` also accepts a `'Date'` resource.** `GliderPRO/Sources/RoomInfo.c:830-855` tries
   `GetPicture(theID)`, then `GetResource('Date', theID)`. `'Date'` is not a graphics type anywhere
   else in the game or in Mac OS; the most plausible reading is a private tag some house-authoring
   tool used to reserve an ID. A port should treat a `'Date'` resource as "the ID exists but has no
   pixels".
3. **`LoadGraphicSpecial` has a three-step fallback**
   (`GliderPRO/Sources/RoomGraphics.c:134-158`), and it is the *only* graphic loader in the game that
   does not fail hard:

   ```
   1. thePicture := GetPicture(resID)                     # house first, then app
   2. if nil:  thePicture := GetResource('Date', resID)   # yes, cast to PicHandle and drawn
   3. if nil:  thePicture := GetPicture(2000)             # kSimpleRoom
   4. if nil:  RedAlert(kErrFailedGraphicLoad)            # = 5, GliderPRO/Headers/Externs.h:187
   5. bounds := (*thePicture)->picFrame
   6. OffsetRect(&bounds, -bounds.left, -bounds.top)      # normalise origin to (0,0)
   7. DrawPicture(thePicture, &bounds)                    # 1:1, no scaling
   ```

   Step 2 is a genuine latent crash: a `'Date'` handle is not a `PicHandle`, and `DrawPicture` on it
   is undefined behaviour. A port must not reproduce step 2; skip straight to step 3.
   Contrast with `LoadGraphic` (`GliderPRO/Sources/Utilities.c:317-333`), which is identical minus
   the fallbacks, and `LoadScaledGraphic` (`:340-349`), which **omits the origin normalisation** and
   maps `picFrame` onto the caller's rect — the difference matters only for pictures whose `picFrame`
   origin is not (0,0), and none of the 152 has one.
4. **`kCustomPict` (0x6E) overloads `data.g.height` as a `PICT` resource ID.** Not a height. The
   object's on-screen size is the picture's own `picFrame`
   (`GliderPRO/Sources/ObjectRects.c:220-237`):

   ```c
   case kCustomPict:
       thePict = GetPicture(who->data.g.height);
       if (thePict == nil) { who->data.g.height = 10000;  *itsRect = srcRects[who->what]; }
       else                { *itsRect = (*thePict)->picFrame; }
       ZeroRectCorner(itsRect);
       QOffsetRect(itsRect, who->data.g.topLeft.h, who->data.g.topLeft.v);
   ```

   The `10000` fallback has a subtle and important asymmetry:

   | Context | `who` points at | Effect of the rewrite |
   |---|---|---|
   | play mode, `DrawARoomsObjects` | a **stack copy** made at `GliderPRO/Sources/ObjectDrawAll.c:50` | the rewrite is discarded after the frame, but `:614` passes `thisObject.data.g.height` — the already-rewritten value — to `DrawCustPictSansWhite`, so the placeholder does get drawn. The house file is never modified. |
   | edit mode, `GetThisRoomsObjRects` | **the real house data**, `thisRoom->objects[i]` (`GliderPRO/Sources/ObjectEdit.c:2281-2293`) | the rewrite is **persistent**: merely opening a house in the editor with a missing custom `PICT` permanently retargets the object to 10000, and saving commits it. |

   `PICT` 10000 is measured 72x34, exactly `srcRects[kCustomPict]` = (0,0,72,34) — so the placeholder
   and the template agree, and the fallback is size-stable.
5. **`DrawCustPictSansWhite` sizes its temp GWorld from the object's rect, not from `srcRects`**
   (`GliderPRO/Sources/ObjectDraw2.c:1412-1436`): `bounds = *theRect; ZeroRectCorner(&bounds);`.
   That is the only structural difference from `DrawPictSansWhiteObject`, and it is what lets a custom
   picture be any size. If the picture is *larger* than the object rect it is silently clipped by the
   GWorld; if smaller, the uncovered part of the GWorld stays white and is therefore keyed out.
6. **Custom backgrounds must be 512x322.** `SetInitialTiles` gives a `background >= kUserBackground`
   the identity tile map `tiles[i] = i` (`GliderPRO/Sources/Room.c:44-53`), and `DrawRoomBackground`
   then copies eight 64-wide columns out of it (§7.13). The Room Info tile editor only exposes eight
   mini-tiles of `kMiniTileWide` = 16 (`GliderPRO/Sources/RoomInfo.c:29`, `:126`), so `tiles[i]` is
   0..7 in practice and the picture needs 8 x 64 = 512 columns and `kTileHigh` = 322 rows. Anything
   narrower reads unpainted GWorld; anything wider is unreachable.

For extraction, the practical consequence is a **two-level asset map**: resolve an ID against the
house's own table first, then the application's. `original-houses.md` §1.5 gives the port note. The
extractor in this document deliberately handles only the application fork, because the house forks are
a separate document's subject; the hook is that `Res.raw()` takes a resource directory, so pointing it
at a house's extracted fork and chaining two `Res` objects is a ten-line change.

### 7.13 Backgrounds and the 8-tile room model

Backgrounds are not sprites and do not go through the atlas, but they are 18 of the 152 `PICT`s and
54% of the art bytes (1,037,846 of 1,919,598), so an extractor has to handle them. The room
composition model itself is specified in `docs/analysis/rendering.md:1739` (§6); repeated here are
only the constants and the measurements an extractor needs, plus the tile-strip consequence for
custom backgrounds.

The model, from `DrawRoomBackground` (`GliderPRO/Sources/RoomGraphics.c:162-253`):

| Constant | Value | Cite |
|---|---|---|
| `kNumTiles` | 8 | `GliderPRO/Headers/GliderDefines.h:496` |
| `kTileWide` | 64 | `:497` |
| `kTileHigh` | 322 | `:498` |
| `kRoomWide` | 512 (= 8 x 64) | `:499` |
| `kNumBackgrounds` | 18 | `:521` |
| `kUserBackground` | 3000 | `:522` |
| `kMaxViewWidth` / `kMaxViewHeight` | 1536 / 1026 | `:267-268` |
| `kScoreboardTall` | 20 | `:515` |

Pseudocode, preserving the original's control flow:

```
 1. if where == kCentralRoom: cache thisBackground and thisTiles[0..7] from the room record
 2. if numLights == 0 and who != kRoomIsEmpty:            # dark room
 3.     PaintRect(localRoomsDest[where]) into backSrcMap; return    # fore colour = black; no art
 4. if who == kRoomIsEmpty:
 5.     if wardBitSet: PaintRect and return
 6.     elif elevation >  1: pictID := kSky    (2015); tiles[0..7] := 2
 7.     elif elevation == 1: pictID := kMeadow (2012); tiles[0..7] := 0
 8.     else:                pictID := kDirt   (2011); tiles[0..7] := 0
 9. else:
10.     pictID := thisHouse->rooms[who].background ; tiles[] := thisHouse->rooms[who].tiles[]
11. SetPort(workSrcMap); LoadGraphicSpecial(pictID)       # whole 512x322 into the scratch GWorld
12. src  := (0,0,64,322) ; dest := (0,0,64,322) offset to localRoomsDest[where].topLeft
13. for i := 0 to 7:
14.     src.left  := tiles[i] * 64 ; src.right := src.left + 64
15.     CopyBits(workSrcMap -> backSrcMap, src, dest, srcCopy, nil)
16.     QOffsetRect(dest, 64, 0)
```

Measured: **all 18 backgrounds are exactly 512x322** — verified by parsing every `picFrame` in
2000-2017. So the tile strip is the full picture, and `tiles[i]` selects which of the eight 64-wide
columns is drawn at slot *i*. A room whose `tiles` are `{0,1,1,1,1,1,1,7}` (the default for the nine
interior backgrounds, `GliderPRO/Sources/Room.c:56-84`) draws a left wall, six repeats of a middle
column, and a right wall.

Three consequences for extraction:

* Emit backgrounds **whole**, at 512x322, opaque, one PNG per ID. Do not pre-split them into eight
  tiles: the tile indices are per-room house data, not asset data.
* Custom (house) backgrounds go through the **identical** path — `LoadGraphicSpecial` then the same
  eight `CopyBits` — so they have the same 512x322 requirement (§7.12 item 6).
* `workSrcMap` is `houseRect`-sized, i.e. `min(screen, 1536x1026)` minus the 20-pixel scoreboard
  (`GliderPRO/Sources/InterfaceInit.c:196-201`, `GliderPRO/Sources/StructuresInit2.c:156-158`). A
  background is drawn into it at (0,0) with no scaling, so on a 512-wide display the strip would be
  clipped. In practice `kMaxViewWidth` 1536 ≫ 512, so this never bites; a Go port should size the
  scratch surface at `max(512, viewWidth)` and not inherit the coupling.

### 7.14 Scaling: where it happens, and where it must not

`LoadScaledGraphic` maps a picture onto an arbitrary rect and therefore scales. Every call site was
enumerated; **not one of them scales object art or background art at play time**:

| Call site | `PICT` | Target rect | Scales? |
|---|---|---|---|
| `GliderPRO/Sources/RoomGraphics.c:282, 300, 317, 335, 353, 371` | 3957 `kManholeThruFloor` | `tempManholes[i]`: width from `srcRects[kManhole]` = 123, height forced to `kFloorSupportTall` = 44 (`GliderPRO/Sources/Objects.c:351-362`, `GliderPRO/Headers/GliderDefines.h:500`) | **no** — 3957 measures 123x44 exactly |
| `GliderPRO/Sources/MainWindow.c:106`, `:143`, `GliderPRO/Sources/Play.c:271` | 1000 `kSplash8BitPICT` | `QSetRect(&tempRect, 0, 0, 640, 460)` | **no** — 1000 measures 640x460 |
| `GliderPRO/Sources/GameOver.c:87` | 1021 `kMilkywayPictID` | full-screen temp rect | no — 1021 measures 640x460 |
| `GliderPRO/Sources/HighScores.c:67` | 1995 `kStarPictID` | full-screen temp rect | no — 1995 measures 640x460 |
| `GliderPRO/Sources/Input.c:85`, `:87` | 1015 / 1016 pause banners | 214x54 | no — both measure 214x54 |
| `GliderPRO/Sources/Banner.c:61` | 1993 `kBannerPageTopPICT` | `partPage` | no — 1993 measures 330x190 |
| `GliderPRO/Sources/Banner.c:224`, `:227` | 1018 / 1017 | 256x64 | no — both measure 256x64 |
| `GliderPRO/Sources/RoomInfo.c:418`, `:541` | 1202/1211/1216/1217 | `tileSrcRect` 128x80 | **no** — all four measure 128x80 |
| `GliderPRO/Sources/RoomInfo.c:420`, `:514`, `:543`, `:555` | a background 2000-2017, or a house `PICT` | `tileSrcRect` 128x80 | **YES** — 512x322 → 128x80 |

So the **only genuine scaling in the entire program is the Room Info background thumbnail**, and it is
editor UI, not gameplay. Combined with §6.8 finding 10 (all 38 object `PICT`s have `srcRects` entries
whose W x H equals the picture's `picFrame` W x H exactly, 38/38, and all 8 paired masks match their
art exactly), the invariant a Go port can rely on is: **blit 1:1 and never implement PICT scaling**,
except for one editor thumbnail where any box filter is acceptable because the original used
QuickDraw's nearest-neighbour and nobody will notice.

### 7.15 What must not be extracted

Three object types have no pixels anywhere. Emitting a PNG for them would silently substitute a wrong
sprite for code that has to be ported:

| `what` | Constant | Drawn by | What it draws |
|---|---|---|---|
| 0x17 | `kCounter` | `DrawCounter` | filled and framed rects only |
| 0x82 | `kMirror` | `DrawMirror` | four `ColorFrameRect` insets; the reflection is a live `srcXor` composite through `mirrorRgn` |
| 0x86 | `kWallWindow` | `DrawWallWindow` | sill and frame rects, `kWindowSillThick` = 7 |

Plus the 13 `none` types (invisible blowers, obstacles, bounces, triggers, lift areas, sliders,
deluxe transports) whose `srcRects` entry is only the default size the editor gives a new one, and the
12 `tmpl` types whose art is in named sub-rects (§7.7). The extractor writes no file for any of these
28 and records the reason in the manifest.

Also do not extract:

* **Mask `PICT`s as standalone images.** They are consumed into alpha. The 20 vestigial mask IDs
  3900-3926 that `#define` but do not exist (§5.6) will make a naive "extract every `k*MaskID`"
  loop fail on 20 of 28.
* **`clut` 128/129 as art.** They are the palette, and identical to each other.
* **The `switch` sheet's mask.** `PICT` 5003 does not exist and the sheet is drawn with opaque
  `srcCopy` (§6.8 finding 3). Alpha is 100% by construction, not by measurement error.

### 7.16 Risk ranking for a port of this pipeline

Ordered by (probability of getting it wrong) x (how quiet the failure is):

1. **`QSetRect` argument order.** `(l, t, r, b)`, but the `Rect` struct is `(t, l, b, r)`
   (`GliderPRO/Sources/RectUtils.c:210`). Swapping them produces rects that are usually still valid
   and crop *something*. Every rect in §6 is printed in `QSetRect` order; `rendering.md` §7.4 prints
   the same rects in struct order. Pick one and assert.
2. **The index plane vs the RGB canvas.** The white key is index 0; testing RGB equality happens to
   work for this palette and will stop working the moment anything is recoloured or filtered (§7.4).
3. **Sizing surfaces from the `PICT` instead of the `QSetRect`.** Furniture is 64x278 declared, 64x221
   on disk; the appliance mask is one row short (§6.2). Either choice works if made consistently;
   mixing them shifts every furniture sub-rect.
4. **`srcRects[kFlower]` (0x85).** Never assigned, never read. A `[144]Rect` zero-valued at index
   0x85 produces a zero-size flower and no visible error (§6.5).
5. **The 28 unassigned slots generally.** `NewPtr`, not `NewPtrClear`
   (`GliderPRO/Sources/StructuresInit2.c:271`), so the original reads heap garbage for those. Carry a
   `valid [144]bool` and panic on an invalid read; the census in `house-format.md:1202-1207` proves
   no undefined `what` occurs in any of the 4,070 rooms of the 22 shipped houses, so the panic is
   unreachable for shipped content and will catch corrupt input.
6. **`DrawFish` reads the wrong sheet.** The static draw uses `enemySrcMap` 4016/5016, *not* the
   8-frame `fishSrcMap` 4017/5017 (§6.8 finding 1). Two different fish, both 4-something.
7. **`DrawDrip` hard-codes `&dripSrc[3]`**, not `srcRects[kDrip]` (§6.8 finding 2).
8. **The five switch draws bypass the atlas** with `srcCopy`, no mask, and
   `srcRects[kMachineSwitch]` (0,48,16,72) disagrees with `machineSwitchSrc[0]` (0,24,16,48)
   (§6.8 finding 3). Trusting `srcRects` here draws the thermostat art on the machine switch.
9. **Mask polarity.** 1 = opaque (§5.2). Inverting it yields a plausible-looking negative-space
   sprite.
10. **`kCustomPict`'s `data.g.height` is a resource ID.** Treating it as a height gives an object
    72 pixels wide and `height` pixels tall that draws the wrong picture or none (§7.12 item 4).
11. **`packType == 0` means "default for the depth", not "unpacked".** At 8 bpp with
    `rowBytes >= 8` that is PackBits; at `rowBytes < 8` it is raw. Getting this backwards produces
    garbage on exactly the small sprites (§3.8).
12. **Per-row length prefix width.** `uint16` if `rowBytes > 250`, else `uint8` (§3.8). Wrong choice
    desynchronises the whole raster and is immediately obvious — the least dangerous item on this
    list.

---

## Open questions

1. **What is the `'Date'` resource type?** `LoadGraphicSpecial`
   (`GliderPRO/Sources/RoomGraphics.c:142`) and `PictIDExists`
   (`GliderPRO/Sources/RoomInfo.c:845`) both accept a `'Date'` resource in place of a `PICT`, and
   `LoadGraphicSpecial` then casts it to `PicHandle` and draws it. No `'Date'` resource exists in
   `Glider PRO.r` and none appears in any shipped house
   (`docs/analysis/resource-fork.md` lists 35 types, none of them `'Date'`). Whether any third-party
   house in the wild uses it is unknowable from this source tree. A port should treat it as "ID
   exists, no pixels".
2. **Why is `srcRects[kFlower]` the only gap inside a group?** The other 27 unassigned slots are all
   at group boundaries (the padding between `kLiftArea` 0x10 and `kTable` 0x11 etc.), which is a
   deliberate numbering scheme. 0x85 sits between `kFireplace` 0x84 and `kWallWindow` 0x86 with no
   padding rationale (§6.5). Most likely flowers were converted to the `flowerSrc[]` variant scheme
   after `InitSrcRects` was written and the line was deleted rather than left as dead weight, but
   there is no comment or changelog to confirm it.
3. **Was the `switch` sheet ever masked?** `PICT` 4003 exists, 5003 does not, the mask GWorld
   `switchMaskMap` is not even declared (`GliderPRO/Headers/Objects.h:18` declares
   `switchSrcMap` and `:19` goes straight on to `lightSrcMap`, with no mask sibling, unlike every other pair), and all five draws use `srcCopy`.
   Either it was never masked, or the removal was thorough enough to erase the variable. The 100%
   opaque measurement is consistent with both.
4. **Why do 20 masks survive as `#define`s (3900-3926) with no resources?** §5.6 establishes the
   correlation exactly — the 20 missing masks are precisely the 20 colour-keyed objects — so the
   direction of the change is clear (mask → key). What is not recoverable is whether the motivation
   was file size (the 20 masks would be roughly 10 KB) or authoring convenience.
5. **What are `PICT` 1994, 1998, and the `ozm5` resource for?** They fall in the `ui` bucket by
   elimination rather than by a located call site. `resource-fork.md` §3.1.1 groups them as
   "game-over / banner / scoreboard art"; the specific draw calls were not traced because they are
   outside this document's scope.
6. **Does any shipped house actually use a `kCustomPict` or a `kUserBackground`?** The extraction
   pipeline here handles only the application fork. `original-houses.md` §1.5 documents which houses
   embed `PICT`s, but the per-object question — how many `kCustomPict` instances exist across the 22
   houses and whether their `data.g.height` IDs all resolve — was not measured here.
7. **Is the `PackBitsRgn` mask region ever non-trivial?** All three occurrences (`PICT` 3968, 3969,
   3995) have a region equal to the picture rect, so ignoring the region is measurably correct for
   this file. Whether QuickDraw would have clipped differently for a complex region is untested
   because there is no test case.
8. **Which tool wrote the 21 extended-v2 pictures?** The `HeaderOp` `version == -2` set and the
   `ctFlags == 0x0000` set are byte-for-byte the same 21 IDs (§3.2 consequence 4, §7.5), and the
   `DefHilite` opcode is present in all 95 standard-v2 pictures and absent from all 21 extended ones.
   That is three independent signals of a second authoring path, but there is nothing in the source
   tree naming it. Immaterial to the port — all three differences are inert — but it explains why the
   opcode histogram is not uniform across the art set.

---

## Porting notes

**Toolbox → Go, subsystem by subsystem.**

| Mac Toolbox | What it does here | Go replacement |
|---|---|---|
| Resource Manager (`GetPicture`, `UseResFile`, `Count1Resources`) | two-level ID lookup, house fork in front of app fork | a `map[ResType]map[int16][]byte` per fork plus an ordered chain; `Get1*` = search head only, `Get*` = search all |
| `GWorldPtr` / `CreateOffScreenGWorld(&gw, &bounds, 8)` | 8-bit indexed offscreen surface, white-initialised | a struct with `Pix []uint8` (palette indices), `Stride`, `Rect`; initialise to 0 (white), *not* to 0-value-means-black |
| 1-bit mask GWorlds | stencils | fold into alpha at load; keep no 1-bit surfaces at runtime |
| `CopyBits(..., srcCopy)` | opaque blit | `copy` per row, or `draw.Src` |
| `CopyBits(..., transparent)` | white colour key | pre-baked alpha + `draw.Over` |
| `CopyBits(..., srcXor)` | the mirror reflection | keep as an index-space XOR; it has no RGBA equivalent |
| `CopyMask(src, mask, dst, sRect, mRect, dRect)` | 1-bit stencil blit | `draw.DrawMask`, or a hand-rolled alpha blit since alpha is binary |
| `DrawPicture(pic, &rect)` | decode + blit, scaling to `rect` | decode once at startup; never scale (§7.14) |
| Palette Manager / `Index2Color` | index → RGB | the hard-coded 256-entry table of §4.2 |
| `PicHandle`, `HLock`/`HUnlock` | relocatable handles | irrelevant; Go slices |
| big-endian on-disk `Rect` = `{int16 top, left, bottom, right}` | every rect in every resource | `binary.BigEndian`; keep a named type and a `FromQSetRect(l,t,r,b)` constructor so the ordering bug of risk 1 is unrepresentable |

**Decisions this document recommends making once, at load time:**

1. **Bake alpha into every sheet.** Produce one RGBA surface per sheet from the art/mask pair. This
   is valid because `CopyMask` always uses the same rect for source and mask
   (`GliderPRO/Sources/ObjectDraw.c:61-64`), and it deletes the entire mask-GWorld subsystem.
2. **Represent the atlas as `[144]image.Rectangle` plus `valid [144]bool`.** 116 valid, 28 invalid.
   Panic on an invalid read rather than returning a zero rect (risk 4/5).
3. **Do not implement PICT scaling.** 38/38 object pictures are 1:1 with their `srcRects` entry; the
   only scaling in the program is one editor thumbnail (§7.14).
4. **Implement exactly nine PICT opcodes and hard-error on the tenth** (§7.2, §7.3), except for
   user-supplied house art, where the fallback chain of §7.12 item 3 applies with step 2 removed.
   Gate on the *opcode*, never on the `HeaderOp` version field: both `-1` (95 pictures) and `-2`
   (21 pictures, including the glider sheet 3999) ship, both payloads are 24 bytes, and both are
   ignored (§3.2 consequence 4).
5. **Keep the palette-index plane, not just RGB.** The white key, the mirror's `srcXor`, and any
   future palette animation are all index-space operations (§7.4).
6. **Load the 14 sheets from the `QSetRect` bounds, not the `PICT` bounds**, and let the two
   documented mismatches (furniture 278 vs 221, appliance mask 268 vs 269) be transparent slack
   (§6.2, §3.12.6).
7. **Port the three procedural objects as code, not art** (§7.15), and port `kTiki`'s pole, `kStool`'s
   legs, the clock hands and the `kCalendar` month text the same way — they are `ColorLine`/
   `ColorFrameRect`/`DrawString` calls, not pixels.
8. **Treat `data.g.height` on a `kCustomPict` as a `int16` resource ID**, and reproduce the 10000
   fallback in play mode only. Do **not** reproduce the editor's persistent rewrite
   (`GliderPRO/Sources/ObjectEdit.c:2285`) — it silently mutates the user's house.
9. **Store extracted art keyed by `(what, constant)`, not by `PICT` ID.** IDs are shadowable by house
   forks; the object type is not.
10. **Fix the two original bugs deliberately, and note them:** the missing `-800` thumbnail
    substitution (`GliderPRO/Sources/RoomInfo.c:514`, `:555`) and the uninitialised
    `SetGWorld(wasCPort, wasWorld)` in `InitTransports` (`GliderPRO/Sources/StructuresInit.c:440`,
    §6.8 finding 4). Neither is observable in the shipped game, but both are undefined behaviour that
    a Go port cannot express and should not emulate.
