package shell

// The Race screen, from the R on the title screen to the Choice that comes out of it.
//
// What these test is the arrangement and nothing else: no socket is opened here and none could
// be, because the shell has no net import (see race.go). So the assertions are all about the
// Choice the host is handed and about the screen's own two rows -- which is the whole of what
// this screen decides, and the reason a screen that arranges a network game is testable with no
// network at all.

import (
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/platform"
	"github.com/bwenstar/gliderGo/internal/render"
)

// typed is one keystroke as a real backend delivers it: the key, and the character the layout
// put on it. Both x11 and win32 fill Text in (internal/platform), so this is the ordinary path
// through scores.Field and not the KeyChar fallback.
func typed(k platform.Key, text string) []platform.Event {
	return []platform.Event{{Kind: platform.EventKeyDown, Key: k, Text: text}}
}

// typeString is an address typed one character at a time, with no key attached: the field reads
// Text first, and what a test about addresses cares about is the string that comes out.
func typeString(s *Shell, text string) {
	for _, r := range text {
		s.event(platform.Event{Kind: platform.EventKeyDown, Text: string(r)})
	}
}

func TestRaceOpensOnRAndEscapeGoesBack(t *testing.T) {
	s, f := shellOver(t, []string{"Slumberland"}, key(platform.KeyR), key(platform.KeyEscape))
	if err := s.Run(); err != nil {
		t.Fatal(err)
	}
	if s.mode != modeSplash {
		t.Errorf("mode is %v after R then Escape, want the title screen back", s.mode)
	}
	if len(f.plays) != 0 {
		t.Errorf("opening the Race screen played %v; it should only arrange one", f.plays)
	}
}

func TestTheMenuHasARaceRow(t *testing.T) {
	s, _ := shellOver(t, []string{"Slumberland"})
	at := -1
	for i, it := range s.menu() {
		if it.label == "Race..." {
			at = i
		}
	}
	if at < 0 {
		t.Fatal("no Race row on the menu")
	}
	// Past the end of MENU 129's own list and above the rows that do not start a game, which
	// is the placement menu() argues for: the original's four keep their 1994 order, and the
	// port's own way of starting a game goes directly under them.
	load := -1
	for i, it := range s.menu() {
		if it.label == "Load House..." {
			load = i
		}
	}
	if load < 0 || at != load+1 {
		t.Errorf("Race is row %d and Load House... -- MENU 129's last -- is row %d; the race "+
			"belongs on the first row past the original's own", at, load)
	}

	// And it carries a note, which matters more for this row than for any other: it is the
	// one item on the menu whose name does not say what it does, because the original has no
	// word for it. The status band is where that sentence goes.
	s.sel = at
	if note := s.menu()[at].note; note == "" {
		t.Error("the Race row has no note, so the status band says nothing about the one row " +
			"whose label cannot explain itself")
	} else if got := s.status(); got != note {
		t.Errorf("the status band says %q with the cursor on Race, want its note %q", got, note)
	}
}

// The row is greyed rather than hidden with no house, for the reason every other row is: a menu
// that changes length tells a player less than one that says why.
func TestRaceNeedsAHouse(t *testing.T) {
	s, f := shellOver(t, nil, key(platform.KeyR))
	if err := s.Run(); err != nil {
		t.Fatal(err)
	}
	if s.mode != modeSplash {
		t.Errorf("mode is %v with no house; R should have refused", s.mode)
	}
	// why()'s own words: with an empty library the reason is the library and not the row, so
	// what this asserts is that something was said, the way the High Scores row's test does.
	if s.msg == "" {
		t.Error("pressing R with no houses said nothing")
	}
	if len(f.plays) != 0 {
		t.Error("a race was arranged with no house to fly")
	}
}

func TestRaceArrowsMoveBetweenTheTwoRows(t *testing.T) {
	s, _ := shellOver(t, []string{"Slumberland"})
	s.openRace()
	if s.raceSel != raceRowHost {
		t.Fatalf("the screen opened on row %d, want the Host row", s.raceSel)
	}
	for i, k := range []platform.Key{platform.KeyDown, platform.KeyUp, platform.KeyDown} {
		s.event(key(k)[0])
		if want := (i + 1) % 2; s.raceSel != want {
			t.Errorf("after %d arrows the cursor is on row %d, want %d -- two rows, so both "+
				"arrows swap them", i+1, s.raceSel, want)
		}
	}
}

