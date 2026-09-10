# gliderGo — implementation plan

**What this is:** a Go port and remaster of *Glider PRO* (1994, John Calhoun,
Casady & Greene; source released under GPLv2 -- the tree says 1.0.4, the resource fork says 1.1.2). The original C source is vendored
read-only in `GliderPRO/`.

**Read these first:**
- `docs/ORIGINAL_GAME.md` — what the original actually does (the consolidated source of truth)
- `docs/analysis/*.md` — byte-level specs per subsystem (the detailed authority)
- `docs/DEV_ENVIRONMENT.md` — how to build here, and the constraints that shaped the design

---

## 1. Goals, in the owner's words and in order

1. **Stage 1** — recreate the game as `gliderGo` in Go: "basically a ported or remastered
   version… behave exactly the same as the original however just newer." Linux first,
   then Windows and macOS, possibly iOS/Android later.
2. **Stage 2** — add new levels ("houses") in the spirit of the originals, with a
   menu choice between *original* and *new* level sets.
3. **Stage 3** — 2-player over the network: one machine hosts, another joins;
   **furthest on one life wins**; the host picks which houses are in play.
4. **Throughout** — every stage committed to git with a detailed message, and a
   development environment good enough that a paused project can be picked up by
   another session cold.

### Non-goals (explicitly out of scope unless asked for later)

- Recreating the Mac Toolbox shell for its own sake (Apple Events, Scrap, DA support).
- Gameplay "improvements". Stage 1 fidelity means *not* fixing the original's quirks.
- Redrawn or upscaled art (decided against — see §2).

---

## 2. Decisions taken (confirmed with the owner 2026-09-10)

| Decision | Choice | Consequence for the build |
|---|---|---|
| **Multiplayer model** | **Race, separate worlds.** Each machine simulates only its own glider; the network carries progress (room, score, lives, alive/dead). | No lockstep, no rollback, no desync class of bug. ~20 bytes/s. Ships as its own stage without destabilising the engine. |
| **Assets** | **Extract the 1994 originals** from `GliderPRO/Glider PRO.r` into PNG + PCM. | Needs a from-scratch QuickDraw PICT decoder and `'snd '` decoder in Python (nothing is downloadable). Buys pixel-exact fidelity and makes frame-diff testing meaningful. |
| **Level data** | **Read the original binary houses byte-exactly; author new houses in a text format** that compiles to the same in-memory model. | Two loaders, one runtime `House` type. The 22 shipped houses stay the source of truth for themselves; new levels are diffable in git. |
| **Stage order** | Faithful port → new levels → multiplayer → editor/platforms. | Matches the owner's stated order and puts the riskiest fidelity work first, while the engine is small enough to change. |
| **Engine** | None. Standard library + our own `internal/platform` port layer. | Forced by the network (no Go module proxy, no reachable GitHub — `docs/DEV_ENVIRONMENT.md` §3), and a good fit: the original is a fixed-resolution software blitter. |
| **Language** | Go 1.23.12, obtained rootlessly from a container image. | The owner's first choice was available after all, so no fallback language is needed. |

---

## 3. Architecture

```
cmd/
  glidergo/            the game
  glideredit/          (stage 5) house editor
  glidertool/          asset + house inspection CLI, useful for debugging fidelity
internal/
  platform/            port layer: Framebuffer, Window, Key, AudioSink  [DONE]
    x11/               Linux backend, cgo + Xlib                        [DONE]
    null/              headless: PNG frames + WAV audio, for tests       [DONE]
    win32/             Windows backend, pure Go syscall to gdi32        [stage 4]
    sdl2/              macOS/iOS/Android backend, hand-written cgo      [stage 6]
  house/               House/Room/Object model; binary loader + text loader/writer
  assets/              extracted sprite sheets, palettes, sounds; sprite atlas indexing
  game/                the simulation: player, objects, collision, room transitions
    player/            glider state machine and physics
    objects/           per-class object behaviour
    room/              room state, links, persistence of taken bonuses etc.
  render/              the original's draw order over a Framebuffer
  audio/               mixer over AudioSink; sound event table
  ui/                  menus, dialogs, scoreboard, house selection, high scores
  net/                 (stage 3) host/join, progress protocol
  fidelity/            frame-diff and trace-diff harness against reference data
tools/                 python3 extractors (PICT, snd, BinHex, house dump)
levels/                new houses in text form (stage 2)
assets/extracted/      generated, gitignored — reproducible from GliderPRO/ via tools/
```

Three rules that keep the port honest:

1. **The simulation never touches the platform.** `internal/game` takes an input snapshot
   and produces state; `internal/render` turns state into pixels. This is what makes the
   headless fidelity harness and the future lockstep option possible.
2. **Original constants live in one place per subsystem, named as in the C.** A reader with
   `GliderPRO/Sources/Dynamics.c` open should be able to follow the Go line by line. Ported
   functions keep the original's name in a doc comment with its `file:line`.
