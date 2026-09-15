package game

// GameOver.c: the two endings.
//
// Winning and losing are separate animations with almost nothing in common, and the
// only reason they share a file in the original is that they share the `pages[8]`
// array. Win: the screen goes to a dark starfield, the house's trailer message is
// written across it, and an angel flies in from the left trailing falling stars. Lose:
// eight sheets of paper flutter down over the room the player died in and stick,
// turning into the eight letters of GAME OVER.
//
// # Both are their own frame loop
//
// Neither runs inside the play loop. Each is a `while` of its own, pacing itself with
// the same two-tick budget RenderFrame uses and publishing through the same dirty-rect
// protocol -- AddRectTo*, then CopyRectsQD, then truncate both lists. So everything
// render_frame.go's comments say about the protocol applies here, with one difference
// worth reading before the code: **the loss animation calls CopyRectsQD inside its
// per-page loop** (GameOver.c:427), so it presents eight times per pass rather than
// once. That is the original, it is why the pages tear against each other on a slow
// machine, and it is transcribed because the alternative -- hoisting the call out of
// the loop -- changes what the animation looks like.
//
// The waiting is the host's, through World.Wait; see wait.go. A nil hook makes both
// animations run flat out and end, which is what lets a headless replay of a game that
// ends in death produce the same final screen a windowed one does.
//
// # What the endings do about the high score
//
// The C's two paths disagree, and docs/IMPROVEMENTS.md 2.60 sets out why and what to do
// about it. The win path asks TestHighScore whether the score qualified and skips the
// splash redraw when it did, because the board is already on the window; the loss path
// throws the same answer away and redraws unconditionally. **This port uses the win
// path's arrangement in both**, which is 2.60's recommendation and is invisible in this
// port either way -- see restoreSplashScreen, which explains why.
//
// # The two bugs that are not transcribed
//
// One out-of-bounds write is skipped rather than reproduced, in the star animation; see
// doGameOverStarAnimation. Everything else stays, including the width-for-height offset
// in initDiedGameOver, which is what gives the paper its run-up.

import (
	"glidergo/internal/game/player"
	"glidergo/internal/render"
)

// The four pictures and the page sheet's frame count (GameOver.c:18-22).
//
// kPagesMaskID is here for the reader rather than for the code: the mask is reached
// through render.Assets.MaskedPlate, whose uiMaskPairs table holds the 1990/1989
// pairing. It is the one mask in the game whose id is *lower* than its art's
// (docs/IMPROVEMENTS.md 2.59), so a call site that derived it by arithmetic would ask
// for 2990 and get nothing.
//
// None of the four is overridden by any shipped house -- measured across all twenty --
// so Plate and UI give the same answer for all of them today. They go through Plate
// anyway, because that is the rule for art drawn while a house is open and because a
// Stage 2 house is as entitled to its own ending as Teddy World is to its own pause
// placard.
const (
	kPageFrames     = 14
	kPagesPictID    = 1990
	kPagesMaskID    = 1989
	kLettersPictID  = 1988
	kMilkywayPictID = 1021

	// kSplash8BitPICT is GliderDefines.h:524, and the splash is the one picture in
	// this file that resolves through UI: it is the shell's own screen and a house
	// that could repaint it could hide the way out of itself. See render.Assets.UI.
	kSplash8BitPICT = 1000
)

// gameOverFill is the palette index DoGameOver floods the window with (GameOver.c:65,
// GameOver.c:81), written as a bare 244 in the C too.
//
// #000011: not quite black, and deliberately -- it is the night sky the milkyway
// picture fades into, so the flooded window and the picture's own edges match. Black8
// would show a seam wherever a dirty rect ended.
const gameOverFill = 244

// The four paper sounds and their priority (GliderDefines.h:103-106, :167).
//
// Priority 807 is the highest in the game -- nothing else is above 500 except the other
// interface sounds -- which is what stops the eight simultaneous pages from being
// dropped by a channel still busy with whatever killed the player.
const (
	Paper1Sound int16 = 48
	Paper2Sound int16 = 49
	Paper3Sound int16 = 50
	Paper4Sound int16 = 51

	PapersPriority int16 = 807
)

