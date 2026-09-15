package game

// The frame: GliderPRO/Sources/Render.c.
//
// This is the smallest file in the game with the largest consequences, because it
// defines what "a frame" means. Everything else in internal/game decides *what* to
// draw; Render.c decides *when the player sees it*, and it does so with a protocol
// that is only four moving parts:
//
//	1. backSrcMap holds the composed room with nothing moving in it.
//	2. workSrcMap holds that room plus this frame's moving things.
//	3. work2MainRects lists what has to be shown.
//	4. back2WorkRects lists what has to be erased before the next frame draws.
//
// Every compositor in the game -- the glider, the dynamics, the flames, the bands --
// draws into the work map and then *registers* its rects in those two lists. It
// never blits to the screen itself. CopyRectsQD, at the end of RenderFrame, replays
// both lists in order: the first list to the screen, then the second back over the
// work map. That ordering is the whole trick and it is stated twice below because
// getting it backwards produces a game that looks right for one frame and then
// erases itself.
//
// Two consequences follow that a reader should have in mind before touching
// anything here:
//
// **Registration order is z-order.** CopyRectsQD replays work2MainRects in the order
// it was appended, so two overlapping rects show the later one. That is why
// RenderFrame's call order is transcribed exactly and why the nine renderers that
// belong to later sub-stages are present *now*, as named no-ops called in their real
// positions (see the block at the bottom of this file). Adding them later, in the
// wrong place, would be a silent visual regression that no compile error catches.
//
// **The cap is 47, not 48.** All three adders guard on
// `numWork2Main < (kMaxGarbageRects - 1)` and, when the guard fails, silently drop
// the rect. A dropped work rect is a patch of screen that is never updated; a dropped
// back rect is a patch of work map that is never erased, so whatever was drawn there
// smears. Both are reachable in a busy room and both are reproduced. See
// docs/IMPROVEMENTS.md 2.11 for the release-build instrumentation of it.

import (
	"glidergo/internal/game/player"
	"glidergo/internal/render"
)

// MaxGarbageRects, the cap on both lists, is in consts.go with the other table
// caps. Note that the effective cap is one *less*: every guard below is
// `< MaxGarbageRects - 1`, as the C's are, so slot 47 is unreachable. The constant is
// not "corrected" to 47, because then the guards would stop reading the way the C's
// do and the off-by-one would stop being visible at the three places it bites.

// shadowSrc is shadowSrc[kNumShadowSrcRects] (StructuresInit.c:216-220):
//
//	QSetRect(&shadowSrc[i], 0, 0, kGliderWide, kShadowHigh);
//	QOffsetRect(&shadowSrc[i], 0, kShadowHigh * i);
//
// Two rects of 48x9, stacked in the 48x18 `shadow` strip -- index 0 for a
// right-facing glider and 1 for a left-facing one. The two frames are not mirror
// images of each other; the shadow is drawn with a directional highlight, which is
// why the atlas has two and RenderGlider indexes by facing.
//
// It is [2]player.Rect rather than [2]Rect because its only reader combines it with
// Glider.DestShadow, which is a player.Rect, and the conversion is better done once
// at the blit than twice per branch.
var shadowSrc = [2]player.Rect{
	{Top: 0, Left: 0, Bottom: player.ShadowHigh, Right: player.GliderWide},
	{Top: player.ShadowHigh, Left: 0, Bottom: 2 * player.ShadowHigh, Right: player.GliderWide},
}

// ---------------------------------------------------------------------------
// The three dirty-rect adders (Render.c:65-132)
// ---------------------------------------------------------------------------
//
// All three take a rect in *screen* coordinates -- the caller has already added
// playOriginH/V -- and all three clamp a *copy*, leaving the caller's rect alone.
// That last point is load-bearing: RenderGlider registers `whole` for the work list
// and `dest` for the back list from the same glider on the same frame, and if either
// adder wrote through, the next frame's arithmetic would start from the clamped
// value. See player.Env's contract note for why the offset is the caller's job.
//
// They differ in exactly one way, and it is not a coordinate-space difference. The
// work adder clamps to justRoomsRect and the back adder clamps to workSrcRect, which
// at 640x480 with nine neighbours are the same rectangle; they diverge only in
// kScoreboardLow mode, where justRoomsRect's top drops to 79 to keep the low
// scoreboard's rows out of the *screen* list. The back list has no such concern
// because the work map has no scoreboard in it. So: the work adder is guarding the
// scoreboard, and that is all.

