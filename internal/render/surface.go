package render

import "image"

// Surface is an 8-bit indexed offscreen GWorld.
//
// Indexed, not RGBA, and that is the whole point. Glider PRO draws its furniture
// shadows with PenPat(gray) + PenMode(patOr), which on an indexed pixmap ORs the
// *palette index* into the destination: k8DkstGrayColor (254) over an arbitrary
// background gives whatever index 254|bg happens to name. Converting to RGB
// first and compositing there produces plausible-looking colours that are not
// the game's colours. Everything here therefore stays in index space until the
// final ToRGBA/ToPNG.
//
// Mask mirrors Pix and models the separate 1-bit mask GWorld that CopyMask takes
// as its second argument: 0xFF where the source is opaque, 0x00 where it is
// transparent. A nil Mask means fully opaque, which is the common case (the
// backgrounds, the floor support, every destination surface).
//
// Surfaces are always bounds-origin (0,0): so were the original's offscreens, so
// port-local coordinates and pixel coordinates coincide, which matters for the
// gray pattern's phase (see FillOvalPatOrGray).
type Surface struct {
	W, H int
	Pix  []uint8
	Mask []uint8
}

// NewSurface allocates a surface filled with index 0.
//
// Index 0 is white, and white is what the original's freshly created GWorlds
// held: NewGWorld does not promise an initialised pixel image, but the shipped
// art depends on it in one place -- the furniture sheet's PICT is 64x221 inside a
// 64x278 GWorld, and the extractor measured the unwritten rows as white. No
// named sub-rect reads there, so this only has to be *a* defined value; white is
// the one the original had.
func NewSurface(w, h int) *Surface {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return &Surface{W: w, H: h, Pix: make([]uint8, w*h)}
}

// Bounds is the surface's rect, origin at (0,0).
func (s *Surface) Bounds() Rect {
	return Rect{Top: 0, Left: 0, Bottom: int16(s.H), Right: int16(s.W)}
}

// opaque reports whether the source pixel at (x,y) should be drawn. A nil Mask
// is fully opaque.
func (s *Surface) opaque(x, y int) bool {
	return s.Mask == nil || s.Mask[y*s.W+x] != 0
}

// clip intersects a rect with the surface and returns integer pixel bounds.
func (s *Surface) clip(r Rect) (x0, y0, x1, y1 int, ok bool) {
	x0, y0 = int(r.Left), int(r.Top)
	x1, y1 = int(r.Right), int(r.Bottom)
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > s.W {
		x1 = s.W
	}
	if y1 > s.H {
		y1 = s.H
	}
	return x0, y0, x1, y1, x0 < x1 && y0 < y1
}

// setMaskOpaque marks a written pixel opaque, for surfaces that carry a mask.
func (s *Surface) setMaskOpaque(i int) {
	if s.Mask != nil {
		s.Mask[i] = 0xFF
	}
}

// Fill is PaintRect under the default patCopy pen: every pixel in the rect
// becomes the given index. ColorRect (ColorUtils.c:36) is Fill with the fore
// colour temporarily set, and PaintRect with no ColorRect wrapper around it fills
// black, because black is the default foreground -- which is how DrawLocale
// clears the whole house rect and how an unlit room is drawn.
func (s *Surface) Fill(r Rect, idx uint8) {
	x0, y0, x1, y1, ok := s.clip(r)
	if !ok {
		return
	}
	// The first row is written a byte at a time -- which the compiler turns into
	// a memset -- and the rest are copied from it. DrawLocale clears 294,400
	// pixels this way before every composition, and every unlit room clears
	// another 164,864.
	first := s.Pix[y0*s.W+x0 : y0*s.W+x1]
	for i := range first {
		first[i] = idx
	}
	for y := y0 + 1; y < y1; y++ {
		copy(s.Pix[y*s.W+x0:y*s.W+x1], first)
	}
	if s.Mask != nil {
		for y := y0; y < y1; y++ {
			m := s.Mask[y*s.W+x0 : y*s.W+x1]
			for i := range m {
				m[i] = 0xFF
			}
		}
	}
}

