package render

// Screen geometry and the three offscreen surfaces a room is composed through.
//
// In the original all of this is global state, set once at launch from the main
// device's bounds (InterfaceInit.c:196-218, StructuresInit2.c:156-162) and never
// changed. It is a struct here for one reason that matters and one that does not:
// the renderer becomes testable at a chosen resolution without a display, and a
// later windowed or resizable build gets somewhere to put a second view. The
// arithmetic is unchanged, and DefaultView() is the 640x480 screen the game was
// written for.

// Room and screen dimensions, from GliderPRO/Headers/GliderDefines.h:496-525.
const (
	kNumTiles = 8 // background tiles across a room

	kTileWide = 64
	kTileHigh = 322

	kRoomWide = kNumTiles * kTileWide // 512

	// kVertLocalOffset is how far a vertically adjacent room sits from this
	// one. The header's comment records it as kTileHigh - 39 and its own
	// history as "was 283, then 295", but the value shipped is a flat 322 --
	// equal to kTileHigh -- so neighbouring floors abut exactly and the floor
	// support beam is drawn into the seam rather than overlapping either room.
	kVertLocalOffset = 322

	kFloorSupportTall = 44
	kScoreboardTall   = 20

	// The largest view the game will build, whatever the monitor. A 1536x1026
	// screen shows nine full rooms; anything bigger is wasted.
	kMaxViewWidth  = 1536
	kMaxViewHeight = 1026

	kMaxRoomObs  = 24 // objects per room
	kRoomIsEmpty = -1 // GetNeighborRoomNumber's "no room there"
)

// The nine local room slots. The order is not arbitrary: DrawLocale walks the
// far rooms first and the central room last so that objects overhanging a room
// boundary are overdrawn by the room they belong to.
const (
	kCentralRoom = iota
	kNorthRoom
	kNorthEastRoom
	kEastRoom
	kSouthEastRoom
	kSouthRoom
	kSouthWestRoom
	kWestRoom
	kNorthWestRoom
)

// The three fallback backgrounds an off-map room is drawn with, chosen by
// elevation (GliderDefines.h:238-242, used at RoomGraphics.c:209-226).
const (
	kDirt   = 2011
	kMeadow = 2012
	kSky    = 2015
)

// kSupportPictID is the horizontal wooden beam between floors, 512x44
// (StructuresInit2.c:99).
const kSupportPictID = 1999

// kManholeThruFloor is the only art in the room path that goes through
// LoadScaledGraphic (RoomGraphics.c:16). In practice it never scales: see
// DrawFloorSupport.
const kManholeThruFloor = 3957

// Resource limits that are visible on screen rather than merely internal.
// Exceeding either one silently drops art, and both are reproduced here for that
// reason -- a room that is over the limit in the original must be over it here.
const (
	// kMaxSavedMaps caps how many background patches can be stashed for later
	// restoration. BackUpToSavedMap returns -1 when full, and every caller
	// gates its draw on the result, so the 25th object of the affected kinds
	// simply does not appear.
	kMaxSavedMaps = 24

	// kMaxGrease caps grease spills, and each one also consumes a savedMaps
	// slot, so the two limits interact.
	kMaxGrease = 16

	// kMaxTempManholes caps the manholes remembered for the floor-support pass.
	kMaxTempManholes = 8
)

// View is the screen-derived geometry: where the rooms land and how big the
// offscreen maps are.
type View struct {
	// Screen is the device rect, origin at (0,0).
	Screen Rect

	// House is the drawable area: the screen less the scoreboard, clamped to
	// the maximum view. Both offscreen maps are this size, cornered at (0,0).
	House Rect

	// OriginH and OriginV centre the central room on the screen. They are
	// computed from Screen, not from House, so the scoreboard does not shift
	// the rooms up -- the room block is centred on the whole display and the
	// scoreboard overlays its bottom 20 pixels.
	OriginH, OriginV int16

	// LocalRoomsDest is where each of the nine local rooms is drawn.
	LocalRoomsDest [9]Rect

	// WorkRect and BackRect are the bounds of the two offscreen maps. They are
	// House zero-cornered, hence equal to each other.
	WorkRect, BackRect Rect

	// JustRoomsRect is `justRoomsRect` (Play.c:48), set at
	// StructuresInit2.c:153-154 to ZeroRectCorner(houseRect) -- so at 640x480 it
	// is numerically identical to WorkRect and BackRect.
	//
	// It is a separate field because it is a separate thing: it is the rect the
	// screen blit is clamped and dumped against (Render.c:70-77, Play.c:172), and
	// the one the update event repaints (Play.c:403). The three agreeing at this
	// resolution is arithmetic, not design, and collapsing them would hide which
	// of the three a given call site meant.
	JustRoomsRect Rect

	// SuppRect is the floor-support beam's bounds, 512x44.
	SuppRect Rect

	// The scoreboard's construction-time geometry (StructuresInit.c:59-163).
	//
	// All twenty-three rects live here, on the view, because every one of them is a
	// function of the screen's size and nothing else -- InitScoreboardMap computes them
	// once at launch from houseRect and never touches them again. The seven that
	// AdjustScoreboardHeight *moves* are copied onto World at the start of a game and
	// moved there; these stay put, so the port's AdjustScoreboardHeight can assign rather
	// than accumulate. See game.World.BoardDestRect and game.AdjustScoreboardHeight.
	//
	// The Src rects are all origin-cornered and are therefore each just a size. They are
	// still named and stored, rather than derived at each blit from the surface's bounds,
	// because the C passes them to CopyBits by name and a reader diffing the two wants to
	// see the same argument.
	BoardSrc  Rect // boardSrcRect: the whole band, screen-wide by 20
	BoardDest Rect // boardDestRect: rows -20..0, and see World for why the port moves it

	BoardTSrc, BoardTDest Rect // 256x12, the room title, into BoardSrc at (137,5)
	BoardGSrc, BoardGDest Rect // 20x10, the glider count, into BoardSrc at (526,5)
	BoardPSrc, BoardPDest Rect // 64x10, the score, into BoardSrc at (570,5)

	// BoardGQDest and BoardPQDest are where the glider count and the score go when they
	// are refreshed *on their own* -- straight to the screen, bypassing the board map.
	// They are the two panels' board-map positions translated up by the band's height,
	// which is exactly BoardDest's own translation, so all three move together.
	BoardGQDest, BoardPQDest Rect

	// BadgeSrc is the four badges' shared sheet, 32x66: two columns by four rows.
	// BadgesBlank indexes the left column and BadgesBadges the right, so blank and lit
	// are the same cell sixteen pixels apart. The rows are not all the same height --
	// foil and bands are 16, battery and helium 17 -- which is why each cell is stored
	// rather than computed from an index.
	BadgeSrc                              Rect
	BadgesBlank, BadgesBadges, BadgesDest [4]Rect
}

