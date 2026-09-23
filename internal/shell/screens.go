package shell

// Drawing the title screen, the menu, the picker and the About box.
//
// Everything here composes into one 640x480 indexed surface with the game's own
// palette and no alpha channel, which is the constraint that decides the whole
// look. There is no way to draw a translucent panel over the splash illustration,
// because there is no channel to be translucent in; what the original does instead,
// everywhere it needs to put something over existing pixels, is PenPat(gray) with
// PenMode(patOr) -- a 50% checkerboard ORed into the destination
// (docs/analysis/rendering.md, and render.Surface.FillPatOrGray). Dimming with
// Black8 that way turns half the pixels black and leaves the other half alone,
// which reads as a dark scrim and cost nothing on a 68k. It is used for every panel
// below.
//
// Text is the port's own 5x7 bitmap font, magnified for anything a player has to
// read from across a desk (render.DrawStringScaled). The original asked the Toolbox
// for TextSize(9) and TextSize(12) and got two designed faces; a single authored
// bitmap cannot do that, so scaling is the honest substitute. Every string that
// lands on artwork gets the original's own trick for legibility: the same text one
// pixel down and right in black first, then the real colour over it
// (render/scoreboard.go does it at 217-218, and docs/analysis/ui-dialogs.md P14).

import (
	"fmt"
	"strings"

	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/prefs"
	"github.com/bwenstar/gliderGo/internal/project"
	"github.com/bwenstar/gliderGo/internal/render"
)

// The port's screen. Fixed, because the game's own coordinate system is: the
// original's window is 640x480 and the splash PICT and every offset written on top
// of it are hard-coded to it (docs/analysis/ui-dialogs.md P15).
const (
	screenWide = 640
	screenTall = 480
)

// Palette indices, by number because render only names the ones the room path
// reaches (render/palette.go's const block).
//
// The palette is the 6x6x6 Mac cube in {FF,CC,99,66,33,00} order, so an index is
// r*36 + g*6 + b in that ordering:
//
//	 8 = (0,1,2) = #FFCC99, the cream the original's dialogs and its scoreboard use
//	28 = (0,4,4) = #FF3333, and it is not a choice -- it is the exact colour
//	     DrawOnSplash sets before writing the house name (SelectHouse.c:87-96)
const (
	cream = 8
	label = 28
)

// The splash PICT is 640x460 drawn at the origin, which leaves 20 rows at the
// bottom of the port's window that the original had nothing in -- its window was
// 640x460 plus a menu bar. That band is where the status line goes: the one thing
// this shell has that the original did not need, because the original could put a
// message in an alert and a modal event loop to dismiss it.
const (
	splashTall = 460
	bandTall   = screenTall - splashTall
	bandText   = 473 // baseline
)

// Where DrawOnSplash puts the house name (SelectHouse.c:92-95: MoveTo(436, 314)
// after TextFont(applFont)/TextSize(9)/TextFace(bold)). Kept exactly, and every
// panel below is placed so as not to cover it.
const (
	houseLabelH = 436
	houseLabelV = 314
)

// The menu panel. Right of centre and above the house label, over what is sky and
// roofline in the shipped splash art rather than over the drawing's subject.
//
// Only three sides are constants. The bottom is derived from the number of rows the menu has
// -- see menuRect -- because the space between this panel and the house label is exactly as
// deep as the menu needs and not a pixel deeper.
//
// The top is not a free choice either, and it stopped being one when the ninth row arrived.
// panel() dims eight rows beyond every edge, so the halo begins at menuTop-8, and above
// splashSkyV that halo would lay a checkerboard across the aeroplane's tail. 102 puts the
// halo's first row exactly on the first row of clear sky: as high as this panel can go
// without touching the drawing, which is as high as it has to go, because its bottom is
// pinned by the house label and the ninth row had to come from somewhere.
const (
	menuTop   = 102
	menuLeft  = 336
	menuRight = 632

	// splashSkyV is the first row of PICT 1000 that is clear sky across this panel's
	// columns, the halo's eight included (x 328 to 640). Measured off the extracted art:
	// rows 78 to 93 hold the tail of the aeroplane, narrowing from 105 pixels to 10, and
	// row 94 is the first with none. It is written down rather than measured at run time
	// because the art is fixed and because the test that reads it has to pass on a build
	// with no assets extracted at all.
	splashSkyV = 94

	menuScale = 2  // 12x18 per character
	menuFirst = 19 // first baseline, relative to menuTop
	menuPitch = 21
	menuInset = 14

	// From the last baseline to the panel's bottom, which is not the height of a row: a
	// glyph at scale s reaches s*2-1 rows below its baseline and shadow() puts a black
	// pixel one scale under that, so the deepest ink of the last row is 3*menuScale-1
	// below it -- and the panel's inner frame is three rows above its bottom. Ten leaves
	// that ink clear of the frame with a row to spare, which is what
	// TestMenuPanelClearsTheHouseLabel checks along with the other end. It is the minimum
	// that has that spare row, so a tenth menu row cannot be bought here.
	menuFoot = 10

	// The menu's own selection bar: how far it rises above the baseline. It cannot be the
	// picker's barRise below, and it is no longer one pitch less than the row's own depth
	// either -- which is what it was until the ninth row had to be found somewhere.
	//
	// Two rows of this bar's padding is what paid for that row. A bar runs from v-menuRise
	// to v+2*menuScale, and what bounds menuRise from above is the *glyphs* of the row above,
	// whose descenders reach v-menuPitch+2*menuScale-1 = v-18; anything up to 17 clears them,
	// and their shadows do not come into it because shadow() draws in Black8 and the panel is
	// already Black8. What menuRise bounds in turn is menuFirst, because the first row's bar
	// is the highest thing in the panel and has to stay a clear row below the inner frame --
	// so a shorter bar lets the first baseline sit nearer the top, and 19+15 is two rows less
	// than 21+17 while still leaving a row of cream above the row's own capitals (which ink
	// from v-7*menuScale). It is tighter than it was. The alternatives were a menu of eight
	// rows or a halo laid across the artwork; see menuTop.
	menuRise = 15
)

