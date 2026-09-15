package game

// The shredded glider: DynamicMaps.c:726-775 (AddAShreddedGlider, RemoveShreds) and
// Render.c:557-612 (RenderShreds).
//
// A glider that flies into a paper shredder is fed through it. HandleShreddingGlider
// (player/handle.go) lowers the sprite into the slot four pixels a frame, clipping it as it
// goes, and on the frame the last of it disappears it calls AddAShreddedGlider. What comes
// out underneath is this: a 40x35 cloud of confetti that grows out of the slot, falls, and
// puffs into a sparkle.
//
// ---------------------------------------------------------------------------
// Fifty-four frames in two arms
// ---------------------------------------------------------------------------
//
// The two arms of RenderShreds are two different animations sharing one counter.
//
//	frame 0        **growth**, 35 frames. The rect's bottom edge moves down one pixel a
//	               frame with the top pinned, and the blit shows the *bottom* `high` rows
//	               of the sprite -- so the confetti slides out of the slot bottom-first,
//	               the way paper leaves a shredder. kShredSound plays on every one of
//	               these 35 frames.
//	frame 1..19    **fall**, 19 frames. The whole rect drops four pixels a frame. The
//	               first eighteen draw the full sprite. The nineteenth is the one whose
//	               increment reaches 20: it moves the cloud a last time, draws nothing,
//	               and emits AddSparkle plus kFadeOutSound at the position it moved to.
//	after          nothing. The entry is inert and holds its slot until RemoveShreds or
//	               the next DrawLocale.
//
// The sparkle is the last frame of the fall and not a frame of its own -- the increment and
// the test share an iteration -- which is why the total is 54 and not 55. Counting it twice
// is the natural mistake and it is the one an off-by-one in the fall arm would look like.
//
// The inert tail is not a leak in the C -- the table is four fixed-size structs -- but it is
// why the cap matters: four shredders' worth of dead entries will refuse a fifth cloud even
// though nothing is on screen.
//
// ---------------------------------------------------------------------------
// Two places the original is wrong, and what the port does about them
// ---------------------------------------------------------------------------
//
// **AddAShreddedGlider can write one element past the end of the table.** The guard is
// `if (numShredded > kMaxShredded) return;` with kMaxShredded == 4, so numShredded == 4
// passes it and the function writes shreds[4] of a four-element allocation and leaves the
// counter at 5. In the 1994 build `shreds` is a NewPtr of exactly `sizeof(shredType) *
// kMaxShredded` (Environ.c:654), so those twenty bytes belong to whatever the Memory
// Manager put next. Reaching it needs four live clouds and a fifth glider shredded before
// any of them is removed, which is why it shipped.
//
// The port tests `>=`. That is a deliberate divergence and the only one in this file: a
// fifth cloud is dropped rather than written out of bounds. It is recorded in
// docs/IMPROVEMENTS.md 2.43; the alternative, reproducing the overrun with a five-element
// array, would be reproducing a bug whose observable behaviour is "corrupt something else".
//
// It is also reported at run time. The guard goes through `badIndex`, which is otherwise
// only used in front of reads, so a refusal is counted in World.Diag and named by
// `glidertool replay` -- the port declining a write the original performed is exactly the
// class of thing a bug report should be able to state.
//
// **RemoveShreds removes one cloud, not all of them.** The name and the caller both read as
// "clear the table" -- OffAMortal calls it under `if (numShredded > 0)` -- but the body
// picks the single entry with the largest frame and swap-removes it. So a death with two
// clouds on screen leaves one behind, still animating, while the new glider fades in over
// it. Transcribed as-is: it is visible in the original, it is reachable in any house with
// two shredders in one room, and "fixing" it changes what a replay looks like.
//
// Note the selection is `frame > largest` starting from largest = 0, so an entry still in
// its growth arm (frame == 0) can never be chosen. A death with exactly one cloud on screen
// and that cloud still growing removes **nothing** and returns with the counter unchanged --
// so OffAMortal's guard fires, RemoveShreds no-ops, and the confetti keeps growing through
// the respawn.

