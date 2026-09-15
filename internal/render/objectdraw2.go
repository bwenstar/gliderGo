package render

// GliderPRO/Sources/ObjectDraw2.c: the rest of the object draw helpers --
// mailboxes, transports, switches, lights, appliances, clutter, and the four
// generic PICT painters every remaining object routes through.
//
// The four generic painters at the bottom of the file are the interesting part,
// because they are where the original's four masking strategies show up as four
// distinct functions:
//
//	DrawPictObject          srcCopy, no mask at all: nine opaque PICTs
//	DrawPictWithMaskObject  a temp GWorld pair, art + 1-bit mask: two PICTs
//	DrawPictSansWhiteObject one temp GWorld, `transparent`, white keyed out: 20
//	DrawCustPictSansWhite   the same, for a house's own picture by id
//
// The extractor has already resolved the first three into an alpha channel (see
// tools/extract_art.py), so strategies two and three collapse to the same Masked
// copy here. That is exact rather than convenient: the background colour is white
// and white is palette index 0, so keying out white and honouring a mask built by
// keying out white are the same operation. House pictures keep their `transparent`
// mode because their PNGs are RGB -- the transfer mode for a house picture is a
// call-site decision, not a property of the resource.
//
// Deliberately absent, for the reason given in objectdraw.go: DrawInvisTransport,
// DrawInvisibleSwitch, DrawTrigger, DrawSoundTrigger and DrawInvisLight.

const (
	// kMailboxBase, like kTableBaseTop and kStoolBase, is the floor level a
	// mailbox's post runs down to regardless of where the mailbox itself sits.
	kMailboxBase = 296

	kWindowSillThick   = 7
	kTrackLightSpacing = 64
	kNumTrackLights    = 3
)

// ---------------------------------------------------------------------------
// Transports
// ---------------------------------------------------------------------------

// DrawMailboxLeft draws the post, then the mailbox. The post is fourteen
// vertical lines, and unlike the table's and the stool's they do not all end at
// the same height: the first ends at kMailboxBase and the next three one, two and
// three pixels lower, which rounds the post's foot off where it meets the ground.
func (s *Scene) DrawMailboxLeft(theRect Rect, down int16) {
	const (
		darkGrayC  = DkGray28
		lightWoodC = PissYellow8
		darkWoodC  = Brown8
	)
	if theRect.Bottom < down+kMailboxBase {
		base := down + kMailboxBase
		s.Back.Line(theRect.Left+49, theRect.Bottom, theRect.Left+49, base, darkGrayC)
		s.Back.Line(theRect.Left+50, theRect.Bottom, theRect.Left+50, base+1, lightWoodC)
		s.Back.Line(theRect.Left+51, theRect.Bottom, theRect.Left+51, base+2, lightWoodC)
		s.Back.Line(theRect.Left+52, theRect.Bottom, theRect.Left+52, base+3, lightWoodC)
		for h := int16(53); h <= 61; h++ {
			s.Back.Line(theRect.Left+h, theRect.Bottom, theRect.Left+h, base+3, darkWoodC)
		}
		s.Back.Line(theRect.Left+62, theRect.Bottom, theRect.Left+62, base+3, darkGrayC)
	}
	s.maskObject(kMailboxLf, srcRects[kMailboxLf], theRect)
}

// DrawMailboxRight is DrawMailboxLeft with the post fifteen pixels left, because
// the mailbox itself is at the other end of the art.
func (s *Scene) DrawMailboxRight(theRect Rect, down int16) {
	const (
		darkGrayC  = DkGray28
		lightWoodC = PissYellow8
		darkWoodC  = Brown8
	)
	if theRect.Bottom < down+kMailboxBase {
		base := down + kMailboxBase
		s.Back.Line(theRect.Left+34, theRect.Bottom, theRect.Left+34, base, darkGrayC)
		s.Back.Line(theRect.Left+35, theRect.Bottom, theRect.Left+35, base+1, lightWoodC)
		s.Back.Line(theRect.Left+36, theRect.Bottom, theRect.Left+36, base+2, lightWoodC)
		s.Back.Line(theRect.Left+37, theRect.Bottom, theRect.Left+37, base+3, lightWoodC)
		for h := int16(38); h <= 46; h++ {
			s.Back.Line(theRect.Left+h, theRect.Bottom, theRect.Left+h, base+3, darkWoodC)
		}
		s.Back.Line(theRect.Left+47, theRect.Bottom, theRect.Left+47, base+3, darkGrayC)
	}
	s.maskObject(kMailboxRt, srcRects[kMailboxRt], theRect)
}