// The inverse-video selection bar of the house picker and the settings screen: how far it
// rises above the baseline and how far it is inset from the panel's edges. The menu's rise is
// menuRise, which is tied to its own pitch.
const (
	barRise = 20
	barPad  = 6
)

// The house picker, which needs the whole screen: forty houses at a size somebody
// can read is ten rows a page.
const (
	pickTop    = 36
	pickLeft   = 56
	pickBottom = 436
	pickRight  = 584

	pickerRows = 10
	pickScale  = 2
	pickFirst  = 100 // first row's baseline, absolute
	pickPitch  = 28
	pickNameH  = 76  // left edge of a name
	pickRightH = 564 // right edge of the room count
	pickTitleV = 68
	pickFootV  = 396
	pickFootV2 = 416

	// The level-set strip shares the title's baseline, between "Load House" and the page
	// counter, and that is not where it was first put. The obvious place is the gap between
	// the title and the first house -- 32 rows, apparently empty. It is not empty: the
	// selected house's inverse-video bar rises barRise above its baseline, so the first row's
	// bar occupies rows 80 to 104 and the title's shadow reaches row 73. Six rows, and a
	// scale-1 word needs ten. The strip went on the title's line because that is the row that
	// actually has room, and a screen that has to be measured before a row can be used is
	// exactly the screen that needs
	// TestTheSetStripFitsBetweenTheTitleAndThePageCounter.
	//
	// pickSetsH is right of "Load House" at scale 2: pickLeft+20 plus ten glyphs of 12,
	// plus the shadow, plus a space.
	pickSetsV = pickTitleV
	pickSetsH = 216

	// setsRise is barRise's scale-1 equivalent. The picker's row bar is sized for scale-2
	// names; a scale-1 word needs a bar as tall as a scale-1 glyph, which inks 7 rows up from
	// the baseline, plus one clear row above it.
	setsRise = 8

	// setsGap is the blank space between two entries in the strip. Two glyph widths
	// (render.FontWide is 6), which reads as a clear break at scale 1 without letting
	// "Original 22" and "New 4" run together when neither has a bar under it.
	setsGap = 12

	// setsPad is how far a set's bar extends either side of its word, for the same reason
	// barPad exists: a word touching the edge of its own highlight is harder to read than one
	// with a margin.
	setsPad = 3
)

// The About panel. Its height is not here, because it is not a constant: the plate
// is either extracted or it is not, and the box is sized to whichever it got (see
// drawAbout).
const (
	aboutLeft  = 88
	aboutRight = 552

	aboutPlateTall = 120 // the tallest plate this box will make room for
	aboutFirst     = 34  // the first baseline, below the panel's top
	aboutFoot      = 10  // below the last baseline

	// aboutPlate is the picture the original's About dialog puts at the top of itself:
	// DITL 150 item 4, 372x100 (docs/analysis/ui-dialogs.md 10.6). It is a constant with
	// a name because the two PICTs either side of it are the fake Okay button's two
	// states, this drew 151 for four stages, and provenance_test.go now holds the number
	// to the document rather than to whoever last read it.
	aboutPlate = 153
)

