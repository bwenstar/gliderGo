package shell

// The shell's half of "Open Saved Game...": whether the row is there, whether it is live,
// what it says when it is not, what it says it would resume, and what it hands to Play.
//
// What a saved game *is* belongs to internal/house, where it goes belongs to internal/saved,
// and what resuming does to a world belongs to internal/game. All three are tested there.
// This file is about the four lines of the menu.

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"glidergo/internal/platform"
	"glidergo/internal/saved"
)

// savedInfo is a header for a host hook to hand back. The numbers are QueryResumeGame's own
// example -- three gliders, 8600 points (Menu.c:710-758 filled its dialogue from smallGame's
// numGliders and score) -- so a test that reads the summary is reading the original's wording
// with the original's kind of content in it.
func savedInfo(house string) saved.Info {
	return saved.Info{
		HouseName: house,
		Version:   0x0200,
		Score:     8600,
		Gliders:   3,
		Room:      6,
		Rooms:     12,
	}
}

// savedHost gives a shell a Saved hook that answers with one thing for every house, and
// counts the questions.
func savedHost(s *Shell, info saved.Info, err error) *int {
	n := 0
	s.host.Saved = func(House) (saved.Info, error) {
		n++
		return info, err
	}
	return &n
}

// rowOfMenu finds a menu row by label, with its index.
func rowOfMenu(t *testing.T, s *Shell, label string) (int, item) {
	t.Helper()
	m := s.menu()
	for i := range m {
		if m[i].label == label {
			return i, m[i]
		}
	}
	t.Fatalf("the menu has no %q row: %v", label, labels(s))
	return -1, item{}
}

func labels(s *Shell) []string {
	var out []string
	for _, it := range s.menu() {
		out = append(out, it.label)
	}
	return out
}

// ------------------------------------------------------------------------- the row itself

// MENU 129's third item, on the key it carried in 1994. The order matters as much as the
// presence: a player who knows the original knows Open Saved Game... sits under the two
// New Game rows and above Load House..., and there is no reason to move it.
func TestTheSavedGameRowIsWhereMENU129PutIt(t *testing.T) {
	s, _ := shellOver(t, []string{"Slumberland"})
	savedHost(s, savedInfo("Slumberland"), nil)

	at, it := rowOfMenu(t, s, "Open Saved Game...")
	if at != 2 {
		t.Errorf("Open Saved Game... is row %d, want row 2 -- MENU 129 has it third, after "+
			"New Game and Two Player Game: %v", at, labels(s))
	}
	if it.key != platform.KeyO {
		t.Errorf("Open Saved Game... is on %v, want O (it was Command-O)", it.key)
	}
	if got := labels(s)[:4]; got[0] != "New Game" || got[1] != "Two Player Game" ||
		got[3] != "Load House..." {
		t.Errorf("the top of the menu is %v, want MENU 129's order", got)
	}
}

// ------------------------------------------------------------------------------- resuming

