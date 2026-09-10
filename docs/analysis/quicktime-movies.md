# Glider PRO 1.0.4 — the QuickTime TV movies

## Scope

Every house in Glider PRO may ship a QuickTime movie that plays on the screen of a
`kTV` object. This document covers that feature end to end:

1. the build-time and run-time gating that decides whether movies exist at all;
2. the complete state machine (`hasMovie` / `tvInRoom` / `tvWithMovieNumber` / `tvOn`)
   and every site that reads or writes it;
3. every Mac Toolbox call the feature makes, what it does, and what a Go port must
   put in its place;
4. the exact geometry — where the movie is placed, how it is centred, how it is
   clipped, with the arithmetic worked through for all three distinct movie sizes;
5. **the shipped movie data itself, decoded byte by byte** — the QuickTime container
   as actually used, the three video codecs actually present (`rle `, `raw `, `smc `),
   the palettes, and a per-file measurement table for all 15 `.mov` files;
6. the headline empirical finding: **14 of the 15 shipped movies have lost their movie
   header** (`moov`) and are unplayable by any conforming QuickTime implementation, and
   why — plus how this document recovers every frame of all 14 anyway;
7. answers to the four porter questions left open by
   [`rendering.md` §19.7](rendering.md), and the answer that closes
   [`original-houses.md` open question 6](original-houses.md).

Section 19 of `docs/analysis/rendering.md` (lines 6114-6301) already documents the
*rendering* side of this feature: `OpenHouseMovie`, `LoopMovie`, `PrerollMovie`,
`tvWithMovieNumber`, the first-TV-only rule, and how `MoviesTask` interacts with the
dirty-rect pipeline. This document does not replace it. It repeats the parts needed to
make the movie-data discussion self-contained, and it goes past it in two directions:
the on-disk format (which §19 does not touch at all) and the full source-side inventory
of every `kTV`-related call site.

Everything measured here was produced by `tools/probe_mov.py`, written for this
document. Re-run it; do not trust the tables if they disagree with the tool.

## Sources read

Source line numbers are from the shipped files with classic-Mac CR line endings
converted to LF (a 1:1 conversion — line numbering is unchanged).

| File | What it contributes |
|---|---|
| `GliderPRO/Sources/HouseIO.c` | `LoopMovie` (:48-63), `OpenHouseMovie` (:67-142), `CloseHouseMovie` (:146-159), the reset in `OpenHouse` (:196-199), `CloseHouseMovie` on close (:550), `YellowAlert` (:640-655) |
| `GliderPRO/Sources/ObjectDrawAll.c` | the `kTV` case (:674-713): movie placement (:679-695) and dynamic registration (:703-710) |
| `GliderPRO/Sources/ObjectDraw2.c` | `DrawTV` (:640-689): the TV sprite and the static screen blits |
| `GliderPRO/Sources/Dynamics.c` | `HandleTV` (:428-478): the two empty `if` bodies at :439-440 and :451-452 that suppress the static screen for the movie TV, and the unguarded off-blit at :467-475 |
| `GliderPRO/Sources/Dynamics3.c` | `HandleDynamics` dispatch (:51-53) and `AddDynamicObject`'s `kTV` case (:264-282), `active = isOn` at :281 |
| `GliderPRO/Sources/Trip.c` | `ToggleTV` (:40-58) |
| `GliderPRO/Sources/Play.c` | `tvOn` definition (:54), `SetMovieGWorld` (:147-152), game-start (:201-211), game-end (:223-230), `MoviesTask` before `RenderFrame` (:465-468, :490-493), `data.g.state = data.g.initial` reset (kTV case label :683, assignment :688-689) |
| `GliderPRO/Sources/Transit.c` | four post-transition restarts (:302-309, :340-347, :379-386, :418-425) |
| `GliderPRO/Sources/RoomGraphics.c` | `DrawLocale`'s per-call reset (:59-60; function starts :44), its nine `DrawARoomsObjects(..., false)` calls (:82, :86, :90, :94, :98, :102, :109, :114, :120), `ReadyLevel`'s stop (:406-413; function starts :402), the dead `RedrawRoomLighting` (:434-461) and its lone `DrawARoomsObjects(kCentralRoom, true)` (:451) |
| `GliderPRO/Sources/StructuresInit.c` | `tvScreen1` / `tvScreen2` (:570-573) |
| `GliderPRO/Sources/StructuresInit2.c` | `srcRects[kTV]` (:436) |
| `GliderPRO/Sources/ObjectRects.c` | `GetObjectRect` for `kTV` (case label :204, shared body :213-218), hot-spot rect (case label :983, body :990-996), `OffsetRectRoomRelative` (:1093-1131) |
| `GliderPRO/Sources/RectUtils.c` | `ZeroRectCorner` (:57-63), `CenterRectInRect` (:142-154) |
| `GliderPRO/Sources/Environ.c` | `DoWeHaveQuickTime` (:196-209), called at :452 |
| `GliderPRO/Sources/Main.c` | `EnterMovies()` (:330-335) |
| `GliderPRO/Sources/MainWindow.c` | the `" (QT)"` splash suffix (:60-63, inside `DrawOnSplash` at :56) |
| `GliderPRO/Sources/Interactions.c` | `ToggleTV` dispatch from a switch/band (:1101-1103) |
| `GliderPRO/Sources/Objects.c` | `tvWithMovieNumber` definition (:77), `SetObjectState` (:593), `GetObjectState` (:822) |
| `GliderPRO/Sources/Render.c` | dead `extern movieRect` (:55) and `extern tvOn` (:59) |
| `GliderPRO/Headers/GliderDefines.h` | `COMPILEQT` (:15), `kYellowQTMovieNotLoaded` = 22 (:39), `kTVOnSound` = 32 / `kTVOffSound` = 33 (:87-88), `kTVOnPriority` = 404 / `kTVOffPriority` = 405 (:145-146), `kCentralRoom` = 0 (:217), `kRoomWide` = 512 (:499), `kVertLocalOffset` = 322 (:501) |
| `GliderPRO/Headers/GliderVars.h` | `#include <Movies.h>` (:8), `extern Movie theMovie` (:42), `extern Rect movieRect` (:43), `extern Boolean hasMovie, tvInRoom` (:44) |
| `GliderPRO/Headers/Environ.h` | `Boolean hasQT` in `macEnviron` (:30) |
| `GliderPRO/Glider PRO.r` | `STR#` 1006 string 22 (the yellow-alert text), `clut` 128 (the palette) |
| `GliderPRO/Houses/*.mov` | the 15 movie files — the subject of sections 10-18 |
| `GliderPRO/Houses/*.binhex` | the 22 house files — `kTV` placements (section 9.5) |
| `GliderPRO/upstream.git` | commit history proving *when and how* the movie headers were lost (section 11.5) |

Tooling: `tools/probe_mov.py` (new, written for this document), `tools/probe_rez.py`
and `tools/probe_pict.py` (for `clut` 128 and the PNG writer), `tools/probe_house.py`
(for the `kTV` census).

---

## 1. The feature in one paragraph

A house may be accompanied by a sibling file `<house name>.mov`. If QuickTime is
installed, Glider PRO loads that movie entirely into RAM when the house is opened,
marks it to loop forever, and binds it to the game window. While playing, if the
*central* room contains a `kTV` appliance, the **first** such TV found becomes "the
movie TV": the movie's box is centred on that TV's 64 x 49 screen rectangle, clipped
to it, and QuickTime draws frames straight into the on-screen window once per game
frame — bypassing Glider's entire offscreen compositing pipeline. Flipping the TV's
switch rewinds and starts, or stops, the movie. Every other TV in the house, and every
TV in a house with no `.mov`, shows a static "snow" bitmap instead.

---

## 2. Gating: three independent switches

### 2.1 Build time — `COMPILEQT`

```
GliderPRO/Headers/GliderDefines.h:15:   #define COMPILEQT
```

Defined unconditionally, so the shipped 1.0.4 build contains the movie code. Every
movie-touching block in the game is wrapped in `#ifdef COMPILEQT`. The complete list of
`#ifdef COMPILEQT` blocks:

| File:line | Block |
|---|---|
| `HouseIO.c:69` … `:141` | body of `OpenHouseMovie` |
| `HouseIO.c:148` … `:157` | body of `CloseHouseMovie` |
| `ObjectDrawAll.c:679` … `:695` | movie placement in the `kTV` draw case |
| `ObjectDrawAll.c:703` … `:710` | `tvWithMovieNumber` / `tvInRoom` registration |
| `Play.c:147` … `:152` | `SetMovieGWorld` |
| `Play.c:201` … `:211` | game-start activate/start |
| `Play.c:223` … `:230` | game-end stop/deactivate |
| `Play.c:465` … `:468` | `MoviesTask` (two-player loop branch) |
| `Play.c:490` … `:493` | `MoviesTask` (one-player loop branch) |
| `Transit.c:302` … `:309`, `:340` … `:347`, `:379` … `:386`, `:418` … `:425` | `RenderFrame()` + post-transition restart |
| `RoomGraphics.c:406` … `:413` | `ReadyLevel` stop |

Note that `Dynamics.c:437-444` and `:449-460`, `Trip.c:43-56` and `MainWindow.c:62`
are **not** `#ifdef`-guarded; they rely on `hasMovie` being `false` when `COMPILEQT` is
undefined. That works because `hasMovie` is a real variable declared outside the
`#ifdef` (`HouseIO.c:36`) and set `false` unconditionally by `CloseHouseMovie`
(`HouseIO.c:158`) and `OpenHouse` (`HouseIO.c:196`). A Go port that keeps a
compile-time switch must preserve exactly that asymmetry, or the TV's static-screen
blits change behaviour.

### 2.2 Run time — `thisMac.hasQT`

```
GliderPRO/Sources/Environ.c:196-209

  Boolean DoWeHaveQuickTime (void)
  {
      theErr = Gestalt(gestaltQuickTime, &theResponse);
      if (theErr == noErr)  haveIt = true;
      else                  haveIt = false;
      return (haveIt);
  }
```

Called once from `CheckEnvirons` (`Environ.c:452`: `thisMac.hasQT = DoWeHaveQuickTime();`).
`theResponse` — which holds the QuickTime version in its low word — is *fetched and
discarded*. There is no minimum-version check anywhere in the game.

`hasQT` is a `Boolean` field of `macEnviron` (`GliderPRO/Headers/Environ.h:30`).

### 2.3 Component initialisation — `EnterMovies`

```
GliderPRO/Sources/Main.c:330-335

  if (thisMac.hasQT)
  {
      theErr = EnterMovies();
      if (theErr != noErr)
          thisMac.hasQT = false;
  }
```

`EnterMovies` is called *after* `OpenMainWindow()` and *before* `InitSound()`,
`InitMusic()`, `BuildHouseList()` and the first `OpenHouse()` — which matters, because
`OpenHouse` calls `OpenHouseMovie`, so the Movie Toolbox must already be initialised.
There is no matching `ExitMovies()` anywhere in the source: the game relies on process
teardown.

**Go replacement.** All three switches collapse into one boolean the port sets itself.
Keep the variable (`hasQT`) so the `" (QT)"` splash suffix and the `HandleTV`
suppression logic stay expressible; there is no reason for it ever to be false in a Go
build unless the port wants to offer a "no movies" preference.

---

## 3. State variables

| Variable | Type | Defined | Meaning |
|---|---|---|---|
| `theMovie` | `Movie` (Toolbox opaque handle) | `HouseIO.c:31` | the single loaded movie; there is never more than one |
| `movieRect` | `Rect` | `HouseIO.c:32` | the movie box. Initially the movie's natural box from `GetMovieBox`; overwritten every time `DrawLocale` places it, at which point it is in **window** coordinates |
| `hasMovie` | `Boolean` | `HouseIO.c:36` | a `<house>.mov` exists, opened, and loaded into RAM |
| `tvInRoom` | `Boolean` | `HouseIO.c:36` | the current *central* room contains the movie-bearing TV |
| `tvWithMovieNumber` | `short` | `Objects.c:77` | index into `dinahs[]` of that TV, or -1 |
| `tvOn` | `Boolean` | `Play.c:54` | the movie TV's power state (mirrors `dinahs[n].active`) |
| `thisMac.hasQT` | `Boolean` | `Environ.h:30` | QuickTime present and `EnterMovies` succeeded |

Externs are declared in `GliderPRO/Headers/GliderVars.h:42-44` (`theMovie`,
`movieRect`, `hasMovie`, `tvInRoom`); `tvWithMovieNumber` and `tvOn` are declared
`extern` ad hoc in each user (`Trip.c:15-16`, `Dynamics.c:24`, `RoomGraphics.c:37`,
`Transit.c:25`, `ObjectDrawAll.c:16-17`, `HouseIO.c:39`, `Render.c:55` and `:59`). Note
that several of these are *combined* declarations covering unrelated variables, so the
line contains more than the movie name: `ObjectDrawAll.c:14` is
`extern Rect localRoomsDest[], movieRect;`, `ObjectDrawAll.c:16` is
`extern short numLights, tvWithMovieNumber;`, `HouseIO.c:39` is
`extern short thisHouseIndex, tvWithMovieNumber;`, `Render.c:55` is
`extern Rect shadowSrc[], justRoomsRect, movieRect;` and `Render.c:59` is
`extern Boolean evenFrame, shadowVisible, twoPlayerGame, tvOn;`.

`Render.c:55` and `:59` declare `movieRect` and `tvOn` `extern` but never use them —
dead declarations, presumably left from an abandoned attempt to render the movie
through the normal pipeline.

### 3.1 The guard expression

Almost every site uses the same conjunction. There are exactly four shapes:

| Guard | Sites |
|---|---|
| `thisMac.hasQT` | `HouseIO.c:78`, `Main.c:330` |
| `thisMac.hasQT && hasMovie` | `HouseIO.c:151`, `Play.c:148`, `MainWindow.c:62` |
| `thisMac.hasQT && hasMovie && tvInRoom` | `Play.c:202`, `Play.c:224`, `RoomGraphics.c:407` |
| `thisMac.hasQT && hasMovie && tvInRoom && tvOn` | `Play.c:466`, `Play.c:491`, `Transit.c:304`, `:342`, `:381`, `:420` |
| `thisMac.hasQT && hasMovie && (neighbor == kCentralRoom) && !tvInRoom` | `ObjectDrawAll.c:680-681`, `:704-705` |
| `thisMac.hasQT && hasMovie && tvInRoom && who == tvWithMovieNumber` | `Dynamics.c:437-438`, `:449-450` |
| `thisMac.hasQT && hasMovie && tvInRoom && tvWithMovieNumber == index` | `Trip.c:43` |

A Go port should factor this into one predicate per shape rather than open-coding it
seven times; the shapes are genuinely different and collapsing them all into one is a
behaviour change.

### 3.2 Reset points

`tvInRoom` and `tvWithMovieNumber` are reset in three places:

| Site | Code |
|---|---|
| `RoomGraphics.c:59-60`, at the **top of `DrawLocale`** (function starts `:44`) | `tvInRoom = false; tvWithMovieNumber = -1;` |
| `RoomGraphics.c:409-410` (`ReadyLevel` at `:402`, inside the QT block `:406-413`) | `tvInRoom = false; tvWithMovieNumber = -1;` |
| `HouseIO.c:196-198` (`OpenHouse`) | `hasMovie = false; tvInRoom = false; tvWithMovieNumber = -1;` |

Plus `Play.c:226` sets `tvInRoom = false` at game end (without touching
`tvWithMovieNumber`, which is harmless because `tvInRoom` gates every use).

The first row is the structurally important one: **`DrawLocale` clears `tvInRoom` and
`tvWithMovieNumber` at its own entry**, at `RoomGraphics.c:59-60`, before it scans the
central room and its eight neighbours. So the `!tvInRoom` guard in the `kTV` draw case
(`ObjectDrawAll.c:681`, `:705`) is a *within-one-`DrawLocale`* latch: it means "no TV has
claimed the movie yet **on this room draw**", not "no TV has ever claimed it". Every room
draw re-runs the first-TV election from scratch.

`ReadyLevel` (`RoomGraphics.c:402`) then stops the movie and clears the same two
variables at `:409-410` — belt and braces, since it calls `NilSavedMaps()` at `:404` and
`DrawLocale` will clear them again — but its real job is the `StopMovie(theMovie)` at
`:411`, so the previous room's movie is not still running while the new room is composed.

A Go port must keep the reset at the top of the room-draw function. Hoisting it to
"once per house" turns the first-TV rule into a first-TV-*ever* rule and the movie stops
following the player between rooms.

---

## 4. `OpenHouseMovie` — the load path

`GliderPRO/Sources/HouseIO.c:67-142`. Numbered pseudocode following the original
control flow; original identifiers in parentheses.

```
 1  if (!thisMac.hasQT) return                                            // :78
 2  theSpec = theHousesSpecs[thisHouseIndex]                              // :80
 3  PasStringConcat(theSpec.name, "\p.mov")        // "Demo House" -> "Demo House.mov"  // :81
 4  if (FSpGetFInfo(&theSpec, &finderInfo) != noErr) return  // silent: no movie  // :83-85
 5  if (OpenMovieFile(&theSpec, &movieRefNum, fsCurPerm) != noErr) {       // :87
 6      YellowAlert(kYellowQTMovieNotLoaded, theErr); return }             // :90-91
 7  if (NewMovieFromFile(&theMovie, movieRefNum, nil, theSpec.name,        // :94-95
 8                       newMovieActive, &dataRefWasChanged) != noErr) {
 9      YellowAlert(kYellowQTMovieNotLoaded, theErr)                       // :98
10      CloseMovieFile(movieRefNum); return }                              // :99-100
11  CloseMovieFile(movieRefNum)                                           // :102
12  spaceSaver = NewHandle(307200L)                                       // :104
13  if (spaceSaver == nil) {                                              // :105
14      YellowAlert(kYellowQTMovieNotLoaded, 749)   // 749 = a made-up code  // :107
15      CloseHouseMovie(); return }                                       // :108-109
16  GoToBeginningOfMovie(theMovie)                                        // :112
17  if (LoadMovieIntoRam(theMovie, GetMovieTime(theMovie, 0L),            // :113-114
18                       GetMovieDuration(theMovie), 0) != noErr) {
19      YellowAlert(kYellowQTMovieNotLoaded, theErr)                       // :117
20      DisposeHandle(spaceSaver); CloseHouseMovie(); return }             // :118-119
21  DisposeHandle(spaceSaver)                                             // :122
22  if (PrerollMovie(theMovie, 0, 0x000F0000) != noErr) {                 // :124
23      YellowAlert(kYellowQTMovieNotLoaded, theErr)                       // :127
24      CloseHouseMovie(); return }                                       // :128-129
25  theTime = GetMovieTimeBase(theMovie)                                  // :132
26  SetTimeBaseFlags(theTime, loopTimeBase)                               // :133
27  SetMovieMasterTimeBase(theMovie, theTime, nil)                        // :134
28  LoopMovie()                                                          // :135
29  GetMovieBox(theMovie, &movieRect)                                     // :137
30  hasMovie = true                                                       // :139
```