// Draw composes the current screen. It draws everything every time -- no dirty
// rectangles, no retained panels -- because a title screen has all the time in the
// world and because a shell that redraws completely cannot leave a stale pixel from
// a panel it has closed. (The game does the opposite, and has to: see
// docs/analysis/rendering.md on RefreshGameWindow.)
func (s *Shell) Draw() {
	scr := s.host.Screen
	if scr == nil {
		return
	}
	// The high-score board is a screen and not a panel: it brings its own backdrop, its own
	// heading and its own way out (docs/analysis/scoring.md 7.9.1), and nothing of the title
	// screen shows through it. So it replaces the backdrop rather than sitting on one, and
	// there is no menu and no house label over it -- the house's name is already the board's
	// title. If it declines to draw, it has put the mode back to the splash and the switch
	// below draws that instead of nothing.
	if s.mode == modeScores && s.drawScores() {
		s.drawBand(scr)
		return
	}
	s.drawBackdrop(scr)
	switch s.mode {
	case modeSplash:
		s.drawMenu(scr)
	case modeHouses:
		s.drawPicker(scr)
	case modeSettings:
		s.drawSettings(scr)
	case modeRace:
		s.drawRace(scr)
	case modeAbout:
		s.drawAbout(scr)
	case modeCredits:
		s.drawCredits(scr)
	}
	s.drawBand(scr)
}

// plate fetches one of the shell's PICTs, or nil. A nil Assets is not a
// programming error here -- see Host.Assets.
func (s *Shell) plate(id int16) *render.Surface {
	if s.host.Assets == nil {
		return nil
	}
	return s.host.Assets.UI(id)
}

// drawBackdrop paints the splash illustration, with this port's name stamped over the
// 1994 one (title.go), or the port's own title screen when there is no artwork to paint.
func (s *Shell) drawBackdrop(scr *render.Surface) {
	art := s.plate(1000)
	if art == nil {
		s.drawOwnTitle(scr)
	} else {
		scr.Fill(render.SetRect(0, 0, screenWide, splashTall), render.Black8)
		src := art.Bounds()
		if src.Right > screenWide {
			src.Right = screenWide
		}
		if src.Bottom > splashTall {
			src.Bottom = splashTall
		}
		scr.Copy(art, src, src, render.SrcCopy)
		stampTitle(scr, art)
	}
	s.drawHouseLabel(scr)
}

// drawOwnTitle is the screen for a build that cannot find its artwork.
//
// A clone ships assets/extracted/, so this is no longer what a first run looks
// like. It is what `make clean-assets`, a bad -art path or an incomplete download
// looks like, and it is still worth drawing: a game that comes up looking like a
// game and names the command that fixes it beats one that exits before drawing
// anything (docs/IMPROVEMENTS.md 2.6).
func (s *Shell) drawOwnTitle(scr *render.Surface) {
	scr.Fill(render.SetRect(0, 0, screenWide, splashTall), render.Black8)

	// A horizon, so the screen has some structure to it and the eye has somewhere
	// to sit. Two lines and a band, in the palette's own greys.
	scr.Fill(render.SetRect(0, 300, screenWide, splashTall), render.DkGray28)
	scr.Line(0, 300, screenWide-1, 300, render.Gray28)
	scr.Line(0, 301, screenWide-1, 301, render.DkGray8)

	// Centred on the column *left* of the menu panel, not on the screen. The shipped
	// splash art has its subject on the left and its sky on the right, which is why the
	// menu sits where it does; a fallback screen that centred on 320 put "no artwork
	// found" half underneath that panel, which was the first thing a fresh clone saw.
	// So the port's own title screen is laid out around the same panel the real one is.
	col := int16(menuLeft - 16)
	centerIn(scr, 0, col, 96, "gliderGo", cream, 5)
	centerIn(scr, 0, col, 140, "Glider PRO, ported", cream, 2)
	centerIn(scr, 0, col, 168, "John Calhoun, 1994", render.LtGray8, 1)

	centerIn(scr, 0, col, 240, "no artwork found", label, 2)
	centerIn(scr, 0, col, 266, "it ships in assets/extracted;", cream, 1)
	centerIn(scr, 0, col, 278, "`make assets` rebuilds it", cream, 1)
}

// drawHouseLabel writes the selected house's name where DrawOnSplash writes it.
//
// The original draws it with the pen at a fixed point and no measurement at all,
// which is fine for a house called "Slumberland" and runs off the right edge of the
// screen for a house called something longer -- there is nothing stopping a player
// from naming one, and the port will read houses the original never saw. So: keep
// the point, slide the whole label left if it would overflow, and shorten it only
// once sliding has run out of screen. Every shipped house name is drawn exactly
// where 1994 drew it, and no name can spill off the edge.
func (s *Shell) drawHouseLabel(scr *render.Surface) {
	h, ok := s.House()
	if !ok {
		return
	}
	text := "House: " + h.Name
	x := int16(houseLabelH)
	if over := x + render.StringWidth(text) - (screenWide - 4); over > 0 {
		x -= over
	}
	if x < 4 {
		x = 4
		text = fit(text, screenWide-8, 1)
	}
	shadow(scr, x, houseLabelV, text, label, 1)
}

