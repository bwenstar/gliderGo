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

### The title screen says GliderGo, and names who ported it (2026-09-23)

The splash is PICT 1000, Calhoun's own title art, and its logo said "Glider PRO". The shell now
paints PRO out and writes "Go" where it stood, so the logo reads GliderGo. It also adds "ported by
brendan ta" on the line under "by john calhoun". Both are drawn in the plate's own hand. The G is
Calhoun's G and the o is the bowl of his d turned round, both reduced to 0.6. The credit's b, y, o,
n and a are the letters of his credit line, pixel for pixel, and the five his line has no use for
(p, r, t, e, d) are drawn to match. All of it is in PRO's lighter brown, so the dark ink is still
his and the light ink is what was added (`docs/IMPROVEMENTS.md` 1.5).

The stamp (`internal/shell/title.go`) paints the screen after the plate is copied. The PICT in
`assets/extracted/` is not touched, and `make assets-check` still proves it is the 1994 bytes. The
stamp is at fixed coordinates, so it first checks that the pixels under it are the shipped PICT's,
by FNV-1a over the area it writes, and it leaves any other splash as it is. Eight tests hold it:

- Every pixel outside PRO's box is the PICT's, or ink laid on its sky. So Calhoun's "Glider", his
  credit and the plane are his to the pixel, and nothing of PRO is left inside the box.
- The ink is PRO's: every pixel PRO had is one of the three shades the stamp draws in.
- The mark has three pixels of sky between it and both the r and the plane.
- Each letter the credit shares with his is found in his line.
- The grids are well formed, and they sit inside the checked area and clear of the menu's halo.
- A one-pixel change to the splash turns the stamp off.
- `Draw` shows the stamp.
- The name after "ported by" is a row under `credits.txt`'s `[this port]`.

Each test was seen to fail against a change that breaks what it guards.

`internal/credits/credits.txt` names Brendan Ta under `[this port]`, which is what the last test
holds. A credit painted into picture data is one nothing can read at runtime, which is the file's
own opening complaint.

Six fidelity screen hashes move: splash, houses, settings, race, about and credits. Those are the
six screens drawn over the splash, and the credits screen has the new row as well. The scores
screen brings its own backdrop, so its hash does not move. `docs/screenshots/title.png` is taken
again. The README's copy predated the Race... row and the 24th house as well.
`docs/screenshots/house-picker.png` is taken again too: the splash shows above the picker's panel,
and the new one differs from the old in 185 pixels, all of them the top of PRO.

### The demo's floor is the whole stream, and the documents stop quoting the run it left behind (2026-09-23)

`TestTheDemoReplaysTheSameWayTwice` guards the records the shipped attract recording consumes, from
below, with `demoRecordsFloor`. The floor was 573, the figure from the days when the glider lost all
three lives in the start room. The `'bnds'` fix (`03d0cf0`, `docs/IMPROVEMENTS.md` 4.24) took the
run to all 1117 records, so the floor sat 544 below the truth and would have passed a change that
threw most of that back. The floor is now `demo.ShippedRecords`
(`internal/replay/replay_test.go:1281-1304`). The branch that logged a request to raise it is gone,
because a cursor cannot consume more records than its stream holds, so it could never run again.

Every figure was measured again before any document was changed. The run consumes 1117 of 1117
records, loses its lives at frames 1781, 2043 and 3417 in rooms 4, 5 and 15, and ends at frame
3432. Frames 3415 to 3417 ask for a key after the last record at 3414, and the third death then
flags the game over. Seeds 0, 1, 7 and 12345 give the same run, as they did at 1.8b. They differ
only in the trace's `rand=` column and its digest, and for seeds 7 and 12345 in the last frame's
main-plane digest. README's fidelity bullet, the `demo.script` header, PLAN 1.8b and IMPROVEMENTS
2.18 and 4.7 said the demo still stopped at 573, and they now say what it does today. Each keeps
573 only where it says what Stage 1.8b measured. 4.8 said the demo test skips on a fresh clone. It
has run there since `8f39609`, and 4.8 now says so. So did six comments in the code (in
`internal/replay`, `internal/scores`, `internal/render`, `internal/house`, `internal/shell` and
`cmd/glidergo`), which now say what an absent tree does mean: `make clean-assets`, a half-written
extraction, or a mistyped flag. The frame numbers are still not asserted,
because they are this port's and not 1994's (2.18). This closes the third same-day bullet of
`docs/PLAN.md`'s release gate.

### Every release archive carries Go's licence, and the README link check can fail (2026-09-23)

Both binaries in every archive have the Go runtime and standard library compiled in, and that code
is BSD-3-Clause. Its second clause says a binary redistribution must reproduce the notice, and no
archive did. The package loop in `.github/workflows/release.yml` now writes
`THIRD-PARTY-NOTICES.txt` beside `LICENSE` (`release.yml:367-372`). It is two lines naming the Go
that built the binaries and saying the licence below covers that code, not gliderGo, then
`$(go env GOROOT)/LICENSE`. Under `GOTOOLCHAIN: local` that is the toolchain that built these exact
binaries, so the notice cannot drift from them. The zip branch converts it to CRLF along with
HOW-TO-RUN.txt. A pre-seal `grep -q '2009 The Go Authors'` (`:543`) fails the build if the file is
missing or lacks the notice. The pattern leaves out go1.23's "(c)", because the 2024 wording drops
it and the x/telemetry copy vendored in go1.23.12 already uses it. The release notes' Licence
section and README's now name the file. Filed as `docs/IMPROVEMENTS.md` 1.4, one of the release
gate's same-day fixes.

The README check beside it could never fail. `! grep -q '](docs/' "$stage/README.md"` sat in the
middle of a `set -e` script, and bash does not exit on a command whose status is inverted with `!`.
It is an `if` that exits 1 now (`:533-536`).

release.yml cannot run on this airgapped host. Its package step was taken out with a YAML parser
and run against the local `bin/cross/` with go1.23.12, the way 4.13 rehearsed it. All six archives
hold the file: 1,651 bytes in the tarballs and 1,681 with CRLF in the zips. After its header the
file is byte-identical to the toolchain's LICENSE. With the block deleted, the step exits 2 at the
assertion before sealing anything. With a `](docs/` link planted in the staged README it now exits
with status 1. The old form sealed all six archives and exited 0. What setup-go's GOROOT holds on
a GitHub runner stays unverified until the next tag's workflow runs.

`glidergo -version` and `glidertool version` print a `go` line under `licence`, "BSD-3-Clause, ©
The Go Authors", for a binary copied out of its archive. It is one constant, `project.GoLicence`,
and `TestTheReadmeAndTheGameAgree` holds README's sentence to it.

### A shredded glider falls as confetti, and the game no longer ends by asking to be reported (2026-09-23)

`World.RenderShreds` asked `Assets.Sheet` for `"shred"`, and `shred` is not a sheet. It is
`shredSrcMap`, a 40×35 GWorld of its own loaded from PICT 4010 with its mask from 5010
(`StructuresInit.c:556-563`), and the port files it with the strips, beside the toast and the fish.
A failed accessor does not stop anything: it records the first error and returns nil, and every blit
in `RenderShreds` is guarded by that nil. So the cloud grew, fell, played its shred sounds and
sparkled with nothing in it, and `glidergo` ended the game with `render: no such sheet "shred"`, the
bug-report footer and exit status 1. Thirteen of the 22 shipped houses have a shredder. The
hostile-house soak planned as `docs/IMPROVEMENTS.md` 4.30 found it, and it is filed as 2.73. The fix
is `Strip` for `Sheet`. It restores what the original draws, so it is faithful and needs no setting.

The fix is one word. What is worth recording is why nothing saw it. Nothing that loads art had ever
shredded a glider. The golden traces stay clear of shredders, the game package's shredder tests run
without art on purpose, and `TestEveryHouseStartsAndRuns` spends each house's hundred frames nowhere
near one. The trace cannot see it either: the reproduction's digest is the same before and after,
because a trace records rects and sounds, and those were all correct. Two tests now close the two
gaps:
- `internal/replay`'s `TestAShreddedGliderFallsAsConfetti` flies a glider into Slumberland's room 43
  with the whole asset tree loaded. It asserts that the run returns nil. It then finds every one of
  the strip's 850 opaque pixels, exactly, in Work on exactly twenty frames: the growth arm's last
  two and the eighteen the fall draws, in one column and four pixels lower each frame. They are
  never in Back.
- `internal/render`'s `TestEveryArtNameIsInItsTable` reads the calls rather than making them. It
  parses every non-test file in the module and follows the helpers that pass a name on, found by
  reading their bodies rather than from a list that would go stale. Every constant that reaches
  `Sheet`, `Strip` or `Object` is held to the table its accessor reads. An `Object` must name an
  object drawn from its own PICT, because the extractor writes a crop of every sheet object into
  `object/` as well, and a wrong `Object` call would load without complaint. It checks 87 names, and
  this was the only one wrong. `TestTheArtNameCheckCatchesEachWayANameCanBeWrong` plants this bug
  and nine other kinds of mistake, and requires each to come back at its line.

No pixel hash moved. None of the seven fidelity screens or the golden traces visits a shredder,
which is the gap this closes. The game package's no-art shredder tests pass with the bug put back,
and their header now says why the picture is held elsewhere.

### A held key does one thing on Linux, which it already did on Windows (2026-09-23)

