package render

// GliderPRO/Sources/ObjectDraw.c: the object draw helpers, one Go method per C
// function, in the order the C file has them.
//
// These are transcriptions, not reimplementations. Every line, every off-by-one
// endpoint and every palette index is the original's, because the art is only
// two thirds pixels: the tables, shelves, cabinets, counters, dressers and
// windows are drawn from rects and single-pixel lines, and their look is entirely
// in the arithmetic. Where the original's arithmetic is odd the oddity is kept
// and commented rather than tidied -- see the dresser's knob offsets and the
// cabinet's trailing FrameRect.
//
// Two conventions run through the whole file:
//
//   - Every drawing call in the original is wrapped in GetGWorld /
//     SetGWorld(backSrcMap) / SetGWorld(wasCPort, wasWorld). That is why almost
//     everything here writes to s.Back. The exceptions are the two places where
//     the original draws *after* restoring the port, and the port it restores to
//     is not backSrcMap -- see s.Work and the note in DrawCabinet.
//
//   - ColorLine, ColorRect, ColorOval and ColorRegion (ColorUtils.c) each save
//     the fore colour, set it, draw and restore it. A bare PaintRect, FrameRect
//     or Line therefore draws in black, which is the default foreground. Both
//     spellings appear below and the difference is deliberate.
//
// Deliberately absent: DrawInvisibleBlower, DrawLiftArea, DrawInvisObstacle,
// DrawInvisBounce, DrawInvisBonus and DrawSlider. All six draw a coloured frame
// around an otherwise invisible object, and all six are unreachable from
// DrawARoomsObjects -- those cases `break` with nothing drawn. They exist for the
// house editor, and belong with it.

// The magic numbers, kept as named constants where the original names them and
// as literals where it does not.
const (
	// Objects anchored to a fixed height in the room rather than to their own
	// rect. A table's legs run down to kTableBaseTop and no further, so a table
	// placed low in the room simply has shorter legs.
	kTikiPoleBase = 300
	kTableBaseTop = 296
	kStoolBase    = 304

	kTableShadowTop    = 312
	kTableShadowOffset = 12

	kBracketInset   = 18
	kShelfDeep      = 4
	kBracketThick   = 5
	kShelfShadowOff = 12
	kShelfThick     = 6 // GliderDefines.h; the shadow region needs it

	kCabinetDeep      = 4
	kCabinetShadowOff = 6

	kCounterFooterHigh = 12
	kCounterStripWide  = 6
	kCounterStripTall  = 29
	kCounterPanelDrop  = 12

	kDresserTopThick   = 4
	kDresserCrease     = 9
	kDresserDrawerDrop = 12
	kDresserSideSpare  = 14
)

// ---------------------------------------------------------------------------
// Blowers
// ---------------------------------------------------------------------------

// DrawSimpleBlowers is the nine vents and fans plus the three candles: a single
// masked blit out of the shared blower sheet.
func (s *Scene) DrawSimpleBlowers(what int16, theRect Rect) {
	s.maskSheet("blower", srcRects[what], theRect)
}

// DrawTiki draws the torch's pole, then the torch. The pole is five vertical
// lines run down to kTikiPoleBase, so a torch mounted high in the room grows a
// longer pole rather than floating.
func (s *Scene) DrawTiki(theRect Rect, down int16) {
	const (
		darkGrayC  = DkstGray8
		lightWoodC = Bamboo8
		darkWoodC  = PissYellow8
	)
	if theRect.Bottom < kTikiPoleBase+down {
		s.Back.Line(theRect.Left+11, theRect.Bottom-1, theRect.Left+11, kTikiPoleBase+down-1, darkGrayC)
		s.Back.Line(theRect.Left+12, theRect.Bottom-1, theRect.Left+12, kTikiPoleBase+down, lightWoodC)
		s.Back.Line(theRect.Left+13, theRect.Bottom-1, theRect.Left+13, kTikiPoleBase+down, darkWoodC)
		s.Back.Line(theRect.Left+14, theRect.Bottom-1, theRect.Left+14, kTikiPoleBase+down, darkWoodC)
		s.Back.Line(theRect.Left+15, theRect.Bottom-1, theRect.Left+15, kTikiPoleBase+down-1, darkGrayC)
	}
	s.maskSheet("blower", srcRects[kTiki], theRect)
}

// ---------------------------------------------------------------------------
// Furniture
// ---------------------------------------------------------------------------

