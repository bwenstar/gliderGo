package render

import "github.com/bwenstar/gliderGo/internal/house"

// GetObjectRect, from GliderPRO/Sources/ObjectRects.c:32-273.
//
// Every object's on-screen rectangle comes from here, and where it comes from
// varies by group in ways that are not guessable: some objects carry an explicit
// bounds rect in their ten data bytes, some carry only a top-left corner and take
// their size from srcRects, and four compute their size from a packed field.
//
// obj is a pointer because the original's is, and the mutation matters: a
// kCustomPict whose picture is missing has its id rewritten to 10000 in place,
// and the caller then draws the 72x34 placeholder rather than nothing. In the
// original that write lands on ObjectDrawAll's local copy of the object and so
// never reaches the house file, which is why passing a copy here is correct.
//
// There is no default case. An undefined `what` leaves the caller's rect
// untouched -- whatever it held from the previous iteration -- and that is
// reproduced rather than corrected: itsRect is an out parameter here too.
func (a *Assets) GetObjectRect(obj *house.Object, itsRect *Rect) {
	switch obj.What {
	case house.ObjectIsEmpty:
		*itsRect = SetRect(0, 0, 0, 0)

	case kFloorVent, kCeilingVent, kFloorBlower, kCeilingBlower, kSewerGrate,
		kLeftFan, kRightFan, kTaper, kCandle, kStubby, kTiki, kBBQ,
		kInvisBlower, kGrecoVent, kSewerBlower:
		d := obj.Blower()
		*itsRect = ZeroCorner(srcRects[obj.What])
		*itsRect = Offset(*itsRect, d.TopLeft.H, d.TopLeft.V)

	case kLiftArea:
		// The one blower whose size is data, not art: distance across and
		// tall*2 down. Note that data.a.tall is a byte, so the tallest lift
		// area a house can hold is 510 pixels.
		d := obj.Blower()
		*itsRect = SetRect(0, 0, d.Distance, int16(d.Tall)*2)
		*itsRect = Offset(*itsRect, d.TopLeft.H, d.TopLeft.V)

		// ObjectRects.c:68-71 has four more statements here, after the break
		// and before the next case label: a copy of the blower-group body,
		// unreachable in the original and unreachable here. Left as this note
		// rather than as dead Go, which the compiler would reject.

	case kTable, kShelf, kCabinet, kFilingCabinet, kWasteBasket, kMilkCrate,
		kCounter, kDresser, kStool, kTrunk, kDeckTable, kInvisObstacle,
		kManhole, kBooks, kInvisBounce:
		// The furniture group is the only one that stores a full rect, which is
		// what lets a house resize a counter or a shelf.
		*itsRect = obj.Furniture().Bounds

	case kRedClock, kBlueClock, kYellowClock, kCuckoo, kPaper, kBattery, kBands,
		kGreaseRt, kGreaseLf, kFoil, kInvisBonus, kStar, kSparkle, kHelium:
		d := obj.Bonus()
		*itsRect = ZeroCorner(srcRects[obj.What])
		*itsRect = Offset(*itsRect, d.TopLeft.H, d.TopLeft.V)

	case kSlider:
		d := obj.Bonus()
		*itsRect = ZeroCorner(srcRects[obj.What])
		*itsRect = Offset(*itsRect, d.TopLeft.H, d.TopLeft.V)
		itsRect.Right = itsRect.Left + d.Length

	case kUpStairs, kDownStairs, kMailboxLf, kMailboxRt, kFloorTrans,
		kCeilingTrans, kDoorInLf, kDoorInRt, kDoorExRt, kDoorExLf,
		kWindowInLf, kWindowInRt, kWindowExRt, kWindowExLf:
		d := obj.Transport()
		*itsRect = ZeroCorner(srcRects[obj.What])
		*itsRect = Offset(*itsRect, d.TopLeft.H, d.TopLeft.V)

	case kInvisTrans:
		d := obj.Transport()
		*itsRect = ZeroCorner(srcRects[obj.What])
		*itsRect = Offset(*itsRect, d.TopLeft.H, d.TopLeft.V)
		itsRect.Bottom = itsRect.Top + d.Tall
		itsRect.Right += int16(d.Wide)

	case kDeluxeTrans:
		// Both dimensions packed into data.d.tall's two bytes and scaled by 4,
		// so a deluxe transport is always a multiple of four pixels on a side.
		d := obj.Transport()
		wide := int16(uint16(d.Tall) >> 8)
		tall := int16(uint16(d.Tall) & 0x00FF)
		*itsRect = SetRect(0, 0, wide*4, tall*4)
		*itsRect = Offset(*itsRect, d.TopLeft.H, d.TopLeft.V)

	case kLightSwitch, kMachineSwitch, kThermostat, kPowerSwitch, kKnifeSwitch,
		kInvisSwitch, kTrigger, kLgTrigger, kSoundTrigger:
		d := obj.Switch()
		*itsRect = ZeroCorner(srcRects[obj.What])
		*itsRect = Offset(*itsRect, d.TopLeft.H, d.TopLeft.V)

	case kCeilingLight, kLightBulb, kTableLamp, kHipLamp, kDecoLamp, kInvisLight:
		d := obj.Light()
		*itsRect = ZeroCorner(srcRects[obj.What])
		*itsRect = Offset(*itsRect, d.TopLeft.H, d.TopLeft.V)

	case kFlourescent, kTrackLight:
		// right = length, assigned *before* the offset, so data.f.length is a
		// width and not an absolute edge. The two stretchable light fixtures are
		// the only objects that work this way.
		d := obj.Light()
		*itsRect = ZeroCorner(srcRects[obj.What])
		itsRect.Right = d.Length
		*itsRect = Offset(*itsRect, d.TopLeft.H, d.TopLeft.V)

	case kShredder, kToaster, kMacPlus, kGuitar, kTV, kCoffee, kOutlet, kVCR,
		kStereo, kMicrowave, kCinderBlock, kFlowerBox, kCDs:
		d := obj.Appliance()
		*itsRect = ZeroCorner(srcRects[obj.What])
		*itsRect = Offset(*itsRect, d.TopLeft.H, d.TopLeft.V)

	case kCustomPict:
		// The size is the picture's own size, so this is the one object whose
		// rect cannot be known without loading art. A missing picture is not an
		// error: the id is rewritten to 10000, whose art is exactly the 72x34
		// srcRects[kCustomPict] fallback used here.
		d := obj.Appliance()
		if frame, ok := a.PictFrame(d.Height); ok {
			*itsRect = frame
		} else {
			d.Height = 10000
			obj.SetAppliance(d)
			*itsRect = srcRects[obj.What]
		}
		*itsRect = ZeroCorner(*itsRect)
		*itsRect = Offset(*itsRect, d.TopLeft.H, d.TopLeft.V)

	case kBalloon, kCopterLf, kCopterRt, kDartLf, kDartRt, kBall, kDrip, kFish,
		kCobweb:
		d := obj.Enemy()
		*itsRect = ZeroCorner(srcRects[obj.What])
		*itsRect = Offset(*itsRect, d.TopLeft.H, d.TopLeft.V)

	case kOzma, kMirror, kMousehole, kFireplace, kFlower, kWallWindow, kBear,
		kCalendar, kVase1, kVase2, kBulletin, kCloud, kFaucet, kRug, kChimes:
		*itsRect = obj.Clutter().Bounds
	}
}
