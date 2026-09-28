package netplay

import (
	"errors"
	"io"
	"sync"
	"time"
)

// ErrStalled is what Close returns when the writer could not finish: the other machine stopped
// reading, so this side's last standing and its goodbye were not sent.
var ErrStalled = errors.New("the other machine stopped reading, so this run's last standing " +
	"and goodbye were not sent")

// closeWait is how long Close waits for the writer to send the last standing and the goodbye.
//
// A healthy writer takes microseconds. One that is still in a Write after this is behind a full
// send buffer, and a peer that is reading never lets that happen: a standing is 36 bytes on the
// wire, sent on change, so filling even a small buffer takes many seconds with nothing
// acknowledged. That is a machine that lost power, went to sleep or dropped off the network, or a
// game that froze. Waiting longer for it would freeze this one as well, before the screen that
// says press Esc is drawn. A variable for the tests, which have no two seconds to spend.
var closeWait = 2 * time.Second

// Race is a match in progress, as the game loop sees it: somewhere to put this peer's standing
// every frame, somewhere to read the other peer's, and a result when there is one.
//
// It exists because the loop around Conn is the same loop for every caller and is easy to get
// wrong in ways that only show up against a real opponent. Three things in particular:
//
//   - **The frame loop must never block on the network.** Report hands the standing to a writer
//     goroutine and returns. A 28-byte write into a socket whose peer has stopped reading blocks
//     until the window opens, and a hung opponent slowing this machine's frame rate is exactly the
//     thing a race with two independent worlds was chosen to avoid.
//   - **One reader, and it cannot be the frame loop either.** Conn.Recv blocks, and a game that
//     polled it would stall whenever the opponent was quiet -- which, since standings are sent on
//     change, is most frames.
//   - **A departure has to be folded in the same way on both machines**, or the two peers compute
//     different results from the same race. Abandoned is that rule and Result applies it here, so
//     no caller has to remember to.
type Race struct {
	c *Conn
	m Match

	mu sync.Mutex

	// theirs is the newest standing the peer has sent.
	//
	// **gone and broken are two different facts and conflating them loses races.** gone means the
	// peer will say nothing more: a MsgBye, an end of stream, a protocol error. broken means this
	// side can no longer send. A peer that said goodbye has finished its *run*, not its
	// connection -- it is sitting on a result screen waiting to hear how this one went -- so a
	// goodbye must not stop this side from reporting its own last standing. The first version of
	// this file used one flag for both, and the failure was exactly that: whoever finished second
	// hung up silently, and whoever finished first waited for a standing that was never sent and
	// scored the race a forfeit. Over a pipe the timing hid it; over a socket it happened first
	// try.
	//
	// err carries the reason when the end was not a clean departure.
	theirs Standing
	gone   bool
	bye    bool // gone by a goodbye; see markGone
	broken bool
	err    error

	// pending is the standing waiting to go out and queued is whether there is one. queued
	// rather than comparing against the zero value, because a Standing of all zeroes is a
	// real report: it is what the first frame of a new game looks like.
	//
	// **A pending standing is replaced, not queued behind.** Standings are cumulative -- rooms
	// and score only rise, the frame counter only rises, and a run that has ended stays ended
	// -- so a newer one carries everything the one it replaces carried. Coalescing therefore
	// loses nothing, and the alternative is a queue that grows without bound while the peer is
	// not reading.
	pending Standing
	queued  bool
	last    Standing // the last standing accepted for sending, to suppress duplicates
	sentAny bool

	wake     chan struct{} // buffered 1: "there is something to send"
	stop     chan struct{} // closed by Close
	stopOnce sync.Once
	done     chan struct{} // closed when the writer has finished

	settled    chan struct{} // closed once the other side's fate is known
	settleOnce sync.Once
}

// Start begins a match on a connection Meet has already agreed. It starts two goroutines.
//
// The transport stays the caller's to close, which is Conn's rule and not this type's -- and
// closing it is also what releases the reader, since a blocked Recv cannot be interrupted any
// other way. So the shape of a caller is: Meet, Start, play, Close, then close the socket.
func Start(c *Conn, m Match) *Race {
	r := &Race{
		c:       c,
		m:       m,
		wake:    make(chan struct{}, 1),
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
		settled: make(chan struct{}),
	}
	go r.read()
	go r.write()
	return r
}

// Match is what the handshake agreed.
func (r *Race) Match() Match { return r.m }

