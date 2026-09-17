package game

import "github.com/bwenstar/gliderGo/internal/render"

// This file is ObjectRects.c:277-1063: AddActiveRect and CreateActiveRects.
//
// It is the largest single switch in the game and the least documented part of the
// original. 93 of the 117 object types produce at least one hot spot here, and the
// rects are mostly *not* the objects' artwork -- a floor vent's is a four-pixel
// column reaching up to the ceiling, a microwave's second rect is a beam from the
// top of the room down to the oven, a manhole's is a strip pinned to the floor
// plane regardless of where the manhole was placed. Fifteen of the types whose
// behaviour is nowhere described in the original's own comments are lethal
// kDissolveIt solids, kTable and kCounter among them, so a port that skipped the
// undocumented cases would ship a house full of furniture the glider flies through.
//
// Three conventions run through the whole switch and are worth stating once:
//
//   - hotSpotNumber starts at -1 and is overwritten by every AddActiveRect, so an
//     object that makes several rects reports only its **last**. See
//     MasterObject.HotNum.
//   - `bounds` is reused across the rects of one object without being re-zeroed,
//     so the second rect is usually a small edit of the first. The microwave's
//     radiation box is the clearest case: it is the oven's own box with the top
//     dragged to y=0.
//   - isOn is not always the object's state. Every flame, and both fans' blade
//     boxes, pass a literal true, which is why a switched-off fan still cuts the
//     glider in half and why there is no way to extinguish a candle.

// AddActiveRect is ObjectRects.c:277-292: append a rect to the collision table and
// return its index, or -1 when the table is full.
//
// The -1 is the whole error handling. Nothing warns and nothing grows; the 57th hot
// spot in a room is simply inert, and the object that tried to make it gets a HotNum
// of -1 and can never be switched. The bound is reachable in principle -- 24 slots
// x up to 3 rects each is 72 against a cap of 56 -- though no shipped room is known
// to hit it.
func (w *World) AddActiveRect(bounds Rect, action, who int16, isOn, doScrutinize bool) int16 {
	if len(w.R.Hot) >= MaxHotSpots {
		return -1
	}
	w.R.Hot = append(w.R.Hot, HotObject{
		Bounds:       bounds,
		Action:       action,
		Who:          who,
		IsOn:         isOn,
		StillOver:    false,
		DoScrutinize: doScrutinize,
	})
	return int16(len(w.R.Hot) - 1)
}

