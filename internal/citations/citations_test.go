package citations

// The receipts, checked.
//
// This port is a transcription, and a transcription's only defence is its citations. There are
// about 17,800 of them carrying a line number -- `GliderPRO/Sources/Player.c:1234` and the shorter
// `Player.c:1234` -- and until this file existed not one of them was ever checked against the
// file it names. That is a worse gap than it sounds, because a citation is the *only* thing
// standing behind a claim like "the original does it this way": a reader who cannot resolve one
// has to take the transcription's word for it, which is the exact position the citations exist to
// get them out of.
//
// Four things go wrong, and all four were found the first time this ran:
//
//  1. The file does not exist. `Environs.c` for `Environ.c`, `Sounds.c` for `Sound.c`,
//     `StructuresInit1.c` for `StructuresInit.c`, `MenuBar.c` for `Menu.c` -- near-misses of
//     real names, which is the kind of error a reader forgives and a `grep` does not survive.
//     One, `PlayerControl.c`, was a file an early draft invented and that has never existed.
//  2. The line does not exist. `RectUtils.c:322` in a 318-line file, `Room.c:1172-1215` in a
//     1206-line one. Nine of these.
//  3. The path names the wrong directory -- `GliderPRO/Headers/Player.c` for a file in
//     `Sources/` -- which resolves to nothing while looking entirely plausible.
//  4. A *range* that runs backwards, which is a typo in one digit and reads as a range.
//
// It also sweeps two classes of citation that point inward rather than at 1994: this repository's
// own `path:line` references, and the test names the prose quotes. Those are the same promise made
// about our own tree -- "`TestFooBar` holds this true" is worth nothing if `TestFooBar` was renamed
// three stages ago, and four had been.
//
// # Why this is a test and not a linter
//
// Because it needs the 1994 C, and the 1994 C is not in this repository. So the checks that need
// it skip when it is absent, and `.github/workflows/ci.yml` has a job that clones it at the pin so
// that they do run somewhere. A skip is a weak thing to rely on, which is why the classes that
// need nothing -- ranges, own paths, test names -- are written so that they never skip, and why
// the one number below is a floor rather than a comment: a regex that stops matching would
// otherwise turn 17,800 assertions into zero silently, and green.
//
// # Why the numbers are not simply clamped
//
// Every one of the nine out-of-range citations was fixed by reading the C and finding the line the
// claim was about, not by lowering the number until it fit. `Banner.c:205-243` became `205-236`
// because `DisplayStarsRemaining` ends at 236; `RectUtils.c:322` became `210-216` because that is
// where `QSetRect` is. A citation that resolves to the wrong place is worse than one that resolves
// to nothing, because nothing announces itself and the wrong place does not.

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/project"
)

// ---------------------------------------------------------------------------
// where things are
// ---------------------------------------------------------------------------

// The pinned tree, as README.md's "Working on the original source" puts it there. Three paths and
// not one root, because `Prefix.h` sits loose at the top of upstream's tree rather than inside
// `Headers/` -- which is how it came to be missing from the copy recipe for four stages while four
// citations pointed at it. That was found by this file.
const (
	upstream    = "GliderPRO"
	sourcesDir  = "Sources"
	headersDir  = "Headers"
	prefixFile  = "Prefix.h"
	pinnedAt    = "94fed96e0b4c810a6ac861e5d4b14d625a5a1c31"
	cloneAdvice = "the 1994 C is not in this repository; README.md " +
		"\"Working on the original source\" has the three commands, or:\n" +
		"\tgit clone https://github.com/softdorothy/glider_pro /tmp/glider_pro\n" +
		"\tgit -C /tmp/glider_pro checkout " + pinnedAt + "\n" +
		"\tcp -r /tmp/glider_pro/Sources /tmp/glider_pro/Headers /tmp/glider_pro/Prefix.h " +
		upstream + "/"
)

// citationFloor is how many line-carrying citations to the 1994 C the sweep must still find.
//
// Deliberately loose. The exact figure today is about 17,800 and it moves every time anybody
// writes a paragraph, so pinning it exactly would make this file fail on ordinary work -- and a
// test that cries wolf on ordinary work is one that gets its number bumped without being read.
// What the floor is actually for is the failure it cannot otherwise see: a regex edit that stops
// matching turns every assertion below into a vacuous pass over an empty slice, and the suite
// stays green. That failure does not arrive at 17,000; it arrives at nought.
const citationFloor = 15000

// skipDirs are the directories the sweep does not read, and why.
//
// stage15-raw is the interesting one: it is gitignored, so it is present on the machine that
// wrote it and absent from every clone, which would make this file's result depend on whose
// checkout it ran in. .gitignore explains why those drafts are not committed -- they still carry
// the errors the review pass corrected -- and a document nobody can read is not one whose
// citations are worth holding.
//
// The entry to be careful with is `extracted`, which was `assets` on the first run and wrong for
// it. assets/ holds two hand-written Go files at its top and the 1994 data in one subdirectory, so
// skipping the whole thing made assets_test.go's declarations invisible -- and the first failure
// this file reported was a test that exists being called dangling. A skip list wants the narrowest
// directory that is actually machine-written.
var skipDirs = map[string]string{
	".git":        "not source",
	upstream:      "the thing being cited, and its own C cites itself constantly",
	"extracted":   "assets/extracted: 1994 data, and its manifests are machine-written",
	"bin":         "build output",
	"dist":        "release staging",
	"stage15-raw": "gitignored working drafts; see .gitignore",
}

// sweepExt is what counts as text worth reading. Deliberately includes .yml and .sh: a workflow
// or a script that names a document is making the same promise a comment does, and CONTRIBUTING's
// walkthrough is only true if the paths in it are.
var sweepExt = map[string]bool{
	".go": true, ".md": true, ".py": true, ".sh": true, ".txt": true, ".yml": true,
}

// repoRoot walks up from the package directory to the directory holding go.mod. internal/module
// and internal/shell have the same helper for the same reason: a sweep has to start from the
// tree, not from wherever `go test` happened to be run.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod above %s", dir)
		}
		dir = parent
	}
}

// ---------------------------------------------------------------------------
// reading the two trees
// ---------------------------------------------------------------------------

// lineCount is how many lines a file has once its line endings are normalised, which is the space
// every citation in this repository is written in.
//
// The normalisation is not incidental. `Sources/*.c` and `Headers/*.h` are classic Mac text with
// CR-only line endings, so `wc -l` answers 0 and most tools see one enormous line; README.md warns
// about it and the docs say outright that their numbers are into an LF-converted copy. A file with
// no terminator on its last line still has that line, which is the case `Prefix.h` is: seven
// `#define`s, six newlines, and `Prefix.h:1` is a real citation.
func lineCount(b []byte) int {
	s := strings.ReplaceAll(strings.ReplaceAll(string(b), "\r\n", "\n"), "\r", "\n")
	n := strings.Count(s, "\n")
	if len(s) > 0 && !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}

