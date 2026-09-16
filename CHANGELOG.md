# Changelog

All notable changes to gliderGo, newest first.

There are no releases and no tags yet, so everything below is under `Unreleased`. The version a
build reports is `git describe --tags --always --dirty`, which with no tags resolves to a bare
short hash — see `docs/IMPROVEMENTS.md` 5.4, which owns tagging and the release pipeline.

Because Stage 1's whole goal was *"behave exactly the same as the original, only newer"*, this
file records stages rather than features, each naming the commit that closed it. Where the port
knowingly departs from 1994 the entry says so and points at the numbered item in
`docs/IMPROVEMENTS.md` that owns the deviation. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) loosely; it does not follow semantic
versioning yet, because nothing has been versioned.

## Unreleased

### The game's data is committed (2026-09-16)

- `assets/extracted/` is now in the repository: 1,877 files, 15.5 MB — the 1994 art, the sounds
  and music decoded to PCM, and all 22 houses. `make run` plays from a clone with no extraction
  step, no python3, and no copy of Glider PRO. This is route (a) of `docs/IMPROVEMENTS.md` 1.2,
  settled by the project owner; the GPLv2 source the content was decoded from stays vendored under
  `GliderPRO/` as the reference the documentation cites and the extractor reads.
- The one thing still not committed is `houses/*.rsrc`: 22 per-house resource forks, 24 MB of the
  39.5 the extractor writes, which `tools/extract_house_art.py` decodes into the `houseart/` PNGs
  and which nothing reads at run time. `make assets-check` now diffs the whole committed tree
  against a fresh extraction rather than comparing manifests, excluding those; the extractor is
  byte-for-byte reproducible, which is what makes that comparison exact.
- CI refuses a checkout whose assets are missing instead of extracting them, and a new `assets`
  job runs `make assets-check` so the committed tree cannot drift from its source.
- `GliderPRO/` is now genuinely optional rather than nominally optional: delete it from a clone
  and `make check` is still green, because the three tests that pinned the credits against its
  README skip when it is gone instead of failing. Verified by deleting it, hiding python3 behind a
  shim that errors, and running the whole target.

### Stage 1.10 — saved games (`9c08d5b`, 2026-09-16)

- Reading and writing the original's saved-game format, reconstructed from four code paths that
  were dead in the 1994 sources. Both shipped saves load: Titanic (room 104, 4,700 points, 4
  gliders) and ImagineHouse PRO II (room 45, 5,900, 5).
- Four decisions the original left ambiguous are recorded as S1–S4 in
  `internal/house/savedgame.go`'s package comment rather than in a commit message, because they
  are load-bearing for anyone reading the codec.

### Public build path (`4517a37`, 2026-09-16)

- `scripts/bootstrap-dev-env.sh` resolves the toolchain and the C libraries from `system`,
  `internal` or `public` and prints which it chose, so the project builds on a machine with the
  open internet as well as on the airgapped host it was written on. The internal mirror is opt-in
  via `GLIDERGO_MIRROR_HOST`; no hostname and no credential is committed.
- `.github/workflows/ci.yml`: `make check` with assets and a display under Xvfb, `make cross`,
  and `go build`/`vet`/`test` natively on Windows and macOS. It has never run — the machine it was
  written on cannot reach GitHub — and says so at the top.

### Stage 1.9 — local two-player race (`db5ce1c`, 2026-09-16)

- Two gliders, one keyboard, separate worlds, furthest-on-one-life wins. This is the local half of
  the networked race that Stage 3 owns.
- Three race strictnesses, because the plan's single "same house, same seed" rule turned out not
  to be enough to make two runs comparable.

### Stage 1.8 — fidelity, pinned to pixels (`a4868db`, `42b7e56`, `b7dfeed`)

- `internal/fidelity`: reference PNGs and a hash corpus, the first pixels checked into the
  repository. A rendering change now needs `go test ./internal/fidelity -update` and a moved hash
  in the diff.
- The 1994 attract-mode demo replays through the port's own physics.
- The original's RNG verified against the C, and the fidelity contract audited in writing —
  which is what found the last two contract items nothing had ever tested.

