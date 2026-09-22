// Package netplay is the race: two copies of this port, on two machines, each simulating its
// own glider in its own world, with the network carrying nothing but how far each has got.
//
// **It is not the original's two-player mode.** That one puts two gliders in one room sharing
// one inventory, and it is internal/game's World.TwoPlayer (docs/PLAN.md 1.9, done). This is a
// different game. The distinction is load-bearing rather than pedantic: nothing in here makes
// the two simulations agree, because nothing has to. Each peer loads its own copy of the
// house, draws from its own RNG, and never learns where the other glider is -- so a desync is
// not a bug this mode can have, and the twenty-odd bytes a second it sends are the whole of
// its network cost. docs/PLAN.md §2 chose that over lock-step deliberately.
//
// **The wire format is not invented here, and that is the one thing worth knowing before
// reading the rest.** docs/analysis/determinism-networking.md §10.4 already specifies a
// complete protocol -- an 8-byte envelope, an allocated msgType space, and §10.4.10's seven
// encoding rules -- for the *lock-step* port that document is about, which is a far harder
// thing than this. It would have been easy to write a small JSON protocol instead, and it
// would have been the same mistake docs/IMPROVEMENTS.md 4.24 records: eight documents state a
// layout and the port went and used another one. So the envelope, the handshake and the
// numbers below are §10.4's, verbatim where they exist, and this package's own additions are
// the three named next.
//
// # What this package adds to §10.4, and why each one had to be added
//
// **A length prefix, because this is a stream.** §10.4's messages are datagram-shaped: each
// one begins with the magic and ends where the packet does. TCP has no packets, so every
// message here is preceded by a uint32 length (Conn.Send, conn.go). That is not a change to
// any message -- the envelope inside is byte-for-byte §10.4.4's -- and it is §10.4.10's own
// "explicit count before every variable-length array" rule applied one level up, to the
// message. docs/PLAN.md Stage 3 called for "length-prefixed JSON or a small binary framing";
// this is the second of those, with the framing the analysis already wrote.
//
// **One new message, MsgStanding (0x30).** The race needs a record sent on every meaningful
// change -- room, score, gliders left, rooms visited, alive or done. §10.4.8's MsgProgress
// (0x20) looks like that and is not: it is a 102-byte fixed part plus 290 bytes *per room*,
// a whole-match snapshot for resuming a dropped lock-step game. Sending 150 KB every time a
// player walks through a door would be absurd, and truncating it to the fields the race wants
// would be a second message wearing the first one's number. So 0x30 is new, chosen outside
// every range §10.4 allocates (0x0x per-frame, 0x1x setup, 0x2x snapshot) so that implementing
// any of those later cannot collide with it.
//
// **A known-but-not-spoken case.** §10.4.10 says to reject an unknown msgType silently, for
// forward compatibility. A peer that sends MsgInputFrames is not unknown, though -- it is a
// lock-step build talking to a race build, and going quiet would leave both sides waiting
// forever for a game the other one is not playing. Conn.Recv skips what it does not recognise
// and fails loudly, naming the message, on one it recognises and does not speak.
//
// # The fields the race does not need and sends anyway
//
// MsgHello and MsgMatchStart carry a kInputDelay, a matchSeed and a numNeighborsView that a
// race with separate worlds has no mechanical use for. They are still sent, and not out of
// obedience: two of them turn out to matter for *fairness*, which is what a race has instead
// of determinism.
//
//   - matchSeed: both peers seed their own World from it (Match.RandSeed), so the balloons
//     drift and the toasters fire the same way on both screens. Nothing breaks if they differ
//     -- the two worlds are independent -- but then one player raced an easier house than the
//     other, and the result means less.
//   - numNeighborsView: prefs.Neighbors is 1, 3 or 9, and a player rendering 9 can see hazards
//     two rooms away that a player rendering 1 cannot. §10.3.5 R3 is why this is informational
//     and not a compatibility gate -- the simulation behaves as if all nine neighbours exist
//     either way -- but it is exactly the kind of difference the UI should mention.
//   - kInputDelay: no use here at all, and honestly reported as such. It is in the format
//     because the format is a lock-step protocol's; a race sends 0 and ignores what it gets.
//
// The one field that *is* a gate is houseHash, and §10.4.7 is emphatic about it: the house
// file is simulation input, down to every blower's direction and every appliance's delay, so
// two peers on differently-built houses of the same name have no shared game to play. See
// HouseHash for what this port hashes, which is not quite what §10.4.7 says to hash, and why.
package netplay

