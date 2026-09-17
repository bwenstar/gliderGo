package scores

// The name and banner prompts: the field's behaviour, and the two dialogs' geometry against
// the DITL resources in docs/analysis/scoring.md 7.11.

import (
	"strconv"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/platform"
	"github.com/bwenstar/gliderGo/internal/render"
)

func key(k platform.Key, text string) platform.Event {
	return platform.Event{Kind: platform.EventKeyDown, Key: k, Text: text}
}

func typeString(f *Field, s string) {
	for _, r := range s {
		f.Event(key(platform.KeyUnknown, string(r)), false)
	}
}

// ------------------------------------------------------------------ the field

// The field arrives fully selected, because GetHighScoreName sets the previous name and then
// selects all of it (HighScores.c:519-520). So the first character replaces it, and a player
// who wants a new name does not have to erase the old one first.
func TestTheFieldStartsSelectedSoTypingReplacesTheOldName(t *testing.T) {
	f := NewField(NameMax, "Your Name")
	if !f.Selected() {
		t.Error("the field is not selected")
	}
	if got := f.Text(); got != "Your Name" {
		t.Errorf("the field holds %q", got)
	}

	typeString(f, "Bo")
	if got := f.Text(); got != "Bo" {
		t.Errorf("after typing two characters the field holds %q, want %q", got, "Bo")
	}
	if f.Selected() {
		t.Error("the field is still selected after a keystroke")
	}
}

// And Return on its own keeps it, which is the other half of the same behaviour: a returning
// player presses one key.
func TestReturnWithoutTypingKeepsThePreviousName(t *testing.T) {
	f := NewField(NameMax, "Ozma")
	if got := f.Event(key(platform.KeyReturn, ""), false); got != ActionCommit {
		t.Errorf("Return gave action %d, want ActionCommit", got)
	}
	if got := f.Text(); got != "Ozma" {
		t.Errorf("the field holds %q", got)
	}
}

// Tab reselects, HighScores.c:474. It is the only way to start over without holding delete.
func TestTabReselectsSoTheNextKeyStartsOver(t *testing.T) {
	f := NewField(NameMax, "")
	typeString(f, "Mistake")
	if f.Selected() {
		t.Fatal("the field should not be selected after typing")
	}
	if got := f.Event(key(platform.KeyTab, ""), false); got != ActionSelect {
		t.Errorf("Tab gave action %d, want ActionSelect", got)
	}
	if !f.Selected() {
		t.Fatal("Tab did not reselect")
	}
	typeString(f, "Ozma")
	if got := f.Text(); got != "Ozma" {
		t.Errorf("the field holds %q, want %q", got, "Ozma")
	}
}

// The deliberate deviation: the sixteenth character is refused rather than typed and then
// silently thrown away by PasStringCopyNum at commit (7.11.1). The counter and the saved name
// therefore always agree.
func TestTheFieldEnforcesItsLimitWhileTyping(t *testing.T) {
	for _, tc := range []struct {
		max  int
		try  string
		want string
	}{
		{NameMax, "Bartholomew Winthrop", "Bartholomew Win"},
		{BannerMax, strings.Repeat("z", 40), strings.Repeat("z", 31)},
	} {
		f := NewField(tc.max, "")
		typeString(f, tc.try)
		if got := f.Text(); got != tc.want {
			t.Errorf("max %d: typing %q gave %q, want %q", tc.max, tc.try, got, tc.want)
		}
		if f.Len() != tc.max {
			t.Errorf("max %d: the counter says %d", tc.max, f.Len())
		}
	}
}

// A limit in *bytes of Mac Roman*, not runes, because the file's name field is fifteen bytes.
// Fifteen accented characters fit; fifteen runes that each need two bytes of UTF-8 would not,
// and counting runes would be a limit that lied about the file.
func TestTheLimitIsInMacRomanBytes(t *testing.T) {
	f := NewField(NameMax, "")
	typeString(f, strings.Repeat("é", 20))
	if f.Len() != NameMax {
		t.Errorf("the field holds %d bytes, want %d", f.Len(), NameMax)
	}
	got := f.Text()
	if n := len([]rune(got)); n != NameMax {
		t.Errorf("the field holds %d characters, want %d", n, NameMax)
	}
	// And it survives the trip through the board's own Pascal string, which is the point.
	var p house.PStr16
	if !p.SetText(got) {
		t.Errorf("%q does not fit a Str15", got)
	}
	if back := p.Text(); back != got {
		t.Errorf("the board turned %q into %q", got, back)
	}
}

