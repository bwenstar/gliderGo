package shell

// The level sets: the model, the filter and the strip that shows it.
//
// The argument these are all about is in sets.go. What is checkable here is narrower and it is
// the part that rots: that a set travels from the Source that declared it to the house that was
// found under it, that every movement in the picker respects the filter rather than merely
// drawing it, and that a build with one set draws the screen it drew before sets existed --
// which is the property that keeps the fidelity corpus measuring the original.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/platform"
	"github.com/bwenstar/gliderGo/internal/render"
)

// inSet is one house for a synthetic library: a name and the set it came from.
type inSet struct {
	name string
	set  Set
}

// setShell is shellOver with a set on each house. It builds the library by hand for the same
// reason shellOver does -- Discover is tested separately, and what these tests are about is
// what the picker does with a list.
func setShell(t *testing.T, houses []inSet, script ...[]platform.Event) (*Shell, *fake) {
	t.Helper()
	lib := &Library{Root: filepath.Join("test", "houses")}
	for i, h := range houses {
		lib.Houses = append(lib.Houses, House{
			Name:  h.name,
			Set:   h.set,
			Rel:   h.name + ".house",
			Path:  filepath.Join(lib.Root, h.name+".house"),
			Rooms: int16(i + 1),
		})
	}
	lib.Sort()

	f := &fake{scr: render.NewSurface(screenWide, screenTall), script: script}
	s, err := New(f.host(), lib)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s, f
}

// mixed is the library every filtering test below uses: two originals and two new houses,
// interleaved by name so that no filter is a contiguous run of the list. That interleaving is
// the point -- a filter implemented as a pair of bounds would pass a test whose sets happened
// to be contiguous, and the sort deliberately does not group by set.
func mixed() []inSet {
	return []inSet{
		{"Apartments", SetOriginal},
		{"Bakery", SetNew},
		{"Cellar", SetOriginal},
		{"Dovecote", SetNew},
	}
}

func TestASetIsWhatItsStringSaysAndAZeroSetIsOriginal(t *testing.T) {
	// The zero value, which is what a House built by a caller with no opinion has -- and
	// almost every house in existence is an original, so that is the right default. This is
	// also why AllSets is negative.
	var zero Set
	if zero != SetOriginal {
		t.Errorf("the zero Set is %v, want SetOriginal: a house nobody declared a set for "+
			"must not land on a filter position", zero)
	}
	if AllSets >= 0 {
		t.Errorf("AllSets is %d; it must be negative so that no real set can collide with it",
			int(AllSets))
	}

	for _, c := range []struct {
		set  Set
		want string
	}{
		{AllSets, "All"},
		{SetOriginal, "Original"},
		{SetNew, "New"},
		{SetOther, "Other"},
	} {
		if got := c.set.String(); got != c.want {
			t.Errorf("Set(%d).String() = %q, want %q", int(c.set), got, c.want)
		}
	}

	// A set nobody has defined still prints something a reader can act on, rather than
	// "%!v(PANIC=...)" or an empty string in the middle of a title.
	if got := Set(99).String(); !strings.Contains(got, "99") {
		t.Errorf("an unknown set prints %q; it should name its number", got)
	}
}

