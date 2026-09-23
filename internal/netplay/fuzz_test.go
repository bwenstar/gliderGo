package netplay

// Hostile bytes, on purpose (docs/IMPROVEMENTS.md 4.30).
//
// The race puts a socket in front of Recv, and whatever answers the port writes the bytes. So the
// two things that read them, Recv and Meet, are fuzz targets. Plain `go test` runs each one over
// its seeds and nothing else. `make fuzz` runs the engine, which writes any input that fails to
// testdata/fuzz/<target>/, and that file is then committed: from there plain `go test` replays it
// for good.
//
// The seeds are built from this package's encoders rather than kept as files, so they cannot drift
// from the format. TestEveryMeetSeedEndsTheWayItSays checks that each still reaches the outcome it
// is named for, because a seed that has stopped reaching its path is a seed that tests nothing.

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"
)

// framed lays messages out on a stream the way Send does: each one behind its length.
func framed(msgs ...[]byte) []byte {
	var b []byte
	for _, m := range msgs {
		at := len(b)
		b = append(b, make([]byte, lenPrefix)...)
		be32(b[at:], uint32(len(m)))
		b = append(b, m...)
	}
	return b
}

// encodeMsg is a received message sent back the way this package would send it. matchID is the
// connection's, which every message but a Hello carries (Recv checks that before decoding).
func encodeMsg(t *testing.T, matchID uint32, m Msg) []byte {
	t.Helper()
	switch m := m.(type) {
	case *Hello:
		return EncodeHello(*m)
	case *MatchStart:
		return EncodeMatchStart(*m)
	case *Report:
		return EncodeStanding(matchID, m.Slot, m.Standing)
	case *Bye:
		return EncodeBye(matchID, m.Slot)
	}
	t.Fatalf("Recv returned a %T, which this package cannot encode", m)
	return nil
}

// sorted is whether an error is one a caller can tell apart from the others, and so one
// cmd/glidergo's meetWords has words for. That is the package's sentinels and the two ways a
// stream ends. The race's screens say something different for each, and an error that is none of
// them reaches a player as its raw sentence.
func sorted(err error) bool {
	for _, s := range []error{io.EOF, io.ErrUnexpectedEOF, ErrMagic, ErrVersion, ErrShort,
		ErrProtocol, ErrHouse, ErrEngine, ErrRules} {
		if errors.Is(err, s) {
			return true
		}
	}
	return false
}

// peerStream is a Conn reading the given bytes, and throwing away everything it sends.
func peerStream(b []byte) *Conn {
	return NewConn(struct {
		io.Reader
		io.Writer
	}{bytes.NewReader(b), io.Discard})
}

// The seeds for FuzzRecv: a real match's traffic, a peer from later and one from lock-step, the
// two programs a port is most often held by, and the ways a length can lie.
func recvSeeds() [][]byte {
	const id = 0x00C0FFEE
	h := hello(7, houseHashA)
	start := EncodeMatchStart(MatchStart{Seed: 0xABCD0000_00C0FFEE, HouseHash: houseHashA})
	stand := func(frame uint32) []byte {
		return EncodeStanding(id, 1, Standing{State: Racing, Rooms: 3, Room: 40, Floor: 2,
			Suite: 1, Mortals: 4, Score: 1500, Frame: frame})
	}
	later := EncodeBye(id, 1)
	later[3] = 0x77 // unallocated, so skipped
	lockstep := EncodeBye(id, 1)
	lockstep[3] = MsgInputFrames
	v2 := EncodeBye(id, 1)
	v2[2] = 2
	tail := append(EncodeHello(h), "a later build's field"...)
	return [][]byte{
		nil,
		framed(EncodeHello(h)),
		framed(tail),
		framed(EncodeHello(h), start, stand(10), stand(10), stand(40), EncodeBye(id, 1)),
		framed(stand(40), stand(10)),
		framed(later, stand(5), later),
		framed(lockstep),
		framed(v2),
		[]byte("SSH-2.0-OpenSSH_9.6\r\n"),
		[]byte("HTTP/1.1 400 Bad Request\r\n\r\n"),
		{0, 0, 0, HeaderSize},
		framed(EncodeHello(h))[:30],
		append(framed(stand(1)), 0xFF, 0xFF, 0xFF, 0xFF),
	}
}

