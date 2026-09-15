package scores

// The high-score screen: DrawHighScores, transcribed from docs/analysis/scoring.md 7.9.
//
// This is the one function in the port whose original *nothing ever called*. DoHighScores
// drew it into workSrcMap and returned without ever copying that to the screen (7.8), so
// what is reproduced here is a layout read out of the source rather than one anybody
// remembers seeing. That makes the analysis's three worked examples -- 640x480, 1024x768 and
// 512x384 -- the only available evidence, and board_test.go checks every coordinate in all
// three against LayoutFor and against the drawing code.
//
// Its geometry is odd in four ways, all of them deliberate here:
//
//   - **Row 1 is above the banner box.** Rows 2..10 hang below it in the usual way, but row
//     0's baseline is dropIt - kScoreSpacing - kKimsLifted, which puts it *over* the banner.
//     So the reading order down the screen is plaque, first place, banner, second through
//     tenth. `kKimsLifted` is named after Kim Money, who drew the plaque.
//   - **The coloured faces sit one pixel above their black shadows**, not below and to the
//     right. The in-game scoreboard does it the other way (5.5); the title on this same
//     screen does it a third way, up *and* to the left. All three are transcribed as they
//     are.
//   - **Column four is always cyan.** The other four columns switch to white on the row that
//     was just achieved; the word "room"/"rooms" does not, because HighScores.c:240 sets
//     cyan unconditionally where its neighbours branch. The highlighted row therefore reads
//     white white white *cyan* white. Reproduced, and flagged: this is an oversight rather
//     than a design, and it is the sort of thing a "remaster" would quietly fix and thereby
//     lose the evidence of.
//   - **The horizontal anchor centres on the screen and the vertical one on the splash.**
//     scoreLeft comes from the screen width directly; dropIt comes from splashOriginV, which
//     is zero for anything shorter than 480. See LayoutFor.

import (
	"fmt"

	"glidergo/internal/house"
	"glidergo/internal/render"
)

// The layout constants, HighScores.c:90-92.
const (
	scoreSpacing = 18  // kScoreSpacing: the vertical pitch between rows
	scoreWide    = 352 // kScoreWide: the nominal width of the whole block
	kimsLifted   = 4   // kKimsLifted: the extra lift on the banner and on row 0

	plaqueWide = 332 // PICT 1994 and its mask 1998
	plaqueTall = 30

	// The splash rect the vertical anchor is measured from. The backdrop art is 640x460
	// and is stretched into 640x480 (7.10), so these are the destination's dimensions and
	// not the picture's.
	splashWide = 640
	splashTall = 480
)

// The artwork ids, GliderDefines.h.
const (
	PlaquePictID   = 1994 // kHighScoresPictID, the 332x30 colour plaque
	BackdropPictID = 1995 // kStarPictID, the starfield
	PlaqueMaskID   = 1998 // kHighScoresMaskID, a version-1 PICT
)

// The three strings DrawHighScores takes from STR# 150, indices 6, 7 and 8
// (GliderPRO/Glider PRO.r, verified in 7.9.2).
//
// They are constants because this port has no localized-string mechanism at all: there is no
// STR# reader and nothing else in internal/ asks for one. When there is one -- and a game
// meant for the public should have one -- these three move into it and this block becomes
// the English fallback. Noted in docs/IMPROVEMENTS.md.
const (
	RoomWord  = "room"  // index 6, used when levels[i] == 1
	RoomsWord = "rooms" // index 7

	// Index 8 is "Click Mouse or Hit a Key to Exit". The mouse is gone: this port's X11
	// backend does not report pointer events at all and the screen is dismissed by a
	// keystroke, so offering the mouse would be an instruction that does not work. The
	// rest of the sentence is the original's, capitals included.
	ExitWord = "Hit a Key to Exit"
)

// Layout is where the screen's two anchors land on a surface of a given size.
//
// The original derives them at HighScores.c:106-107:
//
//	scoreLeft = ((screen.right - screen.left) - kScoreWide) / 2
//	dropIt    = 129 + splashOriginV
//
// The asymmetry is real and is transcribed rather than tidied: the horizontal anchor
// recentres on the full screen width, while the vertical one is measured from the splash
// screen's own origin, which is zero on any display shorter than 480 pixels. On the 640x480
// this port draws into, both reduce to constants -- 144 and 129 -- and the general form is
// kept only so that a larger surface (Stage 6's mobile targets, or a future window that is
// not exactly 640x480) lands where 1994 would have put it.
type Layout struct {
	Left    int // scoreLeft: the left edge of the 352-pixel block
	Drop    int // dropIt: the anchor every vertical position is measured from
	OriginH int // splashOriginH
	OriginV int // splashOriginV
}

