package netplay

import (
	"bytes"
	"errors"
	"testing"
)

// The golden MsgStanding. Written out by hand rather than produced by EncodeStanding, because
// this is the test that pins the *layout* and one built by the encoder would only pin the
// encoder against itself. Every value is chosen to catch a real mistake:
//
//   - room -1 is House.c:203's noRoomAtAll, which GetNeighborRoomNumber really returns, so a
//     uint16 read comes back as 65535 and the panel would show a room the house has not got.
//   - floor -3 is a basement. The 1994 corpus spans floors -7..39
//     (docs/analysis/progression.md §5.2), so signedness here is not hypothetical.
//   - score -12345 is not reachable in play and is in the message as an int32 all the same,
//     because §10.4.10's second rule is that a width never changes in translation and
//     World.Score is a long in the C.
var goldenStanding = []byte{
	0x47, 0x4C, // magic
	0x01,                   // version
	0x30,                   // msgType, MsgStanding
	0x01, 0x02, 0x03, 0x04, // matchID
	0x01,       // senderSlot: player 2
	0x01,       // state: Finished
	0x00, 0x29, // rooms: 41
	0xFF, 0xFF, // room: -1
	0xFF, 0xFD, // floor: -3
	0x00, 0x07, // suite: 7
	0x00, 0x02, // mortals: 2
	0xFF, 0xFF, 0xCF, 0xC7, // score: -12345
	0x00, 0x01, 0xE2, 0x40, // frame: 123456
}

var goldenValue = Standing{
	State:   Finished,
	Rooms:   41,
	Room:    -1,
	Floor:   -3,
	Suite:   7,
	Mortals: 2,
	Score:   -12345,
	Frame:   123456,
}

func TestStandingIsTwentyEightBytesInThatOrder(t *testing.T) {
	if len(goldenStanding) != standingSize {
		t.Fatalf("the golden message is %d bytes and standingSize is %d; one of the two is "+
			"wrong and the layout comment says which it should be", len(goldenStanding),
			standingSize)
	}
	got := EncodeStanding(0x01020304, 1, goldenValue)
	if !bytes.Equal(got, goldenStanding) {
		t.Errorf("EncodeStanding:\n got % X\nwant % X", got, goldenStanding)
	}
	msg, err := decodeReport(goldenStanding)
	if err != nil {
		t.Fatalf("decodeReport: %v", err)
	}
	r, ok := msg.(*Report)
	if !ok {
		t.Fatalf("decodeReport returned %T", msg)
	}
	if r.Slot != 1 {
		t.Errorf("slot = %d, want 1", r.Slot)
	}
	if r.Standing != goldenValue {
		t.Errorf("decoded %+v, want %+v", r.Standing, goldenValue)
	}
}

func TestStandingRoundTripsEveryField(t *testing.T) {
	// Extremes rather than plausible values: a int16 that is only ever written and read
	// through this package's own helpers will round-trip whatever it does to the sign bit.
	for _, s := range []Standing{
		{},
		goldenValue,
		{State: Quit, Rooms: 32767, Room: 32767, Floor: 32767, Suite: 32767,
			Mortals: 32767, Score: 2147483647, Frame: 4294967295},
		{State: Died, Rooms: -32768, Room: -32768, Floor: -32768, Suite: -32768,
			Mortals: -32768, Score: -2147483648, Frame: 0},
	} {
		msg, err := decodeReport(EncodeStanding(1, 0, s))
		if err != nil {
			t.Fatalf("%+v: decode: %v", s, err)
		}
		if got := msg.(*Report).Standing; got != s {
			t.Errorf("round trip changed it:\n got %+v\nwant %+v", got, s)
		}
	}
}

