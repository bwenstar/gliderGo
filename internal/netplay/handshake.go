package netplay

import (
	"crypto/rand"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
)

// Hello is MsgHello (§10.4.7): what each peer says about itself before there is a match.
//
// Both sides send one, unprompted, as their first message. There is no client and no server
// here -- §10.4.7's slot rule is symmetric, and §10.2.8 rejected an authoritative server for
// the lock-step case for reasons that apply at least as strongly to a race, where there is
// nothing for an authority to be authoritative about.
type Hello struct {
	// Slot is §10.4.7's "proposed" senderSlot. Meet does not honour it: the nonce rule
	// decides, symmetrically, and a proposal it contradicts is not an error because the
	// specification calls the field a proposal. It is decoded and exposed all the same, so
	// that a peer that wanted to be player 2 can be told why it is not.
	Slot uint8

	// Nonce breaks the symmetry: the smaller of the two becomes player 1. Use Nonce() to
	// get one, and read its comment first -- the obvious wrong source for this number is
	// the game's own RNG, which starts from seed 1 on both machines.
	Nonce uint64

	// Versions is §10.4.7's protocolVersions bitmap: bit n set means version n is
	// understood. §10.4 does not say which end bit 0 is, so this package fixes it -- bit n
	// for version n, leaving bit 0 permanently clear because there is no version 0.
	//
	// It does nothing today, and the reason is worth being straight about: parseHeader
	// rejects a mismatched version byte before this field is ever looked at, so two builds
	// that could both speak version 1 but announce something else in the header never get
	// here. That is the right order while there is one version. When there are two, this is
	// the field that lets them meet, and the check moves.
	Versions uint16

	// HouseName is the house this peer has loaded, named the way the picker names it: by
	// its file, because houseType has no name field (docs/analysis/original-houses.md
	// §2.1). At most 255 bytes, which is the length byte's limit and also the original's --
	// the prefs file stores a Str255 (Main.c:232's thePrefs).
	HouseName string

	// HouseHash is the gate. See HouseHash for what goes into it.
	HouseHash uint64

	// InputDelay is §10.4.7's proposed kInputDelay, agreed as the larger of the two. A race
	// has no use for it and sends 0; see the package comment.
	InputDelay uint8

	// Neighbors is prefs.Neighbors -- 1, 3 or 9 rooms composed around the player. Purely
	// informational per §10.3.5 R3, and worth showing: a peer rendering 9 sees hazards a
	// peer rendering 1 does not.
	Neighbors uint8
}

func (*Hello) msgType() uint8 { return MsgHello }

// The fixed parts of MsgHello, either side of the variable-length house name. §10.4.7's table
// is the authority for every offset; these two constants exist so that the encoder and the
// decoder cannot disagree about where the name starts and ends.
const (
	helloBeforeName = 21 // header, senderSlot, pad, nonce, protocolVersions, houseNameLen
	helloAfterName  = 10 // houseHash, kInputDelay, numNeighborsView
)

// EncodeHello lays out a MsgHello. matchID is 0, which §10.4.7 requires: there is no match yet,
// and a Hello that claimed one would be a Hello from a peer that had skipped this step.
func EncodeHello(h Hello) []byte {
	name := h.HouseName
	if len(name) > 255 {
		name = name[:255]
	}
	b := make([]byte, helloBeforeName+len(name)+helloAfterName)
	header(b, MsgHello, 0)
	b[8] = h.Slot
	b[9] = 0 // §10.4.7's pad, and §10.4.10's third rule: written, not left over
	be64(b[10:], h.Nonce)
	be16(b[18:], h.Versions)
	b[20] = byte(len(name))
	copy(b[21:], name)
	at := 21 + len(name)
	be64(b[at:], h.HouseHash)
	b[at+8] = h.InputDelay
	b[at+9] = h.Neighbors
	return b
}

func decodeHello(b []byte) (Msg, error) {
	if len(b) < helloBeforeName {
		return nil, short("MsgHello up to its house name", len(b), helloBeforeName)
	}
	if id := u32(b[4:]); id != 0 {
		return nil, fmt.Errorf("%w: MsgHello carries matchID 0x%08X and must carry 0",
			ErrProtocol, id)
	}
	n := int(b[20])
	need := helloBeforeName + n + helloAfterName
	if len(b) < need {
		return nil, short(fmt.Sprintf("MsgHello with a %d-byte house name", n), len(b), need)
	}
	at := helloBeforeName + n
	h := &Hello{
		Slot:     b[8],
		Nonce:    u64(b[10:]),
		Versions: u16(b[18:]),
		// A copy, not a sub-slice, and the field is a string for exactly that reason:
		// Conn.frame hands out its own read buffer and reuses it on the next message, so a
		// []byte field here would be a house name that changed under its holder.
		HouseName:  string(b[21 : 21+n]),
		HouseHash:  u64(b[at:]),
		InputDelay: b[at+8],
		Neighbors:  b[at+9],
	}
	if h.Slot > 1 {
		return nil, fmt.Errorf("%w: MsgHello proposes slot %d, must be 0 or 1",
			ErrProtocol, h.Slot)
	}
	return h, nil
}

