// Package assets is the 1994 data, built into the executable.
//
// Every binary this repository produces carries the whole of assets/extracted inside it: the
// art, the sounds, the score, all 22 houses, the per-house resource forks and the resource
// odds and ends the tools read. A player downloads one file, runs it from wherever it landed,
// and the game finds everything it needs without a directory beside it. That is the point of
// this package and it is the whole of it.
//
// # Why an archive and not a tree of //go:embed patterns
//
// Because three of the houses cannot be embedded by name. "Castle o' the Air", "Nemo's Market"
// and "Rainbow's End" have an apostrophe in them, go:embed accepts only names that are valid
// module file paths, and that rule forbids it -- a named pattern fails the build and a directory
// pattern silently leaves the file out. internal/assetpack has the full argument, including why
// renaming the files was the wrong way out. The upshot is that the bytes travel as a zip, whose
// entry names are the 1994 names exactly, and archive/zip hands them back as an fs.FS.
//
// # Why the package lives here and not under internal/
//
// A //go:embed pattern cannot name anything outside its own package directory, and the archive
// is built from assets/extracted -- where the Makefile writes it, where .gitignore knows about
// it, and where some 9,500 citations in docs/ point. So the package that embeds it is assets/
// itself. Nothing here is meant to be imported by anybody else; the import graph that matters is
// the other direction, and it is deliberate:
//
//   - This package is imported by cmd/glidergo and cmd/glidertool and by nothing else. Every
//     package under internal/ takes an fs.FS from its caller and has no idea whether the bytes
//     came from a disk or from the executable.
//   - So no test binary links 11 MiB it will never read. The tests read the tree from the
//     working copy, which is where a developer's changes to it actually are.
//
// # Overriding it
//
// The four asset flags -- -art, -houseart, -sounds, -houses -- are empty by default, which
// means "the copy in this binary". Give one a directory and that directory is read instead,
// which is what the extractor's own workflow needs (`make assets` writes a tree and
// `make assets-check` compares it) and what a player pointing the game at houses of their own
// will need. internal/assetfs is where that choice is made.
//
// The override is not a search path: a named directory replaces the built-in root rather than
// layering over it. Union semantics -- drop one house in a directory and see 23 -- is a real
// feature and is Stage 2's to design, not something to fall out of a resolution order nobody
// wrote down (docs/IMPROVEMENTS.md 5.3).
package assets

import (
	_ "embed" // for the //go:embed directive below
	"io/fs"

	"github.com/bwenstar/gliderGo/internal/assetpack"
)

// The archive, and the one place in the repository that names it.
//
// It is a build input, so `go build` refuses a checkout with the archive deleted rather than
// producing a binary that cannot draw -- which is the check CI used to make with a shell script.
// It is **not** rebuilt by the build, though, which the loose patterns it replaced were: after
// `make assets` regenerates the tree, `make assets-zip` is what carries the change into the
// executables, and `go test ./assets` is what says so if it was forgotten.
//
//go:embed extracted.zip
var archive []byte

// tree is the archive read as a filesystem. Names in it are the names every asset path in this
// repository uses: "art/ui/1000.png", "houses/Castle o' the Air.house".
//
// A panic, because there is no sensible second answer. The archive is committed and its bytes
// are in the executable; if archive/zip cannot read it then the build is broken in a way no
// caller can do anything about, and a game that quietly started with no art would be a worse
// bug report than a stack trace. It happens at init and therefore on every entry point, which
// is what makes it impossible to ship.
var tree = func() fs.FS {
	fsys, err := assetpack.Open(archive)
	if err != nil {
		panic("assets: the built-in " + assetpack.Name + " will not read: " + err.Error())
	}
	return fsys
}()

// Tree is the built-in copy of assets/extracted.
//
// It is read-only, safe for concurrent use, and costs nothing to hand out: the bytes are in
// the executable's data whether anybody asks for them or not. Reading a file out of it inflates
// that file, so a caller that reads one repeatedly should keep what it got -- which every
// caller here already does, the art having been cached since 1.3.
func Tree() fs.FS { return tree }
