package game

// Triggers: the whole of Triggers.c (205 lines).
//
// A trigger is a pressure plate with a fuse. The glider touches it, ArmTrigger copies
// the *linked* object's identity into a slot with a countdown, HandleTriggers ticks every
// slot once a frame, and when a countdown reaches zero FireTrigger does to the linked
// object whatever that kind of object does. So a trigger in one corner of a room can spill
// grease in another two seconds later, and the delay is the author's, stored on the
// trigger object itself.
//
// The table is a fixed sixteen slots and FindEmptyTriggerSlot returns -1 when they are
// all busy, in which case ArmTrigger **silently does nothing except mark the rect as
// used** -- so a seventeenth simultaneous trigger is consumed and never fires. Kept, per
// the note on the caps in consts.go.
//
// The whole file was ported in 1.5b rather than at 1.5c, because everything except
// FireTrigger's dispatch is self-contained and the timer machinery is worth having
// testable early. Eight of its nine dynamic-object arms became real in 1.5c, with Trip.c;
// only the grease arm's SpillGrease is still a named no-op, charged to 1.5e. Its two
// structural halves -- the localLink split and the grease arm's SetObjectState -- were
// real from the start.

import "encoding/binary"

// MaxTriggers is kMaxTriggers, #defined at Triggers.c:11.
const MaxTriggers = 16

// TriggerSlot is trigType (Triggers.c:14-20): one armed fuse.
//
// Named TriggerSlot rather than Trigger because `Trigger` is already the object type
// code 0x47 in consts.go -- the plate itself. The two are different things and the
// original distinguishes them by case, which Go cannot.
//
// Room and Object are the *linked* object's house coordinates, not the trigger's, which
// is why FireTrigger can pass them straight to SetObjectState. Index is the trigger's
// own master index and is the only way back to it. What is the linked object's type,
// read once at arm time -- and read from `masterObjects[triggers[where].object]`, which
// indexes the master table with an *object number*. See ArmTrigger.
type TriggerSlot struct {
	Object, Room int16
	Index, Timer int16
	What         int16
	Armed        bool
}

// ---------------------------------------------------------------------------
// ArmTrigger (Triggers.c:34-54)
// ---------------------------------------------------------------------------

// ArmTrigger starts a trigger's fuse. Called from the kTriggerIt arm of the collision
// dispatcher.
//
// `who.StillOver` is both the guard and the flag: the early return makes it fire once per
// step-on rather than once per frame, and the unconditional write at the end is what
// arms that guard. Note the write happens even when the slot table was full, which is
// what makes a dropped trigger stay dropped until the glider steps off and back on.
//
// **There is a bug in the `What` line and it is transcribed.** The C reads
//
//	triggers[where].what = masterObjects[triggers[where].object].theObject.what;
//
// where `triggers[where].object` is `masterObjects[whoLinked].objectLink` -- an object
// *number within its room*, 0..23. Indexing the 216-entry master table with it therefore
// names some object of the *first* room in the locale that happens to sit at that slot,
// not the linked object. The correct index is `localLink`, which FireTrigger uses.
//
// Nothing reads TriggerSlot.What. FireTrigger switches on the object it looks up itself, so
// the wrong value is inert -- which is presumably why it survived. It is kept because a
// port that "fixed" it would make the field meaningful and invite a later reader to use
// it, and because a house test that dumps the trigger table has to match the original's
// numbers.
//
// The timer is `delay * 3`: the author's delay is in units of three frames, i.e. about a
// tenth of a second, so the editor's 0..127 covers roughly 0 to 12 seconds.
func (w *World) ArmTrigger(who *HotObject) {
	if who.StillOver {
		return
	}

	where := w.FindEmptyTriggerSlot()

	if where != -1 && who.Who >= 0 && int(who.Who) < len(w.R.Master) {
		whoLinked := who.Who
		mo := &w.R.Master[whoLinked]

		t := &w.Triggers[where]
		t.Room = mo.RoomLink
		t.Object = mo.ObjectLink
		t.Index = whoLinked
		// data.e.delay is a big-endian short at offset 4 of switchType
		// (GliderStructs.h:45-52), not a byte -- the only multi-byte field any of
		// these transcriptions reads.
		t.Timer = int16(binary.BigEndian.Uint16(mo.TheObject.Data[offSwitchDelay:offSwitchDelay+2])) * 3
		// The bug described above, verbatim. Guarded only against a panic.
		if t.Object >= 0 && int(t.Object) < len(w.R.Master) {
			t.What = w.R.Master[t.Object].TheObject.What
		}
		t.Armed = true
	}

	who.StillOver = true
}

