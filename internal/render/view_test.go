package render

// The screen geometry contract items 4 and 19 of docs/ORIGINAL_GAME.md §19 rest on.
//
// Everything else in this package derives its coordinates from the View rather than stating
// them, which is the right way round -- but it means the derivation itself had no test. Change
// `(v.Screen.Tall() - kTileHigh) / 2` to `(v.House.Tall() - kTileHigh) / 2` and every test in
// the repository still passes except the pixel corpus, which reports that 600 frame hashes
// moved and does not say why. This file is the test that says why.
//
// The one to keep hold of is `playOriginV` coming from **`thisMac.screen`, not `houseRect`**
// (`InterfaceInit.c:203-204`, two lines below the `houseRect.bottom -= kScoreboardTall` that
// makes the two differ). The rooms are centred on the whole display and the scoreboard then
// overlays their bottom 20 pixels; it does not push them up. So the room block sits 10 px lower
// than a reading of the C that used `houseRect` would put it, and *every* hard-coded object y in
// the game -- kFloorVentTop 305, kShadowTop 306 -- was authored against that.

import "testing"

func TestTheOriginComesFromTheScreenAndNotTheHouseRect(t *testing.T) {
	v := DefaultView()

	// The two candidate derivations, spelled out so the failure message can show both.
	fromScreen := (v.Screen.Tall() - kTileHigh) / 2 // 79
	fromHouse := (v.House.Tall() - kTileHigh) / 2   // 69

	if fromScreen == fromHouse {
		t.Fatal("the screen and the house rect give the same origin at this size, " +
			"so this test cannot tell them apart any more")
	}
	if v.OriginV != fromScreen {
		t.Errorf("OriginV = %d, want %d (from Screen). %d is the houseRect reading, which "+
			"puts every room %d px too high and moves every object with it "+
			"(InterfaceInit.c:203-204)",
			v.OriginV, fromScreen, fromHouse, fromScreen-fromHouse)
	}

	// Horizontally the two agree -- the scoreboard costs height, not width -- so the H
	// derivation is pinned by value instead.
	if want := int16((640 - kRoomWide) / 2); v.OriginH != want {
		t.Errorf("OriginH = %d, want %d", v.OriginH, want)
	}

	// The 10 px is the number the analysis quotes, and §19 row 4's point is that the
	// scoreboard's 20 and the menu bar's 20 are unrelated quantities that happen to be
	// equal. Stating it as `kScoreboardTall / 2` rather than `10` is what keeps that true:
	// if the board's height ever changed, this offset would follow it and the menu bar's
	// would not.
	if got, want := v.OriginV-fromHouse, int16(kScoreboardTall/2); got != want {
		t.Errorf("the room sits %d px below the houseRect centre, want %d", got, want)
	}
}

