package game

// Scoreboard.c's geometry: where the board goes, and the two calls PlayGame and NewGame
// make into it. The board's *contents* -- the score digits, the three inventory bars, the
// four badges and the glider icons -- are 1.5b's second commit, and their stubs are with
// the rest of the Env surface in env.go.
//
// This file exists ahead of them because of what reading Scoreboard.c turned up, which is
// worth stating before any of the code:
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

// AdjustScoreboardHeight is Scoreboard.c:412-449: put the board where this view's room
// layout leaves room for it.
//
// One caller, NewGame's first statement, so this runs once per game and the layout cannot
// change mid-game.
//
// The C accumulates: it computes a single `offset` and applies it to seven rects with
// QOffsetRect, which is only correct because the guard above it makes the function run at
// most once per transition between the two modes. Call it twice with numNeighbors on 1 and
// the offset applies twice and the board ends up 158 rows below where it belongs. The
// guard is load-bearing, and it is guarding an accumulation that did not need to be one.
//
// **The port assigns instead of accumulating**, because there are exactly two layouts and
// each has one answer:
//
//	High (9 neighbours)   the board is below the rooms   rows 460..480, and justRoomsRect
//	                      is the whole work rect
//	Low (1 or 3)          the board is above the rooms   rows 59..79, in the black band
//	                      over the central room, and justRoomsRect is the central room
//
// so calling it any number of times gives the same answer. The latch is kept and still
// set, because it is the C's and because a reader diffing the two files should see it --
// but here it is an optimisation rather than a correctness requirement, which is the safer
// of the two.
//
// The High arm's row numbers are the port's deviation and the Low arm's are the
// original's; see World.BoardDestRect. Six of the C's seven rects -- the two extra board
// panels and the four badges -- do not exist yet and land with the board's contents in the
// next commit. They take the same two positions relative to BoardDestRect that
// StructuresInit.c gives them.
func (w *World) AdjustScoreboardHeight() {
	newMode := ScoreboardLow
	if w.R.NumNeighbors == 9 {
		newMode = ScoreboardHigh
	}
	if w.WasScoreboardMode == newMode {
		return
	}

	central := w.R.V.LocalRoomsDest[CentralRoom]

	switch newMode {
	case ScoreboardHigh:
		w.R.V.JustRoomsRect = render.ZeroCorner(w.R.V.House)
		w.BoardDestRect = render.Rect{
			Top: w.R.V.Screen.Bottom - ScoreboardTall, Left: w.R.V.Screen.Left,
			Bottom: w.R.V.Screen.Bottom, Right: w.R.V.Screen.Right,
		}

	case ScoreboardLow:
		// justRoomsRect stops being the whole work rect and becomes the central room's
		// rows only, so RenderFrame's whole-screen fallback -- and HandlePlayEvent's
		// update blit -- stop touching the eight neighbours. In this layout the
		// neighbours are off screen, which is the point of choosing it.
		w.R.V.JustRoomsRect = render.ZeroCorner(w.R.V.House)
		w.R.V.JustRoomsRect.Top = central.Top
		w.R.V.JustRoomsRect.Bottom = central.Bottom

		w.BoardDestRect = render.Rect{
			Top: central.Top - ScoreboardTall, Left: w.R.V.Screen.Left,
			Bottom: central.Top, Right: w.R.V.Screen.Right,
		}
	}

	w.WasScoreboardMode = newMode
}

// blitBoardToScreen copies the composed board onto the screen at BoardDestRect.
//
// The single primitive every scoreboard refresh in the original ends with, gathered into
// one function on purpose: there are nine call sites in Scoreboard.c, each with its own
// CopyBits, and having nine of them is what let the destination rect be wrong for years
// without anyone finding the one place to look. With one, the port's deviation is one
// line and the invisible-scoreboard bug is impossible to reintroduce piecemeal.
//
// A no-op until the board surface exists, next commit.
func (w *World) blitBoardToScreen() {}

// HandleDynamicScoreboard is Scoreboard.c:453-500 and belongs to the next commit: the
// per-frame half of the board, which is why PlayGame calls it beside RenderFrame rather
// than beside RefreshScoreboard.
//
// It is the animated part -- the score counting up toward its target a few points a frame,
// and the glider icons -- as distinct from RefreshScoreboard, which redraws the lot at
// once on a room change or a title change.
func (w *World) HandleDynamicScoreboard() {}

// BlackenScoreboard is Scoreboard.c: fill the board black, called from NewGame's teardown
// so the shell does not inherit the last frame's score.
//
// Next commit, with the board surface. Note that this is Scoreboard.c's function and is
// *not* the inline arcade block below, which does something similar to a different rect at
// a different time; the two are separate in the original and are kept separate here.
func (w *World) BlackenScoreboard() {}

// arcadeBlackenBoard is the BUILD_ARCADE_VERSION block inlined twice in Play.c, at :512
// and again at :551, its own comment reading "Need to paint over the scoreboard black."
//
// BUILD_ARCADE_VERSION is defined in the shipped 1.1.2 sources, so both blocks are live
// and this is not dead code. Each is about forty lines: blacken the board's source map,
// blit it out, then reload the scoreboard PICT and draw it back, horizontally centred by
// the same `(RectWide(&boardSrcRect) - kMaxViewWidth) / 2` expression StructuresInit.c
// uses -- so what it actually does is *reset* the board to its empty state, not just
// blank it.
//
// Both call sites call this one function. The C's second copy is unconditional, after the
// loop, so a player who quits mid-game gets the same reset as one who finishes.
//
// Next commit. Two things to fix rather than transcribe when it lands: the PICT reload is
// per-game-over and could be done once at launch, and its failure path is
// RedAlert(kErrFailedGraphicLoad), which kills the application over a cosmetic asset --
// see docs/IMPROVEMENTS.md 2.12.
func (w *World) arcadeBlackenBoard() {}
