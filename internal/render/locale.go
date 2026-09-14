package render

import (
	"fmt"
	"time"

	"glidergo/internal/house"
)

// Composing a room: GliderPRO/Sources/RoomGraphics.c plus the parts of Room.c,
// Objects.c, Link.c, DynamicMaps.c and Grease.c that the composition reads.
//
// DrawLocale is the whole of it. It paints backSrcMap black, then walks the nine
// local rooms in a fixed order -- the six diagonal and vertical neighbours first,
// then west and east, then the central room last -- drawing each one's background
// tiles and then its objects. The order is the whole trick: an object that hangs
// over a room boundary is drawn by both rooms, and the later room wins, so the
// central room's own art is never clipped by a neighbour's.
//
// Three things in here are state rather than pixels, and they are modelled
// because they are visible:
//
//   - numLights. It is recomputed per room before that room is drawn, and it
//     gates almost every object draw. An unlit room is a black rectangle with a
//     handful of always-drawn objects (doors, windows, appliance indicators)
//     floating in it.
//   - the savedMaps table, capped at 24. Clocks, prizes, flames, pendulums,
//     stars and grease each stash the background under themselves so they can be
//     erased later, and every one of them declines to draw when the table is
//     full. Which objects get slots depends on object index order, so the cap has
//     to be modelled to get the right pixels, not merely the right bookkeeping.
//   - tempManholes, capped at 8. A manhole records its rect during the object
//     pass, and DrawFloorSupport then punches PICT 3957 through the beam wherever
//     one lines up.
//
// What is deliberately not here: the flames, pendulums, stars, sparkles and other
// animated objects are registered (so that the savedMaps accounting is right and
// so Stage 1.5 has the tables) but not drawn. None of them contributes a pixel to
// the static background in the original either -- they are composited per frame
// into workSrcMap, over the top of what this file produces.
//
// The one genuine simplification is masterObjects. DrawARoomsObjects sets
// `dynamicNum = masterObjects[i].hotNum` for the six switch kinds, and at the end
// of each object copies dynamicNum into every master-object entry pointing at
// that object. Nothing in that chain affects a pixel -- it is the switch-to-object
// link table, which Stage 1.5 needs and this stage does not -- so it is left out
// and noted at the two places it would appear.

// The eighteen built-in room backgrounds (GliderDefines.h:227-244). kDirt,
// kMeadow and kSky are declared in view.go, where DrawRoomBackground's
// off-the-map fallbacks need them.
const (
	kSimpleRoom     = 2000
	kPaneledRoom    = 2001
	kBasement       = 2002
	kChildsRoom     = 2003
	kAsianRoom      = 2004
	kUnfinishedRoom = 2005
	kSwingersRoom   = 2006
	kBathroom       = 2007
	kLibrary        = 2008
	kGarden         = 2009
	kSkywalk        = 2010
	kField          = 2013
	kRoof           = 2014
	kStratosphere   = 2016
	kStars          = 2017
)

// Where a house's own resources begin, and where IsRoomAStructure stops assuming
// a custom background is an interior (GliderDefines.h:245-246).
const (
	kUserBackground     = 3000
	kUserStructureRange = 3300
)

// kNumUndergroundFloors is the bias in a packed floor/suite link value: floor 0
// in a house file is eight storeys below ground (GliderDefines.h:508).
const kNumUndergroundFloors = 8

// The five flame-like animation tables and the dynamic-object table, with the
// caps from GliderDefines.h:255-265. Every entry in the first five also holds a
// savedMaps slot, so these caps and kMaxSavedMaps interact.
const (
	kMaxCandles        = 20
	kMaxTikis          = 8
	kMaxCoals          = 8
	kMaxPendulums      = 8
	kMaxStars          = 4
	kMaxDynamicObs     = 18
	kNumCandleFrames   = 5
	kNumTikiFrames     = 5
	kNumCoalFrames     = 4
	kNumPendulumFrames = 3
	kNumStarFrames     = 6
)

// SavedMap is one entry in the savedMaps table (DynamicMaps.c:70-93): a patch of
// backSrcMap copied aside before something animated was drawn over it, so the
// animation can restore the background each frame.
type SavedMap struct {
	// Map is the stashed pixels. Its size is the *bounds* the caller asked for,
	// which for a flame or a star is one frame wide and every frame tall -- the
	// slot doubles as scratch for the frame strip.
	Map *Surface

	// Where and Who identify the object, so RestoreFromSavedMap can find it.
	Where int16
	Who   int16
}

// Anim is one entry in one of the five flame-like tables. Only the fields the
// composition determines are here; the frame counters and phases are seeded from
// RandomInt and belong to the animation stage.
type Anim struct {
	Dest     Rect // where on screen the frames are blitted
	SavedMap int  // the savedMaps slot holding the background under Dest
	Where    int16
	Who      int16
}

// Dynamic is one entry in the dinahs table (Dynamics3.c:187): an object that
// moves or blinks. Registered here, animated in Stage 1.5.
type Dynamic struct {
	What int16
	Rect Rect // room-local, playOrigin already subtracted
	Room int16
	Obj  int16
	On   bool
}