// NewView reproduces InterfaceInit.c:196-218 for a screen of the given size.
func NewView(screenW, screenH int16) *View {
	v := &View{Screen: SetRect(0, 0, screenW, screenH)}

	v.House = v.Screen
	v.House.Bottom -= kScoreboardTall
	if v.House.Right > kMaxViewWidth {
		v.House.Right = kMaxViewWidth
	}
	if v.House.Bottom > kMaxViewHeight {
		v.House.Bottom = kMaxViewHeight
	}

	v.OriginH = (v.Screen.Wide() - kRoomWide) / 2
	v.OriginV = (v.Screen.Tall() - kTileHigh) / 2

	for i := 0; i < 9; i++ {
		v.LocalRoomsDest[i] = Offset(SetRect(0, 0, kRoomWide, kTileHigh), v.OriginH, v.OriginV)
	}
	v.LocalRoomsDest[kNorthRoom] = Offset(v.LocalRoomsDest[kNorthRoom], 0, -kVertLocalOffset)
	v.LocalRoomsDest[kNorthEastRoom] = Offset(v.LocalRoomsDest[kNorthEastRoom], kRoomWide, -kVertLocalOffset)
	v.LocalRoomsDest[kEastRoom] = Offset(v.LocalRoomsDest[kEastRoom], kRoomWide, 0)
	v.LocalRoomsDest[kSouthEastRoom] = Offset(v.LocalRoomsDest[kSouthEastRoom], kRoomWide, kVertLocalOffset)
	v.LocalRoomsDest[kSouthRoom] = Offset(v.LocalRoomsDest[kSouthRoom], 0, kVertLocalOffset)
	v.LocalRoomsDest[kSouthWestRoom] = Offset(v.LocalRoomsDest[kSouthWestRoom], -kRoomWide, kVertLocalOffset)
	v.LocalRoomsDest[kWestRoom] = Offset(v.LocalRoomsDest[kWestRoom], -kRoomWide, 0)
	v.LocalRoomsDest[kNorthWestRoom] = Offset(v.LocalRoomsDest[kNorthWestRoom], -kRoomWide, -kVertLocalOffset)

	v.WorkRect = ZeroCorner(v.House)
	v.BackRect = ZeroCorner(v.House)
	v.JustRoomsRect = ZeroCorner(v.House)
	v.SuppRect = SetRect(0, 0, kRoomWide, kFloorSupportTall)
	v.initScoreboard()
	return v
}

// DefaultView is the 640x480 screen Glider PRO was designed around: a 640x460
// house rect, the central room at (64,79) and exactly one room visible.
func DefaultView() *View { return NewView(640, 480) }

// VerticalRoomOffset is ObjectRects.c:1067. It converts a neighbour slot into
// the vertical shift that slot's room art needs, and is separate from
// OffsetRectRoomRelative because a handful of draw helpers apply the vertical
// part on its own.
func VerticalRoomOffset(neighbor int) int16 {
	switch neighbor {
	case kNorthRoom, kNorthEastRoom, kNorthWestRoom:
		return -kVertLocalOffset
	case kSouthEastRoom, kSouthRoom, kSouthWestRoom:
		return kVertLocalOffset
	}
	return 0
}

// OffsetRectRoomRelative is ObjectRects.c:1092: move a room-local rect into
// screen coordinates for one of the nine local rooms.
func (v *View) OffsetRectRoomRelative(r Rect, neighbor int) Rect {
	r = Offset(r, v.OriginH, v.OriginV)
	switch neighbor {
	case kNorthRoom:
		return Offset(r, 0, -kVertLocalOffset)
	case kNorthEastRoom:
		return Offset(r, kRoomWide, -kVertLocalOffset)
	case kEastRoom:
		return Offset(r, kRoomWide, 0)
	case kSouthEastRoom:
		return Offset(r, kRoomWide, kVertLocalOffset)
	case kSouthRoom:
		return Offset(r, 0, kVertLocalOffset)
	case kSouthWestRoom:
		return Offset(r, -kRoomWide, kVertLocalOffset)
	case kWestRoom:
		return Offset(r, -kRoomWide, 0)
	case kNorthWestRoom:
		return Offset(r, -kRoomWide, -kVertLocalOffset)
	}
	return r
}
