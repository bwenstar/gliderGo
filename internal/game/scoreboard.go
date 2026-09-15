package game

// Scoreboard.c in full: where the board goes, what goes on it, and the eleven functions
// that put it there.
//
// The geometry and the five offscreen maps are in render (see render/scoreboard.go, which
// says why the split falls where it does). This file is the half that reads the game's
// counters and decides when to draw, which is also the half where the original's decisions
// are surprising -- and one of them is worth stating before any of the code:
//
// **The original, in its shipped configuration, displays no scoreboard.** Not a subtle
// one, not a clipped one -- none. The chain is four links and every one of them is a
// one-liner:
//
//	StructuresInit.c:70    wasScoreboardMode = kScoreboardHigh   at launch
//	StructuresInit.c:97-98 boardDestRect = boardSrcRect offset by -kScoreboardTall
//	                       ... so the board's destination is rows -20..0 of the window
//	Scoreboard.c:416-421   newMode is kScoreboardHigh when numNeighbors == 9,
//	                       and the whole body is guarded by wasScoreboardMode != newMode
//	Main.c:154, :191-192   numNeighbors defaults to 9, and drops to 1 only on a screen
//	                       512 pixels wide or narrower
//
// So on any normal Macintosh the guard never opens, boardDestRect never leaves rows
// -20..0, and every CopyBits into it is clipped entirely away. The player sees no score,
// no lives, no battery, no bands and no foil.
//
// It is not a bug so much as a consequence: with nine neighbours the view is 640x460 of
// solid rooms and there is nowhere left to put a board. Choose the one- or
// three-neighbour view -- which the game does automatically on a small screen -- and the
// guard opens, the board moves to rows 59..79, and justRoomsRect shrinks to the central
// room. The scoreboard was designed for the small-screen layout and the big-screen layout
// silently gave it up.
//
// A released game cannot ship that. The port keeps the layout and moves the board: see
// World.BoardDestRect for the deviation and docs/IMPROVEMENTS.md 2.9 for the entry.

import "glidergo/internal/render"

// The two scoreboard layouts and the band's height (GliderDefines.h:513-515).
//
// ScoreboardTall duplicates render's unexported kScoreboardTall, which is the value
// render.NewView subtracts from Screen to get House. They are the same constant read by
// two packages and neither should import the other for it.
const (
	ScoreboardHigh int16 = 0
	ScoreboardLow  int16 = 1

	ScoreboardTall int16 = 20
)

// ScoreRollAmount is how many points the displayed score climbs per frame
// (Scoreboard.c:21). At thirty frames a second a 500-point prize takes a little over a
// second to count up, with a tick of sound per step.
const ScoreRollAmount int32 = 13

// The four "running low" thresholds (Scoreboard.c:74-77), each commented `25%` in the
// original. They are what makes a badge blink rather than merely sit there, and each is
// measured against the counter it names -- so HeliumLow is negative, because helium is
// stored as a negative battery charge and runs *up* toward zero.
//
// They are also all off by one in the same direction: the test is `< kFoilLow`, so a badge
// with exactly two units left does not blink and one with one unit does. Two out of eight
// is 25% of the foil's maximum, which is what the comments mean, and the strict comparison
// makes the blink start one unit later than that. Transcribed rather than corrected --
// the original's blink is a warning, not a gauge.
const (
	FoilLow    int16 = 2
	BatteryLow int16 = 17
	HeliumLow  int16 = -38
	BandsLow   int16 = 2
)

// The score tick (GliderDefines.h:72,121): one blip per ScoreRollAmount points. Priority
// 101 is near the bottom of the scale, so anything else happening drowns it out, which is
// what keeps a long roll from talking over the game.
const (
	ScoreTikSound    int16 = 17
	ScoreTikPriority int16 = 101
)

// ---------------------------------------------------------------------------
// Placement
// ---------------------------------------------------------------------------