// MatchStart is MsgMatchStart (§10.4.7): player 1 telling player 2 what they have both already
// worked out.
//
// Every field in it is computable by the receiver from the two Hellos, and it is sent anyway.
// Not for belt and braces: it is the message §10.4.7 defines, a later lock-step mode will have
// to choose kInputDelay and matchSeed in ways a race does not, and a format that changes shape
// between modes is worse than one with a field a mode can predict. What the race gets out of it
// is that every field arrives checkable, and Meet checks all of them -- a peer that sends a
// seed it made up is caught here rather than after somebody has lost a race to a house that
// behaved differently on the two screens.
type MatchStart struct {
	// Slot is §10.4.7's slotAssignment: 0 means the sender is player 1. In this
	// implementation it is always 0, because the peer that sends this message is player 1
	// by construction. A 1 is therefore a peer telling us it computed the nonce rule the
	// other way round, which is a contradiction and not a negotiation.
	Slot       uint8
	InputDelay uint8
	Seed       uint64
	HouseHash  uint64
	StartFrame uint32
}

func (*MatchStart) msgType() uint8 { return MsgMatchStart }

// matchStartSize is §10.4.7's layout: 8 header + slotAssignment + kInputDelay + 8 seed +
// 8 houseHash + 4 startFrame.
const matchStartSize = 30

// EncodeMatchStart lays out a MsgMatchStart. matchID is the low 32 bits of the seed, which is
// §10.4.4's definition of the field and the reason Meet can set it before the message arrives.
func EncodeMatchStart(m MatchStart) []byte {
	b := make([]byte, matchStartSize)
	header(b, MsgMatchStart, uint32(m.Seed))
	b[8] = m.Slot
	b[9] = m.InputDelay
	be64(b[10:], m.Seed)
	be64(b[18:], m.HouseHash)
	be32(b[26:], m.StartFrame)
	return b
}

func decodeMatchStart(b []byte) (Msg, error) {
	if len(b) < matchStartSize {
		return nil, short("MsgMatchStart", len(b), matchStartSize)
	}
	m := &MatchStart{
		Slot:       b[8],
		InputDelay: b[9],
		Seed:       u64(b[10:]),
		HouseHash:  u64(b[18:]),
		StartFrame: u32(b[26:]),
	}
	if m.Slot > 1 {
		return nil, fmt.Errorf("%w: MsgMatchStart assigns slot %d, must be 0 or 1",
			ErrProtocol, m.Slot)
	}
	return m, nil
}

// Match is what Meet returns: everything both peers have agreed, from the point of view of the
// one that called it.
type Match struct {
	// Slot is this peer's: 0 for player 1, 1 for player 2. §10.4.7's table lists six ways
	// the original treats the two differently, and while a race shares no room and so is
	// spared five of them, the number still decides which name a result is filed under.
	Slot uint8

	// ID is the low 32 bits of Seed, and the matchID every message on this connection
	// carries from here on.
	ID uint32

	// Seed is the agreed matchSeed. Hand it to the simulation through RandSeed.
	Seed uint64

	// StartFrame is §10.4.7's, and it is 0. It is in the format so that a lock-step match
	// resumed from a MsgProgress snapshot can begin at the frame it stopped at; a race
	// always starts at the beginning, and -- see Winner -- does not need the two peers to
	// start at the same moment at all.
	StartFrame uint32

	// InputDelay is the agreed kInputDelay, which for a race is 0. Kept so that the value
	// on the wire and the value in the program are the same value.
	InputDelay uint8

	// PeerHouseName and PeerNeighbors are what the other side said about itself. The name
	// is worth keeping even though the hashes matched, because two identical houses can be
	// filed under different names and a scoreboard should say which one was raced.
	PeerHouseName string
	PeerNeighbors uint8
}

// RandSeed is the seed both peers hand to their own World.RandSeed.
//
// Two reductions happen here and both are forced. The generator is a 31-bit Lehmer sequence
// (internal/game/rand.go), so a 64-bit seed has to fold; and 0 and 0x7FFFFFFF are its two fixed
// points, which World.Random nudges to 1 on the way past. Doing the nudge here instead means
// the number the two peers agreed on is the number their generators actually run from -- do it
// only in Random and a match can agree on a seed whose behaviour is 1's without anything
// saying so, which is the sort of thing that is only ever discovered while trying to reproduce
// a race that someone thought was unfair.
func (m Match) RandSeed() int32 {
	v := int32(uint32(m.Seed>>32)^uint32(m.Seed)) & 0x7FFFFFFF
	if v == 0 || v == 0x7FFFFFFF {
		v = 1
	}
	return v
}