// CreateActiveRects is ObjectRects.c:296-1063: make the hot spots for one object of
// the central room and return the index of the last one, or -1 for the 24 types
// that make none.
//
// `who` is an index into Room.Master, not an object slot: the entry has to be in
// the graph already, because this reads its TheObject copy. ListOneRoomsObjects
// appends first and calls this second for exactly that reason.
func (w *World) CreateActiveRects(who int16) int16 {
	if w.badIndex(devMasterObject, int(who), len(w.R.Master)) {
		return -1
	}
	obj := w.R.Master[who].TheObject
	hot := int16(-1)
	var bounds Rect

	// srcRect is srcRects[what]: the object's rect in the sprite atlas, which for
	// most types is where its *size* comes from even when the position does not.
	srcRect := func(what int16) Rect { return render.SrcRect(what) }

	switch obj.What {
	// ---------------------------------------------------------------- blowers
	// The four plain updraught vents are all the same shape: a column
	// FloorColumnWide (4) pixels wide, centred on the artwork, running from
	// `distance` pixels above the object down to its top edge. `distance` is
	// therefore a reach, and the column is deliberately narrow -- riding a vent
	// takes aim.
	case FloorVent, FloorBlower, SewerGrate, GrecoVent, SewerBlower:
		a := obj.Blower()
		bounds = render.SetRect(0, -a.Distance, FloorColumnWide, 0)
		bounds = render.Offset(bounds, render.HalfWide(srcRect(obj.What))-FloorColumnWide/2, 0)
		bounds = render.Offset(bounds, a.TopLeft.H, a.TopLeft.V)
		hot = w.AddActiveRect(bounds, LiftIt, who, a.State != 0, false)

	// The two downdraughts mirror them and are six times wider:
	// CeilingColumnWide is 24, so a ceiling blower is much harder to escape than
	// a floor vent is to catch.
	case CeilingVent, CeilingBlower:
		a := obj.Blower()
		bounds = render.SetRect(0, 0, CeilingColumnWide, a.Distance)
		bounds = render.Offset(bounds, render.HalfWide(srcRect(obj.What))-CeilingColumnWide/2, 0)
		bounds = render.Offset(bounds, a.TopLeft.H, a.TopLeft.V)
		hot = w.AddActiveRect(bounds, DropIt, who, a.State != 0, false)

	// Both fans make two rects: a lethal blade box that is always on, and a push
	// column that obeys the switch. The blade boxes are hand-measured 13x43
	// literals at different offsets, which is the only difference between the two
	// cases besides the direction.
	//
	// Because HotNum keeps the last rect, switching a fan switches only its push
	// column -- the blades stay lethal on a fan that is off. That reads like an
	// oversight and is the behaviour: a stopped fan you can fly past, but not
	// through.
	case LeftFan:
		a := obj.Blower()
		bounds = render.SetRect(0, 0, 13, 43)
		bounds = render.Offset(bounds, a.TopLeft.H+16, a.TopLeft.V+12)
		hot = w.AddActiveRect(bounds, DissolveIt, who, true, true)

		bounds = render.SetRect(0, 0, a.Distance, FanColumnThick)
		bounds = render.Offset(bounds, -a.Distance, FanColumnDown)
		bounds = render.Offset(bounds, a.TopLeft.H, a.TopLeft.V)
		hot = w.AddActiveRect(bounds, PushItLeft, who, a.State != 0, false)

	case RightFan:
		a := obj.Blower()
		bounds = render.SetRect(0, 0, 13, 43)
		bounds = render.Offset(bounds, a.TopLeft.H+6, a.TopLeft.V+12)
		hot = w.AddActiveRect(bounds, DissolveIt, who, true, true)

		bounds = render.SetRect(0, 0, a.Distance, FanColumnThick)
		bounds = render.Offset(bounds, srcRect(RightFan).Wide(), FanColumnDown)
		bounds = render.Offset(bounds, a.TopLeft.H, a.TopLeft.V)
		hot = w.AddActiveRect(bounds, PushItRight, who, a.State != 0, false)

	// The five flames all follow one shape: a thermal column like a vent's, split
	// into a lift part and a burn part, plus a hand-measured box over the flame
	// itself that dissolves the glider outright.
	//
	// The split is the interesting part and is done in place on one rect. If the
	// column is taller than DeadlyFlameHeight (24) the top of it lifts and the
	// bottom 22 pixels burn -- 22, not 24, because the burn rect's top is set to
	// `bottom - 24 + 2`. If the column is 24 or shorter there is no lift at all
	// and the whole thing burns. So a candle with a short reach is pure hazard and
	// a candle with a long one is a thermal you can ride if you stay high.
	//
	// Every rect here passes isOn = true. A flame is not switchable -- see
	// SetObjectState, where the five of them share a `changed = false` case.
	case Taper, Candle, Stubby, Tiki, BBQ:
		a := obj.Blower()

		// The column. BBQ is the one that differs: its bottom is 8 rather than 0,
		// so its thermal starts eight pixels *below* the object's top edge.
		colBottom := int16(0)
		if obj.What == BBQ {
			colBottom = 8
		}
		bounds = render.SetRect(0, -a.Distance, FloorColumnWide, colBottom)

		// The horizontal centring, which is per-type and not derivable: the candle
		// shifts its whole column two pixels left, and the stubby candle's centre
		// is one pixel left of half its artwork.
		hOff := render.HalfWide(srcRect(obj.What)) - FloorColumnWide/2
		vOff := int16(0)
		switch obj.What {
		case Candle:
			bounds = render.Offset(bounds, hOff, 0)
			hOff, vOff = a.TopLeft.H-2, a.TopLeft.V
		case Stubby:
			bounds = render.Offset(bounds, hOff-1, 0)
			hOff, vOff = a.TopLeft.H, a.TopLeft.V
		default:
			bounds = render.Offset(bounds, hOff, 0)
			hOff, vOff = a.TopLeft.H, a.TopLeft.V
		}
		bounds = render.Offset(bounds, hOff, vOff)

		if bounds.Bottom-bounds.Top > DeadlyFlameHeight {
			bounds.Bottom -= DeadlyFlameHeight
			hot = w.AddActiveRect(bounds, LiftIt, who, true, false)
			bounds.Bottom += DeadlyFlameHeight
			bounds.Top = bounds.Bottom - DeadlyFlameHeight + 2
			hot = w.AddActiveRect(bounds, BurnIt, who, true, false)
		} else {
			hot = w.AddActiveRect(bounds, BurnIt, who, true, false)
		}

		// The dissolve box over the flame. Five hand-measured literals; there is
		// no formula relating them to the artwork.
		var dw, dh, dx, dy int16
		switch obj.What {
		case Taper:
			dw, dh, dx, dy = 7, 48, 6, 11
		case Candle:
			dw, dh, dx, dy = 8, 20, 9, 11
		case Stubby:
			dw, dh, dx, dy = 15, 26, 1, 11
		case Tiki:
			dw, dh, dx, dy = 15, 14, 6, 6
		case BBQ:
			dw, dh, dx, dy = 52, 17, 6, 8
		}
		bounds = render.SetRect(0, 0, dw, dh)
		bounds = render.Offset(bounds, a.TopLeft.H+dx, a.TopLeft.V+dy)
		hot = w.AddActiveRect(bounds, DissolveIt, who, true, true)

	// An invisible blower reads its direction from the low nibble of `vector`, and
	// the four cases are the only ones handled: a vector of 0, or with more than
	// one bit set, produces **no hot spot at all** and the object does nothing.
	//
	// Note the +24 on every reach and the compensating offsets. The object is
	// nominally 24x24 and the rect is grown to include it, then pushed back so it
	// starts at the right edge -- which is why the up case offsets by +24
	// vertically and the left case by -distance horizontally.
	case InvisBlower:
		a := obj.Blower()
		switch a.Vector & 0x0F {
		case 1: // up
			bounds = render.SetRect(0, -a.Distance-24, FloorColumnWide, 0)
			bounds = render.Offset(bounds, 12-FloorColumnWide/2, 24)
			bounds = render.Offset(bounds, a.TopLeft.H, a.TopLeft.V)
			hot = w.AddActiveRect(bounds, LiftIt, who, a.State != 0, false)
		case 2: // right
			bounds = render.SetRect(0, 0, a.Distance+24, FanColumnThick)
			bounds = render.Offset(bounds, 0, 12-FanColumnThick/2)
			bounds = render.Offset(bounds, a.TopLeft.H, a.TopLeft.V)
			hot = w.AddActiveRect(bounds, PushItRight, who, a.State != 0, false)
		case 4: // down
			bounds = render.SetRect(0, 0, FloorColumnWide, a.Distance+24)
			bounds = render.Offset(bounds, 12-FloorColumnWide/2, 0)
			bounds = render.Offset(bounds, a.TopLeft.H, a.TopLeft.V)
			hot = w.AddActiveRect(bounds, DropIt, who, a.State != 0, false)
		case 8: // left
			// Note the asymmetry with case 2: the rect is grown by 24 but pushed
			// back by only `distance`, so a leftward invisible blower reaches 24
			// pixels further left than a rightward one reaches right.
			bounds = render.SetRect(0, 0, a.Distance+24, FanColumnThick)
			bounds = render.Offset(bounds, -a.Distance, 12-FanColumnThick/2)
			bounds = render.Offset(bounds, a.TopLeft.H, a.TopLeft.V)
			hot = w.AddActiveRect(bounds, PushItLeft, who, a.State != 0, false)
		}

	// A lift area is the general case: an arbitrary box, `distance` wide and
	// `tall * 2` high, pushing in one of the four directions. Same nibble switch
	// and same silence on an unhandled vector.
	case LiftArea:
		a := obj.Blower()
		bounds = render.SetRect(0, 0, a.Distance, int16(a.Tall)*2)
		bounds = render.Offset(bounds, a.TopLeft.H, a.TopLeft.V)
		switch a.Vector & 0x0F {
		case 1:
			hot = w.AddActiveRect(bounds, LiftIt, who, a.State != 0, false)
		case 2:
			hot = w.AddActiveRect(bounds, PushItRight, who, a.State != 0, false)
		case 4:
			hot = w.AddActiveRect(bounds, DropIt, who, a.State != 0, false)
		case 8:
			hot = w.AddActiveRect(bounds, PushItLeft, who, a.State != 0, false)
		}

	// -------------------------------------------------------------- furniture
	// Eleven solids that kill on contact, using their stored bounds unchanged.
	// None of them is described anywhere in the original's comments and all of
	// them are lethal: a table, a counter and an invisible obstacle end the life
	// of anything that touches them, with the 5px scrutinize inset as the only
	// concession.
	case Table, Shelf, Cabinet, FilingCabinet, WasteBasket, MilkCrate, Counter,
		Dresser, DeckTable, Trunk, InvisObstacle:
		bounds = obj.Furniture().Bounds
		hot = w.AddActiveRect(bounds, DissolveIt, who, true, true)

	// A shelf of books is two pixels narrower than it looks, so the glider can
	// clip the right edge of the artwork and live.
	case Books:
		bounds = obj.Furniture().Bounds
		bounds.Right -= 2
		hot = w.AddActiveRect(bounds, DissolveIt, who, true, true)

	// A manhole is not a solid: it is a hole in the floor, and its rect is pinned
	// to the floor plane rather than to the object. Top is FloorLimit - 1 and
	// bottom is TileHigh whatever the stored bounds say, and the sides are inset
	// by a whole glider width plus 3 on each side -- so the glider has to be
	// entirely over the hole, not merely touching it, to fall through.
	case Manhole:
		bounds = obj.Furniture().Bounds
		bounds.Left += GliderWide + 3
		bounds.Right -= GliderWide + 3
		bounds.Top = FloorLimit - 1
		bounds.Bottom = TileHigh
		hot = w.AddActiveRect(bounds, IgnoreGround, who, true, false)

	case InvisBounce:
		bounds = obj.Furniture().Bounds
		hot = w.AddActiveRect(bounds, BounceIt, who, true, true)

	// A stool is lethal only along its seat: inset one pixel all round, then the
	// bottom pulled up to top + StoolThick (25). The legs are not solid.
	case Stool:
		bounds = obj.Furniture().Bounds
		bounds = render.Inset(bounds, 1, 1)
		bounds.Bottom = bounds.Top + StoolThick
		hot = w.AddActiveRect(bounds, DissolveIt, who, true, true)

	// ---------------------------------------------------------------- bonuses
	// Eleven prizes, each a reward rect the size of its artwork whose isOn is the
	// object's own state -- so a collected prize's rect stays in the table and is
	// simply switched off. kSparkle is absent from this list and kSlider and the
	// two grease jars have their own cases.
	case RedClock, BlueClock, YellowClock, Cuckoo, Paper, Battery, Bands, Foil,
		InvisBonus, Star, Helium:
		c := obj.Bonus()
		bounds = render.ZeroCorner(srcRect(obj.What))
		bounds = render.Offset(bounds, c.TopLeft.H, c.TopLeft.V)
		hot = w.AddActiveRect(bounds, RewardIt, who, c.State != 0, false)

	// A grease jar is two objects in one. Upright (state set) it is a prize. Once
	// knocked over it becomes a slick: a two-pixel-high strip `length - 5` long,
	// starting 31 pixels right of the jar and 27 down, which is where the spilled
	// artwork is drawn. Note that the slide rect is created with isOn = true
	// unconditionally -- a spilled jar is always slippery.
	//
	// This is why kGreaseRt and kGreaseLf are absent from IsThisValid's
	// state-gated list: gating them would make a spilled jar stop existing.
	case GreaseRt:
		c := obj.Bonus()
		if c.State != 0 {
			bounds = render.ZeroCorner(srcRect(obj.What))
			bounds = render.Offset(bounds, c.TopLeft.H, c.TopLeft.V)
			hot = w.AddActiveRect(bounds, RewardIt, who, true, false)
		} else {
			bounds = render.SetRect(0, -2, c.Length-5, 0)
			bounds = render.Offset(bounds, 32-1, 27)
			bounds = render.Offset(bounds, c.TopLeft.H, c.TopLeft.V)
			hot = w.AddActiveRect(bounds, SlideIt, who, true, false)
		}

	case GreaseLf:
		c := obj.Bonus()
		if c.State != 0 {
			bounds = render.ZeroCorner(srcRect(obj.What))
			bounds = render.Offset(bounds, c.TopLeft.H, c.TopLeft.V)
			hot = w.AddActiveRect(bounds, RewardIt, who, true, false)
		} else {
			bounds = render.SetRect(-c.Length+5, -2, 0, 0)
			bounds = render.Offset(bounds, 1, 27)
			bounds = render.Offset(bounds, c.TopLeft.H, c.TopLeft.V)
			hot = w.AddActiveRect(bounds, SlideIt, who, true, false)
		}

	// A sparkle computes a rect and then never uses it: there is no AddActiveRect
	// in the C's case, so this returns -1. The dead computation is left in place
	// because it is what the original does and because its absence is the point --
	// a sparkle is decoration, and the only thing its bonus state controls is
	// whether IsThisValid lets it draw.
	case Sparkle:
		c := obj.Bonus()
		bounds = render.ZeroCorner(srcRect(obj.What))
		bounds = render.Offset(bounds, c.TopLeft.H, c.TopLeft.V)
		_ = bounds

	// A slider is a bare slide strip, 16 pixels high and `length` long. It shares
	// bonusType with the prizes but is not one, and it is the type SetObjectState
	// forgets: see the kSlider note there.
	case Slider:
		c := obj.Bonus()
		bounds = render.SetRect(0, 0, c.Length, 16)
		bounds = render.Offset(bounds, c.TopLeft.H, c.TopLeft.V)
		hot = w.AddActiveRect(bounds, SlideIt, who, true, false)

	// ------------------------------------------------------------- transports
	// A staircase's trigger box is a 112x32 literal at the object's anchor: much
	// smaller than the artwork, and at the top of it.
	case UpStairs:
		d := obj.Transport()
		bounds = render.SetRect(0, 0, 112, 32)
		bounds = render.Offset(bounds, d.TopLeft.H, d.TopLeft.V)
		hot = w.AddActiveRect(bounds, MoveItUp, who, true, false)

	// The down staircase is built backwards from its own right edge and 170 pixels
	// down, which puts its trigger box at the bottom of the flight. The artwork
	// width is read out of the atlas, so this is the one hot spot whose position
	// depends on a sprite's dimensions rather than on the object's data.
	case DownStairs:
		d := obj.Transport()
		bounds = render.SetRect(-80, -56, 0, 0)
		bounds = render.Offset(bounds, srcRect(DownStairs).Right, 170)
		bounds = render.Offset(bounds, d.TopLeft.H, d.TopLeft.V)
		hot = w.AddActiveRect(bounds, MoveItDown, who, true, false)

	// The four link transports all gate on `who != 255`: an unlinked mailbox, duct
	// or transporter is inert rather than fatal. Note the sentinel is 255 and not
	// -1, because the field is a byte; see GetObjectLinked.
	case MailboxLf:
		d := obj.Transport()
		if d.Who != 255 {
			bounds = render.SetRect(-72, 0, 0, 40)
			bounds = render.Offset(bounds, 30, 16)
			bounds = render.Offset(bounds, d.TopLeft.H, d.TopLeft.V)
			hot = w.AddActiveRect(bounds, MailItLeft, who, true, false)
		}

	case MailboxRt:
		d := obj.Transport()
		if d.Who != 255 {
			bounds = render.SetRect(0, 0, 72, 40)
			bounds = render.Offset(bounds, 79, 16)
			bounds = render.Offset(bounds, d.TopLeft.H, d.TopLeft.V)
			hot = w.AddActiveRect(bounds, MailItRight, who, true, false)
		}

	// A floor duct's box hangs below the grating -- 48 pixels of it, offset down by
	// the artwork's own height -- so the glider is caught after it has passed the
	// grate, not on it.
	case FloorTrans:
		d := obj.Transport()
		if d.Who != 255 {
			bounds = render.SetRect(0, -48, 76, 0)
			bounds = render.Offset(bounds, -8, srcRect(FloorTrans).Tall())
			bounds = render.Offset(bounds, d.TopLeft.H, d.TopLeft.V)
			hot = w.AddActiveRect(bounds, DuctItDown, who, true, false)
		}

	case CeilingTrans:
		d := obj.Transport()
		if d.Who != 255 {
			bounds = render.SetRect(0, 0, 76, 48)
			bounds = render.Offset(bounds, -8, 0)
			bounds = render.Offset(bounds, d.TopLeft.H, d.TopLeft.V)
			hot = w.AddActiveRect(bounds, DuctItUp, who, true, false)
		}

	// The eight doors and windows are wall holes: a 16-wide strip that tells the
	// escape check to ignore the wall it is in. They are all the same two shapes
	// with different offsets, and the pairing of type to side is not what the names
	// suggest -- kDoorExRt and kWindowExRt suppress the *right* wall while sitting
	// at offset 0, whereas kDoorInRt sits at offset 128. "In" and "Ex" are interior
	// and exterior artwork, not left and right.
	case DoorInLf:
		d := obj.Transport()
		bounds = render.SetRect(0, 0, 16, 240)
		bounds = render.Offset(bounds, 0, 52)
		bounds = render.Offset(bounds, d.TopLeft.H, d.TopLeft.V)
		hot = w.AddActiveRect(bounds, IgnoreLeftWall, who, true, false)

	case DoorInRt:
		d := obj.Transport()
		bounds = render.SetRect(0, 0, 16, 240)
		bounds = render.Offset(bounds, 128, 52)
		bounds = render.Offset(bounds, d.TopLeft.H, d.TopLeft.V)
		hot = w.AddActiveRect(bounds, IgnoreRightWall, who, true, false)

	case DoorExRt:
		d := obj.Transport()
		bounds = render.SetRect(0, 0, 16, 240)
		bounds = render.Offset(bounds, 0, 52)
		bounds = render.Offset(bounds, d.TopLeft.H, d.TopLeft.V)
		hot = w.AddActiveRect(bounds, IgnoreRightWall, who, true, false)

	case DoorExLf:
		d := obj.Transport()
		bounds = render.SetRect(0, 0, 16, 240)
		bounds = render.Offset(bounds, 0, 52)
		bounds = render.Offset(bounds, d.TopLeft.H, d.TopLeft.V)
		hot = w.AddActiveRect(bounds, IgnoreLeftWall, who, true, false)

	case WindowInLf:
		d := obj.Transport()
		bounds = render.SetRect(0, 0, 16, 44)
		bounds = render.Offset(bounds, 0, 96)
		bounds = render.Offset(bounds, d.TopLeft.H, d.TopLeft.V)
		hot = w.AddActiveRect(bounds, IgnoreLeftWall, who, true, false)

	// The only one of the eight with a non-zero horizontal offset besides
	// kDoorInRt, and it is 4 rather than 128 -- so an interior right window's hole
	// is four pixels right of its anchor.
	case WindowInRt:
		d := obj.Transport()
		bounds = render.SetRect(0, 0, 16, 44)
		bounds = render.Offset(bounds, 4, 96)
		bounds = render.Offset(bounds, d.TopLeft.H, d.TopLeft.V)
		hot = w.AddActiveRect(bounds, IgnoreRightWall, who, true, false)

	case WindowExRt:
		d := obj.Transport()
		bounds = render.SetRect(0, 0, 16, 44)
		bounds = render.Offset(bounds, 0, 96)
		bounds = render.Offset(bounds, d.TopLeft.H, d.TopLeft.V)
		hot = w.AddActiveRect(bounds, IgnoreRightWall, who, true, false)

	case WindowExLf:
		d := obj.Transport()
		bounds = render.SetRect(0, 0, 16, 44)
		bounds = render.Offset(bounds, 0, 96)
		bounds = render.Offset(bounds, d.TopLeft.H, d.TopLeft.V)
		hot = w.AddActiveRect(bounds, IgnoreLeftWall, who, true, false)

	// An invisible transporter starts from a 64x32 default and then has both its
	// bottom and its right overwritten from the object's own tall and wide -- so
	// the 64x32 only survives if tall and wide are 32 and 0. Note `right +=` and
	// `bottom =`: one is a delta and the other an assignment.
	case InvisTrans:
		d := obj.Transport()
		if d.Who != 255 {
			bounds = render.SetRect(0, 0, 64, 32)
			bounds = render.Offset(bounds, d.TopLeft.H, d.TopLeft.V)
			bounds.Bottom = bounds.Top + d.Tall
			bounds.Right += int16(d.Wide)
			hot = w.AddActiveRect(bounds, TransportIt, who, true, false)
		}

	// A deluxe transporter packs its size into the two bytes of `tall` and scales
	// by four, and packs its on-state into the low nibble of `wide`. So `wide` is
	// not a width here and `tall` is not a height: the width is tall's high byte
	// and the height is tall's low byte, both times four.
	case DeluxeTrans:
		d := obj.Transport()
		if d.Who != 255 {
			// (tall & 0xFF00) >> 8 in the C, on a signed short. The mask and shift
			// happen after the integer promotion, so the result is the top byte read
			// as 0..255 -- a `tall` of -1 gives 255, not -1. int16(uint16(x) >> 8) is
			// that exactly; `d.Tall >> 8` alone would sign-extend and give -1.
			wide := int16(uint16(d.Tall) >> 8)
			tall := d.Tall & 0x00FF
			bounds = render.SetRect(0, 0, wide*4, tall*4)
			bounds = render.Offset(bounds, d.TopLeft.H, d.TopLeft.V)
			// d.Wide, not the local `wide`: the state nibble is in the byte, and
			// the local shadows nothing but reads like it should be used here.
			hot = w.AddActiveRect(bounds, TransportIt, who, d.Wide&0x0F != 0, false)
		}

	// ---------------------------------------------------------------- switches
	// Eight kinds, one rect the size of the artwork, and the action decided by
	// whether it is a trigger or a switch. Both branches require a link: an
	// unwired switch makes no rect and does nothing.
	//
	// kKnifeSwitch is in this list, which is worth noting because it is *not* in
	// SetObjectState's -- so a knife switch can be pressed and its target cannot
	// be changed by it. See the kKnifeSwitch note in setstate.go.
	case LightSwitch, MachineSwitch, Thermostat, PowerSwitch, KnifeSwitch,
		InvisSwitch, Trigger, LgTrigger:
		e := obj.Switch()
		bounds = render.ZeroCorner(srcRect(obj.What))
		bounds = render.Offset(bounds, e.TopLeft.H, e.TopLeft.V)
		if e.Where != -1 {
			action := SwitchIt
			if obj.What == Trigger || obj.What == LgTrigger {
				action = TriggerIt
			}
			hot = w.AddActiveRect(bounds, action, who, true, false)
		}

	// A sound trigger is not a trigger. Its `where` is a **sound resource id**, not
	// a packed floor/suite, and the hot spot exists only if that sound loads.
	//
	// LoadTriggerSound has a side effect that makes this a one-per-room limit: it
	// loads into a single reserved slot and fails outright if the slot is already
	// occupied. DrawLocale frees the slot at the top of every room change
	// (RoomGraphics.c:58), so a room's *first* sound trigger works and a second one
	// silently gets no rect. That is reproduced here by Room.TriggerSoundHeld.
	case SoundTrigger:
		e := obj.Switch()
		bounds = render.SetRect(0, 0, 48, 48)
		bounds = render.Offset(bounds, e.TopLeft.H, e.TopLeft.V)
		if w.loadTriggerSound(e.Where) {
			hot = w.AddActiveRect(bounds, SoundIt, who, true, false)
		}

	// ------------------------------------------------------------------ lights
	// All eight lights make no hot spot. A light is drawn, counted by
	// GetNumberOfLights and switched by SetObjectState, and never touched.
	case CeilingLight, LightBulb, TableLamp, HipLamp, DecoLamp, Flourescent,
		TrackLight, InvisLight:

	// -------------------------------------------------------------- appliances
	// The shredder's rect is the one built from an *un-zeroed* atlas rect: the
	// height is forced to ShredderActiveHigh (40) and the width grown by 48 while
	// the rect still carries its atlas position, and only then is the corner
	// zeroed. Then it is moved up and left by (24, 36), so the catchment sits
	// above and around the mouth rather than on it.
	case Shredder:
		g := obj.Appliance()
		bounds = srcRect(obj.What)
		bounds.Bottom = bounds.Top + ShredderActiveHigh
		bounds.Right += 48
		bounds = render.ZeroCorner(bounds)
		bounds = render.Offset(bounds, g.TopLeft.H, g.TopLeft.V)
		bounds = render.Offset(bounds, -24, -36)
		hot = w.AddActiveRect(bounds, ShredIt, who, g.State != 0, true)

	// A guitar is harmless: an 8x96 strip over the strings that plays a chord.
	// It is the only kStrumIt in the game.
	case Guitar:
		g := obj.Appliance()
		bounds = render.SetRect(0, 0, 8, 96)
		bounds = render.Offset(bounds, g.TopLeft.H+34, g.TopLeft.V+32)
		hot = w.AddActiveRect(bounds, StrumIt, who, true, false)

	// An outlet's rect does nothing on contact -- kIgnoreIt is action 0 -- but it
	// still carries the object's state, because the *rect* is what the state
	// change is published through for the sparking animation.
	case Outlet:
		g := obj.Appliance()
		bounds = render.ZeroCorner(srcRect(obj.What))
		bounds = render.Offset(bounds, g.TopLeft.H, g.TopLeft.V)
		hot = w.AddActiveRect(bounds, IgnoreIt, who, g.State != 0, false)

	// A microwave is the two-rect case that is easiest to get wrong. The first
	// rect is the oven, lethal to touch. The second is built from the first by
	// collapsing it -- bottom = top, then top = 0 -- which makes a beam from the
	// very top of the room down to the oven's top edge, the full width of the
	// oven. So the hazard is the column of air above a running microwave, and it
	// reaches the ceiling however low the oven sits.
	case Microwave:
		g := obj.Appliance()
		bounds = render.ZeroCorner(srcRect(obj.What))
		bounds = render.Offset(bounds, g.TopLeft.H, g.TopLeft.V)
		hot = w.AddActiveRect(bounds, DissolveIt, who, true, true)
		bounds.Bottom = bounds.Top
		bounds.Top = 0
		hot = w.AddActiveRect(bounds, MicrowaveIt, who, true, true)

	// Nine more solids, lethal like the furniture and equally undocumented. Note
	// kStereo is here: a stereo is a solid object *and* the thing that toggles the
	// game's music flag when a switch points at it.
	case Toaster, MacPlus, TV, Coffee, VCR, Stereo, CinderBlock, FlowerBox, CDs:
		g := obj.Appliance()
		bounds = render.ZeroCorner(srcRect(obj.What))
		bounds = render.Offset(bounds, g.TopLeft.H, g.TopLeft.V)
		hot = w.AddActiveRect(bounds, DissolveIt, who, true, true)

	// A custom picture is pure decoration and makes no rect, which is the one
	// mercy in the appliance group: an author can paste arbitrary art into a room
	// without making it lethal.
	case CustomPict:

	// ---------------------------------------------------------------- enemies
	// Seven movers whose rect is kIgnoreIt. They are lethal, but not through the
	// hot-spot table: their collision is done by the dynamics pass against the
	// glider directly, and this rect exists only so the object has an entry.
	case Balloon, CopterLf, CopterRt, DartLf, DartRt, Ball, Drip:
		h := obj.Enemy()
		bounds = render.ZeroCorner(srcRect(obj.What))
		bounds = render.Offset(bounds, h.TopLeft.H, h.TopLeft.V)
		hot = w.AddActiveRect(bounds, IgnoreIt, who, true, false)

	// A fish is the exception: it kills through the hot-spot table like furniture.
	case Fish:
		h := obj.Enemy()
		bounds = render.ZeroCorner(srcRect(obj.What))
		bounds = render.Offset(bounds, h.TopLeft.H, h.TopLeft.V)
		hot = w.AddActiveRect(bounds, DissolveIt, who, true, true)

	// A cobweb's catchment is much larger than its artwork: inset by (-24, -10),
	// i.e. **grown** 24 pixels each side and 10 top and bottom. A web spanning a
	// corner catches a glider that never visibly touched it.
	case Cobweb:
		h := obj.Enemy()
		bounds = render.ZeroCorner(srcRect(obj.What))
		bounds = render.Offset(bounds, h.TopLeft.H, h.TopLeft.V)
		bounds = render.Inset(bounds, -24, -10)
		hot = w.AddActiveRect(bounds, WebIt, who, true, true)

	// ---------------------------------------------------------------- clutter
	// Fourteen decorations with no hot spot at all. kMirror is among them: a
	// mirror's effect is on the dirty-rect bookkeeping, not on collision.
	case Ozma, Mirror, Mousehole, Fireplace, Flower, WallWindow, Bear, Calendar,
		Vase1, Vase2, Bulletin, Cloud, Faucet, Rug:

	// Wind chimes are the only clutter with a rect, and the only object that
	// increments a counter from inside this function. NumChimes therefore counts
	// **central-room chimes only**, because CreateActiveRects is called for no
	// other room -- a chime one room over does not shorten the interval.
	//
	// It is also the one case that reads its anchor from a Rect rather than a
	// Point: clutterType stores bounds, so the offset comes from bounds.Left and
	// bounds.Top.
	case Chimes:
		w.R.NumChimes++
		bounds = render.ZeroCorner(srcRect(Chimes))
		b := obj.Clutter().Bounds
		bounds = render.Offset(bounds, b.Left, b.Top)
		hot = w.AddActiveRect(bounds, ChimeIt, who, true, false)
	}

	return hot
}

// loadTriggerSound stands in for Sound.c:265-303. It reports whether the sound
// loaded, which is the only thing CreateActiveRects asks.
//
// Two behaviours of the original are reproduced and one is deferred. The
// single-slot limit is reproduced by Room.TriggerSoundHeld: the second sound
// trigger in a room fails whatever sound it names. The `dontLoadSounds` short
// circuit is reproduced by leaving TriggerSoundExists nil, which is the state of a
// build with no sound system. A build with audio wires it to
// audio.Engine.LoadTriggerSound, which reads the house's own 'snd ' resources -- so
// a house whose sound is MACE-compressed, or missing, composes the room without the
// hot spot exactly as the original would have.
func (w *World) loadTriggerSound(soundID int16) bool {
	if w.TriggerSoundExists == nil || w.R.TriggerSoundHeld {
		return false
	}
	if !w.TriggerSoundExists(soundID) {
		return false
	}
	w.R.TriggerSoundHeld = true
	return true
}
