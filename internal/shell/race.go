package shell

// The Race screen: the two keystrokes that arrange a game between two machines.
//
// # There is nothing to port
//
// Glider PRO has no network play, no second machine and no address anywhere in it. The race
// is this port's own mode (internal/netplay, and docs/analysis/ ends at the 1994 game), so
// every line of this screen is an invention and the only fidelity question it can answer is
// whether it looks and behaves like the rest of the shell. It does that by being built out of
// the same three pieces every other screen here is: a panel with a cream frame on a dimmed
// halo, a cursor moved with the arrows, and a status band that says what Return would do.
//
// # What it replaces, which is a command line
//
// Until this screen existed a race could only be arranged by two people at two shells typing
// `glidergo -host "Fun House"` and `glidergo -join 192.168.1.5:1138 "Fun House"`, which
// docs/IMPROVEMENTS.md 4.28 filed as the mode's real defect: a downloaded executable that is
// double-clicked has no shell in front of it, so the feature was reachable only by the people
// least likely to need it explained. Both command lines still work and still mean exactly
// what they did -- run() sends them past the title screen -- and this screen is the same
// arrangement made with the keyboard that is already in the player's hands.
//
// # The guest still has to be told which house, and that is not fixed here
//
// 4.28 has two halves and this is one of them. The other -- that a guest should be *told*
// what the host is playing rather than made to name it -- needs a protocol change and not a
// screen: netplay.Meet is one symmetric exchange in which both sides send a Hello carrying
// their own house hash, so there is no moment at which the guest knows the host's house and
// has not yet committed to one. What this screen can do is say out loud which house the
// arrangement is about, which it does on its last-but-one line, and what the *host's* waiting
// screen can do is read out the house's name along with the address (cmd/glidergo's
// hostingLines). Between the two, nobody has to guess; but a guest who picks the wrong house
// still finds out from netplay.Meet's mismatch error rather than from this screen, and that is
// recorded as still open.
//
// # Why the field is not internal/scores' Prompt
//
// internal/scores has a working modal text entry -- the one a champion types their name into
// -- and half of it is reused here: scores.Field is the buffer, the Mac Roman limit, the
// select-all-on-open behaviour and the keyboard, and it is generic on purpose. scores.Prompt
// is not reused, because every rectangle in it comes from DLOG 1020's own DITL and this screen
// is not that dialog; borrowing its geometry would put 1994's measurements on a screen 1994
// never drew.

import (
	"strings"

	"github.com/bwenstar/gliderGo/internal/platform"
	"github.com/bwenstar/gliderGo/internal/render"
	"github.com/bwenstar/gliderGo/internal/scores"
)

// Race is what the Race screen asks Play for: this machine waits for the other to connect, or
// it dials an address the other machine is waiting on.
//
// It is a type in this package rather than in internal/netplay for the reason the shell has no
// net import: the shell arranges a race and does not run one. What travels is a boolean and a
// string the player typed, and every decision about what to do with them -- which port, how
// long to wait, what the handshake agrees -- belongs to the host.
type Race struct {
	// Host waits for the other machine instead of dialling it. One side has to.
	Host bool

	// Join is the address to dial: a host name or an address, with an optional `:port`
	// after it, which is exactly what -join takes. Empty when this side is hosting.
	Join string
}

// Wanted reports whether this is a race at all. The zero Race is the ordinary game, which is
// what every other way into Play produces.
func (r Race) Wanted() bool { return r.Host || r.Join != "" }

// The screen's two rows. There is no third: a race has two ends and the player is at one of
// them.
const (
	raceRowHost = 0
	raceRowJoin = 1
)

// The panel and the two rows in it. Fixed rather than measured, for the settings screen's
// reason: nothing on this screen comes out of a house or a file except the house's name, which
// is the one thing here that gets fit() run over it.
const (
	raceLeft   = 96
	raceRight  = 544
	raceTop    = 100
	raceBottom = 356

	raceScale = 2

	raceTitleV = 132 // "Race", centred
	raceHeadV  = 156 // the one line that says what a race is

	raceHostV     = 194 // the Host row's baseline
	raceHostNoteV = 212 // and the scale-1 line under it
	raceJoinV     = 246
	raceJoinNoteV = 300

	raceLabelH = raceLeft + 24 // where a row's label starts
	raceNoteH  = raceLeft + 48 // and its note, indented under it to the field's edge

	raceHouseV = 328 // which house this is about
	raceFootV  = 348 // the keys

	// The address box. It is as wide as the panel allows because an address is the one
	// string on this screen that has to be read back character by character.
	raceFieldLeft   = raceLeft + 48
	raceFieldRight  = raceRight - 48
	raceFieldTop    = 256
	raceFieldBottom = 284
	raceFieldBase   = 276 // the baseline inside it
	raceFieldPad    = 4   // from the frame to the first character

	// raceAddrMax is how much the box can *show* at scale 1, less one character's room for
	// the insertion point, and that is deliberately the whole reason for the number.
	//
	// The alternative limits are both worse. A limit from the protocol would be 47, a
	// bracketed IPv6 literal with a port, and would refuse a perfectly ordinary LAN host
	// name that happened to be longer. No limit at all means a field that scrolls, and a
	// field that scrolls is a field whose contents the player cannot check against what the
	// other machine's screen says -- which is the only thing this box is for.
	raceAddrMax = (raceFieldRight-raceFieldLeft-2*raceFieldPad)/render.FontWide - 1
)

