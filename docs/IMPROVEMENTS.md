# Improvements — what a public release needs that a faithful port does not

`docs/PLAN.md` is the fidelity plan: it tracks what has to be true for gliderGo to
*behave* like Glider PRO. This file tracks what has to be true for gliderGo to be a game
that strangers can download and play. The two lists are different, and nothing in the
stage plan would have surfaced most of what is below.

Rules for this file:

- Stage 1's contract is unchanged: **the game behaves exactly like the 1994 original**.
  So anything here that would change behaviour is either gated behind a setting that
  defaults to the original, or waits for Stage 2 or later. Fidelity is not traded away
  for polish.
- Every item names the stage it belongs to and is either **done**, **planned** (with the
  stage), or **decision needed** (the user's call, not mine).
- An item is only removed from this file when it is done, and then it moves to the
  Done section with the commit that did it.
- **The numbers are load-bearing.** Comments in the Go source cite items by number
  (`docs/IMPROVEMENTS.md 2.17`), because the argument for transcribing a bug faithfully
  is only complete if it says where the fix is written down. Renumber an item and those
  citations point at the wrong thing, so items are appended, never reordered.

---

## 1. Legal — the blocker that has nothing to do with code

### 1.1 gliderGo has no licence file — **DONE, 1.5a**

gliderGo is transcribed from GPLv2 source, function by function, with comments citing
`GliderPRO/Sources/*.c` line numbers. That makes it unambiguously a derivative work, and
GPLv2 §2(b) requires the whole to be licensed under GPLv2. This is not a preference; it
is the only compliant option, and shipping a public binary with no licence file at all is
strictly worse than shipping one. Added `LICENSE` (GPLv2) and a `## Licence` section in
`README.md` naming John Calhoun, Casady & Greene and the upstream repository.

Note for anyone tempted to relicense later: upstream's grant is GPLv2 **only**. There is
no "or later" clause, so gliderGo cannot be moved to GPLv3 and cannot take GPLv3-only
dependencies.

### 1.2 The art, the sounds and the 22 houses are on a different footing from the code — **decision needed, before any public release**

`GliderPRO/README.md` says, exactly: *"The **source** for Glider PRO is released under the
GNU General Public License 2."* It says source. It does not say assets. And the same file
credits the houses to five people who are not John Calhoun:

| House | Author |
|---|---|
| Demo House, CD Demo House | John Calhoun and **Kim Money** |
| Davis Station, Metropolis, Titanic | **Jonathan Chin** and John Calhoun |
| Grand Prix, Leviathan, ImagineHouse PRO II, In The Mirror | **Jonathan Chin** |
| Land of Illusion, Nemo's Market, Rainbow's End, SpacePods | **Ward Hartenstein** |
| Slumberland | John Calhoun, **Jonathan Chin**, **Steve Sullivan**, **Ward Hartenstein** |
| Teddy World | **Shawn Brenneman** |
| The Asylum Pro | **Steve Sullivan** |

Two PICT resources are also derived from other people's illustrations: 3975 from a John R.
Neill plate for *Ozma of Oz*, and 153 (the About box) from a Winsor McCay *Little Nemo*
strip. Both are old enough to be very probably public domain, but "very probably" is not a
licence audit.

The upstream repository does distribute the houses, so re-vendoring them under
`GliderPRO/` is no worse than what the copyright holder already does. Shipping a
**gliderGo release binary with the extracted art baked in** is a different act: a new
distribution of that art, in a new form, by someone who is not the rights holder.

**The architecture is already right, by accident, and should be made deliberate.**
`.gitignore` excludes `/assets/extracted/`, so no original art is in this repository and
none can end up in a release tarball built from it. The release model that follows is:

> gliderGo ships **code only**. On first run it asks for a copy of Glider PRO and
> extracts what it needs locally, the way a ScummVM or a DOSBox front-end does.

That has a consequence worth stating plainly, because it changes the priority of a later
stage: **Stage 2's new houses are the only content gliderGo can legally ship itself.**
They stop being a nice extra and become the default content of the public build, with the
original 22 houses available to anyone who owns the game. Stage 2 should be planned as
original work on that basis — not as imitations of the shipped houses, which would inherit
the same authorship question.

Needed from the user: a decision on whether to (a) ship code-only and extract at first
run, which is what I would do, or (b) contact John Calhoun for an explicit asset grant,
which is the only route to a single-download release of the original content.

### 1.3 No `CHANGELOG.md`, no `public/index.html` — **planned, end of Stage 1**

Both are required by the parent-repository checklist in the root `CLAUDE.md`, along with eight
other integration points (`release.sh`, its CI config, the root `README.md` table,
`public-root/index.html`, `wiki/home.md`). gliderGo currently appears in **none** of them.
Deferred deliberately to the end of Stage 1 rather than done now, because wiring a
half-finished port into the Pages landing page advertises something not yet playable.

---

## 2. Engineering — what a faithful transcription leaves for a release

Items 2.1–2.3 are about the machine the game runs on. From 2.4 they are about the port and
the original: places where transcribing the 1994 code exactly is right for Stage 1 and
wrong for a shipped build. Each of those says what the original does, why it is kept, and
what the fix is, so that the fix is a decision someone can make later rather than a
rediscovery.

### 2.1 Window scaling — **partly done; fullscreen and a runtime toggle planned, 1.7**

Already there: `cmd/glidergo -scale N` does integer nearest-neighbour magnification,
which is the right filter — the art is 8-bit indexed pixel art and bilinear would smear
it. What is missing for a release is fullscreen with letterboxing, a runtime toggle rather
than a launch flag, and remembering the choice. See 2.8 for where the transform has to go.

**A fidelity trap to avoid when that work happens, which the port has so far avoided by
luck rather than intent.** `Main.c:191` forces `numNeighbors` to 1 on a 512px-wide screen
and `Settings.c:888` defaults it to 9. 1.5a measured what that costs: at 1/3/9 neighbours
the CD Demo House resolves 46/141/390 cross-room object links, so **a switch wired to the
room next door stops working on a small screen**. In the port `NumNeighbors` lives on
`render.Scene` and defaults to 9 with nothing reading the window size, so the two are
already decoupled — the risk is a future "derive it from the window like the original did"
change reintroducing the coupling. It should become an explicit setting, documented as
affecting gameplay and not just how much you can see.

### 2.2 The simulation is frame-locked at 30.07 fps and stays that way — **policy; interpolation is 1.7 or later**

`kTicksPerFrame = 2` Mac ticks on a 60.15 Hz clock, i.e. 30.07 frames a second, and every
physics constant in `player/consts.go` was tuned against it. That is not negotiable and
the port keeps it: `TicksPerFrame` is a constant, `Frame` is the only clock the game logic
reads, and no future display refresh rate may change either. Frame *interpolation* for
60/120/144 Hz displays is a separate question — it belongs at the presentation step (2.10)
and must not touch the simulation. The limiter's own cost is 2.17.

### 2.3 Player 2's keys are modifier keys — **worked around in `cmd/glidergo`; a remapping UI is 1.7**

The original hard-codes player 2 to Control, Command, Option and Shift
(`InterfaceInit.c:148-151`), which a modern window manager or desktop environment may
swallow before the game sees them, and which many keyboards cannot report independently.
1.4 anticipated this: the four key indices are per-glider *data* on `player.Glider`, not
constants baked into `GetInput`, specifically so they can be remapped.

`cmd/glidergo` currently binds player 2 to A/D/S/W, which is a documented deviation and is
in the package comment there. 1.7 owes a remapping UI for all eight bindings and a
persisted config. Related: a release needs gamepad support, which the original had no
concept of.

### 2.4 Transitions run at memory speed, so a wipe is a blink — **planned, 1.7**

`WipeScreenOn` (`internal/game/screen.go`) moves a 4-pixel bar across the screen and
presents after each step: 116 steps for a vertical wipe, 160 for a horizontal one. On a
1994 Mac each of those was a real `CopyBits` over the bus and the count *was* the
duration. Here they cost nothing measurable — the headless build runs the whole 116-step
wipe inside one game frame — so a transition that should read as a wipe reads as a single
dropped frame.

The fix is to pace the strips against a fixed wall-clock duration (a third of a second is
about what the original felt like) rather than against the strip count. It is a deliberate
divergence, so it is not made in `screen.go`; it belongs with the rest of the presentation
work in 1.7, and it needs the frame limiter's clock, not a `time.Sleep` per strip.

Two smaller things to fix in the same pass, both reproduced faithfully today:

- The horizontal strip count reads `workSrcRect.right / 4` rather than `theRect`, so a
  caller passing a narrower rect still gets 160 strips. It should clamp to `theRect`.
- Only `top` and `bottom` are clamped inside the loop, so a horizontal wipe's bar walks
  off the far edge for its last few strips and those copies are no-ops absorbed by
  `CopyBits`' clipping.

This item was also the reason the first version of `playTestWorld` in
`internal/game/play_test.go` measured the wrong thing: it budgeted *presents*, and a
transition presents 116 times inside one frame, so "100 frames" bought 61. The budget is
frames now.

`cmd/glidergo`'s `-frames` flag had the identical bug and it is fixed the same way. It
counted its own `Present` calls, so `-frames 300` ran 298 frames in a static room — two
presents happen outside the loop — and 137 in a run where the glider took a door, because
one 160-strip wipe spent more than half the budget. A flag whose meaning depends on where
the glider drifted is not a flag anyone can benchmark or replay with; `wrapPresentLimit`
now tests `World.Frame`. The `-dump` PNG count is deliberately still per present, because
one file per wipe strip is what you want when looking at a transition.

### 2.5 Pausing blocks the process, and unpausing leaves the screen black — **planned, 1.7**

`DoPause` is called from inside `GetInput` and *blocks* until the player unpauses, which is
why a paused game does not advance a frame. Faithful, and fine on a cooperatively
multitasked Mac; on a modern OS a blocking modal inside the input path is how a game stops
responding to the window manager.

`RestoreEntireGameScreen` is the other half. It paints the window black, composes into the
work map, and **never copies the work map to the screen**. What makes the room reappear in
the original is the update event the vanishing dialogue generates, which
`HandlePlayEvent` turns into a whole-`justRoomsRect` blit. So the redraw arrives via the
event pump, one frame late — and with `doBackground` false (the shipped default, see 2.21)
it never arrives at all: the screen stays black except where the dirty rects happen to
fall. That is a visible bug in the original in its default configuration.

A release needs a pause overlay that says what to press, a pause that yields to the OS
instead of blocking, and a `RestoreEntireGameScreen` that publishes what it composed.

### 2.6 A missing or wrong asset directory must not be a crash — **planned, 1.7**

Today the tests `t.Skipf` when `assets/extracted` is absent, which is right for tests and
is not a shipped behaviour. A public binary run by someone who has not pointed it at a
copy of Glider PRO must say so, in a window, with instructions — not panic on a nil
surface or exit silently. This is the user-facing half of item 1.2 and the two should be
built together. `render.Assets` already collects a sticky error rather than dying at the
blit (`assets.Err()`), which is the mechanism this needs; what is missing is a window to
show it in.

### 2.7 No way to quit that a stranger would find — **planned, 1.7**

`cmd/glidergo` maps Escape to quit and that is undiscoverable. The original's answer was a
menu bar, which a port does not have. A release needs a title screen with a Quit item
(3.4), an in-game pause overlay with one (2.5), and a confirmation before abandoning a
game in progress.

### 2.8 The scale transform belongs at the present step and nowhere else — **planned, 1.7**

The game's surfaces are always 640×480 and every rect in `internal/game` and
`internal/render` is in that space: `View.Screen`, `LocalRoomsDest`, the dirty-rect lists,
`playOriginH/V`. A resizable window must therefore insert its transform in exactly one
place — `platform.Window.Present`, or the backend behind it — and never in a coordinate
handed to the game. Written down because the tempting shortcut is to scale `View`, and
that would silently move every collision rect in the game.

### 2.9 The scoreboard is invisible in the original's shipped configuration — **DONE as a deviation, 1.5b**

Four facts compose into a bug the original shipped with:

1. `StructuresInit.c:70` initialises `boardDestRect` to rows −20..0 — above the screen.
2. `Main.c:154` defaults `numNeighbors` to 9, and `:191-192` only forces 1 on a screen
   512px or narrower.
3. `Scoreboard.c:416-421` guards the whole body of `AdjustScoreboardHeight` with
   `wasScoreboardMode != newMode`, and `wasScoreboardMode` starts at `kScoreboardHigh`.
4. `numNeighbors == 9` computes `newMode = kScoreboardHigh`, which equals
   `wasScoreboardMode`, so the guard never opens and the seven `QOffsetRect` calls that
   would move `boardDestRect` on screen never run.

So on any Mac with a screen wider than 512 pixels — which is every Mac the game shipped
for — the scoreboard is blitted to rows −20..0 and clipped away entirely. Score, lives,
battery and bands are simply not displayed.

The port deviates, deliberately and by default: `World.BoardDestRect` is initialised to
rows 460..480 (the band `render.NewView` leaves between `Screen` and `House`, and exactly
the strip `NewGame` paints black at `Play.c:141`), and `AdjustScoreboardHeight` *assigns*
its rects instead of accumulating offsets, which makes it idempotent and turns the latch
into an optimisation rather than a correctness requirement. The High arm's rows are the
port's; the Low arm's are the original's. See the comment on `BoardDestRect` in
`internal/game/world.go` and the head of `internal/game/scoreboard.go`.

This is the one place Stage 1 knowingly shows something the original did not, on the
grounds that a scoreboard nobody can see is not a design decision.

### 2.10 Presentation is welded to the simulation tick — **planned, 1.7 at the earliest**

`RenderFrame` composes, publishes and waits, in that order, once per simulated frame. A
release on a 144 Hz display wants the composition at 30 Hz and the presentation faster,
which means splitting `awaitFrame` from `Present` and interpolating sprite positions at
the present step only. Every position the game computes stays integral and stays at 30 Hz
(2.2); the interpolation is a display artefact and must not feed back.

### 2.11 The dirty-rect lists silently drop past 47 — **counters DONE, 1.5b; the surfacing is 1.7**

All three adders guard on `numWork2Main < (kMaxGarbageRects - 1)` and, when the guard
fails, drop the rect with no report. A dropped *work* rect is a patch of screen that is
never updated; a dropped *back* rect is a patch of work map that is never erased, so
whatever was drawn there smears. Both are reachable in a busy room — a mirror room is the
easy case, because 2.19 registers an unclipped rect per frame — and both are reproduced.

The improvement is not to raise the cap, which would change what the original showed. It
is a debug counter behind a flag, so that a house author or a bug report can say "this
room overflows the rect list" instead of "this room flickers".
`TestPlayGameKeepsPublishingFrames` already asserts the lists stay well under the cap in a
quiet room, which is the regression half of the same idea.

**Done at 1.5b: the counting.** `World.Diag` (`internal/game/guards.go`) holds
`DroppedWorkRects` and `DroppedBackRects`, incremented at all three drop sites in
`render_frame.go`, and both are in every line of a replay trace (`glidertool replay
-trace`) as well as its footer. So the overflow is now a number a bug report carries
rather than a symptom somebody has to name. It is *not* behind a flag: the counters cost
an increment on a path that already dropped the rect, and a diagnostic nobody enables is a
diagnostic nobody has.

**Left for 1.7: telling the player, or rather telling us.** Nothing surfaces the counters
in a running game — there is no debug overlay and no log line — so a stranger who hits it
still just sees flicker. That half needs 1.7's shell to have somewhere to put it.

### 2.12 A missing graphic kills the application — **planned, 1.7**

The original's answer to a failed PICT load is `RedAlert(kErrFailedGraphicLoad)`, which
quits. That is defensible for a game shipped on a CD with its own resource fork and
indefensible for a binary that extracts its assets from someone else's install (1.2). A
release should degrade: draw a placeholder, note it, and carry on. The arcade scoreboard
block is the concrete case — see `arcadeBlackenBoard` in `internal/game/scoreboard.go`,
which reloads the scoreboard PICT *per game over* and dies if it is missing. It should be
loaded once at launch.

### 2.13 One failed sound bank mutes the entire game — **planned, 1.6**

`LoadSounds` treats any failure as fatal to sound as a whole, so a single missing 'snd '
resource costs every effect in the game. A release should load per-sound and lose only
what is actually missing. While there: the original has one volume, set from the Sound
control panel. A release needs at least effects and music separately, and a mute that
survives a restart.

### 2.14 A house whose custom trigger sound fails to load is silently silent — **planned, 1.6**

A house can carry its own 'snd ' resources for triggers. If one fails to load the original
plays nothing and says nothing, so a house author gets no signal that their sound is
broken. The house linter (4.1) should report it, and the game should log it once.

### 2.15 Leaving a game requires a physical key release — **planned, 1.7**

`WaitCommandQReleased` spins until the player physically lets go of Command-Q before the
game will proceed. On a Mac with a real menu bar that prevented a held chord from firing
twice; in a port it is a hang waiting for a key event that may never come (the window can
lose focus mid-chord). It is deliberately **not** transcribed in `NewGame`.

### 2.16 The two-player idle freeze has no visual tell — **planned, 1.9**

`TagGliderIdle` freezes a glider for 30 frames — a full second — with no indication that
this is deliberate. Player 2 in particular starts every two-player game idled and hidden
(`Play.c:198-203`), so the first thing a new player experiences is a second of not
existing. A fade, a shimmer, or anything at all would do.

### 2.17 The frame limiter is a busy-wait, and there is no catch-up — **DONE (the hook), 1.5b; the setting is 1.7**

The original's limiter is `while (TickCount() < nextFrame) { }`, which on a modern machine
pins a core at 100% for whatever fraction of the two ticks the frame did not need, for the
whole time the game is running. `awaitFrame` takes a `World.WaitTick` hook for the loop
body: nil is the C's empty body and is what a fidelity build uses, and `cmd/glidergo`
installs a `time.Sleep(time.Millisecond)`. That changes only *how* the wait is spent, not
when it ends — `awaitFrame` still returns on the same tick — so it is not a fidelity
change and needs no flag.

The second half is still open. `nextFrame` is reseeded from the clock *after* the wait
rather than by adding `kTicksPerFrame` to the previous deadline, so **there is no
catch-up**: a frame that overruns makes the game run slower rather than skip, and every
frame still advances `gameFrame` by exactly one. That is why a slow Mac played the same
game in more wall-clock seconds instead of a different game, and it is the right default.
A release should make it an explicit setting — "keep real time" versus "keep game time" —
because a modern player on a machine that stutters expects the former. Any variable-
timestep work changes that one line and nothing else.

### 2.18 The random stream is unverified against real hardware — **planned, 1.8**

`internal/game/rand.go` transcribes the original's linear congruential generator, and the
demo replay in 1.8 depends on it bit for bit. It has not been checked against a trace from
a real Mac, and until it is, a demo that desyncs is ambiguous between "the RNG is wrong"
and "the input replay is wrong". 1.8 should capture a reference trace first.

Two related notes: `PourScreenOn` is dead code in the original — nothing calls it — and it
*draws from the RNG*, so wiring it up would shift every subsequent random number and
invalidate any recorded demo. And `case kLgTrigger:` in the interaction dispatch is
unreachable, because the object code is never produced; it is transcribed as a dead branch
and should stay dead.

### 2.19 The mirror-room flame blink — **planned, 1.7 (opt-in), 2.x (default)**

`DrawReflection` registers the *unclipped* reflected-glider rect with
`AddRectToBackRects`, so the erase covers the whole rect while the draw covered only the
part inside a mirror. Every frame, the work map outside the mirror is restored from a
background that never had a reflection in it — which is correct — but a candle flame or a
pendulum inside that rect is erased a frame early, because those register no back rects of
their own and rely on their own opaque redraw. In a mirror room with a flame, the flame
blinks.

The fix is to clip the back rect to the mirror rects, which also relieves 2.11 in exactly
the rooms that need it most. It is a visible change to what the original drew, so it is
opt-in first.

### 2.20 A mirror shows the wrong player's foil — **planned, 1.7 (opt-in)**

`DrawReflection` tests `if (showFoil)` with no `!twoPlayerGame` guard, unlike
`RenderGlider`'s. In a two-player game with foil showing, `glid2SrcMap` holds
`kGliderFoil2PictID` — the *other* player's foil sheet — so player 1's reflection is drawn
with player 2's artwork. Clearly unintended, and a player watching a mirror in a
two-player game does see it. A fidelity replay would diverge if it were fixed, so it is
kept and offered as an opt-in correction alongside 2.19 and 2.22.

### 2.21 `doBackground` should not be a user option — **worked around in `cmd/glidergo`; the setting is 1.7**

`doBackground` is a preference (`Main.c:186`, default **false**) that gates the entire
event pump: `PlayGame` only calls `HandlePlayEvent` when it is set. With it off the game
never pauses when it loses the foreground, never processes an update event, and — see 2.5
— never repaints after a dialogue. The original could survive that because the Toolbox
redrew a window's contents from its `WindowRecord`; a modern compositor cannot, and a
window that ignores its own close button is not shippable.

`cmd/glidergo` sets it true unconditionally, with the reasoning at the assignment. A
release should drop the preference entirely and always pause on focus loss. The one thing
to preserve is that the pump loop spins *mid-frame*, after both clocks have been bumped,
so every deactivation costs exactly one frame of animation phase — invisible, but it is
what the original did.

### 2.22 The two-player handshake has three bugs — **planned, 1.9 (opt-in fixes)**

All three are in the limbo/transit handshake and all three are transcribed faithfully:

- **Mismatched-exit deadlock.** Two players leaving the same room by different exits can
  each end up waiting for the other, with no key that breaks it.
- **The `takingTheStairs` leak.** The flag is set on a stair transit and is not always
  cleared, so a later transit in the same game can take the stairs path when it should not.
- **The shared `StillOver` edge detector.** One piece of state serves both gliders, so
  player 2 standing on a trigger can suppress player 1's edge.

A compatibility flag that defaults to the original's behaviour, with the fixes available,
is the right shape — the same shape 2.19 and 2.20 want.

### 2.23 Player 2 has no give-up key and no menu access — **planned, 1.9**

`Delete` abandons a glider waiting in limbo and it is player 1's key only; the menus are
player 1's too. So in a two-player game, player 2 cannot rescue a stuck situation, quit,
pause or change a setting. With the deadlock in 2.22 unfixed that is worse than an
asymmetry. 1.9 owes player 2 a give-up key at minimum.

### 2.24 Two `CopyRect` helpers read from the screen — **note for whenever presentation becomes GPU-backed**

`CopyRectMainToWork` and `CopyRectMainToBack` read the screen surface as a source. That is
free today because `World.Main` is an ordinary `render.Surface` in system memory, and it
becomes a pipeline stall the day the screen is a texture. There are only two of them and
their callers are known; anyone moving presentation onto a GPU should redirect them to a
retained copy rather than read back.

### 2.25 `RedrawRoomLighting` needs a redraw flag — **planned, 1.5c**

The lighting redraw runs on a schedule rather than on demand, which means it can miss a
change made in the same frame and repaint one that did not change. The fix is a dirty flag
set by whatever changes the room's lighting; noted at the call site in
`internal/game/readylevel.go`.

### 2.26 `phoneBitSet` means "no telephone" — **planned, Stage 5 (the editor)**

The flag is inverted with respect to its name: set means the room has *no* phone. Nothing
in the game is wrong — the port reads it the same way the original does — but an editor UI
with a checkbox labelled "phone" wired straight to this field would be backwards. Rename
it or invert it at the editor boundary.

### 2.27 Exposures were selected for and then dropped — **DONE, 1.5b**

The x11 backend asked for `ExposureMask` and had no `case C.Expose`, so the one event that
says "your window's pixels are gone, draw them again" was thrown away. It went unnoticed
because a compositing window manager retains window contents and replays them itself, and
because a running game overwrites the whole framebuffer thirty times a second anyway.

Neither of those covers the case that matters, which is a **paused** game: while
`switchedOut` is set, `PlayGame` presents nothing at all, so an obscured window stays
obscured for as long as the player leaves it. The original had the same requirement and the
same answer — `HandlePlayEvent`'s `updateEvt` arm calls `RefreshGameWindow` — and the port
now has the whole chain: `platform.EventExpose`, emitted by the backend for the last event
of an expose series (`count == 0`, so a multi-rect damage is one redraw), handled in
`cmd/glidergo`'s `PlayEvent` by calling `RefreshGameWindow`.

Any other backend owes the same event. A backend on a platform that guarantees retained
contents may legitimately never send one.

### 2.28 A paused game is indistinguishable from a hung game — **planned, 1.7**

Suspend-on-focus-loss is right (2.21) and it is also invisible: the window holds a frozen
frame with no overlay, no dimming and no text. A player who alt-tabs away and back sees
nothing wrong, but a player whose window manager quietly moves focus elsewhere sees a game
that has stopped responding, and there is nothing on screen to tell them that clicking the
window will resume it.

This bit during development in a form worth recording, because it is the shape the bug will
take for a player. `-frames 300` appeared to hang: the window opened, ran 34 frames, and
stopped. It was not a hang — this host's window manager returns focus to the terminal about
a second after a new window appears, and the game did exactly what it was told and paused.
Nothing on screen or in the terminal said so. `cmd/glidergo` now exempts timed runs
(`-frames`) from suspending at all, since a measurement or a replay has no user to pause
for, but that is a fix for the tooling and not for the player.

1.7 owes the pause a visible state: a dimmed frame and a "paused — click to resume" line,
shared with `DoPause` (2.5) so both kinds of pause look the same.

### 2.29 The scoreboard had no font, so its three text panels were blank — **DONE, 1.5b; the metrics are still owed at 1.7**

`render.Scoreboard.Panel` used to clear a panel to `kGrayBackgroundColor` and drop the string
it was handed. So the band was on screen (2.9) and the four badges lit and blinked, but the
room name, the glider count and the score were three flat gray patches.

The reason was not laziness about glyph rasterising: the original's scoreboard text is the
Macintosh **application font** at 12 point bold, set by `TextFont(applFont)` three times in
`InitScoreboardMap` (`StructuresInit.c:109-133`). `applFont` resolves to Geneva on classic
Mac OS, which is a *system* resource — it is not in Glider PRO's resource fork, so there is
nothing for `glidertool` to extract and no way to be pixel-faithful here by transcription.
A font has to be authored, and authoring one is a different kind of work from porting one.

`internal/render/font.go` is that font: a 5×7 core in a 6×9 cell, ASCII 0x20..0x7F authored
as a sheet of `.` and `#`, wired into `Panel` at the pens the original uses — black at
(1,10), then white at (0,9), which is the order that makes the shadow fall down and right
instead of up and left. Coverage is deliberately wider than the original's problem: a house
is a user-supplied file whose room names are Mac Roman, so all 128 upper-half characters can
reach the title panel, and every one of them resolves — 27 accented lowercase forms by
composing a mark over a base letter, sixteen more authored outright (`…`, the dashes, the
bullet, `¢£§¶†‡◊π µ¬°`, the Apple logo), and the rest folded to the nearest ASCII the font
has (`©` → `(c)`, `Æ` → `AE`, `≤` → `<=`). `TestEveryMacRomanRuneIsRenderable` fails if a
character ever falls through to the hollow box.

**What is still owed, at 1.7.** It is not Geneva and it is not proportional: every character
advances six pixels, so a line is wider or narrower than the original's by whatever Geneva's
metrics differ, and there is no bold — the original's boldface is a one-pixel smear that at
this size would close every counter in the alphabet. Nothing shipped overflows a panel (the
widest of the corpus's 4,070 room names is 162 pixels of the title panel's 255, pinned by
`TestShippedTextFitsItsPanel`), and an authored house that does overflow will clip rather
than ellipsise. A taller face would also let the accented *capitals* keep their accents;
today they fold to the bare letter, because a seven-row capital reaches the top of the cell
and leaves nowhere to put the mark.

Two things became testable the moment it landed, and both now have tests in
`internal/game/scoreboard_test.go`: the glider-count clamp (`refreshNumGliders` clamps
`Mortals` at 0, `QuickGlidersRefresh` deliberately does not, and before the font both drew
the same nothing) and the score roll's intermediate numbers.

### 2.30 `JustRoomsRect` lives on the `View` and is written by game code — **note; revisit if a process ever runs two games**

`game.placeScoreboard` assigns `w.R.V.JustRoomsRect`, reaching through the Scene into the
View to do it. That is the original's structure — `justRoomsRect` is a global that
`AdjustScoreboardHeight` writes — and the port's plan document wanted the field on `World`
instead, where the rest of the per-game state lives.

It is left where it is because the alternative is worse today: `render.RenderFrame` and the
update-event blit both read it, so moving it to `World` means either passing it down through
the renderer or having two copies that can disagree. And there is exactly one game per
process, so the mutation happens once, before the first frame.

What makes it a note rather than a shrug: it is the one field on `View` that is not a pure
function of the screen size, so a second view — a resizable window (2.1), a split-screen
two-player mode, or a test that builds two worlds on one view — would find it stale. The
fix at that point is a `World.JustRoomsRect` plus a parameter on the two render entry
points, and it is cheap; it is only cheap *now* because nothing else writes it.

### 2.31 `DrawCalendar` drew no month — **DONE, 1.5b**

`ObjectDraw2.c:1132-1166` draws the calendar's month name in the same application font:
`GetTime` for the real month, `GetIndString(monthStr, kMonthStringID, month)` for its name
and `ColorText` at `left + (64 - StringWidth(monthStr))/2` to centre it. The port drew the
calendar's picture and stopped there, so a calendar in a room rendered as a blank one.

It was waiting on the font. The other two things it needed were already in place: `Scene.Clock`
exists and is already fixed to 14 October 1994 by the golden-corpus test, so nothing here has
to be nondeterministic, and the twelve month names are `STR# 1005`.

They are transcribed into `objectdraw2.go` rather than loaded, which is the choice
`palette.go` already makes for `'clut'` 128: the data is twelve words that have not changed
since 1994, a room's appearance should not depend on a second asset tree being present at
runtime, and `TestMonthNamesMatchTheResource` decodes the shipped resource and fails if the
copy drifts. When 1.7 needs `STR# 150`'s fifty-one strings the general decoder can be lifted
out of that test; fifty-one strings is where a loader starts paying for itself and twelve is
not.

It was a separate commit from the font because it moved the golden hash of 62 of the corpus's
4,070 rooms, and that diff is worth reading on its own rather than buried in a commit about
glyphs. One finding came out of it and is pinned by
`TestCalendarIsOnePixelNarrowerThanItsCentring`: the C centres the month in **64** pixels and
the calendar picture is **63** wide, so the original's own text sits a pixel right of centre.
The port keeps the 64.

### 2.32 Three things stop the world from inside a frame — **planned, 1.7**

`DisplayStarsRemaining` (`Banner.c:205-243`) is the clearest case. It is called from
`Interactions.c:946`, inside the star-collection arm of `HandleInteraction` — so from the
*middle* of a frame, after the interaction pass and before the glider moves — and it draws
to the main window, then `DelayTicks(60)`, then `WaitForInputEvent(30)`. That is one to one
and a half seconds during which the process does nothing at all: no repaint, no resize, no
window close. `BringUpBanner` (`Banner.c:171-197`) does the same with `WaitForInputEvent(15)`,
and `DoPause` is a third.

Three separate problems, and only the first is cosmetic:

- **The window is dead while it blocks.** A modern compositor notices, and the desktop
  offers to kill the application. This is 2.5 and 2.28 in a different place, which is why
  they should be fixed as one thing: a *pause state* the frame loop knows about, so that
  presentation keeps running while the simulation does not.
- **Input in that window is swallowed.** `WaitForInputEvent` consumes the keystroke that
  dismisses it. So a player holding right when they touch a star is not holding right when
  play resumes — the glider stops. That is 1994 behaviour and it is bad behaviour, and it
  cannot be "fixed" inside Stage 1 without breaking the demo-replay comparison (1.8).
- **It is not reproducible headlessly.** A replay trace cannot represent "and then 90 ticks
  passed", so any script that collects a star would diverge the moment the pause is real.
  The stubs are empty today, which is precisely why the 1.5b traces are stable; that is a
  debt, not a property.

The port's stubs are `internal/game/play.go` (`BringUpBanner`, `DisplayStarsRemaining`) and
`internal/game/env.go` (`DoPause`), and each carries a comment pointing here. When 1.7
implements them, the frame count they consume has to be a simulated count — a mode the loop
runs in for N frames — and not a sleep.

### 2.33 The port declines out-of-range reads the original performs — **DONE as a reported deviation, 1.5b**

The original indexes several arrays with values a house file controls, and does not check
them. The reachable one is the room-object slot: an unlinked transport has `objectLink` of
-1, which the `Byte` parameter turns into 255, and `masterObjects[...]`/`rooms[r].objects[...]`
are then read at that index. On a Mac that read something adjacent and carried on; in Go it
panics, which is not an option for a released game and also not a faithful one.

So `internal/game/guards.go` has exactly one guard, `badIndex`, and every caller reads
`if w.badIndex(kind, i, n) { return <what the C's garbage would most likely have been> }`.
Ten sites: `hotspots.go`, `objects.go`, `setstate.go`, `transit.go` (three), `triggers.go`
(four). Each refusal is counted in `World.Diag.Guarded` and the first sixteen *distinct*
kinds are sampled into `Diag.Seen`, so `glidertool replay` reports both, and a house that
trips one is diagnosable instead of merely surviving.

The two deliberate exclusions matter as much as the guard, because both look like
omissions:

- **`World.Room` is not guarded.** Its nil return is the *designed* answer to
  `GetNeighborRoomNumber` returning -1, which every `ReadyLevel` on an edge room asks for
  several times a frame. Counting it would put thousands of normal events in the same bucket
  as one real fault.
- **`internal/render/locale.go`'s three equivalents are not guarded.** They are the same
  shape, but they sit on `Scene`, which holds no game state and has no business holding a
  counter. If they ever need reporting, the answer is to give `Scene` its own `Diagnostics`,
  not to reach across the package boundary from `render` into `game`.

What is left is a policy question for 1.7: a guarded read means the house is malformed, and
right now nothing tells the *player* that. A house picker that ran `glidertool house
check`'s logic and refused to offer a broken house would be better than diagnosing it after
the fact — see 4.1.

### 2.34 One `PaintRect` in the Carbon conversion lost its destination — **note; binds 1.5c and 1.5e**

Glider PRO's shipped source is a Carbon conversion of the 1994 code, and the conversion
replaced QuickDraw's ambient current-port drawing with explicit `SetGWorld` pairs. Fifteen
`SetPort` calls were commented out in the process. Fourteen are harmless — twelve in
`ObjectDraw2.c` are the *last* statement of a function whose port was already restored by a
`SetGWorld(wasCPort, wasWorld)` two lines up.

The fifteenth is not. `Dynamics.c:577-578`:

```c
//			SetPort((GrafPtr)workSrcMap);
			PaintRect(&dinahs[who].dest);
```

`PaintRect` draws into whatever port is current, and the line that would have made that the
work map is commented out with nothing put in its place. Compare `Grease.c:105-118`, which
was converted correctly and is the model:

```c
GetGWorld(&wasCPort, &wasWorld);
SetGWorld(backSrcMap, nil);
PaintRect(&src);
SetGWorld(workSrcMap, nil);
PaintRect(&src);
AddRectToWorkRects(&src);
SetGWorld(wasCPort, wasWorld);
```

So `HandleOutlet`'s off-frame — the blank rect it paints when the outlet is not zapping —
lands in whichever GWorld the previous drawing call happened to leave set, while the
`AddRectToWorkRects` on the next line asserts it went to the work map. The commented line
names `workSrcMap`, so the 1994 behaviour is not in doubt; only the shipped Carbon build's
is.

**The port has no outlet handler yet** — `HandleDynamics` is an empty stub at
`internal/game/play.go:738`, charged to 1.5c, and `HandleGrease` is one at
`render_frame.go:573`, charged to 1.5e. This entry exists so that whoever writes them
targets the work surface explicitly rather than transcribing a commented-out line as a
comment and leaving a `PaintRect` with no destination. Transcribing the *bug* would be
untestable here for a reason worth stating: the port has no ambient current port to leak
into, so a missing destination is a compile error or an obvious nil — the failure mode the C
has is not available, which makes "be faithful" meaningless and "be correct" the only
reading.

---

## 3. Things the original did not have and a 2026 release is expected to have

### 3.1 High scores — **planned, 1.7 (user-requested)**

The original does keep them (`internal/house.Scores`, and the board sorts on rooms
visited, not points). A release needs them persisted somewhere sane — XDG
`$XDG_DATA_HOME/glidergo/` on Linux, not next to the binary — and it must not corrupt or
crash on a truncated or hand-edited file.

### 3.2 Difficulty is brutal by modern convention — **decision needed, Stage 2 at the earliest**

Glider PRO is a 1994 game: limited lives, no checkpoints, and death sends you a long way
back. Modern players bounce off that. An optional practice mode — infinite lives, or
per-room restarts — would widen the audience a lot. It cannot go in Stage 1, whose whole
contract is exact fidelity, and it must default to off. Worth deciding at Stage 2 when
the new houses are being designed, because the houses can be designed for either.

### 3.3 Accessibility — **planned, 1.7 onward**

Nothing in the original addresses it. The cheap wins: a colour-blind-safe option for the
glider/shadow contrast (which is already low on some backgrounds), a "hold instead of
tap" input option, and not relying on sound alone for any warning. The expensive one is
scaling text, which interacts with 2.1.

### 3.4 There is no way in to the game — **planned, 1.7**

No title screen, no house picker, no options screen, no credits. 1.7 is scoped as "the
shell" and owns all of it. Flagged here because the credits screen is not optional: the
five house authors in item 1.2 and both illustrators must be named in the shipped build
regardless of how the asset question is resolved.

---

## 4. Tooling and content

### 4.1 A house linter — **planned, Stage 2**

`glidertool house check` already round-trips and sanity-checks a house. Stage 2 authors new
houses, and the failure modes it should catch first are the ones the shipped houses
already contain: transit objects whose link points at a room that does not exist, transit
objects that are linked from nowhere, destination rooms with no staircase to arrive on,
and (2.14) custom trigger sounds that do not load. A house that fails to link is not a
crash in the original — the player just cannot get out of the room — which is precisely
why a linter is worth more than a runtime check.

### 4.2 A bug-report format, so that a released game can be debugged — **DONE, 1.5b**

A public release gets reports like "sometimes the glider sticks in the third room", which
nobody can reproduce. `internal/replay` and `glidertool replay` are the answer: a script is
a house, a seed, a start point and a keystroke log, and it reproduces the frame exactly on
any machine with no display, no sound and no timing dependency.

```
glidertool replay -house "CD Demo House" -frames 600 -room 4 -where 420,20
glidertool replay -trace -o bug.trace bug.script
glidertool replay -digest bug.script        # one token, for "do we still agree"
```

Three things had to be pinned to make a run reproducible, and they are the three a bug
report cannot leave to the machine: the random seed, the calendar clock (`DrawCalendar`
reads it — 2.31), and the frame clock (`World.TickCount` nil means the frame number *is* the
time). Everything else falls out of that.

What the trace deliberately does **not** contain is a hash of the screen. A screen digest
would change on every legitimate rendering improvement and tell you nothing about where; the
trace records the frame counters, the dirty-rect counts, the glider's mode and rect, the
score, and `World.RandSeed`, which is the strongest determinism claim available and the only
field in it that is not a fact about the picture. `TestDifferentSeedsAreDifferentRuns` exists
because a harness that pinned the randomness too would pass every determinism test and
reproduce exactly one run of the game.

**A deviation from `docs/PLAN.md` to record.** The plan asks for the seven exit kinds to be
tested "leaving Demo House's first room". That is not possible: Demo House contains four of
the seven kinds in the whole file, and its first room has none of them. CD Demo House — also
one of the 22 originals — has all seven, so `internal/game/exits_test.go` picks rooms by
inspection there instead (28, 192, 193, 4, 32, 69). The seven arrival rects were derived
from the C by hand before the port was run, and each case carries its arithmetic, because a
number that matches the current code is a snapshot and a number that matches the C is a
specification.

Two further notes on that test, both of which cost time to find:

- **`GliderInRect` is containment, not intersection.** A glider merely touching a staircase
  trigger box walks straight past it, so a probe has to place the whole 48×20 glider inside
  the box. A placement that looks right and does nothing is this.
- **The last `Present` of a frame is the one that counts.** A transition presents once per
  wipe strip — 116 or 160 times inside one game frame — and the side doors change room from
  inside `HandleInteraction`, which runs *before* `HandleGlider`. Sampling the first present
  of the transition frame therefore reports the glider offset by `-RoomWide` but not yet
  stepped, and misses the second of the frame's two renders entirely.

---

## Done

| Item | Stage | Commit |
|---|---|---|
| 1.1 GPLv2 `LICENSE` and a `README.md` licence section | 1.5a | earlier |
| 2.9 Scoreboard moved on screen as a documented deviation | 1.5b | this stage |
| 2.11 `World.Diag` counts the silently dropped dirty rects (the surfacing is still 1.7) | 1.5b | this stage |
| 2.17 `World.WaitTick` hook, and a sleeping limiter in `cmd/glidergo` | 1.5b | this stage |
| 2.27 `platform.EventExpose`, emitted by the x11 backend and handled by the host | 1.5b | this stage |
| 2.29 A bitmap font, full Mac Roman coverage, wired into the scoreboard's three panels | 1.5b | this stage |
| 2.31 `DrawCalendar` draws its month, from `STR# 1005` and `Scene.Clock` | 1.5b | this stage |
| 2.33 `badIndex`, one named path for every out-of-range read the C performs and the port refuses | 1.5b | this stage |
| 4.2 `internal/replay` and `glidertool replay`: the bug-report format | 1.5b | this stage |

Two bugs found and fixed in the port itself while writing this, neither of which is an
"improvement" so much as a repair, both recorded here because the reason no test caught
them is worth keeping:

- **`render.bgrxLUT` was built from a zero-valued `Palette`.** Go runs every package-level
  variable initializer before any `init()` and orders them only by dependencies visible in
  the initializer *expressions*; `Palette` has no initializer expression, so a
  `var x = f(Palette)` had no dependency edge, ran first, and read 256 zero entries. Every
  LUT entry came out `0xFF000000`, so `Surface.ToBGRX` returned an opaque black image for
  any input — and nothing but the real host path calls it, while an all-black frame is
  indistinguishable from an uncomposed one. Fixed by deriving the table inside
  `palette.go`'s `init`; guarded by `TestBGRXLUTMatchesPalette`.
- **`playTestWorld` budgeted presents, not frames.** See 2.4. `cmd/glidergo`'s `-frames`
  flag had the same bug and is fixed the same way; the mistake is easy to make twice because
  `Present` is the only per-frame hook a host owns, so it reads like a frame counter.