// placeScoreboard puts all seven movable rects where the given layout wants them, and sets
// JustRoomsRect to match.
//
// **This is where the port's one scoreboard deviation lives, and it is the only place.**
// The C computes a single vertical `offset` and applies it to the seven rects with
// QOffsetRect; the port computes the same single offset and *assigns* every rect as its
// unmoved form on the View plus that offset. The two differ in exactly one way that
// matters: the C's version is only correct if it runs at most once, because a second call
// adds the offset again, and the guard in AdjustScoreboardHeight is what enforces that.
// Assigning makes the operation idempotent, so the latch becomes an optimisation instead
// of a correctness requirement -- and the seven rects keep the relative layout
// InitScoreboardMap gave them, because they all move by the same amount.
//
// The offsets:
//
//	High (9 neighbours)   the C's is -central.Top, which moves the board from rows
//	                      -20..0 to rows -99..-79: further off screen, and invisible.
//	                      The port's is +Screen.Bottom, rows 460..480 -- the band below
//	                      the house rect that no room can draw into.
//	Low (1 or 3)          +central.Top, rows 59..79, in the black band above the central
//	                      room. This one is the original's, unchanged.
func (w *World) placeScoreboard(mode int16) {
	v := w.R.V
	central := v.LocalRoomsDest[CentralRoom]

	var offset int16
	switch mode {
	case ScoreboardHigh:
		// The deviation. See World.BoardDestRect.
		offset = v.Screen.Bottom
		v.JustRoomsRect = render.ZeroCorner(v.House)

	case ScoreboardLow:
		// justRoomsRect stops being the whole work rect and becomes the central room's
		// rows only, so RenderFrame's whole-screen fallback -- and HandlePlayEvent's
		// update blit -- stop touching the eight neighbours. In this layout the
		// neighbours are off screen, which is the point of choosing it.
		offset = central.Top
		v.JustRoomsRect = render.ZeroCorner(v.House)
		v.JustRoomsRect.Top = central.Top
		v.JustRoomsRect.Bottom = central.Bottom
	}

	w.BoardDestRect = render.Offset(v.BoardDest, 0, offset)
	w.BoardGQDestRect = render.Offset(v.BoardGQDest, 0, offset)
	w.BoardPQDestRect = render.Offset(v.BoardPQDest, 0, offset)
	for i := range w.BadgesDestRects {
		w.BadgesDestRects[i] = render.Offset(v.BadgesDest[i], 0, offset)
	}
}

// AdjustScoreboardHeight is Scoreboard.c:412-449: put the board where this view's room
// layout leaves room for it.
//
// One caller, NewGame's first statement, so this runs once per game and the layout cannot
// change mid-game.
//
// The latch is kept and still set, because it is the C's and because a reader diffing the
// two files should see it -- but here it guards an idempotent body rather than an
// accumulation, which is the safer of the two. NewWorld has already placed the rects for
// the mode the latch names, so the common case of this returning early leaves them correct
// rather than leaving them off screen.
func (w *World) AdjustScoreboardHeight() {
	newMode := ScoreboardLow
	if w.R.NumNeighbors == 9 {
		newMode = ScoreboardHigh
	}
	if w.WasScoreboardMode == newMode {
		return
	}
	w.placeScoreboard(newMode)
	w.WasScoreboardMode = newMode
}

// ---------------------------------------------------------------------------
// The whole board
// ---------------------------------------------------------------------------

// RefreshScoreboard is Scoreboard.c:53-68: compose the entire board and put it on screen.
//
// Called on a room change, on a title change, and by the window's update handler -- the
// slow path, as against HandleDynamicScoreboard's per-frame one.
//
// The order is load-bearing in one place. The three text panels are composed *into the
// band* and the band is then blitted to the screen, so they are one flicker-free copy; the
// three badges are blitted from their sheet *straight to the screen*, over the band that
// was just drawn. Refresh the badges first and the band overwrites them.
func (w *World) RefreshScoreboard(mode int16) {
	w.DoRollScore = true

	w.refreshRoomTitle(mode)
	w.refreshNumGliders()
	w.refreshPoints()

	w.blitBoardToScreen()

	w.QuickBatteryRefresh(false)
	w.QuickBandsRefresh(false)
	w.QuickFoilRefresh(false)
}