// DrawTable is the single most intricate helper: a dithered elliptical shadow, a
// brown slab with a five-colour bevel, a five-pixel-wide leg down to
// kTableBaseTop, a shadow along the leg, and the sheet's 22-pixel table top
// centred over the slab.
//
// `down` should be playOriginV plus the neighbour's vertical room offset, and in
// the original it is bare playOriginV -- see the note at the call site.
func (s *Scene) DrawTable(tableTop Rect, down int16) {
	const (
		brownC = Brown8
		tanC   = Tan8
		dkRedC = DkRed28
		blackC = Black8
	)

	// The shadow: an ellipse a tenth as tall as the table is wide, centred on
	// kTableShadowTop and shifted down-left, dithered 50% through the gray pen.
	shadow := SetRect(tableTop.Left, 0, tableTop.Right, tableTop.Wide()/10)
	shadow = Offset(shadow, 0, -HalfTall(shadow)+kTableShadowTop+down)
	shadow = Offset(shadow, kTableShadowOffset, -kTableShadowOffset)
	s.Back.FillOvalPatOrGray(shadow, DkstGray8)

	s.Back.Fill(Inset(tableTop, 0, 1), brownC)

	// Three white pixels round the top corners, then the bevel.
	s.Back.Line(tableTop.Left, tableTop.Top+1, tableTop.Left, tableTop.Top+1, White8)
	s.Back.Line(tableTop.Left+1, tableTop.Top, tableTop.Right-2, tableTop.Top, White8)
	s.Back.Line(tableTop.Right-1, tableTop.Top+1, tableTop.Right-1, tableTop.Top+1, White8)

	s.Back.Line(tableTop.Left+1, tableTop.Top+1, tableTop.Right-2, tableTop.Top+1, tanC)
	s.Back.Line(tableTop.Left, tableTop.Top+2, tableTop.Left, tableTop.Bottom-2, tanC)

	s.Back.Line(tableTop.Left+1, tableTop.Bottom-1, tableTop.Right-2, tableTop.Bottom-1, blackC)
	s.Back.Line(tableTop.Right-1, tableTop.Top+2, tableTop.Right-1, tableTop.Bottom-2, blackC)

	s.Back.Line(tableTop.Left+1, tableTop.Bottom-2, tableTop.Right-2, tableTop.Bottom-2, dkRedC)

	if tableTop.Bottom < kTableBaseTop+down {
		hCenter := (tableTop.Left + tableTop.Right) / 2
		base := int16(kTableBaseTop) + down

		s.Back.Line(hCenter-3, tableTop.Bottom, hCenter-3, base, blackC)
		s.Back.Line(hCenter-2, tableTop.Bottom, hCenter-2, base, LtGray8)
		s.Back.Line(hCenter-1, tableTop.Bottom, hCenter-1, base, Gray8)
		s.Back.Line(hCenter, tableTop.Bottom, hCenter, base, DkGray8)
		s.Back.Line(hCenter+1, tableTop.Bottom, hCenter+1, base, blackC)

		// The leg's own shadow. When it would reach past the base the three
		// darker lines simply run the full length; otherwise they stop short,
		// each one pixel lower than the last, which is what gives the leg its
		// rounded foot.
		vShadow := tableTop.Bottom + tableTop.Wide()/4 - 2
		if vShadow > base {
			s.Back.Line(hCenter-2, tableTop.Bottom, hCenter-2, base, DkGray8)
			s.Back.Line(hCenter-1, tableTop.Bottom, hCenter-1, base, DkGray8)
			s.Back.Line(hCenter, tableTop.Bottom, hCenter, base, blackC)
		} else {
			s.Back.Line(hCenter-2, tableTop.Bottom, hCenter-2, vShadow, DkGray8)
			s.Back.Line(hCenter-1, tableTop.Bottom, hCenter-1, vShadow+1, DkGray8)
			s.Back.Line(hCenter, tableTop.Bottom, hCenter, vShadow+2, blackC)
		}
	}

	// tableSrc's corner is already (0,0), so the original omits the
	// ZeroRectCorner it uses everywhere else.
	dest := Offset(tableSrc, -HalfWide(tableSrc)+tableTop.Left+HalfWide(tableTop), kTableBaseTop+down)
	s.maskSheet("furniture", tableSrc, dest)
}