// Scene is the room composition: the geometry, the art, the house, and the
// offscreen surfaces the original calls backSrcMap and workSrcMap.
//
// In the original every one of these is a global. Grouping them costs nothing and
// buys two things: a composition can be produced with no display attached, which
// is what the golden test does, and a second view becomes possible later without
// unpicking global state.
type Scene struct {
	V *View
	A *Assets
	H *house.House

	// RoomNumber is thisRoomNumber: the room the player is in, and the one drawn
	// into the central slot.
	RoomNumber int16

	// NumNeighbors is 1, 3 or 9 (Settings.c:888 defaults it to 9). It decides how
	// much of the surrounding house is composed: 1 is the central room alone, 3
	// adds west and east, 9 adds the six diagonals and verticals. On the 640x480
	// screen the game was built for it stays 9, so slivers of all eight
	// neighbours are visible around the edges of the play area.
	NumNeighbors int

	// WardBitSet is the house's flag bit 0 (HouseIO.c:416). It makes every
	// off-the-map room black instead of dirt, meadow or sky.
	WardBitSet bool

	// PlayMusicGame is isPlayMusicGame, which is what a stereo's on state comes
	// from -- not the object's own data.
	PlayMusicGame bool

	// Clock is what the three clock faces and the calendar read. It is a field
	// rather than a call to time.Now so a composition is reproducible.
	Clock time.Time

	// Back is backSrcMap, the static room. Work is workSrcMap, which serves two
	// unrelated purposes: it is scratch for decoding a background PICT before the
	// tiles are blitted out of it, and it is the per-frame composite. DrawLocale
	// ends by copying Back over it.
	Back, Work *Surface

	// LocalNumbers and IsStructure are the nine local rooms and whether each is
	// an interior -- the latter decides where floor-support beams go.
	LocalNumbers [9]int16
	IsStructure  [9]bool

	// NumLights is the light count of the room currently being drawn. It is
	// recomputed before each of the nine and gates most object draws.
	NumLights int

	// ThisBackground and ThisTiles are the central room's, cached by
	// DrawRoomBackground for the systems that ask what kind of room this is.
	ThisBackground int16
	ThisTiles      [kNumTiles]int16

	// The dynamic tables. Populated by the composition, drawn by Stage 1.5.
	SavedMaps    []SavedMap
	Flames       []Anim
	TikiFlames   []Anim
	Coals        []Anim
	Pendulums    []Anim
	Stars        []Anim
	Dynamics     []Dynamic
	TempManholes []Rect
	MirrorRects  []Rect

	// numGrease is the count Grease.c keeps beside its own array, which is Stage
	// 1.5's; only the cap is visible here.
	numGrease int

	// ListLocalObjects is ListAllLocalObjects (Objects.c:300-348), called from the
	// middle of DrawLocale. It is a hook because the object graph belongs to
	// internal/game and the composition belongs here, and the two genuinely
	// interleave: DrawLocale builds the graph *before* it draws, because
	// DrawARoomsObjects writes each object's dynaNum back into it
	// (ObjectDrawAll.c:952-960).
	//
	// So this cannot be hoisted to either side of the composition. nil is fine and
	// is what the renderer's own tests use -- nothing the graph holds reaches a
	// pixel, which is why internal/render could be finished without it.
	ListLocalObjects func()
}

// NewScene sets up a composition. Nothing is drawn until DrawLocale.
func NewScene(v *View, a *Assets, h *house.House) *Scene {
	return &Scene{
		V:            v,
		A:            a,
		H:            h,
		RoomNumber:   h.FirstRoom,
		NumNeighbors: 9,
		WardBitSet:   h.Flags&house.FlagWard != 0,
		Back:         NewSurface(int(v.BackRect.Wide()), int(v.BackRect.Tall())),
		Work:         NewSurface(int(v.WorkRect.Wide()), int(v.WorkRect.Tall())),
	}
}

// ---------------------------------------------------------------------------
// DrawLocale
// ---------------------------------------------------------------------------

// DrawLocale is RoomGraphics.c:43-129: compose the room the player is in,
// together with as much of its surroundings as NumNeighbors asks for.
func (s *Scene) DrawLocale() {
	// ZeroFlamesAndTheLike, ZeroDinahs, KillAllBands, ZeroMirrorRegion,
	// ZeroTriggers, numTempManholes = 0. The trigger and band tables belong to
	// later stages; the rest are here.
	s.SavedMaps = s.SavedMaps[:0]
	s.Flames = s.Flames[:0]
	s.TikiFlames = s.TikiFlames[:0]
	s.Coals = s.Coals[:0]
	s.Pendulums = s.Pendulums[:0]
	s.Stars = s.Stars[:0]
	s.Dynamics = s.Dynamics[:0]
	s.TempManholes = s.TempManholes[:0]
	s.MirrorRects = s.MirrorRects[:0]
	s.numGrease = 0

	roomV := int16(0)
	if rm := s.room(s.RoomNumber); rm != nil {
		roomV = rm.Floor
	}

	for i := 0; i < 9; i++ {
		s.LocalNumbers[i] = s.GetNeighborRoomNumber(i)
		s.IsStructure[i] = s.IsRoomAStructure(s.LocalNumbers[i])
	}
	// ListAllLocalObjects() builds the masterObjects table here -- before the
	// drawing, because DrawARoomsObjects writes dynaNum back into it. Nothing it
	// produces reaches a pixel, so a nil hook composes the same image.
	if s.ListLocalObjects != nil {
		s.ListLocalObjects()
	}

	s.Back.Fill(s.V.BackRect, Black8)

	// The far six, drawn first and one elevation out. Note the pairing: each
	// room's light count is computed, then its background, then its objects,
	// before the next room is touched -- numLights is live state, not a lookup.
	if s.NumNeighbors > 3 {
		for _, r := range []struct {
			slot int
			elev int16
		}{
			{kNorthWestRoom, roomV + 1},
			{kNorthEastRoom, roomV + 1},
			{kNorthRoom, roomV + 1},
			{kSouthWestRoom, roomV - 1},
			{kSouthEastRoom, roomV - 1},
			{kSouthRoom, roomV - 1},
		} {
			s.NumLights = s.GetNumberOfLights(s.LocalNumbers[r.slot])
			s.DrawRoomBackground(s.LocalNumbers[r.slot], r.slot, r.elev)
			s.DrawARoomsObjects(r.slot, false)
		}
	}

	if s.NumNeighbors > 1 {
		for _, slot := range []int{kWestRoom, kEastRoom} {
			s.NumLights = s.GetNumberOfLights(s.LocalNumbers[slot])
			s.DrawRoomBackground(s.LocalNumbers[slot], slot, roomV)
			s.DrawARoomsObjects(slot, false)
			s.DrawLighting()
		}
	}

	s.NumLights = s.GetNumberOfLights(s.LocalNumbers[kCentralRoom])
	s.DrawRoomBackground(s.LocalNumbers[kCentralRoom], kCentralRoom, roomV)
	s.DrawARoomsObjects(kCentralRoom, false)
	s.DrawLighting()

	if s.NumNeighbors > 3 {
		s.DrawFloorSupport()
	}
	s.RestoreWorkMap()

	// shadowVisible = IsShadowVisible(); takingTheStairs = false. Both are
	// gameplay state read by the glider, not by the composition.
}

// DrawLighting is RoomGraphics.c:421-430 in full:
//
//	if (numLights == 0)
//		return;
//	else
//	{
//		// for future construction
//	}
//
// A stub in the shipped 1.1.2. There is no lighting effect to port -- the only
// thing numLights does is gate the object draws and blacken unlit rooms.
func (s *Scene) DrawLighting() {}

// ---------------------------------------------------------------------------
// Loading a picture
// ---------------------------------------------------------------------------