func TestRaceHostsOnReturn(t *testing.T) {
	s, f := shellOver(t, []string{"Slumberland"})
	s.openRace()
	s.event(key(platform.KeyReturn)[0])

	if len(f.plays) != 1 {
		t.Fatalf("Return on the Host row arranged %d games, want 1", len(f.plays))
	}
	c := f.plays[0]
	if !c.Race.Host || c.Race.Join != "" {
		t.Errorf("the Choice carried %+v, want Host with no address", c.Race)
	}
	if c.House.Name != "Slumberland" {
		t.Errorf("the race is in %q, want the house the title screen was showing", c.House.Name)
	}
	if c.TwoPlayer || c.Resume {
		t.Errorf("the Choice is also %+v; a race is neither of those (see raceRefusals)", c)
	}
}

func TestRaceJoinsTheAddressThatWasTyped(t *testing.T) {
	s, f := shellOver(t, []string{"Slumberland"})
	s.openRace()
	s.event(key(platform.KeyDown)[0])
	typeString(s, "192.168.1.5:1138 ")
	s.event(key(platform.KeyReturn)[0])

	if len(f.plays) != 1 {
		t.Fatalf("Return on the Join row arranged %d games, want 1", len(f.plays))
	}
	// Trimmed, because a trailing space is what a person who typed one more character than
	// they meant to leaves behind, and net.Dial would report it as part of the host name.
	if got := f.plays[0].Race; got.Host || got.Join != "192.168.1.5:1138" {
		t.Errorf("the Choice carried %+v, want the address with the space off the end", got)
	}
}

// Typing on the Host row moves the cursor to the Join row, because there is nothing else on this
// screen an address could be for. This is the behaviour race.go keys on the contents changing
// rather than on scores.Field's Action, so the test is worth having twice over.
func TestTypingAnAddressMovesToTheJoinRow(t *testing.T) {
	s, _ := shellOver(t, []string{"Slumberland"})
	s.openRace()
	s.event(typed(platform.Key1, "1")[0])
	if s.raceSel != raceRowJoin {
		t.Errorf("a digit typed on the Host row left the cursor on row %d, want the Join row",
			s.raceSel)
	}
	if got := s.addrField().Text(); got != "1" {
		t.Errorf("the field holds %q, want the character that was typed", got)
	}

	// And a key that types nothing does not move it, which is the half that would break if
	// the Action were trusted: Field.Event answers ActionTyped for these too.
	s.raceSel = raceRowHost
	for _, k := range []platform.Key{platform.KeyLeft, platform.KeyRight, platform.KeyF1} {
		s.event(key(k)[0])
		if s.raceSel != raceRowHost {
			t.Errorf("%v moved the cursor to the Join row; it types nothing", k)
		}
	}
}

func TestRaceRefusesAnEmptyAddress(t *testing.T) {
	s, f := shellOver(t, []string{"Slumberland"})
	s.openRace()
	s.event(key(platform.KeyDown)[0])
	s.event(key(platform.KeyReturn)[0])

	if len(f.plays) != 0 {
		t.Fatalf("an empty address arranged %v; there is nothing to dial", f.plays)
	}
	if s.mode != modeRace {
		t.Errorf("mode is %v after the refusal, want the screen still up so it can be typed into",
			s.mode)
	}
	if !strings.Contains(s.msg, "address") {
		t.Errorf("the status band says %q, want it to ask for an address", s.msg)
	}
}

// The address survives leaving the screen, because the first attempt at a race is usually the
// wrong one and retyping it is the part a player should not have to do twice. See addrField.
func TestRaceKeepsTheAddressBetweenVisits(t *testing.T) {
	s, _ := shellOver(t, []string{"Slumberland"})
	s.openRace()
	typeString(s, "10.0.0.7")
	s.event(key(platform.KeyEscape)[0])
	if s.mode != modeSplash {
		t.Fatalf("Escape left the shell on %v", s.mode)
	}
	s.openRace()
	if got := s.addrField().Text(); got != "10.0.0.7" {
		t.Errorf("the field holds %q on the way back in, want the address that was typed", got)
	}
}

