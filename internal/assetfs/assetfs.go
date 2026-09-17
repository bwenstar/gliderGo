// Package assetfs decides where one asset root's bytes come from.
//
// There are two answers -- a directory on this machine, or a subtree of the copy built into the
// executable -- and every caller that has to choose between them does it here, in one function,
// so that the rule is written down once:
//
//	an empty directory means the built-in copy.
//
// That is why the four asset flags default to "" rather than to "assets/extracted/art" and
// friends. A default of a path would have made a repository-relative directory the thing the
// game needs to find, which is the one thing a downloaded binary cannot rely on
// (docs/IMPROVEMENTS.md 5.3).
//
// Nothing here imports the assets package, and nor does anything else under internal/. The
// built-in tree arrives as an fs.FS argument from whichever command was built with it, so a
// test binary links none of it and reads the working copy instead -- which is where a
// developer's changes to an asset are.
package assetfs

import (
	"io/fs"
	"os"
	"path"
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

// Label is how a built-in root is named on a terminal: "built-in:art".
//
// It is deliberately not a path. Somebody reading "no extracted resource fork at
// built-in:houseart/Titanic" should not go looking for a directory of that name, and somebody
// reading it in a bug report should be able to tell at a glance that the binary was carrying
// its own assets.
func Label(sub string) string { return "built-in:" + sub }

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