// LoadGraphicSpecial is RoomGraphics.c:134-152: draw a picture into the current
// port at the picture's own size, cornered at the origin, falling back to PICT
// 2000 if it is missing.
//
// "Special" is about the fallback, not about the drawing. The plain LoadGraphic
// RedAlerts on a missing resource; this one substitutes the simple room, which is
// what lets a house reference a background it does not carry.
//
// The current port when this is called is always workSrcMap -- DrawRoomBackground
// sets it immediately before -- so the destination is Work.
func (s *Scene) LoadGraphicSpecial(resID int16) {
	art := s.A.Background(resID)
	if art == nil {
		return
	}
	s.Work.Copy(art, art.Bounds(), SetRect(0, 0, int16(art.W), int16(art.H)), SrcCopy)
}

// LoadScaledGraphic is Utilities.c:340-349: draw a picture stretched to fill a
// rect. The static room path uses it once, for the manhole seen through a floor
// support, and there the rect is already the picture's size.
//
// Its current port is backSrcMap, set by DrawFloorSupport.
func (s *Scene) LoadScaledGraphic(resID int16, theRect Rect) {
	art := s.A.Pict(resID)
	if art == nil {
		s.A.fail(fmt.Errorf("render: PICT %d is missing", resID))
		return
	}
	s.Back.Copy(art, art.Bounds(), theRect, SrcCopy)
}

// ---------------------------------------------------------------------------
// Backgrounds
// ---------------------------------------------------------------------------

// DrawRoomBackground is RoomGraphics.c:162-253: eight 64x322 tiles cut out of one
// 512x322 picture.
//
// The tiles are indices into that picture, so a room's eight tile values are what
// make two rooms with the same background look different -- a wall panel repeated,
// a doorway moved, a window where the last room had shelving.
//
// It leaves the current port set to workSrcMap and never restores it. That is a
// bug in the original with two visible consequences; see the notes in
// DrawCabinet and DrawCounter.
func (s *Scene) DrawRoomBackground(who int16, where int, elevation int16) {
	if where == kCentralRoom {
		if rm := s.room(who); rm != nil {
			s.ThisBackground = rm.Background
			s.ThisTiles = rm.Tiles
		}
	}

	// An unlit real room is simply black. This branch saves and restores the
	// port, which is why an unlit room does not leave the port on workSrcMap.
	if s.NumLights == 0 && who != kRoomIsEmpty {
		s.Back.Fill(s.V.LocalRoomsDest[where], Black8)
		return
	}

	var pictID int16
	var tiles [kNumTiles]int16

	if who == kRoomIsEmpty {
		// "This call should be smarter than this" -- the original's comment.
		if s.WardBitSet {
			s.Back.Fill(s.V.LocalRoomsDest[where], Black8)
			return
		}
		switch {
		case elevation > 1:
			pictID = kSky
			for i := range tiles {
				tiles[i] = 2
			}
		case elevation == 1:
			pictID = kMeadow
		default:
			pictID = kDirt
		}
	} else {
		rm := s.room(who)
		pictID = rm.Background
		tiles = rm.Tiles
	}

	// SetPort((GrafPtr)workSrcMap), with no restore. See the doc comment.
	s.LoadGraphicSpecial(pictID)

	src := SetRect(0, 0, kTileWide, kTileHigh)
	dest := Offset(SetRect(0, 0, kTileWide, kTileHigh),
		s.V.LocalRoomsDest[where].Left, s.V.LocalRoomsDest[where].Top)
	for i := 0; i < kNumTiles; i++ {
		src.Left = tiles[i] * kTileWide
		src.Right = src.Left + kTileWide
		s.Back.Copy(s.Work, src, dest, SrcCopy)
		dest = Offset(dest, kTileWide, 0)
	}
}

// DrawFloorSupport is RoomGraphics.c:257-376: the wooden beams between storeys.
//
// Six bands, each drawn only if the room it belongs to is an interior -- there is
// no beam under a garden or above the sky. They go in the 44-pixel gap between
// one room's bottom and the next room's top, which is why kVertLocalOffset is
// exactly kTileHigh: the rooms abut and the beam fills the seam.
//
// Wherever a beam crosses a manhole recorded during the object pass, PICT 3957 is
// drawn over it -- the hole seen from inside the floor. tempManholes is mutated
// in the process: the entry's top and bottom are snapped to the beam's, so a
// manhole crossing two beams is drawn once at each.
func (s *Scene) DrawFloorSupport() {
	src := s.V.SuppRect
	supp := s.A.Strip("supp")
	if supp == nil {
		return
	}
	central := s.V.LocalRoomsDest[kCentralRoom]

	band := func(structure bool, left, top int16) {
		if !structure {
			return
		}
		dest := Offset(s.V.SuppRect, left, top)
		s.Back.Copy(supp, src, dest, SrcCopy)

		for i := range s.TempManholes {
			if _, ok := Sect(dest, s.TempManholes[i]); !ok {
				continue
			}
			s.TempManholes[i].Top = dest.Top
			s.TempManholes[i].Bottom = dest.Bottom
			s.LoadScaledGraphic(kManholeThruFloor, s.TempManholes[i])
		}
	}

	// The pairing is not symmetric with the room slots, and deliberately so: a
	// beam above the west room belongs to the *north-west* room's floor, so it is
	// drawn at the west room's left edge but gated on the north-west room.
	band(s.IsStructure[kNorthWestRoom], s.V.LocalRoomsDest[kWestRoom].Left, central.Top-s.V.SuppRect.Bottom)
	band(s.IsStructure[kWestRoom], s.V.LocalRoomsDest[kWestRoom].Left, central.Bottom)
	band(s.IsStructure[kNorthRoom], central.Left, central.Top-s.V.SuppRect.Bottom)
	band(s.IsStructure[kCentralRoom], central.Left, central.Bottom)
	band(s.IsStructure[kNorthEastRoom], s.V.LocalRoomsDest[kEastRoom].Left, central.Top-s.V.SuppRect.Bottom)
	band(s.IsStructure[kEastRoom], s.V.LocalRoomsDest[kEastRoom].Left, central.Bottom)
}

// ReadyBackMap is RoomGraphics.c:380-386: promote the frame just composed in
// workSrcMap to be the new static background.
func (s *Scene) ReadyBackMap() { s.Back.CopyFull(s.Work) }

// RestoreWorkMap is RoomGraphics.c:390-398: reset the frame buffer to the static
// background, undoing whatever the last frame drew into it.
func (s *Scene) RestoreWorkMap() { s.Work.CopyFull(s.Back) }

// ---------------------------------------------------------------------------
// Objects
// ---------------------------------------------------------------------------