Observations that matter for a port:

- **Step 3 builds the filename by Pascal-string concatenation.** `theHousesSpecs[]` is
  an array of `FSSpec`, and only `.name` is modified — `vRefNum` and `parID` are
  inherited, so the movie must be in the same directory as the house file. Note the
  house file itself has **no** extension (`Houses/Demo House`), so the movie is
  `Houses/Demo House.mov`. `PasStringConcat` silently truncates at 255 bytes; house
  names are capped well below that.
- **Step 4 is an existence test only.** `finderInfo` is filled in and then never read.
  The file's type/creator are not checked — a file named `<house>.mov` that is not a
  movie will fail at step 5 or 7 with an alert instead.
- **Step 4 returns silently.** A house with no `.mov` produces no alert and no log.
  Every *other* failure raises `YellowAlert`. So "no movie" and "broken movie" are
  distinguishable to the user.
- **Step 12/21: `NewHandle(307200L)` then `DisposeHandle`.** 307200 = 640 x 480 x 1
  byte = one 8-bit full-screen buffer. It is allocated and immediately freed purely to
  *prove* that much contiguous memory is still available after the movie is in RAM. If
  it fails, the movie is dropped with alert identifier **749** — a magic number with no
  `#define`, not an `OSErr`. A Go port has no equivalent constraint and should drop
  steps 12-15 and 21 entirely.
- **Step 17: `LoadMovieIntoRam(..., 0)`** with flags 0 (i.e. neither `keepInRam` nor
  `flushFromRam`) over the movie's whole duration. This is what makes the whole movie
  resident; the game never streams from disk.
- **Step 22: `PrerollMovie(theMovie, 0, 0x000F0000)`.** The third argument is a
  `Fixed` (16.16) *rate*: `0x000F0000` = 15.0. That is a hint about how fast the movie
  will be played, used to size QuickTime's read-ahead; it is not the playback rate. All
  the shipped movies play at rate 1.0. 15.0 is 15x — almost certainly a copy/paste of a
  "15 fps" idea into a rate argument. Harmless.
- **Steps 25-28 set up looping twice**, by two independent mechanisms:
  - `SetTimeBaseFlags(theTime, loopTimeBase)` makes the *time base* wrap;
  - `LoopMovie()` (below) installs a `'LOOP'` user-data item, which is the convention
    the *Movie Controller* reads.
  Since the game never creates a movie controller, only the time-base flag is doing any
  work at run time. `SetMovieMasterTimeBase(theMovie, theTime, nil)` re-attaches the
  movie to its own (now-looping) time base.
- **Step 29 captures the natural box**, which in Mac `Rect` memory order
  `{top, left, bottom, right}` is `{0, 0, height, width}` for every
  movie in this tree (section 12.3). `movieRect` is then *overwritten* every time
  `DrawLocale` places the movie (section 7), so this initial value is only used if
  `DrawLocale` never runs.

### 4.1 `LoopMovie`

`GliderPRO/Sources/HouseIO.c:48-63`:

```
 1  theLoop = NewHandle(sizeof(long))        // 4 bytes
 2  (** (long **) theLoop) = 0               // contents: 0x00000000
 3  theUserData = GetMovieUserData(theMovie)
 4  theCount = CountUserDataType(theUserData, 'LOOP')
 5  while (theCount--) RemoveUserData(theUserData, 'LOOP', 1)
 6  AddUserData(theUserData, theLoop, 'LOOP')
```

So: remove every existing `'LOOP'` item, then add exactly one whose payload is the
4-byte value 0. (`'LOOP'` = 0x4C4F4F50.) By QuickTime convention a `'LOOP'` user-data
item with value 0 means "normal loop" and 1 means "palindrome loop". `theLoop` is
leaked — `AddUserData` copies the handle's contents but the source handle is never
disposed. A 4-byte leak per house open.

**Measured**: no shipped movie contains a `udta` atom at all (section 12.9), so step 5
never removes anything and the loop flag exists only in memory.

### 4.2 `CloseHouseMovie`

`GliderPRO/Sources/HouseIO.c:146-159`:

```
 1  if (thisMac.hasQT && hasMovie) {
 2      LoadMovieIntoRam(theMovie, GetMovieTime(theMovie, 0L),
 3                       GetMovieDuration(theMovie), flushFromRam)
 4      DisposeMovie(theMovie) }
 5  hasMovie = false                     // unconditional, outside the #ifdef
```

Called from `OpenHouseMovie`'s own error paths (`:108`, `:119`, `:128`) and from
`CloseHouse` (`HouseIO.c:550`). Note the error paths call it when `hasMovie` is still
`false`, so it only sets the flag — the movie handle from a *successful*
`NewMovieFromFile` followed by a *failed* `LoadMovieIntoRam` is **leaked**. A real bug,
but one that can only fire on a memory-starved 1996 Mac.

### 4.3 The user-visible failure message

`YellowAlert(kYellowQTMovieNotLoaded, identifier)` (`HouseIO.c:640-655`) shows `ALRT`
1006 with `GetIndString(errStr, 1006, whichAlert)`.
`kYellowQTMovieNotLoaded` = 22 (`GliderDefines.h:39`), and string 22 of `STR#` 1006 in
`GliderPRO/Glider PRO.r` reads:

> `The QuickTime™ movie that goes with this house will not be used.  Glider PRO™ must have enough memory to easily load the entire movie into RAM.`

(Two spaces after the first sentence, `™` is MacRoman 0xAA.) The numeric `identifier`
is substituted into the alert as `^1`/`^2` by `ParamText`.

### 4.4 The splash-screen tell

`GliderPRO/Sources/MainWindow.c:60-63` (inside `DrawOnSplash`, `:56`):

```
  PasStringCopy("\pHouse: ", houseLoadedStr);
  PasStringConcat(houseLoadedStr, thisHouseName);
  if ((thisMac.hasQT) && (hasMovie))
      PasStringConcat(houseLoadedStr, "\p (QT)");
```

drawn at `(splashOriginH + 436, splashOriginV + 314)` in 9-point application font,
bold (`TextFace(1)`). This is the only place the game tells the player a movie loaded.
For the 14 headerless movies, a faithful port must decide whether `" (QT)"` appears —
see section 19.

---

## 5. The Toolbox API used, and its Go replacement

Every Movie Toolbox call in the game, in one table. `GliderPRO/Headers/GliderVars.h:8`
does `#include <Movies.h>`; that header is **not** in this source tree, so the numeric
values of `newMovieActive`, `loopTimeBase` and `flushFromRam` cannot be derived from
the source in front of us (see Open questions).

| Call | Site(s) | What it does | Go equivalent |
|---|---|---|---|
| `EnterMovies()` | `Main.c:332` | initialises the Movie Toolbox for this process | nothing |
| `Gestalt(gestaltQuickTime, &r)` | `Environ.c:202` | QuickTime presence test | a constant `true` |
| `OpenMovieFile(&spec, &refNum, fsCurPerm)` | `HouseIO.c:87` | opens the movie **file** (data + resource fork) | `os.Open` |
| `NewMovieFromFile(&movie, refNum, nil, name, newMovieActive, &changed)` | `HouseIO.c:94-95` | parses the movie header (from the resource fork's `'moov'` resource if present, else the data fork's `moov` atom) | parse the container (sections 11-12) |
| `CloseMovieFile(refNum)` | `HouseIO.c:99`, `:102` | closes the file | `f.Close()` |
| `GoToBeginningOfMovie(movie)` | `HouseIO.c:112`, `Trip.c:47`, `Transit.c:306`, `:344`, `:383`, `:422` | seeks to time 0 | `frameIndex = 0` |
| `GetMovieTime(movie, 0L)` | `HouseIO.c:114`, `:154` | current time in the movie's time scale | `frameIndex * frameDur` |
| `GetMovieDuration(movie)` | `HouseIO.c:114`, `:154` | total duration in time-scale units | `nFrames * frameDur` |
| `LoadMovieIntoRam(movie, t, d, flags)` | `HouseIO.c:113`, `:153` | with flags 0: makes the range resident; with `flushFromRam`: releases it | decode all frames up front |
| `PrerollMovie(movie, 0, 0x000F0000)` | `HouseIO.c:124` | read-ahead hint at Fixed rate 15.0 | nothing |
| `GetMovieTimeBase(movie)` | `HouseIO.c:132` | the movie's `TimeBase` | nothing |
| `SetTimeBaseFlags(tb, loopTimeBase)` | `HouseIO.c:133` | makes the time base wrap at its duration | `frameIndex = (frameIndex+1) % nFrames` |
| `SetMovieMasterTimeBase(movie, tb, nil)` | `HouseIO.c:134` | re-parents the movie's clock | nothing |
| `GetMovieUserData(movie)` | `HouseIO.c:56` | the movie's user-data list | nothing |
| `CountUserDataType(ud, 'LOOP')` | `HouseIO.c:57` | how many `'LOOP'` items exist | nothing |
| `RemoveUserData(ud, 'LOOP', 1)` | `HouseIO.c:60` | deletes one | nothing |
| `AddUserData(ud, h, 'LOOP')` | `HouseIO.c:62` | appends one | nothing |
| `GetMovieBox(movie, &r)` | `HouseIO.c:137`, `ObjectDrawAll.c:686` | the movie's display box (from `tkhd.trackWidth/Height`) | `image.Rect(0,0,w,h)` |
| `SetMovieBox(movie, &r)` | `ObjectDrawAll.c:688` | moves/resizes the display box | store the dest rect |
| `RectRgn(rgn, &r)` / `NewRgn` / `DisposeRgn` | `ObjectDrawAll.c:689-692` | builds a rectangular region | `image.Rectangle` |
| `SetMovieDisplayClipRgn(movie, rgn)` | `ObjectDrawAll.c:691` | clips all movie drawing to that region | clip the blit |
| `SetMovieGWorld(movie, (CGrafPtr)mainWindow, nil)` | `Play.c:150` | binds output to the **on-screen window** | choose a destination image |
| `SetMovieActive(movie, true/false)` | `Play.c:204`, `:228` | enables/disables the movie's clock | a boolean |
| `StartMovie(movie)` | `HouseIO.c` n/a, `Play.c:207`, `Trip.c:48`, `Transit.c:307`, `:345`, `:384`, `:423` | sets rate to 1.0 | `playing = true` |
| `StopMovie(movie)` | `Play.c:227`, `Trip.c:53`, `RoomGraphics.c:411` | sets rate to 0 | `playing = false` |
| `MoviesTask(movie, 0)` | `Play.c:208`, `:467`, `:492` | *does the drawing*: services the movie, decoding and blitting any frame that is now due. `0` = "as much time as you need" | advance the frame index and blit |
| `DisposeMovie(movie)` | `HouseIO.c:155` | frees it | `nil` the slice |

Toolbox concepts a Go port has to replace outright:

- **`Movie` is an opaque handle** into a Toolbox-owned structure. In Go it becomes a
  plain struct holding the decoded frames, dimensions, palette, and a frame cursor.
- **`GWorld` / `CGrafPtr`**: an offscreen (or onscreen) QuickDraw pixel buffer plus its
  colour table and current clip. See `rendering.md` §3 and §23. `SetMovieGWorld`
  choosing `mainWindow` rather than `workSrcMap` is the single most consequential line
  in this subsystem (section 8).
- **`Region`**: an arbitrary-shape clip. Here it is always a rectangle, so
  `image.Rectangle` suffices.
- **`TimeBase`**: a clock object shared by several movies. Here there is exactly one
  movie, driven off the game frame counter in a port.
- **Big-endian on-disk data.** Every field in a `.mov` file is big-endian; Go must use
  `encoding/binary.BigEndian` throughout. See section 11.
- **`Fixed` (16.16 signed)**: `0x00010000` = 1.0, `0x000F0000` = 15.0,
  `0x00520000` = 82.0. Divide by 65536.0.
- **The Mac epoch**: QuickTime timestamps are seconds since **1904-01-01 00:00** local
  time. Unix epoch delta = **2082844800** seconds.

---

## 6. The TV object itself

### 6.1 Object type and sprite

`kTV` = **0x65 = 101** (`GliderPRO/Sources/House.h` object enum; `tools/probe_house.py:253`
carries the same mapping). It is an *appliance* — union arm `applianceType`
(`data.g`), so it has `topLeft` (a `Point`, stored `v` then `h`), `height`, `byte0`,
`delay`, `initial`, `state`.

| Rect | Value (l, t, r, b) | Size | Source |
|---|---|---|---|
| `srcRects[kTV]` | (0, 0, 92, 77) | 92 x 77 | `StructuresInit2.c:436` |
| `tvScreen1` | (0, 171, 64, 220) | 64 x 49 | `StructuresInit.c:570-571` |
| `tvScreen2` | (0, 220, 64, 269) | 64 x 49 | `StructuresInit.c:572-573` |

`tvScreen1` is the **off** screen; `tvScreen2` is the **on** screen (static/snow). Both
are windows into `applianceSrcMap`. The TV's own artwork is `PICT` 3992 with mask
`PICT` 3912 (`ObjectDraw2.c:98` and `:51`).

`GetObjectRect` for `kTV` (`ObjectRects.c:204` case label, shared body `:213-218`):

```
  *itsRect = srcRects[kTV];          // (0,0,92,77)
  ZeroRectCorner(itsRect);           // no-op here, already at (0,0)
  QOffsetRect(itsRect, who->data.g.topLeft.h, who->data.g.topLeft.v);
```

so the TV occupies `(h, v)` to `(h+92, v+77)` in room coordinates.

### 6.2 The screen offset: (+17, +10)

The 64 x 49 screen sits at a fixed offset inside the 92 x 77 TV, hard-coded in three
places that must agree:

| Site | Code |
|---|---|
| `ObjectDraw2.c:674-676` (`DrawTV`) | `bounds = tvScreen1; ZeroRectCorner(&bounds); QOffsetRect(&bounds, theRect->left + 17, theRect->top + 10);` |
| `ObjectDrawAll.c:683-685` (movie placement) | `whoCares = tvScreen1; ZeroRectCorner(&whoCares); OffsetRect(&whoCares, itsRect.left + 17, itsRect.top + 10);` |
| `Dynamics3.c:265-269` (`AddDynamicObject`) | `dinahs[n].dest = tvScreen1; ZeroRectCorner(...); QOffsetRect(..., where->left + playOriginH + 17, where->top + playOriginV + 10);` |

So the screen's margins inside the bezel are: left 17, top 10, right 92-17-64 = **11**,
bottom 77-10-49 = **18**. (Asymmetric — the TV art has a wider bottom bezel.)

`ZeroRectCorner` (`RectUtils.c:57-63`) is:

```
  theRect->right  -= theRect->left;
  theRect->bottom -= theRect->top;
  theRect->left = 0;  theRect->top = 0;
```

which is why `tvScreen1`, defined at y = 171 inside the appliance sheet, becomes
`(0, 0, 64, 49)` before being offset into place.

### 6.3 `DrawTV` — the static screen

`ObjectDraw2.c:640-689`. Two phases:

1. If `isLit` (the room's lights are on), load `PICT` 3992 and mask `PICT` 3912 into
   temporary GWorlds sized `srcRects[kTV]`, `CopyMask` them into `backSrcMap` at
   `theRect`, and dispose. (One `CreateOffScreenGWorld` + `LoadGraphic` + `DisposeGWorld`
   pair **per TV per room draw** — profligate but not this document's problem.)
2. Unconditionally blit the screen: `tvScreen2` if `isOn`, else `tvScreen1`, from
   `applianceSrcMap` to `backSrcMap` at `bounds`, `srcCopy`.

Note phase 2 happens whether or not this TV is the movie TV, and it draws into
`backSrcMap` (the room background). The movie is drawn later, to the window. So the
static screen is *always* underneath the movie in `backSrcMap`, which is exactly what
makes the "movie stops -> static reappears" behaviour work for free.

### 6.4 `HandleTV` — the per-frame animation, and the QT suppression

`GliderPRO/Sources/Dynamics.c:428-478`, dispatched from `Dynamics3.c:51-53`:

```
 1  if (dinahs[who].timer <= 0) return
 2  dinahs[who].timer--
 3  if (dinahs[who].active) {                       // turning ON / on
 4      if (dinahs[who].timer == 0) {
 5          if (hasQT && hasMovie && tvInRoom && who == tvWithMovieNumber) { }   // :437-444
 6          else AddRectToWorkRects(&dinahs[who].dest)
 7      } else if (dinahs[who].timer == 1) {
 8          PlayPrioritySound(kTVOnSound, kTVOnPriority)                          // :448
 9          if (hasQT && hasMovie && tvInRoom && who == tvWithMovieNumber) { }   // :449-460
10          else {
11              CopyBits(applianceSrcMap -> backSrcMap, tvScreen2,
12                       dinahs[who].dest, srcCopy, nil)
13              AddRectToBackRects(&dinahs[who].dest)
14          }
15      }
16  } else {                                        // turning OFF / off
17      if (dinahs[who].timer == 0) AddRectToWorkRects(&dinahs[who].dest)
18      else if (dinahs[who].timer == 1) {
19          PlayPrioritySound(kTVOffSound, kTVOffPriority)                        // :469
20          CopyBits(applianceSrcMap -> backSrcMap, tvScreen1,
21                   dinahs[who].dest, srcCopy, nil)                             // :470-474
22          AddRectToBackRects(&dinahs[who].dest)
23      }
24  }
```

The two **empty `if` bodies** at lines 5 and 9 are the whole trick: for the movie TV,
neither the `AddRectToWorkRects` nor the `CopyBits(tvScreen2)` happens, so the "snow"
image is never published and the movie is not overdrawn. Note the deliberate asymmetry:
the **off** branch (lines 16-23) has **no** QT guard, because when the TV is switched
off the last movie frame is still sitting in the window and must be replaced by
`tvScreen1`.

Sound constants (`GliderDefines.h:87-88`, `:145-146`):

| Constant | Value |
|---|---|
| `kTVOnSound` | 32 |
| `kTVOffSound` | 33 |
| `kTVOnPriority` | 404 |
| `kTVOffPriority` | 405 |

### 6.5 `AddDynamicObject` for `kTV`

`Dynamics3.c:264-282`:

| Field | Value |
|---|---|
| `dest` | `tvScreen1` zeroed, offset to `(where->left + playOriginH + 17, where->top + playOriginV + 10)` — i.e. the 64 x 49 screen in **window** coordinates |
| `whole` | = `dest` |
| `hVel`, `vVel`, `count`, `frame`, `timer`, `position` | 0 |
| `room` | the room index |
| `byte0` | `(Byte)index` — the object's index within the room |
| `byte1` | 0 |
| `moving` | `false` |
| `active` | `isOn` (i.e. `thisObject.data.g.state`) |

`dest` here is the *only* place the game keeps the movie TV's screen rect in window
coordinates for the dirty-rect machinery. The movie placement in `ObjectDrawAll.c`
computes the same rect independently.

### 6.6 Toggling

Reachable two ways, both funnelling into `ToggleTV`:

- Player touches the TV's hot spot. `ObjectRects.c:983` (case label; body `:990-996`) registers an
  `AddActiveRect` over `srcRects[kTV]` zeroed and offset; `Interactions.c:1101-1103`
  dispatches `case kTV: ToggleTV(masterObjects[linkIndex].dynaNum);`.
- A switch or band linked to the TV, same dispatch.

`GliderPRO/Sources/Trip.c:40-58`:

```
 1  dinahs[index].active = !dinahs[index].active
 2  if (thisMac.hasQT && hasMovie && tvInRoom && tvWithMovieNumber == index) {
 3      if (dinahs[index].active) {
 4          GoToBeginningOfMovie(theMovie); StartMovie(theMovie); tvOn = true }
 5      else { StopMovie(theMovie); tvOn = false }
 6  }
 7  dinahs[index].timer = 4
```

So: switching on **always rewinds**; the movie never resumes mid-stream.
`timer = 4` means `HandleTV` runs for four frames and does its work at `timer == 1`
(sound + optional blit) and `timer == 0` (dirty rect). Compare `ToggleMacPlus`
(`Trip.c:29-36`), which uses `timer = 40` when switching on and `10` when switching off.

### 6.7 Initial state

`Play.c:683` (case label) and `:688-689` (the assignment): at game start every appliance's `data.g.state` is reset to
`data.g.initial`, so a TV that was authored "on" starts on and the movie starts
playing immediately (`Play.c:205-209`).

Measured example — the only TV in Demo House
(`GliderPRO/Houses/Demo House.binhex`, room index **4**, name `"Where To Go?"`, object
index **10** within that room):

| Field | Value |
|---|---|
| `what` | 0x0065 = 101 = `kTV` |
| raw object bytes | `00 65 00 B0 00 75 00 00 00 00 01 01` |
| `topLeft` | v = 0x00B0 = 176, h = 0x0075 = 117 |
| `height` | 0 |
| `byte0`, `delay` | 0, 0 |
| `initial` | 1 (on) |
| `state` | 1 (on) |

---

## 7. Geometry: where the movie lands, exactly

`GliderPRO/Sources/ObjectDrawAll.c:674-713`. The `kTV` case, in order:

```
 1  GetObjectRect(&thisObject, &itsRect)                       // (h,v)..(h+92,v+77)   :675
 2  OffsetRectRoomRelative(&itsRect, neighbor)                 //                      :676
 3  if (!SectRect(&itsRect, &testRect, &whoCares)) break       // off-screen: skip     :677
 4  if (hasQT && hasMovie && neighbor == kCentralRoom && !tvInRoom) {   //         :680-681
 5      whoCares = tvScreen1                                   //                      :683
 6      ZeroRectCorner(&whoCares)                              // -> (0,0,64,49)       :684
 7      OffsetRect(&whoCares, itsRect.left + 17, itsRect.top + 10)  //                 :685
 8      GetMovieBox(theMovie, &movieRect)                      // -> (0,0,W,H)         :686
 9      CenterRectInRect(&movieRect, &whoCares)                //                      :687
10      SetMovieBox(theMovie, &movieRect)                      //                      :688
11      theRgn = NewRgn(); RectRgn(theRgn, &whoCares)           //                  :689-690
12      SetMovieDisplayClipRgn(theMovie, theRgn); DisposeRgn(theRgn)  //           :691-692
13      tvOn = thisObject.data.g.state                         //                      :693
14  }
15  DrawTV(&itsRect, thisObject.data.g.state, isLit)           //                      :696
16  if (!redraw) {                                             //                      :697
17      rectA = itsRect; QOffsetRect(&rectA, -playOriginH, -playOriginV)   //     :699-700
18      dynamicNum = AddDynamicObject(kTV, &rectA, &thisObject,
19                        localNumbers[neighbor], i, thisObject.data.g.state)  // :701-702
20      if (hasQT && hasMovie && neighbor == kCentralRoom && !tvInRoom) {  //     :704-705
21          tvWithMovieNumber = dynamicNum                     //                      :707
22          tvInRoom = true                                    //                      :708
23      }
24  }
```

Three things to notice.

**(a) It is `neighbor == kCentralRoom && !tvInRoom`, so only the FIRST TV in the
CENTRAL room gets the movie.** `kCentralRoom` = 0 (`GliderDefines.h:217`).
`DrawLocale` scans the central room and its eight neighbours; TVs in the neighbouring
rooms are drawn (they are partially visible at the screen edges) but never bound to the
movie. And within the central room, the *first* TV in object order wins, because
`tvInRoom` is set `true` at step 22 and the guard requires `!tvInRoom`.

The election is re-run from scratch on every room draw, because `DrawLocale` clears
both variables at its own entry (`RoomGraphics.c:59-60`, inside the function that
starts at `:44`) before it calls `DrawARoomsObjects` nine times. The call order fixes
the priority: north-west, north-east, north, south-west, south-east, south
(`RoomGraphics.c:82`, `:86`, `:90`, `:94`, `:98`, `:102`, only when
`numNeighbors > 3`), then west, east (`:109`, `:114`, only when `numNeighbors > 1`),
then **central last** (`:120`). Since the guard demands `neighbor == kCentralRoom`,
the eight earlier calls can never claim the movie, and the claim always happens in the
final call — so "first TV" means first in the central room's own object array, and
the eight neighbour passes are irrelevant to it.

Subtlety about `redraw`: step 4 (placement) is *outside* the `if (!redraw)` block and
step 20 (registration) is *inside* it, so on a `redraw == true` pass the movie box
would be re-placed and clipped but `tvWithMovieNumber` / `tvInRoom` would never be set
— leaving the movie playing into the window with nothing in `dinahs[]` bound to it and
`tvInRoom == false`, so a second TV in the same room could then also claim placement.
**In the shipped game this never happens.** `DrawARoomsObjects` has exactly ten call
sites and only one passes `true`:

| Call site | `redraw` |
|---|---|
| `RoomGraphics.c:82`, `:86`, `:90`, `:94`, `:98`, `:102`, `:109`, `:114`, `:120` (all inside `DrawLocale`) | `false` |
| `RoomGraphics.c:451` (inside `RedrawRoomLighting`) | `true` |

and `RedrawRoomLighting` (`RoomGraphics.c:434-461`) is **never called**: the only
occurrences of the identifier in the whole tree are its own comment banner
(`RoomGraphics.c:432`), its definition (`:434`) and its prototype
(`GliderProtos.h:410`). It is dead code. A Go port can therefore treat step 4 and
step 20 as one atomic block guarded once, and does not need a `redraw` parameter at
all unless it also revives `RedrawRoomLighting` — in which case the guard must be
hoisted so that placement and registration stay together.

**(b) Coordinate space.** `OffsetRectRoomRelative` (`ObjectRects.c:1093-1131`) adds
`(playOriginH, playOriginV)` and then, for `kCentralRoom`, nothing else. With
`playOriginH = (screenWidth - kRoomWide)/2` and
`playOriginV = (screenHeight - kTileHigh)/2` (`InterfaceInit.c:203-204`,
`kRoomWide` = 512), `itsRect` is in **window** coordinates. Because
`SetMovieGWorld` binds the movie to `mainWindow` and a `WindowPtr`'s port is
window-local, that is the right space. **`movieRect` therefore depends on the screen
resolution** — it is not a fixed room-space rectangle. A Go port with a fixed logical
framebuffer should compute it in framebuffer space and drop `playOrigin*`.

**(c) The movie is CENTRED, never SCALED.** `CenterRectInRect`
(`RectUtils.c:142-154`), body `:146-153`:

```
  widthA = RectWide(rectA);  tallA = RectTall(rectA);
  rectA->left = rectB->left + (RectWide(rectB) - widthA) / 2;
  rectA->right = rectA->left + widthA;
  rectA->top  = rectB->top  + (RectTall(rectB) - tallA) / 2;
  rectA->bottom = rectA->top + tallA;
```

`widthA`/`tallA` are preserved exactly, so `SetMovieBox` never resizes. A movie larger
than 64 x 49 is centred and then **clipped away** by the display clip region at step
12; a smaller one is centred with the TV's static screen visible in the border.

### 7.1 The arithmetic for all three shipped sizes

Let `S` = the 64 x 49 clip rect (`whoCares` after step 7), and let the movie be
`W x H`. Then

```
  dx = (64 - W) / 2        dy = (49 - H) / 2        // C integer division
  movie box = (S.left + dx, S.top + dy, +W, +H)
```

| Movies | W x H | dx | dy | Result |
|---|---|---|---|---|
| 13 of 15 (see section 17) | 64 x 49 | 0 | 0 | exact fit, nothing clipped |
| Teddy World | 64 x 50 | 0 | `-1/2` | see below |
| Demo House | 82 x 62 | -9 | `-13/2` | 18 columns and 13 rows clipped |

**Teddy World, 64 x 50.** `(49 - 50)/2 = -1/2`. In C89 the sign of the result of
integer division with a negative operand is *implementation-defined*; MPW C and
CodeWarrior both truncate toward zero, giving **0**. So the movie box top coincides
with the clip rect top and the movie's **last row (row 49) is clipped away**. Had the
compiler floored, `dy` would be -1 and row **0** would be lost instead. Go's `/`
truncates toward zero (spec: "the result of integer division is truncated toward
zero"), so a literal Go transcription reproduces the CodeWarrior behaviour.

**Demo House, 82 x 62.** `dx = (64-82)/2 = -18/2 = -9` (exact, no ambiguity).
`dy = (49-62)/2 = -13/2 = -6` with truncation (or -7 with flooring). With `dy = -6`:

| | movie box | clip rect | visible source range |
|---|---|---|---|
| horizontal | `S.left-9` .. `S.left+73` | `S.left` .. `S.left+64` | movie columns **9..72** |
| vertical | `S.top-6` .. `S.top+56` | `S.top` .. `S.top+49` | movie rows **6..54** |

So 9 columns are lost from the left, 9 from the right, 6 rows from the top and 7 from
the bottom. This is not a hypothesis: rendered, the movie is a television newsreader
with an "AIR CRASH" caption in the lower left, and the caption's first characters fall
in the clipped-away columns. Reproduced with `probe_mov.py decode` plus the crop above.

**Why the odd size?** Demo House's movie is 82 x 62 because it was authored before the
TV screen was fixed at 64 x 49, or simply never re-cropped. It is the one movie in the
set whose header survived, and the one whose header proves the game does not scale.

---

## 8. Where the movie is drawn, and what that costs

`GliderPRO/Sources/Play.c:147-152`, in `NewGame`'s setup (`NewGame` begins at
`Play.c:74`; `PlayGame` is a separate function starting at `Play.c:430`), immediately
after the menu-bar strip is painted black:

```
  SetPort((GrafPtr)mainWindow);
  ... PaintRect(&tempRect) ...
  #ifdef COMPILEQT
      if ((thisMac.hasQT) && (hasMovie))
          SetMovieGWorld(theMovie, (CGrafPtr)mainWindow, nil);
  #endif
  SetPort((GrafPtr)workSrcMap);
```

The movie draws **into the on-screen window**, not into `workSrcMap`. Consequences,
all of which a port must decide about consciously:

1. The movie's pixels never enter `workSrcMap`, so the dirty-rect machinery
   (`rendering.md` §9) does not know they exist. Any dirty rect overlapping the TV
   screen republishes `workSrcMap` over the movie for that frame.
2. `MoviesTask` is called *before* `RenderFrame()` in program order
   (`Play.c:465-468` and `:490-493`), so within a frame the movie is painted first and
   `CopyRectsQD` can paint over it. In practice the TV screen rect is only dirty when
   something moves across it — which is exactly when the artifact shows.
3. Conversely, across frames where nothing is dirty there, the movie sits on top of
   everything the renderer published, including the glider. **A glider in front of a
   playing TV disappears behind the movie.**
4. `HandleTV` suppresses the static-screen blits for the movie TV precisely so that the
   movie is not overdrawn (section 6.4).

### 8.1 Per-frame servicing

| Site | Guard | Position in the frame |
|---|---|---|
| `Play.c:465-468` | `hasQT && hasMovie && tvInRoom && tvOn` | two-player branch: after `HandleGlider` x2, immediately before `RenderFrame()` |
| `Play.c:490-493` | same | one-player branch: same position |

Frame order: `HandleDynamics` -> input -> `HandleInteraction` -> `HandleTriggers` ->
`HandleBands` -> `HandleGlider` -> **`MoviesTask(theMovie, 0)`** -> `RenderFrame()` ->
`HandleDynamicScoreboard()`.

`MoviesTask(m, 0)` with `maxMilliSecToUse == 0` means "take as long as you need". It
is the call that actually decodes and blits, driven by the movie's time base — so the
movie plays at its own frame rate, decoupled from the game's 2-tick frame.

### 8.2 Start / stop inventory

| Event | Site | Action |
|---|---|---|
| house opened | `HouseIO.c:196-199` | reset flags, then `OpenHouseMovie()` |
| house closed | `HouseIO.c:550` | `CloseHouseMovie()` |
| game starts | `Play.c:201-211` | `SetMovieActive(true)`; if `tvOn`: `StartMovie` + `MoviesTask(theMovie, 0)` |
| game ends | `Play.c:223-230` | `tvInRoom = false`; `StopMovie`; `SetMovieActive(false)` |
| room becomes current | `RoomGraphics.c:406-413` | `tvInRoom = false`; `tvWithMovieNumber = -1`; `StopMovie` — then `DrawLocale` re-arms |
| after a transition (4 sites) | `Transit.c:302-309`, `:340-347`, `:379-386`, `:418-425` | `RenderFrame()` **first**, then if `tvInRoom && tvOn`: `GoToBeginningOfMovie` + `StartMovie`. Both statements sit inside the same `#ifdef COMPILEQT`, so with `COMPILEQT` undefined these four sites lose their `RenderFrame()` call too |
| player flips the switch | `Trip.c:40-58` | on: rewind + start + `tvOn = true`; off: stop + `tvOn = false` |
| `DrawLocale` entered (every room draw) | `RoomGraphics.c:59-60` | `tvInRoom = false`; `tvWithMovieNumber = -1` |

Note that **every** start is preceded by a rewind. The movie never resumes; walking out
of the room and back restarts it from frame 0.

### 8.3 Dead code

`Render.c:55` (`extern Rect shadowSrc[], justRoomsRect, movieRect;`) and `Render.c:59`
(`extern Boolean evenFrame, shadowVisible, twoPlayerGame, tvOn;`) bring `movieRect` and
`tvOn` into scope, and neither is referenced anywhere in `Render.c`. Nothing in `Render.c`, `RenderFrame`, `CopyRectsQD` or
`AddRectToWorkRects` special-cases the movie. So there was, at some point, a plan to
composite the movie into `workSrcMap` (which would have fixed all four artifacts in
section 8) and it was abandoned. **A Go port is free to take that road**: draw the
movie into the work map before `RenderFrame` publishes it, and the glider will
correctly occlude the TV. That is a deliberate, visible divergence from the original;
see section 19.4.

---

## 9. The corpus

### 9.1 What is on disk

`GliderPRO/Houses/` contains 22 house files (as `.binhex`) and **15** `.mov` files.
There are no AppleDouble (`._*`) sidecars and no `.rsrc` files: whatever was in the
movies' resource forks is simply gone from this tree (section 11.5).

```
$ ls -l GliderPRO/Houses/*.mov          # measured
    28232  Art Museum.mov
    33042  CD Demo House.mov
    21960  Castle o' the Air.mov
    34504  Davis Station.mov
   119789  Demo House.mov
    26407  Grand Prix.mov
    30707  ImagineHouse PRO II.mov
    16016  Land of Illusion.mov
    65920  Leviathan.mov
    28541  Nemo's Market.mov
     7021  Rainbow's End.mov
    37640  Slumberland.mov
    51291  SpacePods.mov
    28587  Teddy World.mov
     6534  Titanic.mov
   ------
   536191  total
```

### 9.2 Which houses have a movie

The name match is exact: `OpenHouseMovie` appends `".mov"` to the house's `FSSpec.name`
(`HouseIO.c:81`), so `Houses/Art Museum` pairs with `Houses/Art Museum.mov`.

| House | `.mov` | Rooms | `kTV` objects |
|---|---|---|---|
| Art Museum | yes | 109 | 2 |
| CD Demo House | yes | 206 | 1 |
| Castle o' the Air | yes | 85 | 2 |
| Davis Station | yes | 65 | 1 |
| Demo House | yes | 45 | 1 |
| Grand Prix | yes | 175 | 2 |
| ImagineHouse PRO II | yes | 279 | 7 |
| Land of Illusion | yes | 303 | 10 |
| Leviathan | yes | 472 | 4 |
| Nemo's Market | yes | 124 | 1 |
| Rainbow's End | yes | 223 | 5 |
| Slumberland | yes | 383 | 8 |
| SpacePods | yes | 402 | 8 |
| Teddy World | yes | 531 | 7 |
| Titanic | yes | 208 | 13 |
| California or Bust! | **no** | 16 | 1 |
| Empty House | **no** | 35 | 0 |
| Fun House | **no** | 43 | 0 |
| In The Mirror | **no** | 97 | 3 |
| Metropolis | **no** | 127 | 2 |
| Sampler | **no** | 2 | 0 |
| The Asylum Pro | **no** | 140 | 2 |

Totals: **15 houses with a movie, 7 without; 80 `kTV` placements across 22 houses.**
Of the 80, at most 15 can ever show a movie (one per house), and only when that TV is
the first one in the central room. Titanic has 13 TVs and exactly one of them — the
first one the player happens to stand in front of — becomes the movie TV, per visit.

`Fun House`, `Empty House` and `Sampler` have no TVs at all, so `hasMovie` would be
irrelevant even if they had a movie.

Method: `tools/probe_house.py` -> `binhex_decode`, `parse_house`, then count objects
with `what == 101`.

### 9.3 File naming edge cases

`Nemo's Market.mov` contains an apostrophe; `Castle o' the Air.mov` two. Both are plain
ASCII 0x27, so `PasStringConcat` handles them without incident. No house name in this
tree contains a MacRoman high byte, so no encoding question arises for the filename
path in a Go port — but a port that supports user houses must decode MacRoman
(`rendering.md` §22 covers the same issue for house names).

---

## 10. Big picture of the finding

```
$ python3 tools/probe_mov.py survey
movie                     bytes  mdat_decl   moov    codec  frames depth    WxH   fps kinds
Art Museum                28232          0     NO     raw        9 8      64x49     - raw=9
CD Demo House             33042          0     NO     rle       32 8      64x49     - delta=31 key=1
Castle o' the Air         21960          0     NO     raw        7 8      64x49     - raw=7
Davis Station             34504          0     NO     raw       11 8      64x49     - raw=11
Demo House               119789     118792 yes @118792     rle       41 36     82x62 4.9587 key=41
Grand Prix                26407          0     NO     smc       21 8      64x49     - smc=21
ImagineHouse PRO II       30707          0     NO     rle       10 8      64x49     - key=10
Land of Illusion          16016          0     NO     rle       28 8      64x49     - delta=27 key=1
Leviathan                 65920          0     NO     rle       21 8      64x49     - key=21
Nemo's Market             28541          0     NO     rle       24 8      64x49     - delta=23 key=1
Rainbow's End              7021          0     NO     rle       25 8      64x49     - delta=21 key=1 no-op=3
Slumberland               37640          0     NO     raw       12 8      64x49     - raw=12
SpacePods                 51291          0     NO     rle       40 8      64x49     - delta=39 key=1
Teddy World               28587          0     NO     rle       45 8      64x50     - delta=38 key=5 no-op=2
Titanic                    6534          0     NO     rle       11 8      64x49     - key=11
-- 15 movies, 536191 bytes total
```

Read the `mdat_decl` and `moov` columns:

- **14 of the 15 files have an `mdat` atom whose declared size field is literally
  `00 00 00 00`, and contain no `moov` atom anywhere.** A size field of 0 in QuickTime
  means "this atom extends to end of file", which is legal — but it is what QuickTime
  *writes* when the `moov` is not in the data fork, because there is then nothing after
  the `mdat` and the writer never goes back to patch the length.
- **Only `Demo House.mov` is self-contained**: 119,789 bytes total, `mdat` declared
  118,792, `moov` at offset 118,792 (size 997), `stsd` codec fourcc `rle ` = Apple
  Animation.

The `codec`, `frames`, `depth`, `WxH` and `kinds` columns for the 14 broken files are
**reconstructed** by `probe_mov.py`, not read from a header — see section 13.

### 10.1 Consequences, stated plainly

`NewMovieFromFile` (`HouseIO.c:94`) needs a movie header. For these 14 files there
isn't one in the data fork, and the tree has no resource forks for them. So:

1. On the original Mac in 1996, all 15 worked, because the `moov` lived in each file's
   **resource fork** as a `'moov'` resource, and `OpenMovieFile` /
   `NewMovieFromFile` look there first.
2. From **this source release**, 14 of 15 are unplayable by any conforming QuickTime
   implementation, and would be unplayable by a faithful Go port that only parses the
   container.
3. `NewMovieFromFile` would fail, `OpenHouseMovie` would raise
   `YellowAlert(kYellowQTMovieNotLoaded, ...)` (`HouseIO.c:98`), `hasMovie` would stay
   `false`, and all 14 houses would show static-snow TVs — plus a modal alert on every
   house open, which is obnoxious.
4. However, **all 14 are fully recoverable**, because their codecs are self-delimiting
   (each sample begins with its own byte count) and every one of them uses one sample
   per chunk with samples laid end to end from offset 8. Section 13 does the recovery
   and it closes to the byte on all 14.

---

## 11. The QuickTime container, as actually used here

### 11.1 The atom

Every atom is:

| Offset | Size | Type | Field |
|---|---|---|---|
| +0 | 4 | uint32 BE | `size` — total atom size **including this header** |
| +4 | 4 | char[4] | `type` — a fourcc, e.g. `moov`, `mdat` |
| +8 | ... | | payload, or child atoms if this is a container |

Special `size` values:

| Value | Meaning |
|---|---|
| `0` | the atom extends to end of file. **Legal only for the last atom.** This is the case for the 14 broken `mdat`s |
| `1` | a 64-bit size follows at +8 (`uint64`); payload starts at +16. Not used by any file here |
| 2..7 | invalid |

`probe_mov.py` treats a `size` of 0 as `filelen - offset` and reports the raw field
separately, which is how the `mdat_decl` column above can show `0`.

Container atoms in this tree, i.e. atoms whose payload is more atoms:
`moov`, `trak`, `mdia`, `minf`, `stbl`, `dinf`, `edts`, `udta`. Everything else is a
leaf with a fixed field layout.

### 11.2 Version / flags

Most leaf atoms begin with a 4-byte word that is `uint8 version` followed by
`uint24 flags`. In this tree **every** version byte is 0, so the "version 1" 64-bit
time variants of `mvhd`/`tkhd`/`mdhd` never occur — a Go port can implement version 0
only, and hard-error on version 1.

### 11.3 Numeric types

| Type | Encoding | Example from this tree |
|---|---|---|
| `uint16` / `int16` / `uint32` | big-endian | `timeScale` = 600 |
| `Fixed` | signed 16.16 | `0x00010000` = 1.0, `0x00520000` = 82.0 |
| `unsigned 8.8` | e.g. `preferredVolume` | `0x00FF` = 0.996 (full) |
| Mac date | uint32 seconds since 1904-01-01 | `2888911668` = 1995-07-18 11:27:48 |
| matrix | 9 x uint32; `[0][0]`,`[0][1]`,`[1][0]`,`[1][1]`,`[2][0]`,`[2][1]` are `Fixed`, `[0][2]`,`[1][2]`,`[2][2]` are `Fract` (2.30) | identity = `{1,0,0}{0,1,0}{0,0,0x40000000}` |
| Pascal string (`Str31`) | 1 length byte + bytes, padded to 32 | `hdlr` component names |

Unix conversion: `unix = mac - 2082844800`.

### 11.4 The atom tree, per file

For the 14 broken files the entire tree is:

```
0        mdat  size field 0 (= filelen - 0)
```

and nothing else. Byte 0..7 of, e.g., `Titanic.mov`:

```
00000000  00 00 00 00 6D 64 61 74  40 00 02 42 00 08 00 00
          ^^^^^^^^^^^ size = 0     ^^^^^^^^^^^ first sample begins here
                      ^^^^^^^^^^^ 'mdat'
```

Compare `Demo House.mov`:

```
00000000  00 01 D0 08 6D 64 61 74  40 00 0B 55 00 08 00 00
          ^^^^^^^^^^^ 0x0001D008 = 118792
```

and at offset 0x1D008 = 118792:

```
0001D000  76 77 77 00 00 00 FF 00  00 00 03 E5 6D 6F 6F 76
                                   ^^^^^^^^^^^ 997  ^^^^^^ 'moov'
0001D010  00 00 00 6C 6D 76 68 64  00 00 00 00 AC 31 4B 34
                                                ^^^^^^^^^^^ 2888911668
```

Note `mdat` comes **first** in `Demo House.mov` — the writer streamed the samples and
appended the header. Every sample offset in `stco` is therefore small (first = 8) and a
port must not assume `moov` precedes `mdat`.

### 11.5 Why the headers are missing — the git evidence

`GliderPRO/upstream.git` is a bare clone of John Calhoun's own repository, and it
answers this conclusively.

```
$ git --git-dir=GliderPRO/upstream.git log --all --oneline -- 'Houses/*.mov'
7a70d18 First check-in.
```

Exactly one commit ever touched the `.mov` files: the initial import,
**`7a70d18` "First check-in." by John Calhoun, 2016-01-28**. They entered the
repository at their present sizes and were never modified again. So they were broken
from the moment the source was published, not damaged later.

The mechanism is stated by the author himself, in four commits over the following two
days. Full messages, verbatim:

**`147a1fb` "Ran macbinary on .rsrc file." (2016-01-28 22:24)**
> GitHub was showing zero bytes for the rsrc. (resource) file so I macbinary encoded the
> file and appended .bin to it. You will have to run macbinary to decode the resource
> file to recover it.

**`2d29a41` "Trying binhex." (2016-01-29 07:01)**
> When I used macbinary to convert the house file and commit to GitHub, **the resource
> fork was lost.** Trying to binhex encode the file instead to see if it all gets
> committed.

**`b54548f` "Binhex'ed the house files." (2016-01-29 07:15)**
> Used 'binhex encode' from the command line to convert the Glider house files into a
> form that GitHub will consume. You will need to binhex decode to use.

**`797cbfb` "Replaced the binhex'ed files with derez'ed files." (2016-01-29 18:24)**
> Replaced the binhex'ed resource files (.rsrc) with derez'ed (.r) files. To return them
> to .rsrc files you will need to rez them (a command line tool part of X-Code).

So the author hit the dual-fork problem, diagnosed it correctly, and fixed it for the two
kinds of file he was thinking about: the **house** files (via BinHex — which is why
`Houses/*.binhex` round-trips both forks) and the **application's** resource fork (via
DeRez — which is why `GliderPRO/Glider PRO.r` exists and this whole documentation effort
has a palette to read). He **never did either for the `.mov` files**, because a `.mov`
looks like an ordinary data file. They were committed once, data fork only, at
`7a70d18`, and never touched again.

Corroborating negative results, all measured:

- No `moov` resource exists in any of the 22 house resource forks. Their resource types
  are exactly: `PICT` (919 present in some), `ICN#` 25, `icl8` 25, `icl4` 23, `ics#` 22,
  `ics8` 22, `ics4` 22, `snd ` 63, `bnds` 70, `vers` 12. Nothing movie-related.
- `GliderPRO/Houses` has no AppleDouble sidecars, no `.rsrc`, and no `__MACOSX`.
- Scanning all 15 `.mov` files for the 4-byte sequence `moov` anywhere (not just at
  atom boundaries) finds exactly one hit, in `Demo House.mov` at offset 118796.

So the movie headers are not hiding anywhere in this tree. Reconstructing them from the
sample data is the only route, and section 13 does that.

---

## 12. `Demo House.mov` — the complete header, field by field

The only file with a `moov`. Everything below is the measured output of
`python3 tools/probe_mov.py moov "Demo House"`.

### 12.1 Atom tree with offsets and sizes

| Offset | Size | Type | Depth |
|---|---|---|---|
| 0 | 118792 | `mdat` | 0 |
| 118792 | 997 | `moov` | 0 |
| 118800 | 108 | `mvhd` | 1 |
| 118908 | 881 | `trak` | 1 |
| 118916 | 92 | `tkhd` | 2 |
| 119008 | 36 | `edts` | 2 |
| 119016 | 28 | `elst` | 3 |
| 119044 | 745 | `mdia` | 2 |
| 119052 | 32 | `mdhd` | 3 |
| 119084 | 58 | `hdlr` (media) | 3 |
| 119142 | 647 | `minf` | 3 |
| 119150 | 20 | `vmhd` | 4 |
| 119170 | 57 | `hdlr` (data) | 4 |
| 119227 | 36 | `dinf` | 4 |
| 119235 | 28 | `dref` | 5 |
| 119263 | 526 | `stbl` | 4 |
| 119271 | 102 | `stsd` | 5 |
| 119373 | 24 | `stts` | 5 |
| 119397 | 28 | `stsc` | 5 |
| 119425 | 184 | `stsz` | 5 |
| 119609 | 180 | `stco` | 5 |

21 atoms. **One track.** No `stss`, no `udta`, no `ctts`, no sound track.
118792 + 997 = 119789 = the file size exactly.

### 12.2 `mvhd` — movie header, 108 bytes (8 header + 100 payload)

| +off (payload) | Size | Field | Measured value |
|---|---|---|---|
| 0 | 1 | version | 0 |
| 1 | 3 | flags | 0 |
| 4 | 4 | creationTime | 2888911668 = **1995-07-18 11:27:48** |
| 8 | 4 | modificationTime | 2888911670 = 1995-07-18 11:27:50 |
| 12 | 4 | **timeScale** | **600** (units per second) |
| 16 | 4 | **duration** | **4961** units = **8.2683 s** |
| 20 | 4 | preferredRate (Fixed) | `0x00010000` = 1.0 |
| 24 | 2 | preferredVolume (8.8) | `0x00FF` = 0.996 |
| 26 | 10 | reserved | zero |
| 36 | 36 | matrix | identity: `0x00010000,0,0 / 0,0x00010000,0 / 0,0,0x40000000` |
| 72 | 4 | previewTime | 0 |
| 76 | 4 | previewDuration | 0 |
| 80 | 4 | posterTime | 0 |
| 84 | 4 | selectionTime | 0 |
| 88 | 4 | selectionDuration | 0 |
| 92 | 4 | currentTime | 0 |
| 96 | 4 | **nextTrackID** | **2** — i.e. exactly one track ever existed |

`nextTrackID` = 2 is the proof that there was never a second (audio) track: QuickTime
increments it per track added.

### 12.3 `tkhd` — track header, 92 bytes (8 + 84)

| +off | Size | Field | Measured value |
|---|---|---|---|
| 0 | 1 | version | 0 |
| 1 | 3 | **flags** | **0x00000F** |
| 4 | 4 | creationTime | 2888911628 = **1995-07-18 11:27:08** |
| 8 | 4 | modificationTime | 2888911670 = 1995-07-18 11:27:50 |
| 12 | 4 | **trackID** | **1** |
| 16 | 4 | reserved | 0 |
| 20 | 4 | **duration** | **4961** (movie time scale) |
| 24 | 8 | reserved | zero |
| 32 | 2 | layer | 0 |
| 34 | 2 | alternateGroup | 0 |
| 36 | 2 | **volume** | **0** (a video track) |
| 38 | 2 | reserved | 0 |
| 40 | 36 | matrix | identity |
| 76 | 4 | **trackWidth** (Fixed) | `0x00520000` = **82.0** |
| 80 | 4 | **trackHeight** (Fixed) | `0x003E0000` = **62.0** |

`tkhd.flags` bits:

| Bit | Value | Name | Set? |
|---|---|---|---|
| 0 | 0x0001 | `trackEnabled` | yes |
| 1 | 0x0002 | `trackInMovie` | yes |
| 2 | 0x0004 | `trackInPreview` | yes |
| 3 | 0x0008 | `trackInPoster` | yes |

`GetMovieBox` returns the movie's spatial bounds transformed by the movie matrix —
identity here. Mind the field order: a Mac `Rect` is stored `{top, left, bottom, right}`,
so in *memory* the returned box is `{0, 0, 62, 82}` = `{top 0, left 0, bottom 62,
right 82}`. Written in the `(left, top, right, bottom)` order this document uses for rect
tables (section 6.1) that is **(0, 0, 82, 62)**: 82 wide, 62 tall. This is the number
that drives all the clipping arithmetic in section 7.1.

`trackWidth`/`trackHeight` are `Fixed`, and they are the *display* size. The `stsd`
`ImageDescription` separately carries the *encoded* size (also 82 x 62 here) — they can
differ, and when they do QuickTime scales. They do not differ in any file here.

### 12.4 `edts` / `elst` — edit list, 36 / 28 bytes

| Field | Value |
|---|---|
| version / flags | 0 / 0 |
| entryCount | 1 |
| entry[0].trackDuration | 4961 |
| entry[0].mediaTime | 0 |
| entry[0].mediaRate (Fixed) | `0x00010000` = 1.0 |

A single identity edit: media time 0 for the whole duration at rate 1. A port can
ignore edit lists for this tree; it must not ignore them for arbitrary movies.

### 12.5 `mdhd` — media header, 32 bytes (8 + 24)

| +off | Size | Field | Value |
|---|---|---|---|
| 0 | 4 | version/flags | 0 |
| 4 | 4 | creationTime | 2888911669 = 1995-07-18 11:27:49 |
| 8 | 4 | modificationTime | 2888911670 = 1995-07-18 11:27:50 |
| 12 | 4 | **timeScale** | **600** |
| 16 | 4 | **duration** | **4961** |
| 20 | 2 | language | 0 (English/system) |
| 22 | 2 | quality | 0 |

The five timestamps in the file order themselves sensibly: `tkhd.creationTime`
2888911628 (11:27:08), `mvhd.creationTime` 2888911668 (11:27:48),
`mdhd.creationTime` 2888911669, then `mvhd`/`tkhd`/`mdhd.modificationTime` all
2888911670 (11:27:50). A 42-second compression run on **1995-07-18**, a year before
Glider PRO 1.0.4 shipped.

Media time scale equals movie time scale (600), so `stts` durations are directly
comparable to `mvhd.duration`.

### 12.6 `hdlr` x 2 — handler references

| Atom | componentType | componentSubType | manufacturer | name |
|---|---|---|---|---|
| `mdia`/`hdlr` @119084 | `mhlr` | `vide` | `appl` | `"Apple Video Media Handler"` |
| `minf`/`hdlr` @119170 | `dhlr` | `alis` | `appl` | `"Apple Alias Data Handler"` |

Layout: version/flags (4), componentType (4), componentSubType (4),
componentManufacturer (4), componentFlags (4), componentFlagsMask (4), then a Pascal
string name (1 length byte + chars) filling the rest of the atom.
`componentSubType == 'vide'` in the media handler is how a port identifies the video
track. There is no `soun` handler anywhere in any file.

### 12.7 `vmhd` — video media information header, 20 bytes (8 + 12)

| +off | Size | Field | Value |
|---|---|---|---|
| 0 | 4 | version/flags | **0x00000001** = `noLeanAhead` |
| 4 | 2 | **graphicsMode** | **64 = `ditherCopy`** |
| 6 | 2 | opColor.red | `0x8000` |
| 8 | 2 | opColor.green | `0x8000` |
| 10 | 2 | opColor.blue | `0x8000` |

`graphicsMode` is a QuickDraw transfer mode:

| Mode | Value | Name |
|---|---|---|
| | 0 | `srcCopy` |
| | 1 | `srcOr` |
| | 2 | `srcXor` |
| | 3 | `srcBic` |
| | 4 | `notSrcCopy` |
| | 32 | `blend` |
| | 36 | `transparent` |
| **this file** | **64** | **`ditherCopy`** |

`ditherCopy` matters: the movie is **4-bit grayscale** (section 12.8) and the game runs
in **8-bit indexed colour** with `clut` 128, so QuickTime dithers the 16 gray levels
into whatever grays the game's palette offers. `clut` 128's gray ramp is coarse, so on
the original the newsreader is visibly dithered. A Go port drawing into an indexed
framebuffer must decide: nearest-colour map (clean but posterised) or an error-diffusion
dither (faithful). See section 19.

`opColor` (0x8000, 0x8000, 0x8000) is only consulted by `blend`/`transparent` modes; it
is inert here.

### 12.8 `stsd` — sample description, 102 bytes

Header: version/flags (4) = 0, entryCount (4) = **1**. Then one 86-byte
`ImageDescription`:

| +off | Size | Type | Field | Measured value |
|---|---|---|---|---|
| 0 | 4 | int32 | idSize | **86** |
| 4 | 4 | fourcc | **cType** | **`rle `** (0x726C6520) = Apple Animation |
| 8 | 4 | | resvd1 | 0 |
| 12 | 2 | | resvd2 | 0 |
| 14 | 2 | int16 | **dataRefIndex** | **1** |
| 16 | 2 | int16 | version | 1 |
| 18 | 2 | int16 | revisionLevel | 1 |
| 20 | 4 | fourcc | vendor | `appl` |
| 24 | 4 | int32 | temporalQuality | 0 (`codecMinQuality`) |
| 28 | 4 | int32 | **spatialQuality** | **1024 = 0x400 = `codecLosslessQuality`** |
| 32 | 2 | int16 | **width** | **82** |
| 34 | 2 | int16 | **height** | **62** |
| 36 | 4 | Fixed | hRes | `0x00480000` = 72.0 dpi |
| 40 | 4 | Fixed | vRes | `0x00480000` = 72.0 dpi |
| 44 | 4 | int32 | dataSize | 0 |
| 48 | 2 | int16 | frameCount | 1 (frames *per sample*) |
| 50 | 32 | Str31 | **name** | `"Animation"` |
| 82 | 2 | int16 | **depth** | **36** |
| 84 | 2 | int16 | **clutID** | **36** |

Two fields carry all the weight:

**`cType` = `rle `.** Apple Animation. Section 14 documents the bitstream and
`probe_mov.py` implements it from scratch.

**`depth` = 36 and `clutID` = 36.** QuickTime's depth encoding:

| depth | Meaning |
|---|---|
| 1, 2, 4, 8, 16, 24, 32 | colour, that many bits per pixel |
| 33 | 1-bit grayscale |
| 34 | 2-bit grayscale |
| **36** | **4-bit grayscale** |
| 40 | 8-bit grayscale |

So each pixel is 4 bits = a gray index 0..15. `clutID` = 36 means "use `GetCTable(36)`",
the Mac's standard 4-bit grayscale ramp, in which entry *i* maps to
`(255 - i*17, 255 - i*17, 255 - i*17)`, i.e. index 0 = white (255) and index 15 = black
(0). Verified against how the frames render: index 0 dominates the newsreader's
background, which is a light studio backdrop, and index 15 appears in the shadows.

`clutID` values a port must handle: **0** = "a `ColorTable` follows this
ImageDescription in the `stsd` payload" (does not occur here — the `stsd` is exactly
8 + 4 + 4 + 86 = 102 bytes with nothing after the ImageDescription);
**-1** = "no table, use the destination's"; **>= 8** = a `'clut'` resource ID.

