package game

// The pause placard: Input.c's DoPause, split at the line between the game and its host.
//
// # What the original does
//
// DoPause (Input.c:77-117) draws a 214x54 picture centred in houseRect, spins on GetKeys
// until the pause key is released, spins again until it is pressed, copies the placard's
// rect back from the work map, and spins a third time until the key is released again. It
// is called from inside GetInput, so a paused game does not advance a frame -- the pause is
// deep inside the simulation rather than around it, which is why it needs nothing else:
// no state, no flag the frame loop reads, no second code path.
//
// # What a port can keep
//
// Everything except the spinning. A windowed program that stops answering the compositor
// is a program the compositor reports as hung, so the three GetKeys loops become the host's
// business (World.Pause) and everything else stays here: the rect, the two pictures, the
// choice between them, and the restore from the work map. That split is the same one
// World.PlayEvent draws, for the same reason -- what differs per platform is how you wait,
// and what must not differ is what the player sees.
//
// The three loops collapse into one rule the host can state in edge terms: **the pause key
// acts on its press edge and a held key does nothing.** Loop (A) is "not the press that
// paused", loop (B) is "the next press resumes", and loop (C) is "not the press that
// resumed". docs/analysis/input.md 10.3 has the reason this matters beyond debouncing: the
// C's theKeys is a shared global that DoPause itself overwrites, so after a pause the
// second glider's GetInput no longer sees the pause bit and cannot pause again in the same
// frame. A host that reports the key level-triggered instead would pause, resume and pause
// again on one press, which looks exactly like a pause key that does not work.
//
// # The other pause
//
// There is a second way a game stops: it loses the foreground, and PlayGame's pump spins on
// switchedOut until it comes back. The C draws nothing for that one, which is what
// docs/IMPROVEMENTS.md 2.28 is about, so pumpWhileSwitchedOut lives here too and borrows the
// placard's rect and its no-art panel. It borrows neither picture, because both of them name
// a key and clicking the window is not one.
//
// The one visible difference is loop (C)'s ordering. The C erases the placard *before* it
// waits for the release, so holding the pause key down after unpausing freezes the game
// with the placard already gone (docs/analysis/input.md 10.2). Here the placard stays up
// until the pause really ends, because the erase is after the wait -- a held key shows a
// paused game rather than a frozen one. See docs/IMPROVEMENTS.md 2.5.

import "glidergo/internal/render"

// The placard: the two pictures (Input.c:17-18) and the rect they are drawn in (:82).
//
// The size is hard-coded in the C and matches both pictures' picFrame exactly, so
// LoadScaledGraphic never actually scales -- and neither does the Copy below, for the
// application's art and for Teddy World's own pair. A house that carried a differently
// sized 1015 would be stretched into these 214x54, which is what the original would do
// with it too.
const (
	kEscPausePictID = 1015
	kTabPausePictID = 1016

	kPausePlateWide = 214
	kPausePlateTall = 54
)

// The hint row under the placard. It has no counterpart in the C; see World.PauseHint and
// pauseHintRect.
const (
	kPauseHintGap  = 6  // between the placard's bottom and the row's top
	kPauseHintTall = 15 // 9 rows of font with three above and three below
	kPauseHintPad  = 6  // either side of the text
	kPauseHintBase = 10 // the baseline, measured from the row's top: 3 + the font's ascent
)

// DoPause is Input.c:77-117: draw the placard, wait, put the pixels back.
//
// The two guards are the whole of the port's own logic. A nil Pause hook makes this a
// no-op, which is the right answer for a headless run rather than a degraded one: with no
// keyboard nothing could ever end the pause, so a faithful transcription would hang the
// fidelity corpus on the first frame a recorded pause key appears. The Paused test stops a
// re-entrant call -- 1.10's save runs from inside the pause loop and will reach code that
// polls -- and is not what keeps the second glider from pausing twice on one press; that is
// the host's press edge, and the file comment says why it has to be.
func (w *World) DoPause() {
	if w.Pause == nil || w.Paused {
		return
	}

	// paused = true, then the loop (Input.c:96-105). The flag is the host's loop
	// condition, so DoCommandKey clearing it from underneath -- Command-Q's
	// `playing = false; paused = false` -- ends the pause here exactly as it does there.
	w.Paused = true
	// Seeded with what the first paint will cover, then widened by every paint. See
	// World.pausePainted: the hint can change while the pause is up, and it is what the
	// restore below is measured against.
	w.pausePainted = w.pauseArea()
	w.Pause(w.paintPause)
	w.Paused = false

	// CopyBits(workSrcMap -> mainWindow, bounds, bounds, srcCopy) (Input.c:107-109). The
	// work map holds the last composited frame, so this is the cheapest possible restore
	// and the reason the placard needs no saved-bits buffer of its own. The area is the
	// placard plus every hint row that was drawn, because all of them went over the same
	// pixels.
	area := w.pausePainted
	w.Main.Copy(w.R.Work, area, area, render.SrcCopy)
	w.present()
}

