package render

// The scoreboard's geometry and its five offscreen maps: InitScoreboardMap
// (StructuresInit.c:59-163) and the panel-painting half of Scoreboard.c.
//
// The split between this file and internal/game/scoreboard.go follows the split in the
// original, which is not obvious from the file names: StructuresInit.c owns the rects and
// the GWorlds, Scoreboard.c owns *what goes in them* and every CopyBits. So the surfaces
// and the one shared panel-painting primitive are here, and the eleven functions that
// decide when to draw and where to blit are in the game package, next to the counters they
// read.
//
// Nothing here is per-locale. DrawLocale discards and rebuilds nine tables on every room
// change (see Scene), and the board must survive that -- the score does not reset when the
// player walks through a door -- so the Scoreboard is created once beside the View and the
// Assets and handed to the World, not hung off the Scene.

// The four badges, in the order InitScoreboardMap lays their cells out down the sheet.
// Exported because the game package indexes the three rect arrays with them.
//
// Battery and helium are two badges for one counter: World.Battery is signed, positive for
// battery charges and negative for helium, so exactly one of these two is ever lit and
// their destination rects are deliberately identical.
const (
	FoilBadge = iota
	BandsBadge
	BatteryBadge
	HeliumBadge
)

// kGrayBackgroundColor is the index the three text panels are cleared to
// (Scoreboard.c:15): 251, the same gray the board art's recesses are drawn in, so a
// cleared panel is invisible against the board.
//
// The C has a second constant beside it, kGrayBackgroundColor4 = 10, used when
// thisMac.isDepth == 4. The port has no 4-bit path -- the extracted art is 8-bit indexed
// and the palette is the 256-entry 'clut' -- so the six `isDepth == 4` arms in
// Scoreboard.c collapse to their else branches. That is a deviation only on hardware that
// no longer exists.
const kGrayBackgroundColor = 251

// The two PICTs the board is assembled from (GliderDefines.h). They are reached through
// Assets.Strip rather than by id, because the extractor writes them out under their
// descriptive names; the ids are kept here so a grep for either finds the other.
const (
	kScoreboardPictID = 1997 // strip/board.png,  1536x20
	kBadgePictID      = 1996 // strip/badge.png,  32x66
)

// initScoreboard is InitScoreboardMap's arithmetic, minus the five CreateOffScreenGWorld
// calls (those are NewScoreboard's). It is called from NewView and is not exported,
// because a view with half a scoreboard is not a thing any caller should be able to make.
//
// **The two hOffsets are different quantities and are deliberately written out twice.**
// The C reuses one local for both, which reads like one value used twice and is not: the
// first centres a 1536-pixel-wide picture inside the board's own width, the second
// centres a 640-pixel *layout* inside the house rect. At 640x480 they are -448 and 0; at
// 1536 wide they are 0 and 448. Folding them together would be correct at exactly one
// resolution.
func (v *View) initScoreboard() {
	// boardSrcRect: the house rect, cornered, with the bottom pulled up to the band's
	// height. So the board is as wide as the drawable area and 20 tall.
	v.BoardSrc = ZeroCorner(v.House)
	v.BoardSrc.Bottom = kScoreboardTall

	// boardDestRect. Rows -20..0 of the window -- above the top edge, outside every clip.
	// That is the original's value and it is why the shipped game shows no scoreboard;
	// the port moves it in AdjustScoreboardHeight and World.BoardDestRect has the whole
	// account. It is stored unmoved here because it is the anchor the seven movable rects
	// are re-derived from, and an anchor that has already been moved cannot serve.
	v.BoardDest = Offset(v.BoardSrc, 0, -kScoreboardTall)

	// hOffset #2 (StructuresInit.c:100-102): the panels' horizontal origin. The literals
	// below -- 137, 526, 570, 432, 449, 467 -- are positions in a 640-wide layout, and
	// this shifts that layout to the middle of a wider board. The `< 0` guard cannot fire
	// at 640 or above; it exists for the sub-640 screens the small-view mode was for, and
	// -128 is a hand-tuned value rather than a formula.
	hOffset := (v.House.Wide() - 640) / 2
	if hOffset < 0 {
		hOffset = -128
	}

	v.BoardTSrc = SetRect(0, 0, 256, 12) // room title
	v.BoardTDest = Offset(v.BoardTSrc, 137+hOffset, 5)

	v.BoardGSrc = SetRect(0, 0, 20, 10) // # gliders
	v.BoardGDest = Offset(v.BoardGSrc, 526+hOffset, 5)

	v.BoardPSrc = SetRect(0, 0, 64, 10) // points
	v.BoardPDest = Offset(v.BoardPSrc, 570+hOffset, 5)

	// The two quick-refresh destinations: the same two panels, but positioned in the
	// window rather than in the board map, hence the extra band's-height shift that
	// BoardDest also carries.
	v.BoardPQDest = Offset(v.BoardPDest, 0, -kScoreboardTall)
	v.BoardGQDest = Offset(v.BoardGDest, 0, -kScoreboardTall)

	// The badge sheet: 32x66, two 16-wide columns by four rows of 16, 16, 17, 17.
	v.BadgeSrc = SetRect(0, 0, 32, 66)
	for i, cell := range [4]struct{ top, tall int16 }{
		FoilBadge:    {0, 16},
		BandsBadge:   {16, 16},
		BatteryBadge: {32, 17},
		HeliumBadge:  {49, 17},
	} {
		v.BadgesBlank[i] = SetRect(0, cell.top, 16, cell.top+cell.tall)
		v.BadgesBadges[i] = Offset(v.BadgesBlank[i], 16, 0)
	}

	// The four destinations. Battery and helium land on the *same* rect, which is not a
	// transcription slip: one signed counter drives both, so only one can be lit and they
	// share the slot. Foil and bands sit a pixel lower than the two 17-tall badges, which
	// is what keeps their bottoms aligned.
	v.BadgesDest[FoilBadge] = Offset(SetRect(0, 0, 16, 16), 432+hOffset, 2-kScoreboardTall)
	v.BadgesDest[BandsBadge] = Offset(SetRect(0, 0, 16, 16), 449+hOffset, 2-kScoreboardTall)
	v.BadgesDest[BatteryBadge] = Offset(SetRect(0, 0, 16, 17), 467+hOffset, 1-kScoreboardTall)
	v.BadgesDest[HeliumBadge] = Offset(SetRect(0, 0, 16, 17), 467+hOffset, 1-kScoreboardTall)
}