### 12.9 `dinf` / `dref` — data reference, 36 / 28 bytes

| Field | Value |
|---|---|
| `dref` version/flags | 0 |
| entryCount | 1 |
| entry[0] size | 12 |
| entry[0] type | `alis` |
| entry[0] version/flags | `0x000001` = `dataRefSelfReference` |
| entry[0] payload | **0 bytes** |

The self-reference flag with an empty payload means "the media is in this same file".
Every sample offset in `stco` is therefore a file offset into `Demo House.mov` itself.
A Go port needs to check this flag and refuse anything else; a movie referencing an
external file would be unloadable.

### 12.10 `stts` — time-to-sample, 24 bytes

| Field | Value |
|---|---|
| version/flags | 0 |
| entryCount | **1** |
| entry[0].sampleCount | **41** |
| entry[0].sampleDuration | **121** |

41 x 121 = **4961** = `mdhd.duration` exactly. So:

- **41 frames**
- constant frame duration 121 units at time scale 600
- **frame rate = 600 / 121 = 4.9587 fps**
- **duration = 4961 / 600 = 8.2683 s**

4.9587 fps is a strange number; it is what you get when you author "5 fps" in a 600
time scale (600/5 = 120) and something rounds. A port can either honour 121/600 exactly
or, more simply, advance one movie frame every N game frames — see section 19.5.

