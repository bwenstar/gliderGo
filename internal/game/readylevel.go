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

import "glidergo/internal/game/player"

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
	// The reset head of DrawLocale is ten statements. Five --
	// ZeroFlamesAndTheLike, ZeroDinahs, KillAllBands, ZeroMirrorRegion and
	// numTempManholes = 0 -- are inside Scene.DrawLocale, which owns those tables.
	// The remaining five are these, in the C's order:
	//
	// ZeroTriggers() is the fifth statement, and the trigger table is the one item of
	// the reset head that lives on this side of the render boundary: a fuse names a
	// house object, not a drawn one. So a trigger armed on the way out of a room never
	// fires -- see ZeroTriggers.
	w.ZeroTriggers()

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

	// Four of the five dinahs hooks, plus three that reach the other way. In the original
	// these are ordinary calls inside RoomGraphics.c and ObjectDrawAll.c, because the
	// dinahs table, the hotSpots table and the master-object graph are all globals in the
	// same program.
	// Here the tables are internal/game's and the numbering is internal/render's, so the
	// object pass reaches back across the boundary four times: it clears the table at the
	// top, registers into it per object, reads a hotSpots index for the six switch kinds,
	// and writes whichever number it ended up with into the master graph. See
	// Scene.AddDynamicObject for why the coordinate conversion is on the render side.
	//
	// Assigned here, next to ListLocalObjects, rather than at Scene construction: the
	// Scene is rebuilt in place by DrawLocale and these four are the same kind of thing
	// ListLocalObjects is -- the game half of one composition -- so a reader looking for
	// what the renderer is allowed to call during a compose finds all five in one
	// paragraph. The fifth dinahs hook, UpdateOutletsLighting, is *not* one of them and is
	// bound in NewWorld; RedrawCentralRoom is its only caller and does not come through
	// here.
	//
	// KillAllBands and ZeroShreds are here for the same mechanical reason rather than the
	// same conceptual one: both are lines in DrawLocale's reset head, and both clear a
	// table that lives on World, so the renderer cannot clear them itself. Unlike the
	// dinahs four, a nil hook for either composes an identical image -- bands and shreds
	// are drawn by RenderBands and RenderShreds, which the renderer's own goldens never
	// reach -- so both are safe to leave unbound in a render-only test. What KillAllBands
	// *means* is that a band in flight does not survive a room change and the ammunition
	// is not refunded; ZeroShreds is numShredded's line in ZeroFlamesAndTheLike, one of its
	// eight assignments. It is not the only one of the eight on this side -- numChimes is
	// too -- but it is the only one that needs a hook, because numChimes is cleared by the
	// direct assignment just below rather than during the compose.
	//
	// RandomInt is the odd one of the seven, and the only hook in the file whose *return*
	// *value* the composition depends on. The five add* functions in render/anim.go each
	// draw a starting cel from it, so with it bound the room-load RNG stream matches the
	// original's, and with it nil every flame starts on cel 0. Nil is deterministic and
	// invisible in a still image -- a filmstrip lives in a saved map and never reaches
	// the composed frame -- which is why the renderer's own tests can leave it unbound
	// and still hash the same pixels. See render/anim.go for why the draws are where
	// they are and what saturating the saved-map table does to the stream.
	w.R.ZeroDinahs = w.ZeroDinahs
	w.R.AddDynamicObject = w.AddDynamicObject
	w.R.SetDynaNum = w.SetDynaNum
	w.R.MasterHotNum = w.MasterHotNum
	w.R.KillAllBands = w.KillAllBands
	w.R.ZeroShreds = w.ZeroShreds
	w.R.RandomInt = w.RandomInt

	w.R.NumChimes = 0
	w.R.DrawLocale()

	// hasMirror. In the C this is set by AddToMirrorRegion (Render.c:759), which sets
	// it unconditionally on every call and is called once per kMirror object as the
	// room is drawn; ZeroMirrorRegion clears it in DrawLocale's reset head. So it is
	// exactly "this locale drew at least one mirror", and Scene.MirrorRects is the same
	// list AddToMirrorRegion was unioning together -- see Surface.CopyClipped.
	//
	// Deriving it from the list rather than carrying a second flag is the one place
	// this file departs from the C's shape, and it removes a way for the two to
	// disagree. The order matters: DrawLocale has to have run.
	w.R.HasMirror = len(w.R.MirrorRects) > 0

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
//
// The six steps are Scene.RedrawCentralRoom's, and the `redraw == true` it passes to
// DrawARoomsObjects is what makes this a redraw of the *pixels* rather than a fresh
// composition: the twenty registration sites are suppressed, so a band in flight survives,
// the mirror region is not rebuilt and a dinah mid-swoop keeps its place. Through 1.5b this
// function called Rebuild -- all nine rooms, redraw == false -- which re-created all of
// them; that was docs/IMPROVEMENTS.md 2.25 and it is closed here, because step four,
// UpdateOutletsLighting, needed the dinahs table 1.5c builds.
//
// The three lines around the six steps are this side's, exactly as the C splits them: the
// recount above (which also produces isLit), the work rect below, and the ShadowVisible
// recache last. Scene.RedrawCentralRoom does not touch any of the three.
func (w *World) RedrawRoomLighting() {
	wasLit := w.R.NumLights > 0
	w.R.NumLights = w.R.GetNumberOfLights(w.R.RoomNumber)
	isLit := w.R.NumLights > 0
	if wasLit == isLit {
		return
	}
	w.R.RedrawCentralRoom()

	// AddRectToWorkRects(&localRoomsDest[kCentralRoom]) -- RoomGraphics.c:458.
	//
	// Without this the room is recomposed into the work map and never reaches the
	// screen, so the lights appear to change only where something else happens to be
	// dirty: the glider's own rect moves and drags a lit-room-shaped hole around
	// behind it. It is the one dirty-rect registration outside Render.c and the
	// dynamics, and it is easy to miss because it is the second-to-last line of a
	// function whose subject is counting lamps.
	w.AddRectToWorkRects(player.Rect(w.R.V.LocalRoomsDest[CentralRoom]))

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
// Without it the first frame of a new room is due in the past, so it is not waited for
// at all and lands the instant the room is composed. Because awaitFrame reseeds from
// the post-wait clock rather than from the old deadline there is no catch-up burst --
// exactly one frame is un-paced, not a run of them -- but that one frame arrives while
// the wipe transition is still on screen, which is enough to see.
func (w *World) InitGarbageRects() {
	w.Work2Main = w.Work2Main[:0]
	w.Back2Work = w.Back2Work[:0]

	// The two effects tables. The counter and the sweep are not redundant: Mode == -1
	// is what makes a slot reusable, so a table that is only counted to zero reads as
	// three live effects and AddSparkle finds nowhere to put a fourth. This is the only
	// sweeper in the game, which is why sparkles.go's tables cannot be modelled as bare
	// slices the way the two rect lists can.
	w.NumSparkles = 0
	for i := 0; i < MaxSparkles; i++ {
		w.Sparkles[i].Mode = -1
	}

	w.NumFlyingPts = 0
	for i := 0; i < MaxFlyingPts; i++ {
		w.FlyingPoints[i].Mode = -1
	}

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
