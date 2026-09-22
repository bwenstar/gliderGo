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

### 1.2 The art, the sounds and the 22 houses are on a different footing from the code — **settled: ship them. Route (a), decided 2026-09-16**

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

A third borrowed work, found in Stage 2 while fixing the About box to draw PICT 153 at all:
the same plate sets `...we have lingered in the chambers of the sea... T.S. Eliot` along its
bottom edge, from the closing stanza of *The Love Song of J. Alfred Prufrock* (1915), elided
at both ends. Upstream's README credits the Nemo strip and not the verse, so nothing in
this tree recorded it until now — the words were in the picture data and picture data is
invisible to `grep`. The poem is public domain on any reading, so this changes nothing about
the obligation; it changes the count, which is the point of the item. Transcribed at
`docs/analysis/ui-dialogs.md` 10.6 and credited in `internal/credits/credits.txt`, whose
`[the illustrations]` section is now `[borrowed from elsewhere]` because a poet is not an
illustrator.

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

The upstream repository does distribute the houses, so re-vendoring them under `GliderPRO/` is
no worse than what the copyright holder already does. Shipping the **decoded** art is a
further step: a new distribution of that art, in a new form, by someone who is not the rights
holder.

**That step has been taken deliberately.** The decision taken on 2026-09-16 is that gliderGo
works with no reference to the original game's source and that the assets are included, which is
route (a) below plus its natural conclusion: `assets/extracted/` is committed (15.5 MB,
1,877 files), so a clone plays with no extraction step, no python3 and no copy of Glider PRO.
The reasoning is upstream's own precedent — the copyright holder distributes the same bytes in
the same repository, in encoded form — and the practical fact that a game whose first
instruction is "now go and decode the assets" is not a game somebody can play.

What that decision does *not* do is answer the licence question, and it makes route (c) worth
more rather than less: an explicit grant from John Calhoun would move gliderGo from "no worse
than upstream" to "unambiguously licensed", and it is the only route that does. It is also the
one thing here that cannot be done from this machine.

**On 2026-09-17 the 1994 C source stopped being redistributed here.**
Sources/, Headers/, `Prefix.h`, the two CodeWarrior project files and `CarbonLib` were removed —
96 files. Note carefully that this *reduces* what the repository redistributes but does not
improve the position this item is about, and arguably narrows the argument above: what was
removed is the part upstream's grant covers beyond doubt, and what remains is the art, the
sounds and the 22 houses, which is the part it does not mention. The "no worse than what the
copyright holder already does" reasoning still holds for those, because the 41 files kept are
byte-for-byte upstream's, in upstream's layout. Route (c) remains open and remains the only
route that settles anything.

**This item used to claim the architecture was already right by accident. It was wrong, and
the correction matters more than the original claim did.** What it said was that `.gitignore`
excludes `/assets/extracted/`, so no original art is in this repository and none can reach a
release tarball built from it. The first half is true and irrelevant; the second is false.
`.gitignore` keeps *derived* files out of git. The originals are **tracked**:

- `GliderPRO/Glider PRO.r` — 15,475,666 bytes, the entire resource fork: every PICT, every
  `snd `, the `'demo'` stream, the lot.
- `GliderPRO/Houses/*.binhex` — all 22 shipped houses, 34 MB.
- `GliderPRO/Houses/*.mov` — the 15 QuickTime movies.

`git ls-files GliderPRO/` is 137 files and 50.7 MB of the 62.2 MB this repository tracks, and
`git archive HEAD` carries all of it. So
this repository already redistributes the art, and so does any archive, zip or GitHub release
tarball made from a clone, unless something is done to exclude it.

That is not a leak to be quietly plugged: it is exactly why a stranger can clone this
repository and play without owning Glider PRO (see 5.6). But it means this decision was live
*now* rather than at release time.

**The three routes, with the chosen one marked:**

- **(a) Keep vendoring — CHOSEN, and extended to the decoded assets.** Upstream distributes the
  same resource fork and the same houses, so re-vendoring them is no worse than what the
  copyright holder already does; release archives carry the assets and one download works. This
  is the honest description of what the repository is.
- **(b) Code only — rejected.** Add `.gitattributes` `export-ignore` for
  `GliderPRO/Glider PRO.r`, `GliderPRO/Houses/` and `assets/extracted/` so `git archive` and
  every release tarball drop them, then write the fetch step `make assets` would need — from
  upstream, or from the player's own copy of the game, the way a ScummVM or DOSBox front-end
  does. Rejected because it makes the first-run experience a scavenger hunt for a 31-year-old
  Macintosh game, and because the fetch step has to exist and needs somewhere to fetch *from*
  that is not this repository.
- **(c) Contact John Calhoun for an explicit asset grant — still open, and now the only
  outstanding part of this item.** It is the one route to content that is unambiguously
  licensed rather than merely no worse than upstream. Route (a) does not depend on it, and it
  would retire this item outright.

One consequence stands whatever happens with (c), because it changes the priority of a later
stage: **Stage 2's new houses are the only content gliderGo can ship without asking anyone.**
They stop being a nice extra and become the content the project owns outright. Stage 2 should be
planned as original work on that basis — not as imitations of the shipped houses, which would
inherit the same authorship question.

If (c) is ever answered with a no, the fallback is (b) applied to release archives only, which
is a `.gitattributes` change and a fetch script rather than a redesign — worth knowing, so that
route (a) is a reversible decision rather than a one-way door.

### 1.3 No `CHANGELOG.md` and none of the conventional repository files — **changelog DONE, end of Stage 1; the rest DONE, 2.0, bar the code of conduct**

gliderGo is a standalone repository: one `Makefile`, one CI workflow
(`.github/workflows/ci.yml`), no parent project, no external release machinery, and — as of the
end of Stage 1 — no reference anywhere in the tree to the private repository it was developed
inside. An earlier version of this item measured gliderGo against that repository's own
integration checklist, which does not apply to it and never did; the checklist item is dead
rather than outstanding.

What a public repository is actually missing:

- `CHANGELOG.md` — **done**, at the end of Stage 1: one section per stage, each naming the
  commit that closed it, with an `Unreleased` heading because there are no tags yet.
- A project page. Deferred until there is a release to link to; 5.4 owns tagging, versioning
  and artefacts. `VERSION` in the Makefile is `git describe --tags --always --dirty`, and with
  no tags at all it resolves to a bare short hash, so every build so far is stamped that way.
- `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`, and issue and pull-request templates:
  none exist. Cheap and conventional, but all four are about *other people*, and they should be
  written when the repository is actually public and there is somebody to address — recorded
  here so that their absence is a decision. See 5.7. **All but the code of conduct written at
  2.0**, plus a third issue form for fidelity differences, which is the report this project most
  wants; 5.7 records what each one says and why the covenant is still the odd one out.

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

### 2.14 A house whose custom trigger sound fails to load is silently silent — **the linter half DONE, 2.0; the in-game log is still open**

A house can carry its own 'snd ' resources for triggers. If one fails to load the original
plays nothing and says nothing, so a house author gets no signal that their sound is
broken. The house linter (4.1) should report it, and the game should log it once.

**2.0 did the linter half, and found that the silence has two causes that must not be reported
the same way.** `house lint` reports `sound-id` when the house names a `snd ` it does not carry
— 13 across the corpus, all real — and `sound-unreadable` when the house does carry it and
gliderGo cannot decode it (2.49). Both findings say the thing this entry had wrong: the failure
is worse than "silent". `LoadTriggerSound` failing makes `CreateActiveRects` compose the room
with no hot spot at all, so the trigger cannot be *touched*, and a house that used one as a
signpost has lost the signpost rather than its sound.

The in-game log is still open, and is smaller than it was: the author-facing signal now exists,
so what remains is one line on `stderr` for a player whose bug report says a trigger does
nothing.

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

### 2.16 The two-player idle freeze has no visual tell — **measured in 1.9; the tell itself is deferred to 2.x**

`TagGliderIdle` freezes a glider for 30 frames — a full second — with no indication that
this is deliberate. Player 2 in particular starts every two-player game idled and hidden
(`Play.c:198-203`), so the first thing a new player experiences is a second of not
existing. A fade, a shimmer, or anything at all would do.

**1.9 nailed down what it actually happens to, which was not what this entry assumed.** The
freeze is not player 2's alone: `ReadyGliderFromTransit` freezes whoever is *not*
`w.FirstPlayer`, and `FirstPlayer`'s zero value is `Player2` — so before anybody has waited on
anybody, it is **player 1** who arrives frozen. The countdown is stored in `g.HVel`, reused as a
counter while the glider cannot move. `game.TestTransitArrivalFreezesWhoeverIsNotFirstPlayer`
tables both directions, and `game.TestOnePlayerTransitDoesNotFreezeAnybody` pins that a solo
game never sees it.

**Why the tell is not in 1.9.** Anything drawn during those 30 frames changes pixels in rooms
that 1.8's fidelity corpus has already hashed, so the visual work has to land together with a
corpus re-record and a decision about whether the tell is a `fixes` flag (recordable both ways)
or an unconditional presentation change (corpus moves once, permanently). Both are cheap; neither
is free, and doing it inside 1.9 would have mixed a rendering change into the stage that pins
two-player *behaviour*. The behaviour is now pinned, so the tell can be added against tests that
already say what it must not disturb.

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

### 2.18 The random stream is unverified against real hardware — **premise disproved, 1.8b; the table test is 1.8c; the physics gap it exposed is open**

`internal/game/rand.go` transcribes the original's linear congruential generator, and the
demo replay in 1.8 depends on it bit for bit. It has not been checked against a trace from
a real Mac, and until it is, a demo that desyncs is ambiguous between "the RNG is wrong"
and "the input replay is wrong". 1.8 should capture a reference trace first.

**The premise above is wrong, and `toolbox-primitives.md` had already disproved it — 1.8b
confirmed the disproof by experiment.** The demo replay does **not** depend on the RNG. §1.12's
proof is in four steps: `GetDemoInput` dispatches on the frame counter and draws nothing; `Player.c`
contains no `Random` call at all, so trajectory is a function of (initial state, input, geometry);
Demo House contains **0** `kSparkle`, **0** `kCoffee`, **0** `kChimes`, `phoneBit` false — so none
of the RNG-consuming *registrars* even fires; and §1.13 enumerates every reader of the five arrays
the RNG feeds and finds nothing but `CopyBits` source-rect selection. R-RNG-2 is normative: a port
could replay the shipped demo frame-exactly with a `Random()` that returns 0 forever.

1.8b's replay is the empirical confirmation. Seeds 0, 1, 7 and 12345 produce the *same* run — the
same three deaths, the same 573 records consumed, the same end frame — and differ only in the pixel
digests, which is exactly the containment §1.13 describes. So the ambiguity this entry was written
about is gone: a demo that desyncs is the physics, not the RNG.

**And 1.8b's own note about clock seeding was wrong, which is worth recording because it inverts
the conclusion.** `ToolBoxInit` does `GetDateTime((UInt32 *)&qd.randSeed)` at `Utilities.c:61`, but
inside `#if !TARGET_CARBON` — and `GliderPRO/Prefix.h:1` sets `TARGET_CARBON 1`. The build this
source tree describes therefore never seeds: it starts from `randSeed == 1` on every launch
(`toolbox-primitives.md` §1.3, §1.4), which is why §1.6's verified table is a seed-1 table. The
clock seeding is the pre-Carbon 68k branch. So the attract mode *was* reproducible, the shipped
demo **is** a legitimate fidelity oracle, and the 573-of-1117 divergence below is a defect in this
port and not an artefact of a stream nobody can reproduce.

What is left of this entry is narrower and still real: `internal/game/rand.go`'s generator is a
reconstruction of a Toolbox trap whose code is not in the tree, and it has never been checked
against anything. §1.6 tabulates 24 verified draws from seed 1 — state, `Random()` as `int16`, and
`RandomInt` at the three ranges the game asks for — so the check is a table test, and 1.8c writes
it. That closes it as far as it can be closed without a Mac to ask; what remains open after that is
only whether Apple's trap really was Park-Miller, which §1.5 argues from the documentation and
§1.13 bounds the cost of.

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
improve, the test that fails should be a fidelity test and not a test about determinism. **That
reasoning is right and it had a hole, closed in Stage 2:** it left the number unguarded in the
direction that is unambiguously bad, so a change that dropped the glider to 300 records would have
gone green while this entry and two other documents went on claiming 573. There is now a one-sided
ratchet, `demoRecordsFloor` in `internal/replay/replay_test.go` — below it fails, at or above it
passes, and above it logs the line that says to raise the floor and names the three documents
quoting the old figure. A physics improvement still does not turn the determinism test red. And a
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

**1.8b establishes that the escape hatch is not needed, for this stream.** Demo House has no
sparkle and no coffee maker at all, so none of the three sites above fires during the shipped
replay — which is the house-specific half of R-RNG-2's proof. It stays written down because it
binds any *other* recording: a demo captured in a house with scenery that draws is only valid at
the window size it was captured at, and `glidertool replay`'s trace header carries the house, the
seed, the neighbourhood and the frame count for that reason.

**1.8c writes the table test, and finds that "the generator matches" was only a third of the
job.** `internal/game/rand_test.go` pins the seed-1 stream state by state against §1.6's 24
verified draws, pins `RandomInt`'s inclusive upper bound by constructing the raw word that reaches
it, pins its exact skew over all 65536 raw words, and shows that the `(16807*seed) % 2147483647`
an int32 transcription would naturally have written leaves the stream by the **third** draw — so
Schrage's split is load-bearing and not decoration. It also proves the containment from the other
side: `TestThePhysicsNeverDrawsFromTheRNG` walks `internal/game/player`'s AST and asserts the
package contains no call to `Random` or `RandomInt` at all, which is R-RNG-2's first step turned
into something that fails if a later stage breaks it.

The two-thirds that were not the generator, both of which a port gets wrong by default:

- **Scope.** `qd.randSeed` is a QuickDraw global, so in the original *one* stream spans every game
  in a process. A `World` seeded afresh per game replays the same "random" opening every time —
  the same candle phase, the same first telephone delay — which is the one way a fixed seed can be
  *less* faithful than a clock. `cmd/glidergo` now holds the stream on `app.randSeed` and reads it
  back out of the `World` after each game, the same hand-off `adoptScore` does for the music
  cursor.
- **The launch draw.** `VariableInit` (`InterfaceInit.c:160`, from `Main.c:321`) spends one draw
  before any game, on the editor's default flower. So the original's first game starts on **16807**,
  not 1, and `game.AdvanceRandSeed` is what puts the port on the same step. `-seed` now defaults to
  1 with `0` meaning "use the clock", which is §1.15's recommended knob and its other branch in one
  flag.

What is still open is unchanged and is only the one thing: whether Apple's trap really was
Park-Miller-by-Schrage, which needs a Mac. `ORIGINAL_GAME.md` §19.1 exception (e) is the written
form of that, together with the two draws the port deliberately does not make (`WriteOutPrefs`'s
conditional `fakeLong`, and `PourScreenOn`, which is dead code).

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

### 2.22 The two-player handshake has three bugs — **the deadlock is characterised, 1.9; the two fixes are open**

All three are in the limbo/transit handshake and all three are transcribed faithfully:

- **Mismatched-exit deadlock.** Two players leaving the same room by different exits can
  each end up waiting for the other. **Corrected in 1.9:** an earlier draft of this bullet said
  "with no key that breaks it", which is wrong — `ForceKillGlider` on Delete does break it, at the
  cost of a mortal, and it is *player 1's key only*, which is what makes the deadlock a real
  hazard rather than an inconvenience and is exactly what 2.23 now fixes. 1.9 also found the
  deadlock is wider than this bullet implied: two gliders in two *different transporters in the
  same room* deadlock as well, because the transit race demands the same physical object
  (`activeRectEscaped == index`, `Interactions.c:1381-1411`) and refuses the mismatch in total
  silence. `game.TestTransitRaceNeedsTheSameObjectNotJustTheSameKind` and
  `game.TestTheGiveUpKeyIsTheOnlyWayOutOfTheDeadlock` pin both halves, including 60 consecutive
  refused frames to show it never resolves itself.
- **The `takingTheStairs` leak.** The flag is set on a stair transit and is not always
  cleared, so a later transit in the same game can take the stairs path when it should not.
- **The shared `StillOver` edge detector.** One piece of state serves both gliders, so
  player 2 standing on a trigger can suppress player 1's edge.

A compatibility flag that defaults to the original's behaviour, with the fixes available,
is the right shape — the same shape 2.19 and 2.20 want. 1.9 shipped the *escape hatch* (2.23)
rather than a flag that makes either exit yield, because the deadlock's own fix has to choose
which player loses their exit and that is a design decision with no answer in the C; giving
both players the abandon key costs one mortal and needs no such choice.

### 2.23 Player 2 has no give-up key — **DONE as an opt-in fix, 1.9 (`fixes.player2_give_up`); the menus are still player 1's**

`Delete` abandons a glider waiting in limbo and it is player 1's key only; the menus are
player 1's too. So in a two-player game, player 2 cannot rescue a stuck situation, quit,
pause or change a setting. With the deadlock in 2.22 unfixed that is worse than an
asymmetry — and 1.9 established that Delete is the *only* way out of it, which makes the
asymmetry the difference between a recoverable game and a dead one.

**Shipped in 1.9 as `fixes.player2_give_up`**, the fourth opt-in fix and the odd one out among
them: 2.19, 2.20 and 2.39 correct something a player *sees*, this one corrects something a
player cannot *do*. It lives on `player.Input` rather than as a 47th `player.Env` method
(`World.GetInput` refreshes it from `w.Fix` each frame, because `NewWorld` does not take a
`Fixes`), and the host now reports `Delete` for player 2 unconditionally so that the *game*
decides whether it counts — one decision in one place. `ForceKillGlider`'s existing
`mode != GliderFadingOut` debounce is what keeps two players holding the key from spending two
mortals. Off by default like the rest, for 1.8's corpus reason.

**Still open: the menus.** Pause and quit remain player 1's, for a reason rather than an
oversight — the pause edge detector is a single variable, so a second poll of the same key would
pause twice — and Command has no equivalent on this host at all. Fixing it properly is
per-player edge state in the shell, which is 2.x work, not a `fixes` flag.

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

`DisplayStarsRemaining` (`Banner.c:205-236`) is the clearest case. It is called from
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

### 2.48 The audio sink is a subprocess, not a device — **DONE on Windows: a native `waveOut` sink, in the binary. The subprocess stays the Linux answer and the fallback everywhere**

For most of the port there was no audio driver at all. `audio.Pipe` writes raw PCM to `pw-play`,
`paplay`, `aplay`, `ffplay` or `play`, whichever is installed, and `audio.WAV` writes a file — and
on Linux that is still what happens, deliberately. That began as a
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
problem again.

Since the win32 backend landed this stopped being a future problem and became a shipping one: two
of the six release archives are Windows archives that draw, and they are silent by default. Three
things were done rather than left:

- `OpenPipe`'s not-found error no longer recites five Linux sound stacks at a Windows reader.
  `installHint` (`internal/audio/sink.go`) branches on `runtime.GOOS` and names the two players
  that actually have Windows builds — FFmpeg's `ffplay` and SoX's `play`. `exec.LookPath` finds
  `ffplay.exe` from the bare name because Windows consults `PATHEXT`, so **a Windows machine with
  either on its `PATH` has sound**, which is a better answer than "none" and was worth finding out
  before writing "silence" in a release note. Whether either then plays raw PCM on stdin there is
  untested, like everything else Windows in this port.
- The release notes and each Windows archive's `HOW-TO-RUN.txt` state it up front, alongside
  `-wav out.wav` as the way to hear what you missed.
- The README's audio paragraph says which platform its five-player answer is good on.

**Then the sink itself was written, because a documented limitation is still a game that makes no
noise.** `internal/audio/waveout_windows.go` is a `waveOut` driver in pure `syscall`: mono s16le at
22255 Hz through `WAVE_MAPPER`, eight blocks in rotation, `WHDR_DONE` polled every 4 ms by one
goroutine, and a tenth of a second of silence written ahead of the first mixed sample so the device
does not starve on the first hiccup. `internal/audio/device.go` is the seam it arrives through —
`Open` tries the native device first and the external players second, and nothing above it learns
which it got, exactly as `internal/platform/backend` does for the display. `-audio list` now says
`waveout` first on a Windows machine, `-audio waveout` insists on it, and the shutdown report gained
a fourth counter (`gaps`) that only a real device can produce.

Four things about it are worth knowing:

- **It has never run here.** Nothing Windows in this port has (see `internal/platform/win32`), so
  the file carries the same caveat at the top and the two ways out stay wired: `-audio ffplay` still
  takes the subprocess path and `-wav out.wav` still writes a file. What *is* checked from here is
  the half where a mistake would be silent — `waveout.go` holds the two structure layouts, the
  constants and the error table with no build tag, and `waveout_test.go` asserts every WAVEHDR and
  WAVEFORMATEX offset, the C structure's 18 declared bytes inside Go's 20, and the tuning numbers
  against each other, on every platform in `go test ./...`. The rest is checked by somebody else's
  machine: `waveout_windows_test.go` runs in CI's `native` job, where it loads winmm out of the
  system directory and resolves all seven entry points for real, and on any machine that has a sound
  card it also opens the device and plays twelve frames of silence through the full rotation. A
  hosted runner has no card, so the first test carries CI and the second skips there — which means a
  misspelt export or a mishandled path length is a red X on the next push, and only the playback
  path is still waiting for a human with a Windows desktop.
- **`waveOut` rather than WASAPI**, which is five COM interfaces and a render thread against seven
  plain functions and a struct, in code nobody here can run. On Windows 10 and 11 `waveOut` is a
  shim over WASAPI in shared mode anyway, so the game mixes with the rest of the desktop.
- **winmm is loaded by absolute path.** It is not a KnownDLL and it is not in Go's own system-DLL
  set, so a bare `LoadLibrary("winmm.dll")` would search the directory the executable was started
  from first — and a release archive tells the player to run the binary from the directory they
  unpacked it into, which is exactly the arrangement a planted DLL needs. `GetSystemDirectoryW`
  answers first.
- **The blocks are package-level arrays**, because `waveOutWrite` keeps the pointer it is given and
  Go's rules for pointers handed to foreign code do not cover that. A global is the one allocation
  whose address the language cannot ever change. `runtime.Pinner` was the first draft and is worse:
  a Pinner collected without `Unpin` panics the process by design, so a bookkeeping slip becomes a
  crash in a shipped game. `VirtualAlloc` was the second and is also worse: reading it back needs a
  pointer made out of an integer, which is what `go vet`'s `unsafeptr` check exists to object to.

CoreAudio at Stage 6 is the same problem again, and the seam is now in place for it.

### 2.49 Five house sounds ship as silence — **the linter says so as of 2.0; the decoder is still blocked on data**

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

**2.0 paid that debt, and these five are the reason the check is two checks.** Writing `sound-id`
— "the house names a `snd ` it does not carry" — produced 20 findings over the corpus, and
splitting them against `assets/extracted/sound/houses/manifest.tsv` showed that 7 of the 20 were
these five resources rather than anything an author did. Reporting those as an authoring defect
would be worse than not reporting them at all, because the author would go looking for a mistake
they did not make. So the linter has `sound-unreadable` as a separate check whose text says out
loud that this one is ours and that the sound played on a 1994 Mac, `audio.Bank.Unreadable` carries
the extractor's `status` column through so a tool can tell the two apart, and `house.SoundStatus`
has three answers rather than two. The remaining 13 `sound-id` findings are all genuine: In The
Mirror names `snd ` 10000 twice, and Teddy World names `snd ` 3000 eleven times while carrying no
`snd ` resources at all.

What is left here is only the decoder. MACE 6:1 is a published fixed-rate ADPCM and would live in
`tools/probe_snd.py` rather than in the game, which reads PCM out of the extracted tree and should
stay a consumer of it. The cost is not the decoder: `make assets-check` holds the extracted tree
byte-for-byte, so closing this is a decoder *plus* a regenerated and re-reviewed asset commit, to
buy five effects in three houses nobody has complained about. Worth doing before a 1.0; not worth
doing in front of Stage 2's own content.

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

### 2.66 The menu panel's bottom edge was a constant, and the seventh row was already outside it — **DONE, 1.10**

