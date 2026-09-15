package scores

// The two entry dialogs: DLOG 1020, where a champion types their name, and DLOG 1021, which
// only first place ever sees, where they rewrite the banner.
//
// docs/analysis/scoring.md 7.11 has the resources item by item, and every rect below is one
// of theirs. What this port cannot reproduce is the *mechanism*: those were Dialog Manager
// modal dialogs, with a Mac window frame, a Mac text-editing field, a Mac button and a
// ModalDialog loop with a key filter hanging off it. There is none of that here, so what is
// transcribed is the layout, the strings, the sounds, the character limits and the two
// behaviours that are easy to miss and change what the player has to do:
//
//   - **The name field starts fully selected.** GetHighScoreName does SetDialogString with
//     the previous name and then SelectDialogItemText(item, 0, 1024) (HighScores.c:519-520),
//     so the first character typed *replaces* "Your Name" rather than appending to it -- and
//     Return on its own keeps it. A port that appended would make every player delete
//     nine characters before they could start.
//   - **Tab reselects.** The filter answers a tab with the same SelectDialogItemText
//     (HighScores.c:474), which is how you get the whole field back after typing. Kept,
//     because it is the only way to start over without holding delete.
//
// And two deliberate deviations, both noted in docs/IMPROVEMENTS.md:
//
//   - **The limit is enforced while typing.** The original's EditText item has no length
//     limit at all: you could type forty characters, watch the counter say 40, and lose
//     everything past the fifteenth to PasStringCopyNum at commit (7.11.1). Here the field
//     refuses the sixteenth character, so the counter never says something the saved name
//     will contradict.
//   - **A character with no Mac Roman equivalent is refused rather than mangled.** The board
//     is 292 bytes of Mac Roman and cannot hold anything else; house.UTF8ToMacRoman answers
//     an unmappable rune with '?', so a player typing their name on a Cyrillic or Japanese
//     layout would get a row of question marks. Refusing is not a good answer either -- it
//     is the least bad one available before the file format changes, which is a Stage 2
//     question.

import (
	"fmt"
	"strings"

	"glidergo/internal/house"
	"glidergo/internal/platform"
	"glidergo/internal/render"
)

// The two character limits, from PasStringCopyNum's third argument.
const (
	NameMax   = 15 // HighScores.c:527, and "(15 letters max.)" in DITL 1020
	BannerMax = 31 // HighScores.c:633, and "(31 letters max.)" in DITL 1021
)

// The sounds the dialogs play, GliderDefines.h. Their ids are here rather than in
// internal/game because these two are the only sounds outside a running game.
const (
	TypingSound      = 52 // kTypingSound: every key that is not Return or Tab
	TypingPriority   = 808
	CarriageSound    = 53 // kCarriageSound: Return, the one that commits
	CarriagePriority = 809
	EnergizeSound    = 6 // kEnergizeSound: the dialog appearing
	EnergizePriority = 803
)

// ---------------------------------------------------------------------------
// Field
// ---------------------------------------------------------------------------

// Action is what one key press did to a Field. It exists so the caller can play the sound
// the original plays -- which is the whole of the feedback these dialogs give.
type Action int

const (
	ActionNone   Action = iota // not a key press, or a key the field ignores
	ActionTyped                // a printable key: kTypingSound, whether or not it fitted
	ActionErased               // delete: also kTypingSound in the original's default arm
	ActionSelect               // tab: the field is reselected, kTypingSound
	ActionCommit               // return or enter: kCarriageSound, then leave
)

// Field is a one-line text field holding Mac Roman bytes.
//
// Mac Roman rather than UTF-8 because the limit is in bytes of the file's own encoding: a
// name is a Str15, fifteen bytes, and "fifteen letters" and "fifteen bytes" are the same
// sentence only in that encoding. Counting runes would let a player type fifteen accented
// characters and have the board hold them, which it can, and then let them type fifteen
// characters the board cannot hold at all.
type Field struct {
	Max int // in Mac Roman bytes

	text     []byte
	selected bool // the whole field is selected: the next character replaces it
}

// NewField makes a field holding s, selected, which is the state
// SetDialogString-then-SelectDialogItemText leaves item 2 in.
func NewField(max int, s string) *Field {
	f := &Field{Max: max}
	f.Set(s)
	return f
}

// Set replaces the contents and selects them all.
func (f *Field) Set(s string) {
	b, _ := house.UTF8ToMacRoman(s)
	if len(b) > f.Max {
		b = b[:f.Max]
	}
	f.text = append(f.text[:0], b...)
	f.selected = true
}

