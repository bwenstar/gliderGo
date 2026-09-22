# Changelog

All notable changes to gliderGo, newest first.

`v0.1.0` and `v0.1.1` have been tagged and published, but this file has not yet been split into
version sections, so everything below is still under `Unreleased` — the entries are the work, and
which tag happened to carry it is in `git log`. The version a build reports is `git describe
--tags --always --dirty`. See `docs/IMPROVEMENTS.md` 5.4, which owns tagging and the release
pipeline.

Because Stage 1's whole goal was *"behave exactly the same as the original, only newer"*, this
file records stages rather than features, each naming the commit that closed it. Where the port
knowingly departs from 1994 the entry says so and points at the numbered item in
`docs/IMPROVEMENTS.md` that owns the deviation. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) loosely; it does not follow semantic
versioning yet, because nothing has been versioned.

## Unreleased

### Open House: the first house of our own, and the sixty pixels that decide what a house can ask for (2026-09-22)

`levels/Open House.house.txt` is 43 rooms on a 7×10 grid, the first house this port wrote. `make
levels` compiles every `levels/*.house.txt` with `glidertool house build` and lints the result, and
`glidergo -levels assets/levels` puts them on the shelf as the **New** set — where it lists as "Open
House · 43 rooms" between "Nemo's Market" and "Rainbow's End", because a house is named by its file
and `houseType` has no name field.

**Written against measurements, not vibes.** `docs/analysis/original-houses.md` §10.2 is a table of
fifteen numeric targets across five size tiers, derived from all 4,070 shipped rooms; this is a
tutorial-tier house and thirteen of those rows are inside the band: 43 rooms (35-45), 7 floors, 10
suites, grid density 0.614 (0.6-0.65), 133 objects (78-138) at 3.09 a room (2.2-3.1), 9 empty rooms
at 20.9 % (~20 %), nothing near the 24-object ceiling, 2 enemies (0.0-0.1/room), 11 prizes
(0.03-0.29/room), one star, 26 distinct object codes (15-48), 11,700 points (8,500-12,500).
`TestOpenHouseMatchesTheTutorialProfile` asserts all of it, so the table in the file's header comment
is an invariant and not a boast — add four objects to make a room look nicer and the test names the
row that left its tier. Two rows are not checked because nothing here can compute them, and both are
filed with the reason (`docs/IMPROVEMENTS.md` 4.16).

**The design turned on an arithmetic error that the simulation caught.** The house was laid out
believing a glider that flies through a floor vent leaves it near the ceiling. It does not. The lift
column is 4 px wide with a 52-px catchment, so a glider crossing at cruise (5 px/frame) is inside it
for about ten frames and gains **60 px** — while `Gravity 3` against `NormalThrust 5` spends 0.6 px
of height on every px travelled. Break-even is a vent every 100 px. Two vents in a 512-px room lose
roughly 100 px of altitude per room, and a trace of the arrow held down dies in the third room.

That is not a flaw in the house and it is not a flaw in the port: **it is the game.** You release the
key, let the column carry you to the ceiling, then press and glide. The originals average 1.49
blowers a room, so they are more dependent on it than this house is, and a house that could be flown
with one key held down would have removed the mechanic. So the layout stayed and the *documentation*
absorbed the correction — the figure is in the house's header, in the keystroke script, and in the
`make levels` output, because it is the one thing a new author will get wrong first.

**Completable, and by a test rather than an assertion.** `internal/replay/testdata/open-house.script`
flies it from the Furnace Room to the star in the Belfry — six staircases, ten rooms, 1,165 frames,
no glider lost — and `TestOpenHouseCanBeFinished` builds the house from the checked-in *text* before
flying it, so a stale `assets/levels/` cannot pass for a source that no longer works and a fresh
clone needs nothing but the Go toolchain. `make check` gained a `levels` step, which means the house
is lint-clean on every run: 43 rooms, 0 notes, 0 warnings, 0 errors.

Two deviations from the recipe, stated because they are choices and not oversights. The background
mix is 72 % interior / 7 % `kDirt` / 21 % air against §10.3's 44 % house-carried art, because a house
of ours has no way to carry art yet — `-levels` adds a root and `-houseart` replaces one, so giving a
new house its own pictures takes the originals' away (4.15). And §10.3's per-room object budgets are
corpus means that no tutorial-tier house can satisfy; the interiors hold 3-5 rather than 11-13, and
the contradiction is filed against the document (4.18). Not verified: **nobody has played it by
hand.** A scripted run proves a route exists; it says nothing about whether a person would find it,
and this host has no way to ask.

### Level sets: Original, New, and a set that says where a house was found rather than what is in it (2026-09-22)

Stage 2 adds houses, and the picker they land in is one flat alphabetical list. That is faithful —
`BuildHouseList` produces exactly that — and it is also a list that would put "Bakery" between
"Asylum Pro" and "CD Demo House" and let it read as something Jonathan Chin or Ward Hartenstein
built in 1994. Wrong in both directions: it takes credit from the five people `docs/IMPROVEMENTS.md`
1.2 names, and it lends the originals our mistakes, because a room the glider cannot leave is a bug
in a 2026 port and evidence about the 1994 game if you believe 1994 wrote it.