// DrawSimpleTransport is the two invisible-to-the-player floor and ceiling
// transports' visible art, plus the in-room duct openings.
func (s *Scene) DrawSimpleTransport(what int16, theRect Rect) {
	s.maskSheet("trans", srcRects[what], theRect)
}

// ---------------------------------------------------------------------------
// Switches
// ---------------------------------------------------------------------------

// DrawSwitch is the five wall switches, whose only difference is which pair of
// sub-rects they index. All five are srcCopy rather than CopyMask -- the switch
// sheet is the one sheet with no companion mask PICT, because a switch is a
// rectangle of wall plate with nothing to cut out.
func (s *Scene) DrawSwitch(what int16, theRect Rect, state bool) {
	var pair *[2]Rect
	switch what {
	case kLightSwitch:
		pair = &lightSwitchSrc
	case kMachineSwitch:
		pair = &machineSwitchSrc
	case kThermostat:
		pair = &thermostatSrc
	case kPowerSwitch:
		pair = &powerSrc
	case kKnifeSwitch:
		pair = &knifeSwitchSrc
	default:
		return
	}
	// Index 0 is the on frame, 1 the off frame -- the opposite way round from
	// every other two-state object in the game.
	i := 1
	if state {
		i = 0
	}
	s.opaqueSheet("switch", pair[i], theRect)
}

// ---------------------------------------------------------------------------
// Lights
// ---------------------------------------------------------------------------

// DrawSimpleLight is the ceiling light, bulb, table lamp and their kin.
func (s *Scene) DrawSimpleLight(what int16, theRect Rect) {
	s.maskSheet("light", srcRects[what], theRect)
}

// DrawFlourescent is a stretchable fixture: twelve horizontal lines of tube from
// sixteen pixels inside each end, then a fixed end cap blitted at each end. The
// tube's twelve rows are a fixed ramp, so the fixture is always twelve pixels
// tall no matter how long the house makes it.
func (s *Scene) DrawFlourescent(theRect Rect) {
	const (
		grayC   = LtGray8
		gray2C  = LtstGray5
		gray3C  = LtstGray4
		gray4C  = LtstGray
		violetC = PaleViolet
	)
	rows := [12]uint8{
		grayC, gray2C, gray2C, gray3C, gray4C, violetC,
		White8, White8, White8, White8, White8, violetC,
	}
	for i, c := range rows {
		v := theRect.Top + int16(i)
		s.Back.Line(theRect.Left+16, v, theRect.Right-17, v, c)
	}

	s.maskSheet("light", flourescentSrc1,
		Offset(ZeroCorner(flourescentSrc1), theRect.Left, theRect.Top))

	// Right-aligned: shift the zero-cornered rect left by its own width first,
	// so the offset by theRect.Right lands its right edge on the fixture's.
	cap2 := ZeroCorner(flourescentSrc2)
	cap2 = Offset(cap2, -cap2.Right, 0)
	s.maskSheet("light", flourescentSrc2, Offset(cap2, theRect.Right, theRect.Top))
}

