package netplay

import (
	"crypto/rand"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
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
	// Meet refuses two bitmaps with no version in common, naming both releases. That can
	// only work because a Hello is always sent under header version 1, whatever else a build
	// speaks (see the package comment): parseHeader refuses any other version byte before a
	// field is read, so a build that sent its Hello as version 2 would reach a version-1
	// build as a bare ErrVersion that names nobody.
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

	// Engine is the build's engine fingerprint, replay.Engine: a hash of what its
	// simulation does on a fixed set of runs. The second gate after the house, and the one
	// a house hash cannot stand in for -- two builds whose gliders fall at different speeds
	// through the same house are not racing each other, whatever they are holding.
	Engine uint64

	// Rules is which of the game's opt-in changes this peer plays with. See Rules for which
	// of them refuse a race and which are only reported.
	Rules Rules

	// Release is the build's version, as -version prints it: a tag's number, or "dev" for a
	// build nobody tagged. Not a gate -- Engine is, so two releases with the same physics
	// race each other -- and in every refusal all the same, because it is the thing a
	// player can do something about. "Install 0.4.0" is advice; an engine fingerprint is
	// not. At most 255 bytes, for the same length byte as HouseName.
	Release string
}

func (*Hello) msgType() uint8 { return MsgHello }

// The fixed parts of MsgHello, either side of the variable-length house name. §10.4.7's table
// is the authority for every offset up to numNeighborsView; the three fields after it are this
// port's (see the package comment), and determinism-networking.md's amended table lists them.
// These constants exist so that the encoder and the decoder cannot disagree about where the
// name starts and ends.
const (
	helloBeforeName = 21 // header, senderSlot, pad, nonce, protocolVersions, houseNameLen
	helloAfterName  = 21 // houseHash, kInputDelay, numNeighborsView, engine, rules, releaseLen
)

// cut255 is a string cut to what a length byte can count.
func cut255(s string) string {
	if len(s) > 255 {
		return s[:255]
	}
	return s
}

// EncodeHello lays out a MsgHello. matchID is 0, which §10.4.7 requires: there is no match yet,
// and a Hello that claimed one would be a Hello from a peer that had skipped this step.
func EncodeHello(h Hello) []byte {
	name, release := cut255(h.HouseName), cut255(h.Release)
	b := make([]byte, helloBeforeName+len(name)+helloAfterName+len(release))
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
	be64(b[at+10:], h.Engine)
	be16(b[at+18:], uint16(h.Rules))
	b[at+20] = byte(len(release))
	copy(b[at+21:], release)
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
	rn := int(b[at+20])
	if len(b) < need+rn {
		return nil, short(fmt.Sprintf("MsgHello with a %d-byte release", rn), len(b), need+rn)
	}
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
		Engine:     u64(b[at+10:]),
		Rules:      Rules(u16(b[at+18:])),
		Release:    string(b[at+21 : at+21+rn]),
	}
	if h.Slot > 1 {
		return nil, fmt.Errorf("%w: MsgHello proposes slot %d, must be 0 or 1",
			ErrProtocol, h.Slot)
	}
	return h, nil
}

// Rules is Hello's rules field: the game's opt-in changes a peer plays with, one bit each.
//
// **Split by position, so that a build can gate on a change it has never heard of.** A
// difference in the low byte is reported and the race goes ahead; a difference in the high byte
// refuses it. Which byte a new change's bit goes in is decided once, when the change is added,
// by whether it alters what the simulation does -- and every build already released then
// refuses or allows it correctly without knowing its name. A single list of gated changes would
// have needed each old build to know every change made after it.
//
// All four of today's fixes are in the low byte. MirrorFoil and Player2GiveUp act only in a
// two-player game, which a race is not; MirrorFlame changes what is drawn and not what is
// simulated; and SwitchSparkle drops a puff of light that draws no random number and that
// nothing reads (docs/IMPROVEMENTS.md 2.19, 2.20, 2.23 and 2.39). They are sent so that a player
// can be told the other screen looks different, which is all they do.
type Rules uint16