**A set is declared by the source a house was walked from.** Not by the file — `houseType` has no
field for it, the 866-byte header is full, and inventing one means writing a house the 1994 program
cannot open. And not by a list of the twenty-two names, which is the tempting answer and wrong
twice: it puts the truth about the shipped set in a second place that can drift from
`assets/extracted/houses`, and it answers confidently in the one case that matters, because a house
opened in an editor and saved back as "Slumberland" is not the 1994 Slumberland and the name is
exactly the part that did not change. A provenance claim that cannot fail is not a provenance claim.

So what a set claims is deliberately narrow: **where a house was found, not what is in it.**
`-houses some/dir` lists as `Other`, because the game cannot know what a player put in a directory;
the built-in root is `Original`; the new `-levels DIR` is `New`. The picker prints each root's label
beside its count, so the claim is always checkable against the place it came from.

`Library.Discover` now takes any number of sources and walks all of them into one sorted list, which
is the union 5.3 deferred to this stage — and it lands in the houses and not in the asset roots,
because an art root has to *replace* the built-in one (two copies of PICT 1000 must resolve to one
picture) while houses accumulate: four of your own and the twenty-two is twenty-six. A source that
cannot be read is an error *and the rest are still walked*, so a mistyped `-levels` names itself on
screen and still leaves a list to play from. Accumulating is not the new part, incidentally: the
original already merged up to eight dropped-on-the-application houses with its folder scan
(`SelectHouse.c:650-664`) and simply never said which row came from where.

On screen: a strip of set names with their counts on the picker's title line, the one showing in
inverse video, **Tab** to cycle, the set in the footer when the list is mixed, and the status band
reading `24 houses: 22 Original, 2 New`. Three decisions in there are not obvious and are argued in
4.14 — the filter opens on *All* rather than on the selected house's set, because a fresh install
selects Slumberland and defaulting to its set would hide every new house from the player who has not
yet found the chooser; the cursor stays an index into the whole library, so changing the filter
changes what is drawn and never what is chosen; and Tab was taken rather than found, having been an
undocumented second Escape here and an undocumented alias of the right arrow in 1994.

The strip sits on the title's own baseline because the 32-row gap below the title is not empty: the
first house's inverse-video selection bar rises into it and leaves six free rows where a scale-1 word
needs ten. That was got wrong first, measured against the first row's glyph ink instead of its
selection bar, and caught by a test counting 2,602 cream pixels that a one-set library must not draw
— the wrong measurement is now in the comment on `pickSetsV` so the next person to want a row there
learns why there isn't one.

**With one set on the shelf the picker draws exactly the screen it drew before**: no strip, no title
suffix, no `Tab set` in the footer. That is what keeps `internal/fidelity`'s reference image a
picture of the 1994 dialog, and it is asserted both ways — a one-set library renders an identical
whole screen whichever set it is, and `make fidelity` passes with `screens.hashes` and the golden
`houses.png` untouched.

