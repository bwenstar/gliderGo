package main

// The host's half of the high-score subsystem: the two modal dialogs, the board on the
// screen, and the file both of them end up in.
//
// internal/game decides *whether* a score qualifies to be offered (World.TestHighScore) and
// internal/scores knows what a board is, how it sorts, what the dialogs look like and where
// the file goes. What is left -- and it is all that is left -- is the part only a machine
// with a window and a clock can do: block until the player has typed a name, wait a second,
// wait for a keystroke, and write the file. That is the same seam every other host hook in
// this program sits on (see play.go's five).
//
// Three of the original's blocking primitives appear here, and each is a loop rather than a
// call because a windowed program cannot stop pumping events:
//
//	ModalDialog          HighScores.c:517 -- the entry loop, until Return or Okay
//	Delay(8)             DialogUtils.c:341 -- the Okay button's flash
//	DelayTicks(60) +     HighScores.c:82-83 -- one second, then up to thirty
//	WaitForInputEvent(30)
//
// **The board reaches the screen here, and it did not in 1994.** DoHighScores draws the
// whole thing into the offscreen work map and then returns without blitting it, because both
// of its DissBits calls are commented out and RedrawSplashScreen's copy runs in the wrong
// direction (docs/analysis/scoring.md 7.8). The analysis's instruction to a porter is
// explicit -- treat the intent as CopyRectWorkToMain and say so -- so this draws into the
// game's own surface and presents it, and the screen a player sees is the one 1994 composed
// and threw away.

import (
	"fmt"
	"os"
	"time"

	"glidergo/internal/game"
	"glidergo/internal/house"
	"glidergo/internal/platform"
	"glidergo/internal/render"
	"glidergo/internal/scores"
)

// The three waits, in the units the analysis states them in. A tick is 1/60.15 s
// (docs/analysis/scoring.md 0.1); these are the only places in the host that measure
// anything in ticks rather than frames, because they are the only places the original does.
const (
	tick = time.Second * 100 / 6015

	flashTicks = 8    // Delay(8) between hilite and unhilite
	holdTicks  = 60   // DelayTicks(60): one second of the board with input ignored
	waitTicks  = 1800 // WaitForInputEvent(30): thirty seconds, then it gives up
)

// pollWait is how long a blocking loop sleeps between polls. Two ticks is
// HandlePlayEvent's `sleep = 2`, which is what the original yields for when it has nothing
// to do, and it keeps the audio pump clocked at roughly the rate a game clocks it.
const pollWait = 2 * tick

// highScoreHook builds World.HighScore for one game.
//
// It closes over the game rather than taking it as an argument because the hook's signature
// is the game's (a score and a room count in, a Boolean out) and everything else it needs is
// this machine's: the window to poll, the surface to draw on, the art, the store, and the
// remembered name in the preferences.
//
// `closed` is play.go's own flag, shared by pointer. A window shut while a name is being
// typed has to reach the shell as "the window is gone" and not as "the game ended", or the
// shell comes back up and draws a title screen into a dead window (see play.go's `closed`).
func (a *app) highScoreHook(w *game.World, name string, assets *render.Assets, closed *bool) func(int32, int16) bool {
	return func(score int32, rooms int16) bool {
		// The board as it stands: the side-car if there is one, seeded from the board the
		// house file itself carries, so a score that has been in Slumberland since 1995 is
		// what a new player has to beat.
		board, notes := a.store.Load(name, w.H.HighScores)
		a.reportScoreNotes(name, notes)

		placing := scores.Qualify(&board, score)
		if placing < 0 {
			// Not on the board. The original has an alert for this case -- ALRT 1046, "you
			// are ineligible" -- and never shows it: nothing calls
			// HeyYourPissingAHighScore except the unreachable resume path
			// (docs/analysis/scoring.md 7.12 and 9.8). Silence is what a player of the
			// shipped game got, and it is also the right answer: a dialog to tell somebody
			// they did not win is a dialog nobody wants.
			return false
		}

		// Step 9: the name, pre-filled with the one this installation remembers. That is the
		// original's `highName` from the preferences file (7.5 note 5), which is why five of
		// the shipped houses have a banner reading "Your Message Here" -- the champion
		// accepted the default.
		who := a.ask(w, scores.NamePrompt(score, placing+1, name), a.p.HighName, closed)

		// Steps 11-13: first place, and only first place, gets to change the banner.
		if placing == 0 {
			banner := a.ask(w, scores.BannerPrompt(), a.p.HighBanner, closed)
			a.p.HighBanner = banner
			scores.SetBanner(&board, banner)
		}
		a.p.HighName = who

		// Steps 14-17, all four in one call: the row goes into slot 9 and the board is
		// sorted, which is where `placing` comes back from.
		row := scores.Insert(&board, who, score, rooms, time.Now())

		// Step 18 is `gameDirty = true`, and this is the port's version of it: the board
		// goes into a file of its own instead of into the house. internal/scores' package
		// comment has the argument; the short version is that the 22 shipped houses are not
		// ours to rewrite and a 98 KB whole-file rewrite to persist 292 bytes is one power
		// cut from a destroyed house.
		if err := a.store.Save(name, &board); err != nil {
			// Not fatal, and not silent. The player has just earned the score and the one
			// thing worse than losing it is losing it without being told.
			fmt.Fprintf(os.Stderr, "glidergo: cannot record the high score: %v\n", err)
		}
		// The remembered name and banner, which the original writes at quit
		// (Main.c:223-224). Written now instead, because a game crashed or killed after a
		// high score should not also forget who set it.
		a.saveNameAndBanner()

		// Step 20's DoHighScores, with the blit the original left out.
		a.showBoard(w, name, &board, row, assets, closed)
		return true
	}
}