// FuzzRecv reads whatever a peer sends, until the stream ends or Recv refuses it.
//
// matchID and peerSlot put the connection where Meet would leave it, so that the checks only a
// match makes -- the matchID on every message, the slot, the order of the standings -- are fuzzed
// too. A peerSlot below 0 is a connection before Meet, which has neither.
//
// What must hold, whatever the bytes:
//   - no panic, and no message and no error together;
//   - every error sorted (see sorted), and io.EOF only ever the bare sentinel, which is what
//     Recv's comment promises the race;
//   - every message one this build speaks, and decoded to the same value when sent back through
//     its encoder -- nothing is kept that the encoder could not have written;
//   - after a match, every standing and Bye from the peer's own slot, and no standing behind one
//     before it;
//   - no message changed by the reads after it. Recv decodes out of a buffer it reuses, and a
//     decoder that kept a slice of it would give back a house name that changed under its holder.
func FuzzRecv(f *testing.F) {
	for _, s := range recvSeeds() {
		f.Add(uint32(0), int8(-1), s)
		f.Add(uint32(0x00C0FFEE), int8(1), s)
	}
	f.Fuzz(func(t *testing.T, matchID uint32, peerSlot int8, stream []byte) {
		c := peerStream(stream)
		if peerSlot >= 0 {
			c.matchID, c.peerSlot = matchID, peerSlot&1
		}
		type got struct {
			msg Msg
			enc []byte
		}
		var seen []got
		var frame uint32
		for {
			if len(seen) > len(stream)/(lenPrefix+HeaderSize) {
				t.Fatalf("%d messages out of %d bytes, and none can be shorter than %d",
					len(seen), len(stream), lenPrefix+HeaderSize)
			}
			msg, err := c.Recv()
			if err != nil {
				if msg != nil {
					t.Errorf("Recv returned a %T and an error, %v", msg, err)
				}
				if !sorted(err) {
					t.Errorf("Recv's error is none of the ones a caller can sort: %v", err)
				}
				if errors.Is(err, io.EOF) && err != io.EOF {
					t.Errorf("Recv wrapped io.EOF, which the race tests for bare: %v", err)
				}
				break
			}
			if msg == nil {
				t.Fatal("Recv returned neither a message nor an error")
			}
			if !spoken(msg.msgType()) {
				t.Fatalf("Recv returned a %s, which this build does not speak",
					MsgName(msg.msgType()))
			}
			enc := encodeMsg(t, c.matchID, msg)
			back, err := decode(msg.msgType(), enc)
			if err != nil {
				t.Fatalf("a received %s does not decode once encoded again: %v",
					MsgName(msg.msgType()), err)
			}
			if !reflect.DeepEqual(back, msg) {
				t.Fatalf("a received %s is %+v, and %+v once encoded and decoded again",
					MsgName(msg.msgType()), msg, back)
			}
			if c.peerSlot >= 0 {
				switch m := msg.(type) {
				case *Report:
					if int8(m.Slot) != c.peerSlot {
						t.Errorf("a standing from slot %d reached a match whose peer is %d",
							m.Slot, c.peerSlot)
					}
					if m.Frame < frame {
						t.Errorf("a standing at frame %d came after one at %d", m.Frame, frame)
					}
					frame = m.Frame
				case *Bye:
					if int8(m.Slot) != c.peerSlot {
						t.Errorf("a Bye from slot %d reached a match whose peer is %d",
							m.Slot, c.peerSlot)
					}
				}
			}
			seen = append(seen, got{msg, enc})
		}
		for i, s := range seen {
			if now := encodeMsg(t, c.matchID, s.msg); !bytes.Equal(now, s.enc) {
				t.Errorf("message %d, a %s, changed after the reads that followed it",
					i, MsgName(s.msg.msgType()))
			}
		}
	})
}

