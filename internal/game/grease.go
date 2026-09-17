package game

// Grease.c's simulation half: knock a jar over, animate the tip, grow the slick, and put it
// back when something else erases part of it.
//
// The registration half -- the table itself, AddGrease, ReBackUpGrease and the four-cel
// filmstrip they bake -- is internal/render/grease.go, which also explains why the table
// lives on the render side. This file is the three functions that run during play.
//
// ---------------------------------------------------------------------------
// The only subsystem that rewrites the collision table mid-frame
// ---------------------------------------------------------------------------
//
// HandleGrease is called from RenderFrame, *after* the interaction sweep has finished with
// this frame's hot spots (Render.c:647). On the frame a jar finishes tipping it reaches into
// the table and changes one entry's Action from kRewardIt to kSlideIt, switches it on, and
// replaces its Bounds with a two-pixel-high rect on the floor. Nothing else in the game
// mutates hotSpots[] outside CreateActiveRects.
//
// The consequence is a one-frame delay that is not a bug and must not be tidied away: **a
// slide rect created on frame N is not collidable until N+1.** A glider standing exactly
// where the slick appears is not sliding on the frame it appears. Reordering HandleGrease
// ahead of the sweep would remove the delay and desynchronise every replay.
//
// ---------------------------------------------------------------------------
// Two coordinate systems, three lines apart
// ---------------------------------------------------------------------------
//
// Grease.Dest and Grease.Start are screen coordinates -- the composition handed AddGrease a
// rect that had already been through OffsetRectRoomRelative. Hot-spot Bounds are room-local.
// So the transition subtracts playOrigin to write the bounds and the spreading arm does not,
// because the spreading arm's rect is for painting rather than colliding. Both are in the C
// and the asymmetry is the thing to check first if a slick ever appears in the wrong place.

import (
	"github.com/bwenstar/gliderGo/internal/game/player"
	"github.com/bwenstar/gliderGo/internal/render"
)

// The spill (GliderDefines.h:77, :166). One sound for the jar going over; the slick itself
// is silent as it spreads.
const (
	GreaseSpillSound    int16 = 22
	GreaseSpillPriority int16 = 806
)

// HandleGrease is Grease.c:43-133, called from RenderFrame: advance every jar that is in the
// middle of falling or spreading.
//
// An idle jar and a finished one cost the loop one comparison each, which is why there is no
// separate list of active spills -- and why the early return on an empty table is the whole
// of the optimisation the original bothered with.
func (w *World) HandleGrease() {
	if len(w.R.Grease) == 0 {
		return
	}

	for i := range w.R.Grease {
		g := &w.R.Grease[i]

		switch g.Mode {
		case render.GreaseFalling:
			// Four frames, not three: Frame starts at -1, so the cels drawn are
			// 0, 1, 2 and then 3 on the frame the test fires. The clamp is
			// therefore doing nothing the first time through -- Frame is exactly 3
			// when it runs -- and exists for the case ReBackUpGrease creates, where
			// a jar can be re-entered mid-fall.
			g.Frame++
			if g.Frame >= 3 {
				g.Frame = 3
				g.Mode = render.GreaseSpreading

				// The jar's reward rect becomes the slick's slide rect. Note that
				// the bounds are computed from the *un-stepped* Dest.Bottom: the
				// two-pixel walk at the bottom of this arm happens afterwards, so
				// the slick sits on the floor line the last cel was drawn at.
				if hot := w.greaseHot(g); hot != nil {
					hot.Action = SlideIt
					hot.IsOn = true

					var src render.Rect
					if g.IsRight {
						src = render.SetRect(0, -2, 2, 0)
					} else {
						src = render.SetRect(-2, -2, 0, 0)
					}
					// Screen -> room-local, because Bounds is room-local and
					// Start and Dest are not. See the file comment.
					src = render.Offset(src, -w.R.V.OriginH, -w.R.V.OriginV)
					src = render.Offset(src, g.Start, g.Dest.Bottom)
					hot.Bounds = src
				}
			}

			// The cel comes out of the saved map, which already holds the wall
			// behind the jar with the jar composited on top -- so this is srcCopy
			// into both maps and not a masked blit. Writing the back map is what
			// makes the tip permanent: the jar does not stand back up when the
			// glider flies away.
			cel := render.Offset(render.SetRect(0, 0, 32, 27), 0, g.Frame*27)
			if strip := w.greaseStrip(g); strip != nil {
				w.R.Work.Copy(strip, cel, g.Dest, render.SrcCopy)
				w.R.Back.Copy(strip, cel, g.Dest, render.SrcCopy)
			}
			w.AddRectToWorkRects(player.Rect(g.Dest))

			if g.IsRight {
				g.Dest = render.Offset(g.Dest, 2, 0)
			} else {
				g.Dest = render.Offset(g.Dest, -2, 0)
			}

		case render.GreaseSpreading:
			// Two pixels of slick a frame, painted black into both maps and added
			// to the collision rect in the same breath. The paint rect is screen
			// coordinates and the widening is room-local, three lines apart.
			var src render.Rect
			if g.IsRight {
				src = render.Offset(render.SetRect(0, -2, 2, 0), g.Start, g.Dest.Bottom)
				g.Start += 2
				if hot := w.greaseHot(g); hot != nil {
					hot.Bounds.Right += 2
				}
			} else {
				src = render.Offset(render.SetRect(-2, -2, 0, 0), g.Start, g.Dest.Bottom)
				g.Start -= 2
				if hot := w.greaseHot(g); hot != nil {
					hot.Bounds.Left -= 2
				}
			}

			// **Both destinations are named at the call site.** This is the
			// correctly-converted half of docs/IMPROVEMENTS.md 2.34 -- the model
			// HandleOutlet's PaintRect should have followed and did not -- so the
			// AddRectToWorkRects below is telling the truth about where the pixels
			// went. Back first, then work, as the C has it.
			w.R.Back.Fill(src, render.Black8)
			w.R.Work.Fill(src, render.Black8)
			w.AddRectToWorkRects(player.Rect(src))

			// Start has already been stepped, so the test is against the position
			// the *next* frame would paint from. A slick therefore stops one step
			// short of Stop rather than one past it.
			if g.IsRight {
				if g.Start >= g.Stop {
					g.Mode = render.GreaseSpiltIdle
				}
			} else {
				if g.Start <= g.Stop {
					g.Mode = render.GreaseSpiltIdle
				}
			}
		}
	}
}

