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
  function the driver calls. (1.3 added a sixth bucket, `houseart/`, for the 919 PICTs in
  the houses' own resource forks, which takes the totals to 1,899 files, 46 MB, 57 s.)
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

**1.2 House loading** ✅ *done*

`internal/house` is the model and both codecs; `cmd/glidertool` is the CLI over them.

- **Byte-exact binary codec.** `Load`/`Save` treat the file as the 866-byte header
  (`offsetof(houseType, rooms)`, not `sizeof`) plus `nRooms × 348`, tolerating exactly the
  0 or 2 trailing bytes the format permits and rejecting any other tail with a message that
  cites the document. `Save(Load(b)) == b` for all 22 shipped houses, residue and stale
  saved games included.
- **The union is bytes plus typed views.** `Object` is `{What int16; Data [10]byte}` with
  nine getter/setter pairs, because the shipped data forbids anything tidier: an undefined
  `what` must survive a round trip, empty slots carry meaning-free bytes in 66,236 of 66,240
  cases, and slot indices are load-bearing (links address objects by index, and six shipped
  rooms have holes). Variants are selected by inclusive range, never by the high nibble —
  0x10 `kLiftArea` is a blower and 0x40 `kDeluxeTrans` is a transport.
- **A text format that is both.** The default output is field-exact and readable — 24 KB for
  Demo House against 16 KB of binary — with `-residue` restoring byte-exactness (80 KB, four
  fifths hex). `Canonical()` states precisely what the readable form drops, so
  `ParseText(WriteText(h)) == h.Canonical()` is a testable claim rather than a hope. The
  writer is idempotent in both modes, which is what lets an authored house live in git.
  Booleans are written as integers because the corpus proves they are not booleans (a blower
  with `state` 23; `unusedBoolean` of 185 and 255).
- **`houseType.timeStamp` is masked, and the analysis was wrong about it.** `WriteHouse` does
  `timeStamp &= 0x7FFFFFFF`, which does not merely clear a sign bit — it discards bit 31 and
  moves every house date ~68 years early. Restored, the 22 houses date 1995-06..1995-12
  (Sampler 2000-05), and 11 of them match the newest *unmasked* score stamp in the same file
  to the day. Written up as `house-format.md` §3.3.1 and pinned by a test.
- *Acceptance met:* all 22 houses load, 4,070 rooms, 31,440 live objects. The corpus tests
  **parse their golden numbers out of `docs/analysis/houses-inventory.md`** rather than
  copying them in, so agreement with `tools/probe_house.py` — a different implementation, in
  a different language, written first — is evidence about the format and the document cannot
  rot: per-house sizes/versions/room/object/type counts, all 117 per-code instance and
  houses-using counts, 69 stars, 6 non-compacted rooms, `visited` 3634/436, 15 locked / 7
  unlocked. Both text round trips hold for every house. `make houses` re-checks all of it
  through the CLI in under a second.

**1.3 Rendering a static room** ✅ *done*

`internal/render` composes a room exactly as `DrawLocale` does; `glidertool render` writes the
result to PNG, which is how it was checked.

- **Indexed, not RGBA, all the way to the last step.** The original's furniture shadows are
  `PenPat(gray)` + `PenMode(patOr)`, which ORs *palette indices*: `k8DkstGrayColor` (254) over
  an arbitrary background gives whatever `254|bg` names. Compositing in RGB would produce
  plausible colours that are not the game's. `Surface` is therefore an 8-bit index plane plus a
  mask plane, and the palette is applied once, in `ToRGBA`.
- **The palette is generated and then proved.** `TestPaletteMatchesClut` checks all 256 entries
  against the shipped `clut` 128 *and* 129 — they are byte-identical — and asserts the two
  properties the rest of the renderer leans on: index 0 is pure white, so the white colour key
  and a mask built from it are the same operation; and RGB→index is injective, so the extracted
  RGBA PNGs go back to indices losslessly.
- **All nine local rooms, because the 640x480 screen shows all nine.** `numNeighbors` is 9
  unless the screen is 512 wide or less (`Main.c:191`), so slivers of eight neighbours and both
  floor-support bands are visible and had to be drawn. Far rooms first, central last, which is
  what lets an object overhang a room boundary and be overdrawn by its own room.
- **Object draw order is a 117-case transcription** of `DrawARoomsObjects`, with the original's
  oddities reproduced and commented rather than tidied: the eight objects that skip the
  intersection test, `kMirror`'s region add sitting outside both guards, `kToaster` lacking the
  `isLit` gate that `kShredder` beside it has, `kStereo` answering with `isPlayMusicGame` instead
  of its own state, and the bare `originV` in `kTable`/`kDeckTable`.
- **Two of the original's bugs are now visible here, and were the hardest part to be sure of.**
  `DrawRoomBackground` sets the port to `workSrcMap` and never restores it, so every later
  helper's "restore" restores to the scratch map — which means `DrawCabinet`'s outline and
  `DrawCounter`'s drawer panels are drawn into scratch and then erased by `RestoreWorkMap`.
  Cabinets have no border and counters no drawer panels in the shipped game, and they have none
  here. Reproducing it needed proof that a lit room always reaches the tile loop before its
  objects are drawn; it does.
- **Resource caps are modelled because they change the picture, not just the bookkeeping.**
  `kMaxSavedMaps` is 24, and candles, tikis, coals, pendulums, stars and grease each consume a
  slot (a star consumes two). Objects are processed in slot order, so the cap decides *which*
  clocks and prizes get drawn at all. The golden file pins the table sizes beside the pixels.
- **House resource forks shadow the application's for every ID, not just IDs ≥ 3000.** 51 of the
  919 shipped house `PICT`s override application art, and three of the overrides are drawn by
  this code path: Metropolis's own floor-support beam (1999) and Fun House's own 2014 and 2015,
  which are two of the eighteen *built-in* backgrounds. Written up as
  `docs/analysis/graphics-assets.md` §7.12.1.
- *Acceptance met:* every room of every shipped house composes — **4,070 rooms, 22 houses, no
  missing resource and no off-palette pixel outside the 24 pictures that carry their own
  `ColorTable`** — and each one's index plane is pinned by a hash in
  `internal/render/testdata/locale_golden.txt` (4 s for the lot). Demo House, Slumberland and
  Teddy World rooms were compared by eye against the original's composition.
- *Deviations, stated plainly:* the hash pins **our** output, since there is no 1994 framebuffer
  to hash — fidelity was established by eye and the hash freezes that judgement. `ColorOval` is
  a per-pixel ellipse test rather than QuickDraw's region builder (affects only dithered
  furniture shadows). `kCalendar`'s month name is not drawn yet — it needs a font engine, which
  arrives with the shell in 1.7. The 24 off-palette house pictures get nearest-in-RGB matching
  instead of the Color Manager's inverse table, counted and reported by
  `Assets.Approximations()`. The `isDepth == 4` 16-grey branch of every helper is not ported.
  `masterObjects` and the link table are deferred to 1.5; nothing they produce reaches a pixel.

**1.4 Player physics** ✅ *done*

`internal/game/player` is the glider: the 24-mode state machine, the integrator, input, the hit
box and the four room boundaries. 3,140 lines of code and 1,533 of tests, transcribed from
`Player.c`, `Modes.c`, `Input.c` and `Interactions.c:54-752`.

- **Integer pixels per frame, everywhere, deliberately.** No floats, no fixed point, no
  sub-pixel accumulator, no delta time. Position lives entirely in QuickDraw rects and the whole
  of the damping is a ±2 ramp toward a desired velocity. Free fall is 0, 2, 3, 3 — terminal
  speed 3 px/frame is *emergent* from the ramp meeting `kGravity`, not a constant anyone wrote.
- **`MoveGlider` resets `hDesiredVel` to 0 and `vDesiredVel` to `kGravity` inside itself**, so
  every accelerating influence has to be re-applied every frame. This is the single most likely
  way to get the subsystem wrong: set thrust once and the glider coasts forever.
  `TestHoldRightTrace` exists to fail loudly if that ever happens.
