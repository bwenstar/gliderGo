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

### Only the original's data is vendored (2026-09-17)

- John Calhoun's 1994 C is no longer redistributed here. `GliderPRO/Sources/` (67 files),
  `Headers/` (25), `Prefix.h`, the two CodeWarrior project files and `CarbonLib` are gone — 96
  files, 1.8 MB. What stays is the 41 files the extractors actually read: `Glider PRO.r`, the 37
  files in `Houses/`, and upstream's own `README.md` and licence. The split is not arbitrary;
  `assets/extracted/manifest.json` records the extractor's 38 inputs and not one of them is a
  `.c` or `.h`, which is what made removing the code safe to do without touching the pipeline.
- Consequently `make assets`, `make assets-check` and the CI `assets` job all still work, and all
  six `internal/credits` tests still pass rather than skipping, because the document they check
  `credits.txt` against is upstream's `README.md` and that is one of the four files kept.
  `make check` passes with no caveats, and the game is unaffected: the frames, the audio mix and
  the replay digests are byte-identical to before.
- The cost is the citations. `docs/` names a path under `Sources/` or `Headers/` about 9,500 times
  — 7,939 and 1,624 — because each one is the receipt for a transcription decision, and they now
  point outside the repository. Rewriting them was not the answer; pinning them was.
  `docs/ORIGINAL_GAME.md` and `README.md` name the exact upstream commit,
  `94fed96e0b4c810a6ac861e5d4b14d625a5a1c31`, and give the three lines that put its `Sources/` and
  `Headers/` back at the paths the docs already use. Both directories are gitignored, so doing
  that leaves the working tree clean; the tree that was read was verified byte-for-byte identical
  to that commit's, all 137 files, before anything was deleted.
- `CarbonLib` is worth its own line: 234 KB of Apple's PowerPC system library, the one file in the
  vendored tree its author had no standing to release under the GPLv2 in the first place, and read
  by nothing here.
- `make assets-check` no longer discards the extractor's stderr. It previously sent both streams
  to `/dev/null`, so a failure produced one line of make noise naming no cause — which is exactly
  what a CI operator would have seen.
- `tools/extract_house_art.py` refuses to run instead of writing an empty result. On a clean
  checkout the `houses/*.rsrc` intermediates it reads are absent, and it would truncate the
  committed 165 KB `houseart/manifest.json` to 185 bytes and exit 0 — a silent loss that looked
  like success. Pre-existing and unrelated to the removal; found while testing it.

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
- `README.md` rewritten for a stranger arriving from a search rather than for whoever wrote it:
  it now opens on Glider PRO and John Calhoun before it opens on Go, carries three screenshots of
  the port's own output, and says what it is up to in three lines instead of a forty-line
  paragraph. Half the length, every practical fact kept, and the per-sentence justification of
  design decisions moved to where it belongs — `docs/IMPROVEMENTS.md` and the packages' own
  comments.
- The tip was clean of the private network the port was written on, but the 35 commits before the
  scrub were not: an audit of every historical tree — rather than just the working tree, which is
  what the earlier sweeps checked — found the internal mirror's hostname in three files and the
  authoring machine's absolute path in 21 documents. Both are gone from all 46 commits now. The
  rewrite is content-only and provably narrow: the tip's tree hash is unchanged, and every
  commit's author, committer, dates and subject are identical before and after, so a reader of the
  history sees the same project with `$HOME/.local/opt`, `the repository root` and an RFC-2606
  placeholder host where a private name used to be. The 35 short hashes this file and
  `docs/PLAN.md` cite were repointed again.
- `LICENSE` is the canonical FSF text of GPLv2 rather than a markdown-reflowed copy of it. The old
  one was missing `END OF TERMS AND CONDITIONS` and the whole "How to Apply These Terms" appendix,
  which is enough for GitHub's licence detection to give up and show no licence at all on a
  repository whose entire premise is that it inherits one.
- Verified the way a stranger would meet it: cloned the repository into a scratch directory,
  removed the remote, and ran `make check` under `env -i` with nothing but `HOME`, `PATH` and a
  Go — green, with the summary correctly listing the on-screen blit as not exercised because
  `DISPLAY` was unset. `make bench` with a display then drew 300 frames at 770 fps.

### The game's data is committed (2026-09-16)

- `assets/extracted/` is now in the repository: 1,877 files, 15.5 MB — the 1994 art, the sounds
  and music decoded to PCM, and all 22 houses. `make run` plays from a clone with no extraction
  step, no python3, and no copy of Glider PRO. This is route (a) of `docs/IMPROVEMENTS.md` 1.2;
  the GPLv2 source the content was decoded from stays vendored under `GliderPRO/` as the reference
  the documentation cites and the extractor reads.
- The one thing still not committed is `houses/*.rsrc`: 22 per-house resource forks, 24 MB of the
  39.5 the extractor writes, which `tools/extract_house_art.py` decodes into the `houseart/` PNGs
  and which nothing reads at run time. `make assets-check` now diffs the whole committed tree
  against a fresh extraction rather than comparing manifests, excluding those; the extractor is
  byte-for-byte reproducible, which is what makes that comparison exact.
- CI refuses a checkout whose assets are missing instead of extracting them, and a new `assets`
  job runs `make assets-check` so the committed tree cannot drift from its source.
- `GliderPRO/` is now optional *for `make check`*: delete it from a clone and that target is
  still green, because the three tests that pinned the credits against its README skip when it is
  gone instead of failing. Verified by deleting it, hiding python3 behind a shim that errors, and
  running the whole target. That verification did not cover `make assets` or `make assets-check`,
  which do need it and which `make check` never invokes — so "optional" was too strong a word for
  this entry to have used, and the `assets` job introduced two bullets above would in fact have
  failed. See *Only the original's data is vendored* below, which resolves it.
