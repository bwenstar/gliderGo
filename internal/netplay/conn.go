package netplay

import (
	"errors"
	"fmt"
	"io"
	"sync"
)

// MaxMsg bounds what one message may ask this process to allocate.
//
// It is not a tuning parameter and there is nothing to tune: the largest message the race
// sends is MsgStanding at 32 bytes, and the largest anything in
// docs/analysis/determinism-networking.md defines is §10.4.8's MsgProgress, 102 bytes plus 290
// per room, which for the biggest house the 1994 corpus contains -- 531 rooms, the "epic" tier
// of docs/analysis/original-houses.md §10.2 -- is about 154 KB. A megabyte leaves that room
// and still means a peer claiming a 4 GB message gets one error instead of an allocation.
const MaxMsg = 1 << 20

// lenPrefix is the four bytes in front of every message on the stream. See the package
// comment: the messages §10.4 specifies are datagram-shaped, and a stream has to be told
// where each one ends.
const lenPrefix = 4

// Conn is one connection to one peer, with the framing and the envelope checks on it.
//
// Sends are serialised, because two goroutines really do send: the game loop reports a
// standing every time something changes, and the shutdown path sends a Bye from wherever the
// window closed. Receives are not, and must not be -- one reader, because a second one would
// take half of somebody's message.
//
// Conn deliberately owns no timeout. A peer whose process is killed closes its socket and the
// reader gets io.EOF, which is the whole of what "killing the guest leaves the host in a
// defined state" needs; a peer whose machine loses power sends nothing at all, and the answer
// to that is the caller's SetReadDeadline on the net.Conn it owns and this package does not.
// §10.2.6's kStallTimeout is the lock-step equivalent and does not apply: nothing here waits
// on the peer to take the next step, because the two simulations do not share one.
//
// The one exception is the handshake, and it is MeetWithin's rather than Conn's: before a match
// is agreed, silence means only that whatever answered is not going to race.
type Conn struct {
	rw io.ReadWriter

	mu  sync.Mutex
	out []byte // Send's scratch, under mu

	in []byte // Recv's scratch; one reader, so no lock

	// matchID is 0 until Meet has agreed one, and thereafter the low 32 bits of the match
	// seed. peerSlot is likewise -1 until then. Both are written by Meet before any other
	// goroutine has a reference to this Conn, and read-only afterwards.
	matchID  uint32
	peerSlot int8

	// peerFrame is the newest frame the peer has reported a standing at, and only Recv
	// touches it. See checkSlot for why it is a rule and not a statistic.
	peerFrame uint32

	// heard is whether the peer has sent a length prefix yet, and only Recv touches it. The
	// first four bytes a connection carries say more than any later four can (StrangerError),
	// and a handshake that timed out before them was talking to nobody (MeetWithin).
	heard bool
}

// NewConn wraps a connection. Closing it stays the caller's job: this type reads and writes,
// and a package that closed a socket it did not open would be one the tests could not run over
// net.Pipe.
func NewConn(rw io.ReadWriter) *Conn {
	return &Conn{rw: rw, peerSlot: -1}
}

// MatchID is the match this connection has agreed on, or 0 before Meet has run.
func (c *Conn) MatchID() uint32 { return c.matchID }