// DrawShelf is a table without legs: a wedge-shaped dithered shadow cast onto
// the wall behind, a brown slab, and two brackets underneath.
func (s *Scene) DrawShelf(shelfTop Rect) {
	const (
		brownC  = Brown8
		ltTanC  = LtTan8
		tanC    = Tan8
		dkRedC  = DkRed28
		blackC  = Black8
		shadowC = DkstGray8
	)

	// OpenRgn/Line/CloseRgn from (left, bottom): down-right, across, up by the
	// shelf's thickness less one, back up-left, and home.
	x, y := shelfTop.Left, shelfTop.Bottom
	pts := []Pt{{V: y, H: x}}
	x, y = x+kShelfShadowOff, y+kShelfShadowOff
	pts = append(pts, Pt{V: y, H: x})
	x += shelfTop.Wide() - kShelfDeep
	pts = append(pts, Pt{V: y, H: x})
	y += -kShelfThick + 1
	pts = append(pts, Pt{V: y, H: x})
	x, y = x-kShelfShadowOff, y-kShelfShadowOff
	pts = append(pts, Pt{V: y, H: x})
	s.Back.FillPolyPatOrGray(pts, shadowC)

	s.Back.Fill(Inset(shelfTop, 0, 1), brownC)

	s.Back.Line(shelfTop.Left+1, shelfTop.Top, shelfTop.Left+1+kShelfDeep, shelfTop.Top, ltTanC)
	s.Back.Line(shelfTop.Left, shelfTop.Top+1, shelfTop.Left+kShelfDeep, shelfTop.Top+1, tanC)
	s.Back.Line(shelfTop.Left, shelfTop.Top+2, shelfTop.Left+kShelfDeep, shelfTop.Top+2, tanC)
	s.Back.Line(shelfTop.Left, shelfTop.Top+3, shelfTop.Left+kShelfDeep, shelfTop.Top+3, tanC)
	s.Back.Line(shelfTop.Left+1, shelfTop.Bottom-1, shelfTop.Left+1+kShelfDeep, shelfTop.Bottom-1, dkRedC)
	s.Back.Line(shelfTop.Left+2+kShelfDeep, shelfTop.Bottom-1, shelfTop.Right-2, shelfTop.Bottom-1, blackC)
	s.Back.Line(shelfTop.Left+2+kShelfDeep, shelfTop.Top, shelfTop.Right-2, shelfTop.Top, tanC)
	s.Back.Line(shelfTop.Right-1, shelfTop.Top+1, shelfTop.Right-1, shelfTop.Bottom-2, blackC)

	bracket := ZeroCorner(shelfSrc)
	s.maskSheet("furniture", shelfSrc, Offset(bracket, shelfTop.Left+kBracketInset, shelfTop.Bottom))
	s.maskSheet("furniture", shelfSrc,
		Offset(bracket, shelfTop.Right-kBracketInset-kShelfDeep-kBracketThick, shelfTop.Bottom))
}

// DrawCabinet is a brown box with a lighter left edge, two nested rectangles of
// bevel lines suggesting a panelled door, two hinges and a handle.
func (s *Scene) DrawCabinet(cabinet Rect) {
	const (
		brownC  = Brown8
		dkGrayC = DkstGray8
		ltTanC  = LtTan8
		tanC    = Tan8
		dkRedC  = DkRed28
	)

	x, y := cabinet.Left, cabinet.Bottom
	pts := []Pt{{V: y, H: x}}
	x, y = x+kCabinetShadowOff, y+kCabinetShadowOff
	pts = append(pts, Pt{V: y, H: x})
	x += cabinet.Wide()
	pts = append(pts, Pt{V: y, H: x})
	y += -cabinet.Tall() + kCabinetDeep
	pts = append(pts, Pt{V: y, H: x})
	x, y = x-kCabinetShadowOff, y-kCabinetShadowOff
	pts = append(pts, Pt{V: y, H: x})
	s.Back.FillPolyPatOrGray(pts, dkGrayC)

	s.Back.Fill(Inset(cabinet, 1, 1), brownC)

	side := cabinet
	side.Right = side.Left + kCabinetDeep
	s.Back.Fill(side, tanC)

	s.Back.Line(cabinet.Left+1, cabinet.Top+1, cabinet.Left+kCabinetDeep, cabinet.Top+1, ltTanC)
	s.Back.Line(cabinet.Left+kCabinetDeep, cabinet.Top+1, cabinet.Right-3, cabinet.Top+1, tanC)

	s.Back.Line(cabinet.Left+kCabinetDeep+3, cabinet.Top+5, cabinet.Left+kCabinetDeep+3, cabinet.Bottom-6, tanC)
	s.Back.Line(cabinet.Left+kCabinetDeep+4, cabinet.Top+5, cabinet.Left+kCabinetDeep+4, cabinet.Bottom-6, tanC)
	s.Back.Line(cabinet.Left+kCabinetDeep+9, cabinet.Top+10, cabinet.Left+kCabinetDeep+9, cabinet.Bottom-11, dkGrayC)

	s.Back.Line(cabinet.Right-4, cabinet.Top+6, cabinet.Right-4, cabinet.Bottom-5, dkRedC)
	s.Back.Line(cabinet.Right-5, cabinet.Top+5, cabinet.Right-5, cabinet.Bottom-6, dkGrayC)
	s.Back.Line(cabinet.Right-10, cabinet.Top+10, cabinet.Right-10, cabinet.Bottom-11, tanC)

	s.Back.Line(cabinet.Left+kCabinetDeep+4, cabinet.Top+4, cabinet.Left+kCabinetDeep+4, cabinet.Top+4, ltTanC)
	s.Back.Line(cabinet.Left+kCabinetDeep+5, cabinet.Top+4, cabinet.Right-6, cabinet.Top+4, tanC)
	s.Back.Line(cabinet.Left+kCabinetDeep+10, cabinet.Top+9, cabinet.Right-11, cabinet.Top+9, dkGrayC)

	s.Back.Line(cabinet.Right-5, cabinet.Bottom-5, cabinet.Right-5, cabinet.Bottom-5, dkRedC)
	s.Back.Line(cabinet.Left+kCabinetDeep+6, cabinet.Bottom-4, cabinet.Right-5, cabinet.Bottom-4, dkRedC)
	s.Back.Line(cabinet.Left+kCabinetDeep+5, cabinet.Bottom-5, cabinet.Right-6, cabinet.Bottom-5, dkGrayC)

	s.Back.Line(cabinet.Left+kCabinetDeep+10, cabinet.Bottom-10, cabinet.Right-11, cabinet.Bottom-10, tanC)

	hinge := ZeroCorner(hingeSrc)
	s.maskSheet("furniture", hingeSrc, Offset(hinge, cabinet.Left+kCabinetDeep+2, cabinet.Top+10))
	s.maskSheet("furniture", hingeSrc, Offset(hinge, cabinet.Left+kCabinetDeep+2, cabinet.Bottom-26))

	handle := ZeroCorner(handleSrc)
	s.maskSheet("furniture", handleSrc,
		Offset(handle, cabinet.Right-8, cabinet.Top+HalfTall(cabinet)-HalfTall(handleSrc)))

	// ObjectDraw.c:483, and it does not do what it looks like it does.
	//
	// This FrameRect comes *after* SetGWorld(wasCPort, wasWorld), so it draws
	// into whatever port was current on entry -- and that is workSrcMap, not
	// backSrcMap, because DrawRoomBackground does a bare SetPort(workSrcMap)
	// (RoomGraphics.c:238) and never restores it. workSrcMap is then overwritten
	// wholesale by RestoreWorkMap() at the end of DrawLocale, so this outline
	// never reaches the screen: cabinets in the shipped game have no black
	// border. Reproduced exactly, including its invisibility.
	s.Work.FrameRect(cabinet, Black8)
}

