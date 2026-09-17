package game

// The game's half of saved games: what CaptureGame puts in a file, what DoSaveGame does to
// the screen while it writes one, and what ResumeSavedGame will and will not accept.
//
// The file format is internal/house's and is tested there; the store is internal/saved's.
// What is tested here is the part that is a reconstruction rather than a transcription --
// the field list -- and the two invariants a caller cannot see: that SavedGame and
// ResumedSavedGame are only ever set together, and that a save is never half applied.
//
// The last test is the stage's acceptance criterion, end to end: a real house is played for
// a hundred frames, saved through the real encoder, and resumed in a *different* World, and
// the two are compared field by field and room by room.

import (
	"bytes"
	"testing"

	"github.com/bwenstar/gliderGo/internal/game/player"
	"github.com/bwenstar/gliderGo/internal/house"
)

// saveWorld is a headless one-room world with a save hook that records rather than writes.
// The house gets a timestamp because that field is the one a resume validates against and a
// zero there would let a transposition hide.
func saveWorld(t *testing.T) (*World, *[]*house.SavedGame) {
	t.Helper()
	h := oneRoomHouse(house.ObjectIsEmpty)
	h.TimeStamp = 0x2A3B4C5D
	w := newTestWorld(h, "", "")
	// A house file records no name of its own -- the name is the file's. See World.HouseName.
	w.HouseName = "Test House"

	var wrote []*house.SavedGame
	w.SaveGame = func(sg *house.SavedGame) error {
		wrote = append(wrote, sg)
		return nil
	}
	return w, &wrote
}

// ------------------------------------------------------------------ the field list

// SaveGame2's seventeen assignments, read back one at a time. Every value below is
// distinguishable from every other, so a field taken from the wrong global shows up as a
// wrong number rather than as another zero.
func TestCaptureGameIsSaveGame2sFieldList(t *testing.T) {
	w, _ := saveWorld(t)

	w.StarsLeft = 9
	w.Score = 31400
	w.Battery = -75 // helium: the counter is signed and negative is not an error
	w.Bands = 6
	w.Foil = 2
	w.Mortals = 3
	w.ShowFoil = true
	w.R.RoomNumber = 0
	w.P1.Dest = player.Rect{Top: 145, Left: 216, Bottom: 165, Right: 264}
	w.P1.Mode = player.GliderBurning
	w.P1.Facing = player.FaceRight

	got := w.CaptureGame()
	want := house.Game{
		Version:      house.SavedGameVersion,
		WasStarsLeft: 9,
		// **The house's stamp, not the clock's.** This is the field OpenSavedGame's
		// second gate compares against the house, so a save stamped with the moment it
		// was written could never be reopened -- which is precisely the bug in the C's
		// other writer, SaveGame (SavedGames.c:320), and it went unnoticed because
		// nothing ever read its output back.
		TimeStamp:   0x2A3B4C5D,
		Where:       house.Point{V: 145, H: 216},
		Score:       31400,
		Energy:      -75,
		Bands:       6,
		RoomNumber:  0,
		GliderState: player.GliderBurning,
		NumGliders:  3,
		Foil:        2,
		Facing:      1,
		ShowFoil:    1,
	}
	if got != want {
		t.Errorf("CaptureGame() =\n\t%+v\nwant\n\t%+v", got, want)
	}

	// The two unused longs are written as zero rather than left alone: SaveGame2 assigns
	// 0L to both, and a port that carried whatever the last house file had there would
	// make its saves depend on which house was loaded before.
	if got.UnusedLong != 0 || got.UnusedLong2 != 0 {
		t.Errorf("unused longs = %d, %d; SaveGame2 writes 0L to both",
			got.UnusedLong, got.UnusedLong2)
	}

	// Facing left is the other byte, and it has to be 0 rather than merely non-1: the
	// resume arm reads `!= 0`, so any other value would come back as facing right.
	w.P1.Facing = player.FaceLeft
	w.ShowFoil = false
	if g := w.CaptureGame(); g.Facing != 0 || g.ShowFoil != 0 {
		t.Errorf("facing left with no foil = %d, %d; want 0, 0", g.Facing, g.ShowFoil)
	}
}

