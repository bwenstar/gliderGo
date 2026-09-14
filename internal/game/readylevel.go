package game

// This file is the room-change entry points: ReadyLevel (RoomGraphics.c:402-419),
// the composition it drives, and InitGarbageRects (Render.c:675-691).
//
// There are two of them and the difference between them is the whole reason this
// file exists. ReadyLevel is a *level* change: new room, new openings, new
// everything, and the frame clock restarts. Rebuild is a *re-composition* of the
// room already loaded, which RestoreEntireGameScreen does after a dialogue has been
// over the window (Play.c:814) -- no NilSavedMaps before it and no InitGarbageRects
// after it, so the dirty-rect state and the frame clock survive.
//
// Collapsing the two, in either direction, is a real bug. Calling ReadyLevel where
// Rebuild belongs resets nextFrame in the middle of a room and gives the player a
// free frame; calling Rebuild where ReadyLevel belongs leaves the previous room's
// openings in place, so the glider can fly through a wall that is now drawn solid.

// ReadyLevel is RoomGraphics.c:402-419: load the central room and everything around
// it, from scratch.
//
// Five steps, and the order is load-bearing at two points. DetermineRoomOpenings has
// to run before the composition because DrawLocale's tail caches ShadowVisible, which
// reads the same background the openings were derived from. And InitGarbageRects has
// to run last because it seeds the frame clock, which must not start ticking until
// there is something to draw.
func (w *World) ReadyLevel() {
	// NilSavedMaps: dispose every saved background patch. In the C this frees 24
	// GWorlds and is the only place they are freed, so skipping it leaks a room's
	// worth of offscreen buffers per room change. Here the slice is truncated and
	// the GC does the rest, which makes this call and the identical one at the head
	// of DrawLocale redundant rather than one being the safety net for the other.
	// Kept because the sequence is the sequence, and because a future backend that
	// holds GPU textures in SavedMap.Map would need the explicit release back.
	w.R.SavedMaps = w.R.SavedMaps[:0]

	// The QuickTime block (COMPILEQT): if a movie is playing in this room's TV, stop
	// it and forget it. Note that this runs *before* DrawLocale's own
	// `tvInRoom = false; tvWithMovieNumber = -1`, and is gated on tvInRoom being
	// true -- so it is the only place StopMovie is called on a room change, and
	// DrawLocale's clearing of the same two fields is what makes it a no-op the
	// second time.
	if w.HasMovie && w.R.TVInRoom {
		w.R.TVInRoom = false
		w.R.TVMovieNumber = -1
		// StopMovie(theMovie). Movie playback is not in scope: the extractor pulled
		// the QuickTime tracks out as index buffers, and 1.5e decides what to do
		// with them.
	}

	w.DetermineRoomOpenings()
	w.Rebuild()
	w.InitGarbageRects()
}

// Rebuild is DrawLocale (RoomGraphics.c:43-129): compose the locale and rebuild every
// table that describes it.
//
// It is a method on World rather than a constructor returning a new Room, and that is
// not a style choice. RestoreEntireGameScreen calls DrawLocale on the room the glider
// is standing in, mid-life, mid-flight (Play.c:814); replacing Room wholesale there
// would discard the room identity and the surfaces the glider is being drawn against.
// The original rebuilds in place and so does this.
//
// The drawing and most of the reset head live in render.Scene.DrawLocale, which was
// finished in Stage 1.3. What is added here is the four resets the renderer has no
// tables for, the object graph, and the two cached flags the composition ends with.
func (w *World) Rebuild() {
	// The reset head of DrawLocale is ten statements. Six -- ZeroFlamesAndTheLike,
	// ZeroDinahs, KillAllBands, ZeroMirrorRegion, ZeroTriggers and
	// numTempManholes = 0 -- are inside Scene.DrawLocale, which owns those tables.
	// The remaining four are these:
	//
	// FlushAnyTriggerPlaying() and DumpTriggerSound() free the one reserved sound
	// slot. Clearing TriggerSoundHeld is what re-arms it, and is what makes the
	// first kSoundTrigger of each room the one that works -- see loadTriggerSound.
	w.R.TriggerSoundHeld = false

	// tvInRoom = false; tvWithMovieNumber = -1. DrawARoomsObjects sets them again if
	// this room has a TV. World.TVOn is deliberately *not* reset -- see the note on
	// that field for why leaving it stale is correct.
	w.R.TVInRoom = false
	w.R.TVMovieNumber = -1

	// The composition. Scene.DrawLocale calls back into ListAllLocalObjects at the
	// point the C does, between the localNumbers loop and the first PaintRect.
	w.R.ListLocalObjects = w.ListAllLocalObjects
	w.R.NumChimes = 0
	w.R.DrawLocale()

	// The two flags DrawLocale ends with. ShadowVisible is cached here and nowhere
	// else during play except RedrawRoomLighting, because the only thing that can
	// change the answer mid-room is a light going on or off.
	w.R.ShadowVisible = w.IsShadowVisible()
	w.R.TakingTheStairs = false
}

