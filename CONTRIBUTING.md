# Contributing to gliderGo

Thanks for looking. This is a port, which makes contributing here a little different from
contributing to a normal Go project: most of the time the question is not "what should this code
do" but "what did the 1994 code do", and that question has an answer you can go and read.

There is one maintainer. Patches are welcome and reviews may be slow — if something has been
sitting for a while, a nudge on the thread is fine.

## Start here

```bash
git clone https://github.com/bwenstar/gliderGo
cd gliderGo
make          # builds bin/glidergo and bin/glidertool
make run      # plays
make check    # everything CI does
```

No dependencies to fetch — the module has none, and the game's art, sounds and every house it ships
are committed and embedded. On Linux you do need `libx11-dev` (or your distribution's name for it): the
X11 backend asks `pkg-config` for it, so without it `make` stops with

```
# [pkg-config --cflags  -- x11]
Package x11 was not found in the pkg-config search path.
```

which is the whole error. `make doctor` checks for it (and for python3, a C compiler, a display and
a sound device) before you build anything, and installs nothing. If you cannot install it,
`CGO_ENABLED=0 make` gives you a binary that builds, tests and renders screenshots with no window
at all — enough to work on everything except the window itself.

[docs/DEV_ENVIRONMENT.md](docs/DEV_ENVIRONMENT.md) has the long version, including the offline path
for a machine with no internet.

## `make check` is the gate

```
embedded fmt-check vet test race build glidertool houses levels headless audio fidelity docs-check cross smoke
```

It compiles for Windows as well as this machine, round-trips every shipped house through both
codecs, rebuilds the houses in `levels/` from their text and lints them, plays a headless session,
renders and hashes seven screens, mixes audio to a WAV, and runs the command lines this file gives.
`race` is the odd one out in the other direction: it runs the race detector over the three places
that start a goroutine and nothing else — `internal/netplay`, `internal/audio` (the pipe to the
sound player) and the loopback race tests in `cmd/glidergo`. `test` deliberately runs without
`-race` — the detector needs cgo and a C compiler and costs an order of magnitude — so if you add
concurrency anywhere else, add it to that target in the same patch.
Under a minute from a cold Go build cache on an eight-core machine and about a dozen seconds after
that, and it is the whole contract — if it passes, CI will too.

`docs-check` is the odd one out: it runs the command lines this file and the README tell you to
run, because those are claims too and for six stages nothing checked them. So if your patch adds a
line to a fenced `bash` block in either document, `go test ./tools/docscheck` will fail until you
say whether it can run unattended — the failure prints the two spellings and the file to put one
in. A block fenced without the `bash` tag is read as sample output and left alone, which is what
the tag is for.

`make check` finishes by printing what it could *not* check on this machine (no `DISPLAY`, cgo off,
no extracted asset tree — and something different on Windows and macOS, where a different backend is
at stake). Read that list; a green run with three caveats is not the same as a green run.

`make fuzz` is not part of the gate. It runs every `func Fuzz` target under Go's fuzzing engine for
30 s each, then plays 3,000 damaged houses, in about six minutes. Run it when your patch touches
something that reads bytes from somebody else: a house or its text, a saved game, a score board, a
replay script, a picture, a house's sounds, or the race's wire. A target that fails writes its
input to `testdata/fuzz/<target>/` in its package. Commit that file with the fix, and plain
`go test` replays it from then on.

## Four rules that a patch can break without anything obvious going wrong

Everything else is ordinary Go and review will sort it out. These four are worth knowing in
advance, because each has a test that will fail in a way that is confusing if you have not been
told why the rule exists.

**1. The standard library only.** `go.mod` has no `require` block and it is not going to get one.
The reason is not minimalism: the port was written on an airgapped machine and being buildable
there is a property worth keeping, because it is the same property as "buildable in ten years".
`internal/module`'s test parses every `.go` file in the tree and fails on any import that is
neither stdlib nor `github.com/bwenstar/gliderGo/...`. That includes test files and tools. If you
need a dependency, open an issue first and expect the answer to be "write the twenty lines".

**2. Pixels are hashed, so intentional changes need a hash in the diff.** `internal/fidelity`
holds SHA-256 hashes of seven rendered screens. If your change moves a pixel, `make check` fails
with a line telling you which screen. Look at it first:

```bash
bin/glidergo -shot /tmp/screen.png -shot-screen about   # splash houses settings race about credits scores
```

If the change was intended, refresh the corpus and commit the new hash *in the same commit as the
change that caused it*:

```bash
go test ./internal/fidelity -update
```

A hash landing on its own, or a screen changing without a hash, is the one thing a pixel corpus
cannot survive. The same applies to `internal/replay`'s determinism traces.

**3. Behaviour that differs from the original needs a citation.** The port's claim is that it
behaves like the 1994 build, and the docs back that with roughly 17,800 line-level citations into
upstream's C — every one of which `go test ./internal/citations/` resolves against the pinned tree,
so a citation that names a line the original does not have fails the build. If you change what the
game *does* — physics, collision, scoring, a transition —
say in the commit message which C function you read and where. `docs/ORIGINAL_GAME.md` and the 29
specs in `docs/analysis/` are where that knowledge lives; if you learn something new about the
original, that is where it goes.

If you find a place where the original was *wrong* and the port copied it, do not fix it silently.
Several such bugs are reproduced on purpose and documented as such. File it, or fix it behind a
setting.

**4. A deliberate divergence, or a known gap, goes in `docs/IMPROVEMENTS.md`.** That file is the
project's running ledger: every improvement noticed, actioned or deferred, with the reasoning and
the citation. It is numbered by stage and it is how the next person (or the next session) knows
what was already considered. Adding an entry is cheap; finding out later that something was
decided and forgotten is not.

## Working on the original source

The 1994 C is *not* in this repository, by choice — but every citation in `docs/` is relative to
one pinned commit, so you can put it where the docs say it is:

```bash
git clone https://github.com/softdorothy/glider_pro /tmp/glider_pro
git -C /tmp/glider_pro checkout 94fed96e0b4c810a6ac861e5d4b14d625a5a1c31
cp -r /tmp/glider_pro/Sources /tmp/glider_pro/Headers /tmp/glider_pro/Prefix.h GliderPRO/
```

All three are gitignored, so your tree stays clean, and nothing in the build needs them.

One trap: those files are classic Mac text with **CR-only line endings**, so `wc -l` says `0` and
most tools see one enormous line. The line numbers in the docs are into the LF-normalised copy:

```bash
tr '\r' '\n' < "GliderPRO/Sources/Player.c" > /tmp/Player.c
```

## Houses

`glidertool` is the authoring path, and it is worth ten minutes before you touch a house:

```bash
bin/glidertool house dump "assets/extracted/houses/Slumberland.house" > slumberland.txt
bin/glidertool house build -o out.house slumberland.txt
bin/glidertool house lint out.house
bin/glidertool house checks          # what every lint finding means
bin/glidertool house stats -tier small out.house   # how it compares to the 22 originals
```

(`GliderPRO/Houses/` holds the same houses as upstream ships them — BinHex-encoded, with their
resource forks attached. `assets/extracted/` is that tree decoded, and it is what the tools and the
game read.)

The text format is documented by the header `house dump` writes. A new house should lint clean —
over the 22 shipped houses the linter reports 637 notes, 48 warnings and exactly one error, and
that calibration is deliberate: a warning means "an author probably did not mean this", so new work
has no excuse for one.

`house stats` is the same idea for the numbers rather than the defects: eighteen measurements —
rooms, objects and prizes per room, the share of the house that is dark, how deep the room graph
goes — taken exactly as §10.2 of `docs/analysis/original-houses.md` took them over the 22 originals,
so a house of yours can be read against the column it belongs in. The walk is `internal/profile`,
which both of the port's own houses are tested against, and the command prints what those tests
assert; if you are adding a house, run it before you ask anyone to fly the thing. What it is not is a
gate. The bands are what 1994 happened to do and several of them are two houses wide, so a miss is
something to explain rather than something to fix — Open House misses one and the argument for
leaving it alone is in its own test.

Adding a lint check means a method on `*linter` in `internal/house/lint.go`, a row in
`LintChecks()`, and a test in `internal/house/lint_test.go`. An AST test holds the catalogue to the
code, so a check whose ID is not in the table (or a table row with no check) fails the build. Then
run it over all 22 originals — if it fires on them, either the severity is wrong or the check is.

### A house that ships

`levels/` is where the port's own houses are authored, one `*.house.txt` per house, and the path
from text to a player's screen is three steps and one commit:

```bash
make levels         # builds every levels/*.house.txt into assets/levels/ and lints each one
make levels-zip     # packs that directory into assets/levels.zip, which is embedded
bin/glidergo -levels assets/levels    # play what you just built, instead of the built-in copy
```

`assets/levels.zip` is committed and `assets/levels/` is not, which is the reverse of the 1994
assets and worth knowing before you fight it: `make levels` runs `bin/glidertool`, which is built
from a package that imports `assets`, which embeds the archive — so a clone with no archive cannot
build the tool that packs it. If you delete it, `git checkout -- assets/levels.zip` is the only way
back. `make levels` refuses if the archive has gone stale, and `go test ./assets` rebuilds every
authored house and compares the bytes, so a house committed without `make levels-zip` fails there
rather than shipping as its previous build.

### A house that carries its own pictures

A room's `background` below 3000 names one of the eighteen pictures the application carries. At 3000
and above it names a picture the *house* carries, and a house authored here carries one by putting a
PNG under `levels/houseart/`:

```bash
mkdir -p "levels/houseart/My House/pict"
cp my-background.png "levels/houseart/My House/pict/3000.png"
make levels && make levels-zip
```

One PNG per `PICT` id, `<id>.png`, room-background size (640×460 as the extracted ones are; a
too-narrow picture is a lint warning, because the tile columns read off the right edge of it). The
directory name is the house's name, which is its file name without `.house`. `make levels` copies the
tree into `assets/levels/` beside the built houses and `make levels-zip` packs it into the archive
every executable embeds — so a downloaded binary has the pictures the same way it has the house, with
no flag and no files beside it. `bnds/<id>.bin` is the other half of a fork if you need it: it says
which sides of a background are room openings, and it is read for any room whose `background` is
3000 or more.

This is a *search*, not a substitution: `-houseart DIR` is looked at first, then the levels tree,
then the 1994 forks under `assets/extracted/houseart/`. So a house of yours can have pictures without
taking the shipped houses' pictures away, which is the thing that did not work before
(`docs/IMPROVEMENTS.md` 4.15). If you point `-houseart` at a tree that does not have the house you are
playing, you are told — that flag exists for testing an extraction, and a silent fall-through would
make a half-extracted tree look complete.

**Sounds do not work this way yet.** A `kSoundTrigger` can name a sound the application already has
and nothing else; custom `snd ` needs a second manifest in the sound root and is filed as
`docs/IMPROVEMENTS.md` 4.29.

**One field is a promise, not a value.** A house's `timestamp` is the key every saved game of that
house is checked against (`internal/saved/store.go:401-407`), so changing it on a house that has
shipped refuses every save any player has made in it — and nothing else notices, because the house
still lints, still finishes and still plays. It is pinned by a test with the published number in it,
and the house's file *name* is the same kind of promise for both saves and high scores. Pick both
before you ship a house; treat them as fixed afterwards. `docs/IMPROVEMENTS.md` 4.19 has the
details.

### Whose house it is

A house in `levels/` is compiled into every executable, so it goes to everyone who downloads one,
under this repository's licence. Three rules follow from that.

- **It is your own work.** You designed its rooms. The tools above work on the 22 originals so that
  you can learn the format from them, and a house you build by reworking a `house dump` of one of
  them counts as your own (`docs/IMPROVEMENTS.md` 4.43). Say in the pull request which one it
  started from. A house somebody else made, reworked or not, is the third rule's.
- **It gets a credits line.** Add a row under `[this port]` in `internal/credits/credits.txt`: your
  name, a `|`, and the house's name as its file is named. `go test ./internal/credits` fails for a
  house in `levels/` with no row. The credits screen is one the fidelity corpus hashes, so run
  `go test ./internal/fidelity -update` and commit the new hash with the row.
- **Never somebody else's house.** The project does not bundle third-party houses, including ones
  that were free to download in the 1990s. Their authors granted this project nothing, and the 22
  originals already raise the licence question in `docs/IMPROVEMENTS.md` 1.2. To play one, point
  the game at it: `glidergo -house path/to/it.house`, or `-levels DIR` to list a directory of them
  as the New set.

## Reporting a bug

Use the templates; they ask for the two things that make a report actionable.

The first is `glidergo -version`, which prints the build, the backend it was compiled with and
where its assets came from. If the game crashed, `crash-last.log` in its data directory already
holds that block and the crash under it, and the start after the crash says where the file is.
The second thing, if you can get it, is a `glidertool replay` script: the
input format exists precisely so a bug can travel as a reproducible input rather than a
description. `bin/glidertool replay -h` shows the shape, and a scripted session runs headlessly and
deterministically from a fixed seed, which means a report carrying one reproduces on the
maintainer's machine exactly. The script's `house` line takes a name — any house the picker lists,
including the ones this port wrote, which have no file to attach because they live inside the
executable (`docs/IMPROVEMENTS.md` 4.20) — or a path, for a house you have and nobody else does.

"This doesn't match the original" is a distinct and very welcome kind of report — there is a
template for it. Say which room, which house, and what the original did; a screenshot of both is
ideal.

## Commits

Write the subject as a sentence about what changes, in the imperative, ≤72 characters, optionally
with a package prefix:

```
house: add a linter, calibrated so the 1994 houses pass it
render, audio: make the pixel and audio arithmetic architecture-independent
Pin Slumberland's inescapable basement as the original's design
```

Then a body that says *why*, wrapped at 80. The history in this repository leans long, because in
a port the interesting content is usually the reasoning — which C function, what it does, what was
kept and what was dropped. Citations belong here. `git log` is the closest thing this project has
to a design document, and it is meant to be read.

One commit per idea. If a change moves pixels, the new hash is part of that commit.

## Releases

A `v*` tag builds and publishes a release through `.github/workflows/release.yml`.
[RELEASING.md](RELEASING.md) is the list of what a tag needs that the workflow cannot do, and the
record of each tag's checks. [docs/RELEASE_TESTING.md](docs/RELEASE_TESTING.md) is the part of
that a person has to do by hand, such as playing a house through or racing a Windows host, and
anyone with the machine for a step can run it on an rc and report what they saw.

## Licence

By contributing you agree your work ships under the **GPLv2** — see [LICENSE](LICENSE). Version 2
*only*: upstream's grant names version 2 with no "or any later version" clause, and this port is a
function-by-function transcription of it, so the licence is not ours to widen. There is no CLA.
A house is covered the same way, which is why one you contribute has to be yours to give
("Whose house it is", above).