// page is GameOver.c:25-30's pageType: one falling thing.
//
// The C declares a single global `pages[8]` and both animations use it, which is why
// its fields read oddly together -- `stuck` and `counter` mean nothing to a falling
// star, and the star animation never touches them. Nothing carries over between the two
// endings (the win path assigns dest, was and frame for the five slots it uses before
// reading any of them), so each animation gets its own array here and the global goes
// away.
type page struct {
	dest, was      Rect
	frame, counter int16
	stuck          bool
}

// ---------------------------------------------------------------------------
// The win ending (GameOver.c:60-230)
// ---------------------------------------------------------------------------

// DoGameOver is GameOver.c:60-69: the player finished the house.
//
// `playing = false` is the first statement and is what ends the play loop -- without it
// the countdown block would re-run every frame for ever. Everything after it is the
// ending itself.
//
// The ColorRect floods the **whole window**, scoreboard band included, and it is the
// only whole-window write in the game. It happens after SetUpFinalScreen has composed
// the starfield into the offscreens, so the player is left looking at a dark screen with
// the trailer message hidden in it, and the animation reveals the message as the angel
// and the stars pass over it -- every dirty rect the animation publishes brings a piece
// of the starfield with it. That is the whole visual idea of the win screen, and it only
// works because the flood goes to the window and not to the work map.
func (w *World) DoGameOver() {
	w.Playing = false

	pages := w.setUpFinalScreen()

	w.Main.Fill(w.R.V.Screen, gameOverFill)
	// The C does not present: on a Mac the flood *is* the screen. The port has to say
	// so, and this is the one moment in the ending where the whole window changes.
	w.present()

	w.doGameOverStarAnimation(&pages)

	if !w.TestHighScore() {
		w.restoreSplashScreen()
	}
}

// setUpFinalScreen is GameOver.c:76-129: compose the starfield and the trailer, and seed
// the five falling stars.
//
// The picture is drawn into a 640x460 rect centred in the work map, which on the
// original's screen is the work map exactly -- the clamp `if (textDown < 0) textDown = 0`
// exists for a screen *narrower* than 640, where CenterRectInRect gives a negative top
// and the text would be written off the top of the map.
//
// The trailer is drawn twice, black one pixel down and right, then white on top: a drop
// shadow, and the only text in the game that has one. It is centred on the **screen**
// width rather than on the rect the picture went into, which is the same number here and
// would not be on a narrow screen.
//
// Then CopyRectWorkToBack promotes the whole composition into the background, which is
// what makes the animation's Back2Work restores put the starfield back rather than
// erasing to black.
func (w *World) setUpFinalScreen() [5]page {
	w.R.Work.Fill(w.R.V.WorkRect, gameOverFill)

	tempRect := render.CenterIn(render.SetRect(0, 0, 640, 460), w.R.V.WorkRect)
	loadScaledGraphic(w.R.Work, w.R.A.Plate(kMilkywayPictID), tempRect)

	textDown := tempRect.Top
	if textDown < 0 {
		textDown = 0
	}

	// GetLineOfText's `do { ... } while (subStr[0] > 0)` draws one extra empty line
	// past the end and stops; linesOfText drops it, along with the carriage return the
	// C leaves on the end of every other line. See linesOfText for why that is safe.
	//
	// TextWidth is measured over the string *including* that carriage return, so the
	// original's centring is a couple of pixels left of true on every line. This port
	// measures what it draws.
	for i, line := range linesOfText(w.H.Trailer.Text()) {
		offset := (w.R.V.Screen.Wide() - render.StringWidth(line)) / 2
		v := textDown + 32 + int16(i)*20
		w.R.Work.DrawString(offset+1, v+1, line, render.Black8)
		w.R.Work.DrawString(offset, v, line, render.White8)
	}

	w.CopyRectWorkToBack(w.R.V.WorkRect)

	// The five falling stars (GameOver.c:120-128), and two things about this loop:
	//
	// **Only the frame survives.** dest and was are both overwritten before anything
	// reads them -- the animation reseeds a slot's dest at the moment it first draws
	// it, and computes was from dest every pass -- so the positions seeded here are
	// dead. frame is not: the animation advances it but never initialises it, so these
	// are the phases the five stars start spinning at.
	//
	// **The three RandomInt calls per star stay anyway**, dead or not. They draw from
	// the same stream every flame phase and pendulum start comes out of, so removing
	// the two useless ones would shift every random value drawn afterwards and change
	// what the next game looks like. rand.go's file comment is the long version.
	//
	// The C's two dead calls sit in different arguments of one QOffsetRect, where the
	// evaluation order is unspecified; Go's is left to right. It cannot matter here --
	// both values are discarded and the count is what the stream cares about -- but it
	// would matter if either were ever read, so it is written down.
	var pages [5]page
	r := w.R.V.WorkRect
	for i := 0; i < 5; i++ {
		pages[i].dest = render.Offset(render.StarSrc[0],
			r.Right+w.RandomInt(r.Right/5)+(r.Right/4)*int16(i),
			w.RandomInt(r.Bottom)-r.Bottom/2)
		pages[i].was = pages[i].dest
		pages[i].frame = w.RandomInt(6)
	}
	return pages
}

