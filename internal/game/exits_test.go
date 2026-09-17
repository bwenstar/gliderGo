package game

// The seven ways out of a room, each driven by scripted input and each checked against an
// arrival rect derived from the C by hand before the test was run.
//
// This is Stage 1.5b's acceptance criterion and the reason it is worth the length. Every
// other test in this package asserts about one function; a transit crosses five --
// CreateActiveRects builds the trigger box, HandleInteraction decides the glider may take
// it, a Start* mode function begins the walk, MoveRoomToRoom or one of the three link
// handlers changes the room, and ReadyGliderFromTransit or OffsetGlider places the glider
// on the other side. A single wrong constant anywhere in that chain produces a glider that
// arrives in the right room at the wrong place, which no unit test in the package can see
// and which is instantly obvious in play.
//
// So the arrival rect is the assertion, and the numbers below were computed from the
// original's source before the port was run. Each case states the derivation, because a
// number that merely matches the current code is a snapshot and a number that matches the C
// is a specification. The two ways they differ:
//
//	a snapshot fails when behaviour changes, and cannot say whether the change was right
//	a derivation fails when behaviour stops matching 1994, which is the only question here
//
// # A deviation from the plan's wording
//
// docs/PLAN.md asks for these seven "leaving Demo House's first room". That is not
// possible: Demo House has four of the seven exit kinds in the whole file and its first
// room has none of them. CD Demo House -- also one of the 22 originals -- has all seven, so
// the rooms are chosen by inspection there instead. Recorded in docs/IMPROVEMENTS.md.
//
// # Why the glider is placed rather than walked to the exit
//
// Each case resumes into the room with the glider already at the exit, using
// ResumeGameMode and a synthesised saved game, and then holds keys. Walking from a house's
// first room to a mailbox takes thousands of frames through rooms full of hazards, and a
// test that did it would be asserting about the hazards. The resume is the one thing here a
// player could not do, and it is the supported entry point -- see World.SavedGame.
//
// Six of the seven placements need no keys at all: a staircase, a duct, a transporter and a
// mailbox all fire from the hot-spot table as soon as the glider is inside the trigger box,
// including while it is still fading in. The two side doors need a direction held, because
// leaving through a wall means walking there.

import (
	"testing"

	"github.com/bwenstar/gliderGo/internal/game/player"
	"github.com/bwenstar/gliderGo/internal/house"
)

// exitCase is one of the seven.
type exitCase struct {
	name string

	// The placement. room and where are the resume point; faceLeft flips the glider,
	// which matters only for the left door.
	room     int16
	where    house.Point
	faceLeft bool

	// keys is held from frame 0 to the end, or the zero value for the six cases that
	// need no input.
	keys player.Keys

	// What must happen. frame is the frame the room changes on -- pinned because the
	// double RenderFrame on a transition is checked against it, and because a transit
	// that took a different number of frames took a different path.
	frame    int64
	wantRoom int16
	wantMode player.Mode
	wantDest player.Rect

	// budget is how long to run. A few frames past `frame` is enough; the assertions
	// are all made on the transition frame itself.
	budget int

	// derivation is quoted in the failure message, so that a failing number arrives
	// with the arithmetic that produced it instead of sending the reader to this file.
	derivation string
}