// DrawTrackLight is a six-line rail with a lamp at each end and one lamp per 64
// pixels in between, cycling through the three lamp frames so the row does not
// look stamped.
//
// The rail is drawn three pixels *above* theRect.Top, which is the one place in
// the game where an object paints outside the rect GetObjectRect gave it.
func (s *Scene) DrawTrackLight(theRect Rect) {
	const (
		grayC  = LtGray8
		gray2C = Gray28
		gray3C = LtstGray4
		gray4C = DkGray8
	)
	rail := [6]uint8{gray2C, grayC, grayC, gray3C, gray4C, gray3C}
	for i, c := range rail {
		v := theRect.Top - 3 + int16(i)
		s.Back.Line(theRect.Left, v, theRect.Right-1, v, c)
	}

	lamp := ZeroCorner(trackLightSrc[0])
	s.maskSheet("light", trackLightSrc[0], Offset(lamp, theRect.Left, theRect.Top))

	// The right-hand lamp is positioned from trackLightSrc[0] but drawn from
	// trackLightSrc[2]. All three frames are 24x24, so the mismatch is harmless
	// -- but it is the original's, and a later frame of a different size would
	// make it visible.
	right := Offset(lamp, -lamp.Right, 0)
	s.maskSheet("light", trackLightSrc[2], Offset(right, theRect.Right, theRect.Top))

	howMany := ((theRect.Wide() - trackLightSrc[0].Wide()) / kTrackLightSpacing) - 1
	if howMany <= 0 {
		return
	}
	spread := (theRect.Wide() - trackLightSrc[0].Wide()) / (howMany + 1)
	which := 0
	for i := int16(0); i < howMany; i++ {
		dst := Offset(lamp, theRect.Left, theRect.Top)
		dst = Offset(dst, spread*(i+1), 0)
		// Incremented before the draw, so the first filler lamp is frame 1 and
		// frame 0 only reappears after wrapping.
		which++
		if which >= kNumTrackLights {
			which = 0
		}
		s.maskSheet("light", trackLightSrc[which], dst)
	}
}

// ---------------------------------------------------------------------------
// Appliances
// ---------------------------------------------------------------------------

// DrawSimpleAppliance is the shredder, toaster, CDs and the rest of the
// appliances with no lit state of their own.
func (s *Scene) DrawSimpleAppliance(what int16, theRect Rect) {
	s.maskSheet("appliance", srcRects[what], theRect)
}

// DrawMacPlus draws the case only when the room is lit, but the screen always:
// a Mac Plus in a dark room is a floating rectangle of screen, which is the
// intended joke and not a bug.
func (s *Scene) DrawMacPlus(theRect Rect, isOn, isLit bool) {
	if isLit {
		s.maskSheet("appliance", srcRects[kMacPlus], theRect)
	}
	screen := Offset(ZeroCorner(PlusScreen1), theRect.Left+10, theRect.Top+7)
	src := PlusScreen1
	if isOn {
		src = PlusScreen2
	}
	s.opaqueSheet("appliance", src, screen)
}

// DrawTV is DrawMacPlus with the case coming from its own PICT pair rather than
// from the shared sheet.
func (s *Scene) DrawTV(theRect Rect, isOn, isLit bool) {
	if isLit {
		s.maskObject(kTV, srcRects[kTV], theRect)
	}
	screen := Offset(ZeroCorner(TVScreen1), theRect.Left+17, theRect.Top+10)
	src := TVScreen1
	if isOn {
		src = TVScreen2
	}
	s.opaqueSheet("appliance", src, screen)
}

// DrawCoffee is the coffee maker: the body when lit, the four-by-eight power LED
// always.
func (s *Scene) DrawCoffee(theRect Rect, isOn, isLit bool) {
	if isLit {
		s.maskSheet("appliance", srcRects[kCoffee], theRect)
	}
	light := Offset(ZeroCorner(CoffeeLight1), theRect.Left+32, theRect.Top+57)
	src := CoffeeLight1
	if isOn {
		src = CoffeeLight2
	}
	s.opaqueSheet("appliance", src, light)
}

// DrawOutlet is the wall socket: masked, and with no on state at all.
func (s *Scene) DrawOutlet(theRect Rect) {
	s.maskSheet("appliance", srcRects[kOutlet], theRect)
}

// DrawVCR draws the body from its own PICT pair when lit, then the clock
// display: blank when off, a blinking 12:00 when on.
func (s *Scene) DrawVCR(theRect Rect, isOn, isLit bool) {
	if isLit {
		s.maskObject(kVCR, srcRects[kVCR], theRect)
	}
	clock := Offset(ZeroCorner(VCRTime1), theRect.Left+64, theRect.Top+6)
	src := VCRTime1
	if isOn {
		src = VCRTime2
	}
	s.opaqueSheet("appliance", src, clock)
}

