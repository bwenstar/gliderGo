package game

// This file is the runtime half of Room.c: the room-identity lookups and the five
// predicates that decide a room's shape.
//
// The other half of Room.c -- NewRoom, DeleteRoom, the numbering and the editor's
// bookkeeping -- is Stage 5's. Nothing here writes a room.
//
// The five predicates all answer from the same two sources and are worth reading
// together, because their *lists differ* and none of the differences is documented
// in the original:
//
//	                        no floor              no ceiling
//	IsShadowVisible    Roof Sky Strat Stars       --
//	DoesRoomHaveFloor       Sky Strat Stars       --
//	DoesRoomHaveCeiling     --                    Garden Meadow Field Roof Sky Strat Stars
//
// So a roof has a floor you cannot fall through and casts no shadow. That is not an
// oversight: the shadow sprite is drawn on an interior floor plane that a roof does
// not have, so drawing it would put a shadow in mid-air, while removing the floor
// would drop the glider out of the room. The two questions look like one and are
// not, which is exactly why the original keeps two functions that differ by one
// label -- and why a port that unified them would either put shadows on roofs or
// make roofs bottomless.

// ForceThisRoom is Room.c:369-386: make a room the central one, without composing
// anything.
//
// It is the *identity* half of a room change: the drawing half is Rebuild. The two
// are separate in the original because RestoreEntireGameScreen re-composes the
// locale without changing which room it is, and MoveRoomToRoom changes which room it
// is several frames before the composition happens.
//
// A roomNumber of -1 is a no-op and not an error -- GetNeighborRoomNumber returns -1
// for a room that is not there, and callers pass that straight in.
func (w *World) ForceThisRoom(roomNumber int16) {
	if roomNumber == -1 {
		return
	}
	// The original copies rooms[roomNumber] into *thisRoom here, or raises
	// kYellowIllegalRoomNum if the index is past nRooms. This port has no separate
	// thisRoom copy (see World.ThisRoom), so an out-of-range index simply leaves
	// ThisRoom returning nil -- which is the same information the alert carried, at
	// the point it matters rather than here.
	w.PrevRoom = w.R.RoomNumber
	w.R.RoomNumber = roomNumber
}

// RoomExists is Room.c:390-421: find the room at a floor and suite.
//
// The linear scan is the original's, and it is the only way round: rooms are stored
// in file order, and (floor, suite) is the address an author wires links to. It runs
// on every link resolution, so a 300-room house pays 300 comparisons per switch --
// which is why ListAllLocalObjects resolves links once per room change into
// LocalLink rather than looking them up per frame.
//
// The `suite < 0` early return is what makes an empty room slot unaddressable:
// kRoomIsEmpty is -1 in the suite field, so a deleted room can never be the target
// of a link even if some other room happens to share its floor.
func (w *World) RoomExists(suite, floor int16) (roomNum int16, ok bool) {
	if suite < 0 {
		return 0, false
	}
	for i := range w.H.Rooms {
		if w.H.Rooms[i].Floor == floor && w.H.Rooms[i].Suite == suite {
			return int16(i), true
		}
	}
	// The C leaves *roomNum untouched on failure, so the caller reads whatever it
	// passed in. Every caller tests the Boolean first; returning 0 is safe and is
	// stated here so nobody relies on it.
	return 0, false
}

// RoomNumExists is Room.c:425-434: is this room index a real room?
//
// It goes the long way round -- index to (floor, suite), then (floor, suite) back to
// an index -- rather than just testing the index against nRooms. That is deliberate
// and catches a case the bounds test would not: a room slot inside the array whose
// suite is kRoomIsEmpty, i.e. a deleted room, which is in range and is not a room.
func (w *World) RoomNumExists(roomNum int16) bool {
	floor, suite, ok := w.GetRoomFloorSuite(roomNum)
	if !ok {
		return false
	}
	_, exists := w.RoomExists(suite, floor)
	return exists
}

