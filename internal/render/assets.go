package render

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sync"

	"glidergo/internal/house"
)

// Assets is the extracted art tree, loaded on demand into indexed Surfaces.
//
// The tree is what tools/extract_art.py writes: PICT resources decoded to RGBA
// PNGs with the original's masking already resolved into an alpha channel. This
// loader undoes the RGBA step, because the game does not composite in RGB (see
// palette.go): every pixel goes back to the palette index it came from, and the
// alpha channel becomes the separate 1-bit mask GWorld that CopyMask expects.
// The palette is injective, so that round trip is lossless.
//
// Loading is lazy and cached, which is also what the original does -- every
// DrawPictObject call is a GetPicture/DrawPicture/ReleaseResource cycle, and the
// sheets are the only art held open. Errors are sticky rather than returned: the
// draw helpers are void, exactly as the original's are, and a missing resource
// there ends in RedAlert. Err() is the equivalent, to be checked once after a
// composition rather than at every blit.
type Assets struct {
	root string

	mu    sync.Mutex
	cache map[string]*Surface
	err   error

	// houseDir is the open house's extracted resource fork, or "" for none.
	// Its PICTs shadow the application's; see housepict.go.
	houseDir string

	// approx counts pixels that had to be colour-matched instead of mapped
	// exactly. Only the 24 house pictures with their own ColorTable can raise
	// it.
	approx int
}

// The 14 sheets, with the GWorld bounds their NewGWorld call asks for.
//
// The bounds are not always the art's size, and that difference is load-bearing
// in one case: the furniture PICT is 64x221 inside a 64x278 GWorld, so rows
// 221..277 hold whatever NewGWorld left there -- white, and transparent in the
// companion mask, since the mask PICT is short too. No named sub-rect reads
// those rows, but the surface has to be the full height or every rect below
// tableSrc would be out of bounds.
//
// Sizes from the QSetRect calls at StructuresInit.c:253,301,350,431,455,501,538,
// 623,632,641,650,659,668 and StructuresInit2.c:63.
var sheetBounds = map[string][2]int{
	"appliance": {80, 269},
	"ball":      {32, 64},
	"balloon":   {24, 240},
	"blower":    {48, 402},
	"bonus":     {88, 378},
	"clutter":   {128, 69},
	"copter":    {32, 300},
	"dart":      {64, 76},
	"drip":      {16, 72},
	"enemy":     {36, 33},
	"furniture": {64, 278},
	"light":     {72, 126},
	"switch":    {32, 104}, // the only sheet with no mask PICT: 5003 does not exist
	"trans":     {56, 32},
}

// The 14 strips: art that no srcRects entry indexes, reached instead through the
// original's named globals. Only `supp` matters to a static room -- it is the
// 512x44 floor-support band -- but the table is complete so that the later
// stages have one place to look. The four glider sheets share mask 4999.
var stripBounds = map[string][2]int{
	"angel":       {96, 44},
	"badge":       {32, 66},
	"bands":       {16, 18},
	"board":       {1536, 20},
	"fish":        {16, 128},
	"glider":      {48, 668},
	"glider2":     {48, 668},
	"gliderFoil":  {48, 668},
	"gliderFoil2": {48, 668},
	"points":      {24, 120},
	"shadow":      {48, 18},
	"shred":       {40, 35},
	"supp":        {512, 44},
	"toast":       {32, 174},
}

// miscNames are the resources that are neither sheet, strip, object nor
// background. There is exactly one, and its file name carries the description
// the extractor gave it.
var miscNames = map[int16]string{3957: "manhole_thru_floor"}

// kFallbackBackground is the PICT LoadGraphicSpecial drops to when a room names
// a background that is not there (RoomGraphics.c:145). Houses can reference
// their own background PICTs, and a house whose resource fork we have not
// extracted lands here rather than failing.
const kFallbackBackground = 2000

// NewAssets returns a loader rooted at an extracted-art directory, typically
// assets/extracted/art. Nothing is read until something is asked for.
func NewAssets(root string) *Assets {
	return &Assets{root: root, cache: make(map[string]*Surface)}
}

// Err returns the first load failure, if any. It stays set once set.
func (a *Assets) Err() error { return a.err }

// fail records the first error and returns nil, so callers can treat a missing
// resource as "draw nothing" and let the caller of the whole composition decide
// whether that is fatal.
func (a *Assets) fail(err error) *Surface {
	if a.err == nil {
		a.err = err
	}
	return nil
}