// SpillGrease is Grease.c:257-265: a jar has been knocked over.
//
// It flags and nothing more -- HandleGrease does the work on the following frames -- and the
// idle test is what makes it idempotent: a jar already falling, spreading or spilt is left
// alone, so a glider bouncing on a spent jar does not restart the animation or re-play the
// sound. That test is the only guard, and it is the reason HandleRewards' grease arm needs no
// StillOver latch of its own.
//
// **Both arguments can be -1.** FireTrigger's remote half passes a dynamic index its own
// branch condition has just proved is -1, which in the original is a read of grease[-1]
// (triggers.go documents the line). The bounds check below is that read refused. `hotNum`
// is not checked here because it is only stored; the read of it is guarded in greaseHot.
func (w *World) SpillGrease(dynaNum, hotNum int16) {
	if dynaNum < 0 || int(dynaNum) >= len(w.R.Grease) {
		return
	}
	g := &w.R.Grease[dynaNum]
	if g.Mode == render.GreaseIdle {
		g.Mode = render.GreaseFalling
		g.HotNum = hotNum
		w.PlayPrioritySound(GreaseSpillSound, GreaseSpillPriority)
	}
}

// RedrawAllGrease is Grease.c:270-301: repaint the black lines of every spilt slick in the
// central room.
//
// Called by the ten reward arms that restore a saved map, and for a reason that is not in the
// name: RestoreFromSavedMap has just written a rectangle of *original* background over
// whatever was there, and if a slick crossed that rectangle the erase has just cut a hole in
// it. This paints the lot back. It has to run after the restore, never before.
//
// Three tests decide what is repainted, and the middle one is the interesting one:
//
//	where == thisRoomNumber   the table holds jars from all nine local rooms and only
//	                          the central room's are on screen to repair
//	height == 2               the hot spot is still the two-pixel slide rect. A jar that
//	                          has not tipped yet still owns its 27-pixel reward rect, so
//	                          this is a second, independent way of asking "is there a
//	                          slick?" -- and it is the test that would catch a jar whose
//	                          hot spot was reused by a room rebuild
//	mode != kGreaseIdle       an untouched jar has no slick
//
// The C reads hotSpots[hotNum] *before* testing any of them, which is safe there because
// hotSpots is allocated at its cap and an unwritten hotNum of 0 indexes garbage that is
// immediately discarded. Here the table is a slice whose length grows during composition, so
// the read is guarded and the order of the tests is otherwise the C's. Nothing in the three
// has a side effect, so the reordering is not observable.
func (w *World) RedrawAllGrease() {
	if len(w.R.Grease) == 0 {
		return
	}

	for i := range w.R.Grease {
		g := &w.R.Grease[i]

		hot := w.greaseHot(g)
		if hot == nil {
			continue
		}
		src := hot.Bounds
		if g.Where == w.R.RoomNumber && src.Bottom-src.Top == 2 && g.Mode != render.GreaseIdle {
			src = render.Offset(src, w.R.V.OriginH, w.R.V.OriginV)
			w.R.Back.Fill(src, render.Black8)
			w.R.Work.Fill(src, render.Black8)
			w.AddRectToWorkRects(player.Rect(src))
		}
	}
}

// greaseHot resolves a jar's HotNum against the hot-spot table.
//
// The port's guard for the C's unguarded index, and the reason it is needed rather than
// merely tidy: HotNum is 0 on a jar nothing has touched, and it can name a rect from a
// *previous* locale on one the table has outlived, because DrawLocale clears the grease table
// while AddGrease never rewrites HotNum. In the C both cases land inside a fixed-size array
// and the value is discarded by the mode test; here the second would be a panic.
//
// Returns a pointer because two of the three callers write through it.
func (w *World) greaseHot(g *render.Grease) *HotObject {
	if g.HotNum < 0 || int(g.HotNum) >= len(w.R.Hot) {
		return nil
	}
	return &w.R.Hot[g.HotNum]
}

// greaseStrip resolves a jar's MapNum to the four-cel filmstrip AddGrease baked.
//
// AddGrease refuses to register a jar whose slot it could not claim, so a live entry always
// has a valid MapNum and this never returns nil in a correctly composed room. It is guarded
// anyway because the render tests compose Scenes with no art loaded, where SavedMap.Map is
// nil and the alternative is a nil dereference in the middle of a frame.
func (w *World) greaseStrip(g *render.Grease) *render.Surface {
	if g.MapNum < 0 || int(g.MapNum) >= len(w.R.SavedMaps) {
		return nil
	}
	return w.R.SavedMaps[g.MapNum].Map
}