func TestSetsCountsChoicesAndTally(t *testing.T) {
	lib := &Library{}
	for _, h := range mixed() {
		lib.Houses = append(lib.Houses, House{Name: h.name, Set: h.set})
	}

	if got, want := lib.Sets(), []Set{SetOriginal, SetNew}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("Sets() = %v, want %v", got, want)
	}
	if n := lib.Count(SetOriginal); n != 2 {
		t.Errorf("Count(Original) = %d, want 2", n)
	}
	if n := lib.Count(SetOther); n != 0 {
		t.Errorf("Count(Other) = %d, want 0: nothing declared it", n)
	}
	if n := lib.Count(AllSets); n != 4 {
		t.Errorf("Count(AllSets) = %d, want 4: it is not a set, it is every set", n)
	}

	// Choices has AllSets on the end, and only because there are two sets to be shown all of.
	if got := lib.Choices(); len(got) != 3 || got[2] != AllSets {
		t.Errorf("Choices() = %v, want the two sets and then AllSets", got)
	}
	if got, want := lib.Tally(), "2 Original, 2 New"; got != want {
		t.Errorf("Tally() = %q, want %q", got, want)
	}

	// An empty set is not offered, which is what stops the picker having a position that
	// leads to an empty list.
	one := &Library{Houses: []House{{Name: "Slumberland"}}}
	if got := one.Sets(); len(got) != 1 || got[0] != SetOriginal {
		t.Errorf("Sets() on a one-set library = %v, want just Original", got)
	}
	if got := one.Choices(); len(got) != 1 {
		t.Errorf("Choices() on a one-set library = %v; there is nothing to choose, so AllSets "+
			"must not be offered -- it is what the picker reads to decide not to draw a "+
			"chooser at all", got)
	}
	if got, want := one.Tally(), "1 Original"; got != want {
		t.Errorf("Tally() = %q, want %q", got, want)
	}

	if got := (&Library{}).Choices(); len(got) != 0 {
		t.Errorf("Choices() on an empty library = %v, want none", got)
	}
	if got := (&Library{}).Tally(); got != "" {
		t.Errorf("Tally() on an empty library = %q, want empty", got)
	}
}

// Discover walks every source it is given into one list, and each house remembers which source
// it came from -- both the set, which the picker shows, and the filesystem, which is what opens
// the file.
func TestDiscoverWalksEverySourceAndEachHouseKeepsItsOwn(t *testing.T) {
	originals, extra := t.TempDir(), t.TempDir()
	write(t, filepath.Join(originals, "Slumberland.house"), houseBytes(t, 3))
	write(t, filepath.Join(originals, "Titanic.house"), houseBytes(t, 4))
	write(t, filepath.Join(extra, "Bakery.house"), houseBytes(t, 5))

	lib, err := Discover(
		Source{FS: os.DirFS(originals), Label: originals, Set: SetOriginal},
		Source{FS: os.DirFS(extra), Label: extra, Set: SetNew},
	)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(lib.Houses) != 3 {
		t.Fatalf("found %d houses, want 3: the roots accumulate, they do not replace",
			len(lib.Houses))
	}

	// Sorted by name across the union and not root by root: Bakery is in the second source
	// and belongs first.
	if lib.Houses[0].Name != "Bakery" {
		t.Errorf("the list starts with %q; a house from the second root still sorts by name",
			lib.Houses[0].Name)
	}

	byName := map[string]House{}
	for _, h := range lib.Houses {
		byName[h.Name] = h
	}
	if got := byName["Bakery"].Set; got != SetNew {
		t.Errorf("Bakery is in set %v, want New: the source declares it", got)
	}
	if got := byName["Slumberland"].Set; got != SetOriginal {
		t.Errorf("Slumberland is in set %v, want Original", got)
	}

	// The filesystem travels too, which is the only reason a house in the second root can be
	// opened at all: there is no single filesystem for the library to resolve Rel against.
	for _, h := range lib.Houses {
		if h.FS == nil {
			t.Fatalf("%s carries no filesystem; nothing could open it", h.Name)
		}
		if _, err := lib.Open(h); err != nil {
			t.Errorf("Open(%s): %v", h.Name, err)
		}
	}

	if got, want := lib.Tally(), "2 Original, 1 New"; got != want {
		t.Errorf("Tally() = %q, want %q", got, want)
	}
	// Root names both roots, because there is more than one place a house could have been and
	// a message that named only the first would send somebody looking in the wrong one.
	if !strings.Contains(lib.Root, originals) || !strings.Contains(lib.Root, extra) {
		t.Errorf("Root is %q; it should name both roots", lib.Root)
	}
}

