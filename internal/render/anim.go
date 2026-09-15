package render

// DynamicMaps.c's five animated families, registration half: candle flames, tiki flames,
// barbecue coals, the cuckoo clock's pendulum and the spinning star.
//
// The simulation half -- the three Render.c passes that step them -- is
// internal/game/anim.go. The split is grease's exactly (see grease.go): the tables are on
// Scene because a registration failure has to be visible to the composition, and the
// animators are on World because they play sounds and register dirty rects.
//
// ---------------------------------------------------------------------------
// A saved-map slot is not a background here, it is a filmstrip
// ---------------------------------------------------------------------------
//
// Every other saved map in the game is a swatch of wall kept so the thing in front of it
// can be erased. These five are films. `BackUpFlames` and its four siblings claim a slot
// one cel wide and *every cel* tall, then fill it: for each cel, copy the background from
// the back map, and mask one frame of the animation over it. So a candle's slot is a 16x75
// strip holding five separately-composited pictures of the same patch of wall with a
// different flame on it.
//
// Two consequences follow, and both are why these functions exist at all rather than the
// animator just masking the sprite each frame.
//
// **The animators are opaque blits with no erase.** RenderFlames does one `srcCopy` from
// the strip into the work map and registers a work rect. It never registers a back rect,
// because the background is already inside the cel it just drew: this frame's blit erases
// the last one. That is the whole trick, and it is why an animated object costs one blit a
// frame where a dynamic object costs two.
//
// **A lighting change invalidates every strip in the room.** The cels were composited
// against the *lit* or *unlit* wall. Flip a light switch and all five families have to be
// re-baked, which is what the ReBackUp pair below is for and why they are called from the
// twenty registration sites' `redraw` arms rather than from anywhere sensible.
//
// ---------------------------------------------------------------------------
// The C's two-step lookup, and the missing `break` that saves it
// ---------------------------------------------------------------------------
//
// The C's ReBackUp* functions do not search their own table. They search *savedMaps* for
// the slot tagged (where, who), then search the animation table for the entry pointing at
// that slot index -- because the C's `flames[].who` is a saved-map index, not an object
// index. Read that pair of loops and the outer one looks like it should `break` on a match.
//
// It must not, and the reason is the star and the cuckoo. Those two are the only objects
// that claim **two** slots with the same (where, who) tag: one by DrawARoomsObjects for the
// object's own rect, and one by AddStar/AddPendulum for the filmstrip. The outer loop finds
// the object-rect slot first, fails to match any animation against it, and only reaches the
// filmstrip slot because it kept going. Add the `break` that looks missing and stars and
// pendulums stop being re-baked by a light switch.
//
// The port does not need the indirection: Anim carries the object identity as well as the
// slot, so `reBackUp` below matches (Where, Who) directly and is equivalent in every case.
// The note is here because the indirection is what a reader comparing against the C will
// find, and "why no break" is the question it raises.
//
// ---------------------------------------------------------------------------
// Five RandomInt draws that are not always drawn
// ---------------------------------------------------------------------------
//
// Four of the five families seed their starting cel from RandomInt, so a room of candles
// does not flicker in lockstep, and the pendulum draws for its starting direction. Each
// draw is **inside `if savedNum != -1`** -- inside the saturation guard.
//
// So when the 24-slot savedMaps table fills, the draw does not happen, and every
// subsequent random number in the game is one place earlier in the sequence than it would
// otherwise have been. Whether registration succeeds depends on how many objects were
// visible, which depends on SectRect against the screen, which depends on the screen size.
// **The RNG stream is a function of the window's dimensions.** That is transcribed as-is,
// because a replay recorded at one size has to reproduce at that size, and it is why
// Stage 3's race has to agree a resolution as well as a seed. docs/IMPROVEMENTS.md 2.42.