import (
	"errors"
	"fmt"
)

// The envelope, from docs/analysis/determinism-networking.md §10.4.4's table. Every message
// in the protocol starts with these eight bytes, and the race's own MsgStanding starts with
// them too.
const (
	// Magic is 0x474C, "GL". It is not a security measure and not a version check; it is
	// what makes a peer that has connected to the wrong port fail on the first message
	// instead of on the fourth field of the fifth.
	Magic uint16 = 0x474C

	// Version is 1, the version §10.4 documents. It pins more than the layout: §10.4.6
	// puts the *hash algorithm* under the version, which is why HouseHash can be FNV-1a
	// without that choice needing a field of its own.
	Version uint8 = 1

	// HeaderSize is the 8 bytes of magic, version, msgType and matchID. Every message's
	// own fields start at offset 8.
	HeaderSize = 8
)

// The msgType space. All seven of §10.4's numbers are named here even though this package
// speaks four of them, because a number that is written down and unused is reserved, and a
// number that is merely unused gets reassigned. Which ones are live is in the comments and
// enforced by Conn.Recv.
const (
	// MsgInputFrames (§10.4.4) is the lock-step per-frame packet: one input word per
	// player per frame, and no entity state ever. Not spoken here.
	MsgInputFrames uint8 = 0x01

	// MsgChecksum (§10.4.5) is the desync detector, a hash of the simulation state every
	// eight frames. Not spoken here, and could not be: the two worlds are not the same
	// world, so their hashes are supposed to differ.
	MsgChecksum uint8 = 0x02

	// MsgAck (§10.4.5) keeps the liveness signal flowing while a lock-step peer has no
	// input to send. Not spoken here; the race has nothing to acknowledge, and a peer that
	// has gone quiet is detected by its connection closing.
	MsgAck uint8 = 0x03

	// MsgBye (§10.4.5) is a voluntary disconnect. **Spoken.** It is the difference between
	// "the other player quit" and "the network died", and those deserve different words on
	// screen even though the race's own answer to both is the same (Outcome, standing.go).
	MsgBye uint8 = 0x04

	// MsgHello (§10.4.7) opens the handshake. **Spoken.**
	MsgHello uint8 = 0x10

	// MsgMatchStart (§10.4.7) closes it. **Spoken.**
	MsgMatchStart uint8 = 0x11

	// MsgProgress (§10.4.8) is the whole-match snapshot for resuming a dropped lock-step
	// game -- 102 bytes plus 290 per room. Not spoken here; see the package comment for
	// why MsgStanding exists rather than a cut-down version of this.
	MsgProgress uint8 = 0x20

	// MsgStanding is this package's own, and the only message the race sends during play.
	// 0x30 because §10.4 allocates 0x0x, 0x1x and 0x2x and nothing above them.
	MsgStanding uint8 = 0x30
)

// MsgName is the name of a msgType, for diagnostics. Unallocated numbers come back as their
// hex, because an error that says "unknown message 0x77" is more use than one that says
// "unknown message" -- a peer sending 0x77 is a peer running something this build has never
// heard of, and the number is the only clue to what.
func MsgName(t uint8) string {
	switch t {
	case MsgInputFrames:
		return "MsgInputFrames"
	case MsgChecksum:
		return "MsgChecksum"
	case MsgAck:
		return "MsgAck"
	case MsgBye:
		return "MsgBye"
	case MsgHello:
		return "MsgHello"
	case MsgMatchStart:
		return "MsgMatchStart"
	case MsgProgress:
		return "MsgProgress"
	case MsgStanding:
		return "MsgStanding"
	}
	return "0x" + hex2(t)
}

func hex2(b byte) string {
	const digits = "0123456789ABCDEF"
	return string([]byte{digits[b>>4], digits[b&0xF]})
}

// spoken reports whether this build implements a msgType. The distinction between "not spoken"
// and "not allocated" is the whole of the package comment's third addition: one is a peer
// playing a different game, which must be said out loud, and the other is a peer from the
// future, which must be ignored.
func spoken(t uint8) bool {
	switch t {
	case MsgBye, MsgHello, MsgMatchStart, MsgStanding:
		return true
	}
	return false
}

// allocated reports whether §10.4 or this package has given a msgType a meaning.
func allocated(t uint8) bool {
	switch t {
	case MsgInputFrames, MsgChecksum, MsgAck, MsgBye,
		MsgHello, MsgMatchStart, MsgProgress, MsgStanding:
		return true
	}
	return false
}