### 12.11 `stsc` — sample-to-chunk, 28 bytes

| Field | Value |
|---|---|
| version/flags | 0 |
| entryCount | 1 |
| entry[0].firstChunk | 1 |
| entry[0].samplesPerChunk | **1** |
| entry[0].sampleDescriptionID | 1 |

**One sample per chunk.** This is the structural fact that makes recovery of the 14
headerless movies possible: chunk offsets = sample offsets, and samples are laid out
consecutively.

### 12.12 `stsz` — sample size, 184 bytes

| Field | Value |
|---|---|
| version/flags | 0 |
| **sampleSize** | **0** (i.e. variable; a table follows) |
| **sampleCount** | **41** |
| then | 41 x uint32 sizes |

Measured: sum = **118784**, min **2877**, max **2917**, mean 2897.2, first **2901**.
118784 = 118792 - 8 = the `mdat` payload **exactly**. So the samples completely tile
the `mdat` with no padding and no gaps.

All 41 sizes, in table order:

```
2901 2911 2886 2910 2883 2895 2911 2897 2905 2891
2893 2891 2898 2905 2881 2901 2907 2889 2897 2896
2893 2909 2901 2907 2901 2900 2884 2913 2888 2907
2877 2907 2895 2891 2899 2917 2883 2890 2896 2897
2881
```

Near-constant (spread 2877..2917, only 40 bytes wide) because every sample is an
independent key frame at `codecLosslessQuality` over a dithered image that RLE cannot
compress. Compare the raw pixel cost, 82 x 62 x 4 bits = 2542 bytes: the "compressed"
samples are ~14% **larger** than uncompressed would be. Section 14.6 revisits this.

### 12.13 `stco` — chunk offset, 180 bytes

| Field | Value |
|---|---|
| version/flags | 0 |
| entryCount | **41** |
| entry[0] | **8** |
| entry[40] | **115911** |

`stco[0]` = 8, immediately after the `mdat` header. The chunk offsets and the sample
sizes agree perfectly, which `probe_mov.py` asserts:

```
  stco[i] + stsz[i] == stco[i+1]   for all i in 0..39      -> true
  stco[40] + stsz[40] == 115911 + 2881 == 118792           -> == mdat size
```

So the 41 samples are laid **contiguously** from offset 8 to offset 118792 with no
padding and no gaps — the property that makes the headerless files recoverable
(section 13). Run `probe_mov.py samples "Demo House"` to see the verified pairing
sample by sample.

Because `samplesPerChunk == 1`, `stco` is a *sample* offset table in all but name. A
Go port implementing the general case must still combine `stsc` + `stco` + `stsz`
properly: sample *k* of chunk *c* is at `stco[c] + sum(stsz[first_of_c .. k-1])`.

### 12.14 `stss` — absent

There is **no** sync-sample atom. In QuickTime, an absent `stss` means **every sample
is a key frame**. Consistent with the measured chunk flag bytes: all 41 samples have
`0x40` in the high byte of their `rle ` chunk header (section 14.2), which is the
Animation codec's own "this is a key frame" bit.

### 12.15 `udta` — absent

No user data atom. So the `'LOOP'` item that `LoopMovie` (`HouseIO.c:48-63`) so
carefully installs **exists only in memory**; it was never written to any shipped file.
Confirms that only `SetTimeBaseFlags(theTime, loopTimeBase)` is making the movie loop
at run time.

---

## 13. Recovering the 14 headerless movies

### 13.1 The idea

All three codecs present in this tree are *self-delimiting*: the first 4 bytes of every
`rle ` and `smc ` sample are `flags<<24 | byteCount`, where `byteCount` counts the
sample's own bytes including those 4. And `raw ` samples at depth 8 are exactly
`width * height` bytes. Combined with `samplesPerChunk == 1` and consecutive layout
(both verified on `Demo House.mov`), the sample table can be walked out of the `mdat`
payload with no header at all:

```
 1  payload = file[8 : filelen]                      // skip the mdat header
 2  p = 0; sizes = []
 3  while p < len(payload):
 4      n = BE_uint32(payload, p) & 0x00FFFFFF        // low 24 bits
 5      if n < 7 or p + n > len(payload): FAIL
 6      sizes.append(n); p += n
 7  succeed iff p == len(payload)                     // must close EXACTLY
```

The lower bound in step 5 must be **7**, not 8: the shortest legal `rle ` sample is
4 (chunk-size long) + 2 (header word) + 1 (end-of-stream `0x00`) = 7 bytes, and such
no-op samples really occur — three in Rainbow's End and two in Teddy World (section
14.3). With `n < 8` the walk rejects them and neither of those two files' chains close.
`probe_mov.py` uses the even looser `n < 4` (`probe_mov.py:921`), which is also safe
because step 7 does the real work.

Step 5's sanity bound and step 7's exact-fit test are what make this a *proof* rather
than a guess: a wrong codec assumption derails within a few samples and step 7 fails. And
because step 4 masks to 24 bits, the flag byte is discarded, which is exactly what the
Animation and Graphics codecs' specifications say to do.

For `raw `, there is no length prefix, so `probe_mov.py` tries the frame size 3136 =
64 x 49 first (justified by `tvScreen1` being 64 x 49, `StructuresInit.c:570-571`) and
falls back to the divisor of the payload length that gives a plausible height near 49.

### 13.2 Result: 15 / 15 exact

```
$ python3 tools/probe_mov.py samples all | grep -c EXACT
15
```

| Movie | Payload bytes | Samples | Sum of sizes | min / max / mean | first |
|---|---|---|---|---|---|
| Art Museum | 28224 | 9 | 28224 | 3136 / 3136 / 3136.0 | 3136 |
| CD Demo House | 33034 | 32 | 33034 | 833 / 1829 / 1032.3 | 1829 |
| Castle o' the Air | 21952 | 7 | 21952 | 3136 / 3136 / 3136.0 | 3136 |
| Davis Station | 34496 | 11 | 34496 | 3136 / 3136 / 3136.0 | 3136 |
| Demo House | 118784 | 41 | 118784 | 2877 / 2917 / 2897.2 | 2901 |
| Grand Prix | 26399 | 21 | 26399 | 1063 / 1453 / 1257.1 | 1194 |
| ImagineHouse PRO II | 30699 | 10 | 30699 | 2468 / 3176 / 3069.9 | 3169 |
| Land of Illusion | 16008 | 28 | 16008 | 43 / 1076 / 571.7 | 358 |
| Leviathan | 65912 | 21 | 65912 | 2889 / 3183 / 3138.7 | 3131 |
| Nemo's Market | 28533 | 24 | 28533 | 37 / 3141 / 1188.9 | 3141 |
| Rainbow's End | 7013 | 25 | 7013 | 7 / 1624 / 280.5 | 1624 |
| Slumberland | 37632 | 12 | 37632 | 3136 / 3136 / 3136.0 | 3136 |
| SpacePods | 51283 | 40 | 51283 | 1220 / 1746 / 1282.1 | 1746 |
| Teddy World | 28579 | 45 | 28579 | 7 / 2201 / 635.1 | 1816 |
| Titanic | 6526 | 11 | 6526 | 545 / 642 / 593.3 | 578 |

Every row's "sum of sizes" equals its "payload bytes". For the four `raw ` movies the
sample size is constant 3136 = 64 x 49 x 1 byte and the payload is an exact multiple of
it (28224 = 9 x 3136, 21952 = 7 x 3136, 34496 = 11 x 3136, 37632 = 12 x 3136), which
independently confirms both the geometry and the depth.

### 13.3 Codec identification without a `stsd`

`probe_mov.py classify` distinguishes the three codecs by trial decoding:

1. **`raw `**: payload length is a multiple of 3136 **and** the first 4 bytes do not
   form a plausible self-delimiting length (i.e. `BE32 & 0xFFFFFF` does not chain to a
   clean tiling). Art Museum's first bytes are `31 79 F8 2B` -> masked length
   `0x79F82B` = 7,993,899, wildly out of range. Immediate rejection of `rle `/`smc `.
2. **`rle `**: the length chain closes **and** a full Animation decode of every sample
   consumes exactly its byte count with no opcode errors and no pixel writes out of
   bounds at 64 x 49 (or 64 x 50 / 82 x 62).
3. **`smc `**: the length chain closes but the Animation decode fails, while a Graphics
   (`smc `) decode over `ceil(w/4) * ceil(h/4)` 4x4 blocks consumes exactly the byte
   count. Only Grand Prix.

Grand Prix's first bytes: `A0 00 04 AA A0 00 02 03 04 ...` -> masked length
`0x0004AA` = 1194, first sample size — and 26399 / 21 samples chains exactly. Its
first opcodes (`A0`, then `00 02 03 04...`) are meaningless as Animation (`0xA0` as a
line-skip of 160 immediately overruns a 64-pixel row) and perfect as Graphics: `0xA0` is
the **4-colour, new-quad** opcode with count `(0x0 + 1) = 1` block (section 16.2), so the
four bytes `00 02 03 04` are the new quad and the next four, `10 45 11 65`, are that
block's 2-bits-per-pixel mask. The byte after those is `0x80` — 2-colour, new pair,
1 block — with pair `00 02` and mask `2D 6F`. Every field lands exactly where `smc `
says it should.

Earlier drafts of this analysis mis-identified Grand Prix as `rpza` (Apple Video). That
was wrong: `rpza` uses 2-byte opcodes over 4x4 blocks of RGB555 and would not close the
chain. The `smc ` decode does, block-count-exactly, at 16 x 13 = 208 blocks per frame
x 21 frames = 4368 blocks.

### 13.4 What cannot be recovered

| Lost | Consequence |
|---|---|
| `mvhd`/`mdhd` **timeScale** and `stts` **sampleDuration** | The frame rate of all 14 is unknown. Only `Demo House.mov` (600/121 = 4.9587 fps) is known |
| `tkhd.trackWidth/Height` | The *display* box is unknown; the *encoded* size is recovered from the pixel data (64 x 49 or 64 x 50) and is almost certainly the same |
| `stsd.depth` / `clutID` | Recovered as depth 8 by decoding (see 13.5), but the specific `clut` is a guess for one file (Grand Prix, section 17.3) |
| `vmhd.graphicsMode` | Unknown; `ditherCopy` is likely for grayscale, `srcCopy` for indexed |
| `stss` | Recovered from the codecs' own key-frame flags |
| `elst` | Assumed identity |

The timeScale loss is the only one that affects gameplay appearance, and section 19.5
gives a defensible substitute.

### 13.5 Depth of the 14

All 14 decode consistently at **depth 8** (one byte = one `clut` index):

- The four `raw ` movies are exactly `w*h` bytes per frame, which at 64 x 49 is only
  possible at 8 bpp (at 4 bpp a frame would be 1568 bytes, and length alone does *not*
  rule that out: all four payloads divide by 1568 exactly — 28224/1568 = 18,
  21952/1568 = 14, 34496/1568 = 22, 37632/1568 = 24 — so this test is not decisive; the decisive test is that
  decoding at 4 bpp produces frames whose content is visibly two side-by-side copies of
  the image, i.e. the classic wrong-stride tell, and at 8 bpp produces coherent images.
  Verified by rendering both, see `probe_mov.py sheet`).
- The nine `rle ` movies write 4 pixels per 4-byte "pixel group" (section 14.4), which
  is the depth-8 grouping; at depth 4 a group would be 8 pixels and the decoded row
  lengths would be 128, not 64. Measured row lengths are exactly 64 (or 82 for Demo
  House at depth 36, where groups *are* 8 pixels — and that is precisely the
  discriminator).
- The one `smc ` movie is by definition 8-bit indexed: the Graphics codec has no other
  mode.

---

## 14. The `rle ` codec (Apple Animation) — complete bitstream

Used by 9 of 15 movies plus `Demo House.mov`. Implemented from scratch in
`probe_mov.py` (`decode_rle`); everything below is derived by decoding the shipped data
until every sample consumed exactly its declared length with no errors, across all 10
files.

### 14.1 Sample layout

```
  +0   uint32  chunk       high 8 bits = flags, low 24 bits = byte count
  +4   uint16  header
  [ if header & 0x0008 : uint16 startLine; uint16 zero; uint16 numLines; uint16 zero ]
  then, for each line: uint8 skip; { int8 opcode ... } terminated by opcode == -1
  then one final 0x00 byte  (end of stream)
```

### 14.2 The chunk word

| Bits | Field | Observed values |
|---|---|---|
| 31..24 | flags | `0x40` on key frames; `0x00` on delta frames; `0x80` and `0xA0` on Grand Prix's `smc ` samples |
| 23..0 | byte count | the sample's total size, including these 4 bytes |

`RLE_FLAG_FULL = 0x40` — "this sample is a complete frame". Measured, this bit agrees
with the reconstructed key/delta classification in every one of the 10 `rle ` files.

### 14.3 The header word

| Bit | Mask | Meaning | Observed |
|---|---|---|---|
| 3 | **0x0008** | a 4-word (startLine, 0, numLines, 0) block follows | set on every non-empty sample in every file |
| others | | undefined by the data | always 0 |

Header `0x0000` with nothing after it (i.e. a 4 + 2 + 1 = **7-byte sample**) is a
**no-op frame**: nothing changes. Three such samples occur in Rainbow's End (indices
22, 23, 24) and two in Teddy World. `probe_mov.py` reports these in the `kinds` column
as `no-op=N`. A port must treat them as "hold the previous frame", not as "blank".