// ---------------------------------------------------------------------------
// The menu
// ---------------------------------------------------------------------------

// menuRect is the panel n menu rows need.
//
// Deriving it is not tidiness. The panel is drawn *after* the house label (see Draw's order)
// and panel() dims eight rows beyond every edge, so the halo's last row is the bottom plus
// seven -- and the name DrawOnSplash writes at baseline 314 inks upwards from row 307, the
// baseline less the font's ascent. A bottom past 299 therefore lays grey over the top of the
// house's name. A constant bottom got that wrong in the other direction too: menuBottom was
// 264 while the seventh row's baseline was already 298, so the last two rows of a seven-row
// menu were drawn outside the box they belong to. With the bottom derived, adding a row moves
// the box instead of overflowing it, and TestMenuPanelClearsTheHouseLabel is what stops a row
// from being added past the point where the box can no longer move.
func menuRect(n int) render.Rect {
	if n < 1 {
		n = 1
	}
	return render.SetRect(menuLeft, menuTop, menuRight,
		int16(menuTop+menuFirst+(n-1)*menuPitch+menuFoot))
}

func (s *Shell) drawMenu(scr *render.Surface) {
	m := s.menu()
	panel(scr, menuRect(len(m)))

	for i := range m {
		v := int16(menuTop + menuFirst + i*menuPitch)
		text := m[i].label
		col := uint8(cream)
		if !m[i].ok {
			col = render.Gray8 // the greyed-out menu item, kept as an idea
		}
		if i == s.sel {
			bar := render.SetRect(int16(menuLeft+barPad), v-menuRise,
				int16(menuRight-barPad), v+int16(menuScale*2))
			if !m[i].ok {
				// An unavailable item under the cursor gets an *outline*, not a filled
				// bar. A filled bar is the port's "this is what Return will do", and
				// putting it under "New Game" on a machine with no houses -- which is
				// what a mistyped -houses shows first -- made the one item that cannot
				// work look like the one the game wanted you to press.
				scr.FrameRect(bar, render.Gray8)
				shadow(scr, int16(menuLeft+menuInset), v, text, render.Gray8, menuScale)
				continue
			}
			// The selection bar is an inverse video row, which is what the
			// original's own list selections are (SelectHouse.c's InvertRect on the
			// chosen house's rect). Cream on black becomes black on cream.
			scr.Fill(bar, cream)
			scr.DrawStringScaled(int16(menuLeft+menuInset), v, text, render.Black8, menuScale)
			continue
		}
		shadow(scr, int16(menuLeft+menuInset), v, text, col, menuScale)
	}
}

// ---------------------------------------------------------------------------
// The picker
// ---------------------------------------------------------------------------

// drawPicker draws the list, one page of pickerRows houses at a time.
//
// It draws the *filtered* view rather than lib.Houses, and everything derived from the list is
// derived from the view with it: the page number, the page count, and which row carries the
// bar. The cursor is still an index into the library -- see Shell.set for why -- so the row it
// lands on is its position within the view, which is what s.at computes.
func (s *Shell) drawPicker(scr *render.Surface) {
	r := render.SetRect(pickLeft, pickTop, pickRight, pickBottom)
	panel(scr, r)

	view := s.view()
	n := len(view)
	at := s.at(view)
	page := 0
	if pickerRows > 0 {
		page = at / pickerRows
	}
	pages := (n + pickerRows - 1) / pickerRows
	if pages == 0 {
		pages = 1
	}

	shadow(scr, pickLeft+20, pickTitleV, "Load House", cream, 2)
	right(scr, pickRightH, pickTitleV, fmt.Sprintf("page %d of %d", page+1, pages), render.LtGray8, 1)
	s.drawSets(scr)

	for row := 0; row < pickerRows; row++ {
		at := page*pickerRows + row
		if at >= n {
			break
		}
		i := view[at]
		h := s.lib.Houses[i]
		v := int16(pickFirst + row*pickPitch)
		name := fit(h.Name, pickRightH-pickNameH-110, pickScale)
		count := counted(int(h.Rooms), "room")

		if i == s.pick {
			bar := render.SetRect(pickLeft+barPad, v-barRise,
				pickRight-barPad, v+int16(pickScale*2))
			scr.Fill(bar, cream)
			scr.DrawStringScaled(pickNameH, v, name, render.Black8, pickScale)
			rightScaled(scr, pickRightH, v, count, render.Black8, 1)
			continue
		}
		shadow(scr, pickNameH, v, name, cream, pickScale)
		right(scr, pickRightH, v, count, render.LtGray8, 1)
	}

	s.drawPickerFooter(scr)
}