// A rune the board cannot hold is refused rather than stored as '?'. Refusing is not a good
// answer -- somebody typing their name on a Cyrillic layout gets nothing -- but a row of
// question marks on the high-score board is a worse one, and the alternative is a file format
// change (docs/IMPROVEMENTS.md).
func TestTheFieldRefusesRunesTheBoardCannotHold(t *testing.T) {
	f := NewField(NameMax, "")
	for _, r := range []rune{'Ж', '日', '→', '𝄞'} {
		if f.Insert(r) {
			t.Errorf("the field accepted %q", r)
		}
	}
	if f.Len() != 0 {
		t.Errorf("the field holds %q", f.Text())
	}

	// The Mac Roman upper half, on the other hand, is exactly what the board is for.
	for _, r := range []rune{'é', 'ü', 'ñ', 'Å', '•', '©', 'ƒ'} {
		if !f.Insert(r) {
			t.Errorf("the field refused %q, which is in Mac Roman", r)
		}
	}
}

// A control character must never reach the field: the board's names are drawn with no
// escaping, and a carriage return in the middle of one is a name that cannot be shown.
func TestTheFieldRefusesControlCharacters(t *testing.T) {
	f := NewField(NameMax, "")
	for _, r := range []rune{'\r', '\n', '\t', 0x00, 0x1B, 0x7F} {
		if f.Insert(r) {
			t.Errorf("the field accepted %U", r)
		}
	}
	if f.Len() != 0 {
		t.Errorf("the field holds %q", f.Text())
	}
}

func TestBackspaceErasesTheSelectionThenTheLastCharacter(t *testing.T) {
	f := NewField(NameMax, "Ozma")
	if !f.Backspace() {
		t.Error("backspace over a selection did nothing")
	}
	if f.Len() != 0 {
		t.Errorf("backspace over a selection left %q", f.Text())
	}
	if f.Backspace() {
		t.Error("backspace on an empty field claimed to erase something")
	}

	typeString(f, "abc")
	f.Backspace()
	if got := f.Text(); got != "ab" {
		t.Errorf("the field holds %q, want %q", got, "ab")
	}
}

// Each key press maps to the action that chooses the sound, which is the whole of what these
// dialogs give back: kCarriageSound on Return, kTypingSound on everything else (7.11.3).
func TestEventReportsTheActionThatChoosesTheSound(t *testing.T) {
	for _, tc := range []struct {
		what string
		ev   platform.Event
		want Action
	}{
		{"return", key(platform.KeyReturn, ""), ActionCommit},
		{"tab", key(platform.KeyTab, ""), ActionSelect},
		{"delete", key(platform.KeyDelete, ""), ActionErased},
		{"a letter", key(platform.KeyA, "a"), ActionTyped},
		{"a letter with no text", key(platform.KeyA, ""), ActionTyped},
		// Escape falls into the filter's default arm and plays the typing sound: there is
		// no Cancel button for it to reach.
		{"escape", key(platform.KeyEscape, ""), ActionTyped},
		{"an arrow", key(platform.KeyLeft, ""), ActionTyped},
		{"a key up", platform.Event{Kind: platform.EventKeyUp, Key: platform.KeyA}, ActionNone},
		{"a focus change", platform.Event{Kind: platform.EventFocus}, ActionNone},
		{"an unmapped key with no text", key(platform.KeyUnknown, ""), ActionNone},
	} {
		f := NewField(NameMax, "")
		if got := f.Event(tc.ev, false); got != tc.want {
			t.Errorf("%s: action %d, want %d", tc.what, got, tc.want)
		}
	}
}