// A peer that hangs up after a message's length and before the end of the message has not said
// goodbye, however little of the message it sent. Recv used to pass on io.ReadFull's io.EOF for
// none of it, wrapped, and the race's reader took that for a clean hang-up and reported nothing.
// FuzzRecv found it on its seeds.
func TestAHangUpInsideAMessageIsNotAPoliteOne(t *testing.T) {
	msg := framed(EncodeBye(0, 0))
	for _, n := range []int{0, 1, byeSize - 1} {
		_, err := peerStream(msg[:lenPrefix+n]).Recv()
		if !errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			t.Errorf("%d bytes of an %d-byte message, then a hang-up: %v", n, byeSize, err)
		}
	}
}

// A meetSeed is one peer's side of a handshake and how Meet should end it. want is nil for a
// match, and otherwise what errors.Is finds in the error.
type meetSeed struct {
	name   string
	nonce  uint64 // this side's
	stream []byte // the peer's
	want   error
}

func meetSeeds() []meetSeed {
	peer := func(edit func(*Hello)) []byte {
		h := hello(9, houseHashA)
		if edit != nil {
			edit(&h)
		}
		return EncodeHello(h)
	}
	// Nonce 9 against 7 makes this side player 2, which then waits for the MatchStart.
	lo := hello(7, houseHashA)
	start := MatchStart{Seed: mixSeed(7, 9), HouseHash: houseHashA}
	wrongSeed := start
	wrongSeed.Seed++
	later := EncodeBye(0, 0)
	later[3] = 0x77
	whole := framed(peer(nil))
	return []meetSeed{
		{"player 1", 7, whole, nil},
		{"player 2", 9, framed(EncodeHello(lo), EncodeMatchStart(start)), nil},
		{"after a message from later", 7, framed(later, peer(nil)), nil},
		{"with a longer hello", 7, framed(append(peer(nil), 1, 2, 3)), nil},
		{"no version in common", 7, framed(peer(func(h *Hello) { h.Versions = 1 << 9 })), ErrVersion},
		{"another engine", 7, framed(peer(func(h *Hello) { h.Engine = engineB })), ErrEngine},
		{"another house", 7, framed(peer(func(h *Hello) { h.HouseHash = houseHashB })), ErrHouse},
		{"a gated rule", 7, framed(peer(func(h *Hello) { h.Rules = RuleAssisted })), ErrRules},
		{"the same nonce", 9, whole, ErrProtocol},
		{"a standing first", 7, framed(EncodeStanding(0, 1, Standing{})), ErrProtocol},
		{"the wrong seed", 9, framed(EncodeHello(lo), EncodeMatchStart(wrongSeed)), ErrProtocol},
		{"hung up", 7, nil, io.EOF},
		{"hung up before the start", 9, framed(EncodeHello(lo)), io.EOF},
		{"hung up partway", 7, whole[:len(whole)/2], io.ErrUnexpectedEOF},
		{"a truncated hello", 7, framed(peer(nil)[:helloBeforeName+2]), ErrShort},
		{"an SSH server", 7, []byte("SSH-2.0-OpenSSH_9.6\r\n"), ErrMagic},
		{"a web server", 7, []byte("HTTP/1.1 400 Bad Request\r\n\r\n"), ErrMagic},
	}
}

func TestEveryMeetSeedEndsTheWayItSays(t *testing.T) {
	for _, s := range meetSeeds() {
		_, err := Meet(peerStream(s.stream), hello(s.nonce, houseHashA))
		switch {
		case s.want == nil && err != nil:
			t.Errorf("%s: Meet failed: %v", s.name, err)
		case s.want != nil && !errors.Is(err, s.want):
			t.Errorf("%s: Meet returned %v, want %v", s.name, err, s.want)
		}
	}
}