// CaptureSavedGame adds the two things a file needs and a house's embedded block does not.
func TestCaptureSavedGameCarriesTheNameAndEveryRoom(t *testing.T) {
	w, _ := saveWorld(t)
	for len(w.H.Rooms) < 5 {
		w.H.Rooms = append(w.H.Rooms, house.Room{Background: SimpleRoom})
	}
	w.H.Rooms[3].Visited = 1
	w.H.Rooms[2].Objects[7].Data[0] = 0x5A

	sg := w.CaptureSavedGame()
	if sg.Format != house.SavedGameFormat {
		t.Errorf("Format = %d, want %d", sg.Format, house.SavedGameFormat)
	}
	if got := sg.HouseName.Text(); got != "Test House" {
		t.Errorf("HouseName = %q, want %q", got, "Test House")
	}
	if len(sg.Rooms) != len(w.H.Rooms) {
		t.Fatalf("captured %d rooms of %d", len(sg.Rooms), len(w.H.Rooms))
	}
	if sg.Rooms[3].Visited != 1 {
		t.Error("the visited flag was not captured")
	}
	if sg.Rooms[2].Objects[7].Data[0] != 0x5A {
		t.Error("object state was not captured")
	}

	// A snapshot, not a view. The C copies structs; a port that aliased the house's
	// object arrays would write whatever the game did *after* the save.
	w.H.Rooms[2].Objects[7].Data[0] = 0xFF
	if sg.Rooms[2].Objects[7].Data[0] != 0x5A {
		t.Error("the capture aliases the house rather than copying it")
	}
}

// A name longer than the Str63 it goes into is truncated, not refused: these are file names
// and a save the player cannot make is worse than a name they can already see is long.
func TestCaptureSavedGameTruncatesAnImpossibleName(t *testing.T) {
	w, _ := saveWorld(t)
	long := ""
	for len(long) < 200 {
		long += "Slumberland "
	}
	w.HouseName = long
	if got := w.CaptureSavedGame().HouseName.Text(); len(got) != 63 {
		t.Errorf("a %d-byte name became %d bytes, want 63", len(long), len(got))
	}
}

// ------------------------------------------------------------------ who may save

func TestCanSaveGame(t *testing.T) {
	cases := []struct {
		name  string
		setUp func(w *World)
		want  bool
	}{
		{name: "one player, somewhere to write", setUp: func(*World) {}, want: true},
		{name: "two players", setUp: func(w *World) { w.TwoPlayer = true }},
		{name: "the attract-mode demo", setUp: func(w *World) { w.DemoGoing = true }},
		{name: "nowhere to write", setUp: func(w *World) { w.SaveGame = nil }},
	}
	for _, c := range cases {
		w, wrote := saveWorld(t)
		c.setUp(w)
		if got := w.CanSaveGame(); got != c.want {
			t.Errorf("%s: CanSaveGame() = %v, want %v", c.name, got, c.want)
		}
		// And the backstop: DoSaveGame asks the same question, so a host that offered
		// the key anyway still cannot write a save that could not be resumed.
		w.DoSaveGame()
		if got := len(*wrote) > 0; got != c.want {
			t.Errorf("%s: DoSaveGame wrote %d saves, want %v", c.name, len(*wrote), c.want)
		}
	}
}

// ------------------------------------------------------------------ the screen

