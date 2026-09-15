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
> like), **1.7a** the way in (the splash screen, the menu, the house picker and the
> About box — `glidergo` with no arguments is now a game you start rather than a house you
> name), **1.7b** preferences and a real pause (a settings screen, all eight key bindings
> rebindable, a native config file, and Tab or Escape pausing with the original's placards),
> and **1.7c** high scores (both entry dialogs, the board on screen, and a per-house
> side-car so a new score is recorded without rewriting a 1994 house file) with the credits
> screen the port owed its contributors.
>
> In progress: the last of **1.7, the shell** — **1.7d** the in-game overlays and game over:
> the house's own banner, the stars-remaining panel, the win and loss animations, and the
> music on the title screen. See [docs/PLAN.md](docs/PLAN.md), and
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
| `H` | the selected house's high scores |
| `S` | settings |
| `A` | about — and `C` from there for the credits |
| `Q` or `Esc` | quit |

| In a game | Player one | Player two |
|---|---|---|
| steer | `←` `→` | `A` `D` |
| throw a rubber band | `↑` | `W` |
| use the battery | `↓` | `S` |
| pause | `Tab` or `Esc` | |
| give up a paused game | `Q` | |
| give up a glider waiting in limbo | `Delete` | |

All eight of the movement keys are the player's — the eight rows above are only the defaults
this build ships. Four of them had to change: player two was on Control, Command, Option and
Shift (`InterfaceInit.c:148-151`), which a modern window manager takes before the game sees
it. The bindings were per-glider data from 1.1 for that reason, and the settings screen is
where they are edited ([docs/IMPROVEMENTS.md](docs/IMPROVEMENTS.md) 2.3).

The bottom three rows are the port's own and are **not** bindable, because each stands in for
something a 1994 Macintosh had and this does not:

- **`Tab` pauses**, as the original does (`isEscPauseKey` is false at `Main.c:184`), and the
  settings screen offers `Esc` instead — but `Esc` pauses either way. It used to end the game
  outright, one keystroke and no prompt, which is worse than the original, where giving up
  meant going to a menu. Now the key a stranger reaches for costs them nothing, and the way
  out is written on the screen they land on (2.7).
- **`Q` gives up a paused game** and hands the title screen back. The original's Command-Q is
  a chord the window manager owns on every platform this builds for; both pause placards
  still read "or Cmd-Q to Quit the game", so the port prints its own line under them.
- **`Delete`** abandons a glider waiting in limbo, which is the original's own key for it.

Closing the window ends the program, from anywhere.

### Settings

`S` from the title screen: the eight bindings, which key pauses, how much of the house is
composed around the player, the window magnification, the volume, and whether the score
plays. `←` `→` change a value, `Return` rebinds one key, `R` resets everything to this
build's defaults, `Esc` saves and goes back. A rebind is checked by the same `prefs.Validate`
that repairs a hand-edited file, so a collision is reported on the status line rather than
silently kept.

Settings live in one JSON file in the platform's config directory
(`$XDG_CONFIG_HOME/glidergo/prefs.json`, or `~/.config/glidergo/prefs.json`; `GLIDERGO_CONFIG`
overrides it), and are saved when the screen closes rather than at quit, so a crash cannot
lose them. The original's five-pane Options dialog had nineteen more fields that were about a
Macintosh rather than about the game — screen-depth switching, colour-table fades, the
editor's window positions — and `internal/prefs/legacy.go` lists every one with the reason it
was dropped.

```bash
make run ARGS='-prefs /tmp/test.json'   # use this file instead of the config directory's
make run ARGS='-prefs none'             # this build's defaults; reads and writes nothing
bin/glidergo -import-prefs "/path/to/Glider Prefs"   # convert a 1994 226-byte prefs file
```

`-import-prefs` refuses to overwrite an existing settings file: it runs before any window
opens, so there is nobody to ask. The four measurement modes — `-shot`, `-frames`, `-bench`
and `-dump` — ignore the config directory for the same reason `make check` has to give the
same answer on every machine (2.53); pass `-prefs <file>` to override even them.

### High scores

Every house carries a board of ten: a name, a score, how many rooms were visited, and a date
nothing has ever drawn. Twenty of the 22 shipped houses arrive with theirs filled in, and what
is on them is the authors' own playtesting — most of it between June and September 1995, `Ozma`
on top of thirteen boards, `Paul` on top of the two of them Jonathan Chin signs as Paul Finn, and the best
run in the box 108 rooms and 47,000 points through ImagineHouse PRO II on 1995-07-03. `H`
on the title screen shows the selected house's board, which is one more way in than the original
had: there, the only route to that screen was to earn a place on it.

A score that beats the tenth row asks for a name; first place also gets to change the banner
across the bottom of the board. **The board is sorted on rooms visited before points**, which is
the original's order and is worth knowing before you optimise for score.

New scores go in a file of their own, one per house, in the platform's data directory
(`$XDG_DATA_HOME/glidergo/<house>.scores`, or `~/.local/share/glidergo/`). The house files
themselves are never written to: they are your own copy of a 1994 game, a board is 292 bytes
inside as much as 185 KB, and a whole-file rewrite to save 292 bytes is one power cut from a
destroyed house.

```bash
make run ARGS='-scores /tmp/boards'   # keep the boards here instead
make run ARGS='-scores none'          # play and record nothing
```

A hand-edited or truncated board is repaired rather than refused: you get a playable board and
one line on stderr per thing that had to be worked around.

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
