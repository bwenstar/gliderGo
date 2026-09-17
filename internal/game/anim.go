package game

// Render.c's three background-animation passes: candle flames, tiki torches and barbecue
// coals (RenderFlames), the cuckoo clock's pendulum (RenderPendulums) and the spinning
// star (RenderStars).
//
// The registration half -- the five tables, the saved-map slot each entry owns and the
// filmstrip baked into it -- is internal/render/anim.go, which is also where the strip
// layout is explained. This file is what runs during play.
//
// ---------------------------------------------------------------------------
// One blit a frame, and no back rect
// ---------------------------------------------------------------------------
//
// Every other moving thing in the game costs two blits: draw it into the work map, and
// register a back rect so the next frame restores the wall underneath before drawing it
// again. These five cost one. A cel is *already* the wall with the flame composited on it,
// so an opaque blit of this frame's cel erases the last frame's — the strip is doing the
// work AddRectToBackRects would otherwise do.
//
// That is why none of the three functions below calls AddRectToBackRects, and it is not an
// omission to be tidied. Adding one would restore bare wall under the flame on the frame
// after every draw, which at 30fps is a flame that flickers to nothing every other frame.
//
// ---------------------------------------------------------------------------
// Two clocks, neither of them the frame counter
// ---------------------------------------------------------------------------
//
// RenderFlames and RenderStars run on **alternate frames** -- RenderFrame calls one or the
// other on EvenFrame -- so both families animate at 15fps while the gliders run at 30. The
// two never run on the same frame, which is worth knowing before reading a replay: a room
// with a candle and a star in it steps exactly one of them per frame.
//
// RenderPendulums runs every frame but *acts* on very few of them, gated on ClockFrame.
// See its own comment for the uneven tick-tock that produces.

import (
	"github.com/bwenstar/gliderGo/internal/game/player"
	"github.com/bwenstar/gliderGo/internal/render"
)

// The clock (GliderDefines.h:67-68, :124-125). Two sounds for the two ends of a swing, at
// adjacent priorities, and low ones: a tik loses to almost anything else in the room.
const (
	TikSound    int16 = 12
	TokSound    int16 = 13
	TikPriority int16 = 200
	TokPriority int16 = 201
)

// ---------------------------------------------------------------------------
// The four wrapping families
// ---------------------------------------------------------------------------

// stepStrip is the body of all four of the C's wrapping animation loops: RenderFlames'
// three (Render.c:200-215, :218-233, :238-255) and RenderStars' one (:429-447).
//
// The four differ only in the table, the cel count and the cel height, and the C's own
// copies have already drifted -- RenderStars tests `mode >= 6` where the other three use a
// named constant. One function, four call sites.
//
// Read the order carefully, because it is the opposite of what "animate then draw" suggests
// and it is what makes the seeded cel index work: **the step happens before the blit**. So
// an entry seeded at cel 3 of 5 shows cel 4 on its first frame, never cel 3.
//
// The wrap is where the seeding-one-past-the-end note in render.Anim.Mode is discharged.
// RandomInt(n) can return n, so Mode can arrive here equal to `frames` with a Src rect one
// cel below the bottom of the strip. The increment makes it frames+1, which fails the same
// `>= frames` test that frames-1 would have failed, and the wrap resets Src **absolutely**
// -- to (0, celH) rather than by subtraction -- so the out-of-range rect is gone before the
// blit reads it. A seed of `frames` and a seed of `frames-1` therefore draw the same first
// cel, which is why the original gets away with it.
//
// The Stopped gate is the star's (`theStars[i].mode != -1`) and is shared with the three
// flame families, which never set it: StopStar and StopPendulum are the only writers and
// they search the star and pendulum tables. So the gate is dead code for candles, torches
// and coals, and folding it in is what lets one function serve all four.
func (w *World) stepStrip(table []render.Anim, frames, celH int16) {
	for i := range table {
		a := &table[i]
		if a.Stopped {
			continue
		}

		a.Mode++
		a.Src = render.Offset(a.Src, 0, celH)
		if a.Mode >= frames {
			a.Mode = 0
			a.Src.Top = 0
			a.Src.Bottom = celH
		}

		if strip := w.animStrip(a.SavedMap); strip != nil {
			w.R.Work.Copy(strip, a.Src, a.Dest, render.SrcCopy)
		}
		w.AddRectToWorkRects(player.Rect(a.Dest))
	}
}

// RenderFlames is Render.c:193-255: three tables, three cel sizes, one pass.
//
// The early return is the original's and is a triple test rather than three separate ones,
// which matters only in that a room with coals but no candles still enters the function.
func (w *World) RenderFlames() {
	if len(w.R.Flames) == 0 && len(w.R.TikiFlames) == 0 && len(w.R.Coals) == 0 {
		return
	}

	w.stepStrip(w.R.Flames, render.NumCandleFrames, 15)
	w.stepStrip(w.R.TikiFlames, render.NumTikiFrames, 10)
	w.stepStrip(w.R.Coals, render.NumCoalFrames, 9)
}