// GetRoomFloorSuite is Room.c:711-734: a room's address, and whether it has one.
//
// The false answer sets floor to 0 and suite to kRoomIsEmpty rather than leaving
// them alone, which matters because RoomNumExists feeds them straight into
// RoomExists -- where the negative suite is what makes the answer false.
func (w *World) GetRoomFloorSuite(room int16) (floor, suite int16, ok bool) {
	rm := w.Room(room)
	if rm == nil {
		// rooms[room] unguarded in the C.
		return 0, house_kRoomIsEmpty, false
	}
	if rm.Suite == house_kRoomIsEmpty {
		return 0, house_kRoomIsEmpty, false
	}
	return rm.Floor, rm.Suite, true
}

// house_kRoomIsEmpty is kRoomIsEmpty (GliderDefines.h): the sentinel in a room's
// suite field marking a slot that is in the file and is not a room.
const house_kRoomIsEmpty int16 = -1

// ---------------------------------------------------------------------------
// Room.c:816-933 -- DetermineRoomOpenings
// ---------------------------------------------------------------------------

// DetermineRoomOpenings is Room.c:816-933: work out which of the four walls the
// glider can leave through, and where the two side walls are.
//
// It writes six fields, and the pairing is not symmetric. Left and right come from
// the room's first and last **background tile** -- tile 0 of the eight interior
// tiles is the one with a wall drawn on it -- while top and bottom come from the
// background alone. So an author opens a side wall by choosing a different tile and
// opens the ceiling by choosing a different background.
//
// Each side produces both a threshold and a flag, and the threshold is itself two
// pieces of information: the escape check compares the glider's position against it,
// and the *caller* compares it against LeftWallLimit to ask whether there is a wall
// there at all. That is why the four constants are what they are -- LeftWallLimit 12
// and NoLeftWallLimit -24 -- and why they cannot be replaced by a boolean and a
// single coordinate.
func (w *World) DetermineRoomOpenings() {
	rm := w.ThisRoom()
	if rm == nil {
		return
	}
	whichBack := rm.Background
	leftTile := rm.Tiles[0]
	rightTile := rm.Tiles[NumTiles-1]

	if whichBack >= UserBackground {
		// A house's own background. Version 2.0 houses store the openings in the
		// room; older ones have to be asked of the background art itself, through
		// its 'bnds' resource. Note `bounds >> 1`: bit 0 of the room's bounds field
		// is something else, so the four opening bits start at bit 1 and the shift
		// lines them up with GetOriginalBounding's 1/2/4/8.
		var boundsCode int16
		if rm.Bounds != 0 {
			boundsCode = rm.Bounds >> 1
		} else {
			boundsCode = w.GetOriginalBounding(whichBack)
		}
		w.R.LeftOpen = boundsCode&0x0001 == 0x0001
		w.R.RightOpen = boundsCode&0x0004 == 0x0004

		if w.R.LeftOpen {
			w.R.LeftThresh = NoLeftWallLimit
		} else {
			w.R.LeftThresh = LeftWallLimit
		}
		if w.R.RightOpen {
			w.R.RightThresh = NoRightWallLimit
		} else {
			w.R.RightThresh = RightWallLimit
		}
	} else {
		switch whichBack {
		// The nine interiors, plus kSky. Tile 0 on the left edge means a wall;
		// tile 7 on the right edge means a wall. kSky is in this list rather than
		// with the open backgrounds because its tile set includes a wall tile --
		// a sky room can be the top of a building.
		case SimpleRoom, PaneledRoom, Basement, ChildsRoom, AsianRoom,
			UnfinishedRoom, SwingersRoom, Bathroom, Library, Sky:
			w.R.LeftThresh, w.R.RightThresh = interiorThresholds(leftTile, rightTile, 0)
			w.R.LeftOpen = leftTile != 0
			w.R.RightOpen = rightTile != NumTiles-1

		// kDirt is the same as the interiors except for one digit, and the digit
		// disagrees with the flag beside it: the threshold tests `leftTile == 1`
		// while leftOpen tests `leftTile != 0`.
		//
		// So a dirt room whose left tile is 1 comes out with leftThresh =
		// LeftWallLimit (there is a wall) and leftOpen = true (there is no wall) at
		// the same time. What that does depends on which of the two a given check
		// reads: CheckGliderInRoom uses the threshold and stops the glider, while
		// the escape functions consult leftOpen and will hand it to the next room.
		// The observable result is a dirt room the glider can be pushed out of
		// through a wall it cannot fly through.
		//
		// Two of the eight tiles disagree, in opposite directions. Tile 1 is walled
		// by the threshold and open by the flag, which is the case above. Tile 0 is
		// the mirror: no wall by the threshold, not open by the flag, so the glider
		// is held against an edge that has nothing drawn on it.
		//
		// Reproduced rather than reconciled, and it is not a corner case: 16 of the
		// 22 shipped houses use kDirt, 421 rooms in all, and 336 of those 421 land on
		// tile 0 or tile 1 (TestDirtInconsistencyIsReachableInShippedHouses pins the
		// census). "Fixing" either half moves a wall in eight out of ten dirt rooms
		// in the shipped game.
		case Dirt:
			w.R.LeftThresh, w.R.RightThresh = interiorThresholds(leftTile, rightTile, 1)
			w.R.LeftOpen = leftTile != 0
			w.R.RightOpen = rightTile != NumTiles-1

		// kMeadow's tile set is numbered differently: 6 is its wall tile on the
		// left and 7 on the right, and here the threshold and the flag agree.
		case Meadow:
			if leftTile == 6 {
				w.R.LeftThresh = LeftWallLimit
			} else {
				w.R.LeftThresh = NoLeftWallLimit
			}
			if rightTile == 7 {
				w.R.RightThresh = RightWallLimit
			} else {
				w.R.RightThresh = NoRightWallLimit
			}
			w.R.LeftOpen = leftTile != 6
			w.R.RightOpen = rightTile != 7

		// Five backgrounds with no side walls at all, whatever their tiles.
		case Garden, Skywalk, Field, Stratosphere, Stars:
			w.R.LeftThresh = NoLeftWallLimit
			w.R.RightThresh = NoRightWallLimit
			w.R.LeftOpen = true
			w.R.RightOpen = true

		// An unrecognised background below kUserBackground -- reachable, since the
		// eighteen ids are contiguous but nothing validates the field -- is treated
		// as an interior. Identical to the first case, and kept separate because
		// the original keeps it separate.
		default:
			w.R.LeftThresh, w.R.RightThresh = interiorThresholds(leftTile, rightTile, 0)
			w.R.LeftOpen = leftTile != 0
			w.R.RightOpen = rightTile != NumTiles-1
		}
	}

	// Top and bottom, from the background only. Note the inversion: these two
	// predicates answer "does it have one", and the fields record "is it open".
	w.R.BottomOpen = !w.DoesRoomHaveFloor()
	w.R.TopOpen = !w.DoesRoomHaveCeiling()
}