import (
	"glidergo/internal/game/player"
	"glidergo/internal/render"
)

// MaxShredded is kMaxShredded (GliderDefines.h:264). Four clouds at once, shared by every
// shredder in the locale.
const MaxShredded = 4

// ShredGrowHeight is how tall the cloud grows before it starts to fall -- Render.c:572's
// `if (high >= 35)`, which is also the full height of shredSrcRect. The two being equal is
// what makes the growth arm end exactly as the last row of the sprite becomes visible.
const ShredGrowHeight int16 = 35

// ShredLastFrame is Render.c:587 and :594's 20: the frame the fall ends on and the sparkle
// is emitted. Nothing is drawn on it.
const ShredLastFrame int16 = 20

// Shred is one entry in shreds[] (Externs.h's shredType): where the cloud is and how far
// through its 55 frames.
type Shred struct {
	// Bounds is **room-local**, unlike almost every other rect that reaches a blit in
	// this package. RenderShreds offsets a copy by playOrigin for the draw and leaves
	// this one alone, which is what lets the cloud survive a scroll.
	Bounds Rect

	// Frame is 0 through the growth arm, 1..19 through the fall, and 20 once the cloud
	// is spent. It is not a cel index -- the sprite is one picture, and what the
	// animation varies is how much of it is showing and where.
	Frame int16
}

// ShredSrc is shredSrcRect (StructuresInit.c:556): the whole of the `shred` sheet, 40x35.
// The growth arm slices rows off its top; the fall arm uses it entire.
var ShredSrc = render.SetRect(0, 0, 40, 35)

// AddAShreddedGlider is DynamicMaps.c:726-742: start a confetti cloud under a shredder.
//
// The offsets place it relative to the glider's own rect at the moment the last of the
// sprite vanished -- four pixels in from the left, fourteen down from the top -- and the
// cloud starts with **zero height**: bottom is set equal to top, and the growth arm's first
// act is to add one. So the first frame shows a single row of confetti.
//
// The `>=` is the port's, not the original's. See the file comment.
//
// It goes through badIndex rather than being written as a bare `>=` so that the refusal is
// *reported*: this is the only site in the port where the guard stands in front of a write
// rather than a read, and "the original would have corrupted twenty bytes here" is worth
// more in a bug report than a cloud that quietly failed to appear. See guards.go and
// docs/IMPROVEMENTS.md 2.33 and 2.43.
func (w *World) AddAShreddedGlider(r player.Rect) {
	if w.badIndex(devShred, int(w.NumShredded), MaxShredded) {
		return
	}

	s := &w.Shreds[w.NumShredded]
	s.Bounds.Left = r.Left + 4
	s.Bounds.Right = s.Bounds.Left + 40
	s.Bounds.Top = r.Top + 14
	s.Bounds.Bottom = s.Bounds.Top
	s.Frame = 0

	w.NumShredded++
}

// RemoveShreds is DynamicMaps.c:744-775: drop the single most-advanced confetti cloud.
//
// One, not all -- see the file comment, and read the caller in OffAMortal with that in mind.
//
// The removal is a swap: the last live entry is copied over the chosen one and the vacated
// slot's Frame is zeroed. Zeroing Frame and not Bounds is deliberate and is why AddA-
// ShreddedGlider writes all four bounds fields rather than offsetting them -- a slot handed
// back to the table still holds the rect of the cloud that left it.
func (w *World) RemoveShreds() {
	largest, who := int16(0), -1
	for i := int16(0); i < w.NumShredded; i++ {
		if w.Shreds[i].Frame > largest {
			largest = w.Shreds[i].Frame
			who = int(i)
		}
	}
	if who == -1 {
		return
	}

	w.NumShredded--
	if int16(who) != w.NumShredded {
		w.Shreds[who] = w.Shreds[w.NumShredded]
	}
	w.Shreds[w.NumShredded].Frame = 0
}

// ZeroShreds is numShredded's line in ZeroFlamesAndTheLike (DynamicMaps.c:795), bound as a
// hook on Scene because the counter is on World and DrawLocale's reset head is not.
//
// It clears the counter and not the table, exactly as the C's single assignment does. Every
// slot is fully written by AddAShreddedGlider before it is read, so the stale rects below
// the counter are unreachable.
func (w *World) ZeroShreds() {
	w.NumShredded = 0
}

