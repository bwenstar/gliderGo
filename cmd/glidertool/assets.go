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

// houseArtRoots is the search list for one house's own pictures: -houseart first because it is
// the explicit one, then the port's own houses' art, then the extracted 1994 forks.
//
// A list rather than a root, and the whole of docs/IMPROVEMENTS.md 4.15 is in that: -houseart used
// to be the only way to hand a house art of its own, and it worked by *replacing* the root that
// half the twenty-two shipped houses need. Searching costs nothing, since the lookup was always by
// house name, and it is what lets `glidertool house lint` check a house of this port's own against
// this port's own pictures without pretending Titanic has none.
//
// glidertool has no -levels flag and does not need one to search the levels art: the tool is
// shipped in the same archive as the game and carries both embedded trees, so the middle root is
// always the built-in one. Somebody linting art they have just written, before `make levels-zip`
// has put it in the archive, names it with -houseart -- which is what the Makefile's own `levels`
// target does, and why a *named* root that misses is worth a line while a built-in one is not.
func houseArtRoots(dir string) assetfs.ArtRoots {
	roots := make(assetfs.ArtRoots, 0, 3)
	if dir != "" {
		roots = append(roots, assetfs.NamedArt(dir))
	}
	if sub := assetfs.Sub(assets.Levels(), houseArtDir); sub != nil {
		roots = append(roots, assetfs.ArtRoot{FS: sub,
			Label: assetfs.Name(assetfs.Label("levels"), houseArtDir)})
	}
	if tree := assets.Tree(); tree != nil {
		roots = append(roots, assetfs.ArtRoot{FS: assetfs.Sub(tree, houseArtDir),
			Label: assetfs.Label(houseArtDir)})
	}
	return roots
}

// houseArtDir is the subdirectory holding per-house forks, in both trees that have one: it is
// `houseart` under assets/extracted and `houseart` at the top of the levels root. One spelling,
// because the second was chosen to match the first. cmd/glidergo has the same constant for the
// same reason neither binary imports the other's package.
const houseArtDir = "houseart"

// builtinTree is the whole built-in tree, for the one caller that hands it on rather than
// resolving a root out of it: a replay.Script carries it so that internal/replay can resolve the
// same four roots the same way, without importing the assets package either.
func builtinTree() fs.FS { return assets.Tree() }

// builtinLevels is the second archive, the port's own houses, handed on the same way.
//
// It is a separate function and not a fifth `sub` of assetRoot because it is a separate archive:
// the houses sit at its root rather than under a directory name (internal/assetpack.LevelsName),
// which is what internal/assetfs.Whole is for. The one caller is `replay`, and the reason it has
// one is that a house shipped inside the executable can only be named, never attached.
func builtinLevels() fs.FS { return assets.Levels() }