// DrawSimpleFurniture is the waste basket and the milk crate: a masked blit and
// nothing else.
func (s *Scene) DrawSimpleFurniture(what int16, theRect Rect) {
	s.maskSheet("furniture", srcRects[what], theRect)
}

// DrawCounter is drawn entirely from rects and lines -- there is no counter PICT
// anywhere in the game. The top is a seven-line grey ramp that reads as a
// polished surface, the body is brown, and the footer is a dark recess.
func (s *Scene) DrawCounter(counter Rect) {
	const (
		brownC   = Brown8
		dkGrayC  = DkstGray8
		tanC     = Tan8
		blackC   = Black8
		dkstRedC = DkRed28
	)

	// The shadow region is traced from the counter's bottom-right corner and
	// runs up the right-hand side: this is the only piece of furniture lit from
	// the left rather than from above.
	x, y := counter.Right-2, counter.Bottom
	pts := []Pt{{V: y, H: x}}
	x, y = x+10, y-10
	pts = append(pts, Pt{V: y, H: x})
	y += -counter.Tall() + 29
	pts = append(pts, Pt{V: y, H: x})
	x += 2
	pts = append(pts, Pt{V: y, H: x})
	y -= 7
	pts = append(pts, Pt{V: y, H: x})
	x, y = x-12, y-12
	pts = append(pts, Pt{V: y, H: x})
	s.Back.FillPolyPatOrGray(pts, dkGrayC)

	s.Back.Fill(Inset(counter, 2, 2), brownC)

	footer := counter
	footer.Top = footer.Bottom - kCounterFooterHigh
	footer.Left += 2
	footer.Right -= 2
	s.Back.Fill(footer, dkGrayC)
	s.Back.Line(counter.Left+2, counter.Bottom-kCounterFooterHigh, counter.Right-3, counter.Bottom-kCounterFooterHigh, blackC)
	s.Back.Line(counter.Left+2, counter.Bottom-kCounterFooterHigh+1, counter.Right-3, counter.Bottom-kCounterFooterHigh+1, blackC)
	s.Back.Line(counter.Right-3, counter.Bottom-kCounterFooterHigh, counter.Right-3, counter.Bottom-1, blackC)
	s.Back.Line(counter.Left+2, counter.Bottom-kCounterFooterHigh, counter.Left+2, counter.Bottom-1, DkGray8)

	s.Back.Line(counter.Right-2, counter.Top, counter.Right-2, counter.Bottom-kCounterFooterHigh-1, dkstRedC)
	s.Back.Line(counter.Left+1, counter.Top+8, counter.Left+1, counter.Bottom-kCounterFooterHigh-1, tanC)

	// The polished top. Note left-1 and right: the ramp deliberately overhangs
	// the counter's rect by a pixel on each side.
	s.Back.Line(counter.Left-1, counter.Top, counter.Right, counter.Top, LtstGray)
	s.Back.Line(counter.Left-1, counter.Top+1, counter.Right, counter.Top+1, LtstGray2)
	s.Back.Line(counter.Left-1, counter.Top+2, counter.Right, counter.Top+2, LtstGray3)
	s.Back.Line(counter.Left-1, counter.Top+3, counter.Right, counter.Top+3, LtstGray4)
	s.Back.Line(counter.Left-1, counter.Top+4, counter.Right, counter.Top+4, LtstGray5)
	s.Back.Line(counter.Left-1, counter.Top+5, counter.Right, counter.Top+5, LtstGray5)
	s.Back.Line(counter.Left-1, counter.Top+6, counter.Right, counter.Top+6, LtstGray5)
	s.Back.Line(counter.Left-1, counter.Top, counter.Left-1, counter.Top+6, LtstGray)

	s.Back.Line(counter.Right, counter.Top, counter.Right, counter.Top+6, LtGray8)
	s.Back.Line(counter.Left+1, counter.Top+7, counter.Right-2, counter.Top+7, dkstRedC)
	s.Back.Line(counter.Left+1, counter.Top+8, counter.Right-2, counter.Top+8, dkstRedC)

	// ObjectDraw.c:630-644, and like DrawCabinet's FrameRect this lands in
	// workSrcMap rather than backSrcMap and is then erased by RestoreWorkMap.
	// Counters in the shipped game have no drawer panels -- the code that would
	// draw them runs on every counter and is thrown away every time.
	nRects := counter.Wide() / 40
	if nRects == 0 {
		nRects = 1
	}
	width := ((counter.Wide() - kCounterStripWide) / nRects) - kCounterStripWide
	panel := SetRect(0, 0, width, counter.Tall()-kCounterStripTall)
	panel = Offset(panel, counter.Left+kCounterStripWide, counter.Top+kCounterPanelDrop)
	for i := int16(0); i < nRects; i++ {
		s.Work.HiliteRect(panel, tanC, dkstRedC)
		s.Work.HiliteRect(Inset(panel, 4, 4), dkstRedC, tanC)
		panel = Offset(panel, kCounterStripWide+width, 0)
	}
}