// HandleDynamicScoreboard is Scoreboard.c:72-132: the per-frame half of the board.
//
// It does two unrelated jobs. The first is the score roll: whenever Score has run ahead of
// DisplayedScore the displayed number climbs by ScoreRollAmount, plays a tick and redraws.
// Note where the sound and the redraw sit -- **outside** the DoRollScore if/else and inside
// the `Score > DisplayedScore` test -- so they happen on the jump-straight-there path too.
// That path is unreachable in the shipped game: DoRollScore is set by every
// RefreshScoreboard and cleared by nothing, anywhere. It is transcribed because the
// structure is what makes the tick fire exactly once per redraw, and a port that hoisted
// the two calls into the branch would double the tick the first time someone did clear the
// flag.
//
// The second job is the low-inventory blink, and it is a state machine written as a switch
// on the frame counter's low three bits. Each badge gets a "show" frame and a "hide" frame
// eight frames apart, so a badge that is running low flashes at about four hertz:
//
//	frame & 7 == 0   show foil        == 5   hide foil
//	             1   hide battery     == 4   show battery
//	             2   show bands       == 7   hide bands
//
// Cases 3 and 6 are absent and there is no default, so those two frames of every eight do
// nothing at all -- which is not an oversight but the reason the three badges blink out of
// phase with each other instead of in unison. The battery's pair is inverted relative to
// the other two (hide at 1, show at 4, rather than show-then-hide), which shifts its blink
// half a cycle away from the foil's.
//
// Every arm is guarded on the counter being *both* non-zero and below its threshold, so a
// badge that is empty stops blinking and simply goes out -- the blank cell is what
// QuickFoilRefresh draws when the count reaches zero anyway.
//
// One port-only consequence of where PlayGame calls this: everything it writes lands on
// screen a frame late. In the C, mainWindow *was* the frame buffer, so a CopyBits into it
// was visible the instant it returned; here Main is a surface and Present is what makes it
// visible, and Present happens inside RenderFrame -- which runs immediately *before* this.
// So a tick of score, or a badge going dark, is presented by the next frame's RenderFrame,
// 33 milliseconds later. Uniform, invisible, and cheaper than the alternative, which is a
// second whole-screen upload per frame; see World.Present on why the presentation points
// are the four they are.
func (w *World) HandleDynamicScoreboard() {
	if w.Score > w.DisplayedScore {
		if w.DoRollScore {
			w.DisplayedScore += ScoreRollAmount
			if w.DisplayedScore > w.Score {
				w.DisplayedScore = w.Score
			}
		} else {
			w.DisplayedScore = w.Score
		}

		w.PlayPrioritySound(ScoreTikSound, ScoreTikPriority)
		w.quickScoreRefresh()
	}

	switch w.Frame & 0x07 {
	case 0: // show foil
		if w.Foil > 0 && w.Foil < FoilLow {
			w.QuickFoilRefresh(false)
		}

	case 1: // hide battery
		if (w.Battery > 0 && w.Battery < BatteryLow) || (w.Battery < 0 && w.Battery > HeliumLow) {
			w.QuickBatteryRefresh(true)
		}

	case 2: // show rubber bands
		if w.Bands > 0 && w.Bands < BandsLow {
			w.QuickBandsRefresh(false)
		}

	case 4: // show battery
		if (w.Battery > 0 && w.Battery < BatteryLow) || (w.Battery < 0 && w.Battery > HeliumLow) {
			w.QuickBatteryRefresh(false)
		}

	case 5: // hide foil
		if w.Foil > 0 && w.Foil < FoilLow {
			w.QuickFoilRefresh(true)
		}

	case 7: // hide rubber bands
		if w.Bands > 0 && w.Bands < BandsLow {
			w.QuickBandsRefresh(true)
		}
	}
}

// ---------------------------------------------------------------------------
// The three text panels
// ---------------------------------------------------------------------------

// refreshRoomTitle is Scoreboard.c:136-188: draw the room's name, or one of the two
// substitute captions, into the band.
//
// Unexported because the C's declaration is at the top of Scoreboard.c and not in
// Externs.h: the three panel refreshes are file-local and RefreshScoreboard is their only
// caller.
func (w *World) refreshRoomTitle(mode int16) {
	w.Board.Panel(w.Board.Title, titleText(w, mode))
	w.Board.Board.Copy(w.Board.Title, w.R.V.BoardTSrc, w.R.V.BoardTDest, render.SrcCopy)
}

// titleText is the C's two identical switch statements, once.
//
// The original writes the switch out twice -- once for the black pen at (1,10) and once for
// the white pen at (0,9) -- which is how the drop shadow is made, and means the string is
// chosen twice. Choosing it once and letting Panel draw it twice is the same picture.
func titleText(w *World, mode int16) string {
	switch mode {
	case EscapedTitleMode:
		return "Hit Delete key if unable to Follow"
	case SavingTitleMode:
		return "Saving Game…"
	}
	if rm := w.ThisRoom(); rm != nil {
		return rm.Name.Text()
	}
	// thisRoom is a copy in the C and is never nil; here it is a lookup that can miss.
	// See World.ThisRoom for why the port made that trade.
	return ""
}

// refreshNumGliders is Scoreboard.c:192-227: draw the remaining glider count into the band.
//
// **It clamps at zero and QuickGlidersRefresh does not**, and the difference is real rather
// than accidental: Mortals goes to -1 for the last death of a one-player game and to -2 for
// the last of a two-player game (see OffAMortal), so an unclamped draw here would show the
// player "-1" gliders in the moment before the game-over sequence takes the window. Which
// is exactly what the quick path does show, because its only caller is the branch where a
// glider remains.
func (w *World) refreshNumGliders() {
	display := w.Mortals
	if display < 0 {
		display = 0
	}
	w.Board.Panel(w.Board.Gliders, itoa16(display))
	w.Board.Board.Copy(w.Board.Gliders, w.R.V.BoardGSrc, w.R.V.BoardGDest, render.SrcCopy)
}