// The row is live when there is a save, it says what resuming would get, and choosing it
// asks Play for a resume of exactly one glider.
func TestChoosingTheRowAsksPlayForAResume(t *testing.T) {
	s, f := shellOver(t, []string{"Slumberland"})
	savedHost(s, savedInfo("Slumberland"), nil)

	at, it := rowOfMenu(t, s, "Open Saved Game...")
	if !it.ok {
		t.Fatalf("the row is unavailable with a save present: %q", it.why)
	}
	if !strings.Contains(it.note, "3 gliders, 8600 points, room 6") {
		t.Errorf("the row's note is %q; it should carry the save's own summary", it.note)
	}

	// What the band says *while* the game loads, captured from inside Play: by the time
	// Play returns the line is the score line, and the loading line is the one thing a
	// player sees between pressing Return and the house appearing.
	var loading string
	base := s.host.Play
	s.host.Play = func(c Choice) (Outcome, error) {
		loading = s.status()
		return base(c)
	}

	s.sel = at
	before := f.presents
	s.event(key(platform.KeyReturn)[0])

	if !strings.Contains(loading, "resuming") {
		t.Errorf("the band said %q while the game loaded, want it to say the game is being "+
			"resumed rather than started", loading)
	}
	if f.presents == before {
		t.Error("the loading line was never presented, so nobody sees it")
	}
	if len(f.plays) != 1 {
		t.Fatalf("Play was called %d times, want once", len(f.plays))
	}
	c := f.plays[0]
	if !c.Resume {
		t.Error("Play was asked for a new game, not a resume")
	}
	if c.TwoPlayer {
		// A gameType holds one glider's room, position, mode and facing. There is nowhere
		// in the format for a second player, so the two can never be combined.
		t.Error("a resume was asked for as a two-player game")
	}
	if c.House.Name != "Slumberland" {
		t.Errorf("Play got the house %q", c.House.Name)
	}
	if !strings.Contains(s.msg, "score") {
		t.Errorf("the status line after the game is %q, want the outcome", s.msg)
	}
}

// And the accelerator alone does it, from wherever the cursor happens to be.
func TestOResumesFromAnyRow(t *testing.T) {
	s, f := shellOver(t, []string{"Slumberland"}, key(platform.KeyO))
	savedHost(s, savedInfo("Slumberland"), nil)
	s.sel = 0

	if err := s.Run(); err != nil {
		t.Fatal(err)
	}
	if len(f.plays) != 1 || !f.plays[0].Resume {
		t.Errorf("O gave %v, want one resume", f.plays)
	}
}

// ---------------------------------------------------------------------- and when it cannot

// Three ways there is nothing to resume, three different things to tell the player, and in
// every one of them the row stays visible and greyed rather than vanishing. An item that is
// there only sometimes is an item nobody learns is there.
func TestTheRowSaysWhyWhenThereIsNothingToResume(t *testing.T) {
	for _, tc := range []struct {
		name string
		hook bool
		err  error
		want string
	}{
		{
			name: "no store at all",
			hook: false,
			want: "nowhere to keep saved games",
		},
		{
			name: "not saved yet",
			hook: true,
			err:  os.ErrNotExist,
			want: "no saved game for Slumberland",
		},
		{
			name: "saved and broken",
			hook: true,
			err:  fmt.Errorf("%s: %w", "Slumberland.save", errors.New("truncated")),
			want: "cannot be read",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, f := shellOver(t, []string{"Slumberland"})
			if tc.hook {
				savedHost(s, saved.Info{}, tc.err)
			}

			_, it := rowOfMenu(t, s, "Open Saved Game...")
			if it.ok {
				t.Fatal("the row is live with nothing to resume")
			}
			if !strings.Contains(it.why, tc.want) {
				t.Errorf("the row's reason is %q, want it to say %q", it.why, tc.want)
			}
			if it.note != "" {
				t.Errorf("an unavailable row has the note %q; the note is the offer", it.note)
			}

			// Pressing the key says the reason instead of doing nothing, which is what the
			// original's greyed-out item did.
			s.event(key(platform.KeyO)[0])
			if len(f.plays) != 0 {
				t.Errorf("O started a game anyway: %v", f.plays)
			}
			if !strings.Contains(s.msg, tc.want) {
				t.Errorf("the status line after O is %q, want the reason", s.msg)
			}
			// The cursor is now on the row, and the status band must still show the reason
			// rather than being taken over by an empty note.
			if got := s.status(); got != s.msg {
				t.Errorf("status() is %q with the cursor on an unavailable row, want the "+
					"reason %q", got, s.msg)
			}

			// A broken save also names the file, so the player can move it out of the way.
			if tc.name == "saved and broken" && !strings.Contains(it.why, "Slumberland.save") {
				t.Errorf("the reason is %q; the store's errors name the file and it belongs "+
					"in the row", it.why)
			}
		})
	}
}

