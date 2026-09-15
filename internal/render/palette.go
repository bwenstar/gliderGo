package render

import "image/color"

// The 256-entry indexed palette every offscreen surface in Glider PRO shares.
//
// The original asks the Palette Manager for 'clut' 128 (StructuresInit.c:
// kCustomPaletteID) and thereafter treats colour purely as an index: ColorRect,
// ColorLine and friends call Index2Color to turn a documented constant into an
// RGB, and CopyBits moves indices around unchanged. Reproducing the table is
// therefore not a cosmetic choice -- the arithmetic in DrawTable's dithered
// shadow ORs palette *indices* together, so a differently-ordered palette gives
// different pixels, not merely different colours.
//
// The table is generated rather than transcribed because it is generated: it is
// the 6x6x6 Macintosh colour cube in {FF,CC,99,66,33,00} order with the final
// all-zero corner omitted (215 entries, 0..214), then four ten-step ramps in
// {EE,DD,BB,AA,88,77,55,44,22,11} -- red, green, blue, grey -- and black last.
// TestPaletteMatchesClut checks it byte for byte against the shipped resource.
//
// Two consequences are load-bearing everywhere below:
//
//   - Index 0 is pure white. It is the "transparent" colour key of the 21
//     white-keyed object PICTs and the value a freshly created GWorld holds.
//   - The table is injective, so RGB -> index is exact and lossless. That is
//     what lets the extracted RGBA PNGs be turned back into index planes with
//     no quantisation and no ambiguity (see assets.go).
var Palette [256]color.RGBA

// paletteIndex inverts Palette. Keyed by r<<16|g<<8|b.
var paletteIndex map[uint32]uint8

func init() {
	levels := [6]uint8{0xFF, 0xCC, 0x99, 0x66, 0x33, 0x00}
	ramp := [10]uint8{0xEE, 0xDD, 0xBB, 0xAA, 0x88, 0x77, 0x55, 0x44, 0x22, 0x11}

	n := 0
	set := func(r, g, b uint8) {
		Palette[n] = color.RGBA{R: r, G: g, B: b, A: 0xFF}
		n++
	}
	for _, r := range levels {
		for _, g := range levels {
			for _, b := range levels {
				if r == 0 && g == 0 && b == 0 {
					continue // the cube's last corner is index 255 instead
				}
				set(r, g, b)
			}
		}
	}
	for _, v := range ramp {
		set(v, 0, 0)
	}
	for _, v := range ramp {
		set(0, v, 0)
	}
	for _, v := range ramp {
		set(0, 0, v)
	}
	for _, v := range ramp {
		set(v, v, v)
	}
	set(0, 0, 0)
	if n != 256 {
		panic("render: palette generation produced " + itoa(n) + " entries")
	}

	// The two derived tables. Both are built *here*, in the same init that fills
	// Palette, and that is a correctness requirement rather than tidiness.
	//
	// Go runs every package-level variable initializer before it runs any init
	// function, and it orders those initializers by the dependencies it can see in
	// the initializer *expressions*. Palette has no initializer expression -- it is
	// filled above -- so a `var x = f(Palette)` elsewhere in the package has no
	// dependency edge to this function, runs first, and reads 256 zero entries.
	//
	// bgrxLUT was exactly that, and the symptom was as quiet as it gets: every
	// entry came out 0xFF000000, so Surface.ToBGRX produced an opaque black image
	// for any input whatsoever, and it went unnoticed until the first frame of the
	// real game reached a real window. Nothing else in the package uses ToBGRX, and
	// black is what an uncomposed screen looks like too.
	paletteIndex = make(map[uint32]uint8, 256)
	for i, c := range Palette {
		paletteIndex[uint32(c.R)<<16|uint32(c.G)<<8|uint32(c.B)] = uint8(i)
		bgrxLUT[i] = uint32(c.B) | uint32(c.G)<<8 | uint32(c.R)<<16 | 0xFF000000
	}
}

// IndexOf returns the palette index of an exact RGB triple. Nothing in the
// extracted art is off-palette, so a miss is a bug in the extractor or a
// hand-edited asset, and the caller is expected to treat it as fatal rather
// than silently pick a near match.
func IndexOf(r, g, b uint8) (uint8, bool) {
	i, ok := paletteIndex[uint32(r)<<16|uint32(g)<<8|uint32(b)]
	return i, ok
}

// GoPalette exposes the table as an image/color.Palette for PNG encoding.
func GoPalette() color.Palette {
	p := make(color.Palette, 256)
	for i := range Palette {
		p[i] = Palette[i]
	}
	return p
}

// The named colour indices the draw code uses, spelled as the original spelled
// them (GliderPRO/Headers/GliderDefines.h). Only the ones the static room path
// actually reaches are listed; the rest are in docs/analysis/graphics-assets.md.
const (
	White8       = 0
	Yellow       = 5
	Gold         = 11
	RedOrange8   = 23
	Red8         = 35
	PaleViolet   = 42
	LtstGray3    = 43
	LtTan8       = 52
	Bamboo8      = 53
	DarkFlesh    = 58
	Orange8      = 59
	Tan8         = 94
	PissYellow8  = 95
	Pumpkin8     = 101
	Brown8       = 137
	Red48        = 143
	Sky8         = 150
	EarthBlue8   = 170
	DkGray38     = 172
	DkRed8       = 222
	DkRed28      = 223
	IntenseGreen = 225
	IntenseBlue  = 235
	LtstGray     = 245
	LtstGray2    = 246
	LtstGray4    = 247
	LtstGray5    = 248
	LtGray8      = 249
	Gray8        = 250
	Gray28       = 251
	DkGray8      = 252
	DkGray28     = 253
	DkstGray8    = 254
	Black8       = 255
)

// The classic 1-bit QuickDraw colours the game names, resolved into this palette.
//
// They are a different colour API from the block above, not more entries in it.
// `ForeColor(yellowColor)` takes one of the eight constants a 1961 Color QuickDraw
// grafport understood (`blackColor` 33, `whiteColor` 30, `redColor` 205,
// `yellowColor` 69, `cyanColor` 273, `blueColor` 409) and asks the current GDevice's
// colour table for the nearest entry it has; `ColorText(str, index)` takes a palette
// index directly. Glider PRO uses both -- the scoreboard indexes the palette (§5.5 of
// docs/analysis/scoring.md) and the high-score screen names ForeColor constants
// (§7.9.6) -- so a port that had only one of them would draw one of the two screens
// in the wrong colours.
//
// The values are what nearestIndex answers for the classic RGB of each constant,
// written down rather than computed so that a call site is a constant and a table
// lookup cannot change under it; TestTheQuickDrawColoursAreTheNearestPaletteEntries
// pins each one to its RGB. QDBlack and QDWhite are omitted deliberately: black and
// white are exactly on the palette and already have names above.
const (
	QDYellow = 5   // #FCF305 -> #FFFF00, which is Yellow above
	QDCyan   = 192 // #02ABEA -> #0099FF
	QDBlue   = 211 // #0000D4 -> #0000CC
	QDRed    = 216 // #DD0806 -> #DD0000, the banner's star count (Banner.c:159)
)

// itoa avoids pulling strconv into a package that is otherwise pure pixels.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