Found while adding "Open Saved Game…" as the title screen's third row, and it predates 1.10:
`menuBottom` was the constant `264` while the rows were laid out from `menuTop + menuFirst` at
`menuPitch` intervals, so with six rows the sixth baseline was at 272 and with seven at 298 — both
*below* the box they were drawn in. Nothing looked broken, because the rows are drawn after the
panel and a menu row over the splash art reads as a menu row either way; what was lost was the
panel, which had stopped being the thing the rows sit in.

The fix is to derive it: `menuRect(n)` returns the box `n` rows need, so adding a row moves the
box instead of overflowing it. That has a ceiling — `panel()` dims eight rows past every edge and
`DrawOnSplash` writes the house's name upward from row 307, so a bottom past 299 lays grey over
the top of the name — and `TestMenuPanelClearsTheHouseLabel` is what stops a future row from being
added past it. Two constants came out of it as well: `menuRise` is now tied to `menuPitch` rather
than sharing the house picker's `barRise`, because a selection bar one pitch tall is the tallest
one that neither overlaps its neighbour nor clips the descenders above it.

**Why it is worth a number rather than a silent fix:** it is the second time a shell constant has
been found to be describing a layout that had moved (2.29 was the first). A geometry that is
computed from the thing it bounds cannot drift; a constant beside it can, and will.

### 2.67 A hint row that shrinks leaves the wider row's pixels on the screen — **DONE, 1.10**

1.10 needed the pause hint to change *while the pause is up* — S replaces it with "game saved", Q
replaces it with the give-up question — and that exposed a latent bug in the pause placard.
`paintPause` fills only the rect the *current* hint occupies, so a hint that gets shorter paints
over a prefix of the old one and leaves the tail behind: the player reads "game saved -- S again
replaces itame". Nothing before 1.10 could reach it, because the hint was set once before the game
started and never changed.

`World.SetPauseHint` is the fix and the only supported way to change the text: it copies the old
row's rect back from the work map first, so whatever the hint used to cover is restored before the
new one is drawn, and `pausePainted` is left covering the union so `DoPause`'s restore is still
right. The alternative — widening `pauseHintRect` to a fixed maximum — would have dimmed a band of
the game the placard has no business touching.

### 2.68 One save per house, where the original's dialogue would have allowed twenty — **decision taken, 1.10; revisit if anybody asks**

`internal/saved` keys a saved game by house name, so saving twice replaces. The original's
`SaveGame2` would have called `StandardPutFile` and let the player name the file, which means as
many saves of one house as they liked — and in exchange, a resume meant finding the file again.

The trade was taken deliberately: keyed saves make resuming one keystroke, which is what makes the
feature usable at all on a title screen with no mouse, and Glider PRO is a game you resume rather
than one you branch. It is a real loss all the same, and it is the kind of thing that is invisible
until somebody wants to keep a save before a hard room. The shape of the fix if anybody does:
`Store` already separates the key from the filename (`datadir.FileName`), so a suffix — `Titanic
(2).save` — is a listing change and a picker, not a format change. The file format needs nothing:
it already carries the house name it belongs to.

### 2.69 "Beginning a new game will overwrite your saved game" — deliberately not ported — **decision taken, 1.10**

`QueryResumeGame` (`Menu.c:710-758`, never called) puts up a modal asking whether to resume, and
the original's design has a second one behind it: with one save per game there is a moment where
starting a new game destroys the save you have. The port does not ask, because in the port that
moment does not exist — a save is keyed by *house*, so starting a new game in Slumberland cannot
touch the save in Titanic, and starting a new game in Slumberland does not overwrite Slumberland's
save either. Only saving does.

Recorded because the absence is a choice and not an oversight, and because 2.68's fix would bring
the question back: numbered saves need a "which one?" and so a picker, and a picker needs the
modal the port currently has no use for.

### 2.70 The status band said one thing at a time, and the resume path has three things to say — **DONE, 1.10**

The title screen's status band was a single `s.msg` string, set by whatever last had something to
report. 1.10 gives it a third source — whether the selected house has a game to resume, and if so
whose save it is and what is in it — which has to coexist with the two that were already there
without either silently winning. `Shell.status()` is now the one place that orders them, so the
band is a function of the shell's state rather than a record of the last write to it.

Worth a number because the failure it prevents is specific: a player who presses `O` on a house
with nothing to resume needs the reason *on the screen they are looking at*, and a band that had
been overwritten by an unrelated message would send them to a terminal they may not have.

---

### 2.71 An audio player that is installed is not an audio player that works — **half DONE; the fallback is the other half**

`OpenPipe` picks the first player on `players` that `exec.LookPath` finds and `cmd.Start`
accepts, and `Start` succeeding means the binary was executed, not that it can reach a sound
server. Run the game over SSH, in a container, or from a session that does not own the seat, and
`pw-play` is present, starts, fails to connect, and exits. The game reports `audio=pw-play` and
then plays nothing.

Found by running `make check` in a clone under `env -i`, which is not a contrived environment:
it is what a CI job, a systemd unit and an `ssh host glidergo` all look like from inside.

The half that is done is attribution. The player's diagnostics have always been passed through
to the terminal, on the reasoning that swallowing them leaves "there is no sound" unexplained,
but they arrived anonymous:

    error: pw_context_connect() failed: Host is down

landing between two `glidergo:` lines, naming neither the program that said it nor which player
the game had chosen. `audio.stderrPrefix` now tags each line with the player's name, so it reads
`pw-play: error: ...` and both questions are answered by the line itself.

The other half is to stop over-claiming: a chosen player that dies at once should be reported in
the port's own voice and the next candidate tried, so a machine with both `pw-play` and `aplay`
lands on the one that works. That needs `Pipe` to own the `Wait` it currently performs in
`Close`, because a startup probe and `Close` cannot both wait on the same process, and it needs a
short grace period at open time that the happy path also pays. Neither is difficult; both are
more than a diagnostic fix, and the shape belonged with 2.48's native driver rather than in front
of it. **2.48 has since landed, so this is no longer waiting on anything**, and the seam it added
is where the retry belongs: `audio.Open` already tries the device and then the players in order,
which is the same loop a player that dies at once needs to fall out of. Until then the end-of-run
line still says `pw-play stopped taking samples`, which is the after-the-fact version of the same
fact.

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

### 3.6 A house may leave the player in a room with no way out, and a faithful port cannot tell them why — **note; the linter half is 4.1 at Stage 2, the player-facing half is 3.2's decision**

A player reported being unable to get out of Slumberland's basement and asked whether the port had
lost a staircase. It had not: this is the house's design, and `internal/game/basement_test.go` now
pins it. Both basement staircases stand over a four-pixel updraught that is the only lift in the
room reaching their 112x32 trigger box, both of those vents are authored `initial 0`, and each is
switched on from somewhere else — "Good Night"'s from the thermostat in "Anabell Lee" next door,
"Going Up? No?"'s from the one in "Switch Me", two floors up in another suite. Walk down the stairs
before finding the plate and there is no way back up. The authors knew: the rooms down there are
called "Going Up? No?", "How do you get out of here?" and "Don't Ask, Just Get Out!".

Two separate things follow, and neither is a fidelity change:

- **The house linter (4.1) should be able to answer the question the player asked.** Reachability is
  computable from the data — for each room, which exits exist, which are gated by an object state,
  and which switch anywhere in the house writes that state. A house where a room's only exit is
  gated by a switch that room cannot reach is worth reporting, not as an error (Slumberland ships
  that way on purpose) but as a fact an author of a *new* house at Stage 2 almost never intends.
  The same pass gives the port a real answer to "is this a bug?" for any house, instead of one
  person reading object tables by hand, which is how this took an afternoon.
- **Nothing tells the player.** The 1994 game's answer was the manual and word of mouth, and neither
  ships with a download. This is 3.2's territory — an optional hint that defaults to off — and the cheapest useful
  version is not a hint system: it is that a player who has lost the last glider in a room with no
  reachable exit is in a state the game already knows how to detect, and could say something about
  once 3.2 decides whether the port is allowed to be kind.

Deliberately not fixed: the vents stay off, the trigger box stays 112x32, and the basement stays a
trap. Stage 1's contract is the 1994 behaviour, and this *is* the 1994 behaviour.

---

## 4. Tooling and content

### 4.1 A house linter — **DONE, 2.0**

`glidertool house check` already round-trips and sanity-checks a house. Stage 2 authors new
houses, and the failure modes it should catch first are the ones the shipped houses
already contain: transit objects whose link points at a room that does not exist, transit
objects that are linked from nowhere, destination rooms with no staircase to arrive on,
and (2.14) custom trigger sounds that do not load. A house that fails to link is not a
crash in the original — the player just cannot get out of the room — which is precisely
why a linter is worth more than a runtime check.

**`internal/house/lint.go`, `glidertool house lint` and `glidertool house checks`.** Twenty-nine
checks at three severities, and the severities are the part that took the work. The 22 shipped
houses produce 634 notes, 48 warnings and **one** error between them, and that shape is the
design rather than an accident: a released game has to lint the originals clean enough that a
`-fail error` step is worth putting in CI, while still saying the true thing about each defect.
So the calibration went the other way round from the usual — the checks were written first, run
over the corpus, and then each class was argued down to the severity the *originals* justify.
189 dangling links across the corpus cannot be errors. Slumberland's inescapable basement is the
original's design (commit `acafec7`), so `stairs-unpaired` cannot be an error either.

Three things are worth keeping from the writing of it:

- **`sound-id` started out blaming the author for a bug of ours.** The check fires when a
  `kSoundTrigger` names a `snd ` the house does not carry, and it found 20. Splitting them
  against `assets/extracted/sound/houses/manifest.tsv` row by row gave 13 real ones (In The
  Mirror names `snd ` 10000 twice; Teddy World names `snd ` 3000 eleven times and carries no
  `snd ` resources at all) and 7 that are gliderGo's fault — the five MACE 6:1 resources our
  extractor skips, across three houses, which 2.49 has been carrying as a note since 1.6. A
  linter that reports the second kind as an authoring defect is worse than one that does not
  report it, because the author will go looking for a mistake they did not make. Hence
  `SoundStatus` with three answers instead of a boolean, `audio.Bank.Unreadable` to carry the
  extractor's `status` column through, and `sound-unreadable` as a separate check whose text says
  out loud that this one is ours.
- **What the finding says matters more than that it fires.** `sound-id` does not say "missing
  sound". It says that `LoadTriggerSound` looks only in the house's own resources and never the
  application's (`Sound.c:265-303`), and that `CreateActiveRects` then composes the room with
  **no hot spot at all** — so the trigger is not merely silent, it cannot be touched. That is a
  different bug report from the one an author would otherwise file, and it is three lines of
  comment rather than a day of theirs.
- **The catalogue is held to the code by an AST test.** `LintChecks()` is what `house checks`
  prints and the only place a reader can look up an id they have just seen. A hand-maintained
  list like that rots silently the first time somebody adds a check, so
  `internal/house/lintcatalogue_test.go` parses `lint.go`, collects every `l.add` call site's
  severity identifier and id literal, and requires the two directions to match *and* the
  declared severity to be the worst any call site uses. A behavioural test would only have
  proved the ids it happened to trigger exist; this proves there are no others. It cost one
  call site its `sev` variable — the dangling-link branch is now two literal `l.add` calls
  sharing a `const` format string — which is a fair price for the guarantee.

**Two checks were considered and deliberately not written.** *Static reachability* — "can the
player get from the first room to a star" — is tempting and would be wrong: `Room.Openings` is a
dead field (0 in all 4,070 rooms) and the real openings are computed at run time by
`DetermineRoomOpenings`, so anything static would be guessing at the geometry. Reachability
belongs to Stage 2's other acceptance criterion, the scripted headless playthrough, which
answers it by actually playing. *Objects outside their room* — the `ForceRectInRect` clamp in
`GetObjectRect` — is deferred because those per-type rects live in `internal/render`, which
`internal/house` cannot import; it wants either a callback like `PictSize` or to live in the
renderer, and neither is worth doing before a new house needs it.

Two smaller repairs fell out of the corpus run. `internal/render` had its own copies of
`ExtractFloorSuite` and `GetRoomNumber`; they now delegate to `internal/house`, and the dead
`kNumUndergroundFloors` beside them is gone. And `internal/game`'s link predicates and
`internal/house`'s are now pinned to each other by `internal/game/linkagreement_test.go`, which
walks all 144 object codes rather than trusting two lists to stay in step — the transcription
keeps its `Objects.c` comments, the map keeps its lookup, and neither can drift.

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

**1.9 added the second script and the shape the rest should copy.** `testdata/two.script` is a
two-player race, and its golden is only half of what it carries:
`TestTheTwoPlayerTraceSeesTheHandshake` reads the same run as *statements* — two idle freezes of
`IdleFrames-1`, one limbo wait, one refused wall, one joint crossing, six deaths out of one
counter, one terminal `PlayerIsDeadForever` — and finds every frame number by searching rather
than writing it down, so a physics improvement moves the numbers and leaves the rules asserted.
That is the answer to this entry's own complaint: the golden says *what changed* and the
companion test says *what broke*, and a script whose glider wanders somewhere else fails with a
sentence instead of a line number. The remaining scripts (one per subsystem) are still owed.

### 4.4 A comment naming a test that does not exist is worse than no comment — **DONE, 1.5e and 1.6; the linter is DONE, 2.1**

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
| A hyphenated line break | `internal/render/locale.go:97` ends a line with `TestFrame-` | Continues `CountsMatchTheSrcTables` on the next line, and that test exists |
| The C's own identifier | `internal/game/play.go:208` cites `TestHighScore` | `HighScores.c:374`, not a Go test |

So the check has to skip a match followed by `*`, **join** a comment's lines before matching —
not merely ignore a trailing `-`, since a hyphen split hides a dangling name exactly as well as
it hides a valid one — and exempt names that appear inside a citation of the original source.
The last one is the interesting one: `TestHighScore` will become a real Go test's name at 1.7,
at which point the false positive resolves itself and the exemption stops being needed — which
suggests the rule should be "a name cited alongside a `.c` filename is the C's", not a
hand-maintained allowlist.

**Built at 2.1, inside 4.12's checker rather than as a `glidertool lint` subcommand.** A test was
the better home for the reason 4.12 gives: a linter that reports its findings to a human is a
linter whose findings can be read and not acted on, and the whole class of defect here is
"somebody did not do the manual step". It found four more, all of them renames: the one this
entry predicted, plus three that had gone stale in the six stages since.

Two of the three design notes above survived contact and one did not.

The wildcard form needed widening: prose writes the family with an **ellipsis** as well as a star,
and this entry's own line 2510 is the proof — "the eight `TestCorpus…` tests" in the sentence, and
`TestCorpus*` in the code span beside it. The checker accepts `*`, `…` and `...`.

The hyphen rule was right to insist on joining and wrong about where the difficulty is. The
trailing-`-` case is not only a line break: line 2511 above *quotes* `TestFrame-` mid-sentence,
as an example of the convention, and no join can rescue it because the rest of the name is prose
about a different thing. So a name ending in `-` is treated as half an identifier and never as a
claim, which is a weaker rule than "join and then check" and the only one that does not fail on
this table.

**And the prediction was wrong.** `TestHighScore` did not become a Go test at 1.7. It became
`World.TestHighScore` — a *method*, transcribed with the C's name, cited in `cmd/glidergo` and
called in `internal/game`. A matcher looking for `func Test…` cannot see a method declaration, so
the reference reads as dangling now for the same reason it did then, and the exemption is still
needed. This is worth recording as a small lesson about deferring a false positive on the grounds
that it will resolve itself: what resolved was the *absence* of the thing, not the shape of the
reference to it.

The suggested rule — "a name cited alongside a `.c` filename is the C's" — was therefore rejected
in favour of the hand-maintained allowlist it was meant to avoid. Two of the five entries are not
about the C at all (`docs/CITATIONS.md` uses a placeholder while explaining the convention; two
other names are quoted in this file as errors that *were* found, and correcting the quotations
would delete the findings), so the proximity rule would have had to be joined by an allowlist for
the rest — and an allowlist with a proximity rule in front of it is strictly harder to reason
about than an allowlist. What makes the list tolerable is 4.12's tier 4: every entry is checked
for still being needed, so the one on this line will fail the build the day `TestHighScore`
becomes a test.

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

### 4.9 The compiler is allowed to fuse a multiply and an add, and two of ours were fused on arm64 only — **DONE for the three sites found; the standing check is a note**

4.5 gave a bug report the vocabulary to say "these two machines drew different pixels". This is the
first thing found that could actually make them, and it was found by looking rather than by a
failure, which is the part worth writing down.

The Go spec permits an implementation to evaluate `a*b + c` with a single fused instruction, keeping
the full product and rounding once, where separate operations round twice. It is not a bug and
there is no flag that turns it off: it is explicitly allowed, and the only thing that forbids it is
an explicit conversion of the intermediate value. amd64 codegen does not take the licence. arm64
does, and `macos-latest` is arm64, so the CI matrix is the first place this port has ever been
compiled by a backend that uses it.

Three sites had it, and `GOOS=darwin GOARCH=arm64 go build -gcflags=-S ./... | grep FMADD` is how
they were found — grep the assembly, not the source, because the source is where the fusion is
*invisible*:

- `internal/render/surface.go`, `FillPolyPatOrGray`: the scanline crossing, `aH + t*(bH-aH)`, twice
  (the compiler unrolled it). The result is then `ceil`ed to a pixel column, so a last-bit
  difference on a crossing that lands exactly on a boundary moves a whole column — `ceil(5.0)` is
  5 and `ceil(5.000000000000001)` is 6. Four callers in `objectdraw.go` draw furniture shadows into
  the background of nearly every room in the game, so an affected polygon would differ in every
  screenshot of that room for the rest of the run.
- `internal/audio/bank.go`, `stepFor`: `rateHz*RateDen/RateNum*FixedOne + 0.5`, truncated to a
  16.16 step. A rate on a boundary would resample by one 65536th on an Apple Silicon Mac and not
  on an x86 one, and every mixed sample after it would differ.

Both are fixed, and differently on purpose. The polygon fill is now exact integer arithmetic: the
vertices are integers and the row centre `y+0.5` doubles to an odd integer, so the crossing is a
rational with a small numerator and denominator and the `ceil` is an integer division — there is no
rounding left to disagree about, on any architecture, for ever. `stepFor` keeps its float64 and
gains a conversion around the product, which forces the intermediate rounding and forbids the
fusion; that is the cheap fix, and it is the right one where the value is a one-off at load time
rather than a per-pixel inner loop.

**What this was not.** It was not the cause of the CI failure that prompted the look, and saying so
is the point of this paragraph. arm64 was emulated here by patching each site to call `math.FMA`
explicitly and running the pixel and audio suites — `internal/fidelity`, `internal/render`,
`internal/game`, `internal/shell`, `internal/audio`, `internal/replay` — and every golden matched.
A 90-million-edge sweep over integer vertices confirms the arithmetic genuinely diverges (1,235,147
crossings land in a different column under FMA), so the hazard is real; it is simply that no
polygon any shipped house draws, and no sample rate any shipped sound carries, sits on one of those
boundaries. The defect was latent, not active.

Which is exactly why it was worth fixing anyway. A golden-hashing test suite that is only
*accidentally* architecture-independent cannot tell anybody that it is: the next vertex a new house
introduces, or the next edit to that inner loop, turns a latent difference into a wrong screenshot
that reproduces on one developer's machine and not another's — and 4.5's report would faithfully
record two different digests with nothing to explain them.

The open half is the standing check. Nothing stops the next float multiply-add from appearing, and
the grep above is a one-liner: a `make fma-check` that fails when `GOOS=darwin GOARCH=arm64`
assembly contains an `FMADD` outside a deliberate allow-list would keep the property instead of
re-establishing it by hand. It is left as a note rather than written now because the allow-list is
empty today and a check with an empty allow-list is indistinguishable from a grep, and because the
honest place for it is beside 4.5's cross-machine digest rather than bolted onto `make check`.

---

### 4.10 Two tests spelled a path the way Linux spells one and asserted it was universal — **DONE; the CI half is done too**

The first CI run on a machine that was not this one went red, and it went red in the one job that
had never executed anywhere: `go test ./...` on `windows-latest`. `go build` and `go vet` passed on
the same runner, and the failure was reported as `Process completed with exit code 1` and nothing
else, because a step that fails without `continue-on-error` skips every step after it and the job
log had expired before anyone opened it.

Two tests could not pass there, and neither depended on anything about the runner — no audio
device, no path length, no scheduler, no Defender. Both are the same mistake: a path literal
written the way this host writes one, compared against a value that the standard library spells
differently on Windows.

