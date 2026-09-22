package replay_test

// The measurement half of docs/PLAN.md Stage 2's "designed against the quantitative
// profile of the originals in docs/analysis/original-houses.md".
//
// 10.2 of that document turns the measurements of all 4,070 shipped rooms into a table of
// bands per size tier, and each of the port's own houses is written against one column of
// it. What the per-house files hold is the column; what this file holds is the three
// assertions that are not a cell of the table.
//
// # The counting used to live here and does not any more
//
// It was a hundred lines of tallies in this file, shared between the two house tests for
// one reason: the houses are at different tiers, so if each counted for itself then a
// divergence between them could be a difference in the houses or a difference in the
// arithmetic, and a failure would not say which. That argument was right, and it kept
// being right one directory further out -- the same numbers are what an author wants
// *printed* before there is a test to fail, which a test cannot do because a test only
// speaks when it fails. So the walk is internal/profile now and `glidertool house stats`
// prints it (docs/IMPROVEMENTS.md 4.16). Three things came with the move:
//
//   - The bands are no longer transcribed here. internal/profile holds one copy of 10.2
//     and TestTheTierTableMatchesTheDocument parses the markdown to check it cell by
//     cell, so the numbers live in one place and the document is the place.
//   - All eighteen rows are checked now, not sixteen. prize:enemy was hand-written in one
//     of the two files and absent from the other, and BFS eccentricity was not computed
//     at all -- it is a graph walk over the room openings, and until the walk existed the
//     row was simply left out with a note saying so.
//   - Because all eighteen are checked, a house that misses one has to say which and why
//     in its own file. See checkTier.
//
// # Why dark rooms is in here and not filed as impossible
//
// Every row except two is computable from internal/house alone. A dark room is a room
// GetNumberOfLights answers 0 for, and that is a method on *render.Scene
// (internal/render/locale.go:1350-1386) because it reads the background's self-lighting
// table and each light object's switch state; internal/house cannot import
// internal/render and would not want to, since the rule is a rendering rule.
//
// openhouse_test.go used to say the row therefore "cannot be computed", and that was
// wrong: a Scene over a parsed house can be asked, for three lines, and the row caught
// the one thing a band on its own does not -- a house whose light went out because its
// background changed. The Bell Turret did exactly that: it was kRoof, which lights
// itself, and became kPaneledRoom, which does not.

import (
	"testing"

	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/profile"
)

// checkTier measures a house against one of 10.2's five columns and insists that the rows
// it misses are exactly the ones named.
//
// Not "misses nothing", which is what this replaced and which is not a claim any house can
// make. 10.2's bands are the observed spread of the 1994 houses tier by tier, so landing
// inside all eighteen would mean being that tier's outlier in eighteen ways at once, and
// neither tutorial-sized original manages it: Empty House is at 34.3 % empty rooms against
// a 15-25 % band and Demo House at 53.3 %. docs/IMPROVEMENTS.md 4.18 is the entry about
// where the tiers and 10.3's construction procedure disagree.
//
// So a miss is allowed and the price of allowing it is naming it in the test and arguing
// for it in the comment above. The map is checked both ways, which is what stops it
// becoming a list of things nobody looks at: an unnamed miss fails, and a named row that
// has stopped missing fails too, because an exemption whose reason has expired is a
// paragraph that has started lying.
func checkTier(t *testing.T, h *house.House, tier profile.Tier, allowed map[string]string) profile.Profile {
	t.Helper()

	// No assets. A house of ours carries no art of its own (docs/IMPROVEMENTS.md 4.15),
	// so no room reads its openings from a 'bnds' resource and the room graph is exact
	// with an empty tree -- which is also why these tests need nothing extracted.
	p := profile.Measure(h, nil)

	missed := map[string]bool{}
	for _, m := range profile.Check(p, tier) {
		missed[m.Row.Target] = true
		band := m.Row.Bands[tier].String(m.Row.Unit)
		if why, ok := allowed[m.Row.Target]; ok {
			// Logged rather than passed over in silence. The lesson of the dark-room row
			// is that a number can be inside its band and still be wrong -- 3.9 % against
			// a 0-4 % band printed green -- so the ones deliberately outside are the last
			// ones that should go unprinted.
			t.Logf("%s is %s where %s wants %s, allowed: %s",
				m.Row.Target, m.Row.Unit.Format(m.Value), tier, band, why)
			continue
		}
		if !m.Measured {
			t.Errorf("%s cannot be measured for this house, and %s wants %s "+
				"(original-houses.md 10.2)", m.Row.Target, tier, band)
			continue
		}
		t.Errorf("%s = %s, want %s (original-houses.md 10.2, %s)",
			m.Row.Target, m.Row.Unit.Format(m.Value), band, tier)
	}

	// A key that is not a row name can never match a miss, so without this it would read
	// as a row that had stopped missing -- the same message for the opposite problem.
	names := map[string]bool{}
	for _, r := range profile.Rows() {
		names[r.Target] = true
	}
	for target, why := range allowed {
		switch {
		case !names[target]:
			t.Errorf("%q is not one of 10.2's rows, so the exemption (%s) can never "+
				"apply: the eighteen names are the Target fields of profile.Rows()",
				target, why)
		case !missed[target]:
			t.Errorf("%s is inside %s's band now, so the exemption is stale: delete it "+
				"and the paragraph arguing for it (%s)", target, tier, why)
		}
	}
	return p
}

