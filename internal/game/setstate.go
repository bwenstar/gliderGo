package game

// This file is Objects.c:366-699: SetObjectState, the one place object state is
// written.
//
// GetObjectState, its counterpart, is already ported as render.Scene.GetObjectState
// and is reachable through the embedded Scene -- the renderer needed it to draw a
// lamp lit or dark, and there is no second copy here.
//
// The two are not mirror images, and the asymmetries are not tidy:
//
//   - kGuitar reads through the appliance group but has its own `changed = false`
//     case for writing, with the original's comment "really no point to change
//     this state".
//   - kKnifeSwitch appears in **neither** switch, so it always reads as on and can
//     never be written. It is the only one of the 117 types that is absent from
//     both.
//   - kSlider has its own bare `break` in both, despite sharing bonusType with the
//     fourteen prizes, so it reads as the default true and writes nothing.
//   - kStereo ignores the object entirely in both directions and is really an
//     accessor for the game's music flag.
//
// The write side also has three bugs that are load-bearing enough to reproduce
// rather than fix. Each is flagged at its site: the prize family writes the wrong
// union member into the master copy, the appliance family indexes the hot-spot
// table without checking for -1, and three paths return an uninitialised local.

// The two sounds a blower makes when switched, and their priorities
// (GliderDefines.h:69-70, :157-158).
const (
	SoundBlowerOn     int16 = 14
	SoundBlowerOff    int16 = 15
	PriorityBlowerOn  int16 = 701
	PriorityBlowerOff int16 = 702
)

// Payload byte offsets of the state fields, from the nine layouts in
// GliderStructs.h:11-106. They are spelled out because SetObjectState's prize-family
// bug *is* an offset confusion, and naming the offsets is what makes it legible: the
// house gets offset 8 and the master copy gets offset 7.
//
//	blowerType    state @ 7   (topLeft 0-3, distance 4-5, initial 6, vector 8, tall 9)
//	bonusType     state @ 8   (topLeft 0-3, length 4-5, points 6-7, initial 9)
//	transportType wide  @ 9   (topLeft 0-3, tall 4-5, where 6-7, who 8)
//	lightType     state @ 9   (topLeft 0-3, length 4-5, byte0 6, byte1 7, initial 8)
//	applianceType state @ 9   (topLeft 0-3, height 4-5, byte0 6, delay 7, initial 8)
//	enemyType     state @ 9   (topLeft 0-3, length 4-5, delay 6, byte0 7, initial 8)
const (
	offBlowerState    = 7
	offBonusState     = 8
	offTransportWide  = 9
	offLightState     = 9
	offApplianceState = 9
	offEnemyState     = 9
)

