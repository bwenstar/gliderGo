# Improvements — what a public release needs that a faithful port does not

`docs/PLAN.md` is the fidelity plan: it tracks what has to be true for gliderGo to
*behave* like Glider PRO. This file tracks what has to be true for gliderGo to be a game
that strangers can download and play. The two lists are different, and nothing in the
stage plan would have surfaced most of what is below.

Rules for this file:

- Stage 1's contract is unchanged: **the game behaves exactly like the 1994 original**.
  So anything here that would change behaviour is either gated behind a setting that
  defaults to the original, or waits for Stage 2 or later. Fidelity is not traded away
  for polish.
- Every item names the stage it belongs to and is either **done**, **planned** (with the
  stage), or **decision needed** (the user's call, not mine).
- An item is only removed from this file when it is done, and then it moves to the
  Done section with the commit that did it.
- **The numbers are load-bearing.** Comments in the Go source cite items by number
  (`docs/IMPROVEMENTS.md 2.17`), because the argument for transcribing a bug faithfully
  is only complete if it says where the fix is written down. Renumber an item and those
  citations point at the wrong thing, so items are appended, never reordered.

---

## 1. Legal — the blocker that has nothing to do with code

### 1.1 gliderGo has no licence file — **DONE, 1.5a**

gliderGo is transcribed from GPLv2 source, function by function, with comments citing
`GliderPRO/Sources/*.c` line numbers. That makes it unambiguously a derivative work, and
GPLv2 §2(b) requires the whole to be licensed under GPLv2. This is not a preference; it
is the only compliant option, and shipping a public binary with no licence file at all is
strictly worse than shipping one. Added `LICENSE` (GPLv2) and a `## Licence` section in
`README.md` naming John Calhoun, Casady & Greene and the upstream repository.

Note for anyone tempted to relicense later: upstream's grant is GPLv2 **only**. There is
no "or later" clause, so gliderGo cannot be moved to GPLv3 and cannot take GPLv3-only
dependencies.

### 1.2 The art, the sounds and the 22 houses are on a different footing from the code — **settled: ship them. Route (a), decided 2026-09-16**

`GliderPRO/README.md` says, exactly: *"The **source** for Glider PRO is released under the
GNU General Public License 2."* It says source. It does not say assets. And the same file
credits the houses to five people who are not John Calhoun:

| House | Author |
|---|---|
| Demo House, CD Demo House | John Calhoun and **Kim Money** |
| Davis Station, Metropolis, Titanic | **Jonathan Chin** and John Calhoun |
| Grand Prix, Leviathan, ImagineHouse PRO II, In The Mirror | **Jonathan Chin** |
| Land of Illusion, Nemo's Market, Rainbow's End, SpacePods | **Ward Hartenstein** |
| Slumberland | John Calhoun, **Jonathan Chin**, **Steve Sullivan**, **Ward Hartenstein** |
| Teddy World | **Shawn Brenneman** |
| The Asylum Pro | **Steve Sullivan** |

Two PICT resources are also derived from other people's illustrations: 3975 from a John R.
Neill plate for *Ozma of Oz*, and 153 (the About box) from a Winsor McCay *Little Nemo*
strip. Both are old enough to be very probably public domain, but "very probably" is not a
licence audit.

A third borrowed work, found in Stage 2 while fixing the About box to draw PICT 153 at all:
the same plate sets `...we have lingered in the chambers of the sea... T.S. Eliot` along its
bottom edge, from the closing stanza of *The Love Song of J. Alfred Prufrock* (1915), elided
at both ends. Upstream's README credits the Nemo strip and not the verse, so nothing in
this tree recorded it until now — the words were in the picture data and picture data is
invisible to `grep`. The poem is public domain on any reading, so this changes nothing about
the obligation; it changes the count, which is the point of the item. Transcribed at
`docs/analysis/ui-dialogs.md` 10.6 and credited in `internal/credits/credits.txt`, whose
`[the illustrations]` section is now `[borrowed from elsewhere]` because a poet is not an
illustrator.

**Six of the twenty-two houses are credited to nobody, and one of them is not from 1995.**
Found while writing 1.7c's credits screen, by reading all 22 house banners: Art Museum,
California or Bust!, Castle o' the Air, Empty House, Fun House and Sampler appear in no line of
the README. `Castle o' the Air`'s own banner says "(by john calhoun)", which settles that one.
`Sampler` does not: its banner reads "Welcome to Omid's Happy Home." and its saved date is
**2000-05-11**, five years after every other shipped house and after Casady & Greene stopped
selling the game. Whoever Omid is, the source release does not say, and a house whose author is
unknown is the one kind this project cannot re-licence by asking. It is two rooms and eleven
objects, so losing it would cost nothing — but that is a decision for the same conversation as
the rest of this item.

The upstream repository does distribute the houses, so re-vendoring them under `GliderPRO/` is
no worse than what the copyright holder already does. Shipping the **decoded** art is a
further step: a new distribution of that art, in a new form, by someone who is not the rights
holder.

**That step has been taken deliberately.** The decision taken on 2026-09-16 is that gliderGo
works with no reference to the original game's source and that the assets are included, which is
route (a) below plus its natural conclusion: `assets/extracted/` is committed (15.5 MB,
1,877 files), so a clone plays with no extraction step, no python3 and no copy of Glider PRO.
The reasoning is upstream's own precedent — the copyright holder distributes the same bytes in
the same repository, in encoded form — and the practical fact that a game whose first
instruction is "now go and decode the assets" is not a game somebody can play.

What that decision does *not* do is answer the licence question, and it makes route (c) worth
more rather than less: an explicit grant from John Calhoun would move gliderGo from "no worse
than upstream" to "unambiguously licensed", and it is the only route that does. It is also the
one thing here that cannot be done from this machine.

**On 2026-09-17 the 1994 C source stopped being redistributed here.**
Sources/, Headers/, `Prefix.h`, the two CodeWarrior project files and `CarbonLib` were removed —
96 files. Note carefully that this *reduces* what the repository redistributes but does not
improve the position this item is about, and arguably narrows the argument above: what was
removed is the part upstream's grant covers beyond doubt, and what remains is the art, the
sounds and the 22 houses, which is the part it does not mention. The "no worse than what the
copyright holder already does" reasoning still holds for those, because the 41 files kept are
byte-for-byte upstream's, in upstream's layout. Route (c) remains open and remains the only
route that settles anything.

**This item used to claim the architecture was already right by accident. It was wrong, and
the correction matters more than the original claim did.** What it said was that `.gitignore`
excludes `/assets/extracted/`, so no original art is in this repository and none can reach a
release tarball built from it. The first half is true and irrelevant; the second is false.
`.gitignore` keeps *derived* files out of git. The originals are **tracked**:

- `GliderPRO/Glider PRO.r` — 15,475,666 bytes, the entire resource fork: every PICT, every
  `snd `, the `'demo'` stream, the lot.
- `GliderPRO/Houses/*.binhex` — all 22 shipped houses, 34 MB.
- `GliderPRO/Houses/*.mov` — the 15 QuickTime movies.

`git ls-files GliderPRO/` is 137 files and 50.7 MB of the 62.2 MB this repository tracks, and
`git archive HEAD` carries all of it. So
this repository already redistributes the art, and so does any archive, zip or GitHub release
tarball made from a clone, unless something is done to exclude it.

That is not a leak to be quietly plugged: it is exactly why a stranger can clone this
repository and play without owning Glider PRO (see 5.6). But it means this decision was live
*now* rather than at release time.

**The three routes, with the chosen one marked:**

- **(a) Keep vendoring — CHOSEN, and extended to the decoded assets.** Upstream distributes the
  same resource fork and the same houses, so re-vendoring them is no worse than what the
  copyright holder already does; release archives carry the assets and one download works. This
  is the honest description of what the repository is.
- **(b) Code only — rejected.** Add `.gitattributes` `export-ignore` for
  `GliderPRO/Glider PRO.r`, `GliderPRO/Houses/` and `assets/extracted/` so `git archive` and
  every release tarball drop them, then write the fetch step `make assets` would need — from
  upstream, or from the player's own copy of the game, the way a ScummVM or DOSBox front-end
  does. Rejected because it makes the first-run experience a scavenger hunt for a 31-year-old
  Macintosh game, and because the fetch step has to exist and needs somewhere to fetch *from*
  that is not this repository.
- **(c) Contact John Calhoun for an explicit asset grant — still open, and now the only
  outstanding part of this item.** It is the one route to content that is unambiguously
  licensed rather than merely no worse than upstream. Route (a) does not depend on it, and it
  would retire this item outright. It need not name the title screen's mark, which 1.5 made from
  his logo: the owner ruled on 2026-09-24 that the mark ships without asking.

One consequence stands whatever happens with (c), because it changes the priority of a later
stage: **Stage 2's new houses are the only content gliderGo can ship without asking anyone.**
They stop being a nice extra and become the content the project owns outright. Stage 2 should be
planned as original work on that basis — not as imitations of the shipped houses, which would
inherit the same authorship question.

If (c) is ever answered with a no, the fallback is (b) applied to release archives only, which
is a `.gitattributes` change and a fetch script rather than a redesign — worth knowing, so that
route (a) is a reversible decision rather than a one-way door.

### 1.3 No `CHANGELOG.md` and none of the conventional repository files — **changelog DONE, end of Stage 1; the rest DONE, 2.0, bar the code of conduct**

gliderGo is a standalone repository: one `Makefile`, one CI workflow
(`.github/workflows/ci.yml`), no parent project, no external release machinery, and — as of the
end of Stage 1 — no reference anywhere in the tree to the private repository it was developed
inside. An earlier version of this item measured gliderGo against that repository's own
integration checklist, which does not apply to it and never did; the checklist item is dead
rather than outstanding.

What a public repository is actually missing:

- `CHANGELOG.md` — **done**, at the end of Stage 1: one section per stage, each naming the
  commit that closed it. Since 2026-09-24 the stages sit in one section per tag, back to `v0.1.0`
  (PLAN §4's release policy, "After the gate").
- A project page. Deferred until there is a release to link to; 5.4 owns tagging, versioning
  and artefacts. **Done by 5.4's amendment:** there are releases now, and the GitHub Releases
  page is the project page. README's opening links it, and so do `-version` and `-help`
  (`project.Releases`). `VERSION` in the Makefile is `git describe --tags --always --dirty`, and a
  clone with no tags fetched resolves to a bare short hash, so a build from one is stamped that way.
- `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`, and issue and pull-request templates:
  none exist. Cheap and conventional, but all four are about *other people*, and they should be
  written when the repository is actually public and there is somebody to address — recorded
  here so that their absence is a decision. See 5.7. **All but the code of conduct written at
  2.0**, plus a third issue form for fidelity differences, which is the report this project most
  wants; 5.7 records what each one says and why the covenant is still the odd one out.

### 1.4 The archives ship Go's runtime and standard library without Go's licence — **DONE, the release gate's first step, as `THIRD-PARTY-NOTICES.txt` in every archive and a `-version` line; rehearsed on this host, unrun on a runner until the next tag**

Every archive has the Go runtime and standard library compiled into both of its binaries: the cgo
linux-amd64 build, the five CGO_ENABLED=0 builds, and every `glidertool`. That code is BSD-3-Clause,
and its second clause is about exactly this. A binary redistribution must "reproduce the above
copyright notice, this list of conditions and the following disclaimer in the documentation and/or
other materials provided with the distribution". Nothing in a gliderGo archive does that. The
packaging step (`release.yml:352-361`) stages the rewritten README, `LICENSE`, `CHANGELOG.md`,
`GliderPRO/README.md` and `GliderPRO/GPLv2-LICENSE.md`, then writes `HOW-TO-RUN.txt`. None of them
names the Go Authors. `-version` prints `built by go1.23.12` and `licence GPL-2.0-only` and stops
there, and `glidertool version` does the same. `credits.txt` has no line for Go, and the release
notes' Licence section mentions only GPLv2. SECURITY.md already says the standard library is the
only third-party code in a binary, so this is the one notice the archives owe, and they don't
include it.

Nothing else is owed. `go.mod` has no `require`, so no other Go code is linked in. The linux-amd64
binary loads libX11, libxcb and glibc dynamically from the player's own system, and an archive that
doesn't ship them owes them nothing. GPLv2 does not conflict either: BSD-3-Clause is compatible with
it, provided the notice goes with the binary. 1.1 settled the port's own licence and 1.2 settled the
1994 assets. Nothing so far has covered the toolchain.

The fix is a few lines in the package loop, next to `cp LICENSE CHANGELOG.md`:

```sh
{
  echo "glidergo and glidertool are built with $(go env GOVERSION), whose runtime and standard"
  echo "library are compiled into both. They are distributed under the licence below."
  echo
  cat "$(go env GOROOT)/LICENSE"
} > "$stage/THIRD-PARTY-NOTICES.txt"
```

The text comes from `go env GOROOT` rather than a committed copy. Under `GOTOOLCHAIN: local` that is
the toolchain that built these exact binaries, so the notice can't drift from them. The zip branch's
CRLF `sed` (`:507`) should convert this file as well as HOW-TO-RUN.txt, for the same Notepad reason.
The pre-seal assertions (`:517`) get one more line,
`grep -q 'Copyright (c) 2009 The Go Authors' "$stage/THIRD-PARTY-NOTICES.txt"`. Then a runner
toolchain with no LICENSE, or a later edit that drops the file, fails the build instead of shipping.
This has been rehearsed against go1.23.12: `$HOME/.local/opt/go/LICENSE` is there (1,479 bytes), and
the block above writes a 1,635-byte file that the grep matches. Whether setup-go's GOROOT on a
GitHub runner has the same file is unverified, like the rest of release.yml, and the grep is what
makes a surprise there loud. Go's `PATENTS` does not need to ship, because it is a grant and not a
condition.

Two sentences go with the file:

- **In the release notes' `### Licence` (`:736`) and in README's Licence section, which is in every
  archive.** "The Go runtime and standard library compiled into both binaries are BSD-3-Clause, ©
  The Go Authors; every release archive carries their licence as THIRD-PARTY-NOTICES.txt."
- **Optionally, a `-version` line.** `go  BSD-3-Clause, © The Go Authors`, beside the existing
  `licence` line (`cmd/glidergo/main.go:419`, `cmd/glidertool/main.go:155`). It covers the one-file
  binary a player copies to `~/bin` without its archive.

The licence only requires the file. The `-version` line is a courtesy. If a line goes into
`credits.txt` as well, make it a note under `[this port]`, not a new section:
`TestTheFileParsesIntoSectionsWithRows` pins the four section titles, and `People()` skips notes, so
no exemption needs editing.

**Done: the file in all six archives, and a check that fails without it.** The block above is in
the package loop at `release.yml:367-372`, after the `GliderPRO/` copies, with its two header lines
reworded. As proposed, "They are distributed under the licence below" followed a sentence whose
subject was glidergo and glidertool, so it could be read as putting the binaries under BSD-3-Clause.
The header now calls the runtime and standard library "That code" and points at LICENSE for
gliderGo's own. The zip branch's `sed` converts the file to CRLF along with HOW-TO-RUN.txt
(`:507`). The assertion is at `:543`, after the README check. Its pattern is `2009 The Go Authors`,
not the `Copyright (c) 2009 The Go Authors` proposed above. go1.23.12's own LICENSE has the "(c)".
The x/telemetry copy vendored in the same toolchain (`src/cmd/vendor/golang.org/x/telemetry/LICENSE`)
already has the 2024 wording, `Copyright 2009 The Go Authors.`, and 5.11 moves releases to a newer
minor. If Go's own LICENSE has changed the same way, the longer pattern would fail the first release
built on that minor, although the notice was there. The shorter one matches both wordings. The
release notes' `### Licence` (`:911-913`) and README's Licence section (`README.md:546-548`) carry
the sentence above, naming `glidergo` and `glidertool` where it said "both binaries". The notes call
the download one file and never mention glidertool, so "both" had nothing to refer to.

The whole "Package the archives" step was rehearsed the way 4.13 did it. Its `run:` block was taken
out of release.yml with a YAML parser and run against this host's `bin/cross/` with go1.23.12. All
six archives hold `THIRD-PARTY-NOTICES.txt`: 1,651 bytes in the four tarballs, and 1,681 with CRLF
in the two zips. Past the three header lines it is byte-identical to `$GOROOT/LICENSE`. `glidergo`
also links one package that GOROOT vendors, `vendor/golang.org/x/net/dns/dnsmessage`, on linux and
windows alike (`go list -deps ./cmd/glidergo`). Its LICENSE is byte-identical to `$GOROOT/LICENSE`
as well, so the same file covers it. With the block deleted, the step stops at the assertion on the
first target, exits 2 and seals nothing. With a GOROOT that has no LICENSE, `cat` stops it first.
The notes step renders the new paragraph with no `VERSION` left in it.

**Found on the way: the README assertion could never fail.** `! grep -q '](docs/' "$stage/README.md"`
sat in the middle of a `set -e` script, and bash does not exit on a command whose status is inverted
with `!`. A rehearsal that appended a `](docs/` link to the staged README sealed all six archives
and exited 0. It is an `if` that exits 1 now (`:533-536`), and the same rehearsal stops at the first
target with `README.md still links into docs/`. The planted link has to be one the rewrite cannot
match, such as `](docs/` with no closing parenthesis, or the `sed` rewrites it and the check passes.
No archive shipped a bad link because of this. README.md at each of the 35 commits from `56fab17`,
which added the pipeline, to `1797f9f` comes out of the rewrite with no `](docs/` left, and 4.13's
rehearsals found none to catch.

**And the `-version` line, the same day.** Both programs print `go        BSD-3-Clause, © The Go
Authors` under `licence` (`cmd/glidergo/main.go:420`, `cmd/glidertool/main.go:156`), so a binary
copied to `~/bin` without its archive still names the licence of the code compiled into it. The text
is `project.GoLicence`, which makes the two lines one string. `TestTheReadmeAndTheGameAgree` holds
README's Licence sentence to it, and README now says `-version` prints it. The licence does not ask
for this line. It is a courtesy.

**What this does not cover.**
- release.yml cannot run on this airgapped host. Whether setup-go's GOROOT on a GitHub runner has a
  LICENSE, and what it says, stays unverified until the next tag's workflow runs. The assertion is
  what makes a surprise there loud.
- Nothing went into `credits.txt`. The licence does not ask for it.
- No Go test reads release.yml or the staged file list, and none was added. The pre-seal
  assertion is the check, and it runs only on a runner.

### 1.5 The title screen said "Glider PRO" — **DONE, 2026-09-23, at the owner's request; the mark's question answered by the owner, 2026-09-24**

The first screen gliderGo showed was PICT 1000, Calhoun's title art, and its logo reads "Glider
PRO / by john calhoun". The README, the About box and the credits all say this is a port of Glider
PRO. The one screen every player sees said it was Glider PRO. Nobody here has checked who holds
that name now, and it doesn't matter: a port's first screen should not claim to be the original.
The screen also named nobody who made the port.

**Done.** `internal/shell/title.go` stamps the screen after the plate is copied:

- PRO is painted out.
- "Go" is written where PRO stood, 64x40, made from Calhoun's own G and the bowl of his d.
- "ported by brendan ta" is written under his credit, in his credit's letters.

All of it is in PRO's lighter brown. The CHANGELOG entry has the detail and the eight tests. The
PICT itself is unchanged. `drawBackdrop` is the only caller, and it stamps only when the pixels
under the stamp are the shipped PICT's. `credits.txt` names Brendan Ta under `[this port]` as well,
and `TestTheTitlesCreditIsInTheCredits` keeps the painted name and the written one the same.

**Decided by the owner, 2026-09-24: the mark ships as it is.** Route (a) in 1.2 rests on shipping
the same bytes upstream distributes, and the PICT is still those bytes. `titleMark` is not: it is
his G and d reduced and rearranged, and the credit reuses five of his letters, so the source holds
a small derivative of his logo that upstream does not distribute. This item had left that to route
(c), with the port's own 5x7 font as the fallback if a grant said no. The owner's ruling is that
the mark needs no permission of its own. A request under route (c) no longer has to name it, and
the fallback is not needed.

The no-art fallback, `drawOwnTitle`, said "Glider PRO, ported" under "gliderGo", and named
nobody as the porter. It is what a missing or mistyped `-art` looks like, so it is seldom seen,
and it was left alone at first. Since 2026-09-24 it says what the art's title says, in the port's
font: "a port of Glider PRO", "by John Calhoun, 1994" and "ported by Brendan Ta". The name comes
from the credits file's "ported it" row, and `TestTheTitlesCreditIsInTheCredits` holds it to the
name the art's title paints.

**Left open.**
- The README's copyright line still reads "the gliderGo authors". It is a statement about who
  holds the copyright, so this change does not touch it.
- `project.Original` is still "Glider PRO", and should be: it names the game this is a port of,
  and the About box and `-version` say "port of Glider PRO".

---

## 2. Engineering — what a faithful transcription leaves for a release

Items 2.1–2.3 are about the machine the game runs on. From 2.4 they are about the port and
the original: places where transcribing the 1994 code exactly is right for Stage 1 and
wrong for a shipped build. Each of those says what the original does, why it is kept, and
what the fix is, so that the fix is a decision someone can make later rather than a
rediscovery.

### 2.1 Window scaling — **remembered, 1.7b; fitted to the monitor, the release gate's step 4; fullscreen and a runtime toggle still open**

Already there: `cmd/glidergo -scale N` does integer nearest-neighbour magnification,
which is the right filter — the art is 8-bit indexed pixel art and bilinear would smear
it. What is missing for a release is fullscreen with letterboxing, a runtime toggle rather
than a launch flag, and remembering the choice. See 2.8 for where the transform has to go.

**1.7b took care of remembering it.** `prefs.Scale` is on the settings screen, is what
`openWindow` reads, and is the one row on that screen carrying a "next launch" note — because
the window is made once, at launch, and resizing it mid-session is a backend change (2.8) rather
than a preference change. The note is deliberately *not* on the other rows whose effect is
invisible from the settings screen (the bindings, the pause key, rooms in view, the music): all
four are read when a game starts, which is the only thing a player can do next, so a note on
each would say "as you would expect" four times and stop the one row that is genuinely different
from standing out.

Both `-scale` and `prefs.Scale` exist and they do not fight: `overrideFromFlags` folds the flag
into the settings before the window opens. The one place the flag is still read directly is
`-shot`, because a screenshot's magnification is a property of the file being asked for rather
than of the player's window — otherwise `-shot -prefs some.json` could change a golden image's
dimensions out from under the comparison it exists for.

**A fidelity trap to avoid when that work happens, which the port has so far avoided by
luck rather than intent.** `Main.c:191` forces `numNeighbors` to 1 on a 512px-wide screen
and `Settings.c:888` defaults it to 9. 1.5a measured what that costs: at 1/3/9 neighbours
the CD Demo House resolves 46/141/390 cross-room object links, so **a switch wired to the
room next door stops working on a small screen**. In the port `NumNeighbors` lives on
`render.Scene` and defaults to 9 with nothing reading the window size, so the two are
already decoupled — the risk is a future "derive it from the window like the original did"
change reintroducing the coupling. It should become an explicit setting, documented as
affecting gameplay and not just how much you can see.

**The next step, before the next tag: the first window fits the monitor.** A per-monitor-DPI-aware
640×480 window is small on 1080p and 1/27 of the area of a 4K panel. The window cannot be resized,
so a saved scale bigger than the screen runs the playfield off it. It is recoverable (`-scale 1`,
editing prefs, or the settings row if it lands on screen), and effectively lost only for a large
mismatch such as 8× on a laptop.

- **`prefs.Scale` 0 means auto, and is the default in a new prefs file.** It is stored as 0 and
  resolved at every launch to the largest N in 1..8 whose 640N×480N client area plus frame fits the
  work area, capped at 3× for now (next bullet). The resolved number is **never** written into
  `p.Scale`, because prefs are re-saved when the shell notices a different house (PLAN `:838-840`),
  and a stored number would stop the next launch re-fitting. `Validate` accepts 0..8. The settings
  row steps auto, 1×…8×, shows "auto (2×)", and Defaults resets to auto.
- **Auto stops at 3× until 4× is shown to be affordable (2.76).** Until 2.76's 4× bench row meets
  its stated budget on both backends, auto resolves to at most 3×, and an explicit 4×–8× is still
  the player's choice. If 2.76's changed-rows upload and server-side row repeat land first, the
  cap is never written.
- **An explicit saved scale that no longer fits is capped for this window only**, a local in
  `openWindow`, with one stderr line from `cmd/glidergo`. It does not go through `p.note`, which a
  backend cannot reach and which `Validate` has run before the screen is known.
- **The fit is one backend-level query** (null returns 1), not `Config.Scale=0` hidden inside
  `New`, because cmd needs the number for the banner (`play.go:1005`) and the row. The
  `platform.Config` comment changes with it.
- **Windows:** `MonitorFromPoint`/`MonitorFromWindow` plus `GetMonitorInfoW`'s `rcWork`, the
  existing `AdjustWindowRect` (`win32.go:289-294`), and the window placed inside `rcWork`, centred,
  instead of `CW_USEDEFAULT`. The cascade can put a fitting 2× window under the taskbar at 1080p.
  `SPI_GETWORKAREA` covers the primary monitor only. All of these are user32, a KnownDLL, so the
  preloading rule holds. Unverified here, so it is checked on the Windows test host in 5.4's
  pre-tag rehearsal.
- **X11:** mutter's per-monitor `_GTK_WORKAREAS_D<desktop>` first (present on this host), then
  `_NET_WORKAREA`, then `DisplayWidth/Height`. The last two span every monitor, so a laptop plus a
  4K screen picks 4× for the laptop. Per-monitor geometry needs Xrandr or Xinerama, which breaks
  "only libX11", so the limitation is written down instead. The frame is unknown before mapping:
  use `_NET_REQUEST_FRAME_EXTENTS` or a fixed allowance (35–49 px of title measured here).
- `-shot` keeps reading `-scale`, whose default stays 1.
- The fidelity `settings` hash changes if Default shows "auto (…)". It is regenerated on purpose,
  with a deterministic headless value.
- `NumNeighbors` is not touched. This entry's own trap stands.

This does not close 3.3's text-scaling item: a low-vision player on 1366×768 still gets 1× or 2×.
3.3 gets a cross-reference.

**Done, PLAN release gate step 4, with 2.76's changed-rows upload.** On this host's desktop a new
prefs file now opens a 1280×960 window. It reads `scale=2 (auto)` on the banner and `auto 2x` on the
settings row. The pieces:

- `platform.Room` and `Room.Fit` (`internal/platform/platform.go`), and `backend.Room()` on all
  three backends. Null answers the game's own size ("no screen"), so the fit there is 1×.
- `cmd/glidergo/scale.go`'s `windowScale` holds the policy, and `TestWindowScale` covers 11 cases
  with the development desktop, 1080p, 4K and a laptop.
  - Auto is `Fit(…, autoMax)`, and `autoMax` is 3.
  - A saved number that does not fit is fitted for this window only. The stderr line names what
    was measured and says the setting is unchanged, and the banner reads `2 (fitted from 3)`.
  - **An explicit `-scale` is kept even when it does not fit**, with a warning naming the largest
    that does. That is not what the plan above said. It is so `make bench -scale 4` stays a 4×
    row on whatever display it gets; a bench that quietly measured 2× would be worse than a
    window off the edge.
- **A timed or hermetic run never asks the display.** `-shot`, `-frames`, `-bench` and `-dump`
  are 2.53's reproducible runs, so auto is 1× there. `-scale`'s default is 0 now rather than 1,
  and `-shot` writes `max(scale, 1)`: the same 1× image as before, and a golden image that does
  not depend on the machine it was taken on.
- **X11** reads `_GTK_WORKAREAS_D<n>` for the monitor nearest the pointer, then `_NET_WORKAREA`,
  then the screen. With a window manager present (`_NET_SUPPORTING_WM_CHECK`) it subtracts a
  fixed 16×56 allowance for the frame. mutter's `_NET_FRAME_EXTENTS` for the window here is
  0, 0, 37, 0: a 37 px title bar and no side borders. `chooseRoom` is a pure function with 8 cases
  in `TestChooseRoom`.
- **The server-reset bug this found.** An X server with no other client resets when its last one
  disconnects, and refuses connections while it does. That is `xvfb-run` without `-noreset`, and a
  bare Xephyr. `Room` opening and closing its own connection just before `New` opened one got
  `New` refused in exactly that window, and the test suite hit it the first time a test closed a
  window. So `Room` keeps its connection as a spare, and `New` closes the spare only after its own
  `XOpenDisplay` succeeds.
- **Windows** does what the plan says: `GetCursorPos`, then `MonitorFromRect` (nearest), then
  `GetMonitorInfoW`'s `rcWork`, less `AdjustWindowRect` of an empty rect. The window is centred in
  that work area instead of placed with `CW_USEDEFAULT`, clamped to its top left, and falls back to
  `CW_USEDEFAULT` if the query fails. It builds and vets for amd64 and arm64 and **has not run**.
  5.4's rehearsal checks it on the Windows test host, and `docs/windows-first-run.md` says what to
  look for.
- **The settings row** steps auto, 1×…8×; Left from 1× reaches auto, and R resets to it. It shows
  `auto 2x` rather than the plan's `auto (2×)`: the value column is 90 px wide at the row's scale,
  before "next launch" starts, and the bracketed form is 108. `TestSettingsValuesFitTheirColumn`
  now measures every value every row can reach, including every key name a binding can show, so
  the next wide value fails a test instead of overprinting a hint. `shell.Host.AutoScale` carries
  the number. `-shot` leaves it 0, so the golden settings screen reads plain `auto`, and that hash
  changed on purpose.
- **An existing prefs file keeps its number.** Files written by 0.1.x say `"scale": 1`, because 1
  was the default, and there is no telling that apart from a player who chose 1×. So there is no
  migration. The CHANGELOG says that stepping below 1× on the row, or R, gives auto.

Not done, and filed: per-monitor geometry on an X11 desktop that is not GNOME, where neither
`_GTK_WORKAREAS` nor anything else in core X names a monitor (2.77).

### 2.2 The simulation is frame-locked at 30.07 fps and stays that way — **policy; interpolation is 1.7 or later**

`kTicksPerFrame = 2` Mac ticks on a 60.15 Hz clock, i.e. 30.07 frames a second, and every
physics constant in `player/consts.go` was tuned against it. That is not negotiable and
the port keeps it: `TicksPerFrame` is a constant, `Frame` is the only clock the game logic
reads, and no future display refresh rate may change either. Frame *interpolation* for
60/120/144 Hz displays is a separate question — it belongs at the presentation step (2.10)
and must not touch the simulation. The limiter's own cost is 2.17.

### 2.3 Player 2's keys are modifier keys — **DONE, 1.7b (all eight bindings are the player's)**

The original hard-codes player 2 to Control, Command, Option and Shift
(`InterfaceInit.c:148-151`), which a modern window manager or desktop environment may
swallow before the game sees them, and which many keyboards cannot report independently.
1.4 anticipated this: the four key indices are per-glider *data* on `player.Glider`, not
constants baked into `GetInput`, specifically so they can be remapped.

`cmd/glidergo` binds player 2 to A/D/S/W by default, which is a documented deviation and is
in the package comment there.

**1.7b made all eight bindings the player's**, in `internal/prefs` and on the settings
screen (`S` from the title screen), and persisted them in the native configuration
directory. Three things about the shape of that are deliberate:

- **A binding is a key *name*, not a scancode.** `prefs.Controls` holds strings
  (`"left"`, `"a"`), which `platform.ParseKey` resolves; a name that will not parse becomes
  `KeyUnknown`, and `Window.KeyDown` answers false for that on every backend, so a
  hand-edited file with a typo in it gives a dead control rather than a wrong one or a crash.
- **A player who wants the 1994 bindings back can have them**, which is what this item asked
  for: the modifier keys are bindable like any other, and whether the window manager lets
  them through is now the player's problem rather than the port's decision.
- **The bindings are resolved once per game**, not once per poll (`cmd/glidergo/play.go`),
  because the settings screen is only reachable from the title screen. So a rebind can never
  be half applied to a game in progress.

Wanted, but not scheduled: gamepad support, which the original had no concept of and which
`internal/platform` has no device layer for.

**The route, when it is wanted.** There are two layers, not two alternatives.

- **The device layer, `internal/platform/pad`.**
  - Linux: `/dev/input/js*` as 8-byte `js_event` records, with `JSIOCGBTNMAP` through `syscall`
    to find `BTN_SOUTH`/`BTN_EAST`/`BTN_START`, because joydev numbering is per device. The
    `/dev/input/js*` glob is re-read every 1–2 s; inotify is not needed.
  - Windows: `XInputGetState` from `xinput1_4.dll`, loaded by absolute path, with winmm
    `joyGetPosEx` as a fallback for HID pads XInput does not see.
- **The seam: a Window decorator, focus-gated.** It answers `KeyDown` for named pad keys
  (`pad1_a`, `pad1_left`). It ORs a fixed per-player overlay into `KeyPoll`: pad1 to P1, pad2 to
  P2, d-pad or stick past a deadzone to Left/Right, two face buttons to Batt/Band, Start to pause.
  It is keyed to the player, not to the arrow keys, or a pad could only ever drive P1.
- **In menus**, A confirms and B goes back. B must do nothing on the title screen, where Escape
  quits with no question. Back/Select maps to the pause screen's give-up.
- **The settings capture sees raw pad names**, not a translated Return/Escape. Otherwise binding
  A binds "return", which `Validate` rejects, and B cancels the rebind.
- The game, the replay format and netplay never learn a pad exists. Text entry still needs a
  keyboard.

Nothing here can be run on hardware this project has, which is the reason it is "wanted" and not
scheduled.

### 2.4 Transitions run at memory speed, so a wipe is a blink — **planned, 1.7**

`WipeScreenOn` (`internal/game/screen.go`) moves a 4-pixel bar across the screen and
presents after each step: 116 steps for a vertical wipe, 160 for a horizontal one. On a
1994 Mac each of those was a real `CopyBits` over the bus and the count *was* the
duration. Here they cost nothing measurable — the headless build runs the whole 116-step
wipe inside one game frame — so a transition that should read as a wipe reads as a single
dropped frame.

The fix is to pace the strips against a fixed wall-clock duration (a third of a second is
about what the original felt like) rather than against the strip count. It is a deliberate
divergence, so it is not made in `screen.go`; it belongs with the rest of the presentation
work in 1.7, and it needs the frame limiter's clock, not a `time.Sleep` per strip.

Two smaller things to fix in the same pass, both reproduced faithfully today:

- The horizontal strip count reads `workSrcRect.right / 4` rather than `theRect`, so a
  caller passing a narrower rect still gets 160 strips. It should clamp to `theRect`.
- Only `top` and `bottom` are clamped inside the loop, so a horizontal wipe's bar walks
  off the far edge for its last few strips and those copies are no-ops absorbed by
  `CopyBits`' clipping.

This item was also the reason the first version of `playTestWorld` in
`internal/game/play_test.go` measured the wrong thing: it budgeted *presents*, and a
transition presents 116 times inside one frame, so "100 frames" bought 61. The budget is
frames now.

`cmd/glidergo`'s `-frames` flag had the identical bug and it is fixed the same way. It
counted its own `Present` calls, so `-frames 300` ran 298 frames in a static room — two
presents happen outside the loop — and 137 in a run where the glider took a door, because
one 160-strip wipe spent more than half the budget. A flag whose meaning depends on where
the glider drifted is not a flag anyone can benchmark or replay with; `wrapPresentLimit`
now tests `World.Frame`. The `-dump` PNG count is deliberately still per present, because
one file per wipe strip is what you want when looking at a transition.

**"Costs nothing measurable" was a headless figure, and on screen it was the opposite.** Every strip
is a present, and until the release gate's step 4 every present sent the whole magnified window. On
this host's Xephyr, a door (`-room 8`, frame 121) took 229 ms at 1×, 1.45 s at 2×, 3.0 s at 3× and
4.2 s at 4×, in one frame. That is a freeze, not a wipe, and 2× became the 1080p default in the same
step. 2.76's changed-rows present sends each strip's 4-pixel bar and whatever the new room changed
under it, so the same door is 99–137 ms at every scale. That is still under this item's third of a
second, so the pacing is still wanted, and a paced wipe can now afford its strips.

### 2.5 Pausing blocks the process, and unpausing leaves the screen black — **DONE, 1.7b**

`DoPause` is called from inside `GetInput` and *blocks* until the player unpauses, which is
why a paused game does not advance a frame. Faithful, and fine on a cooperatively
multitasked Mac; on a modern OS a blocking modal inside the input path is how a game stops
responding to the window manager.

`RestoreEntireGameScreen` is the other half. It paints the window black, composes into the
work map, and **never copies the work map to the screen**. What makes the room reappear in
the original is the update event the vanishing dialogue generates, which
`HandlePlayEvent` turns into a whole-`justRoomsRect` blit. So the redraw arrives via the
event pump, one frame late — and with `doBackground` false (the shipped default, see 2.21)
it never arrives at all: the screen stays black except where the dirty rects happen to
fall. That is a visible bug in the original in its default configuration.

A release needs a pause overlay that says what to press, a pause that yields to the OS
instead of blocking, and a `RestoreEntireGameScreen` that publishes what it composed.

**1.7b built the first two and left the third alone.** The pause is split at the line between
the game and its host: `internal/game/pause.go` draws the placard, holds `World.Paused` up and
restores the rect from the work map afterwards, and `World.Pause` — a hook `cmd/glidergo`
fills in — does the waiting, pumping host events every pass and repainting after each one. So
the window answers the compositor, its own close button and an expose while the game is
paused, and none of that is in `internal/game`. Four consequences worth recording:

- **The pause key acts on its press edge**, in both halves. The C's three `GetKeys` release
  loops exist because `theKeys` is a global that `DoPause` overwrites — see
  `docs/analysis/input.md` §10.2-10.3 — and a host that reported the key level-triggered
  instead would pause, resume and pause again on a single press.
- **One deliberate ordering change.** The C erases the placard and *then* waits for the key to
  come up, so holding the pause key after unpausing shows a frozen game with no placard on it.
  This port waits and then erases: a held key shows a paused game.
- **The artwork names a key this port does not have.** Both PICTs read "or Cmd-Q to Quit the
  game" and there is no Command key here, so `World.PauseHint` draws a row under the placard
  naming the substitute (Q), and the restore covers the union of the two. 2.7 is why that row
  is load-bearing rather than a nicety.
- **A nil `Pause` hook is no pause at all**, which is what the fidelity corpus needs: a replay
  has no keyboard, so a faithful transcription would hang on the first recorded frame with the
  pause key down.

`RestoreEntireGameScreen` still composes without publishing, faithfully. It is unreachable in
this port today — the modal dialogues it tears down are 1.7c's and 1.10's — and the honest
place to fix it is the commit that first calls it, where the fix can be seen to work.

### 2.6 A missing or wrong asset directory must not be a crash — **DONE, 1.7a**

Today the tests `t.Skipf` when `assets/extracted` is absent, which is right for tests and
is not a shipped behaviour. A public binary run by someone who has not pointed it at a
copy of Glider PRO must say so, in a window, with instructions — not panic on a nil
surface or exit silently. This is the user-facing half of item 1.2 and the two should be
built together. `render.Assets` already collects a sticky error rather than dying at the
blit (`assets.Err()`), which is the mechanism this needs; what is missing is a window to
show it in.

**1.7a built the window.** A fresh clone with no assets now comes up on the port's own
title screen — a drawn wordmark, the credit, "no artwork found", and the command that
produces the artwork — with a working menu over it, and the status band naming the
directory it looked in. Three things make that a *tested* behaviour rather than a hopeful
one:

- `shell.Host.Assets` may be nil, and every plate goes through one accessor
  (`Shell.plate`) that answers nil rather than reaching into it. There is no other path to
  a PICT in the package, so a missing plate cannot become a nil dereference.
- `Discover` failing is printed and not returned (`runShell` in `cmd/glidergo/main.go`):
  the useful place to say "there are no houses here" is the screen, not a terminal the
  player may never see. The three menu items that need a house are unavailable and say why
  when pressed.
- `make headless` renders the no-art and no-houses screens on every `make check`
  (`-shot -art /nonexistent -houses /nonexistent`), because it is the one layout nobody
  developing here ever sees by accident — and the first version of it put "no artwork
  found" half underneath the menu panel.

**1.7b extended the same rule into the game.** The pause placard is PICT 1015 or 1016, and a
checkout with no art now gets a drawn panel in the same 214×54 the picture would have filled —
"PAUSED" and the key that resumes — so nothing else on screen moves depending on whether the
art is there. `internal/game/pause.go` is the only place `internal/game` draws text, and
`TestPausePanelIsDrawnWithoutArt` and `TestPausePanelNamesTheKeyThatResumes` are what keep the
fallback from quietly becoming a blank rect.

### 2.7 No way to quit that a stranger would find — **DONE, 1.7a and 1.7b; closing the window mid-game is reopened as a follow-up, next**

`cmd/glidergo` maps Escape to quit and that is undiscoverable. The original's answer was a
menu bar, which a port does not have. A release needs a title screen with a Quit item
(3.4), an in-game pause overlay with one (2.5), and a confirmation before abandoning a
game in progress.

**The title screen's Quit item exists**, on `Q` and on Escape, and the About box lists both
along with every other binding. Escape *in a game* changed meaning in 1.7a to make room for
it — it ended the game and handed the title screen back, where it used to end the process —
and then changed again in 1.7b, below. Closing the window still ends the process, which is why
`shell.Outcome` carries a `Closed` flag: the two exits look identical to the `World` and must
not to the shell.

**1.7b closed the rest of it, and not the way this item expected.** The plan was a
confirmation dialogue in front of the give-up. What shipped instead is one rule with no new
screen in it: **Escape pauses.**

Escape pauses whether or not it is the player's pause key, either key resumes, and `Q` from
inside the pause is the give-up. So the keystroke a stranger reaches for to get out of a game
now stops the game and shows them a placard with the way out written under it, instead of
throwing the game away — which is the confirmation, one keypress earlier and with nothing extra
drawn. Two things fell out of it:

- **`prefs.PauseKey` keeps its 1994 meaning exactly.** It picks the placard
  (`isEscPauseKey`) and nothing else; the host no longer has to ask which key is spoken for,
  and the old `escPauses` branch in `cmd/glidergo/play.go` is gone.
- **The give-up key is not the quit key.** Q ends the game and hands the title screen back;
  closing the window ends the process. `shell.Outcome.Closed` is still what tells those two
  apart, because they look identical to the `World`.

**Reopened as a follow-up: closing the window mid-game discards the run.** Closing during a
saveable one-player game ends the run and the process at the next poll, and saves nothing
(reproduced: frame 41, `Outcome.Closed=true`, no save file).

The follow-up: the first close in such a game pauses on the existing placard and asks the quit
form of the existing `asking` question. Y saves, then quits. N quits. The pause key keeps playing.
A second close quits without saving.

- **These still quit on the first close:** a race (an explicit `netRace != nil` check, because
  `CanSaveGame` is true in a race), `-frames`/`-bench`/`-dump`, the game-over countdown, the
  `Wait` screens and the dialogs. Nothing unsaved is at risk in any of them.
- **It all lives in `cmd/glidergo/play.go`.** Each `WM_CLOSE` or `WM_DELETE_WINDOW` already
  arrives as its own `EventQuit`, and the backends' latches have no production reader.
- **The pause is injected through `KeyPoll`'s Pause bit** (a `pendingPause` flag), because the
  Pause hook calls `PlayEvent` itself.
- **A close while switched out clears `SwitchedOut`**, as the arm already does.

This is a new behaviour of the port. The original's app-level quit mid-game does not prompt. It
follows this entry's own Escape-pauses precedent. README `:218` changes in the same commit.

### 2.8 The scale transform belongs at the present step and nowhere else — **planned, 1.7**

The game's surfaces are always 640×480 and every rect in `internal/game` and
`internal/render` is in that space: `View.Screen`, `LocalRoomsDest`, the dirty-rect lists,
`playOriginH/V`. A resizable window must therefore insert its transform in exactly one
place — `platform.Window.Present`, or the backend behind it — and never in a coordinate
handed to the game. Written down because the tempting shortcut is to scale `View`, and
that would silently move every collision rect in the game.

### 2.9 The scoreboard is invisible in the original's shipped configuration — **DONE as a deviation, 1.5b**

Four facts compose into a bug the original shipped with:

1. `StructuresInit.c:70` initialises `boardDestRect` to rows −20..0 — above the screen.
2. `Main.c:154` defaults `numNeighbors` to 9, and `:191-192` only forces 1 on a screen
   512px or narrower.
3. `Scoreboard.c:416-421` guards the whole body of `AdjustScoreboardHeight` with
   `wasScoreboardMode != newMode`, and `wasScoreboardMode` starts at `kScoreboardHigh`.
4. `numNeighbors == 9` computes `newMode = kScoreboardHigh`, which equals
   `wasScoreboardMode`, so the guard never opens and the seven `QOffsetRect` calls that
   would move `boardDestRect` on screen never run.

So on any Mac with a screen wider than 512 pixels — which is every Mac the game shipped
for — the scoreboard is blitted to rows −20..0 and clipped away entirely. Score, lives,
battery and bands are simply not displayed.

The port deviates, deliberately and by default: `World.BoardDestRect` is initialised to
rows 460..480 (the band `render.NewView` leaves between `Screen` and `House`, and exactly
the strip `NewGame` paints black at `Play.c:141`), and `AdjustScoreboardHeight` *assigns*
its rects instead of accumulating offsets, which makes it idempotent and turns the latch
into an optimisation rather than a correctness requirement. The High arm's rows are the
port's; the Low arm's are the original's. See the comment on `BoardDestRect` in
`internal/game/world.go` and the head of `internal/game/scoreboard.go`.

This is the one place Stage 1 knowingly shows something the original did not, on the
grounds that a scoreboard nobody can see is not a design decision.

### 2.10 Presentation is welded to the simulation tick — **planned, 1.7 at the earliest**

`RenderFrame` composes, publishes and waits, in that order, once per simulated frame. A
release on a 144 Hz display wants the composition at 30 Hz and the presentation faster,
which means splitting `awaitFrame` from `Present` and interpolating sprite positions at
the present step only. Every position the game computes stays integral and stays at 30 Hz
(2.2); the interpolation is a display artefact and must not feed back.

### 2.11 The dirty-rect lists silently drop past 47 — **counters DONE, 1.5b; the surfacing is 1.7**

All three adders guard on `numWork2Main < (kMaxGarbageRects - 1)` and, when the guard
fails, drop the rect with no report. A dropped *work* rect is a patch of screen that is
never updated; a dropped *back* rect is a patch of work map that is never erased, so
whatever was drawn there smears. Both are reachable in a busy room — a mirror room is the
easy case, because 2.19 registers an unclipped rect per frame — and both are reproduced.

The improvement is not to raise the cap, which would change what the original showed. It
is a debug counter behind a flag, so that a house author or a bug report can say "this
room overflows the rect list" instead of "this room flickers".
`TestPlayGameKeepsPublishingFrames` already asserts the lists stay well under the cap in a
quiet room, which is the regression half of the same idea.

**Done at 1.5b: the counting.** `World.Diag` (`internal/game/guards.go`) holds
`DroppedWorkRects` and `DroppedBackRects`, incremented at all three drop sites in
`render_frame.go`, and both are in every line of a replay trace (`glidertool replay
-trace`) as well as its footer. So the overflow is now a number a bug report carries
rather than a symptom somebody has to name. It is *not* behind a flag: the counters cost
an increment on a path that already dropped the rect, and a diagnostic nobody enables is a
diagnostic nobody has.

**Left for 1.7: telling the player, or rather telling us.** Nothing surfaces the counters
in a running game — there is no debug overlay and no log line — so a stranger who hits it
still just sees flicker. That half needs 1.7's shell to have somewhere to put it.

**Measured at 1.5c: three mover types spend rects whether or not anything is happening,
and one shipped room burns a third of the budget standing still.** `RenderBall`,
`RenderDrip` and `RenderFish` are the three renderers with no `Moving` gate — they cannot
have one, for the reasons their comments give (a spent ball sits on the floor and is still
lethal; the drip's hanging cel must survive the first back→work erase; a resting fish's
unmasked water is the animation) — so each costs one work rect and one back rect on *every*
frame for the life of the room. The drip is worse than the other two, because while the drop
hangs its `Whole` is still the union of the last fall, so the rect it registers is the whole
column rather than the drop. Counted over the shipped corpus, `SpacePods.house` room 55 "Ion
Generator" holds 16 of these; `glidertool replay -house SpacePods -room 55 -trace` shows a
flat **17 work and 17 back rects per frame** with the glider standing still and nothing
triggered — 36% of the effective 47 gone before the room does anything. `California or
Bust!` room 10 has 12, `SpacePods` room 105 has 11, `Grand Prix` room 7 has 10.

That is faithful, and it stays. It is recorded here for two reasons. First, it sets the
floor a house author has to budget against, which makes it input to 4.1's linter: a room
with 16 ungated movers plus a mirror (2.19's unclipped per-frame rect) plus a busy glider is
where the 47 actually runs out. Second, if presentation is ever reworked (2.10, 2.24) these
three are the whole reason a naive "only upload dirty rects" backend would not be much
cheaper than uploading the frame — the standing set is not small.

### 2.12 A missing graphic kills the application — **planned, 1.7**

The original's answer to a failed PICT load is `RedAlert(kErrFailedGraphicLoad)`, which
quits. That is defensible for a game shipped on a CD with its own resource fork and
indefensible for a binary that extracts its assets from someone else's install (1.2). A
release should degrade: draw a placeholder, note it, and carry on. The arcade scoreboard
block is the concrete case — see `arcadeBlackenBoard` in `internal/game/scoreboard.go`,
which reloads the scoreboard PICT *per game over* and dies if it is missing. It should be
loaded once at launch.

### 2.13 One failed sound bank mutes the entire game — **planned, 1.6**

`LoadSounds` treats any failure as fatal to sound as a whole, so a single missing 'snd '
resource costs every effect in the game. A release should load per-sound and lose only
what is actually missing. While there: the original has one volume, set from the Sound
control panel. A release needs at least effects and music separately, and a mute that
survives a restart.

### 2.14 A house whose custom trigger sound fails to load is silently silent — **the linter half DONE, 2.0; the in-game log is still open**

A house can carry its own 'snd ' resources for triggers. If one fails to load the original
plays nothing and says nothing, so a house author gets no signal that their sound is
broken. The house linter (4.1) should report it, and the game should log it once.

**2.0 did the linter half, and found that the silence has two causes that must not be reported
the same way.** `house lint` reports `sound-id` when the house names a `snd ` it does not carry
— 13 across the corpus, all real — and `sound-unreadable` when the house does carry it and
gliderGo cannot decode it (2.49). Both findings say the thing this entry had wrong: the failure
is worse than "silent". `LoadTriggerSound` failing makes `CreateActiveRects` compose the room
with no hot spot at all, so the trigger cannot be *touched*, and a house that used one as a
signpost has lost the signpost rather than its sound.

The in-game log is still open, and is smaller than it was: the author-facing signal now exists,
so what remains is one line on `stderr` for a player whose bug report says a trigger does
nothing.

### 2.15 Leaving a game requires a physical key release — **DONE, 1.7b**

`WaitCommandQReleased` spins until the player physically lets go of Command-Q before the
game will proceed. On a Mac with a real menu bar that prevented a held chord from firing
twice; in a port it is a hang waiting for a key event that may never come (the window can
lose focus mid-chord). It is deliberately **not** transcribed in `NewGame`.

**1.7b kept it untranscribed and answered the question it was asking.** What the spin is for
is "do not act twice on one press", and the port's answer is an edge rather than a wait: the
pause key is reported on its press edge (`cmd/glidergo/play.go`'s `pauseHeld`), and the give-up
Q arrives as a single `EventKeyDown` rather than as a held-key poll. Neither can fire twice on
one press, and neither can wait for an event that never comes — which is the property this item
wanted and the spin only approximated. The pause's own wait loop does track a release, but it
is `held` on a key the player is holding *now*, and it exits on the window closing too.

### 2.16 The two-player idle freeze has no visual tell — **measured in 1.9; the tell itself is deferred to 2.x**

`TagGliderIdle` freezes a glider for 30 frames — a full second — with no indication that
this is deliberate. Player 2 in particular starts every two-player game idled and hidden
(`Play.c:198-203`), so the first thing a new player experiences is a second of not
existing. A fade, a shimmer, or anything at all would do.

**1.9 nailed down what it actually happens to, which was not what this entry assumed.** The
freeze is not player 2's alone: `ReadyGliderFromTransit` freezes whoever is *not*
`w.FirstPlayer`, and `FirstPlayer`'s zero value is `Player2` — so before anybody has waited on
anybody, it is **player 1** who arrives frozen. The countdown is stored in `g.HVel`, reused as a
counter while the glider cannot move. `game.TestTransitArrivalFreezesWhoeverIsNotFirstPlayer`
tables both directions, and `game.TestOnePlayerTransitDoesNotFreezeAnybody` pins that a solo
game never sees it.

**Why the tell is not in 1.9.** Anything drawn during those 30 frames changes pixels in rooms
that 1.8's fidelity corpus has already hashed, so the visual work has to land together with a
corpus re-record and a decision about whether the tell is a `fixes` flag (recordable both ways)
or an unconditional presentation change (corpus moves once, permanently). Both are cheap; neither
is free, and doing it inside 1.9 would have mixed a rendering change into the stage that pins
two-player *behaviour*. The behaviour is now pinned, so the tell can be added against tests that
already say what it must not disturb.

### 2.17 The frame limiter is a busy-wait, and there is no catch-up — **DONE (the hook), 1.5b; the setting is declared, 1.7b; the catch-up is 1.8**

The original's limiter is `while (TickCount() < nextFrame) { }`, which on a modern machine
pins a core at 100% for whatever fraction of the two ticks the frame did not need, for the
whole time the game is running. `awaitFrame` takes a `World.WaitTick` hook for the loop
body: nil is the C's empty body and is what a fidelity build uses, and `cmd/glidergo`
installs a `time.Sleep(time.Millisecond)`. That changes only *how* the wait is spent, not
when it ends — `awaitFrame` still returns on the same tick — so it is not a fidelity
change and needs no flag.

The second half is still open. `nextFrame` is reseeded from the clock *after* the wait
rather than by adding `kTicksPerFrame` to the previous deadline, so **there is no
catch-up**: a frame that overruns makes the game run slower rather than skip, and every
frame still advances `gameFrame` by exactly one. That is why a slow Mac played the same
game in more wall-clock seconds instead of a different game, and it is the right default.
A release should make it an explicit setting — "keep real time" versus "keep game time" —
because a modern player on a machine that stutters expects the former. Any variable-
timestep work changes that one line and nothing else.

**1.7b declared the setting and deliberately left it unwired.** `prefs.KeepRealTime` is in the
file, defaults to false — the C's behaviour — and nothing reads it yet, which is stated on the
field rather than left for a reader to discover. The reason is that a catch-up worth having
needs a resync clamp: without one, the first long stall — a room wipe, a pause, a house load —
is followed by a burst of unpaced frames, which is a worse artefact than the dropped time it
was meant to recover. That is frame-pacing work, it belongs with 1.8's timing pass, and a
setting that is written down and honest about being inert is cheaper to finish than one that
has to be invented later along with the file-format change to carry it.

### 2.18 The random stream is unverified against real hardware — **premise disproved, 1.8b; the table test is 1.8c; the demo now plays to its end, its floor is the whole stream, and the recording supports no further oracle**

`internal/game/rand.go` transcribes the original's linear congruential generator, and the
demo replay in 1.8 depends on it bit for bit. It has not been checked against a trace from
a real Mac, and until it is, a demo that desyncs is ambiguous between "the RNG is wrong"
and "the input replay is wrong". 1.8 should capture a reference trace first.

**The premise above is wrong, and `toolbox-primitives.md` had already disproved it — 1.8b
confirmed the disproof by experiment.** The demo replay does **not** depend on the RNG. §1.12's
proof is in four steps: `GetDemoInput` dispatches on the frame counter and draws nothing; `Player.c`
contains no `Random` call at all, so trajectory is a function of (initial state, input, geometry);
Demo House contains **0** `kSparkle`, **0** `kCoffee`, **0** `kChimes`, `phoneBit` false — so none
of the RNG-consuming *registrars* even fires; and §1.13 enumerates every reader of the five arrays
the RNG feeds and finds nothing but `CopyBits` source-rect selection. R-RNG-2 is normative: a port
could replay the shipped demo frame-exactly with a `Random()` that returns 0 forever.

1.8b's replay is the empirical confirmation, and it still holds after 4.24's fix. Seeds 0, 1, 7 and
12345 produce the *same* run — the same three deaths, all 1117 records consumed, the same end frame,
3432 — and differ only where the random stream shows: the trace's `rand=` column and so its digest,
and, for seeds 7 and 12345, the last frame's main-plane digest. That is exactly the containment
§1.13 describes. So the ambiguity this entry was written about is gone: a demo that desyncs is the
physics, not the RNG.

**And 1.8b's own note about clock seeding was wrong, which is worth recording because it inverts
the conclusion.** `ToolBoxInit` does `GetDateTime((UInt32 *)&qd.randSeed)` at `Utilities.c:61`, but
inside `#if !TARGET_CARBON` — and `GliderPRO/Prefix.h:1` sets `TARGET_CARBON 1`. The build this
source tree describes therefore never seeds: it starts from `randSeed == 1` on every launch
(`toolbox-primitives.md` §1.3, §1.4), which is why §1.6's verified table is a seed-1 table. The
clock seeding is the pre-Carbon 68k branch. So the attract mode *was* reproducible, the shipped
demo **is** a legitimate fidelity oracle, and the 573-of-1117 divergence below was a defect in this
port and not an artefact of a stream nobody can reproduce. The defect was 4.24's misread `'bnds'`.

What is left of this entry is narrower and still real: `internal/game/rand.go`'s generator is a
reconstruction of a Toolbox trap whose code is not in the tree, and it has never been checked
against anything. §1.6 tabulates 24 verified draws from seed 1 — state, `Random()` as `int16`, and
`RandomInt` at the three ranges the game asks for — so the check is a table test, and 1.8c writes
it. That closes it as far as it can be closed without a Mac to ask; what remains open after that is
only whether Apple's trap really was Park-Miller, which §1.5 argues from the documentation and
§1.13 bounds the cost of.

**Amended at `b6f4986`'s follow-up: the gap this entry called the sharpest target is closed as far
as the recording can say.** The `'bnds'` fix (4.24) took the demo from 573 to all 1117 records
consumed. The deaths are at f1781, f2043 and f3417. The third flags the game over three frames
after the last record at 3414, and the countdown ends the run at f3432. Nothing shows those are
wrong, and the stream points the other way.

- **The game over fits.** A game over with `kInitialGliders = 2` needs three deaths unless a
  glider is picked up, and the recorder stops logging when `gameOver` is set.
- **The burn fits.** The recorder logs nothing while the glider burns, and the port's burn right
  after the record at f2015 lines up with the 41-frame silence in the stream there.

So all 1117 consumed, a game over just after the stream's end, and no record on a burning frame
are every check this recording can support. What is left is not "raise a number". The next oracle
is a trace from a real Mac. That is an inference, and it is written as one.

Everything that quoted the old run is corrected, with the demo floor (PLAN's gate, done). Each
figure was measured again first:
- this entry's seed result, the Carbon-seeding paragraph, the 1.8b measurement, its evidence and
  the ratchet paragraph now say what the run does today. They keep 573 only where they say what
  1.8b measured. The seed result still holds, with 1117 and 3432;
- 4.7's 1.8b paragraph is in the past tense, and a new paragraph after it says the day it waited
  for has come: `128.bin` is tracked, so a demo corpus row is buildable from a fresh clone;
- README's fidelity bullet and the `demo.script` header (PLAN 1.8b was corrected already);
- `demoRecordsFloor` is `demo.ShippedRecords`, and its dead "raise the floor" branch is dropped.

The port's own death frames are **not** pinned as an expectation, because that would be a golden
of the port and not of 1994. Two things are acceptable as the next step: 4.7's short-prefix demo
corpus row, or a fidelity test asserting only what the stream implies.

**What the demo replay measured at 1.8b, and the number it set to beat.** Until 4.24's fix the port
did not fly the recorded path. Replaying the shipped stream against Demo House:

- the glider never left room 0, "Air Vents" — three `kFloorVent` at v=305 and a `kRedClock`,
  which it *did* collect for 100 points at frame 310, so the first ~300 frames were plausibly
  right;
- it then faded out (mode 2) frozen at `dest=295,387,315,435`, below the floor, losing a life at
  frames 1412, 1573 and 1760; game over at frame 1775, having consumed **573 of 1117 records**;
- the recording expects the vents to carry it rightward out of the room — the longest held run in
  the stream is 66 frames of right from frame 1879, well past where the port had already died.

Three pieces of evidence said the harness was not what was wrong, and all three still hold. The
outcome is **seed-independent**: seeds 0, 1, 7 and 12345 all ended at frame 1775 with 573 records
consumed and mortals at -1, and all now end at frame 3432 with 1117 consumed and mortals at -1. Only
the random draws differ, so the deaths are physics, not RNG. The recording is **one record per held
frame**, not per alternate frame: `glidertool demo info -stats` reports 48 distinct gaps with
`1:1009` of 1116, 108 held stretches (right 84, left 21, band 3) — so the port's frame counter is
the right clock to replay against. And the parity is even, 557 to 560, so no input pass is running
on only one of `World.EvenFrame`'s two phases. That left the vent lift, the fall-through, or the
air-friction integrator, and it was the sharpest fidelity target this project had: **573 of 1117 was
the number to raise.** It was none of the three. Room 0 has no `bounds` of its own and falls back to
the house's `'bnds'` resource, which the port misread as closed on all four sides (4.24). With that
fixed, the glider still collects the clock at frame 310, leaves room 0 at frame 391, and the run
consumes all 1117 records.

Two things follow for whoever picks it up. The frame numbers above live in the script's header
comment, this entry, and a `t.Logf` — deliberately not in an assertion, because the day the physics
improve, the test that fails should be a fidelity test and not a test about determinism. **That
reasoning is right and it had a hole, closed in Stage 2:** it left the number unguarded in the
direction that is unambiguously bad, so a change that dropped the glider to 300 records would have
gone green while this entry and two other documents went on claiming 573. Stage 2 added a one-sided
ratchet, `demoRecordsFloor` in `internal/replay/replay_test.go`, at 573: below it failed, and above
it the test logged a request to raise the floor. The floor is now `demo.ShippedRecords`, all 1117,
and the request branch is gone, because no run can consume more records than the stream holds
(`internal/replay/replay_test.go:1318-1341`). Fewer than 1117 fails. A change to the physics still
does not turn the determinism test red unless it stops the flight short. And the run that plays the
whole stream reads 3 frames past the last record, frames 3415 to 3417, before the third death flags
the game over. That is the off-the-end read the original performed and the port counts as
`Cursor.PastEnd`; `frames 3500` in the script is set past the end on purpose so that the run can
outlive its stream.

Two related notes: `PourScreenOn` is dead code in the original — nothing calls it — and it
*draws from the RNG*, so wiring it up would shift every subsequent random number and
invalidate any recorded demo. And `case kLgTrigger:` in the interaction dispatch is
unreachable, because the object code is never produced; it is transcribed as a dead branch
and should stay dead.

**1.5c makes this harder, and the reason is worth writing down before 1.8 hits it.** Three
dynamics sites draw from the stream:

- `AddDynamicObject` (`Dynamics3.c:208`) draws `RandomInt(60) + 15` per `kSparkle`
  registered. Registration is per *locale*, filtered by `SectRect` against the screen, so
  **the stream position after a room change is a function of how many sparkles were on
  screen** — which is a function of the window size. Any demo trace is therefore only valid
  at the resolution it was captured at, and 2.1's scaling work must not change which rooms
  are composited.
- `HandleSparkleObject` (`Dynamics.c:304`) draws `RandomInt(240) + 60` every 60..299 frames,
  per emitter, forever.
- `HandleCoffee` (`Dynamics.c:492`, `:506`) draws `RandomInt(200)` every 100..299 frames per
  active coffee maker, forever.

So from 1.5c onward the RNG is consumed by *scenery*, at a rate set by what an author put in
the room, and not only by gameplay. 1.8's reference capture has to pin the house, the room,
the window size and the frame count together, and `glidertool replay`'s trace header should
carry all four. The alternative — giving the sparkles and the coffee maker their own
generator — would be a real deviation and is not proposed; it is recorded here as the escape
hatch if the demo replay proves impossible otherwise.

**1.8b establishes that the escape hatch is not needed, for this stream.** Demo House has no
sparkle and no coffee maker at all, so none of the three sites above fires during the shipped
replay — which is the house-specific half of R-RNG-2's proof. It stays written down because it
binds any *other* recording: a demo captured in a house with scenery that draws is only valid at
the window size it was captured at, and `glidertool replay`'s trace header carries the house, the
seed, the neighbourhood and the frame count for that reason.

**1.8c writes the table test, and finds that "the generator matches" was only a third of the
job.** `internal/game/rand_test.go` pins the seed-1 stream state by state against §1.6's 24
verified draws, pins `RandomInt`'s inclusive upper bound by constructing the raw word that reaches
it, pins its exact skew over all 65536 raw words, and shows that the `(16807*seed) % 2147483647`
an int32 transcription would naturally have written leaves the stream by the **third** draw — so
Schrage's split is load-bearing and not decoration. It also proves the containment from the other
side: `TestThePhysicsNeverDrawsFromTheRNG` walks `internal/game/player`'s AST and asserts the
package contains no call to `Random` or `RandomInt` at all, which is R-RNG-2's first step turned
into something that fails if a later stage breaks it.

The two-thirds that were not the generator, both of which a port gets wrong by default:

- **Scope.** `qd.randSeed` is a QuickDraw global, so in the original *one* stream spans every game
  in a process. A `World` seeded afresh per game replays the same "random" opening every time —
  the same candle phase, the same first telephone delay — which is the one way a fixed seed can be
  *less* faithful than a clock. `cmd/glidergo` now holds the stream on `app.randSeed` and reads it
  back out of the `World` after each game, the same hand-off `adoptScore` does for the music
  cursor.
- **The launch draw.** `VariableInit` (`InterfaceInit.c:160`, from `Main.c:321`) spends one draw
  before any game, on the editor's default flower. So the original's first game starts on **16807**,
  not 1, and `game.AdvanceRandSeed` is what puts the port on the same step. `-seed` now defaults to
  1 with `0` meaning "use the clock", which is §1.15's recommended knob and its other branch in one
  flag.

What is still open is unchanged and is only the one thing: whether Apple's trap really was
Park-Miller-by-Schrage, which needs a Mac. `ORIGINAL_GAME.md` §19.1 exception (e) is the written
form of that, together with the two draws the port deliberately does not make (`WriteOutPrefs`'s
conditional `fakeLong`, and `PourScreenOn`, which is dead code).

### 2.19 The mirror-room flame blink — **DONE as an opt-in fix, 1.7b; default is 2.x**

`DrawReflection` registers the *unclipped* reflected-glider rect with
`AddRectToBackRects`, so the erase covers the whole rect while the draw covered only the
part inside a mirror. Every frame, the work map outside the mirror is restored from a
background that never had a reflection in it — which is correct — but a candle flame or a
pendulum inside that rect is erased a frame early, because those register no back rects of
their own and rely on their own opaque redraw. In a mirror room with a flame, the flame
blinks.

The fix is to clip the back rect to the mirror rects, which also relieves 2.11 in exactly
the rooms that need it most. It is a visible change to what the original drew, so it is
opt-in first.

**Shipped in 1.7b as `fixes.mirror_flame`**, off by default. It registers one back rect per
mirror the reflection actually intersects, and none at all when the reflection is entirely
outside every mirror — the defect in its purest form, an erase with no draw behind it. Two
entries in a 47-slot list where the C had one is the cost, and it is still cheaper than what
the unclipped rect costs in erased flames. `game.TestMirrorFlameClipsTheBackRect` asserts
*both* states, because the unflagged path is what 1.8's corpus is recorded against.

### 2.20 A mirror shows the wrong player's foil — **DONE as an opt-in fix, 1.7b**

`DrawReflection` tests `if (showFoil)` with no `!twoPlayerGame` guard, unlike
`RenderGlider`'s. In a two-player game with foil showing, `glid2SrcMap` holds
`kGliderFoil2PictID` — the *other* player's foil sheet — so player 1's reflection is drawn
with player 2's artwork. Clearly unintended, and a player watching a mirror in a
two-player game does see it. A fidelity replay would diverge if it were fixed, so it is
kept and offered as an opt-in correction alongside 2.19 and 2.22.

**Shipped in 1.7b as `fixes.mirror_foil`**, off by default: it adds `RenderGlider`'s missing
`!twoPlayerGame` guard and nothing else. The test that matters is the one for the case the fix
does *not* apply to — `game.TestMirrorFoilChangesNothingInAOnePlayerGame` — because a
correction that turned the foil reflection off in the game almost everybody plays would be a
worse bug than the one it fixed, and it would pass any test that only asserted "the two flags
differ".

### 2.21 `doBackground` should not be a user option — **DONE as a split, 1.7b**

`doBackground` is a preference (`Main.c:186`, default **false**) that gates the entire
event pump: `PlayGame` only calls `HandlePlayEvent` when it is set. With it off the game
never pauses when it loses the foreground, never processes an update event, and — see 2.5
— never repaints after a dialogue. The original could survive that because the Toolbox
redrew a window's contents from its `WindowRecord`; a modern compositor cannot, and a
window that ignores its own close button is not shippable.

`cmd/glidergo` sets it true unconditionally, with the reasoning at the assignment. A
release should drop the preference entirely and always pause on focus loss. The one thing
to preserve is that the pump loop spins *mid-frame*, after both clocks have been bumped,
so every deactivation costs exactly one frame of animation phase — invisible, but it is
what the original did.

**1.7b split the flag in two instead of dropping it**, because it was doing two unrelated
jobs. `World.DoBackground` is the pump and stays true unconditionally — that is the port's
business and not a choice. `prefs.PauseWhenUnfocused` is the player's half, "keep playing while
switched out", and it gates only the `Suspend` call in the host's focus-loss arm. It defaults to
pausing and is deliberately **not** on the settings screen, for the reason this item gives; a
file that turns it off really does keep the game running in the background, which is where an
imported 1994 `doBackground` lands. The mid-frame spin is preserved, and 2.28 is what now draws
on the screen while it spins.

### 2.22 The two-player handshake has three bugs — **the deadlock is characterised, 1.9; the two fixes are open**

All three are in the limbo/transit handshake and all three are transcribed faithfully:

- **Mismatched-exit deadlock.** Two players leaving the same room by different exits can
  each end up waiting for the other. **Corrected in 1.9:** an earlier draft of this bullet said
  "with no key that breaks it", which is wrong — `ForceKillGlider` on Delete does break it, at the
  cost of a mortal, and it is *player 1's key only*, which is what makes the deadlock a real
  hazard rather than an inconvenience and is exactly what 2.23 now fixes. 1.9 also found the
  deadlock is wider than this bullet implied: two gliders in two *different transporters in the
  same room* deadlock as well, because the transit race demands the same physical object
  (`activeRectEscaped == index`, `Interactions.c:1381-1411`) and refuses the mismatch in total
  silence. `game.TestTransitRaceNeedsTheSameObjectNotJustTheSameKind` and
  `game.TestTheGiveUpKeyIsTheOnlyWayOutOfTheDeadlock` pin both halves, including 60 consecutive
  refused frames to show it never resolves itself.
- **The `takingTheStairs` leak.** The flag is set on a stair transit and is not always
  cleared, so a later transit in the same game can take the stairs path when it should not.
- **The shared `StillOver` edge detector.** One piece of state serves both gliders, so
  player 2 standing on a trigger can suppress player 1's edge.

A compatibility flag that defaults to the original's behaviour, with the fixes available,
is the right shape — the same shape 2.19 and 2.20 want. 1.9 shipped the *escape hatch* (2.23)
rather than a flag that makes either exit yield, because the deadlock's own fix has to choose
which player loses their exit and that is a design decision with no answer in the C; giving
both players the abandon key costs one mortal and needs no such choice.

### 2.23 Player 2 has no give-up key — **DONE as an opt-in fix, 1.9 (`fixes.player2_give_up`); the menus are still player 1's**

`Delete` abandons a glider waiting in limbo and it is player 1's key only; the menus are
player 1's too. So in a two-player game, player 2 cannot rescue a stuck situation, quit,
pause or change a setting. With the deadlock in 2.22 unfixed that is worse than an
asymmetry — and 1.9 established that Delete is the *only* way out of it, which makes the
asymmetry the difference between a recoverable game and a dead one.

**Shipped in 1.9 as `fixes.player2_give_up`**, the fourth opt-in fix and the odd one out among
them: 2.19, 2.20 and 2.39 correct something a player *sees*, this one corrects something a
player cannot *do*. It lives on `player.Input` rather than as a 47th `player.Env` method
(`World.GetInput` refreshes it from `w.Fix` each frame, because `NewWorld` does not take a
`Fixes`), and the host now reports `Delete` for player 2 unconditionally so that the *game*
decides whether it counts — one decision in one place. `ForceKillGlider`'s existing
`mode != GliderFadingOut` debounce is what keeps two players holding the key from spending two
mortals. Off by default like the rest, for 1.8's corpus reason.

**Still open: the menus.** Pause and quit remain player 1's, for a reason rather than an
oversight — the pause edge detector is a single variable, so a second poll of the same key would
pause twice — and Command has no equivalent on this host at all. Fixing it properly is
per-player edge state in the shell, which is 2.x work, not a `fixes` flag.

### 2.24 Two `CopyRect` helpers read from the screen — **note for whenever presentation becomes GPU-backed**

`CopyRectMainToWork` and `CopyRectMainToBack` read the screen surface as a source. That is
free today because `World.Main` is an ordinary `render.Surface` in system memory, and it
becomes a pipeline stall the day the screen is a texture. There are only two of them and
their callers are known; anyone moving presentation onto a GPU should redirect them to a
retained copy rather than read back.

### 2.25 `RedrawRoomLighting` redraws nine rooms where the original redraws one — **DONE, 1.5c**

An earlier draft of this entry said the lighting redraw "runs on a schedule rather than on
demand" and asked for a dirty flag. That was wrong on both counts, and is corrected here
rather than quietly deleted: the port's `RedrawRoomLighting`
(`internal/game/readylevel.go:139`) is already on-demand and already gated on the C's
`wasLit != isLit` transition. The real deviation is the *width* of the redraw, and the
function's own comment has recorded it accurately since 1.5b.

`RoomGraphics.c:448-460` recomposes the **central room only**, in six steps:

```c
DrawRoomBackground(localNumbers[kCentralRoom], kCentralRoom, roomV);
DrawARoomsObjects(kCentralRoom, true);          // <- the redraw flag
DrawLighting();
UpdateOutletsLighting(localNumbers[kCentralRoom], numLights);
if (numNeighbors > 3) DrawFloorSupport();
RestoreWorkMap();
```

The port calls `World.Rebuild()`, which is `DrawLocale` — all nine rooms, with
`redraw == false`. `redraw == true` is what suppresses the registration sites inside
`DrawARoomsObjects`, so the original repaints the pixels *without* re-creating the live
objects; `Rebuild` re-creates them, which deletes a band in flight, rebuilds the mirror
region and restarts a dinah mid-swoop. It is tolerable today only because no shipped room
with a light switch also has one of those things.

**Closed in 1.5c.** `Scene.RedrawCentralRoom` is `internal/render/locale.go:373` and holds
the six steps verbatim; `internal/game/readylevel.go:164` calls it in place of `Rebuild`,
keeping the recount, the work rect and the `ShadowVisible` recache on the game side exactly
where the C splits them. A band in flight, the mirror region and a dinah mid-swoop now all
survive a light switch. The paragraphs below are the analysis that got it there, kept because
they say *why* the narrow redraw is the correct one.

**Most of the fix already existed.** `internal/render/locale.go:510` is already
`func (s *Scene) DrawARoomsObjects(neighbor int, redraw bool)` with its twenty `!redraw`
gates in place, and `candleFlame`, `backUpToSavedMap` and `addGrease` all take the flag
through. `Scene.NumLights` (`locale.go:180`) is already the C's live per-room global,
reassigned before each of the nine draws with the central room last (`:283`, `:291`,
`:298`) — which is what makes `wasLit` read the central room's count. What is missing is a
`Scene.RedrawCentralRoom()` holding the six steps above, which is `DrawLocale`'s own
central-room tail (`locale.go:298-306`) with `redraw` flipped to `true` and one hook added.

So the remaining work was one small render method and a one-line change at the call site.
It was timed to 1.5c because step four, `UpdateOutletsLighting`, writes the `dinahs` table
that 1.5c creates — the entry could not have been closed before it, and there was no reason
to defer it past.

### 2.26 `phoneBitSet` means "no telephone" — **planned, Stage 5 (the editor)**

The flag is inverted with respect to its name: set means the room has *no* phone. Nothing
in the game is wrong — the port reads it the same way the original does — but an editor UI
with a checkbox labelled "phone" wired straight to this field would be backwards. Rename
it or invert it at the editor boundary.

### 2.27 Exposures were selected for and then dropped — **DONE, 1.5b**

The x11 backend asked for `ExposureMask` and had no `case C.Expose`, so the one event that
says "your window's pixels are gone, draw them again" was thrown away. It went unnoticed
because a compositing window manager retains window contents and replays them itself, and
because a running game overwrites the whole framebuffer thirty times a second anyway.

Neither of those covers the case that matters, which is a **paused** game: while
`switchedOut` is set, `PlayGame` presents nothing at all, so an obscured window stays
obscured for as long as the player leaves it. The original had the same requirement and the
same answer — `HandlePlayEvent`'s `updateEvt` arm calls `RefreshGameWindow` — and the port
now has the whole chain: `platform.EventExpose`, emitted by the backend for the last event
of an expose series (`count == 0`, so a multi-rect damage is one redraw), handled in
`cmd/glidergo`'s `PlayEvent` by calling `RefreshGameWindow`.

Any other backend owes the same event. A backend on a platform that guarantees retained
contents may legitimately never send one.

### 2.28 A paused game is indistinguishable from a hung game — **DONE, 1.7b**

Suspend-on-focus-loss is right (2.21) and it is also invisible: the window holds a frozen
frame with no overlay, no dimming and no text. A player who alt-tabs away and back sees
nothing wrong, but a player whose window manager quietly moves focus elsewhere sees a game
that has stopped responding, and there is nothing on screen to tell them that clicking the
window will resume it.

This bit during development in a form worth recording, because it is the shape the bug will
take for a player. `-frames 300` appeared to hang: the window opened, ran 34 frames, and
stopped. It was not a hang — this host's window manager returns focus to the terminal about
a second after a new window appears, and the game did exactly what it was told and paused.
Nothing on screen or in the terminal said so. `cmd/glidergo` now exempts timed runs
(`-frames`) from suspending at all, since a measurement or a replay has no user to pause
for, but that is a fix for the tooling and not for the player.

**1.7b gave it one, shared with `DoPause` as this item asked.** `pumpWhileSwitchedOut`
(`internal/game/pause.go`) is the C's `do { HandlePlayEvent(); } while (switchedOut)` with a
panel drawn every pass and the frame restored from the work map on the way out. Three details:

- **It borrows the placard's rect and its no-art panel, and neither picture.** Both PICTs name
  a key to press and the thing that ends this pause is not a key, so the panel says "click the
  window to resume" — the only place in the port that refuses the original's artwork on purpose.
  `game.TestTheSwitchedOutPanelIsNotThePausePlacard` is what stops that regressing to the
  placard, which would tell a player to press a key that does nothing.
- **Every pass, not once**, for the same reason the pause loop repaints: an expose answered
  mid-suspend copies the whole play area up from the work map and would wipe it.
- **A build with no `Present` hook gets the C's loop exactly** — no draw, no restore, no extra
  rect copy per frame in the one configuration that runs millions of them.

A dimmed frame is still not offered, and not for want of trying: 2.50 is why a 50% dim is not
available on an 8-bit indexed surface with no alpha.

### 2.29 The scoreboard had no font, so its three text panels were blank — **DONE, 1.5b; the metrics are still owed at 1.7**

`render.Scoreboard.Panel` used to clear a panel to `kGrayBackgroundColor` and drop the string
it was handed. So the band was on screen (2.9) and the four badges lit and blinked, but the
room name, the glider count and the score were three flat gray patches.

The reason was not laziness about glyph rasterising: the original's scoreboard text is the
Macintosh **application font** at 12 point bold, set by `TextFont(applFont)` three times in
`InitScoreboardMap` (`StructuresInit.c:109-133`). `applFont` resolves to Geneva on classic
Mac OS, which is a *system* resource — it is not in Glider PRO's resource fork, so there is
nothing for `glidertool` to extract and no way to be pixel-faithful here by transcription.
A font has to be authored, and authoring one is a different kind of work from porting one.

`internal/render/font.go` is that font: a 5×7 core in a 6×9 cell, ASCII 0x20..0x7F authored
as a sheet of `.` and `#`, wired into `Panel` at the pens the original uses — black at
(1,10), then white at (0,9), which is the order that makes the shadow fall down and right
instead of up and left. Coverage is deliberately wider than the original's problem: a house
is a user-supplied file whose room names are Mac Roman, so all 128 upper-half characters can
reach the title panel, and every one of them resolves — 27 accented lowercase forms by
composing a mark over a base letter, sixteen more authored outright (`…`, the dashes, the
bullet, `¢£§¶†‡◊π µ¬°`, the Apple logo), and the rest folded to the nearest ASCII the font
has (`©` → `(c)`, `Æ` → `AE`, `≤` → `<=`). `TestEveryMacRomanRuneIsRenderable` fails if a
character ever falls through to the hollow box.

**What is still owed, at 1.7.** It is not Geneva and it is not proportional: every character
advances six pixels, so a line is wider or narrower than the original's by whatever Geneva's
metrics differ, and there is no bold — the original's boldface is a one-pixel smear that at
this size would close every counter in the alphabet. Nothing shipped overflows a panel (the
widest of the corpus's 4,070 room names is 162 pixels of the title panel's 255, pinned by
`TestShippedTextFitsItsPanel`), and an authored house that does overflow will clip rather
than ellipsise. A taller face would also let the accented *capitals* keep their accents;
today they fold to the bare letter, because a seven-row capital reaches the top of the cell
and leaves nowhere to put the mark.

Two things became testable the moment it landed, and both now have tests in
`internal/game/scoreboard_test.go`: the glider-count clamp (`refreshNumGliders` clamps
`Mortals` at 0, `QuickGlidersRefresh` deliberately does not, and before the font both drew
the same nothing) and the score roll's intermediate numbers.

### 2.30 `JustRoomsRect` lives on the `View` and is written by game code — **note; revisit if a process ever runs two games**

`game.placeScoreboard` assigns `w.R.V.JustRoomsRect`, reaching through the Scene into the
View to do it. That is the original's structure — `justRoomsRect` is a global that
`AdjustScoreboardHeight` writes — and the port's plan document wanted the field on `World`
instead, where the rest of the per-game state lives.

It is left where it is because the alternative is worse today: `render.RenderFrame` and the
update-event blit both read it, so moving it to `World` means either passing it down through
the renderer or having two copies that can disagree. And there is exactly one game per
process, so the mutation happens once, before the first frame.

What makes it a note rather than a shrug: it is the one field on `View` that is not a pure
function of the screen size, so a second view — a resizable window (2.1), a split-screen
two-player mode, or a test that builds two worlds on one view — would find it stale. The
fix at that point is a `World.JustRoomsRect` plus a parameter on the two render entry
points, and it is cheap; it is only cheap *now* because nothing else writes it.

### 2.31 `DrawCalendar` drew no month — **DONE, 1.5b**

`ObjectDraw2.c:1132-1166` draws the calendar's month name in the same application font:
`GetTime` for the real month, `GetIndString(monthStr, kMonthStringID, month)` for its name
and `ColorText` at `left + (64 - StringWidth(monthStr))/2` to centre it. The port drew the
calendar's picture and stopped there, so a calendar in a room rendered as a blank one.

It was waiting on the font. The other two things it needed were already in place: `Scene.Clock`
exists and is already fixed to 14 October 1994 by the golden-corpus test, so nothing here has
to be nondeterministic, and the twelve month names are `STR# 1005`.

They are transcribed into `objectdraw2.go` rather than loaded, which is the choice
`palette.go` already makes for `'clut'` 128: the data is twelve words that have not changed
since 1994, a room's appearance should not depend on a second asset tree being present at
runtime, and `TestMonthNamesMatchTheResource` decodes the shipped resource and fails if the
copy drifts. When 1.7 needs `STR# 150`'s fifty-one strings the general decoder can be lifted
out of that test; fifty-one strings is where a loader starts paying for itself and twelve is
not.

It was a separate commit from the font because it moved the golden hash of 62 of the corpus's
4,070 rooms, and that diff is worth reading on its own rather than buried in a commit about
glyphs. One finding came out of it and is pinned by
`TestCalendarIsOnePixelNarrowerThanItsCentring`: the C centres the month in **64** pixels and
the calendar picture is **63** wide, so the original's own text sits a pixel right of centre.
The port keeps the 64.

### 2.32 Three things stop the world from inside a frame — **DONE: `DoPause` 1.7b, both banners 1.7d**

`DisplayStarsRemaining` (`Banner.c:205-236`) is the clearest case. It is called from
`Interactions.c:946`, inside the star-collection arm of `HandleInteraction` — so from the
*middle* of a frame, after the interaction pass and before the glider moves — and it draws
to the main window, then `DelayTicks(60)`, then `WaitForInputEvent(30)`. **Those two numbers
are in different units**: the first is 60 ticks and the second is 30 *seconds*, because
`WaitForInputEvent` multiplies its parameter by 60 (`Utilities.c:445`). So it is one second
during which the process does nothing at all — no repaint, no resize, no window close — and
then up to thirty more. `BringUpBanner` (`Banner.c:171-197`) does the same with
`WaitForInputEvent(15)`: fifteen seconds at the start of every house, four in a demo. `DoPause`
is the third.

Three separate problems, and only the first is cosmetic:

- **The window is dead while it blocks.** A modern compositor notices, and the desktop
  offers to kill the application. This is 2.5 and 2.28 in a different place, which is why
  they should be fixed as one thing: a *pause state* the frame loop knows about, so that
  presentation keeps running while the simulation does not.
- **Input in that window is swallowed.** `WaitForInputEvent` consumes the keystroke that
  dismisses it. So a player holding right when they touch a star is not holding right when
  play resumes — the glider stops. That is 1994 behaviour and it is bad behaviour, and it
  cannot be "fixed" inside Stage 1 without breaking the demo-replay comparison (1.8).
- **It is not reproducible headlessly.** A replay trace cannot represent "and then 90 ticks
  passed", so any script that collects a star would diverge the moment the pause is real.
  The stubs are empty today, which is precisely why the 1.5b traces are stable; that is a
  debt, not a property.

The port's two banners are `internal/game/banner.go` and `DoPause` is `internal/game/pause.go`,
and each carries a comment pointing here. The count they consume had to be a simulated one — not
a sleep — which is what `internal/game/wait.go` is.

**`DoPause` is done (1.7b) and it is the easy one**, because a pause has no duration to
reproduce: it lasts as long as the player holds it up, so there is nothing to convert into a
frame count. What it does establish is the shape the other two should take — the game draws and
hands the waiting to the host through a hook, and a nil hook means "do not wait at all", which
is exactly what a replay needs. The two banners *do* have durations — 60 ticks, and 15 seconds —
so they are the ones that needed the simulated form.

One consequence of the pause landing first: a fidelity trace can now contain the pause key
without hanging the harness, because `World.Pause` is nil in every headless build. That was the
first thing this item warned about and it is closed by construction rather than by care.

**1.7d closes the other two, and both game-over animations with them.** `internal/game/wait.go`
holds the arithmetic — `FlushEvents`, `DelayTicks`, `WaitForInputEvent`, and the two-tick
`pollInput` both endings pace themselves with — and `World.Wait(ticks, discard) Waited` is the
hook underneath all four. `cmd/glidergo`'s answer polls the window, keeps presenting, and honours
the deadline in two-tick slices; the headless one is nil, so a replay crosses every one of these
screens in zero simulated time and sees every pixel they draw. The three durations are asserted
as numbers in `internal/game/banner_test.go` and `gameover_test.go`, because the one way to get
this wrong with no visible symptom is the unit.

Two of the three problems above are therefore closed and the middle one is not. **Input during a
wait is still swallowed**, deliberately: a key pressed while the stars-remaining panel is up is
consumed by the wait and is not held when play resumes, so a player walking right into a star
stops walking. Correcting it means the panel no longer consumes the keystroke, which changes what
a recorded demo does next — so it waits for 1.8's replay corpus to exist to be checked against.

### 2.33 The port declines out-of-range reads the original performs — **DONE as a reported deviation, 1.5b**

The original indexes several arrays with values a house file controls, and does not check
them. The reachable one is the room-object slot: an unlinked transport has `objectLink` of
-1, which the `Byte` parameter turns into 255, and `masterObjects[...]`/`rooms[r].objects[...]`
are then read at that index. On a Mac that read something adjacent and carried on; in Go it
panics, which is not an option for a released game and also not a faithful one.

So `internal/game/guards.go` has exactly one guard, `badIndex`, and every caller reads
`if w.badIndex(kind, i, n) { return <what the C's garbage would most likely have been> }`.
Ten sites: `hotspots.go`, `objects.go`, `setstate.go`, `transit.go` (three), `triggers.go`
(four). Each refusal is counted in `World.Diag.Guarded` and the first sixteen *distinct*
kinds are sampled into `Diag.Seen`, so `glidertool replay` reports both, and a house that
trips one is diagnosable instead of merely surviving.

The two deliberate exclusions matter as much as the guard, because both look like
omissions:

- **`World.Room` is not guarded.** Its nil return is the *designed* answer to
  `GetNeighborRoomNumber` returning -1, which every `ReadyLevel` on an edge room asks for
  several times a frame. Counting it would put thousands of normal events in the same bucket
  as one real fault.
- **`internal/render/locale.go`'s three equivalents are not guarded.** They are the same
  shape, but they sit on `Scene`, which holds no game state and has no business holding a
  counter. If they ever need reporting, the answer is to give `Scene` its own `Diagnostics`,
  not to reach across the package boundary from `render` into `game`.

What is left is a policy question for 1.7: a guarded read means the house is malformed, and
right now nothing tells the *player* that. A house picker that ran `glidertool house
check`'s logic and refused to offer a broken house would be better than diagnosing it after
the fact — see 4.1.

### 2.34 One `PaintRect` in the Carbon conversion lost its destination — **note; binds 1.5c and 1.5e**

Glider PRO's shipped source is a Carbon conversion of the 1994 code, and the conversion
replaced QuickDraw's ambient current-port drawing with explicit `SetGWorld` pairs. Fifteen
`SetPort` calls were commented out in the process. Fourteen are harmless — twelve in
`ObjectDraw2.c` are the *last* statement of a function whose port was already restored by a
`SetGWorld(wasCPort, wasWorld)` two lines up.

The fifteenth is not. `Dynamics.c:577-578`:

```c
//			SetPort((GrafPtr)workSrcMap);
			PaintRect(&dinahs[who].dest);
```

`PaintRect` draws into whatever port is current, and the line that would have made that the
work map is commented out with nothing put in its place. Compare `Grease.c:105-118`, which
was converted correctly and is the model:

```c
GetGWorld(&wasCPort, &wasWorld);
SetGWorld(backSrcMap, nil);
PaintRect(&src);
SetGWorld(workSrcMap, nil);
PaintRect(&src);
AddRectToWorkRects(&src);
SetGWorld(wasCPort, wasWorld);
```

So the blank rect `HandleOutlet` paints lands in whichever GWorld the previous drawing call
happened to leave set, while the `AddRectToWorkRects` on the next line asserts it went to
the work map. The commented line names `workSrcMap`, so the 1994 behaviour is not in doubt;
only the shipped Carbon build's is.

*When* it paints was established while specifying 1.5c and is narrower than this entry
first said. The `PaintRect` is the `else` of
`if ((dinahs[who].position != 0) || (dinahs[who].hVel > 0))`, inside the
`position != 0` zap branch — and `hVel` is the room's light count
(`UpdateOutletsLighting`, `Trip.c:235-244`). So it is reached only on the **final frame of a
zap**, after that frame's `timer <= 0` arm has already cleared `position`, and only in a
room with no lights. Its purpose is to paint the socket out of a dark room rather than
leave `outletSrc[0]` showing. An outlet at rest never reaches this code.

**DONE — the outlet in 1.5c, grease in 1.5e.**
`internal/game/dynamics_appliances.go:427` is
`w.R.Work.Fill(w.Dinahs[who].Dest, render.Black8)` — the destination is named at the call
site, which is what the commented-out `SetPort((GrafPtr)workSrcMap)` was for, and the
`AddRectToWorkRects` on the next line is therefore telling the truth.

`HandleGrease`'s spreading arm (`internal/game/grease.go`) is now the model the C already
was, transcribed rather than repaired: `w.R.Back.Fill(src, ...)` then
`w.R.Work.Fill(src, ...)` then `w.AddRectToWorkRects(...)`, back map first as
`Grease.c:105-118` has it, all three naming the same `src`.
`TestSpreadingGreaseRegistersTheRectItPainted` asserts the registered rect *is* the filled
one, in screen coordinates with no conversion between them — which is the property the
outlet's version cannot have and the one this entry is about.

Transcribing the *bug* would have been untestable here for a reason worth stating: the port
has no ambient current port to leak into, so a missing destination is a compile error or an
obvious nil — the failure mode the C has is not available, which makes "be faithful"
meaningless and "be correct" the only reading. That is why this one is absent from 2.35's
transcribed-bug list.

### 2.35 Stage 1.5c transcribes four bugs the original shipped — **decided in the 1.5c specs**

Distinct from 2.33, which is about reads the port *declines*. These four are wrong but
harmless-to-run, they are visible in play, and 1.8's fidelity replays exist to hold them —
so all four go in as written, each with a comment naming the intended behaviour. Listed here
so that a future reader who spots one does not "fix" it, and so that if the project ever
ships an opt-in fidelity-vs-fixes switch (compare 2.19, 2.20, 2.22) this is the candidate
list. Full arguments in `docs/analysis/stage15-raw/plans/`.

- **`AddDynamicObject` clobbers the global `evenFrame`** (`Dynamics3.c:474`, `:524`). The
  neighbouring `lilFrame = true` is a *local* (`:191`); `evenFrame` is an extern (`:23`). The
  author seeded what he took to be two loop-local toggles and one name resolved to a global,
  so registering a ball or a fish resynchronises every flame and star in the locale on entry.
  `HandleBall`'s third write (`Dynamics2.c:420`) is the deliberate one these were copied
  from — it phase-locks the half-rate gravity so the reverse-engineered launch velocity
  reaches the height the author typed. Transcribing this changes what
  `internal/replay`'s `TestEvenFrameIsAStoredFlag` means, which is exactly what that test was
  written to catch.

  **Measured in 1.5c, and it is one frame wide, not permanent.** An earlier draft of this
  bullet — and the field comment on `World.EvenFrame` — said a ball desynchronises the locale
  for the rest of the room. It does not, because of a detail worth spelling out: the loop head
  (`internal/game/play.go:297`) is the only writer that *toggles*. All three others **assign
  `true`**, so a write forces a phase rather than flipping one, and a later write replaces an
  earlier one instead of compounding it. In every shipped ball room the composition write and
  the ball's first idle write (`dynamics_movers.go:452`) land one frame apart and cancel, which
  leaves exactly one diverging frame at composition time. Confirmed by replaying rooms 30
  "Ball Illusion", 172 "Attica, Greece" and 175 "Dodgeball" of CD Demo House against two
  ball-free controls: `replay.TestABallBreaksTheEvenFrameInvariant`, with the mechanism pinned
  by `game.TestBallResetsParityRatherThanTogglingIt`. A ball switched on *mid*-room by a
  trigger has no cancelling write, so there the shift does persist — no shipped house does it
  (all ten `kBall` lines in CD Demo House are `initial 1`), but a Stage 2 house could.
- **`ObjectDrawAll.c` writes `dynamicNum` from the wrong index** (`:518`, `:531`, `:544`,
  `:557`, `:570`, `:574`): `masterObjects[i].hotNum` with `i` the room-object *slot*, where
  the master index is `n`. In range but wrong, so no guard applies. Correct only for the
  central room, which is composited first with all 24 slots in order; for the other eight a
  trigger wired to a switch throws the central room's hot spot at that slot, or reads
  `hotSpots[-1]` — which *is* 2.33's territory and is refused there.
- **`HandleOutlet`'s two-player fan-out passes `doOffset` inconsistently**
  (`Dynamics.c:539`, `:541` versus `:545`, `:546`, `:550`). The outlet is one of the seven
  appliance types whose registration bakes `playOriginH/V` into `dest`, so the surviving
  player of a two-player game is tested against a screen-coordinate rect and the zap misses
  by the scroll offset. Zero in a room at the house origin, wrong everywhere else. The port
  writes that one fan-out longhand rather than through its shared helper, so the mismatch is
  visible at the call site instead of hidden behind a parameter.
- **`IsRectLeftOfRect` has an operator-precedence bug** (`RectUtils.c:185`):
  `(w1) - (w2) / 2` where the author meant `(w1 - w2) / 2`. With a 48px glider and 24px art
  the intended offset is -12 and the actual one is 0, so the shove direction is decided by
  left edges rather than centres and a dart clipping the glider's left side shoves it
  *into* the dart. One caller in the whole game, `Dynamics.c:53`.

### 2.36 A four-frame enemy reload would fire with no warning sparkle — **note; unreachable by arithmetic, and a Stage 5 editor could reach it**

`enemyWaiting` (`internal/game/dynamics_movers.go:121`) is the shared idle arm of the
balloon, the copter and the dart, and it announces a coming enemy with a sparkle and
`EnemyInSound` in two mutually exclusive places:

- at launch, `if Count < StartSparkle` — a reload so short there is no room to warn early, so
  the puff happens as the enemy appears;
- four frames before launch, `else if Timer == StartSparkle` — the normal case.

`StartSparkle` is 4, and the two conditions leave a hole at exactly `Count == 4`. `Count < 4`
is false, and `Timer` is reset *to* `Count` and then decremented before it is compared, so it
takes the values 3, 2, 1, 0 and never 4. An enemy on a four-frame reload would therefore
launch in complete silence, every time, with no puff — the one reload period the original
cannot announce.

**It is unreachable in any house that exists, and the reason is arithmetic rather than
authoring taste.** All four enemy arms compute `Count = (Delay * 6) / TicksPerFrame`
(`internal/game/dynamics.go:390`, `:415`, `:442`, `:482`) and `TicksPerFrame` is 2, so
`Count == Delay * 3` — always a multiple of three. 4 is not, so no value of the house file's
one-byte `Delay` can produce it. Verified over the whole reachable set by
`game.TestEnemyWaitingHasOneSilentPeriodAndItIsUnreachable`, which asserts both halves: that
`Count == 4` really is silent, and that no `Delay` yields it.

Recorded rather than fixed, and recorded rather than merely left alone, for one reason: the
Stage 5 editor will write `Count` if it ever grows a per-object override, and 1.9's
multiplayer tuning is the other place a frame count gets typed by hand. Anything that writes
a reload period in frames instead of in `Delay` units can land on 4 and produce a bug that
looks like a missing sound file. The test is the guard; this entry is why it exists.

### 2.37 The television's movie branch is a deliberate blank, and the movies are already extracted — **planned, 1.5e (decoder) / note (the divergence)**

`HandleTV` (`internal/game/dynamics_appliances.go:278`) always takes the non-QuickTime path,
which is the behaviour of a 1994 Mac without QuickTime installed: switching a set on blits
`tvScreen2`, the static "on" cel. On a machine that *had* QuickTime, and for a house that
shipped a movie, the C's active arm is an **empty `if`** — no blit and no reveal — because the
movie was expected to draw over that rect instead. The port cannot reach that arm even in
principle: `Room.TVMovieNumber` stays at the `-1` that `Rebuild` resets it to, so the
`who == tvWithMovieNumber` identity test is never true.

**The divergence is bounded and known, but it is not empty.** Nineteen of the 22 shipped
houses contain a television — `Titanic.house` has 13, `Land of Illusion.house` 10,
`Slumberland.house` and `SpacePods.house` 8 each — and **15 houses shipped a movie**, which
the extractor has already pulled out as 8-bit index buffers under `assets/extracted/movie/`
(see `manifest.tsv`: all 64×49 except Demo House's 82×62, 7 to 45 frames apiece, `raw`, `rle`
and `smc` codecs). So the missing feature is a decoder and a playback clock, not the assets.
Until then, a TV in one of those 15 houses shows a still frame where the original showed
video.

Four sites are already named no-ops waiting for it, deliberately kept as functions rather
than as comments so that 1.5e has one body each to fill and cannot fill three out of four:
`World.restartRoomMovie` (`internal/game/transit.go:718`), the `StopMovie` block in
`ReadyLevel` (`readylevel.go:44`), `ToggleTV`'s block (`trip.go:83`), and `HandleTV`'s own
arm. `World.TVOn` is correspondingly never written, which is the correct transcription and
not an omission — on a machine with no movie the C never enters the block that writes it
either.

Two things to get right when the decoder lands. The `raw`/`rle`/`smc` triple means three
decompressors, and `smc` is the only one that is not trivial. And `TVOn` is not reset on a
room change in the original, so a set switched on in one room restarts the movie in every
later room that has one — that is 1994 behaviour to reproduce, not a bug to fix, and it is
the field comment on `World.TVOn` that says so.

### 2.38 A switch wired to a star makes a house impossible to finish — **fix planned, Stage 2 (needed before new houses ship), its shape now a decision; the linter check, left out of 4.1, planned next**

`switchLinkedObject` (`internal/game/switches.go`) groups `kStar` with the eight other
prizes, so a star removed by a switch gets the background restore and the puff of light and
**neither `StopStar` nor the `StarsLeft--`** that `HandleRewards`' own star arm performs. Two
consequences, and the second is not cosmetic:

- the star's six-cel spin keeps animating over the background that was just put back, so the
  room shows a rotating star with no star under it;
- the star object is gone, so nothing can ever collect it, and `StarsLeft` never reaches zero.
  **The house cannot be won.** There is no other decrement anywhere in the game.

`game.TestSwitchOnAStarStrandsTheHouse` pins it in both halves — the count is unchanged, and
a second attempt to collect the removed star also changes nothing.

**No shipped house does it.** `game.TestShippedHousesWireSwitchesToPrizes` walks all 1,563
switch and trigger plates in the 22 original houses, resolves each link the way
`GetRoomLinked` does, and finds 165 links to a bonus object — 146 of them from a switch rather
than a pressure plate — and **zero to a star**. So this is not a defect a player of the
originals can reach; it is a trap for anything that authors new content, which is exactly what
Stage 2 and Stage 5 are.

(The link has to be resolved through `GetRoomLinked` and not read as a room index, which is
what the first version of this survey did. `data.e.where` is a packed floor/suite pair — 6709
is floor 26, suite 53 — so reading it as an index resolves nothing and reports a clean corpus.
The test now fails outright if no link resolves, so the same mistake cannot produce a
reassuring answer twice.)

Three actions, in the order they are needed:

1. **Stage 2, before the first new house ships:** fix it. The star arm needs its own case with
   `StopStar` and the decrement, matching `HandleRewards`. Since no shipped house reaches the
   arm, the fix cannot change any existing house's behaviour — which makes this the rare
   reproduced bug that costs nothing to correct, and is why it is not on 2.35's leave-alone
   list. The fidelity replays (1.8) will not notice, for the same reason.
2. **4.1, the house linter:** an unreachable star is only one way to make a house
   unwinnable; a linter that already walks the object graph should report any switch or
   trigger link that can remove a star, and should do it whether or not the runtime is fixed.
3. **Stage 5, the editor:** the link picker should not offer a star as a switch target at all.

**The linter half, which 4.1 was marked DONE without.** `link-removes-star`, an error in `link()`
(`internal/house/lint.go`) with its `LintChecks` entry. It flags any of the six *switch* types
(`kLightSwitch`, `kMachineSwitch`, `kThermostat`, `kPowerSwitch`, `kKnifeSwitch`, `kInvisSwitch`)
whose link resolves to a `kStar`, local or remote. Triggers are excluded. A trigger wired straight
to a star is inert (`FireTrigger`'s local arm falls to its default, and its remote half handles
only grease), so flagging it would be a false error. A trigger that throws a switch wired to a star
is caught by that switch's own link. Zero shipped links fire it, so `make levels` and CI gate new
houses on it at no cost.

**The runtime half is a decision, and the recipe above misses a case.** A switch whose star is
outside the nine-room locale (`LocalLink -1`) never reaches `switchLinkedObject`. `SetObjectState`
clears the star in the house copy and `StarsLeft` stays one short (reproduced). So a fix keys on
`SetObjectState` having cleared a star, in `HandleSwitches` whatever `linkIndex` is, with
`StopStar` only when the star is local.

"`StarsLeft--`" is also not `HandleRewards`, which pays `StarPoints` and calls `FlagGameOver` at
zero (else `DisplayStarsRemaining`). There are two options:
- **The switch "collects" the star**, running the whole tail, so the last star wins the house.
- **The switch refuses a star target** and the star stays collectible. This matches the 1994
  editor, which never offers a star as a switch target (`Link.c:82-91`).

Third-party 1990s houses loaded with `-levels` could plausibly reach this, because `DeleteObject`
leaves forward links at an emptied slot (unverified). So the runtime half is a
`fixes.switch_star` opt-in, and its bit belongs in 4.31's rules byte.

The comment above `TestSwitchOnAStarStrandsTheHouse` ("fixing it changes what a shipped house
does") is contradicted by the corpus survey and is rewritten.

### 2.39 The switch's spurious corner sparkle fires 145 times in the shipped houses — **DONE as an opt-in fix, 1.7b**

`HandleSwitches` declares `bounds` and never assigns it before the prize arm's `AddSparkle`
reads it (`Interactions.c:995` against `:1043`). On 68k that was whatever was on the stack; in
Go it is the zero rect, so the puff lands at the top-left corner of the play area rather than
somewhere unpredictable. The sparkle the player is *meant* to see comes from
`RestoreFromSavedMap`'s own `doSparkle` arm one line earlier and is correctly placed on the
prize, so this second one is pure noise — nothing reads it, nothing depends on it, and
removing it changes only what is on screen.

**It is not latent.** The same corpus survey counts **145 switch links into the prize arm**
across the 22 original houses — 48 to paper, 40 to foil, 23 to a yellow clock, 12 to helium,
and so on — so a player of the shipped content sees a stray flash at the corner of the room
every time one of those switches is thrown. It is the sort of thing that reads as a rendering
bug in the port rather than as fidelity to 1994.

Kept for now, because 1.8's fidelity replays compare sparkle tables and this is a real
difference in them. Removed behind the same modern-options flag as 2.19, 2.20 and 2.22, where
one line deletes it. `game.TestTheSpuriousSparkleLandsAtTheCorner` asserts the current
behaviour, including that the puff is at the origin rather than merely somewhere.

**Shipped in 1.7b as `fixes.switch_sparkle`**, off by default, and it is one `if`. The tested
property is that it drops *exactly* the second puff and keeps the one on the prize
(`game.TestSwitchSparkleDropsTheSpuriousPuffAndKeepsTheRealOne`): a switch that removed a prize
with no puff of light at all would be a worse outcome than the stray flash, because the puff is
how a player learns that something they were not looking at has just vanished. It also frees a
slot in a three-entry table that silently drops the fourth request, which is the half of the
argument that is not cosmetic.

### 2.40 Three of the twenty-three switch arms have never run in any build — **note; matters to Stage 5's editor**

The second dispatch in `HandleSwitches` runs only when `SetObjectState` returned true, and
`SetObjectState` returns false for `kSlider`, `kSoundTrigger` and `kGuitar`. So all three arms
are dead code in the original as well as in the port, and `game.TestThreeSwitchArmsCannotBeReached`
pins the fact against a future change to `SetObjectState` bringing one to life untested.

`kSlider` and `kGuitar` cost nothing: the slider arm is empty anyway, and the author's own
comment on the guitar is "really no point to change this state". `kSoundTrigger` is the loss.
It sits in `SetObjectState`'s *switch* family, which returns false because a switch has no
state of its own — so **a switch wired to a sound trigger has never played a sound**, in 1994
or now, however plainly `switchLinkedObject` says it should. `FireTrigger`'s own sound-trigger
arm does work, which is presumably why nobody noticed.

Not fixed, because making it work would give existing wiring in the shipped houses a new
audible effect, and because the custom sound loader it wants is 1.6. Recorded because the
Stage 5 editor should not present a link the game cannot honour: either the picker omits
sound triggers as switch targets, or 1.6 makes the arm reachable on purpose and the fidelity
replays get a note.

### 2.41 The rubber band debounce does not debounce — **note; three separate defeats, all transcribed**

`bandHitLast` (`RubberBands.c:29`) is meant to stop a band resting against a switch from
toggling it thirty times a second. It is a single global holding a single hot-spot index, and
it fails in three independent ways. All three are reachable in the shipped houses, all three
are reproduced, and each has a test in `internal/game/bands_test.go`.

**One band, two rects.** The debounce suppresses the *effect* and not the sweep, so after
tripping hot spot 5 the loop carries on and hot spot 9 sees `bandHitLast == 5`, trips too, and
leaves `bandHitLast == 9`. Next frame hot spot 5 trips again because the latch now names 9.
Two switches within sixteen pixels of each other toggle each other's latch for ever.
`TestTwoRectsDefeatTheDebounceForEachOther`.

**Two bands.** The reset is per band, not per frame: `CheckBandCollision` ends with
`if (!nothingCollided) …` else clearing the latch to -1, so a *second* band in free flight
clears it every frame and the first band's latch never survives. This is the easy one to hit
deliberately — fire twice, hold one band against a switch — and it is a straightforward way
for a player to farm a switched prize. `TestSecondBandDefeatsTheDebounce`.

**The initial value.** It is a zeroed global rather than -1, and `KillAllBands` does not reset
it, so the first band collision of a session is silently *swallowed* if it happens to be with
hot spot 0. Go's zero value reproduces that for free, which is why `World.BandHitLast` is
deliberately left uninitialised. `TestFirstCollisionWithHotSpotZeroIsSwallowed` and
`TestKillAllBandsLeavesTheDebounceLatched`.

Not fixed at 1.5e, for the usual reason: the effect is visible in play, so 1.8's fidelity
replays have to hold it before it can move. The shape of a fix is not in doubt though, and it
is smaller than the bug — a per-hot-spot `bandStillOver` flag cleared by the same sweep that
clears the glider's `StillOver`, which is what `HandleSwitches` already has and what the band
path borrows without getting its own. That makes it the same fix as giving bands their own edge
detector, and it belongs with 2.19/2.20/2.22/2.39 behind the modern-options flag rather than as
a silent correction, because the two-band defeat is exploitable and someone's route through a
house may depend on it.

One consequence worth separating out, because it is a *design* decision and not a defect: only
five of the 28 hot-spot actions are offered to a band, and `kRewardIt` is filtered to grease
alone. **Bands cannot collect prizes.** Without that filter a player could farm a room's clocks
from across it. `TestOnlyFiveActionsSeeABand` pins the list in both directions, driven off
`NumHotSpotActions` so that a twenty-ninth action added without a decision about bands fails
there rather than in a house.

### 2.42 The random stream is a function of the play area's size — **note; binds 2.1, 2.8 and Stage 3**

Five of the room-load registrations end by seeding an animation's phase from `RandomInt`: the
candle, the torch and the barbecue pick a starting cel, the star picks one of six, and the
pendulum flips a coin for its direction. In all five the draw sits **after** the
`if (savedNum == -1) return;` — verified one function at a time in `DynamicMaps.c` — so an
object that fails to get a saved-map slot does not consume its draw either.

Two things then follow, and the second is the one that matters.

**Registration is screen-size dependent.** Every one of the five `Add*` functions is called
from `ObjectDrawAll.c` only for objects that `SectRect` against the composed area, so *which*
objects register — and therefore how many `RandomInt` draws a room load performs, and therefore
every subsequent value in the stream — is a function of how much of the house is on screen.
That is `Scene.NumNeighbors` (1, 3 or 9) and the size of the play area.

**Saturation would do the same thing, and no shipped house reaches it.** 1.5f's census
measured it rather than assuming: over 4,070 rooms in the 22 houses the busiest locale claims
16 of the 24 slots, and none drops a registration (see 2.44). So the saturation half of this is
a live code path that shipped content never takes, exactly as `AddDynamicObject`'s cap turned
out to be in 1.5c — but the `SectRect` half is reachable by anyone who changes the window.

What protects it is a rule already written down for another reason: **2.8, the scale transform
belongs at the present step and nowhere else.** Composing 640x480 and scaling on the way to the
display keeps the play area fixed at every window size, so the stream is untouched by 2.1's
scaling and fullscreen work. This item is the reason that rule is not merely tidy.

Two consequences to carry forward:

- a replay script is only valid at the play area it was recorded at. `neighbors` is in the
  script format already; the play-area size is not, because there is currently only one. If a
  resolution option is ever added that changes what is *composed* rather than what is
  presented, the script format needs a line for it and `internal/replay` needs to refuse a
  mismatch rather than silently produce a different run.
- Stage 3's race gives each machine its own world, so two machines drawing different flame
  phases is cosmetic. It stops being cosmetic the moment anything is shared across the wire
  that was derived from the stream — a race seed, a shared house, a spectator view — so the
  handshake should carry the play area and the neighbour count and refuse a mismatch.

Pinned by `TestASaturatedTableDoesNotConsumeARandomDraw` (`internal/render/anim_test.go`),
which fills the table and asserts the draw is *not* consumed, and by the `rand` column of the
replay trace, which is where a shifted stream shows up as a whole-column diff.

### 2.43 `AddAShreddedGlider` writes one element past its table — **DONE as a reported deviation, 1.5f**

`DynamicMaps.c:726` guards with `if (numShredded > kMaxShredded) return;` — strictly greater,
against a table of exactly `kMaxShredded` elements (`shreds` is a `NewPtr` of
`sizeof(shredType) * kMaxShredded`, `Environ.c:654`). So a fifth shredded glider passes the
guard, writes `shreds[4]` of a four-element allocation, and leaves the counter at 5. On a 1994
Mac that scribbled twenty bytes over whatever the Memory Manager put next.

It is more reachable than the numbers suggest. A spent cloud keeps its slot — neither arm of
`RenderShreds` matches at frame 20, so it stops animating and is never reclaimed — and
`RemoveShreds` removes exactly one entry per death and **none at all** if the only cloud is
still growing. So four shreds in one locale visit fills the table permanently, and two-player
mode halves the work.

The port tests `>=` and drops the fifth cloud. That is the one deliberate divergence in
`internal/game/shreds.go`, for 2.33's reason: reproducing it would mean reproducing a bug whose
observable behaviour is "corrupt something else", which is not a behaviour a port can be
faithful to and not one a released game may have.

What is new at 1.5f is that the refusal is **reported**. The guard goes through `badIndex`, so
it lands in `World.Diag.Guarded` and `Diag.Seen` and is named by `glidertool replay` like any
other deviation. It is the only site in the port where that guard stands in front of a *write*
rather than a read, which is why it has its own kind (`devShred`, `guards.go`) — from a bug
report's point of view the two are the same finding, that a house reached a place the original
survived by luck. `TestTheFifthCloudIsDroppedRatherThanWrittenOutOfBounds`.

### 2.44 A refused saved-map registration makes an object invisible, silently — **the report DONE, 1.5f; the author-facing half is Stage 5**

`backUpToSavedMap` answers -1 when the 24-slot `savedMaps` table is full, and **every caller
gates the object's draw on the result.** So the twenty-fifth animated or collectable object in
a locale is not merely un-animated, it is not drawn at all: a candle with no flame, a prize that
cannot be seen and can still be collected by walking into it. The original reports nothing. A
house author found out by playing the room and noticing something missing.

`Scene.SavedMapDrops` records each refusal — the room, the object slot and the rect it asked
for, which names the family without a label because no two request the same size. It is a
report and nothing reads it, on the same terms as `World.Diag`: a diagnostic that could change
the composition would be worse than none.

The census that came with it is the useful half, and it contradicts what `docs/PLAN.md`
assumed. Over **4,070 rooms in 22 houses the busiest locale claims 16 of the 24 slots**
(`Slumberland` room 256 "Flaming Pathway", tied by `Teddy World` room 322) and **not one room
drops a registration**. So the cap is a live path that shipped content never takes — the same
answer 1.5c got for the 18-slot dinahs table, except that three shipped locales *reach* that
one and none comes within eight of this one.

Two things follow. A Stage 2 house has to be **checked** against the cap rather than assumed
under it, which makes this the third item on 4.1's linter list (with 2.38 and the dinahs cap).
And Stage 5's editor can say it while the room is being built, which is the whole difference
between a diagnostic and a tool. Watched corpus-wide by the `dr=` column of
`internal/render/testdata/locale_golden.txt`, room by room, and asserted by
`TestTheSavedMapBudgetSaturatesInShippedContent`, which fails if any shipped room ever starts
dropping; `TestDroppedRegistrationsNameTheObjectThatVanished` fills the table by hand and
enumerates the three refusals in order.

### 2.45 The loop points in the sound headers are ignored — **note; the samples are written to be retriggered**

Every `'snd '` header carries `loopStart` and `loopEnd`, and the port reads neither. That looks
like an omission and is not: the original does not loop either — it issues `bufferCmd` and a
`callBackCmd`, and a `bufferCmd` plays a buffer once — and the four sounds that need to be
continuous are *composed* to be retriggered instead. `Hiss` is 2960 samples, which is four game
frames to the last sample (4 × 740), and `Input.c:174` asks for it every fourth frame while the
helium key is held, so one channel plays it back to back with no seam. `Sizzle` is three frames
long and is asked for every frame, so it occupies three channels and covers itself the same way.
`Thrust` and `Shred` are cut slightly *short* of their cadence, so their copies overlap and
thicken.

Looping would therefore be a bug rather than a feature. A looped `Thrust` would keep firing after
the player let go of the key, and — worse — its channel's completion callback would never run, so
`priorityN` would never fall back to 0 and the channel would be lost for the rest of the session:
2.47's failure, arrived at from the other direction. The table is checked against the extracted
samples by `TestContinuousSoundsCoverTheirCadence`, so the argument fails loudly if a future
extraction disagrees with it.

### 2.46 The mix is labelled 22255 Hz and the samples are 22254.5454… Hz — **note; 35 ppm, accepted deliberately**

The Macintosh rate is the exact fraction 244800/11 Hz, which is what the resource headers carry
(`Fixed 0x56EE8BA3`). The port labels its output stream with the rounded integer, because every
audio API worth writing to — WAV headers included — takes an integer rate, and it does not
resample to get there.

The error is 0.4545 Hz in 22254.5, which is 35 parts per million, or 0.06 cents of pitch: four
hundred times smaller than the smallest pitch difference a human ear can detect, and about a
twentieth of a sample's drift per second. The alternative is a rational resampler on every sample
in the bank — either at load time, which loses information for no audible gain, or in the mixer,
which puts an interpolator in the hot path of a mono 22 kHz mix to correct a 0.06-cent error. It
is not worth its complexity. What *is* worth doing is the per-sound conversion the houses need
(nine distinct rates occur across their 58 usable sounds), and that is implemented: `Sound.Step`
is a 16.16 fixed-point increment and `stepFor` rounds it so that every sound authored at any of
the three spellings of the Macintosh rate gets exactly `FixedOne` and is copied rather than
resampled. Drop-sample and not interpolating is the original's own choice, not a shortcut —
`Sound.c:379` and `Music.c:287` create all four channels with `initNoInterp`.

### 2.47 `FlushAnyTriggerPlaying` silences the game permanently — **DONE as a reported deviation, 1.6**

`FlushAnyTriggerPlaying` (`Sound.c:262-273`) stops a trigger sound that is still playing when the
player leaves the room. It issues `quietCmd` to stop the sample and `flushCmd` to empty the
channel's command queue — and the `callBackCmd` that `PlaySoundN` queued behind the `bufferCmd`
is *in* that queue. Flushing it means the completion callback never runs, and the callback is the
only thing in the program that ever writes `priorityN` back to 0. The channel is left claiming
`kTriggerPriority`, which is 999, for the rest of the session: `InitSound` runs once, at launch
(`Main.c:337`), and nothing else resets it.

One interrupted trigger sound costs a channel. **Three of them leave `PlayPrioritySound` with
`lowestPriority == 999`, which refuses every ordinary request, while the trigger-exclusivity rule
refuses every trigger. The 1994 game goes silent and stays silent until it is relaunched.**
Reaching it needs a house with a sound trigger, a sound long enough to still be playing on the way
out of the room, and a player who leaves — which is a description of `Art Museum`, whose
"Security" is 2.2 seconds and whose triggers are in corridors.

The port quiets the channel *and* resets the three fields, as though the callback had run. Two
things make that safe rather than presumptuous: nothing in the simulation reads channel state, so
no replay and no digest can move; and the deviation can only ever make the port louder than the
original, never quieter, so it cannot hide a sound the 1994 build played. There is also one piece
of evidence that the author knew something was wrong here — the call to `FlushAnyTriggerPlaying`
inside `LoadTriggerSound` is commented out (`Sound.c:275`).

### 2.48 The audio sink is a subprocess, not a device — **DONE on Windows: a native `waveOut` sink, in the binary. The subprocess stays the Linux answer and the fallback everywhere**

For most of the port there was no audio driver at all. `audio.Pipe` writes raw PCM to `pw-play`,
`paplay`, `aplay`, `ffplay` or `play`, whichever is installed, and `audio.WAV` writes a file — and
on Linux that is still what happens, deliberately. That began as a
constraint — the build host is airgapped, so there is no `golang.org/x/sys`, no `oto` and no
`ebiten`, and ALSA or PulseAudio means cgo — but it is a defensible answer on Linux on its own
merits: every desktop ships at least one of those five, they all read s16le mono on stdin, and a
player that crashes takes nothing with it, which is more than can be said for a cgo audio callback
inside the game's address space.

The cost is honest: one process, one pipe, and a latency the port does not choose, because the
player picks its own buffer size. A sound is heard 50 to 200 ms after the frame that asked for it.
Nothing in Glider PRO depends on sub-frame audio timing, so this is fine — but it is fine *on
Linux*. **Windows has no such tool in the box, and this is the one part of the audio path that
Stage 4 cannot simply cross-compile.** `Sink` is two methods wide precisely so that a native
driver can be dropped in behind it without the engine knowing; WASAPI via `syscall` (no cgo, and
no dependency the airgap forbids) is the likely shape, and CoreAudio at Stage 6 is the same
problem again.

Since the win32 backend landed this stopped being a future problem and became a shipping one: two
of the six release archives are Windows archives that draw, and they are silent by default. Three
things were done rather than left:

- `OpenPipe`'s not-found error no longer recites five Linux sound stacks at a Windows reader.
  `installHint` (`internal/audio/sink.go`) branches on `runtime.GOOS` and names the two players
  that actually have Windows builds — FFmpeg's `ffplay` and SoX's `play`. `exec.LookPath` finds
  `ffplay.exe` from the bare name because Windows consults `PATHEXT`, so **a Windows machine with
  either on its `PATH` has sound**, which is a better answer than "none" and was worth finding out
  before writing "silence" in a release note. Whether either then plays raw PCM on stdin there is
  untested, like everything else Windows in this port.
- The release notes and each Windows archive's `HOW-TO-RUN.txt` state it up front, alongside
  `-wav out.wav` as the way to hear what you missed.
- The README's audio paragraph says which platform its five-player answer is good on.

**Then the sink itself was written, because a documented limitation is still a game that makes no
noise.** `internal/audio/waveout_windows.go` is a `waveOut` driver in pure `syscall`: mono s16le at
22255 Hz through `WAVE_MAPPER`, eight blocks in rotation, `WHDR_DONE` polled every 4 ms by one
goroutine, and a tenth of a second of silence written ahead of the first mixed sample so the device
does not starve on the first hiccup. `internal/audio/device.go` is the seam it arrives through —
`Open` tries the native device first and the external players second, and nothing above it learns
which it got, exactly as `internal/platform/backend` does for the display. `-audio list` now says
`waveout` first on a Windows machine, `-audio waveout` insists on it, and the shutdown report gained
a fourth counter (`gaps`) that only a real device can produce.

Four things about it are worth knowing:

- **It has never run here.** Nothing Windows in this port has (see `internal/platform/win32`), so
  the file carries the same caveat at the top and the two ways out stay wired: `-audio ffplay` still
  takes the subprocess path and `-wav out.wav` still writes a file. What *is* checked from here is
  the half where a mistake would be silent — `waveout.go` holds the two structure layouts, the
  constants and the error table with no build tag, and `waveout_test.go` asserts every WAVEHDR and
  WAVEFORMATEX offset, the C structure's 18 declared bytes inside Go's 20, and the tuning numbers
  against each other, on every platform in `go test ./...`. The rest is checked by somebody else's
  machine: `waveout_windows_test.go` runs in CI's `native` job, where it loads winmm out of the
  system directory and resolves all seven entry points for real, and on any machine that has a sound
  card it also opens the device and plays twelve frames of silence through the full rotation. A
  hosted runner has no card, so the first test carries CI and the second skips there — which means a
  misspelt export or a mishandled path length is a red X on the next push, and only the playback
  path is still waiting for a human with a Windows desktop.
- **`waveOut` rather than WASAPI**, which is five COM interfaces and a render thread against seven
  plain functions and a struct, in code nobody here can run. On Windows 10 and 11 `waveOut` is a
  shim over WASAPI in shared mode anyway, so the game mixes with the rest of the desktop.
- **winmm is loaded by absolute path.** It is not a KnownDLL and it is not in Go's own system-DLL
  set, so a bare `LoadLibrary("winmm.dll")` would search the directory the executable was started
  from first — and a release archive tells the player to run the binary from the directory they
  unpacked it into, which is exactly the arrangement a planted DLL needs. `GetSystemDirectoryW`
  answers first.
- **The blocks are package-level arrays**, because `waveOutWrite` keeps the pointer it is given and
  Go's rules for pointers handed to foreign code do not cover that. A global is the one allocation
  whose address the language cannot ever change. `runtime.Pinner` was the first draft and is worse:
  a Pinner collected without `Unpin` panics the process by design, so a bookkeeping slip becomes a
  crash in a shipped game. `VirtualAlloc` was the second and is also worse: reading it back needs a
  pointer made out of an integer, which is what `go vet`'s `unsafeptr` check exists to object to.

CoreAudio at Stage 6 is the same problem again, and the seam is now in place for it.

### 2.49 Five house sounds ship as silence — **the linter says so as of 2.0; the decoder is still blocked on data**

Five of the 63 `'snd '` resources in the shipped houses are MACE 6:1 compressed (`cmpSH`,
`compressionID` 4): `CD Demo House` 3007 "Door Chime", `Demo House` 3011 "Meow",
`Nemo's Market` 3001 "Door Chime", 3003 "Cash Register" and 3004 "Cat Meow". The quantisation
tables that would decode them live in the Sound Manager and are in no file in this tree, so the
extractor records them as explicit skips and the port plays nothing.

That is not merely a missing noise. A sound trigger whose sound does not load gets **no hot spot
at all** (`game/hotspots.go`), which is the C's behaviour too — so those five rooms compose
slightly differently from how their authors saw them, and `Demo House`'s only custom sound is one
of the five. The port is faithful here by accident rather than by choice, which is the reason to
write it down.

Two ways out, neither urgent: take the published MACE tables from an open-source Mac audio decoder
(`audio.md` OQ3), or re-record five short sounds — which is a content change and belongs with the
Stage 2 houses, not in Stage 1. What is owed sooner is that the Stage 2 linter (4.1) already has
"custom trigger sounds that do not load" on its list from 2.14, and these five are the corpus it
should be tested against: a house author whose sound is silently dropped should be told at build
time, not by playing the room.

**2.0 paid that debt, and these five are the reason the check is two checks.** Writing `sound-id`
— "the house names a `snd ` it does not carry" — produced 20 findings over the corpus, and
splitting them against `assets/extracted/sound/houses/manifest.tsv` showed that 7 of the 20 were
these five resources rather than anything an author did. Reporting those as an authoring defect
would be worse than not reporting them at all, because the author would go looking for a mistake
they did not make. So the linter has `sound-unreadable` as a separate check whose text says out
loud that this one is ours and that the sound played on a 1994 Mac, `audio.Bank.Unreadable` carries
the extractor's `status` column through so a tool can tell the two apart, and `house.SoundStatus`
has three answers rather than two. The remaining 13 `sound-id` findings are all genuine: In The
Mirror names `snd ` 10000 twice, and Teddy World names `snd ` 3000 eleven times while carrying no
`snd ` resources at all.

What is left here is only the decoder. MACE 6:1 is a published fixed-rate ADPCM and would live in
`tools/probe_snd.py` rather than in the game, which reads PCM out of the extracted tree and should
stay a consumer of it. The cost is not the decoder: `make assets-check` holds the extracted tree
byte-for-byte, so closing this is a decoder *plus* a regenerated and re-reviewed asset commit, to
buy five effects in three houses nobody has complained about. Worth doing before a 1.0; not worth
doing in front of Stage 2's own content.

---

### 2.50 A 50% dim is not a panel, on a surface with no alpha — **DONE as a decision, 1.7a**

There is no way to draw a translucent panel over the splash illustration, because
`render.Surface` is 8-bit indexed with the 1994 palette and has no channel to be translucent
in. What the original does everywhere it puts something over existing pixels is
`PenPat(gray)` with `PenMode(patOr)` — a 50% checkerboard ORed into the destination
(`render.Surface.FillPatOrGray`), which turns half the pixels black, leaves the other half
alone, and cost nothing on a 68k.

**It is not sufficient for text.** The first version of the menu panel was exactly that dim,
and it was very nearly illegible: the shipped splash art is a bright yellow wall, so cream
letters sat on a checkerboard of black and bright yellow whose light half is as light as the
letters. The inverse-video selection bar on the same panel was perfectly readable, which is
what identified the cause — a solid ground, not a brighter ink.

So panels are a solid `Black8` interior with a cream frame, and the dim survives as an
eight-pixel halo around the box, which keeps the frame from sitting on the picture with a hard
edge. That is also the game's own idiom rather than an invention: the scoreboard fills its band
black and writes cream in it (`render/scoreboard.go`). The general rule, for 1.7b's settings
screen and 1.7c's boards: **the dim is for de-emphasising artwork, never for backing text.**

Two consequences worth writing down. Anything drawn over the splash has to be *placed* rather
than composited, so every panel's rectangle is a named constant chosen against the shipped art
— which is why the menu sits over the sky at the right rather than over the illustration's
subject, and why the port's own fallback title screen is laid out around that same rectangle.
And a magnified bitmap font is the only text this port has, so legibility is bought with
`scale` and the original's one-pixel black drop shadow, not with a lighter weight.

### 2.51 A disabled control under the cursor must not look like the one to press — **DONE, 1.7a**

The original's answer to an unavailable command is a menu item drawn in grey and a keystroke
that does nothing at all, which leaves a player with no idea what is wrong. The port's shell
does two things about that: pressing an unavailable item writes the reason on the status band
("no houses in … — run `make assets`"), and the item is drawn in grey.

The grey was not enough on its own, because the shell also has a *cursor*, and the cursor
starts on "New Game". On a machine with no houses — a fresh clone, which is the first screen a
stranger sees — a filled inverse-video bar sat under the one item that cannot work, and an
inverse-video bar is this shell's "Return does this". An unavailable item under the cursor now
gets an outlined bar instead of a filled one.

The general rule for the rest of 1.7: **every state has to be distinguishable from every other
state it can be confused with, not merely rendered differently.** Enabled-and-selected,
disabled-and-selected, enabled-and-not, disabled-and-not is four states and needs four
appearances. 1.7b's settings screen has more of them than this menu does (a setting can also be
unavailable *because of another setting*), and 2.28's pause overlay is the same problem in the
game.

### 2.52 The About box is the only documentation a player gets, and it was describing the wrong game — **DONE, 1.7a**

Its first draft listed the original's keys. Three of this port's bindings are not the
original's — player two is on A/D/W/S because the originals are modifier keys a window manager
intercepts (2.3), Tab pauses, and Escape ends a game rather than the program (2.7) — so the
1994 manual is actively wrong for this build, and the About box was repeating it.

It now lists what this build actually does, per player. That fixes the immediate error and
creates a maintenance hazard in its place, which is the part worth recording: **the list is a
Go string literal and the bindings are data** (`player.Glider`'s key set). The moment 1.7b makes
them configurable the two will disagree, and a wrong key list is worse than none. 1.7b must
generate the list from the bindings — the same argument as 4.4's dangling-test linter, one step
earlier.

**1.7b generated it.** `drawAbout` reads `Shell.host.Prefs` and formats each player's four keys
from it, falling back to `prefs.Default()` when there is no host — which is what `-shot` and the
tests see, and is still true of a fresh install. So the box is right for whoever is holding the
keyboard, including a player who has rebound everything, and the only literal left in it is the
sentence structure. The two unbindable lines moved with the same change: it now says
"Tab or Esc pauses" and "Q while paused gives up the game", which is what 2.7 made true.

### 2.53 A measurement that reads the player's settings is not a measurement — **DONE as a rule, 1.7b**

The moment there is a preferences file, every golden image and every benchmark in `make check`
depends on the machine it runs on. A developer who has turned the volume down, picked a
one-room view or magnified the window would get different PNGs and a different frame rate from
everyone else, and the difference would look like a regression in whatever they were working on.

The rule `cmd/glidergo` implements is that **the four measurement modes start from this build's
defaults and cannot save**: `-shot`, `-frames`, `-bench` and `-dump` (`hermetic()` in
`cmd/glidergo/prefs.go`). That is one function and no special pleading at any Makefile call
site, which matters because the alternative — `-prefs none` on eight lines of the Makefile — is
a rule nobody can see being followed.

Three edges are deliberate and each is pinned by a test in `cmd/glidergo/prefs_test.go`, the
only tests in that package:

- **An explicit `-prefs <file>` beats even a measurement**, because that is how a golden image
  of the settings screen gets settings to show. Without it, 1.8's screenshot of that screen
  would silently draw the defaults and compare nothing.
- **`-house` is *not* a measurement.** Naming a house is asking to play, and a player who plays
  that way wants their own bindings — so hermeticity follows the *mode*, not the presence of a
  flag.
- **`-prefs none` reads and writes nothing**, so a run on a machine that has a settings file can
  be made reproducible from the outside, and cannot create one either.

The general rule for the rest of the project: **anything that reproducible output depends on has
to be a property of the invocation, never of the machine.** 1.8's corpus is the next thing this
binds; the seed (`-seed`), the house, the room and the window size are already in that category
(2.42).

### 2.54 The pictures drawn over a running game are the house's, not the application's — **DONE, 1.7b**

`render.Assets` had two resolution orders and 1.7b needed a third. `Pict` consults the open
house's resource fork first and records a missing file as a fault, which is right for a room
background. `UI` looks only at the application and stays silent when a plate is absent, which is
right for the shell — a house that could repaint the title screen could hide the way out of it
(`TestUIIgnoresTheOpenHouseFork`), and a checkout with no art still has to reach a menu (2.6).

Neither is right for the pictures drawn *over a running game*: the pause placards, the
stars-remaining panels and the banner plates. On a Mac the open house's fork really does sit in
front of the application's for those ids, and the shipped houses use it — **Teddy World carries
its own PICT 1015 and 1016**. So `Assets.Plate` is the third order: the fork first, then the
application, and silence when neither has it, because these have a drawn fallback.

Counted on disk while settling this, and it is what 1.7d needs before it starts: **thirteen** of
the twenty shipped houses carry their own 1991-1993 (the banner plates) and **four** their own
1017 or 1018 (stars remaining). So `BringUpBanner` and `DisplayStarsRemaining` go through `Plate`
as well, and a port that reached for `UI` there would look right in every house nobody had
customised.

### 2.55 Nothing may destroy the player's settings, including the game itself — **DONE as a decision, 1.7b**

The settings file is the one thing on disk that is about the *player* rather than about the game:
eight bindings somebody chose and got used to. Three decisions protect it, and each is a
deliberate refusal of a more convenient behaviour:

- **Saving happens at the change, not at quit.** The original writes its preferences once, in
  `WriteOutPrefs` at quit (`Main.c:381`), so a crash loses everything the player set that
  session. This port saves in exactly two places — the settings screen closing, and `runShell`
  noticing a different house was chosen — which costs one small file write and cannot lose one.
- **`-import-prefs` refuses to overwrite.** Converting a 1994 `Glider Prefs` file into a
  destination that already exists is an error, not a backup and not a prompt: there is no prompt
  to give (it runs before any window opens), and telling somebody to move their file aside is
  cheaper than a backup scheme they would have to learn.
- **A session with nowhere to write says so on the way out**, rather than pretending: closing the
  settings screen with no saver shows "settings changed for this session only — there is nowhere
  to save them". A screen that silently discarded a rebind would be indistinguishable from one
  that had saved it.

A related rule that costs nothing and prevents the worst case: a preferences file that will not
load, or a configuration directory that does not exist, leaves *usable* settings in hand and a
line on stderr. Refusing to start a game over a file the game itself wrote is not an acceptable
failure mode (`prefs.Load`, `loadPrefs`).

### 2.56 The high-score screen's way out is written in blue on a starfield — **note; the fix is a 2.x option**

`DrawHighScores` writes its footer — the original's `Click Mouse or Hit a Key to Exit`, `STR# 150`
index 8 — in `QDBlue` (palette 211) at baseline 308, over PICT 1995, which is a near-black star
field (`docs/analysis/scoring.md` 7.9.2). It is the least readable text anywhere in this port, and
it is the one line on the screen that tells a player what to do. Transcribed exactly anyway,
because 1.8's corpus measures this port against the original and a colour changed here would be a
corpus measuring the port against itself.

The fix is one palette index and belongs with the other legibility work in 3.3: either the cream
the plaque's own title uses (8) or plain white, behind the same "modern defaults" switch that 2.19,
2.20 and 2.39 sit behind. Worth doing before the first public build, because the screen is the last
thing a player who just earned a score looks at.

### 2.57 Every board stores ten timestamps and nothing has ever drawn one — **note; a layout change, so 1.8 at the earliest**

`scoresType` carries `TimeStamps[10]`, `unsigned long` Mac epoch seconds, written whenever a score
is inserted and read by nothing: `DrawHighScores` shows name, score and rooms and drops the date
(`docs/analysis/scoring.md` 7.5, 7.9.2). So the shipped houses carry dates no player has ever seen,
and decoding all twenty non-empty boards turns out to recover the authors' own playtesting:
`The Asylum Pro` 1995-06-08, then a run of fourteen houses played between 1995-07-03 and 1995-07-28,
`California or Bust!` on 1995-09-08, `Art Museum` on 1996-01-13, `Davis Station` on 1996-10-17, and
two stamped 2000-05-11 — Slumberland's top two rows that morning and the whole of `Sampler`,
39 seconds after the save that created it (1.2). Thirteen of the twenty are topped by `Ozma`, two by
`Paul` — and those two, ImagineHouse PRO II and Grand Prix, are exactly the two houses Jonathan Chin
signs as Paul Finn in their banners, so the boards are a third, independent trace of that pen name.

This port keeps writing them, because the file format is the original's and a side-car that dropped
a field could not be read back into a house. Drawing them is a different matter: the plaque is 332
pixels wide and already carries three columns, so a date column means a new layout, which means the
corpus 1.8 is about to record would be recording it. Deferred rather than declined — a high-score
board that can say "1995" is a better artefact than one that cannot, and this one has the data.

### 2.58 The name dialog appears in the corner of the screen — **note; a 1.8 option**

`BringUpDialog` has its `CenterDialog` call commented out (`GliderPRO/Sources/DialogUtils.c:26`),
so DLOG 1020 appears at its stored bounds — `(0,0,109,316)`, the top-left corner — while DLOG 1021,
whose stored bounds are `(40,40,162,356)`, appears very slightly inset. Two dialogs in the same
sequence, one in the corner and one not, because one resource happened to be saved in a different
place (`docs/analysis/scoring.md` 7.11.1).

Both are reproduced exactly, and `internal/scores`' own test asserts both positions. Centring them
is a one-line change per dialog and it is the right change for a release; it is listed here so that
it is made deliberately, with the corpus regenerated, rather than as a tidy-up.

### 2.59 A mask's resource id is not "the picture's id plus 1000" — **note; matters to every masked draw**

The convention `docs/analysis/scoring.md` 2.8 states — `kPointsPictID` 4006, mask `PICT 5006` — is
real but not general. The masks actually shipped are:

| picture | its mask | offset |
|---|---|---|
| 4006 (points numerals) | 5006 | +1000 |
| 4002 (bonus numerals) | 5002 | +1000 |
| 1994 (high-score plaque) | **1998** | +4 |
| 1019 (the angel) | **1020** | +1 |
| 1990 (the GAME OVER pages) | **1989** | **-1** |

So any code that derives a mask id by arithmetic is wrong for three of the five, and the one it is
most wrong about is the game-over animation, where the mask id is *lower* than the picture's. Every
pairing in this port is written out at the call site instead (`internal/scores`' 1994/1998, and
1.7d's 1019/1020 and 1990/1989 when it gets there). Recorded because "+1000" reads like a rule and
is a coincidence.

### 2.60 The two game-over paths disagree about whether a high score suppresses the splash redraw — **DONE, 1.7d: both paths now ask**

`TestHighScore()` has exactly two callers, and they use it differently
(`docs/analysis/scoring.md` 8.7 and 9.8):

| caller | context | return value | `RedrawSplashScreen()` |
|---|---|---|---|
| `GameOver.c:67` | the win ending | used | only when no score qualified |
| `GameOver.c:503` | the loss ending | ignored | always |

The win path is the considered one: the high-score screen has already covered the window, so
redrawing the splash under it would be a wasted blit and a visible flash. The loss path does it
unconditionally, which means the board a player just earned is drawn over the splash and then the
splash is drawn over the board — except that `RedrawSplashScreen`'s copy runs in the wrong direction
and draws nothing (7.8, and 2.56's cousin), so on a real Mac neither is visible and the bug hides
the bug.

1.7d ports both animations and both endings, and it should port the *win* path's arrangement to
both: ask, and skip the redraw when the answer is yes. The high-score hook this stage installed
already returns exactly that Boolean.

**Done, and it is the only place in 1.7d where the port chose the better of two shipped
behaviours rather than transcribing one.** `internal/game/gameover.go`'s loss path is
`if !w.TestHighScore() { w.restoreSplashScreen() }`, the same line as the win path;
`TestDoGameOverSkipsTheSplashWhenAScoreQualifies` pins the decision. The C's unconditional
redraw is unobservable on a Mac for the reason above, so nothing a player of the original saw
has changed.

### 2.61 The win animation writes three rects outside its array, and seeds five it never reads — **DONE as decisions, 1.7d**

Two defects in the same forty lines of `GameOver.c`, found while transcribing them, both
recorded at length in `docs/analysis/progression.md` 11.2 and in the port's own comments:

* **`DoGameOverStarAnimation` writes `pages[-3]`, `pages[-2]` and `pages[-1]`.** The angel
  starts at `left = -96`, a star is seeded whenever `left % 32 == 0`, and `which =
  left / 32 % 5` keeps the sign of the dividend in C — so the first three seedings write three
  rects below a file-scope array. Nothing reads them (`count` cannot rise for a negative
  `which`), which is why the game shipped. The port skips the write and keeps the chime, which
  the C plays outside the branch; the ending therefore still opens with three sounds and no
  star.
* **`SetUpFinalScreen` seeds five star positions that are dead.** `dest` and `was` are both
  reassigned by the animation before it can draw a slot; only `frame = RandomInt(6)` survives.
  The port keeps all fifteen `RandomInt` calls regardless, because the generator is shared with
  every flame phase and pendulum in the game and dropping ten draws would change what every
  room composed afterwards looks like.

Neither is an "improvement" a player can see, and that is the point of recording them: both are
the sort of thing a later reader would take for a transcription slip and "fix", and one of the
two fixes would silently move the whole random stream.

### 2.62 The quit path repaints the work map with a title screen nothing can display — **DONE, 1.7d: dropped, and the harness gets an invariant back**

`NewGame`'s tail (`Play.c:257-273`) ends a game the player quit by scaling the splash art into
`workSrcMap` and invalidating the window, so the Mac's next update event paints the title screen
from it. Transcribed literally, that is 294,400 pixels written after the last frame — and in this
port not one of them is reachable. The title screen belongs to `internal/shell`: it is composed
on the shell's own surface and presented on the pass after `NewGame` returns, so nothing reads
the game's work map again. A `-dump` run cannot see them either, because the restore presents
nothing and a dump writes on `Present`.

Dropping it is worth more than the pixels, because it is what lets the fidelity harness say
something sharp. `internal/replay`'s `Result.Planes` hashes the three planes *as the last frame
left them*, and the useful consequence is that in a room where nothing animates the work map must
equal the background map — every animated thing registers a back rect and every back rect is
restored. Anything that draws into the work map without registering one shows up as a hash
mismatch, which is exactly the class of bug that is invisible for one frame and then smears
(`replay_test.go`'s room-70 case). A teardown that repaints the whole work map collapses that
check: every script, in every room, ends with the same splash hash. This was live for one stage
— `restoreSplashScreen` was a stub until 1.7d made it real, and the moment it did, two rooms of
the 1200-frame corpus started reporting the identical work plane.

The two ending paths keep their call. There the splash is on screen a moment later, because
`DoGameOver` composes the starfield over it and presents (`gameover.go:140`), so the pixels do
what the C wanted them to do.

### 2.63 The demo recorder can silently kill the demo it is recording — **DONE, 1.8b**

A `'demo'` record is `{long frame; char key; char padding}` and playback is
`if (gameFrame == demoData[demoIndex].frame)`, then `demoIndex++` — an **equality** test, not a
"have we passed it" test. So two records carrying the same frame number strand the cursor: the
first is consumed, the second never matches any later frame, and every record behind it is
unreachable. The demo simply stops responding partway through, with no error, no truncation and
nothing on screen to say it happened.

The original's own recorder can produce that file. `LogDemoKey` is called from inside `GetInput`'s
branches, and two of the four calls sit *above* the test that would have suppressed a second log
on the same frame: the band call is above `if (!fireHeld)` (`Input.c:348`), and the right-key call
is above the both-keys test (`:304`). Nothing anywhere in the C checks the stream it just wrote.
The shipped resource happens to be clean — 1,117 records, 1,116 gaps, none of them zero — which is
luck and the reason the bug survived.

Three places in the port take it seriously, because a format whose failure mode is silence needs
its check somewhere a human will look:

- `demo.Recorder` refuses a duplicate frame and counts it in `Dropped`, so recording cannot
  produce a stream that kills itself.
- `Stream.Validate` rejects one, and `Encode`/`WriteFile` validate **on write and not on read**:
  anything that came off a 1994 disk loads, and anything this port creates is checked.
- `glidertool demo check` is the command that says so out loud, and `demo info`'s status column
  says it for a whole directory at once. `TestCursorRepeatedFrameStalls` pins the stall itself, so
  the port's tolerance of a bad stream is deliberate rather than accidental.

What is *not* changed: playback still compares for equality. The refusal is at the writing end,
where the original's bug was, and a hand-made stream that stalls still stalls exactly as it did in
1994 — that is the behaviour, and the tools now name it instead of the player discovering it.

### 2.64 A guard that fires every frame buries the guards a report is about — **DONE as a named exception, 1.8b**

2.33's rule is that every out-of-range read the C performs is counted every time it happens, and
that rule has been right for all fifteen sites but one. `demoData[demoIndex]` past the last record
(`Input.c:224`) is not an event, it is a *state*: once a demo outlives its recording it reads off
the end on every frame until the glider dies, which for the shipped stream would be hundreds of
identical `Deviation`s. `Diag.Seen` keeps the first sixteen distinct kinds, so a long demo would
fill a bug report with one finding repeated and push out the room-object and trigger guards the
report was actually opened about.

`demoKey` reports it **once per run**, on the first refusal, and `Cursor.PastEnd` carries the real
count for anyone who wants it (`glidertool replay` prints it in the summary). The exception is
written down in three places — the call site, `devDemoRecord`'s comment, and `badIndex`'s doc,
which now says that one caller breaks its rule — because a diagnostic convention with an
undocumented exception is worse than one with none.

The general shape is worth keeping in mind for later stages: any guard inside a per-frame loop
whose condition persists needs a *first-occurrence* report and a counter, not one event per frame.
This is the first such site; Stage 3's network loop will have more.

### 2.65 Touching the controls during the attract mode must give the player the game back — **DONE (the original was already right), 1.8b**

`BUILD_ARCADE_VERSION` is on in the shipped build, and its block at `Input.c:192-201` says that
any of player one's four game keys ends a demo: `playing = false; paused = false`. That is the
behaviour a 2026 player expects and would complain about the absence of, and it is worth recording
as a thing the original got right, because the port had to choose which half of a `#if` to
transcribe and the other half — where only Command-Q and Command-S do anything — would have left
a player watching a ghost fly with no way to interrupt it but the mouse.

Two details of *how* it ends are the kind of thing a port would tidy away by accident. It does not
`return`: the frame's recorded input is still applied and the pause key still tested, because the C
only sets the two flags and falls through, so the demo ends one render later and not on the
keypress. And the two Command keys are genuinely dead in this arm, so the arcade build cannot quit
or save from the attract mode at all. `TestAGameKeyAbortsTheDemo` pins both — the run stops within
two frames of the press, having consumed a strict prefix of the records, and `GameOver` is
**false**, because what stopped is the demo and not a game.

The polish this leaves for the shell (1.7's screen, still optional): `DoDemoGame` re-arms
`incrementModeTime` on the way out, which is the original's way of making sure a demo that has
just ended does not immediately start another. A release should also not start one while the
player is part-way through choosing a house — the idle timer measures idleness, and a house picker
with a cursor in it is not idle.

### 2.66 The menu panel's bottom edge was a constant, and the seventh row was already outside it — **DONE, 1.10**

Found while adding "Open Saved Game…" as the title screen's third row, and it predates 1.10:
`menuBottom` was the constant `264` while the rows were laid out from `menuTop + menuFirst` at
`menuPitch` intervals, so with six rows the sixth baseline was at 272 and with seven at 298 — both
*below* the box they were drawn in. Nothing looked broken, because the rows are drawn after the
panel and a menu row over the splash art reads as a menu row either way; what was lost was the
panel, which had stopped being the thing the rows sit in.

The fix is to derive it: `menuRect(n)` returns the box `n` rows need, so adding a row moves the
box instead of overflowing it. That has a ceiling — `panel()` dims eight rows past every edge and
`DrawOnSplash` writes the house's name upward from row 307, so a bottom past 299 lays grey over
the top of the name — and `TestMenuPanelClearsTheHouseLabel` is what stops a future row from being
added past it. Two constants came out of it as well: `menuRise` is now tied to `menuPitch` rather
than sharing the house picker's `barRise`, because a selection bar one pitch tall is the tallest
one that neither overlaps its neighbour nor clips the descenders above it.

**Why it is worth a number rather than a silent fix:** it is the second time a shell constant has
been found to be describing a layout that had moved (2.29 was the first). A geometry that is
computed from the thing it bounds cannot drift; a constant beside it can, and will.

### 2.67 A hint row that shrinks leaves the wider row's pixels on the screen — **DONE, 1.10**

1.10 needed the pause hint to change *while the pause is up* — S replaces it with "game saved", Q
replaces it with the give-up question — and that exposed a latent bug in the pause placard.
`paintPause` fills only the rect the *current* hint occupies, so a hint that gets shorter paints
over a prefix of the old one and leaves the tail behind: the player reads "game saved -- S again
replaces itame". Nothing before 1.10 could reach it, because the hint was set once before the game
started and never changed.

`World.SetPauseHint` is the fix and the only supported way to change the text: it copies the old
row's rect back from the work map first, so whatever the hint used to cover is restored before the
new one is drawn, and `pausePainted` is left covering the union so `DoPause`'s restore is still
right. The alternative — widening `pauseHintRect` to a fixed maximum — would have dimmed a band of
the game the placard has no business touching.

### 2.68 One save per house, where the original's dialogue would have allowed twenty — **decision taken, 1.10; revisit if anybody asks**

`internal/saved` keys a saved game by house name, so saving twice replaces. The original's
`SaveGame2` would have called `StandardPutFile` and let the player name the file, which means as
many saves of one house as they liked — and in exchange, a resume meant finding the file again.

The trade was taken deliberately: keyed saves make resuming one keystroke, which is what makes the
feature usable at all on a title screen with no mouse, and Glider PRO is a game you resume rather
than one you branch. It is a real loss all the same, and it is the kind of thing that is invisible
until somebody wants to keep a save before a hard room. The shape of the fix if anybody does:
`Store` already separates the key from the filename (`datadir.FileName`), so a suffix — `Titanic
(2).save` — is a listing change and a picker, not a format change. The file format needs nothing:
it already carries the house name it belongs to.

### 2.69 "Beginning a new game will overwrite your saved game" — deliberately not ported — **decision taken, 1.10**

`QueryResumeGame` (`Menu.c:710-758`, never called) puts up a modal asking whether to resume, and
the original's design has a second one behind it: with one save per game there is a moment where
starting a new game destroys the save you have. The port does not ask, because in the port that
moment does not exist — a save is keyed by *house*, so starting a new game in Slumberland cannot
touch the save in Titanic, and starting a new game in Slumberland does not overwrite Slumberland's
save either. Only saving does.

Recorded because the absence is a choice and not an oversight, and because 2.68's fix would bring
the question back: numbered saves need a "which one?" and so a picker, and a picker needs the
modal the port currently has no use for.

### 2.70 The status band said one thing at a time, and the resume path has three things to say — **DONE, 1.10**

The title screen's status band was a single `s.msg` string, set by whatever last had something to
report. 1.10 gives it a third source — whether the selected house has a game to resume, and if so
whose save it is and what is in it — which has to coexist with the two that were already there
without either silently winning. `Shell.status()` is now the one place that orders them, so the
band is a function of the shell's state rather than a record of the last write to it.

Worth a number because the failure it prevents is specific: a player who presses `O` on a house
with nothing to resume needs the reason *on the screen they are looking at*, and a band that had
been overwritten by an unrelated message would send them to a terminal they may not have.

---

### 2.71 An audio player that is installed is not an audio player that works — **half DONE; the fallback is the other half**

`OpenPipe` picks the first player on `players` that `exec.LookPath` finds and `cmd.Start`
accepts, and `Start` succeeding means the binary was executed, not that it can reach a sound
server. Run the game over SSH, in a container, or from a session that does not own the seat, and
`pw-play` is present, starts, fails to connect, and exits. The game reports `audio=pw-play` and
then plays nothing.

Found by running `make check` in a clone under `env -i`, which is not a contrived environment:
it is what a CI job, a systemd unit and an `ssh host glidergo` all look like from inside.

The half that is done is attribution. The player's diagnostics have always been passed through
to the terminal, on the reasoning that swallowing them leaves "there is no sound" unexplained,
but they arrived anonymous:

    error: pw_context_connect() failed: Host is down

landing between two `glidergo:` lines, naming neither the program that said it nor which player
the game had chosen. `audio.stderrPrefix` now tags each line with the player's name, so it reads
`pw-play: error: ...` and both questions are answered by the line itself.

The other half is to stop over-claiming: a chosen player that dies at once should be reported in
the port's own voice and the next candidate tried, so a machine with both `pw-play` and `aplay`
lands on the one that works. That needs `Pipe` to own the `Wait` it currently performs in
`Close`, because a startup probe and `Close` cannot both wait on the same process, and it needs a
short grace period at open time that the happy path also pays. Neither is difficult; both are
more than a diagnostic fix, and the shape belonged with 2.48's native driver rather than in front
of it. **2.48 has since landed, so this is no longer waiting on anything**, and the seam it added
is where the retry belongs: `audio.Open` already tries the device and then the players in order,
which is the same loop a player that dies at once needs to fall out of. Until then the end-of-run
line still says `pw-play stopped taking samples`, which is the after-the-fact version of the same
fact.

**The test half, which lands with or before the fall-through.** A fake player in `internal/audio`
that is the test binary itself, re-executed (the helper-process pattern `os/exec`'s own tests use),
stdlib only, about 100 lines. An internal test replaces the package `players` var with the test
binary's absolute path, and the mode comes from arguments after `--`.

Four modes:
- **discard**, **exit at once** and **exit after N bytes.** All three pass against today's code:
  `Write` never blocks, a dead child is reported through `Err()` as EPIPE, and `Close` reaps. They
  are regression cover.
- **stall** — alive, never reads. This one finds a hang. `Close` waits for `pump` before it closes
  `w`, and `pump` is stuck writing to a full 64 KiB pipe, so **quitting freezes the game**. `aplay`
  without `-N` on a busy device does exactly this (its man page says so). The fix goes in the same
  change: `Close` closes `w`, or kills the child after a short deadline, before waiting on `done`.
  `Dropped` only shows up in this mode.

**The two-candidate test is this entry's acceptance test**, and it fails today: the first fake
exits at once, and `OpenPipe` must land on the second.

`sink.go:184`'s "nil in the tests, which have no child process" describes tests that never existed
(`git log -S newPipe(`). The package joins `make race` once a test starts the pump (4.34).

**Amendment: three of the four modes landed with 4.34, and the stall was real.**
`internal/audio/pipe_test.go` re-executes the test binary as the player, as planned, but through
`newPipe` and not through the `players` var. The `players` form is what the fall-through needs, so
it waits for the fall-through, with the exit-after-N mode and the two-candidate acceptance test. The
package is in `make race` now.
- **echo**, which is discard that hands every byte back. Every sample `Write` accepted reaches the
  player in order and encoded, because `Close` drains the pump before it closes stdin.
- **exit at once.** `Write` never fails, `Err` reports the refused write, and `Close` returns.
  **It found a panic.** A `Write` after `Close` sent on the closed queue. `WaveOut.enqueue` guards
  against exactly that and says why, and the Pipe never got the guard. The block is dropped and
  counted now.
- **stall.** As predicted, `Close` never returned, so the game could not quit. `Close` now gives its
  wait for the pump and its wait for the player one deadline between them, `pipeDrain`. When it
  runs out the player is killed. The kill fails the pump's write and `Err` records it, so the
  end-of-run line still says the player stopped taking samples. The deadline is 2 s, not
  `WaveOut`'s 500 ms `waveDrain`, because a pipe player has a buffer of its own to play out after
  its input ends. Measured here through PipeWire, `aplay` takes 0.62 s and `pw-play` 0.04 s, so
  half a second would have cut the last note off a healthy `aplay` at every quit. `ffplay` and
  SoX are not installed here, and their drain is not measured.

The `errs` comment is corrected. A race-built child also sleeps a second before it exits
(`GORACE`'s `atexit_sleep_ms`), and the harness turns that off.

### 2.72 X11 auto-repeat is undetectable, so every `!ev.Repeat` guard is dead on Linux — **DONE, the release gate's first step, with `Repeat` tracked by keycode**

`platform.Event.Repeat` exists so that a held key does one thing, and the shell and the game test it
wherever that matters. Win32 reports it directly (`lParam` bit 30, `win32.go:534`). The x11 backend
infers it from its own record (`x11.go:244`: a press while the same keycode is already down).
Until the fix below, that inference never fired, because by default an X server sends every auto-repeat as a
`KeyRelease`/`KeyPress` pair, so the bitmap sees a release before each repeated press. On Linux,
therefore, a held key fired its action again 25–33 times a second.

What that did to a player:
- **Holding Escape on the house picker quit the application.** Reproduced with the real binary.
- **A flight key still held when a "press a key" screen appeared ended that screen within about 40
  ms.** This covered the race result (`race.go:776`), which is the process's last screen, so a
  player holding an arrow never saw YOU WIN or YOU LOSE; the high-score board after its 1 s hold;
  and the banner and ending `Wait`s (`play.go:933`). This was from reading the code when it was
  filed. The banner has since been driven (below).

The glider's physics were unaffected, because `KeyDown` reads the bitmap and the bitmap stays held
(one poll in 61 read not-held, and that was the first).

**Done: the one call.** `x11.New` calls `XkbSetDetectableAutoRepeat(dpy, True, &supported)` straight
after `XOpenDisplay` (`x11.go:124`). XKBlib is part of libX11, and `XkbKeycodeToKeysym` in the same
file already used it, so nothing new is linked. The server then sends repeats as presses with no
release between them, which is what Windows does, and the rule at `:244` marks the presses bit 30
marks. It keys on the X keycode, not the `platform.Key` (below), and the comments at both sites say
so. `win32.go:528-533` no longer calls the x11 inference a working second best: it says the
inference works only because `New` asks, and that bit 30 is still the better answer, because it
stays correct across a lost `WM_KEYUP`.

**Reproduced before and after,** on a nested Xephyr with a repeat delay of 660 ms and 25 repeats a
second. xdotool is not installed, and libXtst has its runtime here but not its headers, so each 1.5
s hold came from a throwaway XTest helper outside the tree, with the one prototype declared by hand.
Before: the backend delivered 22 presses and 22 releases, each release at the timestamp of the next
press, and marked none. Escape held on the picker exited the process with status 0, a held Return
dismissed the "Hit anything to begin" banner it had just opened, and a held Backspace emptied the
Race screen's address field. After: 22 presses, 21 marked, and one release. The picker goes back to
the title screen and stays there, the banner stays up and is pixel-identical to a single tap, and
Backspace deletes one character. A held Down still moves the picker 22 rows, to the same pixels as
before, because the shell's filter lets arrow repeats through (`internal/shell/shell.go:439`). The
race result, the high-score board (`highscore.go:231`) and the pause's `S` (`play.go:717`) were not
driven, because that needs a two-window race, a qualifying game over and a paused save. They read
the same flag as the banner, which was.

**The server's answer is not acted on.** A server that cannot do it is not refused, because the
glider reads the bitmap and the release/press pairs leave it held. The package comment says what
such a server loses: `Repeat` is never set, and a held Escape on the picker quits. It says so beside
the 533 fps figure, which is where this backend records what it has been measured on. Xephyr and the
DCV desktop both support the flag. GLFW's fallback stays unwritten until a server needs it: on a
`KeyRelease`, `XEventsQueued` and then `XPeekEvent`, and a `KeyPress` with the same keycode and time
means the release is dropped and the press is marked. It would be a pure function, testable without
a display.

**The test.** `internal/platform/x11` has its first, `TestNewAsksForDetectableAutoRepeat`, built
where `x11.go` is. `DISPLAY` decides, as it does for `make smoke`. Unset, the test skips. Set, a
window must open, so a stale `DISPLAY` fails `go test ./...` rather than skipping. The test asks the
server rather than trusting what `New` was told, through an unexported `detectableAutoRepeat` that
wraps `XkbGetDetectableAutoRepeat`. The wrapper lives in `x11.go` because cgo is not allowed in a
`_test.go` file. On a server that does not support the flag the test skips, since it cannot then
tell whether `New` asked. It passes on Xephyr and on the DCV desktop, and it fails against a scratch
copy of the package with the call removed. It cannot hold a key, because that needs XTest, which is
not a dependency, so `make check` does not repeat the runs above. CI's `make check` has `DISPLAY`
unset, so there the test skips. A second Xvfb step (`ci.yml:149-150`) runs
`go test ./internal/platform/x11/` under its own server, with `-noreset` so that a client
disconnecting cannot reset the server under the test's window. Without it a runner never asks.

**Both side effects came out as predicted, and neither got code.** Held Backspace and held letters
in the Race screen's field now act once on Linux, as they already did on Windows. Before the fix, a
held Backspace repeat-deleted on Linux by accident. The high-score name dialog (`highscore.go:159`)
has never filtered repeats, so it still repeats on both platforms. Whether a text field should
repeat-delete is the shell filter's decision, for both platforms at once: the filter would let
`KeyDelete` and text-bearing repeats through on the Race screen, and still drop Escape and Return.

**Done too: `Repeat` by keycode.** The first version keyed the rule on the `platform.Key` bitmap,
and that got two cases wrong. Keys with no `platform.Key`, such as accented letters, have no slot in
it, so they carried no `Repeat` and typed again on every repeat. And keys that share a
`platform.Key` share its slot: Backspace and Delete, a digit and its keypad twin, Return and keypad
Enter, and each left and right modifier. So Delete tapped while Backspace was held read as a repeat,
and the shell dropped it, where Windows keeps bit 30 per virtual key and lets it through. `Repeat`
now comes from a second array indexed by X keycode (`held`, `x11.go:244-245`), which is always in
range because keycodes are 8..255. It is computed before the `KeyUnknown` branch, so that branch's
text events carry it too. The bitmap still feeds `KeyDown` and is unchanged, so the glider's physics
are too. Driven on the same Xephyr through the real backend. Before: Delete tapped 1 s into a held
Backspace arrived marked `Repeat`, and a held KP_Multiply gave 22 `*` presses with none marked.
After: the Delete press is fresh, and KP_Multiply gives one fresh press and 21 marked, which is what
bit 30 gives.

**Left open.**
- `FocusOut` clears both arrays, so the first repeat of a key held across a focus change reads as a
  fresh press. Bit 30 would not do that. Keeping the arrays across a focus change would be worse: a
  key released while another window had focus would mark its next real press as a repeat. This
  predates the fix and is minor.
- `KeyDown` still reads the `platform.Key` bitmap. So releasing Delete while Backspace is held reads
  as Backspace released too. That is 2.74's, where keycode against keysym is decided.

Windows being right is established by reading the code only: no key has ever been pressed on the
Windows build (PLAN Stage 4). Any future X11 backend (5.10) must keep these semantics.

### 2.73 A shredded glider draws no confetti, and the game says to report it — **DONE, the release gate's first step, after 4.30's soak found it**

`World.RenderShreds` (`internal/game/shreds.go`) asked for `w.R.A.Sheet("shred")`, and had since it
was written. `"shred"` is registered in `stripBounds` (`internal/render/assets.go:96`), not
`sheetBounds`, so the lookup failed. That failure is sticky: `render: no such sheet "shred"`. A
failed accessor does not stop anything. It records the first error and returns nil, and every blit
in `RenderShreds` is guarded by that nil, so the cloud grew, fell, played its shred sounds and
sparkled with nothing in it. In glidergo it meant no shred confetti ever drew, and quitting printed
the bug-report footer and exited 1. 13 of the 22 shipped houses have a shredder.

The fix is `Strip` for `Sheet`, one word. It restores what the original draws, so it is faithful and
needs no opt-in. No golden replay visits a shredder with the full asset tree, which is why nothing
caught it. 4.30's soak fails on any non-nil `Run` error, and now that it is in the tree it is the
general guard.

**Done: `Strip`, and a test for each reason nothing saw it.** `RenderShreds` asks for
`Strip("shred")` (`internal/game/shreds.go:206`), and the comment there cites
`StructuresInit.c:556-563`: `shredSrcMap` is a GWorld of its own, exactly `shredSrcRect` in size,
loaded from PICT 4010 with its mask from 5010, which is what a strip is in this port. Before the
fix, `glidertool replay` into Slumberland's room 43 and a headless `glidergo` run there both exited
1 with the error above. After it, both exit 0, and the headless dump draws the cloud from frame 84:
it grows to 850 pixels in a 40×35 box, then falls 4 pixels a frame. The reproduction's trace digest
is the same before and after, because a trace records rects and sounds, and those were right all
along. No golden, fidelity or replay hash moved.

The first reason nothing caught it is that nothing that loads art had ever shredded a glider. The
golden traces stay clear of shredders, the game package's shredder tests run without art on purpose,
and `TestEveryHouseStartsAndRuns` spends each house's hundred frames nowhere near one. So
`internal/replay`'s `TestAShreddedGliderFallsAsConfetti` flies a glider into Slumberland's room 43
with the whole tree loaded. `RunWatching` must return nil, and all 850 opaque pixels of the strip
must be in Work, exactly, on twenty frames: the growth arm's last two and the eighteen the fall
draws, in one column and four pixels lower each frame. They must never be in Back. The search covers
the whole plane rather than a computed rect, so a cloud drawn in the wrong place fails as one that
moved. The game package's no-art tests pass with the bug put back, which was checked, and
`shreds_test.go`'s header now says so.

The second reason is that a wrong name costs nothing until the game ends. So `internal/render`'s
`TestEveryArtNameIsInItsTable` reads the calls instead of making them. It parses every non-test file
in the module, and every constant that reaches `Sheet`, `Strip` or `Object` is looked up in the
table its accessor reads. That holds whether the name gets there directly or through a helper that
passes it on. The helpers are found by reading function bodies, not from a list: `maskSheet`,
`opaqueSheet`, `bakeStrip`, `maskObject` and the three `DrawPict…Object` functions. An `Object` must
name an `artKey`, `artOpaque` or `artPair` object, because the extractor writes the sheet objects
into `object/` as well, and a wrong call there would load without complaint. The test checks 87
names, and this was the only one wrong. `TestTheArtNameCheckCatchesEachWayANameCanBeWrong` plants
this bug and nine other kinds of mistake, and requires each back at its line.

**What the two tests do not cover.**
- The replay test skips without the extracted tree, like the rest of its package, so on a machine
  with no assets it proves nothing. `make check-caveats` already says when the tree is missing.
- The static test checks constants only. The seven names that are not constant are all
  `thisObject.What` in `internal/render/locale.go`. They are data, and `TestComposeEveryRoom` covers
  composing rooms from data.
- The accessors keyed by number (`Pict`, `Plate`, `MaskedPlate`, `UI`, `Background`, `Misc` and
  `Flower`) were checked by grep, not by the test. `Misc` and `Flower` have no callers outside
  tests. That is not a bug, but both are dead accessors today.
- 4.30's soak was not in the tree when this was fixed. It is now, and it has been re-run against the
  fix: 3,000 damaged-house plays, none with an error from `Run`.

### 2.74 The glider is bound to what a key types, not where it is — **planned, soon after the next tag; before any prefs freeze**

Four comments (`platform.go`, `x11.go`, `win32/keys.go` and `prefs.go`'s "one canonical name for
the physical key") say bindings are physical. They are not. x11 reads
`XkbKeycodeToKeysym(…, 0, 0)`, which is the *first configured* XKB group, not the active one, so
behaviour depends on the order of the layout list. Win32 reads the VK code, and layout DLLs assign
VKs per layout (on French, the US-`;` position is `VK_M`). So `win32/keys.go`'s "a VK_OEM_n code
identifies a position" is itself false.

The players who lose are French and Belgian AZERTY users. Player 2's cluster scatters and the digit
keys break. QWERTZ only swaps Y and Z. Player 1 is on the arrows and capture-rebinding works, so
nobody is locked out.

**The fix:**
- x11: the hardware keycode, looked up by XKB key name (`XkbGetNames` with `XkbKeyNamesMask`, so
  `AC01` maps to A) rather than "evdev + 8", which is wrong on XQuartz.
- Win32: the `lParam` scancode plus the extended bit.
- `XLookupString` and `WM_CHAR` text stay exactly as they are.

**The regression side, and why this is M.** Everything that uses a key as a *letter* works on
AZERTY today and would break: the title-menu letters (`shell.go:520-547`), the picker's
type-select (`letterOf`, `:921-931`), the Y/N give-up prompt (`play.go:690-695`), and every key
name drawn on screen (About's `controlsLine`, the settings capture, the pause hint). All of them
move to `Event.Text`, which is what the original's menus dispatched on (`charCode`). A new optional
`KeyLabel(Key) string` per backend labels bindings through the host layout: x11 through
`XkbKeycodeToKeysym` at the current group from `XkbGetState`, Win32 through `MapVirtualKeyW` or
`GetKeyNameTextW`, and null with US names. Only the glider controls, pause, Delete and the paused
Q/S stay physical, as they are in `Input.c`.

A stored name like `"a"` comes to mean the A *position*. `legacy.go`'s Mac import is already
physical, and only lines up with the port after this change. Land it before 5.10 and Stage 6, so
every backend shares one meaning.

### 2.75 The startup facts that change behaviour are on stderr only — **note; the docs half DONE in PLAN release gate step 5; the on-screen half waits for a launch with no console**

Three facts change what a player gets, and they are said only on stderr:
- no sound (with `audio.installHint`'s per-OS advice, already written at `sink.go:271-282`);
- the prefs file was unreadable or moved to `.bad`, so the bindings are back to defaults;
- the remembered house is gone.

On Windows the console is visible, and on Linux the documented launch is a terminal, so for the
first release this is a gap only for a file-manager launch. It becomes real for a desktop launcher,
a `-H windowsgui` build or a macOS `.app`. Then these facts join the opening status line through a
new `Host` field (say `Notes []string`) that is combined with `opening()`, not substituted for it,
or "24 houses" is lost. `Shell.status()` lets a menu row's note win over `msg`, and that ordering
has to be decided.

Score-board repair notes stay on stderr, as `highscore.go:309-313` decided. Chrome art failures are
already on screen (2.6). The settings footer already shows the resolved prefs path; the one nit is
that with `-prefs none` it could say "this session only".

**Now, docs only:** README "Settings, scores and saves" lists the Windows and macOS paths next to
the Linux ones and says `glidergo -version` prints the resolved paths. The Linux `HOW-TO-RUN` says
sound needs `pw-play`, `paplay` or `aplay`.

**Done.** README's section opens with a table of four rows (settings, high scores, saved games and
the crash report) and one column each for Linux, Windows and macOS. Every path in it was checked
against what `-version` prints under `GLIDERGO_CONFIG`, `GLIDERGO_DATA` and neither. The notes
under the table cover XDG and the two variables. The Linux `HOW-TO-RUN` names the three players
and the two fallbacks, says that `-audio list` shows what was found, and says that with none of
them the game says so once and plays silent. That is the message the sink prints.

**Found doing it, and fixed: a relative `$XDG_CONFIG_HOME` put the settings somewhere different on
every Go.** The basedir spec says a relative value is to be ignored. `internal/datadir` already did
that for `$XDG_DATA_HOME`, but `prefs.Dir` took `os.UserConfigDir` as it came. Go 1.23, which a
source build uses, returns the relative path, so the settings landed under whatever directory the
game was started from. Go 1.26 and 1.27, which releases are built with, return an error, and the
game ran on the defaults, with a note on stderr and nowhere to save a change. `prefs.Dir` now treats a relative value as unset on Linux,
as `datadir` does, and `TestDirDefaultsUnderTheConfigDirectory` covers "", ".", a relative path and
an absolute one. It fails with the check taken out, and passes under 1.23.12 and 1.27.1. README's
note under the table was already right, and it is now true.

### 2.76 Every frame uploads the whole magnified window, and auto scale will make that 4× — **items 1 and 4 and the bench row DONE, the release gate's step 4; the cap stays until Windows is measured; items 2 and 3 open**

Both backends present the same way. `platform.Expand` writes the full pw×ph surface, then
`XPutImage` (`x11.go:215`) or `StretchDIBits` (`win32.go:601`) sends all of it. That happens every
frame, whether anything moved or not. At 4× it is 19.7 MB a frame, about 590 MB/s at 30.07 fps.

Every figure behind "fast enough" was taken at 1× or 2×:
- `x11.go:6-8`'s 533 fps;
- DEV_ENVIRONMENT §6's "Rendering is not a risk";
- `BenchmarkExpand` at 1–3× against a 16.7 ms frame (`expand_test.go:150-155`), which calls those
  "the three scales the shell offers" when `MaxScale` is 8;
- `windows-first-run.md:116-119`, at `-scale 2` only.

`make bench` is hermetic (2.53), so it always runs at 1×.

Measured on this host (Xeon 8488C, 8 cores), HEAD `1797f9f`, `Xephyr :57 -screen 2600x1980x24`.
Unpaced is `-frames 300 -bench`, three runs each. Paced is `-frames 240`, with CPU taken from
`/usr/bin/time` and from Xephyr's `/proc/<pid>/stat`:

| Scale | Unpaced | Paced, glidergo | Paced, X server |
|---|---|---|---|
| 1× | 636–666 fps | 9% of a core | 1.8% |
| 2× | 135–141 fps | 18% | — |
| 3× | 67–70 fps | 29% | — |
| 4× | 40–45 fps | 42% | 19% |

An earlier run under less load reached 49 fps at 4×. So 4× has 1.3–1.6× headroom over 30.07 fps, and
that is on a fast server core. A 4× frame takes about 24 ms. `Expand` is 4.6 ms of it, and the
engine and compositing about 1.5 ms. A bare C `XPutImage` + `XSync` of 2560×1920 on the same Xephyr
takes 16 ms (62 fps), so the upload is about two thirds of the frame.

**Why it matters now.** 2.1's amendment makes auto the default, and auto picks by monitor:
- 4× on 3840×2160, where 1920 plus the frame fits a 2112–2128 px work area;
- 3× on 2560×1600;
- 2× on 1080p, and on 2560×1440, where 1440 plus a 32–48 px bar does not fit.

Every 1080p player's default therefore goes from 1× to 2×, about 4.7× the cost of a frame. A frame
that overruns makes the game slower rather than skipping (2.17). So a 4K screen on a CPU much slower
than this one gives a game that runs slow, and nothing on screen says so. 2.10's 60/144 Hz
presentation multiplies every figure here. 5.10's pure-Go client sends the same bytes, and its
"speed is not a risk" is a 1× figure. Battery use is not measured, but 40–60% of a core for a 30 fps
game from 1994 is enough to run a laptop's fan.

**`Expand` is not where the time goes.** It already copies whole rows for the scale−1 repeated
lines. The only per-pixel work is one 4-byte `copy()` per output pixel of the first line in each
block. A `uint32` loop takes 4× from 4.6 ms to 1.7 ms, with byte-identical output at 1–8×, measured
in a copy of `expand.go`. That is worth doing, but it is 3 ms of 24.

What to do. All of it is stdlib, core X protocol and gdi32, it stays inside the backends, and none
of it touches a game coordinate (2.8):

1. **Send only the rows that changed.** The backend keeps a copy of the last 640×480 it presented.
   Comparing 1.2 MB row by row costs tens of µs. Only the runs of changed rows are then expanded and
   sent: `XPutImage` takes a sub-rectangle of the `XImage`, and `StretchDIBits` takes a source and a
   destination rect. An expose, a map, `WM_PAINT` or a fresh window forces the next Present to send
   the whole frame.

   Tried on a no-input 300-frame Slumberland run (`-tags nullbackend -dump`). Per frame, a median of
   17 of the 480 rows changed (p90 30, max 460), and the changed bounding box was a median 0.3% of
   the frame. A frame that did not change, such as a title screen or a pause, sends nothing. This
   needs nothing from `internal/game`, although its dirty-rect lists are there if a finer cut is
   ever wanted.
2. **On X11, let the server repeat the rows.** This is for the frames that do change a lot, such as
   entering a room or a shell screen. Send the frame widened only horizontally (2560×480 at 4×) to a
   `Pixmap`. Then `XCopyArea` each row `scale` times, which at 4× is 1,920 requests of 28 bytes. The
   GC needs `XSetGraphicsExposures(False)`.

   In a C probe on the same Xephyr:
   - 4× went from 62 to 138–158 fps;
   - 3× from 120 to 245–257;
   - 2× from 270–300 to 460–484.

   At 1× this is slower (920 against 1200), so 1× keeps the plain path. Repeating columns on the
   server as well was slower still (233 fps at 2×). It adds no library, and a core-protocol client
   (5.10) can do the same.
3. **On Windows, stop expanding.** Call `SetStretchBltMode(hdc, COLORONCOLOR)`, then `StretchDIBits`
   straight from a 640×480 DIB into the pw×ph destination. For an integer magnification only
   `HALFTONE` averages pixels, so `win32.go:577-579`'s "GDI's own stretch would smooth" is true of
   `HALFTONE` only. This is unverified here. Read the window back on the Windows test host and
   compare it byte for byte with `Expand` at 2–4× before `Expand` is dropped from that path. The
   same readback checks item 1's rect arithmetic on a top-down DIB.
4. **Present nothing while the window is unmapped or minimised.** `StructureNotifyMask` is already
   selected (`x11.go:149`), but `UnmapNotify` is thrown away. On Windows, use `IsIconic`.

MIT-SHM stays out. It needs libXext headers at build time (DEV_ENVIRONMENT.md:138), and items 1 and
2 take away most of what it would save.

**The budget becomes a bench row.** `-bench`'s summary line reports the process's CPU time next to
its frame rate: `syscall.Getrusage` on Linux, `syscall.GetProcessTimes` on Windows. It also reports
the slowest frame, because item 1 makes the average flatter than the worst case. `make bench` runs
`-scale 1` and `-scale 4`. CI's `xvfb-run -s '-screen 0 640x480x24'` (`ci.yml:143`,
`release.yml:173`) is too small to show a 4× window, so it becomes `2600x1980x24`.

Proposed budget, on the dev host:
- paced 4× under 15% of one core;
- unpaced 4× at 120 fps or more (four times the target), so that a machine a third as fast still
  keeps game time.

The comments at `x11.go:6-8`, DEV_ENVIRONMENT §6 and `expand_test.go:150-155` are corrected in the
same change.

**2.1's amendment carries the cap:** until the 4× row meets that budget on both backends, auto
resolves to at most 3×, and an explicit 4×–8× is still the player's choice. If items 1 and 2 land
first, the cap is never written.

Not measured: bare Xorg or XWayland (Xephyr copies every frame again into its host window), win32
above 2×, any laptop, any battery.

**Done, with 2.1: items 1 and 4, and the bench row.**

- **Item 1 is `platform.Changes`** (`internal/platform/changes.go`). `Diff` compares the frame with
  the last one it was given, row by row. It reports each run of changed rows, cut to the columns
  that changed in any of them. Past 32 runs, the frame is sent as one block that covers them all.
  - `ExpandSpan` expands just that block.
  - X11 sends each block with a sub-rectangle `XPutImage`.
  - Win32 sends each block with `StretchDIBits`, from a DIB header whose bits pointer is the
    block's first row, whose height is the block's, and whose source is `(x, 0, w, h)`. That
    sidesteps the question of which way a bottom-up source rect is counted on a top-down DIB.
  - An expose, a map, `WM_PAINT` and a frame of another size all make the next frame whole.
    `TestSendingOnlyTheChangesShowsTheWholeFrame` is the property: 60 random frames at 1–8×,
    sending only what `Diff` reported, leave the window byte-identical to `Expand` of every frame.
    On X11, `TestPresentSendsWhatTheFrameChanged` reads the real window back with `XGetImage`, and
    a destination offset left at 0,0 fails it.
- **Item 4:** X11 tracks `MapNotify`/`UnmapNotify`, and win32 checks `IsIconic`. An unmapped window
  is sent nothing, and its next frame is whole.
- **`Expand` writes a pixel as one 32-bit word.** That is about half the time at 2–4×, byte for
  byte the same, measured side by side on the loaded host. It did not reproduce the 4.6 → 1.7 ms
  above: `BenchmarkExpand` reads 67 µs at 1× and 2.5–3.9 ms at 2–4× on the same host, now under
  more load.
- **The bench row.** A timed run's last line reads `N of CPU, P% of one core; slowest frame D, at
  frame F`. That is `Getrusage` on Linux and the BSDs, `GetProcessTimes` on Windows, and nothing
  elsewhere. `make bench` runs `-scale 1` and `-scale 4` flat out, then 150 frames paced at 4×.
  CI's two Xvfb steps are `2600x1980x24`.
- **What the new read-back test found on CI's old screen.** `XGetImage` of a window that runs off
  the screen is a BadMatch. With no error handler installed, Xlib's default prints it and calls
  `exit(1)`, so at 640×480 a 2× read-back took the whole test binary down. The test now picks the
  largest scale up to 2 that fits `Room()` and skips on a screen too small for 1×. The handler is
  its own entry (2.78).

Measured on this host, Xephyr `:57 -screen 2600x1980x24`, the same host as the table above, under a
load average of about 4. Unpaced is `-frames 300 -bench`, three runs; paced is `-frames 240`. The
X server's share comes from its `/proc/<pid>/stat`:

| Scale | Unpaced, start room | Unpaced, through a door | Paced, glidergo | Paced, X server |
|---|---|---|---|---|
| 1× | 970–1490 fps | 732 fps | 7–8% of a core | 0.3–0.4% |
| 2× | 985–1080 fps | 571 fps | 7–8% | 0.6–0.7% |
| 3× | 1130–1180 fps | 574 fps | 7–8% | 0.7% |
| 4× | 840–1050 fps | 477 fps | 7–9% | 0.8% |

"Through a door" is `-room 8`, whose glider takes a door at frame 121 with no input. So **on X11
both budgets are met with room to spare**: paced 4× is 7–9% of a core against 15%, and unpaced 4×
is 477 fps or more against 120. The paced 7% is the same at 1× as at 4×, so what is left is the
game and not the window. That covers the engine and the compositing, `ToBGRX`, and the limiter,
which wakes every millisecond (`play.go`'s `WaitTick`, 2.17). The unpaced rows are not a fair comparison with the old table's 40–45 fps at 4×:
the start room's frames each send a median of 1,900 of 307,200 pixels, counted with a temporary
tally in `Present`.

**A whole frame costs what it did**, and `BenchmarkPresent` (`internal/platform/x11`, which waits
for the server with `XSync`) is there to show it:

| Scale | Unchanged | A 32-row sprite | The whole frame |
|---|---|---|---|
| 1× | 97 µs | 152 µs | 1.1 ms |
| 2× | 99 µs | 161 µs | 7.6 ms |
| 3× | 96 µs | 200 µs | 15.0 ms |
| 4× | 95 µs | 271 µs | 32.1 ms |

So a shell screen appearing, or an expose, is still one 32 ms frame at 4×, which is the next item's
case.

**The wipe was the finding.** A room transition is 116 or 160 presents inside one game frame (2.4),
and with whole frames each of those was a whole frame. The same `-room 8` run was built with a
temporary switch that made every present whole, which is the old path plus a copy. The slowest frame
(the wipe) and the run:

| Scale | Changed rows | Every present whole |
|---|---|---|
| 1× | 99 ms; 732 fps | 229 ms; 314 fps |
| 2× | 126 ms; 571 fps | **1.45 s**; 59 fps |
| 3× | 103 ms; 574 fps | **2.98 s**; 31 fps |
| 4× | 137 ms; 477 fps | **4.17 s**; 20 fps |

Before this change, then, 2.1's new 2× default would have frozen the game for about 1.5 s at every
door on this host, and 4× for four seconds. With it, a wipe costs about 0.1 s at every scale, and
most of that is per-present work, not pixels. 2.4 still wants the wipe paced to about a third of a
second, and now a door can afford that.

**The cap stays.** 2.1's rule was "both backends", and win32's changed-rows present has only been
compiled. So `autoMax` is 3, and 4× on a 4K monitor is one keystroke on the settings row. Lifting
it is one constant once 5.4's rehearsal has read a win32 window back and run `make bench`'s three
rows on the Windows host.

Still open:
- **Item 2**, the server-side row repeat, is now for the whole-frame case only: a shell screen, an
  expose, a room entered without a wipe.
- **Item 3**, win32's `COLORONCOLOR` stretch, is unchanged. The same rehearsal that reads a window
  back can check it.

### 2.77 Auto scale on a multi-monitor X11 desktop that is not GNOME measures every monitor at once — **note; after the next tag, and only if a report says it bites**

`x11.Room` (2.1) takes the per-monitor work area from `_GTK_WORKAREAS_D<desktop>`. Only mutter
publishes that, and it is the only per-monitor geometry that core X and libX11 give out.
Elsewhere the fallback is EWMH's `_NET_WORKAREA`. For more than one monitor, that is one rectangle
for the whole desktop: KDE's, Xfce's and most tiling window managers' span every monitor.

So a 1366×768 laptop beside a 4K screen measures as about 5200×2100, and auto picks 3×. If the
window manager then places a 1920×1440 window on the laptop, most of it is off the screen. The
settings row fixes it in one visit, and the stderr line from `windowScale` does not fire, because
by its measure the window fits.

The real answer is per-monitor geometry, which means RandR 1.5's `RRGetMonitors` or Xinerama's
`XineramaQueryScreens`. Both are libraries (`libXrandr`, `libXinerama`), and "only libX11" is the
backend's first rule (DEV_ENVIRONMENT §5). Sending the RandR request by hand over Xlib's connection
would keep the rule's letter and none of its point.

The cheaper options, in order:

1. **Believe the pointer less.** When `_NET_WORKAREA` is much wider than a single monitor's
   aspect allows (wider than 2.4:1, say), take its height and a 16:9 width of it. That is a
   heuristic and it would be written as one.
2. **Say so on stderr.** When the only source was `_NET_WORKAREA` or the screen and the room is
   wider than 2.4:1, `windowScale` says "if this window is too big for your screen, the settings
   row has a smaller one". That costs nothing and catches the case without guessing.

2 is worth doing when a non-GNOME multi-monitor report arrives. 1 is worth doing only if 2 is not
enough. Nothing here is measured: this host is GNOME, and one monitor.

### 2.78 An X protocol error, or a lost X server, ends the process from inside Xlib — **note; with or after 4.35**

The x11 backend installs neither `XSetErrorHandler` nor `XSetIOErrorHandler`. Xlib's defaults
print a line and call `exit(1)`. That skips every deferred function and every Go-side cleanup, and
it skips 4.35's crash file, because Go never sees a panic. Measured once 4.35 landed: a Xephyr
killed under the title screen left `crash.log` with its header and nothing below the rule, so the
next start said nothing.

2.76's read-back test found the first half. `XGetImage` of a window that runs off the screen is a
BadMatch, and on CI's old 640×480 Xvfb it took the whole test binary down rather than failing one
test.

In play, a protocol error needs a request this package gets wrong. Every `XPutImage` rectangle
comes from `Changes.Diff`, clipped to the window, so none is known. The IO half is ordinary,
though:
- an `ssh -X` session dropping;
- `Xephyr` or `Xvfb` being closed under the game;
- a DCV session ending.

Each ends the game with one line and exit status 1. Killing a private Xephyr under the title screen
printed `X connection to :58 broken (explicit kill or server shutdown).` and nothing else. A game in progress is lost,
where a save-on-quit would have kept it, and settings changed since the screen was last closed are
lost too.

What to do:
- An error handler that records the error and returns, so `Present` and `PollEvents` can return it
  as the error they already have a path for. The shell and play loop then end the way a closed
  window does.
- An IO error handler cannot return: Xlib exits once it does. So it can only write the crash file
  4.35 introduces and say which server went away. libX11 1.7's `XSetIOErrorExitHandler` would let
  it unwind properly, but that raises the build floor, so it is not assumed.

### 2.79 An older build that saved a newer build's `prefs.json` deleted every setting it did not know — **DONE, before the `v0.2.0` tag**

`prefs.LoadFile` decodes the file over the defaults, so a key this build has no field for is
ignored on the way in, which is the design. `Save` marshalled the struct and nothing else, so the
same key was gone on the way out. The game saves when the remembered house changes and when a
high-score name is entered, so a player who ran `v0.2.0` after some later release would lose every
setting the later one had added, and `Validate`'s note for a newer file said those settings "are
kept as they are". Found while answering what a version number should promise about formats
(`docs/PLAN.md`, after the gate): a promise that newer files open in older builds is worth
nothing if the older build then writes over them.

`Save` now keeps the file `LoadFile` read and writes back every key it has no field for, at the
top and inside an object both know, such as `fixes`. Keys are matched without regard to case, as
`encoding/json` matches them, so a hand-typed `"Volume"` that was read into the field is not kept
beside the `"volume"` Save writes, where it would have won the next load. The version written is
this build's, because every field it knows it wrote in its own meaning.
`TestANewerBuildsSettingsSurviveASave` holds all of that, and that a second save changes nothing.

It also changes what `Version`'s comment advises. A change of meaning is better made as a new key
beside the old than as a bump, because an older build keeps a new key intact and reads a bumped
one in its own meaning. 2.74's physical keys are the first change that will have to choose.

### 2.80 A settings file that could not be read was reported twice, and a check told the tester to cause it — **DONE, before the `v0.2.0` tag**

`prefs.LoadFile` puts a read error in `Notes` and also returns it. `loadPrefs` in
`cmd/glidergo/prefs.go` printed the error, and `reportPrefsNotes` then printed the note, so one
problem took two lines, the second repeating the first. `loadPrefs` no longer prints it, and
`TestAnUnreadableFileIsANote` holds the fact that makes that safe: the error is always in a note.

It was found through `docs/windows-first-run.md`. Its auto-scale check said to point `-prefs` at a
directory that has no settings file, but `-prefs` names a file, and a directory is the unreadable
case. The check still worked, because unreadable settings fall back to the defaults and auto is
the default, but its first line of output was an error. It now says `-prefs none`, or a file that
does not exist yet.

---

## 3. Things the original did not have and a 2026 release is expected to have

### 3.1 High scores — **DONE, 1.7c**

The original does keep them (`internal/house.Scores`, and the board sorts on **score alone** —
`scores.go` `Sort`, `HighScores.c:281-318`, `docs/analysis/scoring.md` §7.2 fact 4 — while
*rooms visited* is what `levels[]` records and what the race ranks on). A release needs them
persisted somewhere sane — XDG `$XDG_DATA_HOME/glidergo/` on Linux, not next to the binary —
and it must not corrupt or crash on a truncated or hand-edited file.

**1.7c delivered all three.** `internal/scores.Store` writes one side-car per house under
`$XDG_DATA_HOME/glidergo/` (`-scores` moves it; `-scores none` plays without recording), the 22
shipped houses are read and never written, and `Store.Load` never fails: a truncated,
hand-edited or half-restored file yields a playable board plus one note per thing that had to be
worked around, on stderr. The board is reachable from the menu as well as by dying (H, which is
the original's Options > High Scores), and the game asks for a name and — for first place — a
banner, in the original's own two dialogs.

Two things a release still wants, both noted above rather than done: the footer is illegible
(2.56) and the dates the board already stores are never shown (2.57).

**The first paragraph's sort order is corrected in place. README and `world.go` were corrected in
PLAN's release gate, step 5.** README's "sorted on rooms visited before points" is now "sorted on
points alone", with Davis Station as the example: Kimmer's 40 rooms sit below Johnner's 38, on
17,500 points to 17,800. It also says a race ranks the other way round. "From 1995" is now
"from 1995–2000", because Art Museum and Davis Station carry 1996 rows and Sampler's and
Slumberland's top rows are 2000-05-11. `internal/game/world.go`'s comment on `Score` now says the
board sorts on it alone (`internal/scores.Sort`), and that the rooms stored beside it are what a
race ranks on. The other figures in that README paragraph were checked against the dumped boards
and hold: 20 houses carry rows, `Ozma` tops 13, and ImagineHouse PRO II's 108 rooms and 47,000
points (1995-07-03) is the best run on both measures.

**The picker footer shows the shipped "Your Name 10800" (2000-05-11) row as though it were the
player's best on the default house.** `footerBest` keeps "record:" as the top of the merged board
(name, score, rooms, year), so the picker and the H screen still agree on the top row. It adds a
"you:" part only for the best row that is *absent from `h.Scores`*. On a fresh install Slumberland
reads "record: Your Name 10800, 25 rooms (2000)" and no personal line. The label and the year are
what tell it from the player.

This breaks, on purpose, PLAN 1.7c's invariant (`:904-906`) and
`TestThePickerFooterShowsTheMergedBoard` (`internal/shell/scores_test.go:281-309`), which require a
footer from a side-car row to be pixel-identical to one from the same row in the house file. Both
are rewritten around the new rule, and `sets_test.go:587-599` is checked. Record, you, "(locked)"
and "[Original]" on one line come to about 94 characters, about 564 px, against 488, which is why
this lands with 3.7.

### 3.2 A game over is brutal by modern convention — **decision needed; neither option gates a tag**

Glider PRO is a 1994 game, and its one hard edge is the game over. A death costs a glider and
nothing else. `OffAMortal` respawns the glider in the same room at `EnteredRect`
(`mortal.go:98-110`, reproduced), so a per-room restart is already the 1994 death rule. The loss
comes when the last glider goes, three deaths into a game with none collected, and a new game
starts in the house's first room. In one player, S from the pause already keeps a checkpoint that
is never consumed (`store.go:198-206`), at the cost of high-score eligibility through
`ResumedSavedGame`. Anything added here defaults to off, because Stage 1's contract is exact
fidelity. It is worth deciding while new houses are designed, because a house can be designed for
either.

*Corrected in PLAN release gate step 5.* This item used to open by saying there were "no
checkpoints, and death sends you a long way back", and it proposed per-room restarts. Both were
wrong, for the reasons above, and the amendment that said so is folded into the paragraph.

**What is left to decide, and the two options on file.** Both are opt-in and off by default. Both
set **one shared ineligibility flag** that `TestHighScore` treats like `ResumedSavedGame`, which
arguably `-room` should set too (today it still reaches the board). Both are refused or forced off
in a race, as `raceRefusals` treats `-resume` and `-room`.

- **Assists** (`prefs.Assists`):
  - **Game speed** (50/66/75/100%) by scaling only the host's 60.15 Hz `TickCount`. It is not free.
    The live mixer runs on the wall clock and continuous sounds are timed to frames, so at 50% the
    helium hiss (0.133 s) is asked for every 0.266 s and is silent half the time, and about 25% of
    the time at 75%. The item has to choose: accept the stutter, retrigger on a wall-clock cadence,
    or slow the mixer.
  - **Extra or infinite gliders.** This changes simulation state (`Mortals`), so a replay recorded
    with it needs the flag.

  The wording of alert 1046 appears where the assist is switched on, as the original does before a
  resume (`Menu.c:319`), not after the game, because PLAN `:943` keeps it unreachable at game end.
  Slow motion was not ineligible on a slow 1994 Mac (`determinism.md:228`), so making it
  ineligible is a new choice that has to be argued. The rows need 3.9's pages, and they reverse
  `settings.go:113-115`'s policy openly.
- **"Continue from the last room".** The host keeps one automatic snapshot in `internal/saved`'s
  format, under its own key so that it never touches the manual save. It is taken once the glider
  has settled after `R.RoomNumber` changes: `Visited` is set on *leaving*, and a snapshot on the
  transition frame can resume at a slot mouth. It is written outside the frame loop (a full
  snapshot is `110 + 292×rooms` bytes, 155 KB for Teddy World), and offered on the port's own
  title screen. A multi-room practice picker comes later if at all, since it brings back 2.69's
  "which one?".

Both stay "decision needed" and are the user's call under the gameplay non-goal. Neither gates a
tag.

### 3.3 Accessibility — **planned, 1.7 onward**

Nothing in the original addresses it. The cheap wins: a colour-blind-safe option for the
glider/shadow contrast (which is already low on some backgrounds), a "hold instead of
tap" input option, and not relying on sound alone for any warning. The expensive one is
scaling text, which interacts with 2.1.

**The window-size half of text scaling is 2.1's auto scale**, which landed with the release gate's
step 4; this item's own text-scaling part stays open for low-vision players on small screens.

**Photosensitivity, as a measured claim (*next*).** A flash-scan test in `internal/replay` using
WCAG 2.3.1's real rule:
- relative luminance through `render.Palette`;
- a flash area over 25% of a sliding ⅓×⅓ field (8,533 px at 640×480);
- more than three opposing-transition *pairs* in any 30 frames fails.

The current five-script corpus passes. Only room changes cross the area threshold, at most two a
second. The outlet zap (384 px) and 2.19's flame (240 px) are 20–35× under it, so this settles
nothing about 2.19's default.

The corpus gains a **negative control that must trip it**: a toggle light switch under a
vent-pinned glider strobes the room at 2.5 Hz, and two out-of-phase switches at 5 Hz, unattended
and in faithful mode. Reproduced. No probed shipped room does it; Stage 2 houses and user houses
can.

A README note is then scoped to what was measured: the scripted runs plus a probe of the shipped
switches, and "a house can make the lights strobe". It does not say "safe". Follow-on: a
switch-over-vent lint note in 4.1.

**"Hold instead of tap" means toggle instead of hold (*later*).** Opt-in `latch_steer` and
`latch_battery`, applied as a filter inside `cmd/glidergo`'s `KeyPoll`. The band stays one-shot.
The filter needs explicit rules:
- an opposite press clears the other latch on the same frame;
- both keys physically down pass through as both, for the about-face;
- the battery latch clears when `BatteryTotal()` reaches 0, or a new battery (or helium) engages
  at once.

These are unit-tested directly, because replay scripts bypass the filter. Any future recorder
(4.39) or pad layer (2.3) must record what `KeyPoll` *returned*. It helps endurance, not dexterity:
the number of presses and their timing are unchanged. It is prefs-only until 3.9's pages exist.
2.72 matters here, because a split repeat pair would read as a tap.

**Music volume separate from effects (*later*).** `music_volume` 0..7, default 7. `Mix` scales the
music channel's sample before the sum, and the master volume stays where it is.
- At 7 it is byte-identical to today (demo mix digest to frame 1775: `913492dc2deb5974`).
- At 0 it matches `-music=false` (`329040786a50acdb`).
- Both digests are pinned in a test, plus a music-only unit test like `TestVolumeScales`.

It needs wiring through `Validate`, the legacy import default of 7, the R reset (`settings.go:283`),
`ApplyPrefs` (`main.go:1061`), `Engine.SetMusicVolume`, and a flag next to `-volume`. It also needs
a decision on whether 0 makes `MusicAvailable` refuse the score. It is pitched as a standard
option, **not** a clipping fix: the default's clipping is unchanged and faithful
(`engine.go:122-125`), and 4/7 only takes it from 6,853 to 5,835 samples. The row waits for 3.9.

### 3.4 There is no way in to the game — **DONE, 1.7a; the credits DONE, 1.7c**

No title screen, no house picker, no options screen, no credits. 1.7 is scoped as "the
shell" and owns all of it. Flagged here because the credits screen is not optional: the
five house authors in item 1.2 and both illustrators must be named in the shipped build
regardless of how the asset question is resolved.

**1.7a delivered the way in**: `glidergo` with no arguments is now a title screen with a
menu, a house picker over all 22 houses (with each house's shipped high score and its room
count), and an About box. The options screen is 1.7b.

**1.7c finished the credits.** There is a credits screen — About, then C, which the About box
says on its last line — and it names John Calhoun, Casady & Greene, all five house authors, both
illustrators and which houses belong to whom. It is the only screen in the port with no original
to be faithful to: the 1994 game credits nobody anywhere a player can see.

It is data, not a screen full of string literals. `internal/credits` embeds `credits.txt`, and
that file is pinned against `GliderPRO/README.md` by its own tests — a name on the screen that is
not in the README fails, a house the README credits that the screen does not fails, and a house
moved to the wrong author's row fails. `internal/shell` then checks that every name in the data
reaches a placed line inside the panel, so a credit cannot be lost to a layout that outgrew the
screen. That chain is the point: an obligation discharged by a literal inside a drawing function
is one nobody ever diffs against its source.

### 3.5 The original could be translated and this port cannot — **note; Stage 2 at the earliest**

Every string the 1994 game shows a player comes out of a resource: `GetIndString(150, index)`
through `StringUtils.c:321-327`, with `STR# 150`'s 51 strings and `STR# 160`'s alongside it
(`docs/analysis/scoring.md` 7.9.2). That is not decoration — it is the standard Mac localisation
mechanism, and it means a translator could ship a Glider PRO in French by editing a resource fork
and touching no code. Index 6 is `room`, 7 is `rooms`, 8 is `Click Mouse or Hit a Key to Exit`.

This port hard-codes English at every site. That was the right call for Stage 1, whose contract is
fidelity and whose strings are mostly transcriptions with a resource id in a comment beside them,
but it is a capability the original had and this does not — and the longer it is left, the more
sites there are. The cheap version is one package with the shipped strings as data (the shape
`internal/credits` now uses for the credits) and a language preference; the expensive part is that
the port's own strings — the status band, the settings screen, the About box — are longer and more
numerous than the original's 51, and several are composed with `fmt.Sprintf` in an order a
translator may need to change.

Worth deciding at Stage 2 rather than later: the new houses will have names, banners and room names
of their own, and those are content a translator would also want.

### 3.6 A house may leave the player in a room with no way out, and a faithful port cannot tell them why — **note; the linter half is 4.1 at Stage 2, the player-facing half is 3.2's decision**

A player reported being unable to get out of Slumberland's basement and asked whether the port had
lost a staircase. It had not: this is the house's design, and `internal/game/basement_test.go` now
pins it. Both basement staircases stand over a four-pixel updraught that is the only lift in the
room reaching their 112x32 trigger box, both of those vents are authored `initial 0`, and each is
switched on from somewhere else — "Good Night"'s from the thermostat in "Anabell Lee" next door,
"Going Up? No?"'s from the one in "Switch Me", two floors up in another suite. Walk down the stairs
before finding the plate and there is no way back up. The authors knew: the rooms down there are
called "Going Up? No?", "How do you get out of here?" and "Don't Ask, Just Get Out!".

Two separate things follow, and neither is a fidelity change:

- **The house linter (4.1) should be able to answer the question the player asked.** Reachability is
  computable from the data — for each room, which exits exist, which are gated by an object state,
  and which switch anywhere in the house writes that state. A house where a room's only exit is
  gated by a switch that room cannot reach is worth reporting, not as an error (Slumberland ships
  that way on purpose) but as a fact an author of a *new* house at Stage 2 almost never intends.
  The same pass gives the port a real answer to "is this a bug?" for any house, instead of one
  person reading object tables by hand, which is how this took an afternoon.
- **Nothing tells the player.** The 1994 game's answer was the manual and word of mouth, and neither
  ships with a download. This is 3.2's territory — an optional hint that defaults to off — and the cheapest useful
  version is not a hint system: it is that a player who has lost the last glider in a room with no
  reachable exit is in a state the game already knows how to detect, and could say something about
  once 3.2 decides whether the port is allowed to be kind.

Deliberately not fixed: the vents stay off, the trigger box stays 112x32, and the basement stays a
trap. Stage 1's contract is the 1994 behaviour, and this *is* the 1994 behaviour.

### 3.7 The house picker says nothing about what a house is, and points a newcomer nowhere — **planned, soon after the next tag; one commit with 3.1's footer half**

**Three changes and one warning.**

- **Drop "(locked)" from the picker footer.** It is the 1994 editor's write-protect bit, this port
  has no editor until Stage 5, and it shows on 15 of the 22 originals, including the default
  Slumberland. It has not been reported, but "locked" reads as "unplayable" in modern games, and
  nothing a player does depends on it.
- **The first-launch nudge goes on the opening status line, not the footer.** "new to Glider? try
  Demo House, the 1994 tutorial, or Open House." The splash preselects New Game on Slumberland (383
  rooms, dark rooms, a known basement trap), so a newcomer who presses Return never sees the
  picker. First launch is not detected today, so a `FirstRun` bit comes from prefs through the
  `Host`. "No prefs file" clears after the first save, so a sturdier gate is "no house has a
  side-car score yet".
- **The house's banner, where there is one.** Up to two wrapped, ellipsised scale-1 lines in the
  empty band at y≈362–384. Banners use `\r` separators and run up to seven source lines, so `\r`
  becomes a space, `internal/scores/entry.go:423`'s `wrap` moves somewhere shared, and `fit()`
  ellipsises. **13 of the 22 originals have no banner**, and of the nine that do, five say what
  the house is (Demo House, Empty House, California or Bust!, Titanic, CD Demo House), so this is
  worth less than it sounds.
- **The footer's record and the player's own best are split** (3.1's amendment). This is why it is
  one commit.

The layout: banner lines in the empty band; line 396 holds the scores line, which dropping
"(locked)" makes room for, and "[Set]" goes when it is crowded; line 416 stays the keys. The
`houses` fidelity hash is re-recorded once, PLAN Stage 2's "one set draws the identical picker"
sentence is amended on purpose in the same change, and `docs/screenshots/house-picker.png` is
retaken. It is not a fidelity breach: the original's dialog had no footer.

### 3.8 A player's own progress per house — **note; after 4.39's recorder**

`internal/records`: one JSON side-car per house under the data directory, keyed on house name as
saves are (`saved.Store.Path`), with `HouseHash`, `TimeStamp` and the gliderGo version as gates. It
records:
- how many times the house was finished;
- the fastest *fresh* finish in simulated frames, with its replay beside it (resumed runs are
  excluded, because `Frame` resets to 0 on resume and a script cannot carry a save's object state);
- the best rooms;
- the union of rooms ever entered, resumed runs included.

Categories split on players, fixes, assists and resumed-or-fresh.

The picker shows a finished tick and "seen N" **only when a record exists**, so the `houses` hash
holds on a fresh profile. The denominator is never `nRooms`: Teddy World has 401 of 531 rooms
reachable, so a meter over 531 can never fill. It uses `house stats`' reachable count, frozen per
house hash, or no denominator at all. `finalStanding` lives in package `main` (`race.go:529`) and
moves somewhere shared.

Two more pieces:
- `glidertool replay` gains a finished/died/quit verdict and in-game seconds, which is the gap its
  output actually has.
- An opt-in frame timer is a top-level display pref next to `scale` (`show_timer`), not a new
  `extras` block, because it only draws pixels. `fixes` stays for 1994 defect corrections and
  `assists` for rule changes that make a run ineligible.

### 3.9 The settings screen is full, and the next row has nowhere to go — **planned, before the first new row lands**

`settings.go:50-56` says a fifteenth row does not fit, and nothing enforces it: a fifteenth row
would draw over the footer with no test failing. The rows the register backs are 3.3's latch and
colour-blind options, 2.56's colour (once it is its own flag), 2.1's fullscreen (possibly one more
step on the magnification row), and 3.2's assists if they are decided. `KeepRealTime` gets no row
until something reads it.

**Two or three pages, not four:** say Controls | Game and sound | 1994 fixes. Tab cycles pages
instead of closing the screen, and Esc still closes and saves. **Every page name stays on screen**
as a strip like the picker's, or this brings back exactly what `settings.go:17-24`'s "one list, not
five panes" argues against.

The fixes page carries a "1994 exactly / modern / custom" preset over the four existing
`prefs.Fixes` rows. A two-state preset over per-item rows needs the third state, and the item needs
a rule for what R does to the fixes. 2.56's colour joins the preset once 2.56 is its own flag.

This reverses two written decisions, which are amended in the same change: `settings.go:17-24`,
and PLAN `:842-847`'s "offers less than the file holds". The `settings` reference hash is
regenerated. That is routine, because the screen has no 1994 original. A test holds every page's
last baseline above `setFootV`, and that test can land now, alone.

---

## 4. Tooling and content

### 4.1 A house linter — **DONE, 2.0; except 2.38's link-removes-star check, which is 2.38's amendment**

*Status, as checked in PLAN release gate step 5:* 34 checks in `glidertool house checks`, and the
22 originals lint to 637 notes, 48 warnings and 1 error, which are CONTRIBUTING's numbers. The one
check the list below asks for and the linter still lacks is the link that removes a star (2.38).

`glidertool house check` already round-trips and sanity-checks a house. Stage 2 authors new
houses, and the failure modes it should catch first are the ones the shipped houses
already contain: transit objects whose link points at a room that does not exist, transit
objects that are linked from nowhere, destination rooms with no staircase to arrive on,
and (2.14) custom trigger sounds that do not load. A house that fails to link is not a
crash in the original — the player just cannot get out of the room — which is precisely
why a linter is worth more than a runtime check.

**`internal/house/lint.go`, `glidertool house lint` and `glidertool house checks`.** Twenty-nine
checks at three severities, and the severities are the part that took the work. The 22 shipped
houses produce 634 notes, 48 warnings and **one** error between them, and that shape is the
design rather than an accident: a released game has to lint the originals clean enough that a
`-fail error` step is worth putting in CI, while still saying the true thing about each defect.
So the calibration went the other way round from the usual — the checks were written first, run
over the corpus, and then each class was argued down to the severity the *originals* justify.
189 dangling links across the corpus cannot be errors. Slumberland's inescapable basement is the
original's design (commit `acafec7`), so `stairs-unpaired` cannot be an error either.

Three things are worth keeping from the writing of it:

- **`sound-id` started out blaming the author for a bug of ours.** The check fires when a
  `kSoundTrigger` names a `snd ` the house does not carry, and it found 20. Splitting them
  against `assets/extracted/sound/houses/manifest.tsv` row by row gave 13 real ones (In The
  Mirror names `snd ` 10000 twice; Teddy World names `snd ` 3000 eleven times and carries no
  `snd ` resources at all) and 7 that are gliderGo's fault — the five MACE 6:1 resources our
  extractor skips, across three houses, which 2.49 has been carrying as a note since 1.6. A
  linter that reports the second kind as an authoring defect is worse than one that does not
  report it, because the author will go looking for a mistake they did not make. Hence
  `SoundStatus` with three answers instead of a boolean, `audio.Bank.Unreadable` to carry the
  extractor's `status` column through, and `sound-unreadable` as a separate check whose text says
  out loud that this one is ours.
- **What the finding says matters more than that it fires.** `sound-id` does not say "missing
  sound". It says that `LoadTriggerSound` looks only in the house's own resources and never the
  application's (`Sound.c:265-303`), and that `CreateActiveRects` then composes the room with
  **no hot spot at all** — so the trigger is not merely silent, it cannot be touched. That is a
  different bug report from the one an author would otherwise file, and it is three lines of
  comment rather than a day of theirs.
- **The catalogue is held to the code by an AST test.** `LintChecks()` is what `house checks`
  prints and the only place a reader can look up an id they have just seen. A hand-maintained
  list like that rots silently the first time somebody adds a check, so
  `internal/house/lintcatalogue_test.go` parses `lint.go`, collects every `l.add` call site's
  severity identifier and id literal, and requires the two directions to match *and* the
  declared severity to be the worst any call site uses. A behavioural test would only have
  proved the ids it happened to trigger exist; this proves there are no others. It cost one
  call site its `sev` variable — the dangling-link branch is now two literal `l.add` calls
  sharing a `const` format string — which is a fair price for the guarantee.

**Two checks were considered and deliberately not written.** *Static reachability* — "can the
player get from the first room to a star" — is tempting and would be wrong: `Room.Openings` is a
dead field (0 in all 4,070 rooms) and the real openings are computed at run time by
`DetermineRoomOpenings`, so anything static would be guessing at the geometry. Reachability
belongs to Stage 2's other acceptance criterion, the scripted headless playthrough, which
answers it by actually playing. *Objects outside their room* — the `ForceRectInRect` clamp in
`GetObjectRect` — is deferred because those per-type rects live in `internal/render`, which
`internal/house` cannot import; it wants either a callback like `PictSize` or to live in the
renderer, and neither is worth doing before a new house needs it.

Two smaller repairs fell out of the corpus run. `internal/render` had its own copies of
`ExtractFloorSuite` and `GetRoomNumber`; they now delegate to `internal/house`, and the dead
`kNumUndergroundFloors` beside them is gone. And `internal/game`'s link predicates and
`internal/house`'s are now pinned to each other by `internal/game/linkagreement_test.go`, which
walks all 144 object codes rather than trusting two lists to stay in step — the transcription
keeps its `Objects.c` comments, the map keeps its lookup, and neither can drift.

### 4.2 A bug-report format, so that a released game can be debugged — **DONE, 1.5b**

A public release gets reports like "sometimes the glider sticks in the third room", which
nobody can reproduce. `internal/replay` and `glidertool replay` are the answer: a script is
a house, a seed, a start point and a keystroke log, and it reproduces the frame exactly on
any machine with no display, no sound card and no timing dependency.

```
glidertool replay -house "CD Demo House" -frames 600 -room 4 -where 420,20
glidertool replay -trace -o bug.trace bug.script
glidertool replay -digest bug.script        # one token, for "do we still agree"
glidertool replay -wav bug.wav bug.script   # ...and what it sounded like
```

**The audio is part of the format as of 1.6**, which is not a footnote either. "The sound cut out"
is exactly the class of report that is unfalsifiable without one: the trace's `snd=` column names
every sound the frame asked for and which of the three channels it landed on, the footer counts the
requests that were refused and the ones that were cut off mid-sample, and a second digest covers
the mix itself — every sample of every channel, the priority policy's choices and the clip point.
`-wav` writes the same samples to a file, which on a build host with no sound card is the only way
to *hear* a replay at all. Two properties make the pair trustworthy: the recorded path mixes
exactly `SamplesPerFrame` per frame from an engine with no clock, so two runs agree byte for byte;
and the mix digest is a checksum of the WAV payload, so `sha256sum` on the file anybody can produce
answers to the sixteen characters in the report.

Silence is a *different run* and the trace says which it was in its header. A sound trigger whose
sound does not load gets no hot spot at all, so `sound off` can change the composition — and even
where it does not, it changes the digest, because a sound request is a decision the simulation made
at a frame. A port that stopped playing the toaster would otherwise pass every determinism test in
the suite.

Three things had to be pinned to make a run reproducible, and they are the three a bug
report cannot leave to the machine: the random seed, the calendar clock (`DrawCalendar`
reads it — 2.31), and the frame clock (`World.TickCount` nil means the frame number *is* the
time). Everything else falls out of that.

What the trace deliberately does **not** contain is a hash of the screen. A screen digest
would change on every legitimate rendering improvement and tell you nothing about where; the
trace records the frame counters, the dirty-rect counts, the glider's mode and rect, the
score, and `World.RandSeed`, which is the strongest determinism claim available and the only
field in it that is not a fact about the picture. `TestDifferentSeedsAreDifferentRuns` exists
because a harness that pinned the randomness too would pass every determinism test and
reproduce exactly one run of the game.

**A deviation from `docs/PLAN.md` to record.** The plan asks for the seven exit kinds to be
tested "leaving Demo House's first room". That is not possible: Demo House contains four of
the seven kinds in the whole file, and its first room has none of them. CD Demo House — also
one of the 22 originals — has all seven, so `internal/game/exits_test.go` picks rooms by
inspection there instead (28, 192, 193, 4, 32, 69). The seven arrival rects were derived
from the C by hand before the port was run, and each case carries its arithmetic, because a
number that matches the current code is a snapshot and a number that matches the C is a
specification.

Two further notes on that test, both of which cost time to find:

- **`GliderInRect` is containment, not intersection.** A glider merely touching a staircase
  trigger box walks straight past it, so a probe has to place the whole 48×20 glider inside
  the box. A placement that looks right and does nothing is this.
- **The last `Present` of a frame is the one that counts.** A transition presents once per
  wipe strip — 116 or 160 times inside one game frame — and the side doors change room from
  inside `HandleInteraction`, which runs *before* `HandleGlider`. Sampling the first present
  of the transition frame therefore reports the glider offset by `-RoomWide` but not yet
  stepped, and misses the second of the frame's two renders entirely.

### 4.3 The golden trace reported the digest as the first divergence — **DONE, 1.5d; the second half is 1.8**

`duct.trace`'s line 4 is a digest of all 601 samples, so it changes whenever any sample does.
The failure report printed the first differing line, which therefore named line 4 on every
single failure and buried the frame that actually moved. In 1.5d that cost real time: the first
reading of the divergence was "the digest changed", and the event — a switch thrown on frame
224 — was 225 lines further down. The report now skips comment lines when choosing the
difference to print, counts them separately, and says so explicitly in the one case where only
the header moved, which means the digest function changed rather than the game.

**The second half is a design limit and is owed at 1.8.** One 600-frame script can only pin the
subsystems the glider happens to visit, and bringing an object to life can move the glider. That
is exactly what 1.5d did: room 5's ceiling switch arms a transporter, so from frame 225 the
trace is in room 70 and six of the toaster's nine bursts are gone from it. Nothing was lost this
time, because the toaster's arithmetic is pinned closed-form by `TestLaunchVelocityFromHeight`,
but the next stage that animates something may quietly walk the trace away from whatever it was
illustrating. 1.8 should carry several short scripts, one per subsystem, instead of one long one
— and the general rule is the one 2.38's survey also taught: a test that follows the game rather
than asserting about it has to be able to say what it stopped covering.

**1.9 added the second script and the shape the rest should copy.** `testdata/two.script` is a
two-player race, and its golden is only half of what it carries:
`TestTheTwoPlayerTraceSeesTheHandshake` reads the same run as *statements* — two idle freezes of
`IdleFrames-1`, one limbo wait, one refused wall, one joint crossing, six deaths out of one
counter, one terminal `PlayerIsDeadForever` — and finds every frame number by searching rather
than writing it down, so a physics improvement moves the numbers and leaves the rules asserted.
That is the answer to this entry's own complaint: the golden says *what changed* and the
companion test says *what broke*, and a script whose glider wanders somewhere else fails with a
sentence instead of a line number. The remaining scripts (one per subsystem) are still owed.

### 4.4 A comment naming a test that does not exist is worse than no comment — **DONE, 1.5e and 1.6; the linter is DONE, 2.1**

`internal/render/srcrects.go` had promised since 1.5c that "`TestStripRectsTileTheirSheet`
checks each array against its sheet's declared bounds, and a transcribed table is what that
test can actually catch a mistake in". No such test existed. That promise was the stated reason
the nine frame-strip rect tables were written out longhand instead of computed from a stride —
so its absence made the choice pointless: a hand-typed table with nothing checking it is
strictly worse than a loop.

Written in 1.5e rather than deleted, because 1.5e added `BandRects` to exactly the set of tables
the comment describes. `internal/render/srcrects_test.go` now derives each expected rect from
`stripBounds` — transcribed from `StructuresInit.c`'s `QSetRect` calls — while the tables under
test come from its *loops*, so the two independently-transcribed sources have to agree and a
single typo cannot satisfy both. `TestEverySrcRectFitsItsSheet` adds containment for the
irregular tables, which catches the failure a golden image hides: `Surface.Copy` clips silently,
so a rect past the end of a sheet gives a short or empty blit rather than a panic, and an object
that animates to nothing looks like a logic bug in its handler.

One more of the same class was found and fixed at the same time: `bands.go`'s header named
`TestClampedBandSurvivesTheKillTest` where the test is actually
`TestBandBouncesOffTheWallAndSurvivesTheKillTest`.

**The sweep was run by hand at 1.6 and found two more.** This codebase's comments carry a lot
of load — most of what is known about the original lives in them — and a named-but-absent test
is a specific, mechanical failure: it tells a future reader that a property is pinned when it is
not, and it does so most convincingly exactly where the property is subtle. Three lines of
`grep | comm` over the whole tree:

```bash
grep -rhoE "\bTest[A-Z][A-Za-z0-9]+\b" --include=*.go internal cmd | sort -u > /tmp/mentioned
grep -rhoE "^func (Test[A-Z][A-Za-z0-9]+)" --include=*_test.go internal cmd | sed 's/func //' | sort -u > /tmp/defined
comm -23 /tmp/mentioned /tmp/defined
```

- `cmd/glidertool/glidertool_test.go` named `TestCheckFindsUndefined`; the test is
  `TestCheckFindsUndefinedWhat`. Comment corrected.
- `internal/game/grease_test.go` and `internal/game/bands_test.go` both deferred to
  "`TestRenderFrameOrder` in play_test.go", which had never been written — and it was the one
  worth having, because both of those tests explicitly decline to pin the position in the
  sequence on the grounds that something else does. Written, in `play_test.go`: it parses
  `render_frame.go` with `go/parser` and asserts the fifteen receiver calls in `RenderFrame`
  come out in Render.c's order, that the two per-player calls are P1 before P2, and that
  flames and stars are the two arms of one `if` on `EvenFrame` rather than two statements.
  A source test rather than a behavioural one because registration order *is* z-order and the
  behavioural equivalent needs a room with a mirror, grease, a pendulum, a flame, a dynamic, a
  flying point, a sparkle, a shred and a band overlapping at once.

**The linter is still owed at 1.8**, because the sweep above is a thing someone has to remember
to run. `glidertool` should grow a `lint` subcommand that does it, run from `make check`. Three
false-positive shapes this tree already contains, which are the whole design problem — a naive
`\bTest[A-Z]\w+\b` matcher reports all three:

| Shape | Where | Why it is not a dangling reference |
|---|---|---|
| A wildcard standing for a family of tests | `internal/house/house.go:12` says `TestCorpus*` | Matches the eight `TestCorpus…` tests in `corpus_test.go` |
| A hyphenated line break | `internal/render/locale.go:97` ends a line with `TestFrame-` | Continues `CountsMatchTheSrcTables` on the next line, and that test exists |
| The C's own identifier | `internal/game/play.go:208` cites `TestHighScore` | `HighScores.c:374`, not a Go test |

So the check has to skip a match followed by `*`, **join** a comment's lines before matching —
not merely ignore a trailing `-`, since a hyphen split hides a dangling name exactly as well as
it hides a valid one — and exempt names that appear inside a citation of the original source.
The last one is the interesting one: `TestHighScore` will become a real Go test's name at 1.7,
at which point the false positive resolves itself and the exemption stops being needed — which
suggests the rule should be "a name cited alongside a `.c` filename is the C's", not a
hand-maintained allowlist.

**Built at 2.1, inside 4.12's checker rather than as a `glidertool lint` subcommand.** A test was
the better home for the reason 4.12 gives: a linter that reports its findings to a human is a
linter whose findings can be read and not acted on, and the whole class of defect here is
"somebody did not do the manual step". It found four more, all of them renames: the one this
entry predicted, plus three that had gone stale in the six stages since.

Two of the three design notes above survived contact and one did not.

The wildcard form needed widening: prose writes the family with an **ellipsis** as well as a star,
and this entry's own line 2510 is the proof — "the eight `TestCorpus…` tests" in the sentence, and
`TestCorpus*` in the code span beside it. The checker accepts `*`, `…` and `...`.

The hyphen rule was right to insist on joining and wrong about where the difficulty is. The
trailing-`-` case is not only a line break: line 2511 above *quotes* `TestFrame-` mid-sentence,
as an example of the convention, and no join can rescue it because the rest of the name is prose
about a different thing. So a name ending in `-` is treated as half an identifier and never as a
claim, which is a weaker rule than "join and then check" and the only one that does not fail on
this table.

**And the prediction was wrong.** `TestHighScore` did not become a Go test at 1.7. It became
`World.TestHighScore` — a *method*, transcribed with the C's name, cited in `cmd/glidergo` and
called in `internal/game`. A matcher looking for `func Test…` cannot see a method declaration, so
the reference reads as dangling now for the same reason it did then, and the exemption is still
needed. This is worth recording as a small lesson about deferring a false positive on the grounds
that it will resolve itself: what resolved was the *absence* of the thing, not the shape of the
reference to it.

The suggested rule — "a name cited alongside a `.c` filename is the C's" — was therefore rejected
in favour of the hand-maintained allowlist it was meant to avoid. Two of the five entries are not
about the C at all (`docs/CITATIONS.md` uses a placeholder while explaining the convention; two
other names are quoted in this file as errors that *were* found, and correcting the quotations
would delete the findings), so the proximity rule would have had to be joined by an allowlist for
the rest — and an allowlist with a proximity rule in front of it is strictly harder to reason
about than an allowlist. What makes the list tolerable is 4.12's tier 4: every entry is checked
for still being needed, so the one on this line will fail the build the day `TestHighScore`
becomes a test.

### 4.5 A bug report could not say that two machines drew different pixels — **DONE, 1.5f**

`Result.Digest` hashes the sample trace, and the trace records dirty-rect *counts*. That is the
right choice for the reason 4.2 gives — a script mailed in by a player has to survive the next
sub-stage's intentional pixel changes — but it means an entire class of fault is invisible to it.
A flame stuck on one cel, a pendulum swinging the wrong way, a filmstrip baked with the wrong
frame masked into it, a confetti cloud emerging top-first: every one of those registers the same
rect as the correct version, so `w2m` and `b2w` do not move and neither does the digest.
Everything 1.5f added is in that class by construction.

`Result.Planes` closes it with three hashes taken once, after the last frame: the back map, the
work map and the screen. Which of the three moved localises the fault before anyone opens a
debugger — `back` is the composition, so a difference there is a locale drawn differently before
anything moved; `work` is the composition plus everything that animated over it; `main` is the
work map as the dirty rects delivered it, so a difference in `main` with `work` identical is a
dirty-rect bug, the right pixels composed and the wrong ones copied. `glidertool replay` prints
the three on a `screen` line beside the digest.

They are deliberately **not** folded into the digest and not written into the trace, which is
what makes it safe to have both: nothing on disk holds the value, so a deliberate pixel change
costs nothing, while two runs of one script are still required to agree.
`TestTwelveHundredFramesTwiceAreTheSamePicture` is 1.5f's acceptance clause and asserts exactly
that, and it picked up two facts about the frame protocol on the way: a room where nothing
animates ends every frame with the work map identical to the composition (so a renderer that
forgets its back rect fails there), and a room with a cuckoo in it never does, because the
animated families' cels come out of a filmstrip and register no back rect at all.

### 4.6 The screens a player meets first were the only ones no automated check could draw — **DONE, 1.7a; the corpus half in 1.8a**

Every check in `make check` covered the game and none of them covered the way in, and the reason
was structural rather than an oversight: the game's frames come out of the null backend
(`-frames 3 -dump`), and a title screen drawn on the null backend still needs a backend. So
`internal/shell` was built to compose into a surface with no window anywhere in the picture, and
`glidergo -shot FILE -shot-screen splash|houses|about` writes one frame of it as a PNG with no
display, no audio, no house and no game opened.

`make headless` now renders five: the three screens against the real asset tree, and the two
first-run layouts against a directory that does not exist (2.6). It is cheap — no process
outlives the write — and it caught three defects in one sitting that the unit tests could not,
because all three were about what the composite *looks* like rather than what any one function
returns: the illegible dim (2.50), the fallback title screen laid out underneath the menu panel
(2.6), and the selection bar on a disabled item (2.51).

Two notes for the stages that inherit it. The unit tests still assert the things a test can
assert and should keep doing so — `TestEveryScreenDrawsWithoutArt` counts cream and black pixels
and fails on a blank screen, `TestHouseLabelPlacementAndClamping` checks that no ink reaches the
right edge — because a PNG nobody opens is not a check. And `-shot` is the natural place to hang
a golden-image test for the shell in 1.8, which would make these five files a corpus rather than
an artefact; it is deliberately not that yet, because 1.7b, c and d will all change these screens
and pinning them now would only generate churn.

### 4.7 Nothing on disk said what the game is supposed to look like — **DONE, 1.8a**

4.5 gave a report three plane hashes and 4.2 gave it a script that reproduces on any machine, and
between them they answer every question about pixels *except* the one a released game needs
answered: has this build changed since the build somebody looked at? `Result.Planes` compares a
run against another run, always both freshly computed, so a divergence introduced deliberately
and a divergence introduced by accident are indistinguishable — both simply become the new
answer. The only reference was human memory, and human memory of a 640x480 room is the thing
2.50, 2.6 and 2.51 all proved unreliable.

`internal/fidelity` is that reference: per-frame hashes of the three index planes, as text, one
row per frame, checked in. 601 rows for the existing 600-frame duct script, and six rows for the
shell's screens — which no replay script can reach, because `internal/shell` is a separate
program from the simulation by design. `make fidelity` runs it and, on a machine that has the
assets, treats a *skip* as a failure, because a silent skip is exactly the outcome this package
exists to prevent.

Four decisions in it are worth keeping:

- **Hashes, not images.** `git diff` on hashes says *which frames* moved, which is most of the
  finding: one row is a frame, every row is the palette or the view, three rows in the middle of
  a transition is a wipe that got faster. The same corpus as PNGs is 300MB of binary in the
  history of a repository a player is meant to be able to clone. What a hash cannot do is show a
  human the difference, so a failing test re-runs the one frame that diverged and writes it out
  as three PNGs — `Snapshot`, the failure path and only the failure path.
- **The recorder cannot move a pixel.** `internal/replay` grew one hook, `Watch`, and it copies
  the planes at each `Present` and hashes once per frame in `flush`. That ordering is the whole
  trick: `Present` fires up to 161 times inside a transition frame, so hashing there would cost
  161 sweeps of three planes to keep one answer and would pin the middle of a wipe as if it were
  the frame. A test drives the hook with a watcher that scribbles `0xFF` over everything it is
  handed and proves the trace, the planes and the mix all come out unchanged.
- **The shell's screens are hashed as index planes, not as PNG bytes**, or the corpus would be a
  corpus of Go's `image/png` encoder; and the version string the About box draws is pinned to a
  constant, or tagging a release would invalidate six rows for no reason. This closes the promise
  4.6 left open, one stage later than it guessed and by hashing the composition rather than the
  file `-shot` writes.
- **The end state is not the last frame.** Recording the corpus turned this up immediately and it
  had been true since 1.5f: after the frame loop, `PlayGame` runs the unconditional arcade block,
  which blackens the scoreboard band and blits it to the screen (`Play.c:551-593`), and
  `CopyRectsQD` has already restored `Back` over `Work`. So `Result.Planes` describes a picture
  that includes twenty rows no frame ever presented, over an erase the frame itself did not have.
  Neither number is wrong and both are worth having — `Planes` is what the process was left
  holding, the corpus is what the game showed — but a reader who assumed they were the same
  thing would have spent an afternoon on it. The tests now say so out loud.

What it does not close: 4.3's second half is still open, because one script still only pins the
subsystems this glider visits, and the demo-replay codec (`demoType`) is 1.8b.

**1.8b added the codec and deliberately did not add a corpus row for it.** The attract-mode script
then ran 1,775 frames — 1,775 rows, about 320 KB — against a stream that lived in gitignored
`assets/extracted/`, so the corpus would have been large, unbuildable from a fresh clone, and a
checked-in assertion that the physics gap of 2.18 was the correct behaviour. It is a determinism
test instead: the same script twice, compared frame by frame. The row was to become worth having on
the day the demo flew to the end of its stream, and it was to be a short prefix even then.

**That day has come, and the row is not written yet.** Since 4.24 the script runs 3,432 frames and
consumes all 1117 records (2.18's amendment). `assets/extracted/res/demo/128.bin` has been tracked
since `8f39609`, so a demo corpus row is buildable from a fresh clone. It should still be a short
prefix. 2.18 names it as one of the two acceptable next steps.

### 4.8 The demo determinism test needs a 1994 asset, and it should not have to — **note; a 1.8c candidate**

`internal/replay/testdata/demo.script` is the strictest determinism test the harness has, and it
skips on a clone that has not run `make assets` — the stream it replays is an extracted resource
and `assets/extracted/` is gitignored. That is the right decision for *this* script, whose whole
point is the 1994 recording, but it means the demo *path* — the cursor, the equality compare, the
five missing guards of `GetDemoInput` — has no coverage at all on a bare clone.

**Amended: the skip no longer happens.** `assets/extracted/` has been committed since `8f39609`
(`.gitignore:49`), `res/demo/128.bin` with it, so `demo.script` runs on a fresh clone and covers the
demo path there. The round trip below no longer fills a coverage gap. It is still the only test
proposed that would tie the recorder to playback.

The missing half is a round trip, and every piece of it already exists. `World.RecordDemo` returns
a `demo.Recorder` wired to `GetInput`'s four log sites, so a replay script with `at` lines can
*record* a stream; feeding that stream back through the `demo` keyword should produce the same
frames as the script that recorded it. It would need no asset, it would be a handful of records
rather than 1,117, and it would fail loudly on exactly the class of bug that is hardest to see
here: a one-frame offset between the frame a key was logged on and the frame playback applies it
to. The one thing it would *not* catch is a difference between `GetInput` and `GetDemoInput`, since
the recording came from the former — so it complements the shipped stream rather than replacing it.

Two known asymmetries have to be written into the test's expectations rather than treated as bugs
(both are 2.63's territory): an about-face records as a plain right press, because `LogDemoKey(0)`
is above the both-keys test, and a held band key records a code every frame although only the
first fires. So a recorded-then-replayed session is not guaranteed to reproduce the session — it
is guaranteed to reproduce *itself*, which is what a determinism test needs.

### 4.9 The compiler is allowed to fuse a multiply and an add, and two of ours were fused on arm64 only — **DONE for the three sites found; the standing check is a note**

4.5 gave a bug report the vocabulary to say "these two machines drew different pixels". This is the
first thing found that could actually make them, and it was found by looking rather than by a
failure, which is the part worth writing down.

The Go spec permits an implementation to evaluate `a*b + c` with a single fused instruction, keeping
the full product and rounding once, where separate operations round twice. It is not a bug and
there is no flag that turns it off: it is explicitly allowed, and the only thing that forbids it is
an explicit conversion of the intermediate value. amd64 codegen does not take the licence. arm64
does, and `macos-latest` is arm64, so the CI matrix is the first place this port has ever been
compiled by a backend that uses it.

Three sites had it, and `GOOS=darwin GOARCH=arm64 go build -gcflags=-S ./... | grep FMADD` is how
they were found — grep the assembly, not the source, because the source is where the fusion is
*invisible*:

- `internal/render/surface.go`, `FillPolyPatOrGray`: the scanline crossing, `aH + t*(bH-aH)`, twice
  (the compiler unrolled it). The result is then `ceil`ed to a pixel column, so a last-bit
  difference on a crossing that lands exactly on a boundary moves a whole column — `ceil(5.0)` is
  5 and `ceil(5.000000000000001)` is 6. Four callers in `objectdraw.go` draw furniture shadows into
  the background of nearly every room in the game, so an affected polygon would differ in every
  screenshot of that room for the rest of the run.
- `internal/audio/bank.go`, `stepFor`: `rateHz*RateDen/RateNum*FixedOne + 0.5`, truncated to a
  16.16 step. A rate on a boundary would resample by one 65536th on an Apple Silicon Mac and not
  on an x86 one, and every mixed sample after it would differ.

Both are fixed, and differently on purpose. The polygon fill is now exact integer arithmetic: the
vertices are integers and the row centre `y+0.5` doubles to an odd integer, so the crossing is a
rational with a small numerator and denominator and the `ceil` is an integer division — there is no
rounding left to disagree about, on any architecture, for ever. `stepFor` keeps its float64 and
gains a conversion around the product, which forces the intermediate rounding and forbids the
fusion; that is the cheap fix, and it is the right one where the value is a one-off at load time
rather than a per-pixel inner loop.

**What this was not.** It was not the cause of the CI failure that prompted the look, and saying so
is the point of this paragraph. arm64 was emulated here by patching each site to call `math.FMA`
explicitly and running the pixel and audio suites — `internal/fidelity`, `internal/render`,
`internal/game`, `internal/shell`, `internal/audio`, `internal/replay` — and every golden matched.
A 90-million-edge sweep over integer vertices confirms the arithmetic genuinely diverges (1,235,147
crossings land in a different column under FMA), so the hazard is real; it is simply that no
polygon any shipped house draws, and no sample rate any shipped sound carries, sits on one of those
boundaries. The defect was latent, not active.

Which is exactly why it was worth fixing anyway. A golden-hashing test suite that is only
*accidentally* architecture-independent cannot tell anybody that it is: the next vertex a new house
introduces, or the next edit to that inner loop, turns a latent difference into a wrong screenshot
that reproduces on one developer's machine and not another's — and 4.5's report would faithfully
record two different digests with nothing to explain them.

The open half is the standing check. Nothing stops the next float multiply-add from appearing, and
the grep above is a one-liner: a `make fma-check` that fails when `GOOS=darwin GOARCH=arm64`
assembly contains an `FMADD` outside a deliberate allow-list would keep the property instead of
re-establishing it by hand. It is left as a note rather than written now because the allow-list is
empty today and a check with an empty allow-list is indistinguishable from a grep, and because the
honest place for it is beside 4.5's cross-machine digest rather than bolted onto `make check`.

**The standing check, when it is wired in.** `make fma-check`, run after `cross` in `check`. It
builds `./...` — the whole module, not "physics and replay": the simulation is integer-only, and
both past fusions were in `render` and `audio` (putting the old `stepFor` back gives an `FMADDD` at
`audio/bank.go:200`). It builds for `darwin/arm64` and `windows/arm64`, the second because it
compiles the win32 and waveOut files. It uses `-gcflags=-S` and fails on
`\bFN?M(ADD|SUB)[SD]\b` at any `file:line` not on an allow-list, which is empty today. It also
fails if the build fails or prints no assembly. About 15 s cold, 0.2 s warm.

The recipe above, `grep FMADD`, misses FMSUBD, FNMSUBD and FNMADDD, and CHANGELOG repeats it.
Both are corrected. The goldens already run on arm64 in `ci.yml`'s `macos-latest` job; they just
cannot see a latent fusion.

---

### 4.10 Two tests spelled a path the way Linux spells one and asserted it was universal — **DONE; the CI half is done too**

The first CI run on a machine that was not this one went red, and it went red in the one job that
had never executed anywhere: `go test ./...` on `windows-latest`. `go build` and `go vet` passed on
the same runner, and the failure was reported as `Process completed with exit code 1` and nothing
else, because a step that fails without `continue-on-error` skips every step after it and the job
log had expired before anyone opened it.

Two tests could not pass there, and neither depended on anything about the runner — no audio
device, no path length, no scheduler, no Defender. Both are the same mistake: a path literal
written the way this host writes one, compared against a value that the standard library spells
differently on Windows.

`internal/scores/store_test.go` set `GLIDERGO_CONFIG` to `/tmp/glider-portable` and then checked
that the scores directory was inside it. The two sides of that comparison arrive by different
routes. `prefs.Dir` hands the variable back verbatim, so its side keeps the forward slashes;
`scores.Dir` goes through `datadir.Dir`, which returns `filepath.Join(d, sub)`, and the last thing
`filepath.Clean` does is replace every slash with the platform separator (its own documentation says
so). So on Windows the test compared `\tmp\glider-portable\scores` against a prefix of
`/tmp/glider-portable\` and failed — for its separators, not for anything the two packages disagreed
about. The fix is one line: derive the root with `filepath.FromSlash`, which is the identity here and
on macOS, and compare against that one value. Both occurrences of the literal had to move together,
because fixing only the `t.Setenv` would have relocated the failure to the verbatim check two lines
down.

`internal/shell/library_test.go` asserted `House.Rel == filepath.Join("sub", "Nested.glh")`. `Rel` is
an `io/fs` name, not a host path: `Discover` gets it from `fs.WalkDir`, which composes every name
with `path.Join`, and `internal/shell/library.go` does not reference `filepath` once. So `Rel` is
slash-separated on every platform and the expectation had to be the literal `"sub/Nested.glh"` — as
the `t.Errorf` on the next line had been saying all along, which is the tell that the `filepath.Join`
was a slip rather than a decision. Worth being explicit about the direction of this fix, because the
other direction is tempting and wrong: localising `Rel` inside `Discover` would produce a name
`os.DirFS` refuses to open, since `os.dirFS.join` rejects any name containing a backslash. A house
whose `Rel` had one in it would be a house nobody on Windows could load.

Neither of these is a bug in the game; both are bugs in tests, and the production code was right in
both places. That is the useful thing about them. They are the first evidence that the test suite had
absorbed an assumption about its host, and the reason a cross-platform matrix earns its cost even
when the port itself is clean.

Two follow-ons were folded in. `internal/audio/waveout_windows_test.go` burst twelve one-frame
writes into a queue `waveDepth = 8` deep and asserted nothing was dropped, with a comment claiming
the queue was deeper than the test was long — which it never was. It only passed when the pump
goroutine happened to drain four blocks mid-burst. It is now `const frames = waveDepth`, which fits
the channel even if the pump never runs at all, and still exceeds the `waveBlocks - wavePrefill`
blocks that are idle at open, so it forces `acquire` to reclaim every time instead of usually. That
test skips on any machine without an output device, so it was never the CI failure; it would have
been the first failure anybody saw on a real Windows desktop.

And the diagnosis itself was the other finding. An exit code with no log is not a bug report, so the
`native` job now tees the run to a file, copies the `FAIL` lines and their `file.go:NN:` messages
onto the job's summary page, and keeps the whole transcript as an artifact — all three only `if:
failure()`. The summary page renders in a browser with nothing installed, which matters when the
person who has to read it does not have `gh`. Deliberately *not* done: making the step
`continue-on-error`. The step being red is the entire signal.

Left as a note: the same three steps would help the `check` job, which runs the much larger
`make check`, and the sweep that found these two only covered comparisons written as `!=`/`==`
against a literal. A vet-style check that flags a path literal containing `/` in the same expression
as a `filepath` call would cover the class instead of the two instances.

### 4.11 `-wav` is clocked by wall time, so audio cannot be compared across machines the way pixels can — **note; found by the first Windows run**

The first run of the Windows build produced a paced `-wav` capture that differed from the Linux one
in about 124,000 byte positions, which reads like a platform regression and is not one. Two *Linux*
runs of the identical command differ from each other in about 122,000. Same magnitude, same cause —
and those two counts are themselves not repeatable, which is the whole of the finding.

The cause is that `internal/audio`'s mixer is clocked by real time rather than by frames. Under
`-bench` that is visible in the summary line: the same 900-frame command produced 0.9 s of audio on
one machine and 0.5 s on another, because the capture is as long as the run *took*, not as long as
the run represents. A paced run is frame-locked in total length — 600 frames gives 19.9–20.0 s
every time, three measured runs landing on 887,420, 887,970 and 888,060 bytes — but the sample at
which each sound starts still depends on where the pacer happened to be when the mixer asked, so
two captures are never byte-equal.

Why it matters beyond tidiness: `-shot` renders are byte-identical across operating systems and
architectures, which is what makes them a regression check a stranger can run and CI can diff — the
first Windows run leaned entirely on that property. Audio has no equivalent, so there is no cheap
way to notice that a platform's sink has started resampling, mixing at the wrong rate, or dropping
a channel. The `sound -- N requests, N played` counters catch a refusal but say nothing about what
came out.

The fix is to clock the mixer off the frame counter: advance it by exactly `rate/30.07` samples per
frame rather than by elapsed time, with the device consuming from a buffer that is allowed to run
ahead. That makes `-wav` a function of (house, seed, frame count) alone, which is reproducible, and
lets `make check` carry a hash of one the way `internal/fidelity` carries pixel hashes. It does not
change what a player hears, because a paced run already produces a frame-locked length; it changes
whether the file is the same file twice.

Not started. The counters and the clipping figures are worth keeping either way — they are what
distinguished "the device refused this" from "the mix was too loud" during the Windows run.

### 4.12 About 17,800 citations into the 1994 C, and nothing had ever checked that one resolved — **DONE, 2.1**

The port's entire claim is that it behaves like the original, and that claim is made in pieces —
function by function, each backed by a pointer at the C it was read from. There are about 17,800 of
those pointers, naming about 18,300 lines and ranges across all 93 files of upstream `94fed96`. Not
one of them had ever been checked against the file it names.

That is a worse gap than the count makes it sound, because of what a citation is *for*. "The original
clears `mode` before the altitude check" is a thing a reader has to take on trust; the same sentence
with `Interactions.c:412` after it is a thing a reader can go and disagree with. So a citation that
does not resolve is not a small blemish — it is the difference between evidence and assertion,
wearing the costume of the former. It also costs somebody else's time rather than ours, which is the
category of defect this file exists to take seriously.

`internal/citations/citations_test.go` sweeps every `.go`, `.md`, `.py`, `.sh`, `.txt` and `.yml`
file in the tree and resolves three classes of claim. The 1994 C, in four tiers: the file exists;
every line number and range endpoint is inside it; a full path names the directory the file is
actually in, and every range runs forwards; and coverage both ways — all 67 of upstream's `.c` files
are cited somewhere, and every entry on the exemption list is still needed by something. This
repository's own `path:line` references, which is the same promise about a tree where files really do
get renamed, plus the coverage half of that — every `internal/` package has to appear on the map
`README.md` draws, because a package missing from it breaks no link and still reads as complete, so
the only way a reader finds out is by concluding the thing they wanted is not in this project. And
the test names the prose quotes, which is 4.4's linter, finally built (see below). Plus the invariant
the rest lean on: every written mention of the upstream commit names the same commit.

**Twenty-two defects, on prose that had been reviewed repeatedly.** Nine numbers past the end of
their file, six file names that do not exist, one path naming the wrong directory, two references to
documents of ours that are not there, and four test names that had been renamed out from under a
comment. Every out-of-range number was fixed by reading the C and finding the line the claim was
about, not by lowering the number until it fit — `Banner.c:205-236` because that is where
`DisplayStarsRemaining` ends, `RectUtils.c:210-216` because that is where `QSetRect` is. A citation
that resolves to the wrong place is worse than one that resolves to nothing: nothing announces
itself and the wrong place does not.

Two of the twenty-two matter beyond their own line. `Room.c` cited to 1215 in a 1206-line file means
that reading was done against a differently converted copy, so the other numbers from the same
sitting were suspect too — which is the kind of thing a count of one defect does not tell you and a
sweep does. And `PlayerControl.c` was a file an early draft invented, which has never existed in any
version of Glider PRO; it is now named in exactly one place, to say so.

A twenty-third was not a citation defect but caused four. `GliderPRO/Prefix.h` sits at the top of
upstream's tree rather than inside `Headers/`, so it was missing from the `cp` recipe in every
document — which means the four citations to `Prefix.h:1` could not resolve on any machine that had
followed the instructions, for four stages. It is in all four recipes now and in `.gitignore`. The
count of citations in the documentation was also wrong by nearly half: five places said "about
9,500", which is roughly the full-path form alone.

Three decisions inside it worth keeping:

- **The colon is required.** `Banner.c:205` is a citation; "as `Banner.c` 300 times" is prose about a
  file. Matching the space-separated form would fail against a 237-line file on writing that was
  correct, and a checker that is wrong about correct prose gets answered by rewriting the prose.
- **A floor, not an exact count.** The figure moves whenever anybody writes a paragraph, so an exact
  assertion is one that gets its number bumped without being read. The failure a floor actually
  catches is the one nothing else can see: an edit that breaks the matching turns 18,000 assertions
  into a vacuous pass over an empty slice, and the suite stays green. That does not arrive at 17,000.
- **Exemptions carry their reason, and the reasons are checked for still being needed.** Twenty-one
  names upstream does not have are legitimately cited — Apple, Win32 and X11 SDK headers, two
  metasyntactic placeholders, and `PlayerControl.c` named to record that it does not exist. An
  exemption list is the one part of a checker that cannot fail, so it is the part that rots: an
  entry nothing needs any more is a name the next person is quietly free to get wrong.

It needs the C, which is not in this repository, so the tiers that read it skip when it is absent and
print the three clone commands. A skip is a weak thing to rely on, so `.github/workflows/ci.yml` has
a `citations` job that clones upstream at the pin on every push — and that job fails if any check
skipped, because `go test` prints `ok` for a package whose every test skipped and a mistyped path
would otherwise leave it green while proving nothing. The classes that need no C never skip.

`docs/CITATIONS.md` is the reader-facing half, and its §6 is the part worth writing down: what a
green run does **not** promise. Above all that a citation is about the right thing. `Player.c:1234`
resolves, and line 1234 is `FlagGliderNormal(thisGlider);` — whether that is what the sentence beside
it claims, no sweep can say.

Two things this found and did not fix, both deliberate. `docs/analysis/stage15-raw/` is referenced by
three Go files and four documents and is gitignored, so those references are dead on every clone;
5.8 owns it and it cannot simply be repointed, because the "releasePolish N" numbering they quote
exists only there. It is on the exemption list now, which at least makes the deferral
machine-tracked: the day that numbering lands in a committed file, the checker asks for the exemption
back. And two of upstream's 25 headers are cited nowhere — `About.h` (10 lines) and `Play.h` (13) are
prototype-only companions to two of the most heavily cited `.c` files in the tree, with no `typedef`,
`#define` or `struct` between them. Coverage is therefore asserted for the 67 sources and not for the
headers.

### 4.13 The documented command lines nobody had run, and the two that hang or say nothing — **four DONE, 2.1; nine more DONE, 2.4 (the flag ordering, `docs-check`, the walkthrough, the help lines, the `GOOS` guards, `project.Releases`, PLAN's architecture map, the unsigned-binary warnings, `release.yml`'s four `sed`s), and with that the section is closed — and `gh release create`, the last command line nobody had run, has since run three times, for `v0.1.0`, `v0.1.1` and `v0.1.2` (5.4)**

A companion sweep to 4.12, over a different kind of claim. 4.12 checks pointers into the C; this one
is about the instructions this project gives a *person*: the commands in the READMEs, the ones the
binaries' own `-h` offers, and the sentences that say what happens if you are missing something.
None of those had been run since they were written, and a documented command line that does not work
is the most expensive wrong sentence in a repository, because it is the one a newcomer meets in their
first five minutes and the only evidence they have about whether the rest is worth reading.

Four were fixed, and they are worth listing separately because they fail in four different ways.

**`glidertool replay -script -` could not run.** Replay's own usage text offers it as the way to see
the script format — "A script is line-oriented; `glidertool replay -script -` prints one to copy" —
and it failed with `no house to replay`. `-script` writes the resolved script out and exits, so it
describes a run rather than performing one and needs no house; the check for a house was simply
above it. Moved below, and the blank house is filled in with the one the error message already
suggested, so what gets printed is a template that *runs* rather than one whose first line is `house`
with nothing after it.

The interesting part is why it survived six stages: **a test asserted the broken behaviour.**
`TestReplayNeedsAHouse` was written as `run([]string{"replay", "-script", "-"})` on the reasoning
that `-script` was a cheap way to reach the house check without starting a game. It was — and so the
suite pinned the defect, in green. That is a failure mode worth naming: a test that reaches the code
it means to test *through* an unrelated flag silently takes on that flag's behaviour as a
requirement. The case now asks its own question, under the name
`TestReplayNeedsAHouseToRun` — the four words it was missing — and the case it used to occupy is
`TestReplayPrintsATemplateThatRuns`, which pipes the printed template straight back in and replays
it, because a template that parses and does not run is documentation a reader stops trusting.

**`glidergo <house>` was silently ignored.** `cmd/glidergo` called `flag.Parse()` and never looked at
`flag.NArg()`, so the most obvious command a player can type — the name of a house — parsed, ran, and
showed the title screen. That is indistinguishable from the house having been *refused*, which is the
failure mode the `-resume`/`-room` checks twenty lines below are worded to prevent ("Accepting both
would silently ignore one of them, which is the failure mode that costs an hour of wondering why").
The principle was already written down; nothing had applied it to the arguments.

Accepted rather than refused, which is the one place this entry chose ergonomics over strictness, for
two reasons that both point the same way: the 1994 program was a Mac application and a house was one
of its *documents*, so opening one by naming it is the original's gesture rather than a new
convenience — and it is the form an operating system uses when a file type is associated with a
binary, which is what a double-clicked `.house` has to become on Windows and macOS. Two houses, or
`-house` and a bare name together, are still refused, and the message quotes both strings back,
because a name that lost its quotes (`glidergo CD Demo House`) is the commonest way to get there and
seeing the three fragments is what tells the player what happened.

**`CGO_ENABLED=0` builds a game that cannot be quit.** `CONTRIBUTING.md` said that without
`libx11-dev` "the build still succeeds and produces a binary that cannot draw". Both halves were
wrong in opposite directions. The build does not succeed — the X11 backend asks `pkg-config` for
`x11`, so it stops with `Package x11 was not found in the pkg-config search path` — and the binary
that *does* come out of a no-cgo build does something worse than not draw: it selects the null
backend, whose `PollEvents` returns only the events a script gave it, so nothing can ever deliver
`EventQuit` and the run draws frames into nothing until somebody types Ctrl-C. No output, no window,
no error. That is the first five minutes of anybody who cannot install the dev package, which is
exactly the reader that sentence was written for.

Now refused, with the three flags that give such a build something to finish. And the condition is
`endlessHeadlessRun(backendName, o)` rather than a comparison against `backend.Name`, because that
constant is `"x11"` on the host `go test` runs on: a guard written against it directly is a guard no
test can reach, and this one's failure mode is a hang, which is the single failure a test suite
cannot report on its own. The prose now says what the build actually does, and points at `make
doctor`, which already checked for `x11.pc` and was not mentioned.

A smaller trap found underneath it, worth recording because it cost a wrong measurement: the first
attempt to reproduce the missing-`libx11-dev` failure ran `go build` with `PKG_CONFIG_PATH` pointed
at nothing and **succeeded**, because Go's build cache had the cgo action already. It takes `-a` to
see it. Anybody trying to verify a toolchain-dependency claim on this project will hit the same
thing.

**"is DISPLAY set?" was the answer to one of three questions.** The X11 backend's only diagnosis was
`x11: cannot open display (is DISPLAY set?)`, and `XOpenDisplay` reports failure with no reason at
all, so the environment is all there is to go on. Three people read that sentence. One has no
graphical session and wants to be told about `-shot` and `-frames`. One has `DISPLAY` set and an X
server that refused — a wrong value, or no `xauth` cookie, which is what `ssh` without `-X` looks
like — and being asked whether `DISPLAY` is set, when they can see that it is, reads as the program
not knowing. And one is on Wayland with no XWayland, which on a 2026 desktop is the likeliest of the
three and the one the old message sent furthest wrong: there *is* a session, it is graphical, it is
running, and what is missing is a package nothing named.

`platform.DisplayAdvice(display, wayland string) string` now answers each separately, and it lives in
`internal/platform` beside `Expand` for the reason `Expand`'s own comment gives: the backend needs
libX11 and a display, a function over two strings needs neither, and so every branch is reachable
from `go test` on the machine this port is written on. The wording claims no certainty — each branch
says what was observed before what to do about it, so a reader in a fourth situation can see which
of the three they were mistaken for.

**Two documented commands were checked and are fine**, and are recorded so the next sweep does not
re-derive them. `make doctor` runs, installs nothing, and reports `libX11 dev (x11.pc)` along with
python3, a C compiler, a display and a sound device — it is the `preflight` target this entry nearly
added before reading the Makefile. And the README's Windows paragraph is current: the "never been
run" caveat was retired when the build was actually run on Windows Server 2025, and the two
remaining gaps it names — nobody has played it with a keyboard, `windows/arm64` has never run — are
both still true. The `never run` lines in `CHANGELOG.md` are history and stay as written.

**Still open, in rough order of who they cost.** Each is a claim or a command, and each is cheap.
Four were closed in 2.4 — the flag ordering, the `docs-check` target, `CONTRIBUTING.md`'s
walkthrough (whose bullet is half 5.4's and stays open for that half) and the Makefile's two help
lines — and their bullets are kept below with the amendments that closed them, because what each one
cost is the part worth remembering:

- **`glidertool <sub> file -flag` does not work**, because Go's `flag` package stops at the first
  non-flag argument, and several documented lines put the file first. A `partitionArgs` in
  `cmd/glidertool/main.go` that lifts flags out from behind positional arguments would make every
  ordering work. The ordering that fails is the one a shell user writes by habit. *(2.4: the one
  documented line that still had it — the README's `house build my-house.txt -o my-house.house`, in
  the first fenced block anybody reads — was reordered when `house stats` was added to that block.
  Reordering the documents is not the fix; the parser still refuses the habit, and this bullet stays
  open. It is now the case that every command line in `README.md` and `CONTRIBUTING.md` puts its
  flags first, which is a thing a `docs-check` target could hold.)*

  *(2.4, **DONE**. `internal/cliargs.FlagsFirst` rewrites one command line into the order `Parse`
  wants, and all sixteen FlagSets in the tree go through it. It walks the arguments and asks **the
  FlagSet itself** whether each flag takes the token after it — `IsBoolFlag`, an attached `=`, a
  name nobody registered — then emits the flags, a `--`, and the files. Asking rather than keeping a
  list of this project's own value-taking flags is the whole design: a new `-whatever` needs no edit
  there and cannot be got wrong there either.*

  *Measuring the cost first changed what this looked like. The bullet said "does not work", which
  undersells two of the three failures. `house dump f.house -o out.txt` answers `house dump takes
  exactly one house file` — **confidently false**, said to somebody who gave exactly one house file,
  and the reader's next move is to start quoting the file name. `house stats "Demo House.house"
  -tier tutorial` is worse: it prints all eighteen rows **with no tier column at all**, having
  silently demoted `-tier` to a file name, and only then stops with `open -tier: no such file or
  directory`. A flag that is ignored and then blamed for not existing on disk is not a refusal, it
  is a wrong answer with an exit code attached.*

  *It is a package and not the proposed `partitionArgs` because `cmd/glidergo` had the same defect
  and nobody had noticed: `glidergo Slumberland -scale 2` was refused with `one house at a time: 3
  were named (Slumberland, -scale, 2)`, which is 2.1's own fix for this entry reporting the
  arguments it could not reorder. That is the argument for sharing — two callers whose failure mode
  is *moving* an argument rather than refusing one — and it is the opposite call to `internal/project`'s
  nine-line formatter, which is duplicated on purpose.*

  *What it deliberately does not do is become a parser. An unknown flag stays in the flag stream, so
  a typo is still `flag provided but not defined: -teir` and not "no such file"; `-h` still reaches
  `flag.ErrHelp`; `-q true` still leaves `true` a positional, because that is what `flag` does with
  it and a reordering that "fixed" it would eat a file name in one ordering and not the other; and
  `-o --` still takes `--` as the value. Every one of those is a test.*

  *The sweep is the part that will outlive the fix.
  `cliargs.TestEveryCommandLineInTheTreeIsReorderedBeforeItIsParsed` reads the tree's own AST,
  finds every `Parse` on something it knows to be a FlagSet, and requires the arguments to have been
  reordered — because fifteen call sites are fifteen chances for the sixteenth subcommand to copy a
  neighbour's first ten lines and not its eleventh, and a behavioural test over today's fifteen
  would pass forever while that happened. It found a sixteenth site nobody had counted:
  `tools/packassets`, which takes **no** positional arguments and so had no ordering to get wrong,
  and which silently ignored one — `packassets assets/levels` packed `assets/extracted` and reported
  success. Now refused, which is this entry's own principle applied to the one program it had not
  been applied to.*

  *The `docs-check` bullet below is unaffected and still worth doing. Every documented command line
  still puts its flags first; the difference is that this is now a house style rather than a
  requirement, so a reader who departs from it gets what they asked for.)*
- **`make check` does not check the documents.** 4.12's `citations` job checks the citations and
  this entry's four defects were all found by hand. A `docs-check` target that runs the documented
  command lines — the ones in fenced blocks that begin with `bin/glidergo`, `bin/glidertool` or
  `make` — would have caught three of the four. The hard part is that some of them need a display
  and some write files, so it wants a marked subset rather than a scraper.

  *(2.4, **DONE**. `make docs-check` runs `tools/docscheck`, and it is in `make check` between
  `fidelity` and `cross`. It finds **38** command lines in `README.md` and `CONTRIBUTING.md`, runs
  **19** of them in 1.2 seconds, and prints the other nineteen with a sentence each saying why not —
  the same bargain `check-caveats` already makes for the rest of the suite.*

  *The marked-subset-versus-scraper question turned out to be a false choice, and the answer is
  both: scrape to **find** the lines, a table to **decide** about each one. What the marking cannot
  be is a mark in the document, because that is a notation in prose a reader has to step over, and
  it puts the reason in the one place there is no room to say it. The reasons are the good part —
  "opens a window and plays until the player quits", "rewrites the committed pixel corpus, which is
  the thing the corpus exists to stop happening by accident" — and they live beside the rule, in Go,
  where they are printed under the line they are about.*

  *The recogniser is **every non-empty, non-comment line inside a `bash` fence**, and not the three
  prefixes this bullet proposed, because a prefix list is a list that can be silently incomplete.
  `tr '\r' '\n' < GliderPRO/Sources/Player.c` and `go test ./internal/fidelity -update` are
  documented instructions as much as any `make` line is; a reader who runs them is following the
  page exactly as written, and a check built from three prefixes would neither run them nor say that
  it had not. The fence tag was already the marker this bullet wanted: every `bash` block in both
  documents holds commands and nothing else, and every block of sample output is fenced without a
  tag. That convention was kept by hand for six stages. It is load-bearing now, and
  `TestSampleOutputIsNotMistakenForACommand` is what holds it.*

  *Both directions are checked, and the second is what decides whether this is still a check in a
  year. A documented line no rule names fails — that is how the table keeps up with the documents. A
  rule no documented line names **also** fails, because a rule left behind by a reworded command
  reads exactly like coverage and is none: the line it was written for shows up as unclassified,
  while the orphan sits there looking like a considered decision. Both halves are `go test
  ./tools/docscheck` as well as the target, deliberately — they need no binaries, no assets and no
  display, so the person who adds a line to the README hears about it from the test suite they were
  already running, and the failure prints the two spellings and the file to put one in.*

  *Three smaller decisions, each of which was a bug in the first draft. The lines run in a scratch
  directory of symlinks to the repository's top level, so `assets/extracted/houses/*.house` resolves
  while `house build -o my-house.house` cannot leave a stray file in a working tree — `make check`
  has to stay something that can be run on a dirty branch. `.git` is deliberately not among the
  symlinks: a check with a writable path to the repository's own history is a check that can lose
  work, and all the exclusion costs is that `make` inside the scratch tree computes its version as
  `dev`. And every line gets two minutes, which is not about speed — the nineteen take 1.2 seconds
  between them — but because one of the four defects above **was a hang**, and a check that inherits
  a hang has inherited the worst version of it: a `make check` that never ends.*

  *What it found on its first run: nothing, which is the expected answer and not a disappointment.
  2.1 fixed these same lines by hand a stage ago; what this buys is that the next four are found by
  a machine instead. It did catch two stale sentences on the way in — `CONTRIBUTING.md`'s prose copy
  of `check`'s prerequisite list, which goes out of date every time that list changes and did again
  here, and `README.md`'s `tools/` row, which still described the directory as standard-library
  python3 after two Go commands had moved in.*

  *What it still does not cover, stated because a check's caveats are the part that gets forgotten:
  `make run` and `glidergo -levels assets/levels` open a window and play until somebody quits, so
  both are skipped and `make smoke` is what exercises the blit path; `house dump ... | less` runs,
  but under a pipe `less` is `cat`, so what is checked is the dump and the exit status and not the
  pager; and the two `git clone` lines have never been run by anybody on this host, which has no
  network at all. Nineteen of thirty-eight is the honest number, and the target prints it.)*
- **`release.yml`'s `sed` and `CONTRIBUTING.md`'s walkthrough are still unrun.** 5.4 owns the first;
  the second is the "your first patch" section, whose steps have never been performed end to end by
  anybody, which is the same class of defect as the four above and the one most likely to be met.

  *(2.4: the walkthrough half is **DONE**, 5.4 still owns the `sed`. The five `glidertool` lines
  under `CONTRIBUTING.md`'s "Houses" heading were run end to end for the first time — dump
  Slumberland to text, build it back, lint it, print the check catalogue, measure it against the
  small tier — and all five work: the round trip reports `383 rooms, 2996 objects, 134150 bytes`,
  the lint ends on the dangling-link note and the wrong-background warning that the 22 originals are
  calibrated to allow, and `stats` says 210 of 383 rooms are reachable from `Welcome…`. They are not
  left as a one-off: `docs-check` above runs all five on every `make check`, which is the point of
  having built it.*

  *(2.4, the `sed` half too, and it turned out not to need 5.4 at all: the whole "Package the
  archives" step runs on this host unchanged. Extract its `run:` block, dedent it, give it `VERSION`
  and the three `GITHUB_*` variables `sed` interpolates, and point it at the `bin/cross/` layout
  `make cross` already writes — all six archives pack, all six pass their own embedded-asset greps,
  and `SHA256SUMS` comes out with bare filenames. The link rewrite does what its comment claims: the
  README's three screenshots become `/raw/` links and its thirteen documents `/blob/` links, every
  one pinned at the 40-character SHA rather than at a branch, and the `! grep -q '](docs/'` assertion
  underneath finds nothing left behind. (Had it found something, the step would have gone on:
  `set -e` ignores a command inverted with `!`, so that assertion could never fail. It has been an
  `if` since 1.4 was done, `release.yml:533-536`.) The CRLF `sed` and the `VERSION` `sed` in the
  notes step were run the same way. Four `sed`s, none of them unrun now.*

  *The cost of rehearsing it is 150 MB in `/dist/`, which `.gitignore` already excludes and says is
  for exactly this. What it still cannot reach is `gh release create`, which is 5.4's and needs
  github.com — so the file's standing caveat stays, one step shorter than it was.*

  *Two notes for whoever reads this bullet next. This one named a section that does not exist — there
  is no "your first patch" heading in `CONTRIBUTING.md`; the walkthrough is under "Houses", and a
  pointer into our own documents that does not resolve is exactly what 4.12 sweeps for elsewhere. And
  the three lines under "A house that ships" are still not run by `docs-check`, for reasons that are
  covered rather than skipped: `make levels` is a target `make check` runs itself, `make levels-zip`
  rewrites a committed archive and `go test ./assets` is what checks its claim, and the third opens a
  window.)*
- **The Makefile's `## ` help lines skip `smoke` and `all`**, so `make help` lists neither, and
  `smoke` is in `make check`. Two lines.

  *(2.4, **DONE**, and it was two lines. `all` says it is the default target, because `make` on its
  own is the first command `CONTRIBUTING.md` gives and nothing said what it did; `smoke` says it is
  skipped with a note when `DISPLAY` is unset, which is the thing a reader of a green CI log needs to
  know about it. Worth pairing with the bullet above: `make help` is now itself a checked command
  line, and `make tools` lists the Go commands by `tools/*/main.go`, so `docscheck` appeared there
  without being named — which is the sort of thing that only works if the list is derived rather than
  written down.)*
- **`cross`'s caveats are not guarded on `GOOS`**, so a macOS reader is told what a Linux build
  cannot do. Same for the Windows and macOS gate commands, which name Linux paths.

  *(2.4, **DONE**. Three targets print a caveat, and all three of them were Linux assumptions worn
  as universals: `cross`'s last row, both halves of `check-caveats`, and `smoke`'s skip note. Each
  now asks `go env GOOS` first.*

  *What they said before is worth recording, because the two operating systems were wrong in
  opposite directions and only one of them looks like a bug. On macOS everything overstated: `cross`
  built a **darwin** binary with cgo on and printed it as `linux/amd64 +cgo … x11 backend (this
  host)` — a row whose shape was right and whose every fact was false — and `check-caveats` reported
  that cgo was off, or that libx11 metadata was missing, about a platform that has no backend to
  compile either way until stage 6. On Windows it understated, which is the direction `check-caveats`
  exists to prevent: it said the x11 backend was NOT compiled and that `build` had produced the null
  backend, of a build whose win32 backend needs no cgo, **was** compiled, and draws — and then
  blamed an unset `DISPLAY` for the window that did not open, of an OS that has no reason to set one.
  A caveat list that undersells a green run teaches people to stop reading it, which costs exactly as
  much as one that oversells.*

  *The bullet's second sentence names something that does not exist: there are no "Windows and macOS
  gate commands" in any document here — `grep` finds none, and the gate is one `make check` that runs
  everywhere. What it must have meant is these three printed caveats, and they are what was fixed.
  Recorded so the next reader does not go looking.*

  *Two fixes in passing. `smoke`'s old note said the blit path "is still covered by `make headless`",
  which is backwards — `headless` renders through the null backend, so the blit is precisely the one
  thing it does not cover — and that is the same defect as the Windows caveat, one line up. And the
  libx11 branch printed a hardcoded `linux/amd64 +cgo`, so a linux/arm64 host was told about somebody
  else's architecture; it uses `go env GOARCH` now. What did not change is behaviour: on Windows,
  `smoke` still skips rather than benching, because nothing in a Makefile can decide from outside
  whether a window would open there, and turning an honest skip into a possible flake inside
  `make check` would be a worse trade than the wrong sentence was. It now says that, and names `make
  bench`.*

  *How the new wordings were read, given that this host is Linux: `GOOS=darwin make check-caveats`,
  `GOOS=darwin make cross` and `GOOS=windows make smoke BIN=/tmp/x` print exactly what a reader on
  those machines sees, because all three branches ask `go env GOOS` and that answers the environment.
  All four values were run — linux, windows, darwin and a `freebsd` that stands for the default arm.
  Those command lines are in the Makefile's comments, because they are the only test these branches
  have and that is not obvious: CI never reaches any of them. `windows-latest` has no `make` at all
  and the macOS half of the same job calls `go` directly for the same reason, so the `native` job
  compiles and tests the Windows and macOS code without ever running the Makefile that describes
  it.)*
- **`project.Releases` has no caller.** It is the one constant in `internal/project` that nothing
  reads, which means nothing checks it either. Either the release notes and `-version` should point
  at it or it should go; an exported constant with no reader is a string that can rot silently, and
  this package exists specifically to stop that.

  *(2.4, **DONE**: it is gone, and the class of defect is now a test. Of the two options the bullet
  offers, "point at it" turned out to be the one that could not be taken honestly. No tag has been
  pushed (5.4), so the page the constant names has nothing on it — and the README's opening sentence
  is "there is nothing to download and no copy of the old game to find. Clone it and `make run`",
  which is **true today** and is the single best thing that paragraph could say. Pointing a player
  at an empty releases page, in a repository whose 4.13 is a list of documented instructions that
  did not work, would have been the same defect wearing a nicer hat.*

  *(Superseded: two tags are out, and the page has archives on it. The constant is back, read by
  `-version` and `-help` (5.4). The About box and the title screen still do not read it, because
  they are faithful to the original.)*

  *What the deletion cost had to be weighed, because the obvious objection is that 5.4 will want the
  string back within one command of tagging. It is one line, and the reason it is safe to lose is
  that the constant was not protecting anything: `TestEveryGitHubLinkIsOneWeMean` reduces every deep
  link to its repository root before judging it, so a hand-written `.../releases` in the README
  would pass with or without a constant to compare against. `internal/project` protects **Go**
  callers, and a downloads URL has no Go caller — the player holding the binary does not need to be
  told where to download it. 5.4's entry now names the two places the URL belongs when a tag exists.*

  *The general form is worth more than the one deletion. `TestEveryExportedConstantHereIsReadBySomething`
  parses `project.go` for its exported constants and sweeps every other `.go` file in the tree for
  `project.X`; a constant nothing reads fails, with the two options in the failure message. It found
  a second one on the first run, and that one was a false accusation worth keeping: `Module` has no
  external caller either, because `Home = "https://" + Module` is its only reader — and `Module` is
  checked, by `TestModuleMatchesGoMod`, and is the string every URL in the game is built from. So the
  rule is transitive: a constant counts if something outside reads it, or if a constant that counts
  is built from it. A test that had shipped without that distinction would have demanded a caller for
  the most load-bearing string in the package.)*
- **`docs/PLAN.md` §3's architecture map names directories that do not exist.** It was written before
  the tree settled and has not been revisited. Unlike the README's map, which 4.12 now sweeps,
  nothing checks PLAN's — and the two disagree, which is worse than either being wrong alone.

  *(2.4, **DONE**. Four nonexistent directories, and the pattern in them is worth naming: every one
  was a guess about how the code would be organised, made before it was written, and left standing
  after the code disagreed. `internal/ui/` became `internal/shell/`. `internal/assets/` was never
  written at all — the embedding lives in two hand-written files at `assets/`'s top, which is a
  better answer and is why nobody missed the package. `internal/game/objects/` and
  `internal/game/room/` were a decomposition the simulation never wanted; `internal/game` kept both
  in itself. A fifth line was wrong in the other direction: `internal/platform/backend/` exists, is
  the build-tag selector every binary goes through, and the map that claimed to show the port layer
  did not have it.*

  *There was also a stale `[DONE, never run]` on `win32`, a stage after it had been run on Windows
  Server 2025. A plan is allowed to be out of date about the future. Being out of date about the
  present is how a reader learns to stop reading it, and that one line undersold the single most
  expensive thing the previous stage did.*

  *Fixed by deleting the duplication rather than by synchronising it, which is the same call 4.26
  made about `make check`'s step list. The README's "What is in here" table is the exhaustive map and
  already has `TestEveryPackageInTheTreeIsOnTheMapTheReadmeDraws` holding it to the tree; PLAN's
  fence keeps only the layering the three rules under it depend on, plus the three directories that
  do not exist yet with the stage that owns each. Two maps of the same tree is the defect; one map
  and one diagram of it is not.*

  *The check is `TestEveryPathTheArchitectureMapNamesExists` in `internal/citations`, and it reads
  the fence as the indented tree it is: a line's path is its first field joined onto its ancestors'.
  A `[stage N]` marker exempts a line from having to exist — and requires that it does not, because a
  marker left on a directory that has since been written is the `win32` defect exactly. The one line
  that says "not committed" (`assets/levels/`, which `make levels` builds) is checked against
  `.gitignore` instead of against the filesystem: it is present here and absent from a fresh clone,
  and `make check` runs `test` five steps before `levels`, so requiring it would have made the
  result depend on whose checkout it ran in.*

  *Carried along: the README's map was only ever checked in one direction. Every `internal/` package
  had to appear in the table; nothing said a path in the table had to exist. Those are different
  failures and only the first one looks like a hole — a row for a directory that has been renamed
  away reads as complete and sends the reader somewhere that is not there.
  `TestEveryPathTheReadmeMapNamesExists` closes it, resolving the 34 paths the table names. A
  backticked token counts as a path if it has a slash and no space, which excludes the table's
  commands, build directives, resource types and the one 1994 filename with a space in it.*
- **Nothing in the release archives mentions SmartScreen, Mark of the Web, or Gatekeeper
  quarantine.** A player who downloads an unsigned Windows zip gets "Windows protected your PC" with
  no Run anyway button visible until they click More info, and a macOS build is quarantined outright.
  Neither is a bug and both look exactly like one. This is 5.4's release notes, and it is the single
  most likely reason a first-time player never sees the title screen at all.

  *(2.4, **DONE** as far as an offline machine can take it. Three places now say it: a new "Your
  computer will try to stop you, once" section in `release.yml`'s release notes, placed above the
  Windows section because it comes first in the reader's day; a block in the zip archives'
  `HOW-TO-RUN.txt`; and a shorter one appended to the darwin archives' `HOW-TO-RUN.txt` only. That
  last detail is the reason it is appended rather than written into the heredoc — the headless text
  is shared with `linux-arm64`, whose reader has no Gatekeeper and no reason to be told about one,
  and the three-way `if` above it is sorted by what the build can draw, which is a different
  question.*

  *Each of the three gets the click path and not just the name. `More info` → `Run anyway`, because
  the button a player needs is the one the dialog hides. For Mark of the Web, right-click the zip
  **before** extracting → Properties → Unblock, which is the form that clears it once instead of per
  file, with `Get-ChildItem -Recurse | Unblock-File` for somebody who has already extracted. For
  macOS, the fact that decides the outcome: Archive Utility propagates `com.apple.quarantine` to what
  it unpacks and `tar xzf` does not, because the attribute is on the archive file rather than stored
  inside it — so "unpack it in a terminal" is the whole fix, and `xattr -d` is only for a reader who
  already did it the other way.*

  *Not signing is stated as a decision rather than left to look like an oversight. A certificate
  costs a few hundred dollars a year and has to be issued to a named person or company, which is the
  same argument `internal/project`'s `Copyright` makes about "the gliderGo authors": that is a thing
  to do deliberately. (5.12, step 5, adds the rest of the price. SmartScreen reputation belongs to a
  signer or else to one file, so unsigned, every tag's binaries start with none, and a Defender
  false positive cleared for one tag can need clearing again at the next. The notes and
  `release.yml`'s header say so now.) Each of the three sections ends on the distinction that matters, which is that
  none of these checks looks at what is in the archive — `SHA256SUMS` does, and it is already
  published.*

  *What is unverified, and it is a different kind of unverified from the rest of `release.yml`. That
  file's standing caveat is about GitHub, which this host cannot reach; this is about client machines
  nobody here has. The Windows half needs no network at all, though, which is the useful discovery: a
  browser download is an ordinary file with one alternate data stream on it, so
  ``Set-Content -Path .\glidergo.exe -Stream Zone.Identifier -Value "[ZoneTransfer]`nZoneId=3"``
  and then a double-click in Explorer is the real test. The recipe is in
  `docs/windows-first-run.md` under "Rehearsing what a download adds, with no download", which is
  where the next person with a Windows box will look, and it is listed there as gap 4 — the `.exe`
  that ran on Server 2025 arrived over SSH and so carried no mark and met no SmartScreen. It is worth
  doing before the first tag: the notes say `Run anyway` is behind `More info` on the authority of
  documentation, and one double-click would put it on the authority of somebody having seen it.*

### 4.14 The houses this port writes will sit on the same shelf as the 1994 ones, and nothing said which was which — **DONE, 2.2; the built-in New set followed in 2.3, and the amendment below is where the flag's meaning changed**

Stage 2's job is to add houses. The picker they land in is one flat alphabetical list, which is
faithful — `BuildHouseList` produces exactly that — and the moment this port puts a house of its own
into it, the list is making a claim it cannot support. "Bakery" between "Asylum Pro" and "CD Demo
House" reads as something Ward Hartenstein or Jonathan Chin built in 1994. That is wrong in both
directions and each direction costs something real.

It takes credit that belongs to five named people. Section 1.2 has the table: thirteen of the
twenty-two houses were designed by someone other than John Calhoun, the whole reason the assets
needed a decision of their own, and a list that silently mixes ours in with theirs is the one place
that table cannot be seen from. And it lends the originals our mistakes. A new house with a room
the glider cannot leave is a bug in a 2026 port; the same room, believed to be 1994's, is evidence
that the game was always like that. The player has no way to tell, and the port's credibility is
built entirely on being the kind of thing that tells you.

**A set is declared by the *source* a house was walked from.** Not by the file, and not by a list of
names. `internal/shell/sets.go` carries the argument in full; the short form is that the other two
were each tried on paper and each fails in a way worth recording.

The *file* cannot say it. `houseType` has no field for who made it, the 866-byte header is full —
every byte named, the codec round-trips all 22 shipped files exactly — and inventing a field means
writing a house the 1994 program cannot open, which is the one thing this port will not do to the
format it transcribes.

A *list of the twenty-two names* compiled into the port is the tempting one, and it is wrong twice.
It puts the truth about the shipped set in a second place, so the list and `assets/extracted/houses`
can drift, and on the day they do the list is what the game believes. Worse, it answers confidently
in precisely the case where the answer matters: a house opened in an editor, changed, and saved back
as "Slumberland" is not the 1994 Slumberland, and the name is exactly the part that did not change.
**A provenance claim that cannot fail is not a provenance claim.**

So the caller that opens a root declares what the root is, and every house found under it inherits
that. What this claims is deliberately less than it looks, and saying so is the point:
**a set says where a house was found, not what is in it.** `-houses some/dir` is therefore `Other`
and not `Original` — the game cannot know what a player put in a directory — and the picker shows
the root's label beside the count, so the claim is always checkable against the place it came from.
This is the same habit as the About box printing the upstream commit it was pinned against rather
than a nicer sentence about provenance it does not have.

One caller is allowed to declare `Original` for a directory, and the exception is instructive rather
than a loophole. `internal/fidelity`'s `ScreensOpts.houseSource` records the shell's reference images
from `assets/extracted/houses`, and that directory is the extracted twenty-two *by construction*: the
`screens.hashes` checked in beside it is a hash of what that produced, so a directory with anything
else in it changes the hashes and fails. The declaration is true because the test that reads it is
what makes it true. The difference between the two callers is written where each one is.

**What accumulates and what replaces.** 5.3 deferred union semantics for the asset roots to Stage 2,
on the grounds that a union should be designed rather than fall out of a resolution order nobody
wrote down. It is designed here, and it lands in one of the two places and not the other. An art
root *replaces* the built-in one, because two copies of PICT 1000 have to resolve to one picture.
Houses do not resolve; they accumulate, and a player with four of their own and the twenty-two wants
twenty-six. So `Library.Discover` is variadic over sources and `internal/assetfs` is untouched —
and a source that cannot be read is an error *and the rest are still walked*, so a mistyped
`-levels` names itself on the screen the player is looking at and still leaves them a list to play
from.

**Amended when the levels root became a built-in one, and the amendment is the interesting half.**
The paragraph above was written when `-levels DIR` was the only way to get a New set at all, so
"accumulate" had nothing to accumulate *with*: the flag added a second source to the houses root and
that was the whole of it. `assets/levels.zip` changed the question. There is now a built-in New set,
and the flag has two possible readings — add a third source, or replace the built-in one the way
`-art` replaces the built-in art.

It replaces, and the deciding case is the workflow this repository documents. `make levels` builds
`levels/*.house.txt` into `assets/levels/` so an author can play what they just wrote; the way they
play it is `bin/glidergo -levels assets/levels`. Under additive semantics that lists every house
*twice* — once from the archive, once from the directory it was packed from — and the two rows share
one save file and one score file, because a side-car is keyed on the house's name and
`h.TimeStamp` (`internal/saved/store.go:401-407`) and both rows agree on both. The failure additive
semantics creates is therefore not an edge case a player has to go looking for; it is what happens
the first time anybody follows the instructions.

So `internal/assetfs` is no longer untouched: `assetfs.Whole` is `Root` for a tree that is already
the root, and `-levels` goes through it. What accumulates is still the *set* — `Discover` is still
variadic, the Original root and the New root are still two sources — and what a directory replaces
is one root, which is now the rule all five flags follow without exception.

Accumulating is not the new part. `BuildHouseList` already walked two sources into one list: up to
eight specs handed in by `AddExtraHouse` — a house dropped on the application, from anywhere on any
volume — copied in first, then `DoDirSearch` for the rest (`SelectHouse.c:650-664`, `668-675`). What
the original does not do is say which place a row came from. The set is that missing sentence and
nothing else.

**The UI, and the three decisions in it that are not obvious.**

The filter opens on `AllSets`, not on the selected house's set. Defaulting to the selection's set
would hide every new house on a fresh install, because a fresh install selects Slumberland: the
player would have to discover a chooser in order to discover that there was anything to choose. A
list that hides houses by default is a list whose player never finds out what is missing.

`pick` stays an index into the whole library rather than into the filtered view, so changing the
filter changes what is *drawn* and never what is chosen. The one case that needs a rule is opening
the picker on a house the filter does not hold, and `clampPick` widens the filter to `AllSets`
instead of moving the cursor — the cursor is on the house the player asked for and the filter is the
part they did not.

**Tab cycles the sets**, and that is a key taken rather than a key found. In the original's dialog
Tab is an undocumented alias for the right arrow (`SelectHouse.c:277-278`, the two cases share a
body); in this port's picker it had become a second Escape, advertised nowhere. Nothing documented
is lost, and the footer now offers `Tab set` only when there is more than one set to cycle. The
settings screen keeps Tab as its way out, because it has nothing to switch between.

**The layout mistake is written down because the reasoning was wrong in a way that will recur.** The
obvious home for a set chooser is the gap between the picker's title and the first house — 32 rows,
apparently empty. It is not empty. The selected house's inverse-video bar rises `barRise` above its
baseline, so the first row's bar occupies rows 80 to 104, and the title's shadow reaches row 73. Six
free rows, where a scale-1 word needs ten. The error was measuring the gap against the first row's
*glyph ink* at row 86 and not against its *selection bar*, and it was not caught by looking at a
screenshot — `TestOneSetDrawsNothingNewOnTheTitleLine` counted 2,602 cream pixels where a one-set
library must draw none. The strip went onto the title's own baseline, which is the row that actually
has room, and the title lost the set name it had briefly gained, because the strip's highlighted
entry already says which set is showing. The wrong measurement is in the comment on `pickSetsV` so
that the next person to want a row there knows why there isn't one.

**With one set on the shelf the picker draws exactly what it drew before** — no strip, no title
suffix, no `Tab set` in the footer. That is not politeness; it is what keeps `internal/fidelity`'s
reference image a picture of the 1994 dialog, and it is asserted rather than assumed:
`TestOneSetDrawsNothingNewOnTheTitleLine` renders a one-set library twice, once as `Original` and
once as `New`, and requires the two whole screens to be identical. `make fidelity` passes with
`screens.hashes` and the golden `houses.png` unchanged, which is the same claim made from the other
end.

**Where the built-in New set lives — DONE, and it was the half that decided whether any of this
reached a player.** Until it was built, the New set needed `-levels assets/levels` on the command
line, so nobody who downloaded a release archive had a twenty-third house at all: the work in 2.3
existed only for people with a clone and the instructions in front of them.

The two places a reader would guess are both barred, and they stayed barred.
`assets/extracted/houses/` cannot take a new house: `make assets-check` is a full recursive `diff
-r` against the extractor's output and `assetpack.Compare` asserts `extracted.zip` holds exactly
the files in `assets/extracted`, so a 2026 house would be reported as an extra file forever —
correctly, because that tree's whole value is being byte-for-byte reproducible from 1994 data, which
is 1.2's provenance boundary. And `//go:embed` cannot reach outside `assets/`. So it is the second
archive this entry recommended: `assets/levels.zip`, packed from `assets/levels/` by the
`tools/packassets` that already existed, embedded beside `extracted.zip` and opened by
`assets.Levels()`. No new machinery, and the provenance boundary is now visible in the file names.
The alternative — generalising `assetpack.Files`/`Create`/`Compare` to pack two trees under
prefixes — would have bought one file at the cost of making two different contracts read as one:
`extracted.zip` carries a claim about 1994 that `make assets-check` proves, and `levels.zip`
carries no such claim.

Three consequences that are not obvious from the outside, all of them load-bearing:

- **The archive is committed and the directory it is packed from is not**, which is the reverse of
  `assets/extracted{.zip,/}`, where both are. The reason is a cycle: `make levels` runs
  `bin/glidertool`, `cmd/glidertool` imports `assets`, and `assets` embeds `levels.zip` — so a
  checkout without the archive cannot build the tool that packs it. The `embedded` guard's remedy
  for a missing `levels.zip` is therefore `git checkout -- assets/levels.zip` and nothing else, and
  the rule the two archives share is "commit what a clone cannot regenerate, plus whatever the
  build needs in hand before it can regenerate anything".
- **The member sits at the archive's root** — `Open House.house`, not `levels/Open House.house` —
  because what is embedded *is* the levels root, so a prefix inside it would have to be stripped by
  every caller. `assetfs.Whole` exists for the same reason: `Root(tree, dir, ".")` would have
  labelled the source `built-in:.` and drawn paths like `built-in:./Open House.house`.
- **`go test ./assets` is the only thing in `make check` that can see any of this.** Nothing under
  `internal/` can reach the embedded levels root — `internal/shell` takes its sources from its caller and `internal/fidelity` builds its own
  — so `go test ./assets` is the entire coverage, and the two likely ways to ship it wrong are
  both silent all the way to the player: a wrongly-prefixed archive (no house appears, and nothing
  errors) and a stale one (the house that ships is not the house in the text). Hence two tests that
  do not skip: one requires the archive's root to hold exactly the authored house names and no
  directories, the other rebuilds every `levels/*.house.txt` and compares the bytes.

**`docs/PLAN.md` §3's map — DONE.** It said `levels/` was "new houses in text form" while this
entry's flag said `-levels` was a directory of *built* houses; both were true and the naming is now
stated once rather than inferred twice, in §3's map itself: `levels/*.house.txt` authored,
`assets/levels/` built by `house build` and regenerated, `assets/levels.zip` that directory packed
and committed, `-levels DIR` to replace the built-in root at run time.

**Still open.**

- **The chosen set is not remembered between runs.** The filter resets to `AllSets` every launch,
  which is the right default but not obviously the right *only* behaviour once there are two large
  sets. It is one field in `internal/prefs` if anybody asks, and it is deliberately not added on
  speculation: every row on the settings screen is a row a player has to read past, and nobody has
  yet had two sets large enough to be annoyed by this.

### 4.15 A new house cannot have art of its own, because there is nowhere its pictures are allowed to sit — **note; found writing the first one, 2.3; the mechanism is DONE, 2.5 — see the amendment; no house of ours carries art yet, and that half is an art task**

`Open House` (2.3) is drawn entirely with built-in backgrounds 2000-2017 and built-in object art, and
that was a constraint discovered rather than chosen. §10.3 step 3's target background mix is 15 %
indoor, 10 % `kDirt`, 3 % ground-outdoor, 5 % elevated, 22 % air/space, **27 % user-structure and
17 % user-open** — 44 % of the corpus's rooms are painted with art the house carries itself. Ours is
72 % interior, 7 % `kDirt`, 21 % air, and the missing 44 % is not a design preference.

**It is not that gliderGo cannot read house art.** It reads it very happily, and not as a resource
fork: `assets/extracted/houseart/<House Name>/pict/<id>.png` is the whole interface, one PNG per
`PICT` id, and `OpenHouseResFork` (`cmd/glidergo/play.go:427-432`) mounts that directory for as long
as the house is open. Anybody could paint a 512×322 PNG, call it `3000.png`, and a room with
`background 3000` would draw it. The obstruction is two mechanical facts about where those bytes are
allowed to sit, and they were both put there for good reasons that happen to collide here.

**`assets/extracted/` is diff-locked against the extractor.** `make assets-check` is a full
recursive `diff -r` against `tools/extract_all.py`'s output and `assetpack.Compare` asserts
`extracted.zip` holds exactly the files in that tree, so a `houseart/Open House/` directory breaks
both contracts — correctly, because being byte-for-byte reproducible from the 1994 CD is that tree's
entire value, and 1.2's provenance boundary is the reason it has one. This is 4.14's already-filed
"where do the new houses live" problem wearing a second hat, and it wants the same answer:
`assets/levels.zip`, packed by the `tools/packassets` that exists, with the art beside the house.

**That archive now exists** (`assets/levels.zip`, embedded in every executable) **and it holds only
houses.** Art beside the house is still unbuilt, and the shape it wants is clearer for the archive
existing: one directory per house inside the levels archive, mounted by `OpenHouseResFork` the way
`houseart/<House Name>/` is, so that `background 3000` in a house of ours finds a PNG the binary
carries. What that needs is a second lookup in `OpenHouseResFork` and nothing in `assets/extracted/`
touched at all, which is the point.

**`-houseart DIR` is exclusive, and that is the part worth fixing first.** It *substitutes*:
`assetfs.Root` returns `os.DirFS(dir)` and ignores the embedded tree
(`internal/assetfs/assetfs.go:38-46`). So the only way to hand a new house its art today is
`-houseart` pointed at a directory that also contains all twenty-two originals' art, and anyone who
points it at just their own house silently takes the art away from every shipped house that needs it
— which is half of them. A house's art root is *per house* by construction (the lookup is
`IsDir(houseArtFS, name)`), so there is no reason for this one flag to be exclusive; it should search
a list of roots, and the per-house directory name already makes collisions impossible.

This paragraph used to draw the contrast with `-levels`, which it said was *added* rather than
substituted. That is no longer true and the correction is 4.14's amendment: `-levels DIR` replaces
the built-in levels root, because the documented author workflow — `make levels` then
`-levels assets/levels` — listed every house twice under additive semantics, with both rows sharing
one save file. `-houseart` is now the only asymmetry left, and it is a per-house lookup rather than a
root-level one, which is why the answer for it is a search list and not a replacement.

Until then the restriction is the honest one and it is written into the house's own header comment:
no `kUserBackground` (≥ 3000), no `kCustomPict`, no `kTV`, no custom `snd `. That also means §10.3
step 4 — "light every user-art room with a `kInvisLight`", the most common authoring mistake — has
nothing to apply to yet, and §10.5's `kSoundTrigger`-with-no-`snd ` and `kTV`-with-no-`.mov` traps
are unreachable rather than avoided. Worth knowing when the first house does carry art: three of the
recipe's rules go live at once.

**Done, 2.5: the mechanism, both halves of it.** A house of this port's own puts its pictures in
`levels/houseart/<House Name>/pict/<id>.png`; `make levels` copies that tree into `assets/levels/`
beside the built houses, `make levels-zip` packs it into the archive every executable embeds, and a
room with `background 3000` draws it with no flag and no files beside the binary. The other half is
that the lookup now **searches** rather than substitutes. `internal/assetfs.ArtRoots` is an ordered
list — `-houseart` first, then the levels tree's `houseart/`, then the extracted 1994 forks — and
`ArtRoots.Fork(house)` returns the first root that holds a directory of that name. Every caller that
mounts a fork goes through it: `cmd/glidergo/play.go`, `cmd/glidertool`'s `render`, `house lint` and
`house stats`, and `internal/replay`.

Three decisions in that are worth the words, because each was the alternative that looked simpler.

**A named root that misses still says so, and a built-in one does not.** That is `ArtRoot.Named` and
`Fork.Passed`. Making `-houseart` merely additive would have been one line and would have taken away
the thing the flag exists for: somebody testing an extraction points it at their own tree, and a
search that quietly fell through to the copy inside the binary turns a half-extracted tree into one
that looks complete. So a *flag's* root that is passed over is reported — `no resource fork at
/tmp/art/Titanic; using built-in:houseart/Titanic` — while a built-in root that has no fork for a
house says nothing, because nine of the twenty-two carry no pictures at all and that is not news.

**The levels art root is not `Named`, even when `-levels` is a directory somebody typed.** What they
named is a root of *houses*; a house in it with no art beside it is the ordinary case rather than a
mistake worth a line. `Named` is for the flag whose subject is art.

**`houseart` at the top of a root is skipped by the house walker** (`internal/shell/library.go`).
Nothing in an art tree has a house's extension today, so this changes no listing; what it stops is a
future stray extensionless file in somebody's `pict/` turning up in the picker's `Skipped` list as a
house that would not parse, which is a confusing way to be told about a stray file.

The end-to-end proof is `TestAHouseCanCarryArtOfItsOwnInTheLevelsTree`, which gives `Open House`'s
room 0 a `background 3000` and replays it three times: with the picture in the levels tree, with it
nowhere, and with it named by `-houseart`. The first and third must draw identical pixels and the
second must differ — a comparison rather than an assertion about a colour, because a house picture is
colour-matched into the 1994 palette on the way in and a literal RGB would be a test of the matcher.
It was checked against its own absence: with the levels root deleted from the search list, all three
assertions fail.

**Not done: no house of ours carries any art, and that is why this stays open.** The 44 % of rooms
§10.3 wants painted is the original complaint, and it is an art task — 44 % of 43 rooms is nineteen
512×322 paintings, which is not something this host can produce and not something engineering can
substitute for. What has changed is that the obstruction is gone: the header comments of both houses
now say what is possible rather than what was forbidden, and §10.3 step 4's `kInvisLight` rule and
§10.5's two traps are live rather than unreachable. The custom `snd ` half of that header restriction
is *not* closed by this and is filed separately as 4.29 — house sounds arrive through the sound
root's shared `houses/manifest.tsv` (`internal/audio.Bank.LoadHouse`) rather than through `houseart/`,
so they need a different mechanism entirely.

### 4.16 The profile a new house is held to was measured by hand, and two of its fifteen rows cannot be measured at all — **note; the Go half DONE, 2.3; both "impossible" rows turned out to be computable, the second in 2.4 — DONE, see the two amendments**

Stage 2's bullet asks for houses "designed against the quantitative profile of the originals in
`docs/analysis/original-houses.md`". §10.2 is that profile: fifteen rows, five tiers. Hitting it for
`Open House` meant `grep -c` and a pocket calculator over the authored text, twice, because the first
pass got the empty-room count wrong. There is no tool. `glidertool house lint` answers "is this file
legal", which is a different question from "is this file the kind of house it claims to be", and
nothing else in the repository computes a single cell of that table.

**Thirteen of the fifteen rows are now checked in Go, and that is where the argument for a tool
starts rather than ends.** `TestOpenHouseMatchesTheTutorialProfile`
(`internal/replay/openhouse_test.go`) walks the parsed house and asserts each band, so the numbers in
the house's header comment are an invariant instead of a claim — somebody adds four objects to make a
room look better and the test says which row left its tier. But it is hard-coded to one house and one
column, which is exactly the wrong shape: the next house is a different tier, and the thing an author
wants before they have a passing test is *the numbers*, printed, so they can see they are at 2.9
objects a room with a ceiling of 3.1. That is `glidertool house stats` — the same walk, a table on
stdout, `-tier tutorial` to add the bands and an exit status. Both existing callers would then be
checking the same code instead of the same table.

**Two rows cannot be computed from `internal/house` at all, and both for reasons already on file.**

- **Dark rooms** (§10.2 target 0 % for a tutorial, 4.6 % corpus-wide) needs `GetNumberOfLights`,
  which is a method on `*render.Scene` (`internal/render/locale.go:1350`). `internal/render` imports
  `internal/house`, so the dependency cannot be inverted, and re-deriving the rule in `house` means
  two copies of a switch over eighteen backgrounds plus the `kDirt` all-tiles-zero special case plus
  the eight `State != 0` lamp types — a copy that would drift, and drift silently, since a house that
  a linter calls lit and the renderer draws black is worse than no check. 4.1 already made this call
  once, for objects outside their room ("it wants either a callback like `PictSize` or to live in the
  renderer, and neither is worth doing before a new house needs it"). A new house now needs it, and
  the callback is the answer both times: `LintOptions` grows a `Lights func(*Room) int`, `glidertool`
  fills it in from `render`, and the check is skipped with `checks-skipped` when nobody did. That
  keeps the one implementation in the renderer, which is where the pixels are.
- **BFS eccentricity** (11-15 for a tutorial) needs the room graph, and 4.1 explains why there isn't
  one: `Room.Openings` is dead in all 4,070 corpus rooms and the real adjacency comes out of
  `DetermineRoomOpenings` at run time, in `internal/game`. Same shape, same remedy, one more
  callback. It is also the row most worth having, because it is the only one that measures the
  *shape* of a house rather than its contents — `Open House` is 43 rooms on a 7×10 grid and nobody
  here can say whether its longest shortest-path is 9 or 19.

The scripted playthrough covers what those two rows were standing in for — a house that cannot be
crossed does not finish — so this is a gap in *reporting*, not in safety. But an author who has to
choose between "grep, and hope" and "write a 200-line Go test per house" will do neither.

**Amendment, 2.4: the dark-rooms row was computable the whole time, and the sentence above is a good
example of how to get this wrong.** Writing the second house needed the row, so it got another look.
Every word of the bullet is true — `GetNumberOfLights` *is* a method on `*render.Scene`,
`internal/render` *does* import `internal/house`, the dependency *cannot* be inverted, and
re-deriving the rule *would* be two copies of a switch over eighteen backgrounds. What none of it
establishes is the conclusion, because the conclusion is about where the *check* lives and the
argument is about where `internal/house` can import from. The check does not live in
`internal/house`. It lives in `internal/replay`, which imports `internal/render` already and cannot
run a game without it. Three lines:

```go
scene := render.NewScene(render.DefaultView(), render.NewAssets(nil), h)
...
if scene.GetNumberOfLights(int16(i)) == 0 { p.dark++ }
```

The empty asset tree is fine: `GetNumberOfLights` reads the room's background and its light objects'
switch states and touches no picture, so there is nothing for an asset tree to supply. Both houses
now assert the row — `internal/replay/profile_test.go` holds the shared counting — and it earned its
place immediately, on `Boarding House`'s Bell Turret: the room was `kRoof`, which is one of the eight
backgrounds that light themselves, and became `kPaneledRoom`, which is not, so changing a background
put out a light and no other row moved. Checked by eye with `glidertool render -all` it would have
looked like a room at night.

The correction does not touch the case for `glidertool house stats`, which is about an author wanting
the numbers *before* they have a passing test, and it does not touch BFS eccentricity: that one needs
`DetermineRoomOpenings` and a graph walk, is genuinely not three lines, and is what remains of this
item. The `LintOptions.Lights` callback proposed above is also still the right shape for the *linter*,
which does live in `internal/house` — the amendment is that a test does not have to wait for it.

The general lesson is cheaper than the specific one. "X cannot import Y" is a fact about two
packages; "this cannot be checked" is a claim about the whole repository, and the step between them is
"and there is no third package that imports both". That step was never taken, and it was false.

**Second amendment, 2.4: the other row was computable too, and the answer it gives is that `Open
House` misses its tier.** The remaining half of this item is done. `internal/profile` is the package
the two callers were always going to need — sixteen rows of arithmetic over `internal/house`, the
dark-room row through a `*render.Scene`, and the eccentricity row through a static room graph in
`internal/profile/graph.go` that drives `DetermineRoomOpenings` over every room and breadth-first
searches the result. `glidertool house stats` prints all eighteen, with `-tier` for the bands and
`-fail` for an exit status; `internal/replay`'s two house tests now ask the same package instead of
holding a hundred lines of tallies between them.

The first amendment's lesson applies again and the answer this time is better than a test. A graph
walk needs `internal/game`, and `internal/replay` imports it — so the three-lines-in-a-test move would
have worked a second time. It was the wrong move: the numbers are wanted *printed*, and a package
sitting above `house`, `render` and `game` that none of the three knows about costs nothing the test
was not already paying. What "cannot be checked" meant, both times, was "I have not looked for the
place where it can be".

**What the row says, now that it can be asked.** This entry's own words were that eccentricity is "the
row most worth having, because it is the only one that measures the *shape* of a house rather than its
contents — `Open House` is 43 rooms on a 7×10 grid and nobody here can say whether its longest
shortest-path is 9 or 19." It is **10**, and §10.2's tutorial band is 11-15, so the house misses a row
it has been claiming to hit. The miss is worth more than a pass would have been:

- The band is two houses. `Empty House` is 35 rooms at eccentricity 11, `Demo House` is 45 at 15, and
  `Sampler` — the third template-tier original — is 2 rooms and outside the tier's 35-45 room band
  entirely. Nobody chose 11-15.
- `Demo House` reaches 33 of its 45 rooms. `Empty House` reaches all 35. **`Open House` is 43 rooms
  and reaches all 43, inside 10 hops** — more rooms than `Empty House` inside a shorter diameter,
  which is a bushier graph and not a shallower one.
- Its star is at hop 9, in the Belfry, and the four deepest rooms are the roof edges just past it: the
  North and South Louvres, the South Slope and the East Balcony. `Demo House`'s one star sits at depth
  10 of 15 — a third of that house lies beyond its own goal. For a tutorial, having almost nothing
  past the star is the better of the two shapes.

So the house is not being changed, and `TestOpenHouseMatchesTheTutorialProfile` names the row as an
allowed miss with that argument attached. `checkTier` checks the exemption both ways: an unnamed miss
fails, and a named row that has stopped missing fails too, because an exemption whose reason has
expired is a paragraph that has started lying. This is 4.18's position applied to our own house rather
than to the corpus — §10.2 is a measurement of 22 houses and not a rule they obey, and the honest
response to a miss is to look at it and then say what you found.

**Four things the tool found on its first day, which is the case for having built it.**

- **§10.2's `prizes/room` band excludes the house it was measured from.** `Empty House` has one prize
  in 35 rooms: 0.0286, which §8.3 prints as 0.029 and §10.2 rounds into a band floor of 0.03. So
  `house stats -tier tutorial "Empty House.house"` reports a miss on a row whose band that house
  defines. `Band.Contains` is inclusive at both ends precisely so a corpus extreme passes its own
  band; rounding in the document defeats it. Not corrected — the table is a transcription and
  `TestTheTierTableMatchesTheDocument` holds it to the markdown — but it is the second known case, with
  tutorial's `~20 %` empty-rooms midpoint, where 10.2's cell is not quite a band. *(Pointing the same
  command at all 22 houses turned those two known cases into a property of the table, which is 4.25:
  the rounding goes inward at seven cells, and large's objects/room band of 6.0-7.5 contains neither
  of the two houses it was measured from.)*
- **This entry has miscounted 10.2 since it was written.** The table has eighteen rows, not fifteen;
  git says it has not changed since the document was written, so the number was never right. That is
  not pedantry, it is the mechanism: `TestOpenHouseMatchesTheTutorialProfile` asserted a list of
  sixteen rows somebody typed, and `prize:enemy` was simply absent from it — present in the
  `Boarding House` test, hand-written, and missing here, with no test able to notice. `profile.Check`
  iterates the table, so the eighteen are eighteen because the table says so.
- **`-no-assets` was warning everybody.** The closing line said the reachable and eccentricity rows
  read a custom-background room as sealed, whenever the flag was given. True for the seven houses with
  rooms that need a `'bnds'` resource and false for the other fifteen and for every house the port
  ships, whose rooms carry their own `bounds` field. `Profile.ForkBounded` counts them, the note names
  the number or says the measurement is exact, and the corpus total is pinned at 155 against §10.1
  step 10. A caveat that fires on every house is a caveat nobody reads.
- **The one corpus claim the command printed was false.** Beside the caveat it printed a second note,
  that 10.2's bands are a spread and "no shipped house is inside all eighteen of its own tier's" — a
  sentence truncated mid-clause, and wrong: `Leviathan` is inside all eighteen of epic's. The figure is
  21 of 22, it is now measured rather than assumed, and the three mechanisms behind it are 4.25. The
  general lesson is the specific one this entry keeps arriving at from different directions: a claim
  about the corpus that lives in a `fmt.Printf` has nothing holding it, and this project's habit of
  pinning such claims to a test is what the tool now makes possible for claims about *tiers*.

### 4.17 A house with no custom art is told its custom art will fall back — **found and DONE, 2.5; the note's own list of what counts was too long by two**

Rendering `Open House` prints `no extracted resource fork at built-in:houseart/Open House; custom
art will fall back` before every image. It is true and it is useless: the house uses no art above
2017, so there is nothing to fall back, and the one house in the library that provably needs no
warning is the only one that gets it on a complete asset tree. The same line is in two places
(`cmd/glidertool/render.go:85`, `cmd/glidergo/play.go:432`) and both fire on "the directory is
absent" when the condition they mean is "the directory is absent *and* this house asks for something
that would have been in it" — which is one pass over the rooms for a `background >= kUserBackground`
and over the objects for a `kCustomPict`, `kTV` or `kSoundTrigger`, all of which `internal/house`
can already see (`lint.go:783` does the `kCustomPict` half). Left as a note because the message is
suppressed by `-quiet` and misleads nobody who reads the next line, but it is the first thing a new
house's author sees and it tells them they have done something wrong.

**Done, 2.5, with 4.15 because it is the same two call sites.** The predicate is
`house.House.WantsOwnArt` and the words are `assetfs.Fork.Complaint`, one copy of each, so the
condition and the sentence can no longer drift apart in the way that needed this note to name two
files. `Open House` and `Boarding House` now render and play silently on a complete asset tree, which
they should always have done.

**The note's own list of objects was wrong, and finding out is most of what the work was.** It said
`kCustomPict`, `kTV` or `kSoundTrigger`. Only the first belongs. A `kTV` wants a QuickTime movie,
which this port has no support for at all and draws built-in art instead
(`internal/game/dynamics_appliances.go`), so a house full of televisions is not a house with missing
pictures. A `kSoundTrigger` wants a `snd `, which arrives through the sound root's per-house manifest
(`internal/audio.Bank.LoadHouse`) and not through `houseart/` — a different root, a different file
format and a different gap, now filed as 4.29. Counting either would have reproduced the bug this
item is about in a new place: a warning about pictures, on a house whose pictures are all present.

The other correction runs the other way. `kCustomPict` counts **whatever id it names**, including one
the application also has, and the reason is in the 1994 data: Metropolis carries its own `PICT` 1999
and Fun House its own 2014 and 2015, all three shadowing application art. So "below 3000" is no
evidence that the fork is unwanted, and the only exact test — "is this id in the application's chain
and not in the house's" — needs the very fork whose absence is being reported.
`TestWantsOwnArtAgreesWithTheShippedForks` holds the result against the corpus: for all 22 shipped
houses, `WantsOwnArt` and "has a directory under `assets/extracted/houseart`" agree exactly.

### 4.18 §10.3's construction procedure and §10.2's tier table contradict each other, and a tutorial house cannot satisfy both — **note; found by following them, 2.3; narrowed to the tutorial column by the second house, 2.4**

Three of the recipe's steps give per-room object counts that are corpus *means*, and the corpus mean
is 7.7 objects a room. The tutorial column of the table two subsections above allows **2.2-3.1**.
So:

- step 5, "interiors get 11-13 objects" — `Open House` is 72 % interior; at 11 apiece that is 340
  objects in a house whose ceiling is 138, and 7.9 a room.
- step 10, "decorate each star room to ~15 objects (corpus mean 14.75)" — 15 objects is more than a
  tenth of the whole tutorial budget in one room, and §10.2's own "rooms at the 24 ceiling: 0" row is
  the only constraint that survives.
- step 12, "set `firstRoom`'s contents richer than average: 13 objects" — the same figure, and the
  only one of the three whose *intent* survives translation, because "richer than average" is a
  ratio and 13 is not. Ours holds 4 against an average of 3.09, which is the rule; 13 would be a
  third of the tier.

None of this is wrong in §10.3, which was written from the corpus as a whole and says so. It is
unusable as written for the smallest tier, which is the tier a first new house should be, and the
person most likely to follow it step by step is the person least likely to notice that step 5 and the
table disagree. The fix is one sentence per step making the figure relative — "interiors carry 3-4×
what the air rooms do", "the star room is the densest room in the house", "`firstRoom` is above the
house's own mean" — and a line under the table saying the numeric steps below scale with the tier.
`TestOpenHouseMatchesTheTutorialProfile` encodes the ratio reading for the two that are checkable
(start room above average, no enemies in it), so the disagreement is at least pinned on one side.

**Amendment, 2.4: the contradiction is a property of the tutorial column, not of §10.3, and writing a
house at the tier above settles which.** `Boarding House` is 10.2's **small** tier, whose objects/room
band is 7-19 against the tutorial's 2.2-3.1. At that tier step 5 is not merely satisfiable, it is
close to forced: the house's 35 rooms that have a ceiling average **10.00** objects apiece, 24 of them
holding 11 to 13 — step 5's figure, taken literally, with no adjustment — and the whole-house average
lands at 7.61, comfortably inside 7-19, because the 16 open-air rooms average 2.38. Step 10's 15-object star room and step 12's 13-object `firstRoom` are
likewise ordinary numbers at this tier rather than a third of the budget.

So the three numeric steps are not written against the corpus "as a whole" in a way that fails
everywhere. They are written against the corpus mean, 7.7 objects a room, and the small tier *is* the
mean. The tutorial column is the outlier — it is the only tier whose object budget is a third of the
corpus's — and a recipe stated in absolute numbers will contradict exactly that one column and no
other. That is a smaller and more useful defect than "§10.3 and §10.2 contradict each other": the fix
is still the sentence-per-step rewording proposed above, but it now needs only one line under the
table, and it can say which tier it is protecting.

Worth recording as method rather than as content: this was not settled by re-reading either section.
It was settled by writing a second house at a different tier and measuring it with the same code
(`internal/replay/profile_test.go`; the measurement is `internal/profile` since 4.16, and 4.25 is what
asking it about all 22 houses at once then found), which is the only way a claim about a *table of
tiers* can be tested at all. One house fitted to one column cannot distinguish "the table is usable" from "the
column was chosen after the fact".

### 4.19 Two lines in an authored house are its players' saved games, and nothing said so — **note; found before it could cost anything, 2.3; the pin DONE**

A house that ships is a house people save games into, and the port keys those side-cars on the
*house* rather than on the file it came from. `saved.Store.Path` is the house's name,
percent-escaped (`internal/saved/store.go:85`, via `datadir.FileName`), and `Check` then gates the
save on two facts: the name in the saved block matching the house being opened, case- and
diacritical-insensitively (`internal/saved/store.go:390-393`), and the house's stamp matching the
one the save carries (`internal/saved/store.go:401-407` — gate 2, the original's
`kYellowSavedTimeWrong`, `SavedGames.c:237`). High scores are keyed on the name alone:
`scores.FileName` takes a house name and nothing else, and nothing in `internal/scores` compares a
board against the house's stamp. It does keep timestamps — one per row, the day that score was set
(`internal/scores/scores.go:151`) — and none of them is ever checked against `h.TimeStamp`.

Which makes two ordinary-looking lines of an authored house a compatibility surface, and neither
looks like one:

- **`timestamp 1725439552`** (`levels/Open House.house.txt:276`) is the save key. Change it, to any
  value, for any reason, and every saved game of that house in the world is refused with "Open House
  has been modified since this game was saved". That refusal is *right* — a save holds object states
  by room and object index, and an edited house may have moved both — which is exactly why the field
  cannot be treated as cosmetic. High scores survive it; saves do not.
- **The house's file name** is the other half, and it is both keys at once. Renaming `Open
  House.house` orphans the saves *and* the score board together and says nothing, because the new
  name simply has no side-car and a house with no side-car is a house nobody has played.

None of this is a deviation and none of it is a bug: the 1994 program gates on the same field, and
its high scores lived inside the house file, which is why they never needed a stamp. What was
missing was anybody writing it down before the first house shipped — while the cost of a tidy-up
commit to a header is zero, rather than after, when it is every player's progress. It is now in
three places: the comment on the field in the house's own text, CONTRIBUTING's Houses section, and
here.

**The guard is built, because the hazard is an accident and not a decision.**
`TestShippedHousesKeepTheStampTheirSavesAreKeyedOn` (`assets/assets_test.go`) holds a table of every
shipped house's published stamp and fails if the authored text disagrees — and fails just as loudly
for a house with no row, so a second house cannot be added without a decision being made about it.
The failure message is the whole point of the test: it says what the change costs and that the fix
is to put the old number back. A `house lint` check was the other candidate and is the wrong shape;
a linter sees one house and cannot know what that house's published stamp was.

### 4.20 A house that ships inside the binary can only ever be named, and one tool could not take a name — **found and DONE, 2.3**

The moment a house lives in `assets/levels.zip` rather than on a disk, "attach the file" stops being
something a bug report can do. The name is all there is. `glidergo -house "Open House"` was fine —
`resolveHouse` walks the same sources the picker does (`cmd/glidergo/main.go`, `inSources`) — but the
tool built for bug reports was not:

```
$ glidertool replay -house "Open House" -frames 60
glidertool: open Open House.house: file does not exist
```

`internal/replay` resolved a name in the houses root alone, so it named a file that does not exist
anywhere, for the one class of house whose whole point is that it does not exist as a file. A
reporter's only way out was to find a clone, run `make levels` and pass a path — which is to say,
there was no way out for a reporter.

**Fixed, and the shape of the fix is the thing worth keeping.** `replay.Script` now carries the
second root the same way it carries the first (`Levels` beside `Tree`, `LevelDir` beside `HouseDir`,
`leveldir` in the text format, `-levels` on `glidertool replay`), and the lookup tries the houses
root and then the levels root — `cmd/glidergo`'s order, and the same tie the picker's sort breaks the
same way, so an original wins a name it shares with a new house. Only `fs.ErrNotExist` falls through
to the second root: a house that is there and will not parse is the answer. When neither root has it
the error names both roots, because "file does not exist" about a name somebody typed omits the one
fact they need.

The general rule, for the next tool: **anything that accepts a house by name must search every
source the picker does.** Two things in the tree accept a house by *path* only and are correct as
they stand — `glidertool render <house>` and `glidertool house <subcommand> <file>` — because in a
clone every house has a path (`assets/extracted/houses/` is committed and `make levels` writes
`assets/levels/`). They would still be worth revisiting if either grew a `-house NAME` form.

### 4.21 The paragraph justifying the linter's severities cites four examples: one has no instance in the corpus, and one is calibrated as an error and fails it — **found and DONE, 2.4; the narrower rule it yielded is what calibrated 4.22**

`internal/house/lint.go:19-24` is the argument for how the severities were chosen, and the argument
is a good one:

> Severity is calibrated against the 22 shipped houses rather than against an ideal, because those
> houses are the specification of what a playable Glider PRO house is. They contain 189 dangling
> links, an out-of-range `who`, staircases that lead nowhere and a basement you cannot climb out of
> — **so none of those classes can be an error** without making the corpus fail its own linter, which
> would make the linter useless on the day it shipped.

Two of the four examples hold up. 189 dangling links is a count that reproduces, and the basement is
Slumberland's and has a test of its own (`acafec7`, "Pin Slumberland's inescapable basement as the
original's design"). The other two do not, in opposite directions.

**The staircases have no instance.** The linter has four stair rules — `stairs-no-room` and
`stairs-unpaired` at warn, `stairs-doubled` twice at note — and across all 22 houses, 4,070 rooms and
**326 stair objects** they fire **zero times**:

```
$ bin/glidertool house lint assets/extracted/houses/*.house | grep -c stairs-
0
```

Every staircase in the corpus is paired and both ends land in a room that exists. The checks are not
wrong; they were derived from `CheckForStaircasePairs` (HouseLegal.c:961-1045) and they fire on houses
written to provoke them. The phrase is most likely a paraphrase of that function *existing* — Calhoun
wrote a repair pass for this, which implies he had seen it somewhere — but as written it reads as a
measurement of these 22 files, and it is not one.

**The `who` is the reverse, and it is the more consequential half: it is calibrated as
`SeverityError`, it fires, and the corpus therefore does fail its own linter.**

```
$ bin/glidertool house lint assets/extracted/houses/*.house
    error link-slot-range   room 72 "Let's Roll" slot 22: kMailboxRt links to slot 35 of room 72,
                            which holds 24 slots ...
total: 22 files, 637 notes, 48 warnings, 1 errors
glidertool: a finding reached error
$ echo $?
1
```

(The note total was 634 when this was filed and is 637 now: 4.22's `starfield-tiles` fires three
times on Leviathan. Updated here rather than left as written, because a transcript nobody can re-run
is the defect this entry is about.)

One file, `CD Demo House.house`, one object — exactly the "holds exactly one of these" the check's own
message claims (`lint.go:692-699`). So the sentence's conclusion is false of the code it introduces,
and by the sentence's own reasoning the linter was useless on the day it shipped.

It was not, and that is the part worth getting right rather than reverting. `link-slot-range` **should**
be an error: `GenerateRetroLinks` indexes `retroLinkList[who]` with no bound check (House.c:577, :600),
so the original reads into the next room's record — this is a memory bug in a shipped file, not a
stylistic liberty, and a linter that shrugged at it would be the useless one. The calibration rule the
code actually follows is narrower and better than the one the comment states: *a class the corpus
exercises deliberately cannot be an error; a class the corpus exercises by overrunning a buffer can.*
Nothing flagged the mismatch because `make houses` runs `glidertool house check`, which round-trips
and sanity-checks and exits 0, and no target in the tree runs `house lint` over the originals.

**The remedy is smaller than it first looked, because the test that settles it already existed.**
`TestLintCorpus` (`internal/house/lint_test.go:798`) walks all 22 houses and pins a table of per-check
counts — including `link-slot-range` at 1, with the comment "The one error in 4,070 rooms". So the
tree has known since that test was written that the corpus produces an error; the contradiction was
eight hundred lines from the sentence asserting it could not. What the table did *not* hold was a row
for any stair rule, which is why the other half went unexamined for as long as it did.

**Done, 2.4:** three rows added — `stairs-no-room` 0, `stairs-unpaired` 0, `stairs-doubled` 0 — so the
header's claim is now a number a change has to argue with. They are safe to pin at zero because
`TestLintStairs` exercises all four rules on houses written to provoke them, so a zero in the corpus
row is a fact about the corpus rather than a dead check.

**Done, 2.4 (the other half):** `internal/house/lint.go:19-39` rewritten. Three changes, and the
middle one is the point:

- The staircase example is **dropped**, with a sentence saying it was there and why it was wrong —
  the phrase belonged to `CheckForStaircasePairs` existing, not to these 22 files. Dropped rather
  than rewritten because at note and warn those rules need no corpus evidence to justify them, so
  there is nothing for a replacement example to do.
- The narrower rule is now **stated**: *a class the corpus exercises deliberately cannot be an
  error; a class it exercises by overrunning a buffer can.* That is what the code has always done,
  and writing it down is what makes the next check's severity a decision rather than a guess — it
  is the rule 4.22's three new checks were calibrated with, one week later.
- The `who` is no longer offered as a class that *cannot* be an error. It is named as the one
  instance of the second kind, with `House.c:577, :600` for why, and the paragraph now says plainly
  that `house lint` exits 1 on the originals **by design**. A reader who runs the command gets the
  documented answer instead of a contradiction.

The lesson generalises past this file: the sentence was wrong because it argued from a list of
examples, and a list of examples goes stale silently. Two of the four were still true. What would
have caught it at the time is what caught it in the end — running the command the prose implies.

Worth keeping as a pattern: **a comment that cites evidence should cite it precisely enough to be
re-run.** Both defects here sit in examples that read exactly like the two sound ones, and both
surfaced only by running the commands the prose implies. What prompted it was writing a house with
eight staircase pairs, which meant reading the stair checks closely enough to wonder what they had
ever caught.

### 4.22 The linter has no check that an object makes sense for the background it stands in, and the two rules a house author gets wrong first are both in that class — **found and DONE, 2.4; the census widened one rule from three objects to twelve and demoted the other to a note**

`glidertool house lint` has 29 checks and all of them are about a room's *own* fields being legal:
`what` in range, links resolving, `tiles[]` inside the background picture, names fitting `Str27`. Not
one relates an object to the background it sits in. Two configurations that are illegal by the
corpus's own unanimous practice therefore lint clean:

```
$ bin/glidertool house lint /tmp/floorless.house
/tmp/floorless.house: 1 rooms, 0 notes, 0 warnings, 0 errors
```

That house is a `kSky` room holding a `kFloorVent`. `DoesRoomHaveFloor` (`internal/game/room.go:335-348`)
gives `kSky`, `kStratosphere` and `kStars` no floor at all, so the vent's art stands on a hole — and
across the corpus's **1,084** `kRoof`, `kSky`, `kStratosphere` and `kStars` rooms there is not one
`kFloorVent`, `kFloorBlower` or `kSewerGrate`. Zero of 1,084 is not a style preference, it is a rule
every author in 1994 followed without being told. The second case is the same shape: `kStars` with
anything but the identity tiling, where **243 of the corpus's 246** `kStars` rooms are the identity
because the background is one 512-pixel picture, and that also lints clean.

Both were found the hard way — the generator for `Boarding House` produced a `kStars` room carrying
`kSky`'s tiles *and* a `kFloorBlower`, one room with both defects, and nothing in the toolchain
objected. What caught it was a pair of assertions added to the generator, which is a throwaway script:
the checks are now in `/tmp` and the knowledge is in a house comment, which is the wrong place for
both. A hand author gets neither.

What makes this worth a check rather than a note in a document is that the failure is invisible in the
direction people test. The room draws — `kFloorVent` has a picture and it composes fine — so
`glidertool render` shows a vent, the house lints clean, the lift works because `LiftIt` reads the
object and not the floor, and the only thing wrong is that a player sees machinery bolted to the sky.
Nothing in `make check` can fail.

The shape of the fix is `lint.go`'s existing per-object switch plus one table: for each background,
the object groups that cannot stand in it. It is cheap because the two rules above are the whole of
what the corpus establishes unanimously — and the corpus is what decides the severity, as it does for
every other check here. Both should be **warn**, not error, on the evidence: zero instances in 4,070
rooms means no shipped house is made to fail, and a warning is what `-min warn` shows an author by
default. Worth doing at the same time as 4.16's `glidertool house stats`, since both are "what a house
author needs before they have a failing test" and both want the same walk.

**Done, 2.4 — three checks, and the census moved two of the three decisions above.** `mount-no-floor`
and `mount-no-ceiling` at warn, `starfield-tiles` at note, with `TestLintMounting` and
`TestLintStarfieldTiles` firing each on houses written to provoke them and three new rows in
`TestLintCorpus` pinning 0, 0 and 3. The corpus note total moves 634 → 637; `README.md`,
`CONTRIBUTING.md` and `docs/PLAN.md` are updated, and so is 4.21's transcript.

What the entry above got right: the floor rule is real and the corpus is unanimous about it. What
measuring changed:

**The object set is twelve types, not three, and the classification is not a guess from the names.**
Each of these occupies exactly **one** vertical coordinate across all 4,070 corpus rooms, and that
coordinate plus the height of its artwork puts it against one surface or the other:

| floor-standing | v | height | foot | ceiling-hung | v | height |
|---|---|---|---|---|---|---|
| `kFloorVent` | 305 | 11 | 316 | `kCeilingVent` | 8 | 11 |
| `kFloorBlower` | 304 | 15 | 319 | `kCeilingBlower` | 5 | 15 |
| `kSewerGrate` | 303 | 17 | 320 | `kCeilingLight` | 4 | 20 |
| `kGrecoVent` | 303 | 18 | 321 | `kFlourescent` | 12 | 12 |
| `kSewerBlower` | 292 | 12 | 304 | `kTrackLight` | 5 | 24 |
| `kHipLamp` | 23 | 276 | 299 | | | |
| `kDecoLamp` | 91 | 212 | 303 | | | |

The two lamps are the interesting entries. `kHipLamp` is 276 pixels tall and `kDecoLamp` 212, so both
are *standing* lamps whose feet land on the floor line — a name-based rule would have filed them with
the ceiling fixtures and been wrong twice. `kTableLamp`, `kLightBulb` and `kInvisLight` are in neither
set precisely because their v *does* vary: a table lamp sits on whatever furniture the author put
under it. The five updraughts and two downdraughts are also exactly how `CreateActiveRects` groups
them (`internal/game/hotspots.go`, from `ObjectRects.c`), so the two halves of the classification
agree from independent directions.

**The five flames are deliberately excluded**, and this is the case that would have made the check
fail the corpus. `kTaper`, `kCandle`, `kStubby`, `kTiki` and `kBBQ` make a thermal column the same way
an updraught does, so a rule derived from the physics would include them — but 18 `kTiki` and 9 `kBBQ`
stand in corpus rooms with no ceiling, and one `kCandle` and one `kStubby` in rooms with no floor. A
torch on a lawn is a torch on a lawn.

**`kRoof` is in the ceiling list and not the floor list**, which is why the two are separate maps
rather than one predicate. A roof has a floor — you walk on it — and no ceiling. And `kSkywalk` and
`kDirt` are in neither despite looking outdoor, which is a fact about `Room.c:1138-1206` and not
something a check derived from how a background looks would get right.

That also retracts the **1,084** above: it is the count of `kRoof` + `kSky` + `kStratosphere` +
`kStars` rooms, and `kRoof` does not belong in a floor tally. The floorless corpus is **903** rooms
and the ceilingless one **1,220**, which are the two numbers the findings quote. The original figure
was assembled by reading the backgrounds that look outdoor rather than by reading
`DoesRoomHaveFloor`, and it is the smaller version of the same mistake the object classification
would have made from the names.

**The second rule was wrong about severity, and measuring inverted it.** The entry above asks for
warn on "`kStars` with anything but the identity tiling". Two measurements say note:

- Every built-in background is exactly 512 pixels wide — eight `TileWide` columns — so `tiles[]`
  selects eight of eight and can never be out of range in a built-in room. `tile-column` cannot fire
  on one, which is *why* nothing caught the generator's mistake.
- The corpus is not unanimous, and the exceptions look deliberate: 62 of 62 `kStratosphere` rooms are
  the identity, but only 243 of 246 `kStars` rooms, and all three exceptions are Leviathan's — rooms
  172, 206 and 221, one a straight reversal of 0..7. By the rule 4.21 just made explicit, a class the
  corpus exercises deliberately cannot be more than a note.

So the finding is worded to fire without accusing, and each background quotes *its own* tally rather
than one standing in for the other. That distinction only exists because the census was run per
background; a single "243 of 246" in the message would have been the imprecise citation 4.21 is about,
in a check added to settle 4.21.

**One documented limit: only built-in backgrounds are checked.** A user-art room's openings come from
its own `bounds` field, or the background's `'bnds'` resource when that field is 0 (`boundsCode`), and
`internal/house` has `PictSize` but nothing analogous for `'bnds'`. Rather than check half the rule on
half the rooms, `mounting` returns early outside 2000–2017 and `TestLintMounting` has a row asserting
it stays silent there. That is a false negative by choice; a false positive would be a bug.

**Still open, and now a separate finding — see 4.23.** The table above is a stronger fact than the
check uses. Every one of those twelve types sits at one v in 1,458 `kFloorVent`, 507 `kSewerGrate` and
so on down — so a vent at v 200 in an ordinary room is floating art, and nothing in the toolchain says
so. That is the more likely authoring mistake and it is *not* what these checks catch.

### 4.23 Twelve object types occupy exactly one vertical coordinate in all 4,070 shipped rooms, and nothing in the toolchain knows it — **note; found by measuring 4.22, 2.4; both blockers answered and DONE, 2.4 — the rule is 24 types wide, one of the two questions was answered wrongly here, and a second rule came out of it**

4.22's census produced a fact it did not need and could not use. Across all 22 houses, every placement
of each of these is at one `v` and no other — not clustered, not mostly, **one value**:

```
kFloorVent      1458 placements, all at v 305        kCeilingLight   160, all at v  4
kSewerGrate      507 placements, all at v 303        kFlourescent    115, all at v 12
kSewerBlower     191 placements, all at v 292        kTrackLight      81, all at v  5
kGrecoVent       116 placements, all at v 303        kCeilingVent     28, all at v  8
kFloorBlower      83 placements, all at v 304        kCeilingBlower   12, all at v  5
kDecoLamp         62 placements, all at v  91
kHipLamp          26 placements, all at v  23
```

That is 2,839 placements with zero exceptions, across 22 houses credited to four authors. It is not a
coincidence and it is not house style: `GetObjectRect` places these at the absolute coordinates they
store (`ObjectRects.c:32-273`), and every built-in background is the same 512×322 picture size — so the
floor line and the ceiling line fall in the same place in every room, and an object resting against
either one has exactly one `v` available to it.

The two lamps are what make that argument checkable rather than assumed. Their `v` is 23 and 91, nowhere
near the five updraughts' 292–305 — but they are 276 and 212 pixels tall, so their *bottoms* are 299 and
303, which lands them inside that same band. Two objects whose top edges say "ceiling" and whose bottom
edges say "floor", agreeing with the vents once their height is taken into account, is the one thing in
this census that could not have come out right by accident.

**Why this is worth a check and 4.22's rules do not cover it.** 4.22 catches a vent in a room with no
floor. It says nothing about a vent at `v 200` in an ordinary room, which is *floating art* — and that
is the far more likely mistake, because it is what a typo or an off-by-one in a generator produces. The
port's own `Boarding House` generator had both bugs available to it; only the background one happened
to fire. `glidertool render` draws the room happily, the lift column is computed from the object's own
data so the vent works, and `make check` cannot fail.

The three objects deliberately **not** in the list are the evidence that the rule is real rather than an
artefact of counting: `kTableLamp` (70 placements, v 29..245), `kLightBulb` (252, v 0..294) and
`kInvisLight` (1,764, v 0..306) all vary freely, and they are exactly the three that do not rest against
a room surface — a table lamp sits on whatever furniture the author put under it. A census that found
*everything* fixed would mean the census was wrong.

**What has to be settled before this becomes a check, and why it is filed rather than written.** The
numbers above are a measurement of 22 files, not a citation of the C, and 4.21 is the entry about
exactly that distinction — a rule argued from examples goes stale silently. Before a `object-v` check
ships, two things need finding:

1. **Where the floor line actually is**, in the C, as a constant rather than as seven bottom edges that
   nearly agree. `internal/render/view_test.go:16` already names `kFloorVentTop 305` and
   `kShadowTop 306` as authored-against values, which is a start and not a derivation.
2. **Whether the original's editor enforced it.** If the room editor snapped these objects to the floor
   — which would explain 2,839 of 2,839 — then the check is restating a tool's behaviour and should say
   so, and its severity follows from that rather than from the count. `HouseLegal.c` does not do it;
   whether `ObjectEdit.c` or the palette does is unread.

Until then the honest status is a note with the numbers in it. **Severity if it ships:** warn, by the
rule 4.21 made explicit — zero corpus exceptions means no shipped house is made to fail, and an object
drawn floating is something a player sees.

*(4.23, **DONE**, 2.4. Both questions above are answered, and the answers are better than the note
expected in one direction and wrong in the other. `object-top` and `object-left` ship in
`internal/house/lint.go`, with `fixedTops`, `fixedLefts`, `TestLintObjectAnchor`,
`TestLintObjectAnchorFloorTransStray` and `TestCorpusFixedAnchors`.*

***1. There is no floor line.** The question assumed one constant that seven bottom edges agree on.
There are **nineteen**, one per object type, and the census values are not "nearly" those constants —
each is exactly its own. `GliderDefines.h:467-481` carries thirteen, `:483-494` four more, and
`ObjectAdd.c:19-23` defines the last two locally in the only file that uses them, which is a fair
summary of how much of 1994's geometry is editor-side rather than engine-side. So the table in
`fixedTops` is a transcription with a per-row citation, not a measurement — which is what 4.21 asked
for. `internal/render/view_test.go:16` naming `kFloorVentTop 305` was right about the value and was
reading it out of the render code's own history; the constant it half-remembered is
`GliderDefines.h:467`.*

***2. Yes, three ways over — and "`HouseLegal.c` does not do it" above is wrong.** The editor enforces
this at creation, during editing, and on save.*

- ***Creation.** `AddNewObject` assigns the constant (`ObjectAdd.c:99-110`, `:128-131`, `:311`, `:341`,
  `:355`, `:374-440`, `:517-564`). One type is missed: `kSewerGrate` has no branch in the `if`/`else`
  chain at `:99-110` at all, so a fresh one keeps whatever the empty slot held. It is invisible in
  practice because the function ends by calling `KeepObjectLegal` (`ObjectAdd.c:774`), which repairs
  exactly that type — 1994 covering for itself two hundred lines apart.*
- ***Editing.** `DragObject` moves these types **horizontally only**: their case arms add `deltaH` to
  `topLeft.h` and simply do not mention `topLeft.v` (`ObjectEdit.c:565-573`, `:632-645`, `:668-674`).
  This is the one the note guessed at, and the contrast is two arms away in the same `switch` and the
  same union member — `kLeftFan`, `kRightFan`, the five flames, `kInvisBlower` and `kLiftArea` get both
  deltas (`:575-586`), and so do `kTableLamp`, `kLightBulb` and `kInvisLight` (`:676-681`). The three
  the note named as "the evidence the rule is real" are, in the C, the three cases that take `deltaV`.
  The census and the source agree line for line.*
- ***Save.** `KeepObjectLegal` **rewrites** a wrong `v` back to its constant for four types
  (`HouseLegal.c:115-137`), and runs from `CheckHouseForProblems` on every `WriteHouse`
  (`HouseIO.c:470`). The note's claim that `HouseLegal.c` does not do it was made from a search for a
  floor line rather than for the type names, which is how a negative finding usually goes wrong.*

*So the check does restate a tool's behaviour, exactly as the note predicted it might, and that is the
argument **for** shipping it rather than against: gliderGo's houses are not written by that tool. They
are typed into `levels/*.house.txt` with the coordinate spelled out — `object 2 kFloorVent at 305 120`
— and `make levels` lints with `-fail warn`, so from now on a typo in that column fails the build
instead of shipping a vent hanging in the air. Verified by doing it: changing one `305` to `300` makes
`make levels` exit 1 with the finding. Both of the port's houses pass unchanged.*

***Severity: warn, as predicted, but now argued from the ladder rather than from the count.** The
ladder in `lint.go` reserves error for a house the loader has to repair or range-check before it can be
played at all. A wrong `v` needs no repair: the room loads, `GetObjectRect` uses the stored coordinates,
the art composes, the lift column comes from the object's own `distance`. And it is not a note, because
the result is in the way rather than merely untidy. Middle rung.*

***The rule is twice as wide as the census suggested.** 24 types, not twelve, because 4.22's census only
covered the objects that mount against a surface. Beyond the twelve: `kUpStairs` and `kDownStairs` at
`kStairsTop` 28, `kCeilingTrans` at 6, `kFloorTrans` at 302, the four doors at 0 and the four windows at
64. That is 4,041 placements rather than 2,839, and 3,998 of them are at their constant exactly.*

***And a second rule fell out of it.** The four doors and four windows have named **horizontal**
constants too — `kDoorInRtLeft` 368, `kWindowExLfLeft` 0 and six siblings
(`GliderDefines.h:484-494`) — and `KeepObjectLegal` snaps a dragged one to the nearer wall on every
save, rewriting `what` to match the side it chose, so dragging a left door across the room turns it into
a right door (`HouseLegal.c:310-369`). All 167 shipped doors and windows are at their constant. That is
`object-left`, shipping in the same commit: a door is one of a room's exits, so it belongs against a
wall and nowhere between. Both repairs — this one and the four vertical ones — neglect to set
`KeepObjectLegal`'s `unchanged` flag, so the original could silently move your door and report the house
as untouched.*

***The 43 exceptions are a 1994 bug, filed as 4.27.** 43 of the 244 shipped `kFloorTrans` objects are at
`v` 300 rather than 302 — 37 in Slumberland, 5 in Leviathan, 1 in Rainbow's End — because the repair
that would have normalised them is unreachable. `object-top` accepts 300 for that type and only that
type, as a value rather than a tolerance; 299 still warns.*

***Five types left out, and one pair, both for stated reasons.** `kCounter`, `kDresser`, `kManhole`,
`kMousehole` and `kFireplace` are just as immovable — their `DragObject` arms are `h`-only too — but the
original names their **bottom** (`kCounterBottom` 304, `kDresserBottom` 293, `kManholeSits` 322,
`kMouseholeBottom` 295, `kFireplaceBottom` 297), and a bottom becomes a top only by subtracting the
height of the artwork, which `internal/house` cannot see: it reads house files, and the sprite rects
live in `internal/render`. Putting the five differences in as literals would turn a table of citations
back into a table of measurements. `kBalloon`, `kCopterLf` and `kCopterRt` are `h`-only as well
(`ObjectEdit.c:701-705`) and are left out for a different reason: nothing repairs them, and
`AddDynamicObject` overwrites the vertical position at run time regardless of what the file says —
`kBalloonStart` for the balloon (`Dynamics3.c:395`), `kCopterStart` for the two helicopters (`:417`). A
stored coordinate with no consequence is nothing to report.*

***One thing this changed that was not the subject.** Every synthetic fixture in `lint_test.go` placed
its objects at `v` 0, so the new check fired 98 times across 37 subtests the moment it was wired in. The
fixtures were fixed rather than the check exempted: `plain`, `transportTo` and `switchTo` now run their
object through `anchored`, which puts it where the 1994 editor would have. That is what the file's own
header says the synthetic half is for — "the smallest house Lint has nothing to say about, and then
break exactly one thing in it" — and 37 of those houses had quietly stopped being that.)*

---

### 4.24 Eight analysis documents state a resource's layout, one of them warns that getting it wrong opens the wrong walls, and the port got it wrong — **found and DONE, 2.4**

`internal/render.Assets.Bnds` read the `'bnds'` resource as an eight-byte QuickDraw `Rect` of
big-endian `int16`s, on the reasonable-looking grounds that the C declares its handle `boundsHand`.
It is not a `Rect`. `boundsType` is four one-byte `Boolean`s — `left`, `top`, `right`, `bottom`
(`GliderPRO/Headers/GliderStructs.h:266-272`) — and every one of the 70 resources the shipped houses
carry is exactly four bytes. So the guard `len(raw) < 8` rejected **all 70**, `GetOriginalBounding`
answered 0 for every lookup, and 0 means closed on all four sides.

**What that does to the shipped game.** 155 rooms across 7 houses take that fallback — a room with a
house's own background and no `bounds` field of its own — and **146 of them lost at least one exit they
should have had**: 141 the left wall, 138 the right, 109 the ceiling, 79 the floor. Castle o' the Air 55
rooms, Rainbow's End 30, Slumberland 24, Land of Illusion 21, Demo House 10, Leviathan 9, The Asylum Pro
6 — which includes the four largest houses in the corpus by room count and the two most likely to be
opened first.

The 79 are the ones a player would have met. A closed bottom *is* a floor, so those rooms were given one
the original does not give them: the glider could not fall out of a room designed to drop it, and
`IsShadowVisible` drew a shadow on a floor plane that is not there (`Room.c:1103-1133`). The other nine
of the 155 are genuinely sealed, by a resource of four zero bytes, and were the only rooms the broken
decode got right by accident.

**The documents were right, in twenty-nine places, in eight of the thirteen that mention the
resource.** `docs/analysis/interactions.md` §20.5 is titled "`'bnds'` resources — the 4-byte fallback"
and prints a decoded table of all nine of Castle o' the Air's, byte values and resulting thresholds
included. `docs/analysis/constants.md:6224` says outright: "A port must read `'bnds'` from the house's
resource fork; hard-coding 0 breaks room openness in those eight houses." `docs/analysis/house-format.md`
§4.2.1 quotes the struct. And `docs/analysis/structs.md:4332` is item **3** of a numbered list of the
traps a porter falls into, whose whole text is this bug:

> **3. `'bnds'` is `{left, top, right, bottom}` — a *different* order from `Rect`.** The one place in
> the format where the QuickDraw convention does *not* apply (§9.3, `GliderStructs.h:266-272`). Four
> bytes, easy to get backwards, and the symptom is rooms with the wrong walls open.

Written in advance, about this function, naming the symptom. Nothing read it back.

**Why nothing caught it, which is the part worth generalising.** Three things had to line up:

1. **`Bnds` had no test.** It is the only function in `internal/render` that decodes raw resource
   bytes at all — everything else in the package goes through PNGs — and it was the one with nothing
   driving it. Its one caller, `GetOriginalBounding`, had no test either.
2. **A golden hash over the corpus passed.** `TestObjectGraphEveryRoom` hashes `LeftThresh`,
   `RightThresh` and the four opening flags for all 4,070 rooms, so it covered every affected room and
   said nothing — because a golden pins *change*, not correctness. It had recorded the wrong answer as
   the expected one from the day it was written. Fixing this moved six of its 22 house digests, which
   is the only signal it was ever going to give.
3. **The wrong answer is a plausible answer.** "Sealed" is what an unfinished house looks like, and
   `docs/analysis/original-houses.md` §3.6 already reports that every corpus house has unreachable
   rooms, so a low reachable count is not by itself suspicious. There was no screen that looked wrong.

**What found it was a second implementation with published numbers.** `tools/probe_houses_inventory.py`
transcribed `DetermineRoomOpenings` from the C independently, and §3.6 publishes its per-house figures
including rooms-reachable and BFS eccentricity. When `internal/profile`'s Go walk (4.16) was checked
against that table, six houses came out short — and it was *which* six that named the cause, because
they are six of the eight houses that ship `'bnds'`. Two independent decodings of the same bytes,
compared by a test, is the pattern; a golden hash of one decoding is not.

**What shipped.** `render.Bounds` is now a struct of four `bool`s and not a `Rect`, so the type itself
refuses the mistake. `TestBndsIsFourBooleansAndNotARect` drives the decoder from bytes, with cases
chosen to fail under the old reading rather than merely to pass under the new one.
`TestEveryShippedBndsResourceLoads` asserts all 70 are four bytes of 0/1 and that all 70 are found.
`TestTheBndsFallbackFiresAndFindsItsResource` pins the 155-room census per house, checks the flags
arrive as the openings `DetermineRoomOpenings` reads, and confirms
`docs/analysis/houses-inventory.md`'s second claim — that no shipped room ever actually hits the
missing-resource default. Nine of the 155 *are* told they are sealed by a resource of four zero bytes,
and none of the nine is a trap: The Asylum Pro's six, plus Demo House's "Windows" and Slumberland's
"Paul's Room" and "ssalG gnikooL", which leave by a staircase, a window and a `kInvisTrans`
respectively. `tools/extract_house_art.py` had the same wrong assumption in its manifest, where a
`">4h"` unpack over-read a four-byte resource into the next one and published garbage coordinates
beside a correct `"bytes": 4`; it now records the four flags by name.

**Filed, not written: a check that a documented layout and its decoder agree.** The analysis documents
quote dozens of on-disk structures with stated byte widths, and nothing compares any of them to the Go
that reads them. The cheap version is not a parser — it is a per-resource-type assertion that the
extracted tree's records are the width the document claims, which is two lines each and would have
caught this the first time the assets were extracted. The expensive version, a doc-to-struct
comparison, wants the documents to carry machine-readable layout blocks and is 4.12's citation-checker
argument one level down. **Severity if it ships:** error — a decoder that disagrees with its own
specification is not a style question, and the corpus is the evidence either way.

### 4.25 §10.2 says "pick a tier, then hit these", and 21 of the 22 houses it was measured from cannot — **note; found by pointing 4.16's new tool at the corpus, 2.4; the port is corrected, the document is what wants changing**

The first thing `glidertool house stats` was asked, once it existed, was the obvious one: run it over
all 22 shipped houses at the tier §8.3 assigns each of them and see how the originals score against
their own table. The answer is that **only Leviathan is inside all eighteen bands of its own tier**,
and it manages that by having set eight of epic's bounds itself. The other 21 are outside at least
one. Six are outside four or more. Sampler misses eleven of the eighteen.

That is not a defect in the houses and not one in the measurement — the walk agrees with §3.6 on
eight columns for all 22 houses, and the empty-room count now agrees with §5.4's corpus figure of 840
to the room. It is a property of the table, and three separate mechanisms produce it.

**An edge rounded inward excludes the house it was taken from.** §10.2's cells are written to two or
three significant figures and the rounding goes toward the middle of the band, so the extreme house
that defined a bound lands just outside it. The clearest case is large's objects/room, **6.0-7.5**:
the minimum at that tier is Land of Illusion at 5.9934 and the maximum is Rainbow's End at 7.5022, so
the band is exactly those two houses rounded inward and **contains neither of them**. The same thing
at six other cells — Empty House's one prize in 35 rooms is 0.0286 against a tutorial floor of 0.03,
SpacePods is 14.53 objects a room against an epic ceiling of 14.5, Art Museum holds 569 objects
against a medium floor of 570, Metropolis is 1.016 enemies a room against a medium ceiling of 1.0,
Castle o' the Air is 1.412 prizes a room against 1.4, Fun House is 0.395 against a small floor of
0.4. Rounding a measured extreme is right; rounding it the wrong way turns a description into a
rule nothing satisfies. **A band quoted to fewer digits than it was measured at has to be widened,
not rounded.**

**The empty-rooms row is not a spread at all.** Eighteen of the 22 houses are outside it, which is no
longer a rounding story. Its five cells run 10-30 % and the houses run **0 % to 53.3 %**: six shipped
houses have no empty room whatsoever (California or Bust!, Davis Station, Nemo's Market, Sampler,
SpacePods, The Asylum Pro) and Demo House — the tutorial, the house a first author is likeliest to
imitate — is 53.3 % empty. The tutorial cell is written `~20 %`, and 20 % is §5.4's *corpus-wide*
figure, 840 empty rooms in 4,070, which is a mean over rooms and says nothing about houses of any
particular size. So one row of an eighteen-row table of observed spreads is a recommendation wearing
a measurement's clothes, and it is the row that fires on nearly every house. The observed per-tier
ranges, for whoever rewrites the cells: tutorial **0-53.3 %**, small **0-27.9 %**, medium
**0-22.8 %**, large **11.5-35.8 %**, epic **0-37.5 %**.

**§10.2's room bands disagree with §8.3's own grouping.** §8.3's five tiers are the only complete
assignment of the 22 houses to §10.2's five columns — by room count alone, California or Bust! at 16
rooms and Fun House at 43 belong to no column at all — and for medium, large and epic the two
sections state exactly the same range (85-140, 175-303, 383-531). For the two smallest they do not.
§8.3 describes its first tier as 2-45 rooms where §10.2's tutorial column says 35-45, and its second
as 16-65 where small says 45-85. So Sampler (2 rooms), California or Bust! (16) and Fun House (43)
are each outside the room band of the tier the document itself puts them in. That is 4.18's tier
again, from a different direction: the two smallest columns are where §10.2 stops being a
transcription of the corpus.

**What was wrong on our side, and is now right.** The command printed a corpus claim nobody had
measured: "no shipped house is inside all eighteen of its own tier's", a sentence that was both
truncated mid-clause and false. `Check`'s doc comment said the same thing, and so did one of the two
house tests. All three now say 21 of 22 and name Leviathan as the exception.
`profile.TestTheBandsExcludeTheHousesTheyWereMeasuredFrom` parses §8.3's grouping table, measures all
22 houses against their assigned column, and asserts the count, the identity of the one house inside,
the eighteen that miss the empty-rooms row, and which of the five room ranges agree between the two
sections — with the two that disagree written as a failure if they ever *start* agreeing, because
that is the fix and the entry should close when it lands.

Worth recording as method, because it is the same lesson twice. 4.16's first amendment found a row
that "could not be computed" the moment the counting left a test file. This is the second thing the
move bought: the 22 houses could not be compared against the table *at all* while the comparison
lived inside two tests about two of our own houses. A claim about a table of tiers is not checkable
one house at a time, and the sentence that turned out to be wrong had been written from three.
**The cost of a measurement living in a test is that it can only ever be asked about the test's own
subject.**

### 4.26 Three documents describe what `make check` does, and two of them were wrong — **found and DONE, 2.4; the verbatim copy is now machine-checked, the prose is not and cannot be**

Found immediately after 4.13's `docs-check` landed, by the only method that finds this: reading the
documents as a stranger rather than as their author. `make check` is described in three places, and
the three had drifted apart.

`CONTRIBUTING.md` quotes the target's prerequisites **verbatim**, which is the right thing to do —
somebody comparing a CI log against the document should be comparing the same words — and it had
already fallen a step behind, listing the twelve steps that preceded `docs-check`. `README.md:151`
and `docs/DEV_ENVIRONMENT.md:25` each carry a prose summary instead, and DEV_ENVIRONMENT's named six
of the fourteen steps. None of the three is redundant with the others; all three were stale in
different directions.

The timing was worse, because the two numbers were an order of magnitude apart and neither said what
it was measuring. `CONTRIBUTING.md` said `make check` "takes a couple of minutes". DEV_ENVIRONMENT
said "~15 s". Measured on the airgapped host (eight cores, 2026-09-22): **50 s with Go's build cache
cold, 12 s with it warm.** So "~15 s" was the warm number offered as *the* number — the one figure a
newcomer never sees, since their first run is by definition cold — and "a couple of minutes" was not
the cold number either. Both documents now give both, and say which is which.

The verbatim copy is the half that can be held to its source, and now is:
`TestTheStepListInContributingIsTheMakefilesOwn` in `tools/docscheck` reads the Makefile's `check:`
line and fails if `CONTRIBUTING.md` does not contain it, printing the line to paste. It reads the
Makefile rather than running it, so it costs nothing and runs everywhere — including
`windows-latest`, which has no `make`.

**What is deliberately not checked.** The two prose summaries are summaries: they say "pixel corpus"
for `fidelity` and "cross-compile" for `cross`, because those are what the steps *are*, and there is
nothing mechanical to compare a paraphrase against. Reducing them to the target names would make
them checkable and worse to read, for a document whose job is to be read before anything is built.
So the arrangement is one exact copy, held by a test, and prose that no longer pretends to be a list
— DEV_ENVIRONMENT's fence comment now says "everything CI does; `CONTRIBUTING.md` lists the steps"
and keeps no third copy of its own.

**And a decision worth recording rather than leaving implicit:** `docs/DEV_ENVIRONMENT.md` is *not*
in `docscheck`'s document list, and the obvious question is why, since it is the document that tells
a new session how to get a toolchain. Twenty command lines in its `bash` fences, and two of them
could run unattended here. The rest install packages as root through three different package
managers, clone over a network this host does not have, `podman pull` a Go image, unpack a tarball
into `~/.local/opt`, or want an X server. Adding it would buy two checked lines and eighteen written
excuses, and a rules table that is ninety per cent excuses is exactly what
`TestMostOfTheDocumentedLinesAreActuallyRun` exists to catch — a check can be made to look thorough
by widening its subject until it covers nothing. Revisit if that document ever grows commands a
machine can run; the list is one line.

---

### 4.27 A repair in the 1994 editor is unreachable, and 43 objects in three shipped houses are where it would have moved them — **found and DONE as far as this port can take it, 2.4; the bug is the original's and the accommodation is ours**

`KeepObjectLegal` is the original's per-object repair pass: it forces an object's rectangle inside the
room, snaps furniture to the tile grid, and — for four object types — rewrites a wrong vertical
coordinate back to the constant the type is supposed to sit at (`HouseLegal.c:115-137`). Three of the
four work. The fourth never runs.

```c
switch (theObject->what)
{
    case kFloorVent:
    case kCeilingVent:
    ...                                   /* sixteen cases, HouseLegal.c:75-90 */
    case kLiftArea:
    ...
    if ((theObject->what == kFloorVent) && ...)     /* :115  reachable */
    if ((theObject->what == kFloorBlower) && ...)   /* :120  reachable */
    if ((theObject->what == kSewerGrate) && ...)    /* :126  reachable */
    if ((theObject->what == kFloorTrans) && ...)    /* :132  kFloorTrans is not one of the sixteen */
```

`kFloorTrans` is not in that case list. It is handled thirteen arms later, in the block for the `data.d`
transports (`HouseLegal.c:282-385`), which repairs doors, windows and `kInvisTrans` and says nothing
about `v`. So the branch at `:132-137` is dead code: a floor transporter's vertical coordinate is never
repaired, by this or by anything else.

**The corpus shows the consequence.** 43 of the 244 shipped `kFloorTrans` objects sit at `v` 300 rather
than `kFloorTransTop`'s 302 — 37 in Slumberland, 5 in Leviathan, 1 in Rainbow's End. Every other one of
the 24 vertically fixed types is at its constant in all 4,041 placements (4.23). These 43 are the only
exceptions in the whole census, and they are exactly the type whose repair does not run.

Where the 300 came from is not knowable from here, and the honest answer is that it does not matter:
`AddNewObject` does write 302 (`ObjectAdd.c:341`), so these were moved afterwards by something —
an early build, a version of `DragObject` that took `deltaV`, or a house converted from Glider 4.0 —
and with the repair dead, nothing put them back. A two-pixel error nobody could see is how dead code
stays dead for thirty-one years.

**Had it been reachable it would have been wrong anyway**, which is worth recording because it is the
more interesting half. The branch writes through `theObject->data.a` — the blower variant — and
`kFloorTrans` is a `data.d` transport. `topLeft` aliases harmlessly, since all nine variants begin with
the same two shorts. `distance` does not: `data.a.distance` and `data.d.tall` are the same two bytes, so
the `data.a.distance += 2` at `:136` would have grown the transporter's *height* by 2 as a side effect of
moving it. That the three live repairs make the same adjustment to a real `distance` — the length of an
updraught's lift column — is at least coherent for them, since the object itself has just moved; for the
transporter it would have been two bytes of collateral damage from a line that was aiming at a different
variant. It fires once per out-of-place object, not on every save, because the repair that triggers it
also removes its own trigger.

**What this port does.** `object-top` accepts 300 for `kFloorTrans` and only for `kFloorTrans`, as a
value and not a tolerance — 299 warns, 301 warns. The exemption is a named constant,
`floorTransStrayTop`, with this section cited beside it, because a bare `|| got == 300` in a comparison
is indistinguishable from the bug it is accommodating. `TestCorpusFixedAnchors` pins the 43 by house, so
a loader change that moved them shows up here rather than as a silent shift in a count.

**Not fixed, and deliberately so.** The port could normalise all 43 to 302 on load and be rid of the
special case. It will not: `docs/analysis/house-format.md` and this repository's whole approach treat a
shipped house as the specification, and 302 in Slumberland's rooms is not what Slumberland contains. The
alternative — quietly rewriting 43 objects in 3 of the 22 houses to make one linter rule tidier — is the
kind of helpfulness `TestCorpusNonCompacted` exists to forbid.

---

### 4.28 A race can only be arranged from a command line, and the guest is made to name a house it is about to be told — **note; found finishing Stage 3; the first half DONE, 2.6 — the second is a protocol change and stays open**

The networked race works and has no way in. `-host`, `-join <address>` and `-port` are the whole
interface, and `cmd/glidergo/main.go`'s dispatch puts a race on the same path the measurement flags
use — straight into the house, past the title screen — because there is nowhere on that screen to type
an address.

**Why that is worse than it sounds.** The title screen exists, has a menu and a house picker over all
22 houses (3.4, done in 1.7a), and is how every other way of starting a game is reached. A player who
wants to race has to leave it, find a shell, learn their opponent's IP address by some means this port
does not provide, and type a command line with an address in it. On Windows that is the sharpest form
of the problem: a release there is a double-clicked `.exe` from a zip, and the player has no shell in
front of them at all. The mode is, on the platform most likely to receive it, effectively unreachable.

**The second half is smaller and is the same missing dialog.** A guest must name the house on its
command line, and it does not need to: `MsgHello` already carries `HouseName`, so by the time the
handshake finishes the guest has been *told* which house the host opened. Naming it first is what makes
the house-hash gate fire, and the commonest way it will fire in practice is a guest that typed a
different house rather than one that has a different build of the same house — which is a refusal
earned by the interface, not by the houses.

**Not a blocker on the input widget.** `internal/scores`'s `Prompt`/`Field` is a working modal text
entry, written for the high-score name in 1.7c, and an address is a shorter string than a banner. What
is actually missing is a shell concept: `internal/shell` has no notion of a mode that must complete a
*network* transaction before a game can begin, so the waiting screen, the retrying dial and the
give-up live in `cmd/glidergo/race.go` and draw their own surfaces rather than being shell screens
like the other six. Moving them is most of the work; a "Race" menu item and an address field is the
small part.

**Deliberately left.** Stage 3's acceptance is about the protocol and the result, not the way in, and
the flags are enough for two people on one LAN who both have a terminal — which, on the machine this
was written on, is both of them. It is filed rather than fixed so that it is not discovered by
somebody on Windows wondering where the two-player mode went. LAN discovery, which `docs/PLAN.md`
Stage 3 already calls a nice-to-have, belongs to the same dialog: the reason a player needs to know an
IP address at all is that nothing offers them a list.

**Done, 2.6: the half that was a screen.** `internal/shell/race.go` is a Race screen on the title
menu — `R`, or the arrows to the row directly under the original's own four. Two rows and one field:
Host waits for the other machine, Join dials the address, and the address is a `scores.Field`, reused
whole for its Mac Roman limit, its select-all-on-open and its keyboard. What comes out is a
`shell.Race` on the `Choice` the shell already hands to `Play`, so `internal/shell` still has no
`net` import and still knows nothing about a socket: it arranges a race, `cmd/glidergo` runs one.
`-host` and `-join` now build the same value (`asRace`), which is the whole point of the type —
there is one path through `play` and not two, and the flags have become a way of *skipping* the title
screen rather than the only way in. `-port` stopped being refused on its own in the same change,
because it now means "the port the Race screen will use".

Four things fell out of it that this note did not predict, and each is the kind of thing that is
cheaper to write down than to measure twice.

**The menu had nowhere to put a ninth row, and the artwork is what said so.** The panel's bottom is
pinned by the house label, which the original draws at a fixed `MoveTo(436,314)`
(`SelectHouse.c:87-96`), and its top by PICT 1000: `panel()` dims eight rows beyond every edge, so a
panel any higher lays a checkerboard across the aeroplane's tail. The tail was measured off the
extracted art rather than argued about — in the panel's columns including its halo, rows 78–93 hold
ink narrowing from 105 pixels to 10, and row 94 is the first with none — and that number is now the
constant `splashSkyV`, with the geometry test asserting the halo starts on it. Nine rows fit at
`menuTop` 102 with two rows shaved off the selection bar's padding; a tenth does not fit at all, in
one column, and the test says so in its failure message so that the next person does not re-derive
it. What was *not* done, and was considered: moving the house label (it is 1994's coordinate),
narrowing the panel (too narrow for scale-2 labels), or folding "Two Player Game" into this screen
(it would bury a MENU 129 item behind a submenu).

**The row could not go where it reads best.** A race is the other two-player game, so directly under
"Two Player Game" is where it belongs by meaning — and that is one row above where MENU 129 puts
"Open Saved Game...", which `internal/shell/saved_test.go` pins on purpose. The original's order is
not this port's to rearrange, so the row goes on the first line past the end of 1994's list: still
inside the group that starts a game, still above the three that do not.

**Escape on the waiting screen had to become an error.** `errRaceGaveUp` used to be swallowed by
`play`, which was harmless when the only caller was a command line about to exit. With a shell behind
it, a zero `Outcome` and no error puts *"Fun House — score 0, 0 stars left"* on the status band, which
is a report of a game nobody played. So it is returned; the shell shows it on the band, and
`playDirect` unwraps it back into a clean exit.

**The hosting screen's worked example could not be pasted.** It printed
`glidergo -join 10.0.0.5:1138 Fun House`, unquoted, and a positional house argument is one argument:
`parseFlags` answers that line with "one house at a time". Sixteen of the 22 shipped houses have a
space in their name, so the example was broken for nearly every house it could be used with. The address is now on a line of its own and first, because that is what the other
player types into the field; the command line is underneath it and quoted (`shellQuote`, double quotes
because the line gets read on Windows as often as on a shell that understands either).

**Still open, and it is a protocol change rather than a screen.** The second half of this note — that
a guest should be *told* the host's house instead of naming it — cannot be closed by any dialog.
`netplay.Meet` is a single symmetric exchange in which both sides send a `Hello` carrying their own
house hash, so there is no moment at which the guest knows what the host opened and has not yet
committed to a house of its own. Fixing it means an asymmetric first round — the host names the house,
the guest answers — which changes the handshake, the version gate and both ends of `handshake_test.go`,
and it should be done with LAN discovery rather than before it, because a guest that was offered a
*list* of hosts would be told the house as part of the list. What the screens do meanwhile is say the
name out loud at both ends: the Race screen's last-but-one line, and the host's waiting screen. This
note also guessed wrong about where the work was: moving the waiting and dialling screens into the
shell was called "most of the work", and they have not moved and should not. They exist to be the one
place the program is allowed to block on a socket, which is precisely what a package with no `net`
import cannot hold.

**Amendment, with 4.31: the second half needs no change to the wire, and so is not closed by the
tag.** The asymmetric first round can be the guest reading before it writes. Every build's host sends
its hello as soon as it accepts. So a guest that waits for that hello, opens the house it names and
then sends its own gives a host of any release the exchange it already expects. The version gate
and the host's end of `handshake_test.go` stay as they are. That rests on one rule to keep: **a host
always sends its hello first.** It belongs next to the four in `internal/netplay`'s package comment
when this is done.

**The guest can learn the host's house from the refusal, and redial, without a handshake change.**
`Meet` refuses a mismatch after both `Hello`s are exchanged, so the guest *has* the host's house
name and hash. It just throws them into a formatted string, and the status band cuts that string
off before "theirs is".

- **Before the tag:** the refusal leads with the host's house ("the host is racing Demo House"), in
  4.33's wrap work. **Done, 4.33**, and the typed error came with it: `RefusalError`, for all four
  refusals and not only the house, carrying both `Hello`s and unwrapping to `ErrHouse`. What is
  still open is the lookup and the redial below.
- **After 4.32:** `ErrHouse` becomes a typed `HouseError` carrying both `Hello`s (it still unwraps
  to `ErrHouse`). The guest looks the host's hash up in the library it would play from —
  `options.sources()`, meaning `-houses` *in place of* the 1994 set, plus the New set; about 14 ms
  for all 22 — and redials with the match. Or it says "you don't have Demo House", or "you have a
  different build of Fun House" when the name matches and the hash does not.

**Match by hash only. `peer.HouseName` is never passed to `resolveHouse`**, which opens a disk path
when the name holds a separator or an extension (`main.go:1161-1175`). The peer's name is for
display.

Changes that go with it:
- the Race screen's Join side reads "you race whatever the host opened";
- `Outcome` carries the house actually played, because `shell.go:666`'s result line names the
  shell's selection;
- `internal/shell/race.go`'s header and `:256` are amended;
- the next CHANGELOG entry corrects the old one.

LAN discovery, if it comes: the guest broadcasts a query and hosts reply by unicast (msgType 0x31
or 0x40). Hosts do not beacon, because a listening guest would need an inbound UDP socket and might
meet a firewall prompt too (unverified; the Windows test host would settle it).

### 4.29 A new house can carry pictures now, and still cannot carry a sound — **note; found closing 4.15, 2.5**

4.15 is closed for art: a house of this port's own puts PNGs in
`levels/houseart/<House Name>/pict/`, they ride into the embedded archive, and `background 3000`
draws them. **Sounds do not work that way and were never going to.** `internal/audio.Bank.LoadHouse`
takes a house *name* and reads the **sound root's** `houses/manifest.tsv` — one shared file listing
every house's `snd ` resources across all twenty-two — rather than looking beside the house for a
directory of its own. So `levels/houseart/<House Name>/snd/` would be read by nothing, and the sound
root is `assets/extracted/sound`, which is diff-locked against the extractor for exactly the reason
4.15 says the art tree is: being byte-for-byte reproducible from the 1994 CD is that tree's whole
value.

The consequence for an author is one line of the header comment that has not changed: **no custom
`snd `**. A `kSoundTrigger` in a house of ours can name a sound the application already has and
nothing else, which is the narrower half of §10.5's `kSoundTrigger`-with-no-`snd ` trap — the trap is
now reachable for art and still not for sound.

**What it would take, and why the art fix does not generalise.** Art is looked up per picture, by id,
through a chain of filesystems, so adding a filesystem to the chain was the whole change. Sound is
looked up per house through a manifest, so the equivalent is a *second manifest*: `Bank.LoadHouse`
would have to search a list of sound roots the way `ArtRoots` searches art roots, each root carrying
its own `houses/manifest.tsv`, and merge rather than substitute — which is a shape decision about
whether a new house may shadow an original's sound as well as add its own. The levels tree would then
carry `sound/houses/manifest.tsv` plus the WAVs beside it, and `make levels` would copy it the way it
now copies `houseart/`.

**Filed rather than built, and the reason is that nothing wants it yet.** The two houses of ours use
built-in sounds only, and unlike art there is no §10.3 row saying how much custom sound a house of a
given tier ought to have — the corpus's own use of it is thin and lopsided: 63 sounds across 13 of
the 22 houses, half of them in `Leviathan`, `CD Demo House` and `Art Museum`, and nine houses with
none at all. So this is the smaller half of the same complaint, waiting on the same thing 4.15 is
waiting on: somebody with a house that wants it.

### 4.30 Nothing feeds hostile bytes to the decoders on purpose — **DONE: nine fuzz targets, the soak and `make fuzz`, and the bugs they found; a tenth with 2.79**

`SECURITY.md` names parsing other people's files as the whole attack surface, and the race now puts
a socket in front of a decoder too. There are no fuzz targets anywhere in the tree.

**Stdlib `testing.F` targets, seeded from shipped data**, so plain `go test` replays the seeds:
- `house.Load`, with a Save/Load/WriteText/ParseText round trip;
- `ParseText` plus `Lint`;
- `DecodeSavedGame` and `DecodeScores`;
- `netplay` `Recv` and `Meet` over an in-memory `ReadWriter`, with wire-frame seeds under
  `testdata/fuzz`;
- `replay.Parse`. Its round-trip property skips `House==""`, a documented precondition ("House is
  required", and the only `Write` caller guards it). At most `Write` could return an error for it.

`demo.Decode`, `prefs.LoadFile` and `ImportLegacy` are near-zero value and are left out.

**A hostile-house soak, bounded.** It mutates the shipped houses and runs them through
`replay.Run` with `ArtDir`/`HouseArtDir`/`HouseDir` pointed at `assets/extracted`. It fails on a
panic, on a run that does not return in bounded time, and on **any non-nil `Run` error**. It does
**not** assert `Diag.Guarded == 0`. The guards are the designed, reported response to a malformed
house (2.33), and a mutator that is any good trips them (11 of 4,554 plays here). The committed
default is tens of plays; 4,554 plays of 200 frames took 100 s. The soak found 2.73 on its first
honest run, because the earlier runs had ignored `Run`'s return.

**`make fuzz FUZZTIME=30s`, opt-in**, which leaves `make check` alone. Go's `-fuzz` takes one target
per invocation, so it loops over the ten or so targets (about 5 minutes at 30 s) and passes a small
`-fuzzminimizetime`, or the shipped-house seeds spend the budget minimising (7–9 execs in 21 s,
measured). PLAN §5's testing table gains the row. Short runs found no crash, because the dangerous
inputs are size-driven (4.36).

**Done.** Nine targets and the soak are in the tree. Plain `go test` runs each target over its
seeds and over any input committed under its package's `testdata/fuzz/<target>/`, so CI replays
them on every push. Each target checks the codec's own promise, not only that nothing panicked.
- **`internal/netplay`: `FuzzRecv` and `FuzzMeet`.** The seeds are built from the package's own
  encoders rather than kept as files, so they cannot drift from the format, and
  `TestEveryMeetSeedEndsTheWayItSays` checks each still ends the way it is named for. `FuzzRecv`
  requires every error to be one `meetWords` has words for, every message to decode to what its
  encoder would write, and nothing it returned to change under the reads after it (`Recv` reuses
  its buffer). `FuzzMeet` works the rules again from the bytes, and a match has to be the one
  §10.4.7 makes of the two hellos.
- **`internal/house`: `FuzzLoad`, `FuzzParseText`, `FuzzDecodeSavedGame` and `FuzzDecodeScores`.**
  A loaded house saves back byte for byte, its residue text builds back the same bytes, its
  canonical text gives back `Canonical`, and `Lint` finishes. A parsed house that fits a file loads
  and saves to itself, and its canonical text is a fixed point. A decoded save or board encodes to
  itself. The seeds are the shipped houses under 20 KB: the engine copies a seed on every mutation,
  so the 180 KB houses are left to `TestCorpusRoundTrip`.
- **`internal/replay`: `FuzzParse`.** A script `Parse` accepts is written, read back and compared
  on every field `Run` reads, then written again to the same text.
- **`internal/render`: `FuzzPicture`**, which the plan did not have. 4.36 put code of this port's
  own around `image/png`: the header check, the second open, and the two converters to a
  `Surface`. The standard library fuzzes only the decoder. It loads the bytes as art and as a house
  picture. It requires a surface or an error, never both, at the size the header says and inside
  4.36's limits, with planes that size, and with a pixel on the palette keeping its entry.
- **`internal/audio`: `FuzzHouseSounds`**, which the plan did not have either. SECURITY.md names
  the sounds beside a house as in scope, and a house's own sounds are the part of a sound tree a
  house author makes: a manifest row and a `.pcm` file each. It puts both through `LoadHouse`,
  which must load all of a house's sounds or none, and then plays each one. A sound has to hold its
  channel for exactly the samples its length and step say, and then let go.

**The soak** is `TestADamagedHouseStillPlays` (`internal/replay/soak_test.go`). Each play takes a
shipped house and overwrites 1–12 fields or bytes of a room that is not an empty slot. A quarter
of the changes go to another room, and two in five write a 16-bit extreme. The copy is used only if `Load`
takes it. Then the play flies 200 frames there with the whole asset tree and changing keys, and half
the plays start at a random point in the room. A play fails on a panic, on a run still going after
30 s, and on any error `Run` returns. Guards are counted, and they do not fail a play. Each play is
a subtest decided by its number alone, so a failure reruns by name. The committed default is 32
plays, about a second, and `-soak N` asks for more. **3,000 plays took 118 s, and all passed. 2
tripped a guard.** That is also the re-run against 2.73's fix that 2.73 was waiting for.

**`make fuzz`** runs each target for `FUZZTIME` (30 s) with a 5 s minimise, then `SOAK` (3,000)
plays. That is about six minutes. It finds the targets by name, so a new `func Fuzz` joins without
an edit, and it carries on past a failure so that one run reports all of them.

**What they found.** Each fix comes with a test that fails without it.
1. **A race opponent that died mid-message was taken for one that forfeited** (`FuzzRecv`, on its
   seeds). After a message's length has been read, `io.ReadFull` says `io.EOF` if none of the
   message came, and `frame` passed that on wrapped. The race's reader takes `io.EOF` as a clean
   hang-up and reports nothing. `frame` says `io.ErrUnexpectedEOF` there now, and the race says
   the other game ended without saying goodbye (`TestAHangUpInsideAMessageIsNotAPoliteOne`).
2. **The handshake had no words for that error either**, so a player saw Go's sentence.
   `meetWords` now says the other end hung up partway through (`racewords_test.go`).
3. **A length byte past what its string holds did not survive `-residue`** (`FuzzLoad`; the input
   is `testdata/fuzz/FuzzLoad/26ec08a46094567c`). The text wrote the 27
   bytes a room name holds and lost the byte that said 255, so the dump built back a different
   file. The residue text now carries it on a `name.length` line, and on `banner.length` and
   `entry.N.length` for the score board. Those are the only strings short enough for a byte to
   overrun: the house banner and trailer hold 255. The canonical text and `Canonical` clamp the
   byte, as `CheckRoomNameLength` does (HouseLegal.c:870-874) (`TestAnOverlongLengthByteSurvives`).
4. **A script's number too big for its field wrapped, and an argument past the last was dropped**
   (`FuzzParse`). `room 65540` ran room 4, `seed 4294967297` ran seed 1, and `frames 600 1200` ran
   600. Each is a line-numbered error now, because a bug report's script must not quietly run
   another game. A directory with a space in it (`housedir /tmp/my houses`) used to be read as
   `/tmp/my` and is refused now. That is no loss, because `glidertool` takes the directories from
   its flags.
5. **`Write` did not round-trip the way its comment said** (`FuzzParse`). It dropped a clock's
   fraction of a second, and it wrote a House or Demo containing a `#`, a newline or space at an
   end, which `Parse` reads back as something else. The clock is written RFC3339Nano now, and
   `Write` refuses those names and an empty House. Its comment names what it leaves out on purpose:
   the five directories, and the start point's fields where `Run` does not read them.
6. **An empty sound file crashed the game the first time it played.** This one came from checking
   SECURITY.md's scope against the targets, and `FuzzHouseSounds`'s seeds confirm it. A `.pcm`
   file with no bytes loaded, if its row said 0 frames or none, and `channel.next` read byte 0 of
   it on the goroutine that feeds the sound device. The music channel already refused an empty
   piece. The effects and a house's triggers did not. `Bank.load` refuses the file now, with its
   name, and `playSound` declines an empty sample as `musicChannel.begin` does.
7. **A sound's ID wrapped, and a rate could be one no header holds** (the same reading). `id 68536`
   loaded as 3000, the wrap `FuzzParse` found in scripts. A `rate_hz` of `Inf`, `NaN`, a negative
   number or anything unparsable was taken for something, and the conversion of `Inf` to a step
   is left to the machine: it played at the Macintosh rate on amd64 and ended after one sample on
   arm64. Now an ID must fit 16 bits, and a rate must be what a header's unsigned Fixed can hold,
   0 to 65,536 Hz. An empty rate is still the Macintosh rate, and every shipped rate is inside the
   range. `LoadHouse` also forgets the sounds before a bad row, so a house whose sounds do not all
   load has none, which is what the game's "no custom sounds" message already said
   (`TestABadHouseSoundIsRefused`).

The first full `make fuzz` failed once, and the fault was in the test. `FuzzParse` compared a
script's `room -2` with the `room -1` it read back. Every negative room means the house's own
start, and `Write` writes none of them. `scriptDiff` now compares the room as `Run` reads it, and
the input is kept as a seed.

**What is not covered.**
- The engine runs only by hand. CI replays the seeds and committed inputs; it does not mutate.
- `demo.Decode` and `ImportLegacy` are left out, as planned. `demo.Decode` has one error and no
  field it can misread, and the importer reads a fixed 226-byte record field by field.
  `prefs.LoadFile` was left out with them until 2.79 made `Save` build its bytes out of whatever
  `LoadFile` read. Since then `FuzzLoadSave` (`internal/prefs`) holds any file to four things: it
  loads and saves, what was saved loads with no notes and the same settings, every top-level key
  no field has is written back as it was, and a second save is the same bytes. A minute of it
  (37,732 runs) found nothing.
- The soak damages rooms, not the header, the pictures or the sounds. A damaged header either loads
  or does not, which is `FuzzLoad`'s business, and a damaged picture is `FuzzPicture`'s.

### 4.31 A race handshake that cannot say which release or engine is on the other end — **DONE, before the first tag that carries `internal/netplay`**

`Hello` carries a `Versions` bitmap, a nonce, the house name and hash, and `Meet` ignores the
bitmap: a peer advertising only v2 is accepted without error. Nothing says which release or engine
the peer is, or which rules it plays under. Once a build that speaks this is public, it can never
refuse a newer peer cleanly, and it is worse than that. Measured: if the gate lands later, a
first-generation build that is player 1 completes `Meet` before the newer peer refuses; the newer
peer hangs up, and the old build scores that as a forfeit and shows its player a **false win**.
Landing it before the tag means both sides refuse, each for itself.

**Three mandatory fields in the v1 `Hello`**, still under header version 1. No public build speaks
the race yet, so this is the layout, not an optional extension. A version bump would gain nothing
now, and after the tag it would give old builds a bare `ErrVersion` naming no release.
- **The release string** (`main.version`).
- **A simulation-only engine fingerprint**, computed under a fixed configuration (built-in assets,
  fixed seed, sound on, 9 neighbours, never the player's prefs). It comes from the built-in Demo
  House demo plus the duct script and the other cheap test scripts (0.06–0.12 s each). The demo
  alone flies one route, through 10 of Demo House's rooms, and the duct script one more, so a
  physics change elsewhere would pass. Alternatively, a hand-bumped constant that a test ties to
  the checked-in golden traces.
- **A rules byte:** a fixes bitmask plus a reserved assisted bit.

**`Meet` then:**
- refuses a fingerprint mismatch, naming both releases;
- refuses `local.Versions & peer.Versions == 0`, naming both releases;
- gates only on fixes marked as simulation-affecting — none of today's four are, since
  `MirrorFoil` and `Player2GiveUp` act only in two-player games (which a race refuses),
  `MirrorFlame` is render-only and `SwitchSparkle` draws no random number — and shows the rest as
  information.

Sound on/off is not a gate: a `kSoundIt` hot spot only plays a sound (`interactions.go:288-294`).

A test pins that `Hello`, `MatchStart`, `Standing` and `Bye` all accept trailing bytes, and the rule
"`Hello` is always sent with header version 1" is written in the package comment. `Hello`'s field
list lives in `determinism-networking.md` §10.4's amended table as well as in the code (PLAN
Stage 3's own rule).

**This is the one protocol freeze on the list.** Anything else wanted in the handshake before the
tag goes in this change: 4.38's commit-reveal, and 4.28's asymmetric second half if it is
attempted. The replay-side `fix <name>` and build lines belong to 4.39, whose grammar is defined
once with this item's fields.

**Done.** `Hello` gained three fields after `numNeighborsView`: `engine` u64, `rules` u16, and
`releaseLen` u8 followed by the release. `Meet` refuses, in this order:
1. disjoint `Versions` (`ErrVersion`);
2. another engine (`ErrEngine`, new);
3. another house (`ErrHouse`, as before);
4. a difference in a gated rule (`ErrRules`, new).

Every refusal names both releases, and is decided from the two hellos before player 1 sends
`MatchStart`. So both sides refuse, each for itself, and
`TestLoopbackRaceWithAnotherEngineIsRefusedByBothSides` checks that with a real host and a guest on
another engine. What was found on the way:

- **The fingerprint is the measured option, over more than the plan named.** The demo and the cheap
  scripts cover 11 rooms. Engine (`internal/replay/engine.go`) adds 900 frames of holding right from
  the start of each of the other twenty 1994 houses. Most of those end in a game over, which puts
  every house's opening rooms and three deaths per house in the hash. The total is 16,596 frames
  from the built-in tree: seed 1, sound on, nine neighbours, no fixes and no prefs.
  - **It costs 2 s serially, and nearly all of it is decoding PNG backgrounds.** The runs share
    nothing, and the render tables are built in `init` and only read after. So the runs go side by
    side: 0.55 s on eight cores, with the same value. That rests on no hidden shared state, which
    `TestEngineIsTheSameOnOneCoreAsOnMany` checks, and one run under `-race` was clean.
  - **It hashes only what a race can feel:** frame and parity, room, mode, the glider's rect,
    score, gliders, stars and the random stream. It leaves out the render counters, which move when
    a picture is drawn differently, and the sounds. `TestEverySampleFieldIsInOrOutOfTheFingerprint`
    fails on any new `Sample` field until it is put on one side. It also checks each side does what
    it says.
  - **The demo stream comes from the binary, never the disk.** `loadDemo` tries the path on disk
    first, so a stray `res/demo/` in the directory the game was started from would have changed the
    fingerprint. `Script` gained an unexported `demo` stream that `Engine` sets. The duct script is
    restated in code rather than read from `testdata`, which is not in the binary.
  - **Only the vendored 1994 houses.** The port's own houses are the port's to change, and the
    house hash already gates them.
  - `cmd/glidergo` computes it once per process (`sync.OnceValues`) on the connecting goroutine,
    before it accepts or dials, so a peer is answered as soon as there is one.
    `TestTheEngineFingerprintIsPinned` pins E82F4D7565428514. Its message says a change is news:
    from that build on, earlier releases are refused.
  - The loopback races use a stand-in (`a.engine`). The real fingerprint takes 25 s under the race
    detector.
- **The rules field is split by position.** A difference in the low byte is reported, and one in the
  high byte (`RulesGated`, 0xFF00) refuses. So a build refuses a later build's rule, or lets it
  through, without knowing its name, and refusals name unknown bits by number. The four fixes are
  bits 0–3, informational for the reasons above. Bit 15 is reserved for an assist a race allows,
  and nothing sets it: the game has no assists (3.2). **2.38's `fixes.switch_star` goes in the high
  byte when it lands**, because it changes `StarsLeft`. `TestEveryFixHasARaceRule` checks that
  every `prefs.Fixes` field has its own bit, named as the prefs file names it, and in the low byte.
- **The four rules that keep this extendable are in `internal/netplay`'s package comment and in
  §10.4.7:** a `Hello` is always sent under header version 1, its layout only grows at the end,
  every decoder accepts trailing bytes, and a rule gates by its position. The decoders already
  accepted trailing bytes. `TestEveryMessageAcceptsTrailingBytes` now checks all four messages.
- **Two builds with the same release name and different engines** get their own wording ("at
  least one of them was built from changed source"), and "dev" against "dev" is the common case. A
  build with no release is "an unnamed build".
- **Commit-reveal was weighed and left out** (4.38 has the reasoning). `mixSeed`'s comment no
  longer claims neither peer can choose the seed.
- **4.28's second half needs no wire change**, so the freeze does not close it. A guest can read
  the host's hello before sending its own, because every build's host sends first. See 4.28's
  amendment.

**Left for 4.33, and done there.** The informational differences — the other side's rules, release and neighbour
view — reach only the stdout race line. The screen does not mention them. The refusal messages are
the ones 4.33 will sort by cause and wrap on the plate. 4.33 put the release and the rules in small
print on the race's screens, and left the neighbour view on stdout.

### 4.32 One stray connection ends hosting, and a silent host leaves a guest on JOINING forever — **DONE, before the next tag**

`openRace` accepts once and closes the listener, and `Meet` has no read deadline. So:
- **a house-mismatched guest ends hosting** (measured: a Fun House guest against a Slumberland
  host leaves no listener, and on the command line the host exits with the bug-report footer).
  4.28 says a guest who typed a different house is the commonest way the gate fires;
- **a silent connection blocks the host until Escape**, while the real guest is refused;
- **a guest that dials something that accepts and never speaks hangs** on JOINING with no note
  (measured against `http.server`). Something that speaks first gets "protocol violation" and the
  footer (measured with an SSH banner).

**The fix:**
- Loop on `Accept` in `meetRace` (or a `netplay.Listener` helper). Each connection gets a ~5 s
  handshake deadline, cleared once `Meet` succeeds. 4.31's engine fingerprint is worked out before
  `Accept` and before the first dial, so neither deadline has to allow for it.
- **Any `Meet` error before a match exists drops that connection**, is shown through
  `rc.note`/`lastNote`, and the loop continues. That includes `ErrVersion`, the nonce tie and
  write errors, with no list of which errors count. The listener closes only after a `Meet`
  succeeds.
- The guest's `Meet` gets a longer deadline (~15 s). A sequential host loop can leave the real
  guest in the kernel backlog while a silent connection times out, so the guest's deadline must be
  longer than the host's.
- An out-of-range first length is reported as `ErrMagic` ("not a gliderGo peer"), wrapped together
  with `ErrProtocol` (Go 1.23's multiple `%w`), without the footer.

**Tests:** the second-guest test is rewritten to "shut after the first *match*" (now
`TestAListenerStaysOpenUntilItIsClosed`), and
`TestFrameRejectsImpossibleLengths` (`netplay_test.go:182`) still sees `ErrProtocol`. `dial_test`
gains a stray HTTP request, a silent connection, and a house-mismatched guest followed by a matching
one. Internet play with port forwarding is not a stated goal. The argument is the LAN.

**Done**, as planned, with these specifics:

- **The loop is in `cmd/glidergo` (`hostRace`), not in `netplay`.** Each accepted connection has to
  be handed to `raceConnect` so that Escape can close it, and that hand-over is the app's.
  `Listener.Accept` no longer closes the listener. `meetRace` closes it on every way out, so the
  port is still shut once a match is agreed.
- **`netplay.MeetWithin`** is `Meet` with a deadline over the whole handshake, cleared on success.
  It gives two different messages: "the other end said nothing for 15s, so it is not a gliderGo
  game ready to race" when nothing arrived, and "stopped answering partway through the handshake"
  when something did. `Conn` gained a `heard` flag to tell the two apart. The waits are
  `hostHandshakeWait` (5 s) and `guestHandshakeWait` (15 s), and they hold in a player's run as
  well as in a bench run, because a silent connection is not a wait anybody chose.
- **Every `Meet` error turns the connection away, with no list of which errors count.** The note
  is `turned away <host>: <why>`. It carries the machine, without the port, because the machine
  is what tells the guest being waited for apart from a stray. It goes on the waiting screen, to
  stdout, and into a bench host's timeout ("no race started within 30s; turned away …"). A guest
  does not redial after a refusal, because it would only be refused again.
- **The impossible first length** is `strangerErr`, which unwraps to both `ErrMagic` and
  `ErrProtocol` and quotes the four bytes. The same bytes later in a stream stay `ErrProtocol`
  only.
- **The footer.** `refusedByPeer` (`ErrMagic` and the four gates) leaves the bug-report footer
  off, because the message already names which machine to change. A handshake timeout keeps the
  footer, since a stuck gliderGo host is worth a report.
- **Tests.**
  - `dial_test.go`: `TestAListenerStaysOpenUntilItIsClosed` (the rewrite), MeetWithin's two
    timeouts, a deadline that does not outlive the handshake, and the first-length split.
  - `loopback_test.go`, where the loop is: the stray HTTP request, the silent connection and the
    mismatched guest before a matching one (`TestLoopbackRaceHostTurnsAwayWhatIsNotARace`, which
    also checks the port is shut once the race is flying), and a guest that dials an SSH-style
    banner.
  - `TestLoopbackRaceHostThatNeverSpeaksIsGivenUpOn` now pins the words.
  - The engine-refusal test checks the host's side through its note, because a host no longer
    returns on a refusal.
- **Measured with the binary.** A `-bench -host` over Grand Prix turned away curl ("GET ") and a
  Fun House guest, and then raced a Grand Prix guest. The Fun House guest's refusal printed no
  footer. A guest joining a socket that reads and never writes gave up after 15 s, in the words
  above. One joining an SSH-style banner was told `"SSH-"` at once. `python3 -m http.server`
  answered this time, with its HTTP/0.9 error page, so the guest was told `"<!DO"` in 1.3 s rather
  than waiting.

### 4.33 A failed join runs off the screen and blames the other machine — **DONE, before the next tag (the UPnP half is a separate note, not planned)**

`netplay.Join`'s one piece of advice for every failure is "the other machine has to be hosting"
(`race.go:238`). The error line starts on the plate's left frame and runs off the right edge of the
screen, and on a refusal the word "refused" is itself cut off. Nothing in the repository mentions a
firewall.

**Sorted by cause:**
- **refused** — the machine answered but is not hosting yet;
- **no such host** — check the spelling;
- **unreachable** — the address cannot be reached from this network;
- **timeout** — no answer. That is *either* not hosting yet *or* a firewall or router dropping port
  1994 (on Windows, allow gliderGo when asked, on the network you are on).

A timeout does not mean a firewall. A Windows guest waits about 2 s before a closed port reports
refused (Go's `fd_windows.go`), and `joinTimeout` is 1 s. A Windows host, and a Linux host with ufw
or firewalld, drop connections to closed ports silently. So:
- the dial goes through `net.Dialer.DialContext`, cancelled by `rc.cancel`, so the timeout can
  grow to about 3 s without slowing Escape;
- the firewall line gets prominent only after repeated timeouts.

**The Windows file.** The stdlib has no `WSA*` constants for these, and `syscall.ECONNREFUSED`
never matches 10061. So a windows-only file compares `syscall.Errno(10061)`, `(10065)` and
`(10051)`, table-tested, and kept tiny because nothing here can exercise it.

**The rest:**
- The line wraps inside the plate. The plate is 200 px of a 480 px screen and can grow; with two
  addresses the hosting screen uses 8 of its 9 slots.
- The hosting screen lists the default-route address first and marks `IsPrivate` addresses "this
  network only". It does **not** mark 100.64/10 as CGNAT: a local interface there is almost always
  Tailscale, exactly the overlay the README will recommend.
- The refusal leads with the host's house (4.28's amendment), in the same wrap work.
- **Docs:** firewall text in the Windows `HOW-TO-RUN`, and a README section on racing beyond your
  network (port forwarding, either side can host, VPNs work, IPv6 avoids forwarding but home
  routers and the host's own firewall usually block incoming). SECURITY.md changes in the same
  commit (5.7), because the port-forwarding advice is exposure.

**UPnP, NAT-PMP and PCP — a note, not a plan.** The stdlib has no portable routing-table read for
the gateway, many routers speak PCP rather than NAT-PMP, UPnP comes in several service versions,
and none of it can be tested against a router from here. It also widens exposure, and it would need
4.32, a handshake deadline and 4.30's netplay fuzzing first. L, and not wanted until someone asks.

**Amendment (4.34): the failure line during a race is Go's too, and it goes in the same sort.** A
peer whose process ends mid-race resets the connection, because it dies with standings unread. The
survivor's result is right (a forfeit), but the result screen adds "the connection failed:
netplay: sending MsgStanding: write tcp 127.0.0.1:42649->127.0.0.1:59500: write: broken pipe". It
runs off the plate as the join line does, and Windows words it differently. After the handshake a
reset, a refused write and an EOF are one cause: *the other player's game ended without saying
goodbye*. That sentence goes on the plate, and the Go error goes to stdout, where a bug report
quotes it. The loopback test logs the error and does not pin it. Separately, a guest whose host
accepted and then said nothing is told "could not join a race within 30s", though it did join.
That is a handshake timeout, and it needs its own words, which 4.32's handshake deadline names.

**Amendment (4.32): the host's "turned away" note and the handshake's new errors go in the same
sort.** They are long, and so is the chain behind them. For example: `turned away 127.0.0.1:
netplay: waiting for the other player's hello: netplay: not a gliderGo message: the other end
opened with "GET ", …`. On the plate that should be "turned away 127.0.0.1: not gliderGo (it
opened with "GET ")". The full chain goes to stdout, where it already is.

**Amendment (4.31): the handshake refusals go in the same sort, and the differences that do not
refuse go on the screen.** `Meet` now refuses for four causes, and each has its own sentinel:
`ErrVersion`, `ErrEngine`, `ErrHouse` and `ErrRules`. Each message names both releases and is a
sentence or two long, so it needs the wrap more than the join line does. On the plate, the refusal
starts with what to do ("the other player needs release 0.4.0", "open Grand Prix"), and the full
text goes to stdout. Three differences are allowed and are shown only on the stdout race line
today: the other side's release when it differs, a fix it has on that this side does not
(`PeerRules`), and its neighbour view. The first two belong on the race's waiting and result
screens in a line of small print. A player beaten by a glider with `fixes.mirror_foil` on should
be able to see that.

**Done: every failure a race can end on is sorted by cause and said on the screen as what to do.**
The terminal still gets the full sentence, word for word what it was. The words are in
`cmd/glidergo/racewords.go`, and every line puts the advice first, so a line that gets cut loses
the explanation and keeps the fix.

- **`netplay.Join` takes a context and returns a `*JoinError`**, which says which address failed
  and why. The cause is a `JoinFailure`: refused, no such host, unreachable, timed out, or other.
  - The Windows errnos are in `dial_windows.go` (10061, 10051 and 10065), and every other platform
    uses `dial_errno.go`.
  - `TestJoinFailuresAreSortedByCause` wraps each cause as `net.Dialer` does, and it is the only
    test the Windows numbers get.
  - `TestJoinSortsARealRefusalAndARealMissingName` checks the two causes this machine can produce
    for real: a closed loopback port, refused in 160 µs, and a `.invalid` name.
  - A resolver that times out is sorted as other and not as timed out, so a slow DNS server never
    reaches the firewall advice.
  - The old "-- the other machine has to be hosting" suffix is gone from `JoinError`'s text.
- **`joinTimeout` is 3 s, up from 1 s**, and the dial runs under `raceConnect`'s context, so Escape
  still ends it at once (`TestJoinStopsWhenItsContextEnds`, `TestCancelEndsADialInProgress`).
- **The firewall is named after three unanswered dials in a row** (`firewallAfter`), which at 3 s a
  dial is about ten seconds. Any other answer, even a refusal, proves the path is open and resets
  the count (`TestAFirewallIsNamedOnlyAfterSeveralSilences`, `TestAnAnswerResetsTheSilenceCount`).
- **The handshake's failures are typed**, and their `Error` text is unchanged.
  - `RefusalError` covers all four of `Meet`'s refusals and carries both `Hello`s. It unwraps to
    the sentinel, so `errors.Is(err, ErrHouse)` still works, and `refusedByBoth` checks that each
    side's error holds both nonces the right way round.
  - `SilenceError` is `MeetWithin`'s two timeouts.
  - `strangerErr` is exported as `StrangerError`, so that a screen can quote the four bytes.
- **A guest turned away over the house is told the host's house**:
  `the host is racing "Fun House": open it to join`. That is 4.28's first step. A guest with the
  same name and a different hash is told to get the host's copy. The host is told what the guest
  opened.
- **What a peer sent is quoted and cut to 40 runes on the screen** (`peerText`), because a house
  name can be 255 bytes and can hold an escape sequence (`TestThePeersOwnWordsAreQuotedAndCut`).
- **The status band shows an error's `Brief()` when it has one.** That is `internal/shell`'s
  `brief`, tested by `TestAnErrorsBriefLineIsWhatTheBandShows`. `openRace` wraps its failures in a
  `briefErr`, whose `Error` is the full sentence for stderr.
  `TestEveryWayAGuestIsTurnedAwayFitsTheBand` checks that each line fits after a house name.
- **The plate is 300 px tall, up from 200, which is 16 lines instead of 9.** Every line wraps
  inside a 16 px margin (`wrapRace`, `TestAWrappedLineStaysOnThePlate`), splitting a word only when
  it is wider than the plate.
- **The small print is 4.31's leftover, now closed.** The waiting and result screens list, in grey
  under the body, the other side's release when it differs and every fix it has on or off where
  this side does not (`smallPrint`). The neighbour view stays on the stdout line. It is a
  preference and not a rule, and it changes what a player sees, not what the glider does.
- **The result screen says "the other player's game ended without saying goodbye"** in place of
  Go's broken-pipe sentence (`endWords`), which is the 4.34 amendment. A peer that sent something
  unreadable is told apart. stdout keeps the Go error.
- **The hosting screen reads out the default route's address first** (`defaultRoute`, a UDP
  "connect" to TEST-NET-1 that sends nothing).
  - It lists at most three addresses and marks the `IsPrivate` ones "(this network only)". 100.64/10
    is not marked, because a player's machine with an address there is almost always on Tailscale
    (`TestTheDefaultRoutesAddressIsReadOutFirst`, `TestAnAddressOnlyThisNetworkCanReachIsMarked`).
  - The shell command uses the first address.
  - On Windows a line says to allow gliderGo when Windows asks. The moment the screen goes up is
    the moment Windows asks.
- **The "last attempt:" prefix is gone.** The note is a sentence of advice now, and "last attempt"
  read as though there would be no more.
- **Docs.**
  - The Windows `HOW-TO-RUN.txt` has a *Racing another machine* section: the firewall dialog, what
    Cancel does, and how to undo it.
  - The release notes had no word about racing at all, which was a gap in themselves. They gain a
    section.
  - The README has what to do when the other machine cannot be reached, with the ufw and firewalld
    commands, and *Racing beyond your network*: a port forward, a VPN, and IPv6. Its broken line
    wrap at "`-two` and `-resume`" is fixed.
  - SECURITY.md is rewritten (5.7) in the same change.
- **Measured.** The three screens were rendered at 640×480 and read:
  - the hosting screen, with two addresses, the Windows hint and a turned-away note, uses 12 of its
    16 lines;
  - the firewall line wraps to two lines;
  - the result screen's small print sits under the way out.

  Two binary runs checked the terminal's side:
  - a `-bench` Slumberland host against a Fun House guest ends the guest at once, and the full
    sentence still goes to stderr;
  - a refused `-join` gives up at 30 s with the `JoinError` text.

**Not measured, and why:**
- **The Windows side of the sort.** How long a Windows guest takes to say "refused" (about 2 s)
  comes from Go's `fd_windows.go`, not from a run. The errno table is the only test the Windows
  numbers get.
- **The Windows firewall dialog's wording in the HOW-TO-RUN.** The Windows test host is Server
  2025, and nothing here clicked Host on it.

**Found doing it: a stream of connections can keep a host from racing.** The host deals with one
connection at a time, and each silent one holds it for up to five seconds. Anything that keeps
connecting therefore keeps the real guest waiting in the kernel's backlog. SECURITY.md now says so,
with Escape as the answer. Handshakes on concurrent connections would close it, and would need a
rule for which match wins. Not planned. It is a nuisance on a port that is only open while a player
watches it.

### 4.34 `race.go` has no app-level test, and four changes are about to edit it — **DONE, the race chain's first step: four loopback races, 0.3 s under `-race`**

**An app-level loopback race in `cmd/glidergo`.** Two in-process apps (`newApp`, an idle Window,
`platform.NewFramebuffer`, bench plus a short frame limit) meet over 127.0.0.1 through `a.play`
with `shell.Race{Host}` and `shell.Race{Join}`. It covers three cases:
- **a normal race**, where both sides agree on the match and the outcome. A scratch prototype
  passed under `-race` in 4.2 s. Frame-limited runs both score as "left", so a decided winner needs
  a house the idle glider can finish or die in.
- **a guest that drops mid-race**, played by a raw netplay peer that closes its socket with no
  goodbye. A process cannot be killed in-process.
- **a host that accepts and never speaks.**

**The seams it needs:**
- a hook for the listen address or the resolved port, because `openRace` binds
  `netplay.Listen("", o.port)` on every interface and with port 0 the guest cannot learn the port.
  Binding 127.0.0.1 also avoids firewall prompts on the Windows and macOS CI matrix;
- `raceConnectWait` and `raceSettleWait` turned from consts into vars;
- an idle window, because `fakeWin` sends `EventQuit` when its script runs out.

**`make race` gains `cmd/glidergo`, filtered with `-run 'Race|Loopback'`.** Unfiltered, `make race`
goes from about 1.8 s to about 21 s warm, and `make check` runs it at `-count=2`. `internal/audio`
joins once 2.71's harness starts the pump; `waveout_windows.go:290` is a third goroutine no Linux
run can check.

The claim that one package has goroutines is corrected in three places: `Makefile:429-460`,
`Makefile:610-611` and `CONTRIBUTING.md:46-50`. CHANGELOG's Stage 3 entry is history and stays.
`race.go` is about 13% covered by statements.

**Done: `cmd/glidergo/loopback_test.go`, four races through `a.play` over 127.0.0.1.** The house is
Grand Prix, because an idle glider dies in it (frame 351) and one holding right dies further on
(frame 771, 8 rooms). That gives a decided race with no frame limit.
- **The normal race**, idle host against flying guest. The two sides must hold one match from
  opposite slots and score each other's run exactly as it was flown. Both must settle, and both
  must name the same winner, not by forfeit. `play` must hand the shell the same words the result
  screen shows.
- **Two idle gliders from one seed draw**, which the plan did not list. Their standings match to
  the frame, and their random streams end equal although the two apps start from different ones.
  That is the check that the handshake's seed is the one both Worlds ran from. It is also
  `determinism-networking.md` §10.4.9's proposed draw vector, arrived at with no recording.
- **The guest that hangs up mid-race.** A raw peer hears the host's first report, sends one room and
  closes without a goodbye. The host settles without waiting and flies its own run to the end. It
  scores the guest through `Abandoned` and wins by forfeit. It is the likeliest cause of the CI
  failure on its first run on GitHub, and is two subtests now (4.48).
- **The host that accepts and never speaks.** The guest gives up at `raceConnectWait` and closes its
  socket, and the silent end hears its hello and then the end of the stream.

The seams are the three planned: `app.listen`, which is nil in the program (`hostListener`), the
two waits as vars, and `idleWin`. **A fourth made `-race` affordable, and it is in the test alone:
the framebuffer has no pixels.** The first cut took 94 s under the detector, and the normal race
failed in it. The host's settle wait ran out while the guest was still flying, so the two sides
disagreed. The profile put nearly all of it in `ToBGRX`: every present converts 640×480 pixels for
a window that throws them away, and under the detector every byte stored is a call. `ToBGRX`
already leaves alone any row the destination cannot hold. So an empty framebuffer skips it, without
a line of production code, and the four races take 0.3 s. The prototype's 4.2 s was frame-limited.

**What writing them found**, fixed here unless it says otherwise:
- **Both players leaving was headed "A DRAW".** The heading tested the winner's slot alone, and
  `{Slot: -1, ByForfeit}` has no winner either. `raceResult.verdict` says "a draw" only for
  `Drawn`, and "no result" otherwise.
- **The opponent was read from the Race three times**, for the result, the stdout line and the
  screen's "them:" line. A report that arrived between two of those reads could print one run and
  score another. That can happen whenever the wait ended unsettled. `finishRace` takes one
  `raceResult` snapshot, and everything after the run reads that.
- **The title screen forgot the race.** A race started from `Race...` came back to a status band
  that showed the score, which in a race is a tie-break. `shell.Outcome.Race` carries the result
  instead: `Slumberland -- race: you win, more rooms visited`.
- **A peer that dies mid-race arrives as a reset, not an EOF**, because it dies with standings
  unread. The host's next write is refused, `Race.Err` is set, and the result screen adds "the
  connection failed: netplay: sending MsgStanding: write tcp …: write: broken pipe". The result is
  right, and the words are Go's. They are 4.33's (its amendment). `netplay`'s `read` comment
  claimed EOF, and now says this.
- **A guest whose host goes silent is told "could not join a race within 30s"**, which is wrong,
  because it joined. That wording is 4.32 and 4.33's, and the test does not pin it.

**`make race` is three places now, in 6 s:** `internal/netplay`, `internal/audio`, and
`cmd/glidergo` with `-run 'Race|Loopback'`, which picks out the four loopback tests (about 3 s
at `-count=2`). `race_test.go`'s tests check the hosting lines and start no goroutine. `internal/audio`
joined in the same change and did not wait for 2.71: widening the target needed a test that
starts the pump. `pipe_test.go` is three of 2.71's four modes, and it found two bugs (2.71's
amendment). The goroutine claim is corrected in the three places listed above. `race.go` is 71%
covered by statements.

### 4.35 A double-clicked crash on Windows leaves nothing behind — **DONE: a crash file, and a console that waits**

Double-clicking the `.exe` is a documented way to launch it, and when the process exits the console
closes with it, taking any panic trace or error. This first Windows release is a field test in which
no key has been pressed, and this is the maintainer's only telemetry from it.

**On the interactive paths only.** Never on `-frames`/`-bench`/`-shot`/`-dump` (2.53's hermetic
rule, including CI's windows `-frames 300 -bench` step), and only after the `-version`,
`-audio list` and `-import-prefs` early returns:
1. Read the previous run's `<datadir>/crash.log`. If it holds anything after its header
   ("panic:", "fatal error:", "Exception"), rename it `crash-last.log` and put "the last run
   crashed; details in <path>" plus the issue URL on the status band.
2. Reopen `crash.log` `O_TRUNC|O_APPEND`, so two local race instances interleave. Write the whole
   `-version` block as its header (`printVersion` takes an `io.Writer`), and call
   `runtime/debug.SetCrashOutput`. Appending a timestamp to a persistent file would make "it grew"
   true after every clean run (34 bytes, measured).
3. `-version`'s state rows list the path.

`SetCrashOutput` records only unrecovered panics and runtime fatal errors. Most of what `main`
prints are errors returned by `run()`, so on Windows, when `GetConsoleProcessList` returns 1 and
the exit is non-zero, it waits for Enter. That keeps every earlier stderr note visible, which a
`MessageBoxW` of the last error would not. On Linux `main` also appends the error line to the file.

`-H windowsgui` with `AttachConsole` belongs in a **separate M note in §5**, not yet written, to be
done after 2.75 lands. It needs stdout and stderr reopened on `CONOUT$`, cmd and PowerShell return
to the prompt before a GUI-subsystem program prints (which spoils the `-version` paste the footer
asks for), and it removes the console the release notes call the diagnostic channel.

**Done, PLAN release gate step 4.** A run a player starts keeps `crash.log` in the data
directory's root, beside `scores/` and `saves/`:
- `~/.local/share/glidergo/crash.log` on Linux;
- `%AppData%\glidergo\crash.log` on Windows;
- `~/Library/Application Support/glidergo/crash.log` on macOS;
- the directory itself under `GLIDERGO_DATA`.

The file is the whole `-version` block, then `started` and `args` rows, then a rule line. Below
the rule is whatever stopped the run: the runtime's report if it died, or the error `main` stopped
with. The next run a player starts looks below the rule. A crash there is renamed
`crash-last.log`, and stderr and the status band say so; anything else is overwritten. Measurement
runs keep none, and `-version`'s new `crash` row says "nowhere" for them. `cmd/glidergo/crash.go`
and `console_windows.go` are the code.

Where it differs from the plan above:
- **An error is written but not kept.** `main` writes the error below the rule on every platform,
  one `glidergo: ` line per line of it, but the next run does not call it a crash. Almost every
  error `run()` returns is one the player was shown and can fix (a house that is not there, no
  display), and a band that said "the last run crashed" after each of those would be wrong most
  of the time. The line is there for a desktop launcher whose stderr went nowhere, until the next
  start. The prefix is how the two are told apart: nothing the runtime prints as it dies starts
  with it.
- **Two writes, not one `O_TRUNC|O_APPEND` open.** The header is written with a plain truncating
  write, and the file is opened again with `O_APPEND` for `SetCrashOutput`. On Windows Go opens an
  `O_APPEND` file with `FILE_APPEND_DATA` access only, and whether `CREATE_ALWAYS` truncates with
  that is not worth finding out on an unrun platform. Two copies on one machine still interleave.
- **A timestamp, in the header.** The objection above was to appending one to a file that
  persists; this one is rewritten at every start, so `started` does not make it grow.
- **The band has no room for the URL.** At about 90 characters it takes "the last run crashed;
  its report is in " and a path, written as a player would type it: `%AppData%\...` on Windows
  (Explorer's address bar and Win+R expand it), `~/...` elsewhere. That is 426 px of the 568 a
  release's band has. stderr has the full path and the tracker, and the file's own `bugs` row
  says where to send it.
- **The rename falls back to a copy**, for Windows, where a file another copy of the game has open
  cannot be renamed.

**Verified on Linux, the binary.** A sandboxed title screen sent `SIGQUIT` left its 257-line
goroutine dump under the header. The next start printed both stderr lines, and its band said
"the last run crashed; its report is in /tmp/ggsand/data/crash-last.log" (captured from the
Xephyr window). The start after that said nothing and left `crash-last.log` alone. A killed X
server leaves the header and nothing below it, which is 2.78 as predicted.

**Tests.** `TestACrashIsKeptForTheNextRun` runs its own binary as a game that panics on a goroutine
of its own. Nothing but the runtime writes the report it then finds under the header, and it checks
the next start keeps it and the one after does not. `TestAnErrorIsNotACrash` feeds an error with a
newline in it. `TestWhatCountsAsACrash` covers a panic, a fatal error, a Windows exception, a signal
in C, an error and no rule. `TestTheBandWritesThePathAsAPlayerWouldTypeIt` and
`TestVersionSaysWhereACrashIsKept` cover the band and `-version`, and the shell's
`TestANoticeIsTheFirstThingTheBandSays` covers the notice. Four mutations were checked, and each
fails a test:
- no `SetCrashOutput`;
- the prefix on the first line only;
- a notice without the rename;
- errors counted as crashes.

**Not run on Windows:** the console hold (`GetConsoleProcessList` returning 1, then a wait for
Enter) and the file under `%AppData%`. Both are on `docs/windows-first-run.md`'s list. A panic
still closes a double-clicked console at once; the file is what keeps it.

**Found with it:**
- A library of one house opened with "1 houses", and a test pinned it. It says "1 house" now, and
  "1 file skipped". A search for the same `%d nouns` shape found it on every line a player reads:
  - the band after a game ("1 stars left");
  - a house of one room, on the band and in the picker;
  - a set of one house;
  - the race's opponent panel, "them: 1 rooms ... 1 gliders". Its count is rooms *left*, so every
    race passed through it at the first door.

  All of them, and the stderr lines of a timed run ("1 music pieces"), use one `counted` helper
  now, and the band and the panel are pinned by tests. `glidertool`'s lines are left as they are,
  because they are for contributors.
- `statePaths`'s doc comment had drifted onto `audioRoute`, which then began with a paragraph
  about something else. It is back on `statePaths`.

### 4.36 A picture under 1 MB can cost 2 GB — **DONE: the PNG cap, and the smaller items with it**

`housepict.go`'s `loadHousePict` and `assets.go`'s `load` decode whatever PNG they are given. A
300–800 KB file declaring 16384×16384 costs 0.5–2.1 GB (reproduced; about 20 fps after a 9 s
stall). It is reachable today through the documented `-levels`/`-houseart`/`-art` flags.

**The fix:** `png.DecodeConfig` first (re-opening the file, because zip entries cannot seek). Refuse
anything over about 4 Mpx — 14× the largest shipped picture — with a looser side cap of about 4096,
through the existing `a.fail` path. 2048 would leave only 1.33× over the application art's 1536 px
strip, and `-art` loads that tree too. A test builds the 16384² PNG at runtime.

**Smaller, and they can follow:**
- the `0x7FFF` room check moves into `text.go`'s `beginRoom`, because an 11.9 MB text allocates
  1.9 GB before it is rejected. Only `glidertool house build` reaches the parser;
- a `Stat` size refusal above 11,403,784 bytes (866 + 348×32767 + 2) in `house` `peek`, so the
  file is never listed, and in `LoadFile`/`LoadFS`, with `io.LimitReader` for `fs.FS`.
  Binary reads cost 1× the file size and then refuse trailing bytes, so this is minor.

The library walk is **not** a threat: `Discover` calls `PeekFS`, which reads 866 bytes. Demo files
arrive only through glidertool and expand about 2.7×. A zip-entry cap is a stated requirement on
4.41's packs, since no user-supplied zip is read today. SECURITY.md (5.7) points at this item for
its size limits.

**Done.** A picture is read for its header first, and one too large to be art is refused before
anything is allocated for it. So are a house file too large to be a house, and a house text with
more rooms than a house can count.
- **Pictures** (`internal/render` `decodePNG`, used by `assets.go`'s `load` and `housepict.go`'s
  `loadHousePict`). `png.DecodeConfig` reads the header, then the file is opened again, because a
  zip entry cannot seek. A picture over 4096 px on a side, or 4 Mpx in all, is refused through the
  existing `a.fail` path, with its size and the limits in the message. Measured on the binary
  with a `-houseart` tree of 782 KB, 16384² PNGs: **11.85 s to the first frame, 2.13 GB RSS and
  2.6 fps before; 0.12 s and 18 MB after**, with the refusal on the terminal. The largest shipped
  picture is 640×460 (294,400 px), and the widest side is the 1536 px strip, so neither limit is
  near what ships (1,098 PNGs, all loading).
- **House files** (`house.MaxFileSize`, 11,403,784 bytes: the header, 32,767 rooms and the
  slack). `peek` refuses a file past it from `Stat`, so the picker never lists it. `LoadFile` and
  `LoadFS` read through `io.LimitReader` to one byte past it, because a size from `Stat` is not
  enough: `/dev/zero` says it is empty and never ends. Before, `glidergo -house /dev/zero` read
  until it ran out of memory. Now it is refused in 0.06 s.
- **House texts** (`text.go`'s `beginRoom`). Room 32,767 is refused when it begins, not after
  every room has been built. An 18.4 MB text of 1.5 million rooms: **2.00 s and 2.36 GB RSS
  before; 0.04 s and 46 MB after**, refused at line 32,769.

The tests are `TestAPictureTooLargeToBeArtIsRefusedFromItsHeader`, whose runtime-built 16384²
header allocates 1.07 GB without the cap and under 1 MiB with it; `TestThePictureLimitsAreWhereTheySay`;
`TestAPictureInsideTheLimitsStillLoads`; `TestTheLargestHouseStillLoads`, which is every room
nRooms can count, plus the slack; `TestAFileLargerThanAHouseIsRefusedBeforeItIsRead`, on a
sparse file, for both `peek`s and both loads; `TestAFileThatNeverEndsIsReadOnlyAsFarAsAHouseCouldGo`,
on an `fs.FS` whose file never ends and on `/dev/zero`; and
`TestATextStopsAtTheFirstRoomAHouseCannotHave`, which feeds the parser a text that never ends.
Each of those could only pass by stopping.

The entry's own figure was 200 bytes out: it said 11,403,984, and 866 + 348×32,767 + 2 is
11,403,784. The constant is computed, and a test pins it.

**Found doing it: a link to a house was left out of the list, silently.** The library walk took
only regular files, and a symbolic link is not one. A player who linked a house into a `-levels`
directory saw nothing, and no reason, which is what 2.33 is against. The original followed a
Finder alias to a house (`HouseIO.c:177-178`). So the walk follows a link now. A link to a file
is listed as that house and opens. A link that leads nowhere is reported in Skipped. A link to a
directory is not walked into, because links can loop. A link to a pipe is left out, because
opening a pipe waits for a writer. That is safe to do now because of the size check above: a link
to `/dev/zero` is refused, not read (`TestDiscoverFollowsALinkToAHouse`). SECURITY.md said links
were followed. For houses they now are.

**Still open, and not this item's:** a zip-entry cap stays a stated requirement on 4.41's packs,
because no zip a player supplies is read today. The demo reader (`internal/demo`) reads a whole
file with no cap. The game only reads the demo built into it (`replay.Engine`), and `glidertool
demo` reads the file it is given. So the cap matters only once a demo can arrive from somebody
else, and nothing plans that. Whatever first does should add the cap.

### 4.37 The race sends a standing every frame and nothing when the other side goes quiet — **planned, after the next tag; the README correction and a freeze it turned up DONE in PLAN release gate step 5**

`Report` compares whole standings, `Frame` included, so the "on change" test fires every frame:
about 30 messages a second and ~1 KB/s. `netplay.go:9`, `standing.go:179` ("a few hundred bytes a
minute"), `conn.go:28`, PLAN Stage 3's driver bullet and CHANGELOG's Stage 3 entry say
otherwise. The bandwidth is harmless. The comments are wrong, and the waiting screen is blind.

**The fix:**
- The change test leaves `Frame` out, but `Report` keeps the newest standing with its current
  `Frame` even when it does not wake the writer.
- Race's writer resends that standing about once a second, and the reader records when it last
  heard from the peer.
- The waiting screen shows the opponent line plus "flying", "paused N s" or "last heard N s ago".
  **Display only.** Turning silence into a forfeit would let a network split score two different
  results, which breaks the rule that both peers agree without a referee.

A `Report` test runs with a rising `Frame` and an injectable heartbeat interval.
`TestReportSendsNothingWhenNothingChanged` still guards a real case (116–160 `Present`s in one
frame during a wipe).

A heartbeat is what would make a read deadline possible at all (`conn.go:32-37`). That is why
README's "wins on the spot" is corrected now and the deadline is a later decision.

**The README correction is done (PLAN release gate, step 5).** Quit, a closed window or a killed
game is a forfeit on the spot. A machine that loses power, sleeps or drops off the network sends
nothing, so the panel keeps its last position and nothing is decided until the OS gives up on the
connection, which can take many minutes. Escape on the waiting screen leaves at once.

**Found in the same pass, and fixed: the end of a run froze when the other machine had gone
silent.** `Race.Close` waited for the writer, and the writer could be stuck in a socket `Write`
that the peer had stopped reading. The game then froze on the last frame of the run, before the
waiting screen was up, so Escape could not reach it. It stayed frozen until the OS gave up on the
connection, which is about 15 minutes on Linux (`tcp_retries2` = 15). A dead peer sends no ACKs,
so a LAN socket's send buffer, tens of KB at ~1 KB/s of standings, fills about a minute after the
peer dies. Any run that ends later than that would freeze.
- Now `Close` waits at most `closeWait` (2 s) and returns `netplay.ErrStalled`. The writer is left
  to send the last standing and the goodbye if the socket ever frees.
- `finishRace` prints the stall, and the waiting screen and its Escape are then what end the race.
- A stall is not a result, and nothing is marked, for this entry's own reason.
- `TestCloseDoesNotWaitForAPeerThatStoppedReading` covers it, and it times out against the old
  `Close`.
- It was not reproduced end to end. On loopback the kernel gives the host's socket a 2.6 MB send
  buffer, which ~1 KB/s would take about 40 minutes to fill. Shrinking it, or dropping packets,
  needs root or a user namespace, and the development machine has neither.

**Also found, and only documented: a network split can give two different results.** When the
network fails between two machines that are both still running, each side's reader eventually
gets an error from the OS, and each then scores the other as gone, so each side wins its own
race. That is the result this entry's display-only rule exists to avoid, reached by the OS
timeout instead of a deadline. README now says both machines agree "as long as the connection
held". A heartbeat would let the waiting screen show it happening. Making the two sides agree
would need something to decide, and there is no referee.

**A separate finding filed here:** when this side quits (Quit, a forfeit), `finishRace` still waits
on `Settled`. The repro host printed "player 2 wins: by forfeit" and then "the other player never
finished, so there is no result".

### 4.38 The race takes each peer's word for its result — **the honour-system line and the state-machine fix before the next tag; re-simulation much later**

A peer can report any standing it likes. **What is done now:**
- README's race section and SECURITY.md (5.7) say results are on the honour system.
- `checkSlot` refuses:
  - any standing after an `Ended` one — a real hole: it reopens a settled race and turns the
    honest side's screen into NO RESULT;
  - `Rooms` going down, or above the house's room count;
  - more than 2 standings per `Frame`. It has to be at least 2, because every honest run ends with
    its final standing on the same frame as the last one in flight.
- **"Neither peer can choose the seed" is false** (`handshake.go:402`, CHANGELOG's Stage 3
  entry). 16 bits of the seed were set in about 15k tries, and all 31 bits of `RandSeed` are
  reachable in seconds. The gain is small (both players fly the same seed, and the slot does not
  affect `Winner`), so the claim is reworded, or commit-reveal goes into 4.31's `Meet` change and
  nowhere else. **Reworded, with 4.31** (`mixSeed`'s comment, and the CHANGELOG entry that
  repeated it). Commit-reveal was left out. It needs a second message each way before the first
  standing, and what it protects is small: a chosen seed is the same house for both players, and a
  peer that wants to cheat can send a winning standing far more easily.

**Dropped:** a wall-time rate check. `Winner` rewards *fewer* frames (`standing.go:328`), so a liar
sends a small `Frame` that such a check allows. It would also reject honest unpaced `-bench` races
(300 frames in 0.41 s, measured).

None of this stops one 28-byte message (Finished, plausible Rooms, large Score) from winning. Only
re-simulation closes that: each side sends its input log right after its own final standing and
before `Bye`, because `finishRace` sends `Bye` before `awaitSettled`. That waits on 4.39 and 4.31.
It proves "this engine produced this run", not "a person flew it", since a headless bot at about
90× real time passes.

### 4.39 A game somebody played cannot be replayed — **planned, the first item after the next tag**

4.2 built the bug-report format, and nothing records a real game in it. The proposal:
- Wrap `World.KeyPoll` in `cmd/glidergo/play.go` so that it appends a `replay.Hold` whenever the
  keys change.
- When a game ends, write a `replay.Script` from a `defer`, so a closed window or a panic still
  leaves it (a hang does not). It goes to `<datadir>/replays/last-<house>.replay`, and to
  `-record FILE` when that flag is given.

A scratch version reproduced three sessions exactly, one of them paced.

**The script carries:**
- house, the seed given to `NewWorld`, players, neighbours, the start clock;
- `sound` = whether the bank loaded (`a.eng != nil`). It must not come from `-sound` or
  `prefs.Sound`, because `openAudio` builds the engine on a Discard sink when no device opens, and
  `SetSoundOn(false)` keeps the trigger hot spots;
- `room N` and the directory lines when `-room`, `-houses`, `-art`, `-houseart`, `-sounds` or
  `-levels` were used;
- **`frames`: the last frame + 1 when the game ended on its own**, otherwise the last frame. The
  replay's `w.Frame >= s.Frames` stop otherwise cuts off the ending (measured: `rand` differed at
  frame 460, and +1 made all 461 match).

The grammar is extended once, together with 4.31's fields: `fix <name> on`, `househash`, and a
build line. `RunWatching` learns to apply `w.Fix`, which it never sets today; only `Player2GiveUp`
changes physics, but all four need lines. Resumed games are refused or marked.

**Decided by the owner, 2026-09-24: a build from before the new lines refuses them by name.** A
script that uses `fix`, `househash` or the build line fails in every release before 4.39's with
`unknown keyword`, which names the keyword and gives a later release as one reason for it. It
never runs the script without the line (`docs/PLAN.md`, "what a version number promises").
`TestALaterReleasesScriptIsRefusedByName` holds that for this build, and takes other lines once
these are the grammar's own. Recording is a feature, so 4.39 comes in a minor release, and the
new lines need no bump of anything that `cmd/glidergo/promises_test.go` pins.

**The acceptance test compares simulation fields frame by frame between the live loop and the
replay**, with sound off or `snd=` excluded. The live mixer is wall-clocked (4.11): under `-bench`,
24 of 25 events differed. It also checks that `glidertool replay -digest FILE` equals the digest
computed in-process from the same file (measured equal).

`bug_report.yml` asks for the file. 3.8, 4.44 and 4.38's re-simulation build on this.

### 4.40 A house with any tail but 0 or 2 bytes is refused, where the original and D3 accept it — **planned, Stage 2; before 4.41's packs and 4.43's importer**

The 1994 `ReadHouse` keeps any number of bytes after the last room, and
`docs/analysis/format-decisions.md` D3 decided the port would too. `house.Load` accepts only 0 or 2.

**The fix:**
- `Load` keeps any tail as `Slack`, and `Save` and the text format carry any length back
  byte-exact. That touches `binary.go:194-205` and `:347-350`, and `text.go:641`, which requires
  exactly 2 hex bytes.
- Lint and `house info` add a note for a tail other than 0 or 2, saying that 348 bytes or more is
  what the 1994 editor's `ValidateNumberOfRooms` (`HouseLegal.c:629-636`) would re-read as extra
  rooms. That changes the "34 checks" in PLAN Stage 2's
  `glidertool house lint` bullet.
- The three tests pinning the rejection change: `peek_test.go:99-110`, `text_test.go:349` and
  `shell_test.go:424-458`.
- The stale comments go: `binary.go:171-179`, `house.go:53-55` and `:106-107`, `peek.go:25-37`,
  `text.go:186`, PLAN `:157-158` and `:177-180`, and glidertool `house.go:219-221` ("a PowerPC
  save").
- The loader's error message cites §12.4, which is a hexdump. The rule is §2.5 (`:427-430`) and
  D3. `house-format.md`'s own "up to 2" at `:2778-2780` and `:4166-4168` is fixed in the same
  change.

**`Discover` is left alone.** Once the tail is accepted, every binary house `PeekFile` lists also
loads, so the listed-but-unopenable case goes away without the full-load scan the design rejects.
`TestHouseThatSniffsButWillNotLoad` is repurposed as "anything `PeekFile` accepts, `Load` accepts".
PLAN `:771`'s "a truncated house sniffs as one" is already false.

D5's "clamp `nRooms` to what the file holds, never reject a short file" is a related gap, out of
scope here.

### 4.41 The text format README shows off cannot be opened, and a downloaded house has nowhere to go — **planned; the suffix fix any time, the folder next, packs after 4.40**

In order of value:
1. **`.house.txt` opens and lists.** `Discover`'s whitelist (`library.go:135`: "", `.house`,
   `.glh`) skips it on purpose, and a text house passed directly fails with "need 3931008 bytes".
   `Discover`, `PeekFS`, `LoadFS` and `houseName` learn the compound suffix `.house.txt`. Bare
   `.txt` would put every README into Skipped, and trimming only the last extension would name the
   house "Open House.house".
2. **Size caps first** (4.36), and 4.30's targets.
3. **`<datadir>/houses/` as an additive source.** It is an explicit subdirectory, because
   `datadir.Dir("houses")` returns `GLIDERGO_DATA` verbatim with no subdirectory, and in a portable
   install that would be the scores directory. It is silent when missing (`walk()` turns a missing
   root into an on-screen error), and skipped in hermetic modes (2.53). A byte-identical copy of a
   built-in house is skipped, because two rows would share one save/score side-car keyed on
   name + timestamp (`store.go:401-407`). That is 4.14's hazard.
4. **Zip packs in the same layout**, with art resolved from the house's own pack first:
   `ArtRoots.Fork` is a global first-match by name, so a pack house named like a built-in one
   would otherwise get the wrong pictures.
5. **A pack manifest** is shown in the picker as the pack's own claim, and never in the credits,
   which 4.14's provenance rule forbids.

A single-set release already draws two sets, so the claim is narrower: an empty folder adds no
strip entry.

### 4.42 The game has no icon, no class and no version resource — **planned, after the next tag**

The original 1994 application icon (`icl8`/`ICN#` 128, with `ics8`/`ics#` 128 for 16×16) is
already in the embedded archive. Runtime Go decodes its 1.3 KB through `render.Palette` and the
mask, and the backend gets it through `platform.Config`, not by importing render.

- **X11:** `_NET_WM_ICON` passed to `XChangeProperty` as a C `unsigned long` array (8 bytes per
  pixel on LP64; passing `uint32` is the classic bug). Also `WM_CLASS` "glidergo"/"gliderGo", and
  `_NET_WM_PID` with `WM_CLIENT_MACHINE` via `XSetWMProperties`.
- **Win32:** a committed, deterministic, icon-only `rsrc_windows_{amd64,arm64}.syso` in
  `cmd/glidergo`, with `RT_ICON`/`RT_GROUP_ICON` at 16/32/48/256. 48 comes from 16×3 and 256 from
  32×8, nearest-neighbour, so this is not "upscaled art". A `-check` mode guards it the way
  `packassets` is guarded. `LoadImageW(instance, 1, IMAGE_ICON, …)` goes into
  `wndClassExW.icon`/`iconSm`. The release job regenerates the `.syso` with a `VERSIONINFO`,
  because the version is known only at build time. It names gliderGo, its version and licence,
  and credits the original. `CompanyName` is not "Casady & Greene".

No manifest: DPI is set at runtime (`win32.go:404`) and nothing uses Common Controls. A `.desktop`
file is optional, because nothing installs it and `Exec` needs an absolute path. If a pure-Go X11
client lands (5.10), the X11 half goes into it. The macOS `.icns` comes from the same decoder in
Stage 6.

### 4.43 Houses that are not the project's: the rules for contributing one, and an importer for the 1990s ones — **the rules DONE (PLAN release gate step 5), and the owner's question answered, 2026-09-24; the importer later**

**The rules, in CONTRIBUTING's Houses and Licence sections (now).**
- The project never bundles third-party houses.
- A house contributed to `levels/` must be the contributor's own work, and its author gets a
  credits line.
- `credits.txt`'s "None of that work is ours" is corrected, because two houses now are.

A public repository invites exactly the contribution that would undo 1.2.

**Done.**
- CONTRIBUTING's Houses section ends with "Whose house it is", which gives the three rules. A
  third-party house is played with `-house PATH` or `-levels DIR`, and is never bundled. The
  Licence section says a house is covered by the same grant as the code.
- `credits.txt`'s `[this port]` gains the row `gliderGo | Open House, Boarding House`. The note now
  says the 22 houses above are not the port's work and the two on that row are. The credits screen
  was looked at after the change: the section still fits, with nothing cut. Its fidelity hash was
  re-recorded (`credits 963bfb897ba02c1c`), and the other six screens did not move.
- `credits_test.go` used to skip the port's names from a list ("gliderGo", "Brendan Ta", and a
  "Nobody but the people above" that is no longer in the file). It now skips the `[this port]`
  section whole. So a contributor's row needs no edit to the test, and a port name written into a
  1994 section is caught, which the list let through. A new test,
  `TestEveryHouseThisPortShipsIsCredited`, fails for any house in `levels/` that has no row under
  `[this port]`. That is what holds a patch to the second rule. Two mutations were run: dropping
  Boarding House from the row fails the new test, and moving a port name into `[the houses]` fails
  the old one.

**Decided by the owner, 2026-09-24: a reworked house is the contributor's own.** The Houses
section's first command is `house dump` of Slumberland, because that is how the format is learnt,
and the rules had not said whether a house built by editing a dump of one of the 22 counts as its
author's own work. It does. CONTRIBUTING's first rule now says so, drops the issue it asked for
first, and asks the pull request to name the house the new one started from, so that the history
records it. The decision covers the 22 and nothing else: a house somebody made in the 1990s and
published on its own is still never bundled, reworked or not, because its author granted this
project nothing. What is left of an original in a reworked house stands where the 22 themselves
do, on 1.2's route (a).

**`glidertool house import` (after Stage 3, before the Stage 5 editor, which reuses it).**
- It decodes BinHex 4.0, MacBinary and AppleDouble.
- It rejects StuffIt by its magic with "unstuff it first". Most 1990s houses travelled as
  `.sit.hqx`, and today they fail with "file truncated". An optional first step is recognising
  these magics in the loader and naming the format.
- It converts version-1 houses (`peek.go` refuses anything but 0x0200, and `house-format.md` §3.2
  says the conversion is two lines) or refuses them by name. Glider 4 non-PRO houses are out of
  scope.
- It decodes `'Date'` resources the same way as PICT, or at least reports them. The docs disagree
  about them (`rendering.md` §4.4 against `graphics-assets.md` OQ1).
- The embedded filename goes through `datadir.FileName` and is never used as a path, which is
  SECURITY.md's own named risk. The importer brings its own fuzz targets.
- It writes `X.house` plus `houseart/X/pict` (and `bnds`), which `-levels DIR` already plays.
- Aerofoil's converted layout (`.gpd`/`.gpa`/`.gpf`) is not on this list. 5.13 refuses it by name
  first, and adds it here only if one checked sample shows it is cheap.

**Acceptance:** pixel-exact against the 919 extracted PNGs from the 22 vendored `.binhex` files.
MacBinary and AppleDouble have no samples in the tree and need made-up test files. The sound half
waits for 4.29 and a Go `'snd '` decoder. Until then the importer says how many sounds it dropped.
`house-format.md` §13.3 already recommends this import. What was missing was the plan step and the
rules. Depends hard on 4.40.

### 4.44 The race needs a second machine, and a recording could be the other player — **note; after 4.39 and 3.8**

The player races a recorded run offline on the existing panel: their own best, or a friend's file.

**How it runs:**
- The recording is re-simulated headlessly with the same `World` → `Standing` mapping the race
  uses, storing only the frames where the standing changes. Counting distinct `Sample.Room` values
  reads one room ahead, because a room is marked visited on *leaving* (`transit.go:195-217`).
- The recording is refused unless it ends in Finished or Died and reproduces the result line it
  carries. A Quit scores as a forfeit, and running out of frames never settles.
- The live `World` is seeded from it, and it is settled with `netplay.Winner`. The race treats it
  as a race: rooms before frames. It is not a speedrun, so "race your best" picks the run `Winner`
  ranks highest.
- A small interface (`Opponent`, `Report`, `Close`, `Settled`, `Result`, `Match`, `Err`) replaces
  `*netplay.Race` across `wrapPresentRace`, `finishRace`, `awaitSettled`, `showRaceResult` and
  `theirsAsScored`.

**A friend's file is untrusted.** The loader accepts only the recorder's own keywords: house by name
plus `househash`, seed, neighbours, sound, music, players 1, clock, capped frames, and `at` lines.
It takes no paths, no `room`/`where`/`gliders`/`stars`, and it stops re-simulating at game over.
Verification proves the engine can produce the run, not that a person flew it.

Entry points: a `-challenge FILE` flag first (`-ghost` would promise a sprite this rejects), then a
Race-screen row listing recordings from the data directory. That changes the race screen's hash.
M, given 4.39.

### 4.45 The register's statuses are prose, and some point at stages that have closed — **note; after the next tag**

A hand re-triage comes first. Close 2.8, 2.12 and 2.34 (both halves have Done rows), and the
table-test clause of 2.18. Narrow 2.13 to per-sound loading plus separate volumes, and restate 4.8.
Retarget 2.2, 2.4, 2.10, 2.11, 2.17, 2.29, 2.37, 2.57, 2.58, 3.3 and 4.3. Give the open halves that
name no stage (2.1, 2.14, 2.18, 2.22, 2.23, 2.49, 2.71, 4.28, 4.29, 5.5) a stage or "decision
needed". Several of these hide player-visible work that has quietly left the schedule (2.4's wipes,
3.3).

**Then, optionally:**
- The rule at the top of this file admits **note** as a fourth form, and "decision taken" is
  allowed. 34 of the 147 statuses start with "note". A test of the three forms as written would fail
  64 headings the day it landed, and 28 even with both of those admitted. Each heading gets a fixed
  leading status token, because statuses are compound.
- The Done table's "2.N" labels are commit labels, not Stage 2 sub-stages. 2.5 and 2.6 come after
  Stage 3's netplay commits, and "2.0" also collides with the Glider PRO house format. A
  label-to-commit map replaces "this stage" mechanically; digging 126 cells out of history is the
  costliest and least valuable part.
- `internal/citations` gains tests: every open clause names an open stage, and every DONE heading
  has a Done row. They share the stage-status parser that PLAN §4's release policy proposes for its
  status table.
- A generated `OPEN.md` is the lowest-value extra.

### 4.46 The citations test cannot see most of the line numbers this repository cites in itself — **note; after the next tag**

`TestEveryReferenceToOurOwnTreeResolves` matches a path only under `cmd/`, `internal/`, `docs/`,
`tools/`, `scripts/`, `assets/` and `.github/`, and checks a line number only against the file's
length. So these are never checked at all:

- a top-level file: `README.md:N`, `CHANGELOG.md:N`, `RELEASING.md:N`, `CONTRIBUTING.md:N`,
  `Makefile:N`;
- a file named by its base name alone, `release.yml:N` or `handshake.go:N`, and the `:N` shorthand
  that follows a file already named;
- the end of a range, the `M` in `:N-M`.

And a number that is still inside the file but no longer at the line it meant passes. Checking the
`v0.2.0` release turned up ten stale ones without looking for them: `README.md:338` for the house
dump example, `README.md:454-456` and `:741-743` for the Go licence sentence, `Makefile:418-436`,
`Makefile:557` and `CONTRIBUTING.md:46-49` for the race detector, `handshake.go:289` for the seed,
and three CHANGELOG line numbers, which every new entry above them moves. All ten are repointed, and
the CHANGELOG ones now name their entry instead, which does not move.

They are not all of it. A spot check of the fourteen bare `release.yml:N` citations found nearly
every one pointing at a comment, a blank line or another step, because the file has grown by
hundreds of lines since they were written. The eight `ci.yml:N` ones are the same. Those are not
repointed yet, and are the first work here.

The cheap half is to widen `ownPath` to the top-level files and to base names that are unique in the
tree, and to check the end of a range. Drift inside a file's length needs the citation to carry
something to compare, such as a few words of the line, and is the costlier half.

### 4.47 A bench row at a scale its monitor cannot hold says nothing — **note; after the next tag**

`windowScale` in `cmd/glidergo/scale.go` honours a typed `-scale` that does not fit, and warns that
it does not. The warning needs the monitor's size, and `openWindow` in `cmd/glidergo/play.go` asks
for it only outside the four measurement modes, because a measurement must not depend on the
machine (2.53). So `-frames` and `-bench`, which are what `make bench` and the Windows bench rows
run, never warn: a 4× row on a monitor too small for it measures a window partly off the screen,
and says nothing. `docs/windows-first-run.md` check 3 claimed the warning until 2026-09-24. It now
says to check by eye, and `windowScale`'s comment no longer says `make bench` is warned.

The fix is to ask the monitor in a measurement too, when `-scale` was typed, and only to warn,
never to change the scale, so that the run still depends on nothing but its invocation. The null
backend answers with a 640×480 "no screen", which has to be told apart, or every null run above 1×
would warn falsely. That is a change to the game with no gain in play, so it waits until after
`v0.2.0`.

### 4.48 A peer that dies mid-race can take its last standing with it, and the test that said it could not most likely failed CI — **the test DONE, 2026-09-28; the race's half a note, after the next tag**

`go test ./...` failed in CI's `native` job on 2026-09-28, which is the Windows and macOS legs. The
run's annotations name only the step, and its log has not been read here. The likeliest cause is
`TestLoopbackRaceGuestWhoLeavesMidRaceForfeits`, 4.34's guest that hangs up mid-race, which ran on
GitHub for the first time in that run. It could fail two ways, and `make check` on Linux had shown
neither:

- **The host's first report could be its last.** The host is a bench run, which is unpaced, and an
  idle glider in Grand Prix dies at frame 351, a few milliseconds in. A pending standing is
  replaced rather than queued behind (`netplay.Race.Report`). So on a busy machine the first
  standing to go out can already say "out of gliders", and the test stopped there. At
  `GOMAXPROCS=1` it failed 28 of 30 runs.
- **The host could lose the guest's one report.** The guest closed with the host's standings
  unread, which is a reset. Windows discards data that has arrived unread when a reset comes in,
  so the host scored the guest on nothing instead of on its one room. On Windows Server 2025 that
  happened in 12 of 30 runs.

**The test, DONE.** `heldWin` stops the host's run at its first frame's present, until the guest
has gone, so the guest always leaves a race the host is still flying. The host then flies on to the
end, as before. The guest's first read has a deadline, so a held host that says nothing fails the
test instead of hanging it. The test is two subtests now, one for each way the host can hear the
guest go:

- `closing`: the guest shuts its sending side and goes on reading, which no dying process does. The
  host hears the report and then the end of the stream, and nothing it writes is refused. It must
  score exactly that standing, and report no error.
- `reset`: `SetLinger(0)` resets the connection whether or not anything is unread. A dying process
  comes to a reset either way: at its close if anything is unread, and otherwise at the host's next
  write. The result must be the same forfeit, with the reset reported. The guest may be scored on
  nothing.

Both passed 50 runs at `GOMAXPROCS=1`, 30 at each of `-cpu=1,2,4,8`, and 20 at `-cpu=1,4` under
`-race`. With a busy loop on every CPU they passed 100 runs at each of `GOMAXPROCS=1` and `2`.

**What the reset subtest found is the race's, not the test's.** With a busy loop on every CPU, the
host scored the guest on nothing on Linux too: 95 of 100 runs at `GOMAXPROCS=1`, 27 of 100 at
`GOMAXPROCS=2`. Each time, the host's writer met the reset first, and `markBroken` settled the race.
`finishRace` took its result at once, while the guest's report was still unread in the socket. So
on any system, a peer that dies can be scored on a standing older than its last. A plain close
with nothing unread does the same, because the host's next write draws the reset. With the guest's
report sent just before a plain close, 19 of 20 runs at `GOMAXPROCS=1` scored it on nothing, and
none of 20 at `GOMAXPROCS=4`, and all 40 reported a broken pipe.

`read`'s comment said a dying peer arrives as a reset but not what the reset can cost. `Conn`'s said
it arrives as an EOF, and so did `Abandoned`'s. `markBroken`'s said waiting for the reader would
only delay a result already decided. All four are corrected, and `read`'s also says that on Linux a
reset the writer met first reaches the reader as a plain end of stream.

The score it costs is a few frames of progress. For a peer that left mid-flight the result does not
change: it forfeits whichever of its standings it is scored on. It would change the result only if
the lost standing was the one that ended the peer's run. A peer that reported `Died` or `Finished`
and then died would be scored through `Abandoned` from the standing before, as a forfeit rather
than on its run. For that, the final standing and the reset have to arrive together. On Windows,
both have to arrive before this side reads the socket. Elsewhere, this side's own run also has to
end in that moment. A final standing goes out before the waiting screen is drawn, and a player
takes longer than that to close a window. No race has shown it.

The fix, after the next tag, is in two halves:

- the writer's failure is recorded at once, and only its settle waits a moment for the reader. On
  a reset the reader returns at once, and Linux and macOS hand it what arrived before the reset. On
  Linux a writer that met the reset first has taken the error with it, so the reader then sees a
  plain end of stream, which `read` counts as no error. The writer's error is the one to keep;
- nothing on the receiving side can stop Windows discarding data. The side that is leaving on
  purpose can close gracefully instead: shut its sending side, go on reading for a moment, then
  close. Its last standing and the end of its stream then reach the other side before any reset
  can, so a reset that follows costs nothing. A peer still flying draws that reset with its next
  write and is refused the one after, which after a goodbye is not an error (4.49). A killed
  process gets no say.

### 4.49 A player whose opponent said goodbye was told it had not — **DONE, 2026-09-28**

Found by the review of 4.48's test. The first to finish a race sends its last standing and a
goodbye, then waits for the other run. If its player presses Esc and then leaves the result screen,
its socket closes while the other player is still flying. That side's next report draws a reset,
and the one after it is refused. `Race.mark` recorded the refusal as the race's error, because the
goodbye before it had recorded none. So the other player's result screen said "the other player's
game ended without saying goodbye" (`endWords`), and stdout said the connection had failed.
`Race.Err`'s comment says a peer that said goodbye is not an error. The result and the standing
were right, and only the words were wrong.

**DONE.** A goodbye is kept as the reason the peer went, and a write refused after it is not
recorded (`Race.bye`, `markGone`). Sending still stops.
`TestRaceDoesNotBlameAPeerThatSaidGoodbyeForHangingUp` failed before the change, with `write |1:
broken pipe`, and passes after it, 20 runs under `-race`. Its pipe refuses the first write where
TCP would take it and refuse the next, which makes no difference to what it checks.

A peer that goes without a goodbye is unchanged. Its end of stream is followed by the same refused
write, which is still reported, and for a killed process the words are true. `Race.Err`'s comment
now says so.

The goodbye has to be read before the refusal, and from this game it is. A side that says goodbye
while the other is still flying waits for that run, or for its player, before its socket closes. A
peer that closed in the same moment as its goodbye could have one of this side's writes refused
before the goodbye is read, and that refusal would still be recorded. That is 4.48's writer-first
case, and the second half of its fix covers it.

---

## 5. Getting off this machine: the build, the package and the public path

Everything above is about the game. This section is about the fact that the game is being
written on one airgapped host and is meant to end up on GitHub, and that those two things have
different failure modes. Added when the public build path was put in beside the private one.

### 5.1 The public build path cannot be tested from the machine that wrote it — **note; check 1 retired, check 3 answered by the Actions log, check 2 still open against go.dev itself**

`scripts/bootstrap-dev-env.sh --source public` and `.github/workflows/ci.yml` are the two
pieces of this repository that have **never run**. They cannot: this host cannot resolve
`go.dev`, `archive.ubuntu.com` or `github.com`, and a `curl` to any of them fails at connect.
What *is* verified is everything they are made of — every command the workflow invokes was run
here (`make doctor`, `make check` on a no-asset tree with `DISPLAY` unset, `make cross`,
`fmt-check` against a deliberately unformatted file), the script's own logic was reviewed with
`--dry-run` for all three sources, and the `.deb` path was exercised end to end against this
host's own Ubuntu mirror, which is the same code with a different base URL.

The specific things a connected host should check first, in the order they are likely to be
wrong:

1. The action versions. `actions/checkout@v4`, `actions/setup-go@v5`, `actions/cache@v4` and
   `actions/upload-artifact@v4` were the current majors as of the knowledge this was written
   with, and that is the weakest claim in the file.
2. `install_go_public`'s parse of `https://go.dev/dl/?mode=json&include=all`. The shape of that
   index is not documented as an API. The code picks the newest stable non-rc release at or
   above `go.mod`'s floor and then sha256-verifies the tarball against the digest in the same
   index — which is **integrity, not authenticity**: an index served by an attacker would agree
   with its own tarball. `GLIDERGO_GO_SHA256` exists so a digest can be pinned out of band, and
   `GLIDERGO_GO_TARBALL` skips the network entirely.
3. `xvfb-run -a -s '-screen 0 640x480x24' make bench`. The depth argument is not decoration —
   `internal/platform/x11` refuses anything but 24 or 32, Xvfb's historical default is 8, and
   the failure is a hard error rather than a skip. Verified here against `Xephyr` at both
   depths (686 fps at 24, refused at 8) because no `xvfb` is installed on this host.

None of this blocks Stage 1. It blocks the first push to a public remote, which is where it
will announce itself loudly and cheaply.

**Amended after `v0.1.0`, `v0.1.1` and `v0.1.2`.** Three tags have been built and published by
`release.yml`, so this entry is partly answered and partly not. From this host it is **unknown**
whether any of them needed a correction.

1. The action versions ran. `actions/cache@v4` is listed and nothing uses it, so it comes off the
   list. Both workflows now pin `actions/checkout@v5`, `actions/setup-go@v6` and
   `actions/upload-artifact@v6`, and `release.yml` adds `actions/download-artifact@v7`. The first
   paragraph's "never run" now holds for the bootstrap script only: `ci.yml` has run on every
   push since. Both artifact actions were `@v5` until 2026-09-28, when a CI run's annotations said
   `upload-artifact@v5` still targets Node 20. `download-artifact` is run only by a tag, and the
   `v0.1.2` release run's annotations, which would say whether `@v5` does too, have not been read
   here. It moved to `@v7` anyway. By the two actions' release notes, which cannot be fetched from
   this host, the first majors to run on Node 24 by default are `upload-artifact@v6` and
   `download-artifact@v7`. That was not checked against GitHub before this commit.
2. **Still open, against go.dev itself.** No workflow runs `scripts/bootstrap-dev-env.sh`,
   because CI gets Go from `setup-go`, and `--dry-run` cannot close this, because it never touches
   the network. On 2026-09-24 the whole `--source public` path ran end to end for the first time,
   against a local HTTP server standing in for go.dev. Its index held an rc, an older release and
   another platform's archive around the one to pick, and the tarball was a real Go. That run
   found that `public_go_pick` had never been able to read any index: the heredoc holding its
   python program was python's stdin, so the index curl piped in was thrown away and `json.load`
   read nothing. The program now goes in with `python3 -c`, and the same run then picked the
   right archive, verified its sha256, installed it and wrote `scripts/env.sh`. What is left is
   whether go.dev's real index has the shape the code reads, which needs someone on a connected
   host to run `scripts/bootstrap-dev-env.sh --source public`.
3. Answered by the Actions log, in the `check` job's "On-screen bench under Xvfb" step and in
   `release.yml`'s `verify` job. The line is pasted here when someone reads it. The screen in
   both is now `2600x1980x24` rather than `640x480x24`, so that a 4× window fits (2.76). The
   depth argument still matters for the reason given above.

### 5.2 `tools/extract_all.py` writes its output tree in place, and something has already been corrupted by it — **found, planned, and DONE, 2.4 — all four consequences, and the fix turned out to have a fifth property nobody asked for**

The extractor writes straight into `assets/extracted/`: no temp directory, no rename, no lock.
So for the ~70 seconds it runs, that tree is a mixture of the old extraction and the new one, and
anything reading it sees half-written files. This is not theoretical — a `go test ./...` running
concurrently with a `make assets` failed decoding a golden PNG with "unexpected EOF", which took
a while to recognise as a build-system problem rather than a renderer one.

Four consequences, and only the first is currently handled:

- **CI.** `.github/workflows/ci.yml` runs `make assets-check` as its own job, isolated from
  anything that reads the tree, and a comment there says why. That is a workaround in the caller,
  not a fix.
- **The tree is committed now (1.2), so a half-written extraction dirties the working copy** and
  the damage shows up as a `git diff` over binary files. That is at least visible, and
  `git checkout -- assets/extracted` undoes it, which is a better recovery than the old one
  (re-extract and hope). It also raises the stakes on the temp-and-rename fix below: a
  cancelled `make assets` now edits tracked files.
- **A cancelled extraction leaves a tree that looks complete.** `make check`'s guards test for
  contents (`[ -s art/manifest.json ]`, `ls houses/*.house`), which catches an *empty* tree but
  not a half-written one. The manifest is written last, which helps by accident rather than by
  design.
- **Two `make assets` at once** — a developer in one terminal, an editor's build task in
  another — interleave silently.
- **New, and this one is a mitigation rather than a consequence.** `make assets` now packs
  `assets/extracted.zip` from the tree as its last step (5.3), and `go test ./assets` compares the
  two file by file. A half-written or half-repacked tree is therefore a *test failure* naming the
  file whose length disagrees, which is the first tripwire this entry has ever had that fires on
  contents rather than on emptiness.

The fix is the ordinary one: extract into `assets/.extracted.tmp-$$`, `os.replace` the finished
tree into place, and take a lock file for the duration so the second run waits or refuses. It is
maybe thirty lines in `extract_all.py` and it wants doing before a release pipeline exists, since
a pipeline is precisely a place where two jobs share a checkout.

*(5.2, **DONE**, 2.4. Done as prescribed, in the shape prescribed, and it was 150 lines rather
than thirty. The staging tree, the rename and the lock are the thirty; the rest is what having a
staging tree turns out to imply.*

*`tools/extract_all.py` now stages every run in `assets/.extracted.tmp-<pid>`, publishes it with
two renames, and holds `assets/.extracted.lock` for the duration. Both names are dotted siblings
of the output tree rather than children of it: a child would be published along with the tree, and
an undotted sibling would show up in `git status` beside a committed tree as a build artefact
nobody can account for. `.gitignore` carries the one pattern, with the reason.*

***The fifth property, which the entry did not ask for and is the best thing about the fix.***
*`EXPECT` — the eleven counts this script checks against the analysis documents — is now checked
**before** the rename rather than after the write. The old order meant a regressed extractor
replaced 46 MB of good assets with 46 MB of wrong ones and then told you the counts were wrong;
recovery was `git checkout -- assets/extracted`, which you had to know to reach for. The new order
means a failed count publishes nothing, and the run says so in as many words: `NOT PUBLISHED.
assets/extracted is as it was`. It also **keeps** its staging tree in that one case, and names the
path, because a complete extraction with wrong numbers is the thing you want to look at. Every
other exit — interrupt, exception, success — removes it, since an incomplete 46 MB tree under a
name nobody recognises is a smaller copy of the trap this whole entry is about.*

***What staging broke, and the part worth reading.*** *`--only art,sound` looked like a detail and
was the hard half. Two ways it fails against a fresh staging tree: the steps are not independent
(`art` reads `res/`, `houseart` reads `res/` and `houses/`, and both exit saying so when it is
missing), and a partial run still has to publish a **whole** tree or the rename would delete the
five buckets it was not asked to rebuild. Both are fixed by seeding: the buckets a run is not
rebuilding are hardlinked from the published tree into the staging tree first. Hardlinks and not
copies because 46 MB is mostly PNGs and nothing writes to a bucket it is not rebuilding — the
links are read, then publishing unlinks the old tree, which drops a reference and leaves the bytes
owned by the new one. A filesystem without hardlinks falls back to a real copy. The manifest merge
that keeps a partial run from dropping the other buckets' records now reads the **published**
manifest and not the staged one, because seeding carries buckets across and not the record of them.*

***The lock refuses rather than waits***, *which is the one place this departs from the entry's
"waits or refuses". Nothing in here reads a clock, so the second run would produce the same bytes
as the first: waiting for it buys a caller nothing that reading one line and re-running does not.
It is an OS advisory lock — `fcntl.flock` where there is an `fcntl`, `msvcrt.locking` where there
is not — rather than an `O_EXCL` pid file, because the kernel releases those when the holder dies
and CI cancels a job with a signal the process never sees. A lock that had to be cleaned up by its
holder would be stale half the time. **Both imports are guarded, and that is not defensive
habit:** `.github/workflows/ci.yml` extracts to a temp tree on the **Windows** runner as a
best-effort step, so an unguarded `import fcntl` at the top of this file would have turned a
passing job into an `ImportError`.*

***What it is not.*** *Not literally atomic. The old tree is renamed aside before the new one is
renamed in, so there is a two-syscall window in which `assets/extracted` does not exist, and a
reader unlucky enough to land in it fails saying which path is missing. That is chosen and not
overlooked: there is no portable directory swap (Linux's `RENAME_EXCHANGE` is not in the standard
library), and "no such directory" is a failure that names itself, where a truncated PNG is the
failure this entry opens with. If the second rename fails, the first is undone and the old tree is
put back.*

***Verified, four ways, all four observed rather than reasoned about.*** *A full `make assets`
republished all 1,877 tracked files with **no `git diff`** and the leftover state in `assets/` is
one ignored lock file, so the rename path is byte-for-byte what writing in place was; `make
assets-check` still reports the committed tree is what the extractor produces, with its lock
landing outside the compared tree. `--only houses` against a tree holding only `movie/` published
**both** buckets, with `movie/`'s link count back to 1 afterwards — the seeding and the unlink are
both doing what they claim. A `SIGINT` 1.5 s into `--only art` left exit 130, no staging
directory, and a published tree with no `art/` in it. And forcing a count to disagree left the
staging tree named on stdout, the published tree byte-identical, and exit 1.*

***The four consequences the entry lists, one by one.*** *The CI workaround stays, but its comment
no longer says "until 5.2 is actually fixed" — the `assets` job is separate now because it costs
python3 and a minute of CPU the Go jobs have no reason to wait for, which is a scheduling choice
and not a hazard. "A cancelled `make assets` now edits tracked files" is gone outright: a cancelled
run edits nothing. The guards that could catch an empty tree but not a half-written one
(`[ -s art/manifest.json ]`, `ls houses/*.house`) are no longer load-bearing, because a
half-written tree cannot be published for them to inspect. Two `make assets` at once refuse. And
5.3's `assets/extracted.zip` tripwire — `go test ./assets` comparing archive to tree file by file
— is still the backstop for everything downstream of the rename.*

***One thing found on the way, of 4.4's kind.*** *This file's own docstring said
"assets/extracted/ is gitignored on purpose — it is derived data, and the derivation is this
file." It has been committed since 1.2, 1,877 files of it, and `.gitignore` says so at length. The
docstring now explains why derived data is committed here and how staging is part of that bargain:
a deterministic run shows up as no diff at all, which is what `make assets-check` turns into a
test.)*

### 5.3 An installed copy looked for its assets beside the binary — **DONE, embedded; the assets are inside every executable**

The entry used to say every asset path resolved against the working directory (`assets/extracted/art`,
`assets/extracted/houses`, and the two others), which was right for a development tree and wrong
for anything a player downloads: a `glidergo` copied to `~/bin` found nothing and drew the
first-run screen instead. It recommended a search path — the executable's directory, then
`/usr/share/glidergo`, then `$XDG_DATA_HOME/glidergo` — and kept a 3 MB binary.

**The decision taken was the other one: `//go:embed`, no search path, nothing beside the binary.**
The reason is the audience. A search path is the right answer for something a distribution
packages, and gliderGo is a game somebody downloads from a Releases page, unzips into whatever
directory their browser chose, and double-clicks. Every step of a search path is a step that can
find the wrong copy or none, and "it must be run from the directory you unpacked" is an
instruction a player should never have to read.

**What it looks like.** `assets/extracted.zip` (11.3 MB, 1,877 files) is committed beside the tree
and compiled into every executable by `assets/assets.go`. Both binaries carry it — `glidertool`
renders houses and replays recordings, so a tool with no assets in an archive with no assets is
useless. Four things made it fall out cleanly rather than needing a plumbing rewrite:

- `*zip.Reader` is an `fs.FS`, so the loaders needed one change each: take an `fs.FS` from the
  caller instead of opening paths. Nothing under `internal/` knows the built-in copy exists, which
  is also why `go test ./internal/...` links none of these 11 MB.
- `internal/assetfs` holds the resolution rule in one place: an empty flag means the copy built
  into this binary; a named directory *replaces* that root. `-version` prints which of the two
  each root came from, so a bug report says it.
- `internal/assetpack` holds the packing and the comparison, and `tools/packassets` is the only
  thing that runs it. The packer must not import `assets` — that package embeds the archive it is
  about to write — which is the whole reason the logic is not simply in `assets`.
- The pack is deterministic: names sorted, every timestamp fixed at 1994-10-01 UTC, no mode bits,
  no directory entries. `make assets-zip` twice over produces the same bytes, so the committed
  archive is not a source of spurious diffs.

**The trap that decided the shape, and it is worth knowing.** `go:embed` accepts only names that
are valid module file paths, which rules out an apostrophe — and three of the shipped houses have
one: `Castle o' the Air`, `Nemo's Market`, `Rainbow's End`. A pattern that *names* such a file
fails the build, which is survivable. A pattern that names its **directory** skips it in silence:
120 files and 350,674 bytes, three houses and three houses' worth of custom art, gone from the
game with nothing anywhere to say so. A zip has no such rule, and the 1994 names survive
byte for byte inside it. `assets.TestApostropheNamesSurvived` is the tripwire, and it deliberately
does not skip when the tree is absent, because its subject is the bytes in the binary.

**What it costs, stated plainly.** A binary went from about 3.6 MB to 14.9 MB. A release archive
holds two of them, so the same 11.3 MB ships twice in it and the archives went from about 15 MB to
about 26 MB each — already-compressed data, so `tar -z` and `zip` cannot help. The repository now
carries the assets twice as well (the 16.3 MB tree plus the 11.3 MB archive of it), which took the
tracked total from 68.6 MB to 79.9 MB. Three ways to spend that back, none of them urgent, all of
them deferred on purpose:

- **Ship one binary.** Folding `glidertool` into `glidergo` as a subcommand, or leaving it out of
  release archives entirely, halves every download at a stroke. It is the cheapest of the three
  and the one that changes what a release contains, so it wants deciding with 5.4 rather than here.
- **Keep only the archive in git** and unpack it on demand. `make assets-check`, the fidelity
  corpus and the house round-trip name files by path, so this means teaching those three to read
  the archive or to unpack into a scratch tree. Saves 11.3 MB of repository and buys a `git diff`
  over `assets/extracted` that nobody can read.
- **Union the roots** instead of replacing them, so `-art some/dir` could override one PNG and
  inherit the rest from the built-in copy. Today it replaces, which is the honest simple rule and
  the one whose failure mode is obvious. The union is what a modding story would want; there is no
  modding story yet.

**What did not change.** The four flags still take a directory, and pointing one at nothing still
draws the first-run screen (`make headless` renders it via `-art /nonexistent`, which is now the
only way to reach it). `make assets` still re-derives the tree from `GliderPRO/` and now repacks
the archive after it. And the Makefile grew an `embedded` guard: every build target refuses to
build without `assets/extracted.zip` rather than producing an executable that comes up empty.

### 5.4 There is no release pipeline, and the CI that exists deliberately does not publish — **DONE as `release.yml`, and it has run: `v0.1.0`, `v0.1.1` and `v0.1.2` are published; the amendment DONE in PLAN step 5, but for what needs a connected machine**

`.github/workflows/release.yml` triggers on `v*` tags, and it does every item this entry used to
list as future work: `make cross` plus the host's cgo build, six archives of two binaries each
with their assets inside them (5.3), a `SHA256SUMS` manifest,
`README.md`, `CHANGELOG.md`, `LICENSE` and upstream's own README and licence in every one, and a
`gh release create` that runs only on a tag push. `VERSION` is passed explicitly rather than left
to `git describe`, so a build is stamped with the tag it came from instead of a bare short hash.

Three things about it are worth knowing before trusting it.

**It runs its own tests.** `ci.yml` triggers on a push to `main` and on `pull_request`, and a tag
push is neither, so tagging fires no CI at all. Without the `verify` job at the top of
`release.yml` a release would ship binaries that no suite had ever seen.

**It has never executed.** Same reason as `ci.yml`: the machine it was written on cannot reach
github.com. What is verified is everything below the GitHub line — the packaging step was
extracted from the YAML and run verbatim against a real `make cross` tree, twice, once before the
assets moved into the binaries and once after; all six archives build and pass `sha256sum -c`, and
the `linux-amd64` one was unpacked and played with sound — before 5.3 from its own root, and after
it from a directory holding the binary and nothing else, 300 on-screen frames of Slumberland out of
the copy inside itself. What is unverified is action versions, runner package names and
`gh`'s behaviour. The `workflow_dispatch` entry exists to rehearse the build half without
publishing.

**Five of the six archives cannot draw** (§5.5), which is a packaging problem as much as a
backend one. They are named `-headless`, and their `HOW-TO-RUN.txt` does not tell the reader to
run the bare binary: the null backend delivers no quit event and the shell loop has no exit
condition without `-frames`, so that would spin on an invisible title screen until Ctrl-C. It
gives two commands that terminate and produce something instead, and both were run from an
unpacked archive before being written down.

§5.3 used to be the reason an archive had to carry an asset tree and be run from its own root; it
is closed, so an archive is now two binaries and five documents, and `HOW-TO-RUN.txt` no longer
tells anybody where to stand. Two things a release still cannot fix: §1.2's reading of the licence
the 1994 content is shipped under, and the fact that the same 11.3 MB rides in both binaries of
every archive (§5.3's first deferred item).

**One thing to do at the moment the first tag is pushed, and not before.** `README.md`'s opening
paragraph says "there is nothing to download and no copy of the old game to find. Clone it and `make
run`", and the issue templates ask how the reporter's copy was built. Both become incomplete the
instant a release page has something on it, and both are the natural home for
`https://github.com/bwenstar/gliderGo/releases`. That URL used to be a constant,
`project.Releases`, deleted in 2.4 because it had no reader and a page with nothing on it is not
something to point a player at (4.13). Putting it back is one line in `internal/project/project.go`,
and `TestEveryExportedConstantHereIsReadBySomething` will hold it to having a caller this time.

**And one thing to do before the first tag is pushed.** The release notes now walk a player through
SmartScreen, Mark of the Web and Gatekeeper quarantine, which 4.13 called the single most likely
reason a first-time player never reaches the title screen — and they do it from documentation rather
than from anyone having watched it happen. The Windows half needs no network to check: write a
`Zone.Identifier` stream onto the `.exe` by hand and double-click it in Explorer.
`docs/windows-first-run.md`, "Rehearsing what a download adds, with no download", has the two
commands. It is the only unverified claim in `release.yml` that github.com is not required to settle,
which makes it the cheapest one on the list and the last one that has an excuse.

**Amended: it has executed, three times.** "It has never executed" and `release.yml:3-9`'s "it has
never run … Expect the first tag to need a correction" are history now. They are rewritten to say
what is known and what is not: whether any of the three needed a correction is not recorded here.
"**Five** of the six archives cannot draw" is **three**.

**The moment-of-the-first-tag item is overdue, and is done next, before the next tag.**
- `project.Releases` comes back, read by `-version`/`-help`. It is not read by the About box or
  the title screen, which are faithful, so fidelity is untouched.
- `.github/ISSUE_TEMPLATE/config.yml` gets one `contact_link` to it. `bug_report.yml` already asks
  "Where the build came from", with a "Releases page" option, so the templates need a link and not
  a rewrite.
- **README's opening** stops saying "for Linux" and "Clone it and `make run`", and links the
  Releases page, which has Windows zips on it.

**`RELEASING.md` at the root**, the one record on the tree of the steps that happen off this host.
- Pre-tag:
  - an rc tag or `workflow_dispatch`;
  - the hand-written `Zone.Identifier` double-click on the Windows test host (the paragraph
    above);
  - on a connected machine, all four `.exe` hashes on VirusTotal and a Defender download check
    with cloud protection on (5.12), because the test host cannot see Defender's cloud verdict;
  - `govulncheck ./...` clean on `GO_RELEASE`, under `GOOS=linux`, `GOOS=windows` and `GOOS=darwin`
    (5.11; `release.yml`'s `verify` runs it, so this is reading the log);
  - a CHANGELOG section for the tag.
- Post-tag, each marked **"needs a connected machine"**:
  - download from Releases and `sha256sum -c`;
  - run on a clean distro;
  - optionally `objdump` the glibc floor;
  - a real-download Mark-of-the-Web check on a connected Windows machine (the Windows test host is
    airgapped and `scp` strips the stream, so there the hand-written stream is the only option);
  - record the run URL.

Its commands are fenced as non-bash, so `docs-check` leaves them alone. That tool is for front-door
documents, and a dozen skip rules would defeat it.

**The Linux archive's glibc floor, stated and asserted.**
- `release.yml:217` (the build job; `:121` and `:688` do not affect the shipped binary) is pinned
  to `ubuntu-24.04`. That keeps today's measured floor of GLIBC_2.34. 22.04 and bookworm give the
  same 2.34, so neither lowers it, and `-tags netgo,osusergo` removes only `res_search`.
- After `make cross`, a step takes the highest `GLIBC_x.y` from
  `objdump -T bin/cross/glidergo-linux-amd64-x11` and fails above 2.34. That is the part worth
  most: a code change can raise the floor on an unchanged runner (`__isoc23_strtol@GLIBC_2.38`),
  and a pin cannot catch that.
- The Linux `HOW-TO-RUN`, the release notes and README's "Getting a build" say "needs glibc 2.34
  or newer (Ubuntu 22.04, Debian 12, Fedora 35, RHEL 9 or later) and libX11".
- Actually lowering the floor is a separate M decision. It needs an EL8 container (glibc 2.28,
  supported to 2029), because Debian 11 LTS has ended. GitHub runners cannot reach a private
  registry, so a mirror on one is only for rehearsing locally with podman.

**Pre-tag, in addition to the `Zone.Identifier` rehearsal:** the gate list in PLAN §4's release
gate. That includes the first run of step 4's win32 code: auto, the centred placement, the
changed-rows present read back at 2×, and the three bench rows that decide 2.76's cap.
`docs/windows-first-run.md`'s "What has changed since" has the steps.

**Done, in PLAN release gate step 5, except what needs a connected machine.**
- `project.Releases` is back. `-version` has a `releases` row between `home` and `bugs`, in both
  binaries. `-help` ends with the same three lines.
- `config.yml`'s first contact link is "Download a release".
- README's opening says Linux and Windows, links the Releases page, and keeps `make run` as the
  second way in.
- `RELEASING.md` is written. It has seven steps before the tag and five after, a "Still open"
  list, and a "Record" table. The table says what is not recorded here for the `v0.1.x` tags
  rather than guessing. CONTRIBUTING links it under "Releases".
- The build job is pinned to `ubuntu-24.04`. The step after `make cross` fails the tag above
  glibc 2.34. It was rehearsed against this host's own build: a floor of 2.34 passes, and one of
  2.32 fails with the `::error::` line.
- The rewrite this amendment names is done. `release.yml`'s header says which tags were built by
  it, that whether any of them needed a correction is not recorded here, and how to rehearse the
  next one.

**Found doing it: SECURITY.md was a dead link in every archive.** The packaging step rewrites
README's `docs/` links to the tagged tree, because an archive carries no `docs/`. README also
links `SECURITY.md` and `CONTRIBUTING.md`, which no archive carries either, and nothing rewrote
those. So the security policy was a 404 from inside every download. The `sed` now rewrites links
to `SECURITY.md`, `CONTRIBUTING.md` and `RELEASING.md` the same way. After it, a loop fails the
build if any relative link left in the archive's README names a file the archive lacks. That
turns the class into a check rather than fixing one instance. It was run against a staged
archive here, and it fails if the new `sed` expression is removed.

### 5.5 Nothing in here has ever been compiled by a macOS or Windows toolchain — **note; windows/amd64 is now run as well as compiled, macOS and windows/arm64 are still compile-only**

`make cross` builds `windows/amd64`, `windows/arm64`, `darwin/amd64`, `darwin/arm64`,
`linux/arm64` and `linux/amd64` in about four seconds, and it is worth being precise about what
that proves. There are now three backend selectors rather than two, and they are still exact
complements (`internal/platform/backend/doc.go` reads them side by side):
`backend_x11.go` is `linux && cgo && !nullbackend`, `backend_win32.go` is
`windows && !nullbackend`, and `backend_null.go` is the negation of both. So the darwin rows and
cross-compiled `linux/arm64` resolve to the null backend, and a green build for those is evidence
that the game logic, the house codec, the asset pipeline, the shell, the scores and the replay
harness are portable — and *no* evidence that they can draw a pixel. It also cannot see a
path-separator or filesystem-case bug, which is the class of thing that only appears on the real
OS.

The windows rows are a different case since the win32 backend landed: they carry a real backend
even at `CGO_ENABLED=0`, so a green build there means the Windows window code compiles for that
architecture. That is a stronger claim than "the portable part compiles" and a much weaker one
than "it works". **Nothing on this machine can execute it**, and the backend was written here, on
an airgapped Linux host, against nothing but the documentation.

It has since been executed elsewhere, on a real Windows Server 2025 desktop, and `windows/amd64`
is no longer a compile-only claim: the window opened, the blit put pixels on a screen that match a
Linux-rendered frame exactly, and the `waveOut` sink played every sound it was given.
`docs/windows-first-run.md` is the write-up and states the three gaps it left — no key was ever
pressed, nothing touched the window, and `windows/arm64` still has never run anywhere. For that
row, and for both darwin rows, every word above still holds.

CI remains the *automated* half of the verification story, so it is worth knowing exactly what
it does: `go build`, `go vet` and `go test ./...` on `windows-latest` and `macos-latest`; then, on
Windows only, `glidergo -version` and `glidergo -frames 300 -bench`, which opens a real window and
pushes 300 frames through `StretchDIBits`. The bench is `continue-on-error` on purpose — a hosted
runner is not a desktop session, so whether `CreateWindowExW` behaves there as it does in front of
a logged-in user cannot be settled from here, and the log is the interesting output either way.
The `python3 tools/extract_all.py` step is `continue-on-error` for the older reason: the extractor
is stdlib-only python3 that has only ever run on Linux, and a failure is a finding for this file
rather than a reason to hide the step.

macOS is still Stage 6, and there "it compiles" remains the whole claim.

**Next step: account for skips across the whole suite, natively.** No CI job does that today; Linux
enforces it only for `internal/fidelity` and `internal/citations`. `tools/skipcheck`, stdlib, reads
`go test -json ./...`. It echoes the output as text, so the existing failure-summary step's grep
still works, and `go test`'s exit status still fails the step under `pipefail`. It compares every
skip (package, test and subtests) with a committed per-OS allowlist, in two classes:
- **must-skip** — the 4 `internal/citations` tests on every CI OS, and the 2 XDG tests on Windows
  and macOS. It fails if one of these stops skipping;
- **may-skip** — `TestWaveOutPlaysSilence`, and netplay's TEST-NET-3 and two-listener skips.

It fails on any skip not listed. The lists are seeded from the static census and corrected after
one GitHub run, since they cannot be confirmed from here. It was to land after the fresh-clone
citations failure (`TestEveryReferenceToOurOwnTreeResolves` against the gitignored `scripts/env.sh`
and `scripts/local-source.sh`) was fixed, and that is fixed: a reference to a file `.gitignore` names
is held to `.gitignore`, not to the disk.

### 5.6 Is a fresh checkout playable? — **audited and yes; four defects found and fixed, 1.10a**

This item exists because the question had never been asked from the outside. Everything was
verified on the machine that wrote it, where Go was already on `PATH`, `assets/extracted/`
already existed, `DISPLAY` was already set, and `scripts/env.sh` had already been sourced — the
one configuration in which a stranger never finds themselves. So the tree was cloned to a
scratch directory and driven from the README, with nothing pre-warmed.

What the audit established, as facts rather than intentions:

- **A clone contains everything.** `git clone` → `apt-get install build-essential pkg-config
  libx11-dev` → `make run` is the whole sequence, since the decoded assets are committed (1.2).
  Nothing is fetched, because there is nothing to fetch: `internal/module` asserts stdlib-only,
  so there is no `go.sum`, no `vendor/` and no proxy to be unreachable.
- **The original game is not needed.** It never was, because `GliderPRO/` is vendored: `make
  assets` reads exactly **38 committed files** from it — `Glider PRO.r` (15,475,666 bytes, the
  derez'ed resource fork carrying every PICT, `'snd '`, `STR#` and dialog), 22 `.binhex` houses
  and 15 `.mov` movies — and writes 1,899 files in about 70 s. Since 1.2 was settled that output
  is committed too — all but the 22 `.rsrc` intermediates, which are 24 MB of the 39.5: 1,877
  files, 15.5 MB — so playing needs neither the extractor nor python3. `make assets-check`
  re-extracts and compares every file it commits, which is what keeps the committed copy honest.
- **`make check` passes twice over: on a full clone, and on one with the assets removed and no
  `DISPLAY`.** The asset-dependent steps skip by name and `check-caveats` closes by listing what
  it could not verify, so a green run on an empty tree never reads as a green run on a full one.
- **Size.** The port itself is about 11 MB of source, docs and tests. Everything else is 1994
  content: `GliderPRO/` 50.7 MB encoded, `assets/extracted/` 15.5 MB decoded from it. 1.2 records
  why both are here.

Four things the audit found, all fixed in the same commit as this entry:

1. `make smoke`, and therefore `make check`, **failed** on a clone that had a display but no
   extracted houses — it invoked the on-screen bench, which needs a house to fly in. That is the
   first command the README hands a stranger, broken by the one condition nobody developing here
   is ever in. It now skips with a reason and `check-caveats` reports the gap.
2. `-house Titanic` only worked from a directory that happened to contain the file; a bare name
   with an extension was never resolved against `-houses`. Both forms work now.
3. `make cross`'s host cgo build could fail without failing the target: the exit status was
   assigned to a variable that a later `printf` overwrote.
4. `make headless` listed a stale output directory, so a shot from a previous run could be read
   as work just done. It removes the directory first.

Left open, and deliberately: the README's claims about Windows and macOS remain untested by
anyone (5.5), the public dependency path remains untested from here (5.1), and where an
*installed* copy looks for its assets is still repo-relative (5.3). The audit's subject was a
developer's clone, which is the only thing gliderGo currently ships.

**Follow-on, same stage.** The audit's answer to "will the original source be needed for the
assets?" was *no, because it is vendored* — which is true and was not the point. The answer taken
instead was to remove the question: commit the decoded assets, so the first command after
`git clone` is `make run`. That is 1.2's route (a) and it retires the extraction step from the
quick start entirely.

### 5.7 The four files a public repository is expected to have, and the two templates — **five of six DONE, 2.0; `CODE_OF_CONDUCT.md` still deliberately absent; `SECURITY.md` reopened by Stage 3, rewritten with 4.33 and finished with 4.36 and 5.11**

1.3 records that `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md` and the issue and
pull-request templates do not exist. This is where the reasoning lives, because "add the standard
files" is a worse plan than it sounds: each of them is a promise to somebody, and writing a
promise before there is anyone to make it to produces the boilerplate that every reader learns to
skip.

What each one would have to say that is specific to gliderGo, and is therefore not yet writable:

- **`CONTRIBUTING.md`** — the useful content is already spread across the tree and would mostly
  be links: `docs/DEV_ENVIRONMENT.md` for the two ways in, `make check` as the gate,
  `internal/module`'s stdlib-only invariant as the one rule a patch can break without noticing,
  and the fidelity corpus as the reason a rendering change needs
  `go test ./internal/fidelity -update` and a hash in the diff. The part that cannot be written
  yet is the review process, because there is no second reviewer.
- **`CODE_OF_CONDUCT.md`** — Contributor Covenant 2.1, whose only project-specific field is the
  reporting address. There is no address to put in it, and a code of conduct with an unattended
  contact is worse than none.
- **`SECURITY.md`** — the honest text is narrow and worth writing precisely: gliderGo parses
  untrusted binary house files and BinHex, and `internal/house` is the attack surface a malicious
  `.house` reaches. It has no network listener in Stage 1; Stage 3's race protocol adds one and
  that is when this file stops being cheap. Again blocked on a reporting address.
- **Issue and pull-request templates** — the one genuinely valuable field is a `glidertool replay`
  trace, since 4.2 built the format precisely so a bug report can carry a reproducible input
  rather than a description. That template can be written the day the repository is public.

The decision recorded here is that all six wait for the public push, and that when they are
written they say the gliderGo-specific thing above rather than the generic thing.

**Written at 2.0, five of the six.** `CONTRIBUTING.md` (172 lines), `SECURITY.md` (51),
`.github/PULL_REQUEST_TEMPLATE.md`, and three issue forms —
`.github/ISSUE_TEMPLATE/bug_report.yml`, `fidelity_difference.yml` and `config.yml`. Each says the
specific thing this section demanded: `CONTRIBUTING.md` leads on the stdlib-only invariant and the
`-update` hash in the diff; `SECURITY.md`'s scope section is three paragraphs about house-file
parsing and one sentence about what does not exist; and `fidelity_difference.yml` exists as a
*separate* form from the bug report, because "the original did something else" and "this crashed"
want different fields and the former is the report this project most wants to receive.

**What unblocked them was noticing that the reporting address was the wrong blocker.** Both
`SECURITY.md` and `CODE_OF_CONDUCT.md` were deferred above on the grounds that there is no address
to put in them and an unattended contact is worse than none. That is true of an email address and
not true of the mechanism GitHub already provides: private vulnerability reporting routes to the
maintainer, is auditable, needs no inbox, and cannot be scraped. So `SECURITY.md` points at
**Security → Report a vulnerability** and states "one maintainer, no SLA" in place of a promise it
cannot keep — which is the honest version of the same file.

**`CODE_OF_CONDUCT.md` is the one still not written, and now for a better reason than the address.**
The same substitution does not work: GitHub's private reporting is security-only, so a covenant
would have to route conduct reports either to an unattended address or to GitHub's own abuse
process — and pointing at GitHub's process is telling people to go over the maintainer's head by
default, which is not what the document is for. Left absent rather than boilerplated. The trigger
to write it is a second maintainer, not the public push; that is who the reporting clause would
name.

**Reopened by Stage 3.** SECURITY.md `:14` ("a game with no network code") and `:33-37` ("there is
no network listener … this section will need rewriting when it does") are false: `-host`, or
Race… › Host, listens. The rewrite states what the shipped build does:

- **The listener.** TCP on every interface, IPv4 and IPv6, port 1994 or `-port`, only while the
  waiting screen is up. The first connection gets the only slot and learns the house name, hash and
  nonce before sending a byte. After 4.32, the listener closes after the first *match*, and a
  connection that stays silent is dropped after a deadline.
- **What a peer can send, and what is checked.** Frames are capped at `MaxMsg` (1 MiB), checked
  before allocating. Every field that decides something is validated (slot, state, matchID,
  lengths, the `MatchStart` seed and delay). Informational fields and standing values are taken as
  sent, and the house name is printed with `%q`. No file contents ever cross the wire. The surface
  is symmetric: a hostile host can send a guest exactly what a hostile guest can send a host, so
  the text says "peer".
- **What the hash gate is.** A check against honest mismatches, **not authentication**. `Hello`
  goes out before the peer's arrives, so a hostile peer can echo it. Results are on the honour
  system (4.38).
- **Files, more precisely.** `fs.FS` stops a path taken from file contents escaping, but `os.DirFS`
  follows symlinks inside a shared `-levels`, `-houseart` or `-art` tree, and only the house listing
  skips them. Size limits are 4.36's. (With 4.36 the listing follows them too, and the size limits
  are in SECURITY.md.)
- **Not changed:** the bind default. A loopback default breaks the feature, and "LAN only" is not a
  bind address. An optional future `-listen` is listed at most.
- **Dependencies and Versions.** The Go minor that builds releases, and what a Go security fix
  between tags means (5.11).

This is also the file 4.33's port-forwarding advice points at, so both change in one commit.
CHANGELOG gets a line.

**Done, with 4.33.** SECURITY.md's scope now describes the race.
- **The listener**: when it is open, what anything that connects learns first, and one connection
  at a time.
- **What a peer can send**, and the symmetry between host and guest.
- **The hash gate is not authentication.**
- **A forwarded port is exposure**, so close it after the race, or use a VPN instead.
- **What counts as a bug worth reporting.**
- **Files**: `fs.FS`, and the symlinks followed inside a directory passed to `-levels`, `-houseart`
  or `-art`.

Two items stay where they were. The bind default is unchanged, as planned. Dependencies and
Versions wait for 5.11, whose Go-minor text belongs there. **Done with 5.11:** both sections now
name the Go that builds releases and what a Go security fix between tags means.

### 5.8 What the fresh-checkout audit found and deliberately did not fix — **notes, 1.10a**

Recorded so that none of these is rediscovered as a surprise. Each one is small; each was left
alone for a stated reason rather than missed.

- ~~**`go.mod` says `module glidergo`, and a public repository's module path should be its
  URL.**~~ **Closed, 1.10c.** It was the one item here that could not be guessed — it needed the
  account the repository would actually live under — and that account is now known:
  `github.com/bwenstar/gliderGo`, set in `go.mod` and in 248 import lines across 113 files. The
  test that pins it keeps a second copy of the string on purpose (`internal/module`'s
  `modulePath`), because a test that read the path out of the file it is checking would accept a
  typo. Two new cases in that test's table earn their place only now that the path has a dot in
  its first element: `github.com/bwenstar/glidergo` and `github.com/bwenstar/other` must both be
  refused, which is what proves the prefix match is *this* module rather than anything on
  `github.com` or anything of this owner's.
- **`LICENSE` omits GPLv2's "How to Apply These Terms to Your New Programs" appendix.** That
  appendix is instructions to the *next* author, not part of the licence's operative text, and its
  absence changes nothing legally; the terms are complete (sections 0–12 plus the preamble). Left
  out because it invites a reader to think the boilerplate notice it contains has been applied to
  every source file, which is not true — see the next item.
- **No per-file copyright headers.** 394 tracked files carry none; the licence is stated once, in
  `LICENSE` and in the README's licence section. This is a defensible choice and a common one, but
  it is a choice: a file copied out of this tree carries no notice with it. Revisit if there is
  ever a second contributor, at which point 5.7's `CONTRIBUTING.md` is the place to state it.
- **`scripts/bootstrap-dev-env.sh` defaults its Ubuntu suite to `noble`.** Already overridable
  with `GLIDERGO_UBUNTU_SUITE`, so this is a default rather than a hard-coding, and the default
  matches the only host it has ever run on (Ubuntu 24.04.4). It will go stale; the fix is a
  release-file probe, which is more machinery than the problem deserves until somebody hits it.
- **Three Go files cite `docs/analysis/stage15-raw/`, which is gitignored** —
  `internal/game/transit.go:25`, `internal/game/guards.go:6`, `internal/replay/replay.go:7`. The
  citations point at working drafts that were deliberately not committed (`.gitignore` says why:
  they still contain the errors the review pass corrected, and committing both would leave two
  disagreeing sources of truth). A reader of a fresh clone therefore follows those three
  references to nothing. The "releasePolish N" numbering they quote exists only in those drafts,
  so the citations cannot simply be repointed at the committed spec — closing this properly means
  giving the surviving decisions numbers in a committed file. Deferred; noted here so the dead
  reference is a known one.
- ~~**The asset flags still default to paths relative to the working directory** (`-art
  assets/extracted/art`, and three more).~~ **Closed by 5.3.** All four now default to empty,
  meaning the copy inside the executable, and a directory only comes into it when somebody names
  one. `-version` still reports where each root came from, which is what makes the override legible
  from a bug report.
- **Audio still shells out to an external player on Linux and has no sink at all elsewhere.**
  Already 2.48. The audit's only addition is that `-version` prints the backend, so a report from
  a null-backend build is identifiable as one.

### 5.9 A release binary cannot be checked by rebuilding it — **planned; the flag at the next Makefile touch, the rest optional**

`-trimpath` is a `go build` flag: `BUILDFLAGS := -trimpath`, passed in the `build`, `glidertool`,
`headless` and `cross` targets. It does not go in `STAMPED`, which is `-ldflags`, and the linker
rejects it (exit 2, tested). It does not go in an exported `GOFLAGS` either, which would override
`scripts/env.sh`'s `-mod=mod`. Without the flag all 13 cross binaries differ between two checkouts;
with it they match.

- **Optional:** a CI step that rebuilds `make cross` in a second checkout and compares
  `bin/cross/*`.
- **The "verify a release" note says what can actually be checked.** Run `go version -m` on the
  downloaded binary, clone the tag clean (vcs.revision and vcs.modified are stamped), rebuild with
  that exact patch release and `VERSION`, and compare the **extracted binary's** hash.
  `SHA256SUMS` hashes the archives, which record mtimes and owners, so it cannot be reproduced
  unless packaging is made deterministic (`tar --sort=name --mtime=@<commit> --owner=0 --group=0
  --numeric-owner | gzip -n`; `touch -d` plus `zip -X`). That is extra work, and the packaging step
  would need re-rehearsing.
- **The cgo `linux-amd64-x11` binary** is reproducible only on a pinned runner image (5.4's
  glibc amendment), because its bytes also depend on gcc, binutils and libX11-dev.

Public binaries are built on GitHub runners, so they embed runner paths, not the developer's.

### 5.10 linux-arm64 cannot draw, and every Linux binary carries a glibc floor — **the arm64 job next; a pure-Go X11 client much later, opt-in**

**Next (S, off-host).** A native `ubuntu-24.04-arm` job builds the cgo x11 binary, runs `go test`
and a bench under xvfb, and hands the binary to the packaging job. So `linux-arm64` draws instead of
being `-headless`.

It is not a one-line change. The build job both cross-compiles and packages (`release.yml:183-569`),
so it needs `needs:` plus `download-artifact`, `:317` becoming `linux-arm64|-x11||tar`, and
rewritten text at `:30-35`, `:419-421` and README `:52`. The runner is free only for public
repositories. It carries the same GLIBC_2.34 floor: Pi OS Bookworm is fine and Bullseye is not. It
shares its runner-label edits with 5.4's glibc amendment.

**Later (L): `internal/platform/x11wire`, a stdlib X11 client** over the unix socket or TCP (for
`ssh -X`, `localhost:10`), with MIT-MAGIC-COOKIE-1 from `$XAUTHORITY` or `~/.Xauthority`. It would
give static `CGO_ENABLED=0` Linux binaries for every architecture and libc, with no glibc floor. It
has to rebuild everything `XLookupString` gives today:
- `GetKeyboardMapping` and `GetModifierMapping` (Shift, Lock, Mode_switch/level 3);
- the core-protocol keysym column rules, keysym to Latin-1, and a refresh on `MappingNotify`;
- `InternAtom` for `WM_PROTOCOLS`/`WM_DELETE_WINDOW`, and `ConfigureNotify` and `FocusIn`;
- XKB `QueryExtension`/`UseExtension`/`PerClientFlags` for 2.72's repeat semantics.

The guard is **new**. `-shot` draws through no Window backend, so comparing `-shot` output proves
nothing. One test `GetImage`s the window after `Present` and compares it byte for byte with
`platform.Expand`. Another runs the client against an in-process fake X server over `net.Pipe`,
which covers request encoding, strip splitting and event decoding on headless CI. Speed is not a
risk either way: the probe did 466–508 fps and Xlib 502–504 fps on the same Xephyr.

It ships behind a build tag or backend flag first, and becomes the default only after runs on
GNOME/XWayland, KDE and a Raspberry Pi. It absorbs 4.42's X11 half and 2.74's keycode work, and it
is not Stage 6.

### 5.11 Releases are built with Go 1.23, which Go no longer patches, and nothing scans them — **DONE: releases build with Go 1.27.x, govulncheck gates the tag, and the OS floors are stated**

`go.mod` says `go 1.23`, and every `setup-go` step reads it with `go-version-file: go.mod`
(`ci.yml:99`, `:205`, `:276`, `:319`; `release.yml:127` and `:208`). So every release is built with
some 1.23.x. Which patch the runner picks is in the Actions log, not on this host. Go supports "the
past two Go releases" (`$(go env GOROOT)/SECURITY.md:5`), so 1.23 stopped getting fixes when 1.25
shipped. That mattered less while the game only read files. The race-bearing tag listens on TCP
(`internal/netplay/dial.go:65`) and decodes the PNGs a shared house carries
(`internal/render/housepict.go:359`). SECURITY.md says the standard library is "the only third-party
code in a gliderGo binary" (`:48`), so the toolchain *is* the supply chain, and nobody has looked at
it. `govulncheck` has never run, and nothing says which later `net` or `image/png` fix a 1.23 build
is missing.

The `go` line sets the lowest Go that can build this code. It does not have to be the Go that builds
a release. `GOTOOLCHAIN=local` stops this airgapped host from trying to download a newer Go
(DEV_ENVIRONMENT §3). On a runner it only means "keep the Go that setup-go installed" (`ci.yml:66`).
A newer toolchain builds this module unchanged and keeps 1.23's GODEBUG defaults, because those
follow the `go` line (`doc/godebug.md`, "Default GODEBUG Values"). So security fixes land and
GODEBUG-gated behaviour changes do not. The repo already makes this choice for contributors:
`bootstrap-dev-env.sh --source public` installs the newest stable Go at or above the floor, "because
a patch release is where security fixes live" (`:273-275`). `ci.yml:96` calls the `go` line "one
source of truth", but its two readers do different things with it.

- **The toolchain.** Set `GO_RELEASE: "1.NN.x"` once in each workflow's `env`. `1.NN` is a minor Go
  still supports, chosen on a connected host. `release.yml`'s `verify` and `build` and `ci.yml`'s
  `cross` and `native` use `go-version: ${{ env.GO_RELEASE }}`. `check` and `citations` stay on
  `go-version-file: go.mod`, so README's "Go 1.23 or newer" is still tested on the Go this host has.
  `go.mod`, `GOTOOLCHAIN=local` and `GOPROXY=off` do not change. The patch floats on purpose, and
  5.9's rebuild recipe reads the exact one from `go version -m`. The minor moves when Go drops it,
  every six months.
- **The scan.** Add a `vuln` job to `ci.yml` and a step to `release.yml`'s `verify`, both on
  `GO_RELEASE`. Only the install step gets `GOPROXY=https://proxy.golang.org`, for a pinned
  `go install golang.org/x/vuln/cmd/govulncheck@vX.Y.Z`. That installs a tool and writes no
  `require`, so `internal/module` still passes. Then run `govulncheck ./...` under `GOOS=linux` and
  under `GOOS=windows`, because netplay's sockets and the audio sinks differ per OS. In
  `release.yml` a finding stops the tag. In CI it fails the job and gets an entry here saying
  whether netplay or a decoder reaches it. `ci.yml` has no `schedule:` today. A weekly one catches a
  Go security release that lands between tags. None of this can run here, because this host has no
  vulnerability database and no proxy. (That was wrong: see **Done**, which ran all of it here.)
- **The floors, taken from the toolchain.** Here is what 1.23.12 sets, measured on this host:
  - macOS: a `CGO_ENABLED=0` darwin build carries `LC_BUILD_VERSION` minos 11.0.0, which the Go
    linker hard-codes (`ld/macho.go:489` in the toolchain's source).
  - Windows: a windows/amd64 PE header says 6.1, which is Windows 7, so the Windows loader never
    refuses the file. But `runtime/os_windows.go:270` looks up `ProcessPrng` by name, and `randinit`
    calls it at startup without checking it was found (`runtime/rand.go:51`, `os_windows.go:503`).
    On a Windows that lacks it, the game dies inside the runtime before it can say anything. What
    the player actually sees is unverified.
  - Linux: the kernel floor in this source is 2.6.32 (`syscall/syscall_linux_accept.go:5`). An x11
    player meets the glibc 2.34 floor (5.4's amendment) before that.

  This GOROOT carries no release notes. So which Windows versions lack `ProcessPrng`, and what later
  minors raise, are unverified here; "Windows 10 or Server 2016 since 1.21" is remembered, not
  checked. Copy them from the chosen minor's release notes ("Ports") when `GO_RELEASE` is written,
  and re-check them at every bump. They go into README's "Getting a build", the release notes (which
  today say only "Windows needs nothing at all", `release.yml:721`), both HOW-TO-RUN.txt variants
  and the platform tiers in PLAN §4's release policy. `otool -l` on `macos-latest` can assert the
  macOS one.
- **The documents.** This goes into 5.7's rewrite. In SECURITY.md, "Dependencies" names the Go minor
  that builds releases and says `go version -m` shows it. "Versions" says a Go security fix that
  reaches gliderGo is reason enough for a release. RELEASING.md's pre-tag list (5.4's amendment) and
  PLAN §4's release gate get one line: govulncheck is clean on `GO_RELEASE`.

This host does not have to leave 1.23, because `check` keeps the floor honest. If it ever should,
DEV_ENVIRONMENT §3's container trick works unchanged with `golang:1.NN-bookworm`.

**Done.** `GO_RELEASE: "1.27.x"`, as planned, with one change: the scan runs as macOS too.

- **The scans, run here.** The plan said this host had no database and no proxy. It turned out to
  reach a Go module proxy through its registry mirror, and the database is a Go module
  (`golang.org/x/vulndb`), so `cmd/indexdb` built one: 4,474 entries, as of vulndb commit
  `f197f14625e2` (2026-09-17). The Go toolchains came out of `golang:1.27-bookworm` (go1.27.1) and
  `golang:1.26-bookworm` (go1.26.8), by DEV_ENVIRONMENT §3's container trick.
  - **go1.23.12**, the Go the v0.1.x releases were built with, reaches three vulnerabilities,
    identically under `GOOS=linux` and `GOOS=windows`:
    - GO-2026-4971, a panic in `net`'s Dial and LookupPort on a NUL byte, on Windows (fixed in
      1.25.10). It is reached from `cmd/glidergo/race.go:741` (`defaultRoute`) and
      `internal/netplay/dial.go:74` and `:110` (`Listen`, `Join`). The address a player types
      reaches `Join`.
    - GO-2026-4602, a `FileInfo` that can escape an `os.Root` (fixed in 1.25.8). It is reached from
      `internal/assetfs/assetfs.go:242` (`Measure`'s walk) and `tools/docscheck/main.go:420`. The
      game uses no `os.Root`, but govulncheck counts the `ReadDir` it goes through.
    - GO-2026-4342, CPU spent building a zip's index (fixed in 1.24.12). It is reached from
      `internal/house/binary.go:322` (`LoadFS` on a zip), which is how a house in an archive loads.
    - Also 3 in imported packages and 39 in modules, which the code does not call.
  - **go1.26.8 and go1.27.1**: "No vulnerabilities found", as Linux and as Windows, and 1.27.1 as
    macOS too, and as Linux with cgo off.
  - The v0.1.x releases carry the three. The next release is the fix, and its notes should say so.
- **The toolchain.** `release.yml`'s `verify` and `build` and `ci.yml`'s `cross` and `native` use
  `go-version: ${{ env.GO_RELEASE }}` with `check-latest: true`, because without it setup-go takes
  whatever 1.27 patch the runner image has cached. `check` and `citations` stay on `go.mod`.
  `build` records `go env GOVERSION` as an output, and the release notes name it.
  - On go1.27.1, `make check` passes in full, including `make race`, the pixel corpus and
    `docs-check`, and so does `make cross`. `go vet ./...` is clean under linux, windows and darwin
    on both 1.26.8 and 1.27.1, and `go test -count=1 ./...` passes on both.
  - The engine fingerprint does not move. `TestTheEngineFingerprintIsPinned` passes on 1.27.1, so
    a 1.27 build races a 1.23 one, as the GODEBUG argument above predicted.
- **The scan.** It is a `vuln` job in `ci.yml` and a step in `release.yml`'s `verify`, both on
  `GO_RELEASE`. govulncheck is pinned at v1.8.0, which was `@latest` on 2026-09-23 and needs Go 1.26
  (its `go.mod`). It is installed with `GOPROXY=https://proxy.golang.org` on that step alone, then
  run with `GOOS` set to linux, windows and darwin in turn. macOS was added because releases ship
  darwin archives, and the scan costs seconds. The `vuln` job installs libX11, so the linux scan
  compiles the x11 backend the linux-amd64 archive is built with. `ci.yml` gains a weekly
  `schedule:` (Mondays, 06:23 UTC). The whole matrix runs then, not only `vuln`, which is free on a
  public repository and also catches a runner image that changed.
- **The floors, measured on the binaries** `make cross` builds (a `debug/macho` and `debug/pe`
  reader, and `objdump -T` for glibc):

  | Go | macOS `minos` | Windows PE (OS and subsystem) |
  |---|---|---|
  | 1.23.12 | 11.0.0 | 6.1 |
  | 1.26.8 | 12.0.0 | 6.1 |
  | 1.27.1 | 13.0.0 | 10.0 |

  1.27.1's linker sets them, at `ld/macho.go:449` (`macOS = macVersionFlag{13, 0, 0}`) and
  `ld/pe.go:285` (`PeMinimumTargetMajorVersion = 10`) in the toolchain's source. The glibc floor of the x11 build is still
  GLIBC_2.34 on 1.27.1. A 10.0 header is also why 1.27 over 1.26: a Windows older than 10 now
  refuses the file at the loader, rather than starting it and dying inside the runtime on
  `ProcessPrng`, which is what 1.23's 6.1 header allowed. What an old Windows actually shows is
  unverified, since nothing here runs one. The cost is macOS 13 for the headless darwin archives,
  where 1.26 would have been 12. Nothing in the 1.27 source states a Linux kernel floor. From
  memory, not checked here: Go has needed Linux 3.2 since 1.24, and glibc has refused to run on
  anything older since 2.26. So no system with glibc 2.34 is below Go's floor, and the glibc line
  is the one stated.
- **Where the floors are written.** README's "Getting a build" has a table. The release notes'
  "Windows needs nothing at all" became Windows 10 or Server 2016, glibc 2.34 and macOS 13, and the
  Windows, Linux and darwin `HOW-TO-RUN.txt` texts each say their own. PLAN §4's platform tiers
  name them. Found doing it: the Windows `HOW-TO-RUN.txt` still said "you may well be the first
  person to see it draw", which the Server 2025 run made untrue. It now says what the release notes
  say.
- **The documents.** SECURITY.md's "Dependencies" says releases are built with Go 1.27, that
  `go version -m` and the notes name the patch, and that a release is not published past a
  finding. It also warns that a 1.23 build from source carries the three. "Versions" says a Go
  security fix that reaches gliderGo is reason enough for a release. DEV_ENVIRONMENT §3 has the
  offline recipe, with no host named in it. The RELEASING.md line in 5.4's amendment gains
  `GOOS=darwin`.
- **A caveat about the local scans.** This host's mirror proxies modules but not `sum.golang.org`,
  so govulncheck and vulndb were downloaded with `GOSUMDB=off`, trusting the mirror for their
  checksums. Neither is compiled into gliderGo, and CI's install checks against the real checksum
  database. A local result is a second opinion, and CI's is the one that gates.

### 5.12 Defender can quarantine the `.exe` outright, and nothing a player is told covers that — **note; the offline scan has run, and is clean; the notes paragraph and the whole price DONE (PLAN release gate step 5); the connected-machine check before the next tag, the resource experiments after 4.42**

4.13 and 5.4 prepare a player for SmartScreen, Mark of the Web and Gatekeeper. All three are
warnings with a way past them. Microsoft Defender Antivirus is a different component, and it does a
different thing. When it decides a file is malware it quarantines it, either as the zip is extracted
or on the first double-click, and no dialog offers a Run anyway button. The player sees an unpacked
directory with `glidergo.exe` missing, or a double-click that does nothing and a "Threats found"
notification. Nothing in `release.yml`'s notes, either `HOW-TO-RUN.txt`, README, SECURITY.md or
`docs/windows-first-run.md` mentions antivirus. One sentence in the notes points the wrong way.
"None of the three looks at what is *in* the archive" (`release.yml:658`) is true of the three it
names, and a reader will take it to cover the fourth. Defender does look.

The Windows binary has most of the traits that Defender's machine-learning detections
(`Trojan:Win32/Wacatac.B!ml` and its relatives, which the Go FAQ's entry on virus scanners exists to
answer) tend to key on. Measured on `bin/cross/glidergo-windows-amd64.exe` at `1797f9f`:

- it is unsigned and new, so it has no reputation;
- it has no `.rsrc` section at all: no VERSIONINFO, no icon, no manifest;
- its import table names `kernel32.dll` only. `user32`, `gdi32` and `winmm` are resolved at run time
  through `LoadLibraryExW` and `GetProcAddress` (`syscall.NewLazyDLL` in `win32.go:76-78`,
  `syscall.LoadDLL` in `waveout_windows.go:122`). That is ordinary Go, and it is also what a loader
  does;
- 11.4 MB of its 15.7 MB is `.data` at 7.98 bits per byte. That is `assets/extracted.zip`, and to a
  heuristic it looks like a packed payload. `glidertool.exe` carries the same bytes;
- it is linked `-s -w` and is a console program;
- and from the tag that carries `internal/netplay`, it calls `net.Listen("tcp", …)`.

None of that is wrong. Whether this build is actually flagged is **unknown**, and the Windows test
host cannot settle it. The verdicts that matter come from cloud-delivered protection and Block at
First Sight. Those only run on a file carrying Mark of the Web, on a machine that can reach
Microsoft. The test host is offline, and `scp` strips the stream. A
`MpCmdRun.exe -Scan -ScanType 3 -File` there checks local signatures only, so a clean result says
nothing about the ML verdict. The host did not answer on any port while this item was written, and
that weak version has since run, with the result below.

**Measured on the Windows test host, 2026-09-23: nothing fired, which answers a narrower question
than a player's machine will ask.** The probe was `make cross`'s `glidergo-windows-amd64.exe`:
15,716,352 bytes, sha256 `66c459714811f8dc92c6ae4f006fc7b3754a22c0b51dcd841489489cfd87f3f1`, stamped
`9becf4b-dirty` because it was built nineteen seconds before `1797f9f` was committed, and copied
there with `scp`. `9becf4b` is the hash that `1797f9f`'s parent, `dae1a26`, had before the commits
after `v0.1.2` were moved onto the tag's published commit. The host is Windows Server 2025, with
Defender platform 4.18.26030.3011 and engine 1.1.26030.3008. Real-time, on-access, IOAV and
behaviour monitoring are on and there are no exclusions. MAPS is at Advanced, `SubmitSamplesConsent`
is 1, and Block at First Sight is left enabled. Those are the defaults, so the policy is a player's.
Tamper Protection is off, where consumer Windows turns it on, and that does not change what is
detected. Only the connectivity is not a player's.
- The file was still on disk with its hash intact 32 s after the on-write scan, and no Defender
  event had been logged.
- `MpCmdRun -Scan -ScanType 3 -File … -DisableRemediation` reported "found no threats" and exited 0.
  So did a second copy carrying a hand-written `Zone.Identifier` of `ZoneId=3`, and the 2026-09-21
  build that had been run on that host.
- `Get-MpThreatDetection` and `Get-MpThreat` are empty. The Operational log has no detection event
  anywhere in its history back to 2026-04-17, a span that includes the 2026-09-21 build's run with
  behaviour monitoring on.

That rules out a local signature or client-side heuristic that matches this code today. It cannot
rule out the verdicts that matter for a public download, because every one of them happens in the
cloud, and this host has not reached the cloud since 2026-04-17. Its signatures are 1.449.140.0, 159
days old, and every update since has timed out with 0x80072ee2. `MpCmdRun -ValidateMapsConnection`
fails with 0x800705b4 and dates the last good MAPS connection to the same day. So the cloud ML
behind the `Trojan:Win32/Wacatac…!ml` verdicts that unsigned Go binaries are known for went
untested. So did Block at First Sight's upload of a never-seen executable, and SmartScreen's app
reputation. All three are keyed to a hash nobody has yet: the tag build stamps its own version, so
what players download is not these bytes. Nor was the file downloaded through a browser, so the Mark
of the Web was never written by the path that writes it for a player. Step 1 below is still the
check that settles it, and until somebody has run it, "Defender leaves it alone" has been observed
offline and nowhere else.

What to do:

1. **Before the tag, on a connected machine.** This is the line in 5.4's `RELEASING.md` pre-tag list
   next to the `Zone.Identifier` rehearsal. Take the rc or `workflow_dispatch` zips. Look up the
   four `.exe` hashes, two in each zip, on VirusTotal, and upload any that nobody has. The line
   that matters is Microsoft's.
   A handful of small engines flag most fresh Go binaries, so a hit from one of those is recorded,
   not chased. Then use a Windows machine with real-time and cloud protection on:
   `Get-MpComputerStatus` shows `RealTimeProtectionEnabled`, and `Get-MpPreference` shows a non-zero
   `MAPSReporting`. Download the zip in Edge, extract it in Explorer and double-click the exe. The
   expected result is the SmartScreen dialog and nothing from Defender.
2. **If Microsoft flags either file,** submit it at
   `https://www.microsoft.com/en-us/wdsi/filesubmission` as a software developer, marked incorrectly
   detected. Give the release URL, the tag, the zip's `SHA256SUMS` line and the flagged file's own
   SHA-256: `SHA256SUMS` lists the archives, not the `.exe`s inside them. Record the submission ID, and
   the date the verdict cleared, in `RELEASING.md`. A clearance applies to one file, and every tag
   produces new files, so this is a step for every tag, not a one-time fix.
3. **One paragraph in the release notes and the zip's `HOW-TO-RUN.txt`,** under "Your computer will
   try to stop you, once". It should say:
   - what quarantine looks like;
   - that a name ending in `!ml` is a model's guess, not a match against known malware;
   - that the check to run is the file's hash against `SHA256SUMS`;
   - that Windows Security → Protection history → Restore brings the file back and needs an
     administrator;
   - and that a report should be an issue quoting the detection name.

   `release.yml:658` also gets reworded so it no longer reads as covering antivirus.

   **Done, in PLAN release gate step 5.** The notes' "None of the three looks at what is *in* the
   archive" is now "Those three are about who signed the file", followed by "Antivirus is
   different" and a "Windows, if `glidergo.exe` disappears" paragraph. The Windows `HOW-TO-RUN.txt`
   gains "IF GLIDERGO.EXE HAS DISAPPEARED", with `Get-FileHash` on the zip by its real name. The
   check is on the zip because `SHA256SUMS` has no line for the `.exe`. The same fact was wrong in
   step 2 above and in `RELEASING.md`'s step 7, and both are corrected. "Verifying the download"
   gains `Get-FileHash` for Windows. The detection name 5.12 quotes is left out of both, so that
   neither reads as if this build were flagged. The Restore click path is Microsoft's documented UI,
   and nobody here has seen it; the notes' existing "documented behaviour rather than something this
   project has watched happen" covers it.
4. **Later, and only by measurement.** Nobody has tested whether 4.42's VERSIONINFO and icon,
   `-H windowsgui` (4.35's separate note) or a smaller embedded archive changes the verdict. Until
   someone does, these are folklore. Once step 1 has run there is a baseline, and each change can be
   compared by VirusTotal result on an rc build. 4.35's `crash.log` is an ordinary file write in the
   data directory and is not worth an experiment.
5. **Price the unsigned decision fully.** Add one sentence to the decision in 4.13 and
   `release.yml:47-61`. Signing is also what lets reputation carry from one release to the next.
   Unsigned, every tag's binaries start at none, and step 2 may be needed each time. The decision
   can stand, but the price stated for it should be the whole price. **Done with step 3.** The
   notes no longer say a certificate is "the only thing that removes" SmartScreen's dialog, which
   was more than a certificate does. A signed file with no reputation gets the dialog too. They now
   say that an unsigned file's reputation is its own, so every release starts again from none, and
   that a certificate is what lets it carry from one release to the next. `release.yml`'s header and
   4.13 add the Defender half.

### 5.13 A Glider PRO port with a public page and not a word about Aerofoil — **note; the README paragraph waits for a connected host (not done in the gate's docs pass, on purpose), the refusal with 4.43's importer**

At `1797f9f`, `grep -rniE 'aerofoil|lasota'` over the whole tree found the word in one place, and
not about the port: room 222 of ImagineHouse PRO II, one of the 22 original houses, is named
"Aerofoil", and grep reports that house's binary file as a match. There was nothing else: not the
README, not PLAN, not this file, not a code comment. Aerofoil is Eric Lasota's port of the same
GPLv2 source release, and it has been public since about 2020. Somebody who has played Glider PRO in
the last few years has most likely played it there, so the first question they bring to this README
is "why this one?", and nothing here answers it. The README has no section that compares gliderGo
with anything.

**What can be said, sorted by how far it can be trusted.** This host cannot open Aerofoil's
repository, so every claim about it below is labelled.

- *Checked here, about gliderGo:*
  - It is a transcription: about 17,800 citations checked by `internal/citations`, a per-frame hash
    corpus, and the demo replay (README "How faithful is it?").
  - It uses only the standard library, asserted by `internal/module/stdlib_test.go`.
  - It has a two-machine race the original never had (README "Two players, two machines").
  - It has two houses of its own in `levels/`.
  - It draws on Linux/X11 and Windows only, and macOS runs headless. That belongs in the same
    paragraph.
- *Known with confidence, to be confirmed before it is printed:* Aerofoil is Eric Lasota's, at
  `github.com/elasota/Aerofoil`. It is C++ and reimplements the parts of the Mac Toolbox it needs.
  Windows was its first platform.
- *Believed, not checked:* it keeps the original's C game logic rather than rewriting it; it has
  Android and browser builds; it keeps the 1994 house editor, which here is Stage 5; it has no
  network play; it is GPLv2; it stores houses in a layout of its own (below).
- *Not known at all:* what it says about the art and the houses, whether its releases carry them,
  and whether Lasota asked John Calhoun.

**The README must not claim a difference nobody has measured.** "More faithful than Aerofoil" is
exactly that kind of claim. If Aerofoil kept the original's C game logic, which nobody here has
checked, its physics may be as faithful as ours. What gliderGo can claim is its method: its fidelity
is checked by a test suite. The paragraph below is for the release gate's docs pass (PLAN §4, step
5), and goes beside the "Known differences from 1994" list that PLAN §4's release policy proposes.
Fill in the bracketed parts once checked, or cut them:

> **Other ports.** [Aerofoil][aerofoil], by Eric Lasota, is a port of the same source release [that
> runs on more platforms than this one: confirm the list]. gliderGo is a different kind of project.
> It is a transcription into Go, cited line by line against the 1994 C and held to a per-frame pixel
> corpus. It uses nothing but the Go standard library. It adds a race between two machines, which
> the original never had, and houses of its own. [If you want Glider PRO on a phone, in a browser or
> with its house editor, use Aerofoil: confirm each.]

`[aerofoil]` is the repository address above with `https://` in front, and it is left out of the
draft on purpose. `internal/project`'s `TestEveryGitHubLinkIsOneWeMean` fails on any GitHub link
that is neither this project nor the upstream C, this file included, so the link goes into that
test's allowlist in the same change that puts the paragraph in README.

This is a Should, not a Gate.

**4.43's importer does not read Aerofoil's layout, and today that layout is ignored without a
word.** *(Believed, not checked.)* Aerofoil converts Mac houses ahead of time into a `.gpd` data
fork, a `.gpa` resource archive (a zip) and a `.gpf` metadata file. A player coming from an Aerofoil
install may have houses in that form rather than as `.sit.hqx`. `houseExts`
(`internal/shell/library.go:135`) is `""`, `.house` and `.glh`, so a `.gpd` is never listed and
never complained about. What to do:

1. **Refuse it by name first.** `Discover` and the path loader recognise `.gpd`/`.gpa` and say that
   this is an Aerofoil house and gliderGo cannot read it yet. That needs no knowledge of the layout,
   and it is the same move as 4.43's StuffIt rejection.
2. **Check one converted house on a connected host.** If the `.gpd` is the data fork byte for byte,
   `peek.go`'s header check says so in one read, and the data half of the import is a rename. The
   `.gpa` half means listing its entries. BMP pictures would need a small decoder of our own,
   because `image/bmp` is `golang.org/x/image`, not the standard library.
3. **Add the layout to 4.43's list only if step 2 says it is cheap.** Otherwise the refusal stays.
   Houses in their 1990s wrappers (unverified which kind is commoner) are already 4.43's.

If the importer is ever written from Aerofoil's source rather than from sample files, its comments
say so, the way the port cites upstream.

**1.2 route (c) has a precedent that nobody here has read.** Aerofoil faced the same question:
shipping somebody else's art and houses in a new form. What it did is the closest precedent (c) has,
and finding out takes ten minutes on a connected host. Read its README and licence files, see
whether its release archives carry the houses, and see whether it records any permission from
Calhoun. If it records a grant, that is the template for (c). If not, it is a second project on
route (a), which changes nothing about (c). Either way, 1.2 gets one sentence citing what was found.

**What is owed.** No Aerofoil code or data is in this tree, so there is no licence obligation and no
`credits.txt` line. The courtesy is the README paragraph itself. If 4.43 ever reads Aerofoil's
layout, the importer credits the format. The announcement does not go in Aerofoil's issue tracker.

**Not done in PLAN release gate step 5, on purpose.** Each bracket in the draft is a fact about
Aerofoil, and none can be checked from this host. A paragraph with the brackets cut is left
saying only "there is another port", which is not worth a link to a project nobody here has read,
and printing the brackets unconfirmed is what the paragraph's own rule forbids. So it is a step
in `RELEASING.md`'s "Still open" list, for the first connected machine. The `.gpd` refusal is
code, and it goes with 4.43's importer, not with a docs pass.

**The announcement is a step in the gate.** PLAN §4's release gate ends its order of work with a
seventh step, "Announced, last", after step 6's check by a person and after 5.1's connected-host
checks, because the announcement is when strangers arrive. Where to announce is the user's call.
Package-manager manifests (winget, Scoop, Flathub, AUR) have to be updated for every tag, and they
run into 5.4's unsigned-binary warnings, so they are later and not part of the gate.

---

## Done

| Item | Stage | Commit |
|---|---|---|
| 1.1 GPLv2 `LICENSE` and a `README.md` licence section | 1.5a | earlier |
| 2.9 Scoreboard moved on screen as a documented deviation | 1.5b | this stage |
| 2.11 `World.Diag` counts the silently dropped dirty rects (the surfacing is still 1.7) | 1.5b | this stage |
| 2.17 `World.WaitTick` hook, and a sleeping limiter in `cmd/glidergo` | 1.5b | this stage |
| 2.27 `platform.EventExpose`, emitted by the x11 backend and handled by the host | 1.5b | this stage |
| 2.29 A bitmap font, full Mac Roman coverage, wired into the scoreboard's three panels | 1.5b | this stage |
| 2.31 `DrawCalendar` draws its month, from `STR# 1005` and `Scene.Clock` | 1.5b | this stage |
| 2.33 `badIndex`, one named path for every out-of-range read the C performs and the port refuses | 1.5b | this stage |
| 4.2 `internal/replay` and `glidertool replay`: the bug-report format | 1.5b | this stage |
| 2.25 `Scene.RedrawCentralRoom`: a light switch repaints one room, not nine | 1.5c | this stage |
| 2.34 (outlet half) `HandleOutlet` names its destination instead of inheriting an ambient port | 1.5c | this stage |
| 4.3 (first half) The golden trace's failure report names the frame that moved, not the digest | 1.5d | this stage |
| 2.34 (grease half) `HandleGrease` names both destinations at the call site, back map then work | 1.5e | this stage |
| 4.4 `TestStripRectsTileTheirSheet` written, closing a comment that had promised it since 1.5c | 1.5e | this stage |
| 2.43 The fifth confetti cloud is dropped, and the refusal is reported through `badIndex` | 1.5f | this stage |
| 2.44 (the report half) `Scene.SavedMapDrops`, the `dr=` golden column, and the 4,070-room census | 1.5f | this stage |
| 4.5 `Result.Planes`: three index-plane hashes, so a report can say the *pixels* differ | 1.5f | this stage |
| 2.47 `FlushTriggerSound` resets the flushed channel's priority, so three trigger sounds cannot silence the game | 1.6 | this stage |
| 4.2 (audio half) the `snd=` column, the mix digest and `glidertool replay -wav` | 1.6 | this stage |
| 2.6 A fresh clone with no assets comes up on the port's own title screen and says which command produces them | 1.7a | this stage |
| 2.7 (the title screen's half) Quit on the menu, on `Q` and on Escape; Escape in a game returns here | 1.7a | this stage |
| 2.50 Panels are a solid ground with a cream frame; the 50% dim is a halo, never a backing for text | 1.7a | this stage |
| 2.51 An unavailable item under the cursor is outlined, not filled, and says why when pressed | 1.7a | this stage |
| 2.52 The About box lists this build's bindings instead of the original's | 1.7a | this stage |
| 3.4 (the way in) Title screen, menu, house picker over all 22 houses, About box | 1.7a | this stage |
| 4.6 `-shot` and five shell screens rendered by `make headless`, with no display | 1.7a | this stage |
| 2.1 (the setting) `scale` is remembered in the preferences file | 1.7b | this stage |
| 2.3 All eight bindings are the player's, and no glider steers with a modifier key | 1.7b | this stage |
| 2.5 `DoPause` is a pause *state* with PICT 1015/1016, not a blocked process | 1.7b | this stage |
| 2.7 (the in-game half) `Q` while paused gives up the game and comes back to the title screen | 1.7b | this stage |
| 2.15 Leaving a game no longer waits for a physical key release | 1.7b | this stage |
| 2.17 (the setting) `keep_real_time` declared, with the reason it is unwired on the field | 1.7b | this stage |
| 2.19 The mirror-room flame blink, as an opt-in fidelity switch | 1.7b | this stage |
| 2.20 The mirror's player-two foil, as an opt-in fidelity switch | 1.7b | this stage |
| 2.21 `doBackground` split into a fidelity switch and not a user option | 1.7b | this stage |
| 2.28 A paused game says so on screen | 1.7b | this stage |
| 2.32 (one of three) The pause no longer stops the world from inside a frame | 1.7b | this stage |
| 2.39 The switch's spurious corner sparkle, as an opt-in fidelity switch | 1.7b | this stage |
| 2.53 A measurement never reads the player's settings (`-frames`, `-bench`, `-dump`) | 1.7b | this stage |
| 2.54 `Assets.Plate`: the open house's fork first, then the application | 1.7b | this stage |
| 2.55 Settings are saved at the change, `-import-prefs` refuses to overwrite | 1.7b | this stage |
| 3.1 High scores: `internal/scores`, a per-house side-car, both entry dialogs, the board on screen | 1.7c | this stage |
| 3.4 (the credits) `internal/credits`, pinned against `GliderPRO/README.md`, on a screen | 1.7c | this stage |
| 2.32 (the other two) `internal/game/wait.go` and the `World.Wait` hook: no `Delay` anywhere | 1.7d | this stage |
| 2.60 Both endings ask `TestHighScore` before redrawing the splash | 1.7d | this stage |
| 2.61 The win animation's out-of-bounds seeding write, skipped; its dead `RandomInt` draws, kept | 1.7d | this stage |
| 2.62 The quit path's unreachable splash repaint dropped, so the replay corpus can check the erase pass | 1.7d | this stage |
| 4.7 `internal/fidelity`: per-frame pixel hashes checked in, so a moved pixel names its frame | 1.8a | this stage |
| 4.6 (the corpus half) The shell's six screens are compared against a reference, not just rendered | 1.8a | this stage |
| 2.63 The recorder refuses a duplicate frame, `Validate` rejects one, and `demo check` reports it | 1.8b | this stage |
| 2.64 `devDemoRecord` reports once per run, and `badIndex`'s doc names the exception | 1.8b | this stage |
| 2.65 The arcade abort is transcribed and pinned: any game key hands the player the game back | 1.8b | this stage |
| 2.18 (premise) The demo is a determinism oracle, and the shipped Carbon build never seeded at all | 1.8b | this stage |
| 2.18 (as far as it can be without a Mac) The RNG's stream, `RandomInt`'s bound and skew, and the naive-int32 divergence, all pinned as table tests | 1.8c | this stage |
| 2.18 (scope) One random stream per *process*, as `qd.randSeed` is, handed back after every game | 1.8c | this stage |
| 2.18 (the launch draw) `-seed` defaults to 1, `0` means the clock, and `AdvanceRandSeed` accounts for `VariableInit`'s draw so the first game starts on 16807 | 1.8c | this stage |
| Contract item 6 `game.TestPlayGameOrder`: the frame loop's call order *and* each call's guard set, read out of the AST | 1.8c | this stage |
| Contract item 15 `render.TestTheMaskingStrategyOfEveryObjectType`: the 21/9/2/3 painter census, which no pixel test can see | 1.8c | this stage |
| Contract item 4 `internal/render/view_test.go`: the audit's one genuinely unheld row — `playOriginV` comes from the *screen*, not the house rect, and nothing had ever said so | 1.8c | this stage |
| The fidelity contract audited row by row — `ORIGINAL_GAME.md` §19.1, twenty citations and five written exceptions | 1.8c | this stage |
| 2.23 Player 2 gets the abandon key, as the fourth opt-in fix `fixes.player2_give_up` | 1.9 | this stage |
| 2.22 (characterised) The mismatched-exit deadlock: Delete *does* break it, transits deadlock on two objects of the same kind, and 60 refused frames prove it never resolves | 1.9 | this stage |
| 2.16 (measured) The arrival freeze hits whoever is not `FirstPlayer` — which before anybody waits is *player 1*, not player 2 | 1.9 | this stage |
| The three race strictnesses pinned in `game/twoplayer_test.go`: geography refuses audibly, transits refuse silently and demand the same object, the manhole does not race | 1.9 | this stage |
| The shared inventory and the shared throttle pinned as one counter each — one thruster in two-player sounds every frame, not every fourth | 1.9 | this stage |
| `gameFixes` extracted and `TestEveryOptInFixIsCopiedToTheGame` written, because the field-by-field copy's compile-error claim was false | 1.9 | this stage |
| 4.3 (the second script) `testdata/two.script` — a whole two-player game replayed to a golden trace, plus a companion test that reads the same run as sentences | 1.9 | this stage |
| The trace grew a nine-column two-player tail that a one-player trace does not carry, so every golden on file stayed byte-identical | 1.9 | this stage |
| `scripts/bootstrap-dev-env.sh --source system\|internal\|public\|auto`, `make doctor`, `fmt-check` inside `check`, `make cross` over all six targets, and `internal/module`'s stdlib-only assertion | public build path | earlier |
| `internal/saved`: one save per house under `internal/datadir`, written atomically, and the original's five validation gates — the code it wrote out in full and never ran | 1.10 | this stage |
| `internal/house`'s saved-game codec: `game2Type` from offset 6 on, decisions S1–S4 stated where the code is, and the 40-byte block re-encoded byte-for-byte out of all 22 shipped houses | 1.10 | this stage |
| S4 — a house's embedded block resumed with the *house's* timeStamp, without which the original's own gate refuses every game its own writer produced, and Titanic's 1995 save can never be opened | 1.10 | this stage |
| `S` while paused saves, alert 1041's two buttons on `Y`/`N`, and the pause key as a third answer the alert did not have | 1.10 | this stage |
| "Open Saved Game…" on the title screen — MENU 129's third item, on the `O` it had — greyed with the reason, plus `-resume`, `-saves`, and both flag contradictions refused before a window opens | 1.10 | this stage |
| 2.66 `menuRect(n)` derives the menu panel's bottom edge, so the rows cannot fall outside the box they sit in | 1.10 | this stage |
| 2.67 `World.SetPauseHint` erases the old hint row before changing it, so a shrinking hint leaves no tail | 1.10 | this stage |
| 2.70 `Shell.status()` orders the status band's three sources, so the resume message cannot be silently overwritten | 1.10 | this stage |
| 5.6 The fresh-checkout audit: a clone is playable with no network and no copy of Glider PRO, proven by driving the README from a scratch directory | 1.10a | this stage |
| `make smoke` (and so `make check`) no longer fails on a clone that has a display but no extracted houses — the first command the README gives a stranger | 1.10a | this stage |
| `-house Titanic` resolves against `-houses`, so a bare name works from anywhere and not only from the directory holding the file | 1.10a | this stage |
| `make cross`'s host cgo build can fail the target again: its exit status was being overwritten by a later `printf` | 1.10a | this stage |
| `make headless` clears its output directory first, so a shot from a previous run cannot be read as work just done | 1.10a | this stage |
| Every reference to the private repository this was developed inside is gone from the tree, and 46 absolute home paths across 21 documents are now repo-relative | 1.10a | this stage |
| `scripts/bootstrap-dev-env.sh`'s private mirror is opt-in rather than built in, and `deb_arch()` translates `uname -m` instead of answering `amd64` | 1.10a | this stage |
| 1.3 `CHANGELOG.md`: one section per stage, each naming the commit that closed it, under `Unreleased` because there are no tags | 1.10a | this stage |
| The About box and the no-assets title screen say where the art actually comes from — it ships with the source, undecoded — instead of "your own copy" | 1.10a | this stage |
| `.gitattributes`: `* -text`, so a Windows checkout cannot rewrite `Glider PRO.r`'s 199,843 LFs and silently change the extractor's input hashes | 1.10a | this stage |
| `glidergo -version` prints the build, the compiled-in backend, the Go that built it, the platform, and whether each asset tree is there | 1.10a | this stage |
| 5.7 and 5.8 written: the conventional repository files as a dated decision, and seven audit findings deliberately left alone with the reason each | 1.10a | this stage |
| 1.2 (route (a), the decision) `assets/extracted/` committed — 1,877 files, 15.5 MB — so a clone plays with no extraction step, no python3 and no copy of Glider PRO | 1.10b | this stage |
| The 24 MB of `houses/*.rsrc` left out, because they are the one thing in the extracted tree that nothing reads at run time: an intermediate `extract_house_art.py` turns into `houseart/` | 1.10b | this stage |
| `make assets-check` upgraded from a manifest comparison to a full recursive diff, after proving the extractor byte-for-byte reproducible over all 1,899 files | 1.10b | this stage |
| `.gitattributes` marks the tree `linguist-generated` and `binary`, with the three manifests exempted, so a diff of authored code is not drowned in 1,877 generated files | 1.10b | this stage |
| CI refuses a checkout whose assets are missing instead of extracting them, and a separate `assets` job holds the committed tree to `make assets-check` | 1.10b | this stage |
| The About box, the credits, the README and the no-assets screen all say the data ships here — the fourth rewrite of that sentence, and the first one that is true of a clone | 1.10b | this stage |
| `internal/credits`' three transcription tests skip rather than fail when `GliderPRO/` is absent, so a checkout with the 1994 source deleted still reaches a green `make check` — proven by doing it | 1.10b | this stage |
| 2.71 The external audio player's stderr is prefixed with the player's name, so an unreachable sound server no longer prints an unattributed `error:` between two of the game's own lines | 1.10b | this stage |
| 5.8 (the module-path item) `module github.com/bwenstar/gliderGo`, with two new cases in `internal/module`'s table that are only discriminating once the path has a dot in its first element | 1.10c | this stage |
| The private mirror is out of the tree entirely: `--source local` reads an optional gitignored hook, and no doc, script or comment names anyone's internal network | 1.10c | this stage |
| The author email in all 42 commits' metadata is a personal address — a leak that no content grep could have found, since it is in the objects rather than the files | 1.10c | this stage |
| Four commit *messages* named the mirror by product name, found by grepping `git log` rather than the tree, which is the same class of leak one level out | 1.10c | this stage |
| The Makefile reads `GLIDERGO_TOOLCHAIN_DIR` rather than hard-coding `~/.local/opt`, so it cannot disagree with the bootstrap about which Go was just installed | 1.10c | this stage |
| 4.1 `internal/house/lint.go`, `glidertool house lint` and `glidertool house checks`: 29 checks, severities argued down against the 22 shipped houses until they produce 634/48/**1** | 2.0 | this stage |
| 2.14 (the linter half) and 2.49 (the reporting half) — `sound-id` split from `sound-unreadable`, so the 7 findings that are gliderGo's MACE gap do not read as 7 authoring mistakes | 2.0 | this stage |
| `internal/house/lintcatalogue_test.go` reads `lint.go`'s AST, so a check cannot ship without a row in the table a report sends the reader to | 2.0 | this stage |
| `internal/game/linkagreement_test.go`: the two "which objects carry a link" implementations pinned to each other over all 144 object codes, so the `Objects.c` transcription and the linter's map cannot drift | 2.0 | this stage |
| `internal/render`'s duplicate `ExtractFloorSuite`/`GetRoomNumber` deleted in favour of `internal/house`'s, and the dead `kNumUndergroundFloors` beside them | 2.0 | this stage |
| The pre-2.0 link packing cannot express suite ≥ 100 — it *collides* — written into `house-format.md` §7.2 and asserted rather than avoided | 2.0 | this stage |
| Two documents claimed every shipped house has a `kStar`; Fun House has none, which is why `no-stars` is a warning | 2.0 | this stage |
| 4.14 `internal/shell/sets.go`: a level set is declared by the source a house was walked from, so the claim is "where this was found" and never "what is in it" | 2.2 | this stage |
| `Library.Discover` is variadic and accumulates, which `internal/assetfs` deliberately still does not — houses add, art replaces, and 5.3's deferred union lands in exactly one of the two | 2.2 | this stage |
| The picker's set strip, on the title's own line because the apparently-empty row below it is the first house's selection bar; one set draws the identical screen it drew before, so `internal/fidelity`'s reference stays a picture of 1994 | 2.2 | this stage |
| 4.14 (the built-in half) The `New` set ships *inside* the executable — `assets/levels.zip` embedded, `go test ./assets` rebuilding every authored house from its text and comparing bytes | 2.3 | this stage |
| 4.16 (the Go half) `internal/replay/profile_test.go` walks a house and holds it to the tier bands in `docs/analysis/original-houses.md` §10.2, instead of the bands being measured by hand | 2.3 | this stage |
| 4.19 A house's `timestamp` and file *name* are the keys its saved games and high scores hang on, pinned by a test carrying the published numbers | 2.3 | this stage |
| 4.20 `glidertool replay`'s `house` line takes a name as well as a path, because a house that lives in the binary has no path to give | 2.3 | this stage |
| `levels/Open House.house.txt`, the first house this port wrote, built from text by `make levels` and lint-clean | 2.3 | this stage |
| 4.16 (the dark-room row) The row declared uncomputable was three lines in `internal/replay`, which already imports both packages the argument said could not be joined | 2.4 | this stage |
| `levels/Boarding House.house.txt`: 51 rooms at the small tier, all eighteen built-in backgrounds, and a per-background histogram pinned because moving one room out of `kRoof` falsified four figures in its own header while every profile row stayed green | 2.4 | this stage |
| 4.21 `lint.go`'s calibration paragraph rewritten around a narrower rule — a class the corpus exercises deliberately cannot be an error; a class it exercises by overrunning a buffer can — with the staircase example dropped because it fires zero times | 2.4 | this stage |
| 4.22 `mount-no-floor`, `mount-no-ceiling` and `starfield-tiles`: the first three checks that relate an object to the background it stands in, classified by a census rather than by the objects' names | 2.4 | this stage |
| 4.13 The documented command lines nobody had run — four in 2.1, the nine that were left in 2.4, and with that the section is closed: the only command line in this repository nobody has run is `gh release create`, which needs github.com and is 5.4's | 2.1 + 2.4 | this stage |
| 4.24 The `'bnds'` layout eight analysis documents state and the port had wrong, which opened the wrong walls in 155 rooms across seven houses | 2.4 | this stage |
| 4.26 The three documents describing `make check`, two of which were wrong; the verbatim copy of its output is machine-checked now | 2.4 | this stage |
| 4.23 `object-top` and `object-left`: 24 object types whose coordinates are `#define`s in the 1994 editor rather than choices an author was offered, from a table of citations, with the corpus's 4,041 placements as the stronger half of the test | 2.4 | this stage |
| 4.27 The unreachable repair in `KeepObjectLegal` and the 43 floor transporters that are 2 pixels low because of it — filed, accepted as a value rather than a tolerance, and deliberately not normalised on load | 2.4 | this stage |
| 5.2 `tools/extract_all.py` publishes by rename: a staging tree, an OS advisory lock, and the eleven counts checked *before* the rename, so a cancelled or a miscounting run publishes nothing | 2.4 | this stage |
| 4.28 (the first half) `internal/shell/race.go`: a Race screen on the title menu, so the mode that was reachable only from a shell is reachable with the keyboard already in the player's hands — one `shell.Race` built by both the screen and the flags, and a ninth menu row whose geometry the splash artwork's own pixels decided | 2.6 | this stage |
| 2.72 `x11.New` asks the server for XKB's detectable auto-repeat, so `Event.Repeat` is set on Linux and a held key does one thing, as it already did on Windows — and `internal/platform/x11` has its first test, which asks the server rather than the package | release gate, step 1 | this stage |
| 2.73 `RenderShreds` asks for the `shred` strip rather than a sheet, so a shredded glider falls as confetti and the game no longer ends with an asset error — with a replay test that shreds a glider with the art loaded, and a static test that holds every constant art name to the table its accessor reads | release gate, step 1 | this stage |
| 2.79 `prefs.Save` writes back the keys it has no field for, so an older build that saves a newer build's file no longer deletes the newer build's settings | release gate, before the tag | this stage |
| 2.80 An unreadable settings file is reported once, and the Windows auto-scale check names `-prefs none` rather than a directory | release gate, before the tag | this stage |

Five bugs found and fixed in the port itself while writing this, none of which is an
"improvement" so much as a repair, all recorded here because the reason no test caught
them is worth keeping:

- **`render.bgrxLUT` was built from a zero-valued `Palette`.** Go runs every package-level
  variable initializer before any `init()` and orders them only by dependencies visible in
  the initializer *expressions*; `Palette` has no initializer expression, so a
  `var x = f(Palette)` had no dependency edge, ran first, and read 256 zero entries. Every
  LUT entry came out `0xFF000000`, so `Surface.ToBGRX` returned an opaque black image for
  any input — and nothing but the real host path calls it, while an all-black frame is
  indistinguishable from an uncomposed one. Fixed by deriving the table inside
  `palette.go`'s `init`; guarded by `TestBGRXLUTMatchesPalette`.
- **`playTestWorld` budgeted presents, not frames.** See 2.4. `cmd/glidergo`'s `-frames`
  flag had the same bug and is fixed the same way; the mistake is easy to make twice because
  `Present` is the only per-frame hook a host owns, so it reads like a frame counter.
- **`HandleRewards` was missing the `kHelium` arm entirely** (1.5d;
  `Interactions.c:956-977`). A transcription slip, not a decision — the arm was dropped
  between `kStar` and `kSlider`, and nothing complained because a helium balloon is a
  `bonusType` like any other, so it built and ran and simply did nothing when touched. What
  identified it as a slip rather than a deferral is that two comments in the same file already
  counted *ten* restoring arms and *five* doubling arms, and both figures only hold with helium
  present. The recovered arm is the battery's mirror with three sign inversions — the `< 0`
  test, the `= -HeliumSupply` assignment, and a doubling that *subtracts* — and
  `TestBatteryAndHeliumShareOneSignedCounter` exists because getting any one of them wrong
  hands the player thrust where the balloon should have given buoyancy.

  The lesson for the rest of the port is the test that found it, not the bug.
  `TestEveryDispatchableRewardIsConsumed` does not list the reward types: it sweeps every
  object code through `CreateActiveRects`, collects the ones that produce a `kRewardIt` rect,
  and requires `HandleRewards` to consume each. A hand-written list would have had the same
  hole as the switch statement. Any other port function that dispatches on object type is a
  candidate for the same treatment.
- **`World.NewState` was a local, so every switch lever showed the wrong state** (1.5d).
  `SetObjectState` computes the state it writes into a file-scope global that `HandleSwitches`
  reads back to draw the plate, which is what makes a light switch show the *lamp's* new state
  rather than its own — a switch has no state of its own to show. With the value local, every
  lever drew from a zero, so a switch animated to "off" while turning a lamp on. Invisible
  without art, which is why `TestLeverShowsTheLinkedObjectsState` renders the plate twice and
  compares pixels, with a second art-free test on the field for runs where the assets are not
  extracted.
- **The mixer kept asking a finished game for the next piece of music** (1.7d's follow-on).
  `Engine.NextPiece` was assigned in exactly one place, `bindAudio`, and never re-pointed, and
  `NewGame`'s teardown ends every game by starting the idle score. So the music a player heard
  over the title screen was being walked by the `World` of the game they had just quit — and
  every game of a session stayed reachable from the engine and could not be collected. It was
  inaudible, which is why it survived: the cursor is a cursor wherever it lives, and the score
  sounded exactly right. What made it visible was writing the title screen's own score walk and
  having to ask what the engine was pointing at, which is the general shape of the thing —
  *nothing was wrong with the audio; something was wrong with who owned it.* Fixed by
  `startTitleMusic` re-pointing `NextPiece` at the title screen's cursor on the way back out of
  a game (`cmd/glidergo/music.go`), which is also what makes `music_on_title` audible at all.
