package netplay

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

// This file is docs/PLAN.md Stage 3's acceptance criteria, run over a pipe instead of a LAN:
// "two processes on this host race to completion; killing the guest mid-race leaves the host in
// a defined state; a house-set mismatch is rejected with a clear message." The third is in
// handshake_test.go, where the mismatch is refused; the other two are whole races, and a whole
// race is small enough to write down.

// played is everything one peer knows when its race is over -- which is everything a result
// screen is drawn from.
type played struct {
	m      Match
	mine   Standing // this peer's own last standing
	theirs Standing // the other peer's, as last heard, after Abandoned
	result Outcome
	err    error
}

// play is one side of a race: handshake, report a scripted run, listen to the other side, and
// end with a result. It is deliberately the shape the game loop will have -- one reader goroutine
// because Conn allows exactly one, sends from the caller's goroutine because that is where the
// simulation is -- so that what it proves is about this package and not about the test.
//
// The script's last standing has to be an ended one. A peer that stops reporting while still
// racing is the *other* test, and it does not use this function.
func play(e *end, local Hello, script []Standing) played {
	m, err := Meet(e.Conn, local)
	if err != nil {
		return played{err: err}
	}

	type received struct {
		msg Msg
		err error
	}
	// Buffered, so that the reader can always deliver its last message and finish. An
	// unbuffered channel would leave it parked on a send after the loop below has stopped
	// listening, and a goroutine parked on a test's channel outlives the test.
	in := make(chan received, 64)
	go func() {
		defer close(in)
		for {
			msg, err := e.Recv()
			in <- received{msg, err}
			if err != nil {
				return
			}
		}
	}()

	p := played{m: m}
	for _, s := range script {
		p.mine = s
		if err := e.SendStanding(m.Slot, s); err != nil {
			p.err = err
			return p
		}
	}
	// Our run is over and the connection is not: say goodbye, keep reading. A real peer closes
	// the socket after this; here the pipes belong to the test.
	if err := e.Bye(m.Slot); err != nil {
		p.err = err
		return p
	}

listen:
	for {
		select {
		case r, ok := <-in:
			switch {
			case !ok || r.err != nil:
				// Includes the io.EOF of a peer that was killed, which is the point of
				// Abandoned: whether this is a forfeit depends on what it last reported,
				// not on how the connection ended.
				p.theirs = Abandoned(p.theirs)
				break listen
			default:
				switch v := r.msg.(type) {
				case *Report:
					p.theirs = v.Standing
				case *Bye:
					p.theirs = Abandoned(p.theirs)
					break listen
				}
			}
		case <-time.After(10 * time.Second):
			p.err = errors.New("the other side never finished")
			break listen
		}
	}

	p.result = m.Result(p.mine, p.theirs)
	return p
}

// A run of each kind, as two scripts. The frames only ever go up, which is Conn.checkSlot's rule
// and is also what a gameFrame does.
var (
	hostRun = []Standing{
		{State: Racing, Rooms: 1, Room: 0, Mortals: 2, Frame: 30},
		{State: Racing, Rooms: 7, Room: 11, Floor: 1, Mortals: 2, Score: 400, Frame: 900},
		{State: Racing, Rooms: 18, Room: 42, Floor: 3, Mortals: 1, Score: 1200, Frame: 4000},
		{State: Died, Rooms: 30, Room: 51, Floor: 3, Mortals: -1, Score: 2500, Frame: 9000},
	}
	guestRun = []Standing{
		{State: Racing, Rooms: 2, Room: 1, Mortals: 2, Frame: 45},
		{State: Racing, Rooms: 9, Room: 20, Floor: 2, Mortals: 2, Score: 100, Frame: 1500},
		{State: Finished, Rooms: 12, Room: 31, Floor: 2, Mortals: 2, Score: 300, Frame: 3000},
	}
)

