package game

// HandleSwitches (Interactions.c:985-1153) -- the switches.
//
// A hundred and seventy lines over two switch statements, and structurally it is
// FireTrigger's bigger sibling: resolve the switch's link, change the linked object's
// state, animate the lever, and then tell whatever is at the other end of the wire that it
// has changed. Everything a switch can do a trigger can do, and both go through
// SetObjectState.
//
// ---------------------------------------------------------------------------
// Two switches, and the second one is the interesting half
// ---------------------------------------------------------------------------
//
// The first switch is on the *switch's own* type and draws its lever -- five kinds of wall
// plate plus an invisible one. The second is on the *linked object's* type, twenty-three arms
// covering forty-two object types, and it exists because changing a state byte is not
// enough for anything that has a presence on screen: a lamp needs the room's shadow
// recomputed, a prize needs the background put back over it, a dinah needs its table entry
// woken up. That second switch is the whole reason a switch cannot just call
// SetObjectState and stop.
//
// It runs only when `linkIndex != -1` -- when the target is one of the nine local rooms.
// A switch wired to a room the player cannot see writes the house copy and nothing else,
// and the player finds the change on arrival. That is the same split FireTrigger has, and
// it is why both need SetObjectState's `local` argument to be optional.
//
// **Three of the twenty-three arms are unreachable**, and all three for the same reason: the
// second switch runs only when SetObjectState returned true, and SetObjectState returns false
// for kSlider, for kSoundTrigger and for kGuitar. So a switched sound trigger never plays its
// sound and a switched guitar never strums, however plainly the code says they should. All
// three are transcribed anyway -- they record what the author intended, and a later change to
// SetObjectState would bring them to life untested. TestThreeSwitchArmsCannotBeReached pins
// the fact so that such a change is noticed.
//
// ---------------------------------------------------------------------------
// It reads World.NewState, which is why that is a field
// ---------------------------------------------------------------------------
//
// The lever is drawn from the state SetObjectState just computed for the *linked* object,
// which SetObjectState leaves in a global. So a light switch shows the lamp's state, not
// its own -- correct, and impossible without the shared variable. See World.NewState.
//
// ---------------------------------------------------------------------------
// Two bugs here, both reproduced, both recorded in docs/IMPROVEMENTS.md
// ---------------------------------------------------------------------------
//
//	the sparkle rect  `bounds` is declared and never assigned before AddSparkle reads
//	                  it. On 68k that was stack garbage; here it is Go's zero, so the
//	                  spurious puff lands at the corner of the play area instead of
//	                  somewhere random. The *correct* sparkle is emitted a line earlier
//	                  by RestoreFromSavedMap's own doSparkle arm, so this one is pure
//	                  noise -- which is what makes it safe to reproduce and safe to turn
//	                  off, which Fixes.SwitchSparkle does. **The 22 shipped houses reach it
//	                  145 times**, so it is a thing players of the original content see,
//	                  not a curiosity; TestShippedHousesWireSwitchesToPrizes is the count.
//	the missing star  the star arm is grouped with the eight other prizes, so a star
//	                  removed by a switch gets neither StopStar nor a decrement of
//	                  StarsLeft. Two consequences: its six-cel spin keeps animating over
//	                  the restored background, and the house becomes uncompletable,
//	                  because the star can never be collected and the count can never
//	                  reach zero. An author can build a house that cannot be won -- and
//	                  none of the 22 shipped houses does, so this one is a hazard for new
//	                  houses and for the editor rather than a defect in the originals.
//
// ---------------------------------------------------------------------------
// One thing it does not have
// ---------------------------------------------------------------------------
//
// It takes the hot spot and not the glider: nothing a switch does depends on which player
// threw it, so the two-player escape protocol has no branch here and a switch thrown by
// either glider is the same event.

import (
	"glidergo/internal/game/player"
	"glidergo/internal/render"
)

// The switch click (GliderDefines.h:64, :156). One sound for all five plates -- the
// thermostat and the knife switch click like a light switch.
const (
	SwitchSound    int16 = 9
	SwitchPriority int16 = 700
)