// Nonce is a fresh 64-bit number for Hello.Nonce, from crypto/rand.
//
// **Not from the game's RNG, and this is the trap the function exists to close.** World.Random
// is a deterministic Lehmer sequence that both peers start from seed 1 (internal/game/rand.go),
// so two peers drawing their nonce from it would draw the *same* nonce, hit §10.4.7's "ties are
// impossible in practice" case on every single match, and never agree on who is player 1. The
// one number in this protocol that must not be reproducible is the one the whole protocol is
// otherwise built to make reproducible.
//
// crypto/rand rather than a clock for the same reason spelled differently: two machines started
// from the same image on the same LAN have agreed on a surprising number of things, and the
// second-resolution part of a clock is one of them.
func Nonce() (uint64, error) {
	var b [8]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		return 0, fmt.Errorf("netplay: no nonce: %w", err)
	}
	return u64(b[:]), nil
}

// HouseHash is the houseHash both Hellos carry: FNV-1a 64 over a house's canonical binary
// encoding, which is what (*house.House).Save returns.
//
// **Not over the file's bytes**, which is what §10.4.7 says, and the difference is this port's
// rather than a disagreement. A house here has two on-disk forms -- the 1994 binary, vendored
// read-only, and the text form the port's own houses are authored in (docs/PLAN.md §3's three
// `levels` lines) -- and they are the same simulation input. Hashing the canonical encoding
// makes them hash equal, which is correct, and costs nothing in strictness for the 1994 houses:
// internal/house's TestCorpusRoundTrip is Save(Load(b)) == b byte for byte over all 22, so for
// those this *is* §10.4.7's hash of the file bytes.
//
// Two things to know about what it covers:
//
//   - **Hash the house as loaded, never as played.** Room.Visited is written during a game and
//     is 1 in 436 of the 4,070 shipped rooms to begin with, so a house hashed mid-race hashes
//     differently from the same house hashed at the door. That is not a flaw in the hash: the
//     visited flags are what CountRoomsVisited counts, so they are input to the race's own
//     metric and belong inside the gate.
//   - It also covers the house's embedded high-score table, which is *not* simulation input.
//     Excluding it was considered and rejected twice over. A gate that is too strict refuses a
//     race that would have been fair and says exactly which houses differ; a gate that is too
//     loose produces a race whose result is meaningless. And excluding one field means keeping
//     a list of fields to exclude, on which every field added later is on the wrong side by
//     default. This port never writes a board back into a house it did not author
//     (internal/game/highscore.go), so in practice the table is whatever shipped.
//
// FNV-1a needs no field of its own because §10.4.6 puts the choice of hash under Version.
func HouseHash(canonical []byte) uint64 {
	h := fnv.New64a()
	h.Write(canonical)
	return h.Sum64()
}

// mixSeed derives the match seed from the two nonces.
//
// Ordered by value rather than by who sent what, so both peers compute the same number without
// either having to know whose Hello arrived first. Mixed rather than concatenated or XORed
// because neither peer should be able to choose the result: XOR lets the second peer pick the
// seed by picking its nonce, and a race whose house behaviour one side chose is a race the other
// side did not agree to. Whether that matters between two people on one LAN is arguable; the
// version that does not need the argument costs three lines.
func mixSeed(lo, hi uint64) uint64 {
	var b [16]byte
	be64(b[:], lo)
	be64(b[8:], hi)
	h := fnv.New64a()
	h.Write(b[:])
	return h.Sum64()
}

