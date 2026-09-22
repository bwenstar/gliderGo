// Package project is what gliderGo says about itself.
//
// It exists because the same handful of facts -- what this is called, where it lives, what it
// is a port of, and under what licence -- have to appear in the About box, in `-version`, in
// both tools' usage text, in the issue templates and in the README, and a fact spelled out
// separately in six places is a fact that will eventually disagree with itself. A player
// photographing the About box to file a bug is the case that matters: the URL on their screen
// has to be the URL that works.
//
// It is deliberately data and nothing else -- no formatting, no os, no fmt -- so that every
// one of those six callers can import it without dragging anything along. The one thing it
// does not hold is the build version, which is not a property of the project: it is linked in
// at build time by the Makefile's `-X main.version` and travels as a parameter.
//
// project_test.go sweeps the tree and requires that every place these strings appear uses the
// same ones, which is the half that makes this worth a package rather than a comment. It also
// requires that each constant here is read by something: a fact with no reader is not being
// shared, and sharing is the entire argument for this package existing.
package project

const (
	// Name is the port. Lower-case g, capital G: it is a Go port of Glider, and the
	// spelling is load-bearing in the module path, the binary name and the archive names a
	// release attaches.
	Name = "gliderGo"

	// Module is what go.mod declares. internal/module's test owns the invariant that this
	// is the only non-stdlib import prefix in the tree; this copy is here so that Home and
	// the module path cannot drift apart without a test noticing.
	Module = "github.com/bwenstar/gliderGo"

	// Home is the project's page, and the string printed in the About box. No trailing
	// slash: it is concatenated below.
	Home = "https://" + Module

	// Issues is where a bug report goes. Named separately from Home because the About box
	// has room for one URL and the bug-report paths have room for the specific one.
	Issues = Home + "/issues"

	// Licence is the SPDX identifier. GPLv2 *only*: upstream's grant names version 2 with
	// no "or any later version" clause, and this port is a function-by-function
	// transcription and so unambiguously a derivative work. See README.md's Licence
	// section and docs/IMPROVEMENTS.md 1.2.
	Licence = "GPL-2.0-only"

	// Copyright is the port's own notice, spelled as README.md's Licence section spells it.
	// "the gliderGo authors" and not a person: the GPL asks that the notice be kept intact
	// and does not ask anybody to be a legal entity, and a name written into a constant is
	// harder to get out of a released binary than to put in.
	Copyright = "© 2026 the gliderGo authors"
)

// What this is a port of. Separate block because these are facts about 1994 rather than about
// gliderGo, and because the About box and the credits screen draw the line in the same place:
// the original's people are credited by internal/credits, which is checked against upstream's
// own README; what is here is only the handful of names that appear before the credits screen
// does.
const (
	// Original is the 1994 game, spelled as its own splash screen spells it.
	Original = "Glider PRO"

	// OriginalAuthor wrote it and later released the source, which is the only reason this
	// port is allowed to exist.
	OriginalAuthor = "John Calhoun"

	// OriginalPublisher shipped it.
	OriginalPublisher = "Casady & Greene"

	// OriginalYear is the year Glider PRO shipped, and the year the About box names.
	OriginalYear = "1994"

	// OriginalCopyright is the notice the 1994 program itself displays, byte for byte:
	// DITL 150 item 6, the statText across the bottom of the About dialog
	// (docs/analysis/ui-dialogs.md 10.6). It is carried rather than paraphrased because
	// GPLv2 section 1 asks that the notices be kept intact, and because the range is not
	// guessable -- the game shipped in 1994 and the notice runs to 2000, which is the last
	// year Casady & Greene revised it and five years after all but one of the houses.
	// project_test.go holds this against the document that transcribes the resource.
	OriginalCopyright = "© 1994-2000 Casady & Greene, Inc."

	// Upstream is the C this port was written against, and UpstreamCommit is the revision
	// every citation in docs/ is relative to. A citation into a moving target is not a
	// citation, which is why the commit is here rather than left implicit.
	Upstream       = "https://github.com/softdorothy/glider_pro"
	UpstreamCommit = "94fed96"
)