// With no houses the row needs no reason of its own: the shell's own answer -- there are no
// houses, here is where they go -- is the more useful one, and it is what every other row
// gets in that state.
func TestResumeWithNoHousesExplainsTheLibrary(t *testing.T) {
	s, f := shellOver(t, nil, key(platform.KeyO))
	savedHost(s, savedInfo("Slumberland"), nil)

	if err := s.Run(); err != nil {
		t.Fatal(err)
	}
	if len(f.plays) != 0 {
		t.Errorf("a resume was started with no houses: %v", f.plays)
	}
	if !strings.Contains(s.msg, "no houses in") {
		t.Errorf("the status line is %q, want the library's own explanation", s.msg)
	}
}

// ------------------------------------------------------------------------------ the cache

// The menu is rebuilt every frame and the status band asks a second time, so the host must
// be asked once per house per visit -- not sixty times a second. A save is up to sixty
// kilobytes of object state; the header alone is worth reading, and only once.
func TestTheSaveIsAskedForOncePerHouse(t *testing.T) {
	s, _ := shellOver(t, []string{"Amy", "Beth"})
	asked := []string{}
	s.host.Saved = func(h House) (saved.Info, error) {
		asked = append(asked, h.Rel)
		return savedInfo(h.Name), nil
	}

	for i := 0; i < 3; i++ {
		s.menu()
		s.status()
		s.Draw()
	}
	if len(asked) != 1 {
		t.Errorf("the host was asked %d times for one house: %v", len(asked), asked)
	}

	// A different house is a different entry, keyed on the path like the score cache.
	s.sel = 0
	s.mode = modeHouses
	s.pick = 1
	s.commitPick()
	s.menu()
	if len(asked) != 2 || asked[1] != s.lib.Houses[1].Rel {
		t.Errorf("after selecting the second house the host was asked for %v, want a second "+
			"call for %s", asked, s.lib.Houses[1].Rel)
	}
}

// A game can write a save -- or resume one and finish it -- so every answer this shell is
// holding is suspect the moment a game ends. Including one that ended badly.
func TestTheSaveCacheIsDroppedAfterAGame(t *testing.T) {
	for _, name := range []string{"ok", "error"} {
		s, f := shellOver(t, []string{"Slumberland"})
		if name == "error" {
			f.err = errors.New("the house went away mid-game")
		}
		calls := savedHost(s, savedInfo("Slumberland"), nil)

		s.menu()
		if *calls != 1 {
			t.Fatalf("%s: %d calls before the game, want 1", name, *calls)
		}
		s.play(false)
		s.menu()
		if *calls != 2 {
			t.Errorf("%s: %d calls after a game, want the save to have been looked at again",
				name, *calls)
		}
	}
}

// -------------------------------------------------------------------------- the band line

// The note is drawn while the cursor is on the row and gone the moment it leaves, and it
// never overwrites the line the last game left -- which is the whole reason it is read at
// draw time instead of being pushed into msg when the cursor moves.
func TestTheNoteShowsOnlyWhileTheCursorIsOnTheRow(t *testing.T) {
	s, _ := shellOver(t, []string{"Slumberland"})
	savedHost(s, savedInfo("Slumberland"), nil)
	s.msg = "Slumberland -- score 4200, 3 stars left"
	kept := s.msg

	at, _ := rowOfMenu(t, s, "Open Saved Game...")
	s.sel = at
	if got := s.status(); !strings.Contains(got, "resume Slumberland") {
		t.Errorf("status() on the row is %q, want the offer", got)
	}
	if s.msg != kept {
		t.Errorf("msg is now %q; the note must not be written into it", s.msg)
	}

	s.sel = 0
	if got := s.status(); got != kept {
		t.Errorf("status() off the row is %q, want the message back: %q", got, kept)
	}

	// And it is only the splash's business: the picker and the settings screen have their
	// own status lines and their own selections.
	s.sel = at
	s.mode = modeHouses
	if got := s.status(); got != kept {
		t.Errorf("status() in the picker is %q, want the message: %q", got, kept)
	}
}

