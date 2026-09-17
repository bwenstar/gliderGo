package game

// The effects layer: sparkles and flying points. DynamicMaps.c:165-254 (the two
// producers) and Render.c:322-418 (the two renderers).
//
// Four functions and two three-slot tables, kept together because the producer and the
// renderer of each pair only make sense as a pair -- one picks a slot and centres a rect
// in it, the other walks the animation and hands the slot back.
//
// ---------------------------------------------------------------------------
// Free lists, not stacks
// ---------------------------------------------------------------------------
//
// Both tables are three fixed slots with `Mode == -1` meaning free. The producer scans
// for the first free slot; the renderer sets Mode back to -1 when the animation ends.
// **The zero value is therefore not "empty"** -- a table that has never been swept reads
// as three live effects at frame 0, and AddSparkle will refuse to add anything because it
// finds no free slot. InitGarbageRects is the only sweeper and it runs on every room
// change (see ReadyLevel), which is what makes the tables usable; the original has the
// same requirement, because StructuresInit2.c NewPtr's them without clearing.
//
// The counters are redundant with the tables -- NumSparkles is always the number of slots
// whose Mode is not -1 -- and they are kept because they are what the renderers'
// early-out reads. Three slots is not a budget worth optimising; the counter exists so
// that a frame with no effects costs one comparison.
//
// ---------------------------------------------------------------------------
// Screen space, from the moment they are created
// ---------------------------------------------------------------------------
//
// Both producers add playOrigin to the rect they are handed and store the result, so
// neither renderer offsets anything. That is the opposite of the six movers and the same
// as the seven appliances, and it is why every AddSparkle call site passes an *un-offset*
// rect -- see enemyRetire, where the work rect on the line above is offset and the sparkle
// is not.
//
// The C's producers offset the caller's rect *through the pointer*, so they mutate it.
// Four call-site families exist and none can observe it: three pass a local copy, and
// RenderShreds (Render.c:603) passes `shreds[i].bounds` on the last frame of a shred's
// life, after which nothing reads it again. Taking Rect by value here is therefore exact,
// not a simplification -- but 1.5f's RenderShreds should be checked against this note
// rather than assuming it.

import (
	"github.com/bwenstar/gliderGo/internal/game/player"
	"github.com/bwenstar/gliderGo/internal/render"
)

// MaxFlyingPointsLoop is kMaxFlyingPointsLoop (GliderDefines.h:254): how many times a
// flying number replays its three cels before it is retired. Twenty-four, so a score
// numeral is on screen for 72 frames -- about two and a half seconds -- drifting the whole
// time at half the glider's velocity.
//
// There is no equivalent for sparkles: a sparkle plays its five cels once and is gone.
const MaxFlyingPointsLoop int16 = 24

// SparkleSlot is sparkleType (GliderStructs.h:236-239): one puff of light.
//
// Named SparkleSlot because consts.go already has Sparkle -- the object code for the
// *emitter* that produces these. Two different things with one name in the original;
// HandleSparkleObject is the emitter's handler and RenderSparkles draws these.
//
// Mode is both the cel index and the liveness flag: 0..4 while playing, -1 when the slot
// is free. Bounds is in screen coordinates and never moves -- a sparkle is the one effect
// in the game with no velocity.
type SparkleSlot struct {
	Bounds Rect
	Mode   int16
}

// FlyingPoint is flyingPtType (GliderStructs.h:241-249): a score numeral drifting away
// from whatever the glider just collected.
//
// Start and Stop bracket the three cels of *this* number in the fifteen-cel strip, Mode
// walks between them, and Loops counts the replays. HVel and VVel are half the glider's
// velocity at the moment of collection (Interactions.c:773 and its four siblings), so the
// numerals inherit the player's motion and a stationary glider's points rise straight up.
type FlyingPoint struct {
	Dest, Whole Rect
	Start, Stop int16
	Mode        int16
	Loops       int16
	HVel, VVel  int16
}

// ---------------------------------------------------------------------------
// AddSparkle (DynamicMaps.c:165-193)
// ---------------------------------------------------------------------------