// Text is the contents as UTF-8, for the board and for the screen.
func (f *Field) Text() string { return house.MacRomanToUTF8(f.text) }

// Len is the number the dialog's counter shows: characters typed, which for Mac Roman is
// bytes.
func (f *Field) Len() int { return len(f.text) }

// Selected reports whether the whole field is selected, so the caller can draw it that way.
func (f *Field) Selected() bool { return f.selected }

// SelectAll is Tab's answer, SelectDialogItemText(dial, 2, 0, 1024).
func (f *Field) SelectAll() { f.selected = true }

// Clear empties the field.
func (f *Field) Clear() {
	f.text = f.text[:0]
	f.selected = false
}

// Insert types one character. It reports whether the character went in: false for a rune
// with no Mac Roman equivalent, for an unprintable one, and for one that would overflow.
func (f *Field) Insert(r rune) bool {
	if f.selected {
		// A selected field is replaced by the first thing typed into it, which is what
		// makes the prefilled "Your Name" convenient rather than an obstacle.
		f.text = f.text[:0]
		f.selected = false
	}
	if r < 0x20 || r == 0x7F {
		return false
	}
	b, ok := house.UTF8ToMacRoman(string(r))
	if !ok || len(b) != 1 {
		// UTF8ToMacRoman substitutes '?' for a rune it cannot map. Storing that would put
		// a row of question marks on the board in place of somebody's name; refusing at
		// least leaves them able to see that the key did nothing.
		return false
	}
	if len(f.text) >= f.Max {
		return false
	}
	f.text = append(f.text, b[0])
	return true
}

// Backspace erases: the selection if there is one, otherwise the last character.
func (f *Field) Backspace() bool {
	if f.selected {
		f.Clear()
		return true
	}
	if len(f.text) == 0 {
		return false
	}
	f.text = f.text[:len(f.text)-1]
	return true
}

// Event applies one platform event and says what it did.
//
// The two sources of a character are taken in the order that keeps a non-US keyboard
// working: ev.Text is what the layout actually typed, and platform.KeyChar is the fallback
// for a backend that reports keys and no text. See internal/platform/x11, which fills Text
// from XLookupString for exactly this screen.
func (f *Field) Event(ev platform.Event, shift bool) Action {
	if ev.Kind != platform.EventKeyDown {
		return ActionNone
	}
	switch ev.Key {
	case platform.KeyReturn:
		return ActionCommit
	case platform.KeyTab:
		f.SelectAll()
		return ActionSelect
	case platform.KeyDelete:
		f.Backspace()
		return ActionErased
	}

	if ev.Text != "" {
		typed := false
		for _, r := range ev.Text {
			// Insert's answer is deliberately ignored: the original plays its typing sound
			// on every key that is not Return or Tab, including one that overflows the
			// field, so a refused character still sounds like a keystroke.
			f.Insert(r)
			typed = true
		}
		if typed {
			return ActionTyped
		}
	}
	if r, ok := platform.KeyChar(ev.Key, shift); ok {
		f.Insert(r)
		return ActionTyped
	}
	if ev.Key == platform.KeyUnknown {
		return ActionNone
	}
	// A key that types nothing -- escape, an arrow, a function key. The original's filter
	// falls into its default arm and plays the typing sound for these too, and there is no
	// Cancel button for escape to reach (7.11.1).
	return ActionTyped
}

// ---------------------------------------------------------------------------
// The dialogs
// ---------------------------------------------------------------------------

// The word beside the counter, DITL 1020 item 6 and DITL 1021 item 4.
const lettersWord = "letters"

// The Okay button's label, DITL item 1. There is no Cancel in either dialog: the loop only
// leaves on item 1 (HighScores.c:523), so a player who has qualified is going to be asked
// for a name whether they want to be or not.
const okayWord = "Okay"

// Prompt is one of the two dialogs, laid out.
//
// The rects are absolute -- the DLOG's own position plus the DITL's item rects -- because
// both of those come out of the resource fork and neither is this port's to choose. That
// includes DLOG 1020's position, which is the top-left corner of the screen: BringUpDialog
// has `CenterDialog(dialogID)` commented out (DialogUtils.c:26), so the name prompt appears
// jammed into the corner while the banner prompt, whose DLOG bounds are (40,40), does not.
// The analysis asks a faithful port to reproduce both and note the discrepancy (7.11.1), so
// that is what happens here; centring them is a Stage 1.8 question, recorded in
// docs/IMPROVEMENTS.md.
type Prompt struct {
	Bounds  render.Rect // the DLOG's bounds
	Message string      // the wide static text at the top, ParamText already applied
	Ask     string      // the narrower prompt beside the field, with its \r kept as \n

	edit    render.Rect // item 2, the EditText
	okay    render.Rect // item 1, the button
	count   render.Rect // the counter item
	letters render.Rect // the word "letters"
	message render.Rect // the wide static text
	ask     render.Rect // the narrower one, absent in DLOG 1021
}