// drawSets is the level-set strip: each set with its count, the one showing in inverse video,
// on the title's line.
//
// It draws nothing at all when there are fewer than two choices, which is the case every build
// carrying only the 1994 houses is in. That is not a tidiness argument. A chooser with one
// position in it offers a choice that does not exist, and drawing one would move pixels on a
// screen the fidelity corpus holds a hash of -- so a port that adds no houses of its own draws
// the picker unchanged, and the corpus goes on measuring the original rather than this.
//
// Each entry carries its count, which is setLabel's whole reason: the strip is the only place
// on this screen where a player can see that there are four new houses without filtering to
// them and reading the list. The words are the sets' own String values, so there is no second
// list of display names to disagree with the first (sets.go).
func (s *Shell) drawSets(scr *render.Surface) {
	choices := s.lib.Choices()
	if len(choices) < 2 {
		return
	}
	h := int16(pickSetsH)
	for _, set := range choices {
		text := s.setLabel(set)
		w := render.StringWidthScaled(text, 1)
		if set == s.filter {
			// The same inverse-video bar the house rows use, at scale 1, and drawn without
			// shadow() for the reason rightScaled exists: a black shadow under black text on
			// a cream bar only thickens it.
			scr.Fill(render.SetRect(h-setsPad, pickSetsV-setsRise, h+w+setsPad, pickSetsV+2), cream)
			scr.DrawString(h, pickSetsV, text, render.Black8)
		} else {
			shadow(scr, h, pickSetsV, text, render.LtGray8, 1)
		}
		h += w + setsGap
	}
}

// setLabel is one entry of the strip: the set's name and how many houses it holds.
func (s *Shell) setLabel(set Set) string {
	return fmt.Sprintf("%s %d", set, s.lib.Count(set))
}

// setsWide is how far the strip reaches, for the layout test and for nothing else. It is a
// function of the library rather than a constant because the number of sets is.
func (s *Shell) setsWide() int16 {
	choices := s.lib.Choices()
	if len(choices) < 2 {
		return 0
	}
	w := int16(0)
	for i, set := range choices {
		if i > 0 {
			w += setsGap
		}
		w += render.StringWidthScaled(s.setLabel(set), 1)
	}
	return w + 2*setsPad
}

// drawPickerFooter is where the picker earns its keep: the two things the original's
// dialog could not tell you.
//
// The first is what is *in* the house you are about to open -- its high score, so a
// board that has been sitting in a house file since 1994 is visible before the game
// starts rather than only after somebody dies. The second is what was rejected: the
// original silently omits a file that looked like a house and was not, which leaves
// a player who can see the file in the folder with no way to find out why it is not
// in the list (docs/IMPROVEMENTS.md 2.33).
//
// The score shown is off the *merged* board -- the file's rows plus anything this
// installation has recorded since -- because otherwise the number here and the number on
// the High Scores screen would disagree about the same house, and the one that disagreed
// would be the one a player sees first.
// The two lines are composed by footerBest and footerKeys and drawn here, rather than built
// inline, so that a test can read what the picker would say without a surface to look at.
// Both are pure functions of the shell's state, which is the difference between a test that
// catches a wrong string and one that catches a blank screen.
func (s *Shell) drawPickerFooter(scr *render.Surface) {
	w := int16(pickRight - pickLeft - 40)
	shadow(scr, pickLeft+20, pickFootV, fit(s.footerBest(), w, 1), cream, 1)
	shadow(scr, pickLeft+20, pickFootV2, fit(s.footerKeys(), w, 1), render.LtGray8, 1)
}

// footerBest is the first line: what is in the house under the cursor.
func (s *Shell) footerBest() string {
	h, ok := s.picked()
	if !ok {
		return "no house selected"
	}
	best := "no scores yet"
	if who, score, ok := bestOf(s.board(h)); ok {
		best = fmt.Sprintf("best: %s %d", who, score)
	}
	if h.Locked {
		best += "   (locked)"
	}
	// Which set this house is from, but only while the list is showing all of them: with a
	// filter on, the title and the strip both already say it, and repeating it here would
	// spend the one line that tells a player something about the house on something they can
	// read in two other places.
	if s.filter == AllSets && len(s.lib.Sets()) > 1 {
		best += "   [" + h.Set.String() + "]"
	}
	return best
}