// offSwitchDelay is switchType.delay: a short at offset 4 (GliderStructs.h:45-52).
// Alongside the byte offsets in setstate.go.
const offSwitchDelay = 4

// FindEmptyTriggerSlot is Triggers.c:58-74: the first un-armed slot, or -1.
//
// Linear and first-fit, so slot 0 is reused as soon as it frees. That makes the table's
// occupancy order deterministic, which is what lets a fidelity replay compare trigger
// tables at all.
func (w *World) FindEmptyTriggerSlot() int16 {
	for i := range w.Triggers {
		if !w.Triggers[i].Armed {
			return int16(i)
		}
	}
	return -1
}

// ---------------------------------------------------------------------------
// HandleTriggers (Triggers.c:78-96) and ZeroTriggers (Triggers.c:196-205)
// ---------------------------------------------------------------------------

// HandleTriggers ticks every armed fuse once and fires the ones that reach zero.
//
// It is called once per frame from the play loop, and it disarms *before* firing --
// which matters, because FireTrigger can reach SetObjectState, which can re-enter the
// collision dispatcher's world. Disarming first is what stops a trigger re-firing itself
// on the same frame.
//
// A delay of 0 gives Timer 0, which is `<= 0` on the very first tick, so a
// zero-delay trigger fires on the frame after it is armed rather than instantly. That
// one-frame lag is the original's and is visible on a trigger wired to a light.
func (w *World) HandleTriggers() {
	for i := range w.Triggers {
		if w.Triggers[i].Armed {
			w.Triggers[i].Timer--
			if w.Triggers[i].Timer <= 0 {
				w.Triggers[i].Timer = 0
				w.Triggers[i].Armed = false
				w.FireTrigger(int16(i))
			}
		}
	}
}

// ZeroTriggers is Triggers.c:196-205: disarm everything. Part of DrawLocale's reset head,
// so every room change drops fuses that were still burning -- a trigger armed on the way
// out of a room never fires.
//
// It clears only Armed, leaving Timer and the three indices stale, and that is correct
// rather than sloppy: ArmTrigger overwrites all of them, and leaving them readable is
// what makes a post-mortem dump of the table informative.
func (w *World) ZeroTriggers() {
	for i := range w.Triggers {
		w.Triggers[i].Armed = false
	}
}

// ---------------------------------------------------------------------------
// FireTrigger (Triggers.c:100-192)
// ---------------------------------------------------------------------------

