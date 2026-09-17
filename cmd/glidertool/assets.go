package main

import (
	"io/fs"

	"github.com/bwenstar/gliderGo/assets"
	"github.com/bwenstar/gliderGo/internal/assetfs"
)

// assetRoot resolves one of the asset-root flags -- -art, -houseart, -houses, -sounds -- to the
// filesystem to read and a label to name it in messages. An empty flag means the copy of that
// root built into this executable, which is assetfs's rule and the game's.
//
// It exists so that neither render.go nor replay.go has to import the assets package. Both of
// them have a local variable called `assets` -- the render.Assets cache -- and a package of the
// same name in the same file would compile but would read as though one were the other. Keeping
// the import here also means this is the one place in the tool that knows the built-in tree
// exists at all.
//
// The tool takes the built-in assets rather than insisting on a tree beside it because it is
// shipped in the same release archive as the game: `glidertool render Foo.house` has to work in
// the directory a player unpacked, where there is no assets/extracted. The flags remain for the
// extractor's own workflow, which is the one case where the tree on disk is the thing under test.
func assetRoot(dir, sub string) (fs.FS, string) {
	return assetfs.Root(assets.Tree(), dir, sub)
}

// builtinTree is the whole built-in tree, for the one caller that hands it on rather than
// resolving a root out of it: a replay.Script carries it so that internal/replay can resolve the
// same four roots the same way, without importing the assets package either.
func builtinTree() fs.FS { return assets.Tree() }
