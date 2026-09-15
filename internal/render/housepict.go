package render

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"glidergo/internal/house"
)

// GetPicture by numeric id, across both resource chains.
//
// The original opens a house's resource fork when the house is opened and leaves
// it open for the whole session (HouseIO.c's OpenHouseResFork; only CloseHouse
// closes it). A house's fork therefore sits *in front of* the application's on
// the resource chain, so GetPicture(id) finds the house's PICT id before the
// application's for every id, not just the ids above kUserBackground.
//
// That is not a technicality. Metropolis carries its own PICT 1999, which is the
// horizontal floor-support beam drawn between storeys; Fun House carries its own
// 2014 and 2015, which are two of the eighteen built-in room backgrounds. Both
// replace application art, and a loader that only consulted the fork for ids
// >= 3000 would draw the wrong pixels in those two houses and nowhere else --
// the kind of difference that is invisible until someone plays that house.
//
// Colour is exact wherever it can be. The house PNGs are RGB with no alpha,
// because whether a house PICT is a background (srcCopy) or a kCustomPict
// (`transparent`, white keyed out) is decided by the room data rather than by
// the resource, so the extractor cannot bake an alpha channel without guessing;
// the transfer mode is applied at the call site instead, which is where the
// original applies it too. Going back from RGB to palette indices is lossless
// for all but 24 of the 919 shipped house pictures, which carry their own
// ColorTable with colours off 'clut' 128. Those the original hands to QuickDraw,
// which colour-matches them into the 8-bit destination through a Color Manager
// inverse table we do not have; nearestIndex stands in for it and every pixel it
// touches is counted, so the approximation is reported rather than silent.

// OpenHouseResFork puts a house's own PICT resources in front of the
// application's, for as long as that house is open. dir is the house's directory
// under assets/extracted/houseart -- houseart/<house>, holding pict/<id>.png.
//
// An empty dir, or a house with no extracted fork, is not an error: houses may
// legitimately carry no PICTs of their own, and every lookup then falls straight
// through to the application art.
func (a *Assets) OpenHouseResFork(dir string) {
	a.mu.Lock()
	a.houseDir = dir
	a.mu.Unlock()
}

// CloseHouseResFork drops the house's fork off the chain again.
func (a *Assets) CloseHouseResFork() { a.OpenHouseResFork("") }

// Approximations is how many pixels have been colour-matched rather than
// mapped exactly, across every house picture loaded so far. It is zero for 895
// of the 919 shipped house pictures and for all 152 application pictures.
func (a *Assets) Approximations() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.approx
}

// Pict is GetPicture(id) followed by a decode: the house's fork first, then the
// application's art.
//
// It returns nil for an id that is in neither, *without* recording an error,
// because a nil PicHandle is a handled outcome in the two places a room's data
// can name a picture that is not there: LoadGraphicSpecial falls back to PICT
// 2000 (RoomGraphics.c:145) and GetObjectRect rewrites a kCustomPict's id to
// 10000 (ObjectRects.c:216). Every other caller in the original would RedAlert,
// and the callers here record the error themselves.
func (a *Assets) Pict(id int16) *Surface {
	a.mu.Lock()
	dir := a.houseDir
	a.mu.Unlock()

	if dir != "" {
		p := filepath.Join(dir, "pict", fmt.Sprintf("%d.png", id))
		if _, err := os.Stat(p); err == nil {
			return a.loadHousePict(p)
		}
	}
	rel, ok := appPictPath(id)
	if !ok {
		return nil
	}
	return a.load(rel)
}

