package player

// NopEnv, the package's test double for Env. It is in a _test.go file on purpose:
// nothing outside a test may depend on it, and the compiler is what enforces that. A
// convenience implementation that ships in the library is an invitation for production
// code to embed it, and an embedded NopEnv is a whole subsystem silently doing nothing.
//
// It was in env.go through 1.5a, where the package had no other Env implementation to
// develop against. game.World is that implementation now (see the interface assertion at
// the bottom of internal/game/env.go), so the reason is gone.

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
	OriginH, OriginV     int16 // what PlayOriginH/V return; zero is fine for a test
	Dirtied              []Rect
	Copied               []Rect
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

func (e *NopEnv) AddRectToWorkRects(r Rect) { e.Dirtied = append(e.Dirtied, r) }
func (e *NopEnv) CopyRectWorkToMain(r Rect) { e.Copied = append(e.Copied, r) }

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
func (e *NopEnv) PlayOriginH() int16           { return e.OriginH }
func (e *NopEnv) PlayOriginV() int16           { return e.OriginV }
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
