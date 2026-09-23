# Security policy

## Reporting

Use GitHub's private vulnerability reporting: **Security → Report a vulnerability** on
https://github.com/bwenstar/gliderGo. That keeps the report private until there is a fix and it
reaches the maintainer directly.

One maintainer, no SLA. Expect an acknowledgement rather than a schedule. If something is both
serious and being exploited, say so in the title.

## What is actually at risk

This is worth being specific about, because a game with one optional network mode and no
privileges has a small and unusual attack surface, and a vague policy would obscure where it is.

**Parsing other people's files is the whole of it.** `glidergo` and `glidertool` read house files —
a 1994 binary format with fixed-size records, offsets and counts written by a Mac application that
trusted them. A house is a *document people share*: the original shipped 22 of them by five
different authors and players swapped more. So a malicious `.house` reaching a player is a realistic
path, not a theoretical one, and the code it reaches is `internal/house` (the binary decoder, the
text format, saved games and score files) plus `internal/render` and `internal/audio` for the
resource forks and `'snd '` resources beside them.

Go bounds-checks, so the likely outcome of a malformed file is a panic rather than a corrupted
stack. A panic on a hostile input is still a bug and still worth reporting. What would be more
serious: an allocation driven by an unvalidated count in the header, a decode loop that does not
terminate, or a path taken from file contents and used to open something.

**Size limits.** Sizes are checked before anything is allocated for them. A picture is refused
from its header if it is over 4096 pixels on a side or 4 megapixels in all. A house file is
refused past 11,403,784 bytes, which is 32,767 rooms, the most its header can count. That check
is on the bytes actually read, so a file that never ends is refused too. A house text is refused
at its 32,768th room. A file that gets past these to a large allocation is a bug worth reporting.

**Fuzzing.** These readers have fuzz targets: the house file and its text, saved games, score
boards, replay scripts, pictures, a house's own sounds, and the race's reader and handshake.
`make fuzz` runs them. If you find an input that makes one panic or hang, attach the input to
your report. It is the most useful part.

Also in scope, more narrowly: the asset extractors in `tools/` (python3, run over the vendored
1994 data — not over untrusted input in normal use), and the release workflow in `.github/`.

**The race's network connection is the other half.** A networked race (`-host`, `-join`, or
`Race...` on the title screen) is the only time either program touches a network, and it is off
unless a player starts one:

- **Hosting listens.** TCP on every interface, IPv4 and IPv6, on port 1994 or `-port`, and only
  while the waiting screen is up. The listener closes once a race is agreed, and when the player
  gives up. One connection is dealt with at a time. A host sends its hello first, so anything that
  connects learns the house name and hash, the release, the rules and an engine fingerprint before
  it has sent a byte. A connection that is not a race is dropped (a silent one after five seconds),
  and the host goes on waiting. So something that keeps connecting can keep a host from racing,
  five seconds at a time. That is known, and Escape is the answer to it.
- **Joining dials** the address the player typed, and nothing else.
- **What a peer can send.** Frames are capped at 1 MiB (`netplay.MaxMsg`), checked before anything
  is allocated. Every field that decides something is validated: slot, state, match ID, lengths,
  the agreed seed and input delay. Fields that are only reported, such as the other side's house
  name and release, are taken as sent. They reach the terminal through `%q` and the screen quoted
  and cut to length. No file ever crosses the wire, and nothing a peer sends is used as a path. The
  two ends are symmetric: a hostile host can send a guest exactly what a hostile guest can send a
  host. So the code in scope is `internal/netplay`'s decoding and handshake, and the places in
  `cmd/glidergo` that show what a peer said.
- **The house check is not authentication.** Both machines hash the house they opened and refuse
  to race if the hashes differ. That catches two honest players with different files. It is not a
  password: a peer that has seen a host's hello can send the same hash back. Results are on the
  honour system too, since each side reports its own run.
- **Port forwarding is exposure.** The README explains how to race across the internet, and that
  advice means opening port 1994 to anyone who finds it while a host is waiting. What they reach is
  the handshake above and no more. It is still a listening socket on the internet, so close the
  forward when the race is over. A VPN between the two machines, such as Tailscale, avoids the
  forward altogether.

A bug in any of that is in scope and worth reporting. That means a panic on bytes a peer sent, an
allocation it controls, a single connection holding a host longer than those five seconds, or a
peer's string reaching the screen or the terminal unescaped.

**Not in scope, because it does not exist:** there is no server, no update check, no telemetry, no
browser or scripting engine, and nothing runs with elevated privileges. Outside a race nothing
listens or dials. The game reads and writes its own preferences, saves, high scores and crash
report, under the user's config and data directories, and touches nothing else. The crash report
is never sent anywhere; it stays on the machine until a player attaches it to something. Files are read through Go's `fs.FS`, so a
path inside a house cannot escape the directory it came from. Symbolic links inside a directory
passed to `-levels`, `-houseart` or `-art` are followed, though, like any other file in it.

## Versions

Fixes land on `main` and go out in the next release. There are no maintained release branches — the
project is young enough that "upgrade to the latest release" is the entire support matrix.

A Go security fix that reaches gliderGo is reason enough for a release on its own, with nothing
else in it. "Reaches" is `govulncheck`'s word: a vulnerability in a standard-library function the
game calls, such as the dialer, the listener, or the PNG and zip readers. CI runs `govulncheck`
every week, as well as on every push, so a Go security release between two tags is noticed.

## Dependencies

There are none. `go.mod` has no `require` block, by design and enforced by a test
([CONTRIBUTING.md](CONTRIBUTING.md) rule 1), so the usual supply-chain question has an unusually
short answer: the only third-party code in a gliderGo binary is the Go standard library, and the
only non-Go dependency is libX11 on Linux, dynamically linked.

That makes the Go toolchain the supply chain. Releases are built with Go 1.27, the newest patch
release of it when the release is made: `go version -m glidergo` names the exact one, and so do the
release notes. That is newer than the `go 1.23` in `go.mod`, which is only the oldest Go that can
build the code, and which Go no longer patches. A release is not published unless
`govulncheck ./...` reports nothing the game calls, scanned as Linux, as Windows and as macOS. When
Go stops patching 1.27, releases move to a newer minor. If you build gliderGo yourself, use a Go
that is still patched too: built with Go 1.23.12, the game reaches three standard-library
vulnerabilities, in the dialer, a directory walk and the zip reader, whose fixes 1.23 never
received ([docs/IMPROVEMENTS.md](docs/IMPROVEMENTS.md) 5.11).

What ships inside the binary besides the code is data — the 1994 art, sounds and houses, embedded
from `assets/extracted.zip`, which `make assets-check` verifies is exactly what `tools/` produces
from the vendored originals, plus this port's own houses, embedded from `assets/levels.zip`, which
`go test ./assets` verifies is exactly a build of the text in `levels/`.
