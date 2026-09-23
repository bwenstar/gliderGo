package netplay

import (
	"errors"
	"io"
	"sync"
)

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

	wake chan struct{} // buffered 1: "there is something to send"
	stop chan struct{} // closed by Close
	done chan struct{} // closed when the writer has finished

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
// or said goodbye is not an error: it is a forfeit, which Result already accounts for.
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
// could otherwise hang up with the "finished the house" report still in the queue.
func (r *Race) Close() error {
	select {
	case <-r.stop:
		<-r.done
		return nil
	default:
	}
	close(r.stop)
	<-r.done

	r.mu.Lock()
	s, queued := r.pending, r.queued
	r.queued = false
	broken := r.broken
	r.mu.Unlock()

	// Only a failed send stops this, and specifically not the peer having gone: see the comment
	// on gone above. Nor is a failure here returned to the caller -- it is recorded, and Err is
	// where it is read. A window closing has nothing useful to do with "the goodbye did not
	// arrive", and the peer's own reader already treats silence as a departure.
	if broken {
		return nil
	}
	if queued {
		if err := r.c.SendStanding(r.m.Slot, s); err != nil {
			r.markBroken(err)
			return nil
		}
	}
	if err := r.c.Bye(r.m.Slot); err != nil {
		r.markBroken(err)
	}
	return nil
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
			r.markGone(nil)
			return
		}
	}
}

// write drains the pending standing whenever there is one. One goroutine, so Conn's own lock is
// never contended on this path; it is there for the Bye that Close sends from elsewhere.
func (r *Race) write() {
	defer close(r.done)
	for {
		select {
		case <-r.stop:
			return
		case <-r.wake:
			r.mu.Lock()
			s, queued := r.pending, r.queued
			r.queued = false
			broken := r.broken
			r.mu.Unlock()
			if broken {
				return
			}
			if !queued {
				continue
			}
			if err := r.c.SendStanding(r.m.Slot, s); err != nil {
				// The writer often finds out first, because the reader is only
				// waiting: a peer whose machine has gone is a failed write here and
				// nothing at all over there until the operating system gives up.
				r.markBroken(err)
				return
			}
		}
	}
}

// markGone records that the peer will say nothing more, keeping the first reason. The first,
// because the second is usually a consequence: a reader that got a protocol error and a writer that
// then failed to send describe one event, and the reader's is the one that explains it.
func (r *Race) markGone(err error) { r.mark(err, false) }

// markBroken records that this side can no longer send, which also means the peer will say nothing
// more -- a socket that will not take 28 bytes is not going to deliver any either, and waiting for
// the reader to reach the same conclusion only delays a result that is already decided.
func (r *Race) markBroken(err error) { r.mark(err, true) }

func (r *Race) mark(err error, broken bool) {
	r.mu.Lock()
	r.gone = true
	r.broken = r.broken || broken
	if r.err == nil {
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