// kStarFalls is GameOver.c:139: a falling star drops eight pixels a pass.
const kStarFalls = 8

// doGameOverStarAnimation is GameOver.c:136-230: the angel crosses the screen from left
// to right, dropping a star every 32 pixels, and 80 passes after it leaves the animation
// ends.
//
// The angel is 96x44 and starts entirely off the left edge, at left = -96. It advances
// two pixels a pass and is drawn for as long as `left <= workSrcRect.right + 2`, so it is
// drawn on (642 + 96) / 2 + 1 = 370 passes -- twelve seconds at two ticks a pass.
// `pass` is reset to zero on every pass the angel is drawn, so the 80-pass countdown only
// starts once it has gone: about thirteen and a half seconds in total, and then a five
// second wait for the player.
//
// A star is (re)seeded whenever the angel's left edge is a multiple of 32, at the angel's
// bottom-left corner, cycling through the five slots. Each is drawn every pass from then
// on, spinning through the six frames of the bonus sheet's star and falling eight pixels
// until it reaches the bottom of the work map, where it stops and stays -- so the screen
// ends with five stars parked along the bottom edge and a trail of revealed starfield
// behind each one.
//
// **The negative-slot write is not reproduced.** `which = angelDest.left / 32 % 5` is -3,
// -2 and then -1 for the first three seedings, because the angel starts at -96, and the C
// writes `pages[-3]`, `pages[-2]`, `pages[-1]` -- three 24-byte structures before the
// start of the array, which on a 68k Mac landed in whatever global followed it. Nothing
// reads them: `count` stays 0 because `count < which + 1` is false for a negative which,
// so no star is drawn until the angel reaches left = 0. Skipping the seeding when which is
// negative is therefore observationally identical, minus the corruption. The sound is
// *not* skipped -- the C plays kMysticSound on all three -- so the ending opens with three
// chimes and no visible star, exactly as the original does.
func (w *World) doGameOverStarAnimation(pages *[5]page) {
	stars := w.R.A.Sheet("bonus")
	angel := w.R.A.Strip("angel")

	angelSrc := render.SetRect(0, 0, 96, 44)
	angelDest := render.Offset(angelSrc, -96, 0)

	noInteruption := true
	nextLoop := w.Ticks() + 2
	count := 0
	pass := 0

	// The leading flush is the animation's and not the wait's: a key pressed during
	// the frame that finished the house would otherwise still be in the queue and
	// would abort the ending before its first pass was drawn.
	w.FlushEvents()

	for noInteruption {
		if angelDest.Left%32 == 0 {
			w.PlayPrioritySound(MysticSound, MysticPriority)
			which := angelDest.Left / 32 % 5
			if which >= 0 {
				pages[which].dest = render.Offset(
					render.ZeroCorner(pages[which].dest), angelDest.Left, angelDest.Bottom)
				if int(which)+1 > count {
					count = int(which) + 1
				}
			}
		}

		for i := 0; i < count; i++ {
			p := &pages[i]

			p.frame++
			if p.frame >= 6 {
				p.frame = 0
			}
			if stars != nil {
				w.R.Work.Copy(stars, render.StarSrc[p.frame], p.dest, render.Masked)
			}

			// was is this pass's rect grown upwards by the fall distance, which is
			// the union of where the star is and where it was. Both animations do
			// their own union like this rather than calling QUnionSimilarRect; the
			// star's movement is one-directional, so a subtraction does it.
			p.was = p.dest
			p.was.Top -= kStarFalls

			w.AddRectToWorkRectsWhole(player.Rect(p.was))
			w.AddRectToBackRects(player.Rect(p.dest))

			if p.dest.Top < w.R.V.WorkRect.Bottom {
				p.dest = render.Offset(p.dest, 0, kStarFalls)
			}
		}

		if angelDest.Left <= w.R.V.WorkRect.Right+2 {
			if angel != nil {
				w.R.Work.Copy(angel, angelSrc, angelDest, render.Masked)
			}
			// The published rect is two pixels wider on the left than the drawn one,
			// covering the column the angel has just left, and the restored rect is
			// not -- so the background comes back under the angel's current position
			// only, and the trailing column keeps whatever the angel put there. That
			// is how the angel leaves a smear of revealed starfield behind it.
			angelDest.Left -= 2
			w.AddRectToWorkRectsWhole(player.Rect(angelDest))
			angelDest.Left += 2
			w.AddRectToBackRects(player.Rect(angelDest))

			angelDest = render.Offset(angelDest, 2, 0)
			pass = 0
		}

		w.CopyRectsQD()
		w.Work2Main = w.Work2Main[:0]
		w.Back2Work = w.Back2Work[:0]

		// The two-tick budget, as a remaining count. See pollInput.
		if w.pollInput(nextLoop-w.Ticks()) || w.Quitting {
			noInteruption = false
		}
		nextLoop = w.Ticks() + 2

		if pass < 80 {
			pass++
		} else {
			// Five seconds to look at the finished screen, or until the player
			// touches something. The answer is discarded -- unlike
			// DisplayStarsRemaining, this caller does not care whether a resume
			// ended it, because TestHighScore is about to repaint the window
			// anyway.
			w.WaitForInputEvent(5)
			noInteruption = false
		}
	}
}

