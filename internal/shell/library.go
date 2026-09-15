package shell

// Finding the houses, and putting them in the order the original put them in.
//
// The original's discovery is a breadth-first walk of the application's volume
// looking for Finder type 'gliH' and creator 'ozm5', capped at 32 directories and
// at maxFiles (48) houses, with the results de-duplicated and bubble-sorted by
// name (SelectHouse.c:516-648, docs/analysis/ui-dialogs.md 7.2-7.3). Three of
// those five things are gone here and two are kept.
//
// Gone: the type-and-creator test, which no other file system has -- house.PeekFile
// replaces it with a sniff of the header, as docs/analysis/ui-dialogs.md P5 sets out.
// The 32-directory cap, which existed to bound a scan of a whole volume; this walks
// one directory tree, so the bound it needs is the tree. And the 48-house cap, which
// existed because theHousesSpecs was a fixed-size NewPtr allocation made before any
// UI existed (§7.1); a slice has no such reason.
//
// Kept: the sort, exactly, because it decides the order a player sees and because
// the port's own new houses (Stage 2) will be listed alongside the originals and
// should fall where the originals would have put them. And the *reporting* of what
// was rejected, which the original does only as a single yellow alert when nothing
// at all was found -- a file that looked like a house and was not simply never
// appeared, which is the least helpful thing a picker can do (docs/IMPROVEMENTS.md
// 2.33).
//
// One original behaviour is deliberately not reproduced. SortHouseList's duplicate
// removal collapses two houses with the same name in different folders into one,
// through a pair of self-comparison bugs at SelectHouse.c:527-528 that reduce its
// predicate to "same name". There is nothing to reproduce: names here are paths, a
// path is unique, and two files called "Demo House" in two directories are two
// houses. Both are listed.

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"glidergo/internal/house"
)

// House is one entry in the picker: a file that sniffed as a house, and what its
// header said.
//
// Name is the file's name with any extension removed, because that is what a house
// is called: houseType has no name field, and every place the original shows a
// house name it is reading the file's (see cmd/glidergo, and HouseIO.c's
// thisHouseName). Rel is the path relative to the library root and is what
// distinguishes two houses that share a name.
type House struct {
	Name   string
	Rel    string
	Path   string
	Rooms  int16
	Locked bool // the editor cannot change it; the low bit of the timestamp
	Demo   bool // named "Demo House": the original's attract-mode house
	Size   int64
	Banner string
	Scores house.Scores
}

// Skip is a file that looked like it was meant to be a house and was not, kept so
// that the picker can say how many and why rather than silently listing fewer
// houses than the directory holds.
type Skip struct {
	Path string
	Why  error
}

// Library is a directory tree's worth of houses, sorted.
type Library struct {
	Root    string
	Houses  []House
	Skipped []Skip
}

// DemoHouse is the file name the original's attract mode looks for, verbatim
// (SelectHouse.c:641). demoHouseIndex is -1 when it is absent, which is the one
// thing that decides whether Options > Demo... is enabled (§3.5.2).
const DemoHouse = "Demo House"

// houseExts are the file names Discover will even open. The 1994 files have no
// extension at all -- on a Mac the type was metadata -- and the extraction writes
// ".house", so both have to count. Anything with some other extension is ignored
// without comment: a directory of houses is also where a README, a manifest and a
// stray screenshot end up, and reporting those as rejected houses would make the
// report useless.
var houseExts = map[string]bool{"": true, ".house": true, ".glh": true}

// Discover walks a directory tree and returns every house in it.
//
// A tree rather than a single directory because the port will have more than one
// source of houses -- the extracted originals, the new houses of Stage 2, and
// whatever the player drops in -- and because a directory is the obvious way to
// group them. Hidden directories are skipped, and so is the walk's own error on
// any single entry: one unreadable file in a tree of houses is not a reason to
// have no house list.
func Discover(root string) (*Library, error) {
	lib := &Library{Root: root}
	if root == "" {
		return lib, errors.New("shell: no houses directory")
	}
	if st, err := os.Stat(root); err != nil {
		return lib, err
	} else if !st.IsDir() {
		return lib, errors.New("shell: " + root + " is not a directory")
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // an unreadable entry, not an unreadable tree
		}
		if d.IsDir() {
			if path != root && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		if !houseExts[strings.ToLower(filepath.Ext(d.Name()))] {
			return nil
		}
		sum, err := house.PeekFile(path)
		if err != nil {
			lib.Skipped = append(lib.Skipped, Skip{Path: path, Why: err})
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		name := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		lib.Houses = append(lib.Houses, House{
			Name:   name,
			Rel:    rel,
			Path:   path,
			Rooms:  sum.NRooms,
			Locked: !sum.Unlocked,
			Demo:   name == DemoHouse,
			Size:   sum.Size,
			Banner: sum.Banner,
			Scores: sum.Scores,
		})
		return nil
	})
	if err != nil {
		return lib, err
	}

	lib.Sort()
	return lib, nil
}

// Sort puts the list in SortHouseList's order.
func (l *Library) Sort() {
	sort.SliceStable(l.Houses, func(i, j int) bool {
		if l.Houses[i].Name != l.Houses[j].Name {
			return NameFirst(l.Houses[i].Name, l.Houses[j].Name)
		}
		// Two houses of the same name in different directories. The original
		// loses one of them; this keeps both and orders them by where they came
		// from, so the list is stable and the picker can show the difference.
		return l.Houses[i].Rel < l.Houses[j].Rel
	})
	sort.SliceStable(l.Skipped, func(i, j int) bool { return l.Skipped[i].Path < l.Skipped[j].Path })
}

// Find returns the index of the house with this name, or -1.
//
// Name and not path, because that is what a preference file, a command line and a
// high-score board all record, and it is what DoDirSearch matches thisHouseName
// against on the way out (§7.2 step 5). The comparison is EqualString(a, b, false,
// true): case-insensitive.
func (l *Library) Find(name string) int {
	for i := range l.Houses {
		if strings.EqualFold(l.Houses[i].Name, name) {
			return i
		}
	}
	return -1
}

// NameFirst is WhichStringFirst(a, b) == 2 (StringUtils.c:34-87): does a sort
// before b?
//
// The original's rules, all three of them: ASCII lowercase folds to uppercase, so
// the order is case-insensitive; on a common prefix the shorter string sorts
// first; and bytes at or above 0x80 are not folded and compare by value, which is
// why an accented capital sorts after every unaccented letter rather than beside
// its base.
//
// It compares bytes, and the bytes are not quite the Mac's. A Mac file name is Mac
// Roman, so "École" is six bytes beginning 0x83; the extracted file name is UTF-8,
// so it is seven beginning 0xC3 0x89. Both sort after "Zoo" and neither sorts
// where a modern collation would put it, so the *shape* of the original's order
// survives the encoding change even though a specific pair of accented names could
// swap. No shipped house name has a byte above 0x7F, so nothing in the original
// set is affected.
func NameFirst(a, b string) bool {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		ca, cb := upperASCII(a[i]), upperASCII(b[i])
		if ca != cb {
			return ca < cb
		}
	}
	return len(a) < len(b)
}

// upperASCII is the original's fold, byte for byte: `if (c > 0x60 && c < 0x7B) c -= 0x20`.
func upperASCII(c byte) byte {
	if c > 0x60 && c < 0x7B {
		return c - 0x20
	}
	return c
}