// RedrawRoomLighting is RoomGraphics.c:434-462: a light was switched, so recount and
// possibly recompose.
//
// The point of it is the `wasLit != isLit` test: switching a second lamp on in a room
// that already has one changes numLights from 1 to 2 and redraws nothing, because the
// count is only ever used as zero-or-not. So a room with two lamps goes dark only
// when the second one is switched off, and the redraw happens on that transition
// alone.
func (w *World) RedrawRoomLighting() {
	wasLit := w.R.NumLights > 0
	w.R.NumLights = w.R.GetNumberOfLights(w.R.RoomNumber)
	isLit := w.R.NumLights > 0
	if wasLit == isLit {
		return
	}
	w.Rebuild()
	w.R.ShadowVisible = w.IsShadowVisible()
}

// InitGarbageRects is Render.c:675-691: clear the dirty-rect lists and start the
// frame clock.
//
// The name is the original's and undersells it: three of its five jobs are not about
// rectangles. It also drops every live sparkle and flying point, which is why the
// score numerals floating up from a collected prize vanish the instant the player
// changes room rather than following them into it.
//
// The last line is the one that matters most and is the easiest to lose:
//
//	nextFrame = TickCount() + kTicksPerFrame
//
// Without it the first frame of a new room is due in the past, so the loop runs a
// burst of catch-up frames as fast as it can and the glider crosses the room before
// the player sees it. That is the "teleport on room entry" bug, and this line is the
// whole of its absence.
func (w *World) InitGarbageRects() {
	w.Work2Main = w.Work2Main[:0]
	w.Back2Work = w.Back2Work[:0]

	// numSparkles = 0 with every sparkles[i].mode = -1, and the same for
	// flyingPoints. Both tables are the effects layer's and arrive in 1.5e; the
	// mode = -1 sweep is what makes a slot reusable, so they cannot be modelled as
	// bare slices the way the two rect lists can.

	w.NextFrame = w.Ticks() + TicksPerFrame
}

// TicksPerFrame is kTicksPerFrame (GliderDefines.h): two Mac ticks, i.e. 2/60.15 s,
// giving the original's 30.07 fps. The game is frame-locked to it -- every velocity
// in the physics is pixels per frame -- so it is a unit of the simulation and not a
// display preference.
const TicksPerFrame = 2

// Ticks is TickCount(): sixtieths of a second since boot, on a 60.15 Hz clock.
//
// It is a hook rather than a call into the platform layer so that a fidelity run can
// drive the clock from a fixed sequence and get the same frames every time. With no
// hook installed it answers from the frame counter -- Frame * TicksPerFrame -- which
// is a clock that runs at exactly the nominal rate and never drifts. That is the
// right default for a headless build and for every test in this package; the play
// loop installs the real one.
func (w *World) Ticks() int64 {
	if w.TickCount == nil {
		return w.Frame * TicksPerFrame
	}
	return w.TickCount()
}