- **The horizontal clamp is ±16 and there is no vertical clamp at all.** The clamp lives *inside*
  the sign branches, which is why the battery — which adds to `hVel` directly, bypassing the ramp
  — still gets pinned at exactly 16. Vertically nothing is capped: `vVel` doubles as a positional
  snap in three places and can be arbitrarily large. A symmetric "for safety" clamp would break
  grease, wall bounces and ceiling snaps at once, so `TestNoVerticalClamp` drives a velocity of 40
  through a frame and checks all 40 pixels arrive. What the missing clamp does *not* mean is that
  the world can kick the glider by a whole amount in one frame: a ceiling vent assigns
  `vDesiredVel`, so its 8 is a ramp target reached over three frames, and the writes that do target
  `vVel` directly are still docked one `kVImpulse` on the way, because the interaction pass runs
  before `HandleGlider` in the same frame. That asymmetry — no clamp, but never a clean kick — is
  the subtlest thing in the subsystem and the one the spec had backwards.
- **The overloaded fields are kept overloaded.** `frame` is a fade index, a sprite index, an
  animation phase, a `kWasBurning` sentinel, a **y coordinate** and a negative countdown depending
  on mode; `wasMode` is a burn fuse *and* a saved mode; `hVel` is the idle countdown. Splitting
  them would look tidier and would break the place the aliasing is load-bearing:
  `MoveGliderShredding` tells its two phases apart with `frame > 0`. The `kWasBurning` sentinel is
  transcribed too, and is dead — a burning glider is faded out by the interaction gates before it
  can reach a staircase — but the reason it is dead lives in 1.5, not here, so it stays.
- **`batteryTotal` is one signed counter for two power-ups**: positive is battery charges,
  negative is helium, and the same keypress does mechanically unrelated things either side of
  zero (`hVel` directly for one, `vDesiredVel` through the ramp for the other). Both walk it
  toward zero, which is why `DoHeliumEngaged` *increments*.
- **`GliderHitTop`'s name is the opposite way round from what it does**, twice over. It rewinds the
  hit box by `wasHVel`, undoing the frame's horizontal motion, and re-tests: still overlapping means
  the contact was not caused by moving sideways (return true), overlap gone means it ran into the
  side — which the function then handles itself, spending foil and reflecting `hVel`, and returns
  false. So the *false* return is the one that has been dealt with. And at its only caller the
  *true* return is fatal: foil protects a glider against the side of a dissolving object and not
  against landing on it. That rewind is also why `wasHVel` must be rewritten on a stationary frame
  too.
- **Every boundary has two thresholds, and they are different numbers.** `CheckGliderInRoom`
  triggers on the inner limit (ceiling 8, floor 312) but the escape functions require the outer
  one (−10, 332) before letting the glider through; the gap is what makes walking out of a room
  take several frames. Open *sides* are the exception and have no distance check at all.
- **Two-player exits are a race, and there are two different strictnesses of it.** The transit
  handlers let the second arrival through unconditionally; the wall/ceiling/floor checks add a
  third outcome that *refuses* a glider whose partner left by a different exit, so the first one
  out chooses the route for both. Both are reproduced separately rather than merged.
- *Acceptance met:* **57 assertions pass**, every expected number hand-derived from the C before
  the test was run. The traces from `player-physics.md` §7.3 are pinned frame for frame (free
  fall, hold-and-release, helium, battery steady state), as are the fade lengths, the 60-frame
  burn fuse, the two-phase shred (45 fast frames, 17 grinding, then a 68-frame countdown), the
  sound throttle's 1/5/9 pattern, and one test per boundary verdict.
- *Independently audited against the C.* A 73-agent pass over `player-physics.md` found **57
  confirmed errors in the spec** — fused `if` conditions, omitted range guards, inverted outcomes,
  drifted line citations — and a second pass then re-derived every one of them against the Go, with
  three adversarial refuters standing by for any claimed divergence. **None was needed: 46 findings
  are code-faithful and 11 describe behaviour that is Stage 1.5's.** The port survived its own spec
  being wrong because it was transcribed from `Interactions.c` and `Player.c` with the document as a
  map rather than a source; the 57 corrections are now in the document, which grew by a thousand
  lines in the process. Thirteen Go doc comments that had repeated the spec's claims were corrected
  too — the code was right and the comments above it were not, which is the failure mode this kind
  of transcription is most prone to.
- *Deviations, stated plainly:* the outward calls (sound, transitions, scoreboard, inventory) are
  an `Env` interface with a `NopEnv` for tests, because the original is one mutually recursive
  tangle of file-scope globals — 1.5 implements it for real. `GetInput` takes a resolved `Keys`
  value per glider instead of a shared `KeyMap`, which preserves the original's one-poll-two-
  gliders property while removing its ordering hazard. Nothing else is refactored: where the C
  has eight functions that four would cover, this has eight.

**1.5 Objects, collision, room transitions** — ~9,100 lines of C, six sub-stages

The largest stage in the project: about twice everything committed so far, at 1.4's C-to-Go ratio
roughly 9,000 lines of Go and 4,000 of tests. It was one four-line bullet until a reverse-
verification pass sized it; the decomposition, the dependency argument and the per-type behaviour
tables are `docs/analysis/stage-15-spec.md` (5,530 lines) §5.

Four decisions have to be taken inside it rather than left floating, each with an owning
sub-stage: the build configuration (`COMPILEQT` and `BUILD_ARCADE_VERSION` are both defined and
between them add four `RenderFrame` call sites); the fact that **`RenderFrame` is not idempotent
and runs twice on any frame a transition completes**; hoisting the pending-transit triple
`transRect`/`transRoom`/`linkedToWhat` off the glider, where 1.4 put it, and onto the world, where
the C has it as one shared global that `FollowTheLeader` reads from the *dead* leader; and the
`DrawLocale` reset order, which two source reports had wrong.

**1.5a The world and the room: live state, object graph, hot-spot table, room load** — ~2,230
lines of C ✅ *done*
- `internal/game` gains `World` (session-, game- and house-scope state) and `Room` (what
  `DrawLocale` rebuilds), scoped on **the C's own reset sites** rather than on intuition: session =
  `CreatePointers`, game = `Play.c:112-118` + `InitGlider`, room = `DrawLocale`
  (`RoomGraphics.c:51-60`). `Room` embeds `*render.Scene`.
- Ports `Objects.c:89-699` (the link resolvers, `ListAllLocalObjects`, `SetObjectState`),
  `ObjectRects.c:277-1063` (`AddActiveRect` and all 117 cases of `CreateActiveRects`), `Room.c`'s
  runtime half (`DetermineRoomOpenings`, `GetNeighborRoomNumber`, `IsRoomAStructure`, the
  floor/ceiling/shadow predicates), `RoomGraphics.c`'s `ReadyLevel` and `DrawLocale` reset head,
  the four `savedMaps` primitives, and `Play.c`'s `SetObjectsToDefaults` and `InitGlider`.
- `CreateActiveRects` was the single largest unspecified piece: **93 of the 117 types produce a hot
  spot and 24 produce none**, and fifteen of the undocumented ones are lethal `kDissolveIt` solids
  — `kTable`, `kCounter`, `kInvisObstacle` among them. A port that omits them ships a house full of
  pass-through furniture. The per-type table now lives in the spec §3.2-§3.10.
- `Rebuild` must be a **method** on `World`, not a constructor: `RestoreEntireGameScreen` reaches
  `DrawLocale` (`Play.c:814`) during normal play without `NilSavedMaps` or `InitGarbageRects`.
- *Acceptance:* **all 4,070 rooms of all 22 houses build a `masterObjects`/`hotSpots` pair**, each
  pinned by a per-room hash over the counts, every master entry's seven indices and every hot
  spot's `(bounds, action, who, isOn, doScrutinize)` — the same corpus-scale proof 1.2 and 1.3
  were accepted on. Each of the 93 hot-spot-producing types is exercised by a shipped room or
  listed as never instantiated. `DetermineRoomOpenings` and `GetNumberOfLights` are asserted
  against hand-derived values for one named room per house. `SetObjectState` is driven with all
  four actions against all 117 `what` codes without panicking.

**1.5b The frame and the traversal: frame loop, hot-spot dispatcher, room transitions** — ~2,510
lines of C ✅ *done*
- `Play.c`'s `PlayGame` loop and game-over tail, `Render.c`'s frame spine and dirty-rect protocol,
  `Interactions.c:1198-1777` (`HandleHotSpotCollision`'s 28 cases, `CheckForHotSpots`,
  `FlagStillOvers`, `WebGlider`), all of `Transit.c` and `Transitions.c`, `OffAMortal`.