// FuzzMeet is a handshake with whatever answered, from this side's hello to a match or an error.
//
// A match it returns is checked against the rules worked again from the bytes: the peer's hello is
// read back off the stream, and the match has to be the one §10.4.7 makes of the two hellos. Meet
// is the only thing that decides whether a race happens, so a match it should have refused is the
// failure that matters here, more than any error. What this side sent is read back too: its hello,
// and a MatchStart only if it is player 1.
func FuzzMeet(f *testing.F) {
	for _, s := range meetSeeds() {
		f.Add(s.nonce, s.stream)
	}
	f.Fuzz(func(t *testing.T, nonce uint64, stream []byte) {
		var sent bytes.Buffer
		c := NewConn(struct {
			io.Reader
			io.Writer
		}{bytes.NewReader(stream), &sent})
		local := hello(nonce, houseHashA)
		m, err := Meet(c, local)
		if err != nil {
			if !sorted(err) {
				t.Fatalf("Meet's error is none of the ones a caller can sort: %v", err)
			}
			return
		}

		// The peer's side, worked again.
		ref := peerStream(stream)
		msg, err := ref.Recv()
		peer, ok := msg.(*Hello)
		if err != nil || !ok {
			t.Fatalf("Meet agreed a match, and the stream opens with %T, %v", msg, err)
		}
		switch {
		case local.Versions&peer.Versions == 0:
			t.Fatalf("a match with a peer speaking %016b, and this side %016b",
				peer.Versions, local.Versions)
		case peer.Engine != local.Engine:
			t.Fatalf("a match with engine %016X, and this side's is %016X", peer.Engine, local.Engine)
		case peer.HouseHash != local.HouseHash:
			t.Fatalf("a match on house %016X, and this side has %016X",
				peer.HouseHash, local.HouseHash)
		case (peer.Rules^local.Rules)&RulesGated != 0:
			t.Fatalf("a match with rules %04X against %04X", peer.Rules, local.Rules)
		case peer.Nonce == local.Nonce:
			t.Fatalf("a match with both nonces %016X", local.Nonce)
		}
		lo, hi := min(local.Nonce, peer.Nonce), max(local.Nonce, peer.Nonce)
		want := Match{
			Seed:          mixSeed(lo, hi),
			InputDelay:    max(local.InputDelay, peer.InputDelay),
			PeerHouseName: peer.HouseName,
			PeerNeighbors: peer.Neighbors,
			PeerRelease:   peer.Release,
			PeerRules:     peer.Rules,
		}
		want.ID = uint32(want.Seed)
		if local.Nonce > peer.Nonce {
			want.Slot = 1
			ref.matchID, ref.peerSlot = want.ID, 0
			msg, err := ref.Recv()
			start, ok := msg.(*MatchStart)
			if err != nil || !ok {
				t.Fatalf("player 2 has a match, and the peer's second message is %T, %v", msg, err)
			}
			if start.Slot != 0 || start.Seed != want.Seed || start.HouseHash != local.HouseHash ||
				start.InputDelay != want.InputDelay {
				t.Fatalf("player 2 accepted %+v, which the hellos do not make", *start)
			}
			want.StartFrame = start.StartFrame
		}
		if m != want {
			t.Fatalf("Meet agreed %+v; the two hellos make %+v", m, want)
		}
		if c.MatchID() != m.ID {
			t.Fatalf("the connection carries matchID %08X, and the match is %08X", c.MatchID(), m.ID)
		}

		// This side's own words.
		out := peerStream(sent.Bytes())
		msg, err = out.Recv()
		if h, ok := msg.(*Hello); err != nil || !ok || *h != local {
			t.Fatalf("this side opened with %T %+v, %v; want its own hello", msg, msg, err)
		}
		out.matchID, out.peerSlot = m.ID, int8(m.Slot)
		msg, err = out.Recv()
		if m.Slot == 1 {
			if err != io.EOF {
				t.Fatalf("player 2 sent a %T after its hello, %v", msg, err)
			}
			return
		}
		start, ok := msg.(*MatchStart)
		if err != nil || !ok {
			t.Fatalf("player 1 followed its hello with %T, %v", msg, err)
		}
		if start.Slot != 0 || start.Seed != m.Seed || start.HouseHash != local.HouseHash ||
			start.InputDelay != m.InputDelay {
			t.Fatalf("player 1 sent %+v for the match %+v", *start, m)
		}
		if msg, err := out.Recv(); err != io.EOF {
			t.Fatalf("player 1 sent a %T after its MatchStart, %v", msg, err)
		}
	})
}