// DoSaveGame's four screen statements (Input.c:66-70), observed rather than trusted: the
// scoreboard says something different while the save is in flight, the play area comes back
// from the work map, and the board ends up as it started.
func TestDoSaveGameFlipsTheTitleAndPutsTheScreenBack(t *testing.T) {
	w, wrote := saveWorld(t)

	board := func() []byte {
		strip := w.R.V.Screen
		strip.Top = strip.Bottom - ScoreboardTall
		out := make([]byte, 0, int(strip.Wide())*int(strip.Tall()))
		for y := int(strip.Top); y < int(strip.Bottom); y++ {
			out = append(out, w.Main.Pix[y*w.Main.W+int(strip.Left):y*w.Main.W+int(strip.Right)]...)
		}
		return out
	}

	// A distinguishable frame in the work map and something else on screen over it, so
	// that the CopyRectWorkToMain can be told from "nothing happened".
	w.R.Work.Fill(w.R.V.WorkRect, 11)
	w.Main.Fill(w.R.V.House, 22)

	w.RefreshScoreboard(NormalTitleMode)
	before := board()

	presents := 0
	var during []byte
	w.Present = func() { presents++ }
	w.SaveGame = func(sg *house.SavedGame) error {
		// Between the two RefreshScoreboard calls: the title the player is meant to see
		// while a slow disk is being written.
		during = board()
		*wrote = append(*wrote, sg)
		return nil
	}

	w.DoSaveGame()

	if len(*wrote) != 1 {
		t.Fatalf("wrote %d saves, want 1", len(*wrote))
	}
	if bytes.Equal(during, before) {
		t.Error(`the scoreboard did not change while saving; RefreshScoreboard(SavingTitleMode) ` +
			`is what tells the player the key did something`)
	}
	if !bytes.Equal(board(), before) {
		t.Error("the scoreboard was left saying `Saving Game`")
	}
	if presents < 2 {
		t.Errorf("presented %d times; the saving title and the restored screen each need one",
			presents)
	}
	if got := w.Main.Pix[10*w.Main.W+10]; got != 11 {
		t.Errorf("Main at (10,10) = %d, want the work map's 11 -- "+
			"CopyRectWorkToMain(workSrcRect) did not run", got)
	}
}

// A refused save touches nothing at all, which is what lets the host call this without
// asking first: no title flip, no rect copy, no present.
func TestDoSaveGameOnATwoPlayerGameChangesNothing(t *testing.T) {
	w, wrote := saveWorld(t)
	w.TwoPlayer = true
	w.Main.Fill(w.R.V.House, 22)
	presents := 0
	w.Present = func() { presents++ }

	w.DoSaveGame()

	if len(*wrote) != 0 || presents != 0 {
		t.Errorf("wrote %d saves and presented %d times, want 0 and 0", len(*wrote), presents)
	}
	if got := w.Main.Pix[10*w.Main.W+10]; got != 22 {
		t.Errorf("Main at (10,10) = %d, want the 22 that was there", got)
	}
}

// ------------------------------------------------------------------ giving up

// GiveUpGame is DoCommandKey's Q arm: two assignments and a conditional save.
func TestGiveUpGame(t *testing.T) {
	for _, save := range []bool{false, true} {
		w, wrote := saveWorld(t)
		w.Playing, w.Paused = true, true

		w.GiveUpGame(save)

		if w.Playing {
			t.Errorf("save=%v: Playing is still set; `playing = false` ends PlayGame's loop", save)
		}
		if w.Paused {
			t.Errorf("save=%v: Paused is still set; `paused = false` ends the pause the key arrived in", save)
		}
		if got := len(*wrote) > 0; got != save {
			t.Errorf("save=%v: wrote %d saves", save, len(*wrote))
		}
	}

	// The C's guard is around the *question*, not around the save, so a two-player game is
	// never asked -- and a host that asked anyway gets no file rather than an unusable one.
	w, wrote := saveWorld(t)
	w.TwoPlayer = true
	w.Playing = true
	w.GiveUpGame(true)
	if w.Playing || len(*wrote) != 0 {
		t.Errorf("a two-player give-up: Playing=%v, %d saves written", w.Playing, len(*wrote))
	}
}

// ------------------------------------------------------------------ resuming

func TestResumeSavedGameAppliesTheRoomsAndSetsTheBar(t *testing.T) {
	w, _ := saveWorld(t)
	for len(w.H.Rooms) < 4 {
		w.H.Rooms = append(w.H.Rooms, house.Room{Background: SimpleRoom})
	}

	sg := w.CaptureSavedGame()
	sg.Game.Score = 1234
	sg.Game.RoomNumber = 2
	sg.Rooms[1].Visited = 1
	sg.Rooms[3].Objects[5].Data[0] = 0x77

	// Dirt in the house, so that "applied" is a change and not a coincidence.
	w.H.Rooms[1].Visited = 0
	w.H.Rooms[3].Objects[5].Data[0] = 0

	if err := w.ResumeSavedGame(sg); err != nil {
		t.Fatal(err)
	}
	if w.SavedGame != sg.Game {
		t.Errorf("SavedGame = %+v, want %+v", w.SavedGame, sg.Game)
	}
	if !w.ResumedSavedGame {
		t.Error("ResumedSavedGame is not set; a resumed game must not reach the high scores")
	}
	if w.H.Rooms[1].Visited != 1 || w.H.Rooms[3].Objects[5].Data[0] != 0x77 {
		t.Error("the house was not restored from the save")
	}

	// And the bar is real, not just set: TestHighScore is what reads it.
	w.HighScore = func(int32, int16) bool { return true }
	if w.TestHighScore() {
		t.Error("a resumed game qualified for the high scores")
	}
}