`internal/scores/store_test.go` set `GLIDERGO_CONFIG` to `/tmp/glider-portable` and then checked
that the scores directory was inside it. The two sides of that comparison arrive by different
routes. `prefs.Dir` hands the variable back verbatim, so its side keeps the forward slashes;
`scores.Dir` goes through `datadir.Dir`, which returns `filepath.Join(d, sub)`, and the last thing
`filepath.Clean` does is replace every slash with the platform separator (its own documentation says
so). So on Windows the test compared `\tmp\glider-portable\scores` against a prefix of
`/tmp/glider-portable\` and failed — for its separators, not for anything the two packages disagreed
about. The fix is one line: derive the root with `filepath.FromSlash`, which is the identity here and
on macOS, and compare against that one value. Both occurrences of the literal had to move together,
because fixing only the `t.Setenv` would have relocated the failure to the verbatim check two lines
down.

`internal/shell/library_test.go` asserted `House.Rel == filepath.Join("sub", "Nested.glh")`. `Rel` is
an `io/fs` name, not a host path: `Discover` gets it from `fs.WalkDir`, which composes every name
with `path.Join`, and `internal/shell/library.go` does not reference `filepath` once. So `Rel` is
slash-separated on every platform and the expectation had to be the literal `"sub/Nested.glh"` — as
the `t.Errorf` on the next line had been saying all along, which is the tell that the `filepath.Join`
was a slip rather than a decision. Worth being explicit about the direction of this fix, because the
other direction is tempting and wrong: localising `Rel` inside `Discover` would produce a name
`os.DirFS` refuses to open, since `os.dirFS.join` rejects any name containing a backslash. A house
whose `Rel` had one in it would be a house nobody on Windows could load.

Neither of these is a bug in the game; both are bugs in tests, and the production code was right in
both places. That is the useful thing about them. They are the first evidence that the test suite had
absorbed an assumption about its host, and the reason a cross-platform matrix earns its cost even
when the port itself is clean.

Two follow-ons were folded in. `internal/audio/waveout_windows_test.go` burst twelve one-frame
writes into a queue `waveDepth = 8` deep and asserted nothing was dropped, with a comment claiming
the queue was deeper than the test was long — which it never was. It only passed when the pump
goroutine happened to drain four blocks mid-burst. It is now `const frames = waveDepth`, which fits
the channel even if the pump never runs at all, and still exceeds the `waveBlocks - wavePrefill`
blocks that are idle at open, so it forces `acquire` to reclaim every time instead of usually. That
test skips on any machine without an output device, so it was never the CI failure; it would have
been the first failure anybody saw on a real Windows desktop.

And the diagnosis itself was the other finding. An exit code with no log is not a bug report, so the
`native` job now tees the run to a file, copies the `FAIL` lines and their `file.go:NN:` messages
onto the job's summary page, and keeps the whole transcript as an artifact — all three only `if:
failure()`. The summary page renders in a browser with nothing installed, which matters when the
person who has to read it does not have `gh`. Deliberately *not* done: making the step
`continue-on-error`. The step being red is the entire signal.

Left as a note: the same three steps would help the `check` job, which runs the much larger
`make check`, and the sweep that found these two only covered comparisons written as `!=`/`==`
against a literal. A vet-style check that flags a path literal containing `/` in the same expression
as a `filepath` call would cover the class instead of the two instances.

### 4.11 `-wav` is clocked by wall time, so audio cannot be compared across machines the way pixels can — **note; found by the first Windows run**

The first run of the Windows build produced a paced `-wav` capture that differed from the Linux one
in about 124,000 byte positions, which reads like a platform regression and is not one. Two *Linux*
runs of the identical command differ from each other in about 122,000. Same magnitude, same cause —
and those two counts are themselves not repeatable, which is the whole of the finding.

The cause is that `internal/audio`'s mixer is clocked by real time rather than by frames. Under
`-bench` that is visible in the summary line: the same 900-frame command produced 0.9 s of audio on
one machine and 0.5 s on another, because the capture is as long as the run *took*, not as long as
the run represents. A paced run is frame-locked in total length — 600 frames gives 19.9–20.0 s
every time, three measured runs landing on 887,420, 887,970 and 888,060 bytes — but the sample at
which each sound starts still depends on where the pacer happened to be when the mixer asked, so
two captures are never byte-equal.

Why it matters beyond tidiness: `-shot` renders are byte-identical across operating systems and
architectures, which is what makes them a regression check a stranger can run and CI can diff — the
first Windows run leaned entirely on that property. Audio has no equivalent, so there is no cheap
way to notice that a platform's sink has started resampling, mixing at the wrong rate, or dropping
a channel. The `sound -- N requests, N played` counters catch a refusal but say nothing about what
came out.

The fix is to clock the mixer off the frame counter: advance it by exactly `rate/30.07` samples per
frame rather than by elapsed time, with the device consuming from a buffer that is allowed to run
ahead. That makes `-wav` a function of (house, seed, frame count) alone, which is reproducible, and
lets `make check` carry a hash of one the way `internal/fidelity` carries pixel hashes. It does not
change what a player hears, because a paced run already produces a frame-locked length; it changes
whether the file is the same file twice.

Not started. The counters and the clipping figures are worth keeping either way — they are what
distinguished "the device refused this" from "the mix was too loud" during the Windows run.

### 4.12 About 17,800 citations into the 1994 C, and nothing had ever checked that one resolved — **DONE, 2.1**

The port's entire claim is that it behaves like the original, and that claim is made in pieces —
function by function, each backed by a pointer at the C it was read from. There are about 17,800 of
those pointers, naming about 18,300 lines and ranges across all 93 files of upstream `94fed96`. Not
one of them had ever been checked against the file it names.

That is a worse gap than the count makes it sound, because of what a citation is *for*. "The original
clears `mode` before the altitude check" is a thing a reader has to take on trust; the same sentence
with `Interactions.c:412` after it is a thing a reader can go and disagree with. So a citation that
does not resolve is not a small blemish — it is the difference between evidence and assertion,
wearing the costume of the former. It also costs somebody else's time rather than ours, which is the
category of defect this file exists to take seriously.

`internal/citations/citations_test.go` sweeps every `.go`, `.md`, `.py`, `.sh`, `.txt` and `.yml`
file in the tree and resolves three classes of claim. The 1994 C, in four tiers: the file exists;
every line number and range endpoint is inside it; a full path names the directory the file is
actually in, and every range runs forwards; and coverage both ways — all 67 of upstream's `.c` files
are cited somewhere, and every entry on the exemption list is still needed by something. This
repository's own `path:line` references, which is the same promise about a tree where files really do
get renamed, plus the coverage half of that — every `internal/` package has to appear on the map
`README.md` draws, because a package missing from it breaks no link and still reads as complete, so
the only way a reader finds out is by concluding the thing they wanted is not in this project. And
the test names the prose quotes, which is 4.4's linter, finally built (see below). Plus the invariant
the rest lean on: every written mention of the upstream commit names the same commit.

**Twenty-two defects, on prose that had been reviewed repeatedly.** Nine numbers past the end of
their file, six file names that do not exist, one path naming the wrong directory, two references to
documents of ours that are not there, and four test names that had been renamed out from under a
comment. Every out-of-range number was fixed by reading the C and finding the line the claim was
about, not by lowering the number until it fit — `Banner.c:205-236` because that is where
`DisplayStarsRemaining` ends, `RectUtils.c:210-216` because that is where `QSetRect` is. A citation
that resolves to the wrong place is worse than one that resolves to nothing: nothing announces
itself and the wrong place does not.

Two of the twenty-two matter beyond their own line. `Room.c` cited to 1215 in a 1206-line file means
that reading was done against a differently converted copy, so the other numbers from the same
sitting were suspect too — which is the kind of thing a count of one defect does not tell you and a
sweep does. And `PlayerControl.c` was a file an early draft invented, which has never existed in any
version of Glider PRO; it is now named in exactly one place, to say so.

A twenty-third was not a citation defect but caused four. `GliderPRO/Prefix.h` sits at the top of
upstream's tree rather than inside `Headers/`, so it was missing from the `cp` recipe in every
document — which means the four citations to `Prefix.h:1` could not resolve on any machine that had
followed the instructions, for four stages. It is in all four recipes now and in `.gitignore`. The
count of citations in the documentation was also wrong by nearly half: five places said "about
9,500", which is roughly the full-path form alone.

Three decisions inside it worth keeping:

- **The colon is required.** `Banner.c:205` is a citation; "as `Banner.c` 300 times" is prose about a
  file. Matching the space-separated form would fail against a 237-line file on writing that was
  correct, and a checker that is wrong about correct prose gets answered by rewriting the prose.
- **A floor, not an exact count.** The figure moves whenever anybody writes a paragraph, so an exact
  assertion is one that gets its number bumped without being read. The failure a floor actually
  catches is the one nothing else can see: an edit that breaks the matching turns 18,000 assertions
  into a vacuous pass over an empty slice, and the suite stays green. That does not arrive at 17,000.
- **Exemptions carry their reason, and the reasons are checked for still being needed.** Twenty-one
  names upstream does not have are legitimately cited — Apple, Win32 and X11 SDK headers, two
  metasyntactic placeholders, and `PlayerControl.c` named to record that it does not exist. An
  exemption list is the one part of a checker that cannot fail, so it is the part that rots: an
  entry nothing needs any more is a name the next person is quietly free to get wrong.

It needs the C, which is not in this repository, so the tiers that read it skip when it is absent and
print the three clone commands. A skip is a weak thing to rely on, so `.github/workflows/ci.yml` has
a `citations` job that clones upstream at the pin on every push — and that job fails if any check
skipped, because `go test` prints `ok` for a package whose every test skipped and a mistyped path
would otherwise leave it green while proving nothing. The classes that need no C never skip.

`docs/CITATIONS.md` is the reader-facing half, and its §6 is the part worth writing down: what a
green run does **not** promise. Above all that a citation is about the right thing. `Player.c:1234`
resolves, and line 1234 is `FlagGliderNormal(thisGlider);` — whether that is what the sentence beside
it claims, no sweep can say.

Two things this found and did not fix, both deliberate. `docs/analysis/stage15-raw/` is referenced by
three Go files and four documents and is gitignored, so those references are dead on every clone;
5.8 owns it and it cannot simply be repointed, because the "releasePolish N" numbering they quote
exists only there. It is on the exemption list now, which at least makes the deferral
machine-tracked: the day that numbering lands in a committed file, the checker asks for the exemption
back. And two of upstream's 25 headers are cited nowhere — `About.h` (10 lines) and `Play.h` (13) are
prototype-only companions to two of the most heavily cited `.c` files in the tree, with no `typedef`,
`#define` or `struct` between them. Coverage is therefore asserted for the 67 sources and not for the
headers.

### 4.13 The documented command lines nobody had run, and the two that hang or say nothing — **four DONE, 2.1; nine more DONE, 2.4 (the flag ordering, `docs-check`, the walkthrough, the help lines, the `GOOS` guards, `project.Releases`, PLAN's architecture map, the unsigned-binary warnings, `release.yml`'s four `sed`s), and with that the section is closed — the only command line in this repository nobody has run is `gh release create`, which needs github.com and is 5.4's**

A companion sweep to 4.12, over a different kind of claim. 4.12 checks pointers into the C; this one
is about the instructions this project gives a *person*: the commands in the READMEs, the ones the
binaries' own `-h` offers, and the sentences that say what happens if you are missing something.
None of those had been run since they were written, and a documented command line that does not work
is the most expensive wrong sentence in a repository, because it is the one a newcomer meets in their
first five minutes and the only evidence they have about whether the rest is worth reading.

Four were fixed, and they are worth listing separately because they fail in four different ways.

**`glidertool replay -script -` could not run.** Replay's own usage text offers it as the way to see
the script format — "A script is line-oriented; `glidertool replay -script -` prints one to copy" —
and it failed with `no house to replay`. `-script` writes the resolved script out and exits, so it
describes a run rather than performing one and needs no house; the check for a house was simply
above it. Moved below, and the blank house is filled in with the one the error message already
suggested, so what gets printed is a template that *runs* rather than one whose first line is `house`
with nothing after it.

The interesting part is why it survived six stages: **a test asserted the broken behaviour.**
`TestReplayNeedsAHouse` was written as `run([]string{"replay", "-script", "-"})` on the reasoning
that `-script` was a cheap way to reach the house check without starting a game. It was — and so the
suite pinned the defect, in green. That is a failure mode worth naming: a test that reaches the code
it means to test *through* an unrelated flag silently takes on that flag's behaviour as a
requirement. The case now asks its own question, under the name
`TestReplayNeedsAHouseToRun` — the four words it was missing — and the case it used to occupy is
`TestReplayPrintsATemplateThatRuns`, which pipes the printed template straight back in and replays
it, because a template that parses and does not run is documentation a reader stops trusting.

