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

	paletteIndex = make(map[uint32]uint8, 256)
	for i, c := range Palette {
		paletteIndex[uint32(c.R)<<16|uint32(c.G)<<8|uint32(c.B)] = uint8(i)
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
