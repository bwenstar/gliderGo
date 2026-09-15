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

**Six of the twenty-two houses are credited to nobody, and one of them is not from 1995.**
Found while writing 1.7c's credits screen, by reading all 22 house banners: Art Museum,
California or Bust!, Castle o' the Air, Empty House, Fun House and Sampler appear in no line of
the README. `Castle o' the Air`'s own banner says "(by john calhoun)", which settles that one.
`Sampler` does not: its banner reads "Welcome to Omid's Happy Home." and its saved date is
**2000-05-11**, five years after every other shipped house and after Casady & Greene stopped
selling the game. Whoever Omid is, the source release does not say, and a house whose author is
unknown is the one kind this project cannot re-licence by asking. It is two rooms and eleven
objects, so losing it would cost nothing — but that is a decision for the same conversation as
the rest of this item.

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

### 2.1 Window scaling — **remembered, 1.7b; fullscreen and a runtime toggle still open**

Already there: `cmd/glidergo -scale N` does integer nearest-neighbour magnification,
which is the right filter — the art is 8-bit indexed pixel art and bilinear would smear
it. What is missing for a release is fullscreen with letterboxing, a runtime toggle rather
than a launch flag, and remembering the choice. See 2.8 for where the transform has to go.

**1.7b took care of remembering it.** `prefs.Scale` is on the settings screen, is what
`openWindow` reads, and is the one row on that screen carrying a "next launch" note — because
the window is made once, at launch, and resizing it mid-session is a backend change (2.8) rather
than a preference change. The note is deliberately *not* on the other rows whose effect is
invisible from the settings screen (the bindings, the pause key, rooms in view, the music): all
four are read when a game starts, which is the only thing a player can do next, so a note on
each would say "as you would expect" four times and stop the one row that is genuinely different
from standing out.

Both `-scale` and `prefs.Scale` exist and they do not fight: `overrideFromFlags` folds the flag
into the settings before the window opens. The one place the flag is still read directly is
`-shot`, because a screenshot's magnification is a property of the file being asked for rather
than of the player's window — otherwise `-shot -prefs some.json` could change a golden image's
dimensions out from under the comparison it exists for.

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

### 2.3 Player 2's keys are modifier keys — **DONE, 1.7b (all eight bindings are the player's)**

The original hard-codes player 2 to Control, Command, Option and Shift
(`InterfaceInit.c:148-151`), which a modern window manager or desktop environment may
swallow before the game sees them, and which many keyboards cannot report independently.
1.4 anticipated this: the four key indices are per-glider *data* on `player.Glider`, not
constants baked into `GetInput`, specifically so they can be remapped.

`cmd/glidergo` binds player 2 to A/D/S/W by default, which is a documented deviation and is
in the package comment there.

**1.7b made all eight bindings the player's**, in `internal/prefs` and on the settings
screen (`S` from the title screen), and persisted them in the native configuration
directory. Three things about the shape of that are deliberate:

- **A binding is a key *name*, not a scancode.** `prefs.Controls` holds strings
  (`"left"`, `"a"`), which `platform.ParseKey` resolves; a name that will not parse becomes
  `KeyUnknown`, and `Window.KeyDown` answers false for that on every backend, so a
  hand-edited file with a typo in it gives a dead control rather than a wrong one or a crash.
- **A player who wants the 1994 bindings back can have them**, which is what this item asked
  for: the modifier keys are bindable like any other, and whether the window manager lets
  them through is now the player's problem rather than the port's decision.
- **The bindings are resolved once per game**, not once per poll (`cmd/glidergo/play.go`),
  because the settings screen is only reachable from the title screen. So a rebind can never
  be half applied to a game in progress.

Still owed for a release: gamepad support, which the original had no concept of and which
`internal/platform` has no device layer for.

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

### 2.5 Pausing blocks the process, and unpausing leaves the screen black — **DONE, 1.7b**

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

**1.7b built the first two and left the third alone.** The pause is split at the line between
the game and its host: `internal/game/pause.go` draws the placard, holds `World.Paused` up and
restores the rect from the work map afterwards, and `World.Pause` — a hook `cmd/glidergo`
fills in — does the waiting, pumping host events every pass and repainting after each one. So
the window answers the compositor, its own close button and an expose while the game is
paused, and none of that is in `internal/game`. Four consequences worth recording:

- **The pause key acts on its press edge**, in both halves. The C's three `GetKeys` release
  loops exist because `theKeys` is a global that `DoPause` overwrites — see
  `docs/analysis/input.md` §10.2-10.3 — and a host that reported the key level-triggered
  instead would pause, resume and pause again on a single press.
- **One deliberate ordering change.** The C erases the placard and *then* waits for the key to
  come up, so holding the pause key after unpausing shows a frozen game with no placard on it.
  This port waits and then erases: a held key shows a paused game.
- **The artwork names a key this port does not have.** Both PICTs read "or Cmd-Q to Quit the
  game" and there is no Command key here, so `World.PauseHint` draws a row under the placard
  naming the substitute (Q), and the restore covers the union of the two. 2.7 is why that row
  is load-bearing rather than a nicety.
- **A nil `Pause` hook is no pause at all**, which is what the fidelity corpus needs: a replay
  has no keyboard, so a faithful transcription would hang on the first recorded frame with the
  pause key down.

`RestoreEntireGameScreen` still composes without publishing, faithfully. It is unreachable in
this port today — the modal dialogues it tears down are 1.7c's and 1.10's — and the honest
place to fix it is the commit that first calls it, where the fix can be seen to work.

### 2.6 A missing or wrong asset directory must not be a crash — **DONE, 1.7a**

Today the tests `t.Skipf` when `assets/extracted` is absent, which is right for tests and
is not a shipped behaviour. A public binary run by someone who has not pointed it at a
copy of Glider PRO must say so, in a window, with instructions — not panic on a nil
surface or exit silently. This is the user-facing half of item 1.2 and the two should be
built together. `render.Assets` already collects a sticky error rather than dying at the
blit (`assets.Err()`), which is the mechanism this needs; what is missing is a window to
show it in.

**1.7a built the window.** A fresh clone with no assets now comes up on the port's own
title screen — a drawn wordmark, the credit, "no artwork found", and the command that
produces the artwork — with a working menu over it, and the status band naming the
directory it looked in. Three things make that a *tested* behaviour rather than a hopeful
one:

- `shell.Host.Assets` may be nil, and every plate goes through one accessor
  (`Shell.plate`) that answers nil rather than reaching into it. There is no other path to
  a PICT in the package, so a missing plate cannot become a nil dereference.
- `Discover` failing is printed and not returned (`runShell` in `cmd/glidergo/main.go`):
  the useful place to say "there are no houses here" is the screen, not a terminal the
  player may never see. The three menu items that need a house are unavailable and say why
  when pressed.
- `make headless` renders the no-art and no-houses screens on every `make check`
  (`-shot -art /nonexistent -houses /nonexistent`), because it is the one layout nobody
  developing here ever sees by accident — and the first version of it put "no artwork
  found" half underneath the menu panel.

**1.7b extended the same rule into the game.** The pause placard is PICT 1015 or 1016, and a
checkout with no art now gets a drawn panel in the same 214×54 the picture would have filled —
"PAUSED" and the key that resumes — so nothing else on screen moves depending on whether the
art is there. `internal/game/pause.go` is the only place `internal/game` draws text, and
`TestPausePanelIsDrawnWithoutArt` and `TestPausePanelNamesTheKeyThatResumes` are what keep the
fallback from quietly becoming a blank rect.

### 2.7 No way to quit that a stranger would find — **DONE, 1.7a and 1.7b**

`cmd/glidergo` maps Escape to quit and that is undiscoverable. The original's answer was a
menu bar, which a port does not have. A release needs a title screen with a Quit item
(3.4), an in-game pause overlay with one (2.5), and a confirmation before abandoning a
game in progress.

**The title screen's Quit item exists**, on `Q` and on Escape, and the About box lists both
along with every other binding. Escape *in a game* changed meaning in 1.7a to make room for
it — it ended the game and handed the title screen back, where it used to end the process —
and then changed again in 1.7b, below. Closing the window still ends the process, which is why
`shell.Outcome` carries a `Closed` flag: the two exits look identical to the `World` and must
not to the shell.

**1.7b closed the rest of it, and not the way this item expected.** The plan was a
confirmation dialogue in front of the give-up. What shipped instead is one rule with no new
screen in it: **Escape pauses.**

Escape pauses whether or not it is the player's pause key, either key resumes, and `Q` from
inside the pause is the give-up. So the keystroke a stranger reaches for to get out of a game
now stops the game and shows them a placard with the way out written under it, instead of
throwing the game away — which is the confirmation, one keypress earlier and with nothing extra
drawn. Two things fell out of it:

- **`prefs.PauseKey` keeps its 1994 meaning exactly.** It picks the placard
  (`isEscPauseKey`) and nothing else; the host no longer has to ask which key is spoken for,
  and the old `escPauses` branch in `cmd/glidergo/play.go` is gone.
- **The give-up key is not the quit key.** Q ends the game and hands the title screen back;
  closing the window ends the process. `shell.Outcome.Closed` is still what tells those two
  apart, because they look identical to the `World`.

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

**Measured at 1.5c: three mover types spend rects whether or not anything is happening,
and one shipped room burns a third of the budget standing still.** `RenderBall`,
`RenderDrip` and `RenderFish` are the three renderers with no `Moving` gate — they cannot
have one, for the reasons their comments give (a spent ball sits on the floor and is still
lethal; the drip's hanging cel must survive the first back→work erase; a resting fish's
unmasked water is the animation) — so each costs one work rect and one back rect on *every*
frame for the life of the room. The drip is worse than the other two, because while the drop
hangs its `Whole` is still the union of the last fall, so the rect it registers is the whole
column rather than the drop. Counted over the shipped corpus, `SpacePods.house` room 55 "Ion
Generator" holds 16 of these; `glidertool replay -house SpacePods -room 55 -trace` shows a
flat **17 work and 17 back rects per frame** with the glider standing still and nothing
triggered — 36% of the effective 47 gone before the room does anything. `California or
Bust!` room 10 has 12, `SpacePods` room 105 has 11, `Grand Prix` room 7 has 10.