// ---------------------------------------------------------------------------
// The loss ending (GameOver.c:249-506)
// ---------------------------------------------------------------------------

// pagesAnim is the state InitDiedGameOver builds and HandlePages and DrawPages share.
//
// In the C all of it is file-scope globals -- `pages`, `pagesStuck`, `stopPages`,
// `pageSrc`, `lettersSrc`, `roomRgn` and three GWorlds. Gathering them into one
// structure is not a rearrangement of the animation: the three functions are called only
// by DoDiedGameOver, in one order, and nothing else in the game reads any of it.
type pagesAnim struct {
	pages [8]page

	// pageSrc is the fourteen 32x32 cells of the page sheet, and lettersSrc the eight
	// 25x32 cells of the letters. Both are `QSetRect` then `QOffsetRect(0, 32*i)` down
	// a one-cell-wide picture (GameOver.c:275-279, :301-305), so the only thing the
	// tables really hold is a stride -- they are written out because the C writes them
	// out and because the letters' width (25) is not their stride (32).
	pageSrc    [kPageFrames]Rect
	lettersSrc [8]Rect

	// art is PICT 1990 with its 1989 mask applied and letters is PICT 1988. Either may
	// be nil against a checkout with no extracted art, in which case the animation
	// still runs and still ends; see render.Assets.Plate.
	art, letters *render.Surface

	// clip is roomRgn: RectRgn(justRoomsRect), a one-rect region, used for the letters
	// and not for the pages. See drawPages.
	clip []Rect

	// stuck is pagesStuck, the loop's own termination count, and stopPages the row a
	// page sticks at.
	stuck     int
	stopPages int16
}