// RenderShreds is Render.c:557-612. Two arms; see the file comment for what each one is.
//
// The dirty rects are the interesting part, and they are the only place in the renderer
// where the work rect and the back rect are deliberately *different* rects:
//
//	back rect    exactly where the sprite was drawn, so the next frame restores the wall
//	             under it before this function draws again
//	work rect    the same rect extended backwards over where the sprite was *last* frame,
//	             so the screen copy covers the trail it left behind
//
// The extension is four pixels in the fall arm, which is precisely the step, and one pixel
// in the growth arm, which is not -- the growth arm's top edge never moves, so the extra row
// is above the cloud and always blank. Harmless, and left in: it is one row of overdraw a
// frame and removing it would be the kind of tidying that makes a replay diverge for no
// visible gain.
//
// Both rects are registered on the sparkle frame too, even though nothing is drawn. The back
// rect there is a no-op (the previous frame's back rect already erased the last sprite) and
// the work rect is what pushes that erase to the screen.
func (w *World) RenderShreds() {
	if w.NumShredded <= 0 {
		return
	}

	art := w.R.A.Sheet("shred")

	for i := int16(0); i < w.NumShredded; i++ {
		s := &w.Shreds[i]

		switch {
		case s.Frame == 0:
			// Growth. The bottom edge advances one pixel and `high` is the whole
			// height so far, which doubles as the number of rows of sprite to
			// show -- counted up from the *bottom* of shredSrcRect, so the cloud
			// emerges bottom-first.
			s.Bounds.Bottom++
			high := s.Bounds.Bottom - s.Bounds.Top
			if high >= ShredGrowHeight {
				s.Frame = 1
			}

			src := ShredSrc
			src.Top = src.Bottom - high

			dest := render.Offset(s.Bounds, w.R.V.OriginH, w.R.V.OriginV)
			if art != nil {
				w.R.Work.Copy(art, src, dest, render.Masked)
			}
			w.AddRectToBackRects(player.Rect(dest))
			dest.Top--
			w.AddRectToWorkRects(player.Rect(dest))

			// Every frame of the growth, not once: 35 restarts of a shredding
			// noise is what makes the sound continuous. It is also why the shred
			// holds a channel for over a second at priority 903 -- see
			// PlayPrioritySound, where a larger number wins.
			w.PlayPrioritySound(player.ShredSound, player.ShredPriority)

		case s.Frame < ShredLastFrame:
			// Fall. Four pixels a frame, whole sprite, and the counter is
			// incremented *between* moving and drawing -- so the frame that
			// reaches 20 moves the cloud one last time and then sparkles at the
			// position it moved to rather than drawing there.
			s.Bounds.Top += 4
			s.Bounds.Bottom += 4
			dest := render.Offset(s.Bounds, w.R.V.OriginH, w.R.V.OriginV)

			s.Frame++
			if s.Frame < ShredLastFrame {
				if art != nil {
					w.R.Work.Copy(art, ShredSrc, dest, render.Masked)
				}
			} else {
				// AddSparkle centres a 20x19 puff in the 40x35 cloud, so the
				// confetti disappears in a flash at its own middle.
				//
				// The C passes &shreds[i].bounds and AddSparkle offsets its
				// argument *through the pointer*, permanently shifting this
				// entry's Bounds by playOrigin -- and only when a sparkle slot
				// was free. The port passes by value. The difference is
				// unobservable either way: Frame is 20 from here on, so no arm
				// of this function reads Bounds again, and a slot RemoveShreds
				// swaps into a lower index carries Frame 20 with it.
				w.AddSparkle(s.Bounds)
				w.PlayPrioritySound(player.FadeOutSound, player.FadeOutPriority)
			}

			w.AddRectToBackRects(player.Rect(dest))
			dest.Top -= 4
			w.AddRectToWorkRects(player.Rect(dest))
		}
	}
}
