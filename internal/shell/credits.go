package shell

// The credits screen.
//
// The original has none: its About box is three pictures and a dismiss button, and the only
// place the fifteen shipped houses' authors are named is a README file in the source release
// that no player ever saw. docs/IMPROVEMENTS.md 1.2 turns that into an obligation -- the five
// house authors and both illustrators have to be named in a shipped build, whatever is
// decided about the assets -- and 3.4 has been tracking it as unfinished since 1.7a.
//
// So this is one of the few screens in the port with no original to be faithful to. What it
// is faithful to instead is the text in internal/credits, which is checked against
// GliderPRO/README.md by that package's own tests; nothing here is a name typed into a Go
// literal, and adding a contributor is editing a data file.
//
// It hangs off the About box rather than the menu. The menu is already seven items long, and
// the About box is where somebody looking for who made this will look first -- so About
// prints one line saying C, and C comes here.

import (
	"github.com/bwenstar/gliderGo/internal/credits"
	"github.com/bwenstar/gliderGo/internal/render"
)

// The credits panel. Wider than the About box because the rows are two columns of prose --
// four names and a house on one line is the widest thing this shell draws -- and centred
// vertically on whatever it turns out to be tall enough for, as the About box is.
const (
	creditsLeft  = 24
	creditsRight = 616

	creditsFirst = 34 // the first baseline, below the panel's top
	creditsFoot  = 12 // below the last
	creditsInset = 16 // from the panel's left edge to a row's first character
	creditsGap   = 15 // a blank line between sections
	creditsPitch = 11 // one row
	creditsWho   = 12 // pixels between a name and what it is credited for
)

// placed is one string with somewhere to put it: the layout's output and the drawing's
// input. Splitting the two is not architecture for its own sake -- it is the only way a test
// can ask where a name ended up. A test that instead searched the finished surface for a
// pattern of pixels would find one in any solid fill, and this panel is mostly solid fill.
type placed struct {
	h, v  int16
	text  string
	col   uint8
	scale int
}

// layoutCredits places the panel and everything in it, top to bottom.
//
// It measures before it places, so the panel is the size of what is in it -- the same
// arrangement drawAbout uses, and for the same reason: a constant height becomes wrong the
// moment the data file gains a line, and the data file is the easiest thing here to edit.
func layoutCredits() (render.Rect, []placed) {
	secs := credits.Sections()

	// The heading of the first section needs no gap above it; every other one does.
	tall := int16(creditsFirst) + 2*creditsPitch // the title, at scale 2
	for i, sec := range secs {
		if i > 0 {
			tall += creditsGap
		}
		tall += creditsPitch                        // the heading
		tall += int16(len(sec.Rows)) * creditsPitch // its rows
	}
	tall += creditsGap + creditsPitch + creditsFoot // the way out

	top := (int16(splashTall) - tall) / 2
	if top < 16 {
		top = 16
	}
	box := render.SetRect(creditsLeft, top, creditsRight, top+tall)

	var out []placed
	center := func(v int16, text string, col uint8, scale int) {
		h := creditsLeft + (creditsRight-creditsLeft-render.StringWidthScaled(text, scale))/2
		out = append(out, placed{h, v, text, col, scale})
	}

	v := top + creditsFirst
	center(v, "credits", cream, 2)
	v += 2 * creditsPitch

	// A row is two runs of text on one baseline: who, in cream, and what they are credited
	// for after it in grey. Two colours on one line rather than two columns, because the
	// names are of wildly different lengths and a column wide enough for "John Calhoun,
	// Jonathan Chin, Steve Sullivan, Ward Hartenstein" would leave the rest of the screen
	// empty.
	left := int16(creditsLeft + creditsInset)
	wide := int16(creditsRight-creditsLeft) - 2*creditsInset
	for i, sec := range secs {
		if i > 0 {
			v += creditsGap
		}
		out = append(out, placed{left, v, sec.Title, label, 1})
		v += creditsPitch

		for _, r := range sec.Rows {
			if r.Note() {
				// A note is grey and full width: it qualifies the rows around it rather
				// than crediting anybody.
				out = append(out, placed{left, v, fit(r.What, wide, 1), render.LtGray8, 1})
				v += creditsPitch
				continue
			}
			out = append(out, placed{left, v, r.Who, cream, 1})
			h := left + render.StringWidth(r.Who) + creditsWho
			out = append(out, placed{h, v, fit(r.What, creditsRight-creditsInset-h, 1),
				render.LtGray8, 1})
			v += creditsPitch
		}
	}

	v += creditsGap
	center(v, "press any key", label, 1)
	return box, out
}

func (s *Shell) drawCredits(scr *render.Surface) {
	box, lines := layoutCredits()
	panel(scr, box)
	for _, l := range lines {
		shadow(scr, l.h, l.v, l.text, l.col, l.scale)
	}
}