// Meet performs §10.4.7's handshake and returns the agreed match.
//
// Symmetric, and callable identically by both peers: send a Hello, read the other's, work out
// who is player 1 from the nonces, and then -- player 1 sending, player 2 checking -- exchange
// the MatchStart that writes it down. The only asymmetry is which side of that last message
// each peer is on.
//
// **There is no starting gun, and there does not need to be one.** Meet returns as soon as the
// match is agreed, so one peer may be flying while the other is still loading a house. That
// costs nothing, because nothing the race measures is measured in wall clock: the metric is
// rooms visited, the tie-breaks are score and then *frames simulated* (Winner), and a peer that
// started late has simply not simulated as many frames yet. Wall clock would have needed a
// countdown, two clocks that agree, and an answer to what happens when one side's frame rate
// drops -- and docs/IMPROVEMENTS.md 4.11 is the standing note about how much a wall clock can
// vary between two runs of this port on one machine, let alone two.
func Meet(c *Conn, local Hello) (Match, error) {
	if local.Versions == 0 {
		local.Versions = 1 << Version
	}
	if err := c.Send(EncodeHello(local)); err != nil {
		return Match{}, err
	}

	msg, err := c.Recv()
	if err != nil {
		return Match{}, meetErr("waiting for the other player's hello", err)
	}
	peer, ok := msg.(*Hello)
	if !ok {
		return Match{}, fmt.Errorf("%w: peer sent %s before the match started, expected %s",
			ErrProtocol, MsgName(msg.msgType()), MsgName(MsgHello))
	}

	if peer.HouseHash != local.HouseHash {
		// The one error a player is meant to see, so it says what to do about it rather
		// than what went wrong. Both names and both hashes: the names are usually the
		// same -- two builds of one house is the common case, not two different houses --
		// and when they are, the hashes are the only thing that distinguishes them.
		return Match{}, fmt.Errorf("%w: yours is %q (hash %016X), theirs is %q (hash %016X); "+
			"both sides need the same house, byte for byte",
			ErrHouse, local.HouseName, local.HouseHash, peer.HouseName, peer.HouseHash)
	}

	// §10.4.7: the numerically smaller nonce becomes player 1. Ties are impossible in
	// practice with 64 bits and the specification says both peers re-nonce if one happens;
	// this returns instead, and lets the caller decide to try again, because Meet has
	// already sent a Hello on this connection and a second one on the same stream would be
	// a second handshake inside the first.
	if peer.Nonce == local.Nonce {
		return Match{}, fmt.Errorf("%w: both sides drew nonce %016X, so neither can be "+
			"player 1; reconnect for a new pair", ErrProtocol, local.Nonce)
	}
	lo, hi := local.Nonce, peer.Nonce
	if hi < lo {
		lo, hi = hi, lo
	}
	m := Match{
		Seed:          mixSeed(lo, hi),
		InputDelay:    max(local.InputDelay, peer.InputDelay),
		PeerHouseName: peer.HouseName,
		PeerNeighbors: peer.Neighbors,
	}
	m.ID = uint32(m.Seed)
	if local.Nonce > peer.Nonce {
		m.Slot = 1
	}

	// Both of these are set before the MatchStart is read, not after, so that Conn's own
	// envelope check does the verifying: player 2 computes the matchID it expects and then
	// any other matchID on the wire is an error in Recv rather than a comparison somebody
	// has to remember to write here.
	c.matchID = m.ID
	c.peerSlot = int8(1 - m.Slot)

	if m.Slot == 0 {
		start := MatchStart{
			Slot:       0,
			InputDelay: m.InputDelay,
			Seed:       m.Seed,
			HouseHash:  local.HouseHash,
			StartFrame: m.StartFrame,
		}
		if err := c.Send(EncodeMatchStart(start)); err != nil {
			return Match{}, err
		}
		return m, nil
	}

	msg, err = c.Recv()
	if err != nil {
		return Match{}, meetErr("waiting for the match to start", err)
	}
	start, ok := msg.(*MatchStart)
	if !ok {
		return Match{}, fmt.Errorf("%w: peer sent %s, expected %s",
			ErrProtocol, MsgName(msg.msgType()), MsgName(MsgMatchStart))
	}
	// Four checks, one per field player 1 had no freedom about. Seed is the load-bearing one
	// -- mixSeed exists so that neither peer chooses the house's behaviour -- and the others
	// are each a way for the two sides to have computed §10.4.7 differently.
	switch {
	case start.Slot != 0:
		return Match{}, fmt.Errorf("%w: player 1 assigned itself slot %d; the nonces say it "+
			"is player 1 and both sides ran the same rule", ErrProtocol, start.Slot+1)
	case start.Seed != m.Seed:
		return Match{}, fmt.Errorf("%w: match seed is %016X, expected %016X from the two "+
			"nonces", ErrProtocol, start.Seed, m.Seed)
	case start.HouseHash != local.HouseHash:
		return Match{}, fmt.Errorf("%w: MsgMatchStart echoes house hash %016X, the hellos "+
			"agreed on %016X", ErrHouse, start.HouseHash, local.HouseHash)
	case start.InputDelay != m.InputDelay:
		return Match{}, fmt.Errorf("%w: agreed input delay is %d, expected %d",
			ErrProtocol, start.InputDelay, m.InputDelay)
	}
	m.StartFrame = start.StartFrame
	return m, nil
}

// meetErr says which half of the handshake was in progress when the connection failed. A
// dropped connection during Meet is ordinary -- somebody mistyped an address, or closed the
// window while waiting -- and "the other player left before the match started" is a sentence a
// screen can show, where "EOF" is not.
func meetErr(doing string, err error) error {
	if errors.Is(err, io.EOF) {
		return fmt.Errorf("netplay: the other side left while %s", doing)
	}
	return fmt.Errorf("netplay: %s: %w", doing, err)
}
