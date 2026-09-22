package netplay

import (
	"errors"
	"io"
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