// The embedded case: a house's own 40 bytes carry no room snapshot, and resuming them must
// leave the house's object states exactly as the file had them. A save carrying *zero*
// rooms is a different thing and is refused, because that is a room-count mismatch.
func TestResumeSavedGameWithNoRoomSnapshot(t *testing.T) {
	w, _ := saveWorld(t)
	w.H.HasGame = 1
	w.H.SavedGame = house.Game{
		Version: house.SavedGameVersion1, // both shipped saves are 0x0100; see S3
		Score:   999, NumGliders: 2, RoomNumber: 0,
	}
	w.H.Rooms[0].Objects[4].Data[0] = 0x33

	if err := w.ResumeSavedGame(&house.SavedGame{Game: w.H.SavedGame}); err != nil {
		t.Fatal(err)
	}
	if w.SavedGame.Score != 999 {
		t.Errorf("Score = %d, want 999", w.SavedGame.Score)
	}
	if w.H.Rooms[0].Objects[4].Data[0] != 0x33 {
		t.Error("a save with no room snapshot cleared the house's object state")
	}

	// Zero rooms is not the same as no rooms.
	sg := &house.SavedGame{Game: w.H.SavedGame, Rooms: []house.SavedRoom{}}
	if err := w.ResumeSavedGame(sg); err == nil {
		t.Error("a save claiming zero rooms was applied to a one-room house")
	}
}

func TestResumeSavedGameRefusals(t *testing.T) {
	base, _ := saveWorld(t)
	for len(base.H.Rooms) < 3 {
		base.H.Rooms = append(base.H.Rooms, house.Room{Background: SimpleRoom})
	}

	cases := []struct {
		name string
		sg   *house.SavedGame
		w    *World
	}{
		{name: "nothing to resume", sg: nil},
		{name: "no house", sg: &house.SavedGame{}, w: &World{}},
		{name: "room past the end", sg: &house.SavedGame{Game: house.Game{RoomNumber: 3}}},
		{name: "negative room", sg: &house.SavedGame{Game: house.Game{RoomNumber: -1}}},
		{name: "too few rooms", sg: &house.SavedGame{Rooms: make([]house.SavedRoom, 2)}},
		{name: "too many rooms", sg: &house.SavedGame{Rooms: make([]house.SavedRoom, 4)}},
	}
	for _, c := range cases {
		w := c.w
		if w == nil {
			w, _ = saveWorld(t)
			w.H.Rooms = append([]house.Room(nil), base.H.Rooms...)
		}
		if err := w.ResumeSavedGame(c.sg); err == nil {
			t.Errorf("%s: accepted", c.name)
		}
		// Nothing is assigned until everything has been checked, so a refused resume
		// leaves a World that can still start a new game.
		if w.ResumedSavedGame || w.SavedGame != (house.Game{}) {
			t.Errorf("%s: a refused resume left ResumedSavedGame=%v SavedGame=%+v",
				c.name, w.ResumedSavedGame, w.SavedGame)
		}
	}
}

// ------------------------------------------------------------------ end to end

