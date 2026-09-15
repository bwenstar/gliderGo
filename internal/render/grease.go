package render

// Grease.c's registration half: the grease table, and the four-cel tipping strip that is
// baked into a saved map when a room is composed.
//
// ---------------------------------------------------------------------------
// Why the table is here and not in internal/game
// ---------------------------------------------------------------------------
//
// A jar of grease is the one dynamic object both halves of the port write to. The
// composition claims its slot and paints its tipping animation (this file); the game loop
// advances the tip, converts the jar's hot spot into a slide rect and repaints the slick
// (internal/game/grease.go). In the original there is no boundary to place: `grease[]` is a
// global in Grease.c that ObjectDrawAll.c, Interactions.c, Triggers.c and Render.c all
// reach into.
//
// It lives on Scene for the same reason SavedMaps does -- Room embeds *Scene, so the game
// side reaches it as w.R.Grease with no hook and no second copy -- and because the
// alternative loses a picture. The dinahs table went the other way, behind
// Scene.AddDynamicObject, and that works because a nil hook composes *the same image*: an
// unregistered dinah is simply a still one. Grease is not like that. Its draw is gated on
// the registration succeeding (`if (dynamicNum != -1)`), so a nil hook would compose a room
// with no grease jar in it at all, and internal/render's own golden images would stop
// showing one. A table that both sides touch is the honest description, so that is what
// this is.
//
// ---------------------------------------------------------------------------
// The saved map holds art, not just background
// ---------------------------------------------------------------------------
//
// Every other saved map is a copy of the wall behind something, kept so the something can
// be erased. Grease's is a **filmstrip**: four 32x27 cels, each the wall behind the jar
// with one cel of the tipping jar composited on top, stacked into one 32x108 patch. That is
// why the slot is claimed at 32x27*4 rather than at the jar's own size, and it is why
// HandleGrease's falling arm is a plain opaque blit -- the frame it wants is already drawn.
//
// The consequence is that a grease jar costs a saved-map slot four times the size of a
// prize's and can therefore be the object that saturates the 24-slot table. See
// docs/analysis for the census.

// The four grease modes (Grease.c:17-20). A jar is Idle until something knocks it over,
// Falling for the four frames of the tip, Spreading while the slick grows two pixels a
// frame, and SpiltIdle for the rest of the room.
//
// Idle is 0, which matters twice: a fresh table entry is Idle without being written, and
// RedrawAllGrease's `mode != kGreaseIdle` test is what keeps an untouched jar out of it.
const (
	GreaseIdle      int16 = 0
	GreaseFalling   int16 = 1
	GreaseSpreading int16 = 2
	GreaseSpiltIdle int16 = 3
)

// Grease is greaseType (GliderStructs.h:281-288): one jar of grease in the composed locale.
type Grease struct {
	// Dest is the jar's 32x27 rect in *screen* (back-map) coordinates, playOrigin
	// already added -- unlike a hot spot's Bounds, which is room-local. AddGrease is
	// handed itsRect after OffsetRectRoomRelative, so this is what it gets, and it is
	// why HandleGrease's blits take no offset while the hot-spot bounds it writes do.
	//
	// It moves: the falling jar shifts two pixels a frame in the direction it tips.
	Dest Rect

	// MapNum is the SavedMaps slot holding the four-cel strip. Not optional -- AddGrease
	// refuses to register a jar it could not claim a slot for.
	MapNum int16

	// Mode is one of the four constants above.
	Mode int16

	// Who and Where identify the object: an index into the room's object list and a real
	// room number. The pair is what ReBackUpGrease matches on, and what
	// RedrawAllGrease's `where == thisRoomNumber` test reads -- the table holds jars from
	// all nine local rooms and only the central room's are repainted.
	Who   int16
	Where int16

	// Start is the leading edge of the slick, in screen coordinates, and it is the only
	// field the spreading arm advances. Stop is where it stops: `distance` pixels from
	// the jar, where distance is the author's data.c.length.
	//
	// Both run right for a right-facing jar and left for a left-facing one, so the
	// termination test has to be `>=` for one and `<=` for the other. There is no
	// direction-independent spelling of it, which is why the C has two arms.
	Start int16
	Stop  int16

	// Frame is the strip index, and it starts at **-1** rather than 0. HandleGrease
	// pre-increments, so the first falling frame draws cel 0 and the fourth reaches 3 and
	// stops. An idle jar's -1 is never used as an index.
	Frame int16

	// HotNum is the index into the game side's hot-spot table of the rect this jar
	// created -- its kRewardIt rect while it is upright, which HandleGrease rewrites
	// into a kSlideIt rect once the jar is over. It is written by SpillGrease, not here:
	// a jar that is never knocked over never has a meaningful HotNum.
	//
	// **Zero, not -1, is the unwritten value**, matching the C's zeroed global array.
	// RedrawAllGrease reads hotSpots[hotNum] before testing the mode, so an untouched
	// jar's read lands on hot spot 0 and the result is thrown away. In the C that read is
	// always in bounds because hotSpots is allocated at its cap; here the table is a
	// slice whose length grows, so the game side guards the index. See RedrawAllGrease.
	HotNum int16

	// IsRight selects every left/right pair above: which way the jar tips, which of the
	// two art strips is baked, which end of Dest the slick grows from, and the sign of
	// the two-pixel step.
	IsRight bool
}