func TestTwoPeersRaceToTheEnd(t *testing.T) {
	a, b := pair(t)
	ha, hb := hello(7, houseHashA), hello(9, houseHashA)

	done := make(chan played, 2)
	go func() { done <- play(a, ha, hostRun) }()
	go func() { done <- play(b, hb, guestRun) }()
	first, second := <-done, <-done
	if first.err != nil || second.err != nil {
		t.Fatalf("the race did not finish: %v, %v", first.err, second.err)
	}

	host, guest := first, second
	if host.m.Slot != 0 {
		host, guest = second, first
	}
	if host.m.Slot != 0 || guest.m.Slot != 1 {
		t.Fatalf("slots are %d and %d", host.m.Slot, guest.m.Slot)
	}

	// **The property the whole mode rests on**: two peers, no authority, and one result. Each
	// side computed it from its own run and what it heard of the other's.
	if host.result != guest.result {
		t.Errorf("the two sides disagree: host says %s, guest says %s",
			host.result, guest.result)
	}
	want := Outcome{Slot: 1, Reason: ByFinish}
	if host.result != want {
		t.Errorf("result = %s, want %s -- the guest finished the house and the host died "+
			"further in (Winner's rule 3)", host.result, want)
	}
	// And specifically not a forfeit. Both sides sent a MsgBye after their run ended, which
	// is what happens at the end of every ordinary race; if that counted as leaving, every
	// race would be decided by who hung up second. Abandoned is the rule that prevents it.
	if host.result.Reason == ByForfeit {
		t.Error("a goodbye after a finished run was scored as a forfeit")
	}

	// Each side heard the other's whole run, ending with the standing it ended on.
	if host.theirs != guestRun[len(guestRun)-1] {
		t.Errorf("the host's last word on the guest was %+v, want %+v",
			host.theirs, guestRun[len(guestRun)-1])
	}
	if guest.theirs != hostRun[len(hostRun)-1] {
		t.Errorf("the guest's last word on the host was %+v, want %+v",
			guest.theirs, hostRun[len(hostRun)-1])
	}
	// Mortals came across as a negative number, which is the value World has when the run
	// ended in Died (Play.c:408's `Mortals < 0`). Signed on the wire for this reason.
	if host.theirs.Mortals != 2 || guest.theirs.Mortals != -1 {
		t.Errorf("gliders left came across as %d and %d, want 2 and -1",
			host.theirs.Mortals, guest.theirs.Mortals)
	}
}

func TestKillingTheGuestMidRaceLeavesTheHostWithAResult(t *testing.T) {
	// docs/PLAN.md Stage 3's second acceptance clause, and the reason it is worth a test of
	// its own: the host is *still flying* when the guest's process dies. There is no standing
	// to compare against, no run to wait for, and nothing to time out on -- so the defined
	// state has to come from the last thing the guest said plus the fact that it stopped
	// saying anything.
	a, b := pair(t)
	ha, hb := hello(7, houseHashA), hello(9, houseHashA)

	guestReady := make(chan Match, 1)
	guestFailed := make(chan error, 1)
	go func() {
		m, err := Meet(b.Conn, hb)
		if err != nil {
			guestFailed <- err
			return
		}
		// Two standings and then the process is gone: no MsgBye, no final standing, no
		// warning. b.kill() closes the writing end, which is what the other side sees.
		for _, s := range guestRun[:2] {
			if err := b.SendStanding(m.Slot, s); err != nil {
				guestFailed <- err
				return
			}
		}
		guestReady <- m
		b.kill()
	}()

	m, err := Meet(a.Conn, ha)
	if err != nil {
		t.Fatalf("the host could not meet the guest: %v", err)
	}
	select {
	case err := <-guestFailed:
		t.Fatalf("the guest failed before it could be killed: %v", err)
	case <-guestReady:
	case <-time.After(10 * time.Second):
		t.Fatal("the guest never got going")
	}

	// The host reads until the connection goes, which is all the notice it gets.
	var theirs Standing
	var readErr error
	for {
		msg, err := a.Recv()
		if err != nil {
			readErr = err
			break
		}
		if r, ok := msg.(*Report); ok {
			theirs = r.Standing
		}
	}
	if readErr != io.EOF { //nolint:errorlint // a killed peer must arrive as the sentinel
		t.Errorf("the host saw %v (%T), want the plain io.EOF of a peer that stopped",
			readErr, readErr)
	}
	if theirs != guestRun[1] {
		t.Errorf("the host's last word on the guest was %+v, want %+v", theirs, guestRun[1])
	}

	// Still in the air, and there is already a result.
	mine := hostRun[1]
	if mine.State.Ended() {
		t.Fatal("this test is meaningless unless the host is still racing")
	}
	got := m.Result(mine, Abandoned(theirs))
	want := Outcome{Slot: 0, Reason: ByForfeit}
	if got != want {
		t.Errorf("result = %s, want %s", got, want)
	}
	// The guest was ahead on the metric when it died -- 9 rooms to 7 -- and that is exactly
	// why the forfeit is checked first. A player who is losing must not be able to improve
	// the result by pulling the plug.
	if theirs.Rooms <= mine.Rooms {
		t.Errorf("the guest had %d rooms and the host %d; the test only bites if the guest "+
			"was winning when it quit", theirs.Rooms, mine.Rooms)
	}
}