// DrawDresser is the pumpkin-and-yellow chest of drawers: a shadow, a bevelled
// body, a thicker top slab, two vertical creases, then one bevelled drawer per
// 30 pixels of height with a knob at each end.
func (s *Scene) DrawDresser(dresser Rect) {
	const (
		yellowC  = PissYellow8
		brownC   = Brown8
		ltTanC   = LtTan8
		dkstRedC = DkRed28
	)

	// Traced anticlockwise from below the left foot, and the only shadow that
	// starts outside its object's rect.
	x, y := dresser.Left+10, dresser.Bottom+9
	pts := []Pt{{V: y, H: x}}
	x += dresser.Wide() - 11
	pts = append(pts, Pt{V: y, H: x})
	x, y = x+9, y-9
	pts = append(pts, Pt{V: y, H: x})
	y += -dresser.Tall() + 12
	pts = append(pts, Pt{V: y, H: x})
	x, y = x-9, y-9
	pts = append(pts, Pt{V: y, H: x})
	x += -dresser.Wide() + 11
	pts = append(pts, Pt{V: y, H: x})
	s.Back.FillPolyPatOrGray(pts, DkstGray8)

	body := Inset(dresser, 2, 2)
	s.Back.Fill(body, Pumpkin8)
	s.Back.HiliteRect(body, Orange8, dkstRedC)

	slab := dresser
	slab.Bottom = slab.Top + kDresserTopThick
	s.Back.Fill(slab, PissYellow8)
	s.Back.HiliteRect(slab, ltTanC, dkstRedC)
	s.Back.Line(dresser.Left+2, dresser.Top+kDresserTopThick, dresser.Right-3, dresser.Top+kDresserTopThick, Red48)

	s.Back.Line(dresser.Left+kDresserCrease, dresser.Top+kDresserTopThick+1,
		dresser.Left+kDresserCrease, dresser.Bottom-4, Red48)
	s.Back.Line(dresser.Right-kDresserCrease, dresser.Top+kDresserTopThick+1,
		dresser.Right-kDresserCrease, dresser.Bottom-4, Orange8)

	nRects := dresser.Tall() / 30
	if nRects == 0 {
		nRects = 1
	}
	height := (dresser.Tall()-14)/nRects - 4
	drawer := SetRect(0, 0, dresser.Wide()-kDresserSideSpare, height)
	drawer = Offset(drawer, dresser.Left+7, dresser.Top+10)
	for i := int16(0); i < nRects; i++ {
		s.Back.Line(drawer.Left+1, drawer.Bottom, drawer.Right, drawer.Bottom, dkstRedC)
		s.Back.Line(drawer.Right, drawer.Top+1, drawer.Right, drawer.Bottom, dkstRedC)
		s.Back.Fill(drawer, yellowC)
		s.Back.HiliteRect(drawer, ltTanC, brownC)
		s.Back.HiliteRect(Inset(drawer, 1, 1), ltTanC, brownC)

		// The knobs. HalfRectTall is used for *both* the horizontal and the
		// vertical offset in the original -- almost certainly a typo for
		// HalfRectWide on the h axis -- so a drawer's knobs sit half the
		// drawer's *height* in from each end. Wide flat drawers therefore have
		// knobs bunched near the middle. Kept, because it is what the game
		// looks like.
		knob := SetRect(-4, -4, 4, 4)
		knob = Offset(knob, HalfTall(drawer), HalfTall(drawer))
		knob = Offset(knob, drawer.Left, drawer.Top)
		s.opaqueSheet("furniture", knobSrc, knob)

		knob = SetRect(-4, -4, 4, 4)
		knob = Offset(knob, -HalfTall(drawer), HalfTall(drawer))
		knob = Offset(knob, drawer.Right, drawer.Top)
		s.opaqueSheet("furniture", knobSrc, knob)

		drawer = Offset(drawer, 0, kDresserTopThick+height)
	}

	s.maskSheet("furniture", leftFootSrc,
		Offset(ZeroCorner(leftFootSrc), dresser.Left+6, dresser.Bottom-2))
	s.maskSheet("furniture", rightFootSrc,
		Offset(ZeroCorner(rightFootSrc), dresser.Right-19, dresser.Bottom-2))
}

