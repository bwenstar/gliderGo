package player

// NumGliderSrcRects is kNumGliderSrcRects (GliderDefines.h:558).
const NumGliderSrcRects = 31

// GliderAtlasHigh is the total height of the glider sprite sheet, which the rect
// table below exactly fills: 21 normal frames of 20, 8 burning frames of 26, and
// two sliding frames of 20.
const GliderAtlasHigh = 668

// GliderSrc is gliderSrc[], built the way StructuresInit.c:191-205 builds it.
//
// The atlas is one column 48 px wide. It is not a uniform grid: the burning frames
// are 26 px tall rather than 20, so the offsets stop being a simple multiple after
// index 20 and the table has three separate runs. Computing an index as
// `i * GliderHigh` would silently read the wrong row for any burning frame.
//
//	 0..20   48x20 at y = 20*i          normal, tipped, dissolve, about-face
//	21..28   48x26 at y = 420 + 26*(i-21)   burning, right then left
//	   29    48x20 at y = 628           sliding on grease, facing right
//	   30    48x20 at y = 648           sliding on grease, facing left
var GliderSrc = func() [NumGliderSrcRects]Rect {
	var a [NumGliderSrcRects]Rect
	for i := int16(0); i <= 20; i++ {
		top := GliderHigh * i
		a[i] = Rect{Top: top, Left: 0, Bottom: top + GliderHigh, Right: GliderWide}
	}
	for i := int16(21); i <= 28; i++ {
		top := 420 + GliderBurningHigh*(i-21)
		a[i] = Rect{Top: top, Left: 0, Bottom: top + GliderBurningHigh, Right: GliderWide}
	}
	a[29] = Rect{Top: 628, Left: 0, Bottom: 628 + GliderHigh, Right: GliderWide}
	a[30] = Rect{Top: 648, Left: 0, Bottom: 648 + GliderHigh, Right: GliderWide}
	return a
}()

// setSprite assigns both Src and Mask from the atlas, which is what all ~40 of the
// original's sprite assignments do -- each is a pair of identical statements, one
// for src and one for mask. They stay two fields because CopyMask takes the source
// and mask rects independently and several handlers then trim one edge of each.
func (g *Glider) setSprite(i int16) {
	g.Src = GliderSrc[i]
	g.Mask = GliderSrc[i]
}

// setFadeSprite is the dissolve lookup: a fade frame picks a stage out of
// FadeInSequence, and facing left shifts the whole run by LeftFadeOffset. Used by
// the fade in, the fade out, the transporter in and the transporter out, which is
// why all four look identical on screen.
func (g *Glider) setFadeSprite(frame int16) {
	i := FadeInSequence[frame]
	if g.Facing == FaceLeft {
		i += LeftFadeOffset
	}
	g.setSprite(i)
}

// setFoilSprite is the other dissolve, used only while gaining or losing foil. It
// walks *down* from FoilFadeTop instead of indexing a table, so it runs through
// stages 10, 9, 8, 7 rather than the fade's 4, 5, 6, 7 -- the foil change dissolves
// in the opposite direction to a respawn.
func (g *Glider) setFoilSprite(frame int16) {
	i := FoilFadeTop - frame
	if g.Facing == FaceLeft {
		i += LeftFadeOffset
	}
	g.setSprite(i)
}

// setNormalSprite picks the level or banked sprite for the current facing. This is
// the pair FlagGliderNormal uses; MoveGliderNormal has its own copy that also
// handles sliding.
func (g *Glider) setNormalSprite() {
	if g.Facing == FaceLeft {
		g.setSprite(SpriteLeft)
	} else {
		g.setSprite(SpriteRight)
	}
}