// LayoutFor derives the anchors for a surface w by h.
func LayoutFor(w, h int) Layout {
	var l Layout
	if w > splashWide {
		l.OriginH = (w - splashWide) / 2
	}
	if h > splashTall {
		l.OriginV = (h - splashTall) / 2
	}
	l.Left = (w - scoreWide) / 2
	if l.Left < 0 {
		// Unreachable on any screen the original would run on -- Environ.c rejects
		// anything narrower than 512 -- but a negative anchor would push the placing
		// column off the left edge rather than merely crowd it, and this port can be
		// handed any surface at all by a test or a screenshot target.
		l.Left = 0
	}
	l.Drop = 129 + l.OriginV
	return l
}

// Splash is the rect the 640x460 backdrop is stretched into: 640x480 at the splash origin.
func (l Layout) Splash() render.Rect {
	return render.SetRect(int16(l.OriginH), int16(l.OriginV),
		int16(l.OriginH+splashWide), int16(l.OriginV+splashTall))
}

// Plaque is where the 332x30 plaque goes: centred in the 352-pixel block, sixty pixels above
// the anchor. `(352 - 332) / 2` is 10, and the original writes it out that way.
func (l Layout) Plaque() render.Rect {
	left := int16(l.Left + (scoreWide-plaqueWide)/2)
	top := int16(l.Drop - 60)
	return render.SetRect(left, top, left+plaqueWide, top+plaqueTall)
}

// RowBaseline is the black shadow's baseline for row i; the coloured face is one pixel above
// it. Row 0 is lifted above the banner box, which is what makes this a function and not a
// multiplication.
func (l Layout) RowBaseline(i int) int {
	if i == 0 {
		return l.Drop - scoreSpacing - kimsLifted
	}
	return l.Drop + i*scoreSpacing
}

// FooterBaseline is `dropIt - 1 + 10 * kScoreSpacing`, which is one pixel above where an
// eleventh row would have gone.
func (l Layout) FooterBaseline() int { return l.Drop - 1 + Max*scoreSpacing }

// The five columns' shadow offsets from scoreLeft, HighScores.c:182-262. Each face is drawn
// one pixel left and one pixel up from its shadow.
const (
	colPlacing = 1
	colName    = 31
	colRooms   = 161
	colWord    = 193
	colScore   = 291
)

// Draw paints the whole screen: backdrop, plaque, title, banner, rows, footer.
//
// `highlight` is the original's `lastHighScore`: the row just achieved, drawn in white, or -1
// for none. Insert returns exactly that index, so a caller that has just recorded a score
// passes it straight through.
//
// A missing plate is drawn around rather than fatal, which is docs/IMPROVEMENTS.md 2.6: a
// checkout with no extracted art must still reach a working screen. Nothing here consults
// Assets.Err, because a decoration that will not decode is a broken extraction that the
// asset layer has already recorded.
func Draw(dst *render.Surface, a *render.Assets, houseName string, s *house.Scores, highlight int) {
	l := LayoutFor(dst.W, dst.H)

	drawBackdrop(dst, a, l)
	drawPlaque(dst, a, l)
	drawTitle(dst, l, houseName)
	drawBanner(dst, l, s.Banner.Text())
	for i := 0; i < Max; i++ {
		// HighScores.c:176. A zero score is an empty row and is not drawn -- which is why
		// the placeholder name is never seen, and why a board with three scores on it
		// shows three lines and not ten.
		if s.Scores[i] > 0 {
			drawRow(dst, l, s, i, i == highlight)
		}
	}
	drawFooter(dst, l)
}

// drawBackdrop stretches PICT 1995 into the splash rect. Copy scales when the source and
// destination rects differ, so the 640x460 art reaching 480 rows needs nothing special --
// which is the same stretch DoHighScores asks CopyBits for (7.10).
func drawBackdrop(dst *render.Surface, a *render.Assets, l Layout) {
	dst.Fill(dst.Bounds(), render.Black8)
	if a == nil {
		return
	}
	if art := a.UI(BackdropPictID); art != nil {
		dst.Copy(art, art.Bounds(), l.Splash(), render.SrcCopy)
	}
	// No starfield: the black fill above stands in. Every coordinate below is independent
	// of it, so the screen is legible either way.
}