// DrawARoomsObjects is ObjectDrawAll.c:23-966: draw all 24 object slots of one
// local room into backSrcMap.
//
// The 117 object kinds fall into a handful of patterns, and which pattern applies
// is not guessable from what the object is:
//
//   - most are gated on both an intersection with the play area and on isLit;
//   - doors, windows, stairs, mailboxes, wall windows, transports and switches
//     are drawn whether or not the room is lit, because a dark room still has to
//     show you the way out;
//   - appliances with an indicator draw the indicator unlit and the body only
//     when lit, so a dark room shows floating screens and LEDs;
//   - eight of them -- the table, shelf, dresser, deck table, stool, tiki and
//     both mailboxes -- skip the intersection test entirely, so they draw even
//     when wholly outside the play area. That is what lets a table's legs run
//     down past the bottom of its own room.
//
// redraw is false for a fresh composition and true when a room is recomposed in
// place; the difference is that the second time round the animated objects
// re-stash their backgrounds instead of claiming new slots. Only redraw == false
// is exercised by this stage.
func (s *Scene) DrawARoomsObjects(neighbor int, redraw bool) {
	if s.LocalNumbers[neighbor] == kRoomIsEmpty {
		return
	}
	rm := s.room(s.LocalNumbers[neighbor])
	if rm == nil {
		return
	}
	room := s.LocalNumbers[neighbor]

	testRect := ZeroCorner(s.V.House)
	isLit := s.NumLights > 0

	for i := 0; i < kMaxRoomObs; i++ {
		// dynamicNum and legit are reset per object in the original. legit is
		// used here; dynamicNum only ever feeds the master-object link table, so
		// it is left out (see the file header).
		legit := -1

		if !s.IsThisValid(room, i) {
			continue
		}

		// A local copy, as the original's is. It matters for kCustomPict:
		// GetObjectRect rewrites a missing picture's id to 10000 and the draw
		// below reads the rewritten value, but the house file never sees it.
		thisObject := rm.Objects[i]

		var itsRect Rect
		rect := func() Rect {
			s.A.GetObjectRect(&thisObject, &itsRect)
			itsRect = s.V.OffsetRectRoomRelative(itsRect, neighbor)
			return itsRect
		}
		visible := func() bool {
			_, ok := Sect(rect(), testRect)
			return ok
		}

		switch thisObject.What {
		case house.ObjectIsEmpty:

		// --- blowers -------------------------------------------------------
		case kFloorVent, kCeilingVent, kFloorBlower, kCeilingBlower, kSewerGrate,
			kLeftFan, kRightFan, kGrecoVent, kSewerBlower:
			if visible() && isLit {
				s.DrawSimpleBlowers(thisObject.What, itsRect)
			}

		case kTaper:
			if visible() {
				if isLit {
					s.DrawSimpleBlowers(thisObject.What, itsRect)
				}
				s.candleFlame(neighbor, i, itsRect, 10, 7, redraw)
			}

		case kCandle:
			if visible() {
				if isLit {
					s.DrawSimpleBlowers(thisObject.What, itsRect)
				}
				s.candleFlame(neighbor, i, itsRect, 14, 7, redraw)
			}

		case kStubby:
			if visible() {
				if isLit {
					s.DrawSimpleBlowers(thisObject.What, itsRect)
				}
				s.candleFlame(neighbor, i, itsRect, 9, 7, redraw)
			}

		case kTiki:
			// No intersection test: a tiki pole runs down to kTikiPoleBase
			// wherever the torch itself is.
			rect()
			if isLit {
				s.DrawTiki(itsRect, s.V.OriginV+VerticalRoomOffset(neighbor))
			}
			if !redraw {
				s.addTikiFlame(room, int16(i), itsRect.Left+10, itsRect.Top-9)
			}

		case kBBQ:
			if visible() {
				if isLit {
					s.DrawPictSansWhiteObject(thisObject.What, itsRect)
				}
				if !redraw {
					s.addBBQCoals(room, int16(i), itsRect.Left+16, itsRect.Top+9)
				}
			}

		case kInvisBlower, kLiftArea:

		// --- furniture -----------------------------------------------------
		case kTable:
			// playOriginV alone, without VerticalRoomOffset. A neighbouring
			// room's table therefore drops its legs to the central room's floor
			// rather than its own -- a defect in the original, reproduced.
			rect()
			if isLit {
				s.DrawTable(itsRect, s.V.OriginV)
			}

		case kShelf:
			rect()
			if isLit {
				s.DrawShelf(itsRect)
			}

		case kCabinet:
			if visible() && isLit {
				s.DrawCabinet(itsRect)
			}

		case kFilingCabinet, kOzma:
			if visible() && isLit {
				s.DrawPictObject(thisObject.What, itsRect)
			}

		case kWasteBasket, kMilkCrate:
			if visible() && isLit {
				s.DrawSimpleFurniture(thisObject.What, itsRect)
			}

		case kCounter:
			if visible() && isLit {
				s.DrawCounter(itsRect)
			}

		case kDresser:
			rect()
			if isLit {
				s.DrawDresser(itsRect)
			}

		case kDeckTable:
			// The same bare playOriginV as kTable.
			rect()
			if isLit {
				s.DrawDeckTable(itsRect, s.V.OriginV)
			}

		case kStool:
			rect()
			if isLit {
				s.DrawStool(itsRect, s.V.OriginV+VerticalRoomOffset(neighbor))
			}

		case kInvisObstacle, kInvisBounce:

		case kManhole:
			if visible() {
				s.AddTempManholeRect(itsRect)
				if isLit {
					s.DrawPictSansWhiteObject(thisObject.What, itsRect)
				}
			}

		// --- prizes --------------------------------------------------------
		case kRedClock, kBlueClock, kYellowClock:
			if visible() {
				legit = s.backUpToSavedMap(itsRect, room, int16(i), redraw)
				if legit != -1 {
					switch thisObject.What {
					case kRedClock:
						s.DrawRedClock(itsRect)
					case kBlueClock:
						s.DrawBlueClock(itsRect)
					default:
						s.DrawYellowClock(itsRect)
					}
				}
			}

		case kCuckoo:
			if visible() {
				legit = s.backUpToSavedMap(itsRect, room, int16(i), redraw)
				if legit != -1 {
					s.DrawCuckoo(itsRect)
					if !redraw {
						s.addPendulum(room, int16(i), itsRect.Left+4, itsRect.Top+46)
					}
				}
			}

		case kPaper, kBattery, kBands, kHelium:
			if visible() {
				legit = s.backUpToSavedMap(itsRect, room, int16(i), redraw)
				if legit != -1 {
					s.DrawSimplePrizes(thisObject.What, itsRect)
				}
			}

		case kGreaseRt, kGreaseLf:
			c := thisObject.Bonus()
			rect()
			if c.State != 0 {
				// Standing. Capped twice over: AddGrease has its own limit of 16
				// and also takes a savedMaps slot.
				if _, ok := Sect(itsRect, testRect); ok {
					if s.addGrease(room, int16(i), redraw) != -1 {
						s.drawGrease(thisObject.What, itsRect, c.Length, true)
					}
				}
			} else {
				// Fallen grease is part of the static background: no intersection
				// test, no saved map, no cap.
				s.drawGrease(thisObject.What, itsRect, c.Length, false)
			}

		case kFoil:
			if visible() {
				legit = s.backUpToSavedMap(itsRect, room, int16(i), redraw)
				if legit != -1 {
					s.DrawFoil(itsRect)
				}
			}

		case kInvisBonus, kSlider:

		case kStar:
			if visible() {
				legit = s.backUpToSavedMap(itsRect, room, int16(i), redraw)
				if legit != -1 {
					// A star costs two savedMaps slots: one here and one in
					// AddStar for its own six-frame strip.
					if !redraw {
						s.addStar(room, int16(i), itsRect.Left, itsRect.Top)
					}
					s.DrawSimplePrizes(thisObject.What, itsRect)
				}
			}

		case kSparkle:
			// Dynamic only: a sparkle contributes nothing to the static room.
			if visible() && !redraw && neighbor == kCentralRoom {
				s.addDynamicObject(kSparkle, itsRect, room, int16(i), thisObject.Bonus().State != 0)
			}

		// --- transports ----------------------------------------------------
		case kUpStairs, kDoorInLf, kDoorInRt, kWindowInLf, kWindowInRt:
			// No isLit gate: the way out of a dark room is always visible.
			if visible() {
				s.DrawPictSansWhiteObject(thisObject.What, itsRect)
			}

		case kDownStairs, kDoorExRt, kDoorExLf, kWindowExRt, kWindowExLf:
			if visible() {
				s.DrawPictObject(thisObject.What, itsRect)
			}

		case kMailboxLf:
			// Neither test nor gate: a mailbox's post has to reach the ground.
			rect()
			s.DrawMailboxLeft(itsRect, s.V.OriginV+VerticalRoomOffset(neighbor))

		case kMailboxRt:
			rect()
			s.DrawMailboxRight(itsRect, s.V.OriginV+VerticalRoomOffset(neighbor))

		case kFloorTrans, kCeilingTrans:
			if visible() {
				s.DrawSimpleTransport(thisObject.What, itsRect)
			}

		case kInvisTrans, kDeluxeTrans:

		// --- switches ------------------------------------------------------
		case kLightSwitch, kMachineSwitch, kThermostat, kPowerSwitch, kKnifeSwitch:
			// A switch is drawn showing the state of the object it controls, so
			// this is the one draw that reads another room's data.
			if visible() {
				e := thisObject.Switch()
				floor, suite := s.ExtractFloorSuite(e.Where)
				s.DrawSwitch(thisObject.What, itsRect,
					s.GetObjectState(s.GetRoomNumber(floor, suite), int16(e.Who)))
			}
			// dynamicNum = masterObjects[i].hotNum here.

		case kInvisSwitch:
			// dynamicNum = masterObjects[i].hotNum, and nothing else.

		case kTrigger, kLgTrigger, kSoundTrigger:

		// --- lights --------------------------------------------------------
		case kCeilingLight, kLightBulb, kTableLamp:
			if visible() && isLit {
				s.DrawSimpleLight(thisObject.What, itsRect)
			}

		case kTrunk, kBooks, kHipLamp, kDecoLamp, kGuitar, kCinderBlock,
			kFlowerBox, kFireplace, kBear, kVase1, kVase2, kRug, kChimes:
			if visible() && isLit {
				s.DrawPictSansWhiteObject(thisObject.What, itsRect)
			}

		case kCustomPict:
			// data.g.height, after GetObjectRect may have rewritten it to 10000.
			if visible() && isLit {
				s.DrawCustPictSansWhite(thisObject.Appliance().Height, itsRect)
			}

		case kFlourescent:
			if visible() && isLit {
				s.DrawFlourescent(itsRect)
			}

		case kTrackLight:
			if visible() && isLit {
				s.DrawTrackLight(itsRect)
			}

		case kInvisLight:

		// --- appliances ----------------------------------------------------
		case kShredder, kCDs:
			if visible() && isLit {
				s.DrawSimpleAppliance(thisObject.What, itsRect)
			}

		case kToaster:
			// No isLit gate, unlike the shredder beside it.
			if visible() {
				s.DrawSimpleAppliance(thisObject.What, itsRect)
				if !redraw && neighbor == kCentralRoom {
					s.addDynamicObject(kToaster, itsRect, room, int16(i), thisObject.Appliance().State != 0)
				}
			}

		case kMacPlus, kTV, kCoffee, kVCR, kStereo, kMicrowave:
			if visible() {
				g := thisObject.Appliance()
				isOn := g.State != 0
				switch thisObject.What {
				case kMacPlus:
					s.DrawMacPlus(itsRect, isOn, isLit)
				case kTV:
					s.DrawTV(itsRect, isOn, isLit)
				case kCoffee:
					s.DrawCoffee(itsRect, isOn, isLit)
				case kVCR:
					s.DrawVCR(itsRect, isOn, isLit)
				case kStereo:
					// isPlayMusicGame, not the object's own state: a stereo shows
					// whether the *game* has music on.
					s.DrawStereo(itsRect, s.PlayMusicGame, isLit)
				case kMicrowave:
					s.DrawMicrowave(itsRect, isOn, isLit)
				}
				if !redraw {
					s.addDynamicObject(thisObject.What, itsRect, room, int16(i), isOn)
				}
			}

		case kOutlet:
			if visible() {
				if isLit {
					s.DrawOutlet(itsRect)
				}
				if !redraw {
					s.addDynamicObject(kOutlet, itsRect, room, int16(i), thisObject.Appliance().State != 0)
				}
			}

		// --- enemies -------------------------------------------------------
		case kBalloon, kCopterLf, kCopterRt, kDartLf, kDartRt, kBall:
			// Dynamic only, and only in the central room: these six draw nothing
			// into the static background at all.
			if neighbor == kCentralRoom && !redraw {
				rect()
				s.addDynamicObject(thisObject.What, itsRect, room, int16(i), thisObject.Enemy().State != 0)
			}

		case kDrip:
			if visible() {
				s.DrawDrip(itsRect)
				if !redraw && neighbor == kCentralRoom {
					s.addDynamicObject(kDrip, itsRect, room, int16(i), thisObject.Enemy().State != 0)
				}
			}

		case kFish:
			if visible() {
				s.DrawFish(thisObject.What, itsRect)
				if !redraw && neighbor == kCentralRoom {
					s.addDynamicObject(kFish, itsRect, room, int16(i), thisObject.Enemy().State != 0)
				}
			}

		case kCobweb, kCloud:
			if visible() && isLit {
				s.DrawPictWithMaskObject(thisObject.What, itsRect)
			}

		// --- clutter -------------------------------------------------------
		case kMirror:
			if visible() && isLit {
				s.DrawMirror(itsRect)
			}
			// Outside the isLit gate and outside the intersection test: an unlit
			// mirror still reflects.
			if neighbor == kCentralRoom && !redraw {
				s.addToMirrorRegion(Inset(itsRect, 4, 4))
			}

		case kMousehole, kFaucet:
			if visible() && isLit {
				s.DrawSimpleClutter(thisObject.What, itsRect)
			}

		case kFlower:
			if visible() && isLit {
				s.DrawFlower(itsRect, thisObject.Clutter().Pict)
			}

		case kWallWindow:
			if visible() {
				s.DrawWallWindow(itsRect)
			}

		case kCalendar:
			if visible() && isLit {
				s.DrawCalendar(itsRect)
			}

		case kBulletin:
			if visible() && isLit {
				s.DrawBulletin(itsRect)
			}
		}

		// The link-table pass over masterObjects goes here; see the file header.
	}
}