// One unreadable root does not cost the houses in the others. The error still names it, so a
// caller can report the failure and show the list -- which is what cmd/glidergo does.
func TestABadSourceIsReportedAndTheRestAreStillWalked(t *testing.T) {
	good := t.TempDir()
	write(t, filepath.Join(good, "Slumberland.house"), houseBytes(t, 3))
	missing := filepath.Join(t.TempDir(), "nope")

	lib, err := Discover(
		Source{FS: os.DirFS(missing), Label: missing, Set: SetNew},
		Source{FS: os.DirFS(good), Label: good, Set: SetOriginal},
	)
	if err == nil {
		t.Error("a root that is not there should be reported")
	}
	if len(lib.Houses) != 1 {
		t.Fatalf("found %d houses, want the 1 in the root that was readable", len(lib.Houses))
	}
	if lib.Houses[0].Name != "Slumberland" {
		t.Errorf("found %q, want Slumberland", lib.Houses[0].Name)
	}
	// And the failed root is not a set with nothing in it: Sets answers what the library
	// holds, not what it was asked for.
	if got := lib.Sets(); len(got) != 1 || got[0] != SetOriginal {
		t.Errorf("Sets() = %v, want just Original: the New root produced no houses", got)
	}
}

// Two houses with the same name in two sets are two houses, and the shipped one is on top. The
// name is still what sorts first -- a list grouped by set would hide a new house among
// twenty-two originals from the player looking for it where its name belongs.
func TestTheSameNameInTwoSetsPutsTheOriginalFirst(t *testing.T) {
	s, _ := setShell(t, []inSet{
		{"Slumberland", SetNew},
		{"Aardvark", SetNew},
		{"Slumberland", SetOriginal},
	})
	got := make([]string, 0, len(s.lib.Houses))
	for _, h := range s.lib.Houses {
		got = append(got, h.Name+"/"+h.Set.String())
	}
	want := []string{"Aardvark/New", "Slumberland/Original", "Slumberland/New"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("the list is %v, want %v", got, want)
	}
}

// ------------------------------------------------------------------------------- the filter

// The picker opens showing everything. A filter is something a player chooses, and one that is
// on before they have chosen it hides houses from somebody with no way of knowing they are
// there -- which, for the houses this port adds, is exactly the person the sets exist for.
func TestThePickerOpensOnEverySet(t *testing.T) {
	s, _ := setShell(t, mixed())
	if s.filter != AllSets {
		t.Errorf("a fresh shell filters to %v, want AllSets", s.filter)
	}
	s.openPicker()
	if s.filter != AllSets {
		t.Errorf("opening the picker filtered to %v, want AllSets", s.filter)
	}
	if got := len(s.view()); got != 4 {
		t.Errorf("the view holds %d houses, want all 4", got)
	}
}

func TestTabCyclesTheSetsAndTheCursorFollowsWhereItCan(t *testing.T) {
	s, _ := setShell(t, mixed())
	s.openPicker()

	// Original, then New, then All, then round again: Choices' order, which is the sets in
	// set order with AllSets after them.
	want := []Set{SetOriginal, SetNew, AllSets, SetOriginal}
	for i, w := range want {
		s.cycleSet()
		if s.filter != w {
			t.Fatalf("Tab %d left the filter on %v, want %v", i+1, s.filter, w)
		}
		// Every position shows something, and the cursor is always on a house the view
		// holds -- a cursor the list is not drawing is a list with no cursor in it.
		view := s.view()
		if len(view) == 0 {
			t.Fatalf("the %v filter shows nothing; an empty position should not be offered", w)
		}
		if view[s.at(view)] != s.pick {
			t.Errorf("with the %v filter the cursor is on house %d, which the view does not "+
				"hold", w, s.pick)
		}
		if !s.filter.holds(s.lib.Houses[s.pick]) {
			t.Errorf("with the %v filter the cursor is on %s, which is %v",
				w, s.lib.Houses[s.pick].Name, s.lib.Houses[s.pick].Set)
		}
	}

	// And the cursor stays put when the new filter does hold its house. Apartments is
	// Original; filtering from All to Original must not move off it.
	s.filter = AllSets
	s.pick = s.lib.Find("Apartments")
	s.cycleSet()
	if s.filter != SetOriginal {
		t.Fatalf("Tab from AllSets went to %v, want Original", s.filter)
	}
	if got := s.lib.Houses[s.pick].Name; got != "Apartments" {
		t.Errorf("the cursor moved to %s; a filter that still shows the house it was on "+
			"should leave it there", got)
	}
}