// footerKeys is the second line: the keys, or the rejected files if there were any.
//
// Tab appears only when there is a set to switch to, which is what makes its repurposing
// honest -- see pickerKey. The skipped-files message takes the whole line when there is one,
// because a file the player can see in the folder and cannot see in the list is a question
// they are already asking and the keys are not.
func (s *Shell) footerKeys() string {
	if n := len(s.lib.Skipped); n > 0 {
		return fmt.Sprintf("%d file(s) skipped, e.g. %s", n, s.lib.Skipped[0].Why)
	}
	if len(s.lib.Choices()) > 1 {
		return "arrows move   Tab set   Return plays   Space selects   Esc back"
	}
	return "arrows move   Return plays   Space selects   Esc back"
}

// picked is the house under the picker's cursor, which is not the selected one
// until Return or Space says so.
func (s *Shell) picked() (House, bool) {
	if s.pick < 0 || s.pick >= len(s.lib.Houses) {
		return House{}, false
	}
	return s.lib.Houses[s.pick], true
}

// Best is the top row of the house's own high-score board, as it was read off disk.
//
// This is the *file's* board and not the merged one: the picker's footer uses
// bestOf(s.board(h)) instead, so that what it shows includes scores this installation has
// recorded. Best stays because it is the answer to a question about a house on its own --
// no side-car, no host -- which is what house.PeekFile hands back and what the tests ask
// about.
func (h House) Best() (string, int32, bool) { return bestOf(h.Scores) }

// bestOf is the top row of a board.
//
// The board is stored sorted descending -- AddHighScore inserts in place
// (docs/analysis/scoring.md 7.4) -- but this scans all ten rather than trusting row
// zero, because these are 30-year-old files edited by a program with a house editor
// in it and a board that is out of order is not worth a wrong answer. A zero score
// means an empty row, which is how the original tells them apart too.
func bestOf(b house.Scores) (string, int32, bool) {
	best, at := int32(0), -1
	for i, sc := range b.Scores {
		if sc > best {
			best, at = sc, i
		}
	}
	if at < 0 {
		return "", 0, false
	}
	who := b.Names[at].Text()
	if strings.TrimSpace(who) == "" {
		who = "(nameless)"
	}
	return who, best, true
}

// ---------------------------------------------------------------------------
// About
// ---------------------------------------------------------------------------

// drawAbout is the port's About box. The original's is a DLOG with three PICTs in
// it (docs/analysis/ui-dialogs.md 2.4); this draws one of them if it is there and
// otherwise says the same things in text, because what the box is actually for is
// the credit and the licence, and those have to be legible whether or not anybody
// has extracted the artwork.
func (s *Shell) drawAbout(scr *render.Surface) {
	lines := s.aboutLines()

	// The plate, if there is one small enough to be a heading rather than the whole
	// box. It replaces the drawn wordmark, which is what the original's About dialog
	// does with its own PICTs (docs/analysis/ui-dialogs.md 2.4).
	//
	// 153 and not 150 or 151. All three are About box PICTs and the numbers are
	// adjacent, which is how this came to be 151 for four stages: 150 and 151 are the
	// two states of a 63x63 fake Okay button (`kOkayButtPICTNotHiLit` /
	// `kOkayButtPICTHiLit`, ui-dialogs.md 5.5), so the heading of the one screen whose
	// whole job is to look deliberate was a white square with a diamond in it. 153 is
	// the 372x100 title art -- DITL 150's picture item, ui-dialogs.md 10.6 -- and it is
	// also the only place the 1994 game credits three of its house authors on screen,
	// which is why drawing the wrong one cost more than it looked.
	art := s.plate(aboutPlate)
	if art != nil && (art.W > aboutRight-aboutLeft-16 || art.H > aboutPlateTall) {
		art = nil
	}

	// Measure, then place. A panel with a constant bottom had eighty rows of nothing
	// under the last line whenever the plate was missing -- which reads as a drawing
	// that failed rather than as a box that is the size of what is in it. Centring what
	// is left is then free, and puts the box where the eye already is.
	head := int16(aboutFirst)
	if art != nil {
		head = int16(art.H) + 24
	}
	body := int16(0)
	for _, l := range lines {
		body += l.tall()
	}
	tall := aboutFirst + head + body + aboutFoot
	top := (int16(splashTall) - tall) / 2
	if top < 24 {
		top = 24
	}
	panel(scr, render.SetRect(aboutLeft, top, aboutRight, top+tall))

	v := top + aboutFirst
	if art != nil {
		x := aboutLeft + (aboutRight-aboutLeft-int16(art.W))/2
		dst := render.SetRect(x, v-10, x+int16(art.W), v-10+int16(art.H))
		scr.Copy(art, art.Bounds(), dst, render.SrcCopy)
	} else {
		center(scr, v, project.Name, cream, 3)
	}
	v += head

	for _, l := range lines {
		if l.text != "" {
			center(scr, v, l.text, l.col, l.scale)
		}
		v += l.tall()
	}
}

