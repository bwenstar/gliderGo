package render

// The frame-strip rect tables, checked against the sheet bounds they index into.
//
// srcrects.go's own comment has promised this test since 1.5c -- "TestStripRectsTileTheirSheet
// checks each array against its sheet's declared bounds, and a transcribed table is what that
// test can actually catch a mistake in" -- and until 1.5e it did not exist. The promise was
// the reason those tables were written out longhand instead of computed from a stride, so the
// absence made that choice pointless: a hand-typed table with nothing checking it is strictly
// worse than a loop.
//
// What can actually go wrong here, and why a golden image would not catch it:
//
//   - **A typo one row out.** DripSrc[4] written as Top 47 instead of 48 draws a drop with a
//     one-pixel band of the frame above it. Visible in motion, invisible in a still, and
//     invisible in a golden that only ever composes frame 0.
//   - **A rect past the end of its sheet.** Surface.Copy clips, so this does not crash; the
//     last frame of the strip just comes out short, or empty, and the object animates to
//     nothing. The `bands` strip is 18 pixels tall and BandRects[2] ends at exactly 18, so
//     there is no slack at all in the one table 1.5e added.
//   - **A table that does not cover its sheet.** Every one of the nine tiling strips is
//     exactly frames x stride tall, which is not a coincidence: the original allocated the
//     GWorld from the loop bound (StructuresInit.c:688-720). A table covering 7 of 8 frames
//     means a frame constant is wrong somewhere.
//
// The nine tiling tables get the strong assertion. The rest -- the sparkle's palindrome, the
// grease jar's two columns, the outlet's column inside the appliance sheet, the switch and
// light pairs -- are checked only for containment, because their geometry is genuinely
// irregular and asserting a shape on them would just be restating the table.

import (
	"testing"
)

// sheetSize resolves a name in either of the two bounds tables. The split between
// sheetBounds and stripBounds is about how the art is *reached* (srcRects index versus named
// global), not about its shape, so a geometry test has no reason to care which side a name
// falls on -- but it does have reason to fail loudly if a name is in neither, because that
// means a rect table indexes a surface Assets cannot produce.
func sheetSize(t *testing.T, name string) (w, h int16) {
	t.Helper()
	if wh, ok := sheetBounds[name]; ok {
		return int16(wh[0]), int16(wh[1])
	}
	if wh, ok := stripBounds[name]; ok {
		return int16(wh[0]), int16(wh[1])
	}
	t.Fatalf("%q is in neither sheetBounds nor stripBounds, so nothing can load it", name)
	return 0, 0
}

// ---------------------------------------------------------------------------
// The nine single-column strips
// ---------------------------------------------------------------------------