// Tab is advertised in the footer only when there is more than one set, so a player pressing it
// in a one-set build has guessed. A guess with no visible effect at all is indistinguishable
// from a key that does not work, so it says what it did instead of swallowing the keystroke.
func TestTabWithNothingToChooseSaysSo(t *testing.T) {
	s, _ := setShell(t, []inSet{{"Slumberland", SetOriginal}, {"Titanic", SetOriginal}})
	s.openPicker()
	before := s.filter
	s.pickerKey(platform.KeyTab)
	if s.filter != before {
		t.Errorf("the filter moved to %v with only one set to show", s.filter)
	}
	if s.mode != modeHouses {
		t.Errorf("Tab left the picker; Escape is the way out and the footer says so")
	}
	if !strings.Contains(s.msg, "Original") || !strings.Contains(s.msg, "no other set") {
		t.Errorf("the status line is %q; it should say what there is and that there is no "+
			"more of it", s.msg)
	}
}

// Every movement is within the view, which is what makes a filter a filter rather than a
// highlight: paging past the end of New must not walk into the originals, and typing A while
// New is showing must not jump to Apartments.
func TestEveryMovementStaysInsideTheFilter(t *testing.T) {
	s, _ := setShell(t, mixed())
	s.openPicker()
	s.filter = SetNew
	s.pick = s.lib.Find("Bakery")

	// Down wraps within the two new houses rather than stepping into Cellar, which is the
	// next house in the list and is Original.
	s.pickerKey(platform.KeyDown)
	if got := s.lib.Houses[s.pick].Name; got != "Dovecote" {
		t.Errorf("Down from Bakery went to %s, want Dovecote: Cellar is the next house in "+
			"the list and it is not in this set", got)
	}
	s.pickerKey(platform.KeyDown)
	if got := s.lib.Houses[s.pick].Name; got != "Bakery" {
		t.Errorf("Down from the last new house went to %s, want Bakery: it wraps within "+
			"the view", got)
	}
	s.pickerKey(platform.KeyUp)
	if got := s.lib.Houses[s.pick].Name; got != "Dovecote" {
		t.Errorf("Up from the first new house went to %s, want Dovecote", got)
	}

	// Paging clamps to the ends of the view.
	s.pickerKey(platform.KeyRight)
	if got := s.lib.Houses[s.pick].Name; got != "Dovecote" {
		t.Errorf("a page down from the last new house went to %s, want to stay", got)
	}
	s.pickerKey(platform.KeyLeft)
	if got := s.lib.Houses[s.pick].Name; got != "Bakery" {
		t.Errorf("a page up went to %s, want the first house of the view", got)
	}

	// Type-select searches the view. A finds nothing, because Apartments is not in it, and
	// the cursor does not move; D finds Dovecote.
	s.pickerKey(platform.KeyA)
	if got := s.lib.Houses[s.pick].Name; got != "Bakery" {
		t.Errorf("A moved the cursor to %s; Apartments is not in this set and a jump to a "+
			"house the picker is not drawing puts the cursor where the player cannot see it",
			got)
	}
	s.pickerKey(platform.KeyD)
	if got := s.lib.Houses[s.pick].Name; got != "Dovecote" {
		t.Errorf("D went to %s, want Dovecote", got)
	}
}

