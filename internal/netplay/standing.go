package netplay

import "fmt"

// RunState is how one peer's run stands: going, or over in one of three ways.
//
// The two endings are the original's own two, and they are not symmetrical. Play.c:408-411
// branches on `Mortals < 0` after the game-over countdown: DoGameOver is the *win*, the player
// completed the house, and DoDiedGameOver is running out of gliders. A race has to know which,
// and Winner is where that matters.
type RunState uint8

const (
	// Racing: the run is in progress. The only state a standing sent during play can have.
	Racing RunState = 0

	// Finished: the house was completed -- DoGameOver, the original's win
	// (internal/game/play.go, `w.Mortals < 0` is the other branch).
	Finished RunState = 1

	// Died: out of gliders, DoDiedGameOver. A run that ended this way still counts for
	// everything Winner measures; dying is how most races will end.
	Died RunState = 2

	// Quit: the player left without finishing or dying -- closed the window, pressed the
	// quit key, or had their process killed, which is the case docs/PLAN.md Stage 3's
	// acceptance clause names. Distinct from Died because it is a forfeit and Winner treats
	// it as one, and because a player who quits has not been beaten at anything.
	Quit RunState = 3
)

func (s RunState) String() string {
	switch s {
	case Racing:
		return "racing"
	case Finished:
		return "finished the house"
	case Died:
		return "out of gliders"
	case Quit:
		return "left"
	}
	return fmt.Sprintf("state %d", uint8(s))
}

// Ended reports whether this run is over, however it ended.
func (s RunState) Ended() bool { return s != Racing }

// Standing is one peer's progress: everything the other side is shown, and everything the
// result is computed from.
//
// docs/PLAN.md Stage 3 lists what goes in it -- "room index, floor/suite, score, lives,
// alive/dead, finished" -- and two fields here are not on that list. Rooms is the one the
// result turns on and had to be added because the metric is rooms visited, not room index; and
// Frame is the tie-break of last resort, for the reason 4.11 gives about wall clocks.
//
// It is 28 bytes on the wire and nothing in it is a position, a velocity or an object state.
// That is the same restraint §10.4.4 states for the lock-step packet, arrived at from the other
// direction: lock-step must send no state because state is derived, and a race must send no
// state because the other glider is not in this world at all.
type Standing struct {
	// State is how the run stands. Everything below is as of the moment this was made.
	State RunState

	// Rooms is **the metric**: distinct rooms entered, World.CountRoomsVisited, which is
	// House.c:511-531 and the number the original's own high-score table stores in
	// scoresType.levels[] (docs/analysis/scoring.md §7.2). docs/PLAN.md Stage 3 argues the
	// choice and its three rejected alternatives; what matters here is that it is not
	// invented for the race, and that it counts rooms *left*, not rooms stood in, so it is
	// one behind a player counting doors.
	Rooms int16

	// Room, Floor and Suite are where the glider is, for the opponent panel and nothing
	// else. Room is an index into the house's rooms[] and is meaningless as a distance --
	// it is authoring order (docs/PLAN.md Stage 3's first rejected alternative) -- which is
	// exactly why it is here as a label and Rooms is the metric.
	Room  int16
	Floor int16
	Suite int16

	// Mortals is gliders left, World.Mortals. Shown, not scored: a player who finishes the
	// house with one glider left has beaten a player still holding three.
	Mortals int16

	// Score is World.Score, the first tie-break. Second because it can be farmed in one
	// room without travelling, which is the third rejected alternative in the same list.
	Score int32

	// Frame is the sender's own gameFrame when this standing was made -- World.Frame,
	// counted by that peer's simulation and by nothing else. It is the last tie-break and
	// it is also the ordering rule on the wire: a standing whose Frame is behind one already
	// received is a protocol error, not a late packet (Conn.Recv).
	//
	// Frames rather than wall clock, and the reason is not subtle. Two peers need no shared
	// clock, no countdown and no starting gun if the thing being compared is how many frames
	// each has simulated -- so Meet can return at different moments on the two machines and
	// the race is still fair. docs/IMPROVEMENTS.md 4.11 is the standing measurement of how
	// much a wall clock varies between two runs of this port on one machine; between two
	// machines it would be worse, and it would be the *result* that varied.
	Frame uint32
}

// Report is a MsgStanding as received: a standing, and which player sent it.
//
// The slot is on the wire and is not addressing -- there is one other end of the connection,
// so the sender is never in doubt. It is a check, and Conn.checkSlot says what of.
type Report struct {
	Slot uint8
	Standing
}

func (*Report) msgType() uint8 { return MsgStanding }

