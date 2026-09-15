# gliderGo

A Go port and remaster of **Glider PRO** — the 1994 Macintosh game by John Calhoun,
published by Casady & Greene, whose source was later released under the GPLv2.

(The source tree calls itself 1.0.4, but its resource fork says **1.1.2** and it is really a
later Carbon work-in-progress. `docs/ORIGINAL_GAME.md` §2 has the evidence.)

You are a paper glider. You ride the air from furnace vents, dodge the household hazards of
a very large house, and try to get further than you did last time.

> **Status: in development — the game is playable, and it now has a title screen.**
> Complete: Stage 0 (foundations, environment, source archaeology), **1.1** asset
> extraction (the 1994 art, sound, houses and movies), **1.2** house loading (all 22
> original houses read, written and round-tripped byte-for-byte), **1.3** room rendering
> (all 4,070 rooms compose in the original's draw order), **1.4** player physics (the
> 24-mode glider state machine, integer integrator, input, hit box and room boundaries),
> **1.5** objects, collision and room transitions — the largest stage in the project, about
> 9,100 lines of C, specified in
> [docs/analysis/stage-15-spec.md](docs/analysis/stage-15-spec.md) and delivered in six
> sub-stages (117 object types, 19,849 hot spots, bands, grease, switches, rewards and the
> six animated families) — and **1.6** audio (three effect channels with the original's
> priority policy, the music score on a fourth, and a replay that records what it sounded
> like), and **1.7a** the way in (the splash screen, the menu, the house picker and the
> About box — `glidergo` with no arguments is now a game you start rather than a house you
> name).
>
> In progress: the rest of **1.7, the shell** — **1.7b** preferences and a real pause,
> **1.7c** high scores, **1.7d** the in-game overlays and game over. Until those land the
> settings are command-line flags, Tab pauses without drawing anything, and a score is
> printed rather than recorded. See [docs/PLAN.md](docs/PLAN.md), and
> [docs/IMPROVEMENTS.md](docs/IMPROVEMENTS.md) for what still stands between this and a
> release someone else could play.

---

## Quick start

```bash
./scripts/bootstrap-dev-env.sh    # rootless: installs Go 1.23 into ~/.local/opt/go
. scripts/env.sh                  # PATH, GOROOT, GOPROXY=off
make check                        # fmt, vet, test, build, headless, audio, cross-build (+ bench)
make assets                       # extract the 1994 art/sound/houses/movies (57 s)
make run                          # window at the original's 640x480
make run ARGS='-scale 2'          # 2x nearest-neighbour magnification
make houses                       # round-trip every original house through the codec
make audio                        # replay 600 frames to /tmp/glidergo-audio.wav
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

Hearing the 1994 sounds:

```bash
make run ARGS='-audio list'                                 # which players this machine has
make run ARGS='-volume 3'                                   # 0..7, the original's range
make run ARGS='-sound=false'                                # the original's dontLoadSounds
bin/glidertool replay -house "CD Demo House" -frames 600 -wav /tmp/run.wav
```

There is no audio driver in here and there is not meant to be: the mix is written to
`pw-play`, `paplay`, `aplay`, `ffplay` or `play`, whichever the machine has, as raw PCM on
stdin. That keeps the port free of cgo and of every dependency this airgapped network cannot
reach, at the cost of the player's own buffer latency — and it is the one part of the audio
path that Windows will need replaced
([docs/IMPROVEMENTS.md](docs/IMPROVEMENTS.md) 2.48). With no player installed, or on a build
host with no sound card at all, `-wav` writes the samples to a file instead; the developer
machine this was written on has no sound card, so that path is the one under test.

The build needs no network access at run time and no third-party Go modules —
see [why](docs/DEV_ENVIRONMENT.md#3-the-package-mirror-exactly-what-this-network-can-and-cannot-reach).

## Controls

The title screen is keyboard-only, because the game is: `internal/platform` reports no
pointer, and the only mouse in Glider PRO was in its editor.

| Title screen | |
|---|---|
| `↑` `↓` | move the cursor |
| `Return` | choose |
| `N` / `2` | one-player / two-player game |
| `L` | load a house — `↑` `↓` move, `←` `→` page, a letter jumps, `Return` plays, `Space` selects |
| `A` | about |
| `Q` or `Esc` | quit |

| In a game | Player one | Player two |
|---|---|---|
| steer | `←` `→` | `A` `D` |
| throw a rubber band | `↑` | `W` |
| use the battery | `↓` | `S` |
| give up a waiting glider | `Delete` | — |
| pause | `Tab` | — |
| end the game | `Esc` | — |

Three of those are the port's own and not the original's. Player two was on Control, Command,
Option and Shift (`InterfaceInit.c:148-151`), which a modern window manager takes before the
game sees it; the arrows on the title screen were wired straight to menu commands in the
original's arcade build, which had no on-screen menu to move a cursor through; and `Esc` ends
a game rather than quitting the program, so it hands the title screen back. All three become
the player's choice in 1.7b — the bindings are already per-glider data for that reason
([docs/IMPROVEMENTS.md](docs/IMPROVEMENTS.md) 2.3).

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

**GPLv2** — see [LICENSE](LICENSE).

`GliderPRO/README.md` states the grant exactly: *"The source for Glider PRO is released
under the GNU General Public License 2 as published by the Free Software Foundation."*
There is no "or (at your option) any later version" clause, so this is GPLv2-**only**, not
GPLv2-or-later. gliderGo is transcribed from that source function by function and is
unambiguously a derivative work, so it carries the same licence.

Original game by **John Calhoun**, published by Casady & Greene. Upstream source:
[softdorothy/glider_pro](https://github.com/softdorothy/glider_pro).

**The assets are not the source, and the distinction matters for a release.** Upstream's
grant covers the source. The 22 shipped houses are credited to five other authors —
Jonathan Chin, Ward Hartenstein, Steve Sullivan, Shawn Brenneman and Kim Money — and two
PICT resources derive from illustrations by John R. Neill (*Ozma of Oz*) and Winsor McCay
(*Little Nemo*). So gliderGo distributes **no original art**: `assets/extracted/` is
gitignored and is regenerated locally from your own copy of the game. See
[docs/IMPROVEMENTS.md](docs/IMPROVEMENTS.md) §1.2 for what that means for a public build.