// AddRectToWorkRects is Render.c:65-80: register a rect to be copied from the work
// map to the screen at the end of this frame.
//
// It is player.Env's AddRectToWorkRects, unchanged -- the interface fixes the
// argument as player.Rect and the contract as screen coordinates, so this is the C
// function and not an adapter around it.
//
// The clamp is per-axis else-if, not two independent tests. A rect wider than the
// house rect on both sides therefore has only its left edge pulled in, and stays too
// wide; a rect entirely off the right-hand side has its right edge pulled back to
// justRoomsRect.right while its left edge stays beyond it, producing an inverted
// rect that Surface.Copy declines to draw (surface.go:353). That accident is what
// keeps an off-screen rect from wrapping, and it is why the guard is written as the C
// writes it rather than as a Sect().
func (w *World) AddRectToWorkRects(theRect player.Rect) {
	if len(w.Work2Main) >= MaxGarbageRects-1 {
		w.Diag.DroppedWorkRects++
		return
	}
	r := Rect(theRect)
	jr := w.R.V.JustRoomsRect

	if r.Left < jr.Left {
		r.Left = jr.Left
	} else if r.Right > jr.Right {
		r.Right = jr.Right
	}
	if r.Top < jr.Top {
		r.Top = jr.Top
	} else if r.Bottom > jr.Bottom {
		r.Bottom = jr.Bottom
	}

	w.Work2Main = append(w.Work2Main, r)
}

// AddRectToBackRects is Render.c:84-99: register a rect to be restored from the
// clean background to the work map, after this frame has been shown.
//
// Note the asymmetry with the work adder: the low edges are clamped against the
// literal 0 rather than against workSrcRect.left/top. The two are the same number --
// every offscreen is bounds-origin (0,0) -- so this is the C being terse rather than
// the C meaning something different, and it is transcribed as written.
//
// It is deliberately *not* on player.Env. The player package never calls it: only
// the compositors in this file do, and they are here. See player/env.go's note.
func (w *World) AddRectToBackRects(theRect player.Rect) {
	if len(w.Back2Work) >= MaxGarbageRects-1 {
		w.Diag.DroppedBackRects++
		return
	}
	r := Rect(theRect)
	wr := w.R.V.WorkRect

	if r.Left < 0 {
		r.Left = 0
	} else if r.Right > wr.Right {
		r.Right = wr.Right
	}
	if r.Top < 0 {
		r.Top = 0
	} else if r.Bottom > wr.Bottom {
		r.Bottom = wr.Bottom
	}

	w.Back2Work = append(w.Back2Work, r)
}

// AddRectToWorkRectsWhole is Render.c:103-132, and it is the adder the other two
// should have been.
//
// It does the same job as AddRectToWorkRects with two tests they lack: it rejects a
// rect that lies entirely outside the work map *before* clamping, and it drops a
// rect that the clamp has collapsed to zero width or height. Those are exactly the
// two cases the plain adder handles by accident, through Surface.Copy's early
// return -- so the two produce the same pixels and differ in how many of the 47
// slots they burn.
//
// Its only two callers are GameOver.c:183 and :197, the game-over star animation, so
// nothing in the frame loop reaches it. It is transcribed anyway, because it is the
// evidence that the plain adder's missing tests are an oversight rather than a
// design, and because the game-over sequence in 1.5b's second commit needs it.
//
// One C-ism does not survive: on the degenerate-rect path the original has already
// written the rect into work2MainRects[numWork2Main] and returns without
// incrementing, leaving a stale rect in an unused slot. A later successful call
// overwrites it and no reader can see it, so appending only on success is
// observationally identical.
func (w *World) AddRectToWorkRectsWhole(theRect player.Rect) {
	if len(w.Work2Main) >= MaxGarbageRects-1 {
		w.Diag.DroppedWorkRects++
		return
	}
	wr := w.R.V.WorkRect
	if theRect.Right <= wr.Left || theRect.Bottom <= wr.Top ||
		theRect.Left >= wr.Right || theRect.Top >= wr.Bottom {
		return
	}

	r := Rect(theRect)
	if r.Left < wr.Left {
		r.Left = wr.Left
	} else if r.Right > wr.Right {
		r.Right = wr.Right
	}
	if r.Top < wr.Top {
		r.Top = wr.Top
	} else if r.Bottom > wr.Bottom {
		r.Bottom = wr.Bottom
	}

	if r.Right == r.Left || r.Top == r.Bottom {
		return
	}
	w.Work2Main = append(w.Work2Main, r)
}

