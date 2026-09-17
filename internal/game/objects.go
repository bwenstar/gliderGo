package game

import "github.com/bwenstar/gliderGo/internal/house"

// MasterObject is objDataType (GliderStructs.h): one entry in the object graph.
//
// The graph is flat and rebuilt on every room change. It holds every object of all
// nine local rooms -- up to 216 of them -- with each object's links resolved twice
// over: once into (roomLink, objectLink) house coordinates, and once more into
// localLink, an index back into this same slice. That second resolution is the
// whole point of the table. A switch fires by walking to masterObjects[localLink]
// and reading its roomNum and objectNum, so an off-screen light in a room three
// doors away can be flipped without searching the house.
//
// The seven indices are int16 in the C and stay int16 here; -1 is "none" for all
// four link fields.
type MasterObject struct {
	// RoomNum and ObjectNum are where this object lives in the house.
	RoomNum   int16
	ObjectNum int16

	// RoomLink and ObjectLink are where it points, in house coordinates. Only the
	// six link transports and the eight link switches ever have them; everything
	// else is (-1, -1).
	RoomLink   int16
	ObjectLink int16

	// LocalLink is RoomLink/ObjectLink resolved to an index into the enclosing
	// Master slice, or -1 if the target is not one of the nine local rooms. So a
	// switch wired to a room the player cannot see has a valid RoomLink and a
	// LocalLink of -1, and that is the case HandleSwitches has to write straight
	// through the house for.
	LocalLink int16

	// HotNum is the index into Room.Hot of the hot spot this object created, or -1.
	//
	// Two things about it. Only the central room's objects ever get one, so eight
	// ninths of this table has -1 here. And an object that creates more than one
	// rect keeps only the **last** one: a fan's HotNum is its push column, not its
	// lethal blade box, and a tall candle's is its dissolve box, not its lift or
	// burn rect. Anything that switches a multi-rect object on or off through
	// HotNum therefore only switches the last rect -- which for the fans is the
	// right one by luck, and is why the blades stay lethal on a fan that is off.
	HotNum int16

	// DynaNum is an index into one of **three different tables**, chosen by the
	// object's own type, or -1. The C's name for it -- `dynamicNum` -- names only the
	// commonest of the three and is why this needs saying:
	//
	//	the seventeen registrable types  an index into Dinahs
	//	kGreaseRt / kGreaseLf           a Scene.SavedMaps slot (the slick's backdrop)
	//	the six switch types            the object's own HotNum, i.e. a Room.Hot index
	//
	// Everything else keeps -1. So the fourteen Toggle* read it as a dinah, and
	// TriggerSwitch -- alone among the eight Trigger* -- reads the same field as a hot
	// spot; see trip.go. Nothing disambiguates it at runtime, which is why every reader
	// has to already know which family it is holding.
	//
	// Written back by SetDynaNum as the composition runs, one call per object. See
	// internal/render/locale.go's header for the hook.
	DynaNum int16

	// TheObject is a **copy** of the house's object, taken at compose time. It is a
	// copy and not a pointer in the C too, and the staleness is observable: see
	// SetObjectState, which has to write the same change into as many as three
	// copies and gets the union member wrong in one of them.
	TheObject house.Object
}

// HotObject is hotObject (GliderStructs.h): one entry in the collision table.
type HotObject struct {
	// Bounds is in room-local pixels -- playOrigin is added by the caller, not
	// here. Several are deliberately outside the room: a floor vent's lift column
	// has a negative Top, and a microwave's radiation box runs from y=0 down to the
	// oven's own top edge.
	Bounds Rect

	// Action is one of the 28 hot-spot actions.
	Action int16

	// Who is the index into Room.Master of the object that created this rect. Not
	// the object slot, and not the room -- the master index, which is how the
	// dispatcher gets from a collision back to the object's links.
	Who int16

	// IsOn gates the effect. Note it is *not* always the object's own state: the
	// fans' blade boxes and every flame pass a literal true, so a switched-off fan
	// still cuts the glider in half.
	IsOn bool

	// StillOver is the one-frame edge detector: true while the glider has been
	// overlapping this rect since before it was last checked. The one-shot actions
	// are all shaped `if !stillOver { do it; stillOver = true }`, so pre-setting it
	// SUPPRESSES the action rather than triggering it -- which is exactly what
	// FlagStillOvers exists to do for a glider dropping out of a ceiling duct.
	StillOver bool

	// DoScrutinize selects the tight hit box: SectGlider insets the glider by 5px
	// on all four sides before testing. It is true for exactly five actions --
	// DissolveIt, BounceIt, ShredIt, MicrowaveIt, WebIt -- so hazards need a
	// deeper overlap than rewards and vents do.
	DoScrutinize bool
}

// ---------------------------------------------------------------------------
// Objects.c:89-122 -- IsThisValid
// ---------------------------------------------------------------------------

