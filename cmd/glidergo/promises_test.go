package main

// What a version number promises, as values a test can hold (docs/PLAN.md, "After the gate: what
// a version number promises").
//
// From v0.2.0 the numbering is semantic versioning for 0.x. A patch release, 0.2.1 after 0.2.0,
// is bug fixes only. It races every other 0.2.x and reads every file they wrote, and it can
// promise that only because every value pinned in this file is the one the last tag had. A change
// to any of them makes the next tag a minor release, 0.3.0 after 0.2.x, whose CHANGELOG section
// says what it no longer races or reads. So half of choosing the number is one diff (RELEASING.md,
// step 2): a pinned value changed since the last tag is a minor, whatever else the tag carries.
//
// None of these values is wrong to change. What a failure here asks is that the change be a
// decision, made on purpose and written down, and not something a release finds out afterwards.

import (
	"testing"

	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/netplay"
	"github.com/bwenstar/gliderGo/internal/prefs"
)

// news is what every failure here says after its own sentence.
const news = "; if that is a decision, write the new value here, number the next tag as a " +
	"minor release (0.3.0 after 0.2.x), and say in CHANGELOG.md what it no longer races or reads " +
	"(docs/PLAN.md, \"what a version number promises\")"

// The engine fingerprint this build's races carry, pinned.
//
// It is measured, and changes whenever the simulation does (replay.Engine), so it is not wrong to
// change. But from the build that changes it, races with every release before it are refused.
// When this fails, the change that moved it is a change to how the game flies, and that is either
// a bug or a decision.
func TestTheEngineFingerprintIsPinned(t *testing.T) {
	const pinned = 0xE82F4D7565428514
	got, err := engineFingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if got != pinned {
		t.Errorf("the engine fingerprint is %016X, was %016X: this build flies differently from "+
			"the last, and will not race it"+news, got, uint64(pinned))
	}
}

// The numbers that say which files and which peers a build understands, pinned.
//
// Three readers refuse by number: a saved game of another container version, a score side-car of
// another size, and a race whose envelope byte differs. The score side-car carries no number of
// its own, because it is 1994's scoresType, so its size is its format. The settings file is not
// refused, because Validate reads what it can in either direction, but a bump says a field now
// means something else, and a build before it reads the newer meaning into the field it knows.
// prefs.Version's comment gives the way round that: a new key rather than a bump.
func TestTheFormatNumbersArePinned(t *testing.T) {
	for _, c := range []struct {
		what        string
		got, pinned int
	}{
		{"prefs.Version, the settings file's schema", prefs.Version, 1},
		{"house.SavedGameFormat, the saved game's container version", house.SavedGameFormat, 1},
		{"house.SizeofScores, the score side-car's size in bytes", house.SizeofScores, 292},
		{"netplay.Version, the race protocol's envelope byte", int(netplay.Version), 1},
	} {
		if c.got != c.pinned {
			t.Errorf("%s is %d, was %d: the builds before this one refuse or misread what it "+
				"writes"+news, c.what, c.got, c.pinned)
		}
	}
}