// AddGrease is Grease.c:206-251: register a jar of grease and bake its tipping strip.
//
// h and v are the jar's screen position -- itsRect's top-left after
// OffsetRectRoomRelative -- and distance is the author's data.c.length, the length the
// slick will reach. It returns the new table index, or -1, and **the caller must gate the
// jar's draw on that**: a jar that could not be registered is not drawn at all, because
// there would be no strip to animate it out of.
//
// Two things in it are easy to get wrong.
//
// The saved-map slot is claimed with a rect **at the screen origin** -- (0,0,32,108), not
// the jar's own rect -- so what it initially holds is the top-left corner of the back map.
// That is not a bug and does not need repairing: backupGrease overwrites all four cels
// immediately, so no pixel of the claim survives. It does leave the slot's Dest pointing at
// the corner of the play area, which is the hazard RestoreFromSavedMap's "first matching
// slot wins" note describes -- harmless here only because no reward arm restores grease. A
// knocked-over jar becomes a slick in place; it is never erased.
//
// And `src` is passed to backupGrease **by pointer because backupGrease moves it**: it
// steps the rect two pixels four times, once per cel. The ∓8 immediately afterwards is
// undoing exactly that, which is why the sign looks backwards -- a right-facing jar's rect
// has just been advanced +8 and is pulled back to where it started. So Dest ends up at
// (h,v) precisely, and reading the two offsets as a deliberate 8-pixel nudge is the
// misreading to avoid.
func (s *Scene) AddGrease(where, who, h, v, distance int16, isRight bool) int16 {
	if len(s.Grease) >= kMaxGrease {
		return -1
	}

	src := Offset(SetRect(0, 0, 32, 27), h, v)

	savedNum := s.backUpToSavedMap(SetRect(0, 0, 32, 27*4), where, who, false)
	if savedNum == -1 {
		return -1
	}
	s.backupGrease(&src, savedNum, isRight)

	if isRight {
		src = Offset(src, -8, 0)
	} else {
		src = Offset(src, 8, 0)
	}

	g := Grease{
		Who:     who,
		Where:   where,
		Dest:    src,
		MapNum:  int16(savedNum),
		Mode:    GreaseIdle,
		Frame:   -1,
		IsRight: isRight,
	}
	if isRight {
		g.Start = src.Right + 4
		g.Stop = src.Right + distance
	} else {
		g.Start = src.Left - 4
		g.Stop = src.Left - distance
	}

	s.Grease = append(s.Grease, g)
	return int16(len(s.Grease) - 1)
}

// ReBackUpGrease is Grease.c:180-199: a light was switched, so re-bake the tipping strip
// against the room's new brightness.
//
// It claims nothing. The jar is already in the table and already owns a saved-map slot;
// this finds it by (where, who) and repaints the four cels from the current back map. That
// is the difference from ReBackUpSavedMap, which does go through the slot table, and it is
// why this is a separate function rather than a `redraw` flag on the one above.
//
// **It returns the index whether or not it re-baked anything.** The mode test only guards
// the repaint: a jar that has finished spreading has no jar left to draw, so re-baking it
// would be pointless, but the caller still needs the index to gate its draw on. Returning
// -1 there would make a light switch delete every spilt jar in the room.
//
// One artefact of the original is reproduced rather than fixed. A jar caught **mid-fall**
// re-bakes from its *current* Dest, which has already walked two pixels per frame, so the
// four cels are backgrounds from Dest, Dest±2, Dest±4, Dest±6 while the animation is about
// to read the cel for a frame it has already passed. The wall behind the remaining cels is
// therefore offset by up to six pixels for the rest of the tip. Switching a light during
// the four frames a jar takes to fall over is the whole of the exposure.
func (s *Scene) ReBackUpGrease(where, who int16) int16 {
	for i := range s.Grease {
		g := &s.Grease[i]
		if g.Where != where || g.Who != who {
			continue
		}
		if g.Mode == GreaseIdle || g.Mode == GreaseFalling {
			src := g.Dest
			s.backupGrease(&src, int(g.MapNum), g.IsRight)
		}
		return int16(i)
	}
	return -1
}

// backupGrease is Grease.c:141-172: bake the four cels of the tipping animation into a
// saved-map slot.
//
// Each cel is the back map at *src* with one frame of the jar composited on top, and src
// walks two pixels per cel in the direction the jar tips -- so cel 3 is the wall six pixels
// along from cel 0, which is where the jar will have got to by then. Getting that walk
// wrong shows up as the jar appearing to slide back as it falls.
//
// src is a pointer because the walk is visible to the caller and AddGrease depends on it.
// See AddGrease.
func (s *Scene) backupGrease(src *Rect, index int, isRight bool) {
	if index < 0 || index >= len(s.SavedMaps) {
		return
	}
	patch := s.SavedMaps[index].Map
	if patch == nil {
		return
	}
	art := s.A.Sheet("bonus")

	dest := SetRect(0, 0, 32, 27)
	for i := 0; i < 4; i++ {
		patch.Copy(s.Back, *src, dest, SrcCopy)

		if isRight {
			if art != nil {
				patch.Copy(art, greaseSrcRt[i], dest, Masked)
			}
			*src = Offset(*src, 2, 0)
		} else {
			if art != nil {
				patch.Copy(art, greaseSrcLf[i], dest, Masked)
			}
			*src = Offset(*src, -2, 0)
		}

		dest = Offset(dest, 0, 27)
	}
}