// exitCases is the table. The `where` values put the glider wholly inside the trigger box,
// because GliderInRect is containment and not intersection -- a glider merely touching an
// up staircase walks past it.
var exitCases = []exitCase{
	{
		name:     "right door",
		room:     28, // "Black Space", bg 3104, bounds 0x001F: both side walls open
		where:    house.Point{H: 440, V: 150},
		keys:     player.Keys{Right: true},
		frame:    28,
		wantRoom: 29, // floor 3, suite 9: one east of room 28's suite 8
		wantMode: player.GliderNormal,
		wantDest: player.Rect{Top: 182, Left: -16, Bottom: 202, Right: 32},
		budget:   34,
		derivation: "MoveRoomToRoom's ToRight arm calls OffsetGlider(ToLeft), which shifts " +
			"Dest by -RoomWide. The glider was at Left 491 on frame 27 and takes one more " +
			"5px step to 496, which is past the open-wall threshold; 496-512 = -16, and the " +
			"width is GliderWide 48.",
	},
	{
		name:     "left door",
		room:     28,
		where:    house.Point{H: 60, V: 150},
		faceLeft: true,
		keys:     player.Keys{Left: true},
		frame:    35,
		wantRoom: 21, // floor 3, suite 7
		wantMode: player.GliderNormal,
		wantDest: player.Rect{Top: 203, Left: 481, Bottom: 223, Right: 529},
		budget:   41,
		derivation: "The mirror: OffsetGlider(ToRight) adds RoomWide. The glider was at " +
			"Left -26 on frame 34 and steps to -31; -31+512 = 481. The mode is " +
			"GliderNormal only because the glider was already facing left -- see " +
			"TestLeftDoorTurnsARightFacingGlider.",
	},
	{
		name:     "up staircase",
		room:     192, // "(Emergency Stairway)", kUpStairs at v=28 h=307
		where:    house.Point{H: 330, V: 30},
		frame:    6,
		wantRoom: 193, // floor 13, suite 7: the floor above
		wantMode: player.GliderComingUp,
		wantDest: player.Rect{Top: 96, Left: 209, Bottom: 116, Right: 213},
		budget:   12,
		derivation: "ReadyGliderForTripUpStairs sets RightClip = GetUpStairsRightEdge(), " +
			"which searches the *destination* room for a kDownStairs and returns its " +
			"TopLeft.H + SrcRect(DownStairs).Right - 1 = 54 + 160 - 1 = 213. Dest starts at " +
			"(RightClip, GliderAppearsComingUp=100); FinishGliderUpStairs then steps " +
			"(HClimbStairsSpeed, VClimbStairsSpeed) = (-4, -4) to (209, 96) and clips the " +
			"width to RightClip - Dest.Left = 4.",
	},
	{
		name:     "down staircase",
		room:     193, // kDownStairs at v=28 h=54; trigger box {142,134,198,214}
		where:    house.Point{H: 140, V: 150},
		frame:    19,
		wantRoom: 192,
		wantMode: player.GliderComingDown,
		wantDest: player.Rect{Top: 104, Left: 308, Bottom: 124, Right: 312},
		budget:   25,
		derivation: "The mirror, and the crossed pairing is the C's: GetDownStairsLeftEdge " +
			"searches the destination for a kUpStairs and returns TopLeft.H + 1 = 308. Dest " +
			"starts at (LeftClip - GliderWide, GliderAppearsComingDown=100) = (260, 100), " +
			"steps (+4, +4) to (264, 104), and clips the left edge to Dest.Right - LeftClip " +
			"= 312 - 308 = 4.",
	},
	{
		name:     "ceiling duct",
		room:     4, // "Sticky Fly Paper", kCeilingTrans at v=6 h=417
		where:    house.Point{H: 420, V: 20},
		frame:    9,
		wantRoom: 5, // "7 Second Shopping Spree", whose own kCeilingTrans is at h=30
		wantMode: player.GliderDuctingIn,
		wantDest: player.Rect{Top: -16, Left: 34, Bottom: -16, Right: 82},
		budget:   15,
		derivation: "ReadyGliderFromTransit's LinkedToCeilingDuct arm centres the glider in " +
			"the destination duct, zeroes the height, then offsets up by the glider's own " +
			"height. The duct sprite is 56 wide at h=30, so the centre is 58 and the left " +
			"edge 58-24 = 34; Top = Bottom = 0 - GliderHigh + 4 = -16. The room assertion " +
			"alone would not be enough here: room 5 is also room 4's east neighbour, so it " +
			"is the mode and the zero-height rect that say this was the duct.",
	},
	{
		name:     "transporter",
		room:     32, // "Magic Mirror", kInvisTrans at v=122 h=227
		where:    house.Point{H: 240, V: 130},
		frame:    16,
		wantRoom: 33,
		wantMode: player.GliderTransportingIn,
		wantDest: player.Rect{Top: 147, Left: 244, Bottom: 167, Right: 292},
		budget:   22,
		derivation: "LinkedToOther, the general arm: the glider is centred in the " +
			"destination transporter's own rect at full size, 48x20.",
	},
	{
		name:     "mailbox",
		room:     69, // "Post Office", kMailboxLf at v=27 h=418; box {43,376,83,448}
		where:    house.Point{H: 385, V: 50},
		frame:    16,
		wantRoom: 0,
		// GliderMailOutRight, not ...Left: StartGliderMailingOut picks the side from the
		// *destination* object, and room 0's slot is a kMailboxRt.
		wantMode: player.GliderMailOutRight,
		wantDest: player.Rect{Top: 121, Left: 316, Bottom: 141, Right: 316},
		budget:   22,
		derivation: "LinkedToRightMailbox: Clip.Left += 79 and Clip.Bottom -= 25, then Dest " +
			"is given zero width at Clip.Left and a bottom of Clip.Bottom - 4, so the " +
			"glider emerges from a slit and is pushed out over the following frames.",
	},
}