// DrawDeckTable is DrawTable's geometry in bamboo and gold, with a white-topped
// leg instead of a grey one.
func (s *Scene) DrawDeckTable(tableTop Rect, down int16) {
	const (
		bambooC = Bamboo8
		brownC  = Brown8
		dkGrayC = DkstGray8
	)

	shadow := SetRect(tableTop.Left, 0, tableTop.Right, tableTop.Wide()/10)
	shadow = Offset(shadow, 0, -HalfTall(shadow)+kTableShadowTop+down)
	shadow = Offset(shadow, kTableShadowOffset, -kTableShadowOffset)
	s.Back.FillOvalPatOrGray(shadow, dkGrayC)

	s.Back.Fill(Inset(tableTop, 0, 1), Gold)

	s.Back.Line(tableTop.Left, tableTop.Top+1, tableTop.Left, tableTop.Top+1, White8)
	s.Back.Line(tableTop.Left+1, tableTop.Top, tableTop.Right-2, tableTop.Top, White8)
	s.Back.Line(tableTop.Right-1, tableTop.Top+1, tableTop.Right-1, tableTop.Top+1, White8)

	s.Back.Line(tableTop.Left+1, tableTop.Top+1, tableTop.Right-2, tableTop.Top+1, Yellow)
	s.Back.Line(tableTop.Left, tableTop.Top+2, tableTop.Left, tableTop.Bottom-2, Yellow)

	s.Back.Line(tableTop.Left+1, tableTop.Bottom-1, tableTop.Right-2, tableTop.Bottom-1, brownC)
	s.Back.Line(tableTop.Right-1, tableTop.Top+2, tableTop.Right-1, tableTop.Bottom-2, brownC)

	s.Back.Line(tableTop.Left+1, tableTop.Bottom-2, tableTop.Right-2, tableTop.Bottom-2, bambooC)

	if tableTop.Bottom < kTableBaseTop+down {
		hCenter := (tableTop.Left + tableTop.Right) / 2
		base := int16(kTableBaseTop) + down

		s.Back.Line(hCenter-3, tableTop.Bottom, hCenter-3, base, dkGrayC)
		s.Back.Line(hCenter-2, tableTop.Bottom, hCenter-2, base, White8)
		s.Back.Line(hCenter-1, tableTop.Bottom, hCenter-1, base, White8)
		s.Back.Line(hCenter, tableTop.Bottom, hCenter, base, LtGray8)
		s.Back.Line(hCenter+1, tableTop.Bottom, hCenter+1, base, dkGrayC)

		vShadow := tableTop.Bottom + tableTop.Wide()/4 - 2
		if vShadow > base {
			s.Back.Line(hCenter-2, tableTop.Bottom, hCenter-2, base, LtGray8)
			s.Back.Line(hCenter-1, tableTop.Bottom, hCenter-1, base, LtGray8)
			s.Back.Line(hCenter, tableTop.Bottom, hCenter, base, dkGrayC)
		} else {
			s.Back.Line(hCenter-2, tableTop.Bottom, hCenter-2, vShadow, LtGray8)
			s.Back.Line(hCenter-1, tableTop.Bottom, hCenter-1, vShadow+1, LtGray8)
			s.Back.Line(hCenter, tableTop.Bottom, hCenter, vShadow+2, dkGrayC)
		}
	}

	dest := Offset(ZeroCorner(deckSrc), -HalfWide(deckSrc)+tableTop.Left+HalfWide(tableTop), kTableBaseTop+down)
	s.maskSheet("furniture", deckSrc, dest)
}