// runRace is play() again, with the Race type doing the parts play() spelled out. Both exist on
// purpose: play() is the protocol with nothing between it and the test, so a failure there is a
// failure of the protocol, and this one is the driver the game actually uses, so a failure here is
// a failure of the driver. If only the second existed, a bug in Race would look like a bug in the
// wire format.
func runRace(t *testing.T, e *end, local Hello, script []Standing) played {
	t.Helper()
	m, err := Meet(e.Conn, local)
	if err != nil {
		return played{err: err}
	}
	r := Start(e.Conn, m)

	p := played{m: m}
	for _, s := range script {
		p.mine = s
		// Twice, every time, because the frame loop calls Present more than once per frame --
		// once per wipe strip, 116 of them in a transition -- and Report is called from there.
		r.Report(s)
		r.Report(s)
	}
	if err := r.Close(); err != nil {
		p.err = err
		return p
	}

	select {
	case <-r.Settled():
	case <-time.After(10 * time.Second):
		p.err = errors.New("the other side never settled")
		return p
	}
	p.theirs, _ = r.Opponent()
	p.result = r.Result(p.mine)
	p.err = r.Err()
	return p
}

func TestRaceDrivesBothSidesToOneResult(t *testing.T) {
	a, b := pair(t)
	ha, hb := hello(7, houseHashA), hello(9, houseHashA)

	done := make(chan played, 2)
	go func() { done <- runRace(t, a, ha, hostRun) }()
	go func() { done <- runRace(t, b, hb, guestRun) }()
	first, second := <-done, <-done
	if first.err != nil || second.err != nil {
		t.Fatalf("the race did not finish: %v, %v", first.err, second.err)
	}

	host, guest := first, second
	if host.m.Slot != 0 {
		host, guest = second, first
	}
	if host.result != guest.result {
		t.Errorf("the two sides disagree: host says %s, guest says %s",
			host.result, guest.result)
	}
	if want := (Outcome{Slot: 1, Reason: ByFinish}); host.result != want {
		t.Errorf("result = %s, want %s", host.result, want)
	}
	// The last standing of each run arrived, which is the one the result is computed from. The
	// ones before it are not asserted, and deliberately: Report coalesces, so an intermediate
	// standing may be replaced by a newer one before the writer gets to it. The last cannot be,
	// because Close flushes it.
	if host.theirs != guestRun[len(guestRun)-1] {
		t.Errorf("the host's last word on the guest was %+v, want %+v",
			host.theirs, guestRun[len(guestRun)-1])
	}
	if guest.theirs != hostRun[len(hostRun)-1] {
		t.Errorf("the guest's last word on the host was %+v, want %+v",
			guest.theirs, hostRun[len(hostRun)-1])
	}
}