// Line is ColorLine (ColorUtils.c:84): MoveTo + LineTo with a 1x1 pen.
//
// Both endpoints are drawn. This is the single most consequential detail in the
// whole file, and it is settled three independent ways by the original's own
// code: DrawTable draws single pixels as ColorLine(x,y,x,y); DrawTrackLight
// spans left..right-1 for a full-width band; and DrawFlourescent's middle band
// runs left+16..right-17, which exactly fills the gap between two 16-pixel end
// sprites only if both ends are inclusive.
//
// Horizontal and vertical lines are exact. Diagonals (only the clock hands and
// the region outlines use them) are Bresenham, which is the shape of QuickDraw's
// own line walker but not provably the same tie-breaking on 45-degree steps.
func (s *Surface) Line(h0, v0, h1, v1 int16, idx uint8) {
	x0, y0, x1, y1 := int(h0), int(v0), int(h1), int(v1)
	dx, dy := x1-x0, y1-y0
	sx, sy := 1, 1
	if dx < 0 {
		dx, sx = -dx, -1
	}
	if dy < 0 {
		dy, sy = -dy, -1
	}
	err := dx - dy
	for {
		s.plot(x0, y0, idx)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func (s *Surface) plot(x, y int, idx uint8) {
	if x < 0 || y < 0 || x >= s.W || y >= s.H {
		return
	}
	s.Pix[y*s.W+x] = idx
	s.setMaskOpaque(y*s.W + x)
}

// FrameRect is the Toolbox FrameRect with a 1x1 pen: the border lies inside the
// rect, so the bottom and right edges are at bottom-1 and right-1.
func (s *Surface) FrameRect(r Rect, idx uint8) {
	if Empty(r) {
		return
	}
	s.Line(r.Left, r.Top, r.Right-1, r.Top, idx)
	s.Line(r.Left, r.Bottom-1, r.Right-1, r.Bottom-1, idx)
	s.Line(r.Left, r.Top, r.Left, r.Bottom-1, idx)
	s.Line(r.Right-1, r.Top, r.Right-1, r.Bottom-1, idx)
}

// HiliteRect is ColorUtils.c:103 verbatim -- four lines, colour1 on the top and
// left, colour2 on the bottom and right, with the deliberately asymmetric
// right-2/bottom-2/right-1/bottom-1 endpoints that give the 3-D bevel its
// characteristic notched corners. It is not a FrameRect and not an xor.
func (s *Surface) HiliteRect(r Rect, c1, c2 uint8) {
	s.Line(r.Left, r.Top, r.Right-2, r.Top, c1)
	s.Line(r.Left, r.Top, r.Left, r.Bottom-2, c1)
	s.Line(r.Right-1, r.Top, r.Right-1, r.Bottom-2, c2)
	s.Line(r.Left+1, r.Bottom-1, r.Right-1, r.Bottom-1, c2)
}

// grayPatternSet reports whether the QuickDraw `gray` pattern has a bit set at a
// port-local pixel. The pattern is eight rows alternating 0xAA and 0x55, so the
// 1-bits are exactly the pixels where x+y is even, phased to the port origin --
// which for every surface here is (0,0).
func grayPatternSet(x, y int) bool { return (x+y)&1 == 0 }

// FillOvalPatOrGray is ColorOval under PenPat(gray) + PenMode(patOr), which is
// how every piece of procedural furniture casts its shadow.
//
// patOr on an indexed pixmap ORs the pattern's colours in: a 1-bit takes the
// foreground index, a 0-bit takes the background index, and the background is
// white -- index 0 -- so the 0-bits change nothing. With k8DkstGrayColor (254)
// as the foreground the result is a 50% checkerboard of "index |= 254", i.e. a
// near-black dither over whatever the room already had there.
//
// The oval rasterisation is this port's one acknowledged deviation from
// QuickDraw: it is a per-pixel ellipse test at pixel centres rather than
// QuickDraw's integer region builder. It agrees on axis extents and differs, if
// at all, by single pixels on the diagonals of these small shadows.
func (s *Surface) FillOvalPatOrGray(r Rect, idx uint8) {
	x0, y0, x1, y1, ok := s.clip(r)
	if !ok {
		return
	}
	w, h := int(r.Wide()), int(r.Tall())
	if w <= 0 || h <= 0 {
		return
	}
	for y := y0; y < y1; y++ {
		ty := 2*(y-int(r.Top)) + 1 - h
		row := y * s.W
		for x := x0; x < x1; x++ {
			tx := 2*(x-int(r.Left)) + 1 - w
			if tx*tx*h*h+ty*ty*w*w > w*w*h*h {
				continue
			}
			if !grayPatternSet(x, y) {
				continue
			}
			s.Pix[row+x] |= idx
			s.setMaskOpaque(row + x)
		}
	}
}

// FillPolyPatOrGray is ColorRegion under PenPat(gray) + PenMode(patOr): the same
// dithered OR as FillOvalPatOrGray, over a region built by OpenRgn/Line/CloseRgn.
//
// A QuickDraw region is a set of unit pixel squares bounded by the path's grid
// lines, and every shadow polygon in the original has integer vertices, so
// sampling each row at its centre (y+0.5) gives spans with exact integer
// endpoints. The fill rule is even-odd, which for these simple closed outlines
// is the same as any other.
func (s *Surface) FillPolyPatOrGray(pts []Pt, idx uint8) {
	if len(pts) < 3 {
		return
	}
	minY, maxY := int(pts[0].V), int(pts[0].V)
	for _, p := range pts[1:] {
		if int(p.V) < minY {
			minY = int(p.V)
		}
		if int(p.V) > maxY {
			maxY = int(p.V)
		}
	}
	if minY < 0 {
		minY = 0
	}
	if maxY > s.H {
		maxY = s.H
	}
	var xs []float64
	for y := minY; y < maxY; y++ {
		yc := float64(y) + 0.5
		xs = xs[:0]
		for i := range pts {
			a, b := pts[i], pts[(i+1)%len(pts)]
			ay, by := float64(a.V), float64(b.V)
			if ay == by {
				continue
			}
			lo, hi := ay, by
			if lo > hi {
				lo, hi = hi, lo
			}
			if yc < lo || yc >= hi {
				continue
			}
			t := (yc - ay) / (by - ay)
			xs = append(xs, float64(a.H)+t*float64(b.H-a.H))
		}
		if len(xs) < 2 {
			continue
		}
		// Insertion sort: these spans never have more than a handful of
		// crossings, and a stable tiny sort keeps the fill order predictable.
		for i := 1; i < len(xs); i++ {
			for j := i; j > 0 && xs[j] < xs[j-1]; j-- {
				xs[j], xs[j-1] = xs[j-1], xs[j]
			}
		}
		row := y * s.W
		for i := 0; i+1 < len(xs); i += 2 {
			sx := ceilHalf(xs[i])
			ex := ceilHalf(xs[i+1])
			if sx < 0 {
				sx = 0
			}
			if ex > s.W {
				ex = s.W
			}
			for x := sx; x < ex; x++ {
				if !grayPatternSet(x, y) {
					continue
				}
				s.Pix[row+x] |= idx
				s.setMaskOpaque(row + x)
			}
		}
	}
}

// ceilHalf maps a crossing at x to the first pixel column whose centre (x+0.5)
// is at or past it.
func ceilHalf(x float64) int {
	n := int(x)
	if float64(n) < x {
		n++
	}
	return n
}

// CopyMode selects between the three CopyBits/CopyMask behaviours the static
// room path uses. The fourth QuickDraw transfer mode the game touches, patOr,
// belongs to the pen and is handled by the Fill*PatOrGray methods above.
type CopyMode int

const (
	// SrcCopy is CopyBits(..., srcCopy, nil): every source pixel is written,
	// including white. Nine object PICTs and all the room backgrounds rely on
	// it, and so do the deliberately opaque overlay blits (clock digits, the
	// dresser knob, the appliance screens).
	SrcCopy CopyMode = iota

	// Transparent is CopyBits(..., transparent, nil), the white colour key:
	// source pixels equal to the background colour are skipped. The background
	// is white, which is palette index 0, so this skips index 0 exactly.
	Transparent

	// Masked is CopyMask(src, mask, dst, ...): the source's mask decides, pixel
	// by pixel. Where a mask exists it is authoritative and a colour key is not
	// a substitute for it -- several sheets have opaque white pixels that a key
	// would wrongly drop.
	Masked
)

// Copy is CopyBits/CopyMask: move src[srcRect] into dst[dstRect].
//
// Sizes normally match; when they do not, QuickDraw stretches, and so does this,
// by the same nearest-source-pixel rule. The only stretch in the static path is
// LoadScaledGraphic's manhole-through-floor, and there the rect is exactly the
// PICT's 123x44 every time, so the stretch path is a safety net rather than a
// feature.
func (dst *Surface) Copy(src *Surface, srcRect, dstRect Rect, mode CopyMode) {
	sw, sh := int(srcRect.Wide()), int(srcRect.Tall())
	dw, dh := int(dstRect.Wide()), int(dstRect.Tall())
	if sw <= 0 || sh <= 0 || dw <= 0 || dh <= 0 {
		return
	}
	x0, y0, x1, y1, ok := dst.clip(dstRect)
	if !ok {
		return
	}
	if sw == dw && sh == dh {
		dst.copy1to1(src, srcRect, dstRect, mode, x0, y0, x1, y1)
		return
	}
	for dy := y0; dy < y1; dy++ {
		sy := int(srcRect.Top) + (dy-int(dstRect.Top))*sh/dh
		if sy < 0 || sy >= src.H {
			continue
		}
		srow, drow := sy*src.W, dy*dst.W
		for dx := x0; dx < x1; dx++ {
			sx := int(srcRect.Left) + (dx-int(dstRect.Left))*sw/dw
			if sx < 0 || sx >= src.W {
				continue
			}
			v := src.Pix[srow+sx]
			switch mode {
			case Transparent:
				if v == White8 {
					continue
				}
			case Masked:
				if !src.opaque(sx, sy) {
					continue
				}
			}
			dst.Pix[drow+dx] = v
			dst.setMaskOpaque(drow + dx)
		}
	}
}

// copy1to1 is Copy for the unscaled case, which is every blit the game makes but
// one. Splitting it out is worth the duplication: the source column becomes an
// addition instead of a multiply and a divide, the transfer mode leaves the inner
// loop, and an unmasked srcCopy row becomes a single memmove -- which is what a
// background tile is.
func (dst *Surface) copy1to1(src *Surface, srcRect, dstRect Rect, mode CopyMode, x0, y0, x1, y1 int) {
	dh := int(srcRect.Left) - int(dstRect.Left) // sx = dx + dh
	dv := int(srcRect.Top) - int(dstRect.Top)   // sy = dy + dv

	// Narrow the destination columns to those whose source column exists, once,
	// instead of testing every pixel.
	if lo := -dh; lo > x0 {
		x0 = lo
	}
	if hi := src.W - dh; hi < x1 {
		x1 = hi
	}
	if x0 >= x1 {
		return
	}
	srcOpaque := src.Mask == nil
	dstMasked := dst.Mask != nil

	for dy := y0; dy < y1; dy++ {
		sy := dy + dv
		if sy < 0 || sy >= src.H {
			continue
		}
		srow, drow := sy*src.W, dy*dst.W

		if mode == SrcCopy || (mode == Masked && srcOpaque) {
			copy(dst.Pix[drow+x0:drow+x1], src.Pix[srow+x0+dh:srow+x1+dh])
			if dstMasked {
				m := dst.Mask[drow+x0 : drow+x1]
				for i := range m {
					m[i] = 0xFF
				}
			}
			continue
		}

		for dx := x0; dx < x1; dx++ {
			sx := dx + dh
			v := src.Pix[srow+sx]
			if mode == Transparent {
				if v == White8 {
					continue
				}
			} else if src.Mask[srow+sx] == 0 {
				continue
			}
			dst.Pix[drow+dx] = v
			if dstMasked {
				dst.Mask[drow+dx] = 0xFF
			}
		}
	}
}

// CopyFull copies the whole of src over the whole of dst, which is what
// ReadyBackMap and RestoreWorkMap do (RoomGraphics.c:380,393). Both surfaces are
// the same size in every call the game makes.
func (dst *Surface) CopyFull(src *Surface) {
	dst.Copy(src, src.Bounds(), dst.Bounds(), SrcCopy)
}

// Clone returns an independent copy, mask included.
func (s *Surface) Clone() *Surface {
	c := &Surface{W: s.W, H: s.H, Pix: make([]uint8, len(s.Pix))}
	copy(c.Pix, s.Pix)
	if s.Mask != nil {
		c.Mask = make([]uint8, len(s.Mask))
		copy(c.Mask, s.Mask)
	}
	return c
}

// ToPaletted converts to an image/image.Paletted for PNG encoding. The palette
// is shared, so this is a cheap wrapper over a copy of the index plane.
func (s *Surface) ToPaletted() *image.Paletted {
	img := image.NewPaletted(image.Rect(0, 0, s.W, s.H), GoPalette())
	for y := 0; y < s.H; y++ {
		copy(img.Pix[y*img.Stride:y*img.Stride+s.W], s.Pix[y*s.W:(y+1)*s.W])
	}
	return img
}

// ToRGBA converts to straight RGBA, honouring the mask as alpha. This is the
// only place the palette is applied.
func (s *Surface) ToRGBA() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, s.W, s.H))
	for y := 0; y < s.H; y++ {
		for x := 0; x < s.W; x++ {
			c := Palette[s.Pix[y*s.W+x]]
			if !s.opaque(x, y) {
				c.A = 0
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}