// standingSize is this package's own layout, in the shape §10.4's messages are cut to: the
// 8-byte envelope, senderSlot where every other message has it, then fields in descending width
// so that nothing needs a pad byte to sit at its natural offset.
//
//	 0  magic      uint16   0x474C
//	 2  version    uint8    1
//	 3  msgType    uint8    0x30
//	 4  matchID    uint32
//	 8  senderSlot uint8    0 = player 1, 1 = player 2
//	 9  state      uint8    RunState
//	10  rooms      int16    the metric
//	12  room       int16    index into rooms[], display only
//	14  floor      int16    display only
//	16  suite      int16    display only
//	18  mortals    int16    gliders left
//	20  score      int32
//	24  frame      uint32   the sender's gameFrame
const standingSize = 28

// EncodeStanding lays out a MsgStanding.
func EncodeStanding(matchID uint32, slot uint8, s Standing) []byte {
	b := make([]byte, standingSize)
	header(b, MsgStanding, matchID)
	b[8] = slot
	b[9] = uint8(s.State)
	be16(b[10:], uint16(s.Rooms))
	be16(b[12:], uint16(s.Room))
	be16(b[14:], uint16(s.Floor))
	be16(b[16:], uint16(s.Suite))
	be16(b[18:], uint16(s.Mortals))
	be32(b[20:], uint32(s.Score))
	be32(b[24:], s.Frame)
	return b
}

func decodeReport(b []byte) (Msg, error) {
	if len(b) < standingSize {
		return nil, short("MsgStanding", len(b), standingSize)
	}
	r := &Report{Slot: b[8]}
	if r.Slot > 1 {
		return nil, fmt.Errorf("%w: MsgStanding senderSlot is %d, must be 0 or 1",
			ErrProtocol, r.Slot)
	}
	// The state byte is checked against the four that exist rather than taken on trust. This
	// is §10.4.10's Boolean rule generalised: a field with four legal values must not arrive
	// holding a fifth, because the only thing downstream can do with an unknown state is
	// guess, and Winner guessing is a wrong result rather than an error message.
	if b[9] > uint8(Quit) {
		return nil, fmt.Errorf("%w: MsgStanding state is %d, and there are %d states",
			ErrProtocol, b[9], Quit+1)
	}
	r.State = RunState(b[9])
	r.Rooms = i16(b[10:])
	r.Room = i16(b[12:])
	r.Floor = i16(b[14:])
	r.Suite = i16(b[16:])
	r.Mortals = i16(b[18:])
	r.Score = i32(b[20:])
	r.Frame = u32(b[24:])
	return r, nil
}

// SendStanding reports this peer's progress. One per meaningful change, which docs/PLAN.md
// Stage 3 lists: room, floor/suite, score, lives, alive/dead, finished. At 28 bytes a change,
// a player walking briskly through a house sends a few hundred bytes a minute.
func (c *Conn) SendStanding(slot uint8, s Standing) error {
	return c.Send(EncodeStanding(c.matchID, slot, s))
}

// Abandoned is the standing to score a departed peer on: its last one, turned into a forfeit
// only if its run had not already ended.
//
// It exists because "the connection went away" is two different events wearing one face, and the
// difference decides the race. A peer that vanishes while still flying -- MsgBye, the end of stream
// or the reset of a killed process, a pulled cable -- has forfeited, and Winner's first rule says
// the other side wins. A peer that reported Finished or Died and *then* hung up has not forfeited
// anything; it finished the race and closed the window, which is the ordinary way a match ends.
// Fold the two together and the common case comes out as a forfeit, which would mean every
// completed race was decided by whoever quit second.
//
// Both peers must apply this, to the same input, or they disagree about a result they each
// compute for themselves -- so it lives here next to Winner rather than in whatever loop happens
// to notice the connection close.
func Abandoned(last Standing) Standing {
	if last.State.Ended() {
		return last
	}
	last.State = Quit
	return last
}

// Reason is why an Outcome came out the way it did, in the order Winner tries them.
type Reason uint8

const (
	// StillRacing: at least one run has not ended, so there is no result yet.
	StillRacing Reason = iota

	// ByForfeit: somebody left. Checked before anything else -- see Winner.
	ByForfeit

	// ByFinish: one peer completed the house and the other did not.
	ByFinish

	// ByRooms: the metric. The common way a race is decided.
	ByRooms

	// ByScore: docs/PLAN.md Stage 3's first tie-break.
	ByScore

	// ByFrames: its second, and the last thing there is to compare.
	ByFrames

	// Drawn: everything compared equal. Reachable, and not only in theory -- two peers
	// replaying the same recorded demo against the same house would tie on all four
	// (internal/replay exists, and docs/analysis/determinism-networking.md §10.4.9 proposes
	// exactly that as a protocol test vector).
	Drawn
)

