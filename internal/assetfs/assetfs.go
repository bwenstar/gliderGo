// Package assetfs decides where one asset root's bytes come from.
//
// There are two answers -- a directory on this machine, or a subtree of the copy built into the
// executable -- and every caller that has to choose between them does it here, in one function,
// so that the rule is written down once:
//
//	an empty directory means the built-in copy.
//
// That is why the five asset flags default to "" rather than to "assets/extracted/art" and
// friends. A default of a path would have made a repository-relative directory the thing the
// game needs to find, which is the one thing a downloaded binary cannot rely on
// (docs/IMPROVEMENTS.md 5.3).
//
// Five, not four, and the fifth arrives by a different door: -levels resolves against a second
// embedded archive rather than a subdirectory of the first, which is what Whole is for.
//
// One root is one answer, with one exception: the per-house art. A house's own pictures are
// looked up by the house's *name*, so several roots can hold art for different houses without
// ever meaning two things by one path, and a new house must be able to carry pictures without
// taking the twenty-two originals' art away. That is ArtRoots, which searches rather than
// substitutes, and it is the only thing here that takes a list.
//
// Nothing here imports the assets package, and nor does anything else under internal/. The
// built-in tree arrives as an fs.FS argument from whichever command was built with it, so a
// test binary links none of it and reads the working copy instead -- which is where a
// developer's changes to an asset are.
package assetfs

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
)

// Root is the filesystem for one asset root, and a label naming it for messages.
//
// dir is the flag's value and wins when it is set: the caller asked for that directory and gets
// it, missing or not, so the error names the path they typed. Otherwise sub -- "art", "sound",
// "houses", "houseart" -- is taken out of tree.
//
// A nil tree with no dir gives a nil FS, which is not an error and is a state the port already
// had: it is what internal/render calls "no art tree", and every loader here treats it as an
// asset that is not there rather than a reason to stop. Tests use it deliberately.
func Root(tree fs.FS, dir, sub string) (fs.FS, string) {
	if dir != "" {
		return os.DirFS(dir), dir
	}
	if tree == nil {
		return nil, ""
	}
	return Sub(tree, sub), Label(sub)
}

// Whole is Root for a tree that is already the root.
//
// There are two embedded archives and they are shaped differently. assets/extracted.zip is one
// tree with the four roots inside it as subdirectories, so Root takes "art" or "houses" out of
// it. assets/levels.zip holds the port's own houses at its top level, because a root is a root
// and there was nothing to put around them (internal/assetpack.LevelsName). Root(tree, dir, ".")
// resolves that correctly and then labels it "built-in:.", which puts a dot in every message
// about a house in it -- `built-in:./Open House.house`. So the label is a word the caller passes
// rather than a path element taken out of the tree, and the word is the flag's name.
//
// dir wins and a nil tree gives a nil FS, both exactly as in Root: the rule that an empty
// directory means the built-in copy is the same rule.
func Whole(tree fs.FS, dir, name string) (fs.FS, string) {
	if dir != "" {
		return os.DirFS(dir), dir
	}
	if tree == nil {
		return nil, ""
	}
	return tree, Label(name)
}

// ArtRoot is one place a house's own PICTs may sit: the houseart root of some tree, plus the
// label that names it on a terminal.
//
// Named says the root came from a flag rather than from the built-in tree, and it is carried here
// because it changes what a *miss* means. A built-in root that has no fork for a house is the
// ordinary case -- nine of the twenty-two shipped houses carry no PICTs at all -- and is worth no
// words. A directory somebody typed that has no fork for the house they are playing is worth one
// line, because the commonest reason to type it is to test an extraction, and a search that
// quietly fell through to the built-in copy would turn a half-extracted tree into a tree that
// looked complete (docs/IMPROVEMENTS.md 4.15).
type ArtRoot struct {
	FS    fs.FS
	Label string
	Named bool
}

// ArtRoots is the ordered search list for one house's art: first hit wins.
//
// A list and not a root, because a new house has to be able to carry pictures without taking the
// twenty-two originals' art away, and -houseart used to be the only way to hand a house art of
// its own -- by *replacing* the root that half the shipped houses need. Searching instead of
// substituting costs nothing, since the lookup was already per house by name and two houses of
// the same name in two roots is a collision the picker has to resolve anyway
// (docs/IMPROVEMENTS.md 4.15).
type ArtRoots []ArtRoot

// Fork is where one house's own PICTs were found, and what to say about it.
type Fork struct {
	// FS is the house's own subtree -- pict/<id>.png and bnds/<id>.bin -- or nil if no root in
	// the list holds one. Nil is what render.Assets.OpenHouseResFork treats as "this house has
	// no art of its own", which is a legitimate state and not an error.
	FS fs.FS
	// Label names it for cache keys and messages: "built-in:houseart/Titanic". When nothing was
	// found it names every root that was searched, so the message says where to put a fork
	// rather than only that there is not one.
	Label string
	// Passed are the labels of the *named* roots searched before the one that answered, in
	// order. Empty in every ordinary run: it takes a flag to fill it.
	Passed []string
}

// Found reports whether any root in the list holds this house's fork.
func (f Fork) Found() bool { return f.FS != nil }