// interiorThresholds is the tile-comparison shape shared by the interiors, kDirt and
// the default case. wallTile is the left-edge tile that means "walled" -- 0 for the
// interiors and 1 for kDirt, which is the whole difference between those branches.
func interiorThresholds(leftTile, rightTile, wallTile int16) (left, right int16) {
	if leftTile == wallTile {
		left = LeftWallLimit
	} else {
		left = NoLeftWallLimit
	}
	if rightTile == NumTiles-1 {
		right = RightWallLimit
	} else {
		right = NoRightWallLimit
	}
	return left, right
}

// GetOriginalBounding is Room.c:937-966: the opening bits of a house's own
// background, read from its 'bnds' resource.
//
// The resource is eight bytes shaped like a Rect and used as four booleans: any
// non-zero side is an open side. The packing is left=1, top=2, right=4, bottom=8,
// which is the same order as the room's own bounds field once that has been shifted
// down by one -- so the two sources are interchangeable, which is the point.
//
// A missing resource gives 0: closed on all four sides. That is the safe direction
// and is what the original does; it also raises a yellow alert, but only when a PICT
// of the same id *does* exist, i.e. only when an author drew a background and never
// saved its openings. The alert is advisory and not reproduced.
func (w *World) GetOriginalBounding(theID int16) int16 {
	if w.R.A == nil {
		return 0
	}
	r, ok := w.R.A.Bnds(theID)
	if !ok {
		return 0
	}
	code := int16(0)
	if r.Left != 0 {
		code += 1
	}
	if r.Top != 0 {
		code += 2
	}
	if r.Right != 0 {
		code += 4
	}
	if r.Bottom != 0 {
		code += 8
	}
	return code
}