// ask runs one entry dialog to its end and returns what was typed.
//
// This is ModalDialog's loop (HighScores.c:517-524) with the Toolbox taken out of it. There
// is no Cancel in either dialog and the loop only leaves on item 1, so a player who has
// qualified is going to be asked for a name whether they want to be or not -- and this
// reproduces that, with the one exit the original could not have: a closed window.
//
// The dialog is drawn over the game's own surface, which is what a modal dialog does. The
// screen behind it is whatever the game left there -- the last frame of a lost game, or the
// win animation's -- and it is not restored afterwards, because what comes next is the board
// and then the splash screen, both of which cover it completely.
func (a *app) ask(w *game.World, p *scores.Prompt, initial string, closed *bool) string {
	f := p.Field(initial)

	// The dialog appearing is a sound, kEnergizeSound at kEnergizePriority
	// (HighScores.c:512). It is one of exactly two sounds in this program that play outside
	// a running game.
	a.playSound(scores.EnergizeSound, scores.EnergizePriority)

	for {
		p.Draw(w.Main, f)
		a.present(w)

		for _, ev := range a.win.PollEvents() {
			if ev.Kind == platform.EventQuit {
				// The window is gone, so there is nobody left to press Okay. Take what
				// has been typed -- the score is already earned and the name is the least
				// important part of it -- and let the game end.
				*closed = true
				w.Quitting = true
				w.SwitchedOut = false
				return f.Text()
			}

			switch f.Event(ev, a.win.KeyDown(platform.KeyShift)) {
			case scores.ActionCommit:
				// kCarriageSound, then FlashDialogButton: hilite, eight ticks, unhilite
				// (HighScores.c:461-465).
				a.playSound(scores.CarriageSound, scores.CarriagePriority)
				p.Hilite(w.Main, true)
				a.present(w)
				a.sleepTicks(w, flashTicks)
				p.Hilite(w.Main, false)
				a.present(w)
				return f.Text()

			case scores.ActionTyped, scores.ActionErased, scores.ActionSelect:
				// kTypingSound for everything else, which in the original includes the
				// keys that do nothing at all: the filter's default arm plays it before
				// it has any idea whether the character will fit (7.11.3). A key the
				// field ignores entirely is ActionNone and is silent, which is the one
				// place this differs -- an arrow key that made a typing noise would be
				// feedback for something that did not happen.
				a.playSound(scores.TypingSound, scores.TypingPriority)
			}
		}
		time.Sleep(pollWait)
	}
}