// Opening the picker on a house the filter hides widens the filter rather than moving the
// cursor. The cursor is on the house the player asked for -- from the preferences file, from
// -house, or from the menu -- and the filter is the part they did not ask for.
func TestOpeningThePickerOnAHiddenHouseWidensTheFilter(t *testing.T) {
	s, _ := setShell(t, mixed())
	s.filter = SetNew
	if !s.Select("Apartments") {
		t.Fatal("Select(Apartments)")
	}
	s.openPicker()

	if s.filter != AllSets {
		t.Errorf("the filter is %v; opening the picker on an Original house while showing "+
			"New must widen it, or the list would be drawn with no cursor in it", s.filter)
	}
	view := s.view()
	if view[s.at(view)] != s.pick {
		t.Error("the cursor is still outside the view")
	}
	if got := s.lib.Houses[s.pick].Name; got != "Apartments" {
		t.Errorf("the cursor is on %s, want the selected house", got)
	}
}

// The page number is the cursor's place in the *view*, not in the library. With a filter on,
// a list of twenty is a page of five.
func TestThePageCountIsTheViewsAndNotTheLibrarys(t *testing.T) {
	var houses []inSet
	for i := 0; i < pickerRows+4; i++ {
		// Interleaved, so neither set is a contiguous run and neither is one page of the
		// library. Names are padded so the sort order is the order they are made in.
		set := SetOriginal
		if i%2 == 1 {
			set = SetNew
		}
		houses = append(houses, inSet{string(rune('A'+i)) + "-house", set})
	}
	s, f := setShell(t, houses)
	s.openPicker()

	pages := func() int {
		n := len(s.view())
		p := (n + pickerRows - 1) / pickerRows
		if p == 0 {
			p = 1
		}
		return p
	}
	if got := pages(); got != 2 {
		t.Fatalf("AllSets is %d pages, want 2 of %d houses", got, len(s.lib.Houses))
	}
	s.filter = SetNew
	if got := pages(); got != 1 {
		t.Errorf("the New set is %d pages of %d houses, want 1: the page count follows the "+
			"filter or the picker offers a page with nothing on it", got, len(s.view()))
	}

	// And the drawn footer says so. "page 1 of 1" and not "page 1 of 2".
	s.pick = s.view()[0]
	s.Draw()
	if !inkNear(f.scr, pickRightH-int(render.StringWidth("page 1 of 1")), pickTitleV, render.LtGray8) {
		t.Error("the page counter was not drawn")
	}
}

// ------------------------------------------------------------------------------- the strip

// A build with one set draws the picker it drew before sets existed. That is not tidiness: the
// fidelity corpus holds a hash of this screen, and a chooser with one position in it offers a
// choice that does not exist.
//
// The check is over the strip's own patch of the title line, and pixel for pixel rather than by
// colour, because the patch is not blank -- it is whatever the panel and the splash art behind
// it put there. What it compares against is the same shell with the sets taken off its houses,
// which is the screen a build carrying only the twenty-two draws.
func TestOneSetDrawsNothingNewOnTheTitleLine(t *testing.T) {
	houses := []inSet{{"Slumberland", SetOriginal}, {"Titanic", SetOriginal}}
	one, _ := setShell(t, houses)
	one.openPicker()
	one.Draw()
	before := stripPatch(one.host.Screen)

	// The same two houses in two sets: now there is something to choose, and the patch is used.
	two, _ := setShell(t, []inSet{{"Slumberland", SetOriginal}, {"Titanic", SetNew}})
	two.openPicker()
	two.Draw()
	after := stripPatch(two.host.Screen)

	if before == after {
		t.Error("two sets drew nothing on the title line; the chooser is not there")
	}

	// And the one-set screen is unchanged by the sets existing at all: a library whose houses
	// are all in some other single set draws the identical screen, so nothing about this
	// feature can move a pixel until a build actually has two sets in it.
	other, _ := setShell(t, []inSet{{"Slumberland", SetOther}, {"Titanic", SetOther}})
	other.openPicker()
	other.Draw()
	if got := wholeScreen(other.host.Screen); got != wholeScreen(one.host.Screen) {
		t.Error("a one-set library drew a different screen depending on which set it was; " +
			"with nothing to choose, the set must not reach the pixels at all")
	}
}