// original is the pinned C tree as file name -> line count, plus name -> which directory it is in.
// It skips the test when the tree is not there, with the commands that put it there.
func original(t *testing.T) (lines map[string]int, where map[string]string) {
	t.Helper()
	root := filepath.Join(repoRoot(t), upstream)
	lines, where = map[string]int{}, map[string]string{}
	for _, dir := range []string{sourcesDir, headersDir} {
		ents, err := os.ReadDir(filepath.Join(root, dir))
		if err != nil {
			t.Skipf("%s: %s", err, cloneAdvice)
		}
		for _, e := range ents {
			b, err := os.ReadFile(filepath.Join(root, dir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			lines[e.Name()] = lineCount(b)
			where[e.Name()] = dir
		}
	}
	// Prefix.h last and separately, because it is not in either directory and because its
	// absence is a different failure from the tree's: somebody followed an out-of-date recipe.
	b, err := os.ReadFile(filepath.Join(root, prefixFile))
	if err != nil {
		t.Skipf("%s: %s", err, cloneAdvice)
	}
	lines[prefixFile] = lineCount(b)
	where[prefixFile] = ""

	if len(lines) < 90 {
		t.Fatalf("%s holds %d files; upstream %s has 67 sources, 25 headers and Prefix.h",
			root, len(lines), pinnedAt[:7])
	}
	return lines, where
}

// textLine is one line of this repository, with somewhere to point when it is wrong.
type textLine struct {
	path string // relative to the repository root, so the failure is clickable
	n    int    // 1-based
	text string
}

// selfRel is this file, which the sweep skips and one check reads back on purpose.
var selfRel = filepath.Join("internal", "citations", "citations_test.go")

// readLines reads one file of this repository as numbered lines.
func readLines(t *testing.T, rel string) []textLine {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
	if err != nil {
		t.Fatal(err)
	}
	var out []textLine
	for i, ln := range strings.Split(string(b), "\n") {
		out = append(out, textLine{rel, i + 1, ln})
	}
	return out
}

// sweep reads every text file in the repository that is worth citing from.
//
// It excludes this file. That is not tidiness: the checks below are demonstrated against planted
// bad citations -- `Banner.c:243` in a 237-line file and the rest -- and a sweep that read its own
// source would find them and fail, which would make the one test proving the checker works the
// reason the checker fails.
//
// The exclusion has one cost, and it was this file's own second self-inflicted failure: the tests
// declared here are invisible to it, so a comment anywhere in the repository that names one reads
// as a dangling reference. .github/workflows/ci.yml names one, legitimately. Hence readLines above
// and the extra pass in TestEveryTestNameTheProseQuotesExists -- declarations from here count, the
// citations here still do not.
func sweep(t *testing.T) []textLine {
	t.Helper()
	root := repoRoot(t)
	self := selfRel

	var out []textLine
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if _, skip := skipDirs[d.Name()]; skip {
				return fs.SkipDir
			}
			return nil
		}
		if !sweepExt[filepath.Ext(p)] {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		if rel == self {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for i, ln := range strings.Split(string(b), "\n") {
			out = append(out, textLine{rel, i + 1, ln})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatalf("the sweep read nothing under %s", root)
	}
	return out
}

// ---------------------------------------------------------------------------
// class 1: the 1994 C
// ---------------------------------------------------------------------------

// cCite matches a citation to the original, in the form the documents say they use.
//
//	GliderPRO/Sources/Player.c:1234        the full pinned form
//	Player.c:1234                          the short form, once the file is established
//	Play.c:735, 736, 739                   a list
//	Room.c:1103-1206                       a range, hyphen or en-dash
//	Prefix.h:1                             the one file that is in neither directory
//
// A colon before the number is required, and that is the one interesting decision here. The
// space-separated form -- `Banner.c 205` -- is not the documented convention, and matching it
// would mean reading "as `Banner.c` 300 times" as a citation to line 300 of a 237-line file: a
// failure invented by the checker, on prose that was correct. The convention is a colon, so the
// checker wants a colon.
//
// The directory group is captured rather than skipped so that a full path pointing into the wrong
// one can be told apart from a short citation that names no directory at all.
var cCite = regexp.MustCompile(
	`(?:GliderPRO/([A-Za-z]+)/)?\b((?:Sources_|Headers_)?[A-Z][A-Za-z0-9_]*\.[ch])\b:[ ]*` +
		`([0-9]+(?:[ ]*[-\x{2013}][ ]*[0-9]+)?(?:[ ]*,[ ]*[0-9]+(?:[ ]*[-\x{2013}][ ]*[0-9]+)?)*)`)

// cName matches a bare mention of an original source file, with or without a line number.
//
// A second, looser pass, and it checks only one thing: that the file exists. It has to be looser,
// because five of the six wrong file names this found were written without a line number --
// `Interactions.c` / `Environs.c`, `Sources/StructuresInit1.c` -- and cCite would never have seen
// them. It cannot check anything else, because at this width the matches include Apple's Toolbox
// headers and two metasyntactic placeholders; those are what notInTheOriginal is for.
var cName = regexp.MustCompile(`\b((?:Sources_|Headers_)?[A-Z][A-Za-z0-9_]*\.[ch])\b`)

// resolve turns a cited name into the name the pinned tree uses.
//
// One rewrite: `Sources_Dynamics2.c` -> `Dynamics2.c`. docs/analysis/constants.md was written
// against a flat working copy whose files are named for the directory they came from, and its
// regeneration command still is -- the document now shows the `tr`/`basename` loop that produces
// them. Resolving the prefix rather than exempting it means those citations' line numbers get
// checked like every other, which is the point.
func resolve(name string) string {
	for _, p := range []string{"Sources_", "Headers_"} {
		if strings.HasPrefix(name, p) {
			return strings.TrimPrefix(name, p)
		}
	}
	return name
}

// notInTheOriginal is every `Name.c` / `Name.h` this repository names that upstream's tree does
// not contain, with the reason each is legitimate.
//
// It is a map to reasons and not a list of names on purpose. An exemption without a reason is
// indistinguishable from an error somebody silenced, and this list is the one place in the file
// where a wrong entry is invisible -- every other check fails loudly. TestEveryExemptionIsStillEarned
// keeps it from growing stale in the other direction: an entry nothing cites any more is a name
// somebody is now free to get wrong.
var notInTheOriginal = map[string]string{
	// Apple's 1990s Toolbox headers, named where a document explains which system call the C
	// is making. Part of the Mac SDK the original compiled against, not of Calhoun's tree --
	// so this repository cites them and cannot ship them.
	"QuickDraw.h":   "Apple Toolbox: QuickDraw",
	"Quickdraw.h":   "Apple Toolbox: QuickDraw, as the C spells it in places",
	"QDOffscreen.h": "Apple Toolbox: offscreen GWorlds",
	"Sound.h":       "Apple Toolbox: Sound Manager -- not to be confused with Sources/Sound.c",
	"Movies.h":      "Apple Toolbox: QuickTime",
	"Files.h":       "Apple Toolbox: the File Manager",
	"Types.h":       "Apple Toolbox: base types",
	"MacTypes.h":    "Apple Toolbox: base types, Carbon spelling",
	"Fonts.h":       "Apple Toolbox: the Font Manager",
	"Drag.h":        "Apple Toolbox: the Drag Manager",
	"Icons.h":       "Apple Toolbox: the Icon Utilities",
	"Gestalt.h":     "Apple Toolbox: Gestalt",

	// This port's own platform backends name the headers they are re-declaring by hand, which
	// is how a `syscall`-only Windows backend documents where its constants came from.
	"WinUser.h": "Win32 SDK, named by internal/platform/win32's hand-declared constants",
	"WinGDI.h":  "Win32 SDK, same",
	"Xlib.h":    "libX11, named by internal/platform/x11 for the same reason",
	"Xutil.h":   "libX11, same",
	"XKBlib.h":  "libX11's XKB extension, same",
	"SDL.h":     "named once, in a passage about what this port deliberately does not use",

	// Metasyntactic. Both appear in the "how to read a citation" preamble several documents
	// open with, which is the one place a file name is not meant to name a file.
	"File.c": "the placeholder in `GliderPRO/Sources/File.c:NNN`, the form the docs describe",
	"X.c":    "the placeholder in the `tr '\\r' '\\n' < Sources/X.c` conversion command",

	// And the names that are wrong on purpose, quoted in docs/CITATIONS.md 5 as the errors this
	// checker found on its first run. Correcting them would delete the finding, which is the
	// same argument notATest makes for two test names below and docs/IMPROVEMENTS.md makes for
	// the passage notATest is quoting -- a record of a defect has to be allowed to contain it.
	//
	// PlayerControl.c is the strongest case of the four: it is a file an early draft invented,
	// and docs/analysis/determinism.md 2 names it precisely to stop the next reader looking for
	// it. Removing that sentence would not fix anything; it would only make the invention
	// findable again.
	"PlayerControl.c":   "named in determinism.md 2 and CITATIONS.md 5 to record that no such file exists",
	"Environs.c":        "quoted in CITATIONS.md 5 as a found error; the file is Environ.c",
	"Sounds.c":          "quoted in CITATIONS.md 5 as a found error; the file is Sound.c",
	"StructuresInit1.c": "quoted in CITATIONS.md 5 as a found error; the file is StructuresInit.c",
	"MenuBar.c":         "quoted in CITATIONS.md 5 as a found error; the file is Menu.c",
}

// span is one cited line or line range.
type span struct{ lo, hi int }

// parseSpans reads the number part of a citation: `1234`, `1103-1206`, `735, 736, 739`.
func parseSpans(nums string) []span {
	var out []span
	for _, part := range strings.Split(nums, ",") {
		part = strings.ReplaceAll(part, "–", "-")
		lo, hi, isRange := strings.Cut(part, "-")
		a, err := strconv.Atoi(strings.TrimSpace(lo))
		if err != nil {
			continue
		}
		b := a
		if isRange {
			if v, err := strconv.Atoi(strings.TrimSpace(hi)); err == nil {
				b = v
			}
		}
		out = append(out, span{a, b})
	}
	return out
}

// cCitation is one parsed citation to the original.
type cCitation struct {
	at    textLine
	raw   string
	dir   string // the directory the citation named, or "" if it named none
	name  string // resolved to the pinned tree's spelling
	spans []span
}

// cCitations parses every line-carrying citation in the sweep.
func cCitations(lines []textLine) []cCitation {
	var out []cCitation
	for _, ln := range lines {
		for _, m := range cCite.FindAllStringSubmatch(ln.text, -1) {
			out = append(out, cCitation{
				at:    ln,
				raw:   strings.TrimSpace(m[0]),
				dir:   m[1],
				name:  resolve(m[2]),
				spans: parseSpans(m[3]),
			})
		}
	}
	return out
}

// checkSpans reports every way one citation's numbers are wrong, as one string per problem.
//
// A pure function over a name, its numbers and a line count, so that the checker can be shown a
// citation that is wrong -- which the repository, after this file's first run, no longer contains.
// internal/module's checkGoMod is factored out for the same reason and says so at greater length:
// the alternative is a check nobody has ever seen fail.
func checkSpans(name string, spans []span, have int) []string {
	var bad []string
	for _, s := range spans {
		switch {
		case s.lo < 1:
			bad = append(bad, fmt.Sprintf("line %d is not a line", s.lo))
		case s.hi < s.lo:
			bad = append(bad, fmt.Sprintf("the range %d-%d runs backwards", s.lo, s.hi))
		case s.lo > have:
			bad = append(bad, fmt.Sprintf("line %d, but %s has %d lines", s.lo, name, have))
		case s.hi > have:
			bad = append(bad, fmt.Sprintf("the range ends at %d, but %s has %d lines",
				s.hi, name, have))
		}
	}
	return bad
}

// TestEveryCitedFileIsOneTheOriginalHas is tier one: the name resolves.
//
// The loose pass, over bare mentions as well as line citations, because that is where the wrong
// names were: `Environs.c` for `Environ.c` and `Sounds.c` for `Sound.c` are one letter out and
// carry no line number, so nothing narrower would have seen either. Four of the six it found were
// written years apart by somebody reading a file list from memory, which is exactly the failure a
// sweep is better at than a reviewer.
func TestEveryCitedFileIsOneTheOriginalHas(t *testing.T) {
	lines, _ := original(t)

	// One entry per name, holding the first place it was seen: a name that is wrong is usually
	// wrong in one place, and a name that is wrong in forty would otherwise print forty times.
	first := map[string]textLine{}
	for _, ln := range sweep(t) {
		for _, m := range cName.FindAllStringSubmatch(ln.text, -1) {
			name := resolve(m[1])
			if _, ok := lines[name]; ok {
				continue
			}
			if _, ok := notInTheOriginal[name]; ok {
				continue
			}
			if _, seen := first[name]; !seen {
				first[name] = ln
			}
		}
	}
	for _, name := range sorted(first) {
		at := first[name]
		t.Errorf("%s:%d names %s, which upstream %s does not have. If it is a real file "+
			"outside the original -- a Toolbox header, say -- add it to notInTheOriginal "+
			"with the reason; if it is a near-miss of a real name, fix the name",
			at.path, at.n, name, pinnedAt[:7])
	}
}

// TestEveryCitedLineIsInTheFileItCites is tier two, and the one with the most reach: about 17,800
// numbers, none of which was checked before this file existed and nine of which were wrong.
//
// Wrong in the specific way that matters, too. `Room.c:1172-1215` is a 1206-line file cited nine
// lines past its end, which means the citation was written against a differently converted copy --
// so the *other* numbers from that reading are suspect as well, and knowing that is worth more
// than the nine fixes. That is the argument for checking every one rather than sampling.
func TestEveryCitedLineIsInTheFileItCites(t *testing.T) {
	lines, _ := original(t)

	// Two counts, because they measure different things and both get quoted. A citation is one
	// reference a reader follows; a span is one line or range inside it, and `Play.c:735, 736,
	// 739` is one of the first and three of the second. The claims this file checks are spans.
	found, spans := 0, 0
	for _, c := range cCitations(sweep(t)) {
		have, ok := lines[c.name]
		if !ok {
			// TestEveryCitedFileIsOneTheOriginalHas owns that failure; reporting it twice
			// would double every message on the day somebody adds a file name typo.
			continue
		}
		found++
		spans += len(c.spans)
		for _, problem := range checkSpans(c.name, c.spans, have) {
			t.Errorf("%s:%d cites %q: %s. Read the C and find the line the claim is "+
				"about -- do not lower the number until it fits, because a citation "+
				"that resolves to the wrong place is worse than one that resolves to "+
				"nothing", c.at.path, c.at.n, c.raw, problem)
		}
	}
	if found < citationFloor {
		t.Errorf("the sweep found %d line citations to the original and the floor is %d: "+
			"either cCite has stopped matching -- in which case every assertion in this "+
			"file is now a vacuous pass -- or a great many citations were deleted",
			found, citationFloor)
	}
	t.Logf("checked %d citations naming %d lines or ranges, against %d files of the pinned C",
		found, spans, len(lines))
}

// TestEveryPinnedPathNamesTheDirectoryItsFileIsIn is tier three.
//
// `GliderPRO/Headers/Player.c` is the failure: a path that reads correctly, resolves to nothing,
// and passes both checks above because the *file name* is real and the *line* is in range. The
// documents' whole convention is that a pinned path can be pasted after a `cd` and opened, so a
// path is either right about the directory or it is not a pinned path.
func TestEveryPinnedPathNamesTheDirectoryItsFileIsIn(t *testing.T) {
	_, where := original(t)

	for _, c := range cCitations(sweep(t)) {
		if c.dir == "" {
			continue // the short form names no directory, so it cannot name the wrong one
		}
		want, known := where[c.name]
		if !known {
			continue // tier one's failure
		}
		if want == "" {
			t.Errorf("%s:%d writes %q, but %s is loose at the top of upstream's tree: "+
				"the path is GliderPRO/%s", c.at.path, c.at.n, c.raw, c.name, c.name)
			continue
		}
		if c.dir != want {
			t.Errorf("%s:%d writes %q, but %s is in %s/ -- the path resolves to nothing",
				c.at.path, c.at.n, c.raw, c.name, want)
		}
	}
}

// TestEverySourceFileInTheOriginalIsCitedSomewhere is tier four's coverage half, and it is a claim
// about the transcription rather than about the citations.
//
// All 67 of upstream's `.c` files are cited, today, which is the strongest single statement this
// repository can make about having read the original rather than guessed at it. It is worth
// holding: a source file that nothing cites is one nobody has accounted for, and the honest way to
// find that out is not to wait for somebody to notice a missing feature.
//
// Headers are deliberately not held to it. Three -- DynamicMaps.h, RectUtils.h, RubberBands.h --
// are prototype-only companions to heavily cited `.c` files, and there is nothing in them to cite
// that the `.c` does not say better.
func TestEverySourceFileInTheOriginalIsCitedSomewhere(t *testing.T) {
	lines, where := original(t)

	cited := map[string]bool{}
	for _, ln := range sweep(t) {
		for _, m := range cName.FindAllStringSubmatch(ln.text, -1) {
			cited[resolve(m[1])] = true
		}
	}
	var missing []string
	for name := range lines {
		if where[name] == sourcesDir && !cited[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("%d of upstream's sources are cited nowhere in this repository: %s. Either "+
			"the subsystem they hold has not been read, or it has been read and the "+
			"transcription does not say so",
			len(missing), strings.Join(missing, " "))
	}
}

// TestEveryExemptionIsStillEarned is tier four's other half, and it points the other way: at this
// file.
//
// An exemption list is the one part of a checker that cannot fail, so it is the part that rots.
// Every name in notInTheOriginal and notATest was put there because something cited it; when that
// citation goes, the entry stops being a documented exception and becomes a name the next person
// is quietly free to get wrong. A stale entry is not a small thing here -- `PlayerControl.c` is
// exempt *because* one sentence names it to say it does not exist, and if that sentence is ever
// deleted the exemption would let the mistake back in.
func TestEveryExemptionIsStillEarned(t *testing.T) {
	lines := sweep(t)

	seen := map[string]bool{}
	for _, ln := range lines {
		for _, m := range cName.FindAllStringSubmatch(ln.text, -1) {
			seen[m[1]] = true
		}
		for _, name := range testNames(ln.text) {
			seen[name] = true
		}
	}
	for _, from := range []struct {
		list map[string]string
		what string
	}{
		{notInTheOriginal, "notInTheOriginal"},
		{notATest, "notATest"},
	} {
		for _, name := range sorted(from.list) {
			if !seen[name] {
				t.Errorf("%s exempts %s (%q) and nothing cites it any more: delete the "+
					"entry, so that the next person to write the name gets told",
					from.what, name, from.list[name])
			}
		}
	}
}

// ---------------------------------------------------------------------------
// class 2: this repository's own paths
// ---------------------------------------------------------------------------

// ownPath matches a reference to a file in this repository, with an optional line number.
//
// Anchored on the handful of top-level directories rather than on "anything with a slash", because
// prose is full of paths that are not ours -- `/tmp/glider_pro/Sources`, `~/.config/glidergo`,
// `os.UserConfigDir`-shaped examples -- and a checker that demanded those exist would be wrong
// about all of them.
var ownPath = regexp.MustCompile(
	`\b((?:cmd|internal|docs|tools|scripts|assets|\.github)/[A-Za-z0-9_./-]+` +
		`\.(?:go|md|py|sh|json|tsv|txt|yml|hashes))(?::([0-9]+))?`)

// deadOwnPaths are prefixes of our own tree that references may point into even though a clone
// does not have them, with the reason.
//
// One entry, and it is a defect this repository has already argued out in the open:
// docs/IMPROVEMENTS.md records that three Go files cite `docs/analysis/stage15-raw/`, that the
// directory is gitignored because those drafts still contain errors the review pass corrected, and
// that the "releasePolish N" numbering they quote exists only there -- so the references cannot
// simply be repointed and the fix is to give the surviving decisions numbers in a committed file.
// Deferred, deliberately. What this entry adds is that the deferral is now machine-tracked: the
// day somebody commits that file, TestEveryExemptionIsStillEarned asks for this line back.
var deadOwnPaths = map[string]string{
	"docs/analysis/stage15-raw/": "gitignored drafts; the dead reference is owned by " +
		"docs/IMPROVEMENTS.md and cannot be repointed until the surviving decisions are " +
		"numbered in a committed file",
}

// TestEveryReferenceToOurOwnTreeResolves is the same promise as the C citations, made about this
// repository, where it is cheaper to keep and easier to break: files here get renamed.
//
// It found one -- `docs/analysis/sound.md`, which is `audio.md` -- out of 968 references. That
// ratio is the argument for the test rather than against it: the references are nearly all right,
// which is exactly why a reader trusts them, which is why the one that is wrong costs somebody an
// afternoon of looking for a document that was never there.
func TestEveryReferenceToOurOwnTreeResolves(t *testing.T) {
	root := repoRoot(t)

	// Line counts are read once per file and kept, because a document like docs/PLAN.md is
	// referenced hundreds of times and re-reading it each time makes this test slower than the
	// rest of the package put together.
	size := map[string]int{}
	lineCountOf := func(rel string) int {
		if n, ok := size[rel]; ok {
			return n
		}
		n := -1
		if b, err := os.ReadFile(filepath.Join(root, rel)); err == nil {
			n = lineCount(b)
		}
		size[rel] = n
		return n
	}

	dead := func(rel string) bool {
		for p := range deadOwnPaths {
			if strings.HasPrefix(rel, p) {
				return true
			}
		}
		return false
	}

	missing, total := map[string]textLine{}, 0
	for _, ln := range sweep(t) {
		for _, m := range ownPath.FindAllStringSubmatch(ln.text, -1) {
			rel, num := m[1], m[2]
			if dead(rel) {
				continue
			}
			total++
			have := lineCountOf(rel)
			if have < 0 {
				if _, seen := missing[rel]; !seen {
					missing[rel] = ln
				}
				continue
			}
			if num == "" {
				continue
			}
			v, err := strconv.Atoi(num)
			if err != nil {
				continue
			}
			if v < 1 || v > have {
				t.Errorf("%s:%d points at %s:%d, and %s has %d lines",
					ln.path, ln.n, rel, v, rel, have)
			}
		}
	}
	for _, rel := range sorted(missing) {
		at := missing[rel]
		t.Errorf("%s:%d points at %s, which is not in this repository", at.path, at.n, rel)
	}
	if total == 0 {
		t.Error("the sweep found no references to our own tree, so ownPath has stopped matching")
	}
	t.Logf("checked %d references to %d of our own files", total, len(size))
}

// TestEveryPackageInTheTreeIsOnTheMapTheReadmeDraws is the coverage half, and the reason it is
// worth having is the failure it catches rather than the one it prevents.
//
// README.md's "What is in here" table is the first thing a reader looking for the code opens, and
// its job is to answer "where does X live" without a grep. A package missing from it is not a
// broken link -- nothing fails, nothing 404s, and the table still reads as complete -- so the only
// way a reader finds out is by concluding the thing they wanted is not in this project. That is the
// same class of defect as an unresolvable citation, arriving by omission instead of by error.
//
// It found two the first time it ran, and they were the two worst available: internal/shell, which
// is the title screen, the house picker, the settings, the about box and the score board -- 4,900
// lines, and everything a player sees before a room is composed -- and internal/audio, which the
// README discusses at length three sections higher up without ever saying where it is.
//
// The table may group: a row naming ten packages satisfies all ten, which is what keeps the map a
// map rather than a directory listing. What it may not do is leave one out silently.
func TestEveryPackageInTheTreeIsOnTheMapTheReadmeDraws(t *testing.T) {
	root := repoRoot(t)

	entries, err := os.ReadDir(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatalf("reading internal/: %v", err)
	}
	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatalf("reading README.md: %v", err)
	}

	// The table and not the whole file: the prose above it mentions most of these packages in
	// passing, and a check that accepted a passing mention would pass while the map stayed wrong.
	table := mapTable(string(readme))
	if table == "" {
		t.Fatal(`README.md has no "What is in here" table, or its heading has been reworded; ` +
			"this test reads the table and not the file, so it cannot fall back to the prose")
	}

	var absent []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if !strings.Contains(table, "internal/"+e.Name()+"/") {
			absent = append(absent, e.Name())
		}
	}
	for _, name := range absent {
		t.Errorf("internal/%s/ is not on README.md's map, so a reader looking for what is in it "+
			"has no way to find out that it exists", name)
	}
	t.Logf("checked %d packages against the README's map", len(entries))
}

// mapTable returns the body of README.md's "What is in here" table, or "" if the heading is gone.
//
// Deliberately crude -- from the heading to the next one -- because the alternative is a Markdown
// parser, and what is wanted is the region a reader would scan, not a parse tree. Pulled out as a
// function so the heading-is-gone case is a Fatal with a sentence rather than a silent pass: a
// test whose subject has been renamed away must say so, which is the whole lesson of the four
// stale test names class 3 found.
func mapTable(readme string) string {
	const heading = "## What is in here"
	i := strings.Index(readme, heading)
	if i < 0 {
		return ""
	}
	rest := readme[i+len(heading):]
	if j := strings.Index(rest, "\n## "); j >= 0 {
		rest = rest[:j]
	}
	return rest
}

// ---------------------------------------------------------------------------
// class 3: the test names the prose quotes
// ---------------------------------------------------------------------------

// testDecl finds a test, benchmark, fuzz target or example declaration.
var testDecl = regexp.MustCompile(`(?m)^func ((?:Test|Benchmark|Fuzz|Example)[A-Za-z0-9_]*)\s*\(`)

// testRef finds a name in prose that claims to be one of those, with the forms this repository's
// comments actually use once they have been wrapped and typeset:
//
//	TestCorpus*                     a wildcard standing for a family, matched as a prefix
//	TestCorpus…                     the same claim set in prose, with an ellipsis
//	TestFrame-\n// CountsMatch...   an identifier broken across a comment line with a hyphen
//
// All are documented in docs/IMPROVEMENTS.md as legitimate rather than as things to tidy up, and
// that call is respected here: a checker that made people rewrap paragraphs would get worked
// around, and the reason for wanting the names checked is that four of them were wrong.
//
// The ellipsis is the one that had to be learnt from a failure. The same IMPROVEMENTS table that
// documents `TestCorpus*` in a code span writes "the eight `TestCorpus…` tests" in the sentence
// beside it, because that is how the family reads in prose -- and a checker that knew only the star
// reported the table explaining the convention as a violation of it.
var testRef = regexp.MustCompile(
	`\b((?:Test|Benchmark|Fuzz)[A-Z][A-Za-z0-9_]*)(\*|\x{2026}|\.\.\.|-)?`)

// familyOf reports whether a quoted name is a claim about a family of tests rather than about one,
// and if so the prefix it claims.
func familyOf(name string) (string, bool) {
	for _, suffix := range []string{"*", "…", "..."} {
		if strings.HasSuffix(name, suffix) {
			return strings.TrimSuffix(name, suffix), true
		}
	}
	return name, false
}

// notATest is every `Test…`-shaped name in the prose that is not one of our tests, with why.
var notATest = map[string]string{
	// The 1994 C's own function: `TestHighScore` in HighScores.c, which decides whether a
	// score makes the board. It is named about a hundred times, which is what makes it worth
	// an entry rather than a rename -- the port's Go method is called TestHighScore too,
	// deliberately, because the whole file is a transcription of that function.
	"TestHighScore": "HighScores.c's own function, and the port's transcription of it",

	// Apple Toolbox, in a table explaining what Environ.c's screen walk calls.
	"TestDeviceAttribute": "a QuickDraw call, named in architecture.md's Environ.c table",

	// Two names quoted precisely because they were *wrong*. docs/IMPROVEMENTS.md records an
	// audit that found comments naming tests that did not exist, and both entries name the
	// wrong name and then the right one in the same sentence. Exempting them keeps the
	// correction readable; fixing them would delete the finding.
	"TestCheckFindsUndefined": "quoted in IMPROVEMENTS as the wrong name an audit found; " +
		"the real test is named in the same sentence",
	"TestClampedBandSurvivesTheKillTest": "quoted in IMPROVEMENTS as the wrong name an " +
		"audit found; the real test is named in the same sentence",
	"TestEveryKeyHasAName": "quoted in CITATIONS.md 5 as a stale reference this checker " +
		"found; the test is now TestEveryKeyHasANameAndRoundTrips, named in the same row",

	// The same shape, arrived at differently, and the reason the pattern is worth naming: a
	// changelog and a register are *records*, and a record of a rename has to be able to write
	// the old name. Renaming it here would turn "this test asserted the broken behaviour" into
	// a sentence about a test that always asserted the right one, which is the finding deleted.
	// IMPROVEMENTS 4.13 and the CHANGELOG entry above it both name the replacement immediately
	// afterwards, which is the condition that makes an exemption of this kind readable.
	"TestReplayNeedsAHouse": "quoted in IMPROVEMENTS 4.13 and the CHANGELOG as the test that " +
		"pinned a defect in green; it is now TestReplayNeedsAHouseToRun, named alongside",

	// Metasyntactic, and the counterpart of File.c and X.c above: the placeholder that appears
	// where a document explains what checking a quoted test name means.
	"TestFooBar": "the placeholder in CITATIONS.md 4's \"`TestFooBar` holds this true\"",
}

// testNames pulls the candidate test names out of one line, joining the hyphen-break form.
//
// The join is why this takes a line rather than a token: `TestFrame-` at the end of a comment line
// continues on the next one, and treating the two halves as two names would report both as
// missing. A family claim keeps its star or ellipsis, so that the caller -- and any failure message
// -- can tell a prefix claim from an exact one, and quote it back the way it was written.
// A trailing hyphen is dropped along with the name it ends, because a name written with one is the
// first half of a broken identifier and never a claim about a test that exists. At the end of a line
// joinBrokenNames has already taken it; anywhere else -- and there is one place, the row in
// docs/IMPROVEMENTS.md that documents the break by quoting `TestFrame-` inside a sentence -- it is
// somebody writing *about* the convention, and `TestFrame` is not a test either way.
//
// The cost of that rule is a name immediately followed by this repository's `--` em dash, which
// would be skipped rather than checked. Worth it: the failure is one unchecked name, and the
// alternative failure is correct prose reported as wrong, which gets answered by rewriting the
// prose.
func testNames(text string) []string {
	var out []string
	for _, m := range testRef.FindAllStringSubmatch(text, -1) {
		if m[2] == "-" {
			continue
		}
		out = append(out, m[1]+m[2])
	}
	return out
}

// hyphenBreak matches a line whose last thing is the first half of a test name.
var hyphenBreak = regexp.MustCompile(`\b((?:Test|Benchmark|Fuzz)[A-Za-z0-9_]*)-$`)

// joinBrokenNames rewrites a run of consecutive lines so that a name broken across two of them
// reads as one name, and the half left at the end of the first line reads as nothing at all.
//
// One output line per input line, so that a caller can still say which line a name was found on --
// which is the reason this is not simply a flat list of names. Pure, and over a slice rather than a
// stream, so that each shape the prose uses can be handed to it in a test.
//
// docs/IMPROVEMENTS.md argues for joining rather than for ignoring a trailing hyphen, and the
// argument is worth keeping in view here: a break hides a dangling name exactly as well as it hides
// a good one, so a checker that merely skipped the fragment would be blind on precisely the lines
// where a rename is most likely to have been missed. Only the first half is ever carried, because a
// name broken twice does not occur in this tree, and handling it would be untested code.
func joinBrokenNames(lines []string) []string {
	out := make([]string, len(lines))
	carry := ""
	for i, text := range lines {
		if carry != "" {
			// Indentation and a comment marker sit between the two halves of the name.
			text = carry + strings.TrimLeft(strings.TrimLeft(text, " \t"), "/*# \t")
			carry = ""
		}
		trimmed := strings.TrimRight(text, " \t")
		if m := hyphenBreak.FindStringSubmatch(trimmed); m != nil {
			carry = m[1]
			text = strings.TrimSuffix(trimmed, m[0])
		}
		out[i] = text
	}
	return out
}

// TestEveryTestNameTheProseQuotesExists is the inward-facing half of the whole file.
//
// A comment that says "TestFooBar holds this true" is a citation, and it is the citation most
// likely to go stale, because renaming a test is a thing people do without reading the comments
// that named it. Four were stale: `TestEveryKeyHasAName` had become
// `TestEveryKeyHasANameAndRoundTrips`, `TestLegacyOffsets` had become
// `TestLegacyLayoutTilesTheRecord`, a two-player reference named a test that had never existed,
// and one name was wrapped across a line with its last word lost. Each one is a comment that
// sounded like evidence and pointed nowhere.
func TestEveryTestNameTheProseQuotesExists(t *testing.T) {
	lines := sweep(t)

	// Declarations come from the sweep and from this file, which the sweep skips for its planted
	// citations. A test declared here is as real as any other, and something outside is entitled
	// to name it.
	declared := map[string]bool{}
	for _, from := range [][]textLine{lines, readLines(t, selfRel)} {
		for _, ln := range from {
			for _, m := range testDecl.FindAllStringSubmatch(ln.text, -1) {
				declared[m[1]] = true
			}
		}
	}
	if len(declared) < 500 {
		t.Fatalf("the sweep found %d test declarations; this repository has over 900, so "+
			"testDecl has stopped matching", len(declared))
	}
	prefixed := func(stem string) bool {
		for d := range declared {
			if strings.HasPrefix(d, stem) {
				return true
			}
		}
		return false
	}

	// Joining happens over the whole sweep at once, which is safe because sweep() emits a file's
	// lines in order and a join only ever reaches one line forward: the last line of one file
	// cannot carry into the first line of the next unless it ends mid-identifier, and a file
	// whose last line does that has a different problem.
	texts := make([]string, len(lines))
	for i, ln := range lines {
		texts[i] = ln.text
	}
	texts = joinBrokenNames(texts)

	missing := map[string]textLine{}
	for i, text := range texts {
		for _, name := range testNames(text) {
			stem, family := familyOf(name)
			switch {
			case family && prefixed(stem):
			case !family && (declared[name] || notATest[name] != ""):
			default:
				if _, seen := missing[name]; !seen {
					missing[name] = lines[i]
				}
			}
		}
	}
	for _, name := range sorted(missing) {
		at := missing[name]
		t.Errorf("%s:%d names %s, and no test, benchmark or fuzz target is declared with "+
			"that name. Rename the reference, or -- if it is not one of ours -- put it in "+
			"notATest with the reason", at.path, at.n, name)
	}
}

// TestEveryWayAWrappedTestNameIsWrittenIsUnderstood is the checker checking itself, class 3's half.
//
// Necessary for the same reason checkSpans has a test: the sweep passes now, so every branch above
// is one an empty input would also satisfy. And this is the class where a false positive costs the
// most -- a checker that reported correct prose as a dangling reference would be answered by
// rewrapping the paragraph or by deleting the line, and both of those make the documentation worse
// to satisfy a bug. Two of the three shapes below were reported as violations by the first draft.
func TestEveryWayAWrappedTestNameIsWrittenIsUnderstood(t *testing.T) {
	// The hyphen break, exactly as internal/render/locale.go 97 writes it. The joined name is
	// the claim; the fragment on the first line is not, and must not be reported as one.
	got := joinBrokenNames([]string{
		"// and the tables agree, which is what TestFrame-",
		"// CountsMatchTheSrcTables holds.",
	})
	var names []string
	for _, text := range got {
		names = append(names, testNames(text)...)
	}
	if fmt.Sprint(names) != "[TestFrameCountsMatchTheSrcTables]" {
		t.Errorf("a name broken across a comment line reads as %v; want the one joined name, "+
			"and in particular not the half of it before the hyphen", names)
	}

	// Both family forms, which mean the same thing and are written differently a paragraph
	// apart in docs/IMPROVEMENTS.md.
	for _, quoted := range []string{"TestCorpus*", "TestCorpus…", "TestCorpus..."} {
		stem, family := familyOf(testNames("the " + quoted + " tests")[0])
		if !family {
			t.Errorf("%q does not read as a claim about a family of tests", quoted)
		}
		if stem != "TestCorpus" {
			t.Errorf("%q claims the prefix %q, want TestCorpus", quoted, stem)
		}
	}

	// And an ordinary name is neither: a checker that treated every name as a prefix claim
	// would pass a reference to `TestFoo` on the strength of a real `TestFooBar`.
	if stem, family := familyOf("TestCorpusRoundTrip"); family || stem != "TestCorpusRoundTrip" {
		t.Errorf("a plain name reads as a family claim for %q", stem)
	}

	// The break quoted mid-sentence rather than taken at a line end, which is how the table in
	// docs/IMPROVEMENTS.md documents the convention. `TestFrame` is not a test, and the row
	// saying so must not be the one thing that fails for saying it.
	row := "| A hyphenated line break | `locale.go:97` ends a line with `TestFrame-` | Continues |"
	if got := testNames(row); len(got) != 0 {
		t.Errorf("a fragment quoted inside a sentence reads as %v; a name written with a "+
			"trailing hyphen is half an identifier and not a claim about a test", got)
	}

	// While a name that really is missing still has to be found, or none of the above matters.
	if got := testNames("as TestThisDoesNotExist shows"); fmt.Sprint(got) !=
		"[TestThisDoesNotExist]" {
		t.Errorf("an ordinary quoted name reads as %v", got)
	}
}

// ---------------------------------------------------------------------------
// the pin every other check is relative to
// ---------------------------------------------------------------------------

// A revision written out in full, and the abbreviation. Both require at least one digit and one
// letter, because seven hex characters with no digit is also how "defaced" is spelt and seven with
// no letter is also how a date or a byte count is written -- and a checker that demanded those be
// the upstream pin would be wrong about ordinary prose.
var (
	fullSHA  = regexp.MustCompile(`\b(?:[0-9a-f]{40})\b`)
	shortSHA = regexp.MustCompile(`\b(?:[0-9a-f]{7})\b`)
	hasDigit = regexp.MustCompile(`[0-9]`)
	hasAlpha = regexp.MustCompile(`[a-f]`)

	// A short revision is only read as a claim about upstream when its own line says so. This
	// repository quotes its *own* commits in the same shape -- docs/windows-first-run.md 85
	// credits `66f6e6f` and docs/IMPROVEMENTS.md 2314 credits `acafec7` -- and those must not
	// be dragged into an assertion about the 1994 C. Hence the per-line requirement rather
	// than a bare hex match, and hence .github/ISSUE_TEMPLATE/fidelity_difference.yml saying
	// "upstream commit" on the line that names it rather than on the line above.
	aboutUpstream = regexp.MustCompile(`(?i)upstream|softdorothy|glider_pro`)
)

func looksLikeSHA(s string) bool { return hasDigit.MatchString(s) && hasAlpha.MatchString(s) }

// TestEveryPlaceThePinIsWrittenNamesTheSameCommit is the one check in this file that the other nine
// rest on.
//
// project.go says it outright: "a citation into a moving target is not a citation", which is why
// the revision is a constant there rather than left implicit. But the revision is *also* written
// into README.md, CONTRIBUTING.md, CHANGELOG.md, docs/DEV_ENVIRONMENT.md, docs/ORIGINAL_GAME.md,
// .gitignore, .gitattributes, a pull-request template, an issue template and the constant at the
// top of this file -- ten places, six of them a pasteable `git checkout`. Nothing held them
// together.
//
// The failure that matters is a bump, not a typo. Repointing at a newer upstream means re-reading
// 17,819 line numbers, so it is the kind of change that gets done to the README and the constant
// and then stopped for the evening -- after which half the tree tells a contributor to check out
// one commit and the other half checks their citations against another, and both halves are
// self-consistent enough to look right.
func TestEveryPlaceThePinIsWrittenNamesTheSameCommit(t *testing.T) {
	// The short form this port reports at runtime -- `glidergo -version` prints it, and the
	// About box carries it -- has to be the front of the long form the clone commands use, or
	// the thing a bug report quotes is not the thing a reader would check out.
	if !strings.HasPrefix(pinnedAt, project.UpstreamCommit) {
		t.Fatalf("project.UpstreamCommit is %q and this file checks against %q: the revision "+
			"the program reports is not the one the citations were taken against",
			project.UpstreamCommit, pinnedAt)
	}
	if len(project.UpstreamCommit) < 7 {
		t.Errorf("project.UpstreamCommit is %q, which is too short to identify a commit",
			project.UpstreamCommit)
	}

	full, short := 0, 0
	for _, ln := range sweep(t) {
		for _, got := range fullSHA.FindAllString(ln.text, -1) {
			if !looksLikeSHA(got) {
				continue
			}
			full++
			if got != pinnedAt {
				t.Errorf("%s:%d writes the revision %s, and the pin is %s",
					ln.path, ln.n, got, pinnedAt)
			}
		}
		if !aboutUpstream.MatchString(ln.text) {
			continue
		}
		for _, got := range shortSHA.FindAllString(ln.text, -1) {
			if !looksLikeSHA(got) {
				continue
			}
			short++
			if !strings.HasPrefix(pinnedAt, got) {
				t.Errorf("%s:%d abbreviates upstream's revision to %s, and the pin is %s",
					ln.path, ln.n, got, pinnedAt)
			}
		}
	}
	// Both forms have to still be found, because the whole assertion above is satisfied by a
	// tree that has stopped saying which commit it was written against -- which is exactly the
	// state the constant in project.go exists to prevent.
	if full < 4 {
		t.Errorf("only %d full revisions in the tree; README.md, CONTRIBUTING.md, "+
			"docs/DEV_ENVIRONMENT.md and docs/ORIGINAL_GAME.md each carry a pasteable "+
			"`git checkout`, so a clone can be put on the right commit without one", full)
	}
	if short < 3 {
		t.Errorf("only %d abbreviated upstream revisions in the tree, so the pin has stopped "+
			"being stated where a reader meets it", short)
	}
	t.Logf("%d full and %d abbreviated mentions of %s, all in agreement", full, short, pinnedAt)
}

// ---------------------------------------------------------------------------
// the checker, checked
// ---------------------------------------------------------------------------

// TestTheCheckerCatchesEachWayACitationCanBeWrong is the test that keeps the rest honest.
//
// Everything above now passes over a repository whose citations have been corrected, which means
// every assertion in this file is one an empty loop would also satisfy. The only way to know the
// checks work is to hand them citations that are wrong -- and the four cases here are the four
// real ones, taken from the repository as it was before this file existed.
func TestTheCheckerCatchesEachWayACitationCanBeWrong(t *testing.T) {
	// Banner.c really is 237 lines; the three bad numbers below were really committed.
	const have = 237

	for _, tc := range []struct {
		why   string
		nums  string
		wants string
	}{
		{"a line past the end of the file", "243", "has 237 lines"},
		{"a range ending past the end", "205-243", "has 237 lines"},
		{"a range that runs backwards", "240-239", "runs backwards"},
		{"a line number of zero", "0", "is not a line"},
	} {
		got := checkSpans("Banner.c", parseSpans(tc.nums), have)
		if len(got) == 0 {
			t.Errorf("%s (Banner.c:%s) is not caught", tc.why, tc.nums)
			continue
		}
		if !strings.Contains(got[0], tc.wants) {
			t.Errorf("%s (Banner.c:%s) is reported as %q, which does not say %q",
				tc.why, tc.nums, got[0], tc.wants)
		}
	}

	// And the citations that are right have to pass, or the checker is just noise. These four
	// are the corrected forms of the ones above, verified by reading the C.
	for _, nums := range []string{"205-236", "232-233", "237", "1"} {
		if got := checkSpans("Banner.c", parseSpans(nums), have); len(got) > 0 {
			t.Errorf("Banner.c:%s is a valid citation and is reported as %q", nums, got)
		}
	}
}

// TestTheCitationSyntaxIsParsedTheWayTheDocumentsWriteIt pins the parser against the forms that
// are actually in the tree, including the one form deliberately *not* matched.
func TestTheCitationSyntaxIsParsedTheWayTheDocumentsWriteIt(t *testing.T) {
	for _, tc := range []struct {
		text  string
		dir   string
		name  string
		spans []span
	}{
		{"see `GliderPRO/Sources/Player.c:1234` for", "Sources", "Player.c", []span{{1234, 1234}}},
		{"(`Room.c:1103-1206`)", "", "Room.c", []span{{1103, 1206}}},
		{"`Play.c:735, 736, 739`", "", "Play.c", []span{{735, 735}, {736, 736}, {739, 739}}},
		{"`GliderPRO/Prefix.h:1` sets", "", "Prefix.h", []span{{1, 1}}},
		{"`Sources_Dynamics2.c:24`", "", "Dynamics2.c", []span{{24, 24}}},
		{"`GliderPRO/Headers/GliderDefines.h:88`", "Headers", "GliderDefines.h", []span{{88, 88}}},
	} {
		got := cCitations([]textLine{{"x.md", 1, tc.text}})
		if len(got) != 1 {
			t.Errorf("%q parses to %d citations, want 1", tc.text, len(got))
			continue
		}
		c := got[0]
		if c.dir != tc.dir || c.name != tc.name {
			t.Errorf("%q parses to %s/%s, want %s/%s", tc.text, c.dir, c.name, tc.dir, tc.name)
		}
		if fmt.Sprint(c.spans) != fmt.Sprint(tc.spans) {
			t.Errorf("%q parses to lines %v, want %v", tc.text, c.spans, tc.spans)
		}
	}

	// The space-separated form is not a citation, and this is the assertion that says so on
	// purpose rather than by omission. "`Banner.c` 300 times" is ordinary prose about a
	// 237-line file, and a checker that read it as a citation would fail on correct writing.
	for _, text := range []string{
		"as `Banner.c` 300 times shows", "Room.c 1206 lines", "Player.c and 12 others",
	} {
		if got := cCitations([]textLine{{"x.md", 1, text}}); len(got) != 0 {
			t.Errorf("%q is prose, not a citation, and parses to %+v", text, got)
		}
	}
}

// sorted returns a map's keys in order, so that a failure lists the same things in the same order
// twice running. Go randomises map iteration, and a test whose output shuffles between runs is one
// nobody can diff.
func sorted[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