// Text first, KeyChar second. The order matters for a non-US layout: on a French keyboard the
// physical Q types `a`, and only Text knows that.
func TestEventPrefersTheTextTheLayoutTypedOverThePhysicalKey(t *testing.T) {
	f := NewField(NameMax, "")
	f.Event(key(platform.KeyQ, "a"), false)
	if got := f.Text(); got != "a" {
		t.Errorf("the field holds %q, want the text the layout typed", got)
	}

	// And with no Text -- a backend that reports keys only -- the physical key is used,
	// with shift.
	g := NewField(NameMax, "")
	g.Event(key(platform.KeyQ, ""), false)
	g.Event(key(platform.KeyA, ""), true)
	if got := g.Text(); got != "qA" {
		t.Errorf("the field holds %q, want %q", got, "qA")
	}
}

// A key that types nothing must not put anything in the field, however the backend reports
// it. This is the bug internal/platform's own test guards KeyChar against, checked from the
// other side.
func TestKeysThatTypeNothingLeaveTheFieldAlone(t *testing.T) {
	f := NewField(NameMax, "")
	for _, k := range []platform.Key{
		platform.KeyEscape, platform.KeyLeft, platform.KeyRight, platform.KeyUp,
		platform.KeyDown, platform.KeyF1, platform.KeyF12, platform.KeyShift,
		platform.KeyControl, platform.KeyAlt, platform.KeySuper,
	} {
		f.Event(key(k, ""), false)
	}
	if f.Len() != 0 {
		t.Errorf("the field holds %q", f.Text())
	}
}

// ----------------------------------------------------------------- the dialogs

// Every rect from DITL 1020 and 1021, and both DLOG positions. The name prompt really is in
// the corner: BringUpDialog has CenterDialog commented out (DialogUtils.c:26), so DLOG 1020's
// stored bounds of (0,0,109,316) are where it appears, while DLOG 1021's (40,40,162,356) are
// not. 7.11.1 asks a faithful port to reproduce both and note the discrepancy.
func TestThePromptsUseTheResourcesOwnGeometry(t *testing.T) {
	name := NamePrompt(4200, 3, "Slumberland")
	if want := render.SetRect(0, 0, 316, 109); name.Bounds != want {
		t.Errorf("DLOG 1020 bounds %v, want %v", name.Bounds, want)
	}
	for _, tc := range []struct {
		what string
		got  render.Rect
		want render.Rect
	}{
		{"item 1, Okay", name.okay, render.SetRect(250, 81, 308, 101)},
		{"item 2, EditText", name.edit, render.SetRect(157, 52, 297, 68)},
		{"item 3, the message", name.message, render.SetRect(8, 8, 308, 41)},
		{"item 4, the prompt", name.ask, render.SetRect(16, 49, 132, 81)},
		{"item 5, the counter", name.count, render.SetRect(154, 81, 173, 97)},
		{"item 6, letters", name.letters, render.SetRect(175, 81, 224, 97)},
	} {
		if tc.got != tc.want {
			t.Errorf("DITL 1020 %s: %v, want %v", tc.what, tc.got, tc.want)
		}
	}

	banner := BannerPrompt()
	if want := render.SetRect(40, 40, 356, 162); banner.Bounds != want {
		t.Errorf("DLOG 1021 bounds %v, want %v", banner.Bounds, want)
	}
	// The item rects are dialog-relative in the resource and screen-absolute here, so each
	// wanted rect below is the DITL's offset by (40, 40).
	for _, tc := range []struct {
		what string
		got  render.Rect
		want render.Rect
	}{
		{"item 1, Okay", banner.okay, render.SetRect(290, 134, 348, 154)},
		{"item 2, EditText", banner.edit, render.SetRect(51, 107, 345, 123)},
		{"item 3, the message", banner.message, render.SetRect(51, 48, 345, 96)},
		{"item 5, the counter", banner.count, render.SetRect(48, 134, 67, 150)},
		{"item 4, letters", banner.letters, render.SetRect(69, 134, 118, 150)},
	} {
		if tc.got != tc.want {
			t.Errorf("DITL 1021 %s: %v, want %v", tc.what, tc.got, tc.want)
		}
	}
	if !render.Empty(banner.ask) {
		t.Errorf("DITL 1021 has no second static text, but one was laid out at %v", banner.ask)
	}
}

