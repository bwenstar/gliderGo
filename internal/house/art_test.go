package house

// WantsOwnArt tests.
//
// The predicate decides whether a house that found no resource fork gets told about it
// (docs/IMPROVEMENTS.md 4.17), so both answers are load-bearing and both are wrong in a way
// somebody would notice. A false negative is a house whose backgrounds have all silently fallen
// back to PICT 2000 with nothing said; a false positive is the bug 4.17 is about, a warning on a
// house that never wanted a picture -- which is what `Open House` used to get on a complete asset
// tree, and what its author saw first.
//
// The corpus half at the bottom is the part that could not be written synthetically: which of the
// 22 shipped houses want art is a fact about the 1994 data, and the eight that carry forks are the
// list `make assets` produces.

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWantsOwnArtOnBuiltInBackgroundsAlone(t *testing.T) {
	h := testHouse(testRoom("kitchen", 0, 0, star()), testRoom("hall", 0, 1))
	h.Rooms[0].Background = 2011
	h.Rooms[1].Background = FirstUserBackground - 1
	if h.WantsOwnArt() {
		t.Fatalf("a house whose highest background is %d wants no fork; it is the whole of 4.17 "+
			"that this answers false", FirstUserBackground-1)
	}
}

func TestWantsOwnArtOnAUserBackground(t *testing.T) {
	h := testHouse(testRoom("kitchen", 0, 0, star()), testRoom("gallery", 0, 1))
	h.Rooms[1].Background = FirstUserBackground
	if !h.WantsOwnArt() {
		t.Fatalf("background %d is a house resource by definition", FirstUserBackground)
	}
}

// A kCustomPict counts at any id, including one the application also has. See the comment on
// WantsOwnArt: Metropolis's own PICT 1999 and Fun House's 2014 and 2015 all shadow application
// art, so "below 3000" is no evidence that the fork is unwanted.
func TestWantsOwnArtOnACustomPictBelowTheUserRange(t *testing.T) {
	obj := Object{What: codeCustomPict}
	obj.SetClutter(Clutter{Bounds: Rect{Top: 10, Left: 10, Bottom: 60, Right: 60}, Pict: 1999})
	h := testHouse(testRoom("kitchen", 0, 0, star(), obj))
	h.Rooms[0].Background = 2011
	if !h.WantsOwnArt() {
		t.Fatal("a kCustomPict naming PICT 1999 is exactly Metropolis's case, and Metropolis " +
			"carries its own 1999")
	}
}

// The two objects whose resources are not pictures, and so not this predicate's business: a kTV
// wants a QuickTime movie the port has no support for, and a kSoundTrigger wants a 'snd ' that
// arrives through the sound root's per-house manifest rather than through houseart/. A house of
// nothing but these must stay silent, because the message says pictures.
func TestWantsOwnArtIgnoresTheObjectsWhoseResourcesAreNotPictures(t *testing.T) {
	for _, name := range []string{"kTV", "kSoundTrigger"} {
		t.Run(name, func(t *testing.T) {
			h := testHouse(testRoom("den", 0, 0, star(), plain(name)))
			h.Rooms[0].Background = 2011
			if h.WantsOwnArt() {
				t.Fatalf("%s carries a resource, but not one houseart/ holds", name)
			}
		})
	}
}

// An empty house is the degenerate case and it answers false, which matters because the two
// commands that ask run before anything has drawn: a house with no rooms should fail for having
// no rooms, not for art nobody asked for.
func TestWantsOwnArtOnNoRooms(t *testing.T) {
	if (&House{}).WantsOwnArt() {
		t.Fatal("no rooms, no pictures")
	}
}

// TestWantsOwnArtAgreesWithTheShippedForks is the corpus half, and it is the test that would have
// caught the bug: every shipped house that answers true has a directory of its own under
// assets/extracted/houseart, and every house that answers false has none. That is the claim the
// message makes -- "this house wanted art and there is none here" -- checked against the 1994 data
// rather than against a list in this file.
//
// A fork with no pict/ at all still counts as a fork, because 'bnds' is a house resource too and a
// house may carry openings for a *built-in* background; there is none like that among the 22, and
// the assertion is written so that one would fail here rather than be quietly tolerated.
//
// Skipped without the extracted tree, like the rest of this package's corpus tests: the tree is
// output, and `make check-caveats` says so when it is absent.
func TestWantsOwnArtAgreesWithTheShippedForks(t *testing.T) {
	forks := filepath.Join(repoRoot(t), "assets", "extracted", "houseart")
	if _, err := os.Stat(forks); err != nil {
		t.Skipf("no %s -- run `make assets` first", forks)
	}
	for _, e := range loadCorpus(t) {
		st, err := os.Stat(filepath.Join(forks, e.stem))
		hasFork := err == nil && st.IsDir()
		if got := e.house.WantsOwnArt(); got != hasFork {
			t.Errorf("%s: WantsOwnArt() = %v, but a fork directory on disk is %v",
				e.stem, got, hasFork)
		}
	}
}