// Plate is the art for one of the pictures drawn *over* a running game: the two pause
// overlays (1015, 1016), the two stars-remaining panels (1017, 1018) and the three
// banner plates (1991-1993).
//
// It is Pict's resolution order -- the open house's fork first, then the application's
// art -- with UI's answer to a missing file, which is nil and no recorded error. Both
// halves of that are deliberate:
//
//   - **The house comes first**, because on a Mac it does: these are drawn while a house
//     is open, so its resources sit in front of the application's for the same id, and
//     the shipped houses use that. Teddy World carries its own 1015 and 1016, four
//     houses their own 1017 or 1018, and thirteen of the twenty their own banner sheet.
//     The shell's chrome goes through UI instead and is never a house's to redefine --
//     see UI.
//   - **A missing plate is not an error**, because the game must still be playable
//     against a checkout with no extracted art. The overlay draws its own panel
//     instead (internal/game/pause.go), which is docs/IMPROVEMENTS.md 2.6 again: the
//     absence of a decoration is not the absence of a game.
func (a *Assets) Plate(id int16) *Surface {
	a.mu.Lock()
	dir := a.houseDir
	a.mu.Unlock()

	if dir != "" {
		p := filepath.Join(dir, "pict", fmt.Sprintf("%d.png", id))
		if _, err := os.Stat(p); err == nil {
			return a.loadHousePict(p)
		}
	}
	return a.UI(id)
}

// uiMaskPairs is the whole of the "one plate is another plate's mask" convention
// outside the +1000 sheet family: art id -> mask id.
//
// It is a table and not a formula because the original's numbering is not one rule
// but four (docs/analysis/resource-fork.md 683): sheets are art+1000, the angel is
// art+1, these two are art-1, and the high-score plaque is art+4. Deriving any of
// them by arithmetic would silently pair the wrong pictures the first time a fifth
// convention turned up.
//
//	1990/1989  the four game-over pages, 32x448  (1.7d)
//	1992/1991  the banner page's bottom half, 330x30  (1.7d)
//	1994/1998  the high-score plaque, 332x30  (1.7c)
//
// Measured, over the shipped art, with "the mask's non-white pixels" against "the
// art's non-white pixels" -- the same cross-tab docs/analysis/graphics-assets.md 5.2
// runs on the sheet pairs:
//
//	pair       pixels  opaque  agree  mask-opaque & art white  mask-clear & art ink
//	1990/1989  14,336   9,356  9,861                    4,475                     0
//	1992/1991   9,900   9,340  4,500                    5,400                     0
//	1994/1998   9,960   3,409  9,960                        0                     0
//
// So the mask is authoritative for two of the three and a white colour key would
// punch 4,475 and 5,400 holes in them; for the plaque the two agree exactly, which
// TestTheThreeShippedMaskPairsAreMeasured pins as a fact about that one piece of art
// rather than as a licence to key on white.
var uiMaskPairs = map[int16]int16{
	1990: 1989,
	1992: 1991,
	1994: 1998,
}

// MaskedPlate is Plate for the three pictures whose companion plate is their 1-bit
// mask: the art with that mask applied, ready for one Masked copy.
//
// It exists because the extractor cannot do this. The mask is a separate resource
// with its own id, so a house may override the art and not the mask or the mask and
// not the art -- thirteen of the twenty shipped houses carry their own 1991-1993, and
// on a Mac such a house's 1992 pairs with the *application's* 1991. Baking the pair
// together at extraction time would lose that; resolving each id through Plate and
// combining here reproduces it, which is the same argument Plate itself makes for
// consulting the fork first.
//
// Opaque where the mask plate's pixel is not white. That is the polarity
// docs/analysis/graphics-assets.md 5.2 proves for the sheet masks -- a set bit in the
// 1-bit PICT means "copy this pixel" -- and the extractor writes those set bits as
// palette index 1 rather than as black, so the test is "not white" rather than a
// specific index. A house's own mask, which arrives through nearestIndex, is covered
// by the same test.
//
// Two absences are handled rather than reported, both the way the rest of this file
// handles them:
//
//   - **No mask plate**: the art is returned unchanged, so a checkout with a partial
//     extraction draws an opaque rectangle instead of nothing.
//   - **A mask smaller than the art**: the uncovered pixels stay transparent, which is
//     what CopyMask reads out of the unwritten part of a fresh 1-bit GWorld -- the same
//     reasoning padded gives for the sheet that is one row short.
//
// An id with no entry in uiMaskPairs is just Plate, so a call site does not have to
// know which of the plates it draws has a mask.
func (a *Assets) MaskedPlate(id int16) *Surface {
	art := a.Plate(id)
	maskID, paired := uiMaskPairs[id]
	if art == nil || !paired {
		return art
	}

	// Keyed on the house directory as well as the id, because that is the whole of
	// what the two Plate lookups depend on: the same id resolves to different art
	// with a different house open.
	a.mu.Lock()
	key := fmt.Sprintf("masked:%d@%s", id, a.houseDir)
	if s, ok := a.cache[key]; ok {
		a.mu.Unlock()
		return s
	}
	a.mu.Unlock()

	mask := a.Plate(maskID)
	if mask == nil {
		return art
	}

	s := art.Clone()
	s.Mask = make([]uint8, s.W*s.H)
	for y := 0; y < s.H && y < mask.H; y++ {
		for x := 0; x < s.W && x < mask.W; x++ {
			if mask.Pix[y*mask.W+x] != White8 {
				s.Mask[y*s.W+x] = 0xFF
			}
		}
	}

	a.mu.Lock()
	a.cache[key] = s
	a.mu.Unlock()
	return s
}

