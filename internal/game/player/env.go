package player

// Env is everything outside the glider that the glider's own handlers reach for.
//
// In the original these are all free functions over file-scope globals, so the
// player code and the room code are one mutually recursive tangle: MoveGliderUpStairs
// calls MoveRoomToRoom, which rebuilds the room, which calls back into
// FlagGliderNormal. Reproducing that in Go with package-level state would make the
// player untestable and would force internal/render and internal/game/player into
// one package. So the calls the player makes outward are collected here, named
// after the original functions, and stage 1.5 implements them for real.
//
// Of the 46 methods, the function-shaped ones each name a C free function that
// Player.c, Modes.c or Input.c really calls. The remaining ~20 are accessors --
// BatteryTotal, Tile, TopOpen, LeftThresh, TwoPlayerGame and the like -- which
// stand in for the C's direct reads and writes of a file-scope global, since
// there is no function to name. Nothing was invented to make the interface tidy,
// and nothing the player only reads from its own struct appears here.
type Env interface {
	// ---- sound -------------------------------------------------------------
	// PlayPrioritySound is the original's only sound entry point from this code.
	// Priority decides what a busy channel drops; see docs/analysis/sound.md.
	PlayPrioritySound(sound, priority int16)

	// ---- inventory ---------------------------------------------------------
	// BatteryTotal is signed and is the single counter for both power-ups:
	// positive is battery charges, negative is helium. That is why holding the
	// battery key with a negative total gives lift rather than thrust, and why
	// DoBatteryEngaged decrements while DoHeliumEngaged increments -- both walk
	// the same number toward zero (Input.c:138, :163).
	BatteryTotal() int16
	SetBatteryTotal(n int16)
	BandsTotal() int16
	SetBandsTotal(n int16)
	FoilTotal() int16
	// SetFoilTotal exists because GliderHitTop spends a sheet on a side impact
	// (Interactions.c:82). It is the only writer outside the pickup code.
	SetFoilTotal(n int16)

	// ShowFoil controls which glider artwork is loaded. The original swaps the
	// whole sprite sheet rather than tinting, so this is a graphics-state change
	// and not just a flag (Player.c:1144-1152).
	SetShowFoil(on bool)

	// ---- scoreboard and dirty rects ---------------------------------------
	//
	// `flash` is the C's own parameter name and it means **draw the blank badge cell**:
	// it is the hide half of the low-inventory blink, not a "force a redraw" flag. Every
	// call from this package passes false, because a pickup wants the badge lit; the true
	// case has one caller, the blink schedule in Scoreboard.c's HandleDynamicScoreboard.
	// No implementation is allowed to skip the blit, cache the last value or test for a
	// change -- the blink is made of unconditional redraws.
	QuickBatteryRefresh(flash bool)
	QuickBandsRefresh(flash bool)
	QuickFoilRefresh(flash bool)
	RefreshScoreboard(mode int16)

	// AddRectToWorkRects and CopyRectWorkToMain are the dirty-rect calls the mode
	// changes make directly. They are how a shrinking glider erases the pixels it
	// is about to stop covering (Modes.c:87-100, Player.c:601-606).
	//
	// **They take SCREEN coordinates, not room-local ones**, which is the contract
	// Render.c:65 and :713 have. Every rect the glider owns is room-local, so every
	// call site here offsets by PlayOriginH/V first -- exactly as the C does, where
	// the QOffsetRect is written out at each of the six call sites (Modes.c:90, :95,
	// Player.c:602, :606, Play.c:717, :727). Folding the offset into the
	// implementation instead would be tidier and wrong: Input.c:70 calls
	// CopyRectWorkToMain(&workSrcRect) with a rect that is *already* in screen
	// coordinates, so there is no single coordinate system the callee could assume.
	AddRectToWorkRects(r Rect)
	CopyRectWorkToMain(r Rect)

	// PlayOriginH and PlayOriginV are the C globals of the same name: where the
	// 512x322 room sits inside the window. They exist so the two calls above can be
	// handed screen rects.
	PlayOriginH() int16
	PlayOriginV() int16

	// ---- room geometry ----------------------------------------------------
	// GetUpStairsRightEdge and GetDownStairsLeftEdge find the staircase in the
	// current room and give the x the glider is clipped against as it walks
	// behind it.
	GetUpStairsRightEdge() int16
	GetDownStairsLeftEdge() int16

	// IsShadowVisible and SetShadowVisible are a *mixed* pair, and that is not an
	// accident of naming -- it is what the one call site needs.
	//
	// The original has two nearly identically spelled things. Room.c:1103's
	// IsShadowVisible() is a pure predicate that recomputes the answer from the
	// room's background. Player.c:53's `shadowVisible` is a Boolean global. They
	// are not interchangeable, and the whole C tree contains exactly one read of
	// the global (Render.c:465, the shadow draw) and four writes: three of the form
	// `shadowVisible = IsShadowVisible()` -- DrawLocale RoomGraphics.c:126,
	// RedrawRoomLighting :459, and the glider placement at Modes.c:361 -- and one
	// that forces it false, Player.c:1286, the frame a glider starts being pulled
	// into a shredder, so its shadow does not go on lying on the floor while it is
	// dragged off the floor plane.
	//
	// So the two disagree from that frame until the glider is next placed, about a
	// second of real play, with Render.c:465 reading the global throughout.
	//
	// Which means: **IsShadowVisible is the predicate and SetShadowVisible writes
	// the global.** modes.go:262 is Modes.c:361 transcribed --
	// `e.SetShadowVisible(e.IsShadowVisible())` -- and handle.go:685 is
	// Player.c:1286. Nothing in this package reads the global at all; the renderer
	// does. Wiring the getter to the global instead would turn that assignment into
	// a self-assignment, so the answer would never be recomputed on a room change
	// and gliders would keep their shadows on rooftops.
	IsShadowVisible() bool
	SetShadowVisible(v bool)
	// HasMirror is true in a room with a mirror object, which doubles every dirty
	// rect at a fixed offset (Modes.c:92-97).
	HasMirror() bool

	// TopOpen and BottomOpen say whether this room's ceiling and floor let the
	// glider through. LeftThresh and RightThresh are both a coordinate and a flag:
	// the escape checks compare the glider against them, and compare them against
	// LeftWallLimit/RightWallLimit to decide whether there is a wall there at all
	// (Interactions.c:513, :603).
	TopOpen() bool
	BottomOpen() bool
	LeftThresh() int16
	RightThresh() int16

	// Background is thisBackground, the room background's PICT resource ID. The
	// escape logic compares it against Dirt and Roof, which change the ceiling,
	// floor and roof rules entirely.
	Background() int16

	// Tile is thisTiles[i] for i in 0..NumTiles-1, the room's eight tile numbers.
	// In a Dirt or Roof room these are read as a crude collision map; callers
	// range-check i themselves, as the original does.
	Tile(i int16) int16

	// ---- transitions ------------------------------------------------------
	// The four ways a glider leaves a room. Each rebuilds the room around the
	// glider and re-enters it, so each will call back into the mode-entry
	// functions above. `where` is Above, Below, ToRight or ToLeft.
	MoveRoomToRoom(g *Glider, where int16)
	MoveDuctToDuct(g *Glider)
	MoveMailToMail(g *Glider)
	TransportRoomToRoom(g *Glider)
	SetTakingTheStairs(v bool)

	// ---- objects ----------------------------------------------------------
	// AddBand fires a rubber band and reports false when the band array is full,
	// which is what makes the shot not cost a band (Input.c:352-361).
	AddBand(g *Glider, h, v int16, facing bool) bool
	AddAShreddedGlider(r Rect)
	// FlagStillOvers marks every hot spot the glider currently overlaps as
	// already-stood-on, which SUPPRESSES it rather than triggering it. The
	// dispatcher's one-shot cases are all shaped `if (!who->stillOver) { do it;
	// who->stillOver = true; }` (Interactions.c:1344, :1595, :1616), so
	// pre-setting the flag is what stops the arrival frame from counting as a
	// fresh contact (Interactions.c:1723-1725).
	//
	// It has exactly one caller and it is not room entry: FinishGliderDuctingIn
	// (Player.c:954), a glider dropping out of a ceiling duct. So a glider that
	// lands on a switch by falling out of a duct does not flip it, while one that
	// walks onto the same switch does.
	FlagStillOvers(g *Glider)

	// ---- life and death ---------------------------------------------------
	// OffAMortal spends a life. It is the end of both fade-out and shredding.
	OffAMortal(g *Glider)
	ForceKillGlider()

	// ---- two-player state -------------------------------------------------
	// These four globals drive the "first one out waits in limbo for the other"
	// dance in every transit handler. In a one-player game TwoPlayerGame is false
	// and none of the rest is read.
	TwoPlayerGame() bool
	OnePlayerLeft() bool
	PlayerDead() bool
	OtherPlayerEscaped() int16
	SetOtherPlayerEscaped(v int16)
	// Survivor returns the glider of whichever player is not dead. The
	// one-player-left branch of every transit handler moves it instead of the
	// glider that triggered the transition, so a dead player's corpse does not
	// drag the survivor's view around (Player.c:349-352: `if (playerDead ==
	// kPlayer1) MoveRoomToRoom(&theGlider2, ...) else MoveRoomToRoom(&theGlider,
	// ...)` -- it moves the one PlayerDead does not name).
	Survivor() *Glider
	SetFirstPlayer(which bool)
	// SaidFollow counts how many times the "follow me" prompt has played, capped
	// at MaxSaidFollow for the whole game (Modes.c:462-466).
	SaidFollow() int16
	SetSaidFollow(n int16)

	// ---- pause and input --------------------------------------------------
	// DoPause blocks until the player unpauses. It is called from inside GetInput
	// in the original, which is why a paused game does not advance a frame.
	DoPause()
	DoCommandKey()
}