// NamePrompt is DLOG/DITL 1020 with ParamText applied: ^0 is the score, ^1 the 1-based
// placing, ^2 the house's name (HighScores.c:506-510).
func NamePrompt(score int32, placing int, houseName string) *Prompt {
	p := &Prompt{
		Bounds: render.SetRect(0, 0, 316, 109),
		Message: fmt.Sprintf("Your score of %d is #%d on the top ten high scores for %s.",
			score, placing, houseName),
		Ask: "Enter your name:\n(15 letters max.)",
	}
	p.place(
		render.SetRect(250, 81, 308, 101), // item 1, Okay
		render.SetRect(157, 52, 297, 68),  // item 2, EditText
		render.SetRect(8, 8, 308, 41),     // item 3, the wide message
		render.SetRect(16, 49, 132, 81),   // item 4, the prompt
		render.SetRect(154, 81, 173, 97),  // item 5, the counter
		render.SetRect(175, 81, 224, 97),  // item 6, "letters"
	)
	return p
}

// BannerPrompt is DLOG/DITL 1021, which only a first place reaches (HighScores.c:404-408).
// It has no ParamText and no second prompt: its one static text carries both sentences.
func BannerPrompt() *Prompt {
	p := &Prompt{
		Bounds: render.SetRect(40, 40, 356, 162),
		Message: "Getting #1 on the high scores entitles you to change the high score " +
			"banner.\n(31 letters max.)",
	}
	p.place(
		render.SetRect(250, 94, 308, 114), // item 1, Okay
		render.SetRect(11, 67, 305, 83),   // item 2, EditText
		render.SetRect(11, 8, 305, 56),    // item 3, the message
		render.Rect{},                     // no second prompt
		render.SetRect(8, 94, 27, 110),    // item 5, the counter
		render.SetRect(29, 94, 78, 110),   // item 4, "letters"
	)
	return p
}

// place converts DITL rects, which are relative to the dialog's top-left, into screen
// coordinates.
func (p *Prompt) place(okay, edit, message, ask, count, letters render.Rect) {
	h, v := p.Bounds.Left, p.Bounds.Top
	move := func(r render.Rect) render.Rect {
		if render.Empty(r) {
			return r
		}
		return render.Offset(r, h, v)
	}
	p.okay, p.edit, p.message = move(okay), move(edit), move(message)
	p.ask, p.count, p.letters = move(ask), move(count), move(letters)
}

// Field makes the field this prompt edits, holding the value it starts from.
func (p *Prompt) Field(initial string) *Field {
	max := BannerMax
	if p.Ask != "" {
		max = NameMax
	}
	return NewField(max, initial)
}

// Draw paints the dialog over whatever is already on the surface, which is what a modal
// dialog does: the game's screen stays visible behind it.
//
// A Mac dialog is white with a black frame, and its static text is black. That is not a
// choice either -- these are `dBoxProc` windows and the Dialog Manager erases them to the
// window background, which for the System 7 default is white.
func (p *Prompt) Draw(dst *render.Surface, f *Field) {
	// The window. The one-pixel frame stands in for the Mac's shadowed dBoxProc border,
	// which needs a round-rect and a grey pattern this port has no reason to add.
	dst.Fill(p.Bounds, render.White8)
	dst.FrameRect(p.Bounds, render.Black8)

	drawWrapped(dst, p.message, p.Message)
	if p.Ask != "" {
		drawWrapped(dst, p.ask, p.Ask)
	}

	// Item 2. A Mac EditText draws its own one-pixel frame, and a fully selected field is
	// drawn inverted -- black box, white text -- which is how the player can see that
	// typing will replace what is there.
	dst.Fill(p.edit, render.White8)
	dst.FrameRect(p.edit, render.Black8)
	text, ink := f.Text(), uint8(render.Black8)
	if f.Selected() && f.Len() > 0 {
		inner := render.Inset(p.edit, 1, 1)
		dst.Fill(inner, render.Black8)
		ink = render.White8
	}
	// The baseline: the item is sixteen pixels tall and the font puts a capital in the
	// seven rows above the baseline, so v = bottom - 4 centres it.
	baseline := p.edit.Bottom - 4
	dst.DrawString(p.edit.Left+3, baseline, text, ink)
	if !f.Selected() {
		// The insertion point. A Mac caret blinks; this one does not, because the screen
		// is redrawn from a poll loop and a blink would need a clock the loop does not
		// have. Noted rather than faked.
		caret := p.edit.Left + 3 + render.StringWidth(text)
		if caret < p.edit.Right-1 {
			dst.Line(caret, baseline-7, caret, baseline+1, render.Black8)
		}
	}

	// The live counter and the word beside it (7.11.3). The original's lags one keystroke
	// behind, because its filter refreshes the count on the event *after* the character
	// went in; here it is always current, which is a bug fixed rather than transcribed --
	// a counter that disagrees with the field is indistinguishable from a broken limit.
	n := fmt.Sprint(f.Len())
	dst.DrawString(p.count.Right-render.StringWidth(n), p.count.Bottom-4, n, render.Black8)
	dst.DrawString(p.letters.Left, p.letters.Bottom-4, lettersWord, render.Black8)

	p.drawOkay(dst, false)
}