// candleFlame is the tail of the kTaper, kCandle and kStubby cases, which the
// original writes out three times with different offsets.
//
// The offsets are the flame's anchor. The condition is the interesting part: a
// candle in the central room always gets a flame, but a candle in a *neighbouring*
// room gets one only if the flame's 16x15 rect misses the central room's area
// grown by a floor support in each direction. A flame is animated by restoring a
// saved patch of background each frame, and a patch that overlapped the play area
// would erase whatever the glider had drawn there.
func (s *Scene) candleFlame(neighbor, who int, itsRect Rect, hOff, vOff int16, redraw bool) {
	room := s.LocalNumbers[neighbor]
	h, v := itsRect.Left+hOff, itsRect.Top+vOff

	if neighbor != kCentralRoom {
		flame := Offset(SetRect(0, 0, 16, 15), h-8, v-15)
		guard := s.V.LocalRoomsDest[kCentralRoom]
		guard.Top -= kFloorSupportTall
		guard.Bottom += kFloorSupportTall
		if _, ok := Sect(flame, guard); ok {
			return
		}
	}
	if redraw {
		// ReBackUpFlames: re-stash under the existing slot.
		return
	}
	s.addCandleFlame(room, int16(who), h, v)
}

// drawGrease dispatches the two grease directions, which the original writes as
// two near-identical cases.
func (s *Scene) drawGrease(what int16, theRect Rect, distance int16, state bool) {
	if what == kGreaseRt {
		s.DrawGreaseRt(theRect, distance, state)
	} else {
		s.DrawGreaseLf(theRect, distance, state)
	}
}

