# Security policy

## Reporting

Use GitHub's private vulnerability reporting: **Security → Report a vulnerability** on
https://github.com/bwenstar/gliderGo. That keeps the report private until there is a fix and it
reaches the maintainer directly.

One maintainer, no SLA. Expect an acknowledgement rather than a schedule. If something is both
serious and being exploited, say so in the title.

## What is actually at risk

This is worth being specific about, because a game with no network code and no privileges has a
small and unusual attack surface, and a vague policy would obscure where it is.

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

Also in scope, more narrowly: the asset extractors in `tools/` (python3, run over the vendored
1994 data — not over untrusted input in normal use), and the release workflow in `.github/`.

**Not in scope, because it does not exist:** there is no network listener, no server, no update
check, no telemetry, no browser or scripting engine, and nothing runs with elevated privileges. The
game reads and writes its own preferences, saves and high-score files under the user's config
directory and touches nothing else. Stage 3 of the plan adds networked two-player, and this section
will need rewriting when it does.

## Versions

Fixes land on `main` and go out in the next release. There are no maintained release branches — the
project is young enough that "upgrade to the latest release" is the entire support matrix.

## Dependencies

There are none. `go.mod` has no `require` block, by design and enforced by a test
([CONTRIBUTING.md](CONTRIBUTING.md) rule 1), so the usual supply-chain question has an unusually
short answer: the only third-party code in a gliderGo binary is the Go standard library, and the
only non-Go dependency is libX11 on Linux, dynamically linked. What ships inside the binary besides
the code is data — the 1994 art, sounds and houses, embedded from `assets/extracted.zip`, which
`make assets-check` verifies is exactly what `tools/` produces from the vendored originals.
