package netplay

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// DefaultPort is the port a race uses when the player names a machine and not a port.
//
// 1994 because that is when Glider PRO shipped, and because a number somebody can remember is
// worth more here than a number somebody has registered: this is two people on one LAN agreeing
// to play, not a service being discovered. IANA has it down for a Cisco licensing protocol, which
// is the kind of collision that matters when a daemon might already be listening and not when a
// player is about to type the address out loud. `-host :port` overrides it in either direction.
const DefaultPort = "1994"

// Address fills in what a player left out. An empty port means DefaultPort, so that the flag a
// player typed and the default this package ships are resolved in one place instead of two.
//
// "10.0.0.7" becomes "10.0.0.7:1994"; ":2000" and "10.0.0.7:2000" are left alone -- a port in the
// address wins, because it is the more specific of the two things the player said; "" becomes
// ":1994", which listens on every interface.
//
// IPv6 takes two lines of its own because one spelling is genuinely ambiguous: "::1" is a complete
// address and is also, to SplitHostPort, a host with an empty port. It is read here as the address,
// and gets the default port -- "[::1]:1994" -- because a player typing an address into a text field
// is naming a machine, and nobody has ever meant "the loopback host, port nothing". The cost of
// that choice is that **an IPv6 address with a port has to be bracketed**: "fe80::1:2000" is taken
// as a host, not as fe80::1 port 2000. That is the standard spelling anyway, and the alternative
// -- counting colons and hoping -- would be wrong silently instead of wrong in a way the player
// can see in the error. A host that is already bracketed and has no port gets one appended, which
// SplitHostPort will not do for us and JoinHostPort would double the brackets of.
func Address(addr, port string) string {
	if port == "" {
		port = DefaultPort
	}
	if addr == "" {
		return ":" + port
	}
	if strings.HasPrefix(addr, "[") && strings.HasSuffix(addr, "]") {
		return addr + ":" + port
	}
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr
	}
	return net.JoinHostPort(addr, port)
}

// Listener is the host waiting for one guest.
//
// Binding is separate from accepting on purpose. A port already in use is the commonest way this
// fails and it is worth reporting before the screen that says "waiting for the other player" goes
// up -- and Addr below is what that screen shows, which it cannot do until the port is real. The
// original had nothing to draw on here: its two-player mode was two people at one keyboard.
type Listener struct {
	l net.Listener
}

// Listen binds addr, filling in port (or DefaultPort, if that is empty too). It does not wait for
// anybody.
func Listen(addr, port string) (*Listener, error) {
	full := Address(addr, port)
	l, err := net.Listen("tcp", full)
	if err != nil {
		return nil, fmt.Errorf("netplay: cannot host on %s: %w", full, err)
	}
	return &Listener{l: l}, nil
}

// Addr is the address being listened on, with the port resolved -- so a host that asked for port 0
// can still read out where to connect.
func (l *Listener) Addr() string { return l.l.Addr().String() }

// Accept waits for the guest and returns the connection, having closed the listener: a race is
// two players, and a second guest arriving to find the port shut is a better answer than one
// accepted onto a match that is already under way.
//
// Close from another goroutine is how a waiting host gives up; Accept then returns that error.
func (l *Listener) Accept() (net.Conn, error) {
	c, err := l.l.Accept()
	l.Close()
	if err != nil {
		return nil, fmt.Errorf("netplay: no guest arrived: %w", err)
	}
	return c, nil
}

// Close stops listening. Safe to call twice, which matters because Accept already did.
func (l *Listener) Close() error { return l.l.Close() }

// Join dials the host. A zero timeout waits as long as the operating system will.
//
// The error is worth reading rather than passing through: "connection refused" on a LAN almost
// always means the host has not pressed Host yet, and that is a sentence a player can act on,
// where the address-and-errno form is one they have to interpret.
func Join(addr, port string, timeout time.Duration) (net.Conn, error) {
	full := Address(addr, port)
	c, err := net.DialTimeout("tcp", full, timeout)
	if err != nil {
		return nil, fmt.Errorf("netplay: cannot reach %s: %w -- the other machine has to be "+
			"hosting before this one can join", full, err)
	}
	return c, nil
}
