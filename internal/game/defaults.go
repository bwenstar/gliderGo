package game

// SetObjectsToDefaults is Play.c:602-704: put every object in every room back to the
// state its author saved, and mark every room unvisited.
//
// This is what "new game" means. The house handle is the only copy of object state
// (see World.H), and playing a house mutates it in place -- collecting a prize clears
// its state byte, flipping a switch flips its target's. So a second game on the same
// loaded house has to be undone from the `initial` byte that sits beside every state
// byte and is never written during play.
//
// The set of types it restores looks incomplete and is not. Five of the nine families
// are absent entirely, and four more have members missing:
//
//	the five flames        -- state is never read; CreateActiveRects passes literal true
//	kSlider                -- likewise
//	kCobweb                -- likewise
//	the four inert appliances -- no rect and no state
//	furniture, switches, clutter, the fifteen plain transports -- no state at all
//
// Every type whose state is read by anything is in the list. That is worth stating
// positively because the omissions look like oversights: the blower case lists eleven
// and stops short of the flames, the prize case lists fourteen and stops short of
// kSlider, the enemy case lists eight and stops short of kCobweb. In all three the
// omitted members are exactly the ones whose hot spots are unconditionally on.
//
// It reads nRooms, the stored field, rather than the numberRooms global. The two agree
// for every shipped house.
func (w *World) SetObjectsToDefaults() {
	for r := range w.H.Rooms {
		rm := &w.H.Rooms[r]

		// Every room forgets it was entered. This is what resets the map window and
		// the "rooms visited" the score is really counting. It is a byte rather than
		// a bool in internal/house because the field is one byte on disk and houses
		// exist with values other than 0 and 1 in it; the codec preserves whatever
		// is there and only this line and the map window ever interpret it.
		rm.Visited = 0

		for i := range rm.Objects {
			obj := &rm.Objects[i]
			switch obj.What {
			// The eleven switchable air sources. state <- initial, at payload
			// offsets 7 and 6.
			case FloorVent, CeilingVent, FloorBlower, CeilingBlower, LeftFan,
				RightFan, SewerGrate, InvisBlower, GrecoVent, SewerBlower, LiftArea:
				obj.Data[offBlowerState] = obj.Data[offBlowerInitial]

			// The fourteen prizes, kSparkle and both grease jars included. state <-
			// initial, at offsets 8 and 9.
			case RedClock, BlueClock, YellowClock, Cuckoo, Paper, Battery, Bands,
				GreaseRt, GreaseLf, Foil, InvisBonus, Star, Sparkle, Helium:
				obj.Data[offBonusState] = obj.Data[offBonusInitial]

			// A deluxe transporter has no initial byte: both halves live in `wide`,
			// the initial state in the high nibble and the current one in the low.
			// So the restore is nibble arithmetic, and it copies the high nibble's
			// whole value down rather than just its low bit --
			//
			//	initState = (wide & 0xF0) >> 4
			//	wide &= 0xF0
			//	wide += initState
			//
			// -- which leaves wide with its high nibble duplicated into the low. For
			// the 0 and 1 the editor writes that gives 0x00 and 0x11, and the state
			// test is `wide & 0x0F != 0`, so both read correctly. A hand-edited house
			// with a high nibble above 1 would also read as on, which is harmless and
			// is why nothing masks it to a single bit.
			case DeluxeTrans:
				initState := (obj.Data[offTransportWide] & 0xF0) >> 4
				obj.Data[offTransportWide] &= 0xF0
				obj.Data[offTransportWide] += initState

			// The eight lights. state <- initial, at offsets 9 and 8.
			case CeilingLight, LightBulb, TableLamp, HipLamp, DecoLamp,
				Flourescent, TrackLight, InvisLight:
				obj.Data[offLightState] = obj.Data[offLightInitial]

			// A stereo does not restore from its own initial byte. It takes the
			// game's music setting, so a stereo starts playing if and only if music
			// is on -- and its `initial` byte is dead data that the editor writes and
			// nothing ever reads.
			//
			// This case has to come before the appliance group below, which would
			// otherwise match it: kStereo is in the appliance family and shares its
			// layout.
			case Stereo:
				obj.Data[offApplianceState] = b2b(w.R.PlayMusicGame)

			// Nine appliances -- kGuitar included, which is the grouping
			// GetObjectState uses and *not* the one SetObjectState uses, where a
			// guitar has its own case. Restoring a guitar's state is harmless and
			// pointless, since its rect is kStrumIt and always on.
			case Shredder, Toaster, MacPlus, Guitar, TV, Coffee, Outlet, VCR,
				Microwave:
				obj.Data[offApplianceState] = obj.Data[offApplianceInitial]

			// The eight movers. kCobweb is not among them.
			case Balloon, CopterLf, CopterRt, DartLf, DartRt, Ball, Drip, Fish:
				obj.Data[offEnemyState] = obj.Data[offEnemyInitial]
			}
		}
	}
}

// The `initial` byte offsets, the counterparts to the state offsets in setstate.go.
// Both members of each pair are named because the restore is a byte copy between two
// offsets in the same ten-byte payload, and a copy between two numbers is exactly the
// kind of line that is unreviewable without them.
//
//	blowerType     initial @ 6, state @ 7   -- initial comes FIRST
//	bonusType      initial @ 9, state @ 8   -- and here it comes SECOND
//	lightType      initial @ 8, state @ 9
//	applianceType  initial @ 8, state @ 9
//	enemyType      initial @ 8, state @ 9
//
// The blower and bonus layouts put the pair in opposite orders, which is the trap:
// blowerType is `initial, state` and bonusType is `state, initial`. Reading the two
// cases side by side in the C, both spelled `data.x.state = data.x.initial`, gives no
// hint that the byte moves down in one and up in the other.
const (
	offBlowerInitial    = 6
	offBonusInitial     = 9
	offLightInitial     = 8
	offApplianceInitial = 8
	offEnemyInitial     = 8
)