// Anim is one entry in one of the five flame-like tables: candle flames, tiki flames,
// barbecue coals, pendulums and stars.
//
// The C keeps five separate structs with the same five fields under three different names
// (`flames[].who` and `pendulums[].link` mean opposite things), so this is one type used
// five times rather than a transcription of any one of them.
type Anim struct {
	// Dest is where on screen the cels are blitted. Screen coordinates: the
	// composition had already offset the object's rect by playOrigin and the room
	// offset before it got here.
	Dest Rect

	// Src is the cel currently showing, as a rect inside the filmstrip. The animators
	// step it by one cel height per frame and reset it to the top on the wrap, which
	// is why it is stored rather than derived from Mode -- the C stores both and lets
	// them agree, and one place where they briefly do *not* is load-bearing. See Mode.
	Src Rect

	// SavedMap is the savedMaps slot holding the filmstrip. The C calls this `who`.
	SavedMap int

	// Where and Who are the object's room and index -- the C's `where` and `link`,
	// which only the pendulum and the star actually store. Carrying them for all five
	// is what lets reBackUp match directly; see the file comment.
	Where int16
	Who   int16

	// Mode is the cel index. It is seeded from RandomInt for four of the five families
	// and can legitimately be seeded **one past the end** of the strip, because
	// RandomInt(n) can return n -- see World.RandomInt. That is harmless and is not
	// clamped: every animator pre-increments and wraps before its first blit, so an
	// out-of-range Src is always replaced before anything reads it. Clamping here
	// would consume the same draw and produce a different first cel, which is a
	// visible divergence for one flame in 65536.
	Mode int16

	// ToOrFro is the pendulum's swing direction, seeded from a coin flip. Unused by
	// the other four.
	ToOrFro bool

	// Stopped retires the entry. Only the pendulum and the star can be stopped -- by
	// collecting the cuckoo clock or the star they belong to -- and the C spells the
	// same idea two ways: `pendulums[i].active = false` (DynamicMaps.c:698) and
	// `theStars[i].mode = -1` (DynamicMaps.c:714). Both are read as a plain gate by
	// Render.c:278 and :429, so one flag covers both.
	//
	// The two are not quite interchangeable in the C, and the difference does not
	// matter only because nothing ever restarts either: a stopped pendulum keeps its
	// phase in `mode` and would resume where it left off, while a stopped star has
	// overwritten its phase with the sentinel and could not. The port's animators read
	// this flag and leave Mode alone, so a hypothetical restart would resume both --
	// the pendulum's behaviour, not the star's.
	Stopped bool
}

// ---------------------------------------------------------------------------
// Baking a filmstrip
// ---------------------------------------------------------------------------

// bakeStrip is DynamicMaps.c:263-284, :349-370, :435-456, :521-542 and :612-633 -- five
// functions that differ only in the cel size, the frame count, which art sheet the frames
// come from, and the table of source rects.
//
// They are one function here because there is nothing to learn from the other four copies
// and a great deal to lose: the five are exactly the sort of thing that drifts, and the C
// already shows the drift. BackUpStar loops `i < 6` where its four siblings loop to a
// named constant, and BackUpPendulum's cel height is written as a literal 28 in two
// places. Collapsing them means the strip layout is stated once.
//
// Note what does *not* move: `src` stays put for all four or five cels, unlike
// backupGrease's, which walks. Every cel of a flame is the same patch of wall with a
// different flame on it, so there is no walk to reproduce -- and that is why these take
// src by value where grease takes it by pointer.
func (s *Scene) bakeStrip(src Rect, index int, sheet string, cels []Rect) {
	if index < 0 || index >= len(s.SavedMaps) || len(cels) == 0 {
		return
	}
	patch := s.SavedMaps[index].Map
	if patch == nil {
		return
	}
	art := s.A.Sheet(sheet)

	stride := cels[0].Tall()
	dest := SetRect(0, 0, cels[0].Wide(), stride)
	for i := range cels {
		// The background first, opaque, then the cel masked over it. Both land in
		// the saved map rather than on screen.
		patch.Copy(s.Back, src, dest, SrcCopy)
		if art != nil {
			patch.Copy(art, cels[i], dest, Masked)
		}
		dest = Offset(dest, 0, stride)
	}
}

// The five BackUp* functions, each a name and a table. Kept as named wrappers rather than
// collapsed into their call sites so that a reader searching for BackUpBBQCoals finds
// something, and so that the sheet each family draws from is recorded once.
//
// Two families come from the blower sheet and three from the bonus sheet, which is not a
// grouping anyone would choose -- it is where the 1994 artist had room.
func (s *Scene) backUpFlames(src Rect, index int) {
	s.bakeStrip(src, index, "blower", flameSrc[:])
}

func (s *Scene) backUpTikiFlames(src Rect, index int) {
	s.bakeStrip(src, index, "blower", tikiFlameSrc[:])
}

func (s *Scene) backUpBBQCoals(src Rect, index int) {
	s.bakeStrip(src, index, "blower", coalsSrc[:])
}

func (s *Scene) backUpPendulum(src Rect, index int) {
	s.bakeStrip(src, index, "bonus", pendulumSrc[:])
}

func (s *Scene) backUpStar(src Rect, index int) {
	s.bakeStrip(src, index, "bonus", StarSrc[:])
}

// ---------------------------------------------------------------------------
// Re-baking after a lighting change
// ---------------------------------------------------------------------------