// Send writes one already-encoded message, length-prefixed.
//
// One Write for prefix and payload together. Not for speed -- these are 32-byte messages a few
// times a second -- but because a Write per part is a Write another goroutine can land between,
// and the mutex above would then be guarding the wrong thing.
func (c *Conn) Send(msg []byte) error {
	if len(msg) < HeaderSize {
		return short("the message being sent", len(msg), HeaderSize)
	}
	if len(msg) > MaxMsg {
		return fmt.Errorf("netplay: refusing to send %d bytes, the limit is %d",
			len(msg), MaxMsg)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.out = append(c.out[:0], 0, 0, 0, 0)
	be32(c.out, uint32(len(msg)))
	c.out = append(c.out, msg...)
	if _, err := c.rw.Write(c.out); err != nil {
		return fmt.Errorf("netplay: sending %s: %w", MsgName(msg[3]), err)
	}
	return nil
}

// Recv returns the next message this build speaks.
//
// Three outcomes other than a message, and they are three different things:
//
//   - A msgType nothing has allocated is skipped and the read goes round again, which is
//     §10.4.10's last rule. A peer from a later version sending a message this one has never
//     heard of is not an error; it is the forward compatibility that rule buys.
//   - A msgType that *is* allocated and that this build does not speak is an error naming it.
//     That is this package's addition to the rule and the package comment argues it: a peer
//     sending MsgInputFrames is a lock-step build, and two builds silently ignoring each other
//     is the one failure mode worse than an error message.
//   - io.EOF, unchanged and unwrapped, when the peer has gone. Callers test for it, and a race
//     that treats a departed opponent as a forfeit (Outcome) needs that to be the plain
//     sentinel rather than something with a sentence stuck to the front of it.
func (c *Conn) Recv() (Msg, error) {
	for {
		payload, err := c.frame()
		if err != nil {
			return nil, err
		}
		t, matchID, err := parseHeader(payload)
		if err != nil {
			return nil, err
		}
		if !allocated(t) {
			continue
		}
		if !spoken(t) {
			return nil, fmt.Errorf("%w: peer sent %s, which belongs to a lock-step match; "+
				"this build races and speaks the handshake, MsgStanding and MsgBye",
				ErrProtocol, MsgName(t))
		}
		// The matchID check is §10.4.4's, with its reason changed by the transport. There it
		// rejects a datagram left over from a previous match; here a stream carries only its
		// own match, so a wrong matchID cannot be a stale packet and can only be a peer that
		// has lost track of which game it is in. Loud, therefore, rather than dropped.
		if want := c.matchID; t != MsgHello && matchID != want {
			return nil, fmt.Errorf("%w: %s carries matchID 0x%08X, this match is 0x%08X",
				ErrProtocol, MsgName(t), matchID, want)
		}
		msg, err := decode(t, payload)
		if err != nil {
			return nil, err
		}
		if err := c.checkSlot(msg); err != nil {
			return nil, err
		}
		return msg, nil
	}
}

// checkSlot holds the peer to the identity it agreed to, and its standings to the order they
// happened in.
//
// **The slot.** Over one connection the sender is never in doubt -- there is one other end --
// so senderSlot is redundant as addressing and is not redundant as a check. §10.4.7's table is
// why it is worth making: player 1 and player 2 are not interchangeable in the original, and a
// peer that thinks it is player 1 when we think we are is a peer whose result we would be
// scoring against the wrong name. Before Meet has run there is nothing to check against, and
// Hello is where the slots get decided, so both are skipped.
//
// **The frame.** A standing whose Frame is behind one already received is refused. §10.4.4
// drops an out-of-order input word silently and is right to -- there, packets arrive out of
// order as a matter of course, and a retransmission is normal traffic. Here the transport is a
// stream that cannot reorder, so a standing from the past is not a late packet; it is a peer
// whose own frame counter went backwards, and since Frame is the last tie-break in Winner, a
// peer that can wind it back can choose to win. Equal frames are allowed: two things worth
// reporting can happen in one frame.
func (c *Conn) checkSlot(msg Msg) error {
	if c.peerSlot < 0 {
		return nil
	}
	var slot uint8
	switch m := msg.(type) {
	case *Report:
		slot = m.Slot
		if m.Frame < c.peerFrame {
			return fmt.Errorf("%w: peer's standing is at frame %d, behind the %d it has "+
				"already reported", ErrProtocol, m.Frame, c.peerFrame)
		}
		c.peerFrame = m.Frame
	case *Bye:
		slot = m.Slot
	default:
		return nil
	}
	if int8(slot) != c.peerSlot {
		return fmt.Errorf("%w: %s says it is from player %d, the peer is player %d",
			ErrProtocol, MsgName(msg.msgType()), slot+1, c.peerSlot+1)
	}
	return nil
}

// frame reads one length-prefixed message and returns the message, prefix stripped.
//
// The returned slice is Conn's own buffer and is valid until the next Recv. Every decoder in
// this package copies what it keeps, so nothing outlives it; a decoder that held on to it
// would be a bug that only showed up under load, which is why the ones here are written to
// take a []byte and return a value rather than to take a *Msg and fill it in.
func (c *Conn) frame() ([]byte, error) {
	var hdr [lenPrefix]byte
	if _, err := io.ReadFull(c.rw, hdr[:]); err != nil {
		// io.ReadFull turns a clean end-of-stream into io.EOF and a partial read into
		// ErrUnexpectedEOF. Both mean the peer is gone, and only the first is expected;
		// the second gets the sentence, because a connection that died between the length
		// and the message is worth telling apart from one that closed politely.
		if errors.Is(err, io.EOF) {
			return nil, io.EOF
		}
		return nil, fmt.Errorf("netplay: reading the length of the next message: %w", err)
	}
	first := !c.heard
	c.heard = true
	n := u32(hdr[:])
	if n < HeaderSize || n > MaxMsg {
		if first {
			return nil, StrangerError(hdr)
		}
		return nil, fmt.Errorf("%w: message length %d is outside [%d, %d]",
			ErrProtocol, n, HeaderSize, MaxMsg)
	}
	if cap(c.in) < int(n) {
		c.in = make([]byte, n)
	}
	c.in = c.in[:n]
	if _, err := io.ReadFull(c.rw, c.in); err != nil {
		return nil, fmt.Errorf("netplay: reading a %d-byte message: %w", n, err)
	}
	return c.in, nil
}

// A StrangerError is an impossible length in the first four bytes a connection carried, and it is
// both of the things those bytes can mean (docs/IMPROVEMENTS.md 4.32). It is ErrMagic, because
// what opens with an impossible length is nearly always another program. An SSH server's banner
// opens "SSH-" and a web server's reply "HTTP", and read as a length each is over a gigabyte.
// It is also ErrProtocol, which is what the same four bytes mean anywhere later in a stream. The
// bytes are quoted, because they usually name whatever answered. They are the value, and
// exported, so that a screen can quote them without the sentence around them (4.33).
type StrangerError [lenPrefix]byte

func (e StrangerError) Error() string {
	return fmt.Sprintf("%v: the other end opened with %q, which is not how a gliderGo message "+
		"starts, so it is some other program", ErrMagic, e[:])
}

func (StrangerError) Unwrap() []error { return []error{ErrMagic, ErrProtocol} }

// decode turns a validated envelope and its payload into one of the four messages.
func decode(t uint8, b []byte) (Msg, error) {
	switch t {
	case MsgHello:
		return decodeHello(b)
	case MsgMatchStart:
		return decodeMatchStart(b)
	case MsgStanding:
		return decodeReport(b)
	case MsgBye:
		return decodeBye(b)
	}
	// Unreachable: Recv has already refused everything else. Kept as an error rather than a
	// panic because "unreachable" is a claim about two functions agreeing, and the two are
	// not in the same file.
	return nil, fmt.Errorf("%w: no decoder for %s", ErrProtocol, MsgName(t))
}

// Bye is MsgBye (§10.4.5): the peer is leaving on purpose.
//
// It carries the 18-byte lock-step header with count = 0, which is §10.4.5's definition and
// not this package's choice. Ten of those bytes are a frame window a race has no use for, and
// they are sent as zeroes rather than dropped, because a message that is *almost* the
// specified one is worse than either alternative.
type Bye struct {
	Slot uint8 // the leaving player's slot, 0 or 1
}

func (*Bye) msgType() uint8 { return MsgBye }

// byeSize is §10.4.5's 18-byte header.
const byeSize = 18

// EncodeBye builds a MsgBye. Exported because the shutdown path sends one from outside any
// match loop, and because the tests that matter for it are the ones that feed a hand-built
// message to the other side.
func EncodeBye(matchID uint32, slot uint8) []byte {
	b := make([]byte, byeSize)
	header(b, MsgBye, matchID)
	b[8] = slot
	// b[9] is §10.4.4's count, 0 here. b[10:18] are firstFrame and ackFrame, both 0. The
	// zeroes are the message: §10.4.10's third rule is that no byte on the wire is padding
	// with undefined contents, because that is the bug that made the original's own demo
	// resource non-reproducible (§8.5 finding 7).
	return b
}

func decodeBye(b []byte) (Msg, error) {
	if len(b) < byeSize {
		return nil, short("MsgBye", len(b), byeSize)
	}
	slot := b[8]
	if slot > 1 {
		return nil, fmt.Errorf("%w: MsgBye senderSlot is %d, must be 0 or 1", ErrProtocol, slot)
	}
	return &Bye{Slot: slot}, nil
}

// Bye sends a voluntary disconnect on this connection. The caller closes the socket
// afterwards; this only says why.
func (c *Conn) Bye(slot uint8) error { return c.Send(EncodeBye(c.matchID, slot)) }