- **This is where the game becomes playable**, because it is where `Env`'s 46 methods get real
  implementations. Land it in two commits: the movement actions and `Transit.c` first, then the
  lethal and cosmetic ones. `kRewardIt`/`kSwitchIt`/`kTriggerIt` stay stubbed until 1.5d.
- *Acceptance:* `*game.World` implements `player.Env` in full and `NopEnv` appears only in
  `_test.go`. **Demo House's first room is left by each of the seven exit kinds** — left door,
  right door, both staircases, transporter, ceiling duct, mailbox — as seven scripted replays
  asserting destination room, arrival mode and arrival rect against hand-derived numbers. A
  600-frame replay pins `gameFrame`, `evenFrame`, the two dirty-rect counts and `clockFrame` per
  frame, which is what pins the double `RenderFrame`.
- **Two corrections to that acceptance text, found while meeting it.** Both are recorded in
  docs/IMPROVEMENTS.md 4.2; they are repeated here because this is the paragraph somebody
  reads first. (1) It cannot be *Demo House*: that house contains four of the seven exit kinds
  in the whole file and its first room has none of them. **CD Demo House** has all seven and is
  what `internal/game/exits_test.go` uses, choosing one room per kind by inspection. (2) There
  is no `clockFrame` to pin: the port's pendulum animation is `RenderPendulums`, a named empty
  stub charged to 1.5f, so the trace pins `len(Scene.Pendulums)` in its place — which proves
  the pendulum reached the locale even though nothing swings it yet. Add `clockFrame` to the
  trace when 1.5f lands. **Done: 1.5f added the `clock` column**, and the pair is what makes the
  swing legible — `pend` says a clock is in the room and `clock` says whether this was one of the
  two frames in fifteen it moved on. `len(Scene.Pendulums)` stayed, for the same reason it went
  in.

**1.5c Dynamics: the `dinahs` table, appliances, movers, toggles, triggers** — ~2,510 lines of C ✅ *done*
- `Dynamics3.c`'s `AddDynamicObject`/`HandleDynamics`/`RenderDynamics`, all of `Dynamics.c` and
  `Dynamics2.c`, all of `Trip.c` and `Triggers.c`, sparkles and flying points.
- Before rewards and switches **on purpose**: the dependency graph has one genuine cycle, since
  `HandleSwitches` dispatches into all fourteen `Toggle*` while `Trip.c`'s `TriggerSwitch` calls
  back into `HandleSwitches`. Dynamics first leaves one stub — `TriggerSwitch`, and 1.5d closed
  it; switches first would have left fourteen.
- `CheckDynamicCollision` is a **second, earlier** collision channel and must not share an
  implementation with `kDissolveIt`: different mode gate, and one sheet of foil per two frames
  against two sheets every frame.
- *Acceptance:* each of the 17 registrable types pins its post-registration slot fields and one
  frame of its handler; all fourteen `Toggle*` and eight `Trigger*` have a test, and the seven
  target types `FireTrigger` silently ignores are pinned as no-ops; trigger timing asserted at
  three delays; the 18-slot cap asserted against a checked-in census of the busiest shipped room.
- **All five clauses met, in `internal/game/dynamics_test.go` and `trip_test.go`.** The census is
  `TestBusiestShippedLocaleSaturates`, which sweeps all 22 houses and names the answer:
  `California or Bust!.house` room 10 "And the Pets, Too" holds exactly 18 registrable objects,
  three locales in two houses reach the cap, and **none exceeds it** — so
  `AddDynamicObject`'s `return -1` is live but never taken by a shipped house. That closes open
  question 11 of `docs/analysis/object-dynamics.md`, and it means a Stage 2 house has to be
  checked against the cap rather than assumed under it.
- **Two files of depth beyond the stated minimum**, because the six movers and the two effect
  tables turned out to hold most of the sub-stage's real behaviour: `movers_test.go` pins the
  trailing-union invariant for all ten registrable non-appliance types over four frames each,
  the per-handler `EvenFrame` gating, and each mover's retire conditions; `sparkles_test.go`
  pins both effect tables as free lists — including the trap that a table not swept by
  `InitGarbageRects` reads as three live effects and silently drops everything with no counter
  drift — plus the flying point's exact 72-frame lifetime and its hard-against-the-array-end
  cel walk.
- **Four findings recorded rather than fixed**, all in `docs/IMPROVEMENTS.md`: the ungated
  ball/drip/fish renderers spend 17 of the 47 dirty-rect slots standing still in `SpacePods`
  room 55 (2.11); a four-frame enemy reload emits no warning sparkle and is unreachable only
  because `Count = Delay * 3` (2.36); the television's movie branch is a deliberate blank while
  all 15 shipped movies sit already extracted under `assets/extracted/movie` (2.37); and 2.35's
  `evenFrame` bullet is now measured — the desynchronisation is **one frame wide, not
  permanent**, because only the loop head toggles and the other three writers assign, so the
  composition write and the ball's first idle write cancel.
- **One correction carried into 1.5e, and closed there.** 2.34's `PaintRect`-with-no-destination
  was resolved for the outlet in 1.5c (`HandleOutlet` names `w.R.Work` at the call site) and for
  `HandleGrease` in 1.5e, which names both maps at the call site and registers the rect it filled.