// ---------------------------------------------------------------------------
// The glider sheets (Play.c:49, :121-135; StructuresInit.c:177-189)
// ---------------------------------------------------------------------------

// LoadGliderSheets is StructuresInit.c:177-189 and NewGame's glider graphic loads
// (Play.c:121-135) as one call, because in the port they load the same four files.
//
// The original models the sheets as two GWorlds whose *contents* are replaced, which
// is why there are two slots and four PICTs. The slots are not "player 1's sheet"
// and "player 2's sheet":
//
//	two players: glidSrcMap = glider, glid2SrcMap = glider2
//	one player:  glidSrcMap = glider, glid2SrcMap = gliderFoil
//
// So in a one-player game the *second* slot is the foil sheet, and RenderGlider
// reaches for it when showFoil is set. That is the whole reason its test reads
// `(!twoPlayerGame) && (showFoil)`: in a two-player game the second slot holds the
// other player's artwork and reading it for foil would draw the wrong glider. Two
// players get their foil the other way, by SetShowFoil reloading both slots.
//
// The mask does not vary. The C holds one 1-bit glidMaskMap loaded from
// kGliderPictID + 1000 and passes it to all six CopyMask calls in this file, no
// matter which sheet supplies the colours. The port gets that for free because the
// extractor applied mask PICT 4999 to all four glider strips (assets.go:77), so
// every sheet's own Mask *is* glidMaskMap -- which is what makes render.Masked an
// exact CopyMask here rather than an approximation. If a future asset pipeline ever
// gives the four strips different masks, this file needs a real two-surface
// CopyMask instead.
func (w *World) LoadGliderSheets() {
	w.GlidSrc = w.R.A.Strip("glider")
	if w.TwoPlayer {
		w.Glid2Src = w.R.A.Strip("glider2")
	} else {
		w.Glid2Src = w.R.A.Strip("gliderFoil")
	}
	w.ShadowSrc = w.R.A.Strip("shadow")
}

// gliderSheet picks the sheet a glider is drawn from, which is the four-line if the
// two compositors disagree about.
//
// `foilTest` is the difference. RenderGlider passes `(!twoPlayerGame) && showFoil`
// (Render.c:507) and DrawReflection passes a bare `showFoil` (Render.c:163). See
// DrawReflection for why the second is a bug and why it is kept.
func (w *World) gliderSheet(oneOrTwo, foilTest bool) *render.Surface {
	if !oneOrTwo {
		return w.Glid2Src
	}
	if foilTest {
		return w.Glid2Src
	}
	return w.GlidSrc
}

// ---------------------------------------------------------------------------
// DrawReflection (Render.c:136-189)
// ---------------------------------------------------------------------------

// DrawReflection draws the glider a second time, offset up and to the left and
// clipped to the room's mirrors.
//
// It runs *first* in RenderFrame, before anything else touches the work map, and it
// is gated on hasMirror so a room without a mirror pays nothing. The offset is
// (-20,-16) from the glider's real position, which is not a reflection in any
// geometric sense -- it is a fixed parallax that happens to look like one in the
// rooms the original ships, and Modes.c:95 uses the same two numbers to double the
// glider's dirty rect so that the reflection is erased along with it.
//
// Three things here are wrong in the original and all three are kept:
//
// **The foil test.** `if (showFoil)` with no `!twoPlayerGame` guard, unlike
// RenderGlider's. In a two-player game with foil showing, glid2SrcMap holds
// kGliderFoil2PictID -- the *other* player's foil sheet -- so player 1's reflection
// is drawn with player 2's artwork. Harmless-looking and clearly unintended, but a
// player watching a mirror in a two-player game sees it, so a fidelity replay would
// diverge if it were fixed. docs/IMPROVEMENTS.md 2.20 offers it as an opt-in
// correction.
//
// **The unclipped back rect.** `AddRectToBackRects(&dest)` registers the *unclipped*
// destination, so the erase covers the whole reflected glider rect while the draw
// covered only the part inside a mirror. Every frame, the work map outside the mirror
// is restored from a background that never had a reflection in it, which is correct;
// but a candle flame or a pendulum inside that rect is erased a frame early, because
// those register no back rects of their own and rely on their own opaque redraw. That
// is the mirror-room flame blink. IMPROVEMENTS.md 2.19 has the fix (clip the back
// rect to the mirrors), which also relieves the 47-rect cap in a mirror room.
//
// **The unrestored port.** The C does `SetPort(workSrcMap)` and never restores the
// previous port, so whatever ran before RenderFrame is left drawing into the work
// map. Nothing in the frame loop depends on the port afterwards, so it is inert --
// and the port has no ports, so there is nothing to reproduce.
//
// The `wasClip == nil` early return, after dest has been computed and before
// anything is drawn, has no analogue either: it is a NewRgn allocation failure, and
// on that path the C skips the draw *and* both registrations. A Go slice iteration
// cannot fail.
//
// `which`, the facing index, is computed and never used -- dead in the original.
// It is not transcribed; there is nothing for it to select.
func (w *World) DrawReflection(thisGlider *player.Glider, oneOrTwo bool) {
	if thisGlider.DontDraw {
		return
	}

	oh := w.R.V.OriginH + player.MirrorOffsetH
	ov := w.R.V.OriginV + player.MirrorOffsetV

	dest := thisGlider.Dest.Offset(oh, ov)

	// SetClip(mirrorRgn) + CopyMask + SetClip(wasClip). Scene.MirrorRects is the
	// list mirrorRgn was UnionRgn'd together from, so iterating it is the region
	// rather than a stand-in for it; see Surface.CopyClipped.
	if sheet := w.gliderSheet(oneOrTwo, w.ShowFoil); sheet != nil {
		w.R.Work.CopyClipped(sheet, Rect(thisGlider.Src), Rect(dest), render.Masked, w.R.MirrorRects)
	}

	src := thisGlider.Whole.Offset(oh, ov)
	w.AddRectToWorkRects(src)
	w.AddRectToBackRects(dest)
}