// The errors, all sentinels so that a caller can tell the four apart with errors.Is and say
// something different about each. That matters more here than in most packages: three of the
// four are things a player did rather than things a program got wrong, and "your friend is
// running a different build of the house" needs to reach the screen as that sentence and not
// as a stack trace.
var (
	// ErrMagic means the first two bytes were not Magic. Almost always a connection to
	// something that is not this game.
	ErrMagic = errors.New("netplay: not a gliderGo message")

	// ErrVersion means the peer speaks a different protocol version. Loud by §10.4.10's
	// last rule, and there is nothing to negotiate: version 1 is the only one that exists.
	ErrVersion = errors.New("netplay: wrong protocol version")

	// ErrShort means a message ended inside itself. Wrapped with the field it ended in.
	ErrShort = errors.New("netplay: message truncated")

	// ErrProtocol means the peer sent something well-formed that the rules forbid --
	// a MsgStanding before the match started, a slot that contradicts the nonces, a
	// standing older than one already received. Every use of it names the rule.
	ErrProtocol = errors.New("netplay: protocol violation")

	// ErrHouse is the houseHash gate: the two sides are not holding the same house. The
	// one error in this package a player is *expected* to see, so Meet spells out both
	// house names and both hashes rather than just failing.
	ErrHouse = errors.New("netplay: the two sides have different houses")
)

// short is the one error message this package builds more than twice: a message that ended
// before one of its fields did. It names the field, because "truncated" alone leaves the
// reader to count offsets, and the whole reason every message here has a fixed layout is so
// that nobody has to.
func short(what string, have, need int) error {
	return fmt.Errorf("%w: %s needs %d bytes, message has %d", ErrShort, what, need, have)
}

// Msg is one decoded message. Four types implement it -- Hello, MatchStart, Report and Bye --
// and Conn.Recv returns them as this interface for the caller to switch on.
//
// The method is unexported on purpose. Msg is a closed set: a fifth implementation would be a
// message this package cannot encode, and the compiler is the right place to find that out.
type Msg interface {
	msgType() uint8
}

// header writes the eight envelope bytes into b, which must have room for them.
func header(b []byte, t uint8, matchID uint32) {
	be16(b, Magic)
	b[2] = Version
	b[3] = t
	be32(b[4:], matchID)
}

// parseHeader checks the envelope and returns the msgType and matchID.
//
// The order of the checks is the order of how much they cost to be wrong about: magic first,
// because a wrong magic means nothing else in the buffer can be trusted to be a field at all;
// version second, because a version 2 message may have a different layout from the third byte
// on. Only then is msgType a msgType.
func parseHeader(b []byte) (t uint8, matchID uint32, err error) {
	if len(b) < HeaderSize {
		return 0, 0, short("the message header", len(b), HeaderSize)
	}
	if got := u16(b); got != Magic {
		return 0, 0, fmt.Errorf("%w: magic is 0x%04X, want 0x%04X", ErrMagic, got, Magic)
	}
	if b[2] != Version {
		return 0, 0, fmt.Errorf("%w: peer speaks version %d, this build speaks %d",
			ErrVersion, b[2], Version)
	}
	return b[3], u32(b[4:]), nil
}

// The byte-level accessors. Big-endian and explicit-width throughout, which is §10.4.10's
// first two rules; hand-written rather than through encoding/binary's Read/Write because
// those work in reflection over structs and the point of §10.4.10 is that no field's width or
// position is left to a compiler's idea of a struct layout.
func be16(b []byte, v uint16) {
	b[0] = byte(v >> 8)
	b[1] = byte(v)
}

func be32(b []byte, v uint32) {
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
}

func be64(b []byte, v uint64) {
	be32(b, uint32(v>>32))
	be32(b[4:], uint32(v))
}

func u16(b []byte) uint16 { return uint16(b[0])<<8 | uint16(b[1]) }

func u32(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

func u64(b []byte) uint64 { return uint64(u32(b))<<32 | uint64(u32(b[4:])) }

// i16 reads a signed short. Every count in a house is one of these -- room numbers, floors,
// suites, gliders left -- because the original's are `short` and §10.4.10's second rule is
// that a width never changes in translation.
func i16(b []byte) int16 { return int16(u16(b)) }

func i32(b []byte) int32 { return int32(u32(b)) }