// ---------------------------------------------------------------------------
// Room queries
// ---------------------------------------------------------------------------

// room resolves a room number, returning nil for kRoomIsEmpty.
//
// The original indexes rooms[-1] in two places -- GetNumberOfLights and
// GetObjectState can both be reached with kRoomIsEmpty -- and reads whatever is
// in front of the array. Neither result is ever observed: DrawRoomBackground's
// black-fill branch is gated on who != kRoomIsEmpty, DrawARoomsObjects returns
// immediately for an empty room, and a switch pointing at a room that does not
// exist is a broken link the game does not draw differently. Returning nil, and
// then 0 lights / a true state, is safe and is what those two functions do below.
func (s *Scene) room(n int16) *house.Room {
	if n == kRoomIsEmpty || n < 0 || int(n) >= len(s.H.Rooms) {
		return nil
	}
	return &s.H.Rooms[n]
}

// GetNeighborRoomNumber is Room.c:562-633: which room is in a given direction.
//
// Rooms are addressed by floor and suite, not by adjacency, so a neighbour is
// found by arithmetic on the current room's coordinates and then a linear search.
// A house with no room at those coordinates simply has a hole in it, and the hole
// is drawn as dirt, meadow or sky depending on elevation.
//
// Note that north is floor *plus* one: floors count upward from the deepest
// basement.
func (s *Scene) GetNeighborRoomNumber(which int) int16 {
	var hDelta, vDelta int16
	switch which {
	case kCentralRoom:
	case kNorthRoom:
		vDelta = 1
	case kNorthEastRoom:
		hDelta, vDelta = 1, 1
	case kEastRoom:
		hDelta = 1
	case kSouthEastRoom:
		hDelta, vDelta = 1, -1
	case kSouthRoom:
		vDelta = -1
	case kSouthWestRoom:
		hDelta, vDelta = -1, -1
	case kWestRoom:
		hDelta = -1
	case kNorthWestRoom:
		hDelta, vDelta = -1, 1
	}
	rm := s.room(s.RoomNumber)
	if rm == nil {
		return kRoomIsEmpty
	}
	return s.GetRoomNumber(rm.Floor+vDelta, rm.Suite+hDelta)
}

// GetRoomNumber is Room.c:737-760: the index of the room at a floor and suite, or
// kRoomIsEmpty. A linear search, as in the original -- houses top out at a few
// hundred rooms and this runs nine times per composition.
func (s *Scene) GetRoomNumber(floor, suite int16) int16 {
	for i := range s.H.Rooms {
		if s.H.Rooms[i].Floor == floor && s.H.Rooms[i].Suite == suite {
			return int16(i)
		}
	}
	return kRoomIsEmpty
}

// IsRoomAStructure is Room.c:763-812: does this room have a floor and a ceiling?
//
// Only structures get floor-support beams. For the eighteen built-in backgrounds
// it is a fixed list, and the surprises in it are that the basement and the garden
// are *not* structures while the roof is.
//
// For a house's own background the answer comes from the room's bounds field if it
// has one -- bit 5 is the structure flag -- and otherwise from the id: below
// kUserStructureRange (3300) is taken to be an interior. That fallback exists for
// houses written before the bounds field did.
func (s *Scene) IsRoomAStructure(roomNum int16) bool {
	rm := s.room(roomNum)
	if rm == nil {
		return false
	}
	if rm.Background >= kUserBackground {
		if rm.Bounds != 0 {
			return rm.Bounds&32 == 32
		}
		return rm.Background < kUserStructureRange
	}
	switch rm.Background {
	case kPaneledRoom, kSimpleRoom, kChildsRoom, kAsianRoom, kUnfinishedRoom,
		kSwingersRoom, kBathroom, kLibrary, kSkywalk, kRoof:
		return true
	}
	return false
}