// DrawStool draws a six-pixel-wide leg down to kStoolBase, then the stool.
func (s *Scene) DrawStool(theRect Rect, down int16) {
	const (
		grayC   = DkGray28
		dkGrayC = DkstGray8
	)
	if theRect.Bottom < kStoolBase+down {
		base := int16(kStoolBase) + down
		s.Back.Line(theRect.Left+21, theRect.Bottom-1, theRect.Left+21, base-1, DkGray8)
		s.Back.Line(theRect.Left+22, theRect.Bottom-1, theRect.Left+22, base, Gray28)
		s.Back.Line(theRect.Left+23, theRect.Bottom-1, theRect.Left+23, base, DkGray8)
		s.Back.Line(theRect.Left+24, theRect.Bottom-1, theRect.Left+24, base, DkGray38)
		s.Back.Line(theRect.Left+25, theRect.Bottom-1, theRect.Left+25, base, grayC)
		s.Back.Line(theRect.Left+26, theRect.Bottom-1, theRect.Left+26, base-1, dkGrayC)
	}
	s.maskSheet("furniture", srcRects[kStool], theRect)
}

// ---------------------------------------------------------------------------
// Bonuses
// ---------------------------------------------------------------------------

// DrawRedClock is the digital clock: the sprite, then the wall-clock time in
// four 4x6 digits. The leading hour digit is omitted before ten o'clock, and the
// gap it leaves is not closed up -- the minutes stay put, so "9:05" is drawn
// one digit-width right of where "10:05" would start.
func (s *Scene) DrawRedClock(theRect Rect) {
	s.maskSheet("bonus", srcRects[kRedClock], theRect)

	hour := s.Clock.Hour() % 12
	if hour == 0 {
		hour = 12
	}
	minutes := s.Clock.Minute()

	dest := Offset(SetRect(0, 0, 4, 6), theRect.Left+5, theRect.Top+7)
	if hour > 9 {
		s.drawClockDigit(hour/10, dest)
	}
	dest = Offset(dest, 4, 0)
	s.drawClockDigit(hour%10, dest)
	dest = Offset(dest, 6, 0)
	s.drawClockDigit(minutes/10, dest)
	dest = Offset(dest, 4, 0)
	s.drawClockDigit(minutes%10, dest)
}

// drawClockDigit blits one digit opaquely: srcCopy, not CopyMask, so the digit
// cell's white background covers whatever the clock face had there.
func (s *Scene) drawClockDigit(number int, dest Rect) {
	if number < 0 || number > 10 {
		return
	}
	s.opaqueSheet("bonus", digitSrc[number], dest)
}

// DrawBlueClock is the analogue clock: the sprite, then two hands in black. The
// minute hand is quantised to the nearest five minutes, which is one of the
// twelve directions the hand table holds.
func (s *Scene) DrawBlueClock(theRect Rect) {
	s.maskSheet("bonus", srcRects[kBlueClock], theRect)
	s.drawClockHands(Pt{H: theRect.Left + 13, V: theRect.Top + 13},
		((s.Clock.Minute()+2)/5)%12, s.Clock.Hour()%12)
}

// DrawYellowClock is DrawBlueClock two pixels lower.
func (s *Scene) DrawYellowClock(theRect Rect) {
	s.maskSheet("bonus", srcRects[kYellowClock], theRect)
	s.drawClockHands(Pt{H: theRect.Left + 13, V: theRect.Top + 15},
		((s.Clock.Minute()+2)/5)%12, s.Clock.Hour()%12)
}