// Report queues this peer's standing, if it has changed since the last one queued.
//
// Safe to call every frame and meant to be: the change test is here rather than in the caller
// because "a progress record on every meaningful change" is a property of the protocol, and a
// caller that got it wrong would flood the connection at 30 messages a second without anything
// failing. Cheap enough for the frame loop -- a comparison of one 28-byte struct and, on a change,
// a copy and a non-blocking channel send.
func (r *Race) Report(s Standing) {
	r.mu.Lock()
	if r.sentAny && s == r.last {
		r.mu.Unlock()
		return
	}
	r.last, r.sentAny = s, true
	r.pending, r.queued = s, true
	r.mu.Unlock()

	select {
	case r.wake <- struct{}{}:
	default: // the writer has not taken the last nudge yet, which is all a nudge has to say
	}
}

// Opponent is the other peer's newest standing, and whether the connection has gone.
//
// Both halves are needed by the panel and for different reasons: the standing is what it draws,
// and gone is why it stops changing. A gone peer's standing is returned as it was last sent --
// **not** folded through Abandoned -- because the panel should say what the opponent actually
// reached, and turning that into "left" is the result's business and not the display's.
func (r *Race) Opponent() (Standing, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.theirs, r.gone
}

// Settled is closed once the other side's fate is known: it reported a run that ended, or it left.
// Until then Result returns StillRacing, because it has to.
//
// This is what the screen after a run waits on. A player who finishes the house first is not owed a
// result yet -- the opponent is still flying and may yet get further -- so the game shows "waiting
// for the other player" and waits here. A channel rather than a poll because waiting is all that
// screen does, and because a select gives the caller somewhere to put a timeout and a key press for
// the player who would rather not wait.
//
// Note what does *not* close it: this peer's own run ending. Both sides have to agree on the
// result, so both sides have to wait for the same fact.
func (r *Race) Settled() <-chan struct{} { return r.settled }

// Err is the protocol or transport error that ended the match, or nil. A peer that closed cleanly
// or said goodbye is not an error: it is a forfeit, which Result already accounts for. Nor is a
// write refused after the goodbye (docs/IMPROVEMENTS.md 4.49). One refused after a clean close is,
// because that is how a process killed with nothing unread looks.
func (r *Race) Err() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

// Result is the race as it stands, from this peer's point of view. mine is this peer's own current
// standing; it is a parameter rather than the last one Reported because the caller may want the
// result before the queue has drained, and because the World is the authority on its own run.
func (r *Race) Result(mine Standing) Outcome {
	theirs, gone := r.Opponent()
	if gone {
		theirs = Abandoned(theirs)
	}
	return r.m.Result(mine, theirs)
}

// Close says goodbye and stops the writer. The connection stays the caller's to close, and until
// it is closed the reader goroutine is still parked in Recv.
//
// The pending standing is flushed first, and that ordering is the whole point of the method: the
// last standing of a run is the one the result is computed from, and a Close that raced the writer
// could otherwise hang up with the "finished the house" report still in the queue. The writer
// sends both, in that order, so that nothing else can write between them.
//
// **Close waits for the writer for closeWait and no longer.** A writer stuck in a Write -- behind a
// peer that has stopped reading -- used to hold Close, and Close is called the moment a run ends,
// before the waiting screen and its Esc are drawn: the game froze with nothing on screen, until the
// operating system gave up on the connection, which on Linux is a quarter of an hour. Now Close
// returns ErrStalled, and the writer is released when the caller closes the transport.
//
// A stall is not a result. Nothing is marked, so the other player is not scored as having left: a
// machine that went quiet may be a machine that is about to come back, and silence is still the
// panel's business rather than the result's (docs/IMPROVEMENTS.md 4.37). Any other failure to send
// is recorded rather than returned, and Err is where it is read.
func (r *Race) Close() error {
	first := false
	r.stopOnce.Do(func() { close(r.stop); first = true })
	if !first {
		// Called again -- the result screen and the way out of the game loop both close the
		// race. A second call that found the writer still stuck says so at once rather than
		// spending closeWait again.
		select {
		case <-r.done:
			return nil
		default:
			return ErrStalled
		}
	}
	select {
	case <-r.done:
		return nil
	case <-time.After(closeWait):
		return ErrStalled
	}
}