// refreshPoints is Scoreboard.c:231-264: draw the score into the band, and resynchronise
// the roll.
//
// The trailing `displayedScore = theScore` is what stops the roll from replaying: this path
// draws the *real* score, so anything the roll had left to count up to has already been
// shown. Every RefreshScoreboard therefore both arms the roll (DoRollScore) and empties it,
// which is why walking into a room does not restart a count-up that was in progress.
func (w *World) refreshPoints() {
	w.Board.Panel(w.Board.Points, itoa32(w.Score))
	w.Board.Board.Copy(w.Board.Points, w.R.V.BoardPSrc, w.R.V.BoardPDest, render.SrcCopy)
	w.DisplayedScore = w.Score
}

// QuickGlidersRefresh is Scoreboard.c:268-299: redraw just the glider count, straight to
// the screen.
//
// Its one caller is the respawn branch of OffAMortal, and it is called *before* HideGlider
// rather than after, so the count on the scoreboard drops a moment before the glider
// disappears from the room.
//
// It does not clamp; see refreshNumGliders for why that is safe here and not there.
func (w *World) QuickGlidersRefresh() {
	w.Board.Panel(w.Board.Gliders, itoa16(w.Mortals))
	w.Main.Copy(w.Board.Gliders, w.R.V.BoardGSrc, w.BoardGQDestRect, render.SrcCopy)
}

// quickScoreRefresh is Scoreboard.c:303-334: redraw just the score, straight to the screen,
// showing DisplayedScore rather than Score.
//
// The distinction is the entire point of the roll: this is the only function that draws the
// intermediate number, and HandleDynamicScoreboard is its only caller.
func (w *World) quickScoreRefresh() {
	w.Board.Panel(w.Board.Points, itoa32(w.DisplayedScore))
	w.Main.Copy(w.Board.Points, w.R.V.BoardPSrc, w.BoardPQDestRect, render.SrcCopy)
}

// ---------------------------------------------------------------------------
// The four badges
// ---------------------------------------------------------------------------

// QuickBatteryRefresh is Scoreboard.c:338-364. It is one function for two badges, because
// Battery is one signed counter for two power-ups: positive is battery charges and negative
// is helium.
//
// `flash` true means *draw the blank cell* -- it is the hide half of the blink, not a
// "force a redraw" flag, and the parameter is named the same thing in the C. There is no
// cached value and no dirty test anywhere in this file: every call blits, and the blink
// exists precisely because the blit is unconditional.
//
// Note which cell the blank branch reads: badgesBlank[BatteryBadge], even when it is helium
// that is being hidden. The two destination rects are identical, so it makes no visible
// difference, and it leaves badgesBlank[HeliumBadge] as the one rect InitScoreboardMap
// computes that nothing ever uses. Transcribed as-is; the test in scoreboard_test.go pins
// the two blank cells as pixel-identical, which is what makes the omission provably
// invisible rather than merely apparently so.
func (w *World) QuickBatteryRefresh(flash bool) {
	switch {
	case w.Battery > 0 && !flash:
		w.blitBadge(render.BatteryBadge, true)
	case w.Battery < 0 && !flash:
		w.blitBadge(render.HeliumBadge, true)
	default:
		w.blitBadge(render.BatteryBadge, false)
	}
}

// QuickBandsRefresh is Scoreboard.c:368-386. See QuickBatteryRefresh on `flash`.
func (w *World) QuickBandsRefresh(flash bool) {
	w.blitBadge(render.BandsBadge, w.Bands > 0 && !flash)
}

// QuickFoilRefresh is Scoreboard.c:390-408. See QuickBatteryRefresh on `flash`.
//
// It reads Foil, the count, and not ShowFoil, the sheet-loaded flag -- so the badge tracks
// the inventory rather than the sprite, and lights up on the frame the foil is collected
// rather than the frame the sheet swap catches up.
func (w *World) QuickFoilRefresh(flash bool) {
	w.blitBadge(render.FoilBadge, w.Foil > 0 && !flash)
}

// blitBadge is the one CopyBits all three badge refreshes end in: lit cell or blank cell,
// out of the shared sheet and onto the screen.
//
// Six of the C's nine badge CopyBits calls are this, character for character apart from the
// two rect indices, which is how the port can afford to have one.
func (w *World) blitBadge(badge int, lit bool) {
	if w.Board.Badge == nil {
		return // the sheet failed to load; Assets.Err has it
	}
	src := w.R.V.BadgesBlank[badge]
	if lit {
		src = w.R.V.BadgesBadges[badge]
	}
	w.Main.Copy(w.Board.Badge, src, w.BadgesDestRects[badge], render.SrcCopy)
}