// GetNumberOfLights is Room.c:970-1095 (its play-mode branch): how lit a room is.
//
// Outdoor backgrounds are lit by definition. Dirt is lit only if all eight tiles
// are tile 0, which is the open-sky tile -- a dirt room with any solid tile is
// underground and dark. Everything else counts its light sources: interior doors
// and windows always count, lamps and fixtures count when switched on.
//
// The count itself is never used as a number, only as zero-or-not. Edit mode reads
// data.f.initial instead of data.f.state, which is why a lamp switched off during
// play still looks on in the editor.
func (s *Scene) GetNumberOfLights(where int16) int {
	rm := s.room(where)
	if rm == nil {
		// rooms[-1] in the original; see the note on Scene.room.
		return 0
	}

	count := 0
	switch rm.Background {
	case kGarden, kSkywalk, kMeadow, kField, kRoof, kSky, kStratosphere, kStars:
		count = 1
	case kDirt:
		count = 1
		for _, t := range rm.Tiles {
			if t != 0 {
				count = 0
				break
			}
		}
	}
	if count != 0 {
		return count
	}

	for i := 0; i < kMaxRoomObs; i++ {
		switch rm.Objects[i].What {
		case kDoorInLf, kDoorInRt, kWindowInLf, kWindowInRt, kWallWindow:
			count++
		case kCeilingLight, kLightBulb, kTableLamp, kHipLamp, kDecoLamp,
			kFlourescent, kTrackLight, kInvisLight:
			if rm.Objects[i].Light().State != 0 {
				count++
			}
		}
	}
	return count
}

// IsThisValid is Objects.c:87-121: should this object slot be processed at all?
//
// Empty slots are skipped, and so are the twelve collectables that have already
// been taken -- their state byte is cleared when the glider picks them up, and a
// taken prize must not reappear when the room is recomposed.
func (s *Scene) IsThisValid(where int16, who int) bool {
	rm := s.room(where)
	if rm == nil || who < 0 || who >= kMaxRoomObs {
		return false
	}
	obj := rm.Objects[who]
	switch obj.What {
	case house.ObjectIsEmpty:
		return false
	case kRedClock, kBlueClock, kYellowClock, kCuckoo, kPaper, kBattery, kBands,
		kFoil, kInvisBonus, kStar, kSparkle, kHelium:
		return obj.Bonus().State != 0
	}
	return true
}

// GetObjectState is Objects.c:700-865: is the object at (room, object) on?
//
// The default is true, and most kinds keep it -- asking whether a table is on is
// not a meaningful question. The kinds that do answer read their state from
// whichever of the nine data layouts they use, which is why this is a long switch
// rather than one field access.
//
// Two entries are worth noting. A stereo answers with isPlayMusicGame, ignoring
// its own data entirely. And kKnifeSwitch is absent from the original's switch, so
// a knife switch always reports on -- which means a knife switch wired to another
// knife switch draws itself in the on position regardless.
func (s *Scene) GetObjectState(room int16, object int16) bool {
	rm := s.room(room)
	if rm == nil || object < 0 || int(object) >= kMaxRoomObs {
		// rooms[-1] again; see the note on Scene.room. true is the default the
		// original's switch would fall through to.
		return true
	}
	obj := rm.Objects[object]
	switch obj.What {
	case kFloorVent, kCeilingVent, kFloorBlower, kCeilingBlower, kLeftFan,
		kRightFan, kSewerGrate, kInvisBlower, kGrecoVent, kSewerBlower, kLiftArea:
		return obj.Blower().State != 0

	case kRedClock, kBlueClock, kYellowClock, kCuckoo, kPaper, kBattery, kBands,
		kGreaseRt, kGreaseLf, kFoil, kInvisBonus, kStar, kSparkle, kHelium:
		return obj.Bonus().State != 0

	case kDeluxeTrans:
		return obj.Transport().Wide&0x0F != 0

	case kCeilingLight, kLightBulb, kTableLamp, kHipLamp, kDecoLamp,
		kFlourescent, kTrackLight, kInvisLight:
		return obj.Light().State != 0

	case kStereo:
		return s.PlayMusicGame

	case kShredder, kToaster, kMacPlus, kGuitar, kTV, kCoffee, kOutlet, kVCR,
		kMicrowave:
		return obj.Appliance().State != 0

	case kBalloon, kCopterLf, kCopterRt, kDartLf, kDartRt, kBall, kDrip, kFish:
		return obj.Enemy().State != 0
	}
	return true
}

// ExtractFloorSuite is Link.c:39-53: unpack a link's floor and suite.
//
// The two fields share one short, and houses before version 2.0 packed them the
// other way round. Both orders are still read, because the shipped houses include
// both.
func (s *Scene) ExtractFloorSuite(combo int16) (floor, suite int16) {
	if s.H.Version < 0x0200 {
		return combo/100 - kNumUndergroundFloors, combo % 100
	}
	return combo%100 - kNumUndergroundFloors, combo / 100
}

// ---------------------------------------------------------------------------
// The dynamic tables
// ---------------------------------------------------------------------------

// backUpToSavedMap is DynamicMaps.c:70-93: stash the background under theRect so
// an animated object can erase itself later, and return the slot index.
//
// The -1 it returns when the table is full is why this belongs in a static
// renderer at all. Every caller gates its *draw* on the result, so the 25th
// animated object in view is not merely un-animated -- it is invisible.
//
// redraw selects ReBackUpSavedMap (DynamicMaps.c:100-127), which finds the
// existing slot for this object and refreshes it instead of claiming a new one.
// It is not reached by a fresh composition.
func (s *Scene) backUpToSavedMap(theRect Rect, where, who int16, redraw bool) int {
	if redraw {
		for i := range s.SavedMaps {
			if s.SavedMaps[i].Where == where && s.SavedMaps[i].Who == who {
				s.SavedMaps[i].Map = s.patch(theRect)
				return i
			}
		}
		return -1
	}
	if len(s.SavedMaps) >= kMaxSavedMaps {
		return -1
	}
	s.SavedMaps = append(s.SavedMaps, SavedMap{Map: s.patch(theRect), Where: where, Who: who})
	return len(s.SavedMaps) - 1
}

// patch copies a rect out of Back into a surface of its own.
func (s *Scene) patch(r Rect) *Surface {
	p := NewSurface(int(r.Wide()), int(r.Tall()))
	p.Copy(s.Back, r, p.Bounds(), SrcCopy)
	return p
}