// The stage's acceptance criterion. A real house is played until it has changed -- a room
// entered, a prize taken, a score run up -- then saved through the real encoder, decoded in
// a second process's worth of state, and resumed in a *fresh* World. Everything
// NewGame(ResumeGameMode) restores is compared against what the first game had, and so is
// every byte of every room.
//
// The two Worlds share nothing: separate houses loaded from the same file, separate scenes,
// separate gliders. That is what makes this a test of the *file* rather than of a pointer.
func TestSaveAndResumeRoundTripsARealGame(t *testing.T) {
	const houseName, frames = "Slumberland", 120

	w := playTestWorld(t, houseName, frames)
	w.HouseName = houseName
	var saved []byte
	w.SaveGame = func(sg *house.SavedGame) error {
		b, err := sg.Encode()
		saved = b
		return err
	}
	// Held right for the whole run, so that the game moves: a room entered and a prize or
	// two collected is what makes the object state in the file differ from the house
	// file's own, and the comparison below can then tell them apart.
	//
	// The save is triggered from KeyPoll rather than from Present because that is where the
	// real one happens -- DoCommandKey is called from inside GetInput (Input.c:264) -- and
	// because DoSaveGame presents, so a Present hook that called it would recurse.
	done := false
	w.KeyPoll = func(*player.Glider) player.Keys {
		if w.Frame == frames-1 && !done {
			done = true
			w.DoSaveGame()
		}
		return player.Keys{Right: true}
	}
	w.Present = func() {
		if w.Frame >= frames {
			w.Quitting = true
			w.SwitchedOut = false
		}
	}
	w.NewGame(NewGameMode)

	if len(saved) == 0 {
		t.Fatal("the game never saved")
	}
	sg, err := house.DecodeSavedGame(saved)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got := sg.HouseName.Text(); got != houseName {
		t.Errorf("the save names %q, not %q", got, houseName)
	}

	// The second World: a clean load of the same house, then the resume.
	w2 := playTestWorld(t, houseName, 1)
	w2.HouseName = houseName
	if err := w2.ResumeSavedGame(sg); err != nil {
		t.Fatalf("resume: %v", err)
	}

	// InitGlider's resume arm and SetHouseToSavedRoom, which is all of NewGame that reads
	// the save. Called directly rather than through NewGame so that the comparison is
	// against the moment of the save and not against 120 more frames of play.
	w2.InitGlider(&w2.P1, ResumeGameMode)
	w2.SetHouseToSavedRoom()

	if w2.Score != w.Score || w2.Mortals != w.Mortals || w2.StarsLeft != w.StarsLeft {
		t.Errorf("score/lives/stars resumed as %d/%d/%d, want %d/%d/%d",
			w2.Score, w2.Mortals, w2.StarsLeft, w.Score, w.Mortals, w.StarsLeft)
	}
	if w2.Battery != w.Battery || w2.Bands != w.Bands || w2.Foil != w.Foil ||
		w2.ShowFoil != w.ShowFoil {
		t.Errorf("inventory resumed as batt %d bands %d foil %d show %v, want %d %d %d %v",
			w2.Battery, w2.Bands, w2.Foil, w2.ShowFoil,
			w.Battery, w.Bands, w.Foil, w.ShowFoil)
	}
	if w2.R.RoomNumber != sg.Game.RoomNumber {
		t.Errorf("resumed in room %d, want %d", w2.R.RoomNumber, sg.Game.RoomNumber)
	}
	// The glider's *position*, which is the one field the save keeps and the game rounds
	// to a rect: Where is dest's top left and the rect is rebuilt from it.
	if w2.P1.Dest.Top != sg.Game.Where.V || w2.P1.Dest.Left != sg.Game.Where.H {
		t.Errorf("resumed at (%d,%d), want (%d,%d)",
			w2.P1.Dest.Left, w2.P1.Dest.Top, sg.Game.Where.H, sg.Game.Where.V)
	}
	if w2.P1.Facing != w.P1.Facing {
		t.Errorf("resumed facing %v, want %v", w2.P1.Facing, w.P1.Facing)
	}

	// And the house: every object slot and every visited flag of every room.
	if len(w2.H.Rooms) != len(w.H.Rooms) {
		t.Fatalf("the second house has %d rooms, the first %d", len(w2.H.Rooms), len(w.H.Rooms))
	}
	visited := 0
	for r := range w.H.Rooms {
		if w.H.Rooms[r].Visited != 0 {
			visited++
		}
		if w2.H.Rooms[r].Visited != w.H.Rooms[r].Visited {
			t.Errorf("room %d visited = %d, want %d",
				r, w2.H.Rooms[r].Visited, w.H.Rooms[r].Visited)
		}
		if w2.H.Rooms[r].Objects != w.H.Rooms[r].Objects {
			t.Errorf("room %d's objects did not round-trip", r)
		}
	}
	if visited == 0 {
		t.Error("no room was marked visited, so this proved nothing about room state")
	}
}
