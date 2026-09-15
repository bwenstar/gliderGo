package game

// RestoreFromSavedMap (DynamicMaps.c:131-163): make an object disappear.
//
// The composition stashed a copy of the background under every prize before drawing the
// prize on top of it (internal/render/locale.go's backUpToSavedMap). This is the other
// half: paint that copy back, into *both* maps, and the prize is gone -- not hidden, gone,
// because the write to the back map means the erase survives every future frame's
// back->work restore. It is how the game makes a permanent change to a room without
// recomposing it.
//
// **The order of the two blits does not matter but the fact of both does.** Writing only
// the work map leaves the prize in the background, so it reappears the moment the glider
// flies away and the dirty rect is restored. Writing only the back map leaves this frame
// showing the prize. The C writes back first, and so does this.
//
// It lives in its own file rather than in sparkles.go because it is not an effect: it is
// the consumer of the saved-map economy that DrawARoomsObjects produces, and the only one.
// Its two callers are HandleRewards -- a prize collected by touching it -- and
// HandleSwitches, for a prize collected remotely by a switch wired to it.

import (
	"glidergo/internal/game/player"
	"glidergo/internal/render"
)

// RestoreFromSavedMap puts the background back over object `who` of room `where`.
//
// **The first matching slot wins.** A cuckoo clock and a star each own two slots with the
// same (where, who) -- one for the object, claimed by the composition, and one for the
// animation strip, claimed by AddPendulum or AddStar afterwards -- and the second one's
// Dest is a strip-sized rect at the *corner of the screen*. The `break` is therefore
// load-bearing: without it, collecting a star would paint six frames of star art over the
// top-left of the play area. See render.SavedMap.Dest.
//
// doSparkle is false for a prize the glider touched and true for one a switch removed. The
// difference is that a switch's prize vanishes somewhere the player may not be looking, so
// it gets a puff of light and a sound to say what happened; a prize the player flew into
// already has a flying score numeral of its own.
func (w *World) RestoreFromSavedMap(where, who int16, doSparkle bool) {
	for i := range w.R.SavedMaps {
		sm := &w.R.SavedMaps[i]
		if sm.Where != where || sm.Who != who || sm.Map == nil {
			continue
		}

		// mapRect is the patch's own bounds -- the C's ZeroRectCorner of dest, which is
		// the same thing, since BackUpToSavedMap sized the patch from this exact rect.
		mapRect := sm.Map.Bounds()
		w.R.Back.Copy(sm.Map, mapRect, sm.Dest, render.SrcCopy)
		w.R.Work.Copy(sm.Map, mapRect, sm.Dest, render.SrcCopy)

		// Only a work rect, and no back rect. The back list is "restore this much of
		// the work map from the background next frame", and the background *is* what
		// was just written -- so a back rect here would be redundant, not wrong. The
		// work rect is what gets the erase on screen.
		w.AddRectToWorkRects(player.Rect(sm.Dest))

		if doSparkle {
			// Un-offset, because AddSparkle offsets. See sparkles.go's header on the
			// two coordinate conventions -- this is the one place in the port where a
			// screen rect is converted back to room-local purely to be converted
			// forward again, and it is the C's line for line.
			w.AddSparkle(render.Offset(sm.Dest, -w.R.V.OriginH, -w.R.V.OriginV))
			w.PlayPrioritySound(player.FadeOutSound, player.FadeOutPriority)
		}
		break
	}
}