// The rules this build knows by name. A fix's bit is named as the prefs file names the fix,
// because the prefs file is where a player turns it on.
const (
	RuleMirrorFlame   Rules = 1 << 0 // fixes.mirror_flame
	RuleMirrorFoil    Rules = 1 << 1 // fixes.mirror_foil
	RuleSwitchSparkle Rules = 1 << 2 // fixes.switch_sparkle
	RulePlayer2GiveUp Rules = 1 << 3 // fixes.player2_give_up

	// RuleAssisted is reserved for an assist a race allows, and nothing sets it: the game
	// has no assists yet (docs/IMPROVEMENTS.md 3.2), and one added later is refused in a
	// race unless it is given this bit. It is in the high byte because an assist is a change
	// to what the glider can do, which is the definition of a change that decides a race.
	RuleAssisted Rules = 1 << 15

	// RulesGated is the high byte: a difference in any of these bits refuses a race.
	RulesGated Rules = 0xFF00
)

var ruleNames = map[Rules]string{
	RuleMirrorFlame:   "fixes.mirror_flame",
	RuleMirrorFoil:    "fixes.mirror_foil",
	RuleSwitchSparkle: "fixes.switch_sparkle",
	RulePlayer2GiveUp: "fixes.player2_give_up",
	RuleAssisted:      "an assist",
}

