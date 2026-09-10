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

**1.1 Asset extraction (Python, one-time-per-checkout, reproducible)** ✅ *done*

Stage 0 left working prototypes in `tools/` written to *answer questions*, not to feed a
build. 1.1 promoted them to a pipeline and closed the coverage gaps:

- `tools/extract_all.py` is the one command (`make assets`): 908 files, 36 MB, 16 s into
  `assets/extracted/{res,art,sound,houses,movie}/` + `manifest.json`. The probe scripts
  stay as the inspection CLIs they already were, each also exposing its extractor as a
  function the driver calls.
- **Coverage closed.** `extract_art.py` now emits 179 PNGs, not 107: `sheet/` 14,
  `object/` 94, `strip/` 14, **`bg/` 18** (the room backgrounds — without these every
  room is blank), `ui/` 38 and `misc/` 1. `check_accounting()` partitions all 152 PICTs
  into the `graphics-assets.md` §7.8 buckets and raises on an unclassified, duplicated
  or phantom id, which is how the one genuine hole was found: PICT 10000, the
  `kCustomPict` placeholder, reached through a runtime atlas key rather than a numeric
  one, so the emitting branch had never run.
- Sound: 70 application `'snd '` + **58 of the 63 house sounds**; the 5 MACE 6:1
  resources are named as explicit `SKIPPED` rows with the reason, and play as silence
  until a MACE decoder exists. House sounds carry nine distinct sample rates, so the
  manifest's `rate_hz` is load-bearing.
- Houses: both forks written out, so **nothing in the Go port parses BinHex**. 22 houses,
  4,070 rooms, all version 0x0200; `Sampler`'s documented 2-byte PowerPC tail is
  tolerated by name and anything else is rejected.
- Movies: extracted after all — 0.4 s and 1.1 MB for all 15 as flat 8-bit index buffers,
  cheap enough that deferring them would have been the more expensive choice. Whether
  Stage 1 *draws* them is a 1.5 decision, not a data problem.
- The two decoder traps the analysis pass found the hard way are carried: the extended
  PICT v2 header (`version == -2`) is consumed and ignored, never rejected — 21 of 152
  pictures use it, including the glider sheet 3999 and the splash — and where a sheet has
  a mask resource the mask is used rather than keying on white.
- *Acceptance met:* all 152 PICTs and all 133 `'snd '` resources either extract or are
  listed as deliberately skipped, machine-checked against 9 counts that
  `docs/analysis/` derived independently; all 18 room backgrounds present and asserted
  512×322; `bg/2000.png`, `ui/1000.png` (the 1994 splash) and `strip/glider.png`
  inspected by eye. Extraction is deterministic — `make assets-check` re-extracts and
  diffs the manifests to prove it.

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
- Splash, menus, house selection, preferences, scoreboard, game over.
- **High scores** (owner-requested, and the original had them): a **10-row board per house**,
  each row `{name, score, timestamp, roomsVisited}` plus the house's `banner` — the original's
  292-byte `scoresType` at house offset 528 (`docs/analysis/scoring.md` §7.2).
  **Stored in our own per-house file, not in the house.** The original wrote the board back
  into the house file's data fork, but `GliderPRO/Houses/` is vendored read-only, so the port
  keeps the same ten rows and the same fields in its own container keyed by house name. The
  original's unreachable `'gliS'` side-car (`houseIsReadOnly` is hard-wired `false`) is not
  reproduced.
- *Acceptance:* a new player can start the game, pick a house, play, die, and see a score
  without touching a command line; a score good enough to place appears on that house's board
  and survives a restart; the 22 original houses' shipped boards are read and displayed but
  never written back.

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
  runs have ended; ties broken by score, then by time. **The metric is rooms visited** — which
  is not invented for the race: it is what the original's own high-score table records in
  `scoresType.levels[]` (`docs/analysis/scoring.md` §7.2), despite the field's name. That
  matters because a house graph is not linear, so "how far" has no obvious geometric answer;
  counting distinct rooms entered is the definition the 1994 game itself used, and 1.7 already
  has to compute it for the score board (`docs/analysis/progression.md:528` confirms the
  reading of the field). Rejected alternatives, recorded here because nothing in
  `docs/analysis/` argues the case: **room index** (meaningless — the house is a graph and
  indices are authoring order), **floor number** (undefined across suites, and some houses run
  sideways), and **score** (already the tie-breaker, and it rewards farming bonuses in one room
  rather than travelling).
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
| ~~PICT decoding is harder than expected (QuickDraw is a large format)~~ | **retired in 1.1** | Nine opcodes covered all 152 pictures. 179 PNGs extracted, accounting self-checked, art inspected on screen. Kept in the table because the reasoning generalises: only the opcodes actually present needed implementing, which `docs/analysis/graphics-assets.md` had enumerated from the real bytes before a line was written. |
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
make assets                  # assets/extracted/ is gitignored; regenerate it (16 s)
git log --oneline            # every stage is a commit with a detailed message
sed -n '1,60p' docs/PLAN.md  # you are here
```

Then read `docs/ORIGINAL_GAME.md` for the game, and the relevant `docs/analysis/*.md`
for whatever subsystem you are about to touch. The original C is in `GliderPRO/` and is
read-only — remember the CR-only line endings (`tr '\r' '\n'`).