`platform.Event.Repeat` exists so that a held key does one thing, and five places rely on it: the
shell's filter (`internal/shell/shell.go:439`), the race result (`cmd/glidergo/race.go:776`), the
high-score board (`highscore.go:231`), the pause's `S` (`play.go:717`) and the banner and ending
waits (`play.go:933`). On Linux none of them ever saw it set. The x11 backend marks a press as a
repeat when its bitmap says the key is already down, and by default an X server sends every
auto-repeat as a `KeyRelease` and a `KeyPress` with the same timestamp, so the bitmap had always
just seen the key go up. A held key fired its action at the server's repeat rate. Escape held on the
house picker went back to the title screen and then quit the game, a held Return dismissed the "Hit
anything to begin" banner it had just opened, and a held Backspace emptied the Race screen's address
field. Filed as `docs/IMPROVEMENTS.md` 2.72.

`x11.New` now calls `XkbSetDetectableAutoRepeat` straight after `XOpenDisplay`. The server then
sends repeats as presses with no release between them, which is what Windows does, and the existing
rule marks the presses `win32.go`'s `lParam` bit 30 marks. XKB is
part of libX11, so nothing new is linked. All three cases were reproduced before and after on a
nested Xephyr, with the key held through XTest by a throwaway helper that is not part of the build.
Held arrows still move menu cursors, because the shell lets arrow repeats through, and the picker
lands on the same row as before. Xephyr and the DCV desktop both support the flag. A server that
does not is not refused, because the glider reads the bitmap and flies the same either way; the
package comment says what such a server loses, and GLFW's peek fallback is not written until one
turns up.

A held letter in the Race field now acts once, as a held Backspace does, which is what Windows
already did; a text field that should repeat-delete is a change to the shell's filter, for both
platforms at once. The high-score name dialog does not filter repeats on either platform, so held
keys still repeat there on both. `Repeat` is worked out per X keycode rather than from the bitmap,
which is indexed by `platform.Key`, so that two cases the bitmap would get wrong match Windows too.
A key this port has no name for, such as an accented letter, now marks its repeats, and Delete
tapped while Backspace is held, two keys that share one `platform.Key`, is a fresh press.

`internal/platform/x11` has its first test. With `DISPLAY` set, it checks that the connection has
detectable auto-repeat on after `New`, asking the server rather than the package. It is skipped when
`DISPLAY` is unset, which includes CI's `make check`, so CI runs the package again under its own
Xvfb.

### A race can be arranged from the title screen, and the menu found the edge of the artwork (2026-09-23)

Stage 3's networked race worked and had no way in. `-host`, `-join <address>` and `-port` were the
whole interface, and `cmd/glidergo`'s dispatch put a race on the same path the measurement flags use
— straight into the house, past the title screen — because there was nowhere on that screen to type
an address. On Windows, which is where a release is a double-clicked `.exe` from a zip, that made the
mode effectively unreachable: there is no shell in front of the player at all. Filed as
`docs/IMPROVEMENTS.md` 4.28 while finishing the stage, and half of it is closed here.

`internal/shell/race.go` is a Race screen: `R` from the splash, or the arrows to the menu row
directly under the original's own four. Two rows and one field — Host waits for the other machine,
Join dials an address — and the field is `internal/scores`' `Field`, reused whole for its Mac Roman
limit, its select-all-on-open and its keyboard, which is what it was made generic for when the
high-score name needed it. `scores.Prompt` is deliberately *not* reused: every rectangle in it comes
out of DLOG 1020's own item list, and putting 1994's measurements on a screen 1994 never drew would be
this port claiming the original had a network mode.

What the screen produces is a `shell.Race` on the `Choice` the shell already hands to `Play`, so
`internal/shell` still has no `net` import and still knows nothing about a socket: it arranges a race
and `cmd/glidergo` runs one. `-host` and `-join` now build the same value, which is the point of the
type — there is one path through `play` rather than two, and the flags have become a way of *skipping*
the title screen rather than the only way in. `-port` stopped being refused on its own in the same
change, because it now means "the port the Race screen will use". One consequence had to be unpicked:
`errRaceGaveUp` used to be swallowed, which was harmless when the only caller was a command line about
to exit, but a shell that gets back a zero `Outcome` and no error puts *"Fun House — score 0, 0 stars
left"* on the status band, which is a report of a game nobody played. It is returned now; the shell
shows it and stays up, `playDirect` unwraps it into a clean exit.

**The menu had nowhere to put a ninth row, and the 1994 artwork is what decided where it went.** The
panel's bottom is pinned by the house label, which the original draws at a fixed `MoveTo(436,314)`
(`SelectHouse.c:87-96`); its top is pinned by PICT 1000, because `panel()` dims eight rows beyond
every edge and a panel any higher lays a checkerboard across the aeroplane's tail. So the tail was
measured off the extracted art rather than argued about: in the panel's columns, halo included, rows
78–93 hold ink narrowing from 105 pixels to 10, and row 94 is the first with none. That number is the
constant `splashSkyV` now, written down rather than measured at run time because the art is fixed and
the test that reads it has to pass on a build with no assets extracted at all. Nine rows fit at
`menuTop` 102 with two rows shaved off the selection bar's padding; a tenth does not fit in one
column, and the geometry test says so in its failure message so that the next person does not
re-derive it. Moving the label, narrowing the panel and folding "Two Player Game" into the new screen
were all considered and are all recorded as rejected.

**The row could not go where it reads best, either.** A race is the other two-player game, so
directly under "Two Player Game" is where it belongs by meaning — and that is one row above where
MENU 129 puts "Open Saved Game...", which a test pins on purpose. The original's order is not this
port's to rearrange, so the row takes the first line past the end of 1994's list: still inside the
group that starts a game, still above the three that do not.

**And the host's waiting screen was printing an example that could not be pasted.** It read out
`glidergo -join 10.0.0.5:1138 Fun House`, unquoted, and a positional house argument is one argument —
`parseFlags` answers that line with "one house at a time". Sixteen of the 22 shipped houses have a
space in their name. The address is now on a line of its own and first, because that is what the
other player types into the field, with the command line underneath it and quoted; double quotes,
because that line gets read on Windows as often as on a shell that understands either.

The second half of 4.28 stays open, and it is a protocol change rather than a screen. A guest should
be *told* the host's house instead of naming it, but `netplay.Meet` is a single symmetric exchange in
which both sides send a `Hello` carrying their own house hash, so there is no moment at which the
guest knows what the host opened and has not yet committed to a house of its own. That wants an
asymmetric first round, and it wants doing with LAN discovery rather than before it — a guest offered
a *list* of hosts would be told the house as part of the list. Meanwhile both screens say the name
out loud: the Race screen's last-but-one line, and the host's waiting screen.

The screen joins the pixel corpus, which is now seven screens rather than six
(`internal/fidelity.DefaultScreens`, `make headless`, `-shot-screen race`), and the splash's hash
moves in this commit because the menu did.

### A house of our own can carry pictures of its own, and `-houseart` no longer takes the other houses' away (2026-09-23)

A room's `background` at 3000 or above names a picture the *house* carries rather than one of the
eighteen the application carries, and twenty of the 22 shipped houses use that — but until now a
house authored in `levels/` could not, and the flag that looked like the way in was the reason.
`-houseart DIR` *replaced* the built-in forks rather than adding to them, so the only way to give a
new house a picture was to point the flag at a directory, which took the shipped houses' pictures
away for as long as it was pointed there. Both halves of that were filed as `docs/IMPROVEMENTS.md`
4.15 while writing the first house of our own, and the mechanism for both is closed here.

The lookup is a search list now rather than a substitution. `assetfs.ArtRoots` is an ordered list of
roots and a house takes its pictures from the first one holding a directory of its name: `-houseart`
first, then `houseart/` inside the levels tree, then the extracted 1994 forks. So the authoring path
for a house of our own is a file and no flag — a PNG at `levels/houseart/<House Name>/pict/3000.png`,
`make levels` to copy the tree in beside the built houses, `make levels-zip` to pack it into the
archive every executable embeds. A downloaded binary then has the pictures the same way it has the
house, which is the property `assets/levels.zip` exists for; `assets/extracted/` could not be the
place for them, because it is byte-for-byte diff-locked against the extractor. `bnds/<id>.bin` is the
other half of a fork and is read on the same search, so a house-carried background can say which of
its sides are room openings. One thing had to learn to ignore the new directory: the house library
walk in `internal/shell/library.go` now skips `houseart` at the top of the levels tree, by that name
and only there, because a picture is not a house and a room called `pict` is not one either.

Making the flag purely additive would have broken the workflow it exists for, which is testing an
extraction: a tree missing half its houses would have quietly looked complete. So a root remembers
whether a *person* named it, and a named root that gets passed over is reported (`no resource fork at
/tmp/art/Titanic; using built-in:houseart/Titanic`) while a built-in miss stays silent. The levels
tree is deliberately not a named root even when `-levels` was typed on the command line: what was
named there is a directory of houses, so a house in it with no art beside it is ordinary rather than a
mistake worth a line.

The complaint those roots print was itself wrong, which is 4.17, and it was wrong in two ways at
once. It fired whenever a fork was missing, including for the great majority of houses that do not
want one, and it existed in two copies that had already drifted apart. There is one copy now,
`assetfs.Fork.Complaint`, and it asks `house.WantsOwnArt()` rather than guessing: a house wants a fork
if any room's background is 3000 or above, or any object is a `kCustomPict`. The note that filed 4.17
listed two more conditions than that and both are wrong. `kTV` wants a QuickTime movie, which this
port has no support for at all and would not find in a resource fork. `kSoundTrigger` reads the
*sound* root's shared manifest, keyed by house name, and never looks at the art tree. `kCustomPict`
counts at any id, including ids under 3000, because Metropolis carries its own PICT 1999 and Fun House
its own 2014 and 2015. The predicate is checked against the corpus rather than argued: over all 22
shipped houses `WantsOwnArt()` agrees exactly with whether a fork directory exists on disk.

