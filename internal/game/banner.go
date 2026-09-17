package game

// Banner.c: the two things the game says to the player over a running house -- the author's
// message on a sheet of notebook paper, and "N stars to go".
//
// # The banner was never visible
//
// `BringUpBanner` (Banner.c:171-197) draws the page and the message into **workSrcMap**,
// then copies the clean background back over the same rect, then waits fifteen seconds. Two
// `DissBits(&justRoomsRect)` calls that would have put it on screen are commented out in the
// shipped source, above and below the wait, and nothing else blits it. So in the shipped
// build the whole banner is composed into an offscreen, erased from that offscreen, and the
// player looks at a room for fifteen seconds -- which is why every review of Glider PRO
// describes starting a house as a long pause. docs/analysis/rendering.md 15.5 works through
// it and states the porter's instruction: a port that wants the banner must blit the page
// rect to the screen between the draw and the wait.
//
// **This port does that**, in the one line the original is missing, and says so here for the
// same reason `internal/scores` does: the difference is visible, the fifteen seconds are
// otherwise inexplicable, and a reader comparing the two files needs to be told which line
// is new. It is drawn from the work map rather than presented directly, so the erase that
// follows the wait is still the only cleanup and DumpScreenOn -- which NewGame calls next --
// still publishes the un-bannered room.
//
// # The wait is a hook
//
// Both functions block, and neither may block here: docs/IMPROVEMENTS.md 2.32. The durations
// stay (they are what the screens are for) and the waiting is the host's, through
// World.Wait. See wait.go.
//
// A consequence worth stating: DisplayStarsRemaining has a second caller, HandleRewards' star
// arm (rewards.go, Interactions.c:946), which reaches it from the *middle* of a frame. So
// touching a star stops the game for a second and any key held at that moment is thrown away
// by the wait -- a player walking right when they collect a star stops walking. That is 1994
// behaviour, it is bad, and 2.32 argues it cannot be corrected inside Stage 1 without
// breaking the demo comparison 1.8 is built around. It is the reason the second caller is
// called out in a comment there as well as here.

import (
	"strconv"
	"strings"

	"github.com/bwenstar/gliderGo/internal/game/player"
	"github.com/bwenstar/gliderGo/internal/render"
)

// The five pictures (Banner.c:17-21) and the page's measurements (:51, :58, :64).
//
// The page is one picture split in two: 1993 is the top 190 rows, and 1992 -- masked by 1991
// -- is the bottom 30, which is the torn edge of the sheet. The split exists because only
// the torn part needs a mask, and CopyMask on a Mac was not free.
//
// All three sizes match their pictures' picFrame exactly, so the scaled draws never scale.
// Thirteen of the twenty shipped houses carry their own 1991-1993 and four their own 1017 or
// 1018, which is why every one of them is resolved through Plate rather than UI; see
// render.Assets.Plate.
const (
	kBannerPageTopPICT    = 1993
	kBannerPageBottomPICT = 1992
	kBannerPageBottomMask = 1991
	kStarsRemainingPICT   = 1017
	kStarRemainingPICT    = 1018

	kBannerPageWide   = 330
	kBannerPageTall   = 220
	kBannerPageTopUse = 190 // rows of the page that come from 1993
	kBannerPageBotUse = 30  // and from 1992/1991

	// The message's layout inside the page (Banner.c:136, :158, :162). The first line's
	// baseline is 32 rows down, lines are 20 apart, and the star count sits at a fixed
	// 164 and 180 -- 16 apart, not 20, which is the original's own inconsistency and is
	// kept because the plate's ruled lines are what the numbers were measured against.
	kBannerTextIndent = 16
	kBannerTextFirst  = 32
	kBannerLineHeight = 20
	kBannerStarsLine  = 164
	kBannerHintLine   = 180

	// The stars-remaining panel (Banner.c:150-152, :170-171): 256x64, centred on the
	// screen and then lifted 20 rows, with the count drawn 102 pixels in and 23 down.
	kStarsPanelWide = 256
	kStarsPanelTall = 64
	kStarsPanelLift = 20
	kStarsCountH    = 102
	kStarsCountV    = 23
)

// The five strings of STR# 150 the banner uses, transcribed from
// `GliderPRO/Glider PRO.r`:396 -- indices 1 to 5 of the fifty-one localized strings.
//
// They are constants for the reason internal/scores gives for indices 6 to 8: this port has
// no localized-string mechanism, adding one for five strings would be a resource loader
// nobody else calls, and the strings themselves are part of what the screen *is*. The
// singular/plural pair is the original's, including its choice to spell the count out as a
// number in both cases.
const (
	strThereAre     = "There are "
	strThereIs      = "There is "
	strStarsInHouse = " stars in the house."
	strStarInHouse  = " star in the house."
	strGetEveryStar = "Get every star to win."
)

