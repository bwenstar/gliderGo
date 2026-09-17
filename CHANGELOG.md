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

### Ready to publish (2026-09-17)

- The module path is `github.com/bwenstar/gliderGo`, which is where the repository will live —
  `go.mod` plus 248 import lines across 113 files. This closes the one item in
  `docs/IMPROVEMENTS.md` 5.8 that could not be guessed, because it needed the account first.
- Nothing in the tree names the private network the port was written on. `scripts/bootstrap-dev-env.sh`'s
  third source is now `local`: whatever an optional, gitignored `scripts/local-source.sh` defines,
  skipped silently when that file is absent, which it is in a clone. A machine that cannot reach
  `go.dev` has three ways out of it and none of them is an edit to a tracked file —
  `GLIDERGO_GO_TARBALL` for a tarball carried across by hand, `GLIDERGO_GO_DL_HOST` /
  `GLIDERGO_UBUNTU_MIRROR` for a mirror that speaks the same protocols, or that hook for a source
  that needs real logic. No hostname, no credential, and no reference to any of it in the docs.
- Every commit's author and committer email is a personal address rather than the work one the
  machine was configured with, and four commit *messages* that named the mirror by product name
  no longer do. Neither rewrite touched anything else: trees, names, dates and subjects are
  identical, verified by diffing every commit before and after, so only the hashes moved — and
  the 40 hashes this file and `docs/PLAN.md` cite moved with them and were repointed.
- The Makefile reads `GLIDERGO_TOOLCHAIN_DIR` instead of hard-coding `~/.local/opt`, so it agrees
  with the bootstrap script about where a toolchain was installed. Set that variable for the
  bootstrap and `make` would previously have built with a different Go than the one it had just
  installed.
- Verified the way a stranger would meet it: cloned the repository into a scratch directory,
  removed the remote, and ran `make check` under `env -i` with nothing but `HOME`, `PATH` and a
  Go — green, with the summary correctly listing the on-screen blit as not exercised because
  `DISPLAY` was unset. `make bench` with a display then drew 300 frames at 770 fps.

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
- Found by that same clone run: the external audio player's diagnostics arrived anonymous, so an
  unreachable sound server printed a bare `error: pw_context_connect() failed` between two of the
  game's own lines. They are now tagged with the player's name. The player being installed still
  does not mean it works, and falling back to the next one when it does not is recorded as
  `docs/IMPROVEMENTS.md` 2.71.

### Stage 1.10 — saved games (`09ae243`, 2026-09-16)

- Reading and writing the original's saved-game format, reconstructed from four code paths that
  were dead in the 1994 sources. Both shipped saves load: Titanic (room 104, 4,700 points, 4
  gliders) and ImagineHouse PRO II (room 45, 5,900, 5).
- Four decisions the original left ambiguous are recorded as S1–S4 in
  `internal/house/savedgame.go`'s package comment rather than in a commit message, because they
  are load-bearing for anyone reading the codec.

### Public build path (`b382f8f`, 2026-09-16)

- `scripts/bootstrap-dev-env.sh` resolves the toolchain and the C libraries from `system`, `local`
  or `public` and prints which it chose, so the project builds on a machine with the open internet
  as well as on the airgapped host it was written on. A private mirror is opt-in — the machine
  describes its own in a gitignored `scripts/local-source.sh` — so no hostname and no credential
  is committed.
- `.github/workflows/ci.yml`: `make check` with assets and a display under Xvfb, `make cross`,
  and `go build`/`vet`/`test` natively on Windows and macOS. It has never run — the machine it was
  written on cannot reach GitHub — and says so at the top.

### Stage 1.9 — local two-player race (`2ba7171`, 2026-09-16)

- Two gliders, one keyboard, separate worlds, furthest-on-one-life wins. This is the local half of
  the networked race that Stage 3 owns.
- Three race strictnesses, because the plan's single "same house, same seed" rule turned out not
  to be enough to make two runs comparable.

### Stage 1.8 — fidelity, pinned to pixels (`7954779`, `fe521f2`, `fe79eef`)

- `internal/fidelity`: reference PNGs and a hash corpus, the first pixels checked into the
  repository. A rendering change now needs `go test ./internal/fidelity -update` and a moved hash
  in the diff.
- The 1994 attract-mode demo replays through the port's own physics.
- The original's RNG verified against the C, and the fidelity contract audited in writing —
  which is what found the last two contract items nothing had ever tested.

### Stage 1.7 — the shell around the game (`4dd24b1`, `81881d3`, `b76a784`, `29661ec`, `352bcfd`)

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

### Stage 1.6 — audio (`fad3f4c`)

- Three channels with the 1994 priority policy, decoding Mac Sound Manager `'snd '` resources to
  PCM at extraction time.
- `glidertool replay -wav` and a mix digest, so a bug report can carry what the game sounded like
  and not just what it looked like.

### Stage 1.5 — the world (`624a5f2` … `cd79d47`)

The largest stage, specified in full before any of it was written (`624a5f2`), then built in six
parts: the 117-type object graph (`974557f`), the frame loop and room traversal that made the game
playable (`5cf41b0`, `d0cef99`), the scoreboard and a hand-authored bitmap font with full Mac
Roman coverage (`d93fff3`, `7f19302`, `da41e0b`), dynamics — appliances, movers, toggles, triggers
(`859ba43`), rewards and switches and the per-room state byte that outlives its room (`6131d18`),
rubber bands and grease (`289b042`), and the animated locale — flames, stars, pendulums, shreds
(`cd79d47`).

- `badIndex` gives one named path to every out-of-range read the C performs and the port refuses,
  so a refusal is reported rather than silently papered over.
- `internal/replay` and `glidertool replay`: the bug-report format (4.2).
- Fixed a launch-time "hang" that was a lost window and a dropped expose event (`5251c7c`).

### Stage 1.4 — the glider (`95369c8`)

- The player as a 24-mode integer state machine, transcribed from `Player.c`.

### Stage 1.3 — a static room, composed the 1994 way (`90cf69c`)

- Backgrounds, tiles and object art assembled in the original's order.
- `make check` made to work without a display (`bd7f0e8`), which is what the null backend is for.

### Stage 1.2 — the house format (`b2b06d3`)

- `internal/house`: the 1994 binary codec, byte for byte, plus a text codec so new houses can be
  authored by hand. `cmd/glidertool` dumps, builds, checks and lists rooms.

### Stage 1.1 — the asset pipeline (`5d51bb9`)

- `make assets` (`tools/extract_all.py`): 38 committed files under `GliderPRO/` in, 1,899 files
  and 46 MB out, in about 70 s. No Macintosh, no copy of the retail game and no network
  (`docs/IMPROVEMENTS.md` 5.6). The output was gitignored at this point; it is committed now, see
  *The game's data is committed* above.

### Stage 0 — the source of truth (`d908f0a`, `c0c7641`, `3c67f83`)

- The original 1994 Glider PRO source vendored unmodified under `GliderPRO/` as the reference
  (`3c67f83`), and reverse-documented into `docs/ORIGINAL_GAME.md` and `docs/analysis/*.md`
  before any Go was written (`d908f0a`).
- The rootless development environment, with the platform layer proven end to end at 533 fps
  (`c0c7641`).
- GPLv2 `LICENSE` and `docs/IMPROVEMENTS.md`, which tracks everything a public release needs that
  fidelity does not (`698f115`).

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