// read is the one reader. It keeps the newest standing and stops at the first thing that ends a
// match: a goodbye, an end of stream, or an error.
func (r *Race) read() {
	for {
		msg, err := r.c.Recv()
		if err != nil {
			// io.EOF is the peer closing its end, which is a forfeit and not a fault, so
			// it is not reported as an error. Everything else is: a protocol violation or
			// a broken connection is worth putting on the screen next to the result,
			// because a race decided by one is a race somebody will ask about.
			//
			// A peer whose process ends mid-race is usually the second kind and not the
			// first. It dies with this side's standings unread, and a socket closed with
			// data unread is reset rather than closed -- so what arrives is a reset here, or
			// a refused write in the writer, and it is reported. cmd/glidergo's
			// loopback_test.go is where that was found.
			//
			// A reset can also cost the standing the peer sent just before it. Windows
			// discards whatever had arrived unread when the reset comes in. On any system
			// the writer can meet the reset first and settle the race while that standing
			// is still waiting here to be read -- on Linux this reader then gets it and a
			// plain end of stream, because the writer took the error. A caller that takes
			// the result at once, as finishRace does when this side's run has just ended,
			// scores the peer without it (docs/IMPROVEMENTS.md 4.48).
			if errors.Is(err, io.EOF) {
				err = nil
			}
			r.markGone(err)
			return
		}
		switch m := msg.(type) {
		case *Report:
			r.mu.Lock()
			r.theirs = m.Standing
			r.mu.Unlock()
			// A peer whose run has ended keeps its connection open -- there is a result
			// screen on its side too -- so the end of a race is a standing, not a hang-up.
			if m.State.Ended() {
				r.settle()
			}
		case *Bye:
			r.mu.Lock()
			r.bye = true
			r.mu.Unlock()
			r.markGone(nil)
			return
		}
	}
}

// write drains the pending standing whenever there is one, and when Close asks, sends the last one
// and the goodbye. One goroutine, so every standing and the goodbye go out in the order they were
// made.
func (r *Race) write() {
	defer close(r.done)
	for {
		select {
		case <-r.stop:
			// Only a failed send stops the goodbye, and specifically not the peer having gone:
			// see the comment on gone above.
			if r.sendPending() {
				if err := r.c.Bye(r.m.Slot); err != nil {
					r.markBroken(err)
				}
			}
			return
		case <-r.wake:
			if !r.sendPending() {
				return
			}
		}
	}
}

// sendPending sends the pending standing, if there is one, and reports whether this side can
// still send.
func (r *Race) sendPending() bool {
	r.mu.Lock()
	s, queued := r.pending, r.queued
	r.queued = false
	broken := r.broken
	r.mu.Unlock()
	if broken {
		return false
	}
	if !queued {
		return true
	}
	if err := r.c.SendStanding(r.m.Slot, s); err != nil {
		// The writer often finds out first, because the reader is only waiting: a peer whose
		// process has died is a refused write here and, for a moment, nothing over there. A
		// peer whose machine has gone is not even that. The write blocks once the buffer is
		// full, and nothing fails until the operating system gives up (Close, closeWait).
		r.markBroken(err)
		return false
	}
	return true
}

// markGone records that the peer will say nothing more, keeping the first reason. The first,
// because the second is usually a consequence: a reader that got a protocol error and a writer that
// then failed to send describe one event, and the reader's is the one that explains it.
//
// A goodbye is a reason too, though not an error, and what follows it is its consequence. The peer
// has finished its run, and when its player stops waiting and closes the window, this side's next
// write draws a reset and the one after it is refused. Recorded as the race's error, that refusal
// told a player whose opponent had said goodbye that it had not (docs/IMPROVEMENTS.md 4.49).
func (r *Race) markGone(err error) { r.mark(err, false) }

// markBroken records that this side can no longer send, which also means the peer will say nothing
// more -- a socket that will not take 28 bytes is not going to receive any either. It settles the
// race without waiting for the reader, and what the socket received before the failure may still
// be unread then: a caller that takes the result at once scores the peer without it (see read).
func (r *Race) markBroken(err error) { r.mark(err, true) }

func (r *Race) mark(err error, broken bool) {
	r.mu.Lock()
	r.gone = true
	r.broken = r.broken || broken
	if r.err == nil && !r.bye {
		r.err = err
	}
	r.mu.Unlock()
	// Outside the lock, and after the fields are set: settle is what wakes the screen that reads
	// them, and a waiter that got in before them would draw the result of a race it thinks is
	// still running.
	r.settle()
}

// settle closes the settled channel, once, from whichever of the three places gets there first.
func (r *Race) settle() { r.settleOnce.Do(func() { close(r.settled) }) }