// SetObjectState is Objects.c:366-699: change one object's state and publish the
// change everywhere that caches it. It returns whether anything actually changed,
// which is what the callers use to decide whether to redraw and whether to play a
// sound.
//
// The four arguments are the original's. `room` and `object` locate the object in
// the house. `action` is Toggle, ForceOn or ForceOff. `local` is the object's index
// in Room.Master, or -1 when the caller does not know it -- and -1 does **not** mean
// "no object": the house is still written. It means "do not publish", so the change
// lands in the house and no cached copy or hot spot learns about it. That is the
// path the two-player synchroniser and the house editor use.
//
// There is no bounds check on `room` or `object` in the original and rooms[] is
// indexed directly. Here the nil from World.Room stands in for that, and returns
// false rather than reading whatever followed the array.
//
// Nine families, and only five of them can be written at all. The other four --
// flames, furniture, most transports, switches, and clutter -- exist in the switch
// solely to return false, which is worth keeping as explicit cases: an author wiring
// a light switch to a table gets nothing, and that has to be nothing rather than a
// fall-through into a family that would have written a byte.
func (w *World) SetObjectState(room, object, action, local int16) bool {
	rm := w.Room(room)
	if rm == nil || w.badIndex(devRoomObject, int(object), MaxRoomObs) {
		return false
	}
	obj := &rm.Objects[object]

	// `changed` is an uninitialised local in the C. Three paths reach the return
	// without assigning it -- kSlider, kKnifeSwitch and any unrecognised `what` --
	// and return whatever was on the stack. Go zeroes it, so those three paths
	// return false here.
	//
	// This is a deliberate divergence and it is the safe direction: false means
	// "nothing changed", so no redraw and no sound, which is what the C intended in
	// all three cases. A true would have made the caller republish an unchanged
	// state. Recorded rather than silently fixed because a fidelity test that
	// compares return values against a 68k trace will disagree here, and the reason
	// should be findable.
	changed := false

	// `newState` is a **file-scope global** in the C (Objects.c:22), not a local.
	// Nothing reads it outside this function, so the port makes it a local -- but
	// the consequence of the C's choice is worth noting: after a call that matched
	// no case, newState still holds the value the *previous* call left, and a
	// debugger stepping through will show a stale value rather than garbage.
	newState := false

	switch obj.What {
	// ---------------------------------------------------------------- blowers
	// The eleven switchable air sources. This is the only family that makes a
	// sound, and the only one whose hot-spot update is *not* gated on the object
	// being in the central room -- the `hotNum != -1` guard carries that instead,
	// since a neighbour's HotNum is always -1.
	case FloorVent, CeilingVent, FloorBlower, CeilingBlower, LeftFan, RightFan,
		SewerGrate, InvisBlower, GrecoVent, SewerBlower, LiftArea:
		cur := obj.Data[offBlowerState] != 0
		switch action {
		case Toggle:
			newState = !cur
			changed = true
		case ForceOn:
			changed = !cur
			newState = true
		case ForceOff:
			changed = cur
			newState = false
		}
		obj.Data[offBlowerState] = b2b(newState)

		if changed && local != -1 && w.masterValid(local) {
			m := &w.R.Master[local]
			m.TheObject.Data[offBlowerState] = b2b(newState)
			// thisRoom->objects[object].data.a.state = newState is the C's third
			// write. This port has no separate thisRoom copy -- see World.ThisRoom
			// -- so the house write above already covers it.
			if newState {
				w.PlayPrioritySound(SoundBlowerOn, PriorityBlowerOn)
			} else {
				w.PlayPrioritySound(SoundBlowerOff, PriorityBlowerOff)
			}
			if m.HotNum != -1 && int(m.HotNum) < len(w.R.Hot) {
				w.R.Hot[m.HotNum].IsOn = newState
			}
		}

	// The five flames. Not switchable: a candle cannot be blown out, which is why
	// CreateActiveRects passes a literal true for every flame rect.
	case Taper, Candle, Stubby, Tiki, BBQ:
		changed = false

	// -------------------------------------------------------------- furniture
	// All fifteen. Not switchable, and there is nowhere for a state to live --
	// furnitureType is eight bytes of bounds and a picture id.
	case Table, Shelf, Cabinet, FilingCabinet, WasteBasket, MilkCrate, Counter,
		Dresser, Stool, Trunk, DeckTable, InvisObstacle, Manhole, Books,
		InvisBounce:
		changed = false

	// ---------------------------------------------------------------- bonuses
	// Fourteen prizes, and this family is write-once: `action` is ignored entirely
	// and the state is always cleared. There is no way to put a collected prize
	// back, which is why a house has to be reloaded to replay it and why
	// SetObjectsToDefaults exists.
	//
	// Note the two differences from the blower family's publication block. The
	// hot-spot update is nested *inside* the `room == thisRoomNumber` test rather
	// than beside it -- same effect, since only the central room has hot spots, but
	// one more condition to satisfy. And the master copy is written through the
	// wrong union member.
	case RedClock, BlueClock, YellowClock, Cuckoo, Paper, Battery, Bands,
		GreaseRt, GreaseLf, Foil, InvisBonus, Star, Sparkle, Helium:
		changed = obj.Data[offBonusState] != 0
		newState = false
		obj.Data[offBonusState] = 0

		if changed && local != -1 && w.masterValid(local) {
			m := &w.R.Master[local]

			// The bug, reproduced. The C writes
			//
			//	masterObjects[local].theObject.data.a.state = false
			//
			// -- data.a, the *blower* member -- inside the bonus family. That is
			// payload offset 7, which in bonusType is the low byte of `points`, not
			// the state byte at offset 8.
			//
			// Two consequences, both observable. The master copy's bonus state is
			// never cleared, so anything reading a collected prize's state out of
			// the object graph still sees it as available. And an invisible bonus's
			// point value loses its low byte in that copy: a 500-point bonus reads
			// as 256 there, a 255-point one as 0.
			//
			// It is harmless in the shipped game only because every consumer reads
			// the house or thisRoom rather than the master copy. Writing offset 8
			// here -- the "obvious fix" -- would change what the object graph says
			// about every collected prize in the locale.
			m.TheObject.Data[offBlowerState] = 0

			if room == w.R.RoomNumber {
				// thisRoom->objects[object].data.c.state = false: correct member,
				// and already covered by the house write.
				if m.HotNum != -1 && int(m.HotNum) < len(w.R.Hot) {
					w.R.Hot[m.HotNum].IsOn = false
				}
			}
		}

	// A bare `break` in the C, so `changed` is returned uninitialised. See the note
	// on the declaration above: false here. kSlider is the type
	// SetObjectsToDefaults also forgets, so a slider is doubly outside the state
	// machine -- which is consistent, since a slide strip has no on and off.
	case Slider:

	// ------------------------------------------------------------- transports
	// Fifteen of the sixteen. Stairs, doors and windows have no state; the mailboxes
	// and ducts have a link but no switch. Only kDeluxeTrans can be turned off.
	case UpStairs, DownStairs, MailboxLf, MailboxRt, FloorTrans, CeilingTrans,
		DoorInLf, DoorInRt, DoorExRt, DoorExLf, WindowInLf, WindowInRt,
		WindowExRt, WindowExLf, InvisTrans:
		changed = false

	// A deluxe transporter keeps its state in the **low nibble of `wide`**, whose
	// high nibble holds the initial state that SetObjectsToDefaults restores from.
	// So every write is read-modify-write: mask to 0xF0, add the new bit. Getting
	// that wrong does not corrupt the current state, it corrupts the *default*, and
	// the damage only shows on the next new game.
	case DeluxeTrans:
		cur := obj.Data[offTransportWide] & 0x0F
		switch action {
		case Toggle:
			newState = cur == 0 // !(wide & 0x0F), through Boolean
			changed = true
		case ForceOn:
			changed = cur == 0x00
			newState = true
		case ForceOff:
			changed = cur != 0x00
			newState = false
		}
		obj.Data[offTransportWide] &= 0xF0
		obj.Data[offTransportWide] += b2b(newState)

		if changed && local != -1 && w.masterValid(local) {
			m := &w.R.Master[local]
			// The whole byte is copied, not just the nibble -- so the master copy
			// picks up the initial-state nibble too. Correct here, unlike the prize
			// family.
			m.TheObject.Data[offTransportWide] = obj.Data[offTransportWide]
			if m.HotNum != -1 && int(m.HotNum) < len(w.R.Hot) {
				w.R.Hot[m.HotNum].IsOn = newState
			}
		}

	// ---------------------------------------------------------------- switches
	// Eight of the nine kinds: a switch is what *sends* state, and has none of its
	// own. kKnifeSwitch is missing from this list in the original -- and from
	// GetObjectState's -- so it is the one type outside the state machine in both
	// directions. Its rect is still made and it can still be flipped; only its own
	// appearance never changes.
	case LightSwitch, MachineSwitch, Thermostat, PowerSwitch, InvisSwitch,
		Trigger, LgTrigger, SoundTrigger:
		changed = false

	// ------------------------------------------------------------------ lights
	// Eight kinds. The only family with no hot-spot update, because a light has no
	// hot spot at all -- switching one changes numLights, and the caller redraws the
	// room's shadow from that.
	case CeilingLight, LightBulb, TableLamp, HipLamp, DecoLamp, Flourescent,
		TrackLight, InvisLight:
		cur := obj.Data[offLightState] != 0
		switch action {
		case Toggle:
			newState = !cur
			changed = true
		case ForceOn:
			changed = !cur
			newState = true
		case ForceOff:
			changed = cur
			newState = false
		}
		obj.Data[offLightState] = b2b(newState)

		if changed && local != -1 && w.masterValid(local) {
			w.R.Master[local].TheObject.Data[offLightState] = b2b(newState)
		}

	// -------------------------------------------------------------- appliances
	// The C's comment is "really no point to change this state", and it is right:
	// a guitar's rect is kStrumIt and always on.
	case Guitar:
		changed = false

	// A stereo is not an object here. It ignores `action`, ignores `room` and
	// `object`, writes no object state and publishes nothing -- it toggles the
	// game's music flag and reports true unconditionally. So a switch wired to a
	// stereo is a music toggle, every press flips it whatever the action asked for,
	// and a trigger sending ForceOn turns the music *off* if it was on.
	case Stereo:
		newState = !w.R.PlayMusicGame
		w.R.PlayMusicGame = newState
		changed = true

	// Eight switchable appliances. kGuitar and kStereo are handled above; the four
	// inert ones below.
	case Shredder, Toaster, MacPlus, TV, Coffee, Outlet, VCR, Microwave:
		cur := obj.Data[offApplianceState] != 0
		switch action {
		case Toggle:
			newState = !cur
			changed = true
		case ForceOn:
			changed = !cur
			newState = true
		case ForceOff:
			changed = cur
			newState = false
		}
		obj.Data[offApplianceState] = b2b(newState)

		if changed && local != -1 && w.masterValid(local) {
			m := &w.R.Master[local]
			m.TheObject.Data[offApplianceState] = b2b(newState)
			if room == w.R.RoomNumber && obj.What == Shredder {
				// The second bug. The C is
				//
				//	hotSpots[masterObjects[local].hotNum].isOn = newState;
				//
				// with **no hotNum != -1 test**, unlike every other family here. A
				// shredder whose hot spot was never created -- the table was full at
				// the time, so AddActiveRect returned -1 -- writes hotSpots[-1],
				// which on 68k is the four bytes before the array.
				//
				// Unreachable in the shipped houses, since none comes close to 56
				// hot spots. The guard is added rather than reproduced because the
				// alternative is a Go panic, and a panic is not what the original
				// did; the divergence is that a corrupt write becomes a no-op.
				if m.HotNum != -1 && int(m.HotNum) < len(w.R.Hot) {
					w.R.Hot[m.HotNum].IsOn = newState
				}
			}
		}

	// Four inert appliances. kCustomPict is here rather than with the clutter
	// because it uses applianceType's layout, and it is the one appliance that makes
	// no hot spot either.
	case CinderBlock, FlowerBox, CDs, CustomPict:
		changed = false

	// ---------------------------------------------------------------- enemies
	// Eight movers. Switching one off stops it moving and stops it killing; this is
	// how a machine switch disables a room full of darts.
	case Balloon, CopterLf, CopterRt, DartLf, DartRt, Ball, Drip, Fish:
		cur := obj.Data[offEnemyState] != 0
		switch action {
		case Toggle:
			newState = !cur
			changed = true
		case ForceOn:
			changed = !cur
			newState = true
		case ForceOff:
			changed = cur
			newState = false
		}
		obj.Data[offEnemyState] = b2b(newState)

		if changed && local != -1 && w.masterValid(local) {
			w.R.Master[local].TheObject.Data[offEnemyState] = b2b(newState)
		}

	// A cobweb is permanent. Its own state byte exists -- enemyType has one -- and
	// nothing ever writes it, so a web cannot be switched off.
	case Cobweb:
		changed = false

	// ---------------------------------------------------------------- clutter
	// All fifteen, kChimes included. Decoration only.
	case Ozma, Mirror, Mousehole, Fireplace, Flower, WallWindow, Bear, Calendar,
		Vase1, Vase2, Bulletin, Cloud, Faucet, Rug, Chimes:
		changed = false
	}

	// No default. An unrecognised `what` -- and the seven undefined codes in the
	// 0x01..0x8F range are reachable, since nothing validates a house's object
	// codes on load -- falls straight through, returning the uninitialised
	// `changed`. false here.
	return changed
}

// masterValid reports whether a `local` index is in range.
//
// The C has no such test: masterObjects[local] is indexed on any non-negative
// local, and the callers are trusted to have got it from the object graph. It is
// added here because the alternative is a panic on an index the C would merely have
// read garbage from, and because SetObjectState is reachable from the two-player
// synchroniser with an index built on the *other* machine's object graph -- which is
// exactly the case where the two can disagree.
func (w *World) masterValid(local int16) bool {
	return local >= 0 && int(local) < len(w.R.Master)
}

// b2b is the C's Boolean-to-byte conversion, which is what makes the deluxe
// transporter's `wide += newState` arithmetic work: a Boolean is one byte holding 0
// or 1, so adding it sets the low nibble to exactly the state bit.
func b2b(v bool) byte {
	if v {
		return 1
	}
	return 0
}
