package render

import (
	"fmt"
	"time"

	"github.com/bwenstar/gliderGo/internal/house"
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
// The one thing in here that affects no pixel at all is the dynamicNum
// bookkeeping, and it is here anyway. DrawARoomsObjects tracks a per-slot
// dynamicNum -- a savedMaps grease slot, a hotSpots index for the six switch
// kinds, or a dinahs slot for the seventeen animated types -- and writes it back
// into every master-object entry pointing at that object. That is the
// switch-to-object link table the game runs on, and the object pass is the only
// place that knows the numbers, so the three writes and one read cross the
// package boundary as hooks (SetDynaNum, AddDynamicObject, ZeroDinahs,
// MasterHotNum) rather than being reinvented on the game side. The tables they
// index are internal/game's; the numbering is this file's.

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
//
// The caps stay unexported: refusing a registration is this package's decision.
const (
	kMaxCandles    = 20
	kMaxTikis      = 8
	kMaxCoals      = 8
	kMaxPendulums  = 8
	kMaxStars      = 4
	kMaxDynamicObs = 18
)

// The five families' cel counts, which are how tall a filmstrip is baked here and how
// far an animator counts before it wraps in internal/game/anim.go. Exported because
// they are the one part of the strip layout both halves have to agree on; TestFrame-
// CountsMatchTheSrcTables pins each against the src-rect table it indexes.
//
// NumCoalFrames is the C's kNumBBQCoals, renamed for the family it belongs to -- the
// original's name reads like a table cap and is not one (that is kMaxCoals, 8).
// RenderStars does not use its constant at all and writes a bare 6.
const (
	NumCandleFrames   int16 = 5
	NumTikiFrames     int16 = 5
	NumCoalFrames     int16 = 4
	NumPendulumFrames int16 = 3
	NumStarFrames     int16 = 6
)

// SavedMap is one entry in the savedMaps table (DynamicMaps.c:70-93): a patch of
// backSrcMap copied aside before something animated was drawn over it, so the
// animation can restore the background each frame.
type SavedMap struct {
	// Map is the stashed pixels. Its size is the *bounds* the caller asked for,
	// which for a flame or a star is one frame wide and every frame tall -- the
	// slot doubles as scratch for the frame strip.
	Map *Surface

	// Dest is the rect the pixels came from, in screen coordinates, and is where
	// RestoreFromSavedMap puts them back.
	//
	// **It is not always the object's rect**, because the five animated families ask
	// for a strip rather than a swatch: a star's second slot records (0,0,32,186),
	// the top-left corner of the screen, because that is the rect AddStar handed to
	// BackUpToSavedMap. Restoring through such a slot would paint the frame strip
	// over the corner of the play area. Nothing does, and the reason is worth
	// knowing: only prizes are ever restored, a prize's *first* slot is claimed by
	// DrawARoomsObjects with the object's real rect, and RestoreFromSavedMap stops at
	// the first match -- so the star's and the cuckoo's second slots are unreachable
	// from it. See the note on addStar.
	Dest Rect

	// Where and Who identify the object, so RestoreFromSavedMap can find it.
	Where int16
	Who   int16
}

// Anim, the five tables' entry type, is in anim.go beside the functions that fill
// it and the filmstrips they bake.

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
	TempManholes []Rect
	MirrorRects  []Rect

	// Grease is grease[] (Grease.c:26), capped at kMaxGrease. Written by both halves
	// of the port -- see grease.go for why it is on Scene rather than behind a hook.
	Grease []Grease

	// SavedMapDrops is every registration this locale's 24 slots refused, and it has no
	// counterpart in the original: DynamicMaps.c drops them silently.
	//
	// It is recorded because of what a drop *does*. Every caller of backUpToSavedMap
	// gates the object's draw on the result, so the 25th animated object in view is not
	// merely un-animated, it is **invisible** -- and the objects most likely to be over
	// the line are stars and prizes, which are the ones a house is scored on. In 1994 a
	// house author found this out by playing the room and noticing something missing. A
	// port that ships a house editor (Stage 5) can tell them instead, and the release
	// build's diagnostics can say it happened.
	//
	// Cleared by DrawLocale beside SavedMaps itself, so it always describes the locale
	// currently composed. Nothing in the frame loop reads it; it is a report.
	SavedMapDrops []SavedMapDrop

	// ClockFrame is clockFrame (DynamicMaps.c:33), the pendulum phase counter shared
	// by every cuckoo clock in the locale. Written here by addPendulum and stepped by
	// the game side's RenderPendulums, so it sits with the table it paces rather than
	// behind a hook.
	//
	// It is **not** in ZeroFlamesAndTheLike's reset list. addPendulum seeds it to 10
	// on every registration, and a locale with no pendulum never reads it, so a stale
	// value is unobservable -- which is presumably why nobody noticed it was missing.
	ClockFrame int16

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

	// The five dinahs hooks, all for the same reason as ListLocalObjects and all
	// nil-safe: the table is simulation state and lives in internal/game, but the
	// composition is what decides which objects register and where.
	//
	// nil throughout composes exactly the same image -- a room where nothing moves,
	// which is the correct still frame and is what internal/render's own tests and
	// the golden use.

	// ZeroDinahs is ZeroDinahs (Dynamics3.c:160), called from DrawLocale's reset
	// head. A hook rather than something World.Rebuild does before calling
	// DrawLocale, because the C's ordering has it *inside* DrawLocale and
	// RestoreEntireGameScreen reaches DrawLocale directly (Play.c:814) without going
	// through Rebuild -- the trap readylevel.go documents.
	ZeroDinahs func()

	// KillAllBands is KillAllBands (RubberBands.c:307-317), the third line of
	// DrawLocale's reset head. A hook for exactly ZeroDinahs' reason: the band table
	// is simulation state on World and this is the same composition.
	//
	// nil composes the same image -- bands are drawn by RenderBands, which the
	// renderer's own tests never reach -- so unlike Scene.Grease there is nothing
	// lost by keeping the table on the game side. Note what the reset means: **a
	// band in flight does not survive a room change.** Walking through a door
	// deletes it, and the ammunition is not refunded.
	KillAllBands func()

	// ZeroShreds is the `numShredded = 0` of ZeroFlamesAndTheLike (DynamicMaps.c:794),
	// a hook for KillAllBands' reason: the particle table is on World because
	// AddAShreddedGlider is called from the player's own death animation, but the
	// reset belongs to the composition.
	//
	// nil composes the same image, and what the reset means is the same shape as the
	// bands': **a glider mid-shred does not leave its confetti behind on a room
	// change.** Since a shredded glider is about to respawn elsewhere, the reset is
	// the only thing that stops a dead player's pieces raining down in the next room.
	ZeroShreds func()

	// RandomInt is World.RandomInt, and it is the only hook here whose *return value*
	// the composition depends on rather than merely reporting to. Four of the five
	// animated families seed their starting cel from it and the pendulum draws for its
	// direction, all five draws inside the saturation guard -- see anim.go for why
	// that makes the RNG stream depend on the window size.
	//
	// nil returns 0 rather than panicking, which makes a render-only composition start
	// every flame on cel 0. That is deterministic and invisible in the goldens, because
	// a filmstrip lives in a saved map and never reaches the composed image.
	RandomInt func(rng int16) int16

	// AddDynamicObject is AddDynamicObject (Dynamics3.c:187-554), called from
	// seventeen places inside DrawARoomsObjects. `where` is room-local, playOrigin
	// already subtracted, exactly as the C's seventeen call sites pass it. Returns
	// the new slot or -1; DrawARoomsObjects writes that number back through
	// SetDynaNum.
	AddDynamicObject func(what int16, where Rect, obj house.Object, room, index int16, isOn bool) int16

	// SetDynaNum is the write-back at ObjectDrawAll.c:952-960, called once per object
	// slot at the bottom of DrawARoomsObjects' loop -- **including with -1**, which is
	// what clears a stale slot number from the previous room. Gated on !redraw, like
	// the C's `if (!redraw) // set up links`.
	SetDynaNum func(room, obj, dyna int16)

	// MasterHotNum answers the six switch cases' `dynamicNum = masterObjects[i].hotNum`.
	// A read hook where the other two are write hooks; see its game-side comment for
	// why the index it is given is the original's wrong one and stays that way.
	MasterHotNum func(obj int16) int16

	// UpdateOutletsLighting is Trip.c:235-244, step four of RedrawCentralRoom's six.
	// The fifth dinahs hook and the only one not reached from DrawLocale: a light
	// switch changes what an outlet in *this* room paints on its last zap frame, and
	// only the game side can write that.
	UpdateOutletsLighting func(room, nLights int16)
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
	// ZeroTriggers, numTempManholes = 0. The trigger table is reset by
	// World.Rebuild; the rest are here.
	//
	// ZeroFlamesAndTheLike is eight assignments. Six are the six tables cleared below;
	// numShredded is on World and comes back through the ZeroShreds hook; numChimes is
	// World.Rebuild's, because the count is read by the game's ambience clock and not by
	// anything drawn. The one it does *not* touch is clockFrame -- see Scene.ClockFrame.
	s.SavedMaps = s.SavedMaps[:0]
	s.SavedMapDrops = s.SavedMapDrops[:0] // not the C's; see the field
	s.Flames = s.Flames[:0]
	s.TikiFlames = s.TikiFlames[:0]
	s.Coals = s.Coals[:0]
	s.Pendulums = s.Pendulums[:0]
	s.Stars = s.Stars[:0]
	s.TempManholes = s.TempManholes[:0]
	s.MirrorRects = s.MirrorRects[:0]
	s.Grease = s.Grease[:0]
	if s.ZeroDinahs != nil {
		s.ZeroDinahs()
	}
	if s.KillAllBands != nil {
		s.KillAllBands()
	}
	if s.ZeroShreds != nil {
		s.ZeroShreds()
	}

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

// RedrawCentralRoom is the body of RedrawRoomLighting (RoomGraphics.c:448-460): recompose
// the central room alone, because a light in it just went on or off.
//
// It is DrawLocale's own central-room tail with two differences, and both are the point of
// having a separate method:
//
//   - **`redraw` is true**, which suppresses the twenty registration sites inside
//     DrawARoomsObjects. So the pixels are repainted and the *live objects are not
//     re-created*: a band still in flight survives, the mirror region is not rebuilt, a
//     dinah mid-swoop keeps its position, and SetDynaNum is not called with a stale -1.
//     That is the whole reason the C passes a flag here and nowhere else.
//   - **the reset head is skipped.** No ZeroDinahs, no ZeroFlamesAndTheLike, no
//     ListAllLocalObjects, and no `Back.Fill` -- this draws over the eight neighbouring
//     rooms already in backSrcMap rather than clearing them, which is what makes it a
//     redraw of one ninth of the picture instead of a fresh composition.
//
// Step four is UpdateOutletsLighting, and it is a hook because it writes the dinahs table.
// **NumLights is not recounted here.** The C recounts it in RedrawRoomLighting, above the
// six steps, because the recount is also what produces `isLit` for the gate -- so the
// caller owns it, and by the time this runs NumLights already holds the central room's new
// count. Reassigning it here would be harmless but would put the same recount in two
// places, and the one-writer property is what makes UpdateOutletsLighting's room filter
// work at all (see its game-side comment).
//
// The caller likewise owns the gate and the two lines the C runs after RestoreWorkMap --
// the work rect and the ShadowVisible recache -- because both are gameplay state; see
// World.RedrawRoomLighting.
func (s *Scene) RedrawCentralRoom() {
	roomV := int16(0)
	if rm := s.room(s.RoomNumber); rm != nil {
		roomV = rm.Floor
	}

	s.DrawRoomBackground(s.LocalNumbers[kCentralRoom], kCentralRoom, roomV)
	s.DrawARoomsObjects(kCentralRoom, true)
	s.DrawLighting()
	if s.UpdateOutletsLighting != nil {
		s.UpdateOutletsLighting(s.LocalNumbers[kCentralRoom], int16(s.NumLights))
	}

	if s.NumNeighbors > 3 {
		s.DrawFloorSupport()
	}
	s.RestoreWorkMap()
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
		// dynamicNum and legit are both reset per object in the original
		// (ObjectDrawAll.c:45-46). dynamicNum is the slot number this object's
		// registration returned, and the reset is what stops a stale one from the
		// previous room surviving; see the write-back at the bottom of the loop.
		dynamicNum := int16(-1)
		legit := -1

		if !s.IsThisValid(room, i) {
			// The C's write-back is *outside* its IsThisValid guard, so an empty or
			// invalid slot still has its stale dynaNum cleared. This port uses an
			// early continue where the C nests, so the -1 has to be written here as
			// well as at the bottom.
			if !redraw && s.SetDynaNum != nil {
				s.SetDynaNum(room, int16(i), -1)
			}
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
			if redraw {
				s.ReBackUpTikiFlames(room, int16(i))
			} else {
				s.addTikiFlame(room, int16(i), itsRect.Left+10, itsRect.Top-9)
			}

		case kBBQ:
			if visible() {
				if isLit {
					s.DrawPictSansWhiteObject(thisObject.What, itsRect)
				}
				if redraw {
					s.ReBackUpBBQCoals(room, int16(i))
				} else {
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
					// The second of the clock's two saved-map slots: the one
					// above holds the clock face so it can be erased when the
					// prize is collected, and addPendulum claims another for
					// the swing's filmstrip. Both are tagged (room, i), which
					// is what anim.go's "missing break" note is about.
					//
					// **The draw is before the registration**, the opposite of
					// the star's below, and for the opposite reason. A filmstrip
					// is baked from the back map, so baking after DrawCuckoo is
					// what puts the clock's case *behind* the pendulum in all
					// three cels -- the pendulum swings in front of the case, so
					// it needs the case as its background. A star, by contrast,
					// *is* the thing its cels replace, so it registers first.
					if redraw {
						s.ReBackUpPendulum(room, int16(i))
					} else {
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
					// The first of dynaNum's three meanings: a grease slot, not a
					// dinahs slot. See the game side's SetDynaNum.
					//
					// The two branches are two different functions rather than
					// one with a flag, because a redraw must not claim a second
					// saved-map slot for a jar that already has one -- see
					// ReBackUpGrease.
					if redraw {
						dynamicNum = s.ReBackUpGrease(room, int16(i))
					} else {
						dynamicNum = s.AddGrease(room, int16(i),
							itsRect.Left, itsRect.Top, c.Length,
							thisObject.What == kGreaseRt)
					}
					if dynamicNum != -1 {
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
					//
					// **The registration is before the draw, and that is not
					// interchangeable.** Object draws land in the back map and
					// a filmstrip is baked from the back map, so registering
					// first is what keeps the static star *out* of the cels --
					// which is required, because the cels are what replaces it.
					// Swap the two lines and every cel of the spin has a
					// stationary star painted underneath it. Contrast the
					// cuckoo above, which draws first for the mirror-image
					// reason.
					if redraw {
						s.ReBackUpStar(room, int16(i))
					} else {
						s.addStar(room, int16(i), itsRect.Left, itsRect.Top)
					}
					s.DrawSimplePrizes(thisObject.What, itsRect)
				}
			}

		case kSparkle:
			// Dynamic only: a sparkle contributes nothing to the static room.
			if visible() && !redraw && neighbor == kCentralRoom {
				dynamicNum = s.addDynamicObject(kSparkle, itsRect, thisObject, room, int16(i),
					thisObject.Bonus().State != 0)
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
			// The second of dynaNum's three meanings, for all six switch kinds:
			// a *hotSpots* index, which is what makes TriggerSwitch's
			// HandleSwitches(&hotSpots[who]) well-typed. The index this is given
			// is the original's wrong one for eight of the nine rooms -- see the
			// game side's MasterHotNum.
			if s.MasterHotNum != nil {
				dynamicNum = s.MasterHotNum(int16(i))
			}

		case kInvisSwitch:
			// A hotSpots index and nothing else -- an invisible switch draws
			// nothing but is still throwable.
			if s.MasterHotNum != nil {
				dynamicNum = s.MasterHotNum(int16(i))
			}

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
					dynamicNum = s.addDynamicObject(kToaster, itsRect, thisObject, room, int16(i),
						thisObject.Appliance().State != 0)
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
					dynamicNum = s.addDynamicObject(thisObject.What, itsRect, thisObject, room,
						int16(i), isOn)
					// `tvWithMovieNumber = dynamicNum` (ObjectDrawAll.c:707) belongs
					// here, for kTV only: the one place in the game where a
					// QuickTime handle is keyed by a dinahs slot. Not plumbed,
					// because the port has no movie support and Room.TVMovieNumber
					// therefore stays at the -1 Rebuild resets it to -- which makes
					// ToggleTV's four-condition identity test unreachable, as its
					// comment says.
				}
			}

		case kOutlet:
			if visible() {
				if isLit {
					s.DrawOutlet(itsRect)
				}
				if !redraw {
					dynamicNum = s.addDynamicObject(kOutlet, itsRect, thisObject, room, int16(i),
						thisObject.Appliance().State != 0)
				}
			}

		// --- enemies -------------------------------------------------------
		case kBalloon, kCopterLf, kCopterRt, kDartLf, kDartRt, kBall:
			// Dynamic only, and only in the central room: these six draw nothing
			// into the static background at all.
			if neighbor == kCentralRoom && !redraw {
				rect()
				dynamicNum = s.addDynamicObject(thisObject.What, itsRect, thisObject, room,
					int16(i), thisObject.Enemy().State != 0)
			}

		case kDrip:
			if visible() {
				s.DrawDrip(itsRect)
				if !redraw && neighbor == kCentralRoom {
					dynamicNum = s.addDynamicObject(kDrip, itsRect, thisObject, room, int16(i),
						thisObject.Enemy().State != 0)
				}
			}

		case kFish:
			if visible() {
				s.DrawFish(thisObject.What, itsRect)
				if !redraw && neighbor == kCentralRoom {
					dynamicNum = s.addDynamicObject(kFish, itsRect, thisObject, room, int16(i),
						thisObject.Enemy().State != 0)
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

		// `if (!redraw) // set up links` (ObjectDrawAll.c:952-960): record what this
		// object's registration returned, for grease, dinahs and hot spots alike.
		// The !redraw gate matters -- on a recompose the graph is already built and
		// must not be rewritten.
		if !redraw && s.SetDynaNum != nil {
			s.SetDynaNum(room, int16(i), dynamicNum)
		}
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
	// Both arms take the *room number*, not the neighbour index --
	// `ReBackUpFlames(localNumbers[neighbor], i)` in all six of the C's copies. The
	// distinction matters because the guard above reads the neighbour index and the
	// registration reads the room, and the two are only equal for room 0.
	if redraw {
		s.ReBackUpFlames(room, int16(who))
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
//
// **Dest is written only on the claim.** ReBackUpSavedMap re-copies the pixels and
// leaves the rect alone, and that is reproduced rather than tidied: every redraw
// passes the same rect the claim did, so writing it would be a no-op, and *not*
// writing it is what says the slot's identity is fixed at claim time.
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
		// Not in the original, which drops the registration silently. See
		// Scene.SavedMapDrops for why a port that ships to house authors should not.
		s.SavedMapDrops = append(s.SavedMapDrops,
			SavedMapDrop{Where: where, Who: who, Rect: theRect})
		return -1
	}
	s.SavedMaps = append(s.SavedMaps,
		SavedMap{Map: s.patch(theRect), Dest: theRect, Where: where, Who: who})
	return len(s.SavedMaps) - 1
}

// SavedMapDrop is one registration the 24-slot budget refused: which object asked, and for
// what.
//
// The Rect names the family without needing a label, because no two of them ask for the same
// size. A 16x75 request is a candle's filmstrip, 8x50 a torch's, 32x36 a barbecue's, 32x84 a
// pendulum's, 32x186 a star's and 32x108 a grease jar's; anything else is an object claiming
// a slot for its own rect so that it can be erased when it is collected.
type SavedMapDrop struct {
	Where int16 // the object's room number
	Who   int16 // its slot in that room's object array
	Rect  Rect  // the size it asked for
}

// patch copies a rect out of Back into a surface of its own.
func (s *Scene) patch(r Rect) *Surface {
	p := NewSurface(int(r.Wide()), int(r.Tall()))
	p.Copy(s.Back, r, p.Bounds(), SrcCopy)
	return p
}

// The registration functions that were here are in two other files now, beside the
// tables they write and the filmstrips they bake: AddGrease and ReBackUpGrease in
// grease.go, and the five flame-like families' add/backUp/ReBackUp triples in
// anim.go.

// StopPendulum is DynamicMaps.c:693-702 and StopStar is DynamicMaps.c:709-718: the
// cuckoo clock or the star this animation belongs to has been collected, so retire it.
//
// Three things about the pair. They match on the *object* (Where, Who) rather than on
// the table index, because the caller has an object and not an index. They do **not**
// break on the first match -- unlike RestoreFromSavedMap, which does -- so a house that
// somehow registered two pendulums for one clock would stop both; that is the original's
// loop and it costs nothing. And neither erases anything: the pixels are put back by
// RestoreFromSavedMap on the line before, and all these do is stop the animator drawing
// over them again next frame. Calling them without the restore leaves the last cel
// frozen on screen forever.
func (s *Scene) StopPendulum(where, who int16) {
	for i := range s.Pendulums {
		if s.Pendulums[i].Where == where && s.Pendulums[i].Who == who {
			s.Pendulums[i].Stopped = true
		}
	}
}

// StopStar is DynamicMaps.c:709-718. See StopPendulum.
func (s *Scene) StopStar(where, who int16) {
	for i := range s.Stars {
		if s.Stars[i].Where == where && s.Stars[i].Who == who {
			s.Stars[i].Stopped = true
		}
	}
}

// addDynamicObject forwards to the AddDynamicObject hook, subtracting playOrigin
// on the way.
//
// The subtraction is the one line each of the C's seventeen call sites writes for
// itself -- `QOffsetRect(&rectA, -playOriginH, -playOriginV)` immediately before the
// call (ObjectDrawAll.c:455, :651, :667, :700, :724, :741, :757, :773, :789, :801,
// :812, :823, :834, :845, :856, :871, :887). It is hoisted into this wrapper rather
// than repeated at the port's seven collapsed call sites so that there is one place
// to read the coordinate space off, and because getting it wrong at one site out of
// seventeen is exactly the sort of thing that produces an object drawn a screen's
// width away.
//
// A nil hook returns -1, which is indistinguishable from a saturated table.
func (s *Scene) addDynamicObject(what int16, itsRect Rect, obj house.Object, room, index int16, on bool) int16 {
	if s.AddDynamicObject == nil {
		return -1
	}
	return s.AddDynamicObject(what, Offset(itsRect, -s.V.OriginH, -s.V.OriginV),
		obj, room, index, on)
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