func TestDecodeStandingRefusesNonsense(t *testing.T) {
	short := goldenStanding[:standingSize-1]
	if _, err := decodeReport(short); !errors.Is(err, ErrShort) {
		t.Errorf("27 bytes: err = %v, want %v", err, ErrShort)
	}

	badSlot := append([]byte(nil), goldenStanding...)
	badSlot[8] = 2
	if _, err := decodeReport(badSlot); !errors.Is(err, ErrProtocol) {
		t.Errorf("slot 2: err = %v, want a protocol error", err)
	}

	// A fifth state has no meaning, and the only thing Winner could do with one is guess.
	badState := append([]byte(nil), goldenStanding...)
	badState[9] = uint8(Quit) + 1
	if _, err := decodeReport(badState); !errors.Is(err, ErrProtocol) {
		t.Errorf("state 4: err = %v, want a protocol error", err)
	}
	for s := Racing; s <= Quit; s++ {
		ok := append([]byte(nil), goldenStanding...)
		ok[9] = uint8(s)
		if _, err := decodeReport(ok); err != nil {
			t.Errorf("state %d (%s) was refused: %v", s, s, err)
		}
	}
}

func TestStandingsBehindOneAlreadyReceivedAreRefused(t *testing.T) {
	// The ordering rule, and the reason it is a rule rather than a dropped duplicate: Frame
	// is Winner's last tie-break, so a peer that can wind its frame counter back can choose
	// to win a race it drew. Equal frames are allowed -- two reportable things can happen in
	// one frame.
	a, b := pair(t)
	b.matchID = 5
	b.peerSlot = 0
	send := func(frame uint32) {
		if err := a.Send(EncodeStanding(5, 0, Standing{Frame: frame})); err != nil {
			t.Errorf("Send: %v", err)
		}
	}
	send(100)
	send(100)
	send(99)
	for i, want := range []bool{true, true, false} {
		_, err := b.Recv()
		if ok := err == nil; ok != want {
			t.Errorf("standing %d: err = %v, wanted ok = %v", i, err, want)
		}
	}
}

func TestWinner(t *testing.T) {
	// The table is the specification of the result, and every row is a rule from Winner's own
	// comment. Read the wants as "player N wins, because".
	racing := Standing{State: Racing, Rooms: 50, Score: 9999, Frame: 10}
	for _, c := range []struct {
		name   string
		p1, p2 Standing
		want   Outcome
	}{{
		name: "nobody has finished",
		p1:   racing,
		p2:   racing,
		want: Outcome{Slot: -1, Reason: StillRacing},
	}, {
		name: "one is still flying",
		p1:   Standing{State: Died, Rooms: 80},
		p2:   racing,
		want: Outcome{Slot: -1, Reason: StillRacing},
	}, {
		// The acceptance clause: the guest's process is killed, the host is still in the
		// air, and there is a result immediately rather than a wait for a run that will
		// never end.
		name: "the guest was killed mid-race",
		p1:   racing,
		p2:   Standing{State: Quit, Rooms: 80, Score: 99999},
		want: Outcome{Slot: 0, Reason: ByForfeit},
	}, {
		name: "the host quit while the guest flew on",
		p1:   Standing{State: Quit, Rooms: 80},
		p2:   racing,
		want: Outcome{Slot: 1, Reason: ByForfeit},
	}, {
		name: "both walked away",
		p1:   Standing{State: Quit},
		p2:   Standing{State: Quit},
		want: Outcome{Slot: -1, Reason: ByForfeit},
	}, {
		// The departure from a literal reading of docs/PLAN.md Stage 3, and the case that
		// makes it: 20 rooms and out beats 35 rooms and dead, because the house has an end
		// and reaching it is the point.
		name: "finishing the house beats wandering further",
		p1:   Standing{State: Finished, Rooms: 20, Score: 100},
		p2:   Standing{State: Died, Rooms: 35, Score: 50000},
		want: Outcome{Slot: 0, Reason: ByFinish},
	}, {
		name: "between two finishers the metric decides",
		p1:   Standing{State: Finished, Rooms: 20},
		p2:   Standing{State: Finished, Rooms: 21},
		want: Outcome{Slot: 1, Reason: ByRooms},
	}, {
		name: "between two deaths the metric decides",
		p1:   Standing{State: Died, Rooms: 31},
		p2:   Standing{State: Died, Rooms: 30},
		want: Outcome{Slot: 0, Reason: ByRooms},
	}, {
		name: "equal rooms, score breaks it",
		p1:   Standing{State: Died, Rooms: 30, Score: 1000},
		p2:   Standing{State: Died, Rooms: 30, Score: 1001},
		want: Outcome{Slot: 1, Reason: ByScore},
	}, {
		name: "equal rooms and score, fewer frames breaks it",
		p1:   Standing{State: Died, Rooms: 30, Score: 1000, Frame: 5000},
		p2:   Standing{State: Died, Rooms: 30, Score: 1000, Frame: 5001},
		want: Outcome{Slot: 0, Reason: ByFrames},
	}, {
		// Two peers replaying one recorded demo, which is what §10.4.9 proposes as a
		// protocol test vector. Not a hypothetical, and a race that could not report a
		// draw would have to invent a winner for it.
		name: "identical runs are drawn",
		p1:   Standing{State: Died, Rooms: 30, Score: 1000, Frame: 5000},
		p2:   Standing{State: Died, Rooms: 30, Score: 1000, Frame: 5000},
		want: Outcome{Slot: -1, Reason: Drawn},
	}, {
		// Mortals is shown and never scored. One glider left and out of the house beats
		// three in hand and dead, which this row is the proof of only because ByFinish
		// comes first; the row after it is the one that would catch Mortals sneaking into
		// the comparison.
		name: "gliders left do not decide",
		p1:   Standing{State: Finished, Rooms: 20, Mortals: 0},
		p2:   Standing{State: Died, Rooms: 20, Mortals: 3},
		want: Outcome{Slot: 0, Reason: ByFinish},
	}, {
		name: "gliders left do not break a tie either",
		p1:   Standing{State: Died, Rooms: 20, Mortals: 3},
		p2:   Standing{State: Died, Rooms: 20, Mortals: 0},
		want: Outcome{Slot: -1, Reason: Drawn},
	}} {
		if got := Winner(c.p1, c.p2); got != c.want {
			t.Errorf("%s:\n got %+v (%s)\nwant %+v (%s)", c.name, got, got, c.want, c.want)
		}
	}
}