When bit 3 is set, the 8 bytes are `startLine, 0x0000, numLines, 0x0000`. The two zero
words are documented as reserved and are 0 in every sample here. Measured over all ten
`rle ` movies: **277 samples, of which 272 have bit 3 set and 5 are 7-byte no-ops**
(3 in Rainbow's End, 2 in Teddy World); the header word is *only* ever `0x0000` or
`0x0008`, and both reserved words are 0 in all 272.

### 14.4 The line stream

For each of `numLines` lines, starting at absolute row `startLine`:

```
 1  skip = uint8                          // 1-based: skip (skip-1) pixel GROUPS
 2  x = (skip - 1) * pixelsPerGroup
 3  loop:
 4      op = int8
 5      if op == -1:  next line                                   // RLE_END_LINE
 6      if op ==  0:  skip = uint8; x += (skip - 1) * pixelsPerGroup; continue
 7      if op >   0:  copy op groups (op * pixelsPerGroup pixels) literally; x += ...
 8      if op <  -1:  read one group; repeat it (-op) times;       x += (-op)*ppg
```

`pixelsPerGroup` depends on the depth:

| Depth | Bits/pixel | Bytes per group | Pixels per group |
|---|---|---|---|
| 8 | 8 | 4 | **4** |
| 36 (4-bit gray) | 4 | 4 | **8** |

So a "pixel group" is always **4 bytes** on the wire; how many pixels those 4 bytes
carry is the depth question. That is why `Demo House.mov`'s 82-pixel-wide rows are
covered by 11 groups (11 x 8 = 88 >= 82) while a 64-pixel-wide depth-8 row needs 16
groups (16 x 4 = 64).

Worked example, `Titanic.mov` sample 0:

```
00000008  40 00 02 42 00 08 00 00  00 00 00 31 00 00 01 F0
          \_ chunk: 0x40 | 578
                      \_ hdr 0x0008
                            \_ startLine 0
                                  \_ resvd 0
                                        \_ numLines 0x31 = 49
                                              \_ resvd 0
                                                    \_ skip = 1  -> x = 0
                                                       \_ op 0xF0 = -16: run of 16 groups
00000018  CF CF CF CF FF 01 F0 CF ...
          \_ the group: 4 x index 0xCF, repeated 16 times = 64 pixels
                      \_ 0xFF = -1: end of line 0
                         \_ line 1: skip = 1, op -16, group CF...
```

`0xCF` = 207. In `clut` 128, index 207 = RGB (0, 51, 102) — a dark navy blue. Titanic's
movie is an underwater scene; a full-width run of dark navy as the first row of the
first frame is exactly right. The file's last two bytes are `FF 00`: `0xFF` = -1 =
end-of-line for the last line, then the end-of-stream `0x00`.

Worked example, `Demo House.mov` sample 0 (depth 36):

```
00000008  40 00 0B 55 00 08 00 00  00 00 00 3E 00 00 01 0B
          \_ chunk: 0x40 | 0x000B55 = 2901
                      \_ hdr 0x0008
                            \_ startLine 0
                                  \_ resvd 0
                                        \_ numLines 0x3E = 62
                                              \_ resvd 0
                                                    \_ skip = 1  -> x = 0
                                                       \_ op 0x0B = 11 literal groups
00000018  BB BB BB AB AB AB AB AB ...
          \_ 11 groups x 4 bytes = 44 bytes = 88 pixels at 4 bpp (row is only 82)
```

`0xBB` at 4 bpp = two pixels of gray index 11 = `(15-11)*17 = 68` -> RGB(68,68,68), a
dark gray; the frame's first row is the top of a dark studio backdrop. Note 88 > 82:
the last group overruns the row by 6 pixels, which QuickTime discards. **A Go decoder
must clamp writes to the row width**, or it will corrupt the next row. This is not a
one-off: measured, **every one of Demo House's 2542 lines** (41 samples x 62 lines)
writes exactly 88 pixels into an 82-pixel row, so the clamp fires 2542 times per loop of
the only movie that works out of the box.

### 14.5 The trailing end-of-stream byte

Every `rle ` sample in this tree — all 10 files, every sample — ends with **one extra
`0x00` byte** after the last line's `-1` terminator. It is a skip count of 0, which
cannot legally begin a line (skip is 1-based), so it functions as an end-of-stream
sentinel.

This was found empirically: before accounting for it, `probe_mov.py` reported
"used N of N+1 bytes" for every single sample. With it, all 10 files consume their
`mdat` payload exactly, with zero warnings. Any Go decoder that stops after `numLines`
lines will be off by one byte per sample, which does not matter if it uses the sample
table but will destroy a reconstruction-based walker.

### 14.6 Measured opcode usage

`python3 tools/probe_mov.py opcodes`:

| Movie | lineSkip | midSkip | literal ops | literal groups | run ops | run groups | endLine | no-op samples |
|---|---|---|---|---|---|---|---|---|
| CD Demo House | 1053 | 596 | 1883 | 6247 | 477 | 1382 | 1053 | 0 |
| Demo House | 2542 | 0 | 2836 | 27161 | 321 | 801 | 2542 | 0 |
| ImagineHouse PRO II | 490 | 0 | 613 | 6979 | 208 | 861 | 490 | 0 |
| Land of Illusion | 708 | 421 | 1053 | 2653 | 333 | 1523 | 708 | 0 |
| Leviathan | 1029 | 0 | 1075 | 15286 | 264 | 1178 | 1029 | 0 |
| Nemo's Market | 623 | 1418 | 2004 | 5188 | 267 | 582 | 623 | 0 |
| Rainbow's End | 295 | 47 | 360 | 1207 | 158 | 705 | 295 | 3 |
| SpacePods | 1467 | 258 | 1784 | 11136 | 181 | 683 | 1467 | 0 |
| Teddy World | 942 | 839 | 1826 | 4928 | 564 | 2414 | 942 | 2 |
| Titanic | 539 | 0 | 170 | 387 | 713 | 8237 | 539 | 0 |

`lineSkip` always equals `endLine`, as it must: one leading skip byte and one `-1`
terminator per line. Two structural observations for a port:

- **Titanic is run-dominated** (8237 run groups vs 387 literal): a nearly flat dark
  underwater image. Its whole movie is 6534 bytes for 11 frames of 64 x 49.
- **Demo House is literal-dominated** (27161 literal groups vs 801 run): a dithered
  photographic grayscale image, which RLE cannot compress. 2897 bytes per frame for
  82 x 62 x 4 bits = 2542 bytes of raw pixels — i.e. it is *bigger* than uncompressed.
  Lossless Animation on dithered continuous tone is a pessimisation, and the author
  paid 118 KB for it.
- **`midSkip` (op == 0) is used by 6 of the 10 files** and not at all by the other 4
  (Demo House, ImagineHouse PRO II, Leviathan, Titanic). A decoder that omits it will
  silently corrupt CD Demo House, Land of Illusion, Nemo's Market, Rainbow's End,
  SpacePods and Teddy World.

### 14.7 Measured line-range usage

How many distinct `(startLine, numLines)` windows appear across a file's samples:

| Movie | Distinct ranges | Notes |
|---|---|---|
| Demo House | 1 | (0, 62) — every sample repaints the whole frame |
| ImagineHouse PRO II | 1 | (0, 49) |
| Leviathan | 1 | (0, 49) |
| Titanic | 1 | (0, 49) |
| CD Demo House | 3 | key frame (0,49) + two narrow windows |
| SpacePods | 5 | |
| Land of Illusion | 17 | |
| Nemo's Market | 18 | |
| Rainbow's End | 20 | |
| Teddy World | 22 | key frames at samples 0, 10, 20, 30, 40, each (0, 50) |

Teddy World's every-10th-sample key frame pattern is the only periodic-keyframe movie in
the set; the rest are "one key frame then all deltas" (CD Demo House, Land of Illusion,
Nemo's Market, Rainbow's End, SpacePods) or "all key frames" (Demo House, ImagineHouse,
Leviathan, Titanic, and all four `raw ` movies).

**Delta frames mean a port cannot seek randomly.** Playback must start at a key frame
and decode forward. Since `ToggleTV` always rewinds to frame 0 (`Trip.c:47`) and frame
0 is always a key frame, this costs nothing in practice — but a port that pre-decodes
all frames into a slice at load time (recommended) sidesteps the issue entirely, at a
cost of at most **41 x 5084 = 208,444 bytes** — Demo House, the largest by decoded size
(Teddy World, the largest by frame count, is 45 x 3200 = 144,000). The whole corpus is
1,139,580 bytes decoded (section 19.3).

### 14.8 Full per-sample dump, `Rainbow's End.mov`

The smallest interesting file, shown in full as a decoder conformance vector
(`probe_mov.py samples "Rainbow's End"`):

| # | file offset | size | chunkSize | flag | header | startLine | numLines | kind |
|---|---|---|---|---|---|---|---|---|
| 0 | 8 | 1624 | 1624 | 0x40 | 0x0008 | 0 | 49 | **key** |
| 1 | 1632 | 36 | 36 | 0x00 | 0x0008 | 11 | 3 | delta |
| 2 | 1668 | 80 | 80 | 0x00 | 0x0008 | 9 | 7 | delta |
| 3 | 1748 | 127 | 127 | 0x00 | 0x0008 | 9 | 9 | delta |
| 4 | 1875 | 188 | 188 | 0x00 | 0x0008 | 7 | 12 | delta |
| 5 | 2063 | 245 | 245 | 0x00 | 0x0008 | 7 | 14 | delta |
| 6 | 2308 | 294 | 294 | 0x00 | 0x0008 | 9 | 14 | delta |
| 7 | 2602 | 356 | 356 | 0x00 | 0x0008 | 11 | 15 | delta |
| 8 | 2958 | 388 | 388 | 0x00 | 0x0008 | 14 | 15 | delta |
| 9 | 3346 | 416 | 416 | 0x00 | 0x0008 | 17 | 15 | delta |
| 10 | 3762 | 425 | 425 | 0x00 | 0x0008 | 20 | 14 | delta |
| 11 | 4187 | 408 | 408 | 0x00 | 0x0008 | 22 | 13 | delta |
| 12 | 4595 | 363 | 363 | 0x00 | 0x0008 | 23 | 12 | delta |
| 13 | 4958 | 360 | 360 | 0x00 | 0x0008 | 22 | 13 | delta |
| 14 | 5318 | 337 | 337 | 0x00 | 0x0008 | 20 | 14 | delta |
| 15 | 5655 | 336 | 336 | 0x00 | 0x0008 | 18 | 14 | delta |
| 16 | 5991 | 292 | 292 | 0x00 | 0x0008 | 16 | 14 | delta |
| 17 | 6283 | 262 | 262 | 0x00 | 0x0008 | 14 | 14 | delta |
| 18 | 6545 | 212 | 212 | 0x00 | 0x0008 | 13 | 13 | delta |
| 19 | 6757 | 132 | 132 | 0x00 | 0x0008 | 13 | 10 | delta |
| 20 | 6889 | 82 | 82 | 0x00 | 0x0008 | 13 | 9 | delta |
| 21 | 6971 | 29 | 29 | 0x00 | 0x0008 | 13 | 2 | delta |
| 22 | 7000 | 7 | 7 | 0x00 | 0x0000 | — | — | **no-op** |
| 23 | 7007 | 7 | 7 | 0x00 | 0x0000 | — | — | **no-op** |
| 24 | 7014 | 7 | 7 | 0x00 | 0x0000 | — | — | **no-op** |

Sum = 7013 = the `mdat` payload exactly, and 8 + 7013 = 7021 = the file size.
`chunkSize` equals the recovered sample size on every row, which is the whole basis of
the reconstruction. The flag byte is `0x40` on exactly one sample (index 0) and `0x00`
on the other 24, matching the "one key frame then all deltas" classification.

The animation is legible from the line windows alone: the update band opens at rows
11-13 (sample 1, 36 bytes), widens to 15 lines while drifting down the frame
(rows 7 -> 23 over samples 4-12), then narrows back to 2 lines at row 13 (sample 21,
29 bytes), then three frames of literally nothing while the finished picture is held.
That is a rainbow being drawn arc by arc and then admired. 25 frames in 7021 bytes.

Three properties of this table generalise to all 10 `rle ` movies and are worth a Go
port asserting at load time:

1. `chunkSize == sampleSize` for every sample;
2. `startLine + numLines <= height` for every sample;
3. the sizes tile the `mdat` payload exactly.

---

## 15. The `raw ` codec (uncompressed) — 4 movies

Art Museum, Castle o' the Air, Davis Station, Slumberland. The simplest possible case
at depth 8:

```
  sample = width * height bytes, one clut index per pixel,
           row-major, top-to-bottom, left-to-right,
           NO header, NO length prefix, NO row padding
```

Measured: 64 x 49 = **3136 bytes per sample**, and each file's payload is an exact
multiple:

| Movie | Payload | / 3136 | Frames |
|---|---|---|---|
| Art Museum | 28224 | 9.0 | 9 |
| Castle o' the Air | 21952 | 7.0 | 7 |
| Davis Station | 34496 | 11.0 | 11 |
| Slumberland | 37632 | 12.0 | 12 |

**No row padding.** This is the one place QuickTime's `raw ` differs from a QuickDraw
`PixMap`, whose `rowBytes` is always even and usually padded to a multiple of 4 (see
`rendering.md` §3). In a `.mov` sample there is no `rowBytes`; rows are packed. A Go
port that reuses its PICT/PixMap loading path here will get a sheared image. Verified:
decoding Art Museum with `rowBytes = 64` gives the Mona Lisa; with `rowBytes = 68` it
shears.

First 4 bytes of `Art Museum.mov`'s first sample: `31 79 F8 2B` — indices 49, 121, 248,
43. In `clut` 128 those are respectively (204,153,204) pale mauve, (102,153,204) medium
blue, (170,170,170) light gray and (204,204,204) light gray. (These four are *light*
pixels, so they are not part of the dark canvas itself; the first scan line's mean
luminance is 107/255 against 88/255 for the whole frame. What exactly occupies the
top-left corner is not determinable from four bytes — read the contact sheet.) No plausible reading of
`31 79 F8 2B` as a length prefix exists (`0x79F82B` = 7,993,899 > file size), which is
the automatic `raw ` detection in `probe_mov.py classify`.

**Every `raw ` sample is a key frame** — there is no delta mode. So these four movies
are freely seekable.

---

## 16. The `smc ` codec (Apple Graphics) — 1 movie

Only `Grand Prix.mov`. `smc ` is 8-bit-indexed-only and works on **4 x 4 pixel blocks**.

### 16.1 Sample layout

```
  +0   uint32  chunk      high 8 bits = flags, low 24 bits = byte count
  +4   ...     opcode stream over ceil(w/4) * ceil(h/4) blocks
```

Block order is raster over blocks: left to right in 4-pixel columns, then down 4 rows.
At 64 x 49 that is `ceil(64/4) = 16` across by `ceil(49/4) = 13` down = **208 blocks per
frame** (the last block row covers rows 48..51, of which only row 48 exists — writes to
rows 49..51 are discarded). 208 x 21 frames = **4368 blocks**, which is exactly what
`probe_mov.py` counts.

### 16.2 Opcodes

The opcode byte's high nibble selects the operation and the low nibble is a count minus
one, except that for the opcode classes 0x00-0x7F **bit 0x10 is an escape flag**: when
it is set (i.e. the high nibble is odd — 0x1?, 0x3?, 0x5?, 0x7?) the count comes from an
extra byte that follows the opcode, as `count = byte + 1`, and the opcode's own low
nibble is unused. There is no 2-byte-count form. The 0x80-0xEF classes have no escape
flag at all: bit 0x10 there selects new-table versus cached-table, and their count is
always the low nibble plus one (so at most 16 blocks per opcode). This is what the table
below encodes and what `probe_mov.py`'s `decode_smc` implements.

| Opcode | Name | Payload | Blocks affected |
|---|---|---|---|
| 0x00-0x0F | skip | — | 1..16 blocks unchanged from the previous frame |
| 0x10-0x1F | skip (extended) | 1 count byte | count+1 blocks unchanged |
| 0x20-0x2F | repeat last block | — | 1..16 copies of the previous block |
| 0x30-0x3F | repeat last block (extended) | 1 count byte | |
| 0x40-0x4F | repeat last **pair** of blocks | — | |
| 0x50-0x5F | repeat last pair (extended) | 1 count byte | |
| 0x60-0x6F | **1-colour** | 1 index byte | n blocks all of that index |
| 0x70-0x7F | 1-colour (extended) | 1 count + 1 index | |
| 0x80-0x8F | **2-colour**, new pair | 2 index bytes + n x uint16 bitmask | 16 bits, one per pixel, MSB first |
| 0x90-0x9F | **2-colour**, cached pair | 1 cache index + n x uint16 | |
| 0xA0-0xAF | **4-colour**, new quad | 4 index bytes + n x uint32 | 32 bits = 2 bits per pixel |
| 0xB0-0xBF | **4-colour**, cached quad | 1 cache index + n x uint32 | |
| 0xC0-0xCF | **8-colour**, new octet | 8 index bytes + n x 6 bytes | 48 bits = 3 bits per pixel |
| 0xD0-0xDF | **8-colour**, cached octet | 1 cache index + n x 6 bytes | |
| 0xE0-0xEF | **16-colour** (direct) | n x 16 index bytes | one index per pixel |
| 0xF0-0xFF | reserved | | |

The three caches are tables of previously-used colour sets. **They belong to the track,
not to the sample: both their contents and their write pointers persist across frames**
(see 16.3 — this file depends on that heavily).

| Cache | Entries | Colours per entry |
|---|---|---|
| pair cache | 256 | 2 |
| quad cache | 256 | 4 |
| octet cache | 256 | 8 |

`COLORS_PER_TABLE = 256`, `CPAIR = 2`, `CQUAD = 4`, `COCTET = 8`. A "new" opcode
(0x80/0xA0/0xC0) writes its colour set at the cache's current write index and then
advances that index modulo 256; a "cached" opcode (0x90/0xB0/0xD0) reads the slot its
payload byte names. `probe_mov.py`'s `SmcState` class implements exactly this, and is
constructed **once per movie**, not once per sample.

In `Grand Prix.mov` the write pointers end at pair 126, quad 19, octet 126, so the
modulo-256 wrap never actually occurs in this file; a port may not rely on that for
third-party movies.

### 16.3 Measured usage in `Grand Prix.mov`

| Opcode | Count |
|---|---|
| 0x20 (repeat block) | 2 |
| 0x60 (1-colour) | 56 |
| 0x80 (2-colour new) | 126 |
| 0x90 (2-colour cached) | 311 |
| 0xA0 (4-colour new) | 531 |
| 0xB0 (4-colour cached) | 672 |
| 0xC0 (8-colour new) | 382 |
| 0xD0 (8-colour cached) | 320 |
| 0xE0 (16-colour) | 66 |
| **total blocks** | **4368 = 21 x 208** |

Never used: **0x00/0x10 (skip), 0x30, 0x40/0x50 (pair repeat), 0x70, 0xF0**. Because the
skip opcodes are absent, every `smc ` frame here paints all 208 of its blocks, so no
*pixel* is inherited from the previous frame.

**That does not make the movie seekable.** The colour caches are inherited, and Grand
Prix leans on them across frame boundaries: measured, 1214 of its 1303 cached-table
opcodes (290 pair, 624 quad, 300 octet) name a cache slot that was *not* written in the
same frame. Per frame:

| Frame | Opcodes reading a stale cache slot | Blocks they colour |
|---|---|---|
| 0 | 0 | 0 / 208 |
| 1..20 | 48..73 each | 61..122 of 208 each |

So roughly half of every frame after the first is coloured out of tables loaded earlier
in the stream. A decoder that starts at frame *N* > 0, or that resets `SmcState` per
sample, produces garbage colours for about half the picture. **Grand Prix must be decoded
from frame 0 forward with a single persistent `SmcState`**; only then are all 21 frames
byte-exact. The 0x80/0xA0 chunk flag bytes, whatever they meant, do not mark
independently decodable frames.

### 16.4 Corollary: the chunk flag bytes are not key-frame markers

Grand Prix's chunk flag bytes are `0xA0` on 20 samples and `0x80` on
sample 6. Neither corresponds to `rle `'s `0x40` key-frame convention, and no
interpretation is derivable from this data. Masking to 24 bits — which the spec
requires — is what makes the length chain close exactly, so the flags can be ignored.

---

## 17. Palettes

### 17.1 `clut` 128 for the 14 depth-8 movies

The game runs 8-bit indexed and installs `clut` 128 from its own resource fork
(`rendering.md` §5). A depth-8 movie sample with `clutID` -1 (or with no header at all,
as here) draws through the destination's colour table, i.e. `clut` 128. So all 14
headerless movies are rendered by `probe_mov.py` against `clut` 128 via
`probe_rez.parse` + `probe_pict.parse_clut`.

Sanity anchors, measured from `GliderPRO/Glider PRO.r`:

| Index | RGB |
|---|---|
| 0 | (255, 255, 255) white |
| 214 | (0, 0, 51) |
| 215 | (238, 0, 0) |
| 254 | (17, 17, 17) |
| 255 | (0, 0, 0) black |

### 17.2 Plausibility check per movie

For each movie, the four most-frequent indices and their `clut` 128 RGBs, against the
visible content:

Measured over every decoded frame of every movie: the four most-frequent indices with
their `clut` 128 RGBs, plus how many distinct indices the movie touches.

| Movie | Distinct | Range | Top 4 indices -> RGB | Verdict |
|---|---|---|---|---|
| Art Museum | 106 | 0..255 | 255 (0,0,0), 244 (0,0,17), 254 (17,17,17), 85 (153,153,204) | **plausible** — a very dark dithered painting |
| CD Demo House | 59 | 0..255 | 150 (51,204,255), 0 (255,255,255), 36 (204,255,255), 11 (255,204,0) | **plausible** — sky blue, white clouds, orange logo |
| Castle o' the Air | 54 | 0..252 | 150 (51,204,255), 206 (0,51,153), 199 (0,102,204), 107 (153,0,0) | **plausible** — blue sky, blue and red flag |
| Davis Station | 19 | 0..255 | 255 (0,0,0), 137 (102,51,0), 250 (119,119,119), 95 (153,102,0) | **plausible** — dark jar, brown/orange sparks |
| ImagineHouse PRO II | 70 | 7..255 | 255 (0,0,0), 234 (0,17,0), 224 (17,0,0), 254 (17,17,17) | **plausible** — a very dark green-black plant |
| Land of Illusion | 13 | 0..255 | 150 (51,204,255), 0 (255,255,255), 255 (0,0,0), 86 (153,153,153) | **plausible** — white bird on sky blue |
| Leviathan | 81 | 1..255 | 255 (0,0,0), 223 (34,0,0), 224 (17,0,0), 246 (221,221,221) | **plausible** — near-black with dark reds |
| Nemo's Market | 13 | 18..213 | 19 (255,102,204), 55 (204,102,204), 213 (0,0,102), 20 (255,102,153) | **plausible** — hot pink text on navy |
| Rainbow's End | 61 | 4..252 | 150 (51,204,255), 231 (0,85,0), 27 (255,51,102), 114 (102,204,255) | **plausible** — sky, grass, rainbow reds |
| Slumberland | 246 | 0..255 | 255 (0,0,0), 0 (255,255,255), 242 (0,0,68), 232 (0,68,0) | **plausible** — a heavily dithered portrait |
| SpacePods | 49 | 43..255 | 255 (0,0,0), 51 (204,153,102), 93 (153,102,102), 57 (204,102,102) | **plausible** — rusty planet on black |
| Teddy World | 46 | 0..255 | 43 (204,204,204), 126 (102,102,255), 53 (204,153,0), 59 (204,102,0) | **plausible** — gray studio, blue backdrop, brown bear |
| Titanic | **5** | 0..207 | 207 (0,51,102), 114 (102,204,255), 193 (0,153,204), 199 (0,102,204) | **plausible** — four blues and white |
| **Grand Prix** | 58 | 0..58, +255 | 2 (255,255,153), 19 (255,102,204), 4 (255,255,51), 16 (255,153,51) | **IMPLAUSIBLE** — a racing scene in pastel yellow and hot pink |

Two rows deserve comment. **Titanic uses only 5 distinct indices in its entire 11-frame
movie** (0, 114, 193, 199, 207) — which is why it compresses to 6534 bytes and why its
`rle ` stream is 96% runs. And **Slumberland uses 246 of 256**, a full-gamut dithered
photograph, which is why an uncompressed codec was the right choice for it.

The `Range` column is the diagnostic: 13 of the 14 movies scatter indices across the
whole 0..255 space (min 0..43, max 207..255), exactly as you would expect from indices
into a fixed 256-entry system palette. Grand Prix does not: its min/max span looks like
0..255 only because of the single value 255, and everything else is packed into 0..58.

(All 15 rows above were re-measured independently with `probe_mov.py`'s own `Movie` over
every frame of every movie and agree exactly. Grand Prix's row is only reproducible if
the `smc ` colour caches persist across frames — see 16.3; reset them per sample and the
top four come out as 19, 2, 16, 4 instead.)

### 17.3 Grand Prix's palette is unrecoverable

Its `smc ` stream uses exactly these 58 indices:

```
0 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28
29 30 31 32 33 34 35 36 37 38 39 40 41 42 43 44 45 46 47 49 50 51 52 53 54
55 56 57 58 255
```

i.e. **0..58 contiguously except 1 and 48, plus 255**. A compact table packed from index
0 upward, with 255 used as (presumably) a mask or background colour, is the unmistakable
signature of a **custom `ColorTable` embedded in the `stsd`'s `ImageDescription`**
(`clutID` = 0) — the output of the Movie Converter's "optimise palette" pass. It is not
what indices into a fixed 256-entry system palette look like. That table was in the lost
header, and nothing else in this tree contains a 59-entry `clut`.

The consequence: Grand Prix's movie can be decoded structurally (all 21 frames, all
4368 blocks, byte-exact) but **cannot be coloured correctly**. Rendered against
`clut` 128 the shapes are clearly a blocky racing scene — kerbs, a car, a track — in
completely wrong hues.