// DrawStereo is the VCR pattern with a single four-by-one pixel LED.
func (s *Scene) DrawStereo(theRect Rect, isOn, isLit bool) {
	if isLit {
		s.maskObject(kStereo, srcRects[kStereo], theRect)
	}
	light := Offset(ZeroCorner(StereoLight1), theRect.Left+56, theRect.Top+20)
	src := StereoLight1
	if isOn {
		src = StereoLight2
	}
	s.opaqueSheet("appliance", src, light)
}

// DrawMicrowave draws the body when lit, then three copies of a 16x35 door panel
// side by side -- the open panel when on, the closed one when lit.
//
// Note the `else if`: an unlit microwave that is also off draws nothing at all,
// not even its door. Every other appliance here draws its indicator
// unconditionally, so the microwave is the one that vanishes completely in the
// dark.
func (s *Scene) DrawMicrowave(theRect Rect, isOn, isLit bool) {
	if isLit {
		s.maskObject(kMicrowave, srcRects[kMicrowave], theRect)
	}
	panel := Offset(ZeroCorner(MicroOn), theRect.Left+14, theRect.Top+13)
	switch {
	case isOn:
		for i := 0; i < 3; i++ {
			s.opaqueSheet("appliance", MicroOn, panel)
			panel = Offset(panel, 16, 0)
		}
	case isLit:
		for i := 0; i < 3; i++ {
			s.opaqueSheet("appliance", MicroOff, panel)
			panel = Offset(panel, 16, 0)
		}
	}
}

// ---------------------------------------------------------------------------
// Clutter
// ---------------------------------------------------------------------------

// DrawFish is the fish that leaps out of the water. Its static frame comes from
// the 36x33 enemy sheet, not from the eight-frame 16x16 fishSrcMap -- that one is
// animation-only. Unlike almost everything else in this file it does not save and
// restore the port: DrawFish, DrawDrip and DrawBall CopyMask straight into
// backSrcMap and leave the current port alone.
func (s *Scene) DrawFish(what int16, theRect Rect) {
	s.maskSheet("enemy", srcRects[what], theRect)
}

// DrawDrip is the drop of water hanging from a faucet. DripSrc[3] rather than
// srcRects[kDrip]: the two are the same 16x12 size but sit at different heights
// in the sheet, and frame 3 is the hanging one.
func (s *Scene) DrawDrip(theRect Rect) {
	s.maskSheet("drip", DripSrc[3], theRect)
}

// DrawMirror is drawn from four nested frames over a white fill: grey, two rows
// of earth blue, then grey again. Mirrors do not reflect -- the "reflection" a
// player sees is whatever the room background already had there, and this
// paints over it.
func (s *Scene) DrawMirror(mirror Rect) {
	const grayC = DkGray28
	s.Back.Fill(mirror, White8)
	s.Back.FrameRect(mirror, grayC)
	r := Inset(mirror, 1, 1)
	s.Back.FrameRect(r, EarthBlue8)
	r = Inset(r, 1, 1)
	s.Back.FrameRect(r, EarthBlue8)
	r = Inset(r, 1, 1)
	s.Back.FrameRect(r, grayC)
}

// DrawSimpleClutter is the vases, the bear, the rug and the rest: a masked blit
// out of the clutter sheet.
func (s *Scene) DrawSimpleClutter(what int16, theRect Rect) {
	s.maskSheet("clutter", srcRects[what], theRect)
}

// DrawFlower is the one object whose art has no srcRects entry: the six frames
// are different sizes, so flowerSrc[which] is both the art and the size, indexed
// by data.i.pict rather than by any animation state.
func (s *Scene) DrawFlower(theRect Rect, which int16) {
	if which < 0 || which > 5 {
		return
	}
	s.maskSheet("clutter", flowerSrc[which], theRect)
}

