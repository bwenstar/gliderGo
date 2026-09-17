package game

// Slumberland's basement, and why being stuck down there is not a defect.
//
// The house the game opens with has three rooms below its ground floor, and the first one
// a player meets is reached by the staircase in room 5, "Secret Agent Room", four rooms
// right of where the game starts. The room it arrives in is called "Going Up? No?" and it
// means it: there is a staircase back up, and on a fresh game nothing in the room can carry
// the glider to it.
//
// The reason is one byte. An up staircase is entered through a 112x32 trigger box at the
// top of the flight, at the object's own anchor (ObjectRects.c:734), and the only thing in
// either basement room that reaches that high is a floor vent standing directly under it --
// a four-pixel-wide updraught whose reach ends inside the box. Both of those vents are
// authored with `initial` 0, and SetObjectsToDefaults copies `initial` into `state` at the
// start of every new game (Play.c:601-660), so both start switched off. Each is wired to a
// thermostat in another room: "Good Night"'s is thrown from "Anabell Lee" next door, which
// is the escape a player trapped in the basement can actually reach, and "Going Up? No?"'s
// is thrown from "Switch Me" two floors above, which is not. Walk down the stairs before
// finding either plate and the basement is a dead end until the glider is lost.
//
// So this is a puzzle the authors built, not a transit that got dropped in the port, and
// this file is here to stop a later reader "fixing" it. It asserts the design in the house
// data -- which vent, which box, which switch -- and then flies the escape both ways round:
// with the vent as shipped, the glider falls and the room never changes; with the vent
// switched on, the same placement rides it into the staircase and comes up in "Choose Me"
// on the floor above.
//
// The exit machinery itself is not the subject here; exits_test.go already pins all seven
// kinds against the C, including this staircase, in CD Demo House. What is new is that a
// shipped house gates one of them behind a switch in a different room, which no test that
// places a glider in a trigger box can see.

import (
	"testing"

	"github.com/bwenstar/gliderGo/internal/game/player"
	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/render"
)

// The Slumberland rooms this file names, by index into the house's room array -- the same
// numbering `glidertool house rooms` prints and `glidertool replay -room` takes.
const (
	slumberGoingUp    int16 = 49  // "Going Up? No?", floor 0 suite 59: down from room 5
	slumberAnabellLee int16 = 44  // "Anabell Lee", floor 0 suite 60: has the thermostat
	slumberGoodNight  int16 = 45  // "Good Night", floor 0 suite 61: the reachable way out
	slumberChooseMe   int16 = 2   // "Choose Me", floor 1 suite 61: where that way out goes
	slumberSwitchMe   int16 = 116 // "Switch Me", floor 2 suite 57: the unreachable plate
)

// gatedStairs is one of the two basement staircases and the wiring that opens it.
type gatedStairs struct {
	name string

	room   int16 // the basement room
	stairs int   // its kUpStairs slot
	vent   int   // the kFloorVent slot whose column reaches the staircase's trigger box

	// The switch that turns that vent on: which room holds it, which slot it is in, and
	// what it is, quoted in failures because "a thermostat in Switch Me" is the fact a
	// reader needs and "room 116 object 0" is not.
	switchRoom int16
	switchSlot int
	switchWhat string
}

var gatedStairsCases = []gatedStairs{
	{
		name: "Going Up? No?",
		room: slumberGoingUp, stairs: 0, vent: 1,
		switchRoom: slumberSwitchMe, switchSlot: 0,
		switchWhat: "a thermostat in \"Switch Me\", floor 2 suite 57",
	},
	{
		name: "Good Night",
		room: slumberGoodNight, stairs: 0, vent: 3,
		switchRoom: slumberAnabellLee, switchSlot: 1,
		switchWhat: "a thermostat in \"Anabell Lee\", the next room along the basement",
	},
}