// RenderStars is Render.c:420-448, the other half of the parity alternation.
//
// A collected star is gated out by Stopped rather than deleted, so it keeps its slot and
// its cel index for the rest of the locale; see render.Anim.Stopped for the one way that
// differs from the C, and switches.go for the one path that collects a star *without*
// stopping it (docs/IMPROVEMENTS.md 2.38).
func (w *World) RenderStars() {
	if len(w.R.Stars) == 0 {
		return
	}

	w.stepStrip(w.R.Stars, render.NumStarFrames, 31)
}

// ---------------------------------------------------------------------------
// The pendulum
// ---------------------------------------------------------------------------

// RenderPendulums is Render.c:260-322, and it is the odd one of the four in every respect:
// it runs every frame instead of every other one, it swings between two ends instead of
// wrapping, it plays sounds, and it acts on one frame in five or ten rather than all of
// them.
//
// **ClockFrame gives an uneven tick-tock, and that is the original's sound.** The counter
// increments every frame, and the body runs only when it reads exactly 10 or exactly 15 --
// resetting to 0 on the 15. addPendulum seeds it at 10. So the gaps between swings
// alternate five frames and ten:
//
//	seed 10 -> 11 12 13 14 15  swing, reset       5 frames
//	         0 1 .. 9      10  swing, no reset   10 frames
//	          11 .. 14     15  swing, reset       5 frames
//
// A clock that ticked evenly would be a different sound and a different animation, so the
// two magic numbers are transcribed rather than rationalised. Note also that the counter
// only advances when the room *has* a pendulum: the early return is above the increment, so
// a locale with no clock in it leaves ClockFrame frozen wherever the last one left it, and
// the next clock reseeds it anyway.
//
// The swing is four steps, not three, because the flip happens after the step and the ends
// are visited once each:
//
//	mode 1 -> 2  flip, kTikSound
//	     2 -> 1
//	     1 -> 0  flip, kTokSound
//	     0 -> 1
//
// The cels *drawn* are therefore 2, 1, 0, 1 -- the centre twice per cycle and each end once.
// That is the opposite of a real pendulum, which lingers at the extremes, and it is what
// ping-ponging across three cels gets you. Unlike stepStrip's wrap, Src moves only by offset
// here and is never reset absolutely; it does not need to be, because Mode never leaves 0..2.
//
// playedTikTok throttles to **one sound per frame across every pendulum in the locale**, not
// one per pendulum. A room with three grandfather clocks in it therefore ticks once, and
// which of the three wins is table order -- so two clocks swinging opposite ways make a room
// that tiks and never toks. Faithful, and audible in the original.
func (w *World) RenderPendulums() {
	if len(w.R.Pendulums) == 0 {
		return
	}

	w.R.ClockFrame++
	if w.R.ClockFrame != 10 && w.R.ClockFrame != 15 {
		return
	}
	if w.R.ClockFrame >= 15 {
		w.R.ClockFrame = 0
	}

	playedTikTok := false
	for i := range w.R.Pendulums {
		p := &w.R.Pendulums[i]
		if p.Stopped {
			continue
		}

		if p.ToOrFro {
			p.Mode++
			p.Src = render.Offset(p.Src, 0, 28)
			if p.Mode >= 2 {
				p.ToOrFro = !p.ToOrFro
				if !playedTikTok {
					w.PlayPrioritySound(TikSound, TikPriority)
					playedTikTok = true
				}
			}
		} else {
			p.Mode--
			p.Src = render.Offset(p.Src, 0, -28)
			if p.Mode <= 0 {
				p.ToOrFro = !p.ToOrFro
				if !playedTikTok {
					w.PlayPrioritySound(TokSound, TokPriority)
					playedTikTok = true
				}
			}
		}

		if strip := w.animStrip(p.SavedMap); strip != nil {
			w.R.Work.Copy(strip, p.Src, p.Dest, render.SrcCopy)
		}
		w.AddRectToWorkRects(player.Rect(p.Dest))
	}
}

// animStrip resolves an entry's saved-map slot to the filmstrip its registration baked.
//
// The five add* functions refuse to append an entry whose slot they could not claim, so a
// live entry always has a valid index and this never returns nil in a correctly composed
// room. It is guarded for the same reason greaseStrip is: the render tests compose Scenes
// with no art, where SavedMap.Map is nil, and the alternative is a nil dereference in the
// middle of a frame rather than a missing flame.
func (w *World) animStrip(index int) *render.Surface {
	if index < 0 || index >= len(w.R.SavedMaps) {
		return nil
	}
	return w.R.SavedMaps[index].Map
}