// ---------------------------------------------------------------------------
// Room.c:1103-1215 -- the three shape predicates
// ---------------------------------------------------------------------------

// IsShadowVisible is Room.c:1103-1133: should the glider's shadow be drawn on the
// floor of this room?
//
// It is DoesRoomHaveFloor with kRoof added to the no-floor list, and that one label
// is the entire difference between the two functions. See the table at the top of
// this file for why they have to differ.
//
// This is the *predicate*. Room.ShadowVisible is the cached flag, recomputed only by
// Rebuild and by RedrawRoomLighting -- because turning a light off changes what a
// shadow would look like, and that is the only thing during play that can. Reading
// the predicate where the cache is meant, or the reverse, is the mistake this pair
// invites; the note on Room.ShadowVisible says which is which.
func (w *World) IsShadowVisible() bool {
	rm := w.ThisRoom()
	if rm == nil {
		return true
	}
	if rm.Background >= UserBackground {
		return w.boundsCode(rm.Background, rm.Bounds)&0x0008 != 0x0008
	}
	switch rm.Background {
	case Roof, Sky, Stratosphere, Stars:
		return false
	}
	return true
}

// DoesRoomHaveFloor is Room.c:1138-1168: can the glider fall out of the bottom?
//
// Three backgrounds have no floor, and they are the three that are nothing but air.
// kRoof is not among them: a roof has a floor. kBasement and kGarden are not among
// them either, which is worth stating because neither is a *structure* -- see
// IsRoomAStructure, whose list is different again. "Has a floor" and "is a
// structure" are three overlapping questions asked by three different functions,
// and no two of the three lists match.
func (w *World) DoesRoomHaveFloor() bool {
	rm := w.ThisRoom()
	if rm == nil {
		return true
	}
	if rm.Background >= UserBackground {
		return w.boundsCode(rm.Background, rm.Bounds)&0x0008 != 0x0008
	}
	switch rm.Background {
	case Sky, Stratosphere, Stars:
		return false
	}
	return true
}

// DoesRoomHaveCeiling is Room.c:1172-1215: can the glider leave through the top?
//
// Seven backgrounds have no ceiling -- the four with no floor plus the three ground
// levels, kGarden, kMeadow and kField. So a garden is open at the top and closed at
// the bottom, and a sky room is open at both.
//
// Bit 2 rather than bit 4: the room's bounds field is shifted right by one before the
// four opening bits are read, so top is 2 here and in GetOriginalBounding.
func (w *World) DoesRoomHaveCeiling() bool {
	rm := w.ThisRoom()
	if rm == nil {
		return true
	}
	if rm.Background >= UserBackground {
		return w.boundsCode(rm.Background, rm.Bounds)&0x0002 != 0x0002
	}
	switch rm.Background {
	case Garden, Meadow, Field, Roof, Sky, Stratosphere, Stars:
		return false
	}
	return true
}

// boundsCode is the "is this a version 2.0 house?" test the three predicates and
// DetermineRoomOpenings all open with: use the room's own bounds field if it has one,
// otherwise fall back to the background's 'bnds' resource.
//
// A bounds of 0 is indistinguishable from "closed on all four sides and not a
// structure", so a version 2.0 room that really is closed all round takes the
// fallback path too. Harmless -- GetOriginalBounding on a fully closed background
// returns 0 as well -- but it means the field cannot be used to tell house versions
// apart, despite the original's comment calling it exactly that.
func (w *World) boundsCode(background, bounds int16) int16 {
	if bounds != 0 {
		return bounds >> 1
	}
	return w.GetOriginalBounding(background)
}
