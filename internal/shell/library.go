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
// That sentence about the new houses is now load-bearing rather than an intention.
// Discover takes a *list* of sources and walks all of them into one list, which is
// the union the four asset flags deliberately do not do (docs/IMPROVEMENTS.md 5.3
// left it to Stage 2, saying a union should be designed rather than fall out of a
// resolution order nobody wrote down). The union is here and not there because the
// two things are different: an art root *replaces* the built-in one, because two
// copies of PICT 1000 have to resolve to one picture; houses do not resolve, they
// accumulate, and a player with four of their own and the twenty-two wants
// twenty-six. What tells them apart on screen is the set each source declares --
// see sets.go, which has the whole argument for why the source is what declares it.
//
// Accumulating is not the new part, which is worth saying because it would be easy
// to present it as one. BuildHouseList already walked *two* sources into one flat
// list: up to kMaxExtraHouses (8) specs handed in by AddExtraHouse -- a house
// double-clicked or dropped on the application, which may be anywhere on any volume
// -- copied in first, and then DoDirSearch's walk for the rest
// (SelectHouse.c:650-664, 668-675). So a list of houses from more than one place is
// the original's own design. What the original does not do is say which place any
// row came from: the eight and the forty-eight sort together by name and the dialog
// draws the same row for both. The set is that missing sentence and nothing more.
//
// One original behaviour is deliberately not reproduced. SortHouseList's duplicate
// removal collapses two houses with the same name in different folders into one,
// through a pair of self-comparison bugs at SelectHouse.c:527-528 that reduce its
// predicate to "same name". There is nothing to reproduce: names here are paths, a
// path is unique, and two files called "Demo House" in two directories are two
// houses. Both are listed.

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/bwenstar/gliderGo/internal/house"
)