3. **No "while I'm here" cleanups in stage 1.** Quirks are the product. Anything that looks
   like a bug in the original gets a comment and a test pinning the original behaviour, not a fix.

---

## 4. Stages

Each stage ends in a commit (usually several) with a message explaining what was
learned, and leaves `make check` green.

### Stage 0 — foundations ✅ *done*

- Git repo initialised; original source vendored read-only (`b988c2f`).
- Rootless dev environment: Go from a container image, `scripts/bootstrap-dev-env.sh`,
  `make check` (`b472ca4`).
- Platform layer with x11 + null backends, verified on screen at 944 fps.
- Source archaeology: `docs/analysis/*.md` + `docs/ORIGINAL_GAME.md`.

### Stage 1 — the faithful single-player port

The order matters: data before pixels, pixels before physics, physics before polish.
Each step is independently verifiable, which is the only way to catch a fidelity
regression before it is buried under three more subsystems.

**1.1 Asset extraction (Python, one-time-per-checkout, reproducible)**

Stage 0 left working prototypes in `tools/` — `probe_house.py` (BinHex + house dump),
`probe_rez.py` (Rez → 538 resources), `probe_pict.py` (QuickDraw PICT), `probe_snd.py`
(`'snd '` → 70 PCM files), `probe_mov.py` (QuickTime), `extract_art.py` (the driver).
They were written to *answer questions*, not to feed a build, so 1.1 is mostly
promotion and gap-closing rather than new work:

- Promote the probes to a stable pipeline: one `tools/extract_all.py` driver writing
  `assets/extracted/` plus a manifest JSON, with the probe scripts kept as the
  inspection CLIs they already are.
- **Close the coverage gap.** `extract_art.py` today emits only `sheet/` (14 PNGs) and
  `object/` (93 PNGs). Still unextracted, and needed before 1.3 can draw anything:
  - the **18 room backgrounds**, PICT 2000–2017 — without these every room is blank;
  - the UI plates (splash 1000, scoreboard strip 1997, dialog art) and the loose
    `ICON`/`CICN`/`PAT#` resources;
  - the 15 house movies, if Stage 1 keeps the TV objects (`raw `, `rle ` and `smc `
    codecs — `probe_mov.py` already decodes all three).
- Carry the two decoder traps the analysis pass found the hard way: the extended
  PICT v2 header (`version == -2`) must be consumed and ignored, never rejected —
  21 of 152 pictures use it, including the glider sheet 3999 and the splash screen —
  and where a sheet has a mask resource, use the mask; do not key on white.
- *Acceptance:* every PICT and `'snd '` resource either extracts or is listed as
  deliberately skipped, with counts matching `docs/analysis/resource-fork.md`; all 18
  room backgrounds present; sprite sheets visually inspected against the original's
  splash screen art.

**1.2 House loading**
- `internal/house`: the `House`/`Room`/`Object` model, plus a binary loader that reads the
  original format byte-exactly (big-endian, Pascal strings — see `docs/analysis/house-format.md`).
- `cmd/glidertool house dump` prints a house as text.
- *Acceptance:* all 22 original houses load (4,070 rooms); room/object counts match the independently
  produced table in `docs/analysis/houses-inventory.md`; round-tripping a house through
  the text writer and back preserves every field (property test).

**1.3 Rendering a static room**
- `internal/render`: background, tiles, object layers, in the original's draw order.
- *Acceptance:* a chosen room from Demo House rendered to PNG and compared, by eye and
  then by checked-in reference hash, against the original's composition.

**1.4 Player physics**
- `internal/game/player`: the glider state machine, gravity, air, banking, foil/bands/helium.
- *Acceptance:* a trace test — scripted inputs produce a position/velocity trace that matches
  hand-derived expectations from `docs/analysis/player-physics.md` frame for frame.

**1.5 Objects, collision, room transitions**
- Object behaviours class by class, then the collision pipeline in the original's evaluation
  order, then room-to-room movement and per-room state persistence.
- *Acceptance:* a house can be traversed end to end headlessly; each object class has a test
  pinning its trigger condition and effect.

**1.6 Audio**
- `internal/audio`: mixer, the sound-event table, channel policy.
- *Acceptance:* a headless playthrough produces a WAV whose event ordering matches the
  expected sound sequence (verified by ear off-box, since this host has no sound card).

**1.7 The shell**
- Splash, menus, house selection, preferences, scoreboard, high scores, game over.
- *Acceptance:* a new player can start the game, pick a house, play, die, and see a score
  without touching a command line.

**1.8 Fidelity pass**
- `internal/fidelity`: frame-diff harness, input-trace replays, a checked-in corpus of
  reference frames.
- *Acceptance:* the "fidelity contract" list in `docs/ORIGINAL_GAME.md` is either satisfied
  or has an explicit, written exception.

### Stage 2 — new houses

- Text house format (`levels/*.house.txt`) + compiler + validator, sharing the original's
  legality rules (`docs/analysis/house-format.md`).