// Complaint is what to say out loud about this lookup, or "" for the usual case of nothing.
//
// wants is whether the house names a picture only its own fork could supply
// (house.House.WantsOwnArt). It decides the whole of the "found nothing" message, and that is
// docs/IMPROVEMENTS.md 4.17: warning on "there is no fork here" rather than on "there is no fork
// here *and* this house needed one" meant that `Open House` -- drawn entirely with built-in
// backgrounds, and so the one house in the library that provably needs no warning -- was the only
// house to get one on a complete asset tree.
//
// The words live here, in one place, because the two commands that mount a fork used to carry a
// copy each and 4.17 had to name both files.
func (f Fork) Complaint(house string, wants bool) string {
	switch {
	case !f.Found() && !wants:
		return ""
	case !f.Found():
		return fmt.Sprintf("%s has no resource fork in %s, so its custom art will fall back",
			house, f.Label)
	case len(f.Passed) > 0:
		// A flag named a root that does not have this house, and something else did. Said even
		// when the house wants no art, because the claim being checked is about the directory
		// rather than about the house: this is how a half-extracted tree is caught being passed
		// off as a complete one.
		return fmt.Sprintf("no resource fork at %s; using %s",
			strings.Join(f.Passed, ", "), f.Label)
	}
	return ""
}

// NamedArt is the root a flag pointed at: the value *is* the root and there is nothing to resolve
// out of a tree, which is Root's `dir != ""` arm without the argument it has no use for. It is
// here rather than at the call site so that os.DirFS stays inside the package whose whole subject
// is which of the two it should be.
func NamedArt(dir string) ArtRoot { return ArtRoot{FS: os.DirFS(dir), Label: dir, Named: true} }

// Fork searches the list for the house's own art and reports what it found.
//
// The test is IsDir and not "does pict/ exist", because that is the test this lookup has always
// made and a fork holding only 'bnds' records is a real thing: `bnds` describes which sides of a
// *built-in* background are room boundaries, so a house can carry one and no pictures at all.
func (r ArtRoots) Fork(house string) Fork {
	var passed, searched []string
	for _, root := range r {
		if root.FS == nil {
			continue
		}
		searched = append(searched, root.Label)
		if IsDir(root.FS, house) {
			return Fork{FS: Sub(root.FS, house), Label: Name(root.Label, house), Passed: passed}
		}
		if root.Named {
			passed = append(passed, Name(root.Label, house))
		}
	}
	return Fork{Label: strings.Join(searched, ", ")}
}

// Label is how a built-in root is named on a terminal: "built-in:art".
//
// It is deliberately not a path. Somebody reading "no extracted resource fork at
// built-in:houseart/Titanic" should not go looking for a directory of that name, and somebody
// reading it in a bug report should be able to tell at a glance that the binary was carrying
// its own assets.
func Label(sub string) string { return builtIn + sub }

const builtIn = "built-in:"

// Built reports whether a label came from Label rather than from a flag.
//
// One caller, and it is worth the function: `-version` prints a line per root and "missing" means
// two unrelated things there. A built-in label that will not resolve is a broken build, and the
// remedy is a make target in a source tree. A path is a directory somebody typed, and the remedy
// is to look at what they typed -- telling them to rebuild an archive would send a player who has
// only the executable somewhere they cannot go.
//
// A flag whose value happens to begin "built-in:" would be misread here. It would also be a
// directory of that name, relative to the working directory, and os.DirFS would not find it.
func Built(label string) bool { return strings.HasPrefix(label, builtIn) }

// Sub is fs.Sub without the error, which no caller here can act on: the names this package
// passes are compile-time constants and a house name that has already been read out of a
// directory listing. A name that is somehow not a valid fs path gives a nil FS, which reads as
// "not there".
func Sub(fsys fs.FS, name string) fs.FS {
	if fsys == nil {
		return nil
	}
	sub, err := fs.Sub(fsys, name)
	if err != nil {
		return nil
	}
	return sub
}

// IsDir reports whether name is a directory in fsys. It is the test the per-house resource fork
// lookup makes before opening one, and it answers false for a nil FS.
func IsDir(fsys fs.FS, name string) bool {
	if fsys == nil {
		return false
	}
	st, err := fs.Stat(fsys, name)
	return err == nil && st.IsDir()
}

// Exists reports whether name is a readable file in fsys.
func Exists(fsys fs.FS, name string) bool {
	if fsys == nil {
		return false
	}
	st, err := fs.Stat(fsys, name)
	return err == nil && !st.IsDir()
}

// Measure counts the files under fsys and adds up their sizes, for -version to report what a
// build is carrying. An unreadable tree measures zero rather than failing: this is a line of
// diagnostic output, and the diagnosis is the zero.
func Measure(fsys fs.FS) (files int, bytes int64) {
	if fsys == nil {
		return 0, 0
	}
	fs.WalkDir(fsys, ".", func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		files++
		bytes += info.Size()
		return nil
	})
	return files, bytes
}

// Name joins a label and a name for a message. It is path.Join, so it is right for both a
// built-in label and a slash-separated directory, and it is wrong for a Windows path with
// backslashes in it -- which is a message rather than a path anybody opens.
func Name(label, name string) string {
	if label == "" {
		return name
	}
	return path.Join(label, name)
}