// ParamText: ^0 is the score, ^1 the 1-based placing, ^2 the house's name
// (HighScores.c:506-510). The placing the caller passes is already 1-based --
// GetHighScoreName is called with `placing + 1`.
func TestTheNamePromptSubstitutesParamText(t *testing.T) {
	p := NamePrompt(4200, 3, "Slumberland")
	want := "Your score of 4200 is #3 on the top ten high scores for Slumberland."
	if p.Message != want {
		t.Errorf("the message is %q,\n want %q", p.Message, want)
	}
	if !strings.Contains(p.Ask, "15 letters max.") {
		t.Errorf("the prompt is %q", p.Ask)
	}
}

func TestEachPromptsFieldHasTheRightLimit(t *testing.T) {
	if got := NamePrompt(0, 1, "H").Field("").Max; got != NameMax {
		t.Errorf("the name field's limit is %d, want %d", got, NameMax)
	}
	if got := BannerPrompt().Field("").Max; got != BannerMax {
		t.Errorf("the banner field's limit is %d, want %d", got, BannerMax)
	}
	if got := BannerPrompt().Field("Your Message Here").Text(); got != "Your Message Here" {
		t.Errorf("the banner field holds %q", got)
	}
}

// A modal dialog covers part of the screen and leaves the rest: the game or the board stays
// visible behind it. So Draw must touch nothing outside the DLOG's bounds -- except the
// default-button ring, which DrawDefaultButton puts four pixels *outside* item 1 and which is
// inside the dialog for both of these.
func TestDrawStaysInsideTheDialog(t *testing.T) {
	for _, p := range []*Prompt{NamePrompt(4200, 1, "Slumberland"), BannerPrompt()} {
		dst := render.NewSurface(640, 480)
		dst.Fill(dst.Bounds(), render.LtGray8)
		f := p.Field("Somebody")
		p.Draw(dst, f)

		for y := 0; y < dst.H; y++ {
			for x := 0; x < dst.W; x++ {
				in := x >= int(p.Bounds.Left) && x < int(p.Bounds.Right) &&
					y >= int(p.Bounds.Top) && y < int(p.Bounds.Bottom)
				if !in && dst.Pix[y*dst.W+x] != render.LtGray8 {
					t.Fatalf("%v: drew at (%d,%d), outside the dialog", p.Bounds, x, y)
				}
			}
		}
		// And the ring really is inside: item 1 inset by -6 must fit.
		ring := render.Inset(p.okay, -6, -6)
		if ring.Left < p.Bounds.Left || ring.Right > p.Bounds.Right ||
			ring.Top < p.Bounds.Top || ring.Bottom > p.Bounds.Bottom {
			t.Errorf("the default-button ring %v is outside the dialog %v", ring, p.Bounds)
		}
	}
}

// The counter is live, and it is drawn right-aligned in its 19-pixel item -- which is only
// wide enough for three characters, so the alignment is what keeps "31" from hanging out of
// it. Checked by drawing the expected string at the expected place and comparing pixels.
func TestTheCounterShowsTheCurrentLength(t *testing.T) {
	p := NamePrompt(4200, 1, "House")
	for _, n := range []int{0, 1, 9, 10, 15} {
		dst := render.NewSurface(640, 480)
		dst.Fill(dst.Bounds(), render.LtGray8)
		f := p.Field("")
		typeString(f, strings.Repeat("x", n))
		if f.Len() != n {
			t.Fatalf("could not get the field to %d characters", n)
		}
		p.Draw(dst, f)

		want := render.NewSurface(640, 480)
		want.Fill(want.Bounds(), render.White8)
		digits := strconv.Itoa(n)
		want.DrawString(p.count.Right-render.StringWidth(digits), p.count.Bottom-4,
			digits, render.Black8)

		for y := int(p.count.Top); y < int(p.count.Bottom); y++ {
			for x := int(p.count.Left); x < int(p.count.Right); x++ {
				if dst.Pix[y*dst.W+x] != want.Pix[y*want.W+x] {
					t.Fatalf("%d characters: the counter differs at (%d,%d)", n, x, y)
				}
			}
		}
	}
}

