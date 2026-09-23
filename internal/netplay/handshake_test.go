package netplay

import (
	"errors"
	"hash/fnv"
	"reflect"
	"strings"
	"testing"
	"time"
)

// The two houses every handshake test uses: same name, and hashes that are equal or not
// depending on what is being tested. A hash is 8 bytes off a house here rather than a real one
// because HouseHash's own test is the one that cares what goes into it.
const (
	houseName  = "Open House"
	houseHashA = uint64(0x0123456789ABCDEF)
	houseHashB = uint64(0xFEDCBA9876543210)
)

// The engine and release every handshake here claims unless it is testing them.
const (
	engineA  = uint64(0x1111222233334444)
	engineB  = uint64(0x5555666677778888)
	releaseA = "0.3.0"
)

func hello(nonce uint64, hash uint64) Hello {
	return Hello{Nonce: nonce, Versions: 1 << Version, HouseName: houseName, HouseHash: hash,
		Neighbors: 9, Engine: engineA, Release: releaseA}
}

// meet runs both halves of the handshake at once and returns what each side got. Both sides
// run concurrently because that is the only way it happens -- Meet's first act is a send, and
// a test that ran them one after the other would be testing a handshake nobody performs.
func meet(t *testing.T, a, b *end, ha, hb Hello) (Match, error, Match, error) {
	t.Helper()
	type result struct {
		m   Match
		err error
	}
	run := func(c *Conn, h Hello) <-chan result {
		ch := make(chan result, 1)
		go func() {
			m, err := Meet(c, h)
			ch <- result{m, err}
		}()
		return ch
	}
	ca, cb := run(a.Conn, ha), run(b.Conn, hb)
	get := func(who string, ch <-chan result) result {
		select {
		case r := <-ch:
			return r
		case <-time.After(10 * time.Second):
			// A handshake that hangs is a worse failure than one that fails, and a test
			// that hangs is worse still: it takes the whole package's timeout with it and
			// says nothing about which side stopped.
			t.Fatalf("%s's Meet never returned", who)
			return result{}
		}
	}
	ra := get("the first peer", ca)
	rb := get("the second peer", cb)
	return ra.m, ra.err, rb.m, rb.err
}

func TestMeetAgreesOnEverything(t *testing.T) {
	a, b := pair(t)
	ha, hb := hello(7, houseHashA), hello(9, houseHashA)
	hb.InputDelay = 3 // §10.4.7: the agreed delay is the larger of the two proposals
	hb.Neighbors = 1
	ma, ea, mb, eb := meet(t, a, b, ha, hb)
	if ea != nil || eb != nil {
		t.Fatalf("Meet failed: first %v, second %v", ea, eb)
	}

	// The two sides must agree about the match and disagree about nothing except which of
	// them they are. That is the whole property a handshake with no authority has to have.
	if ma.Slot != 0 || mb.Slot != 1 {
		t.Errorf("slots are %d and %d; nonce 7 is smaller, so it is player 1 (§10.4.7)",
			ma.Slot, mb.Slot)
	}
	if ma.Seed != mb.Seed {
		t.Errorf("seeds differ: %016X and %016X", ma.Seed, mb.Seed)
	}
	if ma.ID != mb.ID || ma.ID != uint32(ma.Seed) {
		t.Errorf("matchIDs are %08X and %08X, and the seed's low word is %08X",
			ma.ID, mb.ID, uint32(ma.Seed))
	}
	if ma.RandSeed() != mb.RandSeed() {
		t.Errorf("the two worlds would be seeded differently: %d and %d",
			ma.RandSeed(), mb.RandSeed())
	}
	if ma.InputDelay != 3 || mb.InputDelay != 3 {
		t.Errorf("input delays are %d and %d, want 3 on both (the larger proposal)",
			ma.InputDelay, mb.InputDelay)
	}
	if ma.StartFrame != 0 || mb.StartFrame != 0 {
		t.Errorf("start frames are %d and %d, want 0", ma.StartFrame, mb.StartFrame)
	}
	if ma.PeerHouseName != houseName || mb.PeerHouseName != houseName {
		t.Errorf("peer house names are %q and %q", ma.PeerHouseName, mb.PeerHouseName)
	}
	// §10.3.5 R3's informational field, and the one thing the two sides are allowed to
	// disagree about after a successful handshake.
	if ma.PeerNeighbors != 1 || mb.PeerNeighbors != 9 {
		t.Errorf("neighbour views came back as %d and %d, want 1 and 9",
			ma.PeerNeighbors, mb.PeerNeighbors)
	}
	// The seed is the mix of the two nonces in value order, whichever arrived first.
	// Checking the value and not just the agreement is what makes that a property rather
	// than a coincidence.
	if want := mixSeed(7, 9); ma.Seed != want {
		t.Errorf("seed is %016X, want mixSeed(7, 9) = %016X", ma.Seed, want)
	}
}