// checkStartRoom is 10.3 step 12, which both houses obey and which is not a 10.2 band: the
// first room a player sees is richer than average and cannot kill them.
//
// (The two messages used to cite step 9. Step 9 is the hazard ridge at two-thirds depth;
// step 12 is "set firstRoom's contents richer than average: 13 objects, no enemies". The
// citation was simply wrong and nothing checks a step number.)
//
// The room comes from the profile rather than from the firstRoom field, because the two
// differ for a house that names a room it does not contain -- the original clamps such a
// house to room 0 (play.go:623-633) -- and a house of ours that did that should fail here
// rather than have the wrong room quietly checked.
func checkStartRoom(t *testing.T, h *house.House, p profile.Profile) {
	t.Helper()
	if p.Start < 0 || int(p.Start) >= len(h.Rooms) {
		t.Fatalf("this house has no room to start in at all")
	}
	if p.Start != h.FirstRoom {
		t.Fatalf("firstRoom = %d but the game would start in room %d: 10.1 step 6 wants "+
			"firstRoom to index a real room", h.FirstRoom, p.Start)
	}
	first := &h.Rooms[p.Start]
	if n, avg := first.LiveObjects(), p.PerRoom(p.Objects); float64(n) < avg {
		t.Errorf("the start room %q holds %d objects, below the %.2f average: 10.3 step 12 "+
			"wants it richer than average", first.Name.Text(), n, avg)
	}
	for j := range first.Objects {
		if o := first.Objects[j]; !o.IsEmpty() && o.Group() == house.GroupEnemy {
			t.Errorf("the start room %q holds an enemy (%s): 10.3 step 12 forbids it",
				first.Name.Text(), house.ObjectName(o.What))
		}
	}
}

// checkEveryRoomReachable insists the static room graph can get from the start room to
// every other room in the house.
//
// This is not a row of 10.2 and it is emphatically not a rule the corpus obeys. Four of
// the 22 ship a house the model reaches all of -- California or Bust!, Empty House, The
// Asylum Pro and SpacePods -- and Fun House reaches 5 of its 43. internal/profile's
// graph.go argues at length that the number is therefore reported and never linted, and
// docs/IMPROVEMENTS.md 4.1 declined it as a lint check.
//
// It is an assertion *here* because the model's two blind spots are both things these two
// houses do not have. It cannot follow a door a kLightSwitch opens in another room, and it
// cannot know that a link left at `who == 255` was meant to be connected; neither house
// uses a switch that way, and `glidertool house lint` fails on the unlinked transport. So
// in these two files "unreachable" has no innocent explanation left, and an edit that
// seals a room off is a bug rather than a curiosity.
//
// It is also the condition that makes the eccentricity row worth reading. A longest
// shortest-path of 10 over 43 of 43 rooms describes the house; the same number over 5 of
// 43 describes whichever fragment the walk happened to land in.
func checkEveryRoomReachable(t *testing.T, p profile.Profile) {
	t.Helper()
	if p.Reachable == p.Rooms {
		return
	}
	t.Errorf("the walk reaches %d of the %d rooms from room %d; unreachable: %v",
		p.Reachable, p.Rooms, p.Start, p.Unreachable)
}

// logProfile is the bottom of both house tests: every measured row, printed.
//
// The rows above are bounded and these are read, which is not the same job. dark rooms sat
// at 3.9 % inside a 0-4 % band after the Bell Turret stopped being a kRoof room, and the
// band printed green; what caught it was somebody reading the number. `glidertool house
// stats` is the same output for a house that has no test yet.
func logProfile(t *testing.T, tier profile.Tier, p profile.Profile) {
	t.Helper()
	t.Logf("%s tier: %d rooms, %d objects (%.2f/room), %d empty, %d at the ceiling "+
		"(fullest %d), %d enemies, %d prizes (%d stars, %d batteries, %d bands), "+
		"%d distinct codes, %d points, density %.3f",
		tier, p.Rooms, p.Objects, p.PerRoom(p.Objects), p.Empty, p.AtCeiling, p.Fullest,
		p.Enemies, p.Prizes, p.Stars, p.Batteries, p.Bands, p.Kinds, p.Points, p.Density())
	t.Logf("    %d dark %v; %d of %d rooms reachable from %d, furthest %d hops",
		p.Dark, p.DarkRooms, p.Reachable, p.Rooms, p.Start, p.Eccentricity)
}