// Hilite draws the Okay button pressed or released, which is the two halves of
// FlashDialogButton (DialogUtils.c:335-346): `HiliteControl(255)`, `Delay(8, ...)`,
// `HiliteControl(0)`. The eight ticks in the middle are the caller's, because waiting is the
// host's business everywhere else in this port and there is no reason for this to be the
// exception.
//
// It is the whole of the acknowledgement the player gets that Return was accepted, and it
// matters more here than it did in 1994: the original also played kCarriageSound, and a
// machine with no audio player (internal/audio/sink.go) has only the flash.
func (p *Prompt) Hilite(dst *render.Surface, on bool) { p.drawOkay(dst, on) }

// drawOkay is item 1, plus the default-button ring DrawDefaultButton puts four pixels
// outside it three pixels thick (DialogUtils.c:352-363). Square rather than rounded: the
// original uses FrameRoundRect(16, 16) and this port has no round-rect primitive, which is a
// decoration rather than a behaviour.
//
// `lit` is the Control Manager's hilite state, which for a push button inverts its face.
func (p *Prompt) drawOkay(dst *render.Surface, lit bool) {
	face, ink := uint8(render.White8), uint8(render.Black8)
	if lit {
		face, ink = render.Black8, render.White8
	}
	dst.Fill(p.okay, face)
	dst.FrameRect(p.okay, render.Black8)
	ring := render.Inset(p.okay, -4, -4)
	for i := 0; i < 3; i++ {
		dst.FrameRect(render.Inset(ring, int16(-i), int16(-i)), render.Black8)
	}
	w := render.StringWidth(okayWord)
	dst.DrawString(p.okay.Left+(p.okay.Right-p.okay.Left-w)/2, p.okay.Bottom-6,
		okayWord, ink)
}

// drawWrapped lays a static text item out inside its rect: explicit newlines are the
// original's `\r`, and anything longer than the rect is wrapped on spaces.
//
// The Dialog Manager wrapped these itself, using the font's own metrics. This font is six
// pixels a character, so the wrap points differ from 1994's; what is preserved is that the
// text fits in the item it was given, which is the property the layout depends on.
func drawWrapped(dst *render.Surface, r render.Rect, text string) {
	if render.Empty(r) {
		return
	}
	wide := int(r.Right - r.Left)
	v := r.Top + int16(render.FontTall) - 2 // the first baseline inside the item
	for _, para := range strings.Split(text, "\n") {
		for _, line := range wrap(para, wide) {
			if v > r.Bottom+int16(render.FontTall) {
				return // out of room: better a clipped sentence than one over the button
			}
			dst.DrawString(r.Left, v, line, render.Black8)
			v += int16(render.FontTall) + 1
		}
	}
}

// wrap breaks one paragraph into lines no wider than wide pixels, on spaces where it can and
// mid-word where it cannot.
func wrap(text string, wide int) []string {
	if text == "" {
		return []string{""}
	}
	var out []string
	line := ""
	flush := func() {
		out = append(out, line)
		line = ""
	}
	for _, word := range strings.Fields(text) {
		try := word
		if line != "" {
			try = line + " " + word
		}
		if int(render.StringWidth(try)) <= wide {
			line = try
			continue
		}
		if line != "" {
			flush()
		}
		// A single word too wide for the item: break it rather than let it run out of the
		// dialog. Only reachable for a house name somebody has typed, which is exactly the
		// input this port cannot bound.
		for int(render.StringWidth(word)) > wide && wide >= render.FontWide {
			n := wide / render.FontWide
			if n >= len(word) {
				break
			}
			out = append(out, word[:n])
			word = word[n:]
		}
		line = word
	}
	if line != "" {
		flush()
	}
	if len(out) == 0 {
		out = []string{""}
	}
	return out
}