// TestStripRectsTileTheirSheet is the test srcrects.go names.
//
// A "tiling" strip is one column of equal cels from y=0 to the bottom of the sheet, which is
// what all nine of these are and what makes the strong form of the assertion available: the
// expected rect for frame i is computable from the sheet's height and the frame count alone,
// with no number taken from the table under test.
//
// That last clause is the whole design. `wantTop := i * (h / frames)` derives the answer from
// stripBounds, and stripBounds is transcribed from a different place in the original
// (StructuresInit.c's QSetRect calls) than the rect tables are (its loops). So the two have
// to agree about the frame count *and* the sheet height independently, and a single typo in
// either cannot satisfy both.
func TestStripRectsTileTheirSheet(t *testing.T) {
	cases := []struct {
		name  string // the rect table, for failure messages
		sheet string // the key in sheetBounds or stripBounds
		rects []Rect
	}{
		// The one 1.5e added. Three cels of 16x6 in an 18-pixel sheet: no slack.
		{"BandRects", "bands", BandRects[:]},

		{"BreadSrc", "toast", BreadSrc[:]},
		{"DripSrc", "drip", DripSrc[:]},
		{"BalloonSrc", "balloon", BalloonSrc[:]},
		{"CopterSrc", "copter", CopterSrc[:]},
		{"DartSrc", "dart", DartSrc[:]},
		{"BallSrc", "ball", BallSrc[:]},
		{"FishSrc", "fish", FishSrc[:]},
		{"PointsSrc", "points", PointsSrc[:]},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wide, tall := sheetSize(t, c.sheet)
			frames := int16(len(c.rects))

			if tall%frames != 0 {
				t.Fatalf("%s sheet %q is %d tall, which %d frames do not divide: "+
					"either the frame count or the sheet height is wrong",
					c.name, c.sheet, tall, frames)
			}
			stride := tall / frames

			for i, got := range c.rects {
				want := Rect{
					Top:    int16(i) * stride,
					Left:   0,
					Bottom: int16(i+1) * stride,
					Right:  wide,
				}
				if got != want {
					t.Errorf("%s[%d] = %+v, want %+v (cel %d of %d, stride %d in a %dx%d sheet)",
						c.name, i, got, want, i, frames, stride, wide, tall)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Everything else: containment only
// ---------------------------------------------------------------------------

// TestEverySrcRectFitsItsSheet is the weak assertion, applied to the tables whose geometry is
// irregular on purpose.
//
// It cannot catch a frame one row out, but it does catch the failure that a golden image
// hides: Surface.Copy clips silently, so a rect past the end of a sheet produces a short or
// empty blit rather than a panic. An object that animates to nothing looks like a logic bug
// in the handler and would be looked for in entirely the wrong file.
//
// The grease pair is the reason this test exists in 1.5e rather than later. greaseSrcLf[3]
// ends at (64, 378) and the `bonus` sheet is 88x378 -- the jar's last cel is flush against
// the bottom edge of the largest sheet in the game, and grease.go's four-cel filmstrip walk
// reads all four unconditionally.
func TestEverySrcRectFitsItsSheet(t *testing.T) {
	cases := []struct {
		name  string
		sheet string
		rects []Rect
	}{
		{"greaseSrcRt", "bonus", greaseSrcRt[:]},
		{"greaseSrcLf", "bonus", greaseSrcLf[:]},
		{"SparkleSrc", "bonus", SparkleSrc[:]},
		{"OutletSrc", "appliance", OutletSrc[:]},

		{"lightSwitchSrc", "switch", lightSwitchSrc[:]},
		{"machineSwitchSrc", "switch", machineSwitchSrc[:]},
		{"thermostatSrc", "switch", thermostatSrc[:]},
		{"powerSrc", "switch", powerSrc[:]},
		{"knifeSwitchSrc", "switch", knifeSwitchSrc[:]},

		{"trackLightSrc", "light", trackLightSrc[:]},
		{"flourescentSrc", "light", []Rect{flourescentSrc1, flourescentSrc2}},

		{"applianceOverlays", "appliance", []Rect{
			PlusScreen1, PlusScreen2, TVScreen1, TVScreen2,
			CoffeeLight1, CoffeeLight2, VCRTime1, VCRTime2,
			StereoLight1, StereoLight2, MicroOff, MicroOn,
		}},

		// The tiling nine again. Containment is implied by the exact test above, but
		// listing them here means a future table added to only one of the two lists is
		// still covered by something.
		{"BandRects", "bands", BandRects[:]},
		{"BreadSrc", "toast", BreadSrc[:]},
		{"DripSrc", "drip", DripSrc[:]},
		{"BalloonSrc", "balloon", BalloonSrc[:]},
		{"CopterSrc", "copter", CopterSrc[:]},
		{"DartSrc", "dart", DartSrc[:]},
		{"BallSrc", "ball", BallSrc[:]},
		{"FishSrc", "fish", FishSrc[:]},
		{"PointsSrc", "points", PointsSrc[:]},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wide, tall := sheetSize(t, c.sheet)
			for i, r := range c.rects {
				switch {
				case r.Left < 0 || r.Top < 0:
					t.Errorf("%s[%d] = %+v starts before the sheet origin", c.name, i, r)
				case r.Right > wide || r.Bottom > tall:
					t.Errorf("%s[%d] = %+v runs past the %dx%d %q sheet; Copy would clip "+
						"it silently", c.name, i, r, wide, tall, c.sheet)
				case r.Wide() <= 0 || r.Tall() <= 0:
					t.Errorf("%s[%d] = %+v is empty, so it draws nothing", c.name, i, r)
				}
			}
		})
	}
}