// paintPause draws the placard and the hint, and presents.
//
// The host calls it once per pass through its wait loop, after pumping events, which is
// what makes an expose harmless: RefreshGameWindow answers an update event by copying the
// whole play area up from the work map, wiping the placard, and the next pass puts it back
// before anything is presented. Drawing it once at the top of the pause instead would
// leave a paused game looking like a live one whenever it was covered and uncovered.
//
// It draws into Main and not into the work map, which is Input.c:81's SetPort(mainWindow)
// and is what leaves the work map holding the frame the restore reads back.
func (w *World) paintPause() {
	plate := w.pausePlateRect()

	id := int16(kTabPausePictID)
	if w.EscPause {
		id = kEscPausePictID
	}
	if art := w.R.A.Plate(id); art != nil {
		w.Main.Copy(art, art.Bounds(), plate, render.SrcCopy)
	} else {
		w.drawPausePanel(plate, w.resumeKeyPhrase())
	}

	if r := w.pauseHintRect(); !render.Empty(r) {
		// A black panel with white text, deliberately unlike the cream placard above it:
		// the placard is 1994's artwork and the hint is this port talking over it.
		w.Main.Fill(r, render.Black8)
		w.Main.FrameRect(r, render.White8)
		centerString(w.Main, r.Left, r.Right, r.Top+kPauseHintBase, w.PauseHint, render.White8, 1)
	}

	// Recorded, not recomputed later. The hint's width is the hint's, so a host that
	// replaces a long hint with a short one mid-pause -- which is exactly what the
	// save-and-quit question does -- would otherwise leave the wide row's outer pixels
	// behind after DoPause restored only the narrow one's rect.
	w.pausePainted = render.UnionSimilar(w.pausePainted, w.pauseArea())

	w.present()
}

// SetPauseHint changes the hint row's text, and is the only safe way to change it while a
// pause is up.
//
// The row is a black panel exactly as wide as its own text (pauseHintRect), so replacing a
// long hint with a short one leaves the long one's outer pixels on screen: paintPause fills
// the new rect and knows nothing about the old. That is not hypothetical -- it is what
// happens the moment the host's give-up question is answered, or a save's confirmation
// replaces the resting line. So the old row is put back from the work map first, which is
// DoPause's own restore applied to one rect instead of all of them, and the next paint draws
// the new text onto pixels the game owns.
//
// pausePainted is not narrowed. It is the union of everything drawn during this pause and the
// restore at the end is measured against it, so a hint that shrank still gets its widest row
// cleaned up if a later paint has grown it again.
func (w *World) SetPauseHint(hint string) {
	if hint == w.PauseHint {
		return
	}
	if w.Paused {
		if r := w.pauseHintRect(); !render.Empty(r) {
			w.Main.Copy(w.R.Work, r, r, render.SrcCopy)
		}
	}
	w.PauseHint = hint
}

// drawPausePanel is what a build with no extracted art shows instead of PICT 1015 or 1016,
// and what a switched-out game shows whether the art is there or not.
//
// docs/IMPROVEMENTS.md 2.6: the absence of a decoration is not the absence of a game. A
// player who cannot see a placard must still be able to tell that the game is paused and
// still be told what resumes it, and both fit in the same 214x54 the picture would have
// filled -- so nothing else on screen moves depending on whether the art is there.
//
// The second line is the caller's because there are two ways to resume and only one of them
// is a key. See pumpWhileSwitchedOut.
func (w *World) drawPausePanel(r Rect, how string) {
	w.Main.Fill(r, render.Black8)
	w.Main.FrameRect(r, render.White8)
	w.Main.FrameRect(render.Inset(r, 2, 2), render.Gray8)

	centerString(w.Main, r.Left, r.Right, r.Top+26, "PAUSED", render.White8, 2)
	centerString(w.Main, r.Left, r.Right, r.Top+44, how, render.LtGray8, 1)
}

// resumeKeyPhrase is the no-art panel's second line during a real pause: the key the player
// pressed to get here is the key that gets them out, and isEscPauseKey is which one it was.
func (w *World) resumeKeyPhrase() string {
	if w.EscPause {
		return "press Esc to resume"
	}
	return "press Tab to resume"
}