// The strip shares the title's line, which is the row that has room. Both ends of the space it
// has to fit in, because adding a fourth set -- which Stage 3's host will -- is the thing that
// breaks it.
func TestTheSetStripFitsBetweenTheTitleAndThePageCounter(t *testing.T) {
	// The widest strip this port can produce: four sets, and counts wide enough to cover any
	// library somebody could assemble by hand. The original's own ceiling is the argument for
	// where to stop -- see the second comment below.
	s, _ := setShell(t, []inSet{
		{"Apartments", SetOriginal}, {"Bakery", SetNew}, {"Cellar", SetOther},
	})
	if got := len(s.lib.Choices()); got != 4 {
		t.Fatalf("this library offers %d choices, want all four", got)
	}
	// Each of the four labels, plus two more digits each: what the strip measures when every
	// set holds up to 999 houses. Three digits is not a round number picked for comfort -- it
	// is the original's own ceiling. `maxFiles` defaults to 48 (Main.c:156) and is a preference
	// the player can raise, clamped to between 12 and 500 (Main.c:88-89), and 500 is three
	// digits. This port has no cap at all, so a library above that is possible and would
	// overflow the row; it is the point past which no 1994 configuration could reach, which is
	// a better place to stop than a guess about what somebody will assemble by hand.
	widest := s.setsWide() + int16(len(s.lib.Choices()))*render.StringWidthScaled("99", 1)

	// The title on the left, at scale 2 with its shadow.
	title := int16(pickLeft+20) + render.StringWidthScaled("Load House", 2) + 2
	if pickSetsH <= int(title) {
		t.Errorf("the strip starts at column %d and the title ends at column %d: pickSetsH "+
			"is too small", pickSetsH, title)
	}

	// The page counter on the right, which is right-aligned at pickRightH.
	counter := pickRightH - int(render.StringWidthScaled("page 10 of 10", 1))
	if end := pickSetsH + int(widest); end >= counter {
		t.Errorf("the widest strip reaches column %d and the page counter starts at column "+
			"%d: four sets no longer fit on the title line, so the strip needs its own row "+
			"-- and the row below the title is taken by the first house's selection bar",
			end, counter)
	}

	// And the strip's bar is inside the panel. This is the check that catches somebody moving
	// the panel rather than the strip.
	if bar := pickSetsV - setsRise; bar <= pickTop+3 {
		t.Errorf("the strip's bar starts at row %d, on the panel's inner frame at %d",
			bar, pickTop+3)
	}
	// The house rows below are the other neighbour, and the thing that put the strip up here:
	// the first row's selection bar rises barRise above its baseline.
	if deep, bar := pickSetsV+2, pickFirst-barRise; deep >= bar {
		t.Errorf("the strip reaches row %d and the first house's selection bar starts at row "+
			"%d", deep, bar)
	}
}