// addGrease is Grease.c:206-250. The saved map it takes is 32x108 -- the whole
// spill strip, not the object's rect -- because a spill spreads as it is used.
func (s *Scene) addGrease(where, who int16, redraw bool) int {
	if redraw {
		return s.backUpToSavedMap(SetRect(0, 0, 32, 108), where, who, true)
	}
	if s.numGrease >= kMaxGrease {
		return -1
	}
	if s.backUpToSavedMap(SetRect(0, 0, 32, 108), where, who, false) == -1 {
		return -1
	}
	s.numGrease++
	return s.numGrease - 1
}

// addCandleFlame is DynamicMaps.c:316-343. The bounds handed to the saved map are
// one frame wide and five frames tall, so the slot holds the frame strip as well
// as the background.
func (s *Scene) addCandleFlame(where, who, h, v int16) {
	if len(s.Flames) >= kMaxCandles || h < 16 || v < 15 {
		return
	}
	dest := Offset(SetRect(0, 0, 16, 15), h-8, v-15)
	slot := s.backUpToSavedMap(SetRect(0, 0, 16, 15*kNumCandleFrames), where, who, false)
	if slot == -1 {
		return
	}
	s.Flames = append(s.Flames, Anim{Dest: dest, SavedMap: slot, Where: where, Who: who})
}

// addTikiFlame is DynamicMaps.c:400-428. Unlike the candle the anchor is the
// top-left, not the bottom-centre.
func (s *Scene) addTikiFlame(where, who, h, v int16) {
	if len(s.TikiFlames) >= kMaxTikis || h < 8 || v < 10 {
		return
	}
	dest := Offset(SetRect(0, 0, 8, 10), h, v)
	slot := s.backUpToSavedMap(SetRect(0, 0, 8, 10*kNumTikiFrames), where, who, false)
	if slot == -1 {
		return
	}
	s.TikiFlames = append(s.TikiFlames, Anim{Dest: dest, SavedMap: slot, Where: where, Who: who})
}

// addBBQCoals is DynamicMaps.c:486-514.
func (s *Scene) addBBQCoals(where, who, h, v int16) {
	if len(s.Coals) >= kMaxCoals || h < 32 || v < 9 {
		return
	}
	dest := Offset(SetRect(0, 0, 32, 9), h, v)
	slot := s.backUpToSavedMap(SetRect(0, 0, 32, 9*kNumCoalFrames), where, who, false)
	if slot == -1 {
		return
	}
	s.Coals = append(s.Coals, Anim{Dest: dest, SavedMap: slot, Where: where, Who: who})
}

// addPendulum is DynamicMaps.c:570-604. Note that the original claims the saved
// map *before* computing dest, the opposite order from the flames -- immaterial,
// but it is why a pendulum whose h or v is too small still short-circuits first.
func (s *Scene) addPendulum(where, who, h, v int16) {
	if len(s.Pendulums) >= kMaxPendulums || h < 32 || v < 28 {
		return
	}
	slot := s.backUpToSavedMap(SetRect(0, 0, 32, 28*kNumPendulumFrames), where, who, false)
	if slot == -1 {
		return
	}
	dest := Offset(SetRect(0, 0, 32, 28), h, v)
	s.Pendulums = append(s.Pendulums, Anim{Dest: dest, SavedMap: slot, Where: where, Who: who})
}

// addStar is DynamicMaps.c:662-694. A star is the only object that consumes two
// savedMaps slots: one claimed by DrawARoomsObjects for the star itself and one
// here for its six-frame spin.
func (s *Scene) addStar(where, who, h, v int16) {
	if len(s.Stars) >= kMaxStars {
		return
	}
	dest := Offset(SetRect(0, 0, 32, 31), h, v)
	slot := s.backUpToSavedMap(SetRect(0, 0, 32, 31*kNumStarFrames), where, who, false)
	if slot == -1 {
		return
	}
	s.Stars = append(s.Stars, Anim{Dest: dest, SavedMap: slot, Where: where, Who: who})
}

// addDynamicObject is Dynamics3.c:187: register something that moves or blinks.
//
// This stage records the registration and stops there. Nothing in the original's
// version draws a pixel either -- it fills in a dinahs entry, whose per-kind
// initial velocities, frame counters and RandomInt phases are Stage 1.5's
// business. The rect is stored room-local, with playOrigin subtracted, because
// that is the space the animation loop works in.
func (s *Scene) addDynamicObject(what int16, itsRect Rect, room, obj int16, on bool) int {
	if len(s.Dynamics) >= kMaxDynamicObs {
		return -1
	}
	s.Dynamics = append(s.Dynamics, Dynamic{
		What: what,
		Rect: Offset(itsRect, -s.V.OriginH, -s.V.OriginV),
		Room: room,
		Obj:  obj,
		On:   on,
	})
	return len(s.Dynamics) - 1
}

// AddTempManholeRect is Objects.c:349-362: remember a manhole so
// DrawFloorSupport can punch it through the beam below.
//
// The stored rect is the manhole's left and right but the floor support's height,
// because what gets drawn is the hole seen edge-on through the beam rather than
// the manhole cover itself.
func (s *Scene) AddTempManholeRect(manHole Rect) {
	if len(s.TempManholes) >= kMaxTempManholes {
		return
	}
	manHole.Bottom = manHole.Top + kFloorSupportTall
	s.TempManholes = append(s.TempManholes, manHole)
}

// addToMirrorRegion is Render.c:740-761. The original unions rects into a
// QuickDraw region; a slice of rects is the same thing for every use the game
// makes of it, which is to clip the glider's reflection.
func (s *Scene) addToMirrorRegion(theRect Rect) {
	s.MirrorRects = append(s.MirrorRects, theRect)
}

// ---------------------------------------------------------------------------
// Blitting helpers
// ---------------------------------------------------------------------------

// maskSheet is CopyMask out of one of the shared sheet GWorlds into backSrcMap.
// Almost every object draw is one of these.
func (s *Scene) maskSheet(sheet string, src, dst Rect) {
	if art := s.A.Sheet(sheet); art != nil {
		s.Back.Copy(art, src, dst, Masked)
	}
}

// opaqueSheet is the same blit with srcCopy: the overlays that are meant to cover
// what is under them -- clock digits, appliance screens, indicator LEDs.
func (s *Scene) opaqueSheet(sheet string, src, dst Rect) {
	if art := s.A.Sheet(sheet); art != nil {
		s.Back.Copy(art, src, dst, SrcCopy)
	}
}

// maskObject is the strategy-B and strategy-C draws collapsed into one: an
// object's own PICT, with the extractor's resolved alpha as the mask. See the
// note at the top of objectdraw2.go for why a white colour key and a mask are the
// same thing here.
func (s *Scene) maskObject(what int16, src, dst Rect) {
	if art := s.A.Object(what); art != nil {
		s.Back.Copy(art, src, dst, Masked)
	}
}