### Stage 1.7 — the shell around the game (`20b1180`, `8ea05c3`, `8e5c5f1`, `26323bf`, `19dbb42`)

- Title screen, house picker, settings, a pause that can be got out of, both in-game banners,
  both endings, the credits, and music on the title screen.
- **High scores**, which the original did keep (per house, in the house file itself) — both entry
  dialogs included.
- A clone whose assets are missing comes up on the port's own title screen and names the command
  that puts them back, instead of failing (`docs/IMPROVEMENTS.md` 2.6). Assets ship in the
  repository now, so that path is the one you reach by deleting them rather than the one a clone
  starts in.
- Deviation: the scoreboard sits where the port puts it, not where 1994 put it, and the reason is
  recorded (2.9).

### Stage 1.6 — audio (`e7427a0`)

- Three channels with the 1994 priority policy, decoding Mac Sound Manager `'snd '` resources to
  PCM at extraction time.
- `glidertool replay -wav` and a mix digest, so a bug report can carry what the game sounded like
  and not just what it looked like.

### Stage 1.5 — the world (`09c38fa` … `516f479`)

The largest stage, specified in full before any of it was written (`09c38fa`), then built in six
parts: the 117-type object graph (`db53c28`), the frame loop and room traversal that made the game
playable (`d557701`, `2c21169`), the scoreboard and a hand-authored bitmap font with full Mac
Roman coverage (`474a26c`, `b392fc3`, `da8595f`), dynamics — appliances, movers, toggles, triggers
(`9b5dfeb`), rewards and switches and the per-room state byte that outlives its room (`35d94b1`),
rubber bands and grease (`408c8bf`), and the animated locale — flames, stars, pendulums, shreds
(`516f479`).

- `badIndex` gives one named path to every out-of-range read the C performs and the port refuses,
  so a refusal is reported rather than silently papered over.
- `internal/replay` and `glidertool replay`: the bug-report format (4.2).
- Fixed a launch-time "hang" that was a lost window and a dropped expose event (`5363c78`).

### Stage 1.4 — the glider (`6e137df`)

- The player as a 24-mode integer state machine, transcribed from `Player.c`.

### Stage 1.3 — a static room, composed the 1994 way (`84c5266`)

- Backgrounds, tiles and object art assembled in the original's order.
- `make check` made to work without a display (`638c41f`), which is what the null backend is for.

### Stage 1.2 — the house format (`82c3f79`)

- `internal/house`: the 1994 binary codec, byte for byte, plus a text codec so new houses can be
  authored by hand. `cmd/glidertool` dumps, builds, checks and lists rooms.

### Stage 1.1 — the asset pipeline (`3520b93`)

- `make assets` (`tools/extract_all.py`): 38 committed files under `GliderPRO/` in, 1,899 files
  and 46 MB out, in about 70 s. No Macintosh, no copy of the retail game and no network
  (`docs/IMPROVEMENTS.md` 5.6). The output was gitignored at this point; it is committed now, see
  *The game's data is committed* above.

### Stage 0 — the source of truth (`3ed2c74`, `b472ca4`, `b988c2f`)

- The original 1994 Glider PRO source vendored unmodified under `GliderPRO/` as the reference
  (`b988c2f`), and reverse-documented into `docs/ORIGINAL_GAME.md` and `docs/analysis/*.md`
  before any Go was written (`3ed2c74`).
- The rootless development environment, with the platform layer proven end to end at 533 fps
  (`b472ca4`).
- GPLv2 `LICENSE` and `docs/IMPROVEMENTS.md`, which tracks everything a public release needs that
  fidelity does not (`0bc050f`).

### Known not-yet-done

The full list lives in `docs/IMPROVEMENTS.md`. The four that would matter most to someone reading
this file first:

- **The 1994 art, sounds and houses ship under the GPLv2 the source release carries** (1.2), which
  is a defensible reading of that release and not a cleared one — nobody has asked John Calhoun.
  Shipping the content is decided; confirming it is not.
- No release pipeline: no tags, no packaged builds, no installer (5.4).
- Only Linux/X11 can draw. Windows is Stage 4, macOS Stage 6; everything else compiles and runs
  headless (5.5).
- An installed copy still looks for its assets beside the binary (5.3).