// Sheet returns one of the 14 shared sheet GWorlds, allocated at its GWorld
// bounds with the art pasted at (0,0).
func (a *Assets) Sheet(name string) *Surface {
	wh, ok := sheetBounds[name]
	if !ok {
		return a.fail(fmt.Errorf("render: no such sheet %q", name))
	}
	return a.padded("sheet/"+name+".png", wh[0], wh[1])
}

// Strip returns one of the 14 animation or band strips.
func (a *Assets) Strip(name string) *Surface {
	wh, ok := stripBounds[name]
	if !ok {
		return a.fail(fmt.Errorf("render: no such strip %q", name))
	}
	return a.padded("strip/"+name+".png", wh[0], wh[1])
}

// Object returns the pre-cropped art for an object drawn from its own PICT --
// the artKey, artOpaque and artPair kinds. The artSheet objects deliberately do
// not come through here: their pixels live in a shared sheet and the draw code
// blits srcRects[what] straight out of it, as the original does.
func (a *Assets) Object(what int16) *Surface {
	name := house.ObjectName(what)
	if name == "" {
		return a.fail(fmt.Errorf("render: object art requested for undefined what 0x%02X", what))
	}
	return a.load(fmt.Sprintf("object/%02X_%s.png", what, name))
}

// Flower returns one of the six kFlower frames. kFlower is the one object with
// no srcRects entry: flowerSrc[data.i.pict] is both its art and its size, so the
// six frames differ in size and each is its own file.
func (a *Assets) Flower(i int) *Surface {
	if i < 0 || i > 5 {
		return a.fail(fmt.Errorf("render: flower frame %d out of range", i))
	}
	return a.load(fmt.Sprintf("object/85_kFlower_%d.png", i))
}

// Background returns a 512x322 room background, following
// LoadGraphicSpecial's fallback to PICT 2000 when the named one is absent
// (RoomGraphics.c:134-152).
//
// Resolution goes through Pict, so the open house's own fork is consulted first.
// That is not a nicety: rooms name backgrounds by id, houses may define their own
// above kUserBackground, and Fun House redefines two of the eighteen built-in
// ones (PICT 2014 and 2015) outright.
func (a *Assets) Background(pictID int16) *Surface {
	if s := a.Pict(pictID); s != nil {
		return s
	}
	if pictID == kFallbackBackground {
		return a.fail(fmt.Errorf("render: background PICT %d is missing and it is the fallback", pictID))
	}
	if s := a.Pict(kFallbackBackground); s != nil {
		return s
	}
	return a.fail(fmt.Errorf("render: background PICT %d is missing and so is the fallback %d",
		pictID, kFallbackBackground))
}

// UI returns one of the shell's PICT plates: the splash (1000), the two pause
// overlays (1015, 1016), the stars-remaining panels (1017, 1018), the banner
// sheet (1991-1993), the high-score header and starfield (1994, 1995, 1998), the
// About pictures (150, 151, 153) and the Load House furniture (1001-1004).
// tools/extract_art.py's UI list is the authority for which ids exist.
//
// **A missing plate is not an error here**, and that is the difference between
// this and every other accessor in the file. The room path cannot draw a room
// without its art, so a missing sheet is a bug and Err reports it; the shell can
// always fall back to drawing its own panel, and a first run against a tree with
// no extracted assets should reach a title screen that says so rather than a
// fatal error (docs/IMPROVEMENTS.md 2.6). A plate that is present but will not
// decode still records: that is a broken extraction rather than an absent one,
// and silence would hide it.
//
// The open house's fork is deliberately *not* consulted, unlike Pict. On a Mac it
// would be -- the house's resources sit in front of the application's for every
// id -- and the shipped houses use that freely: thirteen of the twenty carry their
// own 1991-1993 banner sheet, four their own 1017 or 1018, and Teddy World its own
// 1015 and 1016. Every one of those is drawn *over a running game*, where the
// house is open and where its art is the right answer, so they are asked for
// through Plate instead. The shell's own chrome is the application's, because a
// house that could repaint the title screen is a house that could hide the way out
// of it.
func (a *Assets) UI(pictID int16) *Surface {
	rel := fmt.Sprintf("ui/%d.png", pictID)
	if _, err := os.Stat(filepath.Join(a.root, rel)); err != nil {
		return nil
	}
	return a.load(rel)
}

