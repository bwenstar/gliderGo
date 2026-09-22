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
embedded fmt-check vet test build glidertool houses levels headless audio fidelity cross smoke
```

It compiles for Windows as well as this machine, round-trips every shipped house through both
codecs, rebuilds the houses in `levels/` from their text and lints them, plays a headless session,
renders and hashes six screens, and mixes audio to a WAV. It takes a couple of minutes and it is
the whole contract — if it passes, CI will too.

`make check` finishes by printing what it could *not* check on this machine (no `DISPLAY`, cgo off,
no extracted asset tree). Read that list; a green run with three caveats is not the same as a green
run.

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
holds SHA-256 hashes of six rendered screens. If your change moves a pixel, `make check` fails
with a line telling you which screen. Look at it first:

```bash
bin/glidergo -shot /tmp/screen.png -shot-screen about   # splash houses settings about credits scores
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
```

(`GliderPRO/Houses/` holds the same houses as upstream ships them — BinHex-encoded, with their
resource forks attached. `assets/extracted/` is that tree decoded, and it is what the tools and the
game read.)

The text format is documented by the header `house dump` writes. A new house should lint clean —
over the 22 shipped houses the linter reports 634 notes, 48 warnings and exactly one error, and
that calibration is deliberate: a warning means "an author probably did not mean this", so new work
has no excuse for one.

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

**One field is a promise, not a value.** A house's `timestamp` is the key every saved game of that
house is checked against (`internal/saved/store.go:401-407`), so changing it on a house that has
shipped refuses every save any player has made in it — and nothing else notices, because the house
still lints, still finishes and still plays. It is pinned by a test with the published number in it,
and the house's file *name* is the same kind of promise for both saves and high scores. Pick both
before you ship a house; treat them as fixed afterwards. `docs/IMPROVEMENTS.md` 4.19 has the
details.

## Reporting a bug

Use the templates; they ask for the two things that make a report actionable.

The first is `glidergo -version`, which prints the build, the backend it was compiled with and
where its assets came from. The second, if you can get it, is a `glidertool replay` script: the
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

## Licence

By contributing you agree your work ships under the **GPLv2** — see [LICENSE](LICENSE). Version 2
*only*: upstream's grant names version 2 with no "or any later version" clause, and this port is a
function-by-function transcription of it, so the licence is not ours to widen. There is no CLA.