// showBoard is DoHighScores (HighScores.c:58-85): draw the board, hold it for a second, then
// wait up to thirty for a keystroke.
//
// The hold is unconditional and swallows input, which is worth keeping rather than tidying:
// a player who dies with a finger on a key would otherwise never see the board they just got
// onto. The thirty-second limit is the original's too, and it is what makes an unattended
// machine return to its title screen instead of sitting on a board forever.
//
// One difference, and it is a deliberate one. On the Mac the Delay(60) does not drain the
// event queue, so a key pressed during that second was still sitting in it when
// WaitForInputEvent(30) looked -- the wait ended at once and the board flashed past in a
// second. Here the hold discards what arrives, so the second is always a second. That is the
// behaviour the code was written for; the original only failed to get it because Delay and
// GetNextEvent are two unrelated pieces of the Toolbox.
func (a *app) showBoard(w *game.World, name string, b *house.Scores, row int, assets *render.Assets, closed *bool) {
	// The highlight is `lastHighScore`, and here it is the row this game landed in -- which
	// is the only value it can honestly have. The original's is a module global reset once at
	// launch, so its Options > High Scores screen highlights the row from the last qualifying
	// game even in a different house (7.5 note 3); the shell's screen therefore highlights
	// nothing. See internal/shell/scores.go.
	scores.Draw(w.Main, assets, name, b, row)
	a.present(w)

	// DelayTicks(60). Events are pumped and thrown away: this is a Delay, not a wait.
	deadline := time.Now().Add(holdTicks * tick)
	for time.Now().Before(deadline) {
		if a.drainForQuit(w, closed) {
			return
		}
		a.present(w)
		time.Sleep(pollWait)
	}

	// WaitForInputEvent(30). Any key ends it; so does the window closing. The original also
	// takes a mouse click and any of the four modifier keys being held, and this port reports
	// no pointer events at all (see internal/scores' ExitWord, which is why the footer says
	// "Hit a Key to Exit" rather than the original's "Click Mouse or Hit a Key to Exit").
	deadline = time.Now().Add(waitTicks * tick)
	for time.Now().Before(deadline) {
		for _, ev := range a.win.PollEvents() {
			switch {
			case ev.Kind == platform.EventQuit:
				*closed = true
				w.Quitting = true
				w.SwitchedOut = false
				return
			case ev.Kind == platform.EventKeyDown && !ev.Repeat:
				return
			}
		}
		a.present(w)
		time.Sleep(pollWait)
	}
}

// ---------------------------------------------------------------------------
// The small host services the three loops above need
// ---------------------------------------------------------------------------

// present pushes the surface the dialogs and the board are drawn into.
//
// It is not World.Present: that one is wrapped by -frames (wrapPresentLimit), so calling it
// from here would let a screen the player is reading decide the game is over. What it needs
// to do is the two things play.go's Present does -- blit and clock the pump -- and nothing
// else.
func (a *app) present(w *game.World) {
	if a.win == nil {
		return
	}
	w.Main.ToBGRX(a.fb.Pix, a.fb.Stride)
	if err := a.win.Present(a.fb); err != nil {
		// A dead window ends the wait rather than spinning on it. The caller's next poll
		// reports the window closed, which is true.
		w.Quitting = true
	}
	a.pump.ClockTick()
}

// sleepTicks is Delay: a wait that keeps the audio moving. The original's Delay stops the
// world; here the mixer is a pipe to another process and a pause in the clock is a gap in
// the sound.
func (a *app) sleepTicks(w *game.World, n int) {
	deadline := time.Now().Add(time.Duration(n) * tick)
	for time.Now().Before(deadline) {
		a.pump.ClockTick()
		time.Sleep(pollWait)
	}
}

// drainForQuit pumps events, discards them, and reports whether the window closed. It is
// FlushEvents with one exception -- the exception being the only event a program may not
// ignore.
func (a *app) drainForQuit(w *game.World, closed *bool) bool {
	for _, ev := range a.win.PollEvents() {
		if ev.Kind == platform.EventQuit {
			*closed = true
			w.Quitting = true
			w.SwitchedOut = false
			return true
		}
	}
	return false
}

// playSound is PlayPrioritySound for the two sounds that are not part of a game. A build
// with no audio player, or -sound=false, has no engine and plays nothing.
func (a *app) playSound(which, priority int16) {
	if a.eng == nil {
		return
	}
	a.eng.PlayPrioritySound(which, priority)
}

// saveNameAndBanner persists the two remembered strings. Nowhere to write is not an error:
// -prefs none is a deliberate choice and a machine with no configuration directory is a
// real one (see loadPrefs).
func (a *app) saveNameAndBanner() {
	if !a.canSave {
		return
	}
	if err := a.p.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "glidergo: cannot remember the high-score name: %v\n", err)
	}
}

// reportScoreNotes puts a board's repairs on stderr.
//
// Store.Load never fails -- a truncated, hand-edited or half-restored side-car yields a
// playable board plus one note per thing that had to be worked around -- and these are the
// notes. They go to stderr and not to the player: somebody who edited the file wants to know
// what was ignored, and somebody who did not cannot act on it.
func (a *app) reportScoreNotes(name string, notes []string) {
	for _, n := range notes {
		fmt.Fprintf(os.Stderr, "glidergo: %s's high scores: %s\n", name, n)
	}
}
