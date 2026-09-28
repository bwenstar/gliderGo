package netplay

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"
)

// duplex is a read end and a write end pretending to be a connection.
type duplex struct {
	r, w *os.File
}

func (d duplex) Read(p []byte) (int, error)  { return d.r.Read(p) }
func (d duplex) Write(p []byte) (int, error) { return d.w.Write(p) }

// end is one side of a test connection, with the two files kept so that a test can hang up
// abruptly or write bytes no encoder in this package would produce.
type end struct {
	*Conn
	r, w *os.File
}

// kill closes this side's writing end: a clean end of stream, mid-match, with no MsgBye, which is
// what a process killed with nothing unread looks like. docs/PLAN.md Stage 3's second acceptance
// clause is this. Over TCP a killed process often has something unread, and is reset (Race.read).
func (e *end) kill() { e.w.Close() }

// pair gives two Conns talking to each other.
//
// **os.Pipe and not net.Pipe**, and the reason is Meet. net.Pipe is synchronous -- a Write
// blocks until the other end Reads it -- so two peers that both open with a Hello, which is
// what a symmetric handshake means, deadlock in a test and not in reality, where a 31-byte
// message goes into a socket buffer and the sender carries on. A kernel pipe has the buffer
// that makes the test resemble the transport. No listener, no port, no address either: NewConn
// takes an io.ReadWriter precisely so that the protocol can be tested without the two machines
// this mode is for.
func pair(t *testing.T) (*end, *end) {
	t.Helper()
	r1, w1, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	r2, w2, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	a := &end{Conn: NewConn(duplex{r1, w2}), r: r1, w: w2}
	b := &end{Conn: NewConn(duplex{r2, w1}), r: r2, w: w1}
	t.Cleanup(func() {
		r1.Close()
		w1.Close()
		r2.Close()
		w2.Close()
	})
	return a, b
}

func TestEnvelopeRoundTrips(t *testing.T) {
	b := make([]byte, HeaderSize)
	header(b, MsgStanding, 0xDEADBEEF)
	if got := u16(b); got != Magic {
		t.Errorf("magic = 0x%04X, want 0x%04X", got, Magic)
	}
	if b[2] != Version {
		t.Errorf("version byte = %d, want %d", b[2], Version)
	}
	typ, id, err := parseHeader(b)
	if err != nil {
		t.Fatalf("parseHeader: %v", err)
	}
	if typ != MsgStanding {
		t.Errorf("msgType = 0x%02X, want 0x%02X", typ, MsgStanding)
	}
	if id != 0xDEADBEEF {
		t.Errorf("matchID = 0x%08X, want 0xDEADBEEF", id)
	}
	// The envelope is big-endian by §10.4.10's first rule, and the only way to check that is
	// to look at the bytes: a round trip through this package's own accessors would pass
	// whichever order they used.
	want := []byte{0x47, 0x4C, 0x01, 0x30, 0xDE, 0xAD, 0xBE, 0xEF}
	if !bytes.Equal(b, want) {
		t.Errorf("envelope = % X, want % X", b, want)
	}
}

func TestParseHeaderRejectsWhatItShould(t *testing.T) {
	good := make([]byte, HeaderSize)
	header(good, MsgHello, 0)

	short7 := good[:7]
	badMagic := append([]byte(nil), good...)
	badMagic[0] = 'X'
	badVersion := append([]byte(nil), good...)
	badVersion[2] = Version + 1

	for _, c := range []struct {
		name string
		in   []byte
		want error
	}{
		{"truncated", short7, ErrShort},
		{"wrong magic", badMagic, ErrMagic},
		{"wrong version", badVersion, ErrVersion},
	} {
		if _, _, err := parseHeader(c.in); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", c.name, err, c.want)
		}
	}

	// The order matters and is argued in parseHeader: a message with both a wrong magic and
	// a wrong version must complain about the magic, because if the magic is wrong then
	// nothing else in the buffer is known to be a field at all.
	both := append([]byte(nil), badMagic...)
	both[2] = Version + 9
	if _, _, err := parseHeader(both); !errors.Is(err, ErrMagic) {
		t.Errorf("magic and version both wrong: err = %v, want the magic complaint", err)
	}
}

func TestFramingCarriesAMessageUnchanged(t *testing.T) {
	a, b := pair(t)
	sent := EncodeStanding(7, 1, Standing{State: Racing, Rooms: 3, Frame: 99})
	if err := a.Send(sent); err != nil {
		t.Fatalf("Send: %v", err)
	}
	got, err := b.frame()
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	if !bytes.Equal(got, sent) {
		t.Errorf("framed message came back changed:\n got % X\nwant % X", got, sent)
	}
}

func TestRecvSkipsAMessageFromTheFuture(t *testing.T) {
	// §10.4.10's last rule: an unknown msgType is ignored, silently. 0x77 is in no range the
	// specification allocates, so a peer sending one is a later version of this program, and
	// the whole point of the rule is that this one keeps going.
	a, b := pair(t)
	future := make([]byte, HeaderSize+4)
	header(future, 0x77, 0)
	if err := a.Send(future); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if err := a.Send(EncodeBye(0, 0)); err != nil {
		t.Fatalf("Send: %v", err)
	}
	msg, err := b.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if _, ok := msg.(*Bye); !ok {
		t.Fatalf("Recv returned %T, want the *Bye that followed the unknown message", msg)
	}
}