// Scoreboard is the five offscreen maps InitScoreboardMap creates: the assembled band and
// the four small maps the band is composed from.
//
// Sizes come from the View, so a Scoreboard belongs to the View it was made with.
type Scoreboard struct {
	v *View
	a *Assets

	// Board is boardSrcMap: the band as it will be blitted, art plus panels. Every
	// refresh writes panels into this and then copies the whole band out, which is why
	// the score does not flicker -- the assembly is offscreen.
	Board *Surface

	// Badge is badgeSrcMap. It is the cached strip surface rather than a copy, because
	// nothing ever draws into it; the C's own GWorld is written once by LoadGraphic and
	// read forever after.
	Badge *Surface

	// Title, Gliders and Points are boardTSrcMap, boardGSrcMap and boardPSrcMap: one
	// scratch panel per piece of text. They exist as separate surfaces in the original so
	// that a panel can be cleared and redrawn without disturbing the art around it, and
	// they are kept because the two quick-refresh paths blit a panel straight to the
	// screen and need it to be a standalone image.
	Title, Gliders, Points *Surface
}

// NewScoreboard allocates the five maps and draws the board art into the band.
//
// It tolerates missing art: Assets records the failure and returns nil, and the band is
// then left white -- the colour an uninitialised GWorld held. The original's equivalent is
// RedAlert(kErrFailedGraphicLoad), which kills the application over a cosmetic asset; see
// docs/IMPROVEMENTS.md 2.12. A caller that cares checks Assets.Err once at the end.
func NewScoreboard(v *View, a *Assets) *Scoreboard {
	s := &Scoreboard{
		v:       v,
		a:       a,
		Board:   NewSurface(int(v.BoardSrc.Wide()), int(v.BoardSrc.Tall())),
		Badge:   a.Strip("badge"),
		Title:   NewSurface(int(v.BoardTSrc.Wide()), int(v.BoardTSrc.Tall())),
		Gliders: NewSurface(int(v.BoardGSrc.Wide()), int(v.BoardGSrc.Tall())),
		Points:  NewSurface(int(v.BoardPSrc.Wide()), int(v.BoardPSrc.Tall())),
	}
	s.ResetBoardArt()
	return s
}

// ResetBoardArt redraws the board picture over the whole band, discarding whatever text
// was composed into it.
//
// It is called at construction and again from the arcade teardown, which is the only place
// the original reloads the PICT. Doing it from a cached surface instead of a fresh
// GetPicture/ReleaseResource cycle is the port's one change here, and it is what turns a
// per-game-over resource load into a memcpy.
//
// hOffset #1: the picture is 1536 wide -- kMaxViewWidth, the widest board the game will
// ever assemble -- and is centred on the band, so a 640-wide band shows its middle 640
// columns. The `< 640` arm's -576 is again a hand-tuned value for the small-screen layout
// and not a formula: it would be -448 if it followed the same rule.
func (s *Scoreboard) ResetBoardArt() {
	art := s.a.Strip("board")
	if art == nil {
		return
	}
	hOffset := (s.v.BoardSrc.Wide() - kMaxViewWidth) / 2
	if s.v.BoardSrc.Right < 640 {
		hOffset = -576
	}
	s.Board.Copy(art, art.Bounds(), Offset(art.Bounds(), hOffset, 0), SrcCopy)
}

// Blacken fills the band black, which is what the arcade teardown blits to the screen
// before putting the art back.
func (s *Scoreboard) Blacken() { s.Board.Fill(s.v.BoardSrc, Black8) }

// Panel clears one of the three text panels and draws a string into it with the
// original's one-pixel drop shadow: black at pen (1,10), white at pen (0,9).
//
// This is the one primitive all five of Scoreboard.c's text refreshes share -- room title,
// glider count, score, and the two quick versions of the latter two -- and it is one
// function here rather than five copies because in the C the five bodies differ only in
// which panel and which string, and they have already drifted: RefreshNumGliders clamps
// its number at zero and QuickGlidersRefresh does not.
//
// **It draws no glyphs yet.** The original's text is the Mac's application font -- Geneva,
// 12 point, bold -- set by TextFont(applFont) in InitScoreboardMap, and that font is a
// system resource rather than something shipped in Glider PRO's own fork, so there is
// nothing to extract and a font has to be authored. Until then a panel is a flat gray
// patch, which is exactly what the C's PaintRect leaves and is invisible against the
// board's recesses: the score is not wrong, it is absent. The font lands in the next
// commit and the pen positions above are already the interface it plugs into.
func (s *Scoreboard) Panel(p *Surface, text string) {
	if p == nil {
		return
	}
	p.Fill(p.Bounds(), kGrayBackgroundColor)
	_ = text
}