// TestSlumberlandsBasementStairsAreGatedByARemoteSwitch is the data half: the house says
// what the rooms' names imply.
//
// Four things are asserted per staircase, and the third is the one that makes the basement
// a trap rather than merely hard: no *other* air source in the room reaches the trigger
// box, so switching the one vent off closes the only way in.
func TestSlumberlandsBasementStairsAreGatedByARemoteSwitch(t *testing.T) {
	w := playTestWorld(t, "Slumberland", 0)
	w.SetObjectsToDefaults()

	for _, c := range gatedStairsCases {
		t.Run(c.name, func(t *testing.T) {
			rm := &w.H.Rooms[c.room]

			// The trigger box, built the way CreateActiveRects builds it.
			stairs := rm.Objects[c.stairs]
			if stairs.What != UpStairs {
				t.Fatalf("room %d slot %d is object type 0x%02X, want kUpStairs 0x%02X",
					c.room, c.stairs, stairs.What, UpStairs)
			}
			box := render.Offset(render.SetRect(0, 0, 112, 32),
				stairs.Transport().TopLeft.H, stairs.Transport().TopLeft.V)

			// The vent under it, and its column, likewise.
			vent := rm.Objects[c.vent]
			if vent.What != FloorVent {
				t.Fatalf("room %d slot %d is object type 0x%02X, want kFloorVent 0x%02X",
					c.room, c.vent, vent.What, FloorVent)
			}
			col := ventColumn(vent)
			if col.Left < box.Left || col.Right > box.Right || col.Top < box.Top || col.Top >= box.Bottom {
				t.Errorf("the column %v does not end inside the staircase's box %v -- "+
					"the vent this test calls the way up is not under the stairs", col, box)
			}

			// Nothing else in the room gets a glider that high. Every other blower either
			// stands somewhere else along the floor or cannot lift far enough.
			for i := range rm.Objects {
				if i == c.vent {
					continue
				}
				other := rm.Objects[i]
				if !isFloorLift(other.What) {
					continue
				}
				oc := ventColumn(other)
				if oc.Right > box.Left && oc.Left < box.Right && oc.Top < box.Bottom {
					t.Errorf("slot %d (type 0x%02X) also reaches the staircase with column %v, "+
						"so the room has a second way up and is not gated at all",
						i, other.What, oc)
				}
			}

			// Off as authored, and off after a new game starts, which are two different
			// bytes: the editor writes `initial` and the house on disk carries whatever
			// `state` it was last saved with. "Going Up? No?" is saved with its vent on,
			// so reading the wrong byte would make this room escapable and the other not.
			if got := vent.Blower().Initial; got != 0 {
				t.Errorf("the vent's initial byte is %d, want 0 -- it is meant to start off", got)
			}
			if got := vent.Blower().State; got != 0 {
				t.Errorf("after SetObjectsToDefaults the vent's state is %d, want 0 "+
					"(Play.c:601-660 copies initial into state on every new game)", got)
			}

			// And the plate that opens it is somewhere else.
			sw := w.H.Rooms[c.switchRoom].Objects[c.switchSlot]
			if !ObjectIsLinkSwitch(sw.What) {
				t.Fatalf("room %d slot %d is object type 0x%02X, which carries no link -- "+
					"%s is not where the switch is", c.switchRoom, c.switchSlot, sw.What, c.switchWhat)
			}
			if got := w.GetRoomLinked(sw); got != c.room {
				t.Errorf("%s points at room %d, want %d", c.switchWhat, got, c.room)
			}
			if got := w.GetObjectLinked(sw); got != int16(c.vent) {
				t.Errorf("%s points at object %d, want the vent in slot %d", c.switchWhat, got, c.vent)
			}
			if got := sw.Switch().Type; int16(got) != Toggle {
				t.Errorf("%s sends action %d, want Toggle %d", c.switchWhat, got, Toggle)
			}
		})
	}
}

// ventColumn is CreateActiveRects' blower arm (hotspots.go:79-83) for one object: the
// four-pixel updraught, centred on the artwork, running `distance` pixels up from the
// object's top edge.
func ventColumn(obj house.Object) Rect {
	a := obj.Blower()
	col := render.SetRect(0, -a.Distance, FloorColumnWide, 0)
	col = render.Offset(col, render.HalfWide(render.SrcRect(obj.What))-FloorColumnWide/2, 0)
	return render.Offset(col, a.TopLeft.H, a.TopLeft.V)
}

// isFloorLift is the five object types whose hot spot is an upward column of that shape.
// The flames are left out on purpose: their columns are split at DeadlyFlameHeight and
// neither basement room has one.
func isFloorLift(what int16) bool {
	switch what {
	case FloorVent, FloorBlower, SewerGrate, GrecoVent, SewerBlower:
		return true
	}
	return false
}

