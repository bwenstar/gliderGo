package shell

// The settings screen.
//
// # What this replaces
//
// Glider PRO's preferences were a five-pane modal dialog reached from the Options menu
// (DLOG 1024, docs/analysis/ui-dialogs.md 2.6): "Controls" with four editable key names,
// "Sound", "Visuals", "Brains" and "Displays". Four of its five panes are about a
// Macintosh rather than about the game -- screen depth switching, second-monitor play,
// colour table fades, zoom-rectangle transitions, the editor's window positions -- and
// internal/prefs/legacy.go lists all nineteen dropped fields with a reason each. What is
// left is what a player can actually feel: eight key bindings, which key pauses, how much
// of the house is on screen, how big the window is, the volume, and whether the score
// plays.
//
// # One list, not five panes
//
// There is no Dialog Manager here and no mouse (internal/platform), so a dialog with five
// radio-button panes is not portable to this shell even in spirit. A single list with a
// cursor is, and it has one property the original's dialog did not: **every setting is
// visible at once**, so a player looking for the pause key does not have to guess which of
// five panes it is behind. The original hid `isEscPauseKey` in "Brains", next to
// background tasks and the idle demo.
//
// # Every change is checked by the same code that checks the file
//
// A rebind calls prefs.Validate, which is the same function that repairs a hand-edited
// prefs.json, and it reports what it did on the status line. So the collision policy --
// first claimant keeps the key, the loser falls back to its default, and failing that is
// left unbound -- is written once and cannot disagree between the file and this screen.
// That is the whole reason Validate takes no arguments and never fails.

import (
	"fmt"

	"github.com/bwenstar/gliderGo/internal/platform"
	"github.com/bwenstar/gliderGo/internal/prefs"
	"github.com/bwenstar/gliderGo/internal/render"
)

// The panel and its three columns. The columns are fixed rather than measured because
// the row labels are ours -- nothing here comes from a house or a file -- so a table that
// fits at authoring time fits forever, and a measured layout would shift every row when
// one label changed.
const (
	setLeft  = 40
	setTop   = 40
	setRight = 600

	// The panel ends 16 rows above the status band (screens.go's bandTall), which is as
	// far down as it can go, and the block above starts as high as the title allows: with
	// fourteen rows and three group gaps the list is 310 rows deep, and it fits with a
	// row's clearance at each end. A fifteenth row does not fit -- it would have to come
	// out of the pitch, and 22 is two rows of a doubled 8-row font plus 6 to separate
	// them, which is the smallest gap the labels stay readable at.
	setBottom = 444

	setTitleV = 68
	setFirst  = 100 // first row's baseline
	setPitch  = 22
	setGap    = 8 // extra space above the first row of a group
	setFootV  = 428

	setGroupH = 56  // the group name, on the first row of each group
	setLabelH = 190 // the setting's name
	setValueH = 400 // its value
	setHintH  = 490 // a note about the value, at scale 1

	setScale = 2
)

// setRow is one line of the list: which group it opens (empty for a continuation), its
// label, and how its value is shown and changed.
//
// A row is either a binding (bind non-nil: Return captures a keystroke) or a value
// (step non-nil: left and right move it). Nothing here is both, and nothing is neither.
type setRow struct {
	group string
	label string

	// hint is drawn to the right of the value, and it means one thing only: **this row
	// does not apply to the next game.** Only magnification qualifies, because the window
	// is made once at launch (cmd/glidergo's openWindow).
	//
	// It is deliberately not on every row whose effect is invisible from this screen --
	// the bindings, the pause key, rooms in view and the music are all invisible here, and
	// all four are read when a game starts, which is the only thing a player can do next.
	// A note on each would say "as you would expect" four times and make the one row that
	// is genuinely different stop standing out.
	hint string

	// bind points at the binding this row edits, inside the live Prefs.
	bind func(*prefs.Prefs) *string

	// show renders the value. Binding rows leave it nil and show the binding.
	show func(*prefs.Prefs) string

	// step moves the value by -1 or +1. Binding rows leave it nil.
	step func(*prefs.Prefs, int)
}