// IsThisValid is Objects.c:89-122: should this object exist at all right now?
//
// It gates two things -- hot-spot creation and every draw-time registration -- and
// it is almost entirely "yes". Only two answers are ever no: an empty slot, and one
// of twelve consumable prizes whose state byte has been cleared by collecting it.
//
// The list of twelve is worth reading against the fourteen in SetObjectState's
// prize family, because it is not the same list. kGreaseRt, kGreaseLf and kSlider
// share bonusType but are **not** state-gated here: a grease jar that has been
// knocked over is still valid and still composes, now as a slide rect instead of a
// reward. And a consumed kSparkle simply stops registering, which is the only way
// a sparkle ever goes away.
func (w *World) IsThisValid(where int16, who int) bool {
	rm := w.Room(where)
	if rm == nil || w.badIndex(devRoomObject, who, MaxRoomObs) {
		return false
	}
	obj := rm.Objects[who]
	switch obj.What {
	case house.ObjectIsEmpty:
		return false
	case RedClock, BlueClock, YellowClock, Cuckoo, Paper, Battery, Bands,
		Foil, InvisBonus, Star, Sparkle, Helium:
		return obj.Bonus().State != 0
	}
	// No default. Every other kind is unconditionally valid, including the seven
	// undefined `what` codes -- which the original also lets through.
	return true
}

// ---------------------------------------------------------------------------
// Objects.c:126-250 -- the four link resolvers
// ---------------------------------------------------------------------------

// ObjectIsLinkTransport is Objects.c:219-233: is this one of the six transports
// that carries a link?
//
// Six of the sixteen transport types. The other ten -- stairs, doors, windows --
// move the glider by their own geometry and need no destination stored, which is
// why a door has no link and a mailbox does.
func ObjectIsLinkTransport(what int16) bool {
	switch what {
	case MailboxLf, MailboxRt, FloorTrans, CeilingTrans, InvisTrans, DeluxeTrans:
		return true
	}
	return false
}

// ObjectIsLinkSwitch is Objects.c:236-250: is this one of the eight switch kinds
// that carries a link?
//
// Eight of the nine switch types. kSoundTrigger is the exception, and not because
// it has no link: its `where` field is a **sound resource id**, not a room. Treating
// it as a room link would make CreateActiveRects resolve a resource number into a
// floor and suite and wire the trigger to whatever room came out.
func ObjectIsLinkSwitch(what int16) bool {
	switch what {
	case LightSwitch, MachineSwitch, Thermostat, PowerSwitch, KnifeSwitch,
		InvisSwitch, Trigger, LgTrigger:
		return true
	}
	return false
}

// GetRoomLinked is Objects.c:126-174: which room does this object point at?
//
// The two families read the same two bytes through different union members --
// data.d.where for transports, data.e.where for switches -- which alias at payload
// offset 6, so the distinction is documentation rather than arithmetic. It is kept
// because the C keeps it, and because a future divergence in either struct would
// otherwise silently apply to both.
//
// The stored short is a packed floor/suite pair, not a room index, so it goes
// through ExtractFloorSuite and GetRoomNumber. A link to a room the house does not
// contain resolves to -1 there, not here.
func (w *World) GetRoomLinked(who house.Object) int16 {
	var compound int16
	switch {
	case ObjectIsLinkTransport(who.What):
		compound = who.Transport().Where
	case ObjectIsLinkSwitch(who.What):
		compound = who.Switch().Where
	default:
		return -1
	}
	if compound == house.UnlinkedWhere {
		return -1
	}
	floor, suite := w.R.ExtractFloorSuite(compound)
	return w.R.GetRoomNumber(floor, suite)
}

// GetObjectLinked is Objects.c:177-216: which object slot does this object point at?
//
// Note the asymmetry with GetRoomLinked's sentinel. `who` is a Byte, so "unlinked"
// is 255 and not -1; the two fields sit next to each other in the same struct and
// use different conventions (Link.c:333-334). Reading 255 as a signed value, or
// comparing this field against -1, links every unlinked object to slot 255.
func (w *World) GetObjectLinked(who house.Object) int16 {
	var slot byte
	switch {
	case ObjectIsLinkTransport(who.What):
		slot = who.Transport().Who
	case ObjectIsLinkSwitch(who.What):
		slot = who.Switch().Who
	default:
		return -1
	}
	if slot == house.UnlinkedWho {
		return -1
	}
	return int16(slot)
}

// ---------------------------------------------------------------------------
// Objects.c:254-348 -- building the object graph
// ---------------------------------------------------------------------------