// The strip names each set with its count and marks the one showing, and the names are the
// sets' own String values so there is no second list of display names to disagree with the
// first.
func TestTheStripMarksTheSetThatIsShowingAndCountsTheRest(t *testing.T) {
	s, f := setShell(t, mixed())
	s.openPicker()

	// The labels carry the counts, which is the only place on this screen a player can see
	// how many new houses there are without filtering to them and counting the list.
	if got, want := s.setLabel(SetNew), "New 2"; got != want {
		t.Errorf("the New entry reads %q, want %q", got, want)
	}
	if got, want := s.setLabel(AllSets), "All 4"; got != want {
		t.Errorf("the All entry reads %q, want %q", got, want)
	}

	// AllSets is where the picker opens, so the first entry -- Original -- is drawn plain.
	s.Draw()
	if !inkNear(f.scr, pickSetsH, pickSetsV, render.LtGray8) {
		t.Error("the Original entry was not drawn in grey while All was the filter")
	}

	// Filter to Original and its entry is inverse video: black on cream, which is what every
	// other selection in this shell is.
	s.filter = SetOriginal
	s.Draw()
	w := int(render.StringWidthScaled(s.setLabel(SetOriginal), 1))
	if n := inkInRect(f.scr, pickSetsH-setsPad, pickSetsV-setsRise,
		pickSetsH+w+setsPad, pickSetsV+2, cream); n == 0 {
		t.Error("the showing set has no bar under it")
	}
	if !inkNear(f.scr, pickSetsH, pickSetsV, render.Black8) {
		t.Error("the showing set's name is not drawn in black on its bar")
	}
}

// The footer names the house's set while the list is showing all of them, and stops when a
// filter is on -- where the title and the strip both already say it, and the line has better
// things to do with its width.
func TestTheFooterNamesTheSetOnlyWhenTheListIsMixed(t *testing.T) {
	s, _ := setShell(t, mixed())
	s.openPicker()
	s.pick = s.lib.Find("Bakery")

	if got := s.footerBest(); !strings.Contains(got, "New") {
		t.Errorf("with AllSets the first footer line is %q; it should name the house's set", got)
	}
	s.filter = SetNew
	if got := s.footerBest(); strings.Contains(got, "[New]") {
		t.Errorf("with the New filter on the first footer line is %q; the title and the "+
			"strip already say it", got)
	}

	// And a one-set library never says it at all, because there is nothing to distinguish.
	one, _ := setShell(t, []inSet{{"Slumberland", SetOriginal}})
	one.openPicker()
	if got := one.footerBest(); strings.Contains(got, "Original") {
		t.Errorf("a one-set library's footer says %q; there is no other set to tell it "+
			"apart from", got)
	}
}

// The opening status line is where a player finds out the new houses are there at all, before
// they have pressed anything.
func TestTheOpeningLineSaysTheSplitOnlyWhenThereIsOne(t *testing.T) {
	mix, _ := setShell(t, mixed())
	if got := mix.opening(); !strings.Contains(got, "2 Original, 2 New") {
		t.Errorf("the opening line is %q; with more than one set it should say the split, "+
			"because \"4 houses\" is a number a player would read as the originals", got)
	}
	one, _ := setShell(t, []inSet{{"Slumberland", SetOriginal}})
	if got, want := one.opening(), "1 house"; got != want {
		t.Errorf("the opening line is %q, want %q: one set reads as it did before sets "+
			"existed", got, want)
	}
}

// ------------------------------------------------------------------------------- helpers

// stripPatch is the strip's patch of the title line as a string of pixel values, for comparing
// one drawn screen against another. A count of "ink" cannot do this job: the patch sits over
// the panel and the splash art, so it is never blank and there is no background colour to
// subtract.
func stripPatch(s *render.Surface) string {
	var b strings.Builder
	for v := pickSetsV - setsRise; v <= pickSetsV+2; v++ {
		for x := pickSetsH - setsPad; x < pickRightH; x++ {
			b.WriteByte(s.Pix[v*s.W+x])
		}
	}
	return b.String()
}

// wholeScreen is every pixel, for the assertion that a one-set library draws the same screen
// whichever set that is.
func wholeScreen(s *render.Surface) string { return string(s.Pix) }

// inkInRect is inkInRows for one colour in one rectangle.
func inkInRect(s *render.Surface, left, top, right, bottom int, idx uint8) int {
	n := 0
	for v := top; v <= bottom; v++ {
		for x := left; x <= right; x++ {
			if s.Pix[v*s.W+x] == idx {
				n++
			}
		}
	}
	return n
}