// drawPlaque copies PICT 1994 through its 1-bit mask 1998, which is CopyMask at
// HighScores.c:120-126. MaskedPlate is the port's version of that pairing and applies it at
// draw time rather than in the extractor, so both pictures stay on disk as they were.
func drawPlaque(dst *render.Surface, a *render.Assets, l Layout) {
	r := l.Plaque()
	if a != nil {
		if art := a.MaskedPlate(PlaquePictID); art != nil {
			dst.Copy(art, art.Bounds(), r, render.Masked)
			return
		}
	}
	// No plaque art. Its own words are unreadable from a PNG that is not there, so the
	// panel says what the screen is instead -- the alternative is a screen whose heading
	// is a blank strip.
	dst.Fill(r, render.Black8)
	dst.FrameRect(r, render.QDYellow)
	const what = "HIGH SCORES"
	w := int(render.StringWidthScaled(what, 2))
	dst.DrawStringScaled(int16(int(r.Left)+(plaqueWide-w)/2), r.Top+21, what, render.QDYellow, 2)
}

// drawTitle is the house's name between two bullets, in 14 pt bold: a black shadow up and to
// the left of a cyan face (HighScores.c:137-145).
//
// The bullet is Mac Roman 0xA5, which the original writes as a literal byte in a Pascal
// string. Here it is the rune it decodes to, U+2022, and the font resolves it the same way
// it resolves every other character in a house name.
func drawTitle(dst *render.Surface, l Layout, houseName string) {
	title := "• " + houseName + " •"

	// The original's one size change: 14 pt for the title against 12 pt for everything
	// else. This font has one size and an integer scale, so "bigger" is scale 2 -- when it
	// fits. A house with a long name gets scale 1 rather than a title running off both
	// sides of the block, which is a judgement the original never had to make because a
	// 14 pt proportional font simply drew narrower.
	scale := 2
	w := int(render.StringWidthScaled(title, scale))
	if w > scoreWide {
		scale = 1
		w = int(render.StringWidthScaled(title, scale))
	}

	x := l.Left + (scoreWide-w)/2
	dst.DrawStringScaled(int16(x-1), int16(l.Drop-66), title, render.Black8, scale)
	dst.DrawStringScaled(int16(x), int16(l.Drop-65), title, render.QDCyan, scale)
}

// drawBanner is the champion's message and the double box around it, HighScores.c:156-172.
//
// The box is framed twice: black at (dropIt-17), then the same rect moved one pixel up and
// left, framed yellow. The face of the text is one pixel above its shadow with no horizontal
// shift -- a different relationship from the title's, five lines earlier in the same
// function.
func drawBanner(dst *render.Surface, l Layout, banner string) {
	bw := int(render.StringWidth(banner))
	x := l.Left + (scoreWide-bw)/2

	dst.DrawString(int16(x), int16(l.Drop-4), banner, render.Black8)
	dst.DrawString(int16(x), int16(l.Drop-5), banner, render.QDYellow)

	// QSetRect(0, 0, bannerWidth + 8, 18) offset to (scoreLeft - 3 + (352 - bw)/2,
	// dropIt - 17). Written the same way round so the eight and the three stay visible.
	left := int16(l.Left - 3 + (scoreWide-bw)/2)
	top := int16(l.Drop - 17)
	box := render.SetRect(left, top, left+int16(bw+8), top+18)
	dst.FrameRect(box, render.Black8)
	dst.FrameRect(render.Offset(box, -1, -1), render.QDYellow)
}

// drawRow is one line of the board: five columns, each a black shadow with a coloured face
// one pixel up and one pixel left (HighScores.c:180-262).
func drawRow(dst *render.Surface, l Layout, s *house.Scores, i int, lit bool) {
	rooms := s.Levels[i]
	word := RoomsWord
	if rooms == 1 {
		word = RoomWord
	}

	// The face colours. `lit` is `i == lastHighScore`.
	var placingFace, textFace uint8 = render.QDCyan, render.QDYellow
	if lit {
		placingFace, textFace = render.White8, render.White8
	}

	for _, c := range []struct {
		dx   int
		text string
		face uint8
	}{
		{colPlacing, fmt.Sprint(i + 1), placingFace},
		{colName, Name(s, i), textFace},
		{colRooms, fmt.Sprint(rooms), textFace},
		// Column four takes no `lit` branch: HighScores.c:240 sets cyan unconditionally
		// where its four neighbours test lastHighScore. The highlighted row really does
		// read white white white cyan white.
		{colWord, word, render.QDCyan},
		{colScore, fmt.Sprint(s.Scores[i]), textFace},
	} {
		v := l.RowBaseline(i)
		dst.DrawString(int16(l.Left+c.dx), int16(v), c.text, render.Black8)
		dst.DrawString(int16(l.Left+c.dx-1), int16(v-1), c.text, c.face)
	}
}

// drawFooter is the blue 9 pt line at the bottom, HighScores.c:266-272. It is the only text
// on the screen with no shadow behind it.
func drawFooter(dst *render.Surface, l Layout) {
	dst.DrawString(int16(l.Left+80), int16(l.FooterBaseline()), ExitWord, render.QDBlue)
}
