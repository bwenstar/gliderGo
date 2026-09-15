package game

// The whole of Trip.c (245 lines): the fourteen Toggle*, the eight Trigger*, and
// UpdateOutletsLighting.
//
// Twenty-three functions, none longer than twelve lines, and every one of them a *poke*:
// something outside the dynamics -- a switch the glider threw, a fuse that burned down, a
// lamp that came on -- reaches into one dinahs slot and changes one or two fields. The
// handlers in dynamics_movers.go and dynamics_appliances.go then notice on their next frame.
// Nothing here draws, nothing here reads a rect except TriggerFish, and nothing here calls a
// handler directly.
//
// ---------------------------------------------------------------------------
// Toggles are a superset of triggers
// ---------------------------------------------------------------------------
//
// Fourteen types can be switched; eight can be triggered; the eight are a subset of the
// fourteen, with nothing on the other side. A trigger is a switch with a fuse and a smaller
// vocabulary -- so a trigger cannot wake a Mac Plus, a TV, a VCR, a stereo or a microwave,
// and cannot start a ball. Those six plus kSparkle are the seven types FireTrigger's switch
// has no case for; see TestSevenTypesAreNotTriggerable, which pins them as no-ops so that a
// later editor of that switch cannot quietly add a fall-through.
//
// The asymmetry is by design and is not worth resolving. It is what makes a ball a fixture
// of a room's layout rather than a hazard a fuse can spring.
//
// ---------------------------------------------------------------------------
// Every index here comes from DynaNum, and DynaNum can be -1
// ---------------------------------------------------------------------------
//
// All twenty-two pokes index dinahs[index] unguarded in the C, and reach it from
// masterObjects[].dynaNum, which is -1 for anything that never registered: an object past
// the eighteen-slot cap, or one in a locale composed with redraw = true. So dinahs[-1] is
// reachable in the original from a switch wired to a nineteenth appliance. Every function
// here goes through World.dinah instead, which refuses and counts -- one helper for
// twenty-two sites, because a three-line guard inside a one-line function would bury it.
// See IMPROVEMENTS 2.33.
//
// TriggerSwitch is the exception and needs its own guard: its argument is a hotSpots index,
// not a dinahs slot, so it is bounded against a different table.

// ---------------------------------------------------------------------------
// The fourteen Toggle* (Trip.c:20-142)
// ---------------------------------------------------------------------------
//
// HandleSwitches (Interactions.c:1090-1148) is the only caller of all fourteen, and it landed
// in 1.5d, so the two ways into these functions are a glider touching a switch and a trigger
// wired to one (FireTrigger's switch arm forwards through TriggerSwitch). Nothing else in the
// game calls them.
//
// Nine are `active = !active` and nothing else. Three flip and force a four-frame timer.
// One is asymmetric. One can refuse. They are written out one per function rather than
// collapsed into a table because the C has fourteen, HandleSwitches dispatches from
// fourteen separate case arms, and 1.8's fidelity work needs to be able to point at one.

// ToggleToaster is Trip.c:20-25. A switched-off toaster keeps counting -- see HandleToast's
// idle arm, which rewrites Frame = Timer every frame while inactive -- so it always waits a
// full reload period after being switched back on rather than firing at once.
func (w *World) ToggleToaster(index int16) {
	if d := w.dinah(index); d != nil {
		d.Active = !d.Active
	}
}

// ToggleMacPlus is Trip.c:29-36, **the only asymmetric toggle**: forty frames to wake, ten
// to sleep.
//
// HandleMacPlus reads that as the delay before the boot chime (it plays MacOnSound at
// Timer == 30, ten frames in) versus the delay before the screen blanks. So a Mac Plus takes
// four times as long to come on as to go off, which is the joke.
func (w *World) ToggleMacPlus(index int16) {
	d := w.dinah(index)
	if d == nil {
		return
	}
	d.Active = !d.Active
	if d.Active {
		d.Timer = 40
	} else {
		d.Timer = 10
	}
}