Options for a port, in decreasing fidelity:
1. hand-author a 59-entry palette by eye from the shapes (a racing scene: greens, grays,
   reds, a sky) — a guess, but a defensible one;
2. render it in grayscale by mapping index *i* to `i * 255 / 58`;
3. treat Grand Prix as "no movie" and show static snow.

### 17.4 `Demo House.mov`'s grayscale palette

`depth` 36, `clutID` 36: the standard Mac 4-bit gray ramp, `GetCTable(36)`.

| Index | Gray | RGB |
|---|---|---|
| 0 | 255 | (255,255,255) |
| 1 | 238 | (238,238,238) |
| 2 | 221 | ... |
| ... | | |
| 14 | 17 | (17,17,17) |
| 15 | 0 | (0,0,0) |

Formula: `gray(i) = (15 - i) * 17`. `probe_mov.py`'s `palette_gray(4)` generates it. The
other grayscale depths a port might see are 33 (1-bit, `(1-i)*255`), 34 (2-bit,
`(3-i)*85`) and 40 (8-bit, `255-i`); none occur in this tree.

Then `vmhd.graphicsMode` = 64 (`ditherCopy`) means the 16 grays are dithered into the
game's 8-bit `clut` 128 at draw time (section 12.7).

---

## 18. Per-movie summary and content

Everything measured. "Frames" and "W x H" for the 14 are reconstructed (section 13);
for Demo House they are read from the header.

| Movie | Bytes | Codec | Frames | W x H | Depth | Key/delta | Content |
|---|---|---|---|---|---|---|---|
| Art Museum | 28232 | `raw ` | 9 | 64 x 49 | 8 | all key | The Mona Lisa, heavily dithered, dark |
| CD Demo House | 33042 | `rle ` | 32 | 64 x 49 | 8 | 1 key + 31 delta | An orange-and-white bird logo over sky-blue clouds with "CD" |
| Castle o' the Air | 21960 | `raw ` | 7 | 64 x 49 | 8 | all key | A waving blue/red flag bearing a star and a moon |
| Davis Station | 34504 | `raw ` | 11 | 64 x 49 | 8 | all key | Orange sparks swirling inside a dark jar |
| **Demo House** | **119789** | **`rle `** | **41** | **82 x 62** | **36 (4-bit gray)** | **all key** | A TV news anchor, "AIR CRASH" chyron, plane graphic |
| Grand Prix | 26407 | `smc ` | 21 | 64 x 49 | 8 | every frame repaints all pixels, but the colour caches carry over — **not seekable** (16.3) | A blocky racing scene — **palette lost** |
| ImagineHouse PRO II | 30707 | `rle ` | 10 | 64 x 49 | 8 | all key | A Venus-flytrap-like plant, dark |
| Land of Illusion | 16016 | `rle ` | 28 | 64 x 49 | 8 | 1 key + 27 delta | A white origami bird flying on sky blue |
| Leviathan | 65920 | `rle ` | 21 | 64 x 49 | 8 | all key | A dark jungle/organic scene |
| Nemo's Market | 28541 | `rle ` | 24 | 64 x 49 | 8 | 1 key + 23 delta | "nemo's" zooming in, hot pink on navy |
| Rainbow's End | 7021 | `rle ` | 25 | 64 x 49 | 8 | 1 key + 21 delta + 3 no-op | A rainbow forming over grass under clouds |
| Slumberland | 37640 | `raw ` | 12 | 64 x 49 | 8 | all key | A speckled portrait (Little Nemo) |
| SpacePods | 51291 | `rle ` | 40 | 64 x 49 | 8 | 1 key + 39 delta | A rusty planet rotating on black |
| Teddy World | 28587 | `rle ` | 45 | **64 x 50** | 8 | 5 key + 38 delta + 2 no-op | A teddy-bear news anchor, "NEWS UPDATE" |
| Titanic | 6534 | `rle ` | 11 | 64 x 49 | 8 | all key | Dark blue water with pale blue bubbles |

Aggregate: **536,191 bytes, 15 movies, 337 frames, 3 codecs, 2 depths, 3 geometries.**
Frame counts sum as 9+32+7+11+41+21+10+28+21+24+25+12+40+45+11 = 337.

Content was read off contact sheets produced by `probe_mov.py sheet`; the descriptions
above are what those PNGs show.

### 18.1 The two odd sizes

- **Teddy World is 64 x 50**, one row taller than the TV screen. Its 5 key frames
  (samples 0, 10, 20, 30, 40) each declare `startLine = 0, numLines = 50`, which is how
  the height is known: an `rle ` sample cannot declare more lines than the image has.
  Section 7.1 shows the consequence — the bottom row is clipped away.
- **Demo House is 82 x 62**, from its surviving `tkhd`. Section 7.1 shows 9 columns lost
  each side and 6 rows top / 7 rows bottom.

Both are authoring mistakes, both are visible in the original, and a faithful port
reproduces them.

---

## 19. What a Go port should do

`docs/analysis/rendering.md` §19.7 leaves four questions open. This section answers all
four from the measurements above.

### 19.1 Question (a): which houses can actually show a movie in a from-scratch port?

**Strictly by parsing the container: exactly one — Demo House.** The other 14 files have
no movie header, so a port that implements QuickTime faithfully (or that shells out to
any existing decoder) gets 1 of 15.

**With the reconstruction in section 13: 14 of 15.** Every file's samples are recovered
byte-exactly and every frame decodes cleanly. The one that still fails is **Grand Prix**,
which decodes structurally but has lost its colour table (section 17.3), so its 21 frames
render in wrong hues.

Summary table for the port:

| Path | Movies that work | Notes |
|---|---|---|
| parse the container only | 1 (Demo House) | the honest QuickTime answer |
| + reconstruct headerless `mdat`s (section 13) | 14 | all but Grand Prix's colours |
| + hand-author a palette for Grand Prix | 15 | the palette is a guess |
| + pre-transcode everything to a native asset | 15 | **recommended**, see 19.3 |

And note the second-order limit: *even with all 15 working*, a movie is only visible when
the player stands in the room containing that house's **first** `kTV`, so 65 of the 80
`kTV` placements in the shipped houses show static snow no matter what (section 9.2).

### 19.2 Question (b): what should a port render on the 14 broken ones?

Three defensible answers, in increasing order of effort and fidelity.

**B1 — Bug-compatible (least work, worst result).** Treat a missing header exactly as
`OpenHouseMovie` does: leave `hasMovie` false, so
- the TV shows the static snow bitmap `tvScreen2` when on and `tvScreen1` when off,
  drawn by `DrawTV` (`ObjectDraw2.c:677-688`) and re-blitted by `HandleTV`
  (`Dynamics.c:455-458`);
- `tvInRoom` stays false, `tvWithMovieNumber` stays -1;
- the splash line reads `"House: Teddy World"` with **no** `" (QT)"`
  (`MainWindow.c:62`);
- the TV still toggles, still plays `kTVOnSound` (32) and `kTVOffSound` (33), still
  dirties its rect. Only the moving picture is absent.

**One deliberate deviation is required here.** The original would raise
`YellowAlert(kYellowQTMovieNotLoaded, ...)` from `HouseIO.c:98` when
`NewMovieFromFile` failed, i.e. a modal alert on **every one of the 14 house opens**.
That never happened in 1996 because the resource forks existed; it is purely an artifact
of the source release. **Suppress the alert** (or downgrade it to a log line) and take
the silent `hasMovie = false` path that `HouseIO.c:83-85` uses for houses with no movie
file at all. Reproducing the alert would be faithful to the code and unfaithful to the
game.

**B2 — Reconstruct at load time.** Port `probe_mov.py`'s reconstruction (section 13.1)
plus the three codecs (sections 14-16) into the Go loader: ~600 lines total. Then all 14
work. The cost is carrying three 1990s codecs in the binary forever, and the payoff is
that user-authored `.mov` files with intact headers also work.

**B3 — Pre-transcode (recommended).** See 19.3.

For **Grand Prix specifically**, whichever of B1-B3 is chosen, the palette is gone. Rank
the options: (i) hand-author a 59-entry palette from the visible shapes — a racing scene
needs greens, grays, kerb reds and whites, and a sky; (ii) render it grayscale as
`i * 255 / 58`; (iii) treat it as B1, static snow. Option (i) is what a shipping port
should do, with a comment saying the palette is invented.

### 19.3 Question (c): must the `rle ` codec be implemented, or can the movie be transcoded?

**It need not be implemented at run time. Transcode ahead of time.**

The whole corpus is **337 frames** totalling, fully decoded to 8-bit indices,
**1,139,580 bytes**:

| | Movies | Frames | Bytes decoded (frames x w x h x 1) |
|---|---|---|---|
| 64 x 49 | 13 | 251 | 251 x 3136 = **787,136** |
| 64 x 50 (Teddy World) | 1 | 45 | 45 x 3200 = **144,000** |
| 82 x 62 (Demo House) | 1 | 41 | 41 x 5084 = **208,444** |
| **total** | **15** | **337** | **1,139,580** |

1.1 MB uncompressed, which `flate`s to a fraction of that and is trivially embeddable
with `go:embed`. That is 2.1x the 536,191 bytes of original `.mov` files — but it removes
~600 lines of 1990s codec from the binary, and unlike the originals it actually works.

Recommended asset format, one file per movie, deliberately close to what
`probe_mov.py extract` already emits:

```
  header (all little-endian, the port's own format, not QuickTime's):
    magic       "GMOV"                4 bytes
    version     uint16 = 1
    width       uint16                64, 64 or 82
    height      uint16                49, 50 or 62
    nframes     uint16                7..45
    frameTicks  uint16                see 19.5
    paletteKind uint8                 0 = game 'clut' 128, 1 = 4-bit gray ramp,
                                      2 = inline table follows
    reserved    uint8
    [ if paletteKind == 2: uint16 nEntries, then nEntries x {R,G,B} uint8 ]
  then nframes x (width * height) bytes of palette indices, top row first
```

Every frame stored as a full frame: no deltas, no key-frame bookkeeping, seekable, and
`ToggleTV`'s unconditional rewind (`Trip.c:47`) becomes `frameIndex = 0`.

`probe_mov.py extract <outdir>` already writes exactly the payload part (raw 8-bit index
frames) plus a `manifest.tsv` giving width, height, frame count, codec, depth and
palette kind per movie — so the transcode step is one command, run once, checked in.

**When you would still want the codecs:** if the port aims to load third-party houses
with their own `.mov` files. Glider PRO's house format is documented and people authored
houses for years; such a movie will have an intact `moov` (it was distributed with its
resource fork, or as a modern single-fork file) and will be `rle `, `raw `, `smc `,
`rpza`, `cvid` or `jpeg`. That is an open-ended commitment. A reasonable middle path:
implement the container parser plus `rle `/`raw ` (the two easy ones, ~250 lines), and
fall back to `hasMovie = false` for anything else.

### 19.4 Question (d): what should `tvOn` / `tvInRoom` do when `hasMovie` is false?

Read straight off the source. With `hasMovie == false`:

| Variable | Value | Why |
|---|---|---|
| `hasMovie` | `false` | set by `OpenHouse` (`HouseIO.c:196`) and by `CloseHouseMovie` (`HouseIO.c:158`); `OpenHouseMovie` only sets it true at `HouseIO.c:139` |
| `tvInRoom` | **stays `false` forever** | the only site that sets it true is `ObjectDrawAll.c:708`, guarded by `hasMovie` at `:704` |
| `tvWithMovieNumber` | **stays -1 forever** | same guard; reset at `RoomGraphics.c:60`, `:410`, `HouseIO.c:198` |
| `tvOn` | **never assigned** | the three writers are `ObjectDrawAll.c:693`, `Trip.c:49`, `Trip.c:54`, all inside `hasMovie` guards. It keeps whatever it had — `false` at program start (a static `Boolean` in `Play.c:54`) |
| `theMovie` | undefined/`nil` | never touched |

And because `tvInRoom` is false, every consumer short-circuits: `Play.c:202`
(`SetMovieActive`), `Play.c:224` (`StopMovie`), `Play.c:466`/`:491` (`MoviesTask`),
`Transit.c:304`/`:342`/`:381`/`:420` (restart), `RoomGraphics.c:407` (`ReadyLevel`
stop), `Dynamics.c:437`/`:449` (the suppression), `Trip.c:43` (`ToggleTV`).

So the concrete answer for a port: **`tvOn` is dead state when there is no movie — do
not let the TV's on/off *appearance* depend on it.** The TV's visible state comes from
`dinahs[n].active` and `thisObject.data.g.state`, which are maintained
unconditionally (`Trip.c:42`, `Dynamics3.c:281`, `Play.c:688-689`). `tvOn` is only a mirror
kept for the movie's benefit. A Go port is better off deleting `tvOn` entirely and
asking `dinahs[tvWithMovieNumber].active` — the two can never disagree, because
`Trip.c:42` and `:49`/`:54` are a handful of lines apart, and `ObjectDrawAll.c:693` copies from
the same `data.g.state` that `AddDynamicObject` copies into `active`
(`Dynamics3.c:281`).

One trap: `tvOn` is **not** reset when the house changes. If house A's TV was on and
house B has no movie, `tvOn` stays `true`. Harmless in the original (every reader also
checks `tvInRoom`), but a port that simplifies the guards must not drop that check.

### 19.5 The frame-rate problem, and a defensible answer

Only `Demo House.mov` has a known rate: **121 units at time scale 600 = 4.9587 fps**
(section 12.10). The other 14 lost their `mvhd`/`mdhd`/`stts`, so their rates are
**unrecoverable from this tree**.

Constraints a port can use:

1. Glider PRO's game frame is **2 ticks = 2/60 s**, i.e. 30 fps nominal
   (`rendering.md` §12).
2. `MoviesTask` is called once per game frame (`Play.c:466`, `:491`), so the movie's
   effective ceiling is 30 fps.
3. Demo House's 4.9587 fps is 30 / 6.05 — i.e. **one movie frame every 6 game frames**,
   to within 1%.
4. The frame counts are small (7 to 45) and the movies loop forever, so the *duration*
   is the visible quantity: at one frame per 6 game frames, Titanic's 11 frames loop in
   2.2 s, Rainbow's End's 25 in 5.0 s, Teddy World's 45 in 9.0 s. All plausible for a
   television-in-the-corner animation.

**Recommendation: advance one movie frame every 6 game frames (5 fps) for all 15
movies**, i.e. `frameTicks = 6` in the asset header of 19.3, and note in a comment that
this is exact-to-1% for Demo House and a considered guess for the other 14. An
alternative defensible choice is 10 game frames (3 fps) for the four `raw ` movies,
whose 7-12 frame counts suggest slow deliberate loops — but there is no evidence for it
and uniformity is worth more.

