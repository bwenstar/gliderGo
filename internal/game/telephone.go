package game

// The telephone and the wind chimes: InitTelephone (Play.c:733-740), HandleTelephone
// (Play.c:745-789) and StrikeChime (Play.c:793-796).
//
// Neither is an object the glider can touch and neither affects the simulation. They are
// the house's ambience -- a phone that rings somewhere off screen every few minutes, and
// chimes that tinkle in a room that has them -- and they are in Play.c rather than in
// Dynamics.c because they belong to the *house*, not to any room. The phone rings
// wherever the player is.
//
// They share one struct and one function because they are the same mechanism at two
// timescales, but only the phone uses all three fields; see PhoneState.

// The four timing constants, #defined at the top of Play.c (:20-23) and used nowhere else.
//
// RingSpread and RingBaseDelay are in frames, so at the original's 30.07 fps the phone's
// silence between bursts is a uniform draw over about 166 seconds on top of a fixed 166
// -- somewhere between two minutes forty and five minutes twenty. That is long enough
// that most players never hear it twice in a house.
//
// RingDelay is the gap *between* the rings of one burst: 90 frames, almost exactly three
// seconds, which is a British ring cadence rather than an American one.
//
// ChimeDelay is divided by the number of chimes in the room, so a room with six sets of
// chimes is six times as busy. See HandleTelephone.
const (
	RingDelay     int16 = 90
	RingSpread    int16 = 25000
	RingBaseDelay int16 = 5000
	ChimeDelay    int16 = 180
)

// The phone and chime sounds (GliderDefines.h:110-112, :127-128, :155).
//
// There are two chime sounds and HandleTelephone picks between them with a coin flip, at
// two different priorities that differ by one -- so when both are somehow due at once the
// second wins, which is a distinction with no audible consequence.
const (
	PhoneRingSound int16 = 55
	Chime1Sound    int16 = 56
	Chime2Sound    int16 = 57

	Chime1Priority    int16 = 203
	Chime2Priority    int16 = 204
	PhoneRingPriority int16 = 500
)

// PhoneState is phoneType (Play.c): the ringing clock.
//
// Two instances exist, `thePhone` and `theChimes`, and **the chimes only use NextRing**.
// Rings and Delay are dead on that instance: InitTelephone writes neither, and
// HandleTelephone's chime half reads neither. The shared struct is kept rather than split
// into a phone struct and an int16 because it is the original's shape and because saying
// so here is more useful than hiding it -- a reader who sees Chimes.Delay in a debugger
// needs to know it is meaningless rather than zero.
type PhoneState struct {
	// NextRing counts down to the next ring, in frames.
	NextRing int16

	// Rings is how many rings are left in the current burst: three to five, redrawn
	// each burst. Phone only.
	Rings int16

	// Delay counts down between the rings of a burst. Phone only.
	Delay int16
}

// InitTelephone is Play.c:733-740: seed both clocks. Called once per game from NewGame,
// not per room -- the phone's schedule survives every room change and every death.
//
// **Four RandomInt draws in a fixed order**, and their order is part of the game's random
// stream: anything that reseeds or reorders them shifts every later draw in the game.
// That is the constraint that makes this function worth transcribing exactly even though
// nothing it does is visible.
//
// `RandomInt(3) + 3` gives three to *six* rings, not three to five, because RandomInt's
// upper bound is inclusive one time in 65536. See RandomInt.
func (w *World) InitTelephone() {
	w.Phone.NextRing = w.RandomInt(RingSpread) + RingBaseDelay
	w.Phone.Rings = w.RandomInt(3) + 3
	w.Phone.Delay = RingDelay

	w.Chimes.NextRing = w.RandomInt(ChimeDelay) + 1
}

// HandleTelephone is Play.c:745-789: tick both clocks. Called once per frame from the
// play loop.
//
// **PhoneBitSet inverts.** The house flag is called "phone" and setting it *silences* the
// phone: HouseIO.c:417 reads it out of the house's flags word (bit 1) and the whole phone
// half of this function is inside `if (!phoneBitSet)`. So the flag an author ticks in the
// house-info dialogue means "this house has no telephone". Nothing in the name says that,
// and it is the kind of inversion a port gets backwards once and then hears about.
//
// The two halves are otherwise unalike:
//
//   - The phone's cadence is a nested countdown: NextRing to zero, then Delay to zero
//     over and over, one ring each time, until Rings hits zero and both are redrawn.
//     Note that NextRing is left at 0 during a burst and only re-armed when the burst
//     ends, and that Delay is reset *before* the sound rather than after, so the first
//     ring of a burst comes 90 frames after NextRing expires rather than immediately.
//   - The chimes have no burst structure at all. One strike, then a fresh delay drawn
//     from ChimeDelay/NumChimes, floored at 2 -- so more chimes in a room means more
//     frequent strikes, and 90 or more sets of chimes would strike every other frame.
//     The floor is what stops a division by anything reaching zero.
//
// The chime half is gated on NumChimes, which DrawLocale counts per locale, so it stops
// as soon as the player leaves the room -- unlike the phone.
func (w *World) HandleTelephone() {
	if !w.PhoneBitSet {
		if w.Phone.NextRing == 0 {
			if w.Phone.Delay == 0 {
				w.Phone.Delay = RingDelay
				w.PlayPrioritySound(PhoneRingSound, PhoneRingPriority)
				w.Phone.Rings--
				if w.Phone.Rings == 0 {
					w.Phone.NextRing = w.RandomInt(RingSpread) + RingBaseDelay
					w.Phone.Rings = w.RandomInt(3) + 3
				}
			} else {
				w.Phone.Delay--
			}
		} else {
			w.Phone.NextRing--
		}
	}

	// The wind chimes, if the room has any.
	if w.R.NumChimes > 0 {
		if w.Chimes.NextRing == 0 {
			if w.RandomInt(2) == 0 {
				w.PlayPrioritySound(Chime1Sound, Chime1Priority)
			} else {
				w.PlayPrioritySound(Chime2Sound, Chime2Priority)
			}

			// NumChimes is an int here and a short in the C; the division is the
			// C's and cannot underflow because the branch has proved it positive.
			delayTime := ChimeDelay / int16(w.R.NumChimes)
			if delayTime < 2 {
				delayTime = 2
			}

			w.Chimes.NextRing = w.RandomInt(delayTime) + 1
		} else {
			w.Chimes.NextRing--
		}
	}
}

// StrikeChime is Play.c:793-796: one line, and it is how the glider plays the chimes.
//
// Setting NextRing to 0 does not make a sound. It makes the *next* HandleTelephone make
// one -- on the same frame, since the play loop calls HandleTelephone after
// HandleInteraction -- and then draw a fresh delay as though the chime had come up
// naturally. So brushing the chimes both strikes them now and resets their idle rhythm,
// and a player who keeps touching them keeps them silent in between.
//
// It has no effect in a room whose NumChimes is 0, which cannot happen: the only caller
// is the kChimeIt arm of the collision dispatcher, and a kChimeIt rect only exists where
// a chimes object was counted.
func (w *World) StrikeChime() {
	w.Chimes.NextRing = 0
}