func TestRecvRefusesALockStepPeerOutLoud(t *testing.T) {
	// The package's own addition to that rule: MsgInputFrames is allocated and not spoken,
	// so ignoring it would leave a lock-step build and a race build waiting on each other
	// for ever. Every allocated-and-not-spoken type is checked, because the argument is
	// about the class and not about one number.
	for _, typ := range []uint8{MsgInputFrames, MsgChecksum, MsgAck, MsgProgress} {
		a, b := pair(t)
		msg := make([]byte, HeaderSize+10)
		header(msg, typ, 0)
		if err := a.Send(msg); err != nil {
			t.Fatalf("Send: %v", err)
		}
		_, err := b.Recv()
		if !errors.Is(err, ErrProtocol) {
			t.Errorf("%s: err = %v, want a protocol error", MsgName(typ), err)
		}
		if err != nil && !bytes.Contains([]byte(err.Error()), []byte(MsgName(typ))) {
			t.Errorf("%s: error does not name the message: %v", MsgName(typ), err)
		}
	}
}

func TestFrameRejectsImpossibleLengths(t *testing.T) {
	// A length below the envelope cannot be a message, and one above MaxMsg is a peer asking
	// this process to allocate on its behalf. Both are refused before the read, which is the
	// only order in which the second check does anything -- and it is the reason this test
	// can write a prefix claiming four gigabytes and then nothing at all.
	for _, n := range []uint32{0, HeaderSize - 1, MaxMsg + 1, 0xFFFFFFFF} {
		a, b := pair(t)
		var prefix [4]byte
		be32(prefix[:], n)
		if _, err := a.w.Write(prefix[:]); err != nil {
			t.Fatalf("writing the prefix: %v", err)
		}
		if _, err := b.Recv(); !errors.Is(err, ErrProtocol) {
			t.Errorf("length %d: err = %v, want a protocol error", n, err)
		}
	}
}

func TestSendRefusesWhatCannotBeReceived(t *testing.T) {
	a, _ := pair(t)
	if err := a.Send(make([]byte, HeaderSize-1)); !errors.Is(err, ErrShort) {
		t.Errorf("sending 7 bytes: err = %v, want %v", err, ErrShort)
	}
	if err := a.Send(make([]byte, MaxMsg+1)); err == nil {
		t.Error("sending MaxMsg+1 bytes succeeded; the limit has to hold on both sides or " +
			"one peer can always make the other fail")
	}
}

func TestRecvReturnsPlainEOFWhenThePeerGoes(t *testing.T) {
	// Unwrapped and not decorated, because docs/PLAN.md Stage 3's acceptance clause is about
	// a killed process and the code that handles it tests for io.EOF. A wrapped one would
	// still satisfy errors.Is; this is the stricter check on purpose, since a caller reaching
	// for == is the likelier mistake and this is the one place it is safe.
	a, b := pair(t)
	a.kill()
	_, err := b.Recv()
	if err != io.EOF { //nolint:errorlint // the point of the test
		t.Errorf("Recv after the peer closed: err = %v (%T), want the io.EOF sentinel", err, err)
	}
}

func TestHalfAMessageIsNotAnEndOfStream(t *testing.T) {
	// A connection that dies between the length prefix and the message is not a peer leaving
	// politely, and the two must not arrive as the same error: one is "your opponent quit",
	// the other is "the network dropped".
	a, b := pair(t)
	var prefix [4]byte
	be32(prefix[:], standingSize)
	if _, err := a.w.Write(prefix[:]); err != nil {
		t.Fatalf("writing the prefix: %v", err)
	}
	if _, err := a.w.Write(make([]byte, 4)); err != nil {
		t.Fatalf("writing half a message: %v", err)
	}
	a.kill()
	_, err := b.Recv()
	if err == nil || errors.Is(err, io.EOF) {
		t.Errorf("err = %v, want something other than EOF for a truncated message", err)
	}
}

func TestEveryAllocatedTypeIsNamedAndEverySpokenTypeIsAllocated(t *testing.T) {
	// Three tables -- MsgName, spoken and allocated -- that have to agree, and a sweep is
	// cheaper than remembering. The failure this catches is a fifth message added to two of
	// the three, which would make it either unnameable in an error or silently dropped.
	named := 0
	for i := 0; i < 256; i++ {
		typ := uint8(i)
		name := MsgName(typ)
		isHex := len(name) == 4 && name[0] == '0' && name[1] == 'x'
		switch {
		case allocated(typ) && isHex:
			t.Errorf("0x%02X is allocated and MsgName has no name for it", typ)
		case !allocated(typ) && !isHex:
			t.Errorf("0x%02X is not allocated and MsgName calls it %q", typ, name)
		}
		if spoken(typ) && !allocated(typ) {
			t.Errorf("0x%02X is spoken and not allocated; the number is not reserved", typ)
		}
		if allocated(typ) {
			named++
		}
	}
	// Seven from §10.4 plus MsgStanding. A count, so that a number quietly added to the
	// allocated set without a line in the package comment fails here.
	if named != 8 {
		t.Errorf("%d allocated msgTypes, expected 8 (§10.4's seven and MsgStanding)", named)
	}
}

func TestMatchIDMustMatchOnceTheMatchHasStarted(t *testing.T) {
	// §10.4.4's stale-packet check, which on a stream means something else -- see Recv. The
	// peer here sends a standing for a match nobody is playing.
	a, b := pair(t)
	b.matchID = 0x11111111
	b.peerSlot = 0
	if err := a.Send(EncodeStanding(0x22222222, 0, Standing{})); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if _, err := b.Recv(); !errors.Is(err, ErrProtocol) {
		t.Errorf("err = %v, want a protocol error about the matchID", err)
	}
}
