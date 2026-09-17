package game

// The screen: the five CopyRect* primitives (Render.c:695-736) and the three
// screen transitions (Transitions.c).
//
// Three surfaces and five ways to copy between them. backSrcMap is the clean
// composed room, workSrcMap is that room with this frame's moving things drawn on
// top, and mainWindow is what the player sees. The whole rendering strategy of
// Glider PRO is those three plus a list of dirty rectangles:
//
//	back -> work    restore the background under everything that moved
//	work -> main    show the result
//
// and the two lists in RenderFrame are exactly those two directions. The other
// three primitives exist for the handful of places that need a single rect
// immediately rather than at the end of the frame -- a dialogue being torn down,
// a glider being erased -- and each has one or two callers.
//
// All five take rects in *screen* coordinates and use the same rect for source
// and destination, which is only correct because all three surfaces are cornered
// at (0,0). The original relies on the same thing: workSrcRect, backSrcRect and
// justRoomsRect are all ZeroRectCorner(houseRect) (StructuresInit2.c:153-162), and
// mainWindow's port origin is the window's top left. See player.Env's note on
// AddRectToWorkRects for why the offset is the caller's job and not theirs.

import (
	"github.com/bwenstar/gliderGo/internal/game/player"
	"github.com/bwenstar/gliderGo/internal/render"
)

// present makes Main visible, if anything is watching. See World.Present for the
// four call sites and why they are the four.
func (w *World) present() {
	if w.Present != nil {
		w.Present()
	}
}

// ---------------------------------------------------------------------------
// Render.c:695-736 -- the five CopyRect primitives
// ---------------------------------------------------------------------------

// CopyRectBackToWork is Render.c:695: restore one rect of clean background.
func (w *World) CopyRectBackToWork(r Rect) {
	w.R.Work.Copy(w.R.Back, r, r, render.SrcCopy)
}

// CopyRectWorkToBack is Render.c:704: promote one rect of the work map into the
// background, so that whatever is in it stops being erased every frame.
//
// This is how a permanent change to a room gets made -- a light switched, a prize
// taken, grease spilt. The two directions are not symmetric in how often they are
// used: back->work runs 47 times a frame, work->back runs when the room changes.
func (w *World) CopyRectWorkToBack(r Rect) {
	w.R.Back.Copy(w.R.Work, r, r, render.SrcCopy)
}

// CopyRectWorkToMain is Render.c:713: put one rect on screen now.
//
// It takes SCREEN coordinates. Input.c:70 hands it workSrcRect, which is already
// screen-cornered, while Modes.c:90 and Play.c:717 hand it a glider rect they have
// just offset by playOriginH/V -- so the offset cannot be folded in here. That is
// the whole content of player.Env's contract note, and the reason PlayOriginH and
// PlayOriginV are on the interface at all.
//
// It does not present. The player is not shown a half-finished erase: the C's next
// visible moment after any of this function's callers is a RenderFrame or a modal
// dialogue, and both present for themselves.
func (w *World) CopyRectWorkToMain(r player.Rect) {
	rr := Rect(r)
	w.Main.Copy(w.R.Work, rr, rr, render.SrcCopy)
}

// CopyRectMainToWork is Render.c:722 and CopyRectMainToBack is :731: read the
// screen back into an offscreen.
//
// They are the odd pair. Reading the frame buffer back is something a modern
// renderer never wants to do, and the original does it for one reason: a dialogue
// or a menu has been drawn over the window by the Toolbox, which knows nothing
// about workSrcMap, and the pixels it left are the only copy of what is now on
// screen. They are kept because 1.7's shell will need the same trick for the same
// reason, and because leaving them out would make the transcription of the
// dialogue teardown look like it had a choice.
func (w *World) CopyRectMainToWork(r Rect) {
	w.R.Work.Copy(w.Main, r, r, render.SrcCopy)
}

// CopyRectMainToBack is Render.c:731. See CopyRectMainToWork.
func (w *World) CopyRectMainToBack(r Rect) {
	w.R.Back.Copy(w.Main, r, r, render.SrcCopy)
}

// ---------------------------------------------------------------------------
// Utilities.c:340-349 -- LoadScaledGraphic
// ---------------------------------------------------------------------------

// loadScaledGraphic is Utilities.c:340-349: draw a picture stretched to fill a
// rect.
//
// render.Scene has one of these already, and it is not usable here: it resolves the
// resource itself and always draws into Back, which is right for its one caller
// (DrawFloorSupport's manhole) and wrong for all four of this package's, whose
// destinations are the work map and the screen.
//
// So this one takes the destination and the *already-resolved* picture. That is
// deliberate rather than merely convenient: the choice between Pict, Plate, UI and
// MaskedPlate is a statement about whether a house may override the art, it differs
// at every call site here, and burying it inside a helper would hide the one
// interesting thing about each call. render.uiMaskPairs' comment asks for exactly
// this. See banner.go and gameover.go for the four.
//
// A nil picture draws nothing and is not an error, which is Plate's contract: art
// that sits over a running game is optional, and a house that omits it should show
// the room rather than a red alert.
func loadScaledGraphic(dst *render.Surface, art *render.Surface, theRect Rect) {
	if art == nil {
		return
	}
	dst.Copy(art, art.Bounds(), theRect, render.SrcCopy)
}