// aboutLines is what the box says, in order. Separate from the drawing so that a test can
// read it: the box is the port's only documentation, and the two things most likely to be
// wrong in it are a control it describes that nobody has bound and a way out it does not
// mention.
func (s *Shell) aboutLines() []aboutLine {
	// The keys are read out of the live preferences, not written out here, and that is
	// the whole point of listing them: three of the port's bindings are not the
	// original's (see cmd/glidergo's package comment on player two, and
	// docs/IMPROVEMENTS.md 2.3 and 2.52), and since 1.7b all eight are the player's to
	// change -- so a hard-coded list would be wrong for anybody who has visited the
	// settings screen, which is exactly the person most likely to open this box.
	//
	// With no Prefs on the host it describes this build's defaults, which is what -shot
	// and the tests get and is still true of a fresh install.
	p := s.host.Prefs
	if p == nil {
		p = prefs.Default()
	}

	// Which build this is, and where it came from. Both are here for the same reason: this
	// box is the only thing a player who downloaded a binary can read, and the two facts a
	// bug report is useless without are the version and somewhere to send it. A player
	// photographing this screen -- which is what actually happens -- captures both.
	//
	// The URL drops its scheme because nobody types https:// and the line has to fit a
	// 640-pixel panel at 1x. It is still derived from project.Home rather than written out,
	// so the string here cannot come to differ from the one in the README, the usage text
	// and the issue templates (internal/project's own tests).
	build := project.Name
	if s.host.Version != "" {
		build += " " + s.host.Version
	}

	lines := []aboutLine{
		{"a port of " + project.Original, cream, 1},
		{project.OriginalAuthor + " / " + project.OriginalPublisher + ", " + project.OriginalYear,
			render.LtGray8, 1},
		{},
		{build, cream, 1},
		{strings.TrimPrefix(project.Home, "https://"), render.LtGray8, 1},
		{},
		{"source released under the GPL, version 2", render.LtGray8, 1},
		// And *which* source, pinned. This is the line that turns the box from a credit
		// into provenance: a port is a claim about a particular revision of somebody else's
		// code, and "a port of Glider PRO" is not checkable while the revision is implicit.
		// Somebody who wants to know whether a behaviour is 1994's or ours can now find the
		// exact tree the transcription was read from without cloning this repository first.
		{"transcribed from " + strings.TrimPrefix(project.Upstream, "https://") +
			" @ " + project.UpstreamCommit, render.LtGray8, 1},
		// What ships, and whose it is. The count is the 1994 set and is deliberately not the
		// number of rows in the picker: houses written for this port ship in the same binary
		// (assets/levels.zip) and the box would be claiming 1994 wrote them if it added them
		// up. That is 4.14's whole argument, made here because this box is the only
		// documentation a downloaded executable carries -- the picker's set strip says which
		// row is which, but only for somebody already looking at it.
		{"the 1994 art, sounds and 22 houses ship inside it,", render.LtGray8, 1},
		{"with the houses this port adds; nothing to download", render.LtGray8, 1},
		{},
		{"player one:  " + controlsLine(p.Player1), cream, 1},
		{"player two:  " + controlsLine(p.Player2), cream, 1},
		{},
		{upperFirst(p.PauseKey) + " or Esc pauses   Delete gives up a waiting glider", cream, 1},
		{"Q while paused gives up the game and comes back here", cream, 1},
		{"S while paused saves it; O here starts it again", cream, 1},
		{"S on the title screen changes any of this", render.LtGray8, 1},
		{},
		{"C names everybody who made the original", cream, 1},
		{"press any other key", label, 1},
	}
	return lines
}

// controlsLine describes one glider's four keys in one line, in the order somebody
// would try them: steering first, then the two things the player spends.
func controlsLine(c prefs.Controls) string {
	return fmt.Sprintf("%s / %s steer, %s throws a band, %s the battery",
		c.Left, c.Right, c.Band, c.Batt)
}

// upperFirst capitalises a key name for the start of a sentence, so a preference that
// stores "tab" reads as "Tab pauses". ASCII only, which every key name is.
func upperFirst(s string) string {
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		return s
	}
	return string(rune(s[0]-'a'+'A')) + s[1:]
}

// aboutLine is one row of the About box: its text, its colour and its magnification. A
// zero line is a blank one, which is why the fields are not required.
type aboutLine struct {
	text  string
	col   uint8
	scale int
}