// HandleSwitches is Interactions.c:985-1153: the glider has touched a switch.
//
// `who.StillOver` is the once-per-step-on guard, read at the top and written at the
// bottom, so holding the glider against a light switch flips it once rather than sixty
// times a second. Note the write happens whether or not anything changed -- so a switch
// wired to a table (SetObjectState returns false for furniture) still latches, and the
// glider has to step off and back on to try again.
func (w *World) HandleSwitches(who *HotObject) {
	if who.StillOver {
		return
	}

	// The C indexes masterObjects[who->who] unguarded, as everywhere else in this file.
	// The guard is inside the StillOver latch rather than around it, so a bad index still
	// arms the latch -- if it did not, an unresolvable switch would be re-dispatched on
	// every frame of the room.
	if w.masterValid(who.Who) {
		whoLinked := who.Who
		m := &w.R.Master[whoLinked]
		roomLinked := m.RoomLink
		objectLinked := m.ObjectLink
		linkIndex := m.LocalLink

		// `bounds` is the C's uninitialised local, reproduced. See the file comment: the
		// only reader is the prize arm's spurious AddSparkle, and Go's zero value is what
		// makes the glitch deterministic rather than stack-dependent.
		var bounds Rect

		// data.e.type is the author's choice of Toggle, ForceOn or ForceOff, stored on
		// the switch object. It is what makes a "switch" able to be a one-way switch.
		if w.SetObjectState(roomLinked, objectLinked, int16(m.TheObject.Switch().Type), linkIndex) {
			newRect := render.Offset(who.Bounds, w.R.V.OriginH, w.R.V.OriginV)

			// The C spells this as five cases with the same sound and five different
			// Draw functions; render.Scene.DrawSwitch is those five collapsed, because
			// they differ only in which pair of sub-rects they index. kInvisSwitch is
			// the sixth case and is a bare break -- an invisible switch is silent as
			// well as invisible, which is worth stating because the sound is inside the
			// arms rather than above them.
			switch m.TheObject.What {
			case LightSwitch, MachineSwitch, Thermostat, PowerSwitch, KnifeSwitch:
				w.PlayPrioritySound(SwitchSound, SwitchPriority)
				w.R.DrawSwitch(m.TheObject.What, newRect, w.NewState)
			case InvisSwitch:
			}

			// Both of these run for an invisible switch too, outside the switch above.
			// DrawSwitch drew into the *background* -- a thrown lever is a permanent
			// change to the room -- so this promotes it into the work map and then puts
			// that on screen. An invisible switch spends the two rect slots on copying
			// unchanged pixels, which is the original's behaviour and costs one of the
			// 47 (see docs/IMPROVEMENTS.md 2.11).
			w.CopyRectBackToWork(newRect)
			w.AddRectToWorkRects(player.Rect(newRect))

			if linkIndex != -1 && w.masterValid(linkIndex) {
				w.switchLinkedObject(roomLinked, objectLinked, linkIndex, bounds)
			}
		}
	}

	who.StillOver = true
}