func TestMeetGivesPlayerOneToTheSmallerNonce(t *testing.T) {
	// Both orders, because the rule has to be symmetric: the side that happens to call Meet
	// first must not be the side that gets to be player 1.
	for _, c := range []struct{ first, second uint64 }{{7, 9}, {9, 7}} {
		a, b := pair(t)
		ma, ea, mb, eb := meet(t, a, b, hello(c.first, houseHashA), hello(c.second, houseHashA))
		if ea != nil || eb != nil {
			t.Fatalf("nonces %d/%d: %v, %v", c.first, c.second, ea, eb)
		}
		wantA := uint8(0)
		if c.first > c.second {
			wantA = 1
		}
		if ma.Slot != wantA || mb.Slot == ma.Slot {
			t.Errorf("nonces %d/%d gave slots %d and %d", c.first, c.second, ma.Slot, mb.Slot)
		}
	}
}

func TestMeetRefusesTwoDifferentHouses(t *testing.T) {
	// docs/PLAN.md Stage 3's third acceptance clause, and §10.4.7's mandatory gate. Both
	// sides have to refuse, and both have to say enough for a player to act on it: the names
	// are usually identical -- two builds of one house is the common case, not two different
	// houses -- so the hashes are the only thing that distinguishes them on screen.
	a, b := pair(t)
	ha := hello(7, houseHashA)
	hb := hello(9, houseHashB)
	hb.HouseName = "Boarding House"
	_, ea, _, eb := meet(t, a, b, ha, hb)
	for who, err := range map[string]error{"the first peer": ea, "the second peer": eb} {
		if !errors.Is(err, ErrHouse) {
			t.Errorf("%s: err = %v, want %v", who, err, ErrHouse)
			continue
		}
		for _, want := range []string{houseName, "Boarding House", "0123456789ABCDEF",
			"FEDCBA9876543210"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("%s: the message does not mention %q: %v", who, want, err)
			}
		}
	}
}

// refusedByBoth runs a handshake that must fail and checks that it fails on **both** sides with
// want, naming everything in says. Both, because a refusal only one side makes lets the other
// return from Meet and fly: whichever of them is player 1 sends MatchStart and returns, and the
// other hanging up reaches it as a forfeit it wins.
func refusedByBoth(t *testing.T, ha, hb Hello, want error, says ...string) {
	t.Helper()
	a, b := pair(t)
	_, ea, _, eb := meet(t, a, b, ha, hb)
	for _, side := range []struct {
		who string
		err error
	}{{"the first peer", ea}, {"the second peer", eb}} {
		if !errors.Is(side.err, want) {
			t.Errorf("%s: err = %v, want %v", side.who, side.err, want)
			continue
		}
		for _, w := range says {
			if !strings.Contains(side.err.Error(), w) {
				t.Errorf("%s: the message does not mention %q: %v", side.who, w, side.err)
			}
		}
	}
}