// pumpWhileSwitchedOut is PlayGame's `do { HandlePlayEvent(); } while (switchedOut)` with
// one thing added: while the game is in the background it says so on the screen.
//
// docs/IMPROVEMENTS.md 2.28 is the argument. Suspend-on-focus-loss is right and it is also
// invisible -- the window holds the last frame, draws nothing and answers nothing, which is
// exactly what a hung program looks like. A window manager that moves focus without being
// asked (this development host does it about a second after a window opens) therefore turns
// a correct pause into a bug report.
//
// It shares the placard with DoPause so that the two kinds of pause look the same, and it
// deliberately does *not* use the 1994 artwork: both pictures name a key to press, and the
// key that ends this pause is not a key at all. The panel says what actually works.
//
// The two guards are the ones DoPause has, for the same reasons: a host that cannot draw
// gets the C's loop exactly, and painting only when the loop really spun keeps the common
// case -- a game that never loses focus -- free of an extra rect copy per frame.
func (w *World) pumpWhileSwitchedOut() {
	painted := false
	for {
		w.HandlePlayEvent()
		if !w.SwitchedOut {
			break
		}
		if w.Present == nil {
			continue
		}
		// Every pass, like the pause loop, so that an expose answered mid-suspend --
		// RefreshGameWindow copying the whole play area up from the work map -- cannot
		// leave a frozen game looking like a live one.
		w.drawPausePanel(w.pausePlateRect(), "click the window to resume")
		w.present()
		painted = true
	}
	if painted {
		// DoPause's restore, and the same reasoning: the work map still holds the frame
		// this drew over, so the panel needs no saved bits of its own.
		r := w.pausePlateRect()
		w.Main.Copy(w.R.Work, r, r, render.SrcCopy)
		w.present()
	}
}

// pausePlateRect is Input.c:82-83: QSetRect(0,0,214,54) centred in houseRect.
//
// It is computed rather than stored because houseRect is fixed for the life of the
// process (render.NewView) and the placard is one of the few things in the game that does
// not move with the view origin.
func (w *World) pausePlateRect() Rect {
	return render.CenterIn(render.SetRect(0, 0, kPausePlateWide, kPausePlateTall), w.R.V.House)
}

// pauseHintRect is where PauseHint is drawn: a row under the placard, empty when there is
// no hint to draw.
//
// It exists because the artwork is wrong for this port and cannot be corrected. Both
// placards read "or Cmd-Q to Quit the game", and there is no Command key here -- the
// window manager owns Command-Q on every platform this builds for, and cmd/glidergo
// answers a paused player's Q on its own instead (see World.PauseHint). A substitute key
// nobody can see is not a way out, and this row is the whole of how a player learns it --
// which is also why cmd/glidergo makes Escape pause rather than end the game
// (docs/IMPROVEMENTS.md 2.7): the pause is where the question gets asked.
//
// Centred on the placard rather than on the screen, and clamped into houseRect, so that a
// hint longer than the placard grows symmetrically and still cannot spill onto the
// scoreboard or off the edge -- which would leave pixels the restore does not cover.
func (w *World) pauseHintRect() Rect {
	if w.PauseHint == "" {
		return Rect{}
	}
	house := w.R.V.House
	plate := w.pausePlateRect()

	wide := render.StringWidth(w.PauseHint) + 2*kPauseHintPad
	if wide > house.Wide() {
		wide = house.Wide()
	}
	left := plate.Left + (plate.Wide()-wide)/2
	if left < house.Left {
		left = house.Left
	}
	if left+wide > house.Right {
		left = house.Right - wide
	}
	top := plate.Bottom + kPauseHintGap
	return render.SetRect(left, top, left+wide, top+kPauseHintTall)
}

// pauseArea is everything paintPause touched: the union of the placard and the hint row.
//
// The C restores the placard's rect alone because that is all it drew. Restoring the union
// rather than calling RefreshGameWindow keeps the C's cost -- one rect copy, no scoreboard
// refresh -- and keeps its behaviour: a partial copy from the work map cannot disturb a
// scoreboard the pause never covered.
func (w *World) pauseArea() Rect {
	r := w.pausePlateRect()
	hint := w.pauseHintRect()
	if render.Empty(hint) {
		return r
	}
	if hint.Left < r.Left {
		r.Left = hint.Left
	}
	if hint.Top < r.Top {
		r.Top = hint.Top
	}
	if hint.Right > r.Right {
		r.Right = hint.Right
	}
	if hint.Bottom > r.Bottom {
		r.Bottom = hint.Bottom
	}
	return r
}

// centerString draws text centred between two columns, clamped to the left one. The shell
// has the same three lines (internal/shell/screens.go) and they are not shared: this
// package draws text in exactly two places -- here and the no-art panel above it -- and a
// text-layout helper exported between the two would be a package boundary carrying one
// function.
func centerString(s *render.Surface, left, right, v int16, text string, idx uint8, scale int) {
	h := left + (right-left-render.StringWidthScaled(text, scale))/2
	if h < left {
		h = left
	}
	s.DrawStringScaled(h, v, text, idx, scale)
}