// ---------------------------------------------------------------------------
// Keys
// ---------------------------------------------------------------------------

// openRace puts the screen up, or says why it cannot.
//
// A house is the one requirement, and it is the same requirement "New Game" has: a race is
// two machines flying the same house, so there is nothing to arrange until this machine has
// one. The row is greyed rather than hidden for the reason every other row is.
func (s *Shell) openRace() {
	if _, ok := s.House(); !ok {
		s.msg = s.why("Race...")
		return
	}
	s.mode = modeRace
	s.msg = s.raceHelp()
}

// closeRace goes back to the title screen. It keeps the address: see addrField.
func (s *Shell) closeRace() {
	s.mode = modeSplash
	s.msg = s.opening()
}

// raceKey is the whole screen's input.
//
// Four keys are the screen's and everything else is the address field's, which is what makes a
// screen with a text field on it behave: the four are Escape, the two arrows and Return, and
// none of them types a character in any layout.
func (s *Shell) raceKey(ev platform.Event) {
	switch ev.Key {
	case platform.KeyEscape:
		s.closeRace()
		return
	case platform.KeyUp, platform.KeyDown:
		// Two rows, so both arrows do the same thing and neither needs to wrap.
		s.raceSel = raceRowJoin - s.raceSel
		s.msg = s.raceHelp()
		return
	case platform.KeyReturn:
		s.startRace()
		return
	}

	// Everything else goes to the address, and a character typed on the Host row moves the
	// cursor down to it rather than being dropped. Typing an address *is* choosing to join
	// -- there is nothing else on this screen an address could be for -- and the alternative
	// is a printable key that does nothing at all on one of only two rows, which is a dead
	// keystroke on the one screen where the player is expected to type. It is safe here and
	// would not be on the menu because these rows have no letters of their own to collide
	// with.
	//
	// What the cursor moves on is the field's *contents changing*, not the Action, because
	// scores.Field answers ActionTyped for a key that types nothing in any layout -- an
	// arrow, a function key -- and Left and Right reach here.
	f := s.addrField()
	was := f.Text()
	act := f.Event(ev, s.shift)
	if act == scores.ActionSelect || f.Text() != was {
		s.raceSel = raceRowJoin
		s.msg = s.raceHelp()
	}
}

// startRace is Return: hand the arrangement to the host and wait for it.
//
// Nothing here opens a socket or checks an address. An address that cannot be resolved is the
// host's error and comes back through Play, which the status band shows with the shell still
// standing -- exactly as a house that will not load does (docs/IMPROVEMENTS.md 2.33). The one
// thing refused here is the empty address, because that is not a typo, it is a row the player
// has not filled in yet, and sending it would produce "dial tcp :1138" as the explanation.
func (s *Shell) startRace() {
	h, ok := s.House()
	if !ok {
		s.msg = s.why("Race...")
		return
	}

	r := Race{Host: true}
	if s.raceSel == raceRowJoin {
		addr := strings.TrimSpace(s.addrField().Text())
		if addr == "" {
			s.msg = "type the address the other machine's screen is showing, then Return"
			return
		}
		r = Race{Join: addr}
	}
	s.start(Choice{House: h, Race: r})
}

// raceHelp is what the band says while this screen is up: what Return would do from here,
// which is a different sentence on each of the two rows.
func (s *Shell) raceHelp() string {
	if s.raceSel == raceRowHost {
		return "Return waits here for the other machine   Esc goes back"
	}
	return "type the address, then Return   Tab starts it over   Esc goes back"
}

// addrField is the address being typed, made on first use.
//
// It outlives a visit to the screen, which is not laziness: two people arranging a race get it
// wrong the first time -- the host had not pressed Return yet, somebody read a digit out wrong
// -- and a field that emptied itself on the way out would make the second attempt as much work
// as the first. It does not outlive the process, because there is nowhere in prefs.json for it
// and inventing a field there would mean a preferences file that records who this machine last
// played against.
func (s *Shell) addrField() *scores.Field {
	if s.raceAddr == nil {
		s.raceAddr = scores.NewField(raceAddrMax, "")
	}
	return s.raceAddr
}