func TestReportSendsNothingWhenNothingChanged(t *testing.T) {
	// The reason this is a test and not a comment: at 30 frames a second, with Present called
	// over a hundred times in a transition, a Report that sent unconditionally would put
	// thousands of messages a second on the wire and nothing would fail. It would just be slow
	// against a real opponent, on somebody else's LAN, and nowhere near this package.
	a, b := pair(t)

	guestFailed := make(chan error, 1)
	reports := make(chan int, 1)
	go func() {
		if _, err := Meet(b.Conn, hello(9, houseHashA)); err != nil {
			guestFailed <- err
			return
		}
		n := 0
		for {
			msg, err := b.Recv()
			if err != nil {
				guestFailed <- err
				return
			}
			switch msg.(type) {
			case *Report:
				n++
			case *Bye:
				reports <- n
				return
			}
		}
	}()

	m, err := Meet(a.Conn, hello(7, houseHashA))
	if err != nil {
		t.Fatalf("Meet: %v", err)
	}
	r := Start(a.Conn, m)
	only := hostRun[0]
	for i := 0; i < 500; i++ {
		r.Report(only)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	select {
	case err := <-guestFailed:
		t.Fatalf("the guest stopped reading: %v", err)
	case n := <-reports:
		if n != 1 {
			t.Errorf("500 identical Reports put %d messages on the wire, want exactly 1", n)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the guest never saw the goodbye")
	}
}

func TestCloseFlushesTheStandingItIsAboutToSayGoodbyeAfter(t *testing.T) {
	// The ordering that decides every ordinary race: the last standing of a run is the one the
	// result is computed from, and Close is called the instant the run ends. If the goodbye could
	// overtake the standing still sitting in the queue, the peer would score the race off the
	// standing before it -- a run that finished the house would come across as one that was still
	// flying, and the forfeit rule would then award it to the other side.
	a, b := pair(t)

	got := make(chan []Msg, 1)
	failed := make(chan error, 1)
	go func() {
		if _, err := Meet(b.Conn, hello(9, houseHashA)); err != nil {
			failed <- err
			return
		}
		var seen []Msg
		for {
			msg, err := b.Recv()
			if err != nil {
				failed <- err
				return
			}
			seen = append(seen, msg)
			if _, done := msg.(*Bye); done {
				got <- seen
				return
			}
		}
	}()

	m, err := Meet(a.Conn, hello(7, houseHashA))
	if err != nil {
		t.Fatalf("Meet: %v", err)
	}
	r := Start(a.Conn, m)
	last := hostRun[len(hostRun)-1]
	r.Report(last)
	if err := r.Close(); err != nil { // no pause, no yield: the two calls are adjacent
		t.Fatalf("Close: %v", err)
	}

	select {
	case err := <-failed:
		t.Fatalf("the guest stopped reading: %v", err)
	case seen := <-got:
		if len(seen) != 2 {
			t.Fatalf("the guest saw %d messages, want the standing and then the goodbye", len(seen))
		}
		rep, ok := seen[0].(*Report)
		if !ok {
			t.Fatalf("the first message was %T, want the final standing", seen[0])
		}
		if rep.Standing != last {
			t.Errorf("the flushed standing was %+v, want %+v", rep.Standing, last)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the guest never saw the goodbye")
	}
}

func TestRaceTurnsAKilledPeerIntoAForfeitWithoutCallingItAnError(t *testing.T) {
	a, b := pair(t)

	ready := make(chan struct{})
	failed := make(chan error, 1)
	go func() {
		m, err := Meet(b.Conn, hello(9, houseHashA))
		if err != nil {
			failed <- err
			return
		}
		if err := b.SendStanding(m.Slot, guestRun[1]); err != nil {
			failed <- err
			return
		}
		close(ready)
		b.kill()
	}()

	m, err := Meet(a.Conn, hello(7, houseHashA))
	if err != nil {
		t.Fatalf("Meet: %v", err)
	}
	r := Start(a.Conn, m)
	defer r.Close() //nolint:errcheck // there is nobody left to say goodbye to

	select {
	case err := <-failed:
		t.Fatalf("the guest failed before it could be killed: %v", err)
	case <-ready:
	}
	select {
	case <-r.Settled():
	case <-time.After(10 * time.Second):
		t.Fatal("the host never noticed the guest had gone")
	}

	theirs, gone := r.Opponent()
	if !gone {
		t.Error("the host thinks the guest is still there")
	}
	// **The panel shows the run, not the verdict.** Opponent hands back what the guest last
	// reported -- still Racing -- because that is what the opponent reached; turning it into a
	// forfeit is Result's business and happens once, at the end.
	if theirs != guestRun[1] {
		t.Errorf("Opponent = %+v, want the guest's last report %+v", theirs, guestRun[1])
	}
	if err := r.Err(); err != nil {
		t.Errorf("Err = %v; a peer whose process ended is a forfeit, not a fault", err)
	}

	mine := hostRun[1]
	if mine.State.Ended() {
		t.Fatal("this test is meaningless unless the host is still racing")
	}
	if want := (Outcome{Slot: 0, Reason: ByForfeit}); r.Result(mine) != want {
		t.Errorf("result = %s, want %s", r.Result(mine), want)
	}
}

func TestRaceKeepsAProtocolErrorWhereSomebodyWillSeeIt(t *testing.T) {
	// The other kind of ending. A lock-step build's MsgInputFrames is the case the package comment
	// argues about: two versions ignoring each other's messages is worse than a stopped match, so
	// Conn refuses it -- and a race that dropped that error on the floor would present the refusal
	// as an ordinary forfeit, which is a result the loser would be right to query.
	a, b := pair(t)

	failed := make(chan error, 1)
	go func() {
		if _, err := Meet(b.Conn, hello(9, houseHashA)); err != nil {
			failed <- err
			return
		}
		bad := make([]byte, HeaderSize)
		header(bad, MsgInputFrames, b.MatchID())
		if err := b.Send(bad); err != nil {
			failed <- err
		}
	}()

	m, err := Meet(a.Conn, hello(7, houseHashA))
	if err != nil {
		t.Fatalf("Meet: %v", err)
	}
	r := Start(a.Conn, m)
	defer r.Close() //nolint:errcheck // the connection is already finished

	select {
	case err := <-failed:
		t.Fatalf("the guest could not send the bad message: %v", err)
	case <-r.Settled():
	case <-time.After(10 * time.Second):
		t.Fatal("the host never noticed the bad message")
	}
	err = r.Err()
	if !errors.Is(err, ErrProtocol) {
		t.Fatalf("Err = %v, want an ErrProtocol", err)
	}
	if !strings.Contains(err.Error(), "MsgInputFrames") {
		t.Errorf("Err = %q, want it to name the message that arrived", err)
	}
	if _, gone := r.Opponent(); !gone {
		t.Error("a match that ended on a protocol error is still a match that ended")
	}
}

func TestRaceDoesNotBlameAPeerThatSaidGoodbyeForHangingUp(t *testing.T) {
	// The first finisher who does not wait. It sends its last standing and a goodbye, its player
	// presses Esc on the waiting screen, and its window closes while this side is still flying.
	// Over TCP this side's next report draws a reset and the one after it is refused; over the
	// pipe here the next one is refused at once. That refusal is the goodbye's consequence.
	// Recorded as the race's error, it put "the other player's game ended without saying
	// goodbye" on the result screen of a player whose opponent had said it (docs/IMPROVEMENTS.md
	// 4.49).
	a, b := pair(t)

	said := make(chan struct{})
	failed := make(chan error, 1)
	go func() {
		m, err := Meet(b.Conn, hello(9, houseHashA))
		if err != nil {
			failed <- err
			return
		}
		if err := b.SendStanding(m.Slot, guestRun[2]); err != nil {
			failed <- err
			return
		}
		if err := b.Bye(m.Slot); err != nil {
			failed <- err
			return
		}
		close(said)
	}()

	m, err := Meet(a.Conn, hello(7, houseHashA))
	if err != nil {
		t.Fatalf("Meet: %v", err)
	}
	r := Start(a.Conn, m)
	defer r.Close() //nolint:errcheck // closed below; this is for the ways out before that

	select {
	case err := <-failed:
		t.Fatalf("the guest could not say goodbye: %v", err)
	case <-said:
	}
	// The guest hangs up after this side has read the goodbye, as a player does: seconds after,
	// on a result screen.
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(time.Millisecond) {
		if _, gone := r.Opponent(); gone {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the host never heard the goodbye")
		}
	}
	b.r.Close()

	r.Report(hostRun[1])
	if err := r.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	r.mu.Lock()
	broken := r.broken
	r.mu.Unlock()
	if !broken {
		t.Fatal("the host's report went through; this test is meaningless unless it was refused")
	}
	if err := r.Err(); err != nil {
		t.Errorf("Err = %v, for a guest that said goodbye before it hung up", err)
	}
	if theirs, _ := r.Opponent(); theirs != guestRun[2] {
		t.Errorf("Opponent = %+v, want the guest's last report %+v", theirs, guestRun[2])
	}
}

func TestCloseIsSafeTwice(t *testing.T) {
	// Because it will be called twice. The result screen closes the race, and so does the deferred
	// cleanup on the way out of the game loop, and neither knows about the other.
	a, b := pair(t)
	go Meet(b.Conn, hello(9, houseHashA)) //nolint:errcheck // this side is only here to handshake

	m, err := Meet(a.Conn, hello(7, houseHashA))
	if err != nil {
		t.Fatalf("Meet: %v", err)
	}
	r := Start(a.Conn, m)
	r.Report(hostRun[0])
	if err := r.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

// stallingRW is a transport whose writes stop going anywhere once stall is closed: each one
// then blocks until release is closed, and fails. That is a socket in front of a peer that has
// stopped reading, as far as the writer can tell, and then the same socket closed under it.
type stallingRW struct {
	rw      io.ReadWriter
	stall   chan struct{}
	release chan struct{}
}

func (s *stallingRW) Read(p []byte) (int, error) { return s.rw.Read(p) }

func (s *stallingRW) Write(p []byte) (int, error) {
	select {
	case <-s.stall:
		<-s.release
		return 0, errors.New("use of closed network connection")
	default:
		return s.rw.Write(p)
	}
}

// A peer that stops reading must not freeze this side at the end of its run. Close is called the
// moment a run ends and before the waiting screen is drawn, and it used to wait for a writer stuck
// behind a full send buffer: with the other machine asleep, the game sat on a frozen frame, with
// Esc and the close box doing nothing, until the operating system gave up on the connection.
func TestCloseDoesNotWaitForAPeerThatStoppedReading(t *testing.T) {
	defer func(d time.Duration) { closeWait = d }(closeWait)
	closeWait = 50 * time.Millisecond

	a, b := pair(t)
	go Meet(b.Conn, hello(9, houseHashA)) //nolint:errcheck // the peer handshakes, then never reads again
	tr := &stallingRW{rw: duplex{a.r, a.w}, stall: make(chan struct{}), release: make(chan struct{})}
	c := NewConn(tr)
	m, err := Meet(c, hello(7, houseHashA))
	if err != nil {
		t.Fatalf("Meet: %v", err)
	}
	r := Start(c, m)
	close(tr.stall)
	r.Report(hostRun[0])
	r.Report(hostRun[len(hostRun)-1])

	began := time.Now()
	if err := r.Close(); !errors.Is(err, ErrStalled) {
		t.Errorf("Close = %v, want ErrStalled", err)
	}
	if took := time.Since(began); took > 2*time.Second {
		t.Errorf("Close took %v with closeWait at %v", took, closeWait)
	}
	// Silence decides nothing: the other player is not scored as having left.
	select {
	case <-r.Settled():
		t.Error("a stalled writer settled the race")
	default:
	}
	if _, gone := r.Opponent(); gone {
		t.Error("a stalled writer marked the peer gone")
	}
	began = time.Now()
	if err := r.Close(); !errors.Is(err, ErrStalled) || time.Since(began) > closeWait/2 {
		t.Errorf("a second Close returned %v after %v; want ErrStalled at once", err, time.Since(began))
	}

	// Closing the transport is what releases the writer, as Start's comment says.
	close(tr.release)
	select {
	case <-r.done:
	case <-time.After(5 * time.Second):
		t.Fatal("the writer was still stuck after its transport failed")
	}
	if err := r.Close(); err != nil {
		t.Errorf("Close after the writer finished = %v, want nil", err)
	}
}

func TestAbandonedOnlyForfeitsARunThatWasStillGoing(t *testing.T) {
	for _, c := range []struct {
		name string
		in   Standing
		want Standing
	}{{
		name: "a peer that vanished while flying forfeits, keeping what it had",
		in:   Standing{State: Racing, Rooms: 9, Score: 100, Frame: 1500},
		want: Standing{State: Quit, Rooms: 9, Score: 100, Frame: 1500},
	}, {
		name: "a peer that finished and then hung up has not forfeited",
		in:   Standing{State: Finished, Rooms: 12},
		want: Standing{State: Finished, Rooms: 12},
	}, {
		name: "nor has one that died and then hung up",
		in:   Standing{State: Died, Rooms: 30},
		want: Standing{State: Died, Rooms: 30},
	}, {
		name: "and a peer that already said it was leaving stays left",
		in:   Standing{State: Quit, Rooms: 4},
		want: Standing{State: Quit, Rooms: 4},
	}, {
		// The zero value, which is what a peer that connected and died before sending
		// anything leaves behind. A forfeit with nothing to its name, and Winner handles it
		// without a special case.
		name: "a peer that never reported anything forfeits from nothing",
		in:   Standing{},
		want: Standing{State: Quit},
	}} {
		if got := Abandoned(c.in); got != c.want {
			t.Errorf("%s:\n got %+v\nwant %+v", c.name, got, c.want)
		}
	}
}

func TestBothSidesAgreeWhenBothAreKilled(t *testing.T) {
	// Not a real case worth a screen, but a rule with a hole in it is worth finding: if both
	// peers vanish there is no result to disagree about, and the important thing is that
	// neither of them can claim to have won.
	mine := Abandoned(Standing{State: Racing, Rooms: 20})
	theirs := Abandoned(Standing{State: Racing, Rooms: 3})
	host := Match{Slot: 0}.Result(mine, theirs)
	guest := Match{Slot: 1}.Result(theirs, mine)
	if host != guest {
		t.Errorf("the two sides disagree: %s and %s", host, guest)
	}
	if host.Slot >= 0 {
		t.Errorf("result = %s; with both sides gone there is no winner, whatever the "+
			"standings said", host)
	}
}