// DrawWallWindow is drawn entirely from rects: a brown surround, two sills, an
// inside frame, and two sky-blue panes with a bevelled reveal. It is the only
// piece of clutter with no art at all, which is what lets a house make one any
// size.
func (s *Scene) DrawWallWindow(window Rect) {
	const (
		brownC   = Brown8
		tanC     = Tan8
		dkstRedC = DkRed28
	)

	body := Inset(window, 3, 0)
	s.Back.Fill(body, brownC)
	s.Back.HiliteRect(body, tanC, dkstRedC)

	// Each sill is drawn twice: once a pixel narrow, then again full width and
	// two pixels shorter, so the two bevels cross and the sill reads as a
	// moulding rather than a slab.
	sill := window
	sill.Bottom = sill.Top + kWindowSillThick
	sill.Left++
	sill.Right--
	s.Back.Fill(sill, brownC)
	s.Back.HiliteRect(sill, tanC, dkstRedC)
	sill.Left--
	sill.Right++
	sill.Top += 2
	sill.Bottom -= 2
	s.Back.Fill(sill, brownC)
	s.Back.HiliteRect(sill, tanC, dkstRedC)

	// The bottom sill is lifted four pixels, so it sits above the window's
	// bottom edge and the surround shows below it.
	sill = window
	sill.Top = sill.Bottom - kWindowSillThick
	sill = Offset(sill, 0, -4)
	sill.Left++
	sill.Right--
	s.Back.Fill(sill, brownC)
	s.Back.HiliteRect(sill, tanC, dkstRedC)
	sill.Left--
	sill.Right++
	sill.Top += 2
	sill.Bottom -= 2
	s.Back.Fill(sill, brownC)
	s.Back.HiliteRect(sill, tanC, dkstRedC)

	inside := window
	inside.Left += 8
	inside.Right -= 8
	inside.Top += 11
	inside.Bottom -= 15
	s.Back.HiliteRect(inside, dkstRedC, tanC)

	halfWay := (inside.Top + inside.Bottom) / 2

	// The two panes overlap by five pixels at halfWay, which is where the
	// sashes meet.
	pane := inside
	pane.Bottom = halfWay + 2
	pane = Inset(pane, 5, 5)
	s.Back.HiliteRect(pane, dkstRedC, tanC)
	pane = Inset(pane, 1, 1)
	s.Back.Fill(pane, Sky8)

	pane = inside
	pane.Top = halfWay - 3
	pane = Inset(pane, 5, 5)
	s.Back.HiliteRect(pane, dkstRedC, tanC)
	pane = Inset(pane, 1, 1)
	s.Back.Fill(pane, Sky8)

	// The sash bar, drawn relative to the *bottom* pane after both insets, which
	// puts it exactly on the join.
	s.Back.Line(pane.Left-5, pane.Top-7, pane.Right+5, pane.Top-7, tanC)
}

// monthNames is STR# kMonthStringID (1005), "Months", index 1..12 -- the only
// resource-borne string the static room needs, and the only one the port has any
// reason to hold before the shell arrives at 1.7.
//
// It is transcribed rather than loaded, which is the same choice palette.go makes
// for 'clut' 128 and for the same two reasons: the data is twelve short words that
// have not changed since 1994, and a room's appearance should not depend on a
// second asset tree being present. TestMonthNamesMatchTheResource decodes the
// shipped STR# and fails if this drifts from it, so the transcription is checked
// rather than trusted. All caps is the resource's, not a shout.
var monthNames = [12]string{
	"JANUARY", "FEBRUARY", "MARCH", "APRIL", "MAY", "JUNE",
	"JULY", "AUGUST", "SEPTEMBER", "OCTOBER", "NOVEMBER", "DECEMBER",
}

