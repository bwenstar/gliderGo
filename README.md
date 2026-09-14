# gliderGo

A Go port and remaster of **Glider PRO** — the 1994 Macintosh game by John Calhoun,
published by Casady & Greene, whose source was later released under the GPLv2.

(The source tree calls itself 1.0.4, but its resource fork says **1.1.2** and it is really a
later Carbon work-in-progress. `docs/ORIGINAL_GAME.md` §2 has the evidence.)

You are a paper glider. You ride the air from furnace vents, dodge the household hazards of
a very large house, and try to get further than you did last time.

> **Status: in development.** Stage 0 (foundations, environment, source archaeology),
> Stage 1.1 (asset extraction — the 1994 art, sound, houses and movies), Stage 1.2
> (house loading: all 22 original houses read, written and round-tripped byte-for-byte)
> Stage 1.3 (room rendering: all 4,070 rooms of all 22 houses compose in the
> original's draw order) and Stage 1.4 (player physics: the 24-mode glider state
> machine, integer integrator, input, hit box and room boundaries) are complete.
> Stage 1.5 (objects, collision and room transitions) is next — when the glider first
> meets the rooms it can already draw. It is the largest stage in the project, about
> 9,100 lines of C, and is specified in
> [docs/analysis/stage-15-spec.md](docs/analysis/stage-15-spec.md) and split into six
> sub-stages; **1.5a** (the world, the object graph and the hot-spot table) is in progress.
> See [docs/PLAN.md](docs/PLAN.md).

---

## Quick start

```bash
./scripts/bootstrap-dev-env.sh    # rootless: installs Go 1.23 into ~/.local/opt/go
. scripts/env.sh                  # PATH, GOROOT, GOPROXY=off
make check                        # fmt, vet, test, build, headless, cross-build (+ on-screen bench)
make assets                       # extract the 1994 art/sound/houses/movies (57 s)
make run                          # window at the original's 640x480
make run ARGS='-scale 2'          # 2x nearest-neighbour magnification
make houses                       # round-trip every original house through the codec
```

Run these from this directory — the parent directory above has no Makefile, so `make check`
there fails with `No rule to make target 'check'`. `make check` needs neither a display nor
a network; `make run` and `make bench` need an X display, and `make check` runs the
on-screen bench only when `DISPLAY` is set.

Reading the 1994 level data:

```bash
make glidertool
bin/glidertool house info assets/extracted/houses/*.house   # 22 houses, 4,070 rooms
bin/glidertool house rooms "assets/extracted/houses/Demo House.house"
bin/glidertool house dump  "assets/extracted/houses/Demo House.house" | less
bin/glidertool types                                        # the 117 object types
```

`house dump` prints a house as annotated text; `house build` turns that text back into a
1994-compatible binary, byte-for-byte with `-residue`. That is how new houses will be
authored and how houses are diffed in git.

Looking at the 1994 pixels:

```bash
make assets                                                 # needed once, for the art (57 s)
bin/glidertool render -scale 2 -o /tmp/room.png "assets/extracted/houses/Demo House.house"
bin/glidertool render -all -o /tmp/demo "assets/extracted/houses/Demo House.house"
```

`render` composes a room the way the game does — nine local rooms, the original's object draw
order, the original's two port bugs — and writes it out as a PNG. It is how the renderer was
checked by eye, and `-all` over a whole house is the quickest way to notice that something has
gone missing.

The build needs no network access at run time and no third-party Go modules —
see [why](docs/DEV_ENVIRONMENT.md#3-the-package-mirror-exactly-what-this-network-can-and-cannot-reach).

## What is in here

| Path | What it is |
|---|---|
| `GliderPRO/` | The original 1994 C source, vendored **read-only** as the reference. Includes `Glider PRO.r` (the whole resource fork: 538 resources, every sprite and sound) and `Houses/` (the 22 shipped levels, 4,070 rooms). |
| `docs/ORIGINAL_GAME.md` | Consolidated source of truth for how the original behaves. **Read this first.** |
| `docs/analysis/` | 29 per-subsystem, byte-level specs reverse-documented from the C (130k lines). The detailed authority. |
| `docs/PLAN.md` | The staged implementation plan and the decisions behind it. |
| `docs/DEV_ENVIRONMENT.md` | How to build here, what this airgapped network can reach, and the measured performance baseline. |
| `internal/platform/` | The port layer: a 640×480 software framebuffer, backends for X11 (cgo/Xlib) and headless (PNG/WAV). |
| `internal/house/` | The house model and its two codecs: the 1994 binary format (byte-exact both ways) and a line-oriented text format meant to be written by hand and read in a diff. |
| `internal/render/` | The room composition: an 8-bit indexed surface with the game's own 256-colour palette, the sprite atlas, and `DrawLocale`'s draw order object for object. Indexed rather than RGBA because the original's shadows OR palette *indices* together. |
| `cmd/glidertool/` | `house dump` / `build` / `check` / `info` / `rooms`, `render` (compose a room to PNG) and the `types` reference table. |
| `tools/` | Python asset extractors, standard library only: BinHex, Rez, QuickDraw PICT → PNG, `'snd '` → PCM, QuickTime → index buffers. `extract_all.py` is the driver (`make assets`); the `probe_*.py` scripts are inspection CLIs for the same formats. |
| `assets/extracted/` | **Generated, gitignored.** 1,899 files of 1994 art, sound, house forks and movies, reproducible from `GliderPRO/` in 57 s. Deleting it costs nothing; `make assets-check` proves the extraction is deterministic. |

## Planned scope

1. **Stage 1** — the game, behaving exactly like the original, on Linux.
2. **Stage 2** — new houses in the spirit of the originals, selectable alongside the original set.
3. **Stage 3** — 2-player over the network: one machine hosts, another joins, furthest on one life wins.
4. **Stage 4+** — Windows (pure-Go backend, no cgo), a house editor, then macOS and possibly mobile.

## Working on the original source

`GliderPRO/Sources/*.c` and `GliderPRO/Headers/*.h` are classic Mac text files with
**CR-only line endings**, so `wc -l` reports `0` and most tools see one enormous line:

```bash
tr '\r' '\n' < "GliderPRO/Sources/Player.c" > /tmp/Player.c
```

`GliderPRO/Glider PRO.r` is the exception and uses LF. The upstream git history is preserved
at `GliderPRO/upstream.git` (`git --git-dir=GliderPRO/upstream.git log`).

## Licence

The original Glider PRO source is GPLv2-or-later (`GliderPRO/GPLv2-LICENSE.md`). This port
is a derivative work and is distributed under the same terms.

Original game and art by John Calhoun. House credits — Jonathan Chin, Ward Hartenstein,
Steve Sullivan, Shawn Brenneman, Kim Money — are listed in `GliderPRO/README.md`.