Do **not** try to derive a rate from file size or frame count; there is no relationship
in the data (Titanic: 11 frames / 6534 bytes; Teddy World: 45 frames / 28587 bytes).

### 19.6 The clipping question: reproduce it or fix it?

Section 7.1 measures the damage: Teddy World loses its bottom row; Demo House loses 9
columns each side and 6 rows top / 7 bottom, cutting off the "AIR CRASH" chyron.

**Reproduce it.** It is what the original did, it is cheap (`min`/`max` on the blit
rect), and the alternative — scaling the movie to fit — is a visible change to the one
house that most players saw first. If the port wants a "fixed" mode, put it behind an
option and default it off.

The one thing a port must *not* do is what a naive implementation does by accident:
letting the oversized movie draw outside the TV's 64 x 49 screen and over the TV's
bezel. `SetMovieDisplayClipRgn` (`ObjectDrawAll.c:691`) is not decoration.

### 19.7 Where to draw it

The original draws to the window, past the compositor (section 8). A port has a free
choice:

| Choice | Faithful? | Consequences |
|---|---|---|
| draw to the framebuffer after the compositor publishes | **yes** | reproduces all four artifacts of section 8, including the glider vanishing behind the movie |
| draw into the work map before the compositor | no | glider correctly occludes the movie; dirty rects work; the abandoned `Render.c:55`/`:59` externs suggest the author wanted this |

Recommend the **second** for a modern port, with the first available as a compatibility
switch, and document the divergence. The artifact is a genuine bug, it is not
load-bearing for any puzzle (a TV is scenery; `Interactions.c:1101` only toggles it),
and no house depends on the glider being hidden.

### 19.8 Cross-reference map to `rendering.md` §19

| `rendering.md` | This document |
|---|---|
| §19.1 (`OpenHouseMovie`, `HouseIO.c:67-142`) | section 4, with the numbered walkthrough and the `NewHandle(307200)` explanation |
| §19.2 (`LoopMovie`, `PrerollMovie`) | sections 4.1 and 5, plus the measurement that no shipped file has a `udta` (12.15) |
| §19.3 (`tvWithMovieNumber`, `movieRect`) | sections 3, 6.5, 7 — including that `movieRect` is screen-resolution dependent |
| §19.4 (the `ObjectDrawAll.c:680-708` first-TV rule) | section 7, plus the 80-placement census in 9.2 |
| §19.5 (`MoviesTask` and the dirty-rect pipeline) | section 8, with the four enumerated artifacts and the `Render.c` dead externs |
| §19.6 (`DrawTV`, `tvScreen1`/`tvScreen2`) | sections 6.1-6.4 |
| §19.7 (open questions) | **section 19, answered** |
| — (not covered there at all) | sections 10-18: the container, the three codecs, the palettes, the 14 headerless files and their recovery |

---

## 20. Recommended Go runtime model

A minimal shape that covers everything the original does, with no Toolbox concepts left:

```go
// One per house, or nil.
type TVMovie struct {
    W, H       int          // 64x49, 64x50 or 82x62
    Frames     [][]uint8    // len = nframes; each len = W*H palette indices
    Palette    color.Palette // clut 128, the 4-bit gray ramp, or an inline table
    FrameTicks int          // game frames per movie frame; 6 (section 19.5)
}

// Per-game state, replacing theMovie/movieRect/hasMovie/tvInRoom/
// tvWithMovieNumber/tvOn.
type tvState struct {
    movie    *TVMovie   // nil == hasMovie false
    inRoom   bool       // tvInRoom
    dynaNum  int        // tvWithMovieNumber, -1 when none
    playing  bool       // tvOn
    frame    int        // frame cursor; 0 after every start
    tick     int        // 0..FrameTicks-1
    box      image.Rectangle // movieRect, in framebuffer coords
    clip     image.Rectangle // the 64x49 screen rect
}
```

Then:

| Original | Go |
|---|---|
| `OpenHouseMovie` | `loadTVMovie(houseName)` -> `*TVMovie, error`; on error, `movie = nil`, log, no alert (19.2) |
| `CloseHouseMovie` | `s.movie = nil` |
| `GetMovieBox` + `CenterRectInRect` + `SetMovieBox` | `s.box = centerIn(image.Rect(0,0,W,H), s.clip)` with **truncating** division (7.1) |
| `SetMovieDisplayClipRgn` | `dst := s.box.Intersect(s.clip)` at blit time |
| `SetTimeBaseFlags(loopTimeBase)` | `s.frame = (s.frame + 1) % len(s.movie.Frames)` |
| `GoToBeginningOfMovie` + `StartMovie` | `s.frame = 0; s.tick = 0; s.playing = true` |
| `StopMovie` | `s.playing = false` |
| `MoviesTask(m, 0)` | `s.service()`: if `playing`, `tick++`; when `tick == FrameTicks`, `tick = 0` and advance `frame`; then blit `Frames[frame]` through `Palette` into `dst` |
| `SetMovieGWorld(mainWindow)` | pick the destination image (19.7) |
| `PrerollMovie`, `LoadMovieIntoRam`, `EnterMovies`, `NewHandle(307200)`, the `'LOOP'` user data, `SetMovieMasterTimeBase` | **delete** |

Notes:

- `centerIn` must use Go's `/`, which truncates toward zero and therefore matches
  CodeWarrior on the two negative cases (section 7.1). Do **not** use a floor-division
  helper.
- The blit must clip in **both** directions: the movie may be larger than the clip rect
  (Demo House) or smaller (nothing in this tree, but a user movie could be), and the
  source rows may be shorter than a group boundary (14.4).
- Nothing in this subsystem needs floating point.

---

## 21. `tools/probe_mov.py`

```
probe_mov.py survey                 one line per Houses/*.mov (the headline table)
probe_mov.py atoms  [FILE...]       atom tree with offsets, sizes and raw size fields
probe_mov.py moov   [FILE]          full field dump of every moov atom
probe_mov.py samples FILE|all       per-sample table + the EXACT/difference check
probe_mov.py rle    FILE [N]        opcode-level trace of one sample, with hexdump
probe_mov.py opcodes [FILE...]      opcode-usage totals per movie (14.6, 16.3)
probe_mov.py decode FILE OUTDIR     frame_000.png ... for every frame
probe_mov.py sheet  FILE OUT.png    all frames as one contact sheet
probe_mov.py extract OUTDIR         raw 8-bit index frames + manifest.tsv (19.3)
probe_mov.py all                    survey + samples for all 15 files
```

`FILE` accepts a path, a bare house name (`Titanic`), a file name (`Titanic.mov`), or the
literal word `all`.

It depends on `tools/probe_rez.py` (to find `clut` 128 in `GliderPRO/Glider PRO.r`) and
`tools/probe_pict.py` (for `parse_clut` and the PNG writers). It reads
`GliderPRO/Houses/*.mov` and nothing else; it never writes inside `GliderPRO/`.

The three decoders are `decode_rle` (Apple Animation, sections 14.1-14.5),
`decode_raw8` (section 15) and `decode_smc` + `SmcState` (section 16). Each takes an
optional `stats` dict so the tables in this document are generated by the same code path
that renders the frames — there is no second implementation to drift.

Reproducing every number in this document:

```
python3 tools/probe_mov.py survey                       # section 10
python3 tools/probe_mov.py atoms "Demo House"           # section 12.1
python3 tools/probe_mov.py moov  "Demo House"           # sections 12.2-12.15
python3 tools/probe_mov.py samples all | grep EXACT     # section 13.2 (15 lines)
python3 tools/probe_mov.py samples "Rainbow's End"      # section 14.8
python3 tools/probe_mov.py opcodes                      # sections 14.6, 14.7, 16.3
python3 tools/probe_mov.py rle "Titanic" 0              # section 14.4
python3 tools/probe_mov.py sheet "Teddy World" /tmp/tw.png
python3 tools/probe_mov.py extract /tmp/movframes        # section 19.3
```

---

## Open questions

1. **What was the frame rate of the 14 headerless movies?** Their `mvhd`/`mdhd`
   `timeScale` and `stts` `sampleDuration` were in the lost `moov`. Only
   `Demo House.mov` is known (121/600 = 4.9587 fps, `stts` at file offset 119373).
   Section 19.5 recommends 5 fps uniformly, which is a considered guess, not evidence.
   Nothing else in this tree constrains it.

2. **What was Grand Prix's colour table?** Its `smc ` stream uses indices 0..58 minus
   {1, 48} plus 255 — a custom ~59-entry `ColorTable` that lived in the
   `ImageDescription` (section 17.3). It is not recoverable from this tree; no 59-entry
   `clut` exists in `Glider PRO.r` or in any house resource fork. Only an original copy
   of `Grand Prix.mov` with its resource fork, or a screenshot of the running game,
   could settle it.

3. **What are the numeric values of `newMovieActive`, `loopTimeBase`, `flushFromRam`,
   `fsCurPerm` and `gestaltQuickTime`?** They come from `<Movies.h>`, `<Files.h>` and
   `<Gestalt.h>`, included at `GliderPRO/Headers/GliderVars.h:8` and not present in this
   tree. Irrelevant to a port (all five calls disappear), but they cannot be quoted.

4. **Was `Demo House.mov`'s 82 x 62 size ever intended to fit?** `tkhd` says 82 x 62,
   the TV screen is 64 x 49, and the result is that a caption gets cropped
   (section 7.1). Either the TV art changed after the movie was made, or the author
   never looked. The 1995-07-18 movie timestamps predate the shipped 1.0.4 by a year,
   which is consistent with the first explanation but does not prove it.

5. **Is `dy = (49 - 62) / 2` -6 or -7 in the shipping binary?** `CenterRectInRect`
   (`GliderPRO/Sources/RectUtils.c:149`) divides a negative `int` by 2, which C89 leaves
   implementation-defined. CodeWarrior and MPW C both truncate, giving -6, and Go
   matches. But the shipped 1.0.4 binary was not disassembled to confirm, and the
   difference is one row of the Demo House movie. Same question for Teddy World's
   `(49 - 50) / 2`, where it decides whether row 0 or row 49 is lost.

6. **Why does `PrerollMovie` get rate `0x000F0000` (15.0)?**
   (`GliderPRO/Sources/HouseIO.c:124`.) A rate of 15.0 means "I intend to play this at 15x
   speed", which is not what any movie here does. Almost certainly "15 fps" written into
   the wrong argument. Harmless, and no evidence either way survives.

7. **Did any shipped house's `.mov` originally have a sound track?** For Demo House the
   answer is a firm no (`mvhd.nextTrackID` = 2, one `trak`, `tkhd.volume` = 0, one
   `vide` media handler, no `soun` anywhere). For the other 14 the `mdat` payload is
   100% accounted for by video samples (section 13.2), which rules out interleaved audio
   — but an audio-only-in-a-second-`mdat` layout would have been lost with the header,
   and cannot be excluded on principle. The exact tiling makes it very unlikely.

8. **What did `Grand Prix.mov`'s 0x80/0xA0 chunk flag bytes mean?** `rle ` uses 0x40 for
   "key frame"; `smc `'s high byte is 0xA0 on 20 of 21 samples and 0x80 on sample 6
   (section 16.4). They are demonstrably *not* key-frame markers: no frame after the
   first is independently decodable, because roughly half of every later frame is
   coloured out of cache slots loaded by earlier frames (section 16.3). No interpretation
   is derivable from 21 samples.

9. **Would the original have shown the yellow alert for a headerless movie?** Yes, by
   inspection of `HouseIO.c:94-100` — but it never happened in 1996 because the resource
   forks were intact. Whether a port should reproduce it is a judgement call
   (section 19.2 says no).

## Porting notes

Ordered roughly by how likely each is to be got wrong.

1. **Do not expect the shipped `.mov` files to be readable.** 14 of 15 have no `moov`.
   Any port that pipes them to a QuickTime/ffmpeg-style decoder gets exactly one working
   movie and 14 hard failures. Plan for the transcode step (19.3) from the start; do not
   discover this at integration time.

2. **All on-disk QuickTime data is big-endian.** Use `binary.BigEndian` for every field.
   The `.mov` files are the *only* big-endian assets a port reads that are not
   Resource-Manager resources, so the habit is easy to lose.

3. **Atom `size == 0` means "to end of file", not "empty".** All 14 broken files hinge
   on this. `size == 1` means a 64-bit size at +8; reject sizes 2..7.

4. **`moov` may follow `mdat`.** It does in `Demo House.mov` (offset 118792 of 119789).
   Do not assume header-first.

5. **`CenterRectInRect` truncates toward zero on negatives.** `(64-82)/2 = -9`,
   `(49-62)/2 = -6`, `(49-50)/2 = 0`. Go's `/` matches C's. Do not "helpfully" use
   floor division; it shifts Demo House by a row and moves which row Teddy World loses.

6. **Never scale the movie.** `CenterRectInRect` preserves width and height
   (`RectUtils.c:146-153`) and `SetMovieBox` is handed the result verbatim. Oversized
   movies are cropped by `SetMovieDisplayClipRgn`, not fitted.

7. **Clip in both dimensions, on both sides.** Demo House is cropped left, right, top
   and bottom simultaneously (section 7.1). And inside the `rle ` decoder, clamp pixel
   writes to the row width: a literal run of 11 groups at 4 bpp writes 88 pixels into an
   82-pixel row (section 14.4).

8. **`raw ` samples have no row padding.** `w * h` bytes, packed. This differs from
   every QuickDraw `PixMap` in the rest of the game, whose `rowBytes` is padded. Reusing
   the PICT loader's stride logic shears the image.

9. **Handle `rle `'s mid-line skip (opcode 0) and the 7-byte no-op sample.** Six of the
   ten `rle ` movies use opcode 0 (section 14.6); Rainbow's End and Teddy World contain
   samples that are 4+2+1 = 7 bytes with header `0x0000`, meaning "hold the previous
   frame". A decoder that treats a no-op as a blank frame makes those movies flicker.

10. **Consume the trailing `0x00` after the last line.** Every `rle ` sample in the tree
    has one (section 14.5). It only matters for a reconstruction-based walker, and there
    it matters absolutely.

11. **`rle ` "pixel groups" are 4 *bytes*, not 4 pixels.** At depth 8 that is 4 pixels;
    at depth 36 (4-bit gray) it is 8. Getting this backwards halves or doubles every row
    width. Demo House is the only depth-36 file, so a port that only handles depth 8
    will silently corrupt the *one* movie that actually works out of the box.

12. **`depth` 36 is 4-bit **grayscale**, and `clutID` 36 means the standard gray ramp**
    `gray(i) = (15-i)*17`, i.e. index 0 is **white**. Reading depth 36 as "36 bits" or
    treating clutID 36 as an index into `clut` 128 both produce garbage.

13. **`clutID` 0 means "a ColorTable is appended to the ImageDescription"; -1 means "use
    the destination's".** Do not confuse the two. `Demo House.mov`'s `stsd` is exactly
    102 bytes with nothing after the 86-byte ImageDescription, so there is no inline
    table to look for there.

14. **`vmhd.graphicsMode` is 64 = `ditherCopy`** for Demo House. A 4-bit-gray movie
    dithered into `clut` 128 looks visibly different from a nearest-colour map. Pick one
    deliberately and write down which.

15. **The movie box lives in window coordinates and depends on screen resolution.**
    `OffsetRectRoomRelative` adds `playOriginH`/`playOriginV`, computed from the actual
    screen size (`InterfaceInit.c:203-204`). A port with a fixed logical framebuffer
    must recompute rather than transcribe.

16. **Only the first `kTV` in the *central* room gets the movie.** Not the first in the
    house, not every TV, not TVs in the 8 neighbouring rooms that `DrawLocale` also
    draws. 65 of the 80 shipped `kTV` placements never show a movie.
    (`ObjectDrawAll.c:680-681`, `:704-705`.)

17. **The two empty `if` bodies in `HandleTV` are load-bearing.**
    `Dynamics.c:439-440` and `:451-452` (the empty bodies) suppress the static-snow blit and the dirty rect
    for the movie TV only, and the **off** branch (`:463-476`) deliberately has no such
    guard so the last movie frame gets painted over. Deleting the empty blocks as "dead
    code" makes the movie flicker under snow.

18. **Every start rewinds.** `Trip.c:47`, `Transit.c:306`/`:344`/`:383`/`:422`,
    `HouseIO.c:112`. The movie never resumes mid-stream. Combined with frame 0 always
    being a key frame, this means a port never needs mid-stream seek.

19. **`tvOn` is not reset between houses** and is only meaningful while
    `tvInRoom && hasMovie`. Prefer deleting it and reading
    `dinahs[tvWithMovieNumber].active` (section 19.4).

20. **Suppress the yellow alert for a missing/broken movie.** `HouseIO.c:83-85` is silent
    for "no file"; `:98` alerts for "bad file". With 14 broken files the faithful path is
    a modal alert on almost every house open. Take the silent path (section 19.2).

21. **Delete, do not port:** `EnterMovies`, `Gestalt`, `PrerollMovie`,
    `LoadMovieIntoRam`, the `NewHandle(307200)` memory probe, `SetMovieMasterTimeBase`,
    and the `'LOOP'` user-data dance (which no shipped file even contains — section
    12.15). They are all Toolbox bookkeeping with no observable effect in a Go port.

22. **Fix the leaks if you port the load path literally.** `LoopMovie` leaks its 4-byte
    `theLoop` handle (`HouseIO.c:50`, never disposed); `OpenHouseMovie` leaks `theMovie`
    when `LoadMovieIntoRam` fails, because `CloseHouseMovie`'s `DisposeMovie` is behind a
    `hasMovie` check that is still false at that point (`HouseIO.c:118-119` vs `:151`).

23. **The first-TV election is per room draw, and `redraw` must not split it.**
    `DrawLocale` clears `tvInRoom`/`tvWithMovieNumber` at `RoomGraphics.c:59-60` and then
    re-elects during its ninth and last `DrawARoomsObjects` call (`:120`, the central
    room). In the original the placement half of the election sits outside
    `if (!redraw)` (`ObjectDrawAll.c:680-694`) and the registration half sits inside it
    (`:704-709`); that is only harmless because the single `redraw == true` call site,
    `RedrawRoomLighting` (`RoomGraphics.c:451`), is dead code with no callers anywhere in
    the tree. Keep placement and registration in one guarded block in the port
    (section 7, observation (a)).

24. **`smc `'s three colour caches belong to the track, not the sample.** Construct the
    decoder state once per movie and never reset it between frames. Grand Prix depends on
    this: 1214 of its 1303 cached-table opcodes read a slot written by an *earlier* frame,
    so a per-sample reset miscolours about half of every frame after the first, and
    decoding cannot start anywhere but frame 0 (section 16.3). This costs nothing at run
    time because every start rewinds (note 18), but it does mean the transcode step
    (19.3) must decode Grand Prix strictly in order.


