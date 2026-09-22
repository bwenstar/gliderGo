package shell

// Level sets: which collection a house belongs to, and who gets to decide.
//
// docs/PLAN.md §1 asks Stage 2 for "a menu choice between *original* and *new* level sets",
// and its Stage 2 section for "a level-set concept: **Original** (the 22 shipped houses) and
// **New** (ours), chosen in the house-selection UI". This file is the concept. The picker in
// screens.go is the UI, and cmd/glidergo is where the two roots are named.
//
// # A house file cannot say which set it is in
//
// There is nowhere to write it. houseType has a version, a timestamp, a banner, a trailer, a
// start point and 4,070 rooms' worth of everything else, and no field that means "who made
// this" (docs/analysis/house-format.md §3). The 866-byte header is full: every byte of it is
// named and the codec round-trips all 22 shipped files exactly, so there is no slack to
// borrow either. Adding a field would mean writing a house the 1994 game could not read,
// which is the one thing this port does not do to the format it transcribes.
//
// So the set has to be inferred, and there are only two things to infer it from.
//
// # Rejected: infer it from the name
//
// Keep a list of the twenty-two names and call anything on it Original. It reads well and it
// is wrong twice.
//
// It puts the truth about the shipped set in a second place, so the list and
// assets/extracted/houses can drift -- and the day they do, the list is what the game
// believes. And it answers confidently in the one case where the answer matters: a house
// somebody opened in an editor, changed and saved as "Slumberland" is not the 1994
// Slumberland, and a name is exactly the part of it that did not change. A provenance claim
// that cannot fail is not a provenance claim.
//
// # Taken: the source declares it
//
// A Source is a place -- the copy of a root inside this executable, or a directory somebody
// named on the command line -- and the caller that opens one knows what it is. cmd/glidergo
// says so once per root, and every house discovered under it inherits the set.
//
// What that claims is worth stating exactly, because it is less than it looks:
// **a set says where a house was found, not what is in it.** `-houses some/dir` lists whatever
// is in that directory as Other, because the game cannot know what a player put there; and
// the picker draws the root's label beside the count, so the answer is always checkable
// against the place it came from. Compare the About box's upstream pin (screens.go): the
// port's habit is to print the provenance it actually has rather than to infer a better-
// sounding one.
//
// One consequence, and it is the reason this is a type and not a bool: the sets are open at
// the bottom. Stage 3's host picks the house set in play, and a downloaded set of somebody
// else's houses is the obvious fourth root. Adding one is a Source with a Set on it.

import (
	"io/fs"
	"strconv"
	"strings"
)

// Set is which collection a house belongs to.
//
// The order is the order the picker cycles and the order the list breaks a tie in, so
// Original comes first: two houses of the same name in two roots put the shipped one above
// the newcomer, which is the answer a player expects from a list that is otherwise sorted by
// name alone.
type Set int

const (
	// SetOriginal is a house from the root the 22 shipped houses live in. In a clone that is
	// the copy of assets/extracted/houses inside the executable; with -houses it is whatever
	// that directory holds, which is why the picker shows the label too.
	SetOriginal Set = iota

	// SetNew is a house written for this port, from the levels root.
	SetNew

	// SetOther is a house from a root that is neither: a directory a flag named. It is the
	// honest answer to "what is this", and it is the set that keeps the other two from having
	// to lie about a directory nobody here has seen.
	SetOther
)

// AllSets is not a set a house can be in. It is the picker's last filter position -- no
// filter at all -- and it is what the picker opens on, because a list that hides houses by
// default is a list whose player never finds out what is missing.
//
// It is negative so that a zero Set is a real one: a House built by a test or by a caller
// that has no opinion is Original, which is what almost every house in existence is.
const AllSets Set = -1

// String is also what a player reads: the set's name is its display name, so there is only
// one string to keep true and no way for a message and a screen to disagree about it.
func (s Set) String() string {
	switch s {
	case AllSets:
		return "All"
	case SetOriginal:
		return "Original"
	case SetNew:
		return "New"
	case SetOther:
		return "Other"
	}
	return "set " + strconv.Itoa(int(s))
}

// holds reports whether a house belongs in this filter. AllSets holds everything.
func (s Set) holds(h House) bool { return s == AllSets || s == h.Set }

// Source is one place houses come from, and the set the houses in it belong to.
//
// FS and Label are assetfs.Root's two return values, unchanged: the filesystem a house's Rel
// opens against, and what to call it on a terminal ("built-in:houses" for the copy inside
// the executable, otherwise the directory). Set is the caller's declaration -- see the file
// comment for what it does and does not promise.
type Source struct {
	FS    fs.FS
	Label string
	Set   Set
}

// Sets returns the sets that have at least one house in them, in set order.
//
// Only the non-empty ones, because the whole of what the picker does with this is offer
// choices, and a choice that leads to an empty list is not one. A build with no new houses
// yet therefore has exactly one set and shows no chooser at all, rather than a chooser with
// a dead position in it.
func (l *Library) Sets() []Set {
	var out []Set
	for _, s := range []Set{SetOriginal, SetNew, SetOther} {
		if l.Count(s) > 0 {
			out = append(out, s)
		}
	}
	return out
}

// Count is how many houses are in a set. AllSets counts them all.
func (l *Library) Count(set Set) int {
	n := 0
	for i := range l.Houses {
		if set.holds(l.Houses[i]) {
			n++
		}
	}
	return n
}

// Choices is what the picker's Tab cycles through: every set that has houses, and AllSets
// after them once there is more than one set to be shown all of.
//
// Nil for a library with nothing in it, and one element for the ordinary one-set case -- both
// of which the picker reads as "there is nothing to choose here", which is what stops it
// drawing a chooser over a list that has no alternative.
func (l *Library) Choices() []Set {
	sets := l.Sets()
	if len(sets) < 2 {
		return sets
	}
	return append(sets, AllSets)
}

// Tally names each set and its count, for the one status line that has to say what a library
// holds before anybody has opened the picker: "17 Original, 4 New".
//
// Singular by set rather than one total, because the total is the number a player already
// believes they know and the split is the news. With one set it is that set's count and its
// name, which reads the same as the count on its own did.
func (l *Library) Tally() string {
	sets := l.Sets()
	parts := make([]string, 0, len(sets))
	for _, s := range sets {
		parts = append(parts, strconv.Itoa(l.Count(s))+" "+s.String())
	}
	return strings.Join(parts, ", ")
}
