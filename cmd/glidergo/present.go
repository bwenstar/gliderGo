package main

// A game's presents, and the ones a window would rather have later (platform.Coalescer).
//
// One rule, and this type is the whole of it: a present may be put off, but never past the next
// sleep. The browser backend is the window that asks, because a page is painted only while the
// program sleeps, and a room's wipe -- 116 or 160 presents inside one frame -- froze its page for
// half a second converting and comparing frames nobody could see (docs/IMPROVEMENTS.md 3.10). Put
// off, they cost a copy each, and the one sent before the sleep is the frame the wipe ended on.
//
// What is owed is the frame as it was when it was presented, and not the screen as it is when the
// debt is paid. The two differ: the game draws on Main between presents and does not always
// present what it drew -- an erase is not shown until the RenderFrame after it (screen.go's
// CopyRectWorkToMain), and a game's teardown draws after its last frame. Sending Main as it stood
// at the sleep would show a player a screen the original never showed.
//
// Every sleep play.go makes goes through sleep, which is what makes the rule hold. The other
// presents a game makes -- app.present, for the waits and the high-score dialogs, and the race's
// presentSurface -- are always sent, and each says so to superseded: the screen it sent is newer
// than anything owed, and paying the debt after it would put an older frame back. A game opens
// with a burst of presents and then its banner, so without that the banner would blink out for
// one paint in a browser at the start of every game.

import (
	"time"

	"github.com/bwenstar/gliderGo/internal/platform"
	"github.com/bwenstar/gliderGo/internal/render"
)

// presenter sends a game's frames to its window.
type presenter struct {
	co   platform.Coalescer    // the window, if it asks; nil sends every present
	send func(*render.Surface) // converts a frame and presents it

	// owed is the frame a present put off, as it was then, or nil. spare is its storage, kept
	// from one debt to the next: a wipe puts off a hundred presents, and they share one copy.
	owed, spare *render.Surface
}

func newPresenter(win platform.Window, send func(*render.Surface)) *presenter {
	p := &presenter{send: send}
	p.co, _ = win.(platform.Coalescer)
	return p
}

// present is one present of frame: sent now, or copied and owed to the next sleep.
func (p *presenter) present(frame *render.Surface) {
	if p.co != nil && p.co.Coalesce() {
		if p.spare == nil || p.spare.W != frame.W || p.spare.H != frame.H {
			p.spare = render.NewSurface(frame.W, frame.H)
		}
		copy(p.spare.Pix, frame.Pix)
		p.owed = p.spare
		return
	}
	p.owed = nil
	p.send(frame)
}

// sleep is every sleep a game's host makes: it pays what is owed first.
func (p *presenter) sleep(d time.Duration) {
	p.flush()
	time.Sleep(d)
}

// flush sends what is owed now: before a sleep, and once more when the game is over, so that the
// last frame presented is the last one sent.
func (p *presenter) flush() {
	if f := p.owed; f != nil {
		p.owed = nil
		p.send(f)
	}
}

// superseded forgets what is owed, because something newer has just been sent around the
// presenter. Safe on a nil presenter, which is what there is between games.
func (p *presenter) superseded() {
	if p != nil {
		p.owed = nil
	}
}