// DoDiedGameOver is GameOver.c:443-506: the player ran out of gliders.
//
// The two CopyRectMain* grabs at the head are the reason this ending looks the way it
// does: the room the player died in is on the *window* and only on the window, because
// the play loop's last act was to publish it, and reading it back into both offscreens
// is what lets the pages flutter over it and be erased from it. It is the one place in
// the game where reading the frame buffer back is not avoidable.
//
// `pagesStuck = 8` is how the abort works: the player pressing anything sets it, the
// loop ends, and the letters are never drawn. So an impatient player never sees GAME
// OVER, which is either a bug or an economy depending on how impatient they are.
//
// The tail's three differences from the win path are all transcribed:
//
//   - **A demo waits one second where a player waits ten**, and neither waits at all if
//     they aborted.
//   - **A demo does not touch the high-score board.** The attract mode plays itself, and
//     an unattended machine that could post scores would fill the board with them.
//   - `playing = false` is here at the end rather than at the top, unlike DoGameOver.
//     It makes no difference -- nothing between the two points reads it -- and it is
//     left where the C has it.
func (w *World) DoDiedGameOver() {
	userAborted := false
	a := w.initDiedGameOver()

	w.CopyRectMainToWork(w.R.V.WorkRect)
	w.CopyRectMainToBack(w.R.V.WorkRect)
	w.FlushEvents()

	nextLoop := w.Ticks() + 2
	for a.stuck < 8 {
		a.handlePages(w)
		a.drawPages(w)

		if w.pollInput(nextLoop-w.Ticks()) || w.Quitting {
			a.stuck = 8
			userAborted = true
		}
		nextLoop = w.Ticks() + 2
	}

	w.Playing = false

	if w.DemoGoing {
		if !userAborted {
			w.WaitForInputEvent(1)
		}
		w.restoreSplashScreen()
	} else {
		if !userAborted {
			w.WaitForInputEvent(10)
		}
		// docs/IMPROVEMENTS.md 2.60: the C ignores this answer and redraws
		// regardless. The win path's arrangement is the considered one and this is
		// it.
		if !w.TestHighScore() {
			w.restoreSplashScreen()
		}
	}
}

// initDiedGameOver is GameOver.c:249-310.
//
// The eight pages start in a row 40 pixels apart, centred on the screen and then moved
// up and to the left by *half the screen's width in both axes*:
//
//	QOffsetRect(&pages[i].dest, (thisMac.screen.right - thisMac.screen.left) / -2,
//			(thisMac.screen.right - thisMac.screen.left) / -2);
//
// **The second argument should be the height.** On the 640x480 screen the game was drawn
// for that puts the row at v = -96 instead of v = -16, which is 80 rows further off the
// top -- so the pages spend their first seconds falling into view instead of starting
// already visible. The animation is built around that: `stopPages` is only 224, so
// without the extra run-up the paper would stick almost immediately. Transcribed as
// written, and it is the reason a port must not "fix" it.
//
// The parity nudge two lines later is dead. `dest.left % 2 == 1` is testing for an odd
// left edge, and every value it can see is even: 40 and the half-width are both even and
// the centred rect starts at an even column on any even-width screen. On an odd-width
// screen the left edge can be odd *and negative*, and C's % keeps the sign of the
// dividend, so the test would still fail. Kept because it costs nothing and because a
// reader who finds it in the C should find it here too.
func (w *World) initDiedGameOver() *pagesAnim {
	const kPageSpacing = 40

	a := &pagesAnim{
		letters:   w.R.A.Plate(kLettersPictID),
		art:       w.R.A.MaskedPlate(kPagesPictID),
		clip:      []Rect{w.R.V.JustRoomsRect},
		stopPages: w.R.V.Screen.Tall()/2 - 16,
	}

	for i := range a.pageSrc {
		a.pageSrc[i] = render.Offset(render.SetRect(0, 0, 32, 32), 0, 32*int16(i))
	}
	for i := range a.lettersSrc {
		a.lettersSrc[i] = render.Offset(render.SetRect(0, 0, 25, 32), 0, 32*int16(i))
	}

	screen := w.R.V.Screen
	for i := range a.pages {
		dest := render.CenterIn(render.SetRect(0, 0, 32, 32), screen)
		dest = render.Offset(dest, -screen.Left, -screen.Top)
		if i < 4 {
			dest = render.Offset(dest, -kPageSpacing*int16(4-i), 0)
		} else {
			dest = render.Offset(dest, kPageSpacing*int16(i-3), 0)
		}
		dest = render.Offset(dest, screen.Wide()/-2, screen.Wide()/-2)
		if dest.Left%2 == 1 {
			dest = render.Offset(dest, 1, 0)
		}

		a.pages[i].dest = dest
		a.pages[i].was = dest
		a.pages[i].frame = 0
		a.pages[i].counter = w.RandomInt(32)
		a.pages[i].stuck = false
	}

	return a
}