// Bnds is GetResource('bnds', id): a house background's own opening flags.
//
// The resource is eight bytes laid out as a Rect and used as four independent
// booleans -- a non-zero field means that side is open. It is never read as a
// rectangle, which is why the four values in the shipped houses are 0 and 1 rather
// than coordinates. GetOriginalBounding packs it into the same 1/2/4/8 bit code the
// room's own `bounds` field carries.
//
// 'bnds' exists only in house forks, never in the application's -- the eighteen
// built-in backgrounds are handled by name in the switches that would otherwise ask
// for one. So this is the one lookup with no fallback to application art: a missing
// resource means the author never saved openings for that background, and
// GetOriginalBounding's answer is then 0, meaning closed on all four sides.
func (a *Assets) Bnds(id int16) (Rect, bool) {
	a.mu.Lock()
	dir := a.houseDir
	a.mu.Unlock()
	if dir == "" {
		return Rect{}, false
	}
	raw, err := os.ReadFile(filepath.Join(dir, "bnds", fmt.Sprintf("%d.bin", id)))
	if err != nil || len(raw) < 8 {
		// Not an error. The original's GetResource returns nil here and
		// GetOriginalBounding raises a yellow alert only if a PICT of the same id
		// does exist -- i.e. only when the author drew a background and forgot its
		// bounds. That alert is advisory and is not reproduced.
		return Rect{}, false
	}
	be := func(i int) int16 { return int16(raw[i])<<8 | int16(raw[i+1]) }
	return Rect{Top: be(0), Left: be(2), Bottom: be(4), Right: be(6)}, true
}

// PictFrame is the picFrame of a picture, zero-cornered. GetObjectRect needs it
// for kCustomPict, whose size is the art's size and nothing else's.
func (a *Assets) PictFrame(id int16) (Rect, bool) {
	s := a.Pict(id)
	if s == nil {
		return Rect{}, false
	}
	return SetRect(0, 0, int16(s.W), int16(s.H)), true
}

// appPictPath maps a PICT id to its file in the extracted application art, for
// the ids the room path can reach by number rather than by name.
//
// The set is deliberately narrow and complete: the room data supplies a
// background id and a kCustomPict id, and the draw helpers supply their own
// hard-coded ids. Everything else in "Glider PRO.r" is reached through the
// original's named globals (the sheets and strips) and has no business being
// looked up by number.
func appPictPath(id int16) (string, bool) {
	switch {
	case id == kSupportPictID:
		return "strip/supp.png", true
	case id >= 2000 && id <= 2017:
		return fmt.Sprintf("bg/%d.png", id), true
	}
	if name, ok := miscNames[id]; ok {
		return fmt.Sprintf("misc/%d_%s.png", id, name), true
	}
	if what, ok := objPictOwner[id]; ok {
		return fmt.Sprintf("object/%02X_%s.png", what, house.ObjectName(what)), true
	}
	return "", false
}