// A fully selected field is drawn inverted, so the player can see that typing will replace
// what is in it rather than add to it.
func TestASelectedFieldIsDrawnInverted(t *testing.T) {
	p := NamePrompt(4200, 1, "House")

	lit := render.NewSurface(640, 480)
	p.Draw(lit, p.Field("Ozma"))

	f := p.Field("Ozma")
	f.Event(key(platform.KeyTab, ""), false) // still selected
	typeString(f, "O")                       // now it is not
	plain := render.NewSurface(640, 480)
	p.Draw(plain, f)

	inner := render.Inset(p.edit, 1, 1)
	black := tally(lit, inner)[render.Black8]
	if black < int(inner.Right-inner.Left) {
		t.Errorf("the selected field is not filled: %d black pixels in %v", black, inner)
	}
	if white := tally(lit, inner)[render.White8]; white == 0 {
		t.Error("the selected field has no white text in it")
	}
	if got := tally(plain, inner)[render.Black8]; got >= black {
		t.Errorf("the unselected field has %d black pixels and the selected one %d", got, black)
	}
}

// FlashDialogButton's two halves. The face inverts and nothing else on the screen moves, so a
// host can flash the button without redrawing the dialog under it.
func TestHiliteInvertsTheOkayButtonAndNothingElse(t *testing.T) {
	p := NamePrompt(4200, 1, "House")
	scr := render.NewSurface(640, 480)
	p.Draw(scr, p.Field("Ozma"))
	before := tally(scr, p.Bounds)

	p.Hilite(scr, true)
	lit := tally(scr, p.okay)
	if lit[render.Black8] < lit[render.White8] {
		t.Errorf("the hilited button is %d black to %d white; it should be mostly black",
			lit[render.Black8], lit[render.White8])
	}
	if lit[render.White8] == 0 {
		t.Error("the hilited button has no white label on it")
	}

	// And releasing it puts the dialog back exactly as it was, which is what lets the flash
	// be two draws rather than a save and a restore.
	p.Hilite(scr, false)
	after := tally(scr, p.Bounds)
	for idx, n := range before {
		if after[idx] != n {
			t.Errorf("colour %d: %d pixels before the flash, %d after", idx, n, after[idx])
		}
	}
}

// The static text has to fit the item it was given. The Dialog Manager wrapped these with the
// Mac font's metrics; this font is six pixels a character, so the wrap points differ and the
// only property worth checking is the one the layout depends on.
func TestTheStaticTextFitsItsItem(t *testing.T) {
	for _, p := range []*Prompt{
		NamePrompt(4200, 1, "Slumberland"),
		NamePrompt(2147483647, 10, "A House With A Preposterously Long Name Indeed"),
		BannerPrompt(),
	} {
		for _, item := range []struct {
			r    render.Rect
			text string
		}{{p.message, p.Message}, {p.ask, p.Ask}} {
			if render.Empty(item.r) {
				continue
			}
			wide := int(item.r.Right - item.r.Left)
			for _, para := range strings.Split(item.text, "\n") {
				for _, line := range wrap(para, wide) {
					if got := int(render.StringWidth(line)); got > wide {
						t.Errorf("%q is %d wide in a %d-wide item", line, got, wide)
					}
				}
			}
		}
	}
}

// wrap must not lose or duplicate a word, whatever it does with the line breaks.
func TestWrapKeepsEveryWord(t *testing.T) {
	for _, text := range []string{
		"", "one", "a b c",
		"Your score of 4200 is #3 on the top ten high scores for Slumberland.",
		"Getting #1 on the high scores entitles you to change the high score banner.",
		strings.Repeat("supercalifragilistic ", 5),
	} {
		for _, wide := range []int{300, 116, 60, 12, 6} {
			got := strings.Join(wrap(text, wide), " ")
			if strings.Join(strings.Fields(got), "") != strings.Join(strings.Fields(text), "") {
				t.Errorf("wrap(%q, %d) = %q; the words changed", text, wide, got)
			}
		}
	}
}