func (r Reason) String() string {
	switch r {
	case StillRacing:
		return "still racing"
	case ByForfeit:
		return "by forfeit"
	case ByFinish:
		return "finished the house"
	case ByRooms:
		return "more rooms visited"
	case ByScore:
		return "higher score"
	case ByFrames:
		return "fewer frames"
	case Drawn:
		return "drawn"
	}
	return fmt.Sprintf("reason %d", uint8(r))
}

// Outcome is the result of a race.
type Outcome struct {
	// Slot is the winner: 0 for player 1, 1 for player 2. **-1 means nobody**, which is
	// three different situations distinguished by Reason -- StillRacing (not over), Drawn
	// (over, equal), and ByForfeit (both left).
	Slot int8

	// Reason is which comparison decided it.
	Reason Reason
}

// Decided reports whether there is a result to show.
func (o Outcome) Decided() bool { return o.Reason != StillRacing }

// String is the one-line result, for a log or a screen.
func (o Outcome) String() string {
	switch {
	case o.Reason == StillRacing:
		return "still racing"
	case o.Slot < 0:
		return fmt.Sprintf("no winner (%s)", o.Reason)
	}
	return fmt.Sprintf("player %d wins: %s", o.Slot+1, o.Reason)
}

// Winner decides a race. p1 is player 1's standing (slot 0), p2 is player 2's (slot 1), and
// both must be that peer's latest.
//
// The order of the tests is the whole design, so each one gets its reason:
//
//  1. **A forfeit beats everything, and is checked before the race is even over.** A player
//     who quits or whose process is killed has left; the other one wins, still flying or not.
//     That is docs/PLAN.md Stage 3's "killing the guest mid-race leaves the host in a defined
//     state", and the defined state is this: a result, immediately, with a reason that says it
//     was not earned in the air. Both gone is nobody's win.
//  2. **A run still in progress has no result.** Anything else would let a player win by
//     stopping.
//  3. **Finishing the house outranks the metric, and this is a deliberate departure from a
//     literal reading of the plan.** PLAN says the winner is decided by rooms visited, and
//     taken literally that means a player who completed the house on a direct route loses to
//     one who wandered into more rooms and then died. For a race that is plainly wrong: the
//     house has an end, reaching it is the point, and "furthest on one life" -- PLAN's own
//     phrase for the mode -- cannot sensibly rank a death above an escape. So Finished first,
//     and then the metric decides between two finishers (routes differ) or two deaths.
//  4. **Rooms, then score, then fewer frames**, which is PLAN's order with "time" read as
//     frames simulated. See Standing.Frame for why frames and not a clock.
//
// Nothing here consults Mortals. A player who finishes with one glider left has beaten a
// player still holding three, and a player who dies with gliders left has not died -- Died
// means Mortals went below zero, so among two Died runs the field is equal and carries no
// information anyway.
func Winner(p1, p2 Standing) Outcome {
	q1, q2 := p1.State == Quit, p2.State == Quit
	switch {
	case q1 && q2:
		return Outcome{Slot: -1, Reason: ByForfeit}
	case q1:
		return Outcome{Slot: 1, Reason: ByForfeit}
	case q2:
		return Outcome{Slot: 0, Reason: ByForfeit}
	}
	if !p1.State.Ended() || !p2.State.Ended() {
		return Outcome{Slot: -1, Reason: StillRacing}
	}
	if f1, f2 := p1.State == Finished, p2.State == Finished; f1 != f2 {
		return Outcome{Slot: slotOf(f1), Reason: ByFinish}
	}
	if p1.Rooms != p2.Rooms {
		return Outcome{Slot: slotOf(p1.Rooms > p2.Rooms), Reason: ByRooms}
	}
	if p1.Score != p2.Score {
		return Outcome{Slot: slotOf(p1.Score > p2.Score), Reason: ByScore}
	}
	if p1.Frame != p2.Frame {
		return Outcome{Slot: slotOf(p1.Frame < p2.Frame), Reason: ByFrames}
	}
	return Outcome{Slot: -1, Reason: Drawn}
}

// Result is Winner with the two standings put the right way round.
//
// It exists because the interesting property of a race with no authority is that both peers
// compute the result themselves and must agree, and the way to fail at that is for each side to
// pass its own standing first. Winner's arguments are slots, not points of view; this is the
// one place that knows which slot the caller is.
func (m Match) Result(mine, theirs Standing) Outcome {
	if m.Slot == 0 {
		return Winner(mine, theirs)
	}
	return Winner(theirs, mine)
}

// slotOf turns "player 1 won this comparison" into a slot number.
func slotOf(firstWins bool) int8 {
	if firstWins {
		return 0
	}
	return 1
}