// TestEveryExitKindArrivesWhereTheCSays is the seven-case table.
//
// The right-facing mailbox is deliberately absent and there are still seven cases, because
// the seven kinds are the two side doors, the two staircases, the duct, the transporter and
// the mail slot -- the two mailbox arms are one kind. kMailboxRt would need a glider heading
// *left* into the slot (see the facing test in HandleInteraction's MailItRight arm), which a
// resume cannot arrange in one step because the mailbox that is reachable in room 69 opens
// the other way.
func TestEveryExitKindArrivesWhereTheCSays(t *testing.T) {
	for _, c := range exitCases {
		t.Run(c.name, func(t *testing.T) {
			w := resumeAt(t, "CD Demo House", c.room, c.where, c.faceLeft, c.budget)
			w.KeyPoll = func(*player.Glider) player.Keys { return c.keys }

			// The transition frame is sampled through Present rather than after the run,
			// because the arrival rect survives only until the next frame moves it.
			//
			// **The last Present of the frame is the one that counts, and it is not the
			// first.** A transition presents once per wipe strip -- 116 or 160 times inside
			// one game frame -- so overwriting until the frame number changes is what
			// leaves the end-of-frame state behind. It matters for two separate reasons
			// here. The side doors change room from inside HandleInteraction, which runs
			// *before* HandleGlider, so the first present of the transition frame shows the
			// glider offset by -RoomWide but not yet stepped: {179,-21,199,27} rather than
			// {182,-16,202,32}. And the RenderFrames delta is only visible at the end,
			// because the second of the frame's two renders has not happened yet at the
			// first present.
			var got, cur *exitObservation
			renders, prevRenders := int64(0), int64(0)
			frame := int64(-1)
			w.Present = func() {
				if w.Frame != frame {
					if frame == c.frame {
						got = cur
					}
					prevRenders, frame, cur = renders, w.Frame, nil
				}
				cur = &exitObservation{
					room:     w.R.RoomNumber,
					mode:     w.P1.Mode,
					dest:     w.P1.Dest,
					renders:  w.RenderFrames,
					prevRend: prevRenders,
				}
				renders = w.RenderFrames
				if w.Frame >= int64(c.budget) {
					w.Quitting = true
					w.SwitchedOut = false
				}
			}

			w.NewGame(ResumeGameMode)
			if frame == c.frame && got == nil {
				got = cur // the run ended on the transition frame itself
			}

			if got == nil {
				t.Fatalf("frame %d never presented (ran to frame %d, room %d)",
					c.frame, w.Frame, w.R.RoomNumber)
			}
			if got.room != c.wantRoom {
				t.Errorf("frame %d room %d, want %d\n%s", c.frame, got.room, c.wantRoom, c.derivation)
			}
			if got.mode != c.wantMode {
				t.Errorf("frame %d mode %d, want %d\n%s", c.frame, got.mode, c.wantMode, c.derivation)
			}
			if got.dest != c.wantDest {
				t.Errorf("frame %d arrival rect %v, want %v\n%s", c.frame, got.dest, c.wantDest, c.derivation)
			}

			// A transition frame renders twice. Sampling the *last* Present of the frame
			// is what makes that visible and is also why the assertions above see the
			// state the player was left looking at rather than the middle of the wipe.
			if d := got.renders - got.prevRend; d != 2 {
				t.Errorf("frame %d advanced RenderFrames by %d, want 2 "+
					"(MoveRoomToRoom renders, then PlayGame renders again)", c.frame, d)
			}
		})
	}
}