// ToggleTV is Trip.c:40-58: a two-line function wrapped around a QuickTime block the port
// does not have.
//
// The C is
//
//	dinahs[index].active = !dinahs[index].active;
//	if ((thisMac.hasQT) && (hasMovie) && (tvInRoom) && (tvWithMovieNumber == index))
//	{ ... GoToBeginningOfMovie / StartMovie / tvOn = true ... StopMovie / tvOn = false ... }
//	dinahs[index].timer = 4;
//
// and the two writes around the block are real here while the block itself is this comment.
// Note what the fourth condition compares: `tvWithMovieNumber` is set at
// ObjectDrawAll.c:707 from *this registration's return value*, so the identity test is
// against a dinahs slot -- the one place in the game where a QuickTime handle is keyed by
// dinah index rather than by object number.
//
// World.TVOn is therefore never written here, which is the correct transcription and not an
// omission: on a machine with no movie the C never enters the block either, so TVOn keeps
// whatever the last movie-bearing room left in it. See the note on that field.
func (w *World) ToggleTV(index int16) {
	d := w.dinah(index)
	if d == nil {
		return
	}
	d.Active = !d.Active
	d.Timer = 4
}

// ToggleCoffee is Trip.c:62-66. The 4 is unconditional in both directions, so a switch
// thrown while the percolator is mid-cycle restarts the cycle rather than reversing it.
func (w *World) ToggleCoffee(index int16) {
	if d := w.dinah(index); d != nil {
		d.Active = !d.Active
		d.Timer = 4
	}
}

// ToggleOutlet is Trip.c:70-73. No timer: an outlet's Timer is its reload countdown and
// HandleOutlet's idle arm only decrements it while Active, so switching one off freezes the
// countdown where it stands instead of restarting it.
func (w *World) ToggleOutlet(index int16) {
	if d := w.dinah(index); d != nil {
		d.Active = !d.Active
	}
}

// ToggleVCR is Trip.c:77-81.
func (w *World) ToggleVCR(index int16) {
	if d := w.dinah(index); d != nil {
		d.Active = !d.Active
		d.Timer = 4
	}
}

// ToggleStereos is Trip.c:85-92, **the only Toggle* that can refuse** -- and the plural in
// the name is the author's, kept because it is what a reader greps for.
//
// A stereo whose timer is still running ignores the switch entirely: no flip, no timer
// reset. HandleStereo holds Timer non-zero for the whole four-frame fade, so a player
// mashing a stereo switch gets one toggle per four frames instead of one per press. Every
// other appliance's switch is honoured mid-cycle; this one is not.
func (w *World) ToggleStereos(index int16) {
	d := w.dinah(index)
	if d == nil {
		return
	}
	if d.Timer == 0 {
		d.Active = !d.Active
		d.Timer = 4
	}
}

// ToggleMicrowave is Trip.c:96-100.
func (w *World) ToggleMicrowave(index int16) {
	if d := w.dinah(index); d != nil {
		d.Active = !d.Active
		d.Timer = 4
	}
}

// ToggleBalloon is Trip.c:104-107. A switched-off balloon in flight keeps flying: the
// Active test is in HandleBalloon's *idle* arm only, so the switch takes effect at the end
// of the current life and not before.
func (w *World) ToggleBalloon(index int16) {
	if d := w.dinah(index); d != nil {
		d.Active = !d.Active
	}
}

// ToggleCopter is Trip.c:111-114.
func (w *World) ToggleCopter(index int16) {
	if d := w.dinah(index); d != nil {
		d.Active = !d.Active
	}
}

// ToggleDart is Trip.c:118-121.
func (w *World) ToggleDart(index int16) {
	if d := w.dinah(index); d != nil {
		d.Active = !d.Active
	}
}

// ToggleBall is Trip.c:125-128, and switching a ball off does **not** make it safe: a
// resting ball is still lethal, because HandleBall tests collisions above its own Moving
// check. Active only decides whether it bounces again.
func (w *World) ToggleBall(index int16) {
	if d := w.dinah(index); d != nil {
		d.Active = !d.Active
	}
}

// ToggleDrip is Trip.c:132-135.
func (w *World) ToggleDrip(index int16) {
	if d := w.dinah(index); d != nil {
		d.Active = !d.Active
	}
}

// ToggleFish is Trip.c:139-142. A switched-off fish whose Timer happens to sit at 3 mod 4
// twitches forever -- HandleFish's bob runs outside its Active test. See that function.
func (w *World) ToggleFish(index int16) {
	if d := w.dinah(index); d != nil {
		d.Active = !d.Active
	}
}