// ---------------------------------------------------------------------------
// Transitions.c -- the three screen transitions
// ---------------------------------------------------------------------------

// DumpScreenOn is Transitions.c:139-144: one whole-rect work->main copy.
//
// Four callers, all in Play.c (:172, :177, :181 in NewGame's opening ladder). It is
// the transition with no transition -- the C's comment history shows DissBits
// having been here and having been commented out -- and it is what the game uses
// when the player must simply be looking at the new room on the next frame.
func (w *World) DumpScreenOn(r Rect) {
	w.Main.Copy(w.R.Work, r, r, render.SrcCopy)
	w.present()
}

// kWipeRectThick is Transitions.c:76: the wipe advances four pixels at a time.
const kWipeRectThick int16 = 4

// WipeScreenOn is Transitions.c:74-135: reveal the new room by sliding a 4-pixel
// bar across the screen in the direction the glider travelled.
//
// This is the room-change transition, and it is the only animated one the shipped
// game uses. Direction is the *arrival* direction as MoveRoomToRoom understands
// it, so a glider walking out of the right of a room wipes ToRight -- the bar
// starts at the right edge and walks left, uncovering the room the glider is now
// in from the side it came from.
//
// Two things about it are worth stating because they look like bugs:
//
// The strip counts are asymmetric. Vertical wipes take ((bottom-top)/4)+1 strips,
// which at a 460-tall house rect is 116; horizontal wipes take
// workSrcRect.right/4, which is 160 at 640 wide -- and note it reads workSrcRect
// rather than theRect, so a caller passing a narrower rect still gets 160 strips.
//
// Only top and bottom are clamped inside the loop; left and right are not. So a
// horizontal wipe's bar walks clean off the far edge for its last few strips and
// the copies are no-ops, absorbed by CopyBits' own clipping. Both are reproduced,
// because the strip count is the duration and the duration is what the transition
// feels like.
//
// docs/IMPROVEMENTS.md 2.4: at memory speed 116 whole-screen presents are a blink
// rather than a wipe. Pacing it to a fixed wall-clock duration is a release-build
// change and a deliberate divergence, so it is not made here.
func (w *World) WipeScreenOn(direction int16, theRect Rect) {
	var wipeRect Rect
	var hOffset, vOffset, count int16

	switch direction {
	case player.Above:
		wipeRect = theRect
		wipeRect.Bottom = wipeRect.Top + kWipeRectThick
		hOffset = 0
		vOffset = kWipeRectThick
		count = ((theRect.Bottom - theRect.Top) / kWipeRectThick) + 1

	case player.ToRight:
		wipeRect = theRect
		wipeRect.Left = wipeRect.Right - kWipeRectThick
		hOffset = -kWipeRectThick
		vOffset = 0
		count = w.R.V.WorkRect.Right / kWipeRectThick

	case player.Below:
		wipeRect = theRect
		wipeRect.Top = wipeRect.Bottom - kWipeRectThick
		hOffset = 0
		vOffset = -kWipeRectThick
		count = ((theRect.Bottom - theRect.Top) / kWipeRectThick) + 1

	case player.ToLeft:
		wipeRect = theRect
		wipeRect.Right = wipeRect.Left + kWipeRectThick
		hOffset = kWipeRectThick
		vOffset = 0
		count = w.R.V.WorkRect.Right / kWipeRectThick
	}

	for i := int16(0); i < count; i++ {
		w.Main.Copy(w.R.Work, wipeRect, wipeRect, render.SrcCopy)
		w.present()

		wipeRect = render.Offset(wipeRect, hOffset, vOffset)
		if wipeRect.Top < theRect.Top {
			wipeRect.Top = theRect.Top
		}
		if wipeRect.Bottom > theRect.Bottom {
			wipeRect.Bottom = theRect.Bottom
		}
	}
}

// PourScreenOn is Transitions.c:18-70, and it is dead code in the shipped game.
//
// It reveals the room by dropping 16x20 chips down 40 columns in random order, and
// it is not called from anywhere: a grep of all 68 sources finds the definition and
// nothing else. Both of the game's real transitions are DumpScreenOn and
// WipeScreenOn.
//
// It is not transcribed, and that is a decision rather than an omission. The
// function's column picker is a rejection loop over RandomInt(colWide), so running
// it would draw an unbounded number of values out of the same stream that seeds
// every flame phase, every pendulum start and every balloon's jitter. Wiring it in
// -- even behind an option -- would shift that stream and change the appearance of
// every room composed afterwards, so any future use has to draw from a separate
// generator. This comment is here so that whoever finds the C function does not
// assume it was missed.