// Tab selects the whole field, which is the one thing on this screen borrowed wholesale from
// internal/scores' entry dialog: it is how a wrong address is replaced rather than erased.
func TestTabSelectsTheWholeAddress(t *testing.T) {
	s, _ := shellOver(t, []string{"Slumberland"})
	s.openRace()
	typeString(s, "10.0.0.7")
	s.event(key(platform.KeyTab)[0])
	if !s.addrField().Selected() {
		t.Error("Tab did not select the address")
	}
	typeString(s, "1")
	if got := s.addrField().Text(); got != "1" {
		t.Errorf("typing over a selected field left %q, want just the new character", got)
	}
}

// Shift is tracked by the shell itself, because there is no Host hook for a modifier and the
// address field needs one: a colon is shift-semicolon on the layout KeyChar falls back to.
func TestShiftReachesTheAddressField(t *testing.T) {
	s, _ := shellOver(t, []string{"Slumberland"})
	s.openRace()
	s.event(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyShift})
	s.event(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeySemicolon})
	if got := s.addrField().Text(); got != ":" {
		t.Errorf("shift-semicolon typed %q, want a colon -- a port cannot be typed without one", got)
	}
	s.event(platform.Event{Kind: platform.EventKeyUp, Key: platform.KeyShift})
	s.event(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeySemicolon})
	if got := s.addrField().Text(); got != ":;" {
		t.Errorf("the field holds %q; the shift should have been let go of", got)
	}
}

func TestShowOpensTheRaceScreen(t *testing.T) {
	s, _ := shellOver(t, []string{"Slumberland"})
	if err := s.Show("race"); err != nil {
		t.Fatalf("Show: %v", err)
	}
	if s.mode != modeRace {
		t.Errorf("mode is %v, want the Race screen", s.mode)
	}
	if err := s.Show("radio"); err == nil {
		t.Error("an unknown screen name should be an error")
	} else if !strings.Contains(err.Error(), "race") {
		t.Errorf("the error is %q, want the list of screens to include race", err)
	}
}

// The field draws at scale 1 when the address is too long for scale 2, which is the one thing
// this box must never do differently: an address that is cut in half cannot be checked against
// the other machine's screen, and checking it is the box's whole purpose.
func TestALongAddressStaysInsideItsBox(t *testing.T) {
	s, f := shellOver(t, []string{"Slumberland"})
	s.openRace()
	s.event(key(platform.KeyDown)[0])
	typeString(s, strings.Repeat("a", raceAddrMax+10))

	got := s.addrField().Text()
	if len(got) != raceAddrMax {
		t.Errorf("the field took %d characters, want it capped at %d", len(got), raceAddrMax)
	}
	if w := render.StringWidth(got); w > raceFieldRight-raceFieldLeft-2*raceFieldPad {
		t.Errorf("the longest address the field holds is %d pixels wide and the box has %d; "+
			"raceAddrMax and the box disagree", w, raceFieldRight-raceFieldLeft-2*raceFieldPad)
	}

	s.Draw()
	// Nothing outside the box, checked on the two columns the frame occupies: a scale-2
	// address this long would have run straight through the right-hand one.
	for v := raceFieldTop + 1; v < raceFieldBottom-1; v++ {
		if px := f.scr.Pix[int(v)*screenWide+int(raceFieldRight)+1]; px == cream {
			t.Fatalf("cream ink at row %d just outside the address box's right edge", v)
		}
	}
}

func TestTheRaceScreenDrawsWithoutArt(t *testing.T) {
	s, f := shellOver(t, []string{"Slumberland"})
	s.openRace()
	for i := range f.scr.Pix {
		f.scr.Pix[i] = 0
	}
	s.Draw()

	if !messageIsOnScreen(f.scr) {
		t.Error("nothing was drawn in the status band")
	}
	if n := count(f.scr, cream); n < 200 {
		t.Errorf("only %d cream pixels on screen; the panel and its text are missing", n)
	}
	// The house, because the other machine has to open the same one and nothing in the
	// handshake will say which it was (race.go's header). Centred, so the probe is the
	// centred line's own left edge rather than the panel's.
	line := fit("the other machine opens Slumberland too", raceRight-raceLeft-40, 1)
	h := int(raceLeft) + (int(raceRight-raceLeft)-int(render.StringWidth(line)))/2
	if !inkNear(f.scr, h, raceHouseV, cream) {
		t.Error("the line naming the house is missing from the Race screen")
	}
}
