package game

// HandleRewards (Interactions.c:756-979) -- the prizes.
//
// Two hundred and twenty lines, and the largest single function in Interactions.c: one arm
// per collectable, and the collectables are where most of the game's inventory, scoring
// and sound live. The clocks, the paper, the batteries, the bands, the grease, the foil,
// the stars, the helium, the sliders and the invisible bonus all pass through it, and each
// arm has to award points, update a counter, refresh a piece of the scoreboard, play a
// sound, switch the object off, promote the patched background into the back map, and
// start a flying-points animation -- in that order, because the animation reads the score
// it was just given.
//
// It is stubbed here rather than transcribed because three of those seven steps do not
// exist yet: the flying points and the sparkles are 1.5c's, the scoreboard's per-frame
// half is this stage's second commit, and StarSprite's collection count belongs with the
// banner. Porting it against three stubs would mean writing it twice.
//
// **The consequence of the stub is specific and worth stating: prizes are inert.** A
// glider flies through a battery and nothing happens -- no points, no charge, no sound,
// and the battery stays on screen because the arm that switches it off is in here too. The
// game is completable without them, since nothing gates a room's exits on a prize, so this
// does not block the playable milestone. It does mean a 1.5b playthrough has a score of
// nothing but room visits.
//
// Its argument list is the original's, so filling it in touches nothing else.
//
// Landing in 1.5d, rewards and switches.

import "glidergo/internal/game/player"

// HandleRewards is Interactions.c:756-979. See the file comment: stubbed to 1.5d.
func (w *World) HandleRewards(thisGlider *player.Glider, who *HotObject) {}