// tall is how far the baseline advances past this line. A blank line is a gap rather
// than an empty row, so it takes a little more than the font's own height.
func (l aboutLine) tall() int16 {
	if l.text == "" {
		return 15
	}
	return int16(11 * l.scale)
}

// ---------------------------------------------------------------------------
// The status band
// ---------------------------------------------------------------------------

// drawBand fills the 20 rows below the splash art and writes the status line.
//
// It is solid black rather than dimmed, because it is not over anything: it is the
// strip of the port's taller window that the original's art never covered. That
// makes it the one place on screen where small text is reliably legible, which is
// why the messages go here rather than over the picture.
func (s *Shell) drawBand(scr *render.Surface) {
	scr.Fill(render.SetRect(0, splashTall, screenWide, screenTall), render.Black8)
	scr.Line(0, splashTall, screenWide-1, splashTall, render.DkGray8)

	ver := s.host.Version
	room := int16(screenWide - 16)
	if ver != "" {
		w := render.StringWidth(ver)
		right(scr, screenWide-8, bandText, ver, render.Gray8, 1)
		room -= w + 12
	}
	if line := s.status(); line != "" {
		scr.DrawString(8, bandText, fit(line, room-8, 1), cream)
	}
}

// ---------------------------------------------------------------------------
// Drawing helpers
// ---------------------------------------------------------------------------

// panel is a solid black box with a cream frame, standing on a dimmed halo of the
// artwork behind it.
//
// The interior is solid and not the 50% dim, and that is worth saying because the dim
// was the first thing tried: the shipped splash art is a bright yellow wall, and
// cream text on a checkerboard of black and bright yellow is very nearly illegible --
// the yellow half-pixels are as light as the letters. A solid box also happens to be
// what the game's own furniture does. The scoreboard band is filled black and written
// in white and cream (render/scoreboard.go), so a black panel with cream text is the
// game's own idiom rather than an invention.
//
// The dim is still there, as a halo around the box: eight pixels of half-blacked
// artwork, which stops the frame sitting on the picture with a hard edge and is
// exactly the PenPat(gray)+PenMode(patOr) the original uses for its own shadows.
func panel(scr *render.Surface, r render.Rect) {
	scr.FillPatOrGray(render.SetRect(r.Left-8, r.Top-8, r.Right+8, r.Bottom+8), render.Black8)
	scr.Fill(r, render.Black8)
	scr.FrameRect(r, cream)
	inner := render.SetRect(r.Left+2, r.Top+2, r.Right-2, r.Bottom-2)
	scr.FrameRect(inner, render.DkGray8)
}

// shadow draws text with the original's one-pixel black drop shadow under it.
func shadow(scr *render.Surface, h, v int16, text string, idx uint8, scale int) {
	scr.DrawStringScaled(h+int16(scale), v+int16(scale), text, render.Black8, scale)
	scr.DrawStringScaled(h, v, text, idx, scale)
}

// center draws text centred on the screen's width.
func center(scr *render.Surface, v int16, text string, idx uint8, scale int) {
	centerIn(scr, 0, screenWide, v, text, idx, scale)
}

// centerIn draws text centred between two columns, for the parts of the screen that
// have to share it with a panel.
func centerIn(scr *render.Surface, left, right, v int16, text string, idx uint8, scale int) {
	h := left + (right-left-render.StringWidthScaled(text, scale))/2
	if h < left {
		h = left
	}
	shadow(scr, h, v, text, idx, scale)
}

// right draws unscaled text ending at h.
func right(scr *render.Surface, h, v int16, text string, idx uint8, scale int) {
	shadow(scr, h-render.StringWidthScaled(text, scale), v, text, idx, scale)
}

// rightScaled is the same without the shadow, for text on an inverse-video bar
// where a black shadow under black text would only thicken it.
func rightScaled(scr *render.Surface, h, v int16, text string, idx uint8, scale int) {
	scr.DrawStringScaled(h-render.StringWidthScaled(text, scale), v, text, idx, scale)
}

// fit shortens text until it is no wider than px, ending it in an ellipsis.
//
// It trims runes, not bytes, because a house name and an error message can both
// hold anything the file system will carry, and cutting a UTF-8 sequence in half
// would put a replacement character on screen.
func fit(text string, px int16, scale int) string {
	if px <= 0 {
		return ""
	}
	if render.StringWidthScaled(text, scale) <= px {
		return text
	}
	r := []rune(text)
	for len(r) > 1 {
		r = r[:len(r)-1]
		if render.StringWidthScaled(string(r)+"...", scale) <= px {
			return string(r) + "..."
		}
	}
	return ""
}