// Misc returns a resource that is none of the above; only PICT 3957, the manhole
// seen through a floor support, qualifies.
func (a *Assets) Misc(pictID int16) *Surface {
	name, ok := miscNames[pictID]
	if !ok {
		return a.fail(fmt.Errorf("render: no misc resource %d", pictID))
	}
	return a.load(fmt.Sprintf("misc/%d_%s.png", pictID, name))
}

// padded loads a PNG into a surface of the given GWorld bounds, pasting the art
// at (0,0). Where the art is smaller the remainder stays white and transparent,
// which is what an unwritten GWorld and its unwritten mask hold.
func (a *Assets) padded(rel string, w, h int) *Surface {
	key := fmt.Sprintf("%s@%dx%d", rel, w, h)
	a.mu.Lock()
	if s, ok := a.cache[key]; ok {
		a.mu.Unlock()
		return s
	}
	a.mu.Unlock()

	art := a.load(rel)
	if art == nil {
		return nil
	}
	if art.W > w || art.H > h {
		return a.fail(fmt.Errorf("render: %s is %dx%d, larger than its %dx%d GWorld", rel, art.W, art.H, w, h))
	}
	s := art
	if art.W != w || art.H != h {
		s = newMasked(w, h)
		s.Copy(art, art.Bounds(), art.Bounds(), Masked)
	}

	a.mu.Lock()
	a.cache[key] = s
	a.mu.Unlock()
	return s
}

// load decodes one PNG to an indexed surface, cached by path.
func (a *Assets) load(rel string) *Surface {
	a.mu.Lock()
	if s, ok := a.cache[rel]; ok {
		a.mu.Unlock()
		return s
	}
	a.mu.Unlock()

	f, err := os.Open(filepath.Join(a.root, rel))
	if err != nil {
		return a.fail(fmt.Errorf("render: %w", err))
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return a.fail(fmt.Errorf("render: decoding %s: %w", rel, err))
	}
	s, err := surfaceFromImage(img)
	if err != nil {
		return a.fail(fmt.Errorf("render: %s: %w", rel, err))
	}

	a.mu.Lock()
	a.cache[rel] = s
	a.mu.Unlock()
	return s
}

// newMasked allocates a surface that is white everywhere and transparent
// everywhere: a fresh art GWorld beside a fresh mask GWorld. Sheets need this
// because CopyMask reads a real mask; the destination surfaces do not, and
// NewSurface leaves their Mask nil, which means opaque.
func newMasked(w, h int) *Surface {
	s := NewSurface(w, h)
	s.Mask = make([]uint8, w*h)
	return s
}

// surfaceFromImage inverts the extractor's index-plane-to-RGBA step.
//
// Alpha must be 0 or 255: it came from a 1-bit mask PICT or from a white colour
// key, and nothing in the pipeline can produce a partial value. Colour must be
// exactly on the palette, for the same reason. Both are hard errors, because a
// near miss would mean the art no longer matches the palette the game ORs
// indices against, and silently snapping to the nearest entry would hide that.
func surfaceFromImage(img image.Image) (*Surface, error) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	s := newMasked(w, h)

	// image/png decodes 8-bit RGBA to NRGBA, which is straight alpha and so
	// needs no un-premultiplying. Anything else goes the slow way.
	nrgba, _ := img.(*image.NRGBA)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var r, g, bl, al uint8
			if nrgba != nil {
				i := nrgba.PixOffset(b.Min.X+x, b.Min.Y+y)
				r, g, bl, al = nrgba.Pix[i], nrgba.Pix[i+1], nrgba.Pix[i+2], nrgba.Pix[i+3]
			} else {
				r16, g16, b16, a16 := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
				if a16 != 0 && a16 != 0xFFFF {
					return nil, fmt.Errorf("partial alpha %d at (%d,%d)", a16>>8, x, y)
				}
				r, g, bl, al = uint8(r16>>8), uint8(g16>>8), uint8(b16>>8), uint8(a16>>8)
			}
			switch al {
			case 0:
				// Transparent: leave it white and masked out. The stored RGB is
				// meaningless here, and for the colour-keyed PICTs it is white
				// anyway.
				continue
			case 0xFF:
			default:
				return nil, fmt.Errorf("partial alpha %d at (%d,%d)", al, x, y)
			}
			idx, ok := IndexOf(r, g, bl)
			if !ok {
				return nil, fmt.Errorf("off-palette colour #%02X%02X%02X at (%d,%d)", r, g, bl, x, y)
			}
			s.Pix[y*w+x] = idx
			s.Mask[y*w+x] = 0xFF
		}
	}
	return s, nil
}
