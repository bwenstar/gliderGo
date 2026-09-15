package game

// The two blocking waits -- DelayTicks (Utilities.c:731-736) and WaitForInputEvent
// (Utilities.c:439-478) -- split at the same line DoPause is split at.
//
// # What the original does
//
// `DelayTicks(n)` is the Toolbox `Delay(n)`: the process stops for n sixtieths of a second
// and nothing at all happens, not even a repaint. `WaitForInputEvent(seconds)` is a loop
// that ends on any key or click, on any of Command/Option/Shift/Control being *held*, on
// the application being brought back to the foreground, or on a deadline `seconds` seconds
// after entry -- whichever comes first. It flushes the event queue at both ends, so the
// keystroke that dismissed it is consumed and cannot reach the game, and it returns
// `didResume`: **true only when a resume event ended it**, not when input did.
//
// Three functions use them and every one of them is a screen the player is meant to read:
// the opening banner, the "N stars to go" panel, and the two game-over animations. See
// banner.go and gameover.go.
//
// # What a port can keep
//
// The arithmetic, and none of the blocking. `Delay` in a windowed program is the whole of
// docs/IMPROVEMENTS.md 2.32: for as long as it runs the window answers nothing, so a
// compositor reports the game as hung and a player who resizes or closes it is ignored. So
// the deadline stays here, in ticks, and the *waiting* goes to the host through World.Wait
// -- exactly the split DoPause makes, and for the same reason.
//
// 2.32 asks for one more thing: the count consumed has to be a simulated one, so that a
// headless replay crosses these screens without a clock. That falls out of a **nil hook
// meaning "do not wait at all"**. A fidelity run therefore sees every pixel these functions
// draw and none of their duration, which is what makes a game that ends in death byte-
// comparable between a windowed run and a scripted one.
//
// Two deliberate differences from the original, both in the host's half and both noted
// where the host implements them (cmd/glidergo/play.go):
//
//   - **A held modifier does not end a wait.** The C tests the key *map*, so a player
//     resting a finger on Shift skips the banner and both endings instantly and never
//     learns why. Here it takes a key press, which is what the screens say they want.
//   - **The window keeps repainting.** The C's loop dequeues an update event and drops it
//     without answering, so a window uncovered while one of these is up stays stale until
//     the game ends. The host re-presents instead, which it can do because the port keeps
//     Main and the original did not.

// ticksPerSecond is WaitForInputEvent's own `60L` (Utilities.c:445).
//
// It is 60 and not 60.15, because that is the number in the C: the deadline is computed as
// `TickCount() + 60 * seconds`, so the original's "fifteen seconds" is 900 ticks, which is
// 14.96 real seconds. Ticks are what this port measures durations in everywhere else
// (TicksPerFrame, IdleSplashTicks), so the discrepancy is carried rather than corrected --
// it is a quarter of a frame over the longest wait in the game.
const ticksPerSecond = 60

// Waited is what one pass through World.Wait reported.
//
// Input is "the player touched something", which is the C's `waiting = false` on a keyDown
// or a mouseDown and is what the two game-over animations read to cut themselves short.
// Resumed is `didResume`: the application came back to the foreground, which is the one
// thing WaitForInputEvent returns to its caller.
//
// Neither says anything about quitting. A host that sees its window close sets Quitting and
// clears SwitchedOut directly, the way World.PlayEvent's arm does, and every loop in this
// package that can run long tests Quitting for itself.
type Waited struct {
	Input   bool
	Resumed bool
}

// wait is the one place World.Wait is called, so that the nil case is stated once.
//
// A nil hook returns the zero Waited -- no input, no resume -- which is "the deadline
// expired immediately". That is the answer that makes a headless run deterministic: every
// wait in the game is over before it starts and no screen depends on a clock.
func (w *World) wait(ticks int64, discard bool) Waited {
	if w.Wait == nil {
		return Waited{}
	}
	if ticks < 0 {
		ticks = 0
	}
	return w.Wait(ticks, discard)
}

// FlushEvents is `FlushEvents(everyEvent, 0)`: throw away everything the player has typed
// so far.
//
// It is a zero-tick discarding wait rather than a hook of its own, because the host's loop
// already has to do exactly this on its first pass -- drain the queue, drop the key
// presses, keep acting on a close or a focus change. See World.Wait.
//
// The original calls it at both ends of every WaitForInputEvent and once more at the head
// of each game-over animation. The animations' call is the one that matters: without it, a
// key pressed during the frame that ended the game would be sitting in the queue and would
// abort the ending before its first frame was drawn.
func (w *World) FlushEvents() {
	w.wait(0, true)
}

// DelayTicks is Utilities.c:731-736: stop for howLong sixtieths of a second and answer
// nothing.
//
// Input arriving during it is dropped rather than remembered, which is `Delay` followed by
// `WaitForInputEvent`'s leading FlushEvents at its one call site (Banner.c:239-240). So the
// player cannot dismiss the stars-remaining panel during its first second however hard they
// try, and the port keeps that: it is a second of *reading time*, and shortening it on a
// keystroke the player pressed for some other reason is not the behaviour the panel wants.
func (w *World) DelayTicks(howLong int64) {
	w.wait(howLong, true)
}

// WaitForInputEvent is Utilities.c:439-478: wait up to seconds seconds for the player to
// touch something, and report whether a *resume* is what ended it instead.
//
// The return value is the trap most likely to be sprung by a reader of the C: it is not
// "input ended the wait". Only DisplayStarsRemaining looks at it, and what it does with it
// -- RestoreEntireGameScreen -- makes sense only under that reading: the screen has to be
// rebuilt because another application was in front of it, not because a key was pressed.
//
// `seconds == -1` is the C's wait-for-ever, and **no call site anywhere in the shipped
// source passes it** (verified across GliderPRO/Sources/*.c). It is not supported here: a
// wait with no deadline cannot be crossed by a headless run at all, so the one thing it
// could add is a hang in the fidelity corpus. A negative count clamps to zero in wait().
func (w *World) WaitForInputEvent(seconds int16) bool {
	w.FlushEvents()
	r := w.wait(int64(seconds)*ticksPerSecond, false)
	w.FlushEvents()
	return r.Resumed
}

// pollInput is the two-tick do/while both game-over animations pace themselves with
// (GameOver.c:209-220 and :461-478): hold the frame for as long as it has left of its
// budget, and say whether the player wants out.
//
// It is a separate function from WaitForInputEvent because it is a different thing: the C
// writes this loop out twice, inline, and it neither flushes the queue nor cares about a
// resume. Flushing here would be wrong rather than merely redundant -- it runs once per
// animation frame, so a flush would drop the very key press that is meant to end the
// animation unless it landed inside the same two ticks.
//
// **The deadline is passed as a remaining count, not as a period**, and that is the whole of
// the C's pacing. `nextLoop` is held across passes and reseeded from the clock *after* the
// wait, so a pass that took longer than two ticks to draw does not get an extra two on top:
// the period is max(draw time, 2 ticks). awaitFrame's comment makes the same point about
// the frame limiter, which is the same arrangement.
func (w *World) pollInput(ticks int64) bool {
	return w.wait(ticks, false).Input
}