- A level-set concept: **Original** (the 22 shipped houses) and **New** (ours), chosen in
  the house-selection UI.
- New houses designed against the quantitative profile of the originals in
  `docs/analysis/original-houses.md` — comparable room counts, object vocabulary and
  difficulty curve, not just "some rooms".
- *Acceptance:* every new house passes the validator, is completable headlessly by a
  scripted run, and is playable start to finish by hand.

### Stage 3 — 2-player race

- `internal/net`: host listens on TCP, guest joins by address; length-prefixed JSON or a
  small binary framing over one connection. LAN discovery is a nice-to-have, not required.
- Host chooses the house(s) in play; both sides load the identical house set (hash-checked,
  so a mismatch is refused rather than producing a bogus race).
- Each side simulates only its own glider and sends a progress record on every meaningful
  change: room index, floor/suite, score, lives, alive/dead, finished.
- "Furthest on one life wins": the winner is decided by the agreed progress metric when both
  runs have ended; ties broken by score, then by time. The metric must be written down
  before the code — `docs/analysis/progression.md` defines what "furthest" can even mean in a
  house graph that is not linear.
- Both players see a live opponent panel (their room, score, and whether they are still alive).
- *Acceptance:* two processes on this host race to completion; killing the guest mid-race
  leaves the host in a defined state; a house-set mismatch is rejected with a clear message.

### Stage 4 — Windows

- `internal/platform/win32`: pure Go `syscall` to `user32`/`gdi32` (`StretchDIBits`) and
  `winmm` (`waveOutWrite`). No cgo, so it cross-compiles from this box today.
- *Acceptance:* `GOOS=windows make cross-windows` produces a binary that runs on Windows 10+;
  frame output diffed against the Linux build on the same input trace.

### Stage 5 — house editor

- Port the original's editor (`docs/analysis/editor.md`) as `cmd/glideredit`: tools palette,
  object placement, room/house info, linking, the map view, validation.
- Writes the text format; can also import a binary house for editing.

### Stage 6 — macOS, then maybe iOS/Android

- `internal/platform/sdl2` with a hand-written cgo binding (SDL2 is the one C dependency
  obtainable here, and it covers all three targets).
- Requires a Mac build host for signing and testing; treat as a separate project phase.

---

## 5. Testing and fidelity strategy

Fidelity is the whole risk of this project, so it gets real machinery rather than
good intentions:

| Mechanism | Catches |
|---|---|
| **Trace tests** — scripted input → recorded `(frame, x, y, vx, vy, mode)` for the player | physics drift, off-by-one in collision, tick-order mistakes |
| **Frame diffs** via the null backend against checked-in reference PNGs | draw-order and sprite-indexing regressions |
| **House round-trip property tests** | loader/writer field loss, endianness slips |
| **Object behaviour tests**, one per class, from the spec docs | subtle trigger/effect mistakes in the long tail of object types |
| **Constant audit** — a generated table diffed against `docs/analysis/constants.md` | a mistyped literal silently changing feel |
| **Headless completability runs** per house | levels that cannot actually be finished |

Reference data is derived from the *source*, not from a running Mac (there is no Mac and
no emulator here). Where the source is ambiguous, the ambiguity is recorded in the
doc's "Open questions" and the chosen reading is pinned by a test so a later correction
is a one-line change with a visible blast radius.

---

## 6. Risks and how each is handled

| Risk | Severity | Handling |
|---|---|---|
| PICT decoding is harder than expected (QuickDraw is a large format) | **high** — blocks all art | Only the opcodes actually present in these resources need implementing; `docs/analysis/graphics-assets.md` enumerates them from the real bytes. Fallback: render placeholder blocks and keep building the engine, since gameplay does not depend on final art. |
| Physics "feel" diverges subtly from 1994 | high — the point of the project | Constants ported literally with citations; trace tests; no refactoring of the update order. |
| No reference implementation to diff against (no Mac, no emulator) | medium | Source is the oracle; ambiguities recorded and pinned by tests. A Mac emulator on another machine would be a cheap future win if one exists. |
| House format has undocumented version variants | medium | Loader is written against the format spec *and* validated against all 22 shipped houses, which is the entire population of files that matter. |
| No audio hardware on the dev box | low | WAV sink; verify by ear elsewhere. |
| Cross-platform drift (three backends) | low | One input trace, three builds, diff the frames. |
| Project paused and resumed by a different session | medium | `docs/` is written to be read cold; `make check` proves the environment in one command; this plan is the entry point. |

---

## 7. Resuming this project cold

```bash
cd gliderGo
./scripts/bootstrap-dev-env.sh && . scripts/env.sh && make check
git log --oneline            # every stage is a commit with a detailed message
sed -n '1,60p' docs/PLAN.md  # you are here
```

Then read `docs/ORIGINAL_GAME.md` for the game, and the relevant `docs/analysis/*.md`
for whatever subsystem you are about to touch. The original C is in `GliderPRO/` and is
read-only — remember the CR-only line endings (`tr '\r' '\n'`).