type exitObservation struct {
	room     int16
	mode     player.Mode
	dest     player.Rect
	renders  int64
	prevRend int64
}

// TestLeftDoorTurnsARightFacingGlider is the other half of the left-door case, and the
// reason the table's faceLeft field exists.
//
// MoveRoomToRoom's ToLeft arm calls InsureGliderFacingLeft *before* ForceThisRoom, so a
// glider that walked out backwards -- holding left while still facing right, which is what
// a player does -- arrives in the next room mid-about-face. Same room, same arrival rect,
// different mode. Pinning both says the fix-up runs and that it does not disturb the
// placement.
func TestLeftDoorTurnsARightFacingGlider(t *testing.T) {
	const frame, budget = 35, 41
	w := resumeAt(t, "CD Demo House", 28, house.Point{H: 60, V: 150}, false, budget)
	w.KeyPoll = func(*player.Glider) player.Keys { return player.Keys{Left: true} }

	var mode player.Mode = -1
	var dest player.Rect
	var room int16 = -1
	w.Present = func() {
		if w.Frame == frame {
			mode, dest, room = w.P1.Mode, w.P1.Dest, w.R.RoomNumber
		}
		if w.Frame >= budget {
			w.Quitting = true
			w.SwitchedOut = false
		}
	}
	w.NewGame(ResumeGameMode)

	if room != 21 {
		t.Fatalf("frame %d room %d, want 21", frame, room)
	}
	if mode != player.GliderFaceLeft {
		t.Errorf("frame %d mode %d, want GliderFaceLeft %d "+
			"(InsureGliderFacingLeft should have started the about-face)",
			frame, mode, player.GliderFaceLeft)
	}
	if want := (player.Rect{Top: 203, Left: 481, Bottom: 223, Right: 529}); dest != want {
		t.Errorf("frame %d arrival rect %v, want %v -- the about-face must not move the glider",
			frame, dest, want)
	}
}

// resumeAt builds a world that will start in `room` with the glider at `where`.
//
// The saved game is synthesised rather than loaded, which is the whole trick: RoomNumber
// and Where are the only two fields that have to be right, and NewGame(ResumeGameMode)
// treats them as authoritative. SetObjectsToDefaults has to be called explicitly because
// the resume arm skips it -- a real resume is restoring object state the saved game carried,
// and a synthesised one carries none, so without this every clock in the house starts
// already collected.
func resumeAt(t *testing.T, houseName string, room int16, where house.Point, faceLeft bool, budget int) *World {
	t.Helper()
	w := playTestWorld(t, houseName, budget)
	facing := byte(1)
	if faceLeft {
		facing = 0
	}
	w.SavedGame = house.Game{
		RoomNumber:   room,
		Where:        where,
		NumGliders:   InitialGliders,
		WasStarsLeft: w.CountStarsInHouse(),
		GliderState:  player.GliderNormal,
		Facing:       facing,
	}
	w.SetObjectsToDefaults()
	return w
}