// handlePages is GameOver.c:316-394: advance the eight sheets of paper.
//
// A page sticks when its bottom edge plus a random 0..7 passes stopPages, which is why
// the eight do not all stick on the same pass at the same row. **The RandomInt is drawn
// for every page on every pass, stuck or not** -- it is inside the loop and before the
// test -- so the number of values this animation takes out of the stream is fixed by its
// length rather than by what the paper does. That matters only for 1.8's replays, and it
// is another reason not to restructure the loop.
//
// The flutter is a fourteen-frame machine driven by `frame` and `counter`, and it is
// easier to read as a cycle than as a switch:
//
//	frame 0        counter passes of nothing, then frame 1
//	frames 1-4     nothing (the sheet is edge-on and does not move)
//	frame 5        +6,+6
//	frames 6,7     +8,+8 each, and frame 7 draws a new counter of 4..7
//	frame 7 held   +10,+10 per pass until the counter runs out -- the long glide
//	frame 8        nothing, and a paper3/paper4 rustle
//	frame 9        +8,+8
//	frame 10       +6,+6
//	frames 11-13   nothing
//	frame 14       +8,0, a paper1/paper2 rustle, and back to frame 0 with counter 8..15
//
// So a page falls 66 to 96 rows per cycle and drifts eight pixels further right than it
// falls, and the random counters are what stagger the eight of them. From v = -96 to
// stopPages = 224 is four cycles, about 120 passes, four seconds.
//
// **`case 8` is unreachable.** frame is set to 8 by the frame-7 branch, and the next
// pass takes the `else` and increments it to 9 before the switch sees it -- so the switch
// can only be entered with 5, 6, 7, 9, 10, 11, 12, 13 or 14. The case is written out
// because the C writes it, and it is worth knowing that its (8,8) offset never happens:
// the movement above is the movement the animation actually has.
func (a *pagesAnim) handlePages(w *World) {
	for i := range a.pages {
		p := &a.pages[i]

		if p.dest.Bottom+w.RandomInt(8) > a.stopPages {
			p.frame = 0
			if !p.stuck {
				// The page becomes a letter, and a letter is 25 wide where a page
				// is 32. Only the right edge moves, so the letter is aligned with
				// the left edge of the sheet it came from.
				p.dest.Right = p.dest.Left + 25
				p.stuck = true
				a.stuck++
			}
			continue
		}

		switch {
		case p.frame == 0:
			p.counter--
			if p.counter <= 0 {
				p.frame = 1
			}

		case p.frame == 7:
			p.counter--
			if p.counter <= 0 {
				p.frame = 8
				if w.RandomInt(2) == 0 {
					w.PlayPrioritySound(Paper3Sound, PapersPriority)
				} else {
					w.PlayPrioritySound(Paper4Sound, PapersPriority)
				}
			} else {
				p.dest = render.Offset(p.dest, 10, 10)
			}

		default:
			p.frame++
			switch p.frame {
			case 5:
				p.dest = render.Offset(p.dest, 6, 6)
			case 6:
				p.dest = render.Offset(p.dest, 8, 8)
			case 7:
				p.dest = render.Offset(p.dest, 8, 8)
				p.counter = w.RandomInt(4) + 4
			case 8, 9:
				p.dest = render.Offset(p.dest, 8, 8)
			case 10:
				p.dest = render.Offset(p.dest, 6, 6)
			case kPageFrames:
				p.dest = render.Offset(p.dest, 8, 0)
				p.frame = 0
				p.counter = w.RandomInt(8) + 8
				if w.RandomInt(2) == 0 {
					w.PlayPrioritySound(Paper1Sound, PapersPriority)
				} else {
					w.PlayPrioritySound(Paper2Sound, PapersPriority)
				}
			}
		}
	}
}