// Two builds whose simulations differ are refused, and told which releases they are: the
// fingerprints are for a bug report, the releases are what a player can install.
func TestMeetRefusesTwoDifferentEngines(t *testing.T) {
	ha, hb := hello(7, houseHashA), hello(9, houseHashA)
	hb.Engine, hb.Release = engineB, "0.4.0"
	refusedByBoth(t, ha, hb, ErrEngine,
		`release "0.3.0"`, `release "0.4.0"`, "1111222233334444", "5555666677778888")

	// Two builds calling themselves the same release are not two releases, and are told
	// what they are instead: one of them is not the build its name says.
	hb.Release = releaseA
	refusedByBoth(t, ha, hb, ErrEngine, `release "0.3.0"`, "changed source")

	// The engine is decided before the house. Two builds that fly differently cannot race
	// whatever house they open, so telling them about the house first sends them to fix
	// the wrong thing.
	hb.HouseHash = houseHashB
	refusedByBoth(t, ha, hb, ErrEngine)
}

// Two releases with no protocol version in common are refused by name. Today that is only a
// peer from the future -- this build speaks one version -- and the refusal is the whole point of
// the rule that a Hello is always sent under header version 1.
func TestMeetRefusesTwoReleasesWithNoVersionInCommon(t *testing.T) {
	ha, hb := hello(7, houseHashA), hello(9, houseHashA)
	hb.Versions, hb.Release = 1<<2|1<<3, "2.0.0"
	refusedByBoth(t, ha, hb, ErrVersion, `release "0.3.0"`, `release "2.0.0"`,
		"version 1", "versions 2 and 3")

	// A peer that sets no bit at all speaks nothing, and is not treated as speaking
	// everything. Hand-rolled, because Meet fills in an empty bitmap on its own side.
	hb.Versions = 0
	a, b := pair(t)
	if err := a.Send(EncodeHello(hb)); err != nil {
		t.Fatal(err)
	}
	if _, err := Meet(b.Conn, ha); !errors.Is(err, ErrVersion) ||
		!strings.Contains(err.Error(), "no version at all") {
		t.Errorf("a peer offering no version: err = %v, want %v saying so", err, ErrVersion)
	}
}

// A difference in a rule that decides a race refuses it -- including a rule this build has
// never heard of, which is the reason the gate is by position.
func TestMeetRefusesADifferenceInARuleThatDecidesTheRace(t *testing.T) {
	ha, hb := hello(7, houseHashA), hello(9, houseHashA)
	hb.Rules, hb.Release = RuleAssisted, "0.4.0"
	refusedByBoth(t, ha, hb, ErrRules, `release "0.3.0"`, `release "0.4.0"`, "an assist",
		"none of them")

	// Bit 9 has no name in this build. It is refused all the same, by its number.
	hb.Rules = 1 << 9
	refusedByBoth(t, ha, hb, ErrRules, "rule 9")

	// A rule in common is no reason for anything, and is not named: only the difference
	// is what somebody has to change.
	ha.Rules, hb.Rules = RuleAssisted, RuleAssisted|1<<9
	a, b := pair(t)
	_, ea, _, _ := meet(t, a, b, ha, hb)
	if !errors.Is(ea, ErrRules) || strings.Contains(ea.Error(), "assist") {
		t.Errorf("a gated rule both sides play with is in the refusal: %v", ea)
	}
}

// The rules that do not decide a race let it go ahead, and each side is told the other's, with
// its release.
func TestMeetLetsThroughARuleThatOnlyChangesWhatIsSeen(t *testing.T) {
	ha, hb := hello(7, houseHashA), hello(9, houseHashA)
	ha.Rules = RuleMirrorFlame | RuleSwitchSparkle
	hb.Rules, hb.Release = RulePlayer2GiveUp|1<<6, "0.4.0" // bit 6: a later build's, and informational
	a, b := pair(t)
	ma, ea, mb, eb := meet(t, a, b, ha, hb)
	if ea != nil || eb != nil {
		t.Fatalf("Meet refused rules that do not decide a race: %v, %v", ea, eb)
	}
	if ma.PeerRules != hb.Rules || mb.PeerRules != ha.Rules {
		t.Errorf("each side was told the other plays with %v and %v, want %v and %v",
			ma.PeerRules, mb.PeerRules, hb.Rules, ha.Rules)
	}
	if ma.PeerRelease != "0.4.0" || mb.PeerRelease != releaseA {
		t.Errorf("each side was told the other is %q and %q", ma.PeerRelease, mb.PeerRelease)
	}
}