// ---------------------------------------------------------------------------
// The eight Trigger* (Trip.c:146-231)
// ---------------------------------------------------------------------------

// TriggerSwitch is Trip.c:146-149: throw a switch a trigger is wired to.
//
// One line, and **the only Trigger* whose argument is not a dinahs slot**: `who` is a
// hotSpots index. The six switch types carry one in DynaNum instead of a dinah number,
// which is why FireTrigger's switch arm and this function agree without a conversion --
// and which is also hazard H2, since AddDynamicObject's hotSpots index is the wrong one
// in eight of the nine rooms. See the table slice.
//
// So it is bounded against w.R.Hot rather than through World.dinah. HandleSwitches landed in
// 1.5d, so this now forwards into a real function: a trigger wired to a switch throws it, and
// through it reaches everything a switch can reach. See switches_test.go for the effects and
// trip_test.go for the forward and its bound.
func (w *World) TriggerSwitch(who int16) {
	if w.badIndex(devHotSpot, int(who), len(w.R.Hot)) {
		return
	}
	w.HandleSwitches(&w.R.Hot[who])
}

// TriggerToast is Trip.c:153-167.
//
// The active arm launches the bread: VVel = -Count, where Count is the launch speed the
// registration solved for from the author's height, stored positive. (The C's `(short)`
// cast on the negation is redundant -- Count is already a short -- and is the author's
// habit rather than a truncation, so it is not modelled.)
//
// **The else arm is the interesting half.** For a toaster Frame is the idle countdown and
// Timer is the reload period it refills from, so an *inactive* toaster that is triggered
// sets Frame = Timer and thereby **restarts its countdown from full**. Triggering a
// switched-off toaster mid-countdown makes the bread it was about to pop wait another whole
// period -- the poke moves Frame *up*.
func (w *World) TriggerToast(who int16) {
	d := w.dinah(who)
	if d == nil {
		return
	}
	if !d.Moving {
		if d.Active {
			d.VVel = -d.Count
			d.Frame = 0
			d.Moving = true
			w.PlayPrioritySound(ToastLaunchSound, ToastLaunchPriority)
		} else {
			d.Frame = d.Timer
		}
	}
}

// TriggerOutlet is Trip.c:171-184: structurally the toaster's twin, in this type's two
// fields.
//
// Position is the launch/idle enum where the toaster uses Moving, and the inactive arm
// resets Timer from Count where the toaster resets Frame from Timer -- the same "restart the
// countdown" gesture. LengthOfZap is thirty frames, one second at the original's 30.07 fps,
// and HandleOutlet is the only handler in the game that can kill.
func (w *World) TriggerOutlet(who int16) {
	d := w.dinah(who)
	if d == nil {
		return
	}
	if d.Position == 0 {
		if d.Active {
			d.Position = 1
			d.Timer = LengthOfZap
			w.PlayPrioritySound(ZapSound, ZapPriority)
		} else {
			d.Timer = d.Count
		}
	}
}

// TriggerDrip is Trip.c:188-192, **the only Trigger* that shortens a countdown** rather
// than setting or restarting one, and the only one with a two-part gate.
//
// The `Timer > 7` half is what makes it idempotent: a drip already inside its last seven
// frames is left alone, so repeated triggers cannot hold it at 7 forever. Seven is the
// length of the swell animation HandleDrip's idle arm plays on the way to launching -- the
// arms at Timer == 6, 4, 2 and <= 0 -- so the trigger drops the drop into the start of that
// animation rather than launching it outright. The author's own comment is kept.
func (w *World) TriggerDrip(who int16) {
	d := w.dinah(who)
	if d == nil {
		return
	}
	if !d.Moving && d.Timer > 7 {
		d.Timer = 7 // kick off drip
	}
}

// TriggerFish is Trip.c:196-205: the only Trigger* that touches a rect, the only one that
// starts a mover **directly** rather than through the sparkle countdown, and the only one
// gated on Active.
//
// `Whole = Dest` collapses the dirty-rect union to this frame's rect before the first move,
// so the leap's first frame does not repaint the union with wherever the fish was resting
// several seconds ago. Frame = 4 is the mid-strip index a leap starts from.
//
// A triggered fish jumps immediately, with no puff of light -- and a switched-off fish
// cannot be triggered at all, where a switched-off toaster and a switched-off outlet both
// do something.
func (w *World) TriggerFish(who int16) {
	d := w.dinah(who)
	if d == nil {
		return
	}
	if d.Active && !d.Moving {
		d.Whole = d.Dest
		d.Moving = true
		d.Frame = 4
		w.PlayPrioritySound(FishOutSound, FishOutPriority)
	}
}