// ---------------------------------------------------------------------------
// BringUpBanner (Banner.c:171-197)
// ---------------------------------------------------------------------------

// BringUpBanner shows the author's message and waits.
//
// NewGame's ladder calls this before InitGarbageRects rather than after, because it blocks;
// see the note there. Fifteen seconds, or four in a demo -- the demo has to get on with
// playing itself, and the attract mode is the one caller for which a fifteen-second still
// frame would be a bug rather than a pause.
func (w *World) BringUpBanner() {
	page := w.DrawBanner()
	w.DrawBannerMessage(page)

	// The line the original is missing (see the file comment). It goes here, after the
	// message and before the erase, because this is where the C's two commented-out
	// DissBits calls were.
	w.Main.Copy(w.R.Work, page, page, render.SrcCopy)
	w.present()

	// CopyBits(backSrcMap -> workSrcMap, wholePage, ...): the erase. The work map holds
	// the composed room under the page, so the clean background is all the restore needs
	// -- the same trick DoPause's restore uses in the other direction.
	w.CopyRectBackToWork(page)

	if w.DemoGoing {
		w.WaitForInputEvent(4)
	} else {
		w.WaitForInputEvent(15)
	}
}

// DrawBanner is Banner.c:39-84: compose the sheet of notebook paper into the work map, and
// answer where it landed.
//
// The C hands back a Point through an out-parameter and its callers immediately rebuild the
// 330x220 rect from it (Banner.c:180-181); the rect is returned directly here instead,
// because the Point is only ever used to reconstruct it.
//
// The two temporary GWorlds the C creates for the torn edge and its mask are what
// render.Assets.MaskedPlate does, and it does it better: a house may override the art
// without the mask or the mask without the art, and pairing them at load time would lose
// that. See its comment for the cross-tab that proves the mask's polarity.
func (w *World) DrawBanner() Rect {
	// CenterRectInRect(&wholePage, &mapBounds) where mapBounds is ZeroRectCorner of the
	// screen. The port's Screen is already cornered at the origin (render.NewView), so
	// the ZeroRectCorner is a no-op and is kept for the same reason the C has it: on a
	// Mac the screen rect could start anywhere.
	page := render.CenterIn(
		render.SetRect(0, 0, kBannerPageWide, kBannerPageTall),
		render.ZeroCorner(w.R.V.Screen))

	top := page
	top.Bottom = top.Top + kBannerPageTopUse
	loadScaledGraphic(w.R.Work, w.R.A.Plate(kBannerPageTopPICT), top)

	bottom := page
	bottom.Top = bottom.Bottom - kBannerPageBotUse
	if art := w.R.A.MaskedPlate(kBannerPageBottomPICT); art != nil {
		w.R.Work.Copy(art, art.Bounds(), bottom, render.Masked)
	}

	return page
}

// DrawBannerMessage is Banner.c:117-166: the house's own message, and the star count under
// it.
//
// Three things about it are worth knowing before reading the code:
//
// **The message is the house author's, and it is drawn line by line** -- the format stores
// one Pascal string with carriage returns in it (see linesOfText). Nothing wraps it: a line
// too long for the page runs off the right-hand edge, in the original and here. The longest
// in the shipped corpus is 44 characters, and the page holds 49 of this port's font.
//
// **The star count is opt-in per house and the flag is inverted**: it shows when bit 2 of the
// house flags is *clear* (house.BannerStarCount, HouseIO.c:418). Eight of the twenty shipped
// houses have it on.
//
// **It is drawn in red** -- `ForeColor(redColor)`, the classic QuickDraw constant rather than
// a palette index, which is why render.QDRed exists.
func (w *World) DrawBannerMessage(page Rect) {
	// TextFont(applFont); TextFace(bold); TextSize(12). The port has one font at one size
	// and the shell magnifies it when it needs to; here it does not, because 12-point
	// Geneva bold and this font's 6-pixel cell are close enough that the measured lines
	// still fit the page.
	for i, line := range linesOfText(w.H.Banner.Text()) {
		w.R.Work.DrawString(
			page.Left+kBannerTextIndent,
			page.Top+kBannerTextFirst+int16(i)*kBannerLineHeight,
			line, render.Black8)
	}

	if !w.H.BannerStarCount() {
		return
	}

	// `if (numStarsRemaining != 1)` picks the plural, so a house entered with no stars
	// left at all reads "There are 0 stars in the house." That is the original's answer
	// and it is reachable: a resumed saved game whose last star was taken comes through
	// NewGame's other arm, but a house with no kStar in it at all comes through this one.
	line := strThereAre + strconv.Itoa(int(w.StarsLeft)) + strStarsInHouse
	if w.StarsLeft == 1 {
		line = strThereIs + strconv.Itoa(int(w.StarsLeft)) + strStarInHouse
	}
	w.R.Work.DrawString(page.Left+kBannerTextIndent, page.Top+kBannerStarsLine, line, render.QDRed)
	w.R.Work.DrawString(page.Left+kBannerTextIndent, page.Top+kBannerHintLine, strGetEveryStar, render.QDRed)
}