// DrawCalendar draws the calendar picture, then the current month's name across
// it, centred in the picture's 64 pixels (ObjectDraw2.c:1132-1166).
//
// The month comes from Scene.Clock rather than time.Now, which is the port's one
// change and is what makes a composition containing a calendar reproducible; the
// original calls GetTime here, and cmd/glidergo sets Clock from time.Now once at
// startup, so the visible behaviour is the same.
//
// The type is not the original's. The C asks for the application font at nine
// point bold and the port has one hand-authored face (font.go, and
// docs/IMPROVEMENTS.md 2.29), whose seven-pixel capitals are in fact nearer nine
// point Geneva than the twelve point the scoreboard asks for. Centring is
// recomputed from this font's own StringWidth, so a narrower face moves the text
// right rather than leaving it hanging off the left edge: the widest month,
// SEPTEMBER, is 54 of the available 64 pixels.
//
// The 64 is the original's literal and the picture is 63 wide, so the text sits a
// pixel right of true centre. That is transcribed rather than corrected; see
// TestCalendarIsOnePixelNarrowerThanItsCentring.
func (s *Scene) DrawCalendar(theRect Rect) {
	art := s.A.Pict(kCalendarPictID)
	if art == nil {
		return
	}
	bounds := Offset(SetRect(0, 0, int16(art.W), int16(art.H)), theRect.Left, theRect.Top)
	s.Back.Copy(art, art.Bounds(), bounds, SrcCopy)

	// GetIndString is 1-based; time.Month is too.
	month := monthNames[(int(s.Clock.Month())-1)%12]
	s.Back.DrawString(theRect.Left+(64-StringWidth(month))/2, theRect.Top+55, month, DarkFlesh)
}

// DrawBulletin is the notice board: its picture at its own size, offset to the
// object's top-left. theRect's size is ignored, as in DrawPictObject.
func (s *Scene) DrawBulletin(theRect Rect) {
	art := s.A.Pict(kBulletinPictID)
	if art == nil {
		return
	}
	bounds := Offset(SetRect(0, 0, int16(art.W), int16(art.H)), theRect.Left, theRect.Top)
	s.Back.Copy(art, art.Bounds(), bounds, SrcCopy)
}

// ---------------------------------------------------------------------------
// The four generic PICT painters
// ---------------------------------------------------------------------------

// DrawPictObject is the fully opaque strategy: srcCopy, no mask, no colour key.
// The nine objects that use it are all things that fill their own rectangle --
// stairs, exterior doors and windows, the filing cabinet, Ozma.
//
// Note that the destination is srcRects[what] offset to theRect's top-left, not
// theRect itself. Every one of the nine gets its rect from srcRects anyway, so
// the two agree in practice; they would differ if a house could resize one, and
// the original's choice is the one reproduced.
func (s *Scene) DrawPictObject(what int16, theRect Rect) {
	art := s.A.Object(what)
	if art == nil {
		return
	}
	src := srcRects[what]
	s.Back.Copy(art, src, Offset(src, theRect.Left, theRect.Top), SrcCopy)
}

// DrawPictWithMaskObject is the cobweb and the cloud: art PICT plus a companion
// 1-bit mask PICT, built into a temporary GWorld pair and CopyMask'd. The
// extractor has already merged the mask into an alpha channel.
func (s *Scene) DrawPictWithMaskObject(what int16, theRect Rect) {
	s.maskObject(what, srcRects[what], theRect)
}

// DrawPictSansWhiteObject is the twenty objects with no mask PICT, drawn with
// `transparent` so their white background drops out. Since white is index 0 and
// the extractor keyed exactly index 0 to alpha, this is the same Masked copy.
func (s *Scene) DrawPictSansWhiteObject(what int16, theRect Rect) {
	s.maskObject(what, srcRects[what], theRect)
}

// DrawCustPictSansWhite is kCustomPict: a house's own picture, by id, keyed on
// white.
//
// The original builds a temp GWorld the size of *theRect* and then draws the
// picture into it at the picture's own size -- LoadGraphic uses picFrame, not the
// GWorld's bounds, so it neither scales nor tiles. When theRect is larger than
// the picture the surplus stays whatever NewGWorld left, which is white, and
// `transparent` then skips it; when theRect is smaller the picture is clipped.
// Taking the intersection of the two sizes as the source rect reproduces both
// cases exactly, and avoids materialising the temp surface.
func (s *Scene) DrawCustPictSansWhite(pictID int16, theRect Rect) {
	art := s.A.Pict(pictID)
	if art == nil {
		return
	}
	src := SetRect(0, 0,
		min16(int16(art.W), theRect.Wide()),
		min16(int16(art.H), theRect.Tall()))
	if Empty(src) {
		return
	}
	s.Back.Copy(art, src, Offset(src, theRect.Left, theRect.Top), Transparent)
}

// The two PICT ids the clutter painters name directly.
const (
	kBulletinPictID = 3966
	kCalendarPictID = 3970
)