// AddSparkle puts a five-frame puff of light at the centre of theRect, if a slot is free.
//
// **It is centred, not placed.** The sparkle cel is 20x19 and CenterRectInRect keeps that
// size while centring it in whatever rect the caller passed, so a sparkle over a 64-wide
// prize and one over a 24-wide balloon are the same size and both sit on the object's
// centre. That is why callers pass the object's bounds rather than a point.
//
// **Over the cap it silently does nothing.** Three at once is the whole budget for the
// screen, and the fourth request in a frame is dropped rather than queued -- which is
// reachable in the shipped game: a room where two enemies retire on the frame the player
// collects a prize loses one of the three puffs. Kept, because it is the original's
// behaviour and because the alternative (a queue) changes what the player sees in exactly
// the busy rooms where the difference shows.
//
// Twelve callers across five files: the sparkle emitter, the three respawning enemies (in
// and out), the rewards, the switch that removes a prize, and the last frame of a shred.
func (w *World) AddSparkle(theRect Rect) {
	if w.NumSparkles >= MaxSparkles {
		return
	}

	theRect = render.Offset(theRect, w.R.V.OriginH, w.R.V.OriginV)
	centeredRect := render.CenterIn(render.SparkleSrc[0], theRect)

	for i := 0; i < MaxSparkles; i++ {
		if w.Sparkles[i].Mode == -1 {
			w.Sparkles[i].Bounds = centeredRect
			w.Sparkles[i].Mode = 0
			w.NumSparkles++
			break
		}
	}
}

// ---------------------------------------------------------------------------
// AddFlyingPoint (DynamicMaps.c:195-254)
// ---------------------------------------------------------------------------

// AddFlyingPoint starts a score numeral drifting away from theRect.
//
// The switch on `points` is how one 24x120 strip spells five different numbers: each value
// owns three consecutive cels, in descending order of value.
//
//	1000 (and anything else)  0..2
//	 500                      3..5
//	 300                      6..8
//	 250                      9..11
//	 100                      12..14
//
// **The default arm is not a fallback.** kCuckoo is worth 1000 and reaches it through the
// default, and so does a kInvisBonus, whose point value is whatever the author typed --
// so an author who types 700 gets the 1000 art. That is the original's behaviour and it is
// visible in at least one shipped house.
//
// Every caller is a reward (Interactions.c:773, :789, :805, :822, :925), so all five arrived
// with HandleRewards in 1.5d: the three clocks, the cuckoo and the invisible bonus. Nothing
// else in the game makes a flying point.
func (w *World) AddFlyingPoint(theRect Rect, points, hVel, vVel int16) {
	if w.NumFlyingPts >= MaxFlyingPts {
		return
	}

	theRect = render.Offset(theRect, w.R.V.OriginH, w.R.V.OriginV)
	centeredRect := render.CenterIn(render.PointsSrc[0], theRect)

	for i := 0; i < MaxFlyingPts; i++ {
		if w.FlyingPoints[i].Mode != -1 {
			continue
		}
		w.FlyingPoints[i].Dest = centeredRect
		w.FlyingPoints[i].Whole = centeredRect
		w.FlyingPoints[i].Loops = 0
		w.FlyingPoints[i].HVel = hVel
		w.FlyingPoints[i].VVel = vVel
		switch points {
		case 100:
			w.FlyingPoints[i].Start = 12
			w.FlyingPoints[i].Stop = 14
		case 250:
			w.FlyingPoints[i].Start = 9
			w.FlyingPoints[i].Stop = 11
		case 300:
			w.FlyingPoints[i].Start = 6
			w.FlyingPoints[i].Stop = 8
		case 500:
			w.FlyingPoints[i].Start = 3
			w.FlyingPoints[i].Stop = 5
		default:
			w.FlyingPoints[i].Start = 0
			w.FlyingPoints[i].Stop = 2
		}
		w.FlyingPoints[i].Mode = w.FlyingPoints[i].Start
		w.NumFlyingPts++
		break
	}
}

// ---------------------------------------------------------------------------
// RenderFlyingPoints (Render.c:322-380)
// ---------------------------------------------------------------------------

