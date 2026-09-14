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
	QuickBatteryRefresh(force bool)
	QuickBandsRefresh(force bool)
	QuickFoilRefresh(force bool)
	RefreshScoreboard(mode int16)

	// AddRectToWorkRects and CopyRectWorkToMain are the dirty-rect calls the
	// mode changes make directly, in room-local coordinates -- the implementation
	// adds playOriginH/V. They are how a shrinking glider erases the pixels it is
	// about to stop covering (Modes.c:87-100, Player.c:601-606).
	AddRectToWorkRects(r Rect)
	CopyRectWorkToMain(r Rect)

	// ---- room geometry ----------------------------------------------------
	// GetUpStairsRightEdge and GetDownStairsLeftEdge find the staircase in the
	// current room and give the x the glider is clipped against as it walks
	// behind it.
	GetUpStairsRightEdge() int16
	GetDownStairsLeftEdge() int16

	// IsShadowVisible and SetShadowVisible are the *cached* `shadowVisible`
	// global, not the predicate of the same name.
	//
	// The original has both, spelled almost identically, and they are not
	// interchangeable. Room.c's IsShadowVisible() recomputes the answer from the
	// room's background. RoomGraphics.c's `shadowVisible` is a Boolean global with
	// exactly four writers: three set it from that predicate (DrawLocale
	// RoomGraphics.c:126, RedrawRoomLighting :459, and the glider placement at
	// Modes.c:361), and the fourth forces it false -- Player.c:1286, the frame a
	// glider starts being pulled into a shredder, so that its shadow does not go
	// on lying on the floor while it is dragged off the floor plane.
	//
	// So the two disagree from that frame until the glider is next placed, which
	// is a second or so of real play, and Render.c:465 is reading the global the
	// whole time. This pair is the global. A port that wired it to the predicate
	// would leave a shadow under a glider being shredded.
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

// NopEnv implements Env by doing nothing and reporting empty. Embed it to
// implement only the methods a given caller or test actually needs; the compiler
// will then not complain when stage 1.5 adds a method, which is deliberate --
// the alternative is that every test breaks on every extension.
//
// Counters are fields rather than constants so a test can set up an inventory
// without implementing the interface.
type NopEnv struct {
	Battery, Bands, Foil int16
	Shadow               bool
	Mirror               bool
	UpStairsRight        int16
	DownStairsLeft       int16
	BandAdded            bool // what AddBand returns
	Follow               int16
	Sounds               []int16
	Transitions          []string
	MortalsSpent         int
	Pauses, Commands     int
	Kills                int
	Shredded             []Rect

	// Alive is what Survivor returns. Nil is fine as long as TwoPlayerGame is
	// false, which it is for this type: the branch that reads it is unreachable.
	Alive *Glider

	// Room geometry. The zero value is a sealed room with a Marble background and
	// all-zero tiles, i.e. every boundary solid -- which is the safest default for a
	// test that is not about boundaries.
	Top, Bottom bool
	Left, Right int16
	Back        int16
	Tiles       [NumTiles]int16
}

func (e *NopEnv) PlayPrioritySound(sound, priority int16) {
	e.Sounds = append(e.Sounds, sound)
}

func (e *NopEnv) BatteryTotal() int16          { return e.Battery }
func (e *NopEnv) SetBatteryTotal(n int16)      { e.Battery = n }
func (e *NopEnv) BandsTotal() int16            { return e.Bands }
func (e *NopEnv) SetBandsTotal(n int16)        { e.Bands = n }
func (e *NopEnv) FoilTotal() int16             { return e.Foil }
func (e *NopEnv) SetShowFoil(on bool)          {}
func (e *NopEnv) QuickBatteryRefresh(bool)     {}
func (e *NopEnv) QuickBandsRefresh(bool)       {}
func (e *NopEnv) QuickFoilRefresh(bool)        {}
func (e *NopEnv) RefreshScoreboard(int16)      {}
func (e *NopEnv) AddRectToWorkRects(Rect)      {}
func (e *NopEnv) CopyRectWorkToMain(Rect)      {}
func (e *NopEnv) GetUpStairsRightEdge() int16  { return e.UpStairsRight }
func (e *NopEnv) GetDownStairsLeftEdge() int16 { return e.DownStairsLeft }
func (e *NopEnv) IsShadowVisible() bool        { return e.Shadow }
func (e *NopEnv) SetShadowVisible(v bool)      { e.Shadow = v }
func (e *NopEnv) HasMirror() bool              { return e.Mirror }
func (e *NopEnv) SetFoilTotal(n int16)         { e.Foil = n }
func (e *NopEnv) TopOpen() bool                { return e.Top }
func (e *NopEnv) BottomOpen() bool             { return e.Bottom }
func (e *NopEnv) LeftThresh() int16            { return e.Left }
func (e *NopEnv) RightThresh() int16           { return e.Right }
func (e *NopEnv) Background() int16            { return e.Back }
func (e *NopEnv) Tile(i int16) int16           { return e.Tiles[i] }

func (e *NopEnv) MoveRoomToRoom(g *Glider, where int16) {
	e.Transitions = append(e.Transitions, "room")
}
func (e *NopEnv) MoveDuctToDuct(g *Glider) { e.Transitions = append(e.Transitions, "duct") }
func (e *NopEnv) MoveMailToMail(g *Glider) { e.Transitions = append(e.Transitions, "mail") }
func (e *NopEnv) TransportRoomToRoom(g *Glider) {
	e.Transitions = append(e.Transitions, "transport")
}
func (e *NopEnv) SetTakingTheStairs(bool) {}

func (e *NopEnv) AddBand(g *Glider, h, v int16, facing bool) bool { return e.BandAdded }
func (e *NopEnv) AddAShreddedGlider(r Rect)                       { e.Shredded = append(e.Shredded, r) }
func (e *NopEnv) FlagStillOvers(*Glider)                          {}
func (e *NopEnv) OffAMortal(*Glider)                              { e.MortalsSpent++ }
func (e *NopEnv) ForceKillGlider()                                { e.Kills++ }

func (e *NopEnv) TwoPlayerGame() bool         { return false }
func (e *NopEnv) OnePlayerLeft() bool         { return false }
func (e *NopEnv) PlayerDead() bool            { return false }
func (e *NopEnv) OtherPlayerEscaped() int16   { return NoOneEscaped }
func (e *NopEnv) SetOtherPlayerEscaped(int16) {}
func (e *NopEnv) Survivor() *Glider           { return e.Alive }
func (e *NopEnv) SetFirstPlayer(bool)         {}
func (e *NopEnv) SaidFollow() int16           { return e.Follow }
func (e *NopEnv) SetSaidFollow(n int16)       { e.Follow = n }
func (e *NopEnv) DoPause()                    { e.Pauses++ }
func (e *NopEnv) DoCommandKey()               { e.Commands++ }
