package game

// The two copies of "which objects carry a link" have to agree.
//
// internal/game has ObjectIsLinkTransport and ObjectIsLinkSwitch: line-for-line
// transcriptions of the switch statements in Objects.c:219-250, carrying the comments
// that explain why kSoundTrigger is excluded and why the two families read the same
// bytes through different union members. internal/house has LinkCarryingTransport and
// LinkCarryingSwitch: map lookups, needed there because the linter cannot import this
// package (internal/house imports nothing of the port's, which is what lets both
// internal/render and internal/game depend on it).
//
// Collapsing one into the other was considered and rejected. The transcription is
// evidence about the original and reads like the C beside it; the map is the shape the
// linter wants. Keeping both is fine as long as nothing can drift, which is what this
// file is for: it walks every one of the 144 object codes and demands the same answer
// from both.

import (
	"testing"

	"github.com/bwenstar/gliderGo/internal/house"
)

func TestLinkPredicatesAgreeWithHouse(t *testing.T) {
	// The whole `what` space, not just the defined codes: an undefined code must be
	// "no link" in both, and the ranges these predicates live in border each other.
	for what := int16(0); what <= 0x90; what++ {
		if got, want := house.LinkCarryingTransport(what),
			ObjectIsLinkTransport(what); got != want {
			t.Errorf("what 0x%02X (%s): house.LinkCarryingTransport = %v, "+
				"game.ObjectIsLinkTransport = %v",
				uint16(what), house.ObjectName(what), got, want)
		}
		if got, want := house.LinkCarryingSwitch(what),
			ObjectIsLinkSwitch(what); got != want {
			t.Errorf("what 0x%02X (%s): house.LinkCarryingSwitch = %v, "+
				"game.ObjectIsLinkSwitch = %v",
				uint16(what), house.ObjectName(what), got, want)
		}
	}

	// And the counts, so that a change which broke both identically -- deleting a
	// case from the switch and the same name from the map -- still fails. Six
	// transports (Objects.c:219-233) and eight switches (Objects.c:236-250).
	transports, switches := 0, 0
	for what := int16(0); what <= 0x90; what++ {
		if ObjectIsLinkTransport(what) {
			transports++
		}
		if ObjectIsLinkSwitch(what) {
			switches++
		}
	}
	if transports != 6 {
		t.Errorf("%d link-carrying transports, want 6", transports)
	}
	if switches != 8 {
		t.Errorf("%d link-carrying switches, want 8", switches)
	}
}

// TestLinkAccessorsAgreeWithHouse pins the other half: that reading a link through
// house.LinkWhere/LinkWho gives what GetRoomLinked and GetObjectLinked read through
// the union members. The two families alias at payload offset 6, so a mistake here
// would not be a compile error -- it would be a transporter that went to the wrong
// place in one code path and the right one in another.
func TestLinkAccessorsAgreeWithHouse(t *testing.T) {
	cases := []struct {
		what  int16
		where int16
		who   byte
	}{
		{FloorTrans, 1234, 7},
		{CeilingTrans, house.UnlinkedWhere, house.UnlinkedWho},
		{MailboxLf, 8, 0},
		{LightSwitch, 1234, 7},
		{Trigger, 9, 23},
		{SoundTrigger, 3000, house.UnlinkedWho}, // carries no room link at all
		{UpStairs, 0, 0},                        // a transport that stores no link
	}
	for _, c := range cases {
		o := house.Object{What: c.what}
		if house.LinkCarryingSwitch(c.what) || c.what == SoundTrigger {
			o.SetSwitch(house.Switch{Where: c.where, Who: c.who})
		} else {
			o.SetTransport(house.Transport{Where: c.where, Who: c.who})
		}

		name := house.ObjectName(c.what)
		carries := ObjectIsLinkTransport(c.what) || ObjectIsLinkSwitch(c.what)

		gotWhere, ok := house.LinkWhere(o)
		if ok != carries {
			t.Errorf("%s: LinkWhere reports %v, the game's predicates say %v",
				name, ok, carries)
			continue
		}
		if carries && gotWhere != c.where {
			t.Errorf("%s: LinkWhere = %d, want %d", name, gotWhere, c.where)
		}
		gotWho, ok := house.LinkWho(o)
		if ok != carries {
			t.Errorf("%s: LinkWho reports %v, want %v", name, ok, carries)
			continue
		}
		if carries && gotWho != c.who {
			t.Errorf("%s: LinkWho = %d, want %d", name, gotWho, c.who)
		}
	}
}
