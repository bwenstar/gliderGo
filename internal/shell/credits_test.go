package shell

// The credits screen. What is worth checking here is not that the names are right -- that is
// internal/credits' job, and it checks them against GliderPRO/README.md -- but that all of
// them reach a screen a player can get to, and that none of them run off the edge of it.

import (
	"strings"
	"testing"

	"glidergo/internal/credits"
	"glidergo/internal/platform"
	"glidergo/internal/render"
)

// The way in: A opens the About box, C opens the credits from it, and any key leaves. The
// About box has to say so, because C being the one key that does not dismiss it is not
// something anybody would guess.
func TestCreditsOpenFromTheAboutBoxWithC(t *testing.T) {
	s, _ := shellOver(t, []string{"Slumberland"})

	s.event(key(platform.KeyA)[0])
	if s.mode != modeAbout {
		t.Fatalf("A left the shell in mode %d, want the About box", s.mode)
	}
	s.event(key(platform.KeyC)[0])
	if s.mode != modeCredits {
		t.Fatalf("C left the shell in mode %d, want the credits", s.mode)
	}
	s.event(key(platform.KeySpace)[0])
	if s.mode != modeSplash {
		t.Errorf("a key on the credits left the shell in mode %d, want the splash", s.mode)
	}

	// And the About box still goes away on everything else.
	s.event(key(platform.KeyA)[0])
	s.event(key(platform.KeyReturn)[0])
	if s.mode != modeSplash {
		t.Errorf("Return on the About box left the shell in mode %d", s.mode)
	}
}

// The hint on the About box names the key, and the key it names is the one that works. A
// screen reachable only by a key nothing mentions is not reachable.
func TestTheAboutBoxSaysHowToReachTheCredits(t *testing.T) {
	s, _ := shellOver(t, []string{"Slumberland"})
	var said bool
	for _, l := range s.aboutLines() {
		if strings.HasPrefix(l.text, "C ") {
			said = true
		}
	}
	if !said {
		t.Errorf("no line of the About box starts with C: %+v", s.aboutLines())
	}
}

// -shot-screen credits, which is how the screen is checked on a machine with no display
// (`make headless`).
func TestShowOpensTheCredits(t *testing.T) {
	s, _ := shellOver(t, nil) // no houses: the credits need none
	if err := s.Show("credits"); err != nil {
		t.Fatalf("Show(credits): %v", err)
	}
	if s.mode != modeCredits {
		t.Errorf("Show(credits) left the shell in mode %d", s.mode)
	}
	err := s.Show("who made this")
	if err == nil || !strings.Contains(err.Error(), "credits") {
		t.Errorf("the unknown-screen error should list credits: %v", err)
	}
}

// Everybody in the data file is placed on the screen. This is the check the obligation in
// docs/IMPROVEMENTS.md 1.2 actually needs: not that the names are in a file, but that they
// are laid out somewhere a player can read them.
func TestEverybodyNamedIsOnTheScreen(t *testing.T) {
	_, lines := layoutCredits()

	people := credits.People()
	if len(people) < 9 {
		t.Fatalf("internal/credits names only %d people: %v", len(people), people)
	}
	// The five house authors and both illustrators, spelled out as well as derived, so this
	// test still says what it is for if People() ever changes shape.
	want := append([]string{}, people...)
	want = append(want,
		"Kim Money", "Jonathan Chin", "Ward Hartenstein", "Steve Sullivan",
		"Shawn Brenneman", "John R. Neill", "Winsor McCay")

	for _, who := range want {
		found := false
		for _, l := range lines {
			if strings.Contains(l.text, who) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%q is nowhere on the credits screen", who)
		}
	}
}

// No line is truncated. The data file is hand-wrapped -- the notes are written as separate
// lines rather than flowed -- so this is the test that makes that safe: a line edited too
// long fails here instead of quietly losing its end to an ellipsis.
func TestNoCreditLineIsTruncated(t *testing.T) {
	_, lines := layoutCredits()
	for _, l := range lines {
		if strings.HasSuffix(l.text, "...") {
			t.Errorf("this line was cut to fit: %q", l.text)
		}
	}

	// Which is only meaningful if the ellipsis is what truncation looks like, so check the
	// other direction too: the widest thing in the data file against the room it has.
	var widest string
	for _, sec := range credits.Sections() {
		for _, r := range sec.Rows {
			line := r.What
			if !r.Note() {
				line = r.Who + "   " + r.What
			}
			if render.StringWidth(line) > render.StringWidth(widest) {
				widest = line
			}
		}
	}
	room := int16(creditsRight-creditsLeft) - 2*creditsInset
	if got := render.StringWidth(widest); got > room {
		t.Errorf("the widest line is %d pixels in a %d-pixel panel: %q", got, room, widest)
	}
}

// Every placed string is inside the panel, and the panel is inside the splash area -- not
// over the status band, not off the top. A screen whose height comes from a data file has to
// be checked for this, because adding a line to that file is the easiest edit in the project.
func TestTheCreditsPanelFitsTheScreen(t *testing.T) {
	box, lines := layoutCredits()

	if box.Top < 8 {
		t.Errorf("the panel starts at row %d; its own shadow needs eight rows above it",
			box.Top)
	}
	if box.Bottom > splashTall-8 {
		t.Errorf("the panel ends at row %d, which is into the status band at %d",
			box.Bottom, splashTall)
	}

	for _, l := range lines {
		wide := render.StringWidthScaled(l.text, l.scale)
		tallOf := int16(7 * l.scale) // the font's ink above the baseline
		switch {
		case l.h < box.Left:
			t.Errorf("%q starts at %d, left of the panel at %d", l.text, l.h, box.Left)
		case l.h+wide > box.Right:
			t.Errorf("%q ends at %d, right of the panel at %d", l.text, l.h+wide, box.Right)
		case l.v-tallOf < box.Top:
			t.Errorf("%q sits at %d, above the panel at %d", l.text, l.v, box.Top)
		case l.v > box.Bottom:
			t.Errorf("%q sits at %d, below the panel at %d", l.text, l.v, box.Bottom)
		}
	}
}

// And it really is drawn: the panel's frame lands on the screen, the band under it is still
// drawn, and the title screen's menu is not -- the credits are a panel over the backdrop, in
// the About box's manner, and not a screen of their own like the high-score board.
func TestTheCreditsAreDrawnOverTheBackdrop(t *testing.T) {
	s, f := shellOver(t, []string{"Slumberland"})
	s.mode = modeCredits
	s.Draw()

	box, _ := layoutCredits()
	for x := int(box.Left); x < int(box.Right); x++ {
		if got := f.scr.Pix[int(box.Top)*f.scr.W+x]; got != cream {
			t.Fatalf("the panel's top edge at (%d,%d) is colour %d, want the cream frame",
				x, box.Top, got)
		}
	}
	// The status band, which every screen has.
	if count(f.scr, render.Black8) == 0 {
		t.Error("nothing was drawn in black; the band is missing")
	}
	// And the house label is still under the panel's own top edge somewhere, because the
	// backdrop was drawn first.
	if f.scr.Pix[houseLabelV*f.scr.W+houseLabelH] == 0 && box.Bottom < houseLabelV {
		t.Error("the backdrop was not drawn under the credits")
	}
}