**1.5d Rewards, switches, per-room persistence** — ~465 lines of C ✅ *done*
- `Interactions.c:756-1194` (`HandleRewards`' fifteen prizes, `HandleSwitches`, `HandleMicrowave-
  Action`) plus `DisplayStarsRemaining`. Unstubs the three actions 1.5b left and `TriggerSwitch`.
- Small in lines, large in surface: this is where **per-room persistence becomes observable**,
  because every prize and switch writes a `state` byte back into the house through `SetObjectState`
  and that write is what survives leaving and re-entering a room.
- *Acceptance:* **each of the fifteen reward cases and each of the 23 switch-dispatch cases has a
  test pinning its trigger condition and effect** — score delta, inventory delta, sound, and the
  `state` byte written back — which is the per-object-class criterion the old 1.5 bullet asked for.
  Collecting a prize, leaving and re-entering finds it gone, for one of each of the twelve
  state-gated types. Taking the last star sets `gameOver` within one frame.
  (The bullet said **21** switch cases from 1.5c's estimate; counted from `Interactions.c:1040-1147`
  the answer is **23**, and the count matters because the two extra are among the three that turn
  out to be unreachable.)
- **All four clauses met, in `internal/game/rewards_test.go` and `switches_test.go`.** The reward
  table asserts all eleven columns on every one of the fifteen rows — including the zeroes, since
  the subject is a mapping and a row that awards nothing is as much a specification as one that
  awards 5,000 points — and the switch table covers all 23 arms across 42 object types. (The
  reward table has fifteen rows over fourteen arms: `kGreaseRt` and `kGreaseLf` share a body and
  a test still has to prove both directions reach it.)
- **The stage found and fixed two real defects in the port**, both in `docs/IMPROVEMENTS.md`'s
  repairs list. `HandleRewards` was **missing the `kHelium` arm entirely**: a transcription slip
  that built and ran and simply did nothing when a glider touched a helium balloon. And
  `World.NewState` was a local, so **every switch lever drew the wrong state** — a light switch
  animated to "off" while turning a lamp on, which is invisible without art and is now pinned by
  a pixel comparison.
- **The test that found the helium bug is the transferable part.**
  `TestEveryDispatchableRewardIsConsumed` does not carry a list of reward types; it sweeps every
  object code through `CreateActiveRects`, collects the ones that produce a `kRewardIt` rect, and
  requires `HandleRewards` to consume each. A hand-written list would have had the same hole as
  the switch statement it was checking. Every remaining function in the port that dispatches on
  object type is a candidate for the same shape of test.
- **Three findings recorded rather than fixed**, all in `docs/IMPROVEMENTS.md`, and the first is
  the one that matters to Stage 2: a switch wired to a **star** removes it without decrementing
  the star count, so **an author can build a house that cannot be finished** (2.38) — no shipped
  house does, so the fix is free and is scheduled before the first new house ships; the switch's
  uninitialised sparkle rect puts a stray puff at the corner of the play area **145 times across
  the 22 original houses**, so it is something players of the shipped content actually see (2.39);
  and three of the 23 arms — `kSlider`, `kSoundTrigger`, `kGuitar` — have never run in any build,
  because `SetObjectState` returns false for all three, which means **a switch wired to a sound
  trigger has never played its sound** (2.40).
- **A corpus survey answers "does this matter?" with a number, and its first version was wrong.**
  `TestShippedHousesWireSwitchesToPrizes` walks all 1,563 switch and trigger plates in the 22
  houses. The first draft read `data.e.where` as a room index — it is a packed floor/suite pair —
  resolved nothing, and reported a reassuringly clean corpus. It now resolves links through
  `GetRoomLinked`, the way the game does, and **fails outright if no link resolves**, so a broken
  predicate can no longer masquerade as a clean result. That guard is the reusable lesson from
  this sub-stage's tooling: a survey that can find nothing must distinguish "nothing there" from
  "not looking".
- **The 600-frame golden trace changed, and the change is the stage's own proof.** Room 5 has a
  `kInvisSwitch` flat against the ceiling, ForceOn, wired to a room-sized `kDeluxeTrans` that the
  house file ships switched off. With `HandleSwitches` a stub the transporter stayed off and the
  unattended glider bobbed under the ceiling for the whole run; now the glider trips the switch on
  frame 224 and is transported to room 70, "Welcome…", on frame 240 — so the trace covers a switch
  throw, a state change publishing a hot spot that did not previously exist, a room-to-room
  transport and a second room. It also *lost* six of the toaster's nine bursts by leaving room 5,
  which is written up as `docs/IMPROVEMENTS.md` 4.3 and is an argument for several short scripts at
  1.8 rather than one long one.

**1.5e Bands and grease** — ~630 lines of C ✅ *done*
- All of `RubberBands.c` and `Grease.c`, plus `RenderBands`. Three new source files —
  `internal/game/bands.go`, `internal/game/grease.go`, `internal/render/grease.go` — plus the
  `BandRects` table in `internal/render/srcrects.go`, and **six stubs closed**: `AddBand`,
  `HandleBands`, `RenderBands`, `HandleGrease`, `SpillGrease` and `RedrawAllGrease`. Two
  additions carry them: `World.BandHitLast`, and `Scene.KillAllBands` wired as a hook in
  `Rebuild` — the fifth of the cross-boundary hooks and the only one whose nil composes an
  identical image.
- Grease is split across the render boundary and the split is not arbitrary. The **table** lives
  on `Scene`, unlike the dinahs table, which is behind a hook: a nil dinahs hook composes the
  same image, because an unregistered dinah is simply a still one, but a jar's *draw* is gated on
  its registration succeeding, so a nil hook there would compose a room with no grease jar in it
  and the renderer's own goldens would stop showing one. A table both sides write is the honest
  description, so that is what it is.
- The last two writers into `hotSpots[]`, and what makes the table **mutable mid-frame**:
  `HandleGrease` runs *inside* `RenderFrame` and rewrites hot-spot bounds after the interaction
  sweep, so a slide rect created on frame N is not collidable until N+1.
- *Acceptance met.* All five filtered actions, the debounce, the two-band cap, the wall bounce,
  the doorway kill, the floor kill and the phase-1-clamp survival have tests
  (`internal/game/bands_test.go`); a jar walks all four modes with the slide rect pinned per
  frame, and the registration half is `internal/render/grease_test.go`.
- **What the acceptance criteria actually found**, in the order the tests failed:
  - **Phase 1 tests the room's *thresholds* and phase 5 tests the wall *constants*.** So in a
    room with an open side phase 1 declines to clamp and phase 5 deletes the band the moment it
    passes x=12 anyway: **a rubber band cannot travel through a doorway.** That is a rule of the
    game — it keeps bands a within-room tool and stops a player clearing a room they cannot see —
    and it reads as an inconsistency until you notice the other half. The two interlock the other
    way too: phase 1's clamp writes `left = kLeftWallLimit` *exactly*, so phase 5's `<` is false
    by one pixel and the rebounding band survives the frame it rebounds on. One pixel is the
    whole margin.
  - **The debounce does not debounce**, in three independent ways, all reachable. Written up as
    `docs/IMPROVEMENTS.md` 2.41 and transcribed rather than fixed, per 1.8's replays.
  - **Only five of 28 actions see a band, and `kRewardIt` is filtered to grease.** Bands cannot
    collect prizes — a design decision hidden inside a type test.
  - **The band sound is *inside* the two-player escape guard, and the transfer is not gated on
    having an effect.** So two live gliders touching one band give two thuds and one shove, where
    one dead glider gives one thud and one shove. The pair pins the difference between "the arm
    ran and did nothing" and "the arm did not run".
  - **`KillBand` is a swap-remove that does not clear the slot it copies out of**, which is why
    `AddBand` opens with `mode = 0` rather than trusting the slot it is handed.
  - **The grease tip is four frames, not three**, because `Frame` starts at -1 and `HandleGrease`
    pre-increments. The jar steps two pixels on the frame it becomes a slick as well, so it has
    moved eight — which is exactly the ∓8 `AddGrease` subtracts to undo `backupGrease`'s walk.
- Two comments naming tests that did not exist were found and closed, and the class is charged to
  1.8 as a `make check` lint: `docs/IMPROVEMENTS.md` 4.4.

**1.5f Background animations and the saved-map economy** — ~800 lines of C ✅ *done*
- The rest of `DynamicMaps.c` — the five `BackUp`/`ReBackUp`/`Add` triples, shreds — and
  `Render.c`'s four animation passes. Three new source files — `internal/render/anim.go`,
  `internal/game/anim.go`, `internal/game/shreds.go` — and **the last four renderer stubs
  closed** (`RenderPendulums`, `RenderFlames`, `RenderStars`, `RenderShreds`) plus
  `AddAShreddedGlider` and `RemoveShreds`, which were a stub and a wrong no-op respectively.
  `internal/game/env.go`'s list of unimplemented functions is down to **two**, `DoPause` and
  `DoCommandKey`, and both belong to 1.7.
- Two more cross-boundary hooks, bringing the total to seven: `Scene.ZeroShreds`, which is
  `numShredded`'s line in `ZeroFlamesAndTheLike`, and `Scene.RandomInt` — **the only hook whose
  return value the composition depends on.** With it nil every flame starts on cel 0, which is
  deterministic and invisible in a still image, because a filmstrip lives in a saved map and never
  reaches the composed frame. That is what lets the renderer's own goldens leave it unbound.
- Last, and the only purely cosmetic sub-stage — with one exception that earns it real care. The
  five room-load `RandomInt` draws are each *inside* `if (savedNum != -1)`, so when the 24-slot
  `savedMaps` table saturates the draw does not happen and **the whole downstream RNG stream
  shifts**. Registration is screen-size dependent through `SectRect`, so the RNG stream is a
  function of resolution. That matters for Stage 3. Written up as `docs/IMPROVEMENTS.md` 2.42,
  which also records what protects it: 2.8's rule that scaling happens at the present step keeps
  the *composed* area fixed at every window size.
- **The load-bearing discovery is what a saved-map slot holds.** The five animated families do not
  back up a background and composite over it; each claims a strip **one cel wide and N cels tall**
  and bakes into it, per frame, the back map with that frame's art masked on top. So an animation
  step is one *opaque* blit out of the strip and registers **no back rect** — the new frame erases
  the last by covering it. Claim one cel instead of N and the result is a lit, perfectly still
  candle: the room looks composed and nothing moves.
- *Acceptance:* each of the six animated families steps a full wrap cycle with the strip index and
  `src` rect pinned per frame; saturation asserted on the room a checked-in census names, with the
  dropped registrations enumerated. **Two headless replays of 1,200 frames from one seed produce
  byte-identical index planes.**
- **The first and third clauses are met; the second was written on a false premise and is met by
  the measurement that disproved it.** There is no room to assert saturation on: the census
  sweeps all 22 houses and **the busiest locale in 4,070 rooms claims 16 of the 24 slots**
  (`Slumberland` room 256 "Flaming Pathway", tied by `Teddy World` room 322), and **not one room
  drops a registration.** Rather than restate the clause to fit, the mechanism it asked for was
  built and the negative result was made permanent: `Scene.SavedMapDrops` records every refusal,
  the golden's new `dr=` column carries the count room by room,
  `TestTheSavedMapBudgetSaturatesInShippedContent` fails if any shipped room ever starts dropping,
  and `TestDroppedRegistrationsNameTheObjectThatVanished` fills the table by hand and enumerates
  the three refusals in order. The consequence is 1.5c's, one table over: the cap is a live path
  shipped content never takes, so a Stage 2 house has to be **checked** against it rather than
  assumed under it (`docs/IMPROVEMENTS.md` 2.44, and the third item on 4.1's linter list).
- The third clause needed something the harness did not have. `Result.Digest` hashes the sample
  trace, and the trace records dirty-rect *counts* — so a flame stuck on one cel, a pendulum
  swinging backwards or a confetti cloud emerging top-first registers the same rect as the correct
  version and moves nothing in it. **Everything this sub-stage added was invisible to the digest by
  construction**, which is presumably why the clause asks for planes. `Result.Planes` hashes the
  back map, the work map and the screen once, after the last frame, and `glidertool replay` prints
  the three beside the digest; which of them moved localises the fault to the composition, the
  animation or the dirty rects. `TestTwelveHundredFramesTwiceAreTheSamePicture`
  (`docs/IMPROVEMENTS.md` 4.5).
- **What the acceptance criteria found**, in the order the tests failed:
  - **The clock ticks unevenly, and the cel sequence is the opposite of a pendulum.**
    `RenderPendulums` acts only when `clockFrame` reads exactly 10 or exactly 15, resetting at 15,
    so the gaps alternate five frames and ten; and the cels drawn per cycle are 2, 1, 0, 1 — the
    centre twice and each end once, where a real pendulum lingers at the ends. Both are pinned
    against the tidy-up that would break them (`TestTheClockTicksUnevenly`,
    `TestThePendulumSwingsCentreEndCentreEnd`), and the trace's `clock` column is where a
    single-interval "simplification" would show up as a column stepping 0..9 for ever.
  - **A second clock in the same room is silent.** `playedTikTok` throttles to one sound per
    frame across every pendulum in the locale, and two pendulums in opposite phase flip on the
    *same* frames — so the first table entry claims every sound and the room still plays a normal
    alternating tick-tock while the second clock mimes.
    `TestTheFirstPendulumsSoundAlwaysWins`.
  - **`RenderShreds` is 54 frames, not 55.** Thirty-five growth frames and nineteen fall frames,
    where the sparkle *shares* its iteration with the last fall step. This file's own header
    counted it twice until `TestTheFallArmDropsFourPixelsAFrame` was written, and double-counting
    is exactly what an off-by-one in the fall arm would look like.
  - **`RemoveShreds` removes one cloud, and sometimes none.** The name is plural and `OffAMortal`
    reads as a sweep, but the body swap-removes the single most-advanced entry — and because the
    search is `frame > largest` from `largest = 0`, an entry still growing (frame 0) can never be
    chosen. A death with one still-growing cloud on screen removes **nothing**, and the confetti
    keeps growing through the respawn. Transcribed, with the `largest = -1` one-character fix
    named in the test so it is not applied by accident.
  - **`AddAShreddedGlider` writes one element past its four-element table** (`> kMaxShredded`,
    `Environ.c:654`). The port's guard is `>=` and goes through `badIndex`, so the refusal is
    counted and named in a bug report instead of being silent — the only guarded *write* in the
    port. `docs/IMPROVEMENTS.md` 2.43.
  - **`Anim.Mode` can legitimately be seeded one past the end of its strip**, because
    `RandomInt(n)` can return `n`. The wrap rescues it by resetting `Src` *absolutely* to
    `(0, celH)` rather than subtracting a strip height, so clamping the seed would be a silent
    divergence rather than a fix. `TestASeedOnePastTheEndIsLeftUnclamped`.
  - **Object draws during composition write `s.Back`**, so registration-versus-draw order is
    observable: the star registers before its own draw and the cuckoo draws before its pendulum
    strip is baked. Reordering either is a pixel change with no other symptom.

*Independently specified before any code.* Eight parallel readers reverse-verified one subsystem
each against the C, three adversarial critics attacked the result, and six writers produced the
spec from the corrected reports — about 2.3M tokens. It found what the one-bullet plan could not:
the 117-type hot-spot table nobody had written down, `SetObjectState`'s missing bounds check
against shipped houses that pass it `-1`, the non-idempotent `RenderFrame`, and that
`FlagStillOvers` **suppresses** a trigger rather than firing one — which `env.go` had documented
backwards and is now corrected. Two of its load-bearing claims were re-verified by hand against
`Objects.c` and `Interactions.c` before being acted on.

**1.6 Audio** ✅ *done*
- `internal/audio`: mixer, the sound-event table, channel policy.
- **Music too, which is a separate subsystem** (`Music.c`, not `Sound.c`): its own channel, its own
  on/off preference, and 1.3 already depends on it — `kStereo` answers the draw sweep with
  `isPlayMusicGame` rather than its own state (`ObjectDrawAll.c`, noted in 1.3 above).
- *Acceptance:* a headless playthrough produces a WAV whose event ordering matches the
  expected sound sequence (verified by ear off-box, since this host has no sound card); music
  starts, stops and survives a room change independently of the effects channels.
- **Both clauses are met.** `glidertool replay -wav` writes the file, and the ordering is asserted
  rather than eyeballed: the trace carries a `snd=` column naming every sound a frame asked for,
  its priority and the channel it landed on, so the 600-frame golden pins the sequence
  frame by frame — the duct out and in on frames 1 and 10, the cuckoo's `tik`/`tok` alternating
  every fifteen frames, the toaster's launch and land at the boundaries `RenderToast`'s arithmetic
  predicts, and the second transporter at 225/241. `TestTheWAVHoldsTheMix` reads the file back
  through its own header offsets and re-hashes the payload, which is what makes the mix digest a
  checksum anybody can verify with `sha256sum`; the ear is still owed off-box, and is the only part
  of this stage a machine here cannot do. `TestMusicIsIndependentOfEffects` is the second clause,
  and `docs/IMPROVEMENTS.md` 4.2 records what the audio adds to the bug-report format.
- **The engine has no clock and no goroutine**, which is the design decision everything else in the
  package follows from. It turns requests plus a sample count into samples; time enters in exactly
  one file (`pump.go`) as two methods — `FrameTick`, which mixes exactly `SamplesPerFrame` per game
  frame and is what the recorded path uses, and `ClockTick`, which mixes what the wall clock says is
  due and is what `cmd/glidergo` uses. So the two paths share every line of the mixer while only the
  first is bit-exact, and the live path's imprecision is a property of thirty lines rather than of
  the whole subsystem. The only goroutine in the package moves byte slices into a pipe.
- **`SamplesPerFrame` is 740**, from `kTicksPerFrame = 2` on the Mac's 60.15 Hz clock, and the assets
  confirm it independently: `Hiss` is 2960 samples, which is 4 × 740 exactly, and `Input.c:174` asks
  for it every fourth frame. Three more continuous sounds sit in the same arithmetic, which is why
  **the loop points in the headers are never needed and looping would be a bug** — a looped `Thrust`
  would keep firing after the key was released and its channel would never run its completion
  callback (`docs/IMPROVEMENTS.md` 2.45).
- **The audio changes the composition, in one specific way that had to be wired before the first
  room loads.** A `kSoundIt` object gets a hot spot only if its sound resource loads
  (`LoadTriggerSound` answering −1 means no hot spot at all), so a house with custom sounds composes
  differently with audio than without it — and five of the 63 shipped house sounds are MACE 6:1
  compressed and cannot be decoded from anything in this tree, so those rooms are faithful to the
  1994 build by accident (`docs/IMPROVEMENTS.md` 2.49). `sound off` is therefore a *different
  simulation*, not a quieter one, and the replay trace records which of the two it was in its
  header.
- **The house sounds are not all at the Macintosh rate.** Fourteen of the 58 usable ones are not:
  six at half, three at a third, one at a quarter, and four one-off numbers a sound editor wrote
  (22255.0, 22050.0, 11127.5, 9779.0). Each `Sound` therefore carries a 16.16 fixed-point `Step`
  and the mixer drop-samples, which is the original's own method rather than a shortcut — all four
  channels are created with `initNoInterp` (`Sound.c:379`, `Music.c:287`). `stepFor` rounds so that
  every spelling of the Macintosh rate lands on exactly `FixedOne` and the application's 70
  resources are copied rather than resampled; `TestOneSampleRate` fails if any of them stops being.
- **What the acceptance criteria found:**
  - **`FlushAnyTriggerPlaying` silences the 1994 game permanently.** It flushes the command queue
    that holds its own `callBackCmd`, so the callback that would restore `priorityN` to 0 never
    runs and the channel keeps claiming 999 for the session. Three interrupted trigger sounds and
    `PlayPrioritySound` refuses everything. The port resets the priority as though the callback had
    run — the only deliberate deviation in the audio path, and it can only make the port louder
    than the original, never quieter (`docs/IMPROVEMENTS.md` 2.47).
  - **The trace's `snd=` column belongs in the digest, and the samples do not.** A sound request is
    a decision the simulation made at a frame, so it is in `Result.Digest`; the mix is a separate
    hash on `Result.Audio`, because pinning samples in a golden file would make every legitimate
    mixer improvement a test failure. The pair is the same split 4.5 made for the pixels.
  - **A latent panic in `replay.Parse`**, found while adding the fifth directory keyword: the four
    existing ones indexed `fields[1]` unguarded, so a bare `artdir` line crashed the tool instead of
    being rejected. Now a sentence, and in `TestBadScriptsAreRejected`.
  - **The port has no audio driver and will need one for Windows.** The sink is a subprocess —
    `pw-play`, `paplay`, `aplay`, `ffplay` or `play`, whichever exists — which is a defensible
    answer on Linux and no answer at all on Windows, where nothing equivalent is in the box. It is
    the one part of the audio path Stage 4 cannot cross-compile (`docs/IMPROVEMENTS.md` 2.48).

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
- **Split into four, because "the shell" is four unrelated subsystems** sharing only a screen. The
  original interleaves them through the Toolbox — a menu handler that reaches into the preferences,
  a scoreboard that runs a modal dialog, a pause that borrows the game's own port — and porting them
  as one stage would mean one commit with four reasons to be wrong. The split is by *what owns
  state*: the shell owns which screen is up (a), a preferences file owns the settings (b), a
  side-car owns the boards (c), and the running `World` owns the in-game overlays (d).
  - **1.7a The way in** ✅ *done* — `internal/shell`: the splash screen, the menu, the house picker,
    the About box, and the status band. `cmd/glidergo` gains the title screen as its default and
    splits into `main.go` (flags, dispatch, host) and `play.go` (one game).
  - **1.7b Preferences and pause** ✅ *done* — `internal/prefs` (the native config directory, the
    JSON file, `Validate`, the legacy 226-byte `prefsInfo` importer), the settings screen, per-player
    key bindings (`docs/IMPROVEMENTS.md` 2.3), volume, music, scale, `DoPause` as a real pause state
    with PICT 1015/1016 (2.5, 2.32), and the three opt-in fidelity switches (2.19, 2.20, 2.39).
    `DoCommandKey` stays an empty stub with a paragraph saying why — neither of its two chords
    reaches this port's hosts — and Q takes over as the way out of a paused game. Music-on-the-splash
    became the preference `music_on_title`, which 1.7d makes audible.
  - **1.7c High scores** ✅ *done* — `internal/scores` (the board, `Qualify`/`Insert`, the two
    entry dialogs from DLOG/DITL 1020 and 1021, `DrawHighScores`' geometry with PICT 1994/1995/1998,
    and the per-house side-car `Store`), the shell's own High Scores screen and merged picker
    footer, and `cmd/glidergo/highscore.go` — the three blocking loops the host owns. The 22
    shipped boards are read and never written back. Plus the credits screen the shell has owed
    since 1.7a (`internal/credits`, `docs/IMPROVEMENTS.md` 1.2 and 3.4).
  - **1.7d The in-game shell** — `BringUpBanner` (PICT 1991-1993), `DisplayStarsRemaining`
    (1017/1018), both as simulated-frame waits rather than `Delay` (2.32); `DoGameOver` and
    `DoDiedGameOver` feeding 1.7c; `restoreSplashScreen`.
- **1.7a, what it settled.** Three decisions that the rest of 1.7 is built on:
  - **The shell does not import `internal/game`.** A game is reached through one hook,
    `Play(Choice) (Outcome, error)`, which `cmd/glidergo` fills in. So the whole of the way into the
    game is testable with no window, no house, no sound card and no game — `shell_test.go` drives it
    with scripted key events — and the host wiring stays in one file where it can be read. It is
    also what makes 1.7b's settings screen and 1.7c's boards cheap to test.
  - **A house that will not load is a message, not an exit.** `PeekFile` decides what the picker
    lists and `LoadFile` decides what plays, and the two can disagree — a truncated house sniffs as
    one. The shell shows the loader's own sentence on the status band and stays up, because the
    player's next move is to choose a different house (`docs/IMPROVEMENTS.md` 2.33). Tested
    end-to-end against a real file, not a stub.
  - **Keyboard only, and the arrows navigate.** `internal/platform` has no pointer, so the shell is
    an arcade cabinet's: a list, a cursor, and a letter for every item. The original's arcade build
    wires the arrows straight to commands (`Events.c:191-207`); this has an on-screen menu to move a
    cursor through, so the arrows move it and Return chooses, with N/2/L/A/Q kept so that one
    keypress still starts a game. Recorded as a deliberate departure in the package comment.
- **1.7a, what drawing it found.** All four came out of rendering the screens with `-shot` and
  looking at the PNGs, which is why that flag exists:
  - **A 50% dim is not a panel** on this surface. `render.Surface` is 8-bit indexed with no alpha,
    so the only way to darken artwork is the original's `PenPat(gray)` + `PenMode(patOr)`
    checkerboard — and over the shipped splash art, which is a bright yellow wall, cream text on
    half-black-half-yellow is very nearly illegible. Panels are solid black with a cream frame and
    keep the dim only as an eight-pixel halo, which is what the game's own scoreboard does
    (`render/scoreboard.go` blackens its band and writes cream in it).
  - **The fallback title screen has to be laid out around the menu**, not centred on the screen.
    Centring on 320 put "no artwork found" half underneath the menu panel — on the one screen a
    fresh clone sees first.
  - **A selection bar on a disabled item is a lie.** The cursor starts on "New Game", which cannot
    work with no houses, and a filled inverse-video bar under it read as "press this". An
    unavailable item under the cursor gets an outline instead.
  - **The About box's key list was the original's, and this port's bindings are not.** Player two is
    on A/D/W/S rather than the original's modifier keys (2.3), Tab pauses, Escape ends the game.
    The box is the only place a player can find that out, so it lists what this build actually does;
    when 1.7b makes the bindings configurable the list has to be generated from them. *(Done: the
    box reads `Shell.host.Prefs` and falls back to `prefs.Default()`, so it is right for whoever is
    holding the keyboard — and 1.7b then took Escape out of the give-up business altogether, so the
    line it lists now reads "Q while paused gives up the game".)*
- **1.7b, what it settled.** Eight decisions, and the first two are the whole of the pause:
  - **The pause is split in two, and the seam is one hook.** `World.Pause func(paint func())`:
    `internal/game` decides what a paused frame looks like and holds `Paused` up while it is, and the
    host decides how long. The C cannot make that split — `DoPause` spins on `GetKeys` inside the
    frame loop (`Input.c:76-121`) — and without it the game package would need a keyboard, which
    would put a window in every game test. **A nil hook is no pause at all**, which is what the
    fidelity corpus needs: a replay has no keyboard, so a faithful `DoPause` would hang for ever on
    the first recorded frame with the pause key down.
  - **The C's three release loops became one rule stated twice.** `KeyPoll` reports the pause key
    only on its *press edge* (`pauseHeld`, the one piece of state it keeps), and the wait loop tracks
    `held` so the press that raised the pause cannot immediately end it. That is the same behaviour as
    the C's three `while (BitTst(...))` spins with no spinning. **One visible deviation:** the C
    erases the placard and *then* waits for the release, this port waits and then erases, so the
    placard stays up until the key comes back up. Recorded in `docs/IMPROVEMENTS.md` 2.5 — the C's
    order means a slow release shows a running game with no placard on it.
  - **Q gives up a paused game, because Command-Q is not a key this port can see.** The original's
    only way out of a pause is `DoCommandKey`'s Command-Q, and on X11 and on Windows the Command
    equivalent is the window manager's. So `PauseHint` is a row of the port's own text under the
    placard naming the key, and the restore rect is the union of the placard and that row. The row is
    not decoration: both 1994 placards say "or Cmd-Q to Quit the game", which on this port is simply
    false, and a paused game whose only way out is a key the player cannot guess is a hang.
  - **Escape pauses instead of ending the game, which is the confirmation 2.7 asked for.** 1.7a had
    Escape discard a game in one keystroke with no prompt — worse than the original, where giving up
    meant going to a menu. Now Escape pauses whether or not it is *the* pause key, either key
    resumes, and Q from the pause is the give-up. The prompt costs nothing to build because the pause
    already draws a placard and a hint row, and it turns the key a stranger reaches for from a
    destructive one into the one that shows them the way out. `prefs.PauseKey` therefore keeps its
    1994 meaning exactly — it picks the placard (`isEscPauseKey`) — and the host no longer needs to
    ask which key is spoken for.
  - **Three layers of settings, and a measurement reads none of them.** Defaults, then the file, then
    the flags that were *actually given* (`flag.Visit`, not a comparison against the default, because
    `-volume 7` means "make it 7"). `hermetic()` makes `-shot`, `-frames`, `-bench` and `-dump` start
    from `prefs.Default()` and refuse to save, so `make check` produces the same bytes on a machine
    whose owner has been playing the game — with no special pleading at any Makefile call site. An
    explicit `-prefs <file>` beats even a measurement, which is how 1.8's golden image of the settings
    screen will get settings to show; `-house` is deliberately *not* hermetic, because naming a house
    is asking to play and a player who plays that way wants their own bindings.
  - **Saving happens at the change, not at quit.** Two places only: the settings screen closing and
    `runShell` noticing a different house. The original saves once, in `WriteOutPrefs` at quit
    (`Main.c:381`), which loses every setting a player changed if the game crashes; saving at the
    change costs one small file write and cannot lose one.
  - **The settings screen offers less than the file holds, on purpose.** Three preferences are in the
    JSON and not on the screen — `pause_when_unfocused` (2.21 argues a released build should always
    pause and not offer the choice, and it *is* honoured), `music_on_title` (the title screen is silent
    until 1.7d, and a switch that does nothing is worse than no switch), and `keep_real_time` with the
    three `fixes` (they change what the simulation does; a player has no way to judge them and a
    developer has the file) — each with a paragraph in `internal/shell/settings.go` saying so. And
    `sound` is absent for a different reason: `Validate` derives it from `volume` both ways, so no file
    and no flag can make the two disagree, and `-volume 0` mutes exactly as the C's
    `isSoundOn = (isVolume != 0)` does.
  - **The fidelity switches are a `game.Fixes` copied field by field from `prefs.Fixes`.**
    `internal/game` does not import `internal/prefs` — the game has no preferences, it has a caller
    that had some — and the copy is written out field by field so that adding a fix to either side is
    a compile error here rather than a setting that silently does nothing. All three default **off**,
    which is to say the original's behaviour, because 1.8's corpus is recorded against the original
    and a correction that was on by default would be a corpus that measures this port against itself.
- **1.7b, what it found.** Three, and the first changed the assets API:
  - **A house can override the pause placard, and nineteen of the twenty do not.** Teddy World ships
    its own PICT 1015 and 1016, and on a Mac the open house's resource fork sits *in front of* the
    application's for those ids. `Assets.UI` is the wrong rule for that (application only, so that a
    house cannot repaint the title screen and hide the way out of it) and `Assets.Pict` is the wrong
    rule too (a missing file there is a recorded fault, and a placard has a drawn fallback). Hence a
    third accessor, `Assets.Plate`: the fork first, then the application, and silence when neither
    has it.
  - **1.7d's banner and stars panels need `Plate` as well.** Counted on disk: **thirteen** of the
    twenty shipped houses carry their own 1991-1993 and **four** their own 1017 or 1018. Every one of
    those is drawn over a running game, so `BringUpBanner` and `DisplayStarsRemaining` go through
    `Plate` and not `UI` — a fact worth having before 1.7d rather than after.
  - **`keep_real_time` is declared and deliberately unwired.** The limiter is still the original's
    no-catch-up form (`awaitFrame`), because a catch-up worth having needs a resync clamp: without
    one, the first long stall — a room wipe, a pause, a house load — is followed by a burst of
    unpaced frames, which is worse than the dropped time it was meant to recover. That is frame-pacing
    work and it is charged to 1.8, with the reason recorded on the field itself.

- **1.7c, what it settled.** Six decisions, and the first is the one the whole stage turns on:
  - **The shipped houses are read and never written.** A high score is 292 bytes inside a 98 KB
    house file, and the original writes the whole file back to record one (`gameDirty`, then a
    full `WriteHouse`). This port writes a **side-car** instead: `$XDG_DATA_HOME/glidergo/<house>.scores`,
    a board and nothing else. Three reasons, in order of weight — the 22 shipped houses are not
    ours to rewrite (`docs/IMPROVEMENTS.md` 1.2 says the release ships code only and extracts on
    first run, so the house files are the player's own copy); a whole-file rewrite to persist 292
    bytes is one power cut away from a destroyed house; and a read-only asset tree is what lets
    `make assets` be re-runnable. `Store.Load` merges the side-car over the house's own board, so a
    score set in 1995 is still what a new player has to beat.
  - **`internal/scores` owns the board; the host owns the blocking.** The package draws, sorts,
    qualifies, encodes and lays out both dialogs, and never sleeps, polls or writes a file it was
    not handed. The three things only a machine with a window and a clock can do — `ModalDialog`'s
    loop, `Delay(8)`'s button flash, and `DelayTicks(60)` + `WaitForInputEvent(30)` — are
    `cmd/glidergo/highscore.go`, on the same seam as the five other host hooks. That is why the
    board and both dialogs are pixel-testable without a display.
  - **The board reaches the screen, which it did not in 1994.** `DoHighScores` composes the whole
    thing into the offscreen work map and returns without blitting it: both of its `DissBits` calls
    are commented out and `RedrawSplashScreen`'s copy runs in the wrong direction
    (`docs/analysis/scoring.md` 7.8). The analysis's instruction to a porter is explicit — treat the
    intent as `CopyRectWorkToMain` and say so — so the screen a player sees here is the one the
    original composed and threw away.
  - **The high-score screen is a screen, not a panel.** `Shell.Draw` skips the backdrop, the menu
    and the house label for it, because the board brings its own starfield, its own heading and its
    own way out. The picker's footer and the board therefore read the *same* merged board, pinned by
    a test that pixel-compares a footer drawn from a side-car against one drawn from a house file:
    two screens disagreeing about the same house's best score is the bug that arrangement prevents.
  - **The remembered name and banner are saved as soon as they are earned.** The original writes
    `highName` at quit (`Main.c:223-224`). A game killed or crashed after a high score should not
    also forget who set it, so `prefs.Save` runs inside the hook.
  - **The credits are data, not a string literal.** `internal/credits` embeds `credits.txt`, and
    that file is pinned against `GliderPRO/README.md` by its own tests: a person named on the screen
    who is not named in the README fails, and a house the README credits that the screen does not
    fails too. This is the item `docs/IMPROVEMENTS.md` 1.2 makes a legal obligation and 3.4 has been
    tracking as unfinished since 1.7a, and the reason it is data is that an obligation discharged by
    a literal inside a drawing function is one nobody ever diffs against its source.
- **1.7c, what it found.** Six, and two of them are visible on screen:
  - **The blue footer on the starfield is barely legible.** `DrawHighScores` writes "Hit a Key to
    Exit" in `QDBlue` (211) over a near-black star field, which is faithful and is also the least
    readable text in the port. Recorded as `docs/IMPROVEMENTS.md` 2.56 rather than fixed, because
    1.8's corpus is measured against the original.
  - **The stored timestamps were never drawn.** Every board carries a `TimeStamps[10]` written by
    `Insert` and read by nothing: the original's own screen shows name, score and rooms and drops
    the date. This port keeps writing them (the file format is the original's) and does not draw
    them either — 2.57 — because a date column is a layout change and 1.8 measures the layout.
  - **A hold that discards keys is not the original's hold.** `Delay(60)` does not drain the Mac
    event queue, so a key pressed during that second was still queued when `WaitForInputEvent(30)`
    looked and the board flashed past. This port discards during the hold, so the second is always
    a second. Stated in `showBoard`'s comment as a deliberate deviation.
  - **The character limit has to be enforced while typing.** The original lets a sixteenth
    character into the field and then throws it away in `PasStringCopyNum` at commit, so its live
    counter and the name it saves disagree. Refusing the keystroke is the deviation
    (`docs/analysis/scoring.md` 7.11.1), and it also makes the counter honest.
  - **Sampler is credited to nobody, and it is not from 1995.** Writing the credits meant reading
    all 22 house banners. Nineteen of them credit nobody; `Castle o' the Air` names John Calhoun in
    its own banner though the README does not; and `Sampler` — saved 2000-05-11, five years after
    every other house — says only "Welcome to Omid's Happy Home." Whoever Omid is, the source
    release does not say, which matters to 1.2's asset question and is now noted there. The boards
    corroborate the date and widen it: decoding all twenty non-empty ones recovers the authors'
    playtesting, eighteen of the twenty topped by a run between 1995-06-08 and 1996-10-17 with
    `Ozma` on thirteen of them. The only rows outside that run are Slumberland's top two, stamped
    the morning of 2000-05-11, and Sampler's two, stamped that evening 39 seconds after the save
    that created the house. Whoever was there in 2000 touched two houses, not one (2.57).
  - **The ineligible-score alert is unreachable, and should stay that way.** `ALRT 1046` exists to
    tell a player they did not qualify, and nothing in the shipped game calls it
    (`docs/analysis/scoring.md` 7.12). Silence is what a 1994 player got and it is also the right
    answer: a dialog to tell somebody they did not win is a dialog nobody wants.

**1.8 Fidelity pass**
- `internal/fidelity`: frame-diff harness, input-trace replays, a checked-in corpus of
  reference frames.
- **Demo replay is the harness, not a feature.** The original records input as `demoType`
  (`{long frame; char key; char padding}`, `GliderStructs.h`) — a keystroke stream keyed to frame
  numbers. Replaying one is a frame-exact determinism test, which is what Stage 3's race needs and
  what 1.5f's shifting RNG stream threatens. Build the replay for the test; the attract-mode screen
  that uses it is 1.7 shell work and optional.
- *Acceptance:* the "fidelity contract" list in `docs/ORIGINAL_GAME.md` is either satisfied
  or has an explicit, written exception; a recorded input trace replays to a byte-identical frame
  sequence twice in a row, and across a rebuild.

**1.9 Local two-player** — *added after the plan was audited; see "What this plan was missing" below*
- Two gliders in **one** room on one keyboard, which is what the original's two-player mode is.
  Most of it is already built: 1.4 ported the `*Two` escape variants, `twoPlayerGame`,
  `onePlayerLeft`, the `otherPlayerEscaped` handshake, `ForceKillGlider` on the Delete key, and the
  shared signed `batteryTotal`, `foilTotal`, `bandsTotal` and sound throttle. Nothing consumes any
  of it yet.
- Needs: a second resolved `Keys` per frame from one poll (1.4's `GetInput` already takes a
  resolved `Keys` per glider for exactly this reason), the second glider's key set in preferences,
  and the two-player branches of 1.5b's transit handlers exercised for real.
- *Acceptance:* two gliders play one house from one keyboard; the two race strictnesses are
  distinguishable in a test — a transit exit admits the second arrival unconditionally, while a
  wall, ceiling or floor exit **refuses** a glider whose partner left by a different route, so the
  first one out chooses for both; the shared inventory is provably one counter (one player's
  battery use is visible to the other); and Delete kills a straggler only when pressed by player 1.

**1.10 Saved games** *(may slip past Stage 1 — it is the least load-bearing item here)*
- Mid-game save and resume. Half-built already: `internal/house` has parsed `gameType` (offset 820)
  and `hasGame` (860) since 1.2, and Titanic.house carries a live one — room 104, score 4700, two
  gliders, which `make check` already reports.
- The reason it is listed **before** 1.5 finishes rather than after: the original saves a whole
  house as `savedRoom` records (292 bytes each) carrying **per-room object state**, which is exactly
  what 1.5d builds. Shaping 1.5's room state so it serialises into that form costs nothing now and
  is a painful retrofit later. That is the whole of the dependency; the UI can come whenever.
- *Acceptance:* a game saved mid-house and resumed restores room, score, lives, inventory, glider
  mode and every room's object state; the format round-trips against the original's 40-byte
  `gameType` so Titanic.house's shipped save can be resumed.

#### What this plan was missing, and what it got right

Audited against the C after 1.4 landed. Four real gaps, now 1.6's music bullet, 1.8's demo replay,
1.9 and 1.10:

1. **Local two-player was in no stage at all.** Stage 1 promised "behaving exactly like the
   original" and Stage 3 committed to *separate worlds, each machine simulating only its own
   glider* — so nothing in the plan ever produced the original's actual two-player mode, even
   though 1.4 had already ported its entire handshake. Now 1.9.
2. **Music** was absent; 1.6 named only effects. Now in 1.6.
3. **Saved games** existed only as bytes the codec round-trips. Now 1.10, placed early for a
   dependency reason rather than a UI one.
4. **Demo replay** was absent, and it is the determinism harness 1.8 and Stage 3 both want. Now
   in 1.8.

Two things the audit went looking for and found already correct, recorded so they are not
re-litigated: **`Transitions.c`** (the visual wipes) is inside 1.5b, and **high scores** were
already fully specified in 1.7 — including the `roomsVisited` field that Stage 3 reuses as the race
metric, and the judgement that the original's `'gliS'` side-car is unreachable, which is right:
`IsFileReadOnly` is `return false` with its real body commented out (`HouseIO.c:659-664`), so
`houseIsReadOnly` is never true and neither `WriteScoresToDisk` nor `ReadScoresFromDisk` can run.

One open item that is a decision, not a discovery: **gliderGo is its own git repo inside the
parent repository**, as instructed, so the parent repository's CI, Pages and release machinery cannot see it —
`git ls-files gliderGo` is empty, gliderGo appears in none of `release.sh`, its CI config, the
root `README.md`, `public-root/index.html` or `wiki/home.md`, and `public/index.html` and
`CHANGELOG.md` do not exist, against the 11-point checklist in the parent repository's `CLAUDE.md`. Either
is defensible; nothing downstream can be released until it is chosen.

`Map.c` was checked and is **editor-only**, so it belongs to Stage 5 and not Stage 1:
`OpenMapWindow` has one caller, `OpenCloseEditWindows` (`Menu.c:792-799`), gated on
`theMode == kEditMode && houseUnlocked`.

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

**This is the *networked* mode, and it is a different game from the original's two-player.** The
original puts two gliders in one room sharing one inventory (that is 1.9). Here each machine
simulates only its own glider in its own copy of the house and the network carries progress. Both
are wanted; neither substitutes for the other, and conflating them was the plan's biggest hole.
1.9 first is also the cheaper order: it exercises every two-player branch locally, so what remains
here is genuinely just transport.

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
make assets                  # assets/extracted/ is gitignored; regenerate it (57 s)
make houses                  # 1.2's evidence: 22 houses through both codecs, unchanged
git log --oneline            # every stage is a commit with a detailed message
sed -n '1,60p' docs/PLAN.md  # you are here
```

`bin/glidertool house dump <house>` is the fastest way to see what the 1994 data actually
contains; its output is annotated with the `docs/analysis/house-format.md` sections that
explain each field.

Then read `docs/ORIGINAL_GAME.md` for the game, and the relevant `docs/analysis/*.md`
for whatever subsystem you are about to touch. The original C is in `GliderPRO/` and is
read-only — remember the CR-only line endings (`tr '\r' '\n'`).