// linesOfText is what GetLineOfText (StringUtils.c:140-208) should have been.
//
// docs/analysis/ui-dialogs.md P16 says not to translate that function literally, and it is
// right. It takes a string and an index and returns the index-th carriage-return-delimited
// line, by walking the string from the start every time -- so drawing an n-line message is
// O(n²) -- and **it includes the terminating carriage return in the line it returns**, which
// on a Mac draws whatever glyph Geneva has at 0x0D at the end of every line but the last.
// Its two callers then loop `do { ... } while (subStr[0] > 0)`, which draws one extra empty
// line past the end of the message and stops.
//
// This returns the lines, without their separators, in one pass. The empty trailing line is
// gone with them: it drew no pixels, so nothing moves.
//
// A line feed is accepted as a separator as well, which the original could not produce.
// Every banner and trailer in the shipped corpus uses carriage returns only -- verified
// across all twenty-two houses -- so this cannot change what any of them look like; it is
// there because Stage 2's houses are authored in a text file that a person edits with a
// modern editor, and a message typed with the wrong line ending should not run off the page
// as one line.
func linesOfText(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Split(s, "\n")
}

// ---------------------------------------------------------------------------
// DisplayStarsRemaining (Banner.c:205-243)
// ---------------------------------------------------------------------------

// DisplayStarsRemaining puts up the small "N stars to go" panel, holds it for a second, and
// then lets the player dismiss it.
//
// It draws into **Main** and not into the work map, which is Banner.c:210's
// SetPortWindowPort(mainWindow) and is what makes the tail's CopyRectWorkToMain a restore
// rather than a redraw -- the work map still holds the room the panel is covering. Same
// arrangement as the pause placard.
//
// Two oddities of the original are transcribed and one is not:
//
//   - **The test is `< 2`, not `== 1`.** So zero stars remaining draws the singular plate
//     ("Last star!"), which is right for the way the second caller uses it -- rewards.go
//     calls this only when stars are left -- and would be wrong for a house with none.
//   - **The count is drawn only in the plural branch.** The singular plate says "one" in
//     words, so there is no number to place on it; the C computes the string with
//     NumToString before the branch and then uses it in one arm.
//   - `src = bounds; InsetRect(&src, 64, 32)` (Banner.c:154-155) is computed and never
//     read. It is not transcribed, because there is nothing to transcribe: it names no
//     picture and reaches no draw. Recorded here so a reader diffing the two files does
//     not go looking for it.
//
// The wait is `DelayTicks(60)` and then up to thirty seconds -- **seconds, not ticks**:
// WaitForInputEvent's parameter is multiplied by 60 (Utilities.c:445). So the panel is
// unconditionally up for one second and then stays until the player touches something. A
// resume ends it too, and that is the one caller in the game that reads what the wait
// returned; see RestoreEntireGameScreen for why the answer is a screen rebuild.
func (w *World) DisplayStarsRemaining() {
	bounds := render.CenterIn(
		render.SetRect(0, 0, kStarsPanelWide, kStarsPanelTall), w.R.V.Screen)
	// QOffsetRect(&bounds, -thisMac.screen.left, -thisMac.screen.top): a no-op here, as
	// in DrawBanner, because the port's screen rect is cornered at the origin.
	bounds = render.Offset(bounds, 0, -kStarsPanelLift)

	if w.StarsLeft < 2 {
		loadScaledGraphic(w.Main, w.R.A.Plate(kStarRemainingPICT), bounds)
	} else {
		loadScaledGraphic(w.Main, w.R.A.Plate(kStarsRemainingPICT), bounds)
		count := strconv.Itoa(int(w.StarsLeft))
		// ColorText(theStr, 4L) is a palette index and not a QuickDraw colour: entry 4
		// is #FFFF33, the pale yellow the plate's lettering is drawn in.
		w.Main.DrawString(
			bounds.Left+kStarsCountH-render.StringWidth(count)/2,
			bounds.Top+kStarsCountV, count, 4)
	}
	// The C does not present, because on a Mac the window *is* the frame buffer and the
	// draws above are already visible. Everywhere this port writes Main it has to say so.
	w.present()

	w.DelayTicks(60)
	if w.WaitForInputEvent(30) {
		w.RestoreEntireGameScreen()
	}
	w.CopyRectWorkToMain(player.Rect(bounds))
	w.present()
}