**`glidergo <house>` was silently ignored.** `cmd/glidergo` called `flag.Parse()` and never looked at
`flag.NArg()`, so the most obvious command a player can type — the name of a house — parsed, ran, and
showed the title screen. That is indistinguishable from the house having been *refused*, which is the
failure mode the `-resume`/`-room` checks twenty lines below are worded to prevent ("Accepting both
would silently ignore one of them, which is the failure mode that costs an hour of wondering why").
The principle was already written down; nothing had applied it to the arguments.

Accepted rather than refused, which is the one place this entry chose ergonomics over strictness, for
two reasons that both point the same way: the 1994 program was a Mac application and a house was one
of its *documents*, so opening one by naming it is the original's gesture rather than a new
convenience — and it is the form an operating system uses when a file type is associated with a
binary, which is what a double-clicked `.house` has to become on Windows and macOS. Two houses, or
`-house` and a bare name together, are still refused, and the message quotes both strings back,
because a name that lost its quotes (`glidergo CD Demo House`) is the commonest way to get there and
seeing the three fragments is what tells the player what happened.

**`CGO_ENABLED=0` builds a game that cannot be quit.** `CONTRIBUTING.md` said that without
`libx11-dev` "the build still succeeds and produces a binary that cannot draw". Both halves were
wrong in opposite directions. The build does not succeed — the X11 backend asks `pkg-config` for
`x11`, so it stops with `Package x11 was not found in the pkg-config search path` — and the binary
that *does* come out of a no-cgo build does something worse than not draw: it selects the null
backend, whose `PollEvents` returns only the events a script gave it, so nothing can ever deliver
`EventQuit` and the run draws frames into nothing until somebody types Ctrl-C. No output, no window,
no error. That is the first five minutes of anybody who cannot install the dev package, which is
exactly the reader that sentence was written for.

Now refused, with the three flags that give such a build something to finish. And the condition is
`endlessHeadlessRun(backendName, o)` rather than a comparison against `backend.Name`, because that
constant is `"x11"` on the host `go test` runs on: a guard written against it directly is a guard no
test can reach, and this one's failure mode is a hang, which is the single failure a test suite
cannot report on its own. The prose now says what the build actually does, and points at `make
doctor`, which already checked for `x11.pc` and was not mentioned.

A smaller trap found underneath it, worth recording because it cost a wrong measurement: the first
attempt to reproduce the missing-`libx11-dev` failure ran `go build` with `PKG_CONFIG_PATH` pointed
at nothing and **succeeded**, because Go's build cache had the cgo action already. It takes `-a` to
see it. Anybody trying to verify a toolchain-dependency claim on this project will hit the same
thing.

**"is DISPLAY set?" was the answer to one of three questions.** The X11 backend's only diagnosis was
`x11: cannot open display (is DISPLAY set?)`, and `XOpenDisplay` reports failure with no reason at
all, so the environment is all there is to go on. Three people read that sentence. One has no
graphical session and wants to be told about `-shot` and `-frames`. One has `DISPLAY` set and an X
server that refused — a wrong value, or no `xauth` cookie, which is what `ssh` without `-X` looks
like — and being asked whether `DISPLAY` is set, when they can see that it is, reads as the program
not knowing. And one is on Wayland with no XWayland, which on a 2026 desktop is the likeliest of the
three and the one the old message sent furthest wrong: there *is* a session, it is graphical, it is
running, and what is missing is a package nothing named.

`platform.DisplayAdvice(display, wayland string) string` now answers each separately, and it lives in
`internal/platform` beside `Expand` for the reason `Expand`'s own comment gives: the backend needs
libX11 and a display, a function over two strings needs neither, and so every branch is reachable
from `go test` on the machine this port is written on. The wording claims no certainty — each branch
says what was observed before what to do about it, so a reader in a fourth situation can see which
of the three they were mistaken for.

**Two documented commands were checked and are fine**, and are recorded so the next sweep does not
re-derive them. `make doctor` runs, installs nothing, and reports `libX11 dev (x11.pc)` along with
python3, a C compiler, a display and a sound device — it is the `preflight` target this entry nearly
added before reading the Makefile. And the README's Windows paragraph is current: the "never been
run" caveat was retired when the build was actually run on Windows Server 2025, and the two
remaining gaps it names — nobody has played it with a keyboard, `windows/arm64` has never run — are
both still true. The `never run` lines in `CHANGELOG.md` are history and stay as written.

**Still open, in rough order of who they cost.** Each is a claim or a command, and each is cheap.
Four were closed in 2.4 — the flag ordering, the `docs-check` target, `CONTRIBUTING.md`'s
walkthrough (whose bullet is half 5.4's and stays open for that half) and the Makefile's two help
lines — and their bullets are kept below with the amendments that closed them, because what each one
cost is the part worth remembering:

- **`glidertool <sub> file -flag` does not work**, because Go's `flag` package stops at the first
  non-flag argument, and several documented lines put the file first. A `partitionArgs` in
  `cmd/glidertool/main.go` that lifts flags out from behind positional arguments would make every
  ordering work. The ordering that fails is the one a shell user writes by habit. *(2.4: the one
  documented line that still had it — the README's `house build my-house.txt -o my-house.house`, in
  the first fenced block anybody reads — was reordered when `house stats` was added to that block.
  Reordering the documents is not the fix; the parser still refuses the habit, and this bullet stays
  open. It is now the case that every command line in `README.md` and `CONTRIBUTING.md` puts its
  flags first, which is a thing a `docs-check` target could hold.)*

  *(2.4, **DONE**. `internal/cliargs.FlagsFirst` rewrites one command line into the order `Parse`
  wants, and all sixteen FlagSets in the tree go through it. It walks the arguments and asks **the
  FlagSet itself** whether each flag takes the token after it — `IsBoolFlag`, an attached `=`, a
  name nobody registered — then emits the flags, a `--`, and the files. Asking rather than keeping a
  list of this project's own value-taking flags is the whole design: a new `-whatever` needs no edit
  there and cannot be got wrong there either.*

  *Measuring the cost first changed what this looked like. The bullet said "does not work", which
  undersells two of the three failures. `house dump f.house -o out.txt` answers `house dump takes
  exactly one house file` — **confidently false**, said to somebody who gave exactly one house file,
  and the reader's next move is to start quoting the file name. `house stats "Demo House.house"
  -tier tutorial` is worse: it prints all eighteen rows **with no tier column at all**, having
  silently demoted `-tier` to a file name, and only then stops with `open -tier: no such file or
  directory`. A flag that is ignored and then blamed for not existing on disk is not a refusal, it
  is a wrong answer with an exit code attached.*

  *It is a package and not the proposed `partitionArgs` because `cmd/glidergo` had the same defect
  and nobody had noticed: `glidergo Slumberland -scale 2` was refused with `one house at a time: 3
  were named (Slumberland, -scale, 2)`, which is 2.1's own fix for this entry reporting the
  arguments it could not reorder. That is the argument for sharing — two callers whose failure mode
  is *moving* an argument rather than refusing one — and it is the opposite call to `internal/project`'s
  nine-line formatter, which is duplicated on purpose.*

  *What it deliberately does not do is become a parser. An unknown flag stays in the flag stream, so
  a typo is still `flag provided but not defined: -teir` and not "no such file"; `-h` still reaches
  `flag.ErrHelp`; `-q true` still leaves `true` a positional, because that is what `flag` does with
  it and a reordering that "fixed" it would eat a file name in one ordering and not the other; and
  `-o --` still takes `--` as the value. Every one of those is a test.*

  *The sweep is the part that will outlive the fix.
  `cliargs.TestEveryCommandLineInTheTreeIsReorderedBeforeItIsParsed` reads the tree's own AST,
  finds every `Parse` on something it knows to be a FlagSet, and requires the arguments to have been
  reordered — because fifteen call sites are fifteen chances for the sixteenth subcommand to copy a
  neighbour's first ten lines and not its eleventh, and a behavioural test over today's fifteen
  would pass forever while that happened. It found a sixteenth site nobody had counted:
  `tools/packassets`, which takes **no** positional arguments and so had no ordering to get wrong,
  and which silently ignored one — `packassets assets/levels` packed `assets/extracted` and reported
  success. Now refused, which is this entry's own principle applied to the one program it had not
  been applied to.*

  *The `docs-check` bullet below is unaffected and still worth doing. Every documented command line
  still puts its flags first; the difference is that this is now a house style rather than a
  requirement, so a reader who departs from it gets what they asked for.)*
- **`make check` does not check the documents.** 4.12's `citations` job checks the citations and
  this entry's four defects were all found by hand. A `docs-check` target that runs the documented
  command lines — the ones in fenced blocks that begin with `bin/glidergo`, `bin/glidertool` or
  `make` — would have caught three of the four. The hard part is that some of them need a display
  and some write files, so it wants a marked subset rather than a scraper.

  *(2.4, **DONE**. `make docs-check` runs `tools/docscheck`, and it is in `make check` between
  `fidelity` and `cross`. It finds **38** command lines in `README.md` and `CONTRIBUTING.md`, runs
  **19** of them in 1.2 seconds, and prints the other nineteen with a sentence each saying why not —
  the same bargain `check-caveats` already makes for the rest of the suite.*

  *The marked-subset-versus-scraper question turned out to be a false choice, and the answer is
  both: scrape to **find** the lines, a table to **decide** about each one. What the marking cannot
  be is a mark in the document, because that is a notation in prose a reader has to step over, and
  it puts the reason in the one place there is no room to say it. The reasons are the good part —
  "opens a window and plays until the player quits", "rewrites the committed pixel corpus, which is
  the thing the corpus exists to stop happening by accident" — and they live beside the rule, in Go,
  where they are printed under the line they are about.*

  *The recogniser is **every non-empty, non-comment line inside a `bash` fence**, and not the three
  prefixes this bullet proposed, because a prefix list is a list that can be silently incomplete.
  `tr '\r' '\n' < GliderPRO/Sources/Player.c` and `go test ./internal/fidelity -update` are
  documented instructions as much as any `make` line is; a reader who runs them is following the
  page exactly as written, and a check built from three prefixes would neither run them nor say that
  it had not. The fence tag was already the marker this bullet wanted: every `bash` block in both
  documents holds commands and nothing else, and every block of sample output is fenced without a
  tag. That convention was kept by hand for six stages. It is load-bearing now, and
  `TestSampleOutputIsNotMistakenForACommand` is what holds it.*

  *Both directions are checked, and the second is what decides whether this is still a check in a
  year. A documented line no rule names fails — that is how the table keeps up with the documents. A
  rule no documented line names **also** fails, because a rule left behind by a reworded command
  reads exactly like coverage and is none: the line it was written for shows up as unclassified,
  while the orphan sits there looking like a considered decision. Both halves are `go test
  ./tools/docscheck` as well as the target, deliberately — they need no binaries, no assets and no
  display, so the person who adds a line to the README hears about it from the test suite they were
  already running, and the failure prints the two spellings and the file to put one in.*

  *Three smaller decisions, each of which was a bug in the first draft. The lines run in a scratch
  directory of symlinks to the repository's top level, so `assets/extracted/houses/*.house` resolves
  while `house build -o my-house.house` cannot leave a stray file in a working tree — `make check`
  has to stay something that can be run on a dirty branch. `.git` is deliberately not among the
  symlinks: a check with a writable path to the repository's own history is a check that can lose
  work, and all the exclusion costs is that `make` inside the scratch tree computes its version as
  `dev`. And every line gets two minutes, which is not about speed — the nineteen take 1.2 seconds
  between them — but because one of the four defects above **was a hang**, and a check that inherits
  a hang has inherited the worst version of it: a `make check` that never ends.*

  *What it found on its first run: nothing, which is the expected answer and not a disappointment.
  2.1 fixed these same lines by hand a stage ago; what this buys is that the next four are found by
  a machine instead. It did catch two stale sentences on the way in — `CONTRIBUTING.md`'s prose copy
  of `check`'s prerequisite list, which goes out of date every time that list changes and did again
  here, and `README.md`'s `tools/` row, which still described the directory as standard-library
  python3 after two Go commands had moved in.*

  *What it still does not cover, stated because a check's caveats are the part that gets forgotten:
  `make run` and `glidergo -levels assets/levels` open a window and play until somebody quits, so
  both are skipped and `make smoke` is what exercises the blit path; `house dump ... | less` runs,
  but under a pipe `less` is `cat`, so what is checked is the dump and the exit status and not the
  pager; and the two `git clone` lines have never been run by anybody on this host, which has no
  network at all. Nineteen of thirty-eight is the honest number, and the target prints it.)*
- **`release.yml`'s `sed` and `CONTRIBUTING.md`'s walkthrough are still unrun.** 5.4 owns the first;
  the second is the "your first patch" section, whose steps have never been performed end to end by
  anybody, which is the same class of defect as the four above and the one most likely to be met.

  *(2.4: the walkthrough half is **DONE**, 5.4 still owns the `sed`. The five `glidertool` lines
  under `CONTRIBUTING.md`'s "Houses" heading were run end to end for the first time — dump
  Slumberland to text, build it back, lint it, print the check catalogue, measure it against the
  small tier — and all five work: the round trip reports `383 rooms, 2996 objects, 134150 bytes`,
  the lint ends on the dangling-link note and the wrong-background warning that the 22 originals are
  calibrated to allow, and `stats` says 210 of 383 rooms are reachable from `Welcome…`. They are not
  left as a one-off: `docs-check` above runs all five on every `make check`, which is the point of
  having built it.*

  *(2.4, the `sed` half too, and it turned out not to need 5.4 at all: the whole "Package the
  archives" step runs on this host unchanged. Extract its `run:` block, dedent it, give it `VERSION`
  and the three `GITHUB_*` variables `sed` interpolates, and point it at the `bin/cross/` layout
  `make cross` already writes — all six archives pack, all six pass their own embedded-asset greps,
  and `SHA256SUMS` comes out with bare filenames. The link rewrite does what its comment claims: the
  README's three screenshots become `/raw/` links and its thirteen documents `/blob/` links, every
  one pinned at the 40-character SHA rather than at a branch, and the `! grep -q '](docs/'` assertion
  underneath finds nothing left behind. The CRLF `sed` and the `VERSION` `sed` in the notes step were
  run the same way. Four `sed`s, none of them unrun now.*

  *The cost of rehearsing it is 150 MB in `/dist/`, which `.gitignore` already excludes and says is
  for exactly this. What it still cannot reach is `gh release create`, which is 5.4's and needs
  github.com — so the file's standing caveat stays, one step shorter than it was.*

  *Two notes for whoever reads this bullet next. This one named a section that does not exist — there
  is no "your first patch" heading in `CONTRIBUTING.md`; the walkthrough is under "Houses", and a
  pointer into our own documents that does not resolve is exactly what 4.12 sweeps for elsewhere. And
  the three lines under "A house that ships" are still not run by `docs-check`, for reasons that are
  covered rather than skipped: `make levels` is a target `make check` runs itself, `make levels-zip`
  rewrites a committed archive and `go test ./assets` is what checks its claim, and the third opens a
  window.)*
- **The Makefile's `## ` help lines skip `smoke` and `all`**, so `make help` lists neither, and
  `smoke` is in `make check`. Two lines.

  *(2.4, **DONE**, and it was two lines. `all` says it is the default target, because `make` on its
  own is the first command `CONTRIBUTING.md` gives and nothing said what it did; `smoke` says it is
  skipped with a note when `DISPLAY` is unset, which is the thing a reader of a green CI log needs to
  know about it. Worth pairing with the bullet above: `make help` is now itself a checked command
  line, and `make tools` lists the Go commands by `tools/*/main.go`, so `docscheck` appeared there
  without being named — which is the sort of thing that only works if the list is derived rather than
  written down.)*
- **`cross`'s caveats are not guarded on `GOOS`**, so a macOS reader is told what a Linux build
  cannot do. Same for the Windows and macOS gate commands, which name Linux paths.

  *(2.4, **DONE**. Three targets print a caveat, and all three of them were Linux assumptions worn
  as universals: `cross`'s last row, both halves of `check-caveats`, and `smoke`'s skip note. Each
  now asks `go env GOOS` first.*

  *What they said before is worth recording, because the two operating systems were wrong in
  opposite directions and only one of them looks like a bug. On macOS everything overstated: `cross`
  built a **darwin** binary with cgo on and printed it as `linux/amd64 +cgo … x11 backend (this
  host)` — a row whose shape was right and whose every fact was false — and `check-caveats` reported
  that cgo was off, or that libx11 metadata was missing, about a platform that has no backend to
  compile either way until stage 6. On Windows it understated, which is the direction `check-caveats`
  exists to prevent: it said the x11 backend was NOT compiled and that `build` had produced the null
  backend, of a build whose win32 backend needs no cgo, **was** compiled, and draws — and then
  blamed an unset `DISPLAY` for the window that did not open, of an OS that has no reason to set one.
  A caveat list that undersells a green run teaches people to stop reading it, which costs exactly as
  much as one that oversells.*

  *The bullet's second sentence names something that does not exist: there are no "Windows and macOS
  gate commands" in any document here — `grep` finds none, and the gate is one `make check` that runs
  everywhere. What it must have meant is these three printed caveats, and they are what was fixed.
  Recorded so the next reader does not go looking.*

  *Two fixes in passing. `smoke`'s old note said the blit path "is still covered by `make headless`",
  which is backwards — `headless` renders through the null backend, so the blit is precisely the one
  thing it does not cover — and that is the same defect as the Windows caveat, one line up. And the
  libx11 branch printed a hardcoded `linux/amd64 +cgo`, so a linux/arm64 host was told about somebody
  else's architecture; it uses `go env GOARCH` now. What did not change is behaviour: on Windows,
  `smoke` still skips rather than benching, because nothing in a Makefile can decide from outside
  whether a window would open there, and turning an honest skip into a possible flake inside
  `make check` would be a worse trade than the wrong sentence was. It now says that, and names `make
  bench`.*

  *How the new wordings were read, given that this host is Linux: `GOOS=darwin make check-caveats`,
  `GOOS=darwin make cross` and `GOOS=windows make smoke BIN=/tmp/x` print exactly what a reader on
  those machines sees, because all three branches ask `go env GOOS` and that answers the environment.
  All four values were run — linux, windows, darwin and a `freebsd` that stands for the default arm.
  Those command lines are in the Makefile's comments, because they are the only test these branches
  have and that is not obvious: CI never reaches any of them. `windows-latest` has no `make` at all
  and the macOS half of the same job calls `go` directly for the same reason, so the `native` job
  compiles and tests the Windows and macOS code without ever running the Makefile that describes
  it.)*
- **`project.Releases` has no caller.** It is the one constant in `internal/project` that nothing
  reads, which means nothing checks it either. Either the release notes and `-version` should point
  at it or it should go; an exported constant with no reader is a string that can rot silently, and
  this package exists specifically to stop that.

  *(2.4, **DONE**: it is gone, and the class of defect is now a test. Of the two options the bullet
  offers, "point at it" turned out to be the one that could not be taken honestly. No tag has been
  pushed (5.4), so the page the constant names has nothing on it — and the README's opening sentence
  is "there is nothing to download and no copy of the old game to find. Clone it and `make run`",
  which is **true today** and is the single best thing that paragraph could say. Pointing a player
  at an empty releases page, in a repository whose 4.13 is a list of documented instructions that
  did not work, would have been the same defect wearing a nicer hat.*

  *What the deletion cost had to be weighed, because the obvious objection is that 5.4 will want the
  string back within one command of tagging. It is one line, and the reason it is safe to lose is
  that the constant was not protecting anything: `TestEveryGitHubLinkIsOneWeMean` reduces every deep
  link to its repository root before judging it, so a hand-written `.../releases` in the README
  would pass with or without a constant to compare against. `internal/project` protects **Go**
  callers, and a downloads URL has no Go caller — the player holding the binary does not need to be
  told where to download it. 5.4's entry now names the two places the URL belongs when a tag exists.*

  *The general form is worth more than the one deletion. `TestEveryExportedConstantHereIsReadBySomething`
  parses `project.go` for its exported constants and sweeps every other `.go` file in the tree for
  `project.X`; a constant nothing reads fails, with the two options in the failure message. It found
  a second one on the first run, and that one was a false accusation worth keeping: `Module` has no
  external caller either, because `Home = "https://" + Module` is its only reader — and `Module` is
  checked, by `TestModuleMatchesGoMod`, and is the string every URL in the game is built from. So the
  rule is transitive: a constant counts if something outside reads it, or if a constant that counts
  is built from it. A test that had shipped without that distinction would have demanded a caller for
  the most load-bearing string in the package.)*
- **`docs/PLAN.md` §3's architecture map names directories that do not exist.** It was written before
  the tree settled and has not been revisited. Unlike the README's map, which 4.12 now sweeps,
  nothing checks PLAN's — and the two disagree, which is worse than either being wrong alone.

  *(2.4, **DONE**. Four nonexistent directories, and the pattern in them is worth naming: every one
  was a guess about how the code would be organised, made before it was written, and left standing
  after the code disagreed. `internal/ui/` became `internal/shell/`. `internal/assets/` was never
  written at all — the embedding lives in two hand-written files at `assets/`'s top, which is a
  better answer and is why nobody missed the package. `internal/game/objects/` and
  `internal/game/room/` were a decomposition the simulation never wanted; `internal/game` kept both
  in itself. A fifth line was wrong in the other direction: `internal/platform/backend/` exists, is
  the build-tag selector every binary goes through, and the map that claimed to show the port layer
  did not have it.*

  *There was also a stale `[DONE, never run]` on `win32`, a stage after it had been run on Windows
  Server 2025. A plan is allowed to be out of date about the future. Being out of date about the
  present is how a reader learns to stop reading it, and that one line undersold the single most
  expensive thing the previous stage did.*

  *Fixed by deleting the duplication rather than by synchronising it, which is the same call 4.26
  made about `make check`'s step list. The README's "What is in here" table is the exhaustive map and
  already has `TestEveryPackageInTheTreeIsOnTheMapTheReadmeDraws` holding it to the tree; PLAN's
  fence keeps only the layering the three rules under it depend on, plus the three directories that
  do not exist yet with the stage that owns each. Two maps of the same tree is the defect; one map
  and one diagram of it is not.*

  *The check is `TestEveryPathTheArchitectureMapNamesExists` in `internal/citations`, and it reads
  the fence as the indented tree it is: a line's path is its first field joined onto its ancestors'.
  A `[stage N]` marker exempts a line from having to exist — and requires that it does not, because a
  marker left on a directory that has since been written is the `win32` defect exactly. The one line
  that says "not committed" (`assets/levels/`, which `make levels` builds) is checked against
  `.gitignore` instead of against the filesystem: it is present here and absent from a fresh clone,
  and `make check` runs `test` five steps before `levels`, so requiring it would have made the
  result depend on whose checkout it ran in.*

  *Carried along: the README's map was only ever checked in one direction. Every `internal/` package
  had to appear in the table; nothing said a path in the table had to exist. Those are different
  failures and only the first one looks like a hole — a row for a directory that has been renamed
  away reads as complete and sends the reader somewhere that is not there.
  `TestEveryPathTheReadmeMapNamesExists` closes it, resolving the 34 paths the table names. A
  backticked token counts as a path if it has a slash and no space, which excludes the table's
  commands, build directives, resource types and the one 1994 filename with a space in it.*
- **Nothing in the release archives mentions SmartScreen, Mark of the Web, or Gatekeeper
  quarantine.** A player who downloads an unsigned Windows zip gets "Windows protected your PC" with
  no Run anyway button visible until they click More info, and a macOS build is quarantined outright.
  Neither is a bug and both look exactly like one. This is 5.4's release notes, and it is the single
  most likely reason a first-time player never sees the title screen at all.

  *(2.4, **DONE** as far as an offline machine can take it. Three places now say it: a new "Your
  computer will try to stop you, once" section in `release.yml`'s release notes, placed above the
  Windows section because it comes first in the reader's day; a block in the zip archives'
  `HOW-TO-RUN.txt`; and a shorter one appended to the darwin archives' `HOW-TO-RUN.txt` only. That
  last detail is the reason it is appended rather than written into the heredoc — the headless text
  is shared with `linux-arm64`, whose reader has no Gatekeeper and no reason to be told about one,
  and the three-way `if` above it is sorted by what the build can draw, which is a different
  question.*

  *Each of the three gets the click path and not just the name. `More info` → `Run anyway`, because
  the button a player needs is the one the dialog hides. For Mark of the Web, right-click the zip
  **before** extracting → Properties → Unblock, which is the form that clears it once instead of per
  file, with `Get-ChildItem -Recurse | Unblock-File` for somebody who has already extracted. For
  macOS, the fact that decides the outcome: Archive Utility propagates `com.apple.quarantine` to what
  it unpacks and `tar xzf` does not, because the attribute is on the archive file rather than stored
  inside it — so "unpack it in a terminal" is the whole fix, and `xattr -d` is only for a reader who
  already did it the other way.*

  *Not signing is stated as a decision rather than left to look like an oversight. A certificate
  costs a few hundred dollars a year and has to be issued to a named person or company, which is the
  same argument `internal/project`'s `Copyright` makes about "the gliderGo authors": that is a thing
  to do deliberately. Each of the three sections ends on the distinction that matters, which is that
  none of these checks looks at what is in the archive — `SHA256SUMS` does, and it is already
  published.*

  *What is unverified, and it is a different kind of unverified from the rest of `release.yml`. That
  file's standing caveat is about GitHub, which this host cannot reach; this is about client machines
  nobody here has. The Windows half needs no network at all, though, which is the useful discovery: a
  browser download is an ordinary file with one alternate data stream on it, so
  ``Set-Content -Path .\glidergo.exe -Stream Zone.Identifier -Value "[ZoneTransfer]`nZoneId=3"``
  and then a double-click in Explorer is the real test. The recipe is in
  `docs/windows-first-run.md` under "Rehearsing what a download adds, with no download", which is
  where the next person with a Windows box will look, and it is listed there as gap 4 — the `.exe`
  that ran on Server 2025 arrived over SSH and so carried no mark and met no SmartScreen. It is worth
  doing before the first tag: the notes say `Run anyway` is behind `More info` on the authority of
  documentation, and one double-click would put it on the authority of somebody having seen it.*

### 4.14 The houses this port writes will sit on the same shelf as the 1994 ones, and nothing said which was which — **DONE, 2.2; the built-in New set followed in 2.3, and the amendment below is where the flag's meaning changed**

Stage 2's job is to add houses. The picker they land in is one flat alphabetical list, which is
faithful — `BuildHouseList` produces exactly that — and the moment this port puts a house of its own
into it, the list is making a claim it cannot support. "Bakery" between "Asylum Pro" and "CD Demo
House" reads as something Ward Hartenstein or Jonathan Chin built in 1994. That is wrong in both
directions and each direction costs something real.

It takes credit that belongs to five named people. Section 1.2 has the table: thirteen of the
twenty-two houses were designed by someone other than John Calhoun, the whole reason the assets
needed a decision of their own, and a list that silently mixes ours in with theirs is the one place
that table cannot be seen from. And it lends the originals our mistakes. A new house with a room
the glider cannot leave is a bug in a 2026 port; the same room, believed to be 1994's, is evidence
that the game was always like that. The player has no way to tell, and the port's credibility is
built entirely on being the kind of thing that tells you.

**A set is declared by the *source* a house was walked from.** Not by the file, and not by a list of
names. `internal/shell/sets.go` carries the argument in full; the short form is that the other two
were each tried on paper and each fails in a way worth recording.

The *file* cannot say it. `houseType` has no field for who made it, the 866-byte header is full —
every byte named, the codec round-trips all 22 shipped files exactly — and inventing a field means
writing a house the 1994 program cannot open, which is the one thing this port will not do to the
format it transcribes.

A *list of the twenty-two names* compiled into the port is the tempting one, and it is wrong twice.
It puts the truth about the shipped set in a second place, so the list and `assets/extracted/houses`
can drift, and on the day they do the list is what the game believes. Worse, it answers confidently
in precisely the case where the answer matters: a house opened in an editor, changed, and saved back
as "Slumberland" is not the 1994 Slumberland, and the name is exactly the part that did not change.
**A provenance claim that cannot fail is not a provenance claim.**

So the caller that opens a root declares what the root is, and every house found under it inherits
that. What this claims is deliberately less than it looks, and saying so is the point:
**a set says where a house was found, not what is in it.** `-houses some/dir` is therefore `Other`
and not `Original` — the game cannot know what a player put in a directory — and the picker shows
the root's label beside the count, so the claim is always checkable against the place it came from.
This is the same habit as the About box printing the upstream commit it was pinned against rather
than a nicer sentence about provenance it does not have.

One caller is allowed to declare `Original` for a directory, and the exception is instructive rather
than a loophole. `internal/fidelity`'s `ScreensOpts.houseSource` records the shell's reference images
from `assets/extracted/houses`, and that directory is the extracted twenty-two *by construction*: the
`screens.hashes` checked in beside it is a hash of what that produced, so a directory with anything
else in it changes the hashes and fails. The declaration is true because the test that reads it is
what makes it true. The difference between the two callers is written where each one is.

**What accumulates and what replaces.** 5.3 deferred union semantics for the asset roots to Stage 2,
on the grounds that a union should be designed rather than fall out of a resolution order nobody
wrote down. It is designed here, and it lands in one of the two places and not the other. An art
root *replaces* the built-in one, because two copies of PICT 1000 have to resolve to one picture.
Houses do not resolve; they accumulate, and a player with four of their own and the twenty-two wants
twenty-six. So `Library.Discover` is variadic over sources and `internal/assetfs` is untouched —
and a source that cannot be read is an error *and the rest are still walked*, so a mistyped
`-levels` names itself on the screen the player is looking at and still leaves them a list to play
from.

**Amended when the levels root became a built-in one, and the amendment is the interesting half.**
The paragraph above was written when `-levels DIR` was the only way to get a New set at all, so
"accumulate" had nothing to accumulate *with*: the flag added a second source to the houses root and
that was the whole of it. `assets/levels.zip` changed the question. There is now a built-in New set,
and the flag has two possible readings — add a third source, or replace the built-in one the way
`-art` replaces the built-in art.

It replaces, and the deciding case is the workflow this repository documents. `make levels` builds
`levels/*.house.txt` into `assets/levels/` so an author can play what they just wrote; the way they
play it is `bin/glidergo -levels assets/levels`. Under additive semantics that lists every house
*twice* — once from the archive, once from the directory it was packed from — and the two rows share
one save file and one score file, because a side-car is keyed on the house's name and
`h.TimeStamp` (`internal/saved/store.go:401-407`) and both rows agree on both. The failure additive
semantics creates is therefore not an edge case a player has to go looking for; it is what happens
the first time anybody follows the instructions.

So `internal/assetfs` is no longer untouched: `assetfs.Whole` is `Root` for a tree that is already
the root, and `-levels` goes through it. What accumulates is still the *set* — `Discover` is still
variadic, the Original root and the New root are still two sources — and what a directory replaces
is one root, which is now the rule all five flags follow without exception.

Accumulating is not the new part. `BuildHouseList` already walked two sources into one list: up to
eight specs handed in by `AddExtraHouse` — a house dropped on the application, from anywhere on any
volume — copied in first, then `DoDirSearch` for the rest (`SelectHouse.c:650-664`, `668-675`). What
the original does not do is say which place a row came from. The set is that missing sentence and
nothing else.

**The UI, and the three decisions in it that are not obvious.**

The filter opens on `AllSets`, not on the selected house's set. Defaulting to the selection's set
would hide every new house on a fresh install, because a fresh install selects Slumberland: the
player would have to discover a chooser in order to discover that there was anything to choose. A
list that hides houses by default is a list whose player never finds out what is missing.

`pick` stays an index into the whole library rather than into the filtered view, so changing the
filter changes what is *drawn* and never what is chosen. The one case that needs a rule is opening
the picker on a house the filter does not hold, and `clampPick` widens the filter to `AllSets`
instead of moving the cursor — the cursor is on the house the player asked for and the filter is the
part they did not.

**Tab cycles the sets**, and that is a key taken rather than a key found. In the original's dialog
Tab is an undocumented alias for the right arrow (`SelectHouse.c:277-278`, the two cases share a
body); in this port's picker it had become a second Escape, advertised nowhere. Nothing documented
is lost, and the footer now offers `Tab set` only when there is more than one set to cycle. The
settings screen keeps Tab as its way out, because it has nothing to switch between.

**The layout mistake is written down because the reasoning was wrong in a way that will recur.** The
obvious home for a set chooser is the gap between the picker's title and the first house — 32 rows,
apparently empty. It is not empty. The selected house's inverse-video bar rises `barRise` above its
baseline, so the first row's bar occupies rows 80 to 104, and the title's shadow reaches row 73. Six
free rows, where a scale-1 word needs ten. The error was measuring the gap against the first row's
*glyph ink* at row 86 and not against its *selection bar*, and it was not caught by looking at a
screenshot — `TestOneSetDrawsNothingNewOnTheTitleLine` counted 2,602 cream pixels where a one-set
library must draw none. The strip went onto the title's own baseline, which is the row that actually
has room, and the title lost the set name it had briefly gained, because the strip's highlighted
entry already says which set is showing. The wrong measurement is in the comment on `pickSetsV` so
that the next person to want a row there knows why there isn't one.

**With one set on the shelf the picker draws exactly what it drew before** — no strip, no title
suffix, no `Tab set` in the footer. That is not politeness; it is what keeps `internal/fidelity`'s
reference image a picture of the 1994 dialog, and it is asserted rather than assumed:
`TestOneSetDrawsNothingNewOnTheTitleLine` renders a one-set library twice, once as `Original` and
once as `New`, and requires the two whole screens to be identical. `make fidelity` passes with
`screens.hashes` and the golden `houses.png` unchanged, which is the same claim made from the other
end.

**Where the built-in New set lives — DONE, and it was the half that decided whether any of this
reached a player.** Until it was built, the New set needed `-levels assets/levels` on the command
line, so nobody who downloaded a release archive had a twenty-third house at all: the work in 2.3
existed only for people with a clone and the instructions in front of them.

The two places a reader would guess are both barred, and they stayed barred.
`assets/extracted/houses/` cannot take a new house: `make assets-check` is a full recursive `diff
-r` against the extractor's output and `assetpack.Compare` asserts `extracted.zip` holds exactly
the files in `assets/extracted`, so a 2026 house would be reported as an extra file forever —
correctly, because that tree's whole value is being byte-for-byte reproducible from 1994 data, which
is 1.2's provenance boundary. And `//go:embed` cannot reach outside `assets/`. So it is the second
archive this entry recommended: `assets/levels.zip`, packed from `assets/levels/` by the
`tools/packassets` that already existed, embedded beside `extracted.zip` and opened by
`assets.Levels()`. No new machinery, and the provenance boundary is now visible in the file names.
The alternative — generalising `assetpack.Files`/`Create`/`Compare` to pack two trees under
prefixes — would have bought one file at the cost of making two different contracts read as one:
`extracted.zip` carries a claim about 1994 that `make assets-check` proves, and `levels.zip`
carries no such claim.

Three consequences that are not obvious from the outside, all of them load-bearing:

- **The archive is committed and the directory it is packed from is not**, which is the reverse of
  `assets/extracted{.zip,/}`, where both are. The reason is a cycle: `make levels` runs
  `bin/glidertool`, `cmd/glidertool` imports `assets`, and `assets` embeds `levels.zip` — so a
  checkout without the archive cannot build the tool that packs it. The `embedded` guard's remedy
  for a missing `levels.zip` is therefore `git checkout -- assets/levels.zip` and nothing else, and
  the rule the two archives share is "commit what a clone cannot regenerate, plus whatever the
  build needs in hand before it can regenerate anything".
- **The member sits at the archive's root** — `Open House.house`, not `levels/Open House.house` —
  because what is embedded *is* the levels root, so a prefix inside it would have to be stripped by
  every caller. `assetfs.Whole` exists for the same reason: `Root(tree, dir, ".")` would have
  labelled the source `built-in:.` and drawn paths like `built-in:./Open House.house`.
- **`go test ./assets` is the only thing in `make check` that can see any of this.** Nothing under
  `internal/` can reach the embedded levels root — `internal/shell` takes its sources from its caller and `internal/fidelity` builds its own
  — so `go test ./assets` is the entire coverage, and the two likely ways to ship it wrong are
  both silent all the way to the player: a wrongly-prefixed archive (no house appears, and nothing
  errors) and a stale one (the house that ships is not the house in the text). Hence two tests that
  do not skip: one requires the archive's root to hold exactly the authored house names and no
  directories, the other rebuilds every `levels/*.house.txt` and compares the bytes.

**`docs/PLAN.md` §3's map — DONE.** It said `levels/` was "new houses in text form" while this
entry's flag said `-levels` was a directory of *built* houses; both were true and the naming is now
stated once rather than inferred twice, in §3's map itself: `levels/*.house.txt` authored,
`assets/levels/` built by `house build` and regenerated, `assets/levels.zip` that directory packed
and committed, `-levels DIR` to replace the built-in root at run time.

**Still open.**

- **The chosen set is not remembered between runs.** The filter resets to `AllSets` every launch,
  which is the right default but not obviously the right *only* behaviour once there are two large
  sets. It is one field in `internal/prefs` if anybody asks, and it is deliberately not added on
  speculation: every row on the settings screen is a row a player has to read past, and nobody has
  yet had two sets large enough to be annoyed by this.

### 4.15 A new house cannot have art of its own, because there is nowhere its pictures are allowed to sit — **note; found writing the first one, 2.3; half the mechanism has since changed**

`Open House` (2.3) is drawn entirely with built-in backgrounds 2000-2017 and built-in object art, and
that was a constraint discovered rather than chosen. §10.3 step 3's target background mix is 15 %
indoor, 10 % `kDirt`, 3 % ground-outdoor, 5 % elevated, 22 % air/space, **27 % user-structure and
17 % user-open** — 44 % of the corpus's rooms are painted with art the house carries itself. Ours is
72 % interior, 7 % `kDirt`, 21 % air, and the missing 44 % is not a design preference.

**It is not that gliderGo cannot read house art.** It reads it very happily, and not as a resource
fork: `assets/extracted/houseart/<House Name>/pict/<id>.png` is the whole interface, one PNG per
`PICT` id, and `OpenHouseResFork` (`cmd/glidergo/play.go:427-432`) mounts that directory for as long
as the house is open. Anybody could paint a 512×322 PNG, call it `3000.png`, and a room with
`background 3000` would draw it. The obstruction is two mechanical facts about where those bytes are
allowed to sit, and they were both put there for good reasons that happen to collide here.

**`assets/extracted/` is diff-locked against the extractor.** `make assets-check` is a full
recursive `diff -r` against `tools/extract_all.py`'s output and `assetpack.Compare` asserts
`extracted.zip` holds exactly the files in that tree, so a `houseart/Open House/` directory breaks
both contracts — correctly, because being byte-for-byte reproducible from the 1994 CD is that tree's
entire value, and 1.2's provenance boundary is the reason it has one. This is 4.14's already-filed
"where do the new houses live" problem wearing a second hat, and it wants the same answer:
`assets/levels.zip`, packed by the `tools/packassets` that exists, with the art beside the house.

**That archive now exists** (`assets/levels.zip`, embedded in every executable) **and it holds only
houses.** Art beside the house is still unbuilt, and the shape it wants is clearer for the archive
existing: one directory per house inside the levels archive, mounted by `OpenHouseResFork` the way
`houseart/<House Name>/` is, so that `background 3000` in a house of ours finds a PNG the binary
carries. What that needs is a second lookup in `OpenHouseResFork` and nothing in `assets/extracted/`
touched at all, which is the point.

**`-houseart DIR` is exclusive, and that is the part worth fixing first.** It *substitutes*:
`assetfs.Root` returns `os.DirFS(dir)` and ignores the embedded tree
(`internal/assetfs/assetfs.go:38-46`). So the only way to hand a new house its art today is
`-houseart` pointed at a directory that also contains all twenty-two originals' art, and anyone who
points it at just their own house silently takes the art away from every shipped house that needs it
— which is half of them. A house's art root is *per house* by construction (the lookup is
`IsDir(houseArtFS, name)`), so there is no reason for this one flag to be exclusive; it should search
a list of roots, and the per-house directory name already makes collisions impossible.

This paragraph used to draw the contrast with `-levels`, which it said was *added* rather than
substituted. That is no longer true and the correction is 4.14's amendment: `-levels DIR` replaces
the built-in levels root, because the documented author workflow — `make levels` then
`-levels assets/levels` — listed every house twice under additive semantics, with both rows sharing
one save file. `-houseart` is now the only asymmetry left, and it is a per-house lookup rather than a
root-level one, which is why the answer for it is a search list and not a replacement.

Until then the restriction is the honest one and it is written into the house's own header comment:
no `kUserBackground` (≥ 3000), no `kCustomPict`, no `kTV`, no custom `snd `. That also means §10.3
step 4 — "light every user-art room with a `kInvisLight`", the most common authoring mistake — has
nothing to apply to yet, and §10.5's `kSoundTrigger`-with-no-`snd ` and `kTV`-with-no-`.mov` traps
are unreachable rather than avoided. Worth knowing when the first house does carry art: three of the
recipe's rules go live at once.

### 4.16 The profile a new house is held to was measured by hand, and two of its fifteen rows cannot be measured at all — **note; the Go half DONE, 2.3; both "impossible" rows turned out to be computable, the second in 2.4 — DONE, see the two amendments**

Stage 2's bullet asks for houses "designed against the quantitative profile of the originals in
`docs/analysis/original-houses.md`". §10.2 is that profile: fifteen rows, five tiers. Hitting it for
`Open House` meant `grep -c` and a pocket calculator over the authored text, twice, because the first
pass got the empty-room count wrong. There is no tool. `glidertool house lint` answers "is this file
legal", which is a different question from "is this file the kind of house it claims to be", and
nothing else in the repository computes a single cell of that table.

**Thirteen of the fifteen rows are now checked in Go, and that is where the argument for a tool
starts rather than ends.** `TestOpenHouseMatchesTheTutorialProfile`
(`internal/replay/openhouse_test.go`) walks the parsed house and asserts each band, so the numbers in
the house's header comment are an invariant instead of a claim — somebody adds four objects to make a
room look better and the test says which row left its tier. But it is hard-coded to one house and one
column, which is exactly the wrong shape: the next house is a different tier, and the thing an author
wants before they have a passing test is *the numbers*, printed, so they can see they are at 2.9
objects a room with a ceiling of 3.1. That is `glidertool house stats` — the same walk, a table on
stdout, `-tier tutorial` to add the bands and an exit status. Both existing callers would then be
checking the same code instead of the same table.

**Two rows cannot be computed from `internal/house` at all, and both for reasons already on file.**

- **Dark rooms** (§10.2 target 0 % for a tutorial, 4.6 % corpus-wide) needs `GetNumberOfLights`,
  which is a method on `*render.Scene` (`internal/render/locale.go:1350`). `internal/render` imports
  `internal/house`, so the dependency cannot be inverted, and re-deriving the rule in `house` means
  two copies of a switch over eighteen backgrounds plus the `kDirt` all-tiles-zero special case plus
  the eight `State != 0` lamp types — a copy that would drift, and drift silently, since a house that
  a linter calls lit and the renderer draws black is worse than no check. 4.1 already made this call
  once, for objects outside their room ("it wants either a callback like `PictSize` or to live in the
  renderer, and neither is worth doing before a new house needs it"). A new house now needs it, and
  the callback is the answer both times: `LintOptions` grows a `Lights func(*Room) int`, `glidertool`
  fills it in from `render`, and the check is skipped with `checks-skipped` when nobody did. That
  keeps the one implementation in the renderer, which is where the pixels are.
- **BFS eccentricity** (11-15 for a tutorial) needs the room graph, and 4.1 explains why there isn't
  one: `Room.Openings` is dead in all 4,070 corpus rooms and the real adjacency comes out of
  `DetermineRoomOpenings` at run time, in `internal/game`. Same shape, same remedy, one more
  callback. It is also the row most worth having, because it is the only one that measures the
  *shape* of a house rather than its contents — `Open House` is 43 rooms on a 7×10 grid and nobody
  here can say whether its longest shortest-path is 9 or 19.

The scripted playthrough covers what those two rows were standing in for — a house that cannot be
crossed does not finish — so this is a gap in *reporting*, not in safety. But an author who has to
choose between "grep, and hope" and "write a 200-line Go test per house" will do neither.

**Amendment, 2.4: the dark-rooms row was computable the whole time, and the sentence above is a good
example of how to get this wrong.** Writing the second house needed the row, so it got another look.
Every word of the bullet is true — `GetNumberOfLights` *is* a method on `*render.Scene`,
`internal/render` *does* import `internal/house`, the dependency *cannot* be inverted, and
re-deriving the rule *would* be two copies of a switch over eighteen backgrounds. What none of it
establishes is the conclusion, because the conclusion is about where the *check* lives and the
argument is about where `internal/house` can import from. The check does not live in
`internal/house`. It lives in `internal/replay`, which imports `internal/render` already and cannot
run a game without it. Three lines:

```go
scene := render.NewScene(render.DefaultView(), render.NewAssets(nil), h)
...
if scene.GetNumberOfLights(int16(i)) == 0 { p.dark++ }
```

The empty asset tree is fine: `GetNumberOfLights` reads the room's background and its light objects'
switch states and touches no picture, so there is nothing for an asset tree to supply. Both houses
now assert the row — `internal/replay/profile_test.go` holds the shared counting — and it earned its
place immediately, on `Boarding House`'s Bell Turret: the room was `kRoof`, which is one of the eight
backgrounds that light themselves, and became `kPaneledRoom`, which is not, so changing a background
put out a light and no other row moved. Checked by eye with `glidertool render -all` it would have
looked like a room at night.

The correction does not touch the case for `glidertool house stats`, which is about an author wanting
the numbers *before* they have a passing test, and it does not touch BFS eccentricity: that one needs
`DetermineRoomOpenings` and a graph walk, is genuinely not three lines, and is what remains of this
item. The `LintOptions.Lights` callback proposed above is also still the right shape for the *linter*,
which does live in `internal/house` — the amendment is that a test does not have to wait for it.

The general lesson is cheaper than the specific one. "X cannot import Y" is a fact about two
packages; "this cannot be checked" is a claim about the whole repository, and the step between them is
"and there is no third package that imports both". That step was never taken, and it was false.

**Second amendment, 2.4: the other row was computable too, and the answer it gives is that `Open
House` misses its tier.** The remaining half of this item is done. `internal/profile` is the package
the two callers were always going to need — sixteen rows of arithmetic over `internal/house`, the
dark-room row through a `*render.Scene`, and the eccentricity row through a static room graph in
`internal/profile/graph.go` that drives `DetermineRoomOpenings` over every room and breadth-first
searches the result. `glidertool house stats` prints all eighteen, with `-tier` for the bands and
`-fail` for an exit status; `internal/replay`'s two house tests now ask the same package instead of
holding a hundred lines of tallies between them.

The first amendment's lesson applies again and the answer this time is better than a test. A graph
walk needs `internal/game`, and `internal/replay` imports it — so the three-lines-in-a-test move would
have worked a second time. It was the wrong move: the numbers are wanted *printed*, and a package
sitting above `house`, `render` and `game` that none of the three knows about costs nothing the test
was not already paying. What "cannot be checked" meant, both times, was "I have not looked for the
place where it can be".

**What the row says, now that it can be asked.** This entry's own words were that eccentricity is "the
row most worth having, because it is the only one that measures the *shape* of a house rather than its
contents — `Open House` is 43 rooms on a 7×10 grid and nobody here can say whether its longest
shortest-path is 9 or 19." It is **10**, and §10.2's tutorial band is 11-15, so the house misses a row
it has been claiming to hit. The miss is worth more than a pass would have been:

- The band is two houses. `Empty House` is 35 rooms at eccentricity 11, `Demo House` is 45 at 15, and
  `Sampler` — the third template-tier original — is 2 rooms and outside the tier's 35-45 room band
  entirely. Nobody chose 11-15.
- `Demo House` reaches 33 of its 45 rooms. `Empty House` reaches all 35. **`Open House` is 43 rooms
  and reaches all 43, inside 10 hops** — more rooms than `Empty House` inside a shorter diameter,
  which is a bushier graph and not a shallower one.
- Its star is at hop 9, in the Belfry, and the four deepest rooms are the roof edges just past it: the
  North and South Louvres, the South Slope and the East Balcony. `Demo House`'s one star sits at depth
  10 of 15 — a third of that house lies beyond its own goal. For a tutorial, having almost nothing
  past the star is the better of the two shapes.

So the house is not being changed, and `TestOpenHouseMatchesTheTutorialProfile` names the row as an
allowed miss with that argument attached. `checkTier` checks the exemption both ways: an unnamed miss
fails, and a named row that has stopped missing fails too, because an exemption whose reason has
expired is a paragraph that has started lying. This is 4.18's position applied to our own house rather
than to the corpus — §10.2 is a measurement of 22 houses and not a rule they obey, and the honest
response to a miss is to look at it and then say what you found.

**Four things the tool found on its first day, which is the case for having built it.**

- **§10.2's `prizes/room` band excludes the house it was measured from.** `Empty House` has one prize
  in 35 rooms: 0.0286, which §8.3 prints as 0.029 and §10.2 rounds into a band floor of 0.03. So
  `house stats -tier tutorial "Empty House.house"` reports a miss on a row whose band that house
  defines. `Band.Contains` is inclusive at both ends precisely so a corpus extreme passes its own
  band; rounding in the document defeats it. Not corrected — the table is a transcription and
  `TestTheTierTableMatchesTheDocument` holds it to the markdown — but it is the second known case, with
  tutorial's `~20 %` empty-rooms midpoint, where 10.2's cell is not quite a band. *(Pointing the same
  command at all 22 houses turned those two known cases into a property of the table, which is 4.25:
  the rounding goes inward at seven cells, and large's objects/room band of 6.0-7.5 contains neither
  of the two houses it was measured from.)*
- **This entry has miscounted 10.2 since it was written.** The table has eighteen rows, not fifteen;
  git says it has not changed since the document was written, so the number was never right. That is
  not pedantry, it is the mechanism: `TestOpenHouseMatchesTheTutorialProfile` asserted a list of
  sixteen rows somebody typed, and `prize:enemy` was simply absent from it — present in the
  `Boarding House` test, hand-written, and missing here, with no test able to notice. `profile.Check`
  iterates the table, so the eighteen are eighteen because the table says so.
- **`-no-assets` was warning everybody.** The closing line said the reachable and eccentricity rows
  read a custom-background room as sealed, whenever the flag was given. True for the seven houses with
  rooms that need a `'bnds'` resource and false for the other fifteen and for every house the port
  ships, whose rooms carry their own `bounds` field. `Profile.ForkBounded` counts them, the note names
  the number or says the measurement is exact, and the corpus total is pinned at 155 against §10.1
  step 10. A caveat that fires on every house is a caveat nobody reads.
- **The one corpus claim the command printed was false.** Beside the caveat it printed a second note,
  that 10.2's bands are a spread and "no shipped house is inside all eighteen of its own tier's" — a
  sentence truncated mid-clause, and wrong: `Leviathan` is inside all eighteen of epic's. The figure is
  21 of 22, it is now measured rather than assumed, and the three mechanisms behind it are 4.25. The
  general lesson is the specific one this entry keeps arriving at from different directions: a claim
  about the corpus that lives in a `fmt.Printf` has nothing holding it, and this project's habit of
  pinning such claims to a test is what the tool now makes possible for claims about *tiers*.

### 4.17 A house with no custom art is told its custom art will fall back — **note; cosmetic, 2.3**

Rendering `Open House` prints `no extracted resource fork at built-in:houseart/Open House; custom
art will fall back` before every image. It is true and it is useless: the house uses no art above
2017, so there is nothing to fall back, and the one house in the library that provably needs no
warning is the only one that gets it on a complete asset tree. The same line is in two places
(`cmd/glidertool/render.go:85`, `cmd/glidergo/play.go:432`) and both fire on "the directory is
absent" when the condition they mean is "the directory is absent *and* this house asks for something
that would have been in it" — which is one pass over the rooms for a `background >= kUserBackground`
and over the objects for a `kCustomPict`, `kTV` or `kSoundTrigger`, all of which `internal/house`
can already see (`lint.go:783` does the `kCustomPict` half). Left as a note because the message is
suppressed by `-quiet` and misleads nobody who reads the next line, but it is the first thing a new
house's author sees and it tells them they have done something wrong.

### 4.18 §10.3's construction procedure and §10.2's tier table contradict each other, and a tutorial house cannot satisfy both — **note; found by following them, 2.3; narrowed to the tutorial column by the second house, 2.4**

Three of the recipe's steps give per-room object counts that are corpus *means*, and the corpus mean
is 7.7 objects a room. The tutorial column of the table two subsections above allows **2.2-3.1**.
So:

- step 5, "interiors get 11-13 objects" — `Open House` is 72 % interior; at 11 apiece that is 340
  objects in a house whose ceiling is 138, and 7.9 a room.
- step 10, "decorate each star room to ~15 objects (corpus mean 14.75)" — 15 objects is more than a
  tenth of the whole tutorial budget in one room, and §10.2's own "rooms at the 24 ceiling: 0" row is
  the only constraint that survives.
- step 12, "set `firstRoom`'s contents richer than average: 13 objects" — the same figure, and the
  only one of the three whose *intent* survives translation, because "richer than average" is a
  ratio and 13 is not. Ours holds 4 against an average of 3.09, which is the rule; 13 would be a
  third of the tier.

None of this is wrong in §10.3, which was written from the corpus as a whole and says so. It is
unusable as written for the smallest tier, which is the tier a first new house should be, and the
person most likely to follow it step by step is the person least likely to notice that step 5 and the
table disagree. The fix is one sentence per step making the figure relative — "interiors carry 3-4×
what the air rooms do", "the star room is the densest room in the house", "`firstRoom` is above the
house's own mean" — and a line under the table saying the numeric steps below scale with the tier.
`TestOpenHouseMatchesTheTutorialProfile` encodes the ratio reading for the two that are checkable
(start room above average, no enemies in it), so the disagreement is at least pinned on one side.

**Amendment, 2.4: the contradiction is a property of the tutorial column, not of §10.3, and writing a
house at the tier above settles which.** `Boarding House` is 10.2's **small** tier, whose objects/room
band is 7-19 against the tutorial's 2.2-3.1. At that tier step 5 is not merely satisfiable, it is
close to forced: the house's 35 rooms that have a ceiling average **10.00** objects apiece, 24 of them
holding 11 to 13 — step 5's figure, taken literally, with no adjustment — and the whole-house average
lands at 7.61, comfortably inside 7-19, because the 16 open-air rooms average 2.38. Step 10's 15-object star room and step 12's 13-object `firstRoom` are
likewise ordinary numbers at this tier rather than a third of the budget.

So the three numeric steps are not written against the corpus "as a whole" in a way that fails
everywhere. They are written against the corpus mean, 7.7 objects a room, and the small tier *is* the
mean. The tutorial column is the outlier — it is the only tier whose object budget is a third of the
corpus's — and a recipe stated in absolute numbers will contradict exactly that one column and no
other. That is a smaller and more useful defect than "§10.3 and §10.2 contradict each other": the fix
is still the sentence-per-step rewording proposed above, but it now needs only one line under the
table, and it can say which tier it is protecting.

Worth recording as method rather than as content: this was not settled by re-reading either section.
It was settled by writing a second house at a different tier and measuring it with the same code
(`internal/replay/profile_test.go`; the measurement is `internal/profile` since 4.16, and 4.25 is what
asking it about all 22 houses at once then found), which is the only way a claim about a *table of
tiers* can be tested at all. One house fitted to one column cannot distinguish "the table is usable" from "the
column was chosen after the fact".

### 4.19 Two lines in an authored house are its players' saved games, and nothing said so — **note; found before it could cost anything, 2.3; the pin DONE**

A house that ships is a house people save games into, and the port keys those side-cars on the
*house* rather than on the file it came from. `saved.Store.Path` is the house's name,
percent-escaped (`internal/saved/store.go:85`, via `datadir.FileName`), and `Check` then gates the
save on two facts: the name in the saved block matching the house being opened, case- and
diacritical-insensitively (`internal/saved/store.go:390-393`), and the house's stamp matching the
one the save carries (`internal/saved/store.go:401-407` — gate 2, the original's
`kYellowSavedTimeWrong`, `SavedGames.c:237`). High scores are keyed on the name alone:
`scores.FileName` takes a house name and nothing else, and nothing in `internal/scores` compares a
board against the house's stamp. It does keep timestamps — one per row, the day that score was set
(`internal/scores/scores.go:151`) — and none of them is ever checked against `h.TimeStamp`.

Which makes two ordinary-looking lines of an authored house a compatibility surface, and neither
looks like one:

- **`timestamp 1725439552`** (`levels/Open House.house.txt:276`) is the save key. Change it, to any
  value, for any reason, and every saved game of that house in the world is refused with "Open House
  has been modified since this game was saved". That refusal is *right* — a save holds object states
  by room and object index, and an edited house may have moved both — which is exactly why the field
  cannot be treated as cosmetic. High scores survive it; saves do not.
- **The house's file name** is the other half, and it is both keys at once. Renaming `Open
  House.house` orphans the saves *and* the score board together and says nothing, because the new
  name simply has no side-car and a house with no side-car is a house nobody has played.

None of this is a deviation and none of it is a bug: the 1994 program gates on the same field, and
its high scores lived inside the house file, which is why they never needed a stamp. What was
missing was anybody writing it down before the first house shipped — while the cost of a tidy-up
commit to a header is zero, rather than after, when it is every player's progress. It is now in
three places: the comment on the field in the house's own text, CONTRIBUTING's Houses section, and
here.

**The guard is built, because the hazard is an accident and not a decision.**
`TestShippedHousesKeepTheStampTheirSavesAreKeyedOn` (`assets/assets_test.go`) holds a table of every
shipped house's published stamp and fails if the authored text disagrees — and fails just as loudly
for a house with no row, so a second house cannot be added without a decision being made about it.
The failure message is the whole point of the test: it says what the change costs and that the fix
is to put the old number back. A `house lint` check was the other candidate and is the wrong shape;
a linter sees one house and cannot know what that house's published stamp was.

### 4.20 A house that ships inside the binary can only ever be named, and one tool could not take a name — **found and DONE, 2.3**

The moment a house lives in `assets/levels.zip` rather than on a disk, "attach the file" stops being
something a bug report can do. The name is all there is. `glidergo -house "Open House"` was fine —
`resolveHouse` walks the same sources the picker does (`cmd/glidergo/main.go`, `inSources`) — but the
tool built for bug reports was not:

```
$ glidertool replay -house "Open House" -frames 60
glidertool: open Open House.house: file does not exist
```

`internal/replay` resolved a name in the houses root alone, so it named a file that does not exist
anywhere, for the one class of house whose whole point is that it does not exist as a file. A
reporter's only way out was to find a clone, run `make levels` and pass a path — which is to say,
there was no way out for a reporter.

**Fixed, and the shape of the fix is the thing worth keeping.** `replay.Script` now carries the
second root the same way it carries the first (`Levels` beside `Tree`, `LevelDir` beside `HouseDir`,
`leveldir` in the text format, `-levels` on `glidertool replay`), and the lookup tries the houses
root and then the levels root — `cmd/glidergo`'s order, and the same tie the picker's sort breaks the
same way, so an original wins a name it shares with a new house. Only `fs.ErrNotExist` falls through
to the second root: a house that is there and will not parse is the answer. When neither root has it
the error names both roots, because "file does not exist" about a name somebody typed omits the one
fact they need.

The general rule, for the next tool: **anything that accepts a house by name must search every
source the picker does.** Two things in the tree accept a house by *path* only and are correct as
they stand — `glidertool render <house>` and `glidertool house <subcommand> <file>` — because in a
clone every house has a path (`assets/extracted/houses/` is committed and `make levels` writes
`assets/levels/`). They would still be worth revisiting if either grew a `-house NAME` form.

### 4.21 The paragraph justifying the linter's severities cites four examples: one has no instance in the corpus, and one is calibrated as an error and fails it — **found and DONE, 2.4; the narrower rule it yielded is what calibrated 4.22**

`internal/house/lint.go:19-24` is the argument for how the severities were chosen, and the argument
is a good one:

> Severity is calibrated against the 22 shipped houses rather than against an ideal, because those
> houses are the specification of what a playable Glider PRO house is. They contain 189 dangling
> links, an out-of-range `who`, staircases that lead nowhere and a basement you cannot climb out of
> — **so none of those classes can be an error** without making the corpus fail its own linter, which
> would make the linter useless on the day it shipped.

Two of the four examples hold up. 189 dangling links is a count that reproduces, and the basement is
Slumberland's and has a test of its own (`acafec7`, "Pin Slumberland's inescapable basement as the
original's design"). The other two do not, in opposite directions.

**The staircases have no instance.** The linter has four stair rules — `stairs-no-room` and
`stairs-unpaired` at warn, `stairs-doubled` twice at note — and across all 22 houses, 4,070 rooms and
**326 stair objects** they fire **zero times**:

```
$ bin/glidertool house lint assets/extracted/houses/*.house | grep -c stairs-
0
```

Every staircase in the corpus is paired and both ends land in a room that exists. The checks are not
wrong; they were derived from `CheckForStaircasePairs` (HouseLegal.c:961-1045) and they fire on houses
written to provoke them. The phrase is most likely a paraphrase of that function *existing* — Calhoun
wrote a repair pass for this, which implies he had seen it somewhere — but as written it reads as a
measurement of these 22 files, and it is not one.

**The `who` is the reverse, and it is the more consequential half: it is calibrated as
`SeverityError`, it fires, and the corpus therefore does fail its own linter.**

```
$ bin/glidertool house lint assets/extracted/houses/*.house
    error link-slot-range   room 72 "Let's Roll" slot 22: kMailboxRt links to slot 35 of room 72,
                            which holds 24 slots ...
total: 22 files, 637 notes, 48 warnings, 1 errors
glidertool: a finding reached error
$ echo $?
1
```

(The note total was 634 when this was filed and is 637 now: 4.22's `starfield-tiles` fires three
times on Leviathan. Updated here rather than left as written, because a transcript nobody can re-run
is the defect this entry is about.)

One file, `CD Demo House.house`, one object — exactly the "holds exactly one of these" the check's own
message claims (`lint.go:692-699`). So the sentence's conclusion is false of the code it introduces,
and by the sentence's own reasoning the linter was useless on the day it shipped.

It was not, and that is the part worth getting right rather than reverting. `link-slot-range` **should**
be an error: `GenerateRetroLinks` indexes `retroLinkList[who]` with no bound check (House.c:577, :600),
so the original reads into the next room's record — this is a memory bug in a shipped file, not a
stylistic liberty, and a linter that shrugged at it would be the useless one. The calibration rule the
code actually follows is narrower and better than the one the comment states: *a class the corpus
exercises deliberately cannot be an error; a class the corpus exercises by overrunning a buffer can.*
Nothing flagged the mismatch because `make houses` runs `glidertool house check`, which round-trips
and sanity-checks and exits 0, and no target in the tree runs `house lint` over the originals.

**The remedy is smaller than it first looked, because the test that settles it already existed.**
`TestLintCorpus` (`internal/house/lint_test.go:798`) walks all 22 houses and pins a table of per-check
counts — including `link-slot-range` at 1, with the comment "The one error in 4,070 rooms". So the
tree has known since that test was written that the corpus produces an error; the contradiction was
eight hundred lines from the sentence asserting it could not. What the table did *not* hold was a row
for any stair rule, which is why the other half went unexamined for as long as it did.

**Done, 2.4:** three rows added — `stairs-no-room` 0, `stairs-unpaired` 0, `stairs-doubled` 0 — so the
header's claim is now a number a change has to argue with. They are safe to pin at zero because
`TestLintStairs` exercises all four rules on houses written to provoke them, so a zero in the corpus
row is a fact about the corpus rather than a dead check.

**Done, 2.4 (the other half):** `internal/house/lint.go:19-39` rewritten. Three changes, and the
middle one is the point:

- The staircase example is **dropped**, with a sentence saying it was there and why it was wrong —
  the phrase belonged to `CheckForStaircasePairs` existing, not to these 22 files. Dropped rather
  than rewritten because at note and warn those rules need no corpus evidence to justify them, so
  there is nothing for a replacement example to do.
- The narrower rule is now **stated**: *a class the corpus exercises deliberately cannot be an
  error; a class it exercises by overrunning a buffer can.* That is what the code has always done,
  and writing it down is what makes the next check's severity a decision rather than a guess — it
  is the rule 4.22's three new checks were calibrated with, one week later.
- The `who` is no longer offered as a class that *cannot* be an error. It is named as the one
  instance of the second kind, with `House.c:577, :600` for why, and the paragraph now says plainly
  that `house lint` exits 1 on the originals **by design**. A reader who runs the command gets the
  documented answer instead of a contradiction.

The lesson generalises past this file: the sentence was wrong because it argued from a list of
examples, and a list of examples goes stale silently. Two of the four were still true. What would
have caught it at the time is what caught it in the end — running the command the prose implies.

Worth keeping as a pattern: **a comment that cites evidence should cite it precisely enough to be
re-run.** Both defects here sit in examples that read exactly like the two sound ones, and both
surfaced only by running the commands the prose implies. What prompted it was writing a house with
eight staircase pairs, which meant reading the stair checks closely enough to wonder what they had
ever caught.

### 4.22 The linter has no check that an object makes sense for the background it stands in, and the two rules a house author gets wrong first are both in that class — **found and DONE, 2.4; the census widened one rule from three objects to twelve and demoted the other to a note**

`glidertool house lint` has 29 checks and all of them are about a room's *own* fields being legal:
`what` in range, links resolving, `tiles[]` inside the background picture, names fitting `Str27`. Not
one relates an object to the background it sits in. Two configurations that are illegal by the
corpus's own unanimous practice therefore lint clean:

```
$ bin/glidertool house lint /tmp/floorless.house
/tmp/floorless.house: 1 rooms, 0 notes, 0 warnings, 0 errors
```

That house is a `kSky` room holding a `kFloorVent`. `DoesRoomHaveFloor` (`internal/game/room.go:335-348`)
gives `kSky`, `kStratosphere` and `kStars` no floor at all, so the vent's art stands on a hole — and
across the corpus's **1,084** `kRoof`, `kSky`, `kStratosphere` and `kStars` rooms there is not one
`kFloorVent`, `kFloorBlower` or `kSewerGrate`. Zero of 1,084 is not a style preference, it is a rule
every author in 1994 followed without being told. The second case is the same shape: `kStars` with
anything but the identity tiling, where **243 of the corpus's 246** `kStars` rooms are the identity
because the background is one 512-pixel picture, and that also lints clean.

Both were found the hard way — the generator for `Boarding House` produced a `kStars` room carrying
`kSky`'s tiles *and* a `kFloorBlower`, one room with both defects, and nothing in the toolchain
objected. What caught it was a pair of assertions added to the generator, which is a throwaway script:
the checks are now in `/tmp` and the knowledge is in a house comment, which is the wrong place for
both. A hand author gets neither.

What makes this worth a check rather than a note in a document is that the failure is invisible in the
direction people test. The room draws — `kFloorVent` has a picture and it composes fine — so
`glidertool render` shows a vent, the house lints clean, the lift works because `LiftIt` reads the
object and not the floor, and the only thing wrong is that a player sees machinery bolted to the sky.
Nothing in `make check` can fail.

The shape of the fix is `lint.go`'s existing per-object switch plus one table: for each background,
the object groups that cannot stand in it. It is cheap because the two rules above are the whole of
what the corpus establishes unanimously — and the corpus is what decides the severity, as it does for
every other check here. Both should be **warn**, not error, on the evidence: zero instances in 4,070
rooms means no shipped house is made to fail, and a warning is what `-min warn` shows an author by
default. Worth doing at the same time as 4.16's `glidertool house stats`, since both are "what a house
author needs before they have a failing test" and both want the same walk.

**Done, 2.4 — three checks, and the census moved two of the three decisions above.** `mount-no-floor`
and `mount-no-ceiling` at warn, `starfield-tiles` at note, with `TestLintMounting` and
`TestLintStarfieldTiles` firing each on houses written to provoke them and three new rows in
`TestLintCorpus` pinning 0, 0 and 3. The corpus note total moves 634 → 637; `README.md`,
`CONTRIBUTING.md` and `docs/PLAN.md` are updated, and so is 4.21's transcript.

What the entry above got right: the floor rule is real and the corpus is unanimous about it. What
measuring changed:

**The object set is twelve types, not three, and the classification is not a guess from the names.**
Each of these occupies exactly **one** vertical coordinate across all 4,070 corpus rooms, and that
coordinate plus the height of its artwork puts it against one surface or the other:

| floor-standing | v | height | foot | ceiling-hung | v | height |
|---|---|---|---|---|---|---|
| `kFloorVent` | 305 | 11 | 316 | `kCeilingVent` | 8 | 11 |
| `kFloorBlower` | 304 | 15 | 319 | `kCeilingBlower` | 5 | 15 |
| `kSewerGrate` | 303 | 17 | 320 | `kCeilingLight` | 4 | 20 |
| `kGrecoVent` | 303 | 18 | 321 | `kFlourescent` | 12 | 12 |
| `kSewerBlower` | 292 | 12 | 304 | `kTrackLight` | 5 | 24 |
| `kHipLamp` | 23 | 276 | 299 | | | |
| `kDecoLamp` | 91 | 212 | 303 | | | |

The two lamps are the interesting entries. `kHipLamp` is 276 pixels tall and `kDecoLamp` 212, so both
are *standing* lamps whose feet land on the floor line — a name-based rule would have filed them with
the ceiling fixtures and been wrong twice. `kTableLamp`, `kLightBulb` and `kInvisLight` are in neither
set precisely because their v *does* vary: a table lamp sits on whatever furniture the author put
under it. The five updraughts and two downdraughts are also exactly how `CreateActiveRects` groups
them (`internal/game/hotspots.go`, from `ObjectRects.c`), so the two halves of the classification
agree from independent directions.

**The five flames are deliberately excluded**, and this is the case that would have made the check
fail the corpus. `kTaper`, `kCandle`, `kStubby`, `kTiki` and `kBBQ` make a thermal column the same way
an updraught does, so a rule derived from the physics would include them — but 18 `kTiki` and 9 `kBBQ`
stand in corpus rooms with no ceiling, and one `kCandle` and one `kStubby` in rooms with no floor. A
torch on a lawn is a torch on a lawn.

**`kRoof` is in the ceiling list and not the floor list**, which is why the two are separate maps
rather than one predicate. A roof has a floor — you walk on it — and no ceiling. And `kSkywalk` and
`kDirt` are in neither despite looking outdoor, which is a fact about `Room.c:1138-1206` and not
something a check derived from how a background looks would get right.

That also retracts the **1,084** above: it is the count of `kRoof` + `kSky` + `kStratosphere` +
`kStars` rooms, and `kRoof` does not belong in a floor tally. The floorless corpus is **903** rooms
and the ceilingless one **1,220**, which are the two numbers the findings quote. The original figure
was assembled by reading the backgrounds that look outdoor rather than by reading
`DoesRoomHaveFloor`, and it is the smaller version of the same mistake the object classification
would have made from the names.

**The second rule was wrong about severity, and measuring inverted it.** The entry above asks for
warn on "`kStars` with anything but the identity tiling". Two measurements say note:

- Every built-in background is exactly 512 pixels wide — eight `TileWide` columns — so `tiles[]`
  selects eight of eight and can never be out of range in a built-in room. `tile-column` cannot fire
  on one, which is *why* nothing caught the generator's mistake.
- The corpus is not unanimous, and the exceptions look deliberate: 62 of 62 `kStratosphere` rooms are
  the identity, but only 243 of 246 `kStars` rooms, and all three exceptions are Leviathan's — rooms
  172, 206 and 221, one a straight reversal of 0..7. By the rule 4.21 just made explicit, a class the
  corpus exercises deliberately cannot be more than a note.

So the finding is worded to fire without accusing, and each background quotes *its own* tally rather
than one standing in for the other. That distinction only exists because the census was run per
background; a single "243 of 246" in the message would have been the imprecise citation 4.21 is about,
in a check added to settle 4.21.

**One documented limit: only built-in backgrounds are checked.** A user-art room's openings come from
its own `bounds` field, or the background's `'bnds'` resource when that field is 0 (`boundsCode`), and
`internal/house` has `PictSize` but nothing analogous for `'bnds'`. Rather than check half the rule on
half the rooms, `mounting` returns early outside 2000–2017 and `TestLintMounting` has a row asserting
it stays silent there. That is a false negative by choice; a false positive would be a bug.

**Still open, and now a separate finding — see 4.23.** The table above is a stronger fact than the
check uses. Every one of those twelve types sits at one v in 1,458 `kFloorVent`, 507 `kSewerGrate` and
so on down — so a vent at v 200 in an ordinary room is floating art, and nothing in the toolchain says
so. That is the more likely authoring mistake and it is *not* what these checks catch.

### 4.23 Twelve object types occupy exactly one vertical coordinate in all 4,070 shipped rooms, and nothing in the toolchain knows it — **note; found by measuring 4.22, 2.4; both blockers answered and DONE, 2.4 — the rule is 24 types wide, one of the two questions was answered wrongly here, and a second rule came out of it**

4.22's census produced a fact it did not need and could not use. Across all 22 houses, every placement
of each of these is at one `v` and no other — not clustered, not mostly, **one value**:

```
kFloorVent      1458 placements, all at v 305        kCeilingLight   160, all at v  4
kSewerGrate      507 placements, all at v 303        kFlourescent    115, all at v 12
kSewerBlower     191 placements, all at v 292        kTrackLight      81, all at v  5
kGrecoVent       116 placements, all at v 303        kCeilingVent     28, all at v  8
kFloorBlower      83 placements, all at v 304        kCeilingBlower   12, all at v  5
kDecoLamp         62 placements, all at v  91
kHipLamp          26 placements, all at v  23
```

That is 2,839 placements with zero exceptions, across 22 houses credited to four authors. It is not a
coincidence and it is not house style: `GetObjectRect` places these at the absolute coordinates they
store (`ObjectRects.c:32-273`), and every built-in background is the same 512×322 picture size — so the
floor line and the ceiling line fall in the same place in every room, and an object resting against
either one has exactly one `v` available to it.

The two lamps are what make that argument checkable rather than assumed. Their `v` is 23 and 91, nowhere
near the five updraughts' 292–305 — but they are 276 and 212 pixels tall, so their *bottoms* are 299 and
303, which lands them inside that same band. Two objects whose top edges say "ceiling" and whose bottom
edges say "floor", agreeing with the vents once their height is taken into account, is the one thing in
this census that could not have come out right by accident.

**Why this is worth a check and 4.22's rules do not cover it.** 4.22 catches a vent in a room with no
floor. It says nothing about a vent at `v 200` in an ordinary room, which is *floating art* — and that
is the far more likely mistake, because it is what a typo or an off-by-one in a generator produces. The
port's own `Boarding House` generator had both bugs available to it; only the background one happened
to fire. `glidertool render` draws the room happily, the lift column is computed from the object's own
data so the vent works, and `make check` cannot fail.

The three objects deliberately **not** in the list are the evidence that the rule is real rather than an
artefact of counting: `kTableLamp` (70 placements, v 29..245), `kLightBulb` (252, v 0..294) and
`kInvisLight` (1,764, v 0..306) all vary freely, and they are exactly the three that do not rest against
a room surface — a table lamp sits on whatever furniture the author put under it. A census that found
*everything* fixed would mean the census was wrong.

**What has to be settled before this becomes a check, and why it is filed rather than written.** The
numbers above are a measurement of 22 files, not a citation of the C, and 4.21 is the entry about
exactly that distinction — a rule argued from examples goes stale silently. Before a `object-v` check
ships, two things need finding:

1. **Where the floor line actually is**, in the C, as a constant rather than as seven bottom edges that
   nearly agree. `internal/render/view_test.go:16` already names `kFloorVentTop 305` and
   `kShadowTop 306` as authored-against values, which is a start and not a derivation.
2. **Whether the original's editor enforced it.** If the room editor snapped these objects to the floor
   — which would explain 2,839 of 2,839 — then the check is restating a tool's behaviour and should say
   so, and its severity follows from that rather than from the count. `HouseLegal.c` does not do it;
   whether `ObjectEdit.c` or the palette does is unread.

Until then the honest status is a note with the numbers in it. **Severity if it ships:** warn, by the
rule 4.21 made explicit — zero corpus exceptions means no shipped house is made to fail, and an object
drawn floating is something a player sees.

*(4.23, **DONE**, 2.4. Both questions above are answered, and the answers are better than the note
expected in one direction and wrong in the other. `object-top` and `object-left` ship in
`internal/house/lint.go`, with `fixedTops`, `fixedLefts`, `TestLintObjectAnchor`,
`TestLintObjectAnchorFloorTransStray` and `TestCorpusFixedAnchors`.*

***1. There is no floor line.** The question assumed one constant that seven bottom edges agree on.
There are **nineteen**, one per object type, and the census values are not "nearly" those constants —
each is exactly its own. `GliderDefines.h:467-481` carries thirteen, `:483-494` four more, and
`ObjectAdd.c:19-23` defines the last two locally in the only file that uses them, which is a fair
summary of how much of 1994's geometry is editor-side rather than engine-side. So the table in
`fixedTops` is a transcription with a per-row citation, not a measurement — which is what 4.21 asked
for. `internal/render/view_test.go:16` naming `kFloorVentTop 305` was right about the value and was
reading it out of the render code's own history; the constant it half-remembered is
`GliderDefines.h:467`.*

***2. Yes, three ways over — and "`HouseLegal.c` does not do it" above is wrong.** The editor enforces
this at creation, during editing, and on save.*

- ***Creation.** `AddNewObject` assigns the constant (`ObjectAdd.c:99-110`, `:128-131`, `:311`, `:341`,
  `:355`, `:374-440`, `:517-564`). One type is missed: `kSewerGrate` has no branch in the `if`/`else`
  chain at `:99-110` at all, so a fresh one keeps whatever the empty slot held. It is invisible in
  practice because the function ends by calling `KeepObjectLegal` (`ObjectAdd.c:774`), which repairs
  exactly that type — 1994 covering for itself two hundred lines apart.*
- ***Editing.** `DragObject` moves these types **horizontally only**: their case arms add `deltaH` to
  `topLeft.h` and simply do not mention `topLeft.v` (`ObjectEdit.c:565-573`, `:632-645`, `:668-674`).
  This is the one the note guessed at, and the contrast is two arms away in the same `switch` and the
  same union member — `kLeftFan`, `kRightFan`, the five flames, `kInvisBlower` and `kLiftArea` get both
  deltas (`:575-586`), and so do `kTableLamp`, `kLightBulb` and `kInvisLight` (`:676-681`). The three
  the note named as "the evidence the rule is real" are, in the C, the three cases that take `deltaV`.
  The census and the source agree line for line.*
- ***Save.** `KeepObjectLegal` **rewrites** a wrong `v` back to its constant for four types
  (`HouseLegal.c:115-137`), and runs from `CheckHouseForProblems` on every `WriteHouse`
  (`HouseIO.c:470`). The note's claim that `HouseLegal.c` does not do it was made from a search for a
  floor line rather than for the type names, which is how a negative finding usually goes wrong.*

*So the check does restate a tool's behaviour, exactly as the note predicted it might, and that is the
argument **for** shipping it rather than against: gliderGo's houses are not written by that tool. They
are typed into `levels/*.house.txt` with the coordinate spelled out — `object 2 kFloorVent at 305 120`
— and `make levels` lints with `-fail warn`, so from now on a typo in that column fails the build
instead of shipping a vent hanging in the air. Verified by doing it: changing one `305` to `300` makes
`make levels` exit 1 with the finding. Both of the port's houses pass unchanged.*

***Severity: warn, as predicted, but now argued from the ladder rather than from the count.** The
ladder in `lint.go` reserves error for a house the loader has to repair or range-check before it can be
played at all. A wrong `v` needs no repair: the room loads, `GetObjectRect` uses the stored coordinates,
the art composes, the lift column comes from the object's own `distance`. And it is not a note, because
the result is in the way rather than merely untidy. Middle rung.*

***The rule is twice as wide as the census suggested.** 24 types, not twelve, because 4.22's census only
covered the objects that mount against a surface. Beyond the twelve: `kUpStairs` and `kDownStairs` at
`kStairsTop` 28, `kCeilingTrans` at 6, `kFloorTrans` at 302, the four doors at 0 and the four windows at
64. That is 4,041 placements rather than 2,839, and 3,998 of them are at their constant exactly.*

***And a second rule fell out of it.** The four doors and four windows have named **horizontal**
constants too — `kDoorInRtLeft` 368, `kWindowExLfLeft` 0 and six siblings
(`GliderDefines.h:484-494`) — and `KeepObjectLegal` snaps a dragged one to the nearer wall on every
save, rewriting `what` to match the side it chose, so dragging a left door across the room turns it into
a right door (`HouseLegal.c:310-369`). All 167 shipped doors and windows are at their constant. That is
`object-left`, shipping in the same commit: a door is one of a room's exits, so it belongs against a
wall and nowhere between. Both repairs — this one and the four vertical ones — neglect to set
`KeepObjectLegal`'s `unchanged` flag, so the original could silently move your door and report the house
as untouched.*

***The 43 exceptions are a 1994 bug, filed as 4.27.** 43 of the 244 shipped `kFloorTrans` objects are at
`v` 300 rather than 302 — 37 in Slumberland, 5 in Leviathan, 1 in Rainbow's End — because the repair
that would have normalised them is unreachable. `object-top` accepts 300 for that type and only that
type, as a value rather than a tolerance; 299 still warns.*

***Five types left out, and one pair, both for stated reasons.** `kCounter`, `kDresser`, `kManhole`,
`kMousehole` and `kFireplace` are just as immovable — their `DragObject` arms are `h`-only too — but the
original names their **bottom** (`kCounterBottom` 304, `kDresserBottom` 293, `kManholeSits` 322,
`kMouseholeBottom` 295, `kFireplaceBottom` 297), and a bottom becomes a top only by subtracting the
height of the artwork, which `internal/house` cannot see: it reads house files, and the sprite rects
live in `internal/render`. Putting the five differences in as literals would turn a table of citations
back into a table of measurements. `kBalloon`, `kCopterLf` and `kCopterRt` are `h`-only as well
(`ObjectEdit.c:701-705`) and are left out for a different reason: nothing repairs them, and
`AddDynamicObject` overwrites the vertical position at run time regardless of what the file says —
`kBalloonStart` for the balloon (`Dynamics3.c:395`), `kCopterStart` for the two helicopters (`:417`). A
stored coordinate with no consequence is nothing to report.*

***One thing this changed that was not the subject.** Every synthetic fixture in `lint_test.go` placed
its objects at `v` 0, so the new check fired 98 times across 37 subtests the moment it was wired in. The
fixtures were fixed rather than the check exempted: `plain`, `transportTo` and `switchTo` now run their
object through `anchored`, which puts it where the 1994 editor would have. That is what the file's own
header says the synthetic half is for — "the smallest house Lint has nothing to say about, and then
break exactly one thing in it" — and 37 of those houses had quietly stopped being that.)*

---

### 4.24 Eight analysis documents state a resource's layout, one of them warns that getting it wrong opens the wrong walls, and the port got it wrong — **found and DONE, 2.4**

`internal/render.Assets.Bnds` read the `'bnds'` resource as an eight-byte QuickDraw `Rect` of
big-endian `int16`s, on the reasonable-looking grounds that the C declares its handle `boundsHand`.
It is not a `Rect`. `boundsType` is four one-byte `Boolean`s — `left`, `top`, `right`, `bottom`
(`GliderPRO/Headers/GliderStructs.h:266-272`) — and every one of the 70 resources the shipped houses
carry is exactly four bytes. So the guard `len(raw) < 8` rejected **all 70**, `GetOriginalBounding`
answered 0 for every lookup, and 0 means closed on all four sides.

**What that does to the shipped game.** 155 rooms across 7 houses take that fallback — a room with a
house's own background and no `bounds` field of its own — and **146 of them lost at least one exit they
should have had**: 141 the left wall, 138 the right, 109 the ceiling, 79 the floor. Castle o' the Air 55
rooms, Rainbow's End 30, Slumberland 24, Land of Illusion 21, Demo House 10, Leviathan 9, The Asylum Pro
6 — which includes the four largest houses in the corpus by room count and the two most likely to be
opened first.

The 79 are the ones a player would have met. A closed bottom *is* a floor, so those rooms were given one
the original does not give them: the glider could not fall out of a room designed to drop it, and
`IsShadowVisible` drew a shadow on a floor plane that is not there (`Room.c:1103-1133`). The other nine
of the 155 are genuinely sealed, by a resource of four zero bytes, and were the only rooms the broken
decode got right by accident.

**The documents were right, in twenty-nine places, in eight of the thirteen that mention the
resource.** `docs/analysis/interactions.md` §20.5 is titled "`'bnds'` resources — the 4-byte fallback"
and prints a decoded table of all nine of Castle o' the Air's, byte values and resulting thresholds
included. `docs/analysis/constants.md:6224` says outright: "A port must read `'bnds'` from the house's
resource fork; hard-coding 0 breaks room openness in those eight houses." `docs/analysis/house-format.md`
§4.2.1 quotes the struct. And `docs/analysis/structs.md:4332` is item **3** of a numbered list of the
traps a porter falls into, whose whole text is this bug:

> **3. `'bnds'` is `{left, top, right, bottom}` — a *different* order from `Rect`.** The one place in
> the format where the QuickDraw convention does *not* apply (§9.3, `GliderStructs.h:266-272`). Four
> bytes, easy to get backwards, and the symptom is rooms with the wrong walls open.

Written in advance, about this function, naming the symptom. Nothing read it back.

**Why nothing caught it, which is the part worth generalising.** Three things had to line up:

1. **`Bnds` had no test.** It is the only function in `internal/render` that decodes raw resource
   bytes at all — everything else in the package goes through PNGs — and it was the one with nothing
   driving it. Its one caller, `GetOriginalBounding`, had no test either.
2. **A golden hash over the corpus passed.** `TestObjectGraphEveryRoom` hashes `LeftThresh`,
   `RightThresh` and the four opening flags for all 4,070 rooms, so it covered every affected room and
   said nothing — because a golden pins *change*, not correctness. It had recorded the wrong answer as
   the expected one from the day it was written. Fixing this moved six of its 22 house digests, which
   is the only signal it was ever going to give.
3. **The wrong answer is a plausible answer.** "Sealed" is what an unfinished house looks like, and
   `docs/analysis/original-houses.md` §3.6 already reports that every corpus house has unreachable
   rooms, so a low reachable count is not by itself suspicious. There was no screen that looked wrong.

**What found it was a second implementation with published numbers.** `tools/probe_houses_inventory.py`
transcribed `DetermineRoomOpenings` from the C independently, and §3.6 publishes its per-house figures
including rooms-reachable and BFS eccentricity. When `internal/profile`'s Go walk (4.16) was checked
against that table, six houses came out short — and it was *which* six that named the cause, because
they are six of the eight houses that ship `'bnds'`. Two independent decodings of the same bytes,
compared by a test, is the pattern; a golden hash of one decoding is not.

**What shipped.** `render.Bounds` is now a struct of four `bool`s and not a `Rect`, so the type itself
refuses the mistake. `TestBndsIsFourBooleansAndNotARect` drives the decoder from bytes, with cases
chosen to fail under the old reading rather than merely to pass under the new one.
`TestEveryShippedBndsResourceLoads` asserts all 70 are four bytes of 0/1 and that all 70 are found.
`TestTheBndsFallbackFiresAndFindsItsResource` pins the 155-room census per house, checks the flags
arrive as the openings `DetermineRoomOpenings` reads, and confirms
`docs/analysis/houses-inventory.md`'s second claim — that no shipped room ever actually hits the
missing-resource default. Nine of the 155 *are* told they are sealed by a resource of four zero bytes,
and none of the nine is a trap: The Asylum Pro's six, plus Demo House's "Windows" and Slumberland's
"Paul's Room" and "ssalG gnikooL", which leave by a staircase, a window and a `kInvisTrans`
respectively. `tools/extract_house_art.py` had the same wrong assumption in its manifest, where a
`">4h"` unpack over-read a four-byte resource into the next one and published garbage coordinates
beside a correct `"bytes": 4`; it now records the four flags by name.

**Filed, not written: a check that a documented layout and its decoder agree.** The analysis documents
quote dozens of on-disk structures with stated byte widths, and nothing compares any of them to the Go
that reads them. The cheap version is not a parser — it is a per-resource-type assertion that the
extracted tree's records are the width the document claims, which is two lines each and would have
caught this the first time the assets were extracted. The expensive version, a doc-to-struct
comparison, wants the documents to carry machine-readable layout blocks and is 4.12's citation-checker
argument one level down. **Severity if it ships:** error — a decoder that disagrees with its own
specification is not a style question, and the corpus is the evidence either way.

### 4.25 §10.2 says "pick a tier, then hit these", and 21 of the 22 houses it was measured from cannot — **note; found by pointing 4.16's new tool at the corpus, 2.4; the port is corrected, the document is what wants changing**

The first thing `glidertool house stats` was asked, once it existed, was the obvious one: run it over
all 22 shipped houses at the tier §8.3 assigns each of them and see how the originals score against
their own table. The answer is that **only Leviathan is inside all eighteen bands of its own tier**,
and it manages that by having set eight of epic's bounds itself. The other 21 are outside at least
one. Six are outside four or more. Sampler misses eleven of the eighteen.

That is not a defect in the houses and not one in the measurement — the walk agrees with §3.6 on
eight columns for all 22 houses, and the empty-room count now agrees with §5.4's corpus figure of 840
to the room. It is a property of the table, and three separate mechanisms produce it.

**An edge rounded inward excludes the house it was taken from.** §10.2's cells are written to two or
three significant figures and the rounding goes toward the middle of the band, so the extreme house
that defined a bound lands just outside it. The clearest case is large's objects/room, **6.0-7.5**:
the minimum at that tier is Land of Illusion at 5.9934 and the maximum is Rainbow's End at 7.5022, so
the band is exactly those two houses rounded inward and **contains neither of them**. The same thing
at six other cells — Empty House's one prize in 35 rooms is 0.0286 against a tutorial floor of 0.03,
SpacePods is 14.53 objects a room against an epic ceiling of 14.5, Art Museum holds 569 objects
against a medium floor of 570, Metropolis is 1.016 enemies a room against a medium ceiling of 1.0,
Castle o' the Air is 1.412 prizes a room against 1.4, Fun House is 0.395 against a small floor of
0.4. Rounding a measured extreme is right; rounding it the wrong way turns a description into a
rule nothing satisfies. **A band quoted to fewer digits than it was measured at has to be widened,
not rounded.**

**The empty-rooms row is not a spread at all.** Eighteen of the 22 houses are outside it, which is no
longer a rounding story. Its five cells run 10-30 % and the houses run **0 % to 53.3 %**: six shipped
houses have no empty room whatsoever (California or Bust!, Davis Station, Nemo's Market, Sampler,
SpacePods, The Asylum Pro) and Demo House — the tutorial, the house a first author is likeliest to
imitate — is 53.3 % empty. The tutorial cell is written `~20 %`, and 20 % is §5.4's *corpus-wide*
figure, 840 empty rooms in 4,070, which is a mean over rooms and says nothing about houses of any
particular size. So one row of an eighteen-row table of observed spreads is a recommendation wearing
a measurement's clothes, and it is the row that fires on nearly every house. The observed per-tier
ranges, for whoever rewrites the cells: tutorial **0-53.3 %**, small **0-27.9 %**, medium
**0-22.8 %**, large **11.5-35.8 %**, epic **0-37.5 %**.

**§10.2's room bands disagree with §8.3's own grouping.** §8.3's five tiers are the only complete
assignment of the 22 houses to §10.2's five columns — by room count alone, California or Bust! at 16
rooms and Fun House at 43 belong to no column at all — and for medium, large and epic the two
sections state exactly the same range (85-140, 175-303, 383-531). For the two smallest they do not.
§8.3 describes its first tier as 2-45 rooms where §10.2's tutorial column says 35-45, and its second
as 16-65 where small says 45-85. So Sampler (2 rooms), California or Bust! (16) and Fun House (43)
are each outside the room band of the tier the document itself puts them in. That is 4.18's tier
again, from a different direction: the two smallest columns are where §10.2 stops being a
transcription of the corpus.

**What was wrong on our side, and is now right.** The command printed a corpus claim nobody had
measured: "no shipped house is inside all eighteen of its own tier's", a sentence that was both
truncated mid-clause and false. `Check`'s doc comment said the same thing, and so did one of the two
house tests. All three now say 21 of 22 and name Leviathan as the exception.
`profile.TestTheBandsExcludeTheHousesTheyWereMeasuredFrom` parses §8.3's grouping table, measures all
22 houses against their assigned column, and asserts the count, the identity of the one house inside,
the eighteen that miss the empty-rooms row, and which of the five room ranges agree between the two
sections — with the two that disagree written as a failure if they ever *start* agreeing, because
that is the fix and the entry should close when it lands.

Worth recording as method, because it is the same lesson twice. 4.16's first amendment found a row
that "could not be computed" the moment the counting left a test file. This is the second thing the
move bought: the 22 houses could not be compared against the table *at all* while the comparison
lived inside two tests about two of our own houses. A claim about a table of tiers is not checkable
one house at a time, and the sentence that turned out to be wrong had been written from three.
**The cost of a measurement living in a test is that it can only ever be asked about the test's own
subject.**

### 4.26 Three documents describe what `make check` does, and two of them were wrong — **found and DONE, 2.4; the verbatim copy is now machine-checked, the prose is not and cannot be**

Found immediately after 4.13's `docs-check` landed, by the only method that finds this: reading the
documents as a stranger rather than as their author. `make check` is described in three places, and
the three had drifted apart.

`CONTRIBUTING.md` quotes the target's prerequisites **verbatim**, which is the right thing to do —
somebody comparing a CI log against the document should be comparing the same words — and it had
already fallen a step behind, listing the twelve steps that preceded `docs-check`. `README.md:151`
and `docs/DEV_ENVIRONMENT.md:25` each carry a prose summary instead, and DEV_ENVIRONMENT's named six
of the fourteen steps. None of the three is redundant with the others; all three were stale in
different directions.

The timing was worse, because the two numbers were an order of magnitude apart and neither said what
it was measuring. `CONTRIBUTING.md` said `make check` "takes a couple of minutes". DEV_ENVIRONMENT
said "~15 s". Measured on the airgapped host (eight cores, 2026-09-22): **50 s with Go's build cache
cold, 12 s with it warm.** So "~15 s" was the warm number offered as *the* number — the one figure a
newcomer never sees, since their first run is by definition cold — and "a couple of minutes" was not
the cold number either. Both documents now give both, and say which is which.

The verbatim copy is the half that can be held to its source, and now is:
`TestTheStepListInContributingIsTheMakefilesOwn` in `tools/docscheck` reads the Makefile's `check:`
line and fails if `CONTRIBUTING.md` does not contain it, printing the line to paste. It reads the
Makefile rather than running it, so it costs nothing and runs everywhere — including
`windows-latest`, which has no `make`.

**What is deliberately not checked.** The two prose summaries are summaries: they say "pixel corpus"
for `fidelity` and "cross-compile" for `cross`, because those are what the steps *are*, and there is
nothing mechanical to compare a paraphrase against. Reducing them to the target names would make
them checkable and worse to read, for a document whose job is to be read before anything is built.
So the arrangement is one exact copy, held by a test, and prose that no longer pretends to be a list
— DEV_ENVIRONMENT's fence comment now says "everything CI does; `CONTRIBUTING.md` lists the steps"
and keeps no third copy of its own.

**And a decision worth recording rather than leaving implicit:** `docs/DEV_ENVIRONMENT.md` is *not*
in `docscheck`'s document list, and the obvious question is why, since it is the document that tells
a new session how to get a toolchain. Twenty command lines in its `bash` fences, and two of them
could run unattended here. The rest install packages as root through three different package
managers, clone over a network this host does not have, `podman pull` a Go image, unpack a tarball
into `~/.local/opt`, or want an X server. Adding it would buy two checked lines and eighteen written
excuses, and a rules table that is ninety per cent excuses is exactly what
`TestMostOfTheDocumentedLinesAreActuallyRun` exists to catch — a check can be made to look thorough
by widening its subject until it covers nothing. Revisit if that document ever grows commands a
machine can run; the list is one line.

---

### 4.27 A repair in the 1994 editor is unreachable, and 43 objects in three shipped houses are where it would have moved them — **found and DONE as far as this port can take it, 2.4; the bug is the original's and the accommodation is ours**

`KeepObjectLegal` is the original's per-object repair pass: it forces an object's rectangle inside the
room, snaps furniture to the tile grid, and — for four object types — rewrites a wrong vertical
coordinate back to the constant the type is supposed to sit at (`HouseLegal.c:115-137`). Three of the
four work. The fourth never runs.

```c
switch (theObject->what)
{
    case kFloorVent:
    case kCeilingVent:
    ...                                   /* sixteen cases, HouseLegal.c:75-90 */
    case kLiftArea:
    ...
    if ((theObject->what == kFloorVent) && ...)     /* :115  reachable */
    if ((theObject->what == kFloorBlower) && ...)   /* :120  reachable */
    if ((theObject->what == kSewerGrate) && ...)    /* :126  reachable */
    if ((theObject->what == kFloorTrans) && ...)    /* :132  kFloorTrans is not one of the sixteen */
```

`kFloorTrans` is not in that case list. It is handled thirteen arms later, in the block for the `data.d`
transports (`HouseLegal.c:282-385`), which repairs doors, windows and `kInvisTrans` and says nothing
about `v`. So the branch at `:132-137` is dead code: a floor transporter's vertical coordinate is never
repaired, by this or by anything else.

**The corpus shows the consequence.** 43 of the 244 shipped `kFloorTrans` objects sit at `v` 300 rather
than `kFloorTransTop`'s 302 — 37 in Slumberland, 5 in Leviathan, 1 in Rainbow's End. Every other one of
the 24 vertically fixed types is at its constant in all 4,041 placements (4.23). These 43 are the only
exceptions in the whole census, and they are exactly the type whose repair does not run.

Where the 300 came from is not knowable from here, and the honest answer is that it does not matter:
`AddNewObject` does write 302 (`ObjectAdd.c:341`), so these were moved afterwards by something —
an early build, a version of `DragObject` that took `deltaV`, or a house converted from Glider 4.0 —
and with the repair dead, nothing put them back. A two-pixel error nobody could see is how dead code
stays dead for thirty-one years.

**Had it been reachable it would have been wrong anyway**, which is worth recording because it is the
more interesting half. The branch writes through `theObject->data.a` — the blower variant — and
`kFloorTrans` is a `data.d` transport. `topLeft` aliases harmlessly, since all nine variants begin with
the same two shorts. `distance` does not: `data.a.distance` and `data.d.tall` are the same two bytes, so
the `data.a.distance += 2` at `:136` would have grown the transporter's *height* by 2 as a side effect of
moving it. That the three live repairs make the same adjustment to a real `distance` — the length of an
updraught's lift column — is at least coherent for them, since the object itself has just moved; for the
transporter it would have been two bytes of collateral damage from a line that was aiming at a different
variant. It fires once per out-of-place object, not on every save, because the repair that triggers it
also removes its own trigger.

**What this port does.** `object-top` accepts 300 for `kFloorTrans` and only for `kFloorTrans`, as a
value and not a tolerance — 299 warns, 301 warns. The exemption is a named constant,
`floorTransStrayTop`, with this section cited beside it, because a bare `|| got == 300` in a comparison
is indistinguishable from the bug it is accommodating. `TestCorpusFixedAnchors` pins the 43 by house, so
a loader change that moved them shows up here rather than as a silent shift in a count.

**Not fixed, and deliberately so.** The port could normalise all 43 to 302 on load and be rid of the
special case. It will not: `docs/analysis/house-format.md` and this repository's whole approach treat a
shipped house as the specification, and 302 in Slumberland's rooms is not what Slumberland contains. The
alternative — quietly rewriting 43 objects in 3 of the 22 houses to make one linter rule tidier — is the
kind of helpfulness `TestCorpusNonCompacted` exists to forbid.

---

## 5. Getting off this machine: the build, the package and the public path

Everything above is about the game. This section is about the fact that the game is being
written on one airgapped host and is meant to end up on GitHub, and that those two things have
different failure modes. Added when the public build path was put in beside the private one.

### 5.1 The public build path cannot be tested from the machine that wrote it — **note; needs one connected host, before the first public push**

`scripts/bootstrap-dev-env.sh --source public` and `.github/workflows/ci.yml` are the two
pieces of this repository that have **never run**. They cannot: this host cannot resolve
`go.dev`, `archive.ubuntu.com` or `github.com`, and a `curl` to any of them fails at connect.
What *is* verified is everything they are made of — every command the workflow invokes was run
here (`make doctor`, `make check` on a no-asset tree with `DISPLAY` unset, `make cross`,
`fmt-check` against a deliberately unformatted file), the script's own logic was reviewed with
`--dry-run` for all three sources, and the `.deb` path was exercised end to end against this
host's own Ubuntu mirror, which is the same code with a different base URL.

The specific things a connected host should check first, in the order they are likely to be
wrong:

1. The action versions. `actions/checkout@v4`, `actions/setup-go@v5`, `actions/cache@v4` and
   `actions/upload-artifact@v4` were the current majors as of the knowledge this was written
   with, and that is the weakest claim in the file.
2. `install_go_public`'s parse of `https://go.dev/dl/?mode=json&include=all`. The shape of that
   index is not documented as an API. The code picks the newest stable non-rc release at or
   above `go.mod`'s floor and then sha256-verifies the tarball against the digest in the same
   index — which is **integrity, not authenticity**: an index served by an attacker would agree
   with its own tarball. `GLIDERGO_GO_SHA256` exists so a digest can be pinned out of band, and
   `GLIDERGO_GO_TARBALL` skips the network entirely.
3. `xvfb-run -a -s '-screen 0 640x480x24' make bench`. The depth argument is not decoration —
   `internal/platform/x11` refuses anything but 24 or 32, Xvfb's historical default is 8, and
   the failure is a hard error rather than a skip. Verified here against `Xephyr` at both
   depths (686 fps at 24, refused at 8) because no `xvfb` is installed on this host.

None of this blocks Stage 1. It blocks the first push to a public remote, which is where it
will announce itself loudly and cheaply.

### 5.2 `tools/extract_all.py` writes its output tree in place, and something has already been corrupted by it — **planned, before the release pipeline; the CI half is worked around**

The extractor writes straight into `assets/extracted/`: no temp directory, no rename, no lock.
So for the ~70 seconds it runs, that tree is a mixture of the old extraction and the new one, and
anything reading it sees half-written files. This is not theoretical — a `go test ./...` running
concurrently with a `make assets` failed decoding a golden PNG with "unexpected EOF", which took
a while to recognise as a build-system problem rather than a renderer one.

Four consequences, and only the first is currently handled:

- **CI.** `.github/workflows/ci.yml` runs `make assets-check` as its own job, isolated from
  anything that reads the tree, and a comment there says why. That is a workaround in the caller,
  not a fix.
- **The tree is committed now (1.2), so a half-written extraction dirties the working copy** and
  the damage shows up as a `git diff` over binary files. That is at least visible, and
  `git checkout -- assets/extracted` undoes it, which is a better recovery than the old one
  (re-extract and hope). It also raises the stakes on the temp-and-rename fix below: a
  cancelled `make assets` now edits tracked files.
- **A cancelled extraction leaves a tree that looks complete.** `make check`'s guards test for
  contents (`[ -s art/manifest.json ]`, `ls houses/*.house`), which catches an *empty* tree but
  not a half-written one. The manifest is written last, which helps by accident rather than by
  design.
- **Two `make assets` at once** — a developer in one terminal, an editor's build task in
  another — interleave silently.
- **New, and this one is a mitigation rather than a consequence.** `make assets` now packs
  `assets/extracted.zip` from the tree as its last step (5.3), and `go test ./assets` compares the
  two file by file. A half-written or half-repacked tree is therefore a *test failure* naming the
  file whose length disagrees, which is the first tripwire this entry has ever had that fires on
  contents rather than on emptiness.

The fix is the ordinary one: extract into `assets/.extracted.tmp-$$`, `os.replace` the finished
tree into place, and take a lock file for the duration so the second run waits or refuses. It is
maybe thirty lines in `extract_all.py` and it wants doing before a release pipeline exists, since
a pipeline is precisely a place where two jobs share a checkout.

### 5.3 An installed copy looked for its assets beside the binary — **DONE, embedded; the assets are inside every executable**

The entry used to say every asset path resolved against the working directory (`assets/extracted/art`,
`assets/extracted/houses`, and the two others), which was right for a development tree and wrong
for anything a player downloads: a `glidergo` copied to `~/bin` found nothing and drew the
first-run screen instead. It recommended a search path — the executable's directory, then
`/usr/share/glidergo`, then `$XDG_DATA_HOME/glidergo` — and kept a 3 MB binary.

**The decision taken was the other one: `//go:embed`, no search path, nothing beside the binary.**
The reason is the audience. A search path is the right answer for something a distribution
packages, and gliderGo is a game somebody downloads from a Releases page, unzips into whatever
directory their browser chose, and double-clicks. Every step of a search path is a step that can
find the wrong copy or none, and "it must be run from the directory you unpacked" is an
instruction a player should never have to read.

**What it looks like.** `assets/extracted.zip` (11.3 MB, 1,877 files) is committed beside the tree
and compiled into every executable by `assets/assets.go`. Both binaries carry it — `glidertool`
renders houses and replays recordings, so a tool with no assets in an archive with no assets is
useless. Four things made it fall out cleanly rather than needing a plumbing rewrite:

- `*zip.Reader` is an `fs.FS`, so the loaders needed one change each: take an `fs.FS` from the
  caller instead of opening paths. Nothing under `internal/` knows the built-in copy exists, which
  is also why `go test ./internal/...` links none of these 11 MB.
- `internal/assetfs` holds the resolution rule in one place: an empty flag means the copy built
  into this binary; a named directory *replaces* that root. `-version` prints which of the two
  each root came from, so a bug report says it.
- `internal/assetpack` holds the packing and the comparison, and `tools/packassets` is the only
  thing that runs it. The packer must not import `assets` — that package embeds the archive it is
  about to write — which is the whole reason the logic is not simply in `assets`.
- The pack is deterministic: names sorted, every timestamp fixed at 1994-10-01 UTC, no mode bits,
  no directory entries. `make assets-zip` twice over produces the same bytes, so the committed
  archive is not a source of spurious diffs.

**The trap that decided the shape, and it is worth knowing.** `go:embed` accepts only names that
are valid module file paths, which rules out an apostrophe — and three of the shipped houses have
one: `Castle o' the Air`, `Nemo's Market`, `Rainbow's End`. A pattern that *names* such a file
fails the build, which is survivable. A pattern that names its **directory** skips it in silence:
120 files and 350,674 bytes, three houses and three houses' worth of custom art, gone from the
game with nothing anywhere to say so. A zip has no such rule, and the 1994 names survive
byte for byte inside it. `assets.TestApostropheNamesSurvived` is the tripwire, and it deliberately
does not skip when the tree is absent, because its subject is the bytes in the binary.

**What it costs, stated plainly.** A binary went from about 3.6 MB to 14.9 MB. A release archive
holds two of them, so the same 11.3 MB ships twice in it and the archives went from about 15 MB to
about 26 MB each — already-compressed data, so `tar -z` and `zip` cannot help. The repository now
carries the assets twice as well (the 16.3 MB tree plus the 11.3 MB archive of it), which took the
tracked total from 68.6 MB to 79.9 MB. Three ways to spend that back, none of them urgent, all of
them deferred on purpose:

- **Ship one binary.** Folding `glidertool` into `glidergo` as a subcommand, or leaving it out of
  release archives entirely, halves every download at a stroke. It is the cheapest of the three
  and the one that changes what a release contains, so it wants deciding with 5.4 rather than here.
- **Keep only the archive in git** and unpack it on demand. `make assets-check`, the fidelity
  corpus and the house round-trip name files by path, so this means teaching those three to read
  the archive or to unpack into a scratch tree. Saves 11.3 MB of repository and buys a `git diff`
  over `assets/extracted` that nobody can read.
- **Union the roots** instead of replacing them, so `-art some/dir` could override one PNG and
  inherit the rest from the built-in copy. Today it replaces, which is the honest simple rule and
  the one whose failure mode is obvious. The union is what a modding story would want; there is no
  modding story yet.

**What did not change.** The four flags still take a directory, and pointing one at nothing still
draws the first-run screen (`make headless` renders it via `-art /nonexistent`, which is now the
only way to reach it). `make assets` still re-derives the tree from `GliderPRO/` and now repacks
the archive after it. And the Makefile grew an `embedded` guard: every build target refuses to
build without `assets/extracted.zip` rather than producing an executable that comes up empty.

### 5.4 There is no release pipeline, and the CI that exists deliberately does not publish — **DONE as `release.yml`; no tag has been pushed yet**

`.github/workflows/release.yml` triggers on `v*` tags, and it does every item this entry used to
list as future work: `make cross` plus the host's cgo build, six archives of two binaries each
with their assets inside them (5.3), a `SHA256SUMS` manifest,
`README.md`, `CHANGELOG.md`, `LICENSE` and upstream's own README and licence in every one, and a
`gh release create` that runs only on a tag push. `VERSION` is passed explicitly rather than left
to `git describe`, so a build is stamped with the tag it came from instead of a bare short hash.

Three things about it are worth knowing before trusting it.

**It runs its own tests.** `ci.yml` triggers on a push to `main` and on `pull_request`, and a tag
push is neither, so tagging fires no CI at all. Without the `verify` job at the top of
`release.yml` a release would ship binaries that no suite had ever seen.

**It has never executed.** Same reason as `ci.yml`: the machine it was written on cannot reach
github.com. What is verified is everything below the GitHub line — the packaging step was
extracted from the YAML and run verbatim against a real `make cross` tree, twice, once before the
assets moved into the binaries and once after; all six archives build and pass `sha256sum -c`, and
the `linux-amd64` one was unpacked and played with sound — before 5.3 from its own root, and after
it from a directory holding the binary and nothing else, 300 on-screen frames of Slumberland out of
the copy inside itself. What is unverified is action versions, runner package names and
`gh`'s behaviour. The `workflow_dispatch` entry exists to rehearse the build half without
publishing.

**Five of the six archives cannot draw** (§5.5), which is a packaging problem as much as a
backend one. They are named `-headless`, and their `HOW-TO-RUN.txt` does not tell the reader to
run the bare binary: the null backend delivers no quit event and the shell loop has no exit
condition without `-frames`, so that would spin on an invisible title screen until Ctrl-C. It
gives two commands that terminate and produce something instead, and both were run from an
unpacked archive before being written down.

§5.3 used to be the reason an archive had to carry an asset tree and be run from its own root; it
is closed, so an archive is now two binaries and five documents, and `HOW-TO-RUN.txt` no longer
tells anybody where to stand. Two things a release still cannot fix: §1.2's reading of the licence
the 1994 content is shipped under, and the fact that the same 11.3 MB rides in both binaries of
every archive (§5.3's first deferred item).

**One thing to do at the moment the first tag is pushed, and not before.** `README.md`'s opening
paragraph says "there is nothing to download and no copy of the old game to find. Clone it and `make
run`", and the issue templates ask how the reporter's copy was built. Both become incomplete the
instant a release page has something on it, and both are the natural home for
`https://github.com/bwenstar/gliderGo/releases`. That URL used to be a constant,
`project.Releases`, deleted in 2.4 because it had no reader and a page with nothing on it is not
something to point a player at (4.13). Putting it back is one line in `internal/project/project.go`,
and `TestEveryExportedConstantHereIsReadBySomething` will hold it to having a caller this time.

**And one thing to do before the first tag is pushed.** The release notes now walk a player through
SmartScreen, Mark of the Web and Gatekeeper quarantine, which 4.13 called the single most likely
reason a first-time player never reaches the title screen — and they do it from documentation rather
than from anyone having watched it happen. The Windows half needs no network to check: write a
`Zone.Identifier` stream onto the `.exe` by hand and double-click it in Explorer.
`docs/windows-first-run.md`, "Rehearsing what a download adds, with no download", has the two
commands. It is the only unverified claim in `release.yml` that github.com is not required to settle,
which makes it the cheapest one on the list and the last one that has an excuse.

### 5.5 Nothing in here has ever been compiled by a macOS or Windows toolchain — **note; windows/amd64 is now run as well as compiled, macOS and windows/arm64 are still compile-only**

`make cross` builds `windows/amd64`, `windows/arm64`, `darwin/amd64`, `darwin/arm64`,
`linux/arm64` and `linux/amd64` in about four seconds, and it is worth being precise about what
that proves. There are now three backend selectors rather than two, and they are still exact
complements (`internal/platform/backend/doc.go` reads them side by side):
`backend_x11.go` is `linux && cgo && !nullbackend`, `backend_win32.go` is
`windows && !nullbackend`, and `backend_null.go` is the negation of both. So the darwin rows and
cross-compiled `linux/arm64` resolve to the null backend, and a green build for those is evidence
that the game logic, the house codec, the asset pipeline, the shell, the scores and the replay
harness are portable — and *no* evidence that they can draw a pixel. It also cannot see a
path-separator or filesystem-case bug, which is the class of thing that only appears on the real
OS.

The windows rows are a different case since the win32 backend landed: they carry a real backend
even at `CGO_ENABLED=0`, so a green build there means the Windows window code compiles for that
architecture. That is a stronger claim than "the portable part compiles" and a much weaker one
than "it works". **Nothing on this machine can execute it**, and the backend was written here, on
an airgapped Linux host, against nothing but the documentation.

It has since been executed elsewhere, on a real Windows Server 2025 desktop, and `windows/amd64`
is no longer a compile-only claim: the window opened, the blit put pixels on a screen that match a
Linux-rendered frame exactly, and the `waveOut` sink played every sound it was given.
`docs/windows-first-run.md` is the write-up and states the three gaps it left — no key was ever
pressed, nothing touched the window, and `windows/arm64` still has never run anywhere. For that
row, and for both darwin rows, every word above still holds.

CI remains the *automated* half of the verification story, so it is worth knowing exactly what
it does: `go build`, `go vet` and `go test ./...` on `windows-latest` and `macos-latest`; then, on
Windows only, `glidergo -version` and `glidergo -frames 300 -bench`, which opens a real window and
pushes 300 frames through `StretchDIBits`. The bench is `continue-on-error` on purpose — a hosted
runner is not a desktop session, so whether `CreateWindowExW` behaves there as it does in front of
a logged-in user cannot be settled from here, and the log is the interesting output either way.
The `python3 tools/extract_all.py` step is `continue-on-error` for the older reason: the extractor
is stdlib-only python3 that has only ever run on Linux, and a failure is a finding for this file
rather than a reason to hide the step.

macOS is still Stage 6, and there "it compiles" remains the whole claim.

### 5.6 Is a fresh checkout playable? — **audited and yes; four defects found and fixed, 1.10a**

This item exists because the question had never been asked from the outside. Everything was
verified on the machine that wrote it, where Go was already on `PATH`, `assets/extracted/`
already existed, `DISPLAY` was already set, and `scripts/env.sh` had already been sourced — the
one configuration in which a stranger never finds themselves. So the tree was cloned to a
scratch directory and driven from the README, with nothing pre-warmed.

What the audit established, as facts rather than intentions:

- **A clone contains everything.** `git clone` → `apt-get install build-essential pkg-config
  libx11-dev` → `make run` is the whole sequence, since the decoded assets are committed (1.2).
  Nothing is fetched, because there is nothing to fetch: `internal/module` asserts stdlib-only,
  so there is no `go.sum`, no `vendor/` and no proxy to be unreachable.
- **The original game is not needed.** It never was, because `GliderPRO/` is vendored: `make
  assets` reads exactly **38 committed files** from it — `Glider PRO.r` (15,475,666 bytes, the
  derez'ed resource fork carrying every PICT, `'snd '`, `STR#` and dialog), 22 `.binhex` houses
  and 15 `.mov` movies — and writes 1,899 files in about 70 s. Since 1.2 was settled that output
  is committed too — all but the 22 `.rsrc` intermediates, which are 24 MB of the 39.5: 1,877
  files, 15.5 MB — so playing needs neither the extractor nor python3. `make assets-check`
  re-extracts and compares every file it commits, which is what keeps the committed copy honest.
- **`make check` passes twice over: on a full clone, and on one with the assets removed and no
  `DISPLAY`.** The asset-dependent steps skip by name and `check-caveats` closes by listing what
  it could not verify, so a green run on an empty tree never reads as a green run on a full one.
- **Size.** The port itself is about 11 MB of source, docs and tests. Everything else is 1994
  content: `GliderPRO/` 50.7 MB encoded, `assets/extracted/` 15.5 MB decoded from it. 1.2 records
  why both are here.

Four things the audit found, all fixed in the same commit as this entry:

1. `make smoke`, and therefore `make check`, **failed** on a clone that had a display but no
   extracted houses — it invoked the on-screen bench, which needs a house to fly in. That is the
   first command the README hands a stranger, broken by the one condition nobody developing here
   is ever in. It now skips with a reason and `check-caveats` reports the gap.
2. `-house Titanic` only worked from a directory that happened to contain the file; a bare name
   with an extension was never resolved against `-houses`. Both forms work now.
3. `make cross`'s host cgo build could fail without failing the target: the exit status was
   assigned to a variable that a later `printf` overwrote.
4. `make headless` listed a stale output directory, so a shot from a previous run could be read
   as work just done. It removes the directory first.

Left open, and deliberately: the README's claims about Windows and macOS remain untested by
anyone (5.5), the public dependency path remains untested from here (5.1), and where an
*installed* copy looks for its assets is still repo-relative (5.3). The audit's subject was a
developer's clone, which is the only thing gliderGo currently ships.

**Follow-on, same stage.** The audit's answer to "will the original source be needed for the
assets?" was *no, because it is vendored* — which is true and was not the point. The answer taken
instead was to remove the question: commit the decoded assets, so the first command after
`git clone` is `make run`. That is 1.2's route (a) and it retires the extraction step from the
quick start entirely.

### 5.7 The four files a public repository is expected to have, and the two templates — **five of six DONE, 2.0; `CODE_OF_CONDUCT.md` still deliberately absent**

1.3 records that `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md` and the issue and
pull-request templates do not exist. This is where the reasoning lives, because "add the standard
files" is a worse plan than it sounds: each of them is a promise to somebody, and writing a
promise before there is anyone to make it to produces the boilerplate that every reader learns to
skip.

What each one would have to say that is specific to gliderGo, and is therefore not yet writable:

- **`CONTRIBUTING.md`** — the useful content is already spread across the tree and would mostly
  be links: `docs/DEV_ENVIRONMENT.md` for the two ways in, `make check` as the gate,
  `internal/module`'s stdlib-only invariant as the one rule a patch can break without noticing,
  and the fidelity corpus as the reason a rendering change needs
  `go test ./internal/fidelity -update` and a hash in the diff. The part that cannot be written
  yet is the review process, because there is no second reviewer.
- **`CODE_OF_CONDUCT.md`** — Contributor Covenant 2.1, whose only project-specific field is the
  reporting address. There is no address to put in it, and a code of conduct with an unattended
  contact is worse than none.
- **`SECURITY.md`** — the honest text is narrow and worth writing precisely: gliderGo parses
  untrusted binary house files and BinHex, and `internal/house` is the attack surface a malicious
  `.house` reaches. It has no network listener in Stage 1; Stage 3's race protocol adds one and
  that is when this file stops being cheap. Again blocked on a reporting address.
- **Issue and pull-request templates** — the one genuinely valuable field is a `glidertool replay`
  trace, since 4.2 built the format precisely so a bug report can carry a reproducible input
  rather than a description. That template can be written the day the repository is public.

The decision recorded here is that all six wait for the public push, and that when they are
written they say the gliderGo-specific thing above rather than the generic thing.

**Written at 2.0, five of the six.** `CONTRIBUTING.md` (172 lines), `SECURITY.md` (51),
`.github/PULL_REQUEST_TEMPLATE.md`, and three issue forms —
`.github/ISSUE_TEMPLATE/bug_report.yml`, `fidelity_difference.yml` and `config.yml`. Each says the
specific thing this section demanded: `CONTRIBUTING.md` leads on the stdlib-only invariant and the
`-update` hash in the diff; `SECURITY.md`'s scope section is three paragraphs about house-file
parsing and one sentence about what does not exist; and `fidelity_difference.yml` exists as a
*separate* form from the bug report, because "the original did something else" and "this crashed"
want different fields and the former is the report this project most wants to receive.

**What unblocked them was noticing that the reporting address was the wrong blocker.** Both
`SECURITY.md` and `CODE_OF_CONDUCT.md` were deferred above on the grounds that there is no address
to put in them and an unattended contact is worse than none. That is true of an email address and
not true of the mechanism GitHub already provides: private vulnerability reporting routes to the
maintainer, is auditable, needs no inbox, and cannot be scraped. So `SECURITY.md` points at
**Security → Report a vulnerability** and states "one maintainer, no SLA" in place of a promise it
cannot keep — which is the honest version of the same file.

**`CODE_OF_CONDUCT.md` is the one still not written, and now for a better reason than the address.**
The same substitution does not work: GitHub's private reporting is security-only, so a covenant
would have to route conduct reports either to an unattended address or to GitHub's own abuse
process — and pointing at GitHub's process is telling people to go over the maintainer's head by
default, which is not what the document is for. Left absent rather than boilerplated. The trigger
to write it is a second maintainer, not the public push; that is who the reporting clause would
name.

### 5.8 What the fresh-checkout audit found and deliberately did not fix — **notes, 1.10a**

Recorded so that none of these is rediscovered as a surprise. Each one is small; each was left
alone for a stated reason rather than missed.

- ~~**`go.mod` says `module glidergo`, and a public repository's module path should be its
  URL.**~~ **Closed, 1.10c.** It was the one item here that could not be guessed — it needed the
  account the repository would actually live under — and that account is now known:
  `github.com/bwenstar/gliderGo`, set in `go.mod` and in 248 import lines across 113 files. The
  test that pins it keeps a second copy of the string on purpose (`internal/module`'s
  `modulePath`), because a test that read the path out of the file it is checking would accept a
  typo. Two new cases in that test's table earn their place only now that the path has a dot in
  its first element: `github.com/bwenstar/glidergo` and `github.com/bwenstar/other` must both be
  refused, which is what proves the prefix match is *this* module rather than anything on
  `github.com` or anything of this owner's.
- **`LICENSE` omits GPLv2's "How to Apply These Terms to Your New Programs" appendix.** That
  appendix is instructions to the *next* author, not part of the licence's operative text, and its
  absence changes nothing legally; the terms are complete (sections 0–12 plus the preamble). Left
  out because it invites a reader to think the boilerplate notice it contains has been applied to
  every source file, which is not true — see the next item.
- **No per-file copyright headers.** 394 tracked files carry none; the licence is stated once, in
  `LICENSE` and in the README's licence section. This is a defensible choice and a common one, but
  it is a choice: a file copied out of this tree carries no notice with it. Revisit if there is
  ever a second contributor, at which point 5.7's `CONTRIBUTING.md` is the place to state it.
- **`scripts/bootstrap-dev-env.sh` defaults its Ubuntu suite to `noble`.** Already overridable
  with `GLIDERGO_UBUNTU_SUITE`, so this is a default rather than a hard-coding, and the default
  matches the only host it has ever run on (Ubuntu 24.04.4). It will go stale; the fix is a
  release-file probe, which is more machinery than the problem deserves until somebody hits it.
- **Three Go files cite `docs/analysis/stage15-raw/`, which is gitignored** —
  `internal/game/transit.go:25`, `internal/game/guards.go:6`, `internal/replay/replay.go:7`. The
  citations point at working drafts that were deliberately not committed (`.gitignore` says why:
  they still contain the errors the review pass corrected, and committing both would leave two
  disagreeing sources of truth). A reader of a fresh clone therefore follows those three
  references to nothing. The "releasePolish N" numbering they quote exists only in those drafts,
  so the citations cannot simply be repointed at the committed spec — closing this properly means
  giving the surviving decisions numbers in a committed file. Deferred; noted here so the dead
  reference is a known one.
- ~~**The asset flags still default to paths relative to the working directory** (`-art
  assets/extracted/art`, and three more).~~ **Closed by 5.3.** All four now default to empty,
  meaning the copy inside the executable, and a directory only comes into it when somebody names
  one. `-version` still reports where each root came from, which is what makes the override legible
  from a bug report.
- **Audio still shells out to an external player on Linux and has no sink at all elsewhere.**
  Already 2.48. The audit's only addition is that `-version` prints the backend, so a report from
  a null-backend build is identifiable as one.

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
| 2.18 (premise) The demo is a determinism oracle, and the shipped Carbon build never seeded at all | 1.8b | this stage |
| 2.18 (as far as it can be without a Mac) The RNG's stream, `RandomInt`'s bound and skew, and the naive-int32 divergence, all pinned as table tests | 1.8c | this stage |
| 2.18 (scope) One random stream per *process*, as `qd.randSeed` is, handed back after every game | 1.8c | this stage |
| 2.18 (the launch draw) `-seed` defaults to 1, `0` means the clock, and `AdvanceRandSeed` accounts for `VariableInit`'s draw so the first game starts on 16807 | 1.8c | this stage |
| Contract item 6 `game.TestPlayGameOrder`: the frame loop's call order *and* each call's guard set, read out of the AST | 1.8c | this stage |
| Contract item 15 `render.TestTheMaskingStrategyOfEveryObjectType`: the 21/9/2/3 painter census, which no pixel test can see | 1.8c | this stage |
| Contract item 4 `internal/render/view_test.go`: the audit's one genuinely unheld row — `playOriginV` comes from the *screen*, not the house rect, and nothing had ever said so | 1.8c | this stage |
| The fidelity contract audited row by row — `ORIGINAL_GAME.md` §19.1, twenty citations and five written exceptions | 1.8c | this stage |
| 2.23 Player 2 gets the abandon key, as the fourth opt-in fix `fixes.player2_give_up` | 1.9 | this stage |
| 2.22 (characterised) The mismatched-exit deadlock: Delete *does* break it, transits deadlock on two objects of the same kind, and 60 refused frames prove it never resolves | 1.9 | this stage |
| 2.16 (measured) The arrival freeze hits whoever is not `FirstPlayer` — which before anybody waits is *player 1*, not player 2 | 1.9 | this stage |
| The three race strictnesses pinned in `game/twoplayer_test.go`: geography refuses audibly, transits refuse silently and demand the same object, the manhole does not race | 1.9 | this stage |
| The shared inventory and the shared throttle pinned as one counter each — one thruster in two-player sounds every frame, not every fourth | 1.9 | this stage |
| `gameFixes` extracted and `TestEveryOptInFixIsCopiedToTheGame` written, because the field-by-field copy's compile-error claim was false | 1.9 | this stage |
| 4.3 (the second script) `testdata/two.script` — a whole two-player game replayed to a golden trace, plus a companion test that reads the same run as sentences | 1.9 | this stage |
| The trace grew a nine-column two-player tail that a one-player trace does not carry, so every golden on file stayed byte-identical | 1.9 | this stage |
| `scripts/bootstrap-dev-env.sh --source system\|internal\|public\|auto`, `make doctor`, `fmt-check` inside `check`, `make cross` over all six targets, and `internal/module`'s stdlib-only assertion | public build path | earlier |
| `internal/saved`: one save per house under `internal/datadir`, written atomically, and the original's five validation gates — the code it wrote out in full and never ran | 1.10 | this stage |
| `internal/house`'s saved-game codec: `game2Type` from offset 6 on, decisions S1–S4 stated where the code is, and the 40-byte block re-encoded byte-for-byte out of all 22 shipped houses | 1.10 | this stage |
| S4 — a house's embedded block resumed with the *house's* timeStamp, without which the original's own gate refuses every game its own writer produced, and Titanic's 1995 save can never be opened | 1.10 | this stage |
| `S` while paused saves, alert 1041's two buttons on `Y`/`N`, and the pause key as a third answer the alert did not have | 1.10 | this stage |
| "Open Saved Game…" on the title screen — MENU 129's third item, on the `O` it had — greyed with the reason, plus `-resume`, `-saves`, and both flag contradictions refused before a window opens | 1.10 | this stage |
| 2.66 `menuRect(n)` derives the menu panel's bottom edge, so the rows cannot fall outside the box they sit in | 1.10 | this stage |
| 2.67 `World.SetPauseHint` erases the old hint row before changing it, so a shrinking hint leaves no tail | 1.10 | this stage |
| 2.70 `Shell.status()` orders the status band's three sources, so the resume message cannot be silently overwritten | 1.10 | this stage |
| 5.6 The fresh-checkout audit: a clone is playable with no network and no copy of Glider PRO, proven by driving the README from a scratch directory | 1.10a | this stage |
| `make smoke` (and so `make check`) no longer fails on a clone that has a display but no extracted houses — the first command the README gives a stranger | 1.10a | this stage |
| `-house Titanic` resolves against `-houses`, so a bare name works from anywhere and not only from the directory holding the file | 1.10a | this stage |
| `make cross`'s host cgo build can fail the target again: its exit status was being overwritten by a later `printf` | 1.10a | this stage |
| `make headless` clears its output directory first, so a shot from a previous run cannot be read as work just done | 1.10a | this stage |
| Every reference to the private repository this was developed inside is gone from the tree, and 46 absolute home paths across 21 documents are now repo-relative | 1.10a | this stage |
| `scripts/bootstrap-dev-env.sh`'s private mirror is opt-in rather than built in, and `deb_arch()` translates `uname -m` instead of answering `amd64` | 1.10a | this stage |
| 1.3 `CHANGELOG.md`: one section per stage, each naming the commit that closed it, under `Unreleased` because there are no tags | 1.10a | this stage |
| The About box and the no-assets title screen say where the art actually comes from — it ships with the source, undecoded — instead of "your own copy" | 1.10a | this stage |
| `.gitattributes`: `* -text`, so a Windows checkout cannot rewrite `Glider PRO.r`'s 199,843 LFs and silently change the extractor's input hashes | 1.10a | this stage |
| `glidergo -version` prints the build, the compiled-in backend, the Go that built it, the platform, and whether each asset tree is there | 1.10a | this stage |
| 5.7 and 5.8 written: the conventional repository files as a dated decision, and seven audit findings deliberately left alone with the reason each | 1.10a | this stage |
| 1.2 (route (a), the decision) `assets/extracted/` committed — 1,877 files, 15.5 MB — so a clone plays with no extraction step, no python3 and no copy of Glider PRO | 1.10b | this stage |
| The 24 MB of `houses/*.rsrc` left out, because they are the one thing in the extracted tree that nothing reads at run time: an intermediate `extract_house_art.py` turns into `houseart/` | 1.10b | this stage |
| `make assets-check` upgraded from a manifest comparison to a full recursive diff, after proving the extractor byte-for-byte reproducible over all 1,899 files | 1.10b | this stage |
| `.gitattributes` marks the tree `linguist-generated` and `binary`, with the three manifests exempted, so a diff of authored code is not drowned in 1,877 generated files | 1.10b | this stage |
| CI refuses a checkout whose assets are missing instead of extracting them, and a separate `assets` job holds the committed tree to `make assets-check` | 1.10b | this stage |
| The About box, the credits, the README and the no-assets screen all say the data ships here — the fourth rewrite of that sentence, and the first one that is true of a clone | 1.10b | this stage |
| `internal/credits`' three transcription tests skip rather than fail when `GliderPRO/` is absent, so a checkout with the 1994 source deleted still reaches a green `make check` — proven by doing it | 1.10b | this stage |
| 2.71 The external audio player's stderr is prefixed with the player's name, so an unreachable sound server no longer prints an unattributed `error:` between two of the game's own lines | 1.10b | this stage |
| 5.8 (the module-path item) `module github.com/bwenstar/gliderGo`, with two new cases in `internal/module`'s table that are only discriminating once the path has a dot in its first element | 1.10c | this stage |
| The private mirror is out of the tree entirely: `--source local` reads an optional gitignored hook, and no doc, script or comment names anyone's internal network | 1.10c | this stage |
| The author email in all 42 commits' metadata is a personal address — a leak that no content grep could have found, since it is in the objects rather than the files | 1.10c | this stage |
| Four commit *messages* named the mirror by product name, found by grepping `git log` rather than the tree, which is the same class of leak one level out | 1.10c | this stage |
| The Makefile reads `GLIDERGO_TOOLCHAIN_DIR` rather than hard-coding `~/.local/opt`, so it cannot disagree with the bootstrap about which Go was just installed | 1.10c | this stage |
| 4.1 `internal/house/lint.go`, `glidertool house lint` and `glidertool house checks`: 29 checks, severities argued down against the 22 shipped houses until they produce 634/48/**1** | 2.0 | this stage |
| 2.14 (the linter half) and 2.49 (the reporting half) — `sound-id` split from `sound-unreadable`, so the 7 findings that are gliderGo's MACE gap do not read as 7 authoring mistakes | 2.0 | this stage |
| `internal/house/lintcatalogue_test.go` reads `lint.go`'s AST, so a check cannot ship without a row in the table a report sends the reader to | 2.0 | this stage |
| `internal/game/linkagreement_test.go`: the two "which objects carry a link" implementations pinned to each other over all 144 object codes, so the `Objects.c` transcription and the linter's map cannot drift | 2.0 | this stage |
| `internal/render`'s duplicate `ExtractFloorSuite`/`GetRoomNumber` deleted in favour of `internal/house`'s, and the dead `kNumUndergroundFloors` beside them | 2.0 | this stage |
| The pre-2.0 link packing cannot express suite ≥ 100 — it *collides* — written into `house-format.md` §7.2 and asserted rather than avoided | 2.0 | this stage |
| Two documents claimed every shipped house has a `kStar`; Fun House has none, which is why `no-stars` is a warning | 2.0 | this stage |
| 4.14 `internal/shell/sets.go`: a level set is declared by the source a house was walked from, so the claim is "where this was found" and never "what is in it" | 2.2 | this stage |
| `Library.Discover` is variadic and accumulates, which `internal/assetfs` deliberately still does not — houses add, art replaces, and 5.3's deferred union lands in exactly one of the two | 2.2 | this stage |
| The picker's set strip, on the title's own line because the apparently-empty row below it is the first house's selection bar; one set draws the identical screen it drew before, so `internal/fidelity`'s reference stays a picture of 1994 | 2.2 | this stage |
| 4.14 (the built-in half) The `New` set ships *inside* the executable — `assets/levels.zip` embedded, `go test ./assets` rebuilding every authored house from its text and comparing bytes | 2.3 | this stage |
| 4.16 (the Go half) `internal/replay/profile_test.go` walks a house and holds it to the tier bands in `docs/analysis/original-houses.md` §10.2, instead of the bands being measured by hand | 2.3 | this stage |
| 4.19 A house's `timestamp` and file *name* are the keys its saved games and high scores hang on, pinned by a test carrying the published numbers | 2.3 | this stage |
| 4.20 `glidertool replay`'s `house` line takes a name as well as a path, because a house that lives in the binary has no path to give | 2.3 | this stage |
| `levels/Open House.house.txt`, the first house this port wrote, built from text by `make levels` and lint-clean | 2.3 | this stage |
| 4.16 (the dark-room row) The row declared uncomputable was three lines in `internal/replay`, which already imports both packages the argument said could not be joined | 2.4 | this stage |
| `levels/Boarding House.house.txt`: 51 rooms at the small tier, all eighteen built-in backgrounds, and a per-background histogram pinned because moving one room out of `kRoof` falsified four figures in its own header while every profile row stayed green | 2.4 | this stage |
| 4.21 `lint.go`'s calibration paragraph rewritten around a narrower rule — a class the corpus exercises deliberately cannot be an error; a class it exercises by overrunning a buffer can — with the staircase example dropped because it fires zero times | 2.4 | this stage |
| 4.22 `mount-no-floor`, `mount-no-ceiling` and `starfield-tiles`: the first three checks that relate an object to the background it stands in, classified by a census rather than by the objects' names | 2.4 | this stage |

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
