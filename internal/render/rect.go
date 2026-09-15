package render

import "glidergo/internal/house"

// Rect arithmetic, transcribed from GliderPRO/Sources/RectUtils.c.
//
// These are int16 by choice, not by inertia. The original's rects are shorts and
// the draw code offsets them by playOriginH/V and kVertLocalOffset before
// clipping, so intermediate values legitimately go negative and legitimately
// exceed the surface -- and the wrap behaviour at those extremes is part of what
// the port has to match. Widening to int would quietly diverge.
//
// The Q- prefixed originals ("quick") skip Rect normalisation; so do these.

// Rect is house.Rect, re-exported so the draw code reads like the original
// without every file importing the house package for its geometry.
type Rect = house.Rect

// Pt is a QuickDraw Point: vertical first, as on disk.
type Pt = house.Point

// SetRect is QSetRect (RectUtils.c:322): left/top/right/bottom order, which is
// *not* the order the Rect struct stores them in.
func SetRect(left, top, right, bottom int16) Rect {
	return Rect{Top: top, Left: left, Bottom: bottom, Right: right}
}

// Offset is QOffsetRect (RectUtils.c:307).
func Offset(r Rect, h, v int16) Rect {
	r.Left += h
	r.Right += h
	r.Top += v
	r.Bottom += v
	return r
}

// Inset is the Toolbox InsetRect: positive shrinks, negative grows. QuickDraw
// does not clamp an over-inset rect to empty, and neither does this.
func Inset(r Rect, h, v int16) Rect {
	r.Left += h
	r.Right -= h
	r.Top += v
	r.Bottom -= v
	return r
}

// ZeroCorner is ZeroRectCorner (RectUtils.c:57): slide the rect so its top-left
// is the origin, preserving width and height. Used everywhere a rect that
// describes a position on screen has to be reused as a source rect in a
// same-sized offscreen map.
func ZeroCorner(r Rect) Rect {
	r.Right -= r.Left
	r.Bottom -= r.Top
	r.Left = 0
	r.Top = 0
	return r
}

// HalfWide and HalfTall are HalfRectWide/HalfRectTall (RectUtils.c:79,87). C
// division truncates toward zero and so does Go's, which matters because these
// feed centring arithmetic on odd widths.
func HalfWide(r Rect) int16 { return (r.Right - r.Left) / 2 }
func HalfTall(r Rect) int16 { return (r.Bottom - r.Top) / 2 }

// Sect is the Toolbox SectRect. It reports whether the two rects overlap and
// returns the intersection; on no overlap the returned rect is the empty rect,
// as QuickDraw's is.
//
// The draw code uses this as a clip test far more often than for its result:
// "does this object touch the visible house rect at all", where a false answer
// suppresses the object entirely rather than clipping it.
func Sect(a, b Rect) (Rect, bool) {
	s := Rect{
		Top:    max16(a.Top, b.Top),
		Left:   max16(a.Left, b.Left),
		Bottom: min16(a.Bottom, b.Bottom),
		Right:  min16(a.Right, b.Right),
	}
	if s.Left >= s.Right || s.Top >= s.Bottom {
		return Rect{}, false
	}
	return s, true
}

// UnionSimilar is QUnionSimilarRect (RectUtils.c:272-298): the smallest rect
// containing both of two rects.
//
// The C's comment says the two are "assumed to have the same width and height",
// and its one caller obeys that -- a fluttering game-over page and where that page
// was last frame, both 32x32. The assumption buys nothing: the four comparisons are
// a plain union and are correct for any pair, which is why this is not named
// UnionSameSizeRect.
//
// It is the union QuickDraw's own UnionRect would give, minus the empty-rect
// special case: UnionRect treats an empty rect as contributing nothing, and this
// does not. No caller passes one.
func UnionSimilar(a, b Rect) Rect {
	return Rect{
		Top:    min16(a.Top, b.Top),
		Left:   min16(a.Left, b.Left),
		Bottom: max16(a.Bottom, b.Bottom),
		Right:  max16(a.Right, b.Right),
	}
}

// CenterIn is CenterRectInRect (RectUtils.c:142): centre the first rect inside
// the second, keeping its size. Note that it is (Bwide-Awide)/2 and not
// Bwide/2-Awide/2 -- the two disagree by a pixel whenever the difference is odd,
// and the original's form is the one the art was drawn against.
func CenterIn(r, in Rect) Rect {
	w, h := r.Wide(), r.Tall()
	r.Left = in.Left + (in.Wide()-w)/2
	r.Right = r.Left + w
	r.Top = in.Top + (in.Tall()-h)/2
	r.Bottom = r.Top + h
	return r
}

// IsRectLeftOfRect is RectUtils.c:181-190, and it does not do what its name says.
//
//	offset = (rect1->right - rect1->left) - (rect2->right - rect2->left) / 2;
//	if ((rect1->left) < (rect2->left + offset))
//
// **The `/ 2` binds to the second width alone.** The author plainly meant
// `(width1 - width2) / 2`, which is the offset that would put the comparison at the
// two rects' centres; what is written is `width1 - width2/2`. With a 24-wide dart
// and a 48-wide glider the intended offset is -12 and the actual one is 0, so the
// test degenerates to "is the dart's left edge left of the glider's left edge".
//
// It has exactly one caller -- CheckDynamicCollision (Dynamics.c:53), where it picks
// which way a foil-clad glider is shoved -- so this is the only place the difference
// can be seen, and what it produces is a dart overlapping the glider's left half
// shoving the player *left*, into the dart. Transcribed rather than fixed: the shove
// direction is exactly the sort of thing 1.8's fidelity replays pin, and "correcting"
// it would change how a dart hit feels. See docs/IMPROVEMENTS.md.
//
// It lives here, next to CenterIn, because it is centring arithmetic gone wrong and
// because RectUtils.c is where the original keeps it.
func IsRectLeftOfRect(rect1, rect2 Rect) bool {
	offset := (rect1.Right - rect1.Left) - (rect2.Right-rect2.Left)/2
	return rect1.Left < rect2.Left+offset
}

// Empty reports the QuickDraw notion of an empty rect.
func Empty(r Rect) bool { return r.Left >= r.Right || r.Top >= r.Bottom }

func max16(a, b int16) int16 {
	if a > b {
		return a
	}
	return b
}

func min16(a, b int16) int16 {
	if a < b {
		return a
	}
	return b
}