Two commands stay silent on purpose. `house lint` already reports a missing picture as
`background-pict`, with the id and the room name, which is a better report than a one-line summary —
and adding the summary beside it would recreate the two-copies problem that 4.17 was. `house stats`
is a measurement, and `forkBounded` already counts the rooms a missing fork affected. A replay is
silent for a third reason: its report is a trace, and a missing picture shows up in the frame hash.

No house of ours carries art yet, and that half of 4.15 stays open, because it is an art task and not
an engineering one — the share of Open House that would want a painting is nineteen of its 43 rooms,
and a single new room at background 3000 moves every number `internal/profile`'s tests assert. What
proves the mechanism instead is an end-to-end test that builds the tree in a temporary directory and
plays two frames: the back plane's hash differs with the art present and matches what `-houseart`
produces for the same tree. Sounds are the remaining gap and are now filed as 4.29 — the 22 originals
carry 63 of them across 13 houses, and a house authored here still cannot carry one.

### Stage 3, second half: the race is something a player can start, watch and win (2026-09-23)

`-host` waits for another machine, `-join <address>` finds one, and `-port` moves both off 1994. The
protocol half had no UI; this half is the three screens it needed and the one mapping it was missing,
from `World` to `Standing`. Two machines on a LAN now race a house from the same seed and both arrive
at the same answer about who won.

A race goes down the same path the measurement flags use — straight into the house, no title
screen — because there is nowhere on the title screen to type an address. That is a gap and not a
design: `docs/IMPROVEMENTS.md` now carries it, along with the related observation that a guest is
made to name the house it already learns from `Hello.PeerHouseName` during the handshake. What the
flags do refuse is five combinations, each with its own sentence: `-host` with `-join`, because one
machine hosts and the other joins it; a race with `-two`, because `-two` is two players at one
keyboard and a race is one glider on each machine; a race with `-resume`, because a race starts both
players at the house's beginning; a race with `-seed`, because the two peers agree a seed between
themselves; and `-port` without either, which is the one that catches a typo rather than a
misunderstanding.

The waiting screen reads out the whole command the other player types, `glidergo -join <ip>:<port>
<house>`, with up to three of this machine's addresses — IPv4 first, IPv6 bracketed, interfaces that
are down or loopback or link-local left out. The port comes from the *listener* and not from the
flag, so a host that asked for port 0 still tells the truth. Getting out of that screen is the
interesting part: all three of the calls a peer blocks in before a match exists — `Accept`, a dial's
`DialTimeout`, and `Meet`'s first `Recv` — are uninterruptible by asking, and can only be released by
closing the thing underneath them from another goroutine. So the connecting goroutine hands each
resource to a small mutex-guarded struct as it acquires it, and Escape closes whatever is held. The
same struct refuses a handover after a cancel, which is what stops a socket that arrived a moment too
late from being leaked instead of closed. A failed dial is retried every half second rather than
reported, because the commonest way joining fails is joining before the other player has pressed
host, and the last failure is shown on the screen while it retries so that a wrong address still
looks wrong.

There is one exception to "the player presses Escape", and it is the reason a 30-second give-up
exists at all: **a run with nobody in front of it has no Escape key**. A `-frames` or `-bench` or
`-shot` run that joined an address nobody is hosting would retry until somebody killed it, which in a
script is a hang and not an error. The test for which kind of run this is is `hermetic`, in
`cmd/glidergo/prefs.go`, and not a fresh condition written beside it, because "is there somebody in
front of this" already had one name and two answers to it would eventually disagree. The same
function now decides whether the result screen is drawn and whether the wait for the other player
blocks, which is a small tidy-up of a condition that had been spelled out by hand.

The live opponent panel is where the frame loop and the network actually meet, and the asymmetry in
it is the substantive result of this half. `Present` is called more than once per frame — 116 or 160
times during a wipe, once per strip — and building a standing calls `CountRoomsVisited`, which walks
every room in the house (383 in Slumberland). So the report to the network is gated on the frame
number changing, and the panel is *drawn on every call*, because the game composes each frame from
its work map and restores the pixels under anything that moved there. Nothing in this path writes to
the work map, so the game's own idea of the screen is untouched and the panel survives by being
redrawn rather than by being remembered.

It sits in the top-left corner, and it is worth saying plainly that there was no good place to put
it. The screen is 640x480 with a 512-wide room in the middle; the scoreboard takes rows 460..480 with
nine neighbours and rows 59..79 with one or three, and with nine neighbours those bands hold slivers
of the rooms next door. Every option covers something. The corner covers the least, and the panel
disappears the moment the run ends so that the game-over and high-score screens are the original's.

Finishing is two screens and one rule. `finalStanding` is the mapping, and its three cases are the
game's own: a run that ended without `GameOver` was given up, `Mortals < 0` is a death, and anything
else finished the house — which follows `internal/game/play.go`'s own branch rather than inventing a
second reading of the same two fields. Then the race waits, because **the race is not over until both
runs are, however far ahead you finished**, and only then prints and draws the result. The panel and
the result screen deliberately describe a departed opponent differently: the panel says "left" and
shows the last standing as it arrived, while the result folds it through `netplay.Abandoned` first,
because a forfeit is a scoring rule and a live readout is not.

Verified in two processes over a real loopback socket on a `nullbackend` build, which is the only way
any of this could be exercised without two people: both peers agreed the match ID and seed, took
different slots, and independently reached the same outcome. Killing the guest three seconds into a
300-frame race left the host with the guest's last standing, a named `broken pipe` on its result, and
its own run finished normally.

### Stage 3, first half: the race protocol, from the specification that was already written (2026-09-22)

`internal/netplay` is the networked two-player race — **not** the original's two-player mode, which
is a shared room and shipped separately as 1.9. Here each machine simulates only its own glider in
its own copy of the house and the connection carries progress. This half is the protocol: the
envelope and framing, the handshake, the standing message and the arithmetic that decides a winner.
No UI yet, and every one of Stage 3's three acceptance clauses covered by a test over a
pipe — two peers racing to a result each computes for itself, a guest killed mid-race, and a house
mismatch refused by name and hash — then again over a loopback socket.

The plan offered "length-prefixed JSON or a small binary framing" as a free choice, and that line
predated `docs/analysis/determinism-networking.md` §10.4, which specifies a complete binary protocol
for the lock-step mode this port is not building. Inventing a second format beside it would have
repeated `docs/IMPROVEMENTS.md` 4.24 exactly — documents that state a layout, and a port that used
another — so the race speaks §10.4: its 8-byte envelope, its message-type numbering, and its seven
encoding rules. Three additions were needed and each is argued where it lives: a `uint32` length
prefix, because §10.4's messages are datagram-shaped and TCP is a stream; one new message type,
`MsgStanding` (0x30), because §10.4.8's `MsgProgress` is a ~150 KB resume snapshot rather than a
per-change record; and a rule §10.4 has no reason to state, that a message which is allocated but
*not spoken by this build* — `MsgInputFrames`, from a lock-step peer — is refused out loud, because
two builds silently ignoring each other's traffic is the one failure worse than an error message. A
type from outside every allocated range is still skipped in silence, which is §10.4.10's rule and
buys the forward compatibility it was written for.

The handshake is symmetric, with no host authority, because there is nothing for an authority to be
authoritative about: both sides send a `MsgHello`, the smaller nonce becomes player 1, and the match
seed is an FNV-1a mix of the two nonces in value order, so neither peer can choose the house's
behaviour by choosing its own number. The two houses must hash identically or the match is refused
with both names and both hashes on screen — the names are usually the same, two builds of one house
being the common case, so the hashes are the only thing that tells them apart. There is no starting
gun and no countdown: the metric is rooms visited and the tie-breaks are score and then *frames
simulated*, so a peer that started late has simply simulated fewer frames, and nothing has to agree
about a clock (which `docs/IMPROVEMENTS.md` 4.11 is the standing measurement of the futility of).

Two rules in `Winner` depart from a literal reading of the plan's "the winner is decided by rooms
visited", and `docs/PLAN.md` Stage 3 now records both. **Finishing the house outranks the metric**,
because otherwise a player who completed the house on a direct route loses to one who wandered into
more rooms and then died, and a race whose winner is a corpse is not a race. And **a forfeit beats
everything and is settled before the race is over**: a player whose process is killed has left, so
the other one wins immediately, still in the air or not. Its mirror image is the easier one to get
wrong, and has its own rule and its own test — a peer that reported `Finished` or `Died` and *then*
hung up has forfeited nothing, so the goodbye at the end of every ordinary race must not decide it.

The transport is ordinary TCP and the only decisions in it are the ones a player sees. Port **1994**
by default, because a number somebody can remember is worth more than a number somebody has
registered for two people agreeing to play on one LAN; binding is separate from accepting, so a port
already in use is reported before the waiting screen goes up rather than instead of it; the listener
closes as soon as the guest arrives, because a third machine told "shut" is better served than one
left waiting on a match it will never be part of; and closing the listener is how a host that has
changed its mind gets out of a blocked `Accept`, which nothing else can interrupt. A player may type
a bare host, a bare port or both, and an IPv6 address unbracketed — that last one is read as an
address and given the default port, which costs the ability to write `fe80::1:2000` and is the
correct trade, since nobody has ever meant "the loopback host, port nothing".

