package netplay

import (
	"context"
	"errors"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

// errNeverSettled is the socket race's own giving-up, so that the side struct below always carries
// an error and never a formatted string somebody has to read to find out what happened.
var errNeverSettled = errors.New("the other side never settled")

func TestAddressFillsInWhatThePlayerLeftOut(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"", ":" + DefaultPort},                             // host with no arguments: every interface
		{"10.0.0.7", "10.0.0.7:" + DefaultPort},             // the common case, a machine on the LAN
		{"macintosh", "macintosh:" + DefaultPort},           // a name, resolved by the dialler
		{":2000", ":2000"},                                  // a host that wants a different port
		{"10.0.0.7:2000", "10.0.0.7:2000"},                  // both given, nothing to add
		{"[::1]:2000", "[::1]:2000"},                        // IPv6, bracketed, port given
		{"[::1]", "[::1]:" + DefaultPort},                   // IPv6, bracketed, port left out
		{"[fe80::1%eth0]", "[fe80::1%eth0]:" + DefaultPort}, // a zone survives it
		{"::1", "[::1]:" + DefaultPort},                     // read as an address, not a host with no port
		{"fe80::1", "[fe80::1]:" + DefaultPort},             // and bracketed on the player's behalf
	} {
		if got := Address(c.in, ""); got != c.want {
			t.Errorf("Address(%q, \"\") = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestEveryAddressAddressProducesCanBeSplitAgain(t *testing.T) {
	// The property that matters more than any single row above: whatever the player typed, what
	// comes out is something net.Dial and net.Listen will accept. Both of them start by splitting
	// it, so a spelling that does not survive SplitHostPort is one the player would meet as a
	// parse error rather than as a connection.
	//
	// The unbracketed IPv6 spellings are why this is here. They are the ones where Address has to
	// *change* the string rather than append to it, and getting that wrong yields "[[::1]]:1994",
	// which reads plausibly and dials nothing.
	for _, in := range []string{
		"", "10.0.0.7", "macintosh", ":2000", "10.0.0.7:2000",
		"::1", "fe80::1", "[::1]", "[::1]:2000", "[fe80::1%eth0]",
	} {
		full := Address(in, "")
		host, port, err := net.SplitHostPort(full)
		if err != nil {
			t.Errorf("Address(%q) = %q, which net.Dial cannot parse: %v", in, full, err)
			continue
		}
		if port == "" {
			t.Errorf("Address(%q) = %q, which has no port", in, full)
		}
		if strings.HasPrefix(host, "[") {
			t.Errorf("Address(%q) = %q: the host came back as %q, so the brackets were "+
				"doubled", in, full, host)
		}
	}
}

// loopback is a host and a guest on a real socket, which is the one thing os.Pipe cannot stand in
// for: an address, a port, a listener and the kernel's TCP stack between the two Conns.
func loopback(t *testing.T) (host, guest net.Conn) {
	t.Helper()
	l, err := Listen("127.0.0.1", "0") // port 0: the operating system picks one nobody is using
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	type accepted struct {
		c   net.Conn
		err error
	}
	done := make(chan accepted, 1)
	go func() {
		c, err := l.Accept()
		done <- accepted{c, err}
	}()

	guest, err = Join(context.Background(), l.Addr(), "", 10*time.Second)
	if err != nil {
		l.Close()
		t.Fatalf("Join(%s): %v", l.Addr(), err)
	}
	a := <-done
	l.Close() // one guest is all any of these want
	if a.err != nil {
		guest.Close()
		t.Fatalf("Accept: %v", a.err)
	}
	t.Cleanup(func() {
		a.c.Close()
		guest.Close()
	})
	return a.c, guest
}

func TestAWholeRaceOverARealSocket(t *testing.T) {
	// Everything the mode is made of, in the order a player meets it: a port, a connection, the
	// handshake, two runs reported over TCP, and one result both sides computed for themselves.
	// The other tests in this package prove the protocol over a pipe; this one proves there is
	// nothing about a socket that the protocol was relying on not being there.
	hostConn, guestConn := loopback(t)

	type side struct {
		m      Match
		theirs Standing
		result Outcome
		err    error
	}
	run := func(c net.Conn, nonce uint64, script []Standing) side {
		conn := NewConn(c)
		m, err := Meet(conn, hello(nonce, houseHashA))
		if err != nil {
			return side{err: err}
		}
		r := Start(conn, m)
		var mine Standing
		for _, s := range script {
			mine = s
			r.Report(s)
		}
		if err := r.Close(); err != nil {
			return side{err: err}
		}
		select {
		case <-r.Settled():
		case <-time.After(10 * time.Second):
			return side{m: m, err: errNeverSettled}
		}
		theirs, _ := r.Opponent()
		return side{m: m, theirs: theirs, result: r.Result(mine), err: r.Err()}
	}

	done := make(chan side, 2)
	go func() { done <- run(hostConn, 7, hostRun) }()
	go func() { done <- run(guestConn, 9, guestRun) }()
	first, second := <-done, <-done
	if first.err != nil || second.err != nil {
		t.Fatalf("the race did not finish: %v, %v", first.err, second.err)
	}
	if first.result != second.result {
		t.Errorf("the two sides disagree over a socket: %s and %s", first.result, second.result)
	}
	if want := (Outcome{Slot: 1, Reason: ByFinish}); first.result != want {
		t.Errorf("result = %s, want %s", first.result, want)
	}
	// And the handshake really did run over the wire: the two sides took different slots, which
	// only their nonces could have decided.
	if first.m.Slot == second.m.Slot {
		t.Errorf("both sides think they are player %d", first.m.Slot+1)
	}
}

func TestAListenerStaysOpenUntilItIsClosed(t *testing.T) {
	// The first connection is not always the guest (docs/IMPROVEMENTS.md 4.32). It can be a
	// guest with another house, or a browser somebody pointed at the port. So Accept leaves the
	// port open, and the host shuts it once a match is agreed. cmd/glidergo's loopback tests
	// check that the host does. This checks that Accept is not what shuts it.
	l, err := Listen("127.0.0.1", "0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer l.Close()
	addr := l.Addr()
	accepted := make(chan error, 2)
	go func() {
		for range 2 {
			c, err := l.Accept()
			if c != nil {
				defer c.Close()
			}
			accepted <- err
		}
	}()

	for _, who := range []string{"first", "second"} {
		g, err := Join(context.Background(), addr, "", 10*time.Second)
		if err != nil {
			t.Fatalf("the %s connection could not join: %v", who, err)
		}
		defer g.Close()
		if err := <-accepted; err != nil {
			t.Fatalf("Accept, for the %s connection: %v", who, err)
		}
	}

	// A race is two players. The alternative to closing the listener is a third machine
	// connecting into a match already under way and being ignored, which looks -- from that
	// machine -- exactly like the host not being there, except that it waits forever first. On
	// Linux a connect to a closed port is refused outright. The assertion is only that it does
	// not succeed, because how the refusal arrives is the operating system's business.
	l.Close()
	second, err := Join(context.Background(), addr, "", 2*time.Second)
	if err == nil {
		second.Close()
		t.Fatal("a second guest was let in to a match already under way")
	}
	if !strings.Contains(err.Error(), addr) {
		t.Errorf("the refusal was %q, which does not name the address that refused", err)
	}
	// What to check is the caller's to say (cmd/glidergo words it for a screen); what it has
	// to go on is the cause, and "connection refused" and "no answer" are both causes.
	var je *JoinError
	if !errors.As(err, &je) || je.Why == JoinOther {
		t.Errorf("the refusal was %q, sorted as %v; want a cause a player can be told", err, je)
	}
}

func TestCloseIsHowAWaitingHostGivesUp(t *testing.T) {
	// The host presses Escape on the "waiting for the other player" screen. Accept is parked in
	// the kernel at that point and cannot be told anything; closing the listener from the other
	// goroutine is the only way to get it back, and it has to come back as an error rather than
	// hanging or returning a nil connection that the caller then hands to NewConn.
	l, err := Listen("127.0.0.1", "0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	type accepted struct {
		c   net.Conn
		err error
	}
	done := make(chan accepted, 1)
	go func() {
		c, err := l.Accept()
		done <- accepted{c, err}
	}()

	// Nothing to synchronise on -- there is no way to observe a goroutine having reached a
	// blocking syscall -- so the close may land before Accept or after it. Both have to work,
	// and running this test with -count=2 under the race detector is what makes that more than
	// a hope.
	if err := l.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	select {
	case a := <-done:
		if a.err == nil {
			a.c.Close()
			t.Fatal("Accept returned a connection after the listener was closed")
		}
		if a.c != nil {
			a.c.Close()
			t.Error("Accept returned both a connection and an error")
		}
		if !strings.Contains(a.err.Error(), "no guest arrived") {
			t.Errorf("Accept failed with %q, which does not read like giving up", a.err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Close did not release the waiting Accept")
	}

	// And a second Close does no harm. That matters because a host closes its listener on
	// every way out of the wait, and Escape may already have closed it.
	if err := l.Close(); err == nil {
		t.Log("a second Close was accepted")
	}
}

func TestListenSaysWhyItCouldNotBind(t *testing.T) {
	// The commonest way hosting fails, and the reason binding is separate from accepting: the
	// player gets told before the waiting screen goes up, and the message has the address in it
	// because "address already in use" on its own does not say which.
	first, err := Listen("127.0.0.1", "0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer first.Close()

	second, err := Listen(first.Addr(), "")
	if err == nil {
		second.Close()
		t.Skip("this machine allows two listeners on one port; nothing to check")
	}
	if !strings.Contains(err.Error(), first.Addr()) {
		t.Errorf("the failure was %q, which does not say which address it was", err)
	}
	if !strings.Contains(err.Error(), "cannot host") {
		t.Errorf("the failure was %q; it should read as hosting having failed", err)
	}
}

func TestAddrResolvesThePortTheHostDidNotChoose(t *testing.T) {
	// What the waiting screen reads out. A host that asked for port 0 -- or the default, on a
	// machine where something else already has 1994 -- still has to be able to tell the other
	// player where to connect.
	l, err := Listen("127.0.0.1", "0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer l.Close()
	_, port, err := net.SplitHostPort(l.Addr())
	if err != nil {
		t.Fatalf("Addr() = %q, which is not a host and port: %v", l.Addr(), err)
	}
	if port == "0" || port == "" {
		t.Errorf("Addr() = %q; the port has to be the resolved one, not the one asked for",
			l.Addr())
	}
}

func TestJoinGivesUpWhenAskedTo(t *testing.T) {
	// A timeout the player can sit through. 203.0.113.0/24 is TEST-NET-3, reserved by RFC 5737
	// for documentation and guaranteed not to be routed, so this dials something that cannot
	// answer without depending on the machine being offline -- which, on the machine this port
	// was written on, it is.
	start := time.Now()
	c, err := Join(context.Background(), "203.0.113.1", "1994", 250*time.Millisecond)
	if err == nil {
		c.Close()
		t.Skip("something answered on TEST-NET-3; this network is not one this test can use")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("Join waited %v on a 250ms timeout", elapsed)
	}
	var je *JoinError
	if !errors.As(err, &je) {
		t.Fatalf("err = %v (%T), want a *JoinError", err, err)
	}
	// Whatever this network does with TEST-NET-3 -- nothing, here -- it is not a refusal: no
	// machine answered.
	if je.Why == JoinRefused || je.Addr != "203.0.113.1:1994" {
		t.Errorf("Join on TEST-NET-3 = %+v, want a failure that is not a refusal, at the "+
			"address with its port", *je)
	}
}

// Escape during a dial is the context ending, and the dial stops then rather than when its
// timeout would have (docs/IMPROVEMENTS.md 4.33). A long timeout here is the point: a guest's
// timeout can grow to three seconds only because this holds.
func TestJoinStopsWhenItsContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)
	start := time.Now()
	c, err := Join(ctx, "203.0.113.1", "1994", time.Minute)
	if err == nil {
		c.Close()
		t.Skip("something answered on TEST-NET-3; this network is not one this test can use")
	}
	if took := time.Since(start); took > 5*time.Second {
		t.Errorf("Join took %v to notice a context cancelled after 100ms", took)
	}
	if !errors.Is(err, context.Canceled) {
		t.Skipf("this network failed TEST-NET-3 before the context ended (%v)", err)
	}
}

// The sort, over each failure as a dial returns it. **This is the only test the Windows numbers
// get**: on Windows the table below is built from Winsock's errnos (dial_windows.go) and nothing
// in this repository can run it there. So each case wraps its cause as net.Dialer does -- an
// OpError around a SyscallError around the errno -- rather than passing the bare errno.
func TestJoinFailuresAreSortedByCause(t *testing.T) {
	dial := func(err error) error {
		return &net.OpError{Op: "dial", Net: "tcp", Err: err}
	}
	for _, tc := range []struct {
		name string
		err  error
		want JoinFailure
	}{
		{"refused", dial(os.NewSyscallError("connect", errRefused)), JoinRefused},
		{"no route to the host", dial(os.NewSyscallError("connect", errHostUnreachable)), JoinUnreachable},
		{"no route to the network", dial(os.NewSyscallError("connect", errNetUnreachable)), JoinUnreachable},
		{"no such host", dial(&net.DNSError{Err: "no such host", Name: "macintosh", IsNotFound: true}), JoinNoSuchHost},
		// The resolver not answering is not the host not answering, and it must not reach the
		// firewall advice.
		{"the resolver timed out", dial(&net.DNSError{Err: "i/o timeout", Name: "macintosh", IsTimeout: true}), JoinOther},
		{"timeout", dial(os.ErrDeadlineExceeded), JoinTimedOut},
		{"context deadline", dial(context.DeadlineExceeded), JoinTimedOut},
		{"cancelled", dial(context.Canceled), JoinOther},
		{"reset", dial(os.NewSyscallError("connect", errors.New("connection reset by peer"))), JoinOther},
	} {
		if got := joinFailure(tc.err); got != tc.want {
			t.Errorf("%s: joinFailure(%v) = %d, want %d", tc.name, tc.err, got, tc.want)
		}
	}
}

// The two causes this machine can produce for real, so that the table above is known to describe
// what a dial actually returns and not only what it was believed to.
func TestJoinSortsARealRefusalAndARealMissingName(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()

	var je *JoinError
	_, err = Join(context.Background(), addr, "", 5*time.Second)
	if !errors.As(err, &je) || je.Why != JoinRefused {
		t.Errorf("Join on a closed loopback port = %v, want JoinRefused", err)
	}

	// .invalid is reserved by RFC 6761 to never resolve. A resolver that answers it anyway, or
	// does not answer at all, is this network's business and not this test's.
	_, err = Join(context.Background(), "gliderGo.invalid", "", 5*time.Second)
	if !errors.As(err, &je) || je.Why != JoinNoSuchHost {
		t.Skipf("Join on a .invalid name = %v, which this network did not report as no such "+
			"host", err)
	}
}

// Something that accepts and never speaks is given up on, in words, within MeetWithin's wait:
// docs/IMPROVEMENTS.md 4.32's guest parked on JOINING, measured against a web server. A web server
// is the silent end here as well, because it answers a request and a hello is not one.
func TestMeetWithinGivesUpOnAPeerThatNeverSpeaks(t *testing.T) {
	host, _ := loopback(t)
	const wait = 200 * time.Millisecond
	start := time.Now()
	_, err := MeetWithin(NewConn(host), hello(7, houseHashA), wait)
	if took := time.Since(start); took > wait+5*time.Second {
		t.Errorf("MeetWithin(%v) took %v", wait, took)
	}
	if !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("err = %v, want a deadline", err)
	}
	if !strings.Contains(err.Error(), "said nothing for 200ms") {
		t.Errorf("err = %q; a peer that never spoke should be told apart from one that stopped", err)
	}
}

// One that says hello and then stops is a different failure, and it is told apart from one that
// never spoke. Here the silent end is player 1, which owes the other a MatchStart and never
// sends it.
func TestMeetWithinSaysWhenThePeerStoppedPartway(t *testing.T) {
	host, guest := loopback(t)
	peer := NewConn(guest)
	if err := peer.Send(EncodeHello(hello(1, houseHashA))); err != nil {
		t.Fatal(err)
	}
	_, err := MeetWithin(NewConn(host), hello(9, houseHashA), 200*time.Millisecond)
	if !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("err = %v, want a deadline", err)
	}
	if !strings.Contains(err.Error(), "partway") {
		t.Errorf("err = %q; the peer said hello, so it did not say nothing", err)
	}
}

// The deadline is the handshake's and not the race's. A race lasts minutes, and one that dropped
// its connection five seconds after it started, because the handshake's deadline was still set,
// would be a forfeit nobody could explain.
func TestMeetWithinLeavesNoDeadlineOnTheRace(t *testing.T) {
	host, guest := loopback(t)
	const wait = 100 * time.Millisecond
	type side struct {
		c   *Conn
		m   Match
		err error
	}
	done := make(chan side, 2)
	for _, s := range []struct {
		trans net.Conn
		nonce uint64
	}{{host, 7}, {guest, 9}} {
		go func() {
			c := NewConn(s.trans)
			m, err := MeetWithin(c, hello(s.nonce, houseHashA), wait)
			done <- side{c, m, err}
		}()
	}
	a, b := <-done, <-done
	if a.err != nil || b.err != nil {
		t.Fatalf("the handshake failed: %v, %v", a.err, b.err)
	}

	time.Sleep(3 * wait)
	if err := a.c.SendStanding(a.m.Slot, Standing{Rooms: 1, Frame: 1}); err != nil {
		t.Fatalf("a standing sent after the handshake's deadline had passed: %v", err)
	}
	if _, err := b.c.Recv(); err != nil {
		t.Fatalf("a standing read after the handshake's deadline had passed: %v", err)
	}
}

// A connection whose first four bytes are no possible length is some other program, and it is
// said as that: ErrMagic, with the bytes quoted, which is usually the program's name. It is still
// an ErrProtocol for whatever tested for that before. The same four bytes later in a stream are
// a peer that lost its place rather than a stranger, and stay a protocol violation only.
func TestAnImpossibleFirstLengthIsAnotherProgram(t *testing.T) {
	a, b := pair(t)
	if _, err := a.w.Write([]byte("SSH-2.0-OpenSSH_9.6\r\n")); err != nil {
		t.Fatal(err)
	}
	_, err := b.Recv()
	if !errors.Is(err, ErrMagic) || !errors.Is(err, ErrProtocol) {
		t.Fatalf("err = %v, want both %v and %v", err, ErrMagic, ErrProtocol)
	}
	if !strings.Contains(err.Error(), `"SSH-"`) {
		t.Errorf("err = %q, which does not quote what the other end opened with", err)
	}

	a, b = pair(t)
	if err := a.SendStanding(0, Standing{}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Recv(); err != nil {
		t.Fatal(err)
	}
	if _, err := a.w.Write([]byte("SSH-")); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Recv(); !errors.Is(err, ErrProtocol) || errors.Is(err, ErrMagic) {
		t.Errorf("an impossible length after a message: err = %v, want %v and not %v",
			err, ErrProtocol, ErrMagic)
	}
}
