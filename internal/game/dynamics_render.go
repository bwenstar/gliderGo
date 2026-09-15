package game

// The seven dynamic renderers: Dynamics.c:112-286.
//
// One shape, seven times: blit the current cel into the *work* map at Dest, register
// Dest on the back list so next frame erases it, and register Whole on the work list so
// the screen copy covers the trail. Six lines, and the whole of RenderBalloon,
// RenderCopter, RenderDart, RenderBall and RenderDrip is those six lines with a different
// sheet and a different strip.
//
// They are here rather than in dynamics_movers.go because Dynamics.c is where the
// original keeps them -- all seven, including RenderToast, whose handler is in
// dynamics_appliances.go with the other Dynamics.c appliances.
//
// ---------------------------------------------------------------------------
// Room-local, unlike the appliances
// ---------------------------------------------------------------------------
//
// Every rect here is offset by playOrigin at blit time, because all six movers keep Dest
// in room-local coordinates. The appliances are the mirror image: their registration adds
// the origin once and their handlers store screen coordinates, so they never offset. The
// two conventions meet in CheckDynamicCollision's doOffset flag, and nowhere else.
//
// ---------------------------------------------------------------------------
// Three deviations from the shape
// ---------------------------------------------------------------------------
//
//	RenderToast  clips the cel to the toaster slot, so bread rises out of the slot
//	             rather than appearing whole. Written longhand for it.
//	RenderBall   are *not* gated on Moving, so a ball at rest and a drop hanging on
//	RenderDrip   the ceiling are still drawn every frame -- which they have to be,
//	             because neither is in the static background.
//	RenderFish   switches blitter: masked while leaping, unmasked while resting.
//
// The Moving gate on the other five is what makes their static room draw sufficient: a
// waiting balloon, copter or dart is painted into the background by the composition (see
// internal/render/objectdraw2.go) and the renderer leaves it alone until it launches.

import (
	"glidergo/internal/game/player"
	"glidergo/internal/render"
)

// renderDinah is the six-line shape, shared by six of the seven.
//
// **The two registrations use different rects, and that is the pattern.** Dest -- where
// the sprite is now -- goes on the back list, so exactly that much of the work map is
// restored from the clean background next frame. Whole -- the union of now and last frame
// -- goes on the work list, so the screen copy covers the trail the sprite left. Getting
// them the wrong way round leaves either a permanent smear or a flicker, and every
// compositor in the port follows the same convention (see RenderGlider).
//
// A nil sheet still registers both rects. That is deliberate: a missing art file should
// leave a hole rather than desynchronise the dirty-rect lists from what the handlers
// think is on screen.
func (w *World) renderDinah(who int16, art *render.Surface, src Rect, mode render.CopyMode) {
	dest := render.Offset(w.Dinahs[who].Dest, w.R.V.OriginH, w.R.V.OriginV)
	if art != nil {
		w.R.Work.Copy(art, src, dest, mode)
	}
	w.AddRectToBackRects(player.Rect(dest))

	whole := render.Offset(w.Dinahs[who].Whole, w.R.V.OriginH, w.R.V.OriginV)
	w.AddRectToWorkRects(player.Rect(whole))
}

// RenderToast is Dynamics.c:112-139: the shape, plus the slot clip.
//
// **HVel is not a velocity for a toaster; it is the clip line** -- the Y of the top of
// the toaster's slot, recorded at registration. `vClip = Dest.Bottom - HVel` is how far
// the bread's bottom has sunk past that line, and while it is positive both the source
// and the destination lose that many rows from the bottom, so the slice appears to emerge
// from the slot instead of floating in front of it. Once the bread has cleared the slot
// vClip goes negative and the full cel is drawn.
//
// **The clip can consume the whole cel** on the frame the bread is launched, when Dest is
// still entirely inside the slot: dest.Bottom drops below dest.Top and the rect inverts.
// QuickDraw draws nothing for an inverted rect, and Surface.Copy returns early on any
// non-positive extent, so the two agree -- but the rect is still registered on both lists,
// exactly as the C registers it, because CopyRectsQD has to see the same rects the
// original's did.
//
// The clip is applied to `src` and `dest` and never to Whole, so the work rect covers the
// unclipped travel. That is what erases the bread's trail once it is above the slot.
func (w *World) RenderToast(who int16) {
	if !w.Dinahs[who].Moving {
		return
	}

	dest := render.Offset(w.Dinahs[who].Dest, w.R.V.OriginH, w.R.V.OriginV)
	src := render.BreadSrc[w.Dinahs[who].Frame]
	vClip := w.Dinahs[who].Dest.Bottom - w.Dinahs[who].HVel
	if vClip > 0 {
		src.Bottom -= vClip
		dest.Bottom -= vClip
	}

	if art := w.R.A.Strip("toast"); art != nil {
		w.R.Work.Copy(art, src, dest, render.Masked)
	}
	w.AddRectToBackRects(player.Rect(dest))

	whole := render.Offset(w.Dinahs[who].Whole, w.R.V.OriginH, w.R.V.OriginV)
	w.AddRectToWorkRects(player.Rect(whole))
}

