package game

// Diagnostics and the one guard every out-of-range index in this package goes through.
//
// This file exists because of two release-polish items that turn out to want the same
// field on World (docs/analysis/stage15-raw/plans/spec-normative.md, releasePolish 1 and
// 6). They are opposite in kind and that is the point of keeping them side by side:
//
//	the dropped dirty rects are FAITHFUL. The original drops a rect past 47 of 48 with
//	no report, the port drops it too, and the only new thing here is that somebody can
//	now ask how often. See docs/IMPROVEMENTS.md 2.11.
//
//	the guarded reads are DEVIATIONS. The original reads past the end of the array and
//	the port declines to. Every one of them is a place where a shipped house could make
//	the game read something that is not there. See docs/IMPROVEMENTS.md 2.33.
//
// A bug report wants both numbers, which is why one struct carries them and why
// `glidertool replay` prints them on the same line.

import "fmt"

// Deviation names one out-of-range read the port declined to perform.
//
// Kind is the array in the *original's* terms -- "room object", "master object",
// "trigger" -- rather than the Go expression that would have panicked, because the
// question a maintainer actually has is "which of the C's out-of-bounds reads am I
// standing in for", and answering that at a glance is the whole of releasePolish 6.
type Deviation struct {
	Kind  string
	Index int
	Limit int
}

func (d Deviation) String() string {
	return fmt.Sprintf("%s[%d] of %d", d.Kind, d.Index, d.Limit)
}

// MaxDeviationsSeen caps Diagnostics.Seen.
//
// The list is a sample and not a log: a malformed house can trip the same guard on every
// frame, and a slice that grew with the frame count would turn a diagnostic into a leak.
// The counters are exact; the list is the first sixteen distinct kinds.
const MaxDeviationsSeen = 16

// Diagnostics is World.Diag: what a released build needs to be able to say about a frame
// nobody can see.
//
// Every field is written only by the two helpers below and by the three dirty-rect
// adders, and nothing in the game ever reads them -- so a diagnostic can never change
// what the simulation does. That is deliberate: the replay harness compares these
// numbers between runs, which only means anything if they are outputs.
type Diagnostics struct {
	// DroppedWorkRects and DroppedBackRects count the faithful silent drops at the
	// 47-rect cap, one counter per list. Nonzero means the player saw litter.
	DroppedWorkRects int64
	DroppedBackRects int64

	// Guarded counts every call to badIndex that refused an index, across all kinds.
	Guarded int64

	// Seen is the first MaxDeviationsSeen distinct kinds, in the order they first
	// fired. Distinct by Kind alone: two bad object slots in the same array are the
	// same finding, and the index of the first one is the useful one.
	Seen []Deviation

	// On is a hook for a test or a tool that wants every occurrence rather than the
	// sample. nil in a released build; the replay harness sets it. It is called after
	// the counters are updated, from inside whatever guard fired, so it must not touch
	// the World.
	On func(Deviation)
}

// note records a deviation. See Diagnostics.
func (d *Diagnostics) note(dv Deviation) {
	d.Guarded++
	fresh := true
	for _, s := range d.Seen {
		if s.Kind == dv.Kind {
			fresh = false
			break
		}
	}
	if fresh && len(d.Seen) < MaxDeviationsSeen {
		d.Seen = append(d.Seen, dv)
	}
	if d.On != nil {
		d.On(dv)
	}
}

// badIndex is the single named error path releasePolish 6 asks for: it reports whether i
// is outside [0, n), and records the refusal if it is.
//
// Every caller reads the same way -- `if w.badIndex(...) { return <the C's zero> }` -- and
// the callers are the ones the spec's §6.4 item 3 counts: the room-object slot (24 slots,
// reachable from an unlinked transport, whose objectLink is -1 and whose Byte parameter
// makes that 255), the master-object index, and the trigger index. Ten sites at the time
// of writing; grep for badIndex to enumerate them.
//
// **World.Room is deliberately not one of them**, and neither are
// internal/render/locale.go's three equivalents. Room's nil is the *designed* answer to
// GetNeighborRoomNumber's -1, which every ReadyLevel on an edge room asks for several
// times a frame; counting it would bury the signal under thousands of normal events. The
// render package's guards are the same shape but sit on Scene, which has no game state and
// no business holding a counter -- if they ever need to be reported the answer is to give
// Scene its own Diagnostics, not to reach across the package boundary. Both exclusions are
// recorded in docs/IMPROVEMENTS.md 2.33 rather than only here.
func (w *World) badIndex(kind string, i, n int) bool {
	if i >= 0 && i < n {
		return false
	}
	w.Diag.note(Deviation{Kind: kind, Index: i, Limit: n})
	return true
}

// The kinds. Named constants rather than literals at the call sites, so that the set of
// arrays the port guards is enumerable by reading this file.
const (
	devRoomObject   = "room object"   // rooms[r].objects[i], 24 slots
	devMasterObject = "master object" // masterObjects[i], the composed nine-room graph
	devTrigger      = "trigger"       // theTriggers[i]
	devDynamic      = "dynamic"       // dinahs[i], 18 slots -- reached from DynaNum, which is -1
	devHotSpot      = "hot spot"      // hotSpots[i] -- reached from DynaNum too, for a switch
)
