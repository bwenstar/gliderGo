package game

// The four opt-in corrections: 1994 defects a player may choose to have fixed.
//
// Each is one line of game code, each is off by default, and each is off for the same
// reason. Stage 1.8 replays the fidelity corpus and compares this port's sparkle table,
// dirty-rect lists and pixels against the C's; a build that quietly corrected a defect
// could no longer be checked against the thing it is a port of, and "our version looks
// better" is not an answer to "does it behave the same". So the transcription stays
// faithful and the correction is a flag over the top of it.
//
// Why offer them at all: three of the four are visible while playing, and a player who has
// never read Render.c has no way to know they are the original's rather than this port's.
// A mirror room whose candle flame blinks looks like a broken port. See
// docs/IMPROVEMENTS.md 2.19, 2.20, 2.39 and 2.23.
//
// The sites are DrawReflection (render_frame.go) for the first two, switchLinkedObject
// (switches.go) for the third and player.Input.GetInput for the fourth. Each reads its
// flag next to the transcribed line and says which flag it is, so the faithful behaviour
// and the correction are always readable together.
//
// prefs.Fixes is the same three settings as a player's saved choice, and cmd/glidergo
// copies one into the other. They are separate types on purpose: this package must not
// import internal/prefs, because the game does not have preferences -- it has a caller
// that had some.
type Fixes struct {
	// MirrorFlame clips the reflection's back rect to the mirrors it was drawn in.
	//
	// The C registers the *unclipped* rect, so every frame the erase covers ground the
	// draw never touched, and anything inside that ground which redraws itself opaquely
	// rather than registering its own back rect -- a candle flame, a pendulum -- is wiped
	// a frame early and blinks. Clipping makes the erase match the draw, which is what
	// every other compositor in the file already does, and as a side effect stops one
	// glider from spending most of a mirror room's 47 back-rect budget.
	MirrorFlame bool

	// MirrorFoil adds the `!twoPlayerGame` guard that DrawReflection is missing and
	// RenderGlider has.
	//
	// Without it, a mirror in a two-player game draws player one's reflection out of
	// glid2SrcMap, which in a two-player game holds player *two's* sheet rather than the
	// foil sheet the test was reaching for. It costs nothing when there is no mirror, no
	// second player or no foil, which is why it went unnoticed.
	MirrorFoil bool

	// SwitchSparkle drops the second AddSparkle in the switch's prize arm, the one on the
	// uninitialised `bounds`.
	//
	// The puff the player is meant to see comes from RestoreFromSavedMap on the line
	// above; this one lands wherever the stack garbage pointed, which in this port is the
	// zero rect and therefore the top-left corner of the play area. It fires 145 times
	// across the twenty-two shipped houses. Turning it off also gives the sparkle table's
	// three slots back to effects that were asked for on purpose.
	SwitchSparkle bool

	// Player2GiveUp lets player 2 press the abandon key too, rather than only player 1.
	//
	// The odd one out in this struct: the other three correct something a player can see
	// going wrong, and this one corrects something a player cannot do. The give-up key is
	// the only exit from a two-player deadlock, the original binds it to player 1's
	// keyboard alone, and in a game whose whole point is that two people are sitting at
	// one machine that means player 2 has to ask to be rescued. It is opt-in with the rest
	// because it changes what a recorded input stream does. docs/IMPROVEMENTS.md 2.23.
	//
	// The flag reaches its one call site through player.Input, which World.GetInput
	// refreshes from here every frame; see that field for why it is not an Env method.
	Player2GiveUp bool
}