// switchLinkedObject is the second of HandleSwitches' two switch statements
// (Interactions.c:1035-1148): tell the object at the other end of the wire that its state
// byte has changed.
//
// Split out only for depth -- the C has it nested four levels in. Its arms are in the
// original's order, which groups them by what they need rather than by object family, and
// that grouping is the useful thing about it:
//
//	nine prizes           the background comes back and a puff of light says so
//	the cuckoo            the same, plus its pendulum stops
//	the two grease jars   SpillGrease, exactly as the reward path does it
//	four types            nothing at all, explicitly
//	the sound trigger     plays the house's custom sound -- unreachable
//	the eight lights      RedrawRoomLighting, which may recompose the whole room
//	the guitar            strums -- unreachable
//	the fourteen dinahs   the matching Toggle*, all of which landed in 1.5c
//
// The three unreachable arms are marked above and in place below. kSlider is the third and
// is inside the "four types" group, which is why it does not show in this summary.
//
// `bounds` is the caller's uninitialised rect. It is passed in rather than re-declared so
// that there is one place to read the bug off.
func (w *World) switchLinkedObject(roomLinked, objectLinked, linkIndex int16, bounds Rect) {
	m := &w.R.Master[linkIndex]

	switch m.TheObject.What {
	// Nine of the fourteen prizes. The restore is doSparkle=true here and false on the
	// reward path, because a prize removed by a switch vanishes somewhere the player may
	// not be looking: the puff of light and the fade-out sound are what say it happened.
	//
	// **kStar is in this group and should not be**: see the file comment. No StopStar, so
	// the spin outlives the star, and no StarsLeft decrement, so the house's count is
	// permanently one short of winnable.
	case RedClock, BlueClock, YellowClock, Paper, Battery, Bands, Foil, Star, Helium:
		w.RestoreFromSavedMap(roomLinked, objectLinked, true)
		// The spurious second sparkle, on the zero rect. The one the player is meant to
		// see was emitted by the line above, which is why Fixes.SwitchSparkle can drop
		// this one without the switch losing its puff of light.
		if !w.Fix.SwitchSparkle {
			w.AddSparkle(bounds)
		}

	// The cuckoo is the one prize whose arm differs, and the difference is the pendulum
	// rather than the sparkle -- it gets no AddSparkle at all, spurious or otherwise, so
	// the only puff it produces is the restore's own.
	case Cuckoo:
		w.RestoreFromSavedMap(roomLinked, objectLinked, true)
		w.R.StopPendulum(roomLinked, objectLinked)

	// Identical to the reward path's grease arm, including the absence of a restore: a
	// knocked-over jar becomes a slick in place rather than being removed. 1.5e.
	case GreaseRt, GreaseLf:
		w.SpillGrease(m.DynaNum, m.HotNum)

	// Four explicit nothings. The first two have no presence to update -- an invisible
	// bonus was never drawn and a slider has no state -- and the deluxe transporter's
	// state is read at use time, so switching it off needs no redraw. A shredder's blades
	// are drawn by its own dynamic handler.
	//
	// kSlider is the first of the three unreachable arms: having no state is exactly why
	// SetObjectState's kSlider case is a bare break returning false. Its being grouped with
	// kInvisBonus makes no difference, since both arms are empty.
	case InvisBonus, Slider:
	case DeluxeTrans:
	case Shredder:

	// A switch wired to a sound trigger was how a house author was to get an arbitrary sound
	// out of a switch. TriggerPriority is 999, the highest in the game.
	//
	// The second unreachable arm, and the one whose deadness is a loss rather than a
	// curiosity: kSoundTrigger sits in SetObjectState's *switch* family, which returns false
	// because a switch has no state of its own, so this line has never run in any build of
	// the game. FireTrigger's own kSoundTrigger arm does work, which is presumably why
	// nobody noticed.
	case SoundTrigger:
		w.PlayPrioritySound(TriggerSound, TriggerPriority)

	// The eight lights. RedrawRoomLighting recounts and, only if the room crossed the
	// lit/unlit boundary, recomposes the central room -- so switching the second of two
	// lamps costs a recount and nothing else. See readylevel.go.
	case CeilingLight, LightBulb, TableLamp, HipLamp, DecoLamp, Flourescent,
		TrackLight, InvisLight:
		w.RedrawRoomLighting()

	// The third and last unreachable arm. The guitar is the one appliance with no state to
	// change -- SetObjectState's kGuitar arm returns false, with the author's own comment
	// "really no point to change this state" -- and the caller only gets here when
	// SetObjectState returned true. Transcribed because the original has it, and because it
	// says what the author intended a switched guitar to do.
	case Guitar:
		w.PlayPrioritySound(ChordSound, ChordPriority)

	// The fourteen dinahs. All fourteen Toggle* landed in 1.5c (trip.go) and each takes
	// the object's DynaNum -- an index into the Dinahs table, which is only valid because
	// linkIndex != -1 put the object in the locale. An object whose registration was
	// refused (the table was full) has DynaNum -1, and every Toggle* refuses it.
	case Toaster:
		w.ToggleToaster(m.DynaNum)
	case MacPlus:
		w.ToggleMacPlus(m.DynaNum)
	case TV:
		w.ToggleTV(m.DynaNum)
	case Coffee:
		w.ToggleCoffee(m.DynaNum)
	case Outlet:
		w.ToggleOutlet(m.DynaNum)
	case VCR:
		w.ToggleVCR(m.DynaNum)
	case Stereo:
		w.ToggleStereos(m.DynaNum)
	case Microwave:
		w.ToggleMicrowave(m.DynaNum)
	case Balloon:
		w.ToggleBalloon(m.DynaNum)
	case CopterLf, CopterRt:
		w.ToggleCopter(m.DynaNum)
	case DartLf, DartRt:
		w.ToggleDart(m.DynaNum)
	case Ball:
		w.ToggleBall(m.DynaNum)
	case Drip:
		w.ToggleDrip(m.DynaNum)
	case Fish:
		w.ToggleFish(m.DynaNum)
	}
}