// TriggerBalloon is Trip.c:209-213, and TriggerCopter and TriggerDart below it are the same
// three lines. **The `+ 1` is load-bearing and must not be simplified.**
//
// The three idle arms decrement Timer and then test it in this order:
//
//	Timer--
//	if Timer <= 0            { launch, sparkle only if Count < StartSparkle }
//	else if Timer == StartSparkle { AddSparkle; EnemyInSound }
//
// so Timer = StartSparkle + 1 = 5 gives **a puff on the next frame and a launch four frames
// after that**: the sparkle gets exactly StartSparkle frames of lead, which is what keeps it
// on screen when the enemy appears, RenderSparkles running one down in five frames.
//
// Setting StartSparkle itself would take Timer to 3 on the first decrement, the `== 4` arm
// would never fire, and the enemy would appear with no puff at all.
//
// The `Count < StartSparkle` guard inside the launch arm is the same problem at the other
// end: an author who sets a reload period shorter than four frames never sees Timer == 4 on
// the way down either, so the puff is issued at launch instead. A deliberate short-period
// fallback, not a bug.
//
// The `!Moving` gate means a trigger fired at a balloon already in flight does nothing and
// does not extend its life. Three functions rather than one with a switch, because the C has
// three and they are dispatched from three separate case arms.
func (w *World) TriggerBalloon(who int16) {
	if d := w.dinah(who); d != nil && !d.Moving {
		d.Timer = StartSparkle + 1
	}
}

// TriggerCopter is Trip.c:218-222. See TriggerBalloon.
func (w *World) TriggerCopter(who int16) {
	if d := w.dinah(who); d != nil && !d.Moving {
		d.Timer = StartSparkle + 1
	}
}

// TriggerDart is Trip.c:227-231. See TriggerBalloon.
func (w *World) TriggerDart(who int16) {
	if d := w.dinah(who); d != nil && !d.Moving {
		d.Timer = StartSparkle + 1
	}
}

// ---------------------------------------------------------------------------
// UpdateOutletsLighting (Trip.c:235-244)
// ---------------------------------------------------------------------------

// UpdateOutletsLighting tells every outlet in one room how many lights that room now has.
//
// An outlet's HVel is its room's light count, and HandleOutlet's last line reads it to
// decide whether the socket's final zap frame redraws the idle art or paints the socket out
// of a dark room. This is the only writer after registration.
//
// **This is also the only reader of Dynamic.Room in the entire game** -- grepped across all
// 67 sources -- and two things follow from that:
//
// The dart never assigns Room (Dynamics3.c:437-462 is the one registration case that omits
// it), and that is harmless by construction rather than by luck. This reader's subjects are
// outlets, and an outlet is one of the seven appliances, which are the only types that can
// register from a room other than the central one. Every other type, the dart included, is
// gated on neighbor == kCentralRoom, so its Room could only ever hold the central room
// number -- the one value the filter never needs to distinguish. A second reader would have
// to be another multi-room type and there are none left, so the dart's omission is not a
// latent bug. Do not "complete" it; TestDartRegistrationLeavesRoomZero pins it.
//
// And **the room filter is load-bearing, not decorative.** numLights is a global that
// DrawLocale reassigns before each of the nine room draws, and AddDynamicObject reads it as
// it registers -- so each room's outlets carry their own room's count. Without the filter,
// turning the central room's lights on would overwrite a neighbouring dark room's outlets
// with a non-zero count and its sockets would stop painting themselves out.
//
// One caller: RedrawRoomLighting, which is gated on the central room crossing between dark
// and lit, and which is itself reached only from HandleSwitches' eight light arms. Both of
// those landed in 1.5d, so throwing a light switch is now the whole path in.
func (w *World) UpdateOutletsLighting(room, nLights int16) {
	for i := int16(0); i < w.NumDynamics; i++ {
		if w.Dinahs[i].Type == Outlet && w.Dinahs[i].Room == room {
			w.Dinahs[i].HVel = nLights
		}
	}
}