func TestBothPeersComputeTheSameResult(t *testing.T) {
	// The property that makes a race need no authority: each side runs Result on its own
	// standing and the one it last heard, and the two answers agree. Match.Result exists
	// because the way to get this wrong is for each peer to pass itself first.
	mine := Standing{State: Died, Rooms: 30, Score: 1000, Frame: 9000}
	theirs := Standing{State: Finished, Rooms: 12, Score: 10, Frame: 3000}
	host := Match{Slot: 0}
	guest := Match{Slot: 1}
	a := host.Result(mine, theirs)
	b := guest.Result(theirs, mine)
	if a != b {
		t.Errorf("the two sides disagree: host says %+v, guest says %+v", a, b)
	}
	if a.Slot != 1 || a.Reason != ByFinish {
		t.Errorf("result = %+v, want player 2 by finishing the house", a)
	}
}

func TestOutcomeAndStateReadLikeEnglish(t *testing.T) {
	// These strings reach a screen, so the empty case and the nobody case are worth pinning:
	// "player 0 wins" and "player -1 wins" are both things an off-by-one would produce.
	for _, c := range []struct {
		in   Outcome
		want string
	}{
		{Outcome{Slot: -1, Reason: StillRacing}, "still racing"},
		{Outcome{Slot: 0, Reason: ByRooms}, "player 1 wins: more rooms visited"},
		{Outcome{Slot: 1, Reason: ByForfeit}, "player 2 wins: by forfeit"},
		{Outcome{Slot: -1, Reason: Drawn}, "no winner (drawn)"},
		{Outcome{Slot: -1, Reason: ByForfeit}, "no winner (by forfeit)"},
	} {
		if got := c.in.String(); got != c.want {
			t.Errorf("%+v: %q, want %q", c.in, got, c.want)
		}
	}
	if (Outcome{Reason: StillRacing}).Decided() {
		t.Error("a race in progress reports itself decided")
	}
	if !(Outcome{Slot: -1, Reason: Drawn}).Decided() {
		t.Error("a draw is a result and has to report itself as one")
	}
	for s, want := range map[RunState]string{
		Racing:   "racing",
		Finished: "finished the house",
		Died:     "out of gliders",
		Quit:     "left",
		99:       "state 99",
	} {
		if got := s.String(); got != want {
			t.Errorf("RunState(%d) = %q, want %q", s, got, want)
		}
	}
}