// ---------------------------------------------------------------------------
// The band, and taking it away again
// ---------------------------------------------------------------------------

// blitBoardToScreen copies the composed band onto the screen at BoardDestRect.
//
// The single primitive every whole-board refresh in the original ends with, gathered into
// one function on purpose: there are three CopyBits calls with this exact shape in the
// shipped source, each with its own copy of the destination rect, and having three of them
// is what let the destination be wrong for years without anyone finding the one place to
// look. With one, the port's deviation is one line and the invisible-scoreboard bug is
// impossible to reintroduce piecemeal.
func (w *World) blitBoardToScreen() {
	w.Main.Copy(w.Board.Board, w.R.V.BoardSrc, w.BoardDestRect, render.SrcCopy)
}

// BlackenScoreboard is Scoreboard.c:453-456, and **it does not touch the scoreboard.**
//
// Its entire body is `UpdateMenuBarWindow()`, which (MainWindow.c:160-169) returns early if
// there is no menu-bar window and otherwise does `GetLocalWindowRect(menuWindow, &bounds);
// PaintRect(&bounds)`. menuWindow is the arcade build's 20-pixel-tall cover over the Mac
// menu bar (MainWindow.c:214-220) -- a *different* window from mainWindow, and a different
// twenty rows from the board's. And the PaintRect has no SetPort before it, so it paints
// that window's rect into whatever port happens to be current, which at this call site is
// not menuWindow either.
//
// So the function's name describes neither what it does nor where it does it. The port has
// no menu bar to cover, and by the time this runs PlayGame's teardown has already
// blackened the band (see arcadeBlackenBoard), so the honest port is a named no-op that
// says so. NewGame's call site is transcribed and live.
func (w *World) BlackenScoreboard() {}

// arcadeBlackenBoard is the BUILD_ARCADE_VERSION block inlined twice in Play.c, at :512 and
// again at :556, its own comment reading "Need to paint over the scoreboard black."
//
// BUILD_ARCADE_VERSION is defined in the shipped 1.1.2 sources, so both blocks are live and
// this is not dead code. Each does three things: blacken the band's source map, blit it to
// the screen -- which is what the comment describes -- and then redraw the board picture
// into the map, so what it really does is *reset* the board to its empty state ready for
// the next game.
//
// **The two copies are not the same, and one of them is broken.** The first, in the
// game-over arm, wraps all three steps in `SetGWorld(boardSrcMap, nil)`, so its DrawPicture
// lands in the band's map. The second, after the loop, restores the previous port
// (Play.c:570) *before* its DrawPicture (:590) -- so the picture is drawn into whatever port
// PlayGame left current, and the band's map is left black. Two visible consequences in the
// original: a stray 1536-wide scoreboard graphic is painted across the top of some other
// port for the instant before the splash screen redraws over it, and the map stays black,
// so **the second game of a session has a black scoreboard for its whole duration** -- on
// the small-screen layout where the board is visible at all.
//
// The port has one function, and it puts the picture where both copies' comment says it
// goes. That is a deviation from the second block's behaviour and it is deliberate; it is
// also the cheaper of the two readings, since the reset now costs a memcpy from a cached
// surface rather than a GetPicture/DrawPicture/ReleaseResource cycle whose failure path is
// RedAlert(kErrFailedGraphicLoad) -- an application-killing error over a cosmetic asset,
// see docs/IMPROVEMENTS.md 2.12.
func (w *World) arcadeBlackenBoard() {
	w.Board.Blacken()
	w.blitBoardToScreen()
	w.Board.ResetBoardArt()
}

// ---------------------------------------------------------------------------
// NumToString
// ---------------------------------------------------------------------------

// itoa16 and itoa32 are NumToString, which the C calls on a short widened to a long and on
// theScore directly.
//
// strconv would do, and is not used, for one reason: these two exist so that the only
// number formatting on the scoreboard path is in one place when the font arrives and starts
// caring about glyph widths. They are also the whole of the port's dependency on how a
// negative count prints, which refreshNumGliders and QuickGlidersRefresh deliberately
// disagree about.
func itoa16(n int16) string { return itoa32(int32(n)) }

func itoa32(n int32) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	// Negated as int32 rather than in place, so that the most negative value does not
	// overflow back onto itself.
	v := int64(n)
	if neg {
		v = -v
	}
	var buf [12]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