`netplay.Race` is the driver the game loop will use, and it exists so that the loop is written once
here rather than once per caller. Three properties are the reason: **the frame loop never blocks on
the network** (a standing goes to a writer goroutine, so an opponent whose machine has hung cannot
slow this one's frame rate); **reports go out only on a change**, because `Present` is called more
than once per frame — 116 times during a wipe — and a report per call would be thousands of messages
a second with nothing failing to show it; and a departure is folded into the result by one rule in
one place, so two peers cannot reach two answers. A pending standing is *replaced* rather than
queued, which loses nothing, since a standing is cumulative and a newer one carries everything the
one before it did — and `Close` flushes the last one before saying goodbye, because that is the one
the result is computed from. `Settled` is the channel the "waiting for the other player" screen will
wait on: the other side's fate, not this side's, because both machines have to wait for the same
fact.

One bug worth recording, because the socket test found it and the pipe tests could not. The driver
first had a single flag for "the peer has gone", set both by a goodbye and by a broken connection,
and `Close` skipped its own goodbye when it was set. Those are two different facts: a peer that said
goodbye has finished its *run* and is sitting on a result screen waiting to hear how this one went.
So whoever finished second hung up in silence, and whoever finished first waited for a standing that
was never sent and then scored the race a forfeit — over a pipe the timing hid it, over a socket it
happened on the first run. The flag is now two flags with the difference written down next to them.

`make race` is new and `make check` runs it, because this is the first package in the port with a
goroutine in it and nothing would otherwise have kept it honest. It covers `internal/netplay` and
nothing else, for the reason it is worth having at all: the concurrency is confined to one small
package on purpose, so the detector costs a second rather than the order of magnitude it would cost
across the whole suite. On a machine without cgo or a C compiler it skips and `check-caveats` adds a
line saying the goroutines were only tested single-threaded, which is the same rule the rest of that
list follows — a check a machine cannot perform must not be reported as one it passed.

### The asset extractor publishes by rename, so nothing ever reads a half-written PNG (2026-09-22)

`tools/extract_all.py` wrote straight into `assets/extracted/`, which meant that for the ~70 seconds
it ran, the tree was a mixture of the old extraction and the new one. That was not theoretical: a
`go test ./...` beside a `make assets` once failed decoding a golden PNG with "unexpected EOF", and
it took a while to recognise as a build-system problem rather than a renderer one.

It now stages each run in `assets/.extracted.tmp-<pid>` and renames the finished tree into place,
holding `assets/.extracted.lock` — an OS advisory lock, so a cancelled CI job cannot leave a stale
one — for the duration. A second run refuses rather than interleaving. A `^C`, a crash and a failed
count all publish nothing, which is the part that was not in the plan: the eleven counts this script
checks against the analysis documents are now verified *before* the rename, so a regressed extractor
no longer replaces 46 MB of good assets with 46 MB of wrong ones and then reports the failure. That
one case keeps its staging tree and names the path, because a complete extraction with wrong numbers
is what you want to look at.

`--only art,sound` was the hard half. A partial run has to publish a whole tree, and two of the six
steps read a bucket they do not write, so the buckets a run is not rebuilding are hardlinked from
the published tree into the staging tree first. Verified by running it: `--only houses` against a
tree holding only `movie/` publishes both, a full `make assets` republishes all 1,877 tracked files
with no `git diff`, and `make assets-check` still proves the committed tree is what the extractor
produces. `docs/IMPROVEMENTS.md` 5.2, which had been open since 1.2 and wanted closing before a tag
exists — a release pipeline is precisely a place where two jobs share a checkout.

### Two lint checks for the coordinates the 1994 editor never let an author choose (2026-09-22)

24 object types do not have a vertical position in Glider PRO — they have a `#define`. A floor vent is
at `v` 305 because `kFloorVentTop` is 305 (`GliderDefines.h:467`); `AddNewObject` writes the constant,
`DragObject`'s case arm for that type adds `deltaH` and does not mention `deltaV`, and
`KeepObjectLegal` rewrites a wrong one back on every save. Eight of the 24 — the four doors and four
windows — have named horizontal constants as well, and a dragged door is snapped to the nearer wall
with its `what` rewritten to match the side it landed on.

So `object-top` and `object-left` in `internal/house/lint.go`, both `warn`, from a table of 32 rows
that are citations rather than measurements. The 22 shipped houses raise neither: 3,998 of 4,041
vertical placements and 167 of 167 horizontal ones are at their constant, and the other 43 are one
type, one value and one 1994 bug (below). `TestCorpusFixedAnchors` is what says the constants were
transcribed correctly, because it checks them against 22 files nobody involved can edit.

This is a check that restates a tool's behaviour, which is the argument for it: gliderGo's own houses
are not written by that tool. They are typed into `levels/*.house.txt` with the coordinate spelled out,
and `make levels` lints with `-fail warn` — so a mistyped column now fails the build instead of
shipping a ventilation grille hanging two pixels above the floor. Both of the port's houses pass
unchanged. `docs/IMPROVEMENTS.md` 4.23, which filed this as a note and named two questions that had to
be answered from the C first; both are answered there, including that one of the note's own claims
about `HouseLegal.c` was wrong.

One side effect worth naming: 37 subtests in `lint_test.go` built fixtures with every object at `v` 0,
so the new check fired 98 times the moment it was wired in. The fixtures were fixed, not the check
exempted — `plain`, `transportTo` and `switchTo` now place their object where the original would have.
The file's header says the synthetic half builds a house Lint has nothing to say about and then breaks
one thing in it, and those 37 had quietly stopped being that.

### A dead branch in the original's repair pass, and the 43 objects that are where it left them (2026-09-22)

`HouseLegal.c:132-137` repairs a `kFloorTrans`'s vertical coordinate to 302. It sits inside a `case`
block whose case list (`:75-90`) does not include `kFloorTrans`, which is handled thirteen arms later
by the block for `data.d` transports — so the branch has never executed. 43 of the 244 shipped floor
transporters are at `v` 300 instead of 302: 37 in Slumberland, 5 in Leviathan, 1 in Rainbow's End.
They are the only exceptions in the whole 4,041-placement census, and they are exactly the type whose
repair does not run.

Had it been reachable it would also have been wrong: the branch writes through `data.a`, and
`data.a.distance` is the same two bytes as `data.d.tall`, so repairing the position would have grown
the transporter's height by 2.

`object-top` accepts 300 for that one type as a value and not a tolerance — 299 and 301 still warn —
through a named constant with the finding cited beside it. The 43 are not normalised on load: a shipped
house is the specification, and 302 is not what Slumberland contains. `docs/IMPROVEMENTS.md` 4.27.

### The release packaging step has now been run, on a machine that cannot reach GitHub (2026-09-22)

`release.yml` has a standing caveat that it has never run, and it was doing more work than it needed
to. The "Package the archives" step needs nothing from GitHub: extract its `run:` block, dedent it,
give it `VERSION` and the three `GITHUB_*` variables its `sed` interpolates, and point it at the
`bin/cross/` layout `make cross` already writes. All six archives pack, all six pass the greps that
assert the assets really are inside the binary, and `SHA256SUMS` comes out with bare filenames.

The link rewrite does what its comment claims — the README's three screenshots become `/raw/` links
and its thirteen documents `/blob/` links, every one pinned at the 40-character commit SHA rather
than a branch, with nothing left pointing at a `docs/` that the archive does not carry. The CRLF
conversion and the notes step's `VERSION` substitution were run the same way. That was the last
documented command line in this repository nobody had executed except `gh release create`, which
genuinely does need the network (`docs/IMPROVEMENTS.md` 4.13, now closed, and 5.4).

### The release archives now say what Windows and macOS will do to an unsigned download (2026-09-22)

Nothing here is signed, and both operating systems object in a way that is indistinguishable from a
corrupt download. A Windows player gets Defender SmartScreen's "Windows protected your PC", whose
only visible button is *Don't run* — *Run anyway* is behind *More info*, which is a click nobody
makes without being told. Explorer then copies a downloaded zip's Mark of the Web onto every file it
extracts, so the dialog can return after it has been dismissed once. On macOS a download carries
`com.apple.quarantine`, and Archive Utility passes it to what it unpacks while `tar xzf` does not.
Three operating-system behaviours, none of them a fault in the game, all three looking exactly like
one — and until now neither the release notes nor `HOW-TO-RUN.txt` mentioned any of them.

All three are now named with their click path: in a new "Your computer will try to stop you, once"
section of the release notes, in a block in the Windows archives' `HOW-TO-RUN.txt`, and in a shorter
one appended to the macOS archives' `HOW-TO-RUN.txt` alone — the headless text is shared with
`linux-arm64`, whose reader has no Gatekeeper to hit. Not signing is stated as a decision rather than
left to look like an oversight, and each section ends on the distinction that matters: none of these
checks looks at what is *in* the archive, and `SHA256SUMS` already does.

These click paths are documented behaviour rather than something this project has watched, which it
now says out loud. The Windows half can be rehearsed offline, though, because a browser download is
an ordinary file with one alternate data stream on it: `docs/windows-first-run.md` has the two
PowerShell commands that reproduce it, listed alongside the newly-stated fourth gap in that document
— the `.exe` that ran on Windows Server 2025 arrived over SSH, so it carried no mark and SmartScreen
never had an opinion about it.

### Both maps of this tree now have to name paths that exist (2026-09-22)

`docs/PLAN.md` §3 drew the tree as it was imagined before it was written, and the code had since
disagreed with it in four places: `internal/ui/` for what is `internal/shell/`, an
`internal/assets/` that was never written because the embedding turned out to want two files at
`assets/`'s top instead, and `internal/game/objects/` and `internal/game/room/` for a decomposition
the simulation never grew. It was also missing `internal/platform/backend/`, the build-tag selector
every binary actually goes through, and still called the Windows backend "never run" a stage after it
was run on Windows Server 2025.

The map is no longer an exhaustive listing — `README.md`'s "What is in here" table is that, and has
been swept package by package since 2.4 — so PLAN keeps the layering, the three directories that do
not exist yet with the stage that owns each, and the `levels` pipeline. What is new is that both maps
are now checked. `TestEveryPathTheArchitectureMapNamesExists` walks PLAN's fence as the indented tree
it is and resolves all 25 paths; a `[stage N]` line is exempt from existing and is required *not* to,
so a marker left behind on a directory that has since been written fails the way `win32`'s did.
`TestEveryPathTheReadmeMapNamesExists` closes the direction the README's sweep never had: every
package had to be on the table, but nothing said a path on the table had to exist, and a row for a
directory that has been renamed away reads exactly as complete as one that has not.

### A constant in `internal/project` that nothing reads is now a test failure (2026-09-22)

That package holds the handful of facts the About box, `-version`, both tools' usage text, the issue
templates and the README all have to agree on, and its whole argument is that a fact spelled out in
six places will eventually disagree with itself. `project.Releases` was spelled out in one place —
its own declaration — and pointed at a release page with nothing on it, because no tag has been
pushed yet. It is deleted rather than linked: the README's "there is nothing to download and no copy
of the old game to find" is true today, and pointing a player at an empty page would have been the
same kind of defect as a documented command that does not run.

`TestEveryExportedConstantHereIsReadBySomething` is the general form: it parses the package for its
exported constants and sweeps every other `.go` file for readers. The rule is transitive, because the
first run found a second constant with no caller and it was a false accusation — `Home` is
`"https://" + Module`, so `Module` has exactly one reader and is the string every URL in the game is
built from. `docs/IMPROVEMENTS.md` 4.13 has the reasoning, and 5.4 records where the URL belongs on
the day there is something to download.

### How long `make check` takes, measured (2026-09-22)

Three documents describe what `make check` does and two of them were wrong. `CONTRIBUTING.md` quotes
the target's steps verbatim and had fallen one behind; `docs/DEV_ENVIRONMENT.md` named six of the
fourteen. On the timing they disagreed by an order of magnitude — "a couple of minutes" against
"~15 s" — because neither said whether Go's build cache was cold. It is **50 s cold and 12 s warm**
on an eight-core machine, and both documents now give both numbers.

The verbatim copy is held to its source from now on: `go test ./tools/docscheck` reads the Makefile's
`check:` line and fails if `CONTRIBUTING.md` does not contain it, printing the line to paste. The two
prose summaries are left as prose, deliberately — see `docs/IMPROVEMENTS.md` 4.26, which also records
why `docs/DEV_ENVIRONMENT.md` is not one of the documents `docs-check` runs.

### What `make check` says it could *not* check is now true on Windows and macOS (2026-09-22)

Three places print a caveat — the last row of `make cross`, both halves of `check-caveats`, and
`smoke`'s note when it skips — and all three were written on Linux and said so without meaning to.
They ask `go env GOOS` now.

The two platforms were wrong in opposite directions. On macOS everything overstated: `make cross`
built a *darwin* binary with cgo on and printed it as `linux/amd64 +cgo … x11 backend (this host)`,
and `check-caveats` reported that cgo was off — or that libx11 metadata was missing — about a
platform that has no backend to compile either way yet. On Windows it understated, which is the
direction that list exists to prevent: it announced that the x11 backend had not been compiled and
that `build` had produced the null backend, of a build whose win32 backend needs no cgo, *was*
compiled, and draws. Then it blamed an unset `DISPLAY`, a variable Windows has no reason to set, for
the window that did not open.

```
  darwin/arm64 +cgo            --       not attempted: cgo buys macOS nothing until stage 6
  - darwin has no backend of its own yet (stage 6), so `build` produced the null
    backend and cgo makes no difference to that

smoke: not attempted on windows -- DISPLAY is not what decides a window here, and
       nothing in this Makefile can tell whether one would open. `make bench` is
       the on-screen run, and it is the one that would say so.
```

Only the sentences changed. On Windows `smoke` still skips rather than benching, because nothing in
a Makefile can decide from outside whether a window would open there, and an honest skip is worth
more inside `make check` than a possible flake. Two smaller things went with it: `smoke` used to say
the blit path was "still covered by `make headless`", which is backwards — `headless` renders
through the null backend, so the blit is the one thing it does not cover — and the libx11 row named
`linux/amd64` whatever architecture you were on.

None of these branches is reachable from CI: `windows-latest` has no `make`, and the macOS half of
the same job calls `go` directly for the same reason, so the job that compiles and tests the Windows
and macOS code never runs the Makefile that describes it. `GOOS=darwin make check-caveats` and its
two siblings print what those readers see, and the Makefile's comments name them, because they are
the only test these branches have. `docs/IMPROVEMENTS.md` 4.13 has the rest.

### `make check` now runs the command lines the documents tell you to run (2026-09-22)

A documented command that does not work is the most expensive wrong sentence in a repository: it is
the one a newcomer meets in their first five minutes, and the only evidence they have about whether
the rest is worth reading. This project had nine such defects, every one found by a person typing a
line into a shell — including a command offered by `glidertool replay`'s own usage text that answered
`no house to replay`, which had been wrong for six stages. Nothing checked the documents.

`make docs-check` does, and it is part of `make check`. `tools/docscheck` finds the 38 command lines
in the `bash` fences of `README.md` and `CONTRIBUTING.md`, runs the 19 that can run unattended, and
prints the rest with a sentence each saying why not:

```
   292  ok     0.00s  bin/glidertool house build -o my-house.house my-house.txt
   296  ok     0.76s  bin/glidertool render -all -o /tmp/demo "assets/extracted/houses/Demo House.house"

not run -- 19 lines, and why
  README.md:132  make run
      opens a window and plays until the player quits; -shot and -frames are the headless forms

docscheck: 19 of 38 documented command lines ran, and all of them worked
```

Each line runs in its own `bash -o pipefail -c`, in a scratch directory of symlinks to the
repository — so relative paths resolve, a documented `-o` cannot leave a file in your working tree,
and `.git` is deliberately not reachable. Each gets two minutes, because one of the nine defects was
a hang and a check that inherits a hang never ends.

The half that will still be working in a year is the bookkeeping. A documented line that no rule
classifies fails, and a rule for a line no document gives any more fails too — a stale rule reads
exactly like coverage and is none. Both are `go test ./tools/docscheck`, which needs no binaries, no
assets and no display, so adding a command to the README tells you so from the test suite you were
already running.

On the way in it caught two stale sentences (`CONTRIBUTING.md`'s copy of what `make check` runs, and
`README.md`'s claim that `tools/` is python3 only), and `CONTRIBUTING.md`'s five-line house-authoring
walkthrough was run end to end for the first time. `make help` now lists `all` and `smoke`, which it
had always skipped. `docs/IMPROVEMENTS.md` 4.13 has the reasoning, including what is still not
covered and why.

### Flags can now come after the file names, which is where people put them (2026-09-22)

Go's `flag` package stops at the first argument that is not a flag. For `go test ./... -run X` that is
the point; for a tool whose positional argument is a file it is a trap, because the order a person
types is the order they think in — name the thing, then say what to do with it — and that was the
order that did not work:

```
glidertool house stats "Demo House.house" -tier tutorial     # now means what it looks like
glidergo Slumberland -scale 2                                # so does this
```

Both of those failed, and neither failed cleanly. `house stats` printed the whole eighteen-row table
**with no tier column at all**, having silently demoted `-tier` to a file name, and then stopped with
`open -tier: no such file or directory` — a flag ignored and then blamed for not existing on disk.
`house dump f.house -o out.txt` answered `house dump takes exactly one house file`, which is
confidently false to somebody who gave exactly one. And `glidergo Slumberland -scale 2` was refused
with `one house at a time: 3 were named (Slumberland, -scale, 2)`.

`internal/cliargs.FlagsFirst` reorders one command line before `Parse` sees it, at all sixteen
FlagSets in the tree. It asks **the FlagSet itself** whether each flag takes the token after it —
`IsBoolFlag`, an attached `=`, a name nobody registered — so a new flag needs no edit there and
cannot be got wrong there. It is not a parser and does not become one: an unknown flag stays
refusable by name (`flag provided but not defined: -teir`, not "no such file"), `-h` still reaches
`flag.ErrHelp`, `-q true` still leaves `true` a positional because that is what `flag` does with it,
`house build -` is still stdin, and a file named after `--` survives being moved.

`TestEveryCommandLineInTheTreeIsReorderedBeforeItIsParsed` reads the tree's own AST and requires
every FlagSet parse to have been reordered, because fifteen call sites are fifteen chances for the
sixteenth subcommand to copy a neighbour's first ten lines and not its eleventh. It found a
sixteenth site nobody had counted — `tools/packassets`, which takes no positional arguments at all
and silently ignored one, so `packassets assets/levels` packed `assets/extracted` and reported
success. That is now refused.

`docs/IMPROVEMENTS.md` 4.13 is the entry, and this closes its first open bullet. Every documented
command line still puts its flags first; the difference is that this is a house style now rather than
a requirement.

### A house can be measured against the 1994 ones without writing a test first, and the port's own tutorial house turns out to miss a row (2026-09-22)

`internal/profile` measures a house against §10.2 of `docs/analysis/original-houses.md` — eighteen rows
of bands across five size tiers, derived from all 4,070 shipped rooms — and `glidertool house stats`
prints it:

```
glidertool house stats -tier tutorial "Open House.house"
glidertool house stats -summary assets/extracted/houses/*.house
```

`-tier` adds the bands and a verdict column, `-fail` turns a miss into an exit status, `-rooms` names
the dark and unreachable rooms by index as well as by name, and `-summary` is one line per house in the
columns §3.6 publishes, so the output can be read straight against the document.

**Two of §10.2's rows had been filed as uncomputable and both were computable.** The dark-room row is
three lines against a `*render.Scene`, which was fixed earlier; the last one, BFS eccentricity, needed
a room graph. `internal/profile/graph.go` is that: it drives `DetermineRoomOpenings` over every room,
adds the manholes, staircases, one-way walls, dirt-tile skylights and resolved transports, and
breadth-first searches the result. The edges are directed, because leaving a room to the east needs
*this* room's right wall open and the neighbour is not consulted (`Interactions.c:662`,
`Transit.c:398-404`). It is checked against `tools/probe_houses_inventory.py`'s published figures for
all 22 houses — a transcription written before any of this existed — so agreement across 4,070 rooms is
the pin, and a set of hand-built one- and two-room houses in which exactly one thing is true says what
each individual rule is, since a rule that fires in no shipped house and one that fires in all of them
both pass a 22-house comparison unchanged. The
reachable count is reported and never linted: every shipped house has rooms this model cannot reach,
Art Museum 76 of 109 and Fun House 5 of 43, and `docs/IMPROVEMENTS.md` 4.1 declined it as a lint check.

**`Open House` misses the eccentricity row and is not being changed.** Its longest shortest-path is 10
and the tutorial band is 11-15. The band is two houses — `Empty House` at 11 and `Demo House` at 15 —
and the miss is the house being better connected rather than smaller: 43 rooms all reachable inside 10
hops, against `Empty House`'s 35 in 11 and `Demo House`'s 33 of 45 in 15. Its star sits at hop 9 and
the four deepest rooms are the roof edges just past it, where a third of `Demo House` lies beyond its
own goal. `TestOpenHouseMatchesTheTutorialProfile` names the row as an allowed miss with that argument
attached, and fails if a different row starts missing *or* if the exemption stops being needed.

Both house tests now ask the package instead of holding a hundred lines of tallies between them, which
closed a gap the hand-written lists had: they asserted sixteen and seventeen of the eighteen rows, and
`prize:enemy` was simply absent from one of them with nothing able to notice. Three things the tool
found on its first run are in `docs/IMPROVEMENTS.md` 4.16, which this closes — including that §10.2's
`prizes/room` band excludes `Empty House`, the house it was measured from, by rounding 0.0286 up to a
floor of 0.03.

**And the fourth thing it found is about §10.2 rather than about any house: 21 of the 22 shipped
houses are outside a band of their own tier.** Only `Leviathan` is inside all eighteen, which it
manages by having set eight of epic's bounds itself. `glidertool house stats -tier` used to print
"no shipped house is inside all eighteen of its own tier's" — a sentence truncated mid-clause and
false, written from three houses before anything could ask all 22. Three mechanisms produce the 21,
and all three are properties of the table (`docs/IMPROVEMENTS.md` 4.25): edges rounded *inward*, so
that large's 6.0-7.5 objects a room contains neither `Land of Illusion` at 5.9934 nor `Rainbow's End`
at 7.5022, the two houses it was measured from; an empty-rooms row that is the corpus-wide 20 % rather
than a per-tier spread, which 18 of the 22 houses are outside — six have no empty room at all and
`Demo House` is 53.3 % empty; and §8.3's grouping disagreeing with §10.2's room counts for the two
smallest tiers, so `Sampler`, `California or Bust!` and `Fun House` are each outside the room band of
the tier the document itself puts them in. The note now says 21 of 22 and
`TestTheBandsExcludeTheHousesTheyWereMeasuredFrom` measures it, alongside a new corpus pin that the
840 empty rooms the walk counts are §5.4's own 840.

### 155 rooms in the shipped houses were sealed on all four sides, because a four-byte resource was read as an eight-byte rectangle (2026-09-22)

**A fidelity bug, and not a small one.** `internal/render.Assets.Bnds` decoded the `'bnds'` resource as
a QuickDraw `Rect` — eight bytes, four big-endian `int16`s — because the C declares its handle
`boundsHand`. `boundsType` is four one-byte `Boolean`s in the order left, top, right, bottom
(`GliderPRO/Headers/GliderStructs.h:266-272`), and every one of the 70 resources the shipped houses
carry is exactly four bytes. The `len(raw) < 8` guard therefore rejected all 70, `GetOriginalBounding`
answered 0 for every lookup, and 0 means closed on all four sides.

A room takes that path when it has a house's own background and no `bounds` field of its own. **155
rooms across 7 houses do**: Castle o' the Air 55, Rainbow's End 30, Slumberland 24, Land of Illusion 21,
Demo House 10, Leviathan 9, The Asylum Pro 6. **146 of the 155 lost at least one exit they should have
had** — 141 the left wall, 138 the right, 109 the ceiling, 79 the floor. The other nine really are
sealed, by a resource of four zero bytes, and none of those nine is a trap: they leave by a staircase, a
window or a `kInvisTrans`.

The 79 are the ones a player would have noticed. A closed bottom *is* a floor, so those rooms got a
floor the original does not give them: the glider could not fall out of a room designed to drop it, and
`IsShadowVisible` drew a shadow on a floor plane that is not there (`Room.c:1103-1133`).

`render.Bounds` is now a struct of four `bool`s rather than a `Rect`, so the type refuses the mistake,
and three tests stand where none did: a byte-driven decode test whose cases fail under the old reading,
a pass over all 70 shipped resources asserting each is four bytes of 0/1 and each is found, and a census
in `internal/game` pinning the 155 rooms per house and checking the flags arrive as the openings
`DetermineRoomOpenings` reads. Six of the 22 house digests in
`internal/game/testdata/objects_golden.txt` move as a result, which is the whole visible consequence —
`master` and `hot` counts are unchanged, since object rectangles never depended on this.
`tools/extract_house_art.py` carried the same assumption in its manifest, where a `">4h"` unpack
over-read each four-byte resource into the next one; it now records the four flags by name, and
`assets/extracted/houseart/manifest.json` and the archive are regenerated.

**The documents had it right the whole time** — in twenty-nine places across eight of the thirteen that
mention the resource, including a numbered list of porting traps in `docs/analysis/structs.md:4332`
whose item 3 is this bug by name: *"Four bytes, easy to get backwards, and the symptom is rooms with the
wrong walls open."* Nothing compared the documents to the code. What found it was `internal/profile`'s
room-graph walk, checked against the independent Python transcription's published figures, coming out
short on exactly six of the eight houses that ship the resource. `docs/IMPROVEMENTS.md` 4.24 is the entry, and it
files the general check: a decoder and a documented layout should be made to agree by a test.

### An object is now read against the background it stands in, and the paragraph that justified the severities was audited first (2026-09-22)

Three new lint checks — `mount-no-floor` and `mount-no-ceiling` at warn, `starfield-tiles` at note —
which brings `glidertool house lint` to **32 checks** producing **637 notes, 48 warnings and exactly
one error** over the 22 shipped houses. They are the first checks in the linter that relate an object
to anything outside its own record. Every existing check asks whether a room's own fields are legal:
`what` in range, links resolving, `tiles[]` inside the picture, names fitting `Str27`. None of them
could see a floor vent bolted to the sky.

That was found by making the mistake. `Boarding House`'s generator produced a `kStars` room carrying
`kSky`'s `tiles[]` *and* a `kFloorBlower` — one room, two defects — and nothing in the toolchain
objected. The reason the tiling half got through is worth stating, because it is a gap and not an
oversight: every built-in background is exactly 512 pixels wide, eight `TileWide` columns, so a
built-in room's `tiles[]` selects eight of eight and `tile-column` **can never fire on one**. The
check that looks like it covers this is structurally incapable of it.

**The first half of the work was auditing the paragraph that sets the severities, not adding
anything.** `lint.go`'s header justifies calibrating against the originals with four examples, and two
of them were wrong. "Staircases that lead nowhere" has **zero** instances: the four stair rules fire
not once across all 4,070 corpus rooms, so the phrase belonged to `CheckForStaircasePairs`
(`HouseLegal.c:961-1045`) existing — Calhoun wrote a repair pass, so he had seen the problem — and not
to these 22 files, which it read as a measurement of. And the paragraph implied the corpus lints
clean, which it does not: `link-slot-range` is an error and CD Demo House room 72 trips it. Three
stair rows and the correction are now in `TestLintCorpus`, and the header states the rule it was
reaching for:

> a class the corpus exercises deliberately cannot be an error; a class it exercises by overrunning a
> buffer can

There is exactly one of the latter. Room 72's link has `who` 35, `GenerateRetroLinks` indexes
`retroLinkList[who]` with no bound check (`House.c:577`, `:600`), and the C reads into the next room's
record. So `house lint` exits 1 on the originals by design, and a test pins that count at 1 so it
stays deliberate.

**That rule is what calibrated the new checks, and it changed two of the three decisions the filed
entry had made.** `docs/IMPROVEMENTS.md` 4.22 asked for two rules over three object types. Measuring
the corpus first gave twelve types and inverted one severity.

The classification is a census, not a reading of the names. Each of these twelve occupies exactly
**one** vertical coordinate in all 4,070 rooms — 1,458 `kFloorVent` all at v 305, 507 `kSewerGrate`
all at 303, on down to 12 `kCeilingBlower` all at 5; 2,839 placements with no exceptions. The two
lamps are why that is evidence rather than trivia: `kHipLamp` sits at v 23 and `kDecoLamp` at 91,
nowhere near the vents, but they are 276 and 212 pixels tall, so their *feet* land at 299 and 303 —
inside the updraughts' own 292–305 band. They are standing lamps. A rule written from the names would
have filed both with the pendant fixtures and been wrong twice. The five updraughts and two
downdraughts are also exactly how `CreateActiveRects` groups them (`internal/game/hotspots.go`, from
`ObjectRects.c`), so the two halves agree from independent directions.

**The five flames are excluded on purpose, and that is the case that would have failed the corpus.**
`kTaper`, `kCandle`, `kStubby`, `kTiki` and `kBBQ` make a thermal lift column the same way an
updraught does, so a rule derived from the physics would include them — and 18 `kTiki` and 9 `kBBQ`
stand in rooms with no ceiling, one `kCandle` and one `kStubby` in rooms with no floor. A torch on a
lawn is a torch on a lawn. For the same reason the two background maps are separate rather than one
predicate: `kRoof` has a floor — you walk on it — and no ceiling, while `kSkywalk` and `kDirt` have
both despite looking outdoor. Those are facts about `Room.c:1138-1206`, not about how a picture looks.

**The tiling rule went from warn to note because the corpus is not unanimous.** 62 of 62
`kStratosphere` rooms are the identity tiling but only 243 of 246 `kStars` rooms, and all three
exceptions are Leviathan's — rooms 172, 206 and 221, one of them a straight reversal of 0..7. By the
rule above, that is a note. Each background also quotes its own tally in the finding rather than one
standing in for the other, and a test asserts the wording, because a single "243 of 246" covering both
would have been the imprecise citation this stage is about, in the check added to settle it.

**One limit, chosen and documented.** Only backgrounds 2000–2017 are checked. A user-art room's
openings come from its `bounds` field or the background's `'bnds'` resource, and `internal/house` has
`PictSize` and nothing analogous — so `mounting` returns early outside that range, with a test row
asserting it stays silent. A false negative by choice; a false positive would be a bug.

The stronger fact the census turned up is filed as 4.23 and deliberately not implemented. A vent at
v 200 in an ordinary room is floating art, and that is the likelier authoring mistake than a vent in a
floorless room — but the numbers above are a measurement of 22 files, not a citation, and 4.21 is the
entry about exactly that distinction. Two things need finding first: where the floor line is in the C
as a constant, and whether the 1994 room editor snapped these objects, which would make the check a
restatement of a tool's behaviour and change what its severity rests on.

### A second house, at a tier the first one could not reach, and the roof tiles that turned out to be physics (2026-09-22)

`levels/Boarding House.house.txt` is 51 rooms on a 7×13 grid, and the reason it exists is not "more
content". `Open House` was written against the tutorial column of `docs/analysis/original-houses.md`
§10.2 and hits every row of it, which sounds like evidence that the table is usable and is not: a
single house fitted to a single column is also exactly what you get by drawing a house and then
choosing the column it happens to land in. The claim can only be tested at a second tier, so this one
is the **small** column — 45 to 85 rooms — whose bands differ in kind and not just in width. 7 to 19
objects a room against 2.2 to 3.1. Enemies, where the tutorial tier permits almost none. Up to 4 %
dark rooms, where it permits none at all. The two houses share no layout, and all seventeen of this
one's rows are inside their band: 51 rooms, 388 objects at 7.61 a room, 33 enemies, 47 prizes, 3
stars, 65 distinct object codes, 27,600 points for a clear.

**Both houses are now counted by the same code.** `internal/replay/profile_test.go` holds the walk and
each house's test holds only its column, because with the arithmetic duplicated a disagreement between
the two could be a difference in the houses or a difference in the counting, and a failure would not
say which. Factored, a divergence is always the house. It also closed a row that had been declared
uncomputable — see below — and it caught the row-versus-header mismatch in the older test, which
asserted "the fullest room holds 0..23 objects" where its house's header claims "rooms at the
24-object ceiling: 0". Those are different quantities and the first says almost nothing, since 24 is
the ceiling. `Open House` now asserts all sixteen of its table's rows rather than thirteen.

**The dark-room row was computable the whole time, and the reasoning that said otherwise is the useful
part.** `docs/IMPROVEMENTS.md` 4.16 argued that a dark room needs `GetNumberOfLights`, which is a
method on `*render.Scene`; that `internal/render` imports `internal/house` so the dependency cannot be
inverted; and that re-deriving the rule means a second copy of a switch over eighteen backgrounds
that would drift silently. Every clause is true. The conclusion — that the row cannot be checked —
does not follow from any of them, because it is a claim about where the *check* lives and the argument
is about what `internal/house` may import. The check lives in `internal/replay`, which imports
`internal/render` already and cannot run a game without it. Three lines. "X cannot import Y" is a fact
about two packages; "this cannot be checked" needs the extra step "and no third package imports
both", and that step was never taken and was false.

It earned its place immediately. `Boarding House`'s Bell Turret was `kRoof`, one of the eight
backgrounds that light themselves, and became `kPaneledRoom`, which is not — so changing a background
put out a light, no other row moved, and the house's dark-room figure went to 3.9 % against a 4 %
ceiling. **The band did not catch it. Reading the number did**, which is why both profile tests print
their measurements and the names of the dark rooms rather than only a verdict.

**That same edit had a second consequence nothing caught, and finding it is why the background
histogram is now pinned too.** Moving one room from `kRoof` to `kPaneledRoom` moves it from the
open-air side of the house to the ceilinged side, which falsified four figures the house's header
quotes — the room counts on each side and their object means — while every profile row stayed green,
because a background is not an object. They were wrong in the header until recomputed by hand. So
`TestBoardingHouseUsesEveryBuiltInBackground` now asserts the exact count per background rather than
only that all eighteen appear: one line of data, and any background edit in any room has to come
through it. The alternative — computing "has a ceiling" in the test — means duplicating
`DoesRoomHaveCeiling`, which is a method on the running `World` and not a function of a background, and
every derived figure is downstream of this table anyway. The corrected split is 30 rooms of interior
art against 21 outdoor, 16 of those with no ceiling, the ceilinged rooms averaging 10.00 objects and
the open-air ones 2.38.

**Eighteen of eighteen built-in backgrounds, and `kRoof` is where that stopped being a coverage
exercise.** No house in the repository had used them all — not ours, and none of the 22 from 1994.
Doing it means eighteen tile sets, four different wall rules and three different ceiling rules in one
file a person can also play, and it means `kRoof`, whose eight tiles are **collision geometry wearing
a picture**: `CheckRoofCollision` (`internal/game/player/escape.go:309-348`) gives tiles 1, 2, 5 and 6
a sloping surface (`250−dx`, `186−dx`, `122+dx`, `186+dx`) and gives 0, 3, 4 and 7 nothing at all, so a
glider whose centre is over one of those four below v 122 dies. A roof that looks continuous can be
lethal in four of its eight columns. The three roof rooms here were laid out against that arithmetic
and each verified to sit above a room that has a ceiling.

**Two corpus rules were reproduced only after getting both wrong in one room, and that room is now
`docs/IMPROVEMENTS.md` 4.22.** `DoesRoomHaveFloor` gives `kSky`, `kStratosphere` and `kStars` no floor
at all, which is why there is not one `kFloorVent`, `kFloorBlower` or `kSewerGrate` among the corpus's
**1,084** roof-and-sky rooms — the art would stand on a hole. And **243 of the corpus's 246** `kStars`
rooms carry the identity tiling, because the background is one 512-pixel picture. A room here was
written with `kSky`'s tiles under `kStars` *and* a `kFloorBlower`, and nothing in the toolchain
objected: `glidertool house lint` reports 0 notes, 0 warnings, 0 errors for a floor vent bolted to the
sky. The linter's 29 checks are all about a room's own fields being legal and not one relates an object
to the background it stands in — which is invisible in the direction people test, because the room
draws, the lift works, and the only thing wrong is what the player sees. Filed with the numbers that
decide its severity.

**The playthrough is pinned, and the frame numbers came out of the simulation because arithmetic
cannot produce them.** `TestBoardingHouseCanBeFinished` flies
`internal/replay/testdata/boarding-house.script` from the Vestibule to the bell in the turret: twelve
rooms, five staircases, all three stars, 1,447 frames, 16,100 points, neither glider lost. Two things
about deriving it are worth keeping. The hover cut-off is `top = V − D − 20 = 16`, but the glider
arrives there carrying `VVel` −6 against gravity's +3 a frame, so it **coasts past** and tops out
between 6 and 10 — all 23 key presses in the script are at top 8, 9 or 10, and a rule set written from
the cut-off reproduces none of them. And `GliderInRect` is containment, not overlap
(`internal/game/player/hitbox.go:55-58`), so a staircase's 112×32 trigger needs the whole 48×20 glider
inside it, in a band that sits *below* `CeilingLimit` — which means a glider fresh off a vent is above
the box and cannot be in it, and every staircase is taken on the way back down. Five events a stair
room, not three. The script's header carries the four rules that regenerate every frame number in it.

**`docs/IMPROVEMENTS.md` 4.18 is narrowed by measurement rather than by argument.** It said §10.3's
construction recipe and §10.2's tier table contradict each other, because the recipe's "interiors get
11-13 objects" is unreachable inside a tutorial budget of 138. At the small tier they do not
contradict: this house's 35 rooms that have a ceiling average **10.00** objects and 24 of them hold 11
to 13, the recipe's figure taken literally, with the whole house at 7.61. The recipe is written against the
corpus mean of 7.7 and the small tier *is* the mean — the tutorial column is the outlier, and a recipe
in absolute numbers will contradict exactly that one column and no other. Which is a smaller and more
actionable defect than the one originally filed, and it could only be found by writing a house at
another tier and measuring it with the same code.

**One new item, `docs/IMPROVEMENTS.md` 4.21, and three rows of test for the half of it that was
cheap.** The paragraph at `internal/house/lint.go:19-24` justifies the linter's severities by naming
four defects the 22 shipped houses contain, "so none of those classes can be an error". Two of the
four do not hold. There are **no** staircases that lead nowhere: four stair rules, 326 stair objects,
4,070 rooms, zero findings — every flight in the corpus is paired and both ends land in a room that
exists. And the out-of-range `who` **is** an error, does fire once, and makes `glidertool house lint`
exit 1 on the originals, which the paragraph says cannot happen. It should be an error — the original
indexes `retroLinkList[who]` with no bound check and reads into the next room's record, and a linter
that shrugged at that would be the useless one — so the rule the code follows is narrower and better
than the rule the comment states. `TestLintCorpus` already pinned that error at 1; what it had no row
for was any stair rule, which is why the other half went unexamined. Three rows added, safe at zero
because `TestLintStairs` fires all four rules on houses written to provoke them. The comment itself is
still to fix.

### The twenty-third house ships inside the binary, and the flag that had to start replacing (2026-09-22)

`Open House` existed and no player could reach it. It needed `-levels assets/levels` on the command
line, so the house, its test and its lint-clean build were for people with a clone and the
instructions in front of them; anybody who downloaded a release archive got twenty-two houses and no
sign there was a twenty-third. `assets/levels.zip` closes that: a second `//go:embed` beside
`extracted.zip`, opened by `assets.Levels()`, so `glidergo` with no flags now lists **23 houses, 0
skipped** and reports five asset roots, `levels    found (built-in:levels)` among them.

**The archive is committed and the directory it is packed from is not**, which is the reverse of
`assets/extracted{.zip,/}` — both of those are committed, because no clone can re-run the extractor
without the 1994 CD. Here the reason is a cycle: `make levels` runs `bin/glidertool`, `cmd/glidertool`
imports `assets`, and `assets` embeds `levels.zip`, so a checkout without the archive cannot build
the tool that packs it. The `embedded` guard's remedy for a missing one is therefore `git checkout --
assets/levels.zip` and nothing else, and the rule both archives follow is "commit what a clone cannot
regenerate, plus whatever the build needs in hand before it can regenerate anything". Houses sit at
the archive's *root* — `Open House.house`, not `levels/Open House.house` — because what is embedded
is the levels root itself, and `assetfs.Whole` exists so that the label for it is `built-in:levels`
rather than the `built-in:.` that `Root(tree, dir, ".")` would have produced, which would have put a
dot in the middle of every path in every message about a house.

**`-levels DIR` now replaces the built-in root instead of adding to it**, and the deciding case is
this repository's own documented workflow. `make levels` builds `levels/*.house.txt` into
`assets/levels/` so an author can play what they just wrote, and the way they play it is `bin/glidergo
-levels assets/levels`. Additive semantics list every house **twice** — once from the archive, once
from the directory it was packed from — and the two rows share one save file and one score file,
because a side-car is keyed on the house's name and `h.TimeStamp` and both rows agree on both. That
is not an edge case a player has to go looking for; it is what happens the first time anybody follows
the instructions. So all five root flags replace, without exception, and what accumulates is the
*set*: `Library.Discover` is still variadic and the Original and New roots are still two sources.
`docs/IMPROVEMENTS.md` 4.14 carries the amendment, because the paragraph it corrects argued the other
way and is left standing. Two tests in `cmd/glidergo/levels_test.go` hold the decision down, on the
real command line and the real archives: `-levels DIR` where DIR holds a copy of a house already in
the binary lists that house **once**, and `-levels` at an empty directory leaves no New set at all —
which is the crisper half, because an empty directory adds nothing and could only remove the built-in
houses by replacing them.

**`go test ./assets` is the only thing in `make check` that can see any of this, which is why two
tests were the point of the commit.** Nothing under `internal/` can reach the embedded levels root — `internal/shell` takes its sources from its
caller, `internal/fidelity` builds its own — so `go test ./assets` is the entire coverage, and both
likely ways to ship it wrong are silent all the way to the player: a wrongly-prefixed archive lists
no new house and errors nowhere, and a stale one ships a house that is not the house in the text.
`TestLevelArchiveHoldsEveryAuthoredHouseAtItsRoot` requires the root to hold exactly the authored
names and no directories; `TestLevelArchiveIsFresh` rebuilds every source with the same encoder
`glidertool house build` uses and compares the bytes. Neither skips. `make levels` gained the same
check from the other end (`packassets -check`), `make levels-zip` repacks, and the release workflow
now derives the expected house count from the two source trees rather than hard-coding it and greps
each house's name out of the packaged executable for both platforms.

**A house inside the executable can only be named, never attached, and one tool could not take a
name.** `glidergo -house "Open House"` played it from the first commit, but `glidertool replay -house
"Open House"` answered `open Open House.house: file does not exist` — a file nobody on earth has,
about the one kind of house whose whole point is that there is no file. `replay.Script` now carries
the levels root as well (`Levels`, `LevelDir`, the `leveldir` script keyword, `glidertool replay
-levels DIR`) and resolves a name in the houses root first and the levels root second, which is
`cmd/glidergo`'s order and the tie the picker's sort breaks the same way: an original wins a name it
shares with a new house. Only a missing file falls through to the second root — a house that is there
and will not parse is still the answer — and when neither root has it the error names both instead of
repeating the file name back. That last part matters for the four flags too: a mistyped `-art` used
to be told to run `make assets`, in a source tree a player does not have.

`-version`'s root rows now give two different remedies, because "missing" was two different
failures sharing one sentence: a `built-in:` label that will not resolve is a broken build and names
the make target that rebuilds *that* archive (`make levels-zip` for the levels row, which the old
single message sent to `make assets`), while a path is a directory a flag named and there is nothing
to rebuild. `assetfs.Built` is the one-line predicate that tells them apart.

Three things in the `Makefile` that this change put a foot through. `fmt-check` checked `cmd` and
`internal` while `fmt` reformatted `$(PKG)`, so an unformatted `assets/assets.go` or
`tools/packassets/main.go` passed `make check` and was then rewritten by the next unrelated `make
fmt`; it now checks all four directories that hold Go, for 0.1 s. `make levels` did not clear its own
output, so renaming a house left the old build in `assets/levels/` where it was indistinguishable
from a house — the freshness check then failed and its remedy, `make levels-zip`, would have packed
the orphan *into* the archive. And `check` now asks the `embedded` guard first, so a checkout missing
an archive gets the sentence with `git checkout --` in it rather than the compiler's `pattern
levels.zip: no matching files found` from `vet`, three steps earlier than the guard used to run.

`.gitattributes` describes both archives, and the `-merge` it now states explicitly is a restatement
rather than a fix: `binary` is a macro for `-diff -merge -text`, so `assets/extracted.zip` was
already unmergeable and its effective attributes have not changed. What is written down is the
failure that rule prevents — git line-merging two differing zips writes conflict markers into a
deflate stream, and the result is a file that exists, has the right name and panics at init in every
entry point — because that is the thing somebody relaxing the rule needs to know.

**One field of an authored house is now a promise to strangers.** A saved game is refused unless the
house's `timestamp` still matches the one the save carries, so shipping a house publishes that
number: editing it later refuses every save any player has made in it, and nothing else notices,
because the house still lints, still finishes and still plays. It is pinned by
`TestShippedHousesKeepTheStampTheirSavesAreKeyedOn` — which fails just as loudly for a house with no
row as for a changed number — stated in a comment on the field itself, in CONTRIBUTING's Houses
section, and argued in 4.19. High scores are keyed on the house's name alone, so they survive a
timestamp change and do not survive a rename.

The About box says it too, and deliberately does not add up: *"the 1994 art, sounds and 22 houses ship
inside it, with the houses this port adds"*. The count stays the 1994 set's, because the box is the
only documentation a downloaded executable carries and a single number would be the box claiming 1994
wrote all of them — which is the thing 4.14 exists to prevent. That is the one changed screen in
`internal/fidelity/testdata/screens.hashes`.

### Open House: the first house of our own, and the sixty pixels that decide what a house can ask for (2026-09-22)

`levels/Open House.house.txt` is 43 rooms on a 7×10 grid, the first house this port wrote. `make
levels` compiles every `levels/*.house.txt` with `glidertool house build` and lints the result, and
`glidergo -levels assets/levels` puts them on the shelf as the **New** set — where it lists as "Open
House · 43 rooms" between "Nemo's Market" and "Rainbow's End", because a house is named by its file
and `houseType` has no name field. (That flag was the *only* way to see it on the day this landed.
The entry above is the one that put it inside the binary, and it also changed what the flag means:
`-levels DIR` now replaces the built-in set rather than adding to it.)

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
of ours has no way to carry art yet — the levels archive holds houses and nothing else, and
`-houseart DIR` replaces the one art root rather than adding to it, so giving a new house its own
pictures takes the twenty-two originals' away (4.15). And §10.3's per-room object budgets are
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

Where the new houses that ship *inside* the executable live was filed rather than pre-built here:
`assets/extracted/houses/` is barred by two contracts that exist for good reasons (`make
assets-check`'s recursive diff and `assetpack.Compare`), `//go:embed` cannot reach outside `assets/`,
and the recommendation was a second archive packed by the `tools/packassets` that already exists. See
4.14 — and the entry above, which is that archive, built two commits later. On the day of this one
`-levels DIR` was the whole of the New set, which was enough to author against and not enough to
ship.

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