// ---------------------------------------------------------------------------
// Drawing
// ---------------------------------------------------------------------------

func (s *Shell) drawRace(scr *render.Surface) {
	panel(scr, render.SetRect(raceLeft, raceTop, raceRight, raceBottom))
	centerIn(scr, raceLeft, raceRight, raceTitleV, "Race", cream, raceScale)
	centerIn(scr, raceLeft, raceRight, raceHeadV,
		"two machines, one house, one glider each", render.LtGray8, 1)

	s.drawRaceRow(scr, raceRowHost, raceHostV, "Host", raceHostNoteV,
		"this machine waits; the next screen has the address")
	s.drawRaceRow(scr, raceRowJoin, raceJoinV, "Join", raceJoinNoteV,
		"the address the other machine's screen is showing")
	s.drawRaceField(scr)

	// Which house, because the other machine has to open the same one and nothing in the
	// handshake will tell them which it was (see this file's header). fit() because a house
	// name is the one string on this screen that came out of a file.
	if h, ok := s.House(); ok {
		centerIn(scr, raceLeft, raceRight, raceHouseV,
			fit("the other machine opens "+h.Name+" too", raceRight-raceLeft-40, 1), cream, 1)
	}
	centerIn(scr, raceLeft, raceRight, raceFootV,
		"arrows move   Return starts   Esc goes back", render.LtGray8, 1)
}

// drawRaceRow draws one of the two rows: its label, in inverse video when the cursor is on it,
// and the scale-1 line under it that says what choosing it means.
//
// The note is drawn the same whether the row is chosen or not, which the settings screen's
// hints are not: there are two rows and the sentence under each is half of the explanation of
// the screen, so hiding one of them would leave a player reading half of it.
func (s *Shell) drawRaceRow(scr *render.Surface, row int, v int16, text string, noteV int16, note string) {
	if row == s.raceSel {
		bar := render.SetRect(raceLeft+barPad, v-barRise, raceRight-barPad, v+int16(raceScale*2))
		scr.Fill(bar, cream)
		scr.DrawStringScaled(raceLabelH, v, text, render.Black8, raceScale)
	} else {
		shadow(scr, raceLabelH, v, text, cream, raceScale)
	}
	shadow(scr, raceNoteH, noteV, note, render.LtGray8, 1)
}

// drawRaceField draws the address box: the Join row's value, in the idiom internal/scores'
// entry dialog draws item 2 in, turned inside out for a dark panel.
//
// The frame says whether the box is live -- cream on the Join row, grey on the Host row --
// because the box is the Join row's value and a value that looks editable on a row that is
// not selected is a box a player types into and watches do nothing.
func (s *Shell) drawRaceField(scr *render.Surface) {
	box := render.SetRect(raceFieldLeft, raceFieldTop, raceFieldRight, raceFieldBottom)
	scr.Fill(box, render.Black8)
	frame := uint8(render.Gray8)
	if s.raceSel == raceRowJoin {
		frame = cream
	}
	scr.FrameRect(box, frame)

	f := s.addrField()
	text, ink := f.Text(), uint8(cream)

	// Scale 2 while it fits and scale 1 when it does not, which is the honest answer to a
	// box that has to hold both `192.168.1.5` and a bracketed IPv6 literal with a port. An
	// address a person read out loud gets the big text; one that could only have been copied
	// from another screen gets small text rather than getting cut in half, and cutting it in
	// half is the one thing this box must not do.
	sc := raceScale
	if render.StringWidthScaled(text, sc) > box.Right-box.Left-2*raceFieldPad {
		sc = 1
	}

	if f.Selected() && f.Len() > 0 {
		// A fully selected field, drawn inverted, because the next character typed replaces
		// it -- the same warning the high-score dialog gives (internal/scores' Prompt).
		scr.Fill(render.Inset(box, 1, 1), cream)
		ink = render.Black8
	}
	scr.DrawStringScaled(box.Left+raceFieldPad, raceFieldBase, text, ink, sc)

	if !f.Selected() || f.Len() == 0 {
		// The insertion point, and it is drawn for an empty selected field as well: a box
		// with nothing in it and no caret reads as a box that is not for typing in, which
		// is exactly what the player has to do next. It does not blink, for the reason
		// scores.Prompt's does not -- the screen is redrawn from a poll loop with no clock.
		caret := box.Left + raceFieldPad + render.StringWidthScaled(text, sc)
		if caret < box.Right-2 {
			scr.Line(caret, raceFieldBase-int16(7*sc), caret, raceFieldBase+int16(sc), ink)
		}
	}
}