// settings is the screen, in order.
//
// Three real preferences are deliberately not here, and each is a decision rather than an
// omission:
//
//   - `pause_when_unfocused`, because docs/IMPROVEMENTS.md 2.21 argues a released build
//     should always pause when it loses the foreground and should not offer the choice.
//     It is honoured -- cmd/glidergo's focus-loss arm reads it, and a file that turns it
//     off really does keep the game running in the background, which is where an imported
//     1994 `doBackground` lands. Not offered is not the same as not implemented.
//   - `keep_real_time` and the three entries under `fixes`, because they change what the
//     simulation does and 1.8's fidelity replays compare against the C. A player has no
//     way to judge them; a developer has prefs.json.
//
// `sound` is not here either, for a different reason: it is the original's `isSoundOn`,
// which the original derives from the volume (`isSoundOn = (isVolume != 0)`, Main.c) and
// so does prefs.Validate. Volume 0 *is* the mute, and a row for each would let a player
// set the two to disagree.
var settings = []setRow{
	{group: "Player One", label: "steer left", bind: func(p *prefs.Prefs) *string { return &p.Player1.Left }},
	{label: "steer right", bind: func(p *prefs.Prefs) *string { return &p.Player1.Right }},
	{label: "rubber band", bind: func(p *prefs.Prefs) *string { return &p.Player1.Band }},
	{label: "battery", bind: func(p *prefs.Prefs) *string { return &p.Player1.Batt }},

	{group: "Player Two", label: "steer left", bind: func(p *prefs.Prefs) *string { return &p.Player2.Left }},
	{label: "steer right", bind: func(p *prefs.Prefs) *string { return &p.Player2.Right }},
	{label: "rubber band", bind: func(p *prefs.Prefs) *string { return &p.Player2.Band }},
	{label: "battery", bind: func(p *prefs.Prefs) *string { return &p.Player2.Batt }},

	{
		group: "General", label: "pause key",
		show: func(p *prefs.Prefs) string { return p.PauseKey },
		// The original's own choice is exactly this binary -- isEscPauseKey, a
		// checkbox in the Brains pane -- and it stays binary because the pause
		// overlay is a picture with the key drawn into it (PICT 1015 and 1016).
		// There is no plate for any third key.
		step: func(p *prefs.Prefs, _ int) {
			if p.EscPause() {
				p.PauseKey = "tab"
			} else {
				p.PauseKey = "escape"
			}
		},
	}, {
		label: "rooms in view",
		show:  func(p *prefs.Prefs) string { return fmt.Sprintf("%d", p.Neighbors) },
		step: func(p *prefs.Prefs, d int) {
			// 1, 3 or 9 and nothing between: numNeighbors selects which of the
			// original's three composition paths runs (Render.c), not a radius.
			steps := []int{1, 3, 9}
			p.Neighbors = cycle(steps, p.Neighbors, d)
		},
	}, {
		label: "magnification", hint: "next launch",
		show: func(p *prefs.Prefs) string { return fmt.Sprintf("%dx", p.Scale) },
		step: func(p *prefs.Prefs, d int) { p.Scale = clamp(p.Scale+d, 1, prefs.MaxScale) },
	},

	{
		group: "Sound", label: "volume",
		show: func(p *prefs.Prefs) string { return fmt.Sprintf("%d of %d", p.Volume, prefs.MaxVolume) },
		step: func(p *prefs.Prefs, d int) { p.Volume = clamp(p.Volume+d, 0, prefs.MaxVolume) },
	}, {
		label: "music in a game",
		show:  func(p *prefs.Prefs) string { return yesNo(p.MusicInGame) },
		step:  func(p *prefs.Prefs, _ int) { p.MusicInGame = !p.MusicInGame },
	}, {
		// The original's other music preference (`isPlayMusicIdle`), and the only row
		// on this screen whose effect the player is listening to while the screen is
		// still up: the host starts and stops the score from ApplyPrefs. It gets no
		// hint for that reason -- there is nothing to explain about a setting that
		// takes effect as it is pressed.
		//
		// "while idle" rather than "on the title screen" because the label column is
		// 210 rows wide (setValueH - setLabelH) and the honest phrase measures 300 at
		// setScale. It is also the original's own word for it -- idle music is what
		// plays when nobody is playing -- and it is true of the house picker and this
		// screen as well as the splash, which "on the title screen" is not.
		label: "music while idle",
		show:  func(p *prefs.Prefs) string { return yesNo(p.MusicOnTitle) },
		step:  func(p *prefs.Prefs, _ int) { p.MusicOnTitle = !p.MusicOnTitle },
	},
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// cycle moves through a fixed set, wrapping. A value not in the set lands on the first
// entry, which is what a hand-edited file can hand us for one pass before Validate runs.
func cycle(set []int, cur, d int) int {
	for i, v := range set {
		if v == cur {
			return set[((i+d)%len(set)+len(set))%len(set)]
		}
	}
	return set[0]
}

// settingsBaseline is where row i is drawn. Draw and the selection bar both call it, so
// the bar cannot drift off its row when a group is added.
func settingsBaseline(i int) int16 {
	v := int16(setFirst)
	for j := 0; j < i && j+1 < len(settings); j++ {
		v += setPitch
		if settings[j+1].group != "" {
			v += setGap
		}
	}
	return v
}

// ---------------------------------------------------------------------------
// Keys
// ---------------------------------------------------------------------------

// openSettings puts the screen up, or explains why it cannot.
func (s *Shell) openSettings() {
	if s.host.Prefs == nil {
		s.msg = "this build has no preferences file"
		return
	}
	s.mode = modeSettings
	s.capture = -1
	s.msg = s.settingsHelp()
}

func (s *Shell) settingsHelp() string {
	return "arrows move and change   Return rebinds   R resets   Esc saves and goes back"
}

// settingsKey is the whole screen's input.
func (s *Shell) settingsKey(k platform.Key) {
	p := s.host.Prefs
	if p == nil { // cannot happen through openSettings; a direct Show could
		s.mode = modeSplash
		return
	}
	if s.capture >= 0 {
		s.captured(p, k)
		return
	}

	switch k {
	case platform.KeyUp:
		s.set = (s.set + len(settings) - 1) % len(settings)
	case platform.KeyDown:
		s.set = (s.set + 1) % len(settings)
	case platform.KeyLeft:
		s.adjust(p, -1)
	case platform.KeyRight:
		s.adjust(p, +1)
	case platform.KeyReturn, platform.KeySpace:
		if settings[s.set].bind != nil {
			s.capture = s.set
			s.msg = "press a key for " + s.rowName(s.set) +
				"   Delete clears it   Esc cancels"
			return
		}
		s.adjust(p, +1)
	case platform.KeyR:
		// The way back from a keyboard somebody has locked themselves out of. It is
		// also the only way to recover a binding that Validate left unbound, if the
		// key it wanted is now on another control.
		d := prefs.Default()
		p.Player1, p.Player2 = d.Player1, d.Player2
		p.PauseKey, p.Neighbors, p.Scale = d.PauseKey, d.Neighbors, d.Scale
		p.Volume, p.MusicInGame, p.MusicOnTitle = d.Volume, d.MusicInGame, d.MusicOnTitle
		s.changed(p, "settings reset to this build's defaults")
	case platform.KeyEscape, platform.KeyTab:
		s.closeSettings()
	}
}

// captured takes the keystroke a rebind was waiting for.
//
// Escape cancels and Delete clears, and neither is a binding this screen will accept:
// both are in Validate's reserved set, along with Return and whichever key pauses. So the
// three keys this screen needs for itself are exactly the three it refuses to hand out,
// which is why it can capture *any* other key without asking what it is for.
func (s *Shell) captured(p *prefs.Prefs, k platform.Key) {
	row := settings[s.capture]
	s.capture = -1

	switch k {
	case platform.KeyEscape:
		s.msg = s.settingsHelp()
		return
	case platform.KeyDelete:
		*row.bind(p) = prefs.Unbound
		s.changed(p, row.label+" cleared -- set it before playing")
		return
	}

	name := platform.KeyName(k)
	*row.bind(p) = name
	s.changed(p, "")
}

// adjust moves one value row and says what it became. A binding row is left alone: an
// arrow key sliding a binding through the alphabet would rebind by accident, and the
// alphabet is not an order a player thinks in anyway.
func (s *Shell) adjust(p *prefs.Prefs, d int) {
	row := settings[s.set]
	if row.step == nil {
		return
	}
	row.step(p, d)
	s.changed(p, "")
}

// changed is the one path out of every edit: validate, report, apply, remember to save.
//
// Validate is called on every change rather than only on the way out, because a
// collision has to be reported next to the keystroke that caused it. Its notes are taken
// rather than left on the Prefs: they are a running commentary for this screen, and a
// session of rebinding would otherwise pile up a hundred of them for whoever prints
// Notes next.
func (s *Shell) changed(p *prefs.Prefs, msg string) {
	s.setDirty = true

	mark := len(p.Notes)
	p.Validate()
	if len(p.Notes) > mark {
		msg = p.Notes[len(p.Notes)-1]
		p.Notes = p.Notes[:mark]
	}
	if msg == "" {
		msg = s.rowName(s.set) + " is " + s.rowValue(p, s.set)
	}
	s.msg = msg

	if s.host.ApplyPrefs != nil {
		s.host.ApplyPrefs()
	}
}

// closeSettings leaves the screen, saving if anything changed.
//
// The original writes its preferences once, at quit (WriteOutPrefs, Main.c:381), which
// loses every setting if the program is killed or crashes. Saving on the way out of the
// screen instead costs one file write per visit and cannot lose a change that was made.
func (s *Shell) closeSettings() {
	s.mode = modeSplash
	s.capture = -1
	if !s.setDirty {
		s.msg = s.opening()
		return
	}
	s.setDirty = false

	if s.host.SavePrefs == nil {
		s.msg = "settings changed for this session only -- there is nowhere to save them"
		return
	}
	if err := s.host.SavePrefs(); err != nil {
		s.msg = "could not save the settings: " + err.Error()
		s.notify("glidergo: " + s.msg)
		return
	}
	if p := s.host.Prefs.Path(); p != "" {
		s.msg = "settings saved in " + p
		return
	}
	s.msg = "settings saved"
}

// rowName is how a row is named in a sentence on the status line: "player one's steer
// left", not "steer left", because four of the rows share their label with four others.
func (s *Shell) rowName(i int) string {
	who := ""
	for j := i; j >= 0; j-- {
		if settings[j].group != "" {
			who = settings[j].group
			break
		}
	}
	switch who {
	case "Player One", "Player Two":
		return lowerFirst(who) + "'s " + settings[i].label
	default:
		return settings[i].label
	}
}

func (s *Shell) rowValue(p *prefs.Prefs, i int) string {
	row := settings[i]
	if row.bind != nil {
		return *row.bind(p)
	}
	return row.show(p)
}

func lowerFirst(s string) string {
	if s == "" || s[0] < 'A' || s[0] > 'Z' {
		return s
	}
	return string(rune(s[0]-'A'+'a')) + s[1:]
}

// ---------------------------------------------------------------------------
// Drawing
// ---------------------------------------------------------------------------

func (s *Shell) drawSettings(scr *render.Surface) {
	p := s.host.Prefs
	if p == nil {
		return
	}
	panel(scr, render.SetRect(setLeft, setTop, setRight, setBottom))
	shadow(scr, setLeft+20, setTitleV, "Settings", cream, 2)

	for i, row := range settings {
		v := settingsBaseline(i)

		if row.group != "" {
			shadow(scr, setGroupH, v, row.group, label, setScale)
		}

		text := row.label
		value := s.rowValue(p, i)
		col := uint8(cream)
		if s.capture == i {
			// The row being rebound says so where its value was, so the prompt on the
			// status line is not the only thing on screen that has changed.
			value = "press a key"
			col = label
		}
		if value == prefs.Unbound {
			// An unbound control is not a neutral state -- that glider cannot steer --
			// so it is drawn in the warning colour rather than the calm one.
			col = label
		}

		if i == s.set {
			bar := render.SetRect(setLabelH-barPad, v-barRise,
				int16(setRight-barPad), v+int16(setScale*2))
			scr.Fill(bar, cream)
			scr.DrawStringScaled(setLabelH, v, text, render.Black8, setScale)
			scr.DrawStringScaled(setValueH, v, value, render.Black8, setScale)
			if row.hint != "" {
				scr.DrawString(setHintH, v, row.hint, render.Black8)
			}
			continue
		}
		shadow(scr, setLabelH, v, text, cream, setScale)
		shadow(scr, setValueH, v, value, col, setScale)
		if row.hint != "" {
			shadow(scr, setHintH, v, row.hint, render.Gray8, 1)
		}
	}

	foot := "the rest of the settings are in " + prefs.Name
	if p := s.host.Prefs.Path(); p != "" {
		foot = "the rest of the settings are in " + p
	}
	shadow(scr, setLeft+20, setFootV, fit(foot, setRight-setLeft-40, 1), render.LtGray8, 1)
}
