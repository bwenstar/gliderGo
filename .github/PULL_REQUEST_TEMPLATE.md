<!--
Thanks. Keep whatever is useful and delete the rest -- this is a checklist, not a form, and a
one-line PR for a one-line fix is fine.

CONTRIBUTING.md has the long version of everything below.
-->

## What this changes

<!-- And why. If it fixes an issue, "Fixes #N". -->

## If it changes what the game does

<!--
Delete this section if it doesn't. If it does, the one thing worth saying is which 1994 code you
read: a function name in Sources/*.c is worth a paragraph of prose, and every citation in docs/ is
relative to upstream commit 94fed96.

If this diverges from the original on purpose, say so here and add an entry to
docs/IMPROVEMENTS.md -- that file is how the next person knows it was a decision.
-->

## Checks

- [ ] `make check` passes (and I read the caveat list it prints at the end)
- [ ] No new non-stdlib imports — `go.mod` still has no `require` block
- [ ] If pixels moved: `go test ./internal/fidelity -update`, and the new hash is **in this PR**
- [ ] If a house changed or is new: `glidertool house lint` is clean
- [ ] Commit messages say why, not just what