// DrawCuckoo is the grandfather clock: a bigger face, so bigger hands, and they
// are drawn in white rather than black.
func (s *Scene) DrawCuckoo(theRect Rect) {
	s.maskSheet("bonus", srcRects[kCuckoo], theRect)
	s.drawLargeClockHands(Pt{H: theRect.Left + 19, V: theRect.Top + 31},
		((s.Clock.Minute()+2)/5)%12, s.Clock.Hour()%12)
}

// The twelve clock-hand directions, as (dh, dv) deltas from the hand's pivot.
// Three lengths are used: 6 and 4 for the small clocks' minute and hour hands,
// 10 and 6 for the cuckoo's. Transcribed rather than computed from sin/cos --
// they are hand-chosen and a trigonometric rounding would not reproduce them.
var (
	handDelta6 = [12]Pt{
		{H: 0, V: -6}, {H: 3, V: -5}, {H: 5, V: -3}, {H: 6, V: 0},
		{H: 5, V: 3}, {H: 3, V: 5}, {H: 0, V: 6}, {H: -3, V: 5},
		{H: -5, V: 3}, {H: -6, V: 0}, {H: -5, V: -3}, {H: -3, V: -5},
	}
	handDelta4 = [12]Pt{
		{H: 0, V: -4}, {H: 2, V: -3}, {H: 3, V: -2}, {H: 4, V: 0},
		{H: 3, V: 2}, {H: 2, V: 3}, {H: 0, V: 4}, {H: -2, V: 3},
		{H: -3, V: 2}, {H: -4, V: 0}, {H: -3, V: -2}, {H: -2, V: -3},
	}
	handDelta10 = [12]Pt{
		{H: 0, V: -10}, {H: 5, V: -9}, {H: 9, V: -5}, {H: 10, V: 0},
		{H: 9, V: 5}, {H: 5, V: 9}, {H: 0, V: 10}, {H: -5, V: 9},
		{H: -9, V: 5}, {H: -10, V: 0}, {H: -9, V: -5}, {H: -5, V: -9},
	}
)

func (s *Scene) drawClockHands(where Pt, bigHand, littleHand int) {
	s.hand(where, handDelta6, bigHand, Black8)
	s.hand(where, handDelta4, littleHand, Black8)
}

// drawLargeClockHands is the only helper in the game that changes the pen
// colour: ForeColor(whiteColor) before and ForeColor(blackColor) after, so the
// cuckoo's hands are white where every other clock's are black.
func (s *Scene) drawLargeClockHands(where Pt, bigHand, littleHand int) {
	s.hand(where, handDelta10, bigHand, White8)
	s.hand(where, handDelta6, littleHand, White8)
}

// hand is MoveTo(where) + Line(delta): a line from the pivot to the pivot plus
// the delta, both endpoints inclusive, so the pivot pixel is always drawn.
func (s *Scene) hand(where Pt, deltas [12]Pt, which int, idx uint8) {
	if which < 0 || which > 11 {
		return
	}
	d := deltas[which]
	s.Back.Line(where.H, where.V, where.H+d.H, where.V+d.V, idx)
}

// DrawSimplePrizes is the paper, battery, rubber bands, helium and star: a
// masked blit out of the bonus sheet.
func (s *Scene) DrawSimplePrizes(what int16, theRect Rect) {
	s.maskSheet("bonus", srcRects[what], theRect)
}

// DrawGreaseRt draws a right-facing grease canister, upright or spilled. The
// spill itself is a two-pixel-high black bar `distance-5` long, painted with the
// default pen -- PaintRect, not ColorRect, so it really is black rather than any
// palette colour.
func (s *Scene) DrawGreaseRt(theRect Rect, distance int16, state bool) {
	dest := theRect
	if state {
		s.maskSheet("bonus", greaseSrcRt[0], dest)
		return
	}
	dest = Offset(dest, 6, 0)
	s.maskSheet("bonus", greaseSrcRt[3], dest)
	spill := Offset(SetRect(0, -2, distance-5, 0), dest.Right-1, dest.Bottom)
	s.Back.Fill(spill, Black8)
}

// DrawGreaseLf mirrors DrawGreaseRt.
func (s *Scene) DrawGreaseLf(theRect Rect, distance int16, state bool) {
	dest := theRect
	if state {
		s.maskSheet("bonus", greaseSrcLf[0], dest)
		return
	}
	dest = Offset(dest, -6, 0)
	s.maskSheet("bonus", greaseSrcLf[3], dest)
	spill := Offset(SetRect(-distance+5, -2, 0, 0), dest.Left+1, dest.Bottom)
	s.Back.Fill(spill, Black8)
}

// DrawFoil is the aluminium foil bonus.
func (s *Scene) DrawFoil(theRect Rect) {
	s.maskSheet("bonus", srcRects[kFoil], theRect)
}