// reBackUp is the body of ReBackUpFlames (DynamicMaps.c:292-312) and its four siblings:
// find this object's entry and repaint its filmstrip against the back map as it is now.
//
// It claims nothing and returns nothing. The entry already owns its slot; a light was
// switched, so the cels are stale. See the file comment for why the C reaches the same
// entry through savedMaps and why its outer loop must not break.
//
// **It repaints from the entry's current Dest**, which for these five is the same rect the
// registration used -- unlike grease, whose jar may have walked. So there is no equivalent
// of ReBackUpGrease's mid-fall artefact here: a flame does not move.
func (s *Scene) reBackUp(table []Anim, where, who int16, bake func(src Rect, index int)) {
	for i := range table {
		if table[i].Where == where && table[i].Who == who {
			bake(table[i].Dest, table[i].SavedMap)
			return
		}
	}
}

// ReBackUpFlames is DynamicMaps.c:292-312, called from the `redraw` arm of the kTaper,
// kCandle and kStubby cases. Exported because the composition's redraw pass is the only
// caller and it reads better at the call site than a lowercase name would.
func (s *Scene) ReBackUpFlames(where, who int16) {
	s.reBackUp(s.Flames, where, who, s.backUpFlames)
}

// ReBackUpTikiFlames is DynamicMaps.c:376-396.
func (s *Scene) ReBackUpTikiFlames(where, who int16) {
	s.reBackUp(s.TikiFlames, where, who, s.backUpTikiFlames)
}

// ReBackUpBBQCoals is DynamicMaps.c:462-482.
func (s *Scene) ReBackUpBBQCoals(where, who int16) {
	s.reBackUp(s.Coals, where, who, s.backUpBBQCoals)
}

// ReBackUpPendulum is DynamicMaps.c:546-566. One of the two whose object claims two
// saved-map slots; see the file comment.
func (s *Scene) ReBackUpPendulum(where, who int16) {
	s.reBackUp(s.Pendulums, where, who, s.backUpPendulum)
}

// ReBackUpStar is DynamicMaps.c:638-658. The other one.
func (s *Scene) ReBackUpStar(where, who int16) {
	s.reBackUp(s.Stars, where, who, s.backUpStar)
}

// ---------------------------------------------------------------------------
// Registration
// ---------------------------------------------------------------------------

// randomInt forwards to the RandomInt hook, which is World.RandomInt during play.
//
// **A nil hook returns 0, and that is a deliberate choice rather than a fallback.** The
// renderer's own tests compose rooms with no World attached, and a nil hook there means
// every flame in every golden starts on cel 0 -- reproducible, and invisible in the
// goldens either way, because a filmstrip lives in a saved map and never touches the
// composed image. What a nil hook must not do is consume entropy from somewhere else or
// panic: the five draws are load-bearing for the *stream*, not for the picture. See the
// file comment.
func (s *Scene) randomInt(rng int16) int16 {
	if s.RandomInt == nil {
		return 0
	}
	return s.RandomInt(rng)
}

// seedCel fills in the two fields every family seeds the same way: a random starting cel
// and the Src rect that names it.
//
// The draw happens here, which is to say *after* the caller's saturation guard, because
// that is where the C puts it. Hoisting it above the guard would tidy four functions and
// change every random number in the game from the first saturated room onward.
func seedCel(a *Anim, mode, celW, celH int16) {
	a.Mode = mode
	a.Src = Offset(SetRect(0, 0, celW, celH), 0, mode*celH)
}

// addCandleFlame is DynamicMaps.c:316-343: register a candle flame and bake its five cels.
//
// The h/v guard is the original's and is not a bounds check on anything in particular --
// `h < 16 || v < 15` refuses a flame whose 16x15 rect would start left of or above the
// screen origin, which for a candle in the top-left neighbour room is reachable. The rect
// is anchored **bottom-centre**: h-8, v-15. It is the only one of the five that is.
//
// The 4-bit-depth even-column alignment (`thisMac.isDepth == 4`) is skipped in all five, as
// it is everywhere else in the port: the extracted art is 8-bit indexed and there is no
// 16-colour mode to align for. It would shift a flame one pixel left on a 4-bit Mac.
func (s *Scene) addCandleFlame(where, who, h, v int16) {
	if len(s.Flames) >= kMaxCandles || h < 16 || v < 15 {
		return
	}
	dest := Offset(SetRect(0, 0, 16, 15), h-8, v-15)
	slot := s.backUpToSavedMap(SetRect(0, 0, 16, 15*NumCandleFrames), where, who, false)
	if slot == -1 {
		return
	}
	s.backUpFlames(dest, slot)

	a := Anim{Dest: dest, SavedMap: slot, Where: where, Who: who}
	seedCel(&a, s.randomInt(NumCandleFrames), 16, 15)
	s.Flames = append(s.Flames, a)
}