// House is one entry in the picker: a file that sniffed as a house, and what its
// header said.
//
// Name is the file's name with any extension removed, because that is what a house
// is called: houseType has no name field, and every place the original shows a
// house name it is reading the file's (see cmd/glidergo, and HouseIO.c's
// thisHouseName). Rel is the path relative to the library root and is what
// distinguishes two houses that share a name.
//
// **FS and Rel together are what opens the file** -- house.LoadFS(h.FS, h.Rel), which is what
// Library.Open does. They travel on the house rather than on the library because a library is
// a union of sources now and there is no single filesystem to resolve Rel against; a house
// that did not carry its own would be a row in a list that nothing could load.
//
// Path is Rel with the source's label in front of it and exists for messages: it is a real
// path when the source is a directory and reads "built-in:houses/Titanic.house" when the
// houses are the ones inside the executable, which is not something to hand to os.Open.
type House struct {
	Name string
	Rel  string
	Path string

	// FS is the source's filesystem, and Set is the source's declared set. Both are copied
	// from the Source this house was found in; see sets.go.
	FS  fs.FS
	Set Set

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

// Library is every house Discover found, from every source, in one sorted list.
//
// Sources is what it was asked to walk, in the order it was given them. Root is those
// sources' labels joined, for the one message that has to name where the houses were looked
// for -- which is a sentence and not a path, and is why it is a string rather than a slice
// somebody would be tempted to index.
type Library struct {
	Root    string
	Sources []Source
	Houses  []House
	Skipped []Skip
}

// Open reads one of the listed houses, from the source it was found in.
func (l *Library) Open(h House) (*house.House, error) { return house.LoadFS(h.FS, h.Rel) }

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

// houseArtDir is the one subdirectory name a root can hold that is known not to be houses: the
// per-house pictures, spelled the same in the levels root and under assets/extracted. See walk.
const houseArtDir = "houseart"

// Discover walks every source it is given and returns one sorted list of the houses in all
// of them.
//
// Each source is walked as a *tree* rather than as a single directory, because a directory is
// the obvious way to group houses and always was -- the 22 inside the executable, the new
// houses of Stage 2, and whatever the player drops in. Hidden directories are skipped, and so
// is the walk's own error on any single entry: one unreadable file in a tree of houses is not
// a reason to have no house list.
//
// **A source that cannot be read is an error and the rest are still walked.** The returned
// library holds every house that was found and the error names every root that failed, joined,
// so a caller can report the failure and still show a list -- which is what cmd/glidergo does,
// because a missing houses directory belongs on the screen the player is looking at and not
// only on a terminal they may never see (docs/IMPROVEMENTS.md 2.6). A caller that gives no
// sources at all gets the error and an empty library, which is what a build carrying no assets
// with no flags set hands over.
func Discover(sources ...Source) (*Library, error) {
	lib := &Library{Sources: sources}

	labels := make([]string, 0, len(sources))
	for _, src := range sources {
		if src.Label != "" {
			labels = append(labels, src.Label)
		}
	}
	lib.Root = strings.Join(labels, ", ")

	if len(sources) == 0 {
		return lib, errors.New("shell: no houses directory")
	}
	var errs []error
	for _, src := range sources {
		if err := lib.walk(src); err != nil {
			errs = append(errs, err)
		}
	}

	lib.Sort()
	return lib, errors.Join(errs...)
}

// walk is Discover for one source, appending what it finds.
func (l *Library) walk(src Source) error {
	fsys, label := src.FS, src.Label
	if fsys == nil {
		return errors.New("shell: no houses directory")
	}
	// The label goes in front, because a root that will not stat says only `stat .: no such
	// file or directory` on its own: os.DirFS reports the error against the name asked for and
	// not against the directory it joined it to. With two roots a player can name -- -houses
	// and -levels -- two typos produced two identical pathless lines, and neither said which
	// flag was wrong.
	if st, err := fs.Stat(fsys, "."); err != nil {
		return fmt.Errorf("shell: %s: %w", label, err)
	} else if !st.IsDir() {
		return errors.New("shell: " + label + " is not a directory")
	}

	return fs.WalkDir(fsys, ".", func(rel string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // an unreadable entry, not an unreadable tree
		}
		if d.IsDir() {
			if rel != "." && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			// "houseart" at the top of a root holds pictures, not houses. It is skipped by
			// name, and only at the top, because that is exactly where the convention puts it
			// (internal/assetpack.LevelsName): a levels root carries `houseart/<House
			// Name>/pict/<id>.png` beside its houses. Nothing in there has a house's
			// extension today, so this changes no listing -- what it stops is a future
			// extensionless file in somebody's art tree turning up in Skipped as a house that
			// would not parse, which is a confusing way to be told about a stray file.
			//
			// At the top only, and not anywhere: a directory of houses that happens to have a
			// subdirectory of that name deeper down is not making this claim, and a walker
			// that silently dropped it would be the picker deciding what a player's own
			// folders mean.
			if rel == houseArtDir {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		if !houseExts[strings.ToLower(path.Ext(d.Name()))] {
			return nil
		}
		sum, err := house.PeekFS(fsys, rel)
		if err != nil {
			l.Skipped = append(l.Skipped, Skip{Path: path.Join(label, rel), Why: err})
			return nil
		}
		name := strings.TrimSuffix(d.Name(), path.Ext(d.Name()))
		l.Houses = append(l.Houses, House{
			Name:   name,
			Rel:    rel,
			Path:   path.Join(label, rel),
			FS:     fsys,
			Set:    src.Set,
			Rooms:  sum.NRooms,
			Locked: !sum.Unlocked,
			Demo:   name == DemoHouse,
			Size:   sum.Size,
			Banner: sum.Banner,
			Scores: sum.Scores,
		})
		return nil
	})
}

// Sort puts the list in SortHouseList's order.
func (l *Library) Sort() {
	sort.SliceStable(l.Houses, func(i, j int) bool {
		a, b := &l.Houses[i], &l.Houses[j]
		if a.Name != b.Name {
			return NameFirst(a.Name, b.Name)
		}
		// Two houses of the same name, which the original loses one of
		// (SelectHouse.c:527-528) and this keeps both of. Ordered by set first, so a
		// shipped Slumberland sits above somebody else's, and by path within a set,
		// so the list is stable and the picker can show the difference. Name first
		// and set second and not the other way round: the list is sorted by name,
		// and grouping it by set would hide a new house among twenty-two originals
		// from the player who was looking for it where its name belongs.
		if a.Set != b.Set {
			return a.Set < b.Set
		}
		return a.Rel < b.Rel
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