// TestSlumberlandsBasementEscapeNeedsTheVentSwitchedOn is the same claim from the glider's
// side, flown twice from one placement in "Good Night".
//
// The placement is the vent's column, 250 pixels up the room and squarely under the
// staircase: it satisfies both the column's four-pixel width and the trigger box's, which
// is the alignment the room asks a player for. With the vent as shipped nothing happens to
// the glider except the floor. With it toggled -- the poke the thermostat next door sends
// -- the same placement rides up into the box and the staircase takes it.
func TestSlumberlandsBasementEscapeNeedsTheVentSwitchedOn(t *testing.T) {
	const (
		where  = 340 // the glider spans 340..388: over the column at 358..362, inside the box's 325..437
		startV = 250
		budget = 150
	)

	for _, on := range []bool{false, true} {
		name := "vent off, as the house ships"
		if on {
			name = "vent switched on from Anabell Lee"
		}
		t.Run(name, func(t *testing.T) {
			w := resumeAt(t, "Slumberland", slumberGoodNight,
				house.Point{H: where, V: startV}, false, budget)
			if on {
				// What HandleSwitches does when the glider touches the thermostat, with
				// local = -1 because the room has not been composed yet: write the house
				// and publish nothing. The switch's own wiring is asserted above; this is
				// only the poke it sends.
				if !w.SetObjectState(slumberGoodNight, int16(gatedStairsCases[1].vent), Toggle, -1) {
					t.Fatal("Toggle on the staircase vent changed nothing")
				}
			}

			// Sampled through Present because the arrival rect survives only until the
			// next frame moves the glider, the same reason exits_test.go samples there.
			var (
				arrived             bool
				room                int16
				mode                player.Mode
				dest                player.Rect
				frame               int64
				highest             int16 = 32767
				firstLeft, lastLeft int16
			)
			w.Present = func() {
				if !arrived && w.R.RoomNumber != slumberGoodNight {
					arrived, room, mode, dest, frame = true, w.R.RoomNumber, w.P1.Mode, w.P1.Dest, w.Frame
				}
				if w.P1.Dest.Top < highest {
					highest = w.P1.Dest.Top
				}
				if firstLeft == 0 {
					firstLeft = w.Mortals
				}
				lastLeft = w.Mortals
				if w.Frame >= int64(budget) {
					w.Quitting = true
					w.SwitchedOut = false
				}
			}
			w.NewGame(ResumeGameMode)

			if !on {
				if arrived {
					t.Errorf("frame %d left for room %d with the vent off -- "+
						"the staircase must be out of reach until it is switched on", frame, room)
				}
				if highest < startV {
					t.Errorf("the glider rose to %d from %d with the vent off, "+
						"so something is lifting it", highest, startV)
				}
				if lastLeft >= firstLeft {
					t.Errorf("the glider still has %d lives of %d after %d frames -- "+
						"with nothing to ride it should have hit the floor",
						lastLeft, firstLeft, budget)
				}
				return
			}

			if !arrived {
				t.Fatalf("the room never changed in %d frames; the glider got as high as %d "+
					"and the staircase's box starts at 28", budget, highest)
			}
			if room != slumberChooseMe {
				t.Errorf("frame %d arrived in room %d, want %d (\"Choose Me\", floor 1 suite 61: "+
					"MoveRoomToRoom takes an up staircase to the same suite one floor up)",
					frame, room, slumberChooseMe)
			}
			if mode != player.GliderComingUp {
				t.Errorf("frame %d mode %d, want GliderComingUp %d", frame, mode, player.GliderComingUp)
			}
			// Derived from the C, not from this port: ReadyGliderForTripUpStairs sets
			// RightClip = GetUpStairsRightEdge(), which searches the *destination* for a
			// kDownStairs and returns TopLeft.H + SrcRect(DownStairs).Right - 1. "Choose
			// Me" has one at h=24, so that is 24 + 160 - 1 = 183. Dest starts at
			// (RightClip, GliderAppearsComingUp = 100); FinishGliderUpStairs steps
			// (-4, -4) to (179, 96) and clips the width to RightClip - Dest.Left = 4.
			if want := (player.Rect{Top: 96, Left: 179, Bottom: 116, Right: 183}); dest != want {
				t.Errorf("frame %d arrival rect %v, want %v", frame, dest, want)
			}
		})
	}
}