// objPictOwner inverts the atlas: art PICT id -> the object that owns it. The
// nine artOpaque, twenty-one artKey and eight artPair objects each have exactly
// one, and no two share an id.
var objPictOwner = map[int16]int16{}

func init() {
	for _, e := range atlas {
		if e.pict != 0 {
			objPictOwner[e.pict] = e.what
		}
	}
	// kCustomPict's default art. GetObjectRect writes 10000 into data.g.height
	// when a house's own picture is missing, so this id has to resolve.
	objPictOwner[10000] = kCustomPict
}

// loadHousePict decodes one house PICT, which the extractor wrote as RGB with no
// alpha. The surface comes back fully opaque (Mask nil): the caller picks
// SrcCopy or Transparent, as the original does.
func (a *Assets) loadHousePict(path string) *Surface {
	key := "abs:" + path
	a.mu.Lock()
	if s, ok := a.cache[key]; ok {
		a.mu.Unlock()
		return s
	}
	a.mu.Unlock()

	f, err := os.Open(path)
	if err != nil {
		return a.fail(fmt.Errorf("render: %w", err))
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return a.fail(fmt.Errorf("render: decoding %s: %w", path, err))
	}
	s, approx, err := opaqueSurfaceFromImage(img)
	if err != nil {
		return a.fail(fmt.Errorf("render: %s: %w", path, err))
	}

	a.mu.Lock()
	a.cache[key] = s
	a.approx += approx
	a.mu.Unlock()
	return s
}

// opaqueSurfaceFromImage converts an RGB image to an opaque indexed surface,
// returning how many pixels needed a colour match rather than an exact one.
//
// Alpha is required to be fully opaque rather than merely ignored: a house PNG
// with transparency would mean the extractor had decided a transfer mode, and
// that decision belongs to the call site.
func opaqueSurfaceFromImage(img image.Image) (*Surface, int, error) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	s := NewSurface(w, h)

	// A PICT with its own ColorTable has at most 256 distinct colours, so the
	// colour match is memoised per picture: the 512x322 background that costs
	// 165,000 nearest-neighbour searches without it costs a few hundred with one.
	// The memo is local, so it cannot make two pictures disagree.
	memo := map[uint32]uint8{}

	approx := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r16, g16, b16, a16 := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			if a16 != 0xFFFF {
				return nil, 0, fmt.Errorf("alpha %d at (%d,%d): a house picture must be opaque", a16>>8, x, y)
			}
			r, g, bl := uint8(r16>>8), uint8(g16>>8), uint8(b16>>8)
			idx, ok := IndexOf(r, g, bl)
			if !ok {
				key := uint32(r)<<16 | uint32(g)<<8 | uint32(bl)
				if idx, ok = memo[key]; !ok {
					idx = nearestIndex(r, g, bl)
					memo[key] = idx
				}
				approx++
			}
			s.Pix[y*w+x] = idx
		}
	}
	return s, approx, nil
}

// nearestIndex is the stand-in for the Color Manager's inverse table: the
// palette entry closest in unweighted RGB, lowest index winning a tie.
//
// Unweighted because that is what Color2Index does -- it walks the inverse table
// built from the device's 'clut' by straight RGB proximity, with no perceptual
// weighting. This is not bit-exact with the Color Manager, whose inverse table
// quantises the colour cube to 4 or 5 bits per channel before looking up, so a
// colour sitting almost exactly between two entries may land on the other one.
// It affects only the 24 pictures noted above.
func nearestIndex(r, g, b uint8) uint8 {
	best, bestD := uint8(0), 1<<30
	for i := 0; i < 256; i++ {
		c := Palette[i]
		dr, dg, db := int(c.R)-int(r), int(c.G)-int(g), int(c.B)-int(b)
		if d := dr*dr + dg*dg + db*db; d < bestD {
			bestD, best = d, uint8(i)
		}
	}
	return best
}