// ---------------------------------------------------------------------------
// RenderGlider (Render.c:452-530)
// ---------------------------------------------------------------------------

// RenderGlider draws one glider and its shadow into the work map.
//
// oneOrTwo is true for theGlider and false for theGlider2, and it selects the sheet
// rather than naming the player: see gliderSheet.
//
// The shadow is a separate blit with its own pair of rects, and its source is
// trimmed rather than scaled in two of the transit modes. A glider walking out of the
// top of a room (kGliderGoingDown) or arriving from below (kGliderComingUp) is
// horizontally clipped by the mode handler, so DestShadow has been narrowed too and
// the shadow atlas rect is cut to match -- from the left for those two, from the
// right for kGliderComingDown. Every other mode uses the whole 48x9 rect. Without
// the trim, Surface.Copy would see mismatched extents and *stretch* the shadow,
// which is what QuickDraw would have done too.
//
// The two registrations use different rects on purpose, and this is the pattern every
// compositor in the file follows: the work rect is `whole`, which is the union of
// where the sprite is and where it was, so the screen copy covers the trail; the back
// rect is `dest`, which is only where it is now, so only that much of the work map is
// erased. RenderFlyingPoints ends with `whole = dest` to advance the union; the
// glider does not, because HandleGlider maintains Whole itself as it moves.
//
// Note what is missing: no dontDraw check on the shadow, no test that shadowVisible
// belongs to *this* glider. shadowVisible is one room-scoped flag for both players
// (Room.ShadowVisible), so in a two-player game both gliders have a shadow or
// neither does.
func (w *World) RenderGlider(thisGlider *player.Glider, oneOrTwo bool) {
	if thisGlider.DontDraw {
		return
	}

	which := 0
	if thisGlider.Facing != player.FaceRight {
		which = 1
	}

	oh, ov := w.R.V.OriginH, w.R.V.OriginV

	if w.R.ShadowVisible {
		dest := thisGlider.DestShadow.Offset(oh, ov)

		src := shadowSrc[which]
		switch thisGlider.Mode {
		case player.GliderComingUp, player.GliderGoingDown:
			src.Right = src.Left + dest.Wide()
		case player.GliderComingDown:
			src.Left = src.Right - dest.Wide()
		}
		if w.ShadowSrc != nil {
			w.R.Work.Copy(w.ShadowSrc, Rect(src), Rect(dest), render.Masked)
		}

		whole := thisGlider.WholeShadow.Offset(oh, ov)
		w.AddRectToWorkRects(whole)
		w.AddRectToBackRects(dest)
	}

	dest := thisGlider.Dest.Offset(oh, ov)

	if sheet := w.gliderSheet(oneOrTwo, !w.TwoPlayer && w.ShowFoil); sheet != nil {
		w.R.Work.Copy(sheet, Rect(thisGlider.Src), Rect(dest), render.Masked)
	}

	src := thisGlider.Whole.Offset(oh, ov)
	w.AddRectToWorkRects(src)
	w.AddRectToBackRects(dest)
}