// String lists the rules by name, or says "none". A bit this build has no name for is given by
// its number, which is the most an older build can say about a newer one's rule and is still
// enough to look up.
func (r Rules) String() string {
	if r == 0 {
		return "none"
	}
	var names []string
	for bit := 0; bit < 16; bit++ {
		b := Rules(1) << bit
		if r&b == 0 {
			continue
		}
		if name, ok := ruleNames[b]; ok {
			names = append(names, name)
		} else {
			names = append(names, fmt.Sprintf("rule %d", bit))
		}
	}
	return strings.Join(names, ", ")
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

	// PeerRelease and PeerRules are the other side's release and rules. The engines are the
	// same or there would be no match, so the release is here for the record rather than
	// the gate -- a bug report about a race between 0.3.0 and 0.4.0 should say so. The rules
	// can differ in their low byte, and when they do the player should hear about it.
	PeerRelease string
	PeerRules   Rules
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
// either having to know whose Hello arrived first. Mixed rather than concatenated or XORed so
// that choosing the seed takes work: XOR would hand it to whichever peer reads the other's nonce
// before sending its own, for free.
//
// **It does not stop a peer choosing the seed**, and this comment used to say it did. A peer that
// waits for the other's Hello and then tries nonces can set any 16 bits of the seed in about
// fifteen thousand tries, and any of the 31 bits RandSeed keeps in seconds (docs/IMPROVEMENTS.md
// 4.38). What that buys is small -- both players fly the one seed, so a peer that chose it chose
// the house's weather for both -- and closing it needs a commit-reveal round, a second message
// each way before the first standing. That was weighed for the handshake freeze and left out: the
// race is on the honour system in any case, and a peer that wants to cheat can send a winning
// standing with far less effort than a chosen seed.
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

	// **Every refusal from here to the nonce is computed from the two Hellos alone, and so
	// the same on both sides**, and every one of them is decided before player 1 sends
	// MatchStart and returns. That is what they are for. A refusal only one side could see
	// would let player 1 return with a match and fly, and the other side hanging up would
	// reach it as a forfeit and a win nobody earned.
	//
	// The order is the order of what the player can do about it. A protocol they cannot
	// speak and an engine they do not share both mean installing another release, and are
	// said first, because a house or a rule is not worth fixing between two builds that
	// cannot race anyway.
	if local.Versions&peer.Versions == 0 {
		return Match{}, fmt.Errorf("%w: this is %s, which speaks protocol %s, and the other "+
			"side is %s, which speaks %s; both machines need releases with a version in common",
			ErrVersion, release(local.Release), versions(local.Versions),
			release(peer.Release), versions(peer.Versions))
	}
	if peer.Engine != local.Engine {
		if peer.Release == local.Release {
			// The same name on two different engines: at least one of the two is not
			// the build it says it is. Nearly always somebody's own build from changed
			// source, and "dev" and "dev" is the commonest case of it.
			both := "both sides are " + release(local.Release)
			if local.Release == "" {
				both = "neither side names its release"
			}
			return Match{}, fmt.Errorf("%w: %s, and their engines differ (%016X here, %016X "+
				"there), so at least one of them was built from changed source; build both "+
				"from the same source, or install the same release on both",
				ErrEngine, both, local.Engine, peer.Engine)
		}
		return Match{}, fmt.Errorf("%w: this is %s (engine %016X) and the other side is %s "+
			"(engine %016X); the two releases fly differently, so both machines need the same "+
			"one", ErrEngine, release(local.Release), local.Engine,
			release(peer.Release), peer.Engine)
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
	if diff := (local.Rules ^ peer.Rules) & RulesGated; diff != 0 {
		// Only the bits that differ, which is what there is to change. A rule both sides
		// play with is not the reason for anything.
		return Match{}, fmt.Errorf("%w: of the rules that decide a race, this side (%s) plays "+
			"with %s and the other side (%s) with %s; both sides need the same",
			ErrRules, release(local.Release), ruleList(local.Rules&diff),
			release(peer.Release), ruleList(peer.Rules&diff))
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
		PeerRelease:   peer.Release,
		PeerRules:     peer.Rules,
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
	// -- it is the house's behaviour -- and the others are each a way for the two sides to
	// have computed §10.4.7 differently. Unlike the refusals above, these can only fail
	// against a peer that is broken or lying, which is why player 1 not hearing about them
	// is acceptable.
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

// MeetWithin is Meet with a deadline over the whole handshake, and it is how both ends of a race
// meet (docs/IMPROVEMENTS.md 4.32).
//
// Conn keeps no timeout, and a race does not want one (see Conn). The handshake is different,
// because nothing has been agreed with the other end yet, and silence there has one meaning:
// whatever answered is not going to race. A web server waiting for a request line, or a port
// something holds open, would otherwise keep Meet in its first Recv for as long as the socket
// lasts. A host stuck in one of those is a host the real guest cannot reach.
//
// The deadline is cleared once the match is agreed, so it never reaches the race. The
// connection has to be able to keep one: a net.Conn can, and so can net.Pipe.
func MeetWithin(c *Conn, local Hello, wait time.Duration) (Match, error) {
	d, ok := c.rw.(interface{ SetDeadline(time.Time) error })
	if !ok {
		return Match{}, fmt.Errorf("netplay: a %T cannot keep the handshake's deadline", c.rw)
	}
	if err := d.SetDeadline(time.Now().Add(wait)); err != nil {
		return Match{}, fmt.Errorf("netplay: setting the handshake's deadline: %w", err)
	}
	m, err := Meet(c, local)
	switch {
	case err == nil:
	case !errors.Is(err, os.ErrDeadlineExceeded):
		return Match{}, err
	case !c.heard:
		// The sentence rather than the socket's "read tcp a->b: i/o timeout": a player
		// sees this one, and the addresses are already on their screen.
		return Match{}, fmt.Errorf("netplay: the other end said nothing for %v, so it is not "+
			"a gliderGo game ready to race: %w", wait, os.ErrDeadlineExceeded)
	default:
		return Match{}, fmt.Errorf("netplay: the other side stopped answering partway through "+
			"the handshake, and it did not finish within %v: %w", wait, os.ErrDeadlineExceeded)
	}
	if err := d.SetDeadline(time.Time{}); err != nil {
		return Match{}, fmt.Errorf("netplay: clearing the handshake's deadline: %w", err)
	}
	return m, nil
}

// release is a Hello's release as a refusal names it. A peer that sent none is one whose build
// set nothing, and "an unnamed build" is a thing a player can at least ask the other one about.
func release(r string) string {
	if r == "" {
		return "an unnamed build"
	}
	return "release " + strconv.Quote(r)
}

// ruleList is Rules.String for the middle of a sentence, where "none" would read as a rule.
func ruleList(r Rules) string {
	if r == 0 {
		return "none of them"
	}
	return r.String()
}

// versions is a protocolVersions bitmap as a refusal names it: "version 1", "versions 1 and 2",
// or "no version at all" from a peer that set no bit.
func versions(v uint16) string {
	var nums []string
	for n := 1; n < 16; n++ {
		if v&(1<<n) != 0 {
			nums = append(nums, strconv.Itoa(n))
		}
	}
	switch len(nums) {
	case 0:
		return "no version at all"
	case 1:
		return "version " + nums[0]
	}
	return "versions " + strings.Join(nums[:len(nums)-1], ", ") + " and " + nums[len(nums)-1]
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