func TestRulesSayWhatTheyAre(t *testing.T) {
	for _, c := range []struct {
		r    Rules
		want string
	}{
		{0, "none"},
		{RuleMirrorFlame, "fixes.mirror_flame"},
		{RuleMirrorFoil | RulePlayer2GiveUp, "fixes.mirror_foil, fixes.player2_give_up"},
		{RuleSwitchSparkle | 1<<5 | RuleAssisted, "fixes.switch_sparkle, rule 5, an assist"},
	} {
		if got := c.r.String(); got != c.want {
			t.Errorf("Rules(0x%04X) = %q, want %q", uint16(c.r), got, c.want)
		}
	}
	// Every named rule is on the side of the split its comment puts it: the fixes change
	// what is seen, and an assist changes what the glider can do.
	for r, name := range ruleNames {
		if gated := r&RulesGated != 0; gated != (r == RuleAssisted) {
			t.Errorf("%s is on the wrong side of RulesGated", name)
		}
	}
}

// Every message this build speaks reads a longer one from a later build as the message it
// knows, and ignores the rest. That is the rule that lets a field be added at the end, and it is
// a rule about every decoder, because any of them refusing a longer message would make that
// message's layout frozen for good.
func TestEveryMessageAcceptsTrailingBytes(t *testing.T) {
	h := hello(7, houseHashA)
	h.Rules = RuleMirrorFlame
	start := MatchStart{InputDelay: 2, Seed: 0x1234_5678_9ABC_DEF0, HouseHash: houseHashA,
		StartFrame: 3}
	standing := Standing{Room: 3, Rooms: 5, Score: 1200, Frame: 777, State: Finished}
	for _, c := range []struct {
		name string
		msg  []byte
		want Msg
	}{
		{"MsgHello", EncodeHello(h), &h},
		{"MsgMatchStart", EncodeMatchStart(start), &start},
		{"MsgStanding", EncodeStanding(9, 1, standing), nil},
		{"MsgBye", EncodeBye(9, 1), nil},
	} {
		plain, err := decode(c.msg[3], c.msg)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		longer := append(append([]byte(nil), c.msg...), 0xDE, 0xAD, 0xBE, 0xEF)
		got, err := decode(longer[3], longer)
		if err != nil {
			t.Errorf("%s with four bytes more: %v", c.name, err)
			continue
		}
		if !reflect.DeepEqual(got, plain) {
			t.Errorf("%s with four bytes more reads as %+v, want %+v", c.name, got, plain)
		}
		if c.want != nil && !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s reads as %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestMeetRefusesEqualNonces(t *testing.T) {
	// §10.4.7 says ties are impossible in practice and that both peers re-nonce if one
	// happens. They cannot be resolved silently: with equal nonces the slot rule has no
	// answer, and a rule with no answer that returns one anyway would give both peers the
	// same slot. Nonce's own comment has the case where this stops being hypothetical -- a
	// nonce drawn from the game's deterministic RNG is the same number on both machines.
	a, b := pair(t)
	_, ea, _, eb := meet(t, a, b, hello(42, houseHashA), hello(42, houseHashA))
	if !errors.Is(ea, ErrProtocol) || !errors.Is(eb, ErrProtocol) {
		t.Errorf("equal nonces: errors are %v and %v, want protocol errors", ea, eb)
	}
}

func TestMeetChecksEveryFieldPlayerOneHadNoFreedomAbout(t *testing.T) {
	// The local side draws the larger nonce, so it is player 2 and its Meet waits for a
	// MsgMatchStart it can check. The peer is hand-rolled, which is the only way to send one
	// that is wrong -- EncodeMatchStart from a correct Match cannot be.
	const localNonce, peerNonce = 9, 7
	seed := mixSeed(peerNonce, localNonce)
	good := MatchStart{Slot: 0, InputDelay: 0, Seed: seed, HouseHash: houseHashA}

	forge := func(f func(*MatchStart)) MatchStart {
		m := good
		f(&m)
		return m
	}
	for _, c := range []struct {
		name  string
		start MatchStart
		want  error
	}{{
		name: "it claims to be player 2",
		// The peer sent the smaller nonce, so both sides computed the same rule and it
		// is player 1. A 1 here is a contradiction, not a negotiation.
		start: forge(func(m *MatchStart) { m.Slot = 1 }),
		want:  ErrProtocol,
	}, {
		name: "it invented a seed",
		// The low 32 bits are left alone on purpose: matchID is the low word of the seed,
		// so flipping a high bit gets the message past Recv's envelope check and proves
		// the seed comparison is doing work of its own.
		start: forge(func(m *MatchStart) { m.Seed ^= 1 << 40 }),
		want:  ErrProtocol,
	}, {
		name:  "it echoed the wrong house",
		start: forge(func(m *MatchStart) { m.HouseHash = houseHashB }),
		want:  ErrHouse,
	}, {
		name:  "it agreed a delay neither side proposed",
		start: forge(func(m *MatchStart) { m.InputDelay = 4 }),
		want:  ErrProtocol,
	}, {
		name:  "nothing is wrong with it",
		start: good,
		want:  nil,
	}} {
		a, b := pair(t)
		if err := a.Send(EncodeHello(hello(peerNonce, houseHashA))); err != nil {
			t.Fatalf("%s: sending the fake hello: %v", c.name, err)
		}
		if err := a.Send(EncodeMatchStart(c.start)); err != nil {
			t.Fatalf("%s: sending the fake match start: %v", c.name, err)
		}
		m, err := Meet(b.Conn, hello(localNonce, houseHashA))
		switch {
		case c.want == nil && err != nil:
			t.Errorf("%s: Meet failed: %v", c.name, err)
		case c.want == nil && m.Slot != 1:
			t.Errorf("%s: slot = %d, want 1", c.name, m.Slot)
		case c.want != nil && !errors.Is(err, c.want):
			t.Errorf("%s: err = %v, want %v", c.name, err, c.want)
		}
	}
}

func TestMeetRefusesAnythingButAHelloFirst(t *testing.T) {
	for _, c := range []struct {
		name string
		msg  []byte
	}{
		{"a standing", EncodeStanding(0, 0, Standing{})},
		{"a goodbye", EncodeBye(0, 0)},
	} {
		a, b := pair(t)
		if err := a.Send(c.msg); err != nil {
			t.Fatalf("%s: Send: %v", c.name, err)
		}
		_, err := Meet(b.Conn, hello(1, houseHashA))
		if !errors.Is(err, ErrProtocol) {
			t.Errorf("%s before the hello: err = %v, want a protocol error", c.name, err)
		}
	}
}

func TestMeetSaysSoWhenTheOtherSideNeverArrives(t *testing.T) {
	// Somebody typed the wrong address, or closed the window while waiting. Ordinary enough
	// that the message has to be a sentence a screen can show.
	a, b := pair(t)
	a.kill()
	_, err := Meet(b.Conn, hello(1, houseHashA))
	if err == nil || !strings.Contains(err.Error(), "left while waiting") {
		t.Errorf("err = %v, want something about the other side leaving", err)
	}
}

func TestHelloRoundTripsIncludingItsNameLength(t *testing.T) {
	long := strings.Repeat("h", 255)
	for _, in := range []Hello{
		{},
		{Slot: 1, Nonce: ^uint64(0), Versions: 0xFFFF, HouseName: houseName,
			HouseHash: houseHashA, InputDelay: 255, Neighbors: 9, Engine: ^uint64(0),
			Rules: 0xFFFF, Release: "0.3.0-rc.1"},
		{HouseName: ""},
		{HouseName: long},
		{Release: long},
		{HouseName: long, Release: long, Engine: engineA, Rules: RulesGated},
	} {
		msg, err := decodeHello(EncodeHello(in))
		if err != nil {
			t.Fatalf("%q: decode: %v", in.HouseName, err)
		}
		if got := *msg.(*Hello); got != in {
			t.Errorf("round trip changed it:\n got %+v\nwant %+v", got, in)
		}
	}

	// A name longer than the length byte can describe is truncated rather than refused. The
	// house name is a label -- the hash is the gate -- and a house whose name is 300 bytes is
	// a house nobody should be unable to race over.
	// The same for a release, which is a label for the same reason: the engine is the gate.
	over := Hello{HouseName: long + "!", Release: long + "?"}
	msg, err := decodeHello(EncodeHello(over))
	if err != nil {
		t.Fatalf("256-byte name and release: %v", err)
	}
	if got := msg.(*Hello); got.HouseName != long || got.Release != long {
		t.Errorf("a 256-byte name and release came back as %d and %d bytes, want 255 and 255",
			len(got.HouseName), len(got.Release))
	}
}

func TestDecodeHelloRefusesNonsense(t *testing.T) {
	good := EncodeHello(hello(1, houseHashA))

	if _, err := decodeHello(good[:helloBeforeName-1]); !errors.Is(err, ErrShort) {
		t.Errorf("truncated before the name: err = %v, want %v", err, ErrShort)
	}
	// Truncated *after* the name length, which is the case a fixed-size check would miss:
	// the message is long enough to read houseNameLen and not long enough to hold what it
	// promises. §10.4.10's "explicit count before every variable-length array" is only worth
	// anything if the count is checked against what followed it.
	if _, err := decodeHello(good[:len(good)-len(releaseA)-1]); !errors.Is(err, ErrShort) {
		t.Errorf("truncated after the name: err = %v, want %v", err, ErrShort)
	}
	// And after the release's length, for the same reason and the second variable-length
	// field.
	if _, err := decodeHello(good[:len(good)-1]); !errors.Is(err, ErrShort) {
		t.Errorf("truncated inside the release: err = %v, want %v", err, ErrShort)
	}

	withMatch := append([]byte(nil), good...)
	be32(withMatch[4:], 1)
	if _, err := decodeHello(withMatch); !errors.Is(err, ErrProtocol) {
		t.Errorf("a hello claiming a match: err = %v, want a protocol error", err)
	}

	badSlot := append([]byte(nil), good...)
	badSlot[8] = 2
	if _, err := decodeHello(badSlot); !errors.Is(err, ErrProtocol) {
		t.Errorf("a hello proposing slot 2: err = %v, want a protocol error", err)
	}
}

func TestMeetFillsInTheVersionBitmap(t *testing.T) {
	// A caller that says nothing about versions gets the one this build speaks, with the bit
	// numbering Hello.Versions fixes: bit n for version n, so bit 0 stays clear because there
	// is no version 0. Read off the wire rather than out of the struct, because Meet defaults
	// the field on its own copy and a test on the caller's copy would pass either way.
	a, b := pair(t)
	go Meet(a.Conn, hello(1, houseHashA)) //nolint:errcheck // the hello is the subject
	msg, err := b.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if got := msg.(*Hello).Versions; got != 1<<Version {
		t.Errorf("Versions = 0x%04X, want 0x%04X (bit %d set)", got, 1<<Version, Version)
	}
	if 1<<Version&1 != 0 {
		t.Error("bit 0 is set, and there is no version 0")
	}
}

func TestHouseHashIsFNV1a(t *testing.T) {
	// Pinned values, and pinned against an implementation that is not Go's: these three were
	// computed with a five-line FNV-1a in Python, which also reproduces the published test
	// vector FNV-1a("a") = 0xAF63DC4C8601EC8C. That is what makes this a check on the choice
	// of algorithm rather than a check on hash/fnv against itself -- and the algorithm is a
	// compatibility surface, because §10.4.6 puts it under Version rather than in a field, so
	// two builds that hash houses differently would disagree about every house at the gate
	// while both claiming version 1.
	for _, c := range []struct {
		in   string
		want uint64
	}{
		{"", 0xCBF29CE484222325}, // the offset basis, unchanged by an empty house
		{"Glider PRO", 0x1BD11D8ACE6C6B6B},
	} {
		if got := HouseHash([]byte(c.in)); got != c.want {
			t.Errorf("HouseHash(%q) = %016X, want %016X", c.in, got, c.want)
		}
	}
	if got := mixSeed(1, 2); got != 0xF4B5C85BCC646AEC {
		t.Errorf("mixSeed(1, 2) = %016X, want F4B5C85BCC646AEC", got)
	}

	// One flipped byte has to change it. Trivially true of any hash and worth stating: the
	// gate exists because §10.4.7's blower directions and appliance delays are single bytes
	// in a house file, and two houses that differ in one of them are two different games.
	house := []byte("a house, thousands of bytes of blowers and delays")
	flipped := append([]byte(nil), house...)
	flipped[7] ^= 1
	if HouseHash(house) == HouseHash(flipped) {
		t.Error("two houses differing in one bit hash the same")
	}

	// And the algorithm is fed the bytes in order, not as a set: a hash that summed its input
	// would pass everything above.
	if HouseHash([]byte("ab")) == HouseHash([]byte("ba")) {
		t.Error("the hash does not depend on the order of its input")
	}
	h := fnv.New64a()
	h.Write([]byte("ab"))
	if HouseHash([]byte("ab")) != h.Sum64() {
		t.Error("HouseHash is not hash/fnv's 64a")
	}
}

func TestRandSeedStaysInsideTheGenerator(t *testing.T) {
	// internal/game's Lehmer generator has two fixed points, 0 and 0x7FFFFFFF, and
	// World.Random nudges both to 1. RandSeed does the nudge here instead so that the number
	// the two peers agreed on is the number their generators run from; the two seeds below
	// are the ones that reach each fixed point.
	for _, c := range []struct{ seed uint64 }{
		{0},
		{0x7FFFFFFF},            // folds to the upper fixed point
		{0xFFFFFFFF_FFFFFFFF},   // folds to 0
		{0x1234_5678_9ABC_DEF0}, // an ordinary one
		{0x8000_0000_0000_0000}, // only the top bit, which the fold moves down
		{0x0000_0000_8000_0000}, // only the bit the mask drops
	} {
		got := (Match{Seed: c.seed}).RandSeed()
		if got <= 0 || got >= 0x7FFFFFFF {
			t.Errorf("seed %016X gives RandSeed %d, which is outside (0, 0x7FFFFFFF)",
				c.seed, got)
		}
	}
	// Different seeds should mostly give different starting states -- a fold that collapsed
	// the space would make every match's house behave the same way.
	seen := map[int32]bool{}
	for i := 0; i < 1000; i++ {
		seen[(Match{Seed: mixSeed(uint64(i), uint64(i*7+1))}).RandSeed()] = true
	}
	if len(seen) < 990 {
		t.Errorf("1000 match seeds produced only %d distinct generator seeds", len(seen))
	}
}

func TestNonceIsNotAConstant(t *testing.T) {
	// The weakest useful check on the strongest requirement in the package: two peers that
	// draw the same nonce cannot start a match at all (TestMeetRefusesEqualNonces), so a
	// Nonce that returned a fixed value -- or 0 -- would break every race and no other test.
	seen := map[uint64]bool{}
	for i := 0; i < 64; i++ {
		n, err := Nonce()
		if err != nil {
			t.Fatalf("Nonce: %v", err)
		}
		if seen[n] {
			t.Fatalf("Nonce returned %016X twice in %d draws", n, i+1)
		}
		seen[n] = true
	}
}