// ---------------------------------------------------------------------------
// CopyRectsQD (Render.c:616-635)
// ---------------------------------------------------------------------------

// CopyRectsQD replays both dirty-rect lists. Publish first, erase second.
//
// The order is the whole protocol. Loop one copies the work map -- room plus this
// frame's moving things -- to the screen, which is the only moment in the frame the
// player's view changes. Loop two copies the clean background back over the work map,
// which un-draws everything loop one just showed, so that the next frame's
// compositors start from a room with nothing moving in it.
//
// Reversing them would erase the frame before showing it and the game would render a
// static room. Merging them into one loop over one list would erase each rect
// immediately after publishing it, which works only while no two rects overlap.
//
// Neither counter is reset here; RenderFrame does that after this returns, which is
// what lets a caller inspect the lists between the copy and the clear. The
// scoreboard appears in neither list -- Scoreboard.c writes the screen directly, and
// rows 460..480 are outside both offscreens.
func (w *World) CopyRectsQD() {
	for _, r := range w.Work2Main {
		w.Main.Copy(w.R.Work, r, r, render.SrcCopy)
	}

	// The frame becomes visible here. In the original it became visible inside the
	// loop above, one CopyBits at a time, because mainWindow *was* the frame buffer;
	// the port batches it because every backend it can have takes a whole image. The
	// second loop does not touch Main, so presenting between the loops rather than
	// after them is the same pixels either way -- it is placed here because this is
	// where the C's last visible write happened.
	w.present()

	for _, r := range w.Back2Work {
		w.R.Work.Copy(w.R.Back, r, r, render.SrcCopy)
	}
}

// ---------------------------------------------------------------------------
// The frame limiter (Render.c:662-665)
// ---------------------------------------------------------------------------

// awaitFrame is the two statements at the end of RenderFrame:
//
//	while (TickCount() < nextFrame) { }
//	nextFrame = TickCount() + kTicksPerFrame;
//
// Two things about it decide how the whole game behaves under load.
//
// **There is no catch-up.** nextFrame is reseeded from the clock *after* the wait,
// not by adding kTicksPerFrame to the previous deadline. So a frame that overruns its
// two ticks does not cause the next one to be rushed or dropped: the game simply runs
// slower for as long as it is overrunning, and every frame still advances gameFrame
// by exactly one. That is why the original's timers are all in frames and why a slow
// Mac played the same game in more wall-clock seconds rather than a different game.
// Any future variable-timestep work has to change this line and nothing else.
//
// **The blit is outside the budget.** CopyRectsQD runs after this returns, so the
// time spent publishing 47 rects is not counted against the two ticks. The budget
// covers composition only.
//
// The wait is skipped entirely when TickCount is nil, and that is required rather
// than an optimisation. With no tick source, Ticks() derives the clock from Frame
// (readylevel.go), which cannot advance inside this loop -- and Transit.c calls
// RenderFrame up to four times within a single frame, so the second call would spin
// for ever. The reseed happens either way, so a headless run keeps the same
// nextFrame arithmetic a windowed one has and a replay can pin it.
//
// WaitTick is the loop body. nil is the C's empty body, i.e. a busy-wait, which is
// what the original does and what a faithful build does. A released build should pass
// a sleeping implementation instead -- see docs/IMPROVEMENTS.md 2.17; that is a
// change of *how* the frame is waited for, not of when it ends, so it is a hook here
// rather than an edit.
func (w *World) awaitFrame() {
	if w.TickCount != nil {
		for w.Ticks() < w.NextFrame {
			if w.WaitTick == nil {
				continue
			}
			w.WaitTick()
		}
	}
	w.NextFrame = w.Ticks() + TicksPerFrame
}

// ---------------------------------------------------------------------------
// RenderFrame (Render.c:639-671)
// ---------------------------------------------------------------------------