// The note reaches the screen, not just status(): with no message at all the band is still
// written while the cursor is on a live row.
func TestTheNoteIsDrawnInTheBand(t *testing.T) {
	s, f := shellOver(t, []string{"Slumberland"})
	savedHost(s, savedInfo("Slumberland"), nil)
	s.msg = ""
	at, _ := rowOfMenu(t, s, "Open Saved Game...")
	s.sel = at

	for i := range f.scr.Pix {
		f.scr.Pix[i] = 0
	}
	s.Draw()
	if !messageIsOnScreen(f.scr) {
		t.Error("the status band is empty with the cursor on the saved-game row")
	}
}

// ------------------------------------------------------------------------------- geometry

// The panel the menu is drawn in has to grow downwards without reaching the house name, and
// its rows have to stay inside it. Both ends of that, because adding a row -- which is what
// this stage did -- is the thing that breaks it.
//
// The lower bound is why menuBottom is gone: at seven rows a constant 264 left the last two
// baselines below the box. The upper bound is where the derived bottom stops being able to
// help, and that is the row this test refuses to let a ninth item cross.
func TestMenuPanelClearsTheHouseLabel(t *testing.T) {
	// render's fontAscent, which is not exported: a glyph with its baseline at v inks from
	// row v-7 down (internal/render/font.go's const block).
	const fontAscent = 7

	s, _ := shellOver(t, []string{"Slumberland"})
	savedHost(s, savedInfo("Slumberland"), nil)
	n := len(s.menu())
	if n < 8 {
		t.Fatalf("the menu has %d rows; this test is about the eight it grew to", n)
	}

	r := menuRect(n)

	// panel() dims eight rows past the bottom, so the halo's last row is Bottom+7. The
	// label's first row is its baseline less the ascent.
	if halo, name := int(r.Bottom)+7, houseLabelV-fontAscent; halo >= name {
		t.Errorf("the panel's halo reaches row %d and the house name starts at row %d: "+
			"%d rows at a pitch of %d no longer fit above the label. Either the row is not "+
			"worth the name or the menu needs two columns", halo, name, n, menuPitch)
	}

	// The last row's deepest ink -- a descender at 2*scale-1 below the baseline, plus
	// shadow()'s black pixel one scale under that -- against the inner frame, which
	// panel() draws three rows above the bottom.
	last := menuTop + menuFirst + (n-1)*menuPitch
	if ink, frame := last+3*menuScale-1, int(r.Bottom)-3; ink >= frame {
		t.Errorf("the last row inks down to row %d and the panel's inner frame is at row %d; "+
			"menuFoot is %d and needs to be larger", ink, frame, menuFoot)
	}

	// The selection bar of the first row stays inside the frame too: it is the tallest
	// thing in the panel and the one that would touch the top edge first.
	if bar, frame := menuTop+menuFirst-menuRise, menuTop+3; bar <= frame {
		t.Errorf("the first row's selection bar starts at row %d, on the inner frame at %d; "+
			"menuFirst is %d and needs to be larger", bar, frame, menuFirst)
	}

	// One row is the floor -- there is no such menu, but menuRect must not return an
	// inverted rectangle if there ever is one -- and each row past it moves the bottom
	// down by exactly one pitch.
	if menuRect(0) != menuRect(1) {
		t.Error("menuRect(0) should be the one-row panel, not an empty or inverted one")
	}
	if grew := menuRect(n+1).Bottom - r.Bottom; int(grew) != menuPitch {
		t.Errorf("a ninth row moved the bottom by %d, want one pitch of %d", grew, menuPitch)
	}

	// And the label survives the panel on a real draw, which is the check that does not
	// depend on any of the arithmetic above being right.
	s.Draw()
	if !inkNear(s.host.Screen, houseLabelH, houseLabelV, label) {
		t.Error("the house name is gone from the splash: the menu panel is drawn over it")
	}
}