// addTikiFlame is DynamicMaps.c:400-428. Unlike the candle the anchor is the top-left, so
// the caller has already offset it -- see the kTiki case, which passes itsRect.Top-9.
func (s *Scene) addTikiFlame(where, who, h, v int16) {
	if len(s.TikiFlames) >= kMaxTikis || h < 8 || v < 10 {
		return
	}
	dest := Offset(SetRect(0, 0, 8, 10), h, v)
	slot := s.backUpToSavedMap(SetRect(0, 0, 8, 10*NumTikiFrames), where, who, false)
	if slot == -1 {
		return
	}
	s.backUpTikiFlames(dest, slot)

	a := Anim{Dest: dest, SavedMap: slot, Where: where, Who: who}
	seedCel(&a, s.randomInt(NumTikiFrames), 8, 10)
	s.TikiFlames = append(s.TikiFlames, a)
}

// addBBQCoals is DynamicMaps.c:486-514. Four cels rather than five, and the only family
// whose art is overlaid on a PICT object rather than a sheet-drawn one.
func (s *Scene) addBBQCoals(where, who, h, v int16) {
	if len(s.Coals) >= kMaxCoals || h < 32 || v < 9 {
		return
	}
	dest := Offset(SetRect(0, 0, 32, 9), h, v)
	slot := s.backUpToSavedMap(SetRect(0, 0, 32, 9*NumCoalFrames), where, who, false)
	if slot == -1 {
		return
	}
	s.backUpBBQCoals(dest, slot)

	a := Anim{Dest: dest, SavedMap: slot, Where: where, Who: who}
	seedCel(&a, s.randomInt(NumCoalFrames), 32, 9)
	s.Coals = append(s.Coals, a)
}

// addPendulum is DynamicMaps.c:570-604, and it is the odd one of the five in four ways.
//
// It claims its slot **before** computing dest, the opposite order from the flames. It does
// not seed Mode from RandomInt: the swing always starts at cel 1, the middle of three, and
// the *draw* is a coin flip for the direction instead. It writes ClockFrame, a phase
// counter shared by every pendulum in the locale. And its cel index is the one place a
// stored Src and a stored Mode are set from different expressions -- Mode is 1 and Src is
// offset by one stride -- which is why Anim carries both rather than deriving one.
//
// ClockFrame = 10 is not "start at frame 10 of something". RenderPendulums swings when the
// counter reads 10 or 15 and resets it at 15, so seeding it at 10 makes the first swing
// happen five frames later rather than ten. See RenderPendulums for the uneven tick-tock
// that falls out of it.
func (s *Scene) addPendulum(where, who, h, v int16) {
	if len(s.Pendulums) >= kMaxPendulums || h < 32 || v < 28 {
		return
	}
	s.ClockFrame = 10
	slot := s.backUpToSavedMap(SetRect(0, 0, 32, 28*NumPendulumFrames), where, who, false)
	if slot == -1 {
		return
	}
	dest := Offset(SetRect(0, 0, 32, 28), h, v)
	s.backUpPendulum(dest, slot)

	a := Anim{Dest: dest, SavedMap: slot, Where: where, Who: who}
	seedCel(&a, 1, 32, 28)
	// `if (RandomInt(2) == 0) toOrFro = true; else false` -- the draw is consumed
	// either way, which is the only thing about it that matters to the stream.
	a.ToOrFro = s.randomInt(2) == 0
	s.Pendulums = append(s.Pendulums, a)
}

// addStar is DynamicMaps.c:662-694, the only one of the five with no h/v guard at all, and
// the only object in the game that consumes **two** savedMaps slots: one claimed by
// DrawARoomsObjects for the star itself, so the star can be erased when it is collected,
// and one here for its six-cel spin.
//
// That doubling is what the note on SavedMap.Dest is about and what the file comment's
// "missing break" paragraph turns on.
func (s *Scene) addStar(where, who, h, v int16) {
	if len(s.Stars) >= kMaxStars {
		return
	}
	dest := Offset(SetRect(0, 0, 32, 31), h, v)
	slot := s.backUpToSavedMap(SetRect(0, 0, 32, 31*NumStarFrames), where, who, false)
	if slot == -1 {
		return
	}
	s.backUpStar(dest, slot)

	a := Anim{Dest: dest, SavedMap: slot, Where: where, Who: who}
	seedCel(&a, s.randomInt(NumStarFrames), 32, 31)
	s.Stars = append(s.Stars, a)
}