// RenderFrame draws one frame and shows it.
//
// The order of the calls below is transcribed exactly, because it is the z-order:
// each renderer appends to work2MainRects and CopyRectsQD replays that list in
// order, so a renderer called later covers one called earlier. Reordering two lines
// here changes what the player sees with no other symptom.
//
// Reading the order: the reflection goes down first, under everything, because a
// mirror is behind the room. Then grease, which is a floor stain. Then the
// background animations -- pendulums always, and then flames *or* stars depending on
// frame parity, which is the only parity test in the function and is how those two
// run at 15fps while everything else runs at 30. Then the dynamics (the moving
// furniture and enemies), the score particles and sparkles, then the gliders on top
// of all of it, and finally shreds and bands, which are the two things allowed to
// cover a glider.
//
// What RenderFrame does *not* do is bookkeeping. It never writes gameFrame and never
// writes evenFrame: PlayGame's loop head owns both, in the two statements before the
// event pump (Play.c:434-435), which is why Transit.c can call this function four times
// inside one frame to animate a transition without the animation clocks advancing.
//
// evenFrame is a stored flag and not `gameFrame & 1`. It has four writers -- the
// per-frame toggle, a stalled ball or fish being kicked into motion mid-frame
// (Dynamics2.c:420, whose write this function reads later in the same frame), and
// the kBall and kFish arms of AddDynamicObject at room-build time -- so a room with
// a ball in it desynchronises the flame/star alternation from frame parity and stays
// desynchronised. See World.EvenFrame.
func (w *World) RenderFrame() {
	// RenderFrames is the port's own counter and has no counterpart in the C. It
	// exists because "how many times was a frame rendered" is otherwise unobservable
	// -- gameFrame counts *game* frames, and a transition renders several per game
	// frame -- and the fidelity replays pin it.
	w.RenderFrames++

	if w.R.HasMirror {
		w.DrawReflection(&w.P1, true)
		if w.TwoPlayer {
			w.DrawReflection(&w.P2, false)
		}
	}

	w.HandleGrease()
	w.RenderPendulums()

	if w.EvenFrame {
		w.RenderFlames()
	} else {
		w.RenderStars()
	}

	w.RenderDynamics()
	w.RenderFlyingPoints()
	w.RenderSparkles()

	w.RenderGlider(&w.P1, true)
	if w.TwoPlayer {
		w.RenderGlider(&w.P2, false)
	}

	w.RenderShreds()
	w.RenderBands()

	w.awaitFrame()
	w.CopyRectsQD()

	w.Work2Main = w.Work2Main[:0]
	w.Back2Work = w.Back2Work[:0]
}

// ---------------------------------------------------------------------------
// The nine renderers that belong to later sub-stages
// ---------------------------------------------------------------------------
//
// These are named no-ops, called from RenderFrame in exactly their real positions.
//
// They are here now, empty, rather than added when their subsystems land, because
// registration order is z-order (see RenderFrame) and an empty call in the right
// place is a correct z-order with nothing in it. Filling one in later is then a
// change to one function body and cannot move anything else. Adding the call later
// is the failure mode this avoids: it would compile, run, and draw the bands behind
// the glider.
//
// Each names its sub-stage and what the game looks like without it, because that is
// the difference between a stub and a gap.

// HandleGrease is Grease.c:79-121 and belongs to 1.5e. Without it a spilt bucket
// leaves no slick, so the glider never slides.
//
// Grease.c:105-118 is worth reading first whatever order these get written in: it is the
// one place the Carbon conversion got the port switching right, and it is the model
// HandleOutlet should have followed. docs/IMPROVEMENTS.md 2.34.
func (w *World) HandleGrease() {}

// RenderPendulums is Render.c:260-322 and belongs to 1.5f. Without it a grandfather
// clock's pendulum stands still and the room is silent -- it owns the tik/tok sounds,
// one pair per swing across all the room's pendulums, and clockFrame, which is a
// phase counter and not a frame clock.
func (w *World) RenderPendulums() {}

// RenderFlames is Render.c:193-255 and belongs to 1.5f: candles, tiki torches and
// barbecue coals, three independent loops over three tables. Without it those three
// show their first frame and never move. It registers no back rects -- the blit
// comes from savedMaps and is opaque, so each frame erases the last by covering it.
func (w *World) RenderFlames() {}

// RenderStars is Render.c:420-448 and belongs to 1.5f. Same shape as the flames and
// the other half of the parity alternation, so a star and a candle in one room are
// each animated on alternate frames.
func (w *World) RenderStars() {}

// RenderDynamics, RenderFlyingPoints and RenderSparkles landed with 1.5c and live in
// dynamics.go and sparkles.go.

// RenderShreds is Render.c:559-612 and belongs to 1.5f: the confetti a paper
// shredder makes of a glider. Without it a shredded glider dies with no animation.
// Note when it lands that kShredSound is replayed on *every* growth frame, not once.
func (w *World) RenderShreds() {}

// RenderBands is Render.c:534-555 and belongs to 1.5e. Without it firing a rubber
// band costs nothing and draws nothing; see World.AddBand's stub, which returns false
// so that it also costs no ammunition.
func (w *World) RenderBands() {}