// drawPages is GameOver.c:401-435, and it is the one place in the game that presents
// inside a loop.
//
// Each page is drawn, its rect and its previous rect are published, the background is
// queued to come back under it -- and then CopyRectsQD runs, for that page alone. So a
// pass over the eight is eight presents, and the first page is on the screen before the
// eighth has been drawn. Hoisting the call out of the loop would be one line and would
// halve the work; it would also change the animation, because a torn frame is what this
// looks like on a machine slower than the animation wants.
//
// The letters are clipped to justRoomsRect and the pages are not. The region matters not
// at all on the original's screen -- justRoomsRect *is* the work map there, and a page
// sticks at row 224 -- but it is what the C asks for and CopyClipped is a one-rect
// region, so it costs a slice to say so. The pages' own draw needs no clip because
// CopyMask clipped to the GWorld's bounds, which is what Surface.Copy does.
//
// The letters go on with srcCopy, not a mask: the eight cells of PICT 1988 are opaque,
// so each letter arrives with its own white background and covers the page that was
// there. Nothing erases the letter afterwards -- it is published, and the next pass
// publishes it again from the same unchanged work map -- so the word builds up as the
// paper reaches the floor.
func (a *pagesAnim) drawPages(w *World) {
	for i := range a.pages {
		p := &a.pages[i]

		if p.stuck {
			if a.letters != nil {
				w.R.Work.CopyClipped(a.letters, a.lettersSrc[i], p.dest, render.SrcCopy, a.clip)
			}
		} else if a.art != nil {
			w.R.Work.Copy(a.art, a.pageSrc[p.frame], p.dest, render.Masked)
		}

		p.was = render.UnionSimilar(p.dest, p.was)
		w.AddRectToWorkRects(player.Rect(p.was))
		w.AddRectToBackRects(player.Rect(p.dest))

		w.CopyRectsQD()
		w.Work2Main = w.Work2Main[:0]
		w.Back2Work = w.Back2Work[:0]

		p.was = p.dest
	}
}

// ---------------------------------------------------------------------------
// The splash screen (Play.c:257-273, MainWindow.c:98-114)
// ---------------------------------------------------------------------------

// restoreSplashScreen puts the splash picture back into the work map on the way out of a
// game.
//
// It stands for two C functions, and they are not the same function:
//
//	Play.c:257-273      the tail of NewGame, for a game the player quit: invalidate the
//	                    window, paint the work map, scale the splash into it.
//	MainWindow.c:98-114 RedrawSplashScreen, called by both endings: the same paint and
//	                    scale, plus DrawOnSplash's house name, and then
//	                    `CopyRectMainToWork(&workSrcRect)`.
//
// **That last line is backwards.** It copies the window over the work map, discarding the
// splash screen the function has just composed, where CopyRectWorkToMain would have shown
// it. docs/analysis/scoring.md 7.8 and docs/IMPROVEMENTS.md 2.60 have the consequence: on
// a real Mac neither ending puts the splash screen back at all, and the window keeps
// showing the fluttering letters or the starfield until the next update event repaints
// it. It also hides the other bug -- the loss path's unconditional redraw over a
// high-score board the player just earned -- because there is no redraw.
//
// This port has neither problem and does not need either arrangement, which is why one
// function serves both call sites -- and why there are only two of them. The title screen
// is internal/shell's, drawn on the shell's own surface and presented on the pass after a
// game returns (shell.go's Run), so a work map painted on the way out of a game is not
// visible to anybody: NewGame's tail does not call this at all, for the reasons set out
// there and in docs/IMPROVEMENTS.md 2.62.
//
// The callers that remain are the two endings, where the splash is on screen a moment
// later because the starfield is composed over it and presented. What is left of the C
// here is that part: the animation wants the title screen behind it rather than the last
// frame of a dead glider.
func (w *World) restoreSplashScreen() {
	w.R.Work.Fill(w.R.V.WorkRect, render.Black8)

	// splashOriginH/V (MainWindow.c:238-243), clamped at zero: the picture is 640x460
	// and is centred on a larger screen rather than scaled up to it. Both are 0 on the
	// 640x480 screen this port draws, and the arithmetic is kept because the port's
	// View is built for a screen size rather than for a constant.
	originH := (w.R.V.Screen.Wide() - 640) / 2
	if originH < 0 {
		originH = 0
	}
	originV := (w.R.V.Screen.Tall() - 480) / 2
	if originV < 0 {
		originV = 0
	}

	loadScaledGraphic(w.R.Work, w.R.A.UI(kSplash8BitPICT),
		render.Offset(render.SetRect(0, 0, 640, 460), originH, originV))
}