Where the new houses that ship *inside* the executable will live is filed rather than pre-built:
`assets/extracted/houses/` is barred by two contracts that exist for good reasons (`make
assets-check`'s recursive diff and `assetpack.Compare`), `//go:embed` cannot reach outside `assets/`,
and the recommendation is a second archive packed by the `tools/packassets` that already exists. See
4.14. Until then `-levels DIR` is the whole of the New set, which is enough to author against.

### The receipts, checked: 17,800 citations into the 1994 C, and the documents that were wrong about this project (2026-09-22)

This port's whole claim is that it behaves like the original, and the claim is made in pieces — about
17,800 pointers at a file and a line of John Calhoun's C, naming about 18,300 lines across all 93
files of upstream `94fed96`. Nothing had ever checked that one of them resolved.

`go test ./internal/citations/` now does. Three classes of claim and one invariant: the 1994 C in four
tiers (the file exists; every line and range endpoint is inside it; a full path names the directory
the file is really in and no range runs backwards; coverage both ways, so all 67 of upstream's `.c`
files are cited somewhere and every exemption is still needed), this repository's own `path:line`
references, the test names the prose quotes — and the thing the other ten rest on, that all ten places
the upstream commit is written name the same commit. A `citations` job in CI clones upstream at the pin
on every push and **fails if any check skipped**, because `go test` prints `ok` for a package whose
every test skipped. [`docs/CITATIONS.md`](docs/CITATIONS.md) is the reader-facing half, including a
section on what a green run does *not* promise.

**It found twenty-two broken citations** on prose that had been reviewed repeatedly: nine line numbers
past the end of their file, six file names that do not exist, one path naming the wrong directory, two
references to documents of ours that are not there, four test names renamed out from under a comment.
Every out-of-range number was fixed by reading the C and finding the line the claim was about, not by
lowering the number until it fit — a citation that resolves to the wrong place is worse than one that
resolves to nothing, because nothing announces itself. Two implied more than they cost: `Room.c` cited
to 1215 in a 1206-line file means that reading was done against a differently converted copy, and
`PlayerControl.c` is a file an early draft invented which has never existed in any version of Glider
PRO. A twenty-third was not a citation defect but caused four — `GliderPRO/Prefix.h` sits at the top of
upstream's tree rather than in `Headers/`, so it was missing from the copy recipe in every document,
and the four citations to it could not resolve on any machine that had followed the instructions.

The documentation's own count was wrong by nearly half: five places said "about 9,500", which is
roughly the full-path form alone. This also built the linter `docs/IMPROVEMENTS.md` 4.4 had been owing
since Stage 1, and 4.4's prediction that its one false positive would resolve itself did not come
true — `TestHighScore` became `World.TestHighScore`, a *method*, which a matcher looking for
`func Test…` cannot see.

**A companion sweep over the instructions this project gives a person**, which had never been run
either. Four defects, each failing differently:

- **`glidertool replay -script -` could not run**, though replay's own `-h` offers it as the way to
  see the script format. The house check sat above the write-back. It survived six stages because a
  *test asserted the broken behaviour* — `TestReplayNeedsAHouse` used `-script -` as a cheap way to
  reach the check, and so pinned it in green; it is now `TestReplayNeedsAHouseToRun`. The printed
  template names a house and replays.
- **`glidergo <house>` was silently ignored.** `flag.NArg()` was never read, so the most obvious
  command a player can type showed the title screen — indistinguishable from the house being refused.
  Now accepted, because a house was one of the 1994 Mac application's *documents* and because that is
  the form an OS uses for a file association; two houses, or `-house` and a bare name, are refused
  with both strings quoted back.
- **`CGO_ENABLED=0` builds a game that cannot be quit.** `CONTRIBUTING.md` said the build "still
  succeeds and produces a binary that cannot draw"; it does not succeed (`pkg-config` stops it), and
  the binary a no-cgo build *does* produce selects the null backend, which can never deliver
  `EventQuit` — so it drew frames into nothing until Ctrl-C. Now refused, naming the three flags that
  give such a build an end, and the prose says what actually happens and points at `make doctor`.
- **"is DISPLAY set?" answered one of three questions.** `platform.DisplayAdvice` now distinguishes no
  session, a refused connection, and a Wayland session with no XWayland — the last being the likeliest
  on a 2026 desktop and the one the old sentence sent furthest wrong.

`README.md`'s map gained `internal/shell`, `internal/audio` and `internal/citations` — the first being
everything a player sees before a room is composed — and a test now fails if any `internal/` package is
missing from it. `CONTRIBUTING.md`, `SECURITY.md`, three issue forms and a pull-request template landed
(`docs/IMPROVEMENTS.md` 5.7); `CODE_OF_CONDUCT.md` deliberately did not, and 5.7 records why the
reporting address turned out to be the wrong blocker for one of those and the right one for the other.

### A house linter, calibrated against the 22 houses it has to tolerate (2026-09-21)

Stage 2 authors new houses, and the first thing that needs to exist is something that reads a house
and says what will not work. `glidertool house lint` is that; `glidertool house checks` prints the
table of what each finding means. Twenty-nine checks at three severities, covering the links (a
transporter whose destination room does not exist, a link to an object slot that is empty or past
the 24 a room holds, a transit object with no destination at all), the staircases (a floor with no
room on it, a destination with no counterpart to arrive on), the art (a background, a `kCustomPict`
or a `tiles[]` column that resolves to nothing), the sounds, and the header arithmetic. `-min`
chooses what prints, `-check` filters to named checks, `-fail` turns the worst finding into an exit
status, and `-no-assets` skips the art and sound checks while saying in the report that it did.

The interesting work was not the checks, it was the severities. `house lint` is only worth putting in
CI if the exit code means something, and it is only trustworthy if it tells the truth about the
originals — so each check was written, run over all 22 shipped houses, and then argued down to the
level those houses justify. The corpus produces 634 notes, 48 warnings and exactly **one** error.
189 dangling links cannot be errors when the 1994 houses carry that many. Slumberland's inescapable
basement is the original's design (see `acafec7`), so a staircase with nothing to arrive on cannot be
an error either. The one error is the out-of-range object slot in CD Demo House room 72 that
`docs/analysis/` has documented since Stage 1.

**One check turned out to be blaming the author for a bug of ours.** `sound-id` fires when a
`kSoundTrigger` names a `snd ` resource the house does not carry, and it found 20. Checking them
against the sound manifest row by row split them 13 / 7: thirteen real ones (In The Mirror names
`snd ` 10000 twice; Teddy World names `snd ` 3000 eleven times and ships no `snd ` resources at all),
and seven that are the five MACE 6:1 resources gliderGo's extractor cannot decode
(`docs/IMPROVEMENTS.md` 2.49, open since Stage 1). Telling an author their sound is missing when it is
sitting in their resource fork and played fine in 1994 is worse than saying nothing, so that is now a
separate check, `sound-unreadable`, whose text says out loud that the gap is ours. Both findings also
say the part that matters more than the silence: a trigger whose sound fails to load gets **no hot
spot at all**, so it cannot be touched, and a house using one as a signpost has lost the signpost.

Two checks were deliberately not written, with the reasons recorded in `docs/IMPROVEMENTS.md` 4.1.
Static reachability — "can the player get from the first room to a star" — would be guesswork,
because `Room.Openings` is a dead field in all 4,070 rooms and the real openings are computed at run
time; reachability belongs to Stage 2's scripted playthrough, which answers it by playing. And
"objects outside their room" needs `GetObjectRect`'s per-type geometry, which lives in
`internal/render` where `internal/house` cannot reach it.

The catalogue is held to the code rather than maintained beside it.
`internal/house/lintcatalogue_test.go` parses `lint.go`'s syntax tree, collects every `l.add` call
site's check id and severity, and requires that the table lists exactly the ids the code can emit
and declares for each the worst severity any call site uses. A behavioural test would only have
proved the ids it happened to trigger exist.

Three repairs fell out of writing it. `internal/render` had its own copies of `ExtractFloorSuite`
and `GetRoomNumber`; they delegate to `internal/house` now, and the dead `kNumUndergroundFloors`
beside them is gone. `internal/game`'s link predicates and `internal/house`'s are pinned to each
other across all 144 object codes by `internal/game/linkagreement_test.go`, so the transcription can
keep its `Objects.c` comments and the map can keep its lookup without the two drifting. And two
documents claimed every shipped house has at least one `kStar` — Fun House has none, which is
precisely why `no-stars` is a warning rather than an error.

Also written down: the pre-2.0 link packing cannot express suite 100 or above, because the version-1
layout puts the suite in the low two decimal digits. It is not just unreachable, it *collides* —
`MergeFloorSuite(-7, 100)` and `MergeFloorSuite(-6, 0)` are the same `short` — which is very likely
why version 2.0 swapped the two fields. `docs/analysis/house-format.md` §7.2 and a test that asserts
the collision rather than avoiding it.

### The Windows backend has been run by somebody (2026-09-21)

Every release so far shipped two Windows archives under a caveat in bold: *"the Windows code has
never been run."* It was true. `internal/platform/win32/` and `internal/audio/waveout_windows.go`
were written on an offline Linux machine with no Windows on it, and CI compiling them on a runner is
not the same claim. That caveat is now retired for `windows/amd64`, and kept for `windows/arm64`.

`v0.1.1`'s own `windows-amd64` binary was run on a Windows Server 2025 desktop (build 26100): six
`-shot` renders, five unpaced bench runs, and one paced 600-frame run that held 29.9 fps against the
original's 30.07 target. 4,320 frames in total, from a directory containing nothing but the `.exe`.

The result worth the entry is that the pixels are **byte-identical to Linux's**, and that this was
checked on the actual desktop and not only in a file. All six title-screen renders hash the same on
both platforms. More to the point, a screenshot taken by the operating system off the running
window, cropped to its client area, is an exact pixel-for-pixel match for a frame the Linux build
renders — which is the only way to prove the blit happened, because `Present` deliberately does not
check what `StretchDIBits` returns, so a clean exit and a good frame rate are equally consistent
with a window that never painted. Sound: `-audio list` found `waveout`, and across the four runs
that used the device rather than a WAV file the game asked for 73 sounds and the driver played 73,
refusing none.

Two things that looked like platform bugs and were not, both written up rather than quietly
dropped. The harness's last step checked for files in `%AppData%\glidergo` and found none, which
was the harness being wrong and not Windows: `-house` with `-frames` goes through `playDirect`,
whose own comment says it saves nothing, and a Linux run with a fresh `HOME` creates nothing
either. That branch of `internal/datadir` was then exercised properly with `-import-prefs`, which
did create `…\AppData\Roaming\glidergo\prefs.json`. And the Windows and Linux `-wav` captures
differ in about 124,000 byte positions — but two *Linux* runs of the identical command differ from
each other in about 122,000, so it is the mixer's wall-clock timing rather than anything about the
platform, and neither count is repeatable twice running. A frame-locked
mixer clock would make `-wav` reproducible enough for CI to diff audio the way it already diffs
pixels; that is now `docs/IMPROVEMENTS.md` 4.11.

What it did not cover, because the release notes now point at this: no key was ever pressed, so
`internal/platform/win32/keys.go` is still the least-exercised file in the package on the platform
it exists for; nothing touched the window itself (no resize, focus change or close); and
`windows/arm64` has never executed at all. `docs/windows-first-run.md` is the full write-up, with
the six hashes and the commands to reproduce them. The honest-caveat headers in `win32.go`,
`keys.go` and `waveout_windows.go`, the release-note block in `.github/workflows/release.yml`, the
archive table, the `Makefile`'s per-target labels and the README all now say what is true instead
of what was true.

### The test suite had learned to spell paths the way Linux spells them (2026-09-21)

The first CI run on a machine that was not the development host failed, in the only job that had
never run anywhere before: `go test ./...` on `windows-latest`. `go build` and `go vet` passed on the
same runner.

Two tests were responsible, both with the same defect and neither depending on anything about the
runner. `internal/scores` set `GLIDERGO_CONFIG` to `/tmp/glider-portable` and compared it against a
directory that had been through `filepath.Join`, whose `Clean` rewrites every slash as a backslash on
Windows — so the test failed over its separators rather than over the thing it was checking. And
`internal/shell` expected `House.Rel` to equal `filepath.Join("sub", "Nested.glh")`, when `Rel` is an
`io/fs` name that `fs.WalkDir` builds with `path.Join` and is slash-separated everywhere; the
assertion's own error message had said `want sub/Nested.glh` all along. The production code was
correct in both cases. See `docs/IMPROVEMENTS.md` 4.10.

Folded in: `internal/audio`'s waveOut test wrote twelve frames into a queue eight deep and asserted
none were dropped, which was only true when the pump goroutine won a race. It now writes exactly
`waveDepth` frames, which cannot drop and still forces the device to hand a block back. That test
skips without an output device, so it was never the CI failure — it was the next one, on the first
real Windows desktop to run the suite.

Also: a failing `go test` in that job now puts the `FAIL` lines on the run's summary page and keeps
the full transcript as an artifact, because this failure had to be diagnosed from an exit code and an
expired log. The step is still allowed to fail the job.

### Arithmetic that cannot differ between an x86 and an Apple Silicon machine (2026-09-21)

Two functions computed a float `a*b + c`, which the Go spec lets a compiler fuse into one
instruction that rounds once instead of twice. amd64 codegen does not take that licence; arm64 does,
and `macos-latest` is arm64, so the CI matrix is the first arm64 toolchain this port has met.

- `FillPolyPatOrGray`'s scanline crossing is now exact integer arithmetic. The vertices are integers
  and the row centre `y+0.5` doubles to an odd integer, so the crossing is a small rational and the
  rounding to a pixel column is an integer division. `ceilHalf(float64) int` becomes
  `ceilDiv(n, den int64) int64`. This matters because the crossing is then `ceil`ed: a last-bit
  difference on a boundary moves an entire column, and four callers in `objectdraw.go` draw furniture
  shadows into the background of nearly every room in the game.
- `audio.stepFor` keeps its float64 and gains a conversion around the product, which forces the
  intermediate rounding. A sample rate on a boundary would otherwise resample by one 65536th on one
  architecture and not the other, and every mixed sample after it would differ.

No golden changed, and that is deliberate rather than lucky-sounding: arm64 was emulated by patching
each site to call `math.FMA` and re-running the pixel and audio suites, and every digest matched. So
this was a latent difference, not an active one — worth closing because a suite that hashes pixels
and audio bytes should be architecture-independent *by construction*, not by the accident that no
shipped house draws a polygon on a boundary. `docs/IMPROVEMENTS.md` 4.9 records the finding, the
`GOOS=darwin GOARCH=arm64 go build -gcflags=-S | grep FMADD` recipe that found it, and the standing
check that is still a note.

Also in CI: every action reference is bumped past the Node 20 runtime that GitHub has deprecated —
`checkout@v5`, `setup-go@v6`, `upload-artifact@v5`, `download-artifact@v5`.

### Slumberland's basement is a trap on purpose (2026-09-18)

A player went down the stairs in Slumberland and could not get back up, and asked whether the port
had lost a staircase. It had not. Both basement staircases are entered through a 112x32 box at the
top of the flight, the only lift in either room that reaches that box is a four-pixel updraught
standing directly under it, and both of those vents are authored `initial 0` — off at the start of
every game, because `SetObjectsToDefaults` copies `initial` into `state` (`Play.c:601-660`). Each is
switched on from another room: "Good Night"'s from the thermostat in "Anabell Lee" next door, which
is reachable from the basement, and "Going Up? No?"'s from the one in "Switch Me", two floors up in
another suite, which is not. The room names are the authors' own commentary — "Going Up? No?", "How
do you get out of here?", "Don't Ask, Just Get Out!".

No behaviour changed. What is new is `internal/game/basement_test.go`, so that the next reader to
meet the basement finds an argument instead of repeating the investigation:

- The data half rebuilds both trigger boxes and both vent columns from the shipped house and asserts
  the column ends inside the box, that **no other blower in the room reaches it**, that the vent is
  off both as authored and after `SetObjectsToDefaults`, and that the switch which opens it lives in
  the room the house says it does and sends `Toggle`.
- The flying half puts a glider on the column in "Good Night" twice. With the vent as the house
  ships it, the room never changes and the glider is lost. With the vent toggled — the poke the
  thermostat next door sends — the same placement rides it into the staircase and arrives in "Choose
  Me" on the floor above, in `GliderComingUp`, at the rect derived from `GetUpStairsRightEdge` rather
  than from this port.
- `docs/IMPROVEMENTS.md` 3.6 records the two things this *does* argue for and neither is a fidelity
  change: the Stage 2 house linter (4.1) should compute exit reachability, so the port can answer
  "is this a bug?" for any house without anybody reading object tables by hand; and telling the
  player anything at all belongs to 3.2's decision about whether the port may be kind.

### Windows makes a noise on its own (2026-09-17)

The Windows archives drew and were silent, because the port had no audio driver at all: it encodes
the mix as raw PCM and pipes it to whichever command-line player the machine has, and Windows ships
none of the five it knows. The advice in the release notes — install FFmpeg or SoX — was true and was
not good enough for a game. `docs/IMPROVEMENTS.md` 2.48 is now closed on Windows.

- `internal/audio/waveout_windows.go` is a winmm `waveOut` sink in pure `syscall`: mono s16le at
  22255 Hz through `WAVE_MAPPER`, which resamples, so the odd rate the 1994 samples want is not the
  device's problem. Eight blocks rotate, one goroutine owns the device, and `WHDR_DONE` is polled
  every 4 ms — no cgo, no COM, no redistributable, and nothing to install.
- `internal/audio/device.go` is the seam it arrives through. `Open` tries the native device first and
  the external players second and nothing above it learns which it got, which is the job
  `internal/platform/backend` does for the display. Linux keeps the subprocess deliberately: ALSA,
  PulseAudio and PipeWire are C libraries, there is no cgo on the audio path, and a player that dies
  takes nothing with it. macOS at Stage 6 wants a CoreAudio sink behind the same seam.
- **A tenth of a second of silence goes in before the first mixed sample.** The device consumes at
  its own crystal rate while the game produces what the wall clock says is due, so a device with no
  cushion starves on the first hiccup — the difference between sound and sound with clicks in it. It
  is deliberate lag bought as underrun immunity, and the sink does not try to rebuild it later,
  because inserted silence is latency that never comes back. The gaps are counted instead and the
  shutdown report has a fourth number: `4 gaps at waveout`.
- **winmm is loaded by absolute path**, via `GetSystemDirectoryW`. It is not on Windows' KnownDLLs
  list and not in Go's own system-DLL set, so a bare name would search the directory the executable
  was started from first — and the release archive tells the player to run the binary from the
  directory they unpacked it into, which is exactly the arrangement a planted `winmm.dll` wants.
- **The blocks the device reads are package-level arrays.** `waveOutWrite` keeps the pointer it is
  handed and reads that memory from a driver thread afterwards, which is the one case Go's rules for
  pointers passed to foreign code do not cover; a global is the only allocation whose address the
  language cannot ever change. `runtime.Pinner` was tried first and rejected — a Pinner collected
  without `Unpin` panics the process by design, turning a bookkeeping slip into a crash in a shipped
  game — and `VirtualAlloc` second, because reading it back needs a pointer made out of an integer,
  which is what `go vet`'s `unsafeptr` check exists to object to.
- `-audio list` now lists outputs rather than players and names `waveout` first where it exists;
  `-audio waveout` insists on it. A mistyped name is still an error rather than a silent fallback,
  and the message now lists the device alongside the players so the reply contains the spelling.
- `reportAudio` looks inside a `Tee`, which fixes a small blindness the new counter exposed: with
  `-audio` and `-wav` together the drop counter went unreported, so the one run that was recording
  *because* the sound was wrong was the run whose numbers were missing.
- **It has never been run here**, like everything else Windows in this port, and it says so at the top
  of the file. What is checked from Linux is the half where a mistake is silent:
  `internal/audio/waveout.go` holds the two structure layouts, the constants and the MMRESULT table
  with no build tag, and `waveout_test.go` asserts every `WAVEHDR` and `WAVEFORMATEX` offset, the C
  structure's 18 declared bytes inside Go's 20, the alignment the atomic `WHDR_DONE` load needs, and
  the tuning constants against each other. `-audio ffplay` and `-wav out.wav` remain the two ways
  round it.
- **CI runs the rest of it**, which is new: `waveout_windows_test.go` loads winmm out of the system
  directory and resolves all seven entry points on a real Windows, so a misspelt export or a
  mishandled path length is a failed test rather than a silent evening. On a machine that has a sound
  card the same file opens the device and plays twelve frames of silence through the whole rotation —
  prepare, write, poll, reclaim, reset, close — and on a hosted runner, which has no card, that half
  skips and says so.

### The assets ride inside the binaries (2026-09-17)

A downloaded `glidergo` now runs from anywhere, with nothing beside it. `docs/IMPROVEMENTS.md` 5.3
had been open since Stage 0 asking where an installed copy should look for its art; the answer taken
is that it should not look anywhere.

- `assets/extracted.zip` — 1,877 files, 11.3 MB — is committed beside the tree and compiled into
  every executable by `assets/assets.go`. `glidertool` carries it too: a tool that renders houses and
  replays recordings is no use in an archive that no longer ships an asset tree.
- The four asset flags (`-art`, `-houses`, `-houseart`, `-sound`) now default to empty, meaning the
  copy inside this binary; naming a directory replaces that root. `-version` prints which of the two
  each root came from, so a bug report says it.
- **A zip rather than an `embed.FS` over the directory, and this is the load-bearing detail.**
  `go:embed` accepts only names that are valid module file paths, so an apostrophe is out — and three
  shipped houses have one: `Castle o' the Air`, `Nemo's Market`, `Rainbow's End`. Naming such a file
  in a pattern fails the build, which is survivable. Naming its *directory* skips it **in silence**:
  120 files, 350,674 bytes, three houses and three houses' worth of custom art missing from the game
  with nothing anywhere to say so. A zip has no such rule and the 1994 names survive byte for byte
  inside it. `assets.TestApostropheNamesSurvived` is the tripwire, and it does not skip when the tree
  is absent, because its subject is the bytes in the binary.
- The pack is deterministic — names sorted, every timestamp fixed at 1994-10-01 UTC, no mode bits, no
  directory entries — so `make assets-zip` twice gives the same bytes and the committed archive is
  not a source of spurious diffs. `go test ./assets` compares the archive against the tree file by
  file and by content rather than by archive bytes, since a toolchain's deflate is free to change;
  a tampered byte in the tree fails it with the file and both lengths named.
- The plumbing changed shape once, cleanly: `*zip.Reader` is an `fs.FS`, so every loader takes an
  `fs.FS` from its caller instead of opening paths. Nothing under `internal/` knows the built-in copy
  exists — only the two commands import `assets` — which is also why `go test ./internal/...` links
  none of these 11 MB. `internal/assetfs` holds the resolution rule in one place, and
  `internal/assetpack` holds the pack and the comparison so that `tools/packassets` can run them
  without importing the package that embeds what it is about to write.
- The Makefile gained an `embedded` guard: every build target refuses without the archive rather than
  producing an executable that comes up empty, and says both how to restore it and how to rebuild it.
  `make assets` repacks after extracting. Three targets that used to skip on a tree-less checkout now
  do real work — `audio`, `headless` (3 frames and 8 screens) and `check-caveats` — because they no
  longer need one; `houses` still skips, since it names `.house` files by path.
- A release archive is now two binaries and five documents, no `assets/` at all, and no
  `HOW-TO-RUN.txt` tells anybody where to stand. `release.yml` proves the claim rather than asserting
  it: it copies one cross-built binary into an empty directory, runs `-version` and `-shot` there,
  and then greps every packaged executable for two asset filenames — zip stores member names
  uncompressed, so that check works on the Windows and macOS binaries a Linux runner cannot execute.
- **What it costs.** A binary went from about 3.6 MB to 14.9 MB; an archive holds two of them, so the
  same 11.3 MB ships twice and the six archives went from about 15 MB to about 26 MB each. The
  repository carries the assets twice as well, 68.6 MB tracked to 79.9 MB. Three ways to spend that
  back — one binary instead of two, keeping only the archive in git, unioning the roots instead of
  replacing them — are written up as deferred in 5.3 rather than half-done here.
- Verified two ways beyond the suite. A replay produces the identical digest `7364a572f7b6d7d7`
  whether it reads the built-in copy or the tree, so the embed changed no pixel and no sample; and
  the `linux-amd64` archive's binary, alone in an empty directory, played 300 on-screen frames of
  Slumberland with sound at 552 fps unpaced.
- Unrelated but adjacent, and it was costing every push: `ci.yml` ran its whole matrix twice over the
  same commit whenever a branch with an open pull request was pushed, once for `push: branches:
  ["**"]` and once for `pull_request`. Concurrency groups cannot collapse those two — the refs differ
  — so `push` is now `main` only, with a concurrency group per ref that cancels superseded runs.

### A Windows backend, so the Windows archives can draw (2026-09-17)

Stage 4's window half, brought forward ahead of Stages 2 and 3. The release pipeline was already
packaging two Windows archives and both were `-headless`; shipping those was worse than doing this
early.

- `internal/platform/win32`: one window at an integer multiple of 640×480, a `StretchDIBits` blit
  per frame, `WM_KEYDOWN`/`WM_KEYUP` for the glider and `WM_CHAR` for typing a high-score name.
  Pure `syscall` to `user32`, `gdi32` and `kernel32` — no cgo, no dependency, nothing for a player
  to install — so it cross-compiles from Linux and `CGO_ENABLED=0` release archives get a real
  window rather than the null backend.
- The blit is a copy, not a conversion. A BI_RGB 32bpp DIB scan line is `0x00RRGGBB` per pixel as a
  little-endian DWORD, which is byte-for-byte the port's own BGRX `Framebuffer`; a negative
  `biHeight` makes the DIB top-down so the rows go up in the order they already sit in.
- `platform.Expand`, the nearest-neighbour scaler, was extracted from the X11 backend and is now
  shared by both. That was the point: the arithmetic in the Windows blit path is the part most
  likely to be wrong and the part a Linux machine can still test, and it now runs under `go test`
  at four scales with padded strides on both surfaces. X11 draws through the shared version at the
  same frame rate as before (844.9 fps at 1:1, 122.4 fps at 2:1 on this host).
- Keys and text are split into `keys.go`, deliberately with **no build tag**, so the two pure
  pieces — the 256-entry virtual-key table and the UTF-16 accumulator that recombines surrogate
  pairs — are unit-tested on Linux. One test walks `platform.KeyNames()` so a key added to the enum
  with no Windows mapping fails there rather than in a bug report; another round-trips the whole
  BMP through `utf16.Encode`. Both were proven to fail on injected defects before being trusted.
- Backend selection is still compile-time only, and there are now three selectors rather than two.
  `internal/platform/backend/doc.go` reads them side by side and states the two load-bearing
  details: `!windows` on the cgo clause, so a `CGO_ENABLED=0` Windows build still gets a window,
  and `nullbackend` first, so `make headless` and the fidelity corpus win on a host that could
  open one. Verified empirically with `go list` across six configurations, not by reasoning about
  the tags.
- `OpenPipe`'s "no audio player" message no longer recites five Linux sound stacks at a Windows
  reader. It names the two that have Windows builds, `ffplay` and `play`, because `exec.LookPath`
  finds `ffplay.exe` from the bare name — so a Windows machine with FFmpeg or SoX on its `PATH`
  does have sound, which is a better answer than the "silence" that was expected (2.48).
- **It has never run.** It was written on the same airgapped Linux host as everything else here,
  which has no Windows to execute it on. What is verified: it compiles and vets for `windows/amd64`
  and `windows/arm64`, the pure pieces are tested, and the shared expansion is tested and
  benchmarked. What is not: window creation, the message pump and the blit. The package comment
  opens by saying so, `ci.yml`'s `native` job now runs `-frames 300 -bench` on `windows-latest` as
  the first execution in existence, and both the release notes and each Windows archive's
  `HOW-TO-RUN.txt` tell the reader they may be the first person to see it draw.
- Consequently the two Windows archives lose their `-headless` suffix, and `make cross` stops
  calling those rows null. Three of six archives draw now; the packaging change was rehearsed by
  extracting the step from the YAML and running it verbatim against a real `make cross` tree.

### A release pipeline (2026-09-17)

- `.github/workflows/release.yml`: a `v*` tag runs the test suite, cross-compiles every target,
  packages six archives with a `SHA256SUMS`, and creates the GitHub Release. This is
  `docs/IMPROVEMENTS.md` 5.4, which had been open since Stage 0.
- The `verify` job is not redundant with `ci.yml`, and the reason is worth writing down: `ci.yml`
  triggers on `push: branches` and `pull_request`, and **a tag push is neither**. Without it,
  tagging would run no tests at all and a release would ship binaries no suite had ever seen.
- Each archive carries its own copy of `assets/extracted` at the relative path the binaries look
  for, because they look for it relative to the working directory and not to themselves (5.3, closed
  by the entry above). So an archive is self-contained and has to be run from its own root, and its
  `HOW-TO-RUN.txt` says so.
- Five of the six cannot draw, so they are named `-headless` rather than left to disappoint
  somebody. Their `HOW-TO-RUN.txt` deliberately does not tell the reader to run the bare binary:
  the null backend delivers no quit event and the shell loop has no exit condition without
  `-frames`, so that spins on an invisible title screen until Ctrl-C. It gives two commands that
  terminate and produce something — a `-shot` PNG, and 300 frames of a house dumped as PNGs — and
  both were run from an unpacked archive before being written down.
- The dispatch input reaches the shell through `env:` rather than `${{ }}`, and is then narrowed to
  an anchored `^v[0-9][A-Za-z0-9.+_-]*$`. `${{ }}` is textual substitution performed before bash
  sees the script, so the obvious spelling executes whatever a quote in that input contains, in the
  job that builds what gets published. Verified by feeding the resolver an apostrophe, a `;`, a
  `$(id)`, a space, a slash and an embedded newline: six refusals, and the newline case also closes
  a `$GITHUB_OUTPUT` key injection.
- `publish` is gated on the push event as well as on the tag, because the dispatch ref picker
  accepts a tag: a tag-only gate would let a manual run publish a release named after the tag while
  every asset in it was named after the dispatch input. It also takes its version from the build
  job's output rather than deriving a second one, so the release cannot advertise a filename it did
  not attach. Both halves are kept — either alone closes the hole.
- Attach-if-it-exists rather than create-or-die, because publishing through the Releases UI creates
  the tag, which fires the workflow, which would then build for half an hour and refuse to attach
  what it built.
- The README's nine links into `docs/` are rewritten to absolute URLs pinned at the built commit
  when it is copied into an archive — three of them are the screenshots at the top, and an archive
  carries no `docs/`. Shipping 9.2 MB of development notes six times over was the alternative.
- The notes' checksum command is `sha256sum --ignore-missing -c`, because `SHA256SUMS` lists all
  six archives and almost nobody downloads six: the plain form reports the five absent ones as
  `FAILED open or read` and exits non-zero, which reads exactly like a corrupt download.
- It has never run — same airgapped host as `ci.yml`, and it says so at the top. Everything below
  the GitHub line is verified: the packaging step was extracted from the YAML and run verbatim
  against a real `make cross` tree, all six archives pass `sha256sum -c`, and the `linux-amd64` one
  was unpacked and played 300 frames at 30 fps with sound and the right version string from its own
  root.

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

- `internal/fidelity`: a per-frame hash corpus, the first pixels checked into the repository —
  as text, one line per frame, so that a moved pixel is a moved line. A rendering change now needs
  `go test ./internal/fidelity -update` and a moved hash in the diff.
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
- The release pipeline exists but has never run, and nothing is tagged yet, so a build still
  reports a bare short hash as its version (5.4). There is no installer either.
- Linux/X11 and Windows/GDI draw; macOS and cross-compiled arm64 still run headless (5.5). The
  Windows backend has never been run by a human — see the entry at the top of this file — and
  Windows has no sound unless FFmpeg or SoX is on the `PATH` (2.48).
- An installed copy still looks for its assets beside the binary (5.3).