// RenderBalloon is Dynamics.c:143-166. Cels 0..5 are the balloon and 6..7 the burst, so
// this one function draws both a rising balloon and a popped one falling.
func (w *World) RenderBalloon(who int16) {
	if !w.Dinahs[who].Moving {
		return
	}
	w.renderDinah(who, w.R.A.Sheet("balloon"), render.BalloonSrc[w.Dinahs[who].Frame], render.Masked)
}

// RenderCopter is Dynamics.c:170-193. Ten cels: 0..7 spin, 8..9 crumple.
func (w *World) RenderCopter(who int16) {
	if !w.Dinahs[who].Moving {
		return
	}
	w.renderDinah(who, w.R.A.Sheet("copter"), render.CopterSrc[w.Dinahs[who].Frame], render.Masked)
}

// RenderDart is Dynamics.c:197-220. Frame is a *direction* here, not an animation phase:
// 0 and 1 are the leftward dart flying and crumpled, 2 and 3 the rightward pair.
func (w *World) RenderDart(who int16) {
	if !w.Dinahs[who].Moving {
		return
	}
	w.renderDinah(who, w.R.A.Sheet("dart"), render.DartSrc[w.Dinahs[who].Frame], render.Masked)
}

// RenderBall is Dynamics.c:224-244, and **it has no Moving gate.**
//
// It cannot have one: a ball that has run down its bounces sits on the floor forever, is
// still lethal (HandleBall tests collisions outside its own Moving check), and is not in
// the static background -- the composition draws nothing for a kBall, because a ball's
// resting position is wherever it stopped rather than where the author put it. So the
// renderer keeps drawing it, and keeps spending two rect slots doing so, for the rest of
// the room's life. That standing cost is measured against the 47-rect budget in
// docs/IMPROVEMENTS.md 2.11 -- SpacePods room 55 holds 16 of these three types and spends
// 17 of 47 before the glider moves.
//
// Cel 0 is the round ball and cel 1 the squashed one, which HandleBall sets for the
// single frame after a bounce.
func (w *World) RenderBall(who int16) {
	w.renderDinah(who, w.R.A.Sheet("ball"), render.BallSrc[w.Dinahs[who].Frame], render.Masked)
}

// RenderDrip is Dynamics.c:248-268, ungated for the same reason as the ball: cel 3, the
// hanging drop, has to be drawn every resting frame.
//
// The composition *does* paint cel 3 into the static background (objectdraw2.go's
// DrawDrip), so for one frame after the room is composed the two agree -- and then the
// first back->work erase would remove it if the renderer were gated. Hence no gate.
//
// This is also where the drip's stale Whole shows up: while the drop hangs, Whole is
// still the union of the last fall, so every resting frame registers the full column as a
// work rect -- the most expensive of the three standing costs. See HandleDrip, and
// docs/IMPROVEMENTS.md 2.11 for what it does to the rect budget.
func (w *World) RenderDrip(who int16) {
	w.renderDinah(who, w.R.A.Sheet("drip"), render.DripSrc[w.Dinahs[who].Frame], render.Masked)
}

// RenderFish is Dynamics.c:272-286, the only renderer that changes blitter.
//
// A leaping fish is masked, so the room shows through around it. A resting one is
// **unmasked**, so the cel's own surround -- the water it sits in -- is painted too, and
// the ripple animation the four idle cels draw is visible. Masking a resting fish would
// leave the background where the water should be, which is why the two branches of the C
// differ only in the blit call.
//
// **The strip's own opacity is what makes the switch load-bearing.** Measured against the
// shipped art by TestRenderFishSwitchesBlitters: the four idle cels are solid 16x16 water
// tiles, 0 of 256 pixels transparent, so Masked and SrcCopy would draw them identically and
// the mode only matters once the fish is in the air; the four leap cels are roughly 48%
// transparent (120, 124, 124 and 121 pixels of 256), which is the room showing through
// around the fish. So a test that holds a *resting* cel cannot tell the two modes apart --
// it has to hold a leap cel and suppress Moving.
//
// Both branches register the same two rects, so the switch is purely a transfer mode.
func (w *World) RenderFish(who int16) {
	mode := render.SrcCopy
	if w.Dinahs[who].Moving {
		mode = render.Masked
	}
	w.renderDinah(who, w.R.A.Strip("fish"), render.FishSrc[w.Dinahs[who].Frame], mode)
}
