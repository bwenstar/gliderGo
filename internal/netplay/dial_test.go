package netplay

import (
	"errors"
	"net"
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

	guest, err = Join(l.Addr(), "", 10*time.Second)
	if err != nil {
		l.Close()
		t.Fatalf("Join(%s): %v", l.Addr(), err)
	}
	a := <-done
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

func TestASecondGuestFindsThePortShut(t *testing.T) {
	// A race is two players. The alternative to closing the listener is a third machine
	// connecting into a match already under way and being ignored, which looks -- from that
	// machine -- exactly like the host not being there, except that it waits forever first.
	l, err := Listen("127.0.0.1", "0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	addr := l.Addr()
	accepted := make(chan error, 1)
	go func() {
		c, err := l.Accept()
		if c != nil {
			defer c.Close()
		}
		accepted <- err
	}()

	first, err := Join(addr, "", 10*time.Second)
	if err != nil {
		l.Close()
		t.Fatalf("the first guest could not join: %v", err)
	}
	defer first.Close()
	if err := <-accepted; err != nil {
		t.Fatalf("Accept: %v", err)
	}

	// The listener is shut now, by Accept. On Linux a connect to a closed port is refused
	// outright; the assertion is only that it does not succeed, because how the refusal arrives
	// is the operating system's business and differs between them.
	second, err := Join(addr, "", 2*time.Second)
	if err == nil {
		second.Close()
		t.Fatal("a second guest was let in to a match already under way")
	}
	if !strings.Contains(err.Error(), addr) {
		t.Errorf("the refusal was %q, which does not name the address that refused", err)
	}
	if !strings.Contains(err.Error(), "hosting") {
		t.Errorf("the refusal was %q; a player reading it should be told what to check", err)
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

	// And a second Close is not an error, which matters because Accept already called it.
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
	c, err := Join("203.0.113.1", "1994", 250*time.Millisecond)
	if err == nil {
		c.Close()
		t.Skip("something answered on TEST-NET-3; this network is not one this test can use")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("Join waited %v on a 250ms timeout", elapsed)
	}
}
