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

---

## 1. Legal — the blocker that has nothing to do with code

### 1.1 gliderGo has no licence file — **DONE, this stage**

gliderGo is transcribed from GPLv2 source, function by function, with comments citing
`GliderPRO/Sources/*.c` line numbers. That makes it unambiguously a derivative work, and
GPLv2 §2(b) requires the whole to be licensed under GPLv2. This is not a preference; it
is the only compliant option, and shipping a public binary with no licence file at all is
strictly worse than shipping one. Added `LICENSE` (GPLv2) and a `## Licence` section in
`README.md` naming John Calhoun, Casady & Greene and the upstream repository.

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

## 2. It has to run on the machine someone actually owns

### 2.1 Window scaling — **partly done; fullscreen and a runtime toggle planned, 1.7**

Already there: `cmd/glidergo -scale N` does integer nearest-neighbour magnification
(`main.go:29`), which is the right filter — the art is 8-bit indexed pixel art and
bilinear would smear it. What is missing for a release is fullscreen with letterboxing, a
runtime toggle rather than a launch flag, and remembering the choice.

**A fidelity trap to avoid when that work happens, which the port has so far avoided by
luck rather than intent.** `Main.c:191` forces `numNeighbors` to 1 on a 512px-wide screen
and `Settings.c:888` defaults it to 9. 1.5a measured what that costs: at 1/3/9 neighbours
the CD Demo House resolves 46/141/390 cross-room object links, so **a switch wired to the
room next door stops working on a small screen**. In the port `NumNeighbors` lives on
`render.Scene` and defaults to 9 with nothing reading the window size, so the two are
already decoupled — the risk is a future "derive it from the window like the original did"
change reintroducing the coupling. It should become an explicit setting, documented as
affecting gameplay and not just how much you can see.

### 2.2 Frame pacing — **planned, 1.5b (this stage), with a note for 1.7**

The simulation is frame-locked at `kTicksPerFrame = 2` Mac ticks, i.e. 30.07 fps, and
every physics constant was tuned against it. That is not negotiable and the port keeps it.
But the original's limiter is a busy-wait on `TickCount()`, which on a modern machine
means one core pinned at 100% and a hot laptop. 1.5b must implement the limiter as a real
sleep to the deadline, which is behaviour-identical and not a fidelity change. Frame
*interpolation* for 60/120/144 Hz displays is a separate question and belongs to 1.7 or
later; it must not touch the simulation.

### 2.3 Player 2's keys are modifier keys — **planned, 1.7**

The original hard-codes player 2 to modifier keys, which a modern window manager or
desktop environment may swallow before the game sees them. 1.4 already anticipated this:
`internal/game/player/glider.go:95-98` keeps the four key indices as per-glider *data*
rather than constants, specifically so 1.7 can remap them. 1.7 owes a remapping UI and a
persisted config. Related: a release needs gamepad support, which the original had no
concept of.

### 2.4 A missing or wrong asset directory must not be a crash — **planned, 1.7**

Today the tests `t.Skipf` when `assets/extracted` is absent, which is right for tests and
is not a shipped behaviour. A public binary run by someone who has not pointed it at a
copy of Glider PRO must say so, in a window, with instructions — not panic on a nil
surface or exit silently. This is the user-facing half of item 1.2 and the two should be
built together.

### 2.5 No way to pause or quit that a stranger would find — **planned, 1.7**

`Env.DoPause()` blocks until unpaused, faithfully. A shipped game needs a pause overlay
that says what to press, and a quit path that is not "close the window" or `kill`.

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

## Done

| Item | Stage | Commit |
|---|---|---|
| 1.1 GPLv2 `LICENSE` and a `README.md` licence section | 1.5b | (this stage) |