func TestARoomIsFiveHundredAndTwelveByThreeHundredAndTwentyTwo(t *testing.T) {
	v := DefaultView()

	// The central room, by value. Unscaled: the port has no magnification inside the
	// simulation at all -- `-scale` magnifies the finished 640x480 image in the host, after
	// every coordinate in this package has already been computed at 1:1.
	c := v.LocalRoomsDest[kCentralRoom]
	if c.Wide() != kRoomWide || c.Tall() != kTileHigh {
		t.Errorf("the central room is %dx%d, want %dx%d",
			c.Wide(), c.Tall(), kRoomWide, kTileHigh)
	}
	if c.Left != v.OriginH || c.Top != v.OriginV {
		t.Errorf("the central room is cornered at (%d,%d), want the origin (%d,%d)",
			c.Left, c.Top, v.OriginH, v.OriginV)
	}

	// The 3x3 neighbourhood, each slot at its own displacement from the centre. Vertically
	// the step is kVertLocalOffset and not kTileHigh: the header calls them different
	// things and the shipped values are equal, so a test that used kTileHigh here would
	// pass for the wrong reason and stop failing if only one of them ever changed.
	for _, c := range []struct {
		slot   int
		dh, dv int16
	}{
		{kCentralRoom, 0, 0},
		{kNorthRoom, 0, -kVertLocalOffset},
		{kNorthEastRoom, kRoomWide, -kVertLocalOffset},
		{kEastRoom, kRoomWide, 0},
		{kSouthEastRoom, kRoomWide, kVertLocalOffset},
		{kSouthRoom, 0, kVertLocalOffset},
		{kSouthWestRoom, -kRoomWide, kVertLocalOffset},
		{kWestRoom, -kRoomWide, 0},
		{kNorthWestRoom, -kRoomWide, -kVertLocalOffset},
	} {
		got := v.LocalRoomsDest[c.slot]
		want := Offset(SetRect(0, 0, kRoomWide, kTileHigh), v.OriginH+c.dh, v.OriginV+c.dv)
		if got != want {
			t.Errorf("local room slot %d is %v, want %v", c.slot, got, want)
		}
		// OffsetRectRoomRelative is the same arithmetic reached from the other side, and
		// the two disagreeing is a real class of bug: it is what every object rect in a
		// neighbouring room goes through, while LocalRoomsDest is what the background art
		// goes through. Objects one room over would draw 322 px from their floor.
		if r := v.OffsetRectRoomRelative(SetRect(0, 0, kRoomWide, kTileHigh), c.slot); r != want {
			t.Errorf("OffsetRectRoomRelative for slot %d is %v, want %v", c.slot, r, want)
		}
	}
}

func TestTheHouseRectIsTheScreenLessTheScoreboard(t *testing.T) {
	v := DefaultView()

	if v.House.Tall() != v.Screen.Tall()-kScoreboardTall {
		t.Errorf("House is %d tall for a %d-tall screen, want %d",
			v.House.Tall(), v.Screen.Tall(), v.Screen.Tall()-kScoreboardTall)
	}
	if v.House.Wide() != v.Screen.Wide() {
		t.Errorf("House is %d wide, want the screen's %d -- the board costs height only",
			v.House.Wide(), v.Screen.Wide())
	}

	// The three rects that are numerically identical at 640x480 and are still three fields
	// (see View.JustRoomsRect). Asserting they agree here is not redundancy: it is the
	// statement that at *this* resolution a call site cannot be caught out by having picked
	// the wrong one, which is why the corpus at 640x480 cannot be the test that finds it.
	if v.WorkRect != v.BackRect || v.WorkRect != v.JustRoomsRect {
		t.Errorf("work %v, back %v, justRooms %v: all three are ZeroCorner(houseRect) "+
			"at this size", v.WorkRect, v.BackRect, v.JustRoomsRect)
	}
	if want := ZeroCorner(v.House); v.WorkRect != want {
		t.Errorf("WorkRect %v, want %v", v.WorkRect, want)
	}

	// The clamp, which nothing in a 640x480 build exercises. A 4K monitor gets nine rooms
	// and a wasted border, not a bigger offscreen map -- and the two limits are checked
	// independently, so a wide-and-short screen clamps only its width.
	big := NewView(3840, 2160)
	if big.House.Right != kMaxViewWidth || big.House.Bottom != kMaxViewHeight {
		t.Errorf("a 3840x2160 screen gives a house rect of %v, want it clamped to %dx%d",
			big.House, kMaxViewWidth, kMaxViewHeight)
	}
	if big.OriginV != (2160-kTileHigh)/2 {
		t.Error("the clamp moved the origin; playOriginH/V come from the unclamped screen " +
			"(InterfaceInit.c:203), so a huge display centres the rooms on the glass and " +
			"leaves the offscreen maps behind")
	}
	wide := NewView(2560, 480)
	if wide.House.Right != kMaxViewWidth || wide.House.Bottom != 480-kScoreboardTall {
		t.Errorf("a 2560x480 screen gives %v, want the width clamped and the height not",
			wide.House)
	}
}