// ListOneRoomsObjects is Objects.c:254-296: append one local room's 24 object
// slots to the graph.
//
// Three things it does that read like accidents and are not:
//
// It appends all 24 slots, empty ones included. The graph is indexed by position,
// so skipping an empty slot would shift every object after it and break the
// localLink resolution that ListAllLocalObjects is about to do. An empty slot gets
// an entry with What == ObjectIsEmpty and HotNum == -1.
//
// It creates hot spots only for the central room, and only for valid objects. Every
// other room's HotNum is -1. So the collision table describes the room the glider
// is standing in and nothing else, while the graph describes all nine.
//
// It stops silently at MaxMasterObjects. With 9 rooms x 24 slots = 216 and the cap
// at 216 that is unreachable for a nine-room locale, but ListAllLocalObjects can be
// called with numNeighbors of 1 or 3 as well, and the guard is the C's.
func (w *World) ListOneRoomsObjects(where int) {
	roomNum := w.R.LocalNumbers[where]
	if roomNum == -1 {
		return
	}
	rm := w.Room(roomNum)
	if rm == nil {
		// rooms[roomNum] with roomNum past nRooms. localNumbers only ever holds
		// -1 or a real index, so this cannot fire on a well-formed house; it
		// stands in for the C's unguarded index on a malformed one.
		return
	}

	for n := 0; n < MaxRoomObs; n++ {
		if len(w.R.Master) >= MaxMasterObjects {
			break
		}
		obj := rm.Objects[n]
		m := MasterObject{
			RoomNum:    roomNum,
			ObjectNum:  int16(n),
			RoomLink:   w.GetRoomLinked(obj),
			ObjectLink: w.GetObjectLinked(obj),
			LocalLink:  -1,
			HotNum:     -1,
			DynaNum:    -1,
			TheObject:  obj,
		}
		w.R.Master = append(w.R.Master, m)

		// CreateActiveRects reads masterObjects[who], so the entry has to be in
		// the table before the call -- it takes an index, not an object.
		if where == CentralRoom && w.IsThisValid(roomNum, n) {
			i := len(w.R.Master) - 1
			w.R.Master[i].HotNum = w.CreateActiveRects(int16(i))
		}
		if where == CentralRoom {
			w.R.NumLocalMaster++
		}
	}
}

// ListAllLocalObjects is Objects.c:300-348: build the whole object graph.
//
// The order the nine rooms are listed in is fixed and is **not** the order they are
// drawn in. Listing is central, east, west, then north, north-east, south-east,
// south, south-west, north-west (Objects.c:312-328); drawing is north-west,
// north-east, north, south-west, south-east, south, west, east, central
// (RoomGraphics.c:80-120). Neither is the reverse of the other. The listing order
// is what fixes master indices, and the drawing order is what fixes savedMaps,
// dinahs and grease slot indices, so the two orders have to be kept separately --
// deriving one from the other, in either direction, renumbers half the tables.
//
// The final pass is the one that makes the graph a graph: it walks every entry with
// both a room and an object link and searches the whole table for the entry that
// *is* that (room, object). Two properties of that search matter. It is O(n^2) over
// 216 entries and runs on every room change, which is why the original could not
// afford a larger locale. And it keeps the **last** match rather than breaking on
// the first -- there is no break in the C's inner loop -- which is only harmless
// because (roomNum, objectNum) is unique in the table.
func (w *World) ListAllLocalObjects() {
	w.R.Master = w.R.Master[:0]
	w.R.NumLocalMaster = 0
	w.R.Hot = w.R.Hot[:0]

	w.ListOneRoomsObjects(CentralRoom)

	if w.R.NumNeighbors > 1 {
		w.ListOneRoomsObjects(EastRoom)
		w.ListOneRoomsObjects(WestRoom)
	}

	if w.R.NumNeighbors > 3 {
		w.ListOneRoomsObjects(NorthRoom)
		w.ListOneRoomsObjects(NorthEastRoom)
		w.ListOneRoomsObjects(SouthEastRoom)
		w.ListOneRoomsObjects(SouthRoom)
		w.ListOneRoomsObjects(SouthWestRoom)
		w.ListOneRoomsObjects(NorthWestRoom)
	}

	for i := range w.R.Master {
		if w.R.Master[i].RoomLink == -1 || w.R.Master[i].ObjectLink == -1 {
			continue
		}
		for n := range w.R.Master {
			if w.R.Master[i].RoomLink == w.R.Master[n].RoomNum &&
				w.R.Master[i].ObjectLink == w.R.Master[n].ObjectNum {
				w.R.Master[i].LocalLink = int16(n)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Objects.c:351-362 -- AddTempManholeRect
// ---------------------------------------------------------------------------

// AddTempManholeRect is Objects.c:351-362: remember a manhole so the floor support
// beam drawn over it can be cut away.
//
// The stored rect is not the manhole: its bottom is forced to top + FloorSupportTall
// (44), i.e. exactly the beam's height, because what the caller wants is the hole in
// the beam and not the hole in the floor.
//
// internal/render has its own copy of this, because the beam is drawn there and the
// beam is all the renderer needs it for. This one is the game's, and both write to
// the same table on the embedded Scene.
func (w *World) AddTempManholeRect(manHole Rect) {
	if len(w.R.TempManholes) >= MaxTempManholes {
		return
	}
	r := manHole
	r.Bottom = r.Top + FloorSupportTall
	w.R.TempManholes = append(w.R.TempManholes, r)
}