// RenderFlyingPoints moves and draws every live score numeral, and retires the ones that
// have finished.
//
// **The union is grown at the leading edge here, where the movers push the trailing edge
// back.** Both produce the same thing -- a rect covering this frame and last -- because
// the last line of the loop is `Whole = Dest`, so Whole starts each frame as the *previous*
// position and is stretched to reach the new one. The movers cannot do it that way: they
// recompute Whole from Dest every frame, which is what stops it drifting when the velocity
// changes. Here the velocity never changes, so accumulating is safe.
//
// The retire path adds Dest -- not Whole -- to the work list, which is the last erase, and
// hands the slot back. Note it draws nothing on that frame, so the numeral's final cel is
// on screen for one frame less than the arithmetic suggests.
//
// The `Mode > Stop` reset is checked *before* the loop count, so the frame that would have
// drawn cel Stop+1 draws Start instead. That is why a numeral shows exactly three cels per
// loop and not four.
func (w *World) RenderFlyingPoints() {
	if w.NumFlyingPts == 0 {
		return
	}

	for i := 0; i < MaxFlyingPts; i++ {
		if w.FlyingPoints[i].Mode == -1 {
			continue
		}

		if w.FlyingPoints[i].Mode > w.FlyingPoints[i].Stop {
			w.FlyingPoints[i].Mode = w.FlyingPoints[i].Start
			w.FlyingPoints[i].Loops++
		}

		if w.FlyingPoints[i].Loops >= MaxFlyingPointsLoop {
			w.AddRectToWorkRects(player.Rect(w.FlyingPoints[i].Dest))
			w.FlyingPoints[i].Mode = -1
			w.NumFlyingPts--
			continue
		}

		w.FlyingPoints[i].Dest.Left += w.FlyingPoints[i].HVel
		w.FlyingPoints[i].Dest.Right += w.FlyingPoints[i].HVel
		if w.FlyingPoints[i].HVel > 0 {
			w.FlyingPoints[i].Whole.Right = w.FlyingPoints[i].Dest.Right
		} else {
			w.FlyingPoints[i].Whole.Left = w.FlyingPoints[i].Dest.Left
		}

		w.FlyingPoints[i].Dest.Top += w.FlyingPoints[i].VVel
		w.FlyingPoints[i].Dest.Bottom += w.FlyingPoints[i].VVel
		if w.FlyingPoints[i].VVel > 0 {
			w.FlyingPoints[i].Whole.Bottom = w.FlyingPoints[i].Dest.Bottom
		} else {
			w.FlyingPoints[i].Whole.Top = w.FlyingPoints[i].Dest.Top
		}

		if art := w.R.A.Strip("points"); art != nil {
			w.R.Work.Copy(art, render.PointsSrc[w.FlyingPoints[i].Mode],
				w.FlyingPoints[i].Dest, render.Masked)
		}

		w.AddRectToWorkRects(player.Rect(w.FlyingPoints[i].Whole))
		w.AddRectToBackRects(player.Rect(w.FlyingPoints[i].Dest))
		w.FlyingPoints[i].Whole = w.FlyingPoints[i].Dest
		w.FlyingPoints[i].Mode++
	}
}

// ---------------------------------------------------------------------------
// RenderSparkles (Render.c:384-418)
// ---------------------------------------------------------------------------

// RenderSparkles draws every live sparkle and retires the ones that have played out.
//
// The simplest renderer in the game: no movement, one rect for both lists because Dest and
// Whole are the same thing when nothing moves, and five cels played once.
//
// **The five cels are three pictures.** SparkleSrc[0] aliases [4] and [1] aliases [3]
// (StructuresInit.c:397-401), so the strip plays out and back: small, medium, large,
// medium, small. Mode is the index into that palindrome, which is why the cap is
// NumSparkleModes and not a frame count.
//
// A sparkle costs six frames and eleven rect registrations: two a frame for five frames,
// then one to erase. With three live at once that is a third of the 47 usable rect slots,
// which is the practical reason the cap is three.
func (w *World) RenderSparkles() {
	if w.NumSparkles == 0 {
		return
	}

	for i := 0; i < MaxSparkles; i++ {
		if w.Sparkles[i].Mode == -1 {
			continue
		}

		if w.Sparkles[i].Mode >= NumSparkleModes {
			w.AddRectToWorkRects(player.Rect(w.Sparkles[i].Bounds))
			w.Sparkles[i].Mode = -1
			w.NumSparkles--
			continue
		}

		if art := w.R.A.Sheet("bonus"); art != nil {
			w.R.Work.Copy(art, render.SparkleSrc[w.Sparkles[i].Mode],
				w.Sparkles[i].Bounds, render.Masked)
		}

		w.AddRectToWorkRects(player.Rect(w.Sparkles[i].Bounds))
		w.AddRectToBackRects(player.Rect(w.Sparkles[i].Bounds))
		w.Sparkles[i].Mode++
	}
}