// FireTrigger does to the linked object whatever that kind of object does when poked.
//
// Its shape is the one every link-following function in the game has: if the target is
// one of the nine local rooms (LocalLink != -1) it can be poked live, and if it is not,
// only the house copy can be written. But here the two halves are wildly lopsided --
// thirteen arms live and *one* arm remote -- because twelve of the thirteen need a
// dynamic-object slot, and only objects in the locale have one.
//
// So a trigger wired to a balloon two rooms away does nothing at all, and a trigger wired
// to grease two rooms away does spill it, invisibly, and the player finds it on arrival.
// That asymmetry is a feature of the level design and not an accident: it is the only way
// a house can change a room the player has not reached.
//
// **The remote half reads `masterObjects[triggerIs].localLink` after having established
// that it is -1**, and hands that -1 to SetObjectState as `local` -- which is exactly
// what SetObjectState's -1 means ("write the house, publish nothing"), so the apparently
// pointless line is load-bearing. It then hands the same -1 to SpillGrease as a dynamic
// index, which is a genuine out-of-bounds read in the original; the port's SpillGrease
// will have to refuse it.
//
// kSoundTrigger's arm has the author's own comment, `// Change me`, and plays the guitar
// chord because the custom sound loader was not written yet. Reproduced, comment and all
// -- a remotely triggered sound trigger in a shipped house strums.
func (w *World) FireTrigger(index int16) {
	if w.badIndex(devTrigger, int(index), len(w.Triggers)) {
		return
	}
	trig := &w.Triggers[index]

	triggerIs := trig.Index
	if w.badIndex(devMasterObject, int(triggerIs), len(w.R.Master)) {
		return
	}

	if w.R.Master[triggerIs].LocalLink != -1 {
		triggeredIs := w.R.Master[triggerIs].LocalLink
		if w.badIndex(devMasterObject, int(triggeredIs), len(w.R.Master)) {
			return
		}
		target := &w.R.Master[triggeredIs]

		switch target.TheObject.What {
		case GreaseRt, GreaseLf:
			// The one arm shared with the remote half. ForceOn rather than Toggle: a
			// trigger can only ever spill grease, never un-spill it.
			if w.SetObjectState(trig.Room, trig.Object, ForceOn, triggeredIs) {
				w.SpillGrease(target.DynaNum, target.HotNum)
			}

		case LightSwitch, MachineSwitch, Thermostat, PowerSwitch, KnifeSwitch, InvisSwitch:
			// A trigger wired to a switch throws it -- so triggers chain, and a
			// trigger can reach anything a switch can.
			w.TriggerSwitch(target.DynaNum)

		case SoundTrigger:
			// The author's "// Change me". See the note above.
			w.PlayPrioritySound(ChordSound, ChordPriority)

		case Toaster:
			w.TriggerToast(target.DynaNum)

		case Guitar:
			w.PlayPrioritySound(ChordSound, ChordPriority)

		case Coffee:
			w.PlayPrioritySound(CoffeeSound, CoffeePriority)

		case Outlet:
			w.TriggerOutlet(target.DynaNum)

		case Balloon:
			w.TriggerBalloon(target.DynaNum)

		case CopterLf, CopterRt:
			w.TriggerCopter(target.DynaNum)

		case DartLf, DartRt:
			w.TriggerDart(target.DynaNum)

		case Drip:
			w.TriggerDrip(target.DynaNum)

		case Fish:
			w.TriggerFish(target.DynaNum)

		default:
			// Every other object type: a trigger wired to a table, a clock or a mirror
			// arms, counts down, fires and does nothing. The C has no default either.
		}
	} else {
		// The remote half. HGetState/HLock/HSetState around it are the Toolbox's
		// handle locking and have no analogue here.
		rm := w.Room(trig.Room)
		if rm == nil || w.badIndex(devRoomObject, int(trig.Object), MaxRoomObs) {
			return
		}
		// localLink, which the branch condition has just proved is -1. Not redundant:
		// SetObjectState's -1 means "do not publish". See the note above.
		triggeredIs := w.R.Master[triggerIs].LocalLink

		switch rm.Objects[trig.Object].What {
		case GreaseRt, GreaseLf:
			if w.SetObjectState(trig.Room, trig.Object, ForceOn, triggeredIs) {
				// triggeredIs is -1 here, always. SpillGrease has to refuse it; the C
				// reads dinahs[-1].
				w.SpillGrease(triggeredIs, -1)
			}
		}
	}
}

// CoffeeSound and CoffeePriority (GliderDefines.h): the percolator, used only from
// FireTrigger's kCoffee arm and from the coffee cup's own dynamic handler.
const (
	CoffeeSound    int16 = 34
	CoffeePriority int16 = 306
)

// ---------------------------------------------------------------------------
// Where the grease went
// ---------------------------------------------------------------------------
//
// The other eight pokes FireTrigger dispatches to -- TriggerSwitch, TriggerToast,
// TriggerOutlet, TriggerBalloon, TriggerCopter, TriggerDart, TriggerDrip, TriggerFish --
// landed with the rest of Trip.c in 1.5c and are in trip.go.
//
// SpillGrease and RedrawAllGrease landed in 1.5e and are in grease.go, with the rest of
// Grease.c's simulation half. They were stubs here for two stages because the reward path
// needed the names; the file comment there is the one to read for why ten of HandleRewards'
// fourteen arms call RedrawAllGrease at all, which is not obvious from its name.