That is faithful, and it stays. It is recorded here for two reasons. First, it sets the
floor a house author has to budget against, which makes it input to 4.1's linter: a room
with 16 ungated movers plus a mirror (2.19's unclipped per-frame rect) plus a busy glider is
where the 47 actually runs out. Second, if presentation is ever reworked (2.10, 2.24) these
three are the whole reason a naive "only upload dirty rects" backend would not be much
cheaper than uploading the frame — the standing set is not small.

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

### 2.15 Leaving a game requires a physical key release — **DONE, 1.7b**

`WaitCommandQReleased` spins until the player physically lets go of Command-Q before the
game will proceed. On a Mac with a real menu bar that prevented a held chord from firing
twice; in a port it is a hang waiting for a key event that may never come (the window can
lose focus mid-chord). It is deliberately **not** transcribed in `NewGame`.

**1.7b kept it untranscribed and answered the question it was asking.** What the spin is for
is "do not act twice on one press", and the port's answer is an edge rather than a wait: the
pause key is reported on its press edge (`cmd/glidergo/play.go`'s `pauseHeld`), and the give-up
Q arrives as a single `EventKeyDown` rather than as a held-key poll. Neither can fire twice on
one press, and neither can wait for an event that never comes — which is the property this item
wanted and the spin only approximated. The pause's own wait loop does track a release, but it
is `held` on a key the player is holding *now*, and it exits on the window closing too.

### 2.16 The two-player idle freeze has no visual tell — **planned, 1.9**

`TagGliderIdle` freezes a glider for 30 frames — a full second — with no indication that
this is deliberate. Player 2 in particular starts every two-player game idled and hidden
(`Play.c:198-203`), so the first thing a new player experiences is a second of not
existing. A fade, a shimmer, or anything at all would do.

### 2.17 The frame limiter is a busy-wait, and there is no catch-up — **DONE (the hook), 1.5b; the setting is declared, 1.7b; the catch-up is 1.8**

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

**1.7b declared the setting and deliberately left it unwired.** `prefs.KeepRealTime` is in the
file, defaults to false — the C's behaviour — and nothing reads it yet, which is stated on the
field rather than left for a reader to discover. The reason is that a catch-up worth having
needs a resync clamp: without one, the first long stall — a room wipe, a pause, a house load —
is followed by a burst of unpaced frames, which is a worse artefact than the dropped time it
was meant to recover. That is frame-pacing work, it belongs with 1.8's timing pass, and a
setting that is written down and honest about being inert is cheaper to finish than one that
has to be invented later along with the file-format change to carry it.

### 2.18 The random stream is unverified against real hardware — **answered as far as it can be, 1.8b; the physics gap it exposed is open**

`internal/game/rand.go` transcribes the original's linear congruential generator, and the
demo replay in 1.8 depends on it bit for bit. It has not been checked against a trace from
a real Mac, and until it is, a demo that desyncs is ambiguous between "the RNG is wrong"
and "the input replay is wrong". 1.8 should capture a reference trace first.

**1.8b closed the ambiguity from the other end, and the answer changes what this entry is
asking for.** There is no reference trace to capture, because *the original had no reproducible
one*: `ToolBoxInit` does `GetDateTime((UInt32 *)&qd.randSeed)` (`Utilities.c:61`) and nothing
in the shipped game ever reseeds — there are no `InitRandomSeed` callers — so the 1994 attract
mode drew a different random stream on every launch. A player who watched the demo twice saw two
different demos. `RandomInt` reaches gameplay through `ObjectAdd.c` (the drip and toast delays,
`10 + RandomInt(10)`; `RandomInt(kNumFlowers)`) and `Play.c` (phone rings, chimes), so this is
not a cosmetic difference. The consequence: **the shipped `'demo'` resource cannot be a bit-exact
fidelity oracle, and no amount of work on rand.go would make it one.** What it can be is a
determinism oracle for this port — the same stream, the same seed, the same frames, twice — and
that is what `internal/replay/testdata/demo.script` and `TestTheDemoReplaysTheSameWayTwice` are.
Verifying rand.go itself is now an *independent* task (a direct table test against hand-computed
LCG output, 1.8c) rather than something the demo can settle.

**What the demo replay does measure, and the number to beat.** The port does not fly the recorded
path. Replaying the shipped stream against Demo House:

- the glider never leaves room 0, "Air Vents" — three `kFloorVent` at v=305 and a `kRedClock`,
  which it *does* collect for 100 points at frame 310, so the first ~300 frames are plausibly
  right;
- it then fades out (mode 2) frozen at `dest=295,387,315,435`, below the floor, losing a life at
  frames 1412, 1573 and 1760; game over at frame 1775, having consumed **573 of 1117 records**;
- the recording expects the vents to carry it rightward out of the room — the longest held run in
  the stream is 66 frames of right from frame 1879, well past where the port has already died.

Three pieces of evidence say the harness is not what is wrong. The outcome is **seed-independent**:
seeds 0, 1, 7 and 12345 all end at frame 1775 with 573 records consumed and mortals at -1, and only
the pixel digests differ — so the death is physics, not RNG. The recording is **one record per held
frame**, not per alternate frame: `glidertool demo info -stats` reports 48 distinct gaps with
`1:1009` of 1116, 108 held stretches (right 84, left 21, band 3) — so the port's frame counter is
the right clock to replay against. And the parity is even, 557 to 560, so no input pass is running
on only one of `World.EvenFrame`'s two phases. That leaves the vent lift, the fall-through, or the
air-friction integrator, and it is the sharpest fidelity target this project has: **573 of 1117 is
the number to raise.**

Two things follow for whoever picks it up. The frame numbers above live in the script's header
comment, this entry, and a `t.Logf` — deliberately not in an assertion, because the day the physics
improve, the test that fails should be a fidelity test and not a test about determinism. And a
demo that ran the whole stream would read 86 frames past the last record, which is the
off-the-end read the original performed and the port counts as `Cursor.PastEnd`; `frames 3500` in
the script is set past the end on purpose so that a fixed port exercises it.

Two related notes: `PourScreenOn` is dead code in the original — nothing calls it — and it
*draws from the RNG*, so wiring it up would shift every subsequent random number and
invalidate any recorded demo. And `case kLgTrigger:` in the interaction dispatch is
unreachable, because the object code is never produced; it is transcribed as a dead branch
and should stay dead.

**1.5c makes this harder, and the reason is worth writing down before 1.8 hits it.** Three
dynamics sites draw from the stream:

- `AddDynamicObject` (`Dynamics3.c:208`) draws `RandomInt(60) + 15` per `kSparkle`
  registered. Registration is per *locale*, filtered by `SectRect` against the screen, so
  **the stream position after a room change is a function of how many sparkles were on
  screen** — which is a function of the window size. Any demo trace is therefore only valid
  at the resolution it was captured at, and 2.1's scaling work must not change which rooms
  are composited.
- `HandleSparkleObject` (`Dynamics.c:304`) draws `RandomInt(240) + 60` every 60..299 frames,
  per emitter, forever.
- `HandleCoffee` (`Dynamics.c:492`, `:506`) draws `RandomInt(200)` every 100..299 frames per
  active coffee maker, forever.

So from 1.5c onward the RNG is consumed by *scenery*, at a rate set by what an author put in
the room, and not only by gameplay. 1.8's reference capture has to pin the house, the room,
the window size and the frame count together, and `glidertool replay`'s trace header should
carry all four. The alternative — giving the sparkles and the coffee maker their own
generator — would be a real deviation and is not proposed; it is recorded here as the escape
hatch if the demo replay proves impossible otherwise.

### 2.19 The mirror-room flame blink — **DONE as an opt-in fix, 1.7b; default is 2.x**

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

**Shipped in 1.7b as `fixes.mirror_flame`**, off by default. It registers one back rect per
mirror the reflection actually intersects, and none at all when the reflection is entirely
outside every mirror — the defect in its purest form, an erase with no draw behind it. Two
entries in a 47-slot list where the C had one is the cost, and it is still cheaper than what
the unclipped rect costs in erased flames. `game.TestMirrorFlameClipsTheBackRect` asserts
*both* states, because the unflagged path is what 1.8's corpus is recorded against.

### 2.20 A mirror shows the wrong player's foil — **DONE as an opt-in fix, 1.7b**

`DrawReflection` tests `if (showFoil)` with no `!twoPlayerGame` guard, unlike
`RenderGlider`'s. In a two-player game with foil showing, `glid2SrcMap` holds
`kGliderFoil2PictID` — the *other* player's foil sheet — so player 1's reflection is drawn
with player 2's artwork. Clearly unintended, and a player watching a mirror in a
two-player game does see it. A fidelity replay would diverge if it were fixed, so it is
kept and offered as an opt-in correction alongside 2.19 and 2.22.

**Shipped in 1.7b as `fixes.mirror_foil`**, off by default: it adds `RenderGlider`'s missing
`!twoPlayerGame` guard and nothing else. The test that matters is the one for the case the fix
does *not* apply to — `game.TestMirrorFoilChangesNothingInAOnePlayerGame` — because a
correction that turned the foil reflection off in the game almost everybody plays would be a
worse bug than the one it fixed, and it would pass any test that only asserted "the two flags
differ".

### 2.21 `doBackground` should not be a user option — **DONE as a split, 1.7b**

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

**1.7b split the flag in two instead of dropping it**, because it was doing two unrelated
jobs. `World.DoBackground` is the pump and stays true unconditionally — that is the port's
business and not a choice. `prefs.PauseWhenUnfocused` is the player's half, "keep playing while
switched out", and it gates only the `Suspend` call in the host's focus-loss arm. It defaults to
pausing and is deliberately **not** on the settings screen, for the reason this item gives; a
file that turns it off really does keep the game running in the background, which is where an
imported 1994 `doBackground` lands. The mid-frame spin is preserved, and 2.28 is what now draws
on the screen while it spins.

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

### 2.25 `RedrawRoomLighting` redraws nine rooms where the original redraws one — **DONE, 1.5c**

An earlier draft of this entry said the lighting redraw "runs on a schedule rather than on
demand" and asked for a dirty flag. That was wrong on both counts, and is corrected here
rather than quietly deleted: the port's `RedrawRoomLighting`
(`internal/game/readylevel.go:139`) is already on-demand and already gated on the C's
`wasLit != isLit` transition. The real deviation is the *width* of the redraw, and the
function's own comment has recorded it accurately since 1.5b.

`RoomGraphics.c:448-460` recomposes the **central room only**, in six steps:

```c
DrawRoomBackground(localNumbers[kCentralRoom], kCentralRoom, roomV);
DrawARoomsObjects(kCentralRoom, true);          // <- the redraw flag
DrawLighting();
UpdateOutletsLighting(localNumbers[kCentralRoom], numLights);
if (numNeighbors > 3) DrawFloorSupport();
RestoreWorkMap();
```

The port calls `World.Rebuild()`, which is `DrawLocale` — all nine rooms, with
`redraw == false`. `redraw == true` is what suppresses the registration sites inside
`DrawARoomsObjects`, so the original repaints the pixels *without* re-creating the live
objects; `Rebuild` re-creates them, which deletes a band in flight, rebuilds the mirror
region and restarts a dinah mid-swoop. It is tolerable today only because no shipped room
with a light switch also has one of those things.

**Closed in 1.5c.** `Scene.RedrawCentralRoom` is `internal/render/locale.go:373` and holds
the six steps verbatim; `internal/game/readylevel.go:164` calls it in place of `Rebuild`,
keeping the recount, the work rect and the `ShadowVisible` recache on the game side exactly
where the C splits them. A band in flight, the mirror region and a dinah mid-swoop now all
survive a light switch. The paragraphs below are the analysis that got it there, kept because
they say *why* the narrow redraw is the correct one.

**Most of the fix already existed.** `internal/render/locale.go:510` is already
`func (s *Scene) DrawARoomsObjects(neighbor int, redraw bool)` with its twenty `!redraw`
gates in place, and `candleFlame`, `backUpToSavedMap` and `addGrease` all take the flag
through. `Scene.NumLights` (`locale.go:180`) is already the C's live per-room global,
reassigned before each of the nine draws with the central room last (`:283`, `:291`,
`:298`) — which is what makes `wasLit` read the central room's count. What is missing is a
`Scene.RedrawCentralRoom()` holding the six steps above, which is `DrawLocale`'s own
central-room tail (`locale.go:298-306`) with `redraw` flipped to `true` and one hook added.

So the remaining work was one small render method and a one-line change at the call site.
It was timed to 1.5c because step four, `UpdateOutletsLighting`, writes the `dinahs` table
that 1.5c creates — the entry could not have been closed before it, and there was no reason
to defer it past.

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

### 2.28 A paused game is indistinguishable from a hung game — **DONE, 1.7b**

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

**1.7b gave it one, shared with `DoPause` as this item asked.** `pumpWhileSwitchedOut`
(`internal/game/pause.go`) is the C's `do { HandlePlayEvent(); } while (switchedOut)` with a
panel drawn every pass and the frame restored from the work map on the way out. Three details:

- **It borrows the placard's rect and its no-art panel, and neither picture.** Both PICTs name
  a key to press and the thing that ends this pause is not a key, so the panel says "click the
  window to resume" — the only place in the port that refuses the original's artwork on purpose.
  `game.TestTheSwitchedOutPanelIsNotThePausePlacard` is what stops that regressing to the
  placard, which would tell a player to press a key that does nothing.
- **Every pass, not once**, for the same reason the pause loop repaints: an expose answered
  mid-suspend copies the whole play area up from the work map and would wipe it.
- **A build with no `Present` hook gets the C's loop exactly** — no draw, no restore, no extra
  rect copy per frame in the one configuration that runs millions of them.

A dimmed frame is still not offered, and not for want of trying: 2.50 is why a 50% dim is not
available on an 8-bit indexed surface with no alpha.

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

### 2.32 Three things stop the world from inside a frame — **DONE: `DoPause` 1.7b, both banners 1.7d**

`DisplayStarsRemaining` (`Banner.c:205-243`) is the clearest case. It is called from
`Interactions.c:946`, inside the star-collection arm of `HandleInteraction` — so from the
*middle* of a frame, after the interaction pass and before the glider moves — and it draws
to the main window, then `DelayTicks(60)`, then `WaitForInputEvent(30)`. **Those two numbers
are in different units**: the first is 60 ticks and the second is 30 *seconds*, because
`WaitForInputEvent` multiplies its parameter by 60 (`Utilities.c:445`). So it is one second
during which the process does nothing at all — no repaint, no resize, no window close — and
then up to thirty more. `BringUpBanner` (`Banner.c:171-197`) does the same with
`WaitForInputEvent(15)`: fifteen seconds at the start of every house, four in a demo. `DoPause`
is the third.

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

The port's two banners are `internal/game/banner.go` and `DoPause` is `internal/game/pause.go`,
and each carries a comment pointing here. The count they consume had to be a simulated one — not
a sleep — which is what `internal/game/wait.go` is.

**`DoPause` is done (1.7b) and it is the easy one**, because a pause has no duration to
reproduce: it lasts as long as the player holds it up, so there is nothing to convert into a
frame count. What it does establish is the shape the other two should take — the game draws and
hands the waiting to the host through a hook, and a nil hook means "do not wait at all", which
is exactly what a replay needs. The two banners *do* have durations — 60 ticks, and 15 seconds —
so they are the ones that needed the simulated form.

One consequence of the pause landing first: a fidelity trace can now contain the pause key
without hanging the harness, because `World.Pause` is nil in every headless build. That was the
first thing this item warned about and it is closed by construction rather than by care.

**1.7d closes the other two, and both game-over animations with them.** `internal/game/wait.go`
holds the arithmetic — `FlushEvents`, `DelayTicks`, `WaitForInputEvent`, and the two-tick
`pollInput` both endings pace themselves with — and `World.Wait(ticks, discard) Waited` is the
hook underneath all four. `cmd/glidergo`'s answer polls the window, keeps presenting, and honours
the deadline in two-tick slices; the headless one is nil, so a replay crosses every one of these
screens in zero simulated time and sees every pixel they draw. The three durations are asserted
as numbers in `internal/game/banner_test.go` and `gameover_test.go`, because the one way to get
this wrong with no visible symptom is the unit.

Two of the three problems above are therefore closed and the middle one is not. **Input during a
wait is still swallowed**, deliberately: a key pressed while the stars-remaining panel is up is
consumed by the wait and is not held when play resumes, so a player walking right into a star
stops walking. Correcting it means the panel no longer consumes the keystroke, which changes what
a recorded demo does next — so it waits for 1.8's replay corpus to exist to be checked against.

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

So the blank rect `HandleOutlet` paints lands in whichever GWorld the previous drawing call
happened to leave set, while the `AddRectToWorkRects` on the next line asserts it went to
the work map. The commented line names `workSrcMap`, so the 1994 behaviour is not in doubt;
only the shipped Carbon build's is.

*When* it paints was established while specifying 1.5c and is narrower than this entry
first said. The `PaintRect` is the `else` of
`if ((dinahs[who].position != 0) || (dinahs[who].hVel > 0))`, inside the
`position != 0` zap branch — and `hVel` is the room's light count
(`UpdateOutletsLighting`, `Trip.c:235-244`). So it is reached only on the **final frame of a
zap**, after that frame's `timer <= 0` arm has already cleared `position`, and only in a
room with no lights. Its purpose is to paint the socket out of a dark room rather than
leave `outletSrc[0]` showing. An outlet at rest never reaches this code.

**DONE — the outlet in 1.5c, grease in 1.5e.**
`internal/game/dynamics_appliances.go:427` is
`w.R.Work.Fill(w.Dinahs[who].Dest, render.Black8)` — the destination is named at the call
site, which is what the commented-out `SetPort((GrafPtr)workSrcMap)` was for, and the
`AddRectToWorkRects` on the next line is therefore telling the truth.

`HandleGrease`'s spreading arm (`internal/game/grease.go`) is now the model the C already
was, transcribed rather than repaired: `w.R.Back.Fill(src, ...)` then
`w.R.Work.Fill(src, ...)` then `w.AddRectToWorkRects(...)`, back map first as
`Grease.c:105-118` has it, all three naming the same `src`.
`TestSpreadingGreaseRegistersTheRectItPainted` asserts the registered rect *is* the filled
one, in screen coordinates with no conversion between them — which is the property the
outlet's version cannot have and the one this entry is about.

Transcribing the *bug* would have been untestable here for a reason worth stating: the port
has no ambient current port to leak into, so a missing destination is a compile error or an
obvious nil — the failure mode the C has is not available, which makes "be faithful"
meaningless and "be correct" the only reading. That is why this one is absent from 2.35's
transcribed-bug list.

### 2.35 Stage 1.5c transcribes four bugs the original shipped — **decided in the 1.5c specs**

Distinct from 2.33, which is about reads the port *declines*. These four are wrong but
harmless-to-run, they are visible in play, and 1.8's fidelity replays exist to hold them —
so all four go in as written, each with a comment naming the intended behaviour. Listed here
so that a future reader who spots one does not "fix" it, and so that if the project ever
ships an opt-in fidelity-vs-fixes switch (compare 2.19, 2.20, 2.22) this is the candidate
list. Full arguments in `docs/analysis/stage15-raw/plans/`.

- **`AddDynamicObject` clobbers the global `evenFrame`** (`Dynamics3.c:474`, `:524`). The
  neighbouring `lilFrame = true` is a *local* (`:191`); `evenFrame` is an extern (`:23`). The
  author seeded what he took to be two loop-local toggles and one name resolved to a global,
  so registering a ball or a fish resynchronises every flame and star in the locale on entry.
  `HandleBall`'s third write (`Dynamics2.c:420`) is the deliberate one these were copied
  from — it phase-locks the half-rate gravity so the reverse-engineered launch velocity
  reaches the height the author typed. Transcribing this changes what
  `internal/replay`'s `TestEvenFrameIsAStoredFlag` means, which is exactly what that test was
  written to catch.

  **Measured in 1.5c, and it is one frame wide, not permanent.** An earlier draft of this
  bullet — and the field comment on `World.EvenFrame` — said a ball desynchronises the locale
  for the rest of the room. It does not, because of a detail worth spelling out: the loop head
  (`internal/game/play.go:297`) is the only writer that *toggles*. All three others **assign
  `true`**, so a write forces a phase rather than flipping one, and a later write replaces an
  earlier one instead of compounding it. In every shipped ball room the composition write and
  the ball's first idle write (`dynamics_movers.go:452`) land one frame apart and cancel, which
  leaves exactly one diverging frame at composition time. Confirmed by replaying rooms 30
  "Ball Illusion", 172 "Attica, Greece" and 175 "Dodgeball" of CD Demo House against two
  ball-free controls: `replay.TestABallBreaksTheEvenFrameInvariant`, with the mechanism pinned
  by `game.TestBallResetsParityRatherThanTogglingIt`. A ball switched on *mid*-room by a
  trigger has no cancelling write, so there the shift does persist — no shipped house does it
  (all ten `kBall` lines in CD Demo House are `initial 1`), but a Stage 2 house could.
- **`ObjectDrawAll.c` writes `dynamicNum` from the wrong index** (`:518`, `:531`, `:544`,
  `:557`, `:570`, `:574`): `masterObjects[i].hotNum` with `i` the room-object *slot*, where
  the master index is `n`. In range but wrong, so no guard applies. Correct only for the
  central room, which is composited first with all 24 slots in order; for the other eight a
  trigger wired to a switch throws the central room's hot spot at that slot, or reads
  `hotSpots[-1]` — which *is* 2.33's territory and is refused there.
- **`HandleOutlet`'s two-player fan-out passes `doOffset` inconsistently**
  (`Dynamics.c:539`, `:541` versus `:545`, `:546`, `:550`). The outlet is one of the seven
  appliance types whose registration bakes `playOriginH/V` into `dest`, so the surviving
  player of a two-player game is tested against a screen-coordinate rect and the zap misses
  by the scroll offset. Zero in a room at the house origin, wrong everywhere else. The port
  writes that one fan-out longhand rather than through its shared helper, so the mismatch is
  visible at the call site instead of hidden behind a parameter.
- **`IsRectLeftOfRect` has an operator-precedence bug** (`RectUtils.c:185`):
  `(w1) - (w2) / 2` where the author meant `(w1 - w2) / 2`. With a 48px glider and 24px art
  the intended offset is -12 and the actual one is 0, so the shove direction is decided by
  left edges rather than centres and a dart clipping the glider's left side shoves it
  *into* the dart. One caller in the whole game, `Dynamics.c:53`.

### 2.36 A four-frame enemy reload would fire with no warning sparkle — **note; unreachable by arithmetic, and a Stage 5 editor could reach it**

`enemyWaiting` (`internal/game/dynamics_movers.go:121`) is the shared idle arm of the
balloon, the copter and the dart, and it announces a coming enemy with a sparkle and
`EnemyInSound` in two mutually exclusive places:

- at launch, `if Count < StartSparkle` — a reload so short there is no room to warn early, so
  the puff happens as the enemy appears;
- four frames before launch, `else if Timer == StartSparkle` — the normal case.

`StartSparkle` is 4, and the two conditions leave a hole at exactly `Count == 4`. `Count < 4`
is false, and `Timer` is reset *to* `Count` and then decremented before it is compared, so it
takes the values 3, 2, 1, 0 and never 4. An enemy on a four-frame reload would therefore
launch in complete silence, every time, with no puff — the one reload period the original
cannot announce.

**It is unreachable in any house that exists, and the reason is arithmetic rather than
authoring taste.** All four enemy arms compute `Count = (Delay * 6) / TicksPerFrame`
(`internal/game/dynamics.go:390`, `:415`, `:442`, `:482`) and `TicksPerFrame` is 2, so
`Count == Delay * 3` — always a multiple of three. 4 is not, so no value of the house file's
one-byte `Delay` can produce it. Verified over the whole reachable set by
`game.TestEnemyWaitingHasOneSilentPeriodAndItIsUnreachable`, which asserts both halves: that
`Count == 4` really is silent, and that no `Delay` yields it.

Recorded rather than fixed, and recorded rather than merely left alone, for one reason: the
Stage 5 editor will write `Count` if it ever grows a per-object override, and 1.9's
multiplayer tuning is the other place a frame count gets typed by hand. Anything that writes
a reload period in frames instead of in `Delay` units can land on 4 and produce a bug that
looks like a missing sound file. The test is the guard; this entry is why it exists.

### 2.37 The television's movie branch is a deliberate blank, and the movies are already extracted — **planned, 1.5e (decoder) / note (the divergence)**

`HandleTV` (`internal/game/dynamics_appliances.go:278`) always takes the non-QuickTime path,
which is the behaviour of a 1994 Mac without QuickTime installed: switching a set on blits
`tvScreen2`, the static "on" cel. On a machine that *had* QuickTime, and for a house that
shipped a movie, the C's active arm is an **empty `if`** — no blit and no reveal — because the
movie was expected to draw over that rect instead. The port cannot reach that arm even in
principle: `Room.TVMovieNumber` stays at the `-1` that `Rebuild` resets it to, so the
`who == tvWithMovieNumber` identity test is never true.

**The divergence is bounded and known, but it is not empty.** Nineteen of the 22 shipped
houses contain a television — `Titanic.house` has 13, `Land of Illusion.house` 10,
`Slumberland.house` and `SpacePods.house` 8 each — and **15 houses shipped a movie**, which
the extractor has already pulled out as 8-bit index buffers under `assets/extracted/movie/`
(see `manifest.tsv`: all 64×49 except Demo House's 82×62, 7 to 45 frames apiece, `raw`, `rle`
and `smc` codecs). So the missing feature is a decoder and a playback clock, not the assets.
Until then, a TV in one of those 15 houses shows a still frame where the original showed
video.

Four sites are already named no-ops waiting for it, deliberately kept as functions rather
than as comments so that 1.5e has one body each to fill and cannot fill three out of four:
`World.restartRoomMovie` (`internal/game/transit.go:718`), the `StopMovie` block in
`ReadyLevel` (`readylevel.go:44`), `ToggleTV`'s block (`trip.go:83`), and `HandleTV`'s own
arm. `World.TVOn` is correspondingly never written, which is the correct transcription and
not an omission — on a machine with no movie the C never enters the block that writes it
either.

Two things to get right when the decoder lands. The `raw`/`rle`/`smc` triple means three
decompressors, and `smc` is the only one that is not trivial. And `TVOn` is not reset on a
room change in the original, so a set switched on in one room restarts the movie in every
later room that has one — that is 1994 behaviour to reproduce, not a bug to fix, and it is
the field comment on `World.TVOn` that says so.

### 2.38 A switch wired to a star makes a house impossible to finish — **fix planned, Stage 2 (needed before new houses ship); the linter is 4.1**

`switchLinkedObject` (`internal/game/switches.go`) groups `kStar` with the eight other
prizes, so a star removed by a switch gets the background restore and the puff of light and
**neither `StopStar` nor the `StarsLeft--`** that `HandleRewards`' own star arm performs. Two
consequences, and the second is not cosmetic:

- the star's six-cel spin keeps animating over the background that was just put back, so the
  room shows a rotating star with no star under it;
- the star object is gone, so nothing can ever collect it, and `StarsLeft` never reaches zero.
  **The house cannot be won.** There is no other decrement anywhere in the game.

`game.TestSwitchOnAStarStrandsTheHouse` pins it in both halves — the count is unchanged, and
a second attempt to collect the removed star also changes nothing.

**No shipped house does it.** `game.TestShippedHousesWireSwitchesToPrizes` walks all 1,563
switch and trigger plates in the 22 original houses, resolves each link the way
`GetRoomLinked` does, and finds 165 links to a bonus object — 146 of them from a switch rather
than a pressure plate — and **zero to a star**. So this is not a defect a player of the
originals can reach; it is a trap for anything that authors new content, which is exactly what
Stage 2 and Stage 5 are.

(The link has to be resolved through `GetRoomLinked` and not read as a room index, which is
what the first version of this survey did. `data.e.where` is a packed floor/suite pair — 6709
is floor 26, suite 53 — so reading it as an index resolves nothing and reports a clean corpus.
The test now fails outright if no link resolves, so the same mistake cannot produce a
reassuring answer twice.)

Three actions, in the order they are needed:

1. **Stage 2, before the first new house ships:** fix it. The star arm needs its own case with
   `StopStar` and the decrement, matching `HandleRewards`. Since no shipped house reaches the
   arm, the fix cannot change any existing house's behaviour — which makes this the rare
   reproduced bug that costs nothing to correct, and is why it is not on 2.35's leave-alone
   list. The fidelity replays (1.8) will not notice, for the same reason.
2. **4.1, the house linter:** an unreachable star is only one way to make a house
   unwinnable; a linter that already walks the object graph should report any switch or
   trigger link that can remove a star, and should do it whether or not the runtime is fixed.
3. **Stage 5, the editor:** the link picker should not offer a star as a switch target at all.

### 2.39 The switch's spurious corner sparkle fires 145 times in the shipped houses — **DONE as an opt-in fix, 1.7b**

`HandleSwitches` declares `bounds` and never assigns it before the prize arm's `AddSparkle`
reads it (`Interactions.c:995` against `:1043`). On 68k that was whatever was on the stack; in
Go it is the zero rect, so the puff lands at the top-left corner of the play area rather than
somewhere unpredictable. The sparkle the player is *meant* to see comes from
`RestoreFromSavedMap`'s own `doSparkle` arm one line earlier and is correctly placed on the
prize, so this second one is pure noise — nothing reads it, nothing depends on it, and
removing it changes only what is on screen.

**It is not latent.** The same corpus survey counts **145 switch links into the prize arm**
across the 22 original houses — 48 to paper, 40 to foil, 23 to a yellow clock, 12 to helium,
and so on — so a player of the shipped content sees a stray flash at the corner of the room
every time one of those switches is thrown. It is the sort of thing that reads as a rendering
bug in the port rather than as fidelity to 1994.

Kept for now, because 1.8's fidelity replays compare sparkle tables and this is a real
difference in them. Removed behind the same modern-options flag as 2.19, 2.20 and 2.22, where
one line deletes it. `game.TestTheSpuriousSparkleLandsAtTheCorner` asserts the current
behaviour, including that the puff is at the origin rather than merely somewhere.

**Shipped in 1.7b as `fixes.switch_sparkle`**, off by default, and it is one `if`. The tested
property is that it drops *exactly* the second puff and keeps the one on the prize
(`game.TestSwitchSparkleDropsTheSpuriousPuffAndKeepsTheRealOne`): a switch that removed a prize
with no puff of light at all would be a worse outcome than the stray flash, because the puff is
how a player learns that something they were not looking at has just vanished. It also frees a
slot in a three-entry table that silently drops the fourth request, which is the half of the
argument that is not cosmetic.

### 2.40 Three of the twenty-three switch arms have never run in any build — **note; matters to Stage 5's editor**

The second dispatch in `HandleSwitches` runs only when `SetObjectState` returned true, and
`SetObjectState` returns false for `kSlider`, `kSoundTrigger` and `kGuitar`. So all three arms
are dead code in the original as well as in the port, and `game.TestThreeSwitchArmsCannotBeReached`
pins the fact against a future change to `SetObjectState` bringing one to life untested.

`kSlider` and `kGuitar` cost nothing: the slider arm is empty anyway, and the author's own
comment on the guitar is "really no point to change this state". `kSoundTrigger` is the loss.
It sits in `SetObjectState`'s *switch* family, which returns false because a switch has no
state of its own — so **a switch wired to a sound trigger has never played a sound**, in 1994
or now, however plainly `switchLinkedObject` says it should. `FireTrigger`'s own sound-trigger
arm does work, which is presumably why nobody noticed.

Not fixed, because making it work would give existing wiring in the shipped houses a new
audible effect, and because the custom sound loader it wants is 1.6. Recorded because the
Stage 5 editor should not present a link the game cannot honour: either the picker omits
sound triggers as switch targets, or 1.6 makes the arm reachable on purpose and the fidelity
replays get a note.

### 2.41 The rubber band debounce does not debounce — **note; three separate defeats, all transcribed**

`bandHitLast` (`RubberBands.c:29`) is meant to stop a band resting against a switch from
toggling it thirty times a second. It is a single global holding a single hot-spot index, and
it fails in three independent ways. All three are reachable in the shipped houses, all three
are reproduced, and each has a test in `internal/game/bands_test.go`.

**One band, two rects.** The debounce suppresses the *effect* and not the sweep, so after
tripping hot spot 5 the loop carries on and hot spot 9 sees `bandHitLast == 5`, trips too, and
leaves `bandHitLast == 9`. Next frame hot spot 5 trips again because the latch now names 9.
Two switches within sixteen pixels of each other toggle each other's latch for ever.
`TestTwoRectsDefeatTheDebounceForEachOther`.

**Two bands.** The reset is per band, not per frame: `CheckBandCollision` ends with
`if (!nothingCollided) …` else clearing the latch to -1, so a *second* band in free flight
clears it every frame and the first band's latch never survives. This is the easy one to hit
deliberately — fire twice, hold one band against a switch — and it is a straightforward way
for a player to farm a switched prize. `TestSecondBandDefeatsTheDebounce`.

**The initial value.** It is a zeroed global rather than -1, and `KillAllBands` does not reset
it, so the first band collision of a session is silently *swallowed* if it happens to be with
hot spot 0. Go's zero value reproduces that for free, which is why `World.BandHitLast` is
deliberately left uninitialised. `TestFirstCollisionWithHotSpotZeroIsSwallowed` and
`TestKillAllBandsLeavesTheDebounceLatched`.

Not fixed at 1.5e, for the usual reason: the effect is visible in play, so 1.8's fidelity
replays have to hold it before it can move. The shape of a fix is not in doubt though, and it
is smaller than the bug — a per-hot-spot `bandStillOver` flag cleared by the same sweep that
clears the glider's `StillOver`, which is what `HandleSwitches` already has and what the band
path borrows without getting its own. That makes it the same fix as giving bands their own edge
detector, and it belongs with 2.19/2.20/2.22/2.39 behind the modern-options flag rather than as
a silent correction, because the two-band defeat is exploitable and someone's route through a
house may depend on it.

One consequence worth separating out, because it is a *design* decision and not a defect: only
five of the 28 hot-spot actions are offered to a band, and `kRewardIt` is filtered to grease
alone. **Bands cannot collect prizes.** Without that filter a player could farm a room's clocks
from across it. `TestOnlyFiveActionsSeeABand` pins the list in both directions, driven off
`NumHotSpotActions` so that a twenty-ninth action added without a decision about bands fails
there rather than in a house.

### 2.42 The random stream is a function of the play area's size — **note; binds 2.1, 2.8 and Stage 3**

Five of the room-load registrations end by seeding an animation's phase from `RandomInt`: the
candle, the torch and the barbecue pick a starting cel, the star picks one of six, and the
pendulum flips a coin for its direction. In all five the draw sits **after** the
`if (savedNum == -1) return;` — verified one function at a time in `DynamicMaps.c` — so an
object that fails to get a saved-map slot does not consume its draw either.

Two things then follow, and the second is the one that matters.

**Registration is screen-size dependent.** Every one of the five `Add*` functions is called
from `ObjectDrawAll.c` only for objects that `SectRect` against the composed area, so *which*
objects register — and therefore how many `RandomInt` draws a room load performs, and therefore
every subsequent value in the stream — is a function of how much of the house is on screen.
That is `Scene.NumNeighbors` (1, 3 or 9) and the size of the play area.

**Saturation would do the same thing, and no shipped house reaches it.** 1.5f's census
measured it rather than assuming: over 4,070 rooms in the 22 houses the busiest locale claims
16 of the 24 slots, and none drops a registration (see 2.44). So the saturation half of this is
a live code path that shipped content never takes, exactly as `AddDynamicObject`'s cap turned
out to be in 1.5c — but the `SectRect` half is reachable by anyone who changes the window.

What protects it is a rule already written down for another reason: **2.8, the scale transform
belongs at the present step and nowhere else.** Composing 640x480 and scaling on the way to the
display keeps the play area fixed at every window size, so the stream is untouched by 2.1's
scaling and fullscreen work. This item is the reason that rule is not merely tidy.

Two consequences to carry forward:

- a replay script is only valid at the play area it was recorded at. `neighbors` is in the
  script format already; the play-area size is not, because there is currently only one. If a
  resolution option is ever added that changes what is *composed* rather than what is
  presented, the script format needs a line for it and `internal/replay` needs to refuse a
  mismatch rather than silently produce a different run.
- Stage 3's race gives each machine its own world, so two machines drawing different flame
  phases is cosmetic. It stops being cosmetic the moment anything is shared across the wire
  that was derived from the stream — a race seed, a shared house, a spectator view — so the
  handshake should carry the play area and the neighbour count and refuse a mismatch.

Pinned by `TestASaturatedTableDoesNotConsumeARandomDraw` (`internal/render/anim_test.go`),
which fills the table and asserts the draw is *not* consumed, and by the `rand` column of the
replay trace, which is where a shifted stream shows up as a whole-column diff.

### 2.43 `AddAShreddedGlider` writes one element past its table — **DONE as a reported deviation, 1.5f**

`DynamicMaps.c:726` guards with `if (numShredded > kMaxShredded) return;` — strictly greater,
against a table of exactly `kMaxShredded` elements (`shreds` is a `NewPtr` of
`sizeof(shredType) * kMaxShredded`, `Environ.c:654`). So a fifth shredded glider passes the
guard, writes `shreds[4]` of a four-element allocation, and leaves the counter at 5. On a 1994
Mac that scribbled twenty bytes over whatever the Memory Manager put next.

It is more reachable than the numbers suggest. A spent cloud keeps its slot — neither arm of
`RenderShreds` matches at frame 20, so it stops animating and is never reclaimed — and
`RemoveShreds` removes exactly one entry per death and **none at all** if the only cloud is
still growing. So four shreds in one locale visit fills the table permanently, and two-player
mode halves the work.

The port tests `>=` and drops the fifth cloud. That is the one deliberate divergence in
`internal/game/shreds.go`, for 2.33's reason: reproducing it would mean reproducing a bug whose
observable behaviour is "corrupt something else", which is not a behaviour a port can be
faithful to and not one a released game may have.

What is new at 1.5f is that the refusal is **reported**. The guard goes through `badIndex`, so
it lands in `World.Diag.Guarded` and `Diag.Seen` and is named by `glidertool replay` like any
other deviation. It is the only site in the port where that guard stands in front of a *write*
rather than a read, which is why it has its own kind (`devShred`, `guards.go`) — from a bug
report's point of view the two are the same finding, that a house reached a place the original
survived by luck. `TestTheFifthCloudIsDroppedRatherThanWrittenOutOfBounds`.

### 2.44 A refused saved-map registration makes an object invisible, silently — **the report DONE, 1.5f; the author-facing half is Stage 5**

`backUpToSavedMap` answers -1 when the 24-slot `savedMaps` table is full, and **every caller
gates the object's draw on the result.** So the twenty-fifth animated or collectable object in
a locale is not merely un-animated, it is not drawn at all: a candle with no flame, a prize that
cannot be seen and can still be collected by walking into it. The original reports nothing. A
house author found out by playing the room and noticing something missing.

`Scene.SavedMapDrops` records each refusal — the room, the object slot and the rect it asked
for, which names the family without a label because no two request the same size. It is a
report and nothing reads it, on the same terms as `World.Diag`: a diagnostic that could change
the composition would be worse than none.

The census that came with it is the useful half, and it contradicts what `docs/PLAN.md`
assumed. Over **4,070 rooms in 22 houses the busiest locale claims 16 of the 24 slots**
(`Slumberland` room 256 "Flaming Pathway", tied by `Teddy World` room 322) and **not one room
drops a registration**. So the cap is a live path that shipped content never takes — the same
answer 1.5c got for the 18-slot dinahs table, except that three shipped locales *reach* that
one and none comes within eight of this one.

Two things follow. A Stage 2 house has to be **checked** against the cap rather than assumed
under it, which makes this the third item on 4.1's linter list (with 2.38 and the dinahs cap).
And Stage 5's editor can say it while the room is being built, which is the whole difference
between a diagnostic and a tool. Watched corpus-wide by the `dr=` column of
`internal/render/testdata/locale_golden.txt`, room by room, and asserted by
`TestTheSavedMapBudgetSaturatesInShippedContent`, which fails if any shipped room ever starts
dropping; `TestDroppedRegistrationsNameTheObjectThatVanished` fills the table by hand and
enumerates the three refusals in order.

### 2.45 The loop points in the sound headers are ignored — **note; the samples are written to be retriggered**

Every `'snd '` header carries `loopStart` and `loopEnd`, and the port reads neither. That looks
like an omission and is not: the original does not loop either — it issues `bufferCmd` and a
`callBackCmd`, and a `bufferCmd` plays a buffer once — and the four sounds that need to be
continuous are *composed* to be retriggered instead. `Hiss` is 2960 samples, which is four game
frames to the last sample (4 × 740), and `Input.c:174` asks for it every fourth frame while the
helium key is held, so one channel plays it back to back with no seam. `Sizzle` is three frames
long and is asked for every frame, so it occupies three channels and covers itself the same way.
`Thrust` and `Shred` are cut slightly *short* of their cadence, so their copies overlap and
thicken.

Looping would therefore be a bug rather than a feature. A looped `Thrust` would keep firing after
the player let go of the key, and — worse — its channel's completion callback would never run, so
`priorityN` would never fall back to 0 and the channel would be lost for the rest of the session:
2.47's failure, arrived at from the other direction. The table is checked against the extracted
samples by `TestContinuousSoundsCoverTheirCadence`, so the argument fails loudly if a future
extraction disagrees with it.

### 2.46 The mix is labelled 22255 Hz and the samples are 22254.5454… Hz — **note; 35 ppm, accepted deliberately**

The Macintosh rate is the exact fraction 244800/11 Hz, which is what the resource headers carry
(`Fixed 0x56EE8BA3`). The port labels its output stream with the rounded integer, because every
audio API worth writing to — WAV headers included — takes an integer rate, and it does not
resample to get there.

The error is 0.4545 Hz in 22254.5, which is 35 parts per million, or 0.06 cents of pitch: four
hundred times smaller than the smallest pitch difference a human ear can detect, and about a
twentieth of a sample's drift per second. The alternative is a rational resampler on every sample
in the bank — either at load time, which loses information for no audible gain, or in the mixer,
which puts an interpolator in the hot path of a mono 22 kHz mix to correct a 0.06-cent error. It
is not worth its complexity. What *is* worth doing is the per-sound conversion the houses need
(nine distinct rates occur across their 58 usable sounds), and that is implemented: `Sound.Step`
is a 16.16 fixed-point increment and `stepFor` rounds it so that every sound authored at any of
the three spellings of the Macintosh rate gets exactly `FixedOne` and is copied rather than
resampled. Drop-sample and not interpolating is the original's own choice, not a shortcut —
`Sound.c:379` and `Music.c:287` create all four channels with `initNoInterp`.

### 2.47 `FlushAnyTriggerPlaying` silences the game permanently — **DONE as a reported deviation, 1.6**

`FlushAnyTriggerPlaying` (`Sound.c:262-273`) stops a trigger sound that is still playing when the
player leaves the room. It issues `quietCmd` to stop the sample and `flushCmd` to empty the
channel's command queue — and the `callBackCmd` that `PlaySoundN` queued behind the `bufferCmd`
is *in* that queue. Flushing it means the completion callback never runs, and the callback is the
only thing in the program that ever writes `priorityN` back to 0. The channel is left claiming
`kTriggerPriority`, which is 999, for the rest of the session: `InitSound` runs once, at launch
(`Main.c:337`), and nothing else resets it.

One interrupted trigger sound costs a channel. **Three of them leave `PlayPrioritySound` with
`lowestPriority == 999`, which refuses every ordinary request, while the trigger-exclusivity rule
refuses every trigger. The 1994 game goes silent and stays silent until it is relaunched.**
Reaching it needs a house with a sound trigger, a sound long enough to still be playing on the way
out of the room, and a player who leaves — which is a description of `Art Museum`, whose
"Security" is 2.2 seconds and whose triggers are in corridors.

The port quiets the channel *and* resets the three fields, as though the callback had run. Two
things make that safe rather than presumptuous: nothing in the simulation reads channel state, so
no replay and no digest can move; and the deviation can only ever make the port louder than the
original, never quieter, so it cannot hide a sound the 1994 build played. There is also one piece
of evidence that the author knew something was wrong here — the call to `FlushAnyTriggerPlaying`
inside `LoadTriggerSound` is commented out (`Sound.c:275`).

### 2.48 The audio sink is a subprocess, not a device — **note; a native driver is Stage 4 and Stage 6 work**

The port has no audio driver. `audio.Pipe` writes raw PCM to `pw-play`, `paplay`, `aplay`,
`ffplay` or `play`, whichever is installed, and `audio.WAV` writes a file. That began as a
constraint — the build host is airgapped, so there is no `golang.org/x/sys`, no `oto` and no
`ebiten`, and ALSA or PulseAudio means cgo — but it is a defensible answer on Linux on its own
merits: every desktop ships at least one of those five, they all read s16le mono on stdin, and a
player that crashes takes nothing with it, which is more than can be said for a cgo audio callback
inside the game's address space.

The cost is honest: one process, one pipe, and a latency the port does not choose, because the
player picks its own buffer size. A sound is heard 50 to 200 ms after the frame that asked for it.
Nothing in Glider PRO depends on sub-frame audio timing, so this is fine — but it is fine *on
Linux*. **Windows has no such tool in the box, and this is the one part of the audio path that
Stage 4 cannot simply cross-compile.** `Sink` is two methods wide precisely so that a native
driver can be dropped in behind it without the engine knowing; WASAPI via `syscall` (no cgo, and
no dependency the airgap forbids) is the likely shape, and CoreAudio at Stage 6 is the same
problem again. Until then Windows gets `-wav` and silence, which must be stated in the release
notes rather than discovered.

### 2.49 Five house sounds ship as silence — **note; blocked on data, and 4.1's linter should say so**

Five of the 63 `'snd '` resources in the shipped houses are MACE 6:1 compressed (`cmpSH`,
`compressionID` 4): `CD Demo House` 3007 "Door Chime", `Demo House` 3011 "Meow",
`Nemo's Market` 3001 "Door Chime", 3003 "Cash Register" and 3004 "Cat Meow". The quantisation
tables that would decode them live in the Sound Manager and are in no file in this tree, so the
extractor records them as explicit skips and the port plays nothing.

That is not merely a missing noise. A sound trigger whose sound does not load gets **no hot spot
at all** (`game/hotspots.go`), which is the C's behaviour too — so those five rooms compose
slightly differently from how their authors saw them, and `Demo House`'s only custom sound is one
of the five. The port is faithful here by accident rather than by choice, which is the reason to
write it down.

Two ways out, neither urgent: take the published MACE tables from an open-source Mac audio decoder
(`audio.md` OQ3), or re-record five short sounds — which is a content change and belongs with the
Stage 2 houses, not in Stage 1. What is owed sooner is that the Stage 2 linter (4.1) already has
"custom trigger sounds that do not load" on its list from 2.14, and these five are the corpus it
should be tested against: a house author whose sound is silently dropped should be told at build
time, not by playing the room.

---

### 2.50 A 50% dim is not a panel, on a surface with no alpha — **DONE as a decision, 1.7a**

There is no way to draw a translucent panel over the splash illustration, because
`render.Surface` is 8-bit indexed with the 1994 palette and has no channel to be translucent
in. What the original does everywhere it puts something over existing pixels is
`PenPat(gray)` with `PenMode(patOr)` — a 50% checkerboard ORed into the destination
(`render.Surface.FillPatOrGray`), which turns half the pixels black, leaves the other half
alone, and cost nothing on a 68k.

**It is not sufficient for text.** The first version of the menu panel was exactly that dim,
and it was very nearly illegible: the shipped splash art is a bright yellow wall, so cream
letters sat on a checkerboard of black and bright yellow whose light half is as light as the
letters. The inverse-video selection bar on the same panel was perfectly readable, which is
what identified the cause — a solid ground, not a brighter ink.

So panels are a solid `Black8` interior with a cream frame, and the dim survives as an
eight-pixel halo around the box, which keeps the frame from sitting on the picture with a hard
edge. That is also the game's own idiom rather than an invention: the scoreboard fills its band
black and writes cream in it (`render/scoreboard.go`). The general rule, for 1.7b's settings
screen and 1.7c's boards: **the dim is for de-emphasising artwork, never for backing text.**

Two consequences worth writing down. Anything drawn over the splash has to be *placed* rather
than composited, so every panel's rectangle is a named constant chosen against the shipped art
— which is why the menu sits over the sky at the right rather than over the illustration's
subject, and why the port's own fallback title screen is laid out around that same rectangle.
And a magnified bitmap font is the only text this port has, so legibility is bought with
`scale` and the original's one-pixel black drop shadow, not with a lighter weight.

### 2.51 A disabled control under the cursor must not look like the one to press — **DONE, 1.7a**

The original's answer to an unavailable command is a menu item drawn in grey and a keystroke
that does nothing at all, which leaves a player with no idea what is wrong. The port's shell
does two things about that: pressing an unavailable item writes the reason on the status band
("no houses in … — run `make assets`"), and the item is drawn in grey.

The grey was not enough on its own, because the shell also has a *cursor*, and the cursor
starts on "New Game". On a machine with no houses — a fresh clone, which is the first screen a
stranger sees — a filled inverse-video bar sat under the one item that cannot work, and an
inverse-video bar is this shell's "Return does this". An unavailable item under the cursor now
gets an outlined bar instead of a filled one.

The general rule for the rest of 1.7: **every state has to be distinguishable from every other
state it can be confused with, not merely rendered differently.** Enabled-and-selected,
disabled-and-selected, enabled-and-not, disabled-and-not is four states and needs four
appearances. 1.7b's settings screen has more of them than this menu does (a setting can also be
unavailable *because of another setting*), and 2.28's pause overlay is the same problem in the
game.

### 2.52 The About box is the only documentation a player gets, and it was describing the wrong game — **DONE, 1.7a**

Its first draft listed the original's keys. Three of this port's bindings are not the
original's — player two is on A/D/W/S because the originals are modifier keys a window manager
intercepts (2.3), Tab pauses, and Escape ends a game rather than the program (2.7) — so the
1994 manual is actively wrong for this build, and the About box was repeating it.

It now lists what this build actually does, per player. That fixes the immediate error and
creates a maintenance hazard in its place, which is the part worth recording: **the list is a
Go string literal and the bindings are data** (`player.Glider`'s key set). The moment 1.7b makes
them configurable the two will disagree, and a wrong key list is worse than none. 1.7b must
generate the list from the bindings — the same argument as 4.4's dangling-test linter, one step
earlier.

**1.7b generated it.** `drawAbout` reads `Shell.host.Prefs` and formats each player's four keys
from it, falling back to `prefs.Default()` when there is no host — which is what `-shot` and the
tests see, and is still true of a fresh install. So the box is right for whoever is holding the
keyboard, including a player who has rebound everything, and the only literal left in it is the
sentence structure. The two unbindable lines moved with the same change: it now says
"Tab or Esc pauses" and "Q while paused gives up the game", which is what 2.7 made true.

### 2.53 A measurement that reads the player's settings is not a measurement — **DONE as a rule, 1.7b**

The moment there is a preferences file, every golden image and every benchmark in `make check`
depends on the machine it runs on. A developer who has turned the volume down, picked a
one-room view or magnified the window would get different PNGs and a different frame rate from
everyone else, and the difference would look like a regression in whatever they were working on.

The rule `cmd/glidergo` implements is that **the four measurement modes start from this build's
defaults and cannot save**: `-shot`, `-frames`, `-bench` and `-dump` (`hermetic()` in
`cmd/glidergo/prefs.go`). That is one function and no special pleading at any Makefile call
site, which matters because the alternative — `-prefs none` on eight lines of the Makefile — is
a rule nobody can see being followed.

Three edges are deliberate and each is pinned by a test in `cmd/glidergo/prefs_test.go`, the
only tests in that package:

- **An explicit `-prefs <file>` beats even a measurement**, because that is how a golden image
  of the settings screen gets settings to show. Without it, 1.8's screenshot of that screen
  would silently draw the defaults and compare nothing.
- **`-house` is *not* a measurement.** Naming a house is asking to play, and a player who plays
  that way wants their own bindings — so hermeticity follows the *mode*, not the presence of a
  flag.
- **`-prefs none` reads and writes nothing**, so a run on a machine that has a settings file can
  be made reproducible from the outside, and cannot create one either.

The general rule for the rest of the project: **anything that reproducible output depends on has
to be a property of the invocation, never of the machine.** 1.8's corpus is the next thing this
binds; the seed (`-seed`), the house, the room and the window size are already in that category
(2.42).

### 2.54 The pictures drawn over a running game are the house's, not the application's — **DONE, 1.7b**

`render.Assets` had two resolution orders and 1.7b needed a third. `Pict` consults the open
house's resource fork first and records a missing file as a fault, which is right for a room
background. `UI` looks only at the application and stays silent when a plate is absent, which is
right for the shell — a house that could repaint the title screen could hide the way out of it
(`TestUIIgnoresTheOpenHouseFork`), and a checkout with no art still has to reach a menu (2.6).

Neither is right for the pictures drawn *over a running game*: the pause placards, the
stars-remaining panels and the banner plates. On a Mac the open house's fork really does sit in
front of the application's for those ids, and the shipped houses use it — **Teddy World carries
its own PICT 1015 and 1016**. So `Assets.Plate` is the third order: the fork first, then the
application, and silence when neither has it, because these have a drawn fallback.

Counted on disk while settling this, and it is what 1.7d needs before it starts: **thirteen** of
the twenty shipped houses carry their own 1991-1993 (the banner plates) and **four** their own
1017 or 1018 (stars remaining). So `BringUpBanner` and `DisplayStarsRemaining` go through `Plate`
as well, and a port that reached for `UI` there would look right in every house nobody had
customised.

### 2.55 Nothing may destroy the player's settings, including the game itself — **DONE as a decision, 1.7b**

The settings file is the one thing on disk that is about the *player* rather than about the game:
eight bindings somebody chose and got used to. Three decisions protect it, and each is a
deliberate refusal of a more convenient behaviour:

- **Saving happens at the change, not at quit.** The original writes its preferences once, in
  `WriteOutPrefs` at quit (`Main.c:381`), so a crash loses everything the player set that
  session. This port saves in exactly two places — the settings screen closing, and `runShell`
  noticing a different house was chosen — which costs one small file write and cannot lose one.
- **`-import-prefs` refuses to overwrite.** Converting a 1994 `Glider Prefs` file into a
  destination that already exists is an error, not a backup and not a prompt: there is no prompt
  to give (it runs before any window opens), and telling somebody to move their file aside is
  cheaper than a backup scheme they would have to learn.
- **A session with nowhere to write says so on the way out**, rather than pretending: closing the
  settings screen with no saver shows "settings changed for this session only — there is nowhere
  to save them". A screen that silently discarded a rebind would be indistinguishable from one
  that had saved it.

A related rule that costs nothing and prevents the worst case: a preferences file that will not
load, or a configuration directory that does not exist, leaves *usable* settings in hand and a
line on stderr. Refusing to start a game over a file the game itself wrote is not an acceptable
failure mode (`prefs.Load`, `loadPrefs`).

### 2.56 The high-score screen's way out is written in blue on a starfield — **note; the fix is a 2.x option**

`DrawHighScores` writes its footer — the original's `Click Mouse or Hit a Key to Exit`, `STR# 150`
index 8 — in `QDBlue` (palette 211) at baseline 308, over PICT 1995, which is a near-black star
field (`docs/analysis/scoring.md` 7.9.2). It is the least readable text anywhere in this port, and
it is the one line on the screen that tells a player what to do. Transcribed exactly anyway,
because 1.8's corpus measures this port against the original and a colour changed here would be a
corpus measuring the port against itself.

The fix is one palette index and belongs with the other legibility work in 3.3: either the cream
the plaque's own title uses (8) or plain white, behind the same "modern defaults" switch that 2.19,
2.20 and 2.39 sit behind. Worth doing before the first public build, because the screen is the last
thing a player who just earned a score looks at.

### 2.57 Every board stores ten timestamps and nothing has ever drawn one — **note; a layout change, so 1.8 at the earliest**

`scoresType` carries `TimeStamps[10]`, `unsigned long` Mac epoch seconds, written whenever a score
is inserted and read by nothing: `DrawHighScores` shows name, score and rooms and drops the date
(`docs/analysis/scoring.md` 7.5, 7.9.2). So the shipped houses carry dates no player has ever seen,
and decoding all twenty non-empty boards turns out to recover the authors' own playtesting:
`The Asylum Pro` 1995-06-08, then a run of fourteen houses played between 1995-07-03 and 1995-07-28,
`California or Bust!` on 1995-09-08, `Art Museum` on 1996-01-13, `Davis Station` on 1996-10-17, and
two stamped 2000-05-11 — Slumberland's top two rows that morning and the whole of `Sampler`,
39 seconds after the save that created it (1.2). Thirteen of the twenty are topped by `Ozma`, two by
`Paul` — and those two, ImagineHouse PRO II and Grand Prix, are exactly the two houses Jonathan Chin
signs as Paul Finn in their banners, so the boards are a third, independent trace of that pen name.

This port keeps writing them, because the file format is the original's and a side-car that dropped
a field could not be read back into a house. Drawing them is a different matter: the plaque is 332
pixels wide and already carries three columns, so a date column means a new layout, which means the
corpus 1.8 is about to record would be recording it. Deferred rather than declined — a high-score
board that can say "1995" is a better artefact than one that cannot, and this one has the data.

### 2.58 The name dialog appears in the corner of the screen — **note; a 1.8 option**

`BringUpDialog` has its `CenterDialog` call commented out (`GliderPRO/Sources/DialogUtils.c:26`),
so DLOG 1020 appears at its stored bounds — `(0,0,109,316)`, the top-left corner — while DLOG 1021,
whose stored bounds are `(40,40,162,356)`, appears very slightly inset. Two dialogs in the same
sequence, one in the corner and one not, because one resource happened to be saved in a different
place (`docs/analysis/scoring.md` 7.11.1).

Both are reproduced exactly, and `internal/scores`' own test asserts both positions. Centring them
is a one-line change per dialog and it is the right change for a release; it is listed here so that
it is made deliberately, with the corpus regenerated, rather than as a tidy-up.

### 2.59 A mask's resource id is not "the picture's id plus 1000" — **note; matters to every masked draw**

The convention `docs/analysis/scoring.md` 2.8 states — `kPointsPictID` 4006, mask `PICT 5006` — is
real but not general. The masks actually shipped are:

| picture | its mask | offset |
|---|---|---|
| 4006 (points numerals) | 5006 | +1000 |
| 4002 (bonus numerals) | 5002 | +1000 |
| 1994 (high-score plaque) | **1998** | +4 |
| 1019 (the angel) | **1020** | +1 |
| 1990 (the GAME OVER pages) | **1989** | **-1** |

So any code that derives a mask id by arithmetic is wrong for three of the five, and the one it is
most wrong about is the game-over animation, where the mask id is *lower* than the picture's. Every
pairing in this port is written out at the call site instead (`internal/scores`' 1994/1998, and
1.7d's 1019/1020 and 1990/1989 when it gets there). Recorded because "+1000" reads like a rule and
is a coincidence.

### 2.60 The two game-over paths disagree about whether a high score suppresses the splash redraw — **DONE, 1.7d: both paths now ask**

`TestHighScore()` has exactly two callers, and they use it differently
(`docs/analysis/scoring.md` 8.7 and 9.8):

| caller | context | return value | `RedrawSplashScreen()` |
|---|---|---|---|
| `GameOver.c:67` | the win ending | used | only when no score qualified |
| `GameOver.c:503` | the loss ending | ignored | always |

The win path is the considered one: the high-score screen has already covered the window, so
redrawing the splash under it would be a wasted blit and a visible flash. The loss path does it
unconditionally, which means the board a player just earned is drawn over the splash and then the
splash is drawn over the board — except that `RedrawSplashScreen`'s copy runs in the wrong direction
and draws nothing (7.8, and 2.56's cousin), so on a real Mac neither is visible and the bug hides
the bug.

1.7d ports both animations and both endings, and it should port the *win* path's arrangement to
both: ask, and skip the redraw when the answer is yes. The high-score hook this stage installed
already returns exactly that Boolean.

**Done, and it is the only place in 1.7d where the port chose the better of two shipped
behaviours rather than transcribing one.** `internal/game/gameover.go`'s loss path is
`if !w.TestHighScore() { w.restoreSplashScreen() }`, the same line as the win path;
`TestDoGameOverSkipsTheSplashWhenAScoreQualifies` pins the decision. The C's unconditional
redraw is unobservable on a Mac for the reason above, so nothing a player of the original saw
has changed.

### 2.61 The win animation writes three rects outside its array, and seeds five it never reads — **DONE as decisions, 1.7d**

Two defects in the same forty lines of `GameOver.c`, found while transcribing them, both
recorded at length in `docs/analysis/progression.md` 11.2 and in the port's own comments:

* **`DoGameOverStarAnimation` writes `pages[-3]`, `pages[-2]` and `pages[-1]`.** The angel
  starts at `left = -96`, a star is seeded whenever `left % 32 == 0`, and `which =
  left / 32 % 5` keeps the sign of the dividend in C — so the first three seedings write three
  rects below a file-scope array. Nothing reads them (`count` cannot rise for a negative
  `which`), which is why the game shipped. The port skips the write and keeps the chime, which
  the C plays outside the branch; the ending therefore still opens with three sounds and no
  star.
* **`SetUpFinalScreen` seeds five star positions that are dead.** `dest` and `was` are both
  reassigned by the animation before it can draw a slot; only `frame = RandomInt(6)` survives.
  The port keeps all fifteen `RandomInt` calls regardless, because the generator is shared with
  every flame phase and pendulum in the game and dropping ten draws would change what every
  room composed afterwards looks like.

Neither is an "improvement" a player can see, and that is the point of recording them: both are
the sort of thing a later reader would take for a transcription slip and "fix", and one of the
two fixes would silently move the whole random stream.

### 2.62 The quit path repaints the work map with a title screen nothing can display — **DONE, 1.7d: dropped, and the harness gets an invariant back**

`NewGame`'s tail (`Play.c:257-273`) ends a game the player quit by scaling the splash art into
`workSrcMap` and invalidating the window, so the Mac's next update event paints the title screen
from it. Transcribed literally, that is 294,400 pixels written after the last frame — and in this
port not one of them is reachable. The title screen belongs to `internal/shell`: it is composed
on the shell's own surface and presented on the pass after `NewGame` returns, so nothing reads
the game's work map again. A `-dump` run cannot see them either, because the restore presents
nothing and a dump writes on `Present`.

Dropping it is worth more than the pixels, because it is what lets the fidelity harness say
something sharp. `internal/replay`'s `Result.Planes` hashes the three planes *as the last frame
left them*, and the useful consequence is that in a room where nothing animates the work map must
equal the background map — every animated thing registers a back rect and every back rect is
restored. Anything that draws into the work map without registering one shows up as a hash
mismatch, which is exactly the class of bug that is invisible for one frame and then smears
(`replay_test.go`'s room-70 case). A teardown that repaints the whole work map collapses that
check: every script, in every room, ends with the same splash hash. This was live for one stage
— `restoreSplashScreen` was a stub until 1.7d made it real, and the moment it did, two rooms of
the 1200-frame corpus started reporting the identical work plane.

The two ending paths keep their call. There the splash is on screen a moment later, because
`DoGameOver` composes the starfield over it and presents (`gameover.go:140`), so the pixels do
what the C wanted them to do.

### 2.63 The demo recorder can silently kill the demo it is recording — **DONE, 1.8b**

A `'demo'` record is `{long frame; char key; char padding}` and playback is
`if (gameFrame == demoData[demoIndex].frame)`, then `demoIndex++` — an **equality** test, not a
"have we passed it" test. So two records carrying the same frame number strand the cursor: the
first is consumed, the second never matches any later frame, and every record behind it is
unreachable. The demo simply stops responding partway through, with no error, no truncation and
nothing on screen to say it happened.

The original's own recorder can produce that file. `LogDemoKey` is called from inside `GetInput`'s
branches, and two of the four calls sit *above* the test that would have suppressed a second log
on the same frame: the band call is above `if (!fireHeld)` (`Input.c:348`), and the right-key call
is above the both-keys test (`:304`). Nothing anywhere in the C checks the stream it just wrote.
The shipped resource happens to be clean — 1,117 records, 1,116 gaps, none of them zero — which is
luck and the reason the bug survived.

Three places in the port take it seriously, because a format whose failure mode is silence needs
its check somewhere a human will look:

- `demo.Recorder` refuses a duplicate frame and counts it in `Dropped`, so recording cannot
  produce a stream that kills itself.
- `Stream.Validate` rejects one, and `Encode`/`WriteFile` validate **on write and not on read**:
  anything that came off a 1994 disk loads, and anything this port creates is checked.
- `glidertool demo check` is the command that says so out loud, and `demo info`'s status column
  says it for a whole directory at once. `TestCursorRepeatedFrameStalls` pins the stall itself, so
  the port's tolerance of a bad stream is deliberate rather than accidental.

What is *not* changed: playback still compares for equality. The refusal is at the writing end,
where the original's bug was, and a hand-made stream that stalls still stalls exactly as it did in
1994 — that is the behaviour, and the tools now name it instead of the player discovering it.

### 2.64 A guard that fires every frame buries the guards a report is about — **DONE as a named exception, 1.8b**

2.33's rule is that every out-of-range read the C performs is counted every time it happens, and
that rule has been right for all fifteen sites but one. `demoData[demoIndex]` past the last record
(`Input.c:224`) is not an event, it is a *state*: once a demo outlives its recording it reads off
the end on every frame until the glider dies, which for the shipped stream would be hundreds of
identical `Deviation`s. `Diag.Seen` keeps the first sixteen distinct kinds, so a long demo would
fill a bug report with one finding repeated and push out the room-object and trigger guards the
report was actually opened about.

`demoKey` reports it **once per run**, on the first refusal, and `Cursor.PastEnd` carries the real
count for anyone who wants it (`glidertool replay` prints it in the summary). The exception is
written down in three places — the call site, `devDemoRecord`'s comment, and `badIndex`'s doc,
which now says that one caller breaks its rule — because a diagnostic convention with an
undocumented exception is worse than one with none.

The general shape is worth keeping in mind for later stages: any guard inside a per-frame loop
whose condition persists needs a *first-occurrence* report and a counter, not one event per frame.
This is the first such site; Stage 3's network loop will have more.

### 2.65 Touching the controls during the attract mode must give the player the game back — **DONE (the original was already right), 1.8b**

`BUILD_ARCADE_VERSION` is on in the shipped build, and its block at `Input.c:192-201` says that
any of player one's four game keys ends a demo: `playing = false; paused = false`. That is the
behaviour a 2026 player expects and would complain about the absence of, and it is worth recording
as a thing the original got right, because the port had to choose which half of a `#if` to
transcribe and the other half — where only Command-Q and Command-S do anything — would have left
a player watching a ghost fly with no way to interrupt it but the mouse.

Two details of *how* it ends are the kind of thing a port would tidy away by accident. It does not
`return`: the frame's recorded input is still applied and the pause key still tested, because the C
only sets the two flags and falls through, so the demo ends one render later and not on the
keypress. And the two Command keys are genuinely dead in this arm, so the arcade build cannot quit
or save from the attract mode at all. `TestAGameKeyAbortsTheDemo` pins both — the run stops within
two frames of the press, having consumed a strict prefix of the records, and `GameOver` is
**false**, because what stopped is the demo and not a game.

The polish this leaves for the shell (1.7's screen, still optional): `DoDemoGame` re-arms
`incrementModeTime` on the way out, which is the original's way of making sure a demo that has
just ended does not immediately start another. A release should also not start one while the
player is part-way through choosing a house — the idle timer measures idleness, and a house picker
with a cursor in it is not idle.

---

## 3. Things the original did not have and a 2026 release is expected to have

### 3.1 High scores — **DONE, 1.7c**

The original does keep them (`internal/house.Scores`, and the board sorts on rooms
visited, not points). A release needs them persisted somewhere sane — XDG
`$XDG_DATA_HOME/glidergo/` on Linux, not next to the binary — and it must not corrupt or
crash on a truncated or hand-edited file.

**1.7c delivered all three.** `internal/scores.Store` writes one side-car per house under
`$XDG_DATA_HOME/glidergo/` (`-scores` moves it; `-scores none` plays without recording), the 22
shipped houses are read and never written, and `Store.Load` never fails: a truncated,
hand-edited or half-restored file yields a playable board plus one note per thing that had to be
worked around, on stderr. The board is reachable from the menu as well as by dying (H, which is
the original's Options > High Scores), and the game asks for a name and — for first place — a
banner, in the original's own two dialogs.

Two things a release still wants, both noted above rather than done: the footer is illegible
(2.56) and the dates the board already stores are never shown (2.57).

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

### 3.4 There is no way in to the game — **DONE, 1.7a; the credits DONE, 1.7c**

No title screen, no house picker, no options screen, no credits. 1.7 is scoped as "the
shell" and owns all of it. Flagged here because the credits screen is not optional: the
five house authors in item 1.2 and both illustrators must be named in the shipped build
regardless of how the asset question is resolved.

**1.7a delivered the way in**: `glidergo` with no arguments is now a title screen with a
menu, a house picker over all 22 houses (with each house's shipped high score and its room
count), and an About box. The options screen is 1.7b.

**1.7c finished the credits.** There is a credits screen — About, then C, which the About box
says on its last line — and it names John Calhoun, Casady & Greene, all five house authors, both
illustrators and which houses belong to whom. It is the only screen in the port with no original
to be faithful to: the 1994 game credits nobody anywhere a player can see.

It is data, not a screen full of string literals. `internal/credits` embeds `credits.txt`, and
that file is pinned against `GliderPRO/README.md` by its own tests — a name on the screen that is
not in the README fails, a house the README credits that the screen does not fails, and a house
moved to the wrong author's row fails. `internal/shell` then checks that every name in the data
reaches a placed line inside the panel, so a credit cannot be lost to a layout that outgrew the
screen. That chain is the point: an obligation discharged by a literal inside a drawing function
is one nobody ever diffs against its source.

### 3.5 The original could be translated and this port cannot — **note; Stage 2 at the earliest**

Every string the 1994 game shows a player comes out of a resource: `GetIndString(150, index)`
through `StringUtils.c:321-327`, with `STR# 150`'s 51 strings and `STR# 160`'s alongside it
(`docs/analysis/scoring.md` 7.9.2). That is not decoration — it is the standard Mac localisation
mechanism, and it means a translator could ship a Glider PRO in French by editing a resource fork
and touching no code. Index 6 is `room`, 7 is `rooms`, 8 is `Click Mouse or Hit a Key to Exit`.

This port hard-codes English at every site. That was the right call for Stage 1, whose contract is
fidelity and whose strings are mostly transcriptions with a resource id in a comment beside them,
but it is a capability the original had and this does not — and the longer it is left, the more
sites there are. The cheap version is one package with the shipped strings as data (the shape
`internal/credits` now uses for the credits) and a language preference; the expensive part is that
the port's own strings — the status band, the settings screen, the About box — are longer and more
numerous than the original's 51, and several are composed with `fmt.Sprintf` in an order a
translator may need to change.

Worth deciding at Stage 2 rather than later: the new houses will have names, banners and room names
of their own, and those are content a translator would also want.

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
any machine with no display, no sound card and no timing dependency.

```
glidertool replay -house "CD Demo House" -frames 600 -room 4 -where 420,20
glidertool replay -trace -o bug.trace bug.script
glidertool replay -digest bug.script        # one token, for "do we still agree"
glidertool replay -wav bug.wav bug.script   # ...and what it sounded like
```

**The audio is part of the format as of 1.6**, which is not a footnote either. "The sound cut out"
is exactly the class of report that is unfalsifiable without one: the trace's `snd=` column names
every sound the frame asked for and which of the three channels it landed on, the footer counts the
requests that were refused and the ones that were cut off mid-sample, and a second digest covers
the mix itself — every sample of every channel, the priority policy's choices and the clip point.
`-wav` writes the same samples to a file, which on a build host with no sound card is the only way
to *hear* a replay at all. Two properties make the pair trustworthy: the recorded path mixes
exactly `SamplesPerFrame` per frame from an engine with no clock, so two runs agree byte for byte;
and the mix digest is a checksum of the WAV payload, so `sha256sum` on the file anybody can produce
answers to the sixteen characters in the report.

Silence is a *different run* and the trace says which it was in its header. A sound trigger whose
sound does not load gets no hot spot at all, so `sound off` can change the composition — and even
where it does not, it changes the digest, because a sound request is a decision the simulation made
at a frame. A port that stopped playing the toaster would otherwise pass every determinism test in
the suite.

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

### 4.3 The golden trace reported the digest as the first divergence — **DONE, 1.5d; the second half is 1.8**

`duct.trace`'s line 4 is a digest of all 601 samples, so it changes whenever any sample does.
The failure report printed the first differing line, which therefore named line 4 on every
single failure and buried the frame that actually moved. In 1.5d that cost real time: the first
reading of the divergence was "the digest changed", and the event — a switch thrown on frame
224 — was 225 lines further down. The report now skips comment lines when choosing the
difference to print, counts them separately, and says so explicitly in the one case where only
the header moved, which means the digest function changed rather than the game.

**The second half is a design limit and is owed at 1.8.** One 600-frame script can only pin the
subsystems the glider happens to visit, and bringing an object to life can move the glider. That
is exactly what 1.5d did: room 5's ceiling switch arms a transporter, so from frame 225 the
trace is in room 70 and six of the toaster's nine bursts are gone from it. Nothing was lost this
time, because the toaster's arithmetic is pinned closed-form by `TestLaunchVelocityFromHeight`,
but the next stage that animates something may quietly walk the trace away from whatever it was
illustrating. 1.8 should carry several short scripts, one per subsystem, instead of one long one
— and the general rule is the one 2.38's survey also taught: a test that follows the game rather
than asserting about it has to be able to say what it stopped covering.

### 4.4 A comment naming a test that does not exist is worse than no comment — **DONE, 1.5e and 1.6; the linter is 1.8**

`internal/render/srcrects.go` had promised since 1.5c that "`TestStripRectsTileTheirSheet`
checks each array against its sheet's declared bounds, and a transcribed table is what that
test can actually catch a mistake in". No such test existed. That promise was the stated reason
the nine frame-strip rect tables were written out longhand instead of computed from a stride —
so its absence made the choice pointless: a hand-typed table with nothing checking it is
strictly worse than a loop.

Written in 1.5e rather than deleted, because 1.5e added `BandRects` to exactly the set of tables
the comment describes. `internal/render/srcrects_test.go` now derives each expected rect from
`stripBounds` — transcribed from `StructuresInit.c`'s `QSetRect` calls — while the tables under
test come from its *loops*, so the two independently-transcribed sources have to agree and a
single typo cannot satisfy both. `TestEverySrcRectFitsItsSheet` adds containment for the
irregular tables, which catches the failure a golden image hides: `Surface.Copy` clips silently,
so a rect past the end of a sheet gives a short or empty blit rather than a panic, and an object
that animates to nothing looks like a logic bug in its handler.

One more of the same class was found and fixed at the same time: `bands.go`'s header named
`TestClampedBandSurvivesTheKillTest` where the test is actually
`TestBandBouncesOffTheWallAndSurvivesTheKillTest`.

**The sweep was run by hand at 1.6 and found two more.** This codebase's comments carry a lot
of load — most of what is known about the original lives in them — and a named-but-absent test
is a specific, mechanical failure: it tells a future reader that a property is pinned when it is
not, and it does so most convincingly exactly where the property is subtle. Three lines of
`grep | comm` over the whole tree:

```bash
grep -rhoE "\bTest[A-Z][A-Za-z0-9]+\b" --include=*.go internal cmd | sort -u > /tmp/mentioned
grep -rhoE "^func (Test[A-Z][A-Za-z0-9]+)" --include=*_test.go internal cmd | sed 's/func //' | sort -u > /tmp/defined
comm -23 /tmp/mentioned /tmp/defined
```

- `cmd/glidertool/glidertool_test.go` named `TestCheckFindsUndefined`; the test is
  `TestCheckFindsUndefinedWhat`. Comment corrected.
- `internal/game/grease_test.go` and `internal/game/bands_test.go` both deferred to
  "`TestRenderFrameOrder` in play_test.go", which had never been written — and it was the one
  worth having, because both of those tests explicitly decline to pin the position in the
  sequence on the grounds that something else does. Written, in `play_test.go`: it parses
  `render_frame.go` with `go/parser` and asserts the fifteen receiver calls in `RenderFrame`
  come out in Render.c's order, that the two per-player calls are P1 before P2, and that
  flames and stars are the two arms of one `if` on `EvenFrame` rather than two statements.
  A source test rather than a behavioural one because registration order *is* z-order and the
  behavioural equivalent needs a room with a mirror, grease, a pendulum, a flame, a dynamic, a
  flying point, a sparkle, a shred and a band overlapping at once.

**The linter is still owed at 1.8**, because the sweep above is a thing someone has to remember
to run. `glidertool` should grow a `lint` subcommand that does it, run from `make check`. Three
false-positive shapes this tree already contains, which are the whole design problem — a naive
`\bTest[A-Z]\w+\b` matcher reports all three:

| Shape | Where | Why it is not a dangling reference |
|---|---|---|
| A wildcard standing for a family of tests | `internal/house/house.go:12` says `TestCorpus*` | Matches the eight `TestCorpus…` tests in `corpus_test.go` |
| A hyphenated line break | `internal/render/locale.go:101` ends a line with `TestFrame-` | Continues `CountsMatchTheSrcTables` on the next line, and that test exists |
| The C's own identifier | `internal/game/play.go:208` cites `TestHighScore` | `HighScores.c:374`, not a Go test |

So the check has to skip a match followed by `*`, **join** a comment's lines before matching —
not merely ignore a trailing `-`, since a hyphen split hides a dangling name exactly as well as
it hides a valid one — and exempt names that appear inside a citation of the original source.
The last one is the interesting one: `TestHighScore` will become a real Go test's name at 1.7,
at which point the false positive resolves itself and the exemption stops being needed — which
suggests the rule should be "a name cited alongside a `.c` filename is the C's", not a
hand-maintained allowlist.

### 4.5 A bug report could not say that two machines drew different pixels — **DONE, 1.5f**

`Result.Digest` hashes the sample trace, and the trace records dirty-rect *counts*. That is the
right choice for the reason 4.2 gives — a script mailed in by a player has to survive the next
sub-stage's intentional pixel changes — but it means an entire class of fault is invisible to it.
A flame stuck on one cel, a pendulum swinging the wrong way, a filmstrip baked with the wrong
frame masked into it, a confetti cloud emerging top-first: every one of those registers the same
rect as the correct version, so `w2m` and `b2w` do not move and neither does the digest.
Everything 1.5f added is in that class by construction.

`Result.Planes` closes it with three hashes taken once, after the last frame: the back map, the
work map and the screen. Which of the three moved localises the fault before anyone opens a
debugger — `back` is the composition, so a difference there is a locale drawn differently before
anything moved; `work` is the composition plus everything that animated over it; `main` is the
work map as the dirty rects delivered it, so a difference in `main` with `work` identical is a
dirty-rect bug, the right pixels composed and the wrong ones copied. `glidertool replay` prints
the three on a `screen` line beside the digest.

They are deliberately **not** folded into the digest and not written into the trace, which is
what makes it safe to have both: nothing on disk holds the value, so a deliberate pixel change
costs nothing, while two runs of one script are still required to agree.
`TestTwelveHundredFramesTwiceAreTheSamePicture` is 1.5f's acceptance clause and asserts exactly
that, and it picked up two facts about the frame protocol on the way: a room where nothing
animates ends every frame with the work map identical to the composition (so a renderer that
forgets its back rect fails there), and a room with a cuckoo in it never does, because the
animated families' cels come out of a filmstrip and register no back rect at all.

### 4.6 The screens a player meets first were the only ones no automated check could draw — **DONE, 1.7a; the corpus half in 1.8a**

Every check in `make check` covered the game and none of them covered the way in, and the reason
was structural rather than an oversight: the game's frames come out of the null backend
(`-frames 3 -dump`), and a title screen drawn on the null backend still needs a backend. So
`internal/shell` was built to compose into a surface with no window anywhere in the picture, and
`glidergo -shot FILE -shot-screen splash|houses|about` writes one frame of it as a PNG with no
display, no audio, no house and no game opened.

`make headless` now renders five: the three screens against the real asset tree, and the two
first-run layouts against a directory that does not exist (2.6). It is cheap — no process
outlives the write — and it caught three defects in one sitting that the unit tests could not,
because all three were about what the composite *looks* like rather than what any one function
returns: the illegible dim (2.50), the fallback title screen laid out underneath the menu panel
(2.6), and the selection bar on a disabled item (2.51).

Two notes for the stages that inherit it. The unit tests still assert the things a test can
assert and should keep doing so — `TestEveryScreenDrawsWithoutArt` counts cream and black pixels
and fails on a blank screen, `TestHouseLabelPlacementAndClamping` checks that no ink reaches the
right edge — because a PNG nobody opens is not a check. And `-shot` is the natural place to hang
a golden-image test for the shell in 1.8, which would make these five files a corpus rather than
an artefact; it is deliberately not that yet, because 1.7b, c and d will all change these screens
and pinning them now would only generate churn.

### 4.7 Nothing on disk said what the game is supposed to look like — **DONE, 1.8a**

4.5 gave a report three plane hashes and 4.2 gave it a script that reproduces on any machine, and
between them they answer every question about pixels *except* the one a released game needs
answered: has this build changed since the build somebody looked at? `Result.Planes` compares a
run against another run, always both freshly computed, so a divergence introduced deliberately
and a divergence introduced by accident are indistinguishable — both simply become the new
answer. The only reference was human memory, and human memory of a 640x480 room is the thing
2.50, 2.6 and 2.51 all proved unreliable.

`internal/fidelity` is that reference: per-frame hashes of the three index planes, as text, one
row per frame, checked in. 601 rows for the existing 600-frame duct script, and six rows for the
shell's screens — which no replay script can reach, because `internal/shell` is a separate
program from the simulation by design. `make fidelity` runs it and, on a machine that has the
assets, treats a *skip* as a failure, because a silent skip is exactly the outcome this package
exists to prevent.

Four decisions in it are worth keeping:

- **Hashes, not images.** `git diff` on hashes says *which frames* moved, which is most of the
  finding: one row is a frame, every row is the palette or the view, three rows in the middle of
  a transition is a wipe that got faster. The same corpus as PNGs is 300MB of binary in the
  history of a repository a player is meant to be able to clone. What a hash cannot do is show a
  human the difference, so a failing test re-runs the one frame that diverged and writes it out
  as three PNGs — `Snapshot`, the failure path and only the failure path.
- **The recorder cannot move a pixel.** `internal/replay` grew one hook, `Watch`, and it copies
  the planes at each `Present` and hashes once per frame in `flush`. That ordering is the whole
  trick: `Present` fires up to 161 times inside a transition frame, so hashing there would cost
  161 sweeps of three planes to keep one answer and would pin the middle of a wipe as if it were
  the frame. A test drives the hook with a watcher that scribbles `0xFF` over everything it is
  handed and proves the trace, the planes and the mix all come out unchanged.
- **The shell's screens are hashed as index planes, not as PNG bytes**, or the corpus would be a
  corpus of Go's `image/png` encoder; and the version string the About box draws is pinned to a
  constant, or tagging a release would invalidate six rows for no reason. This closes the promise
  4.6 left open, one stage later than it guessed and by hashing the composition rather than the
  file `-shot` writes.
- **The end state is not the last frame.** Recording the corpus turned this up immediately and it
  had been true since 1.5f: after the frame loop, `PlayGame` runs the unconditional arcade block,
  which blackens the scoreboard band and blits it to the screen (`Play.c:551-593`), and
  `CopyRectsQD` has already restored `Back` over `Work`. So `Result.Planes` describes a picture
  that includes twenty rows no frame ever presented, over an erase the frame itself did not have.
  Neither number is wrong and both are worth having — `Planes` is what the process was left
  holding, the corpus is what the game showed — but a reader who assumed they were the same
  thing would have spent an afternoon on it. The tests now say so out loud.

What it does not close: 4.3's second half is still open, because one script still only pins the
subsystems this glider visits, and the demo-replay codec (`demoType`) is 1.8b.

**1.8b added the codec and deliberately did not add a corpus row for it.** The attract-mode script
runs 1,775 frames — 1,775 rows, about 320 KB — against a stream that lives in gitignored
`assets/extracted/`, so the corpus would be large, unbuildable from a fresh clone, and a
checked-in assertion that the physics gap of 2.18 is the correct behaviour. It is a determinism
test instead: the same script twice, compared frame by frame. The row becomes worth having on the
day the demo flies to the end of its stream, and it should be a short prefix even then.

### 4.8 The demo determinism test needs a 1994 asset, and it should not have to — **note; a 1.8c candidate**

`internal/replay/testdata/demo.script` is the strictest determinism test the harness has, and it
skips on a clone that has not run `make assets` — the stream it replays is an extracted resource
and `assets/extracted/` is gitignored. That is the right decision for *this* script, whose whole
point is the 1994 recording, but it means the demo *path* — the cursor, the equality compare, the
five missing guards of `GetDemoInput` — has no coverage at all on a bare clone.

The missing half is a round trip, and every piece of it already exists. `World.RecordDemo` returns
a `demo.Recorder` wired to `GetInput`'s four log sites, so a replay script with `at` lines can
*record* a stream; feeding that stream back through the `demo` keyword should produce the same
frames as the script that recorded it. It would need no asset, it would be a handful of records
rather than 1,117, and it would fail loudly on exactly the class of bug that is hardest to see
here: a one-frame offset between the frame a key was logged on and the frame playback applies it
to. The one thing it would *not* catch is a difference between `GetInput` and `GetDemoInput`, since
the recording came from the former — so it complements the shipped stream rather than replacing it.

Two known asymmetries have to be written into the test's expectations rather than treated as bugs
(both are 2.63's territory): an about-face records as a plain right press, because `LogDemoKey(0)`
is above the both-keys test, and a held band key records a code every frame although only the
first fires. So a recorded-then-replayed session is not guaranteed to reproduce the session — it
is guaranteed to reproduce *itself*, which is what a determinism test needs.

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
| 2.25 `Scene.RedrawCentralRoom`: a light switch repaints one room, not nine | 1.5c | this stage |
| 2.34 (outlet half) `HandleOutlet` names its destination instead of inheriting an ambient port | 1.5c | this stage |
| 4.3 (first half) The golden trace's failure report names the frame that moved, not the digest | 1.5d | this stage |
| 2.34 (grease half) `HandleGrease` names both destinations at the call site, back map then work | 1.5e | this stage |
| 4.4 `TestStripRectsTileTheirSheet` written, closing a comment that had promised it since 1.5c | 1.5e | this stage |
| 2.43 The fifth confetti cloud is dropped, and the refusal is reported through `badIndex` | 1.5f | this stage |
| 2.44 (the report half) `Scene.SavedMapDrops`, the `dr=` golden column, and the 4,070-room census | 1.5f | this stage |
| 4.5 `Result.Planes`: three index-plane hashes, so a report can say the *pixels* differ | 1.5f | this stage |
| 2.47 `FlushTriggerSound` resets the flushed channel's priority, so three trigger sounds cannot silence the game | 1.6 | this stage |
| 4.2 (audio half) the `snd=` column, the mix digest and `glidertool replay -wav` | 1.6 | this stage |
| 2.6 A fresh clone with no assets comes up on the port's own title screen and says which command produces them | 1.7a | this stage |
| 2.7 (the title screen's half) Quit on the menu, on `Q` and on Escape; Escape in a game returns here | 1.7a | this stage |
| 2.50 Panels are a solid ground with a cream frame; the 50% dim is a halo, never a backing for text | 1.7a | this stage |
| 2.51 An unavailable item under the cursor is outlined, not filled, and says why when pressed | 1.7a | this stage |
| 2.52 The About box lists this build's bindings instead of the original's | 1.7a | this stage |
| 3.4 (the way in) Title screen, menu, house picker over all 22 houses, About box | 1.7a | this stage |
| 4.6 `-shot` and five shell screens rendered by `make headless`, with no display | 1.7a | this stage |
| 2.1 (the setting) `scale` is remembered in the preferences file | 1.7b | this stage |
| 2.3 All eight bindings are the player's, and no glider steers with a modifier key | 1.7b | this stage |
| 2.5 `DoPause` is a pause *state* with PICT 1015/1016, not a blocked process | 1.7b | this stage |
| 2.7 (the in-game half) `Q` while paused gives up the game and comes back to the title screen | 1.7b | this stage |
| 2.15 Leaving a game no longer waits for a physical key release | 1.7b | this stage |
| 2.17 (the setting) `keep_real_time` declared, with the reason it is unwired on the field | 1.7b | this stage |
| 2.19 The mirror-room flame blink, as an opt-in fidelity switch | 1.7b | this stage |
| 2.20 The mirror's player-two foil, as an opt-in fidelity switch | 1.7b | this stage |
| 2.21 `doBackground` split into a fidelity switch and not a user option | 1.7b | this stage |
| 2.28 A paused game says so on screen | 1.7b | this stage |
| 2.32 (one of three) The pause no longer stops the world from inside a frame | 1.7b | this stage |
| 2.39 The switch's spurious corner sparkle, as an opt-in fidelity switch | 1.7b | this stage |
| 2.53 A measurement never reads the player's settings (`-frames`, `-bench`, `-dump`) | 1.7b | this stage |
| 2.54 `Assets.Plate`: the open house's fork first, then the application | 1.7b | this stage |
| 2.55 Settings are saved at the change, `-import-prefs` refuses to overwrite | 1.7b | this stage |
| 3.1 High scores: `internal/scores`, a per-house side-car, both entry dialogs, the board on screen | 1.7c | this stage |
| 3.4 (the credits) `internal/credits`, pinned against `GliderPRO/README.md`, on a screen | 1.7c | this stage |
| 2.32 (the other two) `internal/game/wait.go` and the `World.Wait` hook: no `Delay` anywhere | 1.7d | this stage |
| 2.60 Both endings ask `TestHighScore` before redrawing the splash | 1.7d | this stage |
| 2.61 The win animation's out-of-bounds seeding write, skipped; its dead `RandomInt` draws, kept | 1.7d | this stage |
| 2.62 The quit path's unreachable splash repaint dropped, so the replay corpus can check the erase pass | 1.7d | this stage |
| 4.7 `internal/fidelity`: per-frame pixel hashes checked in, so a moved pixel names its frame | 1.8a | this stage |
| 4.6 (the corpus half) The shell's six screens are compared against a reference, not just rendered | 1.8a | this stage |
| 2.63 The recorder refuses a duplicate frame, `Validate` rejects one, and `demo check` reports it | 1.8b | this stage |
| 2.64 `devDemoRecord` reports once per run, and `badIndex`'s doc names the exception | 1.8b | this stage |
| 2.65 The arcade abort is transcribed and pinned: any game key hands the player the game back | 1.8b | this stage |
| 2.18 (as far as it can be) The demo is a determinism oracle; the 1994 RNG was clock-seeded | 1.8b | this stage |

Five bugs found and fixed in the port itself while writing this, none of which is an
"improvement" so much as a repair, all recorded here because the reason no test caught
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
- **`HandleRewards` was missing the `kHelium` arm entirely** (1.5d;
  `Interactions.c:956-977`). A transcription slip, not a decision — the arm was dropped
  between `kStar` and `kSlider`, and nothing complained because a helium balloon is a
  `bonusType` like any other, so it built and ran and simply did nothing when touched. What
  identified it as a slip rather than a deferral is that two comments in the same file already
  counted *ten* restoring arms and *five* doubling arms, and both figures only hold with helium
  present. The recovered arm is the battery's mirror with three sign inversions — the `< 0`
  test, the `= -HeliumSupply` assignment, and a doubling that *subtracts* — and
  `TestBatteryAndHeliumShareOneSignedCounter` exists because getting any one of them wrong
  hands the player thrust where the balloon should have given buoyancy.

  The lesson for the rest of the port is the test that found it, not the bug.
  `TestEveryDispatchableRewardIsConsumed` does not list the reward types: it sweeps every
  object code through `CreateActiveRects`, collects the ones that produce a `kRewardIt` rect,
  and requires `HandleRewards` to consume each. A hand-written list would have had the same
  hole as the switch statement. Any other port function that dispatches on object type is a
  candidate for the same treatment.
- **`World.NewState` was a local, so every switch lever showed the wrong state** (1.5d).
  `SetObjectState` computes the state it writes into a file-scope global that `HandleSwitches`
  reads back to draw the plate, which is what makes a light switch show the *lamp's* new state
  rather than its own — a switch has no state of its own to show. With the value local, every
  lever drew from a zero, so a switch animated to "off" while turning a lamp on. Invisible
  without art, which is why `TestLeverShowsTheLinkedObjectsState` renders the plate twice and
  compares pixels, with a second art-free test on the field for runs where the assets are not
  extracted.
- **The mixer kept asking a finished game for the next piece of music** (1.7d's follow-on).
  `Engine.NextPiece` was assigned in exactly one place, `bindAudio`, and never re-pointed, and
  `NewGame`'s teardown ends every game by starting the idle score. So the music a player heard
  over the title screen was being walked by the `World` of the game they had just quit — and
  every game of a session stayed reachable from the engine and could not be collected. It was
  inaudible, which is why it survived: the cursor is a cursor wherever it lives, and the score
  sounded exactly right. What made it visible was writing the title screen's own score walk and
  having to ask what the engine was pointing at, which is the general shape of the thing —
  *nothing was wrong with the audio; something was wrong with who owned it.* Fixed by
  `startTitleMusic` re-pointing `NextPiece` at the title screen's cursor on the way back out of
  a game (`cmd/glidergo/music.go`), which is also what makes `music_on_title` audible at all.