- Found by that same clone run: the external audio player's diagnostics arrived anonymous, so an
  unreachable sound server printed a bare `error: pw_context_connect() failed` between two of the
  game's own lines. They are now tagged with the player's name. The player being installed still
  does not mean it works, and falling back to the next one when it does not is recorded as
  `docs/IMPROVEMENTS.md` 2.71.

### Stage 1.10 — saved games (`ed3effa`, 2026-09-16)

- Reading and writing the original's saved-game format, reconstructed from four code paths that
  were dead in the 1994 sources. Both shipped saves load: Titanic (room 104, 4,700 points, 4
  gliders) and ImagineHouse PRO II (room 45, 5,900, 5).
- Four decisions the original left ambiguous are recorded as S1–S4 in
  `internal/house/savedgame.go`'s package comment rather than in a commit message, because they
  are load-bearing for anyone reading the codec.

### Public build path (`d693b3c`, 2026-09-16)

- `scripts/bootstrap-dev-env.sh` resolves the toolchain and the C libraries from `system`, `local`
  or `public` and prints which it chose, so the project builds on a machine with the open internet
  as well as on the airgapped host it was written on. A private mirror is opt-in — the machine
  describes its own in a gitignored `scripts/local-source.sh` — so no hostname and no credential
  is committed.
- `.github/workflows/ci.yml`: `make check` with assets and a display under Xvfb, `make cross`,
  and `go build`/`vet`/`test` natively on Windows and macOS. It has never run — the machine it was
  written on cannot reach GitHub — and says so at the top.

### Stage 1.9 — local two-player race (`b84bb02`, 2026-09-16)

- Two gliders, one keyboard, separate worlds, furthest-on-one-life wins. This is the local half of
  the networked race that Stage 3 owns.
- Three race strictnesses, because the plan's single "same house, same seed" rule turned out not
  to be enough to make two runs comparable.

### Stage 1.8 — fidelity, pinned to pixels (`721082e`, `7036c62`, `3ba40ac`)

- `internal/fidelity`: reference PNGs and a hash corpus, the first pixels checked into the
  repository. A rendering change now needs `go test ./internal/fidelity -update` and a moved hash
  in the diff.
- The 1994 attract-mode demo replays through the port's own physics.
- The original's RNG verified against the C, and the fidelity contract audited in writing —
  which is what found the last two contract items nothing had ever tested.

### Stage 1.7 — the shell around the game (`1b87fe7`, `14607bb`, `2441bbd`, `67c160e`, `dc2fc58`)

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

### Stage 1.6 — audio (`1ab983a`)

- Three channels with the 1994 priority policy, decoding Mac Sound Manager `'snd '` resources to
  PCM at extraction time.
- `glidertool replay -wav` and a mix digest, so a bug report can carry what the game sounded like
  and not just what it looked like.

### Stage 1.5 — the world (`e1a10e7` … `8dd1b84`)

The largest stage, specified in full before any of it was written (`e1a10e7`), then built in six
parts: the 117-type object graph (`ad14e63`), the frame loop and room traversal that made the game
playable (`b907cdd`, `44b01fa`), the scoreboard and a hand-authored bitmap font with full Mac
Roman coverage (`d5d66aa`, `5ec920f`, `470494b`), dynamics — appliances, movers, toggles, triggers
(`b75fa6e`), rewards and switches and the per-room state byte that outlives its room (`664e9d1`),
rubber bands and grease (`fc0be40`), and the animated locale — flames, stars, pendulums, shreds
(`8dd1b84`).

- `badIndex` gives one named path to every out-of-range read the C performs and the port refuses,
  so a refusal is reported rather than silently papered over.
- `internal/replay` and `glidertool replay`: the bug-report format (4.2).
- Fixed a launch-time "hang" that was a lost window and a dropped expose event (`33761d9`).

### Stage 1.4 — the glider (`ec50449`)

- The player as a 24-mode integer state machine, transcribed from `Player.c`.

### Stage 1.3 — a static room, composed the 1994 way (`42b6a31`)

- Backgrounds, tiles and object art assembled in the original's order.
- `make check` made to work without a display (`6d675ce`), which is what the null backend is for.

### Stage 1.2 — the house format (`e6f67fe`)

- `internal/house`: the 1994 binary codec, byte for byte, plus a text codec so new houses can be
  authored by hand. `cmd/glidertool` dumps, builds, checks and lists rooms.

### Stage 1.1 — the asset pipeline (`7f1d794`)

- `make assets` (`tools/extract_all.py`): 38 committed files under `GliderPRO/` in, 1,899 files
  and 46 MB out, in about 70 s. No Macintosh, no copy of the retail game and no network
  (`docs/IMPROVEMENTS.md` 5.6). The output was gitignored at this point; it is committed now, see
  *The game's data is committed* above.

### Stage 0 — the source of truth (`6a26b06`, `ca6ce0b`, `f1451b9`)

- The original 1994 Glider PRO source vendored unmodified under `GliderPRO/` as the reference
  (`f1451b9`), and reverse-documented into `docs/ORIGINAL_GAME.md` and `docs/analysis/*.md`
  before any Go was written (`6a26b06`).
- The rootless development environment, with the platform layer proven end to end at 533 fps
  (`ca6ce0b`).
- GPLv2 `LICENSE` and `docs/IMPROVEMENTS.md`, which tracks everything a public release needs that
  fidelity does not (`4e4063d`).

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
