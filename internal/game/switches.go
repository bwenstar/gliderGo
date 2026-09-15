package game

// HandleSwitches (Interactions.c:985-1153) -- the switches.
//
// A hundred and seventy lines over the nine switch types, and structurally it is
// FireTrigger's bigger sibling: resolve the switch's link, and if the target is in the
// locale poke it live, otherwise write the house copy and let the player find the change
// when they arrive. Everything a switch can do, a trigger can do, and both go through
// SetObjectState -- which is already ported, so the remaining work here is the dispatch
// and the animation of the switch's own lever.
//
// It is stubbed for the same reason HandleRewards is: most of its arms end in a dynamic
// object's Trigger* function or in RedrawRoomLighting plus a scoreboard refresh, and two
// of those three are still stubs. See rewards.go.
//
// **The consequence of the stub: switches do nothing, including light switches.** A room
// composed dark stays dark and a room composed lit stays lit. That is more visible than
// the inert prizes, because a few of the shipped houses use a light switch as the way to
// see a room's obstacles -- so those rooms are playable but hard. No exit is gated on a
// switch, so no house becomes uncompletable.
//
// Note it takes only the hot spot and not the glider: nothing a switch does depends on
// which player threw it. That is why the two-player escape protocol has no branch here.
//
// Landing in 1.5d, rewards and switches.

// HandleSwitches is Interactions.c:985-1153. See the file comment: stubbed to 1.5d.
func (w *World) HandleSwitches(who *HotObject) {}
