// Package shell is the way into the game: the title screen, the menu, the house
// picker and whatever else stands between launching the program and playing.
//
// # What this replaces
//
// In 1994 all of it was the Macintosh Toolbox. The splash screen was a window with
// a PICT scaled into it; the menu was the system menu bar with four MENUs in it,
// enabled and disabled by one function (Menu.c's UpdateMenus); the house picker was
// a modal dialog of twelve icon-and-name userItems with a filterproc for the
// keyboard; and every message was an ALRT. docs/analysis/ui-dialogs.md documents
// all of it, and its porting notes are why almost none of it is transcribed:
// there is no menu bar to put a menu in, no Dialog Manager, no modal event loop and
// -- see internal/platform -- no mouse. What is faithful here is the geometry that
// shows (the splash PICT and the "House:" label DrawOnSplash writes over it), the
// discovery and sort rules (library.go), and the discipline that one function
// decides what is available and everything else asks it (P8, and menu below).
//
// # Keyboard only, on purpose
//
// internal/platform reports keys and window events and no pointer, because the game
// itself needs none: Glider PRO is played with four keys and the only mouse in it is
// in the editor. The shell is therefore built the way an arcade cabinet's is -- a
// list, a cursor, and a letter for every item -- and that is close to how the
// original's own arcade build behaved (BUILD_ARCADE_VERSION is 1, and Events.c:191-207
// wires the arrow keys straight to menu commands, §2.6). The one deliberate departure
// is which arrows: the original's fire commands (Left = high scores, Right = demo,
// Up and Down both = new game) because it had no on-screen menu to move a cursor
// through. This has one, so the arrows move the cursor and Return chooses, which is
// what someone sitting down in front of it will try first. Every item also has its
// letter, so the arcade property -- one keypress starts a game -- is kept.
//
// # What the shell is not
//
// It does not import internal/game. The game is reached through the Play hook, which
// cmd/glidergo fills in with the thing that builds a World and calls NewGame. That
// keeps the shell testable without a display, a house, a sound card or a game (see
// shell_test.go, which drives the whole thing with scripted key events and a Play
// that records what it was asked for), and it keeps the host wiring in one file
// where it can be read.
package shell

import (
	"errors"
	"fmt"

	"glidergo/internal/platform"
	"glidergo/internal/render"
)

// Host is the machine's side of the shell: a surface to draw on, a way to show it,
// a source of events, and a way to play a game. Everything else is optional.
type Host struct {
	// Screen is the 640x480 indexed surface the shell composes into. It is not the
	// World's Main -- a game builds its own -- and the two are published through the
	// same window by whatever Present does.
	Screen *render.Surface

	// Assets is the extracted application art, for the splash PICT and the other
	// plates. A nil Assets, or one with no art behind it, is not an error: the shell
	// draws its own chrome instead and says so on the status line
	// (docs/IMPROVEMENTS.md 2.6).
	Assets *render.Assets

	// Present publishes Screen. Required.
	Present func()

	// Poll drains host events. Required, and it must not block: the shell redraws
	// and presents on every pass, so a blocking poll would freeze the title screen.
	Poll func() []platform.Event

	// Idle is called once per pass after the present, to give the frame back to the
	// machine. Optional -- the tests leave it nil and spin -- but a host that leaves
	// it nil will burn a core showing a static screen.
	Idle func()

	// Play runs one game and returns when it is over. Required.
	//
	// It returns an error for a house that cannot be played -- a file that sniffed
	// as a house and would not load, art that will not decode -- and that is not
	// fatal: the shell shows it and stays up, because the player's next move is to
	// pick a different house (docs/IMPROVEMENTS.md 2.33).
	Play func(Choice) (Outcome, error)

	// Title sets the window caption. Optional.
	Title func(string)

	// Notify puts one line where a developer will see it, typically stderr.
	// Optional; the status line is where a player sees things.
	Notify func(string)

	// Version is shown on the status line.
	Version string
}

// Choice is what the shell asks Play for.
type Choice struct {
	House     House
	TwoPlayer bool
}

// Outcome is what came back. Score and StarsLeft are the two numbers the status
// line shows and the two 1.7c will put on a high-score board; Frames is the
// simulation's own length, which is the honest measure of a game rather than the
// wall clock.
type Outcome struct {
	Score     int32
	StarsLeft int16
	Frames    int64

	// Closed says the window went away during the game. The shell quits rather
	// than coming back to a title screen nobody can see.
	Closed bool
}

// mode is which screen is up. It is not the original's theMode -- that is
// SplashMode/PlayMode/EditMode and lives on the World -- but the finer state the
// original kept in its modal dialog nesting.
type mode int

const (
	modeSplash mode = iota
	modeHouses
	modeAbout
)

// Shell is one title screen, with its state.
type Shell struct {
	host Host
	lib  *Library

	mode mode
	cur  int // the selected house, an index into lib.Houses, or -1 for none
	sel  int // the menu cursor
	pick int // the picker's cursor, an index into lib.Houses
	msg  string
	quit bool

	// Frames counts passes through the loop. The tests use it as a clock, and it is
	// the only way to tell from outside that the shell is alive.
	Frames int64
}

// New checks the host over and returns a shell sitting on the splash screen.
//
// lib may be empty. A shell with no houses is not a broken shell: it comes up, says
// what is missing on the status line, and refuses to start a game -- which is what
// somebody who has cloned the repository and not run `make assets` should meet,
// rather than an error on a terminal they may not be looking at.
func New(h Host, lib *Library) (*Shell, error) {
	switch {
	case h.Screen == nil:
		return nil, errors.New("shell: no screen")
	case h.Screen.W != screenWide || h.Screen.H != screenTall:
		return nil, fmt.Errorf("shell: screen is %dx%d, want %dx%d",
			h.Screen.W, h.Screen.H, screenWide, screenTall)
	case h.Present == nil:
		return nil, errors.New("shell: no Present hook")
	case h.Poll == nil:
		return nil, errors.New("shell: no Poll hook")
	case h.Play == nil:
		return nil, errors.New("shell: no Play hook")
	}
	if lib == nil {
		lib = &Library{}
	}

	s := &Shell{host: h, lib: lib, cur: -1}
	if len(lib.Houses) > 0 {
		s.cur = 0
	}
	s.msg = s.opening()
	s.setTitle()
	return s, nil
}

// Select points the shell at a house by name, as the preferences file and the
// command line both want to. An unknown name leaves the selection alone and says
// so, which is the honest answer for a saved preference naming a house that has
// since been deleted.
func (s *Shell) Select(name string) bool {
	i := s.lib.Find(name)
	if i < 0 {
		return false
	}
	s.cur, s.pick = i, i
	s.setTitle()
	return true
}

// Show puts one named screen up without a keypress, which is what -shot needs:
// drawing the picker on a machine with no display should not require somebody to have
// pressed L. The names are the ones the flag documents, and an unknown one is an error
// rather than a silent splash screen -- a screenshot of the wrong screen is worse than
// none, because it looks like it worked.
func (s *Shell) Show(screen string) error {
	switch screen {
	case "splash":
		s.mode = modeSplash
	case "houses", "picker":
		s.openPicker()
		if s.mode != modeHouses {
			return fmt.Errorf("shell: cannot show the house picker: %s", s.msg)
		}
	case "about":
		s.mode = modeAbout
	default:
		return fmt.Errorf("shell: no screen called %q (splash, houses or about)", screen)
	}
	return nil
}

// House returns the selected house and whether there is one.
func (s *Shell) House() (House, bool) {
	if s.cur < 0 || s.cur >= len(s.lib.Houses) {
		return House{}, false
	}
	return s.lib.Houses[s.cur], true
}

// Run draws, presents and handles events until something asks to quit. It returns
// nil for an ordinary quit; the only errors it can return are a host's.
func (s *Shell) Run() error {
	for !s.quit {
		s.Draw()
		s.host.Present()
		s.Frames++
		for _, ev := range s.host.Poll() {
			s.event(ev)
			if s.quit {
				break
			}
		}
		if s.host.Idle != nil {
			s.host.Idle()
		}
	}
	return nil
}

// event routes one host event.
//
// Key *presses* only, and repeats are dropped for everything except the two
// navigation keys: a held Return on the splash screen would otherwise start a game,
// end it, and start another. The original has the same problem and solves it by
// blocking until the key is physically released (WaitCommandQReleased, Play.c:255),
// which docs/IMPROVEMENTS.md 2.15 argues a modern build should not do.
func (s *Shell) event(ev platform.Event) {
	switch ev.Kind {
	case platform.EventQuit:
		s.quit = true
	case platform.EventKeyDown:
		if ev.Repeat && ev.Key != platform.KeyUp && ev.Key != platform.KeyDown {
			return
		}
		switch s.mode {
		case modeSplash:
			s.splashKey(ev.Key)
		case modeHouses:
			s.pickerKey(ev.Key)
		case modeAbout:
			s.mode = modeSplash
		}
	}
	// Focus, expose and resize need nothing: the shell composes the whole screen
	// and presents it every pass, so a damaged window is repaired by the next one.
	// That is the luxury of a static screen -- a game cannot do it and does not
	// (see cmd/glidergo's PlayEvent, and docs/IMPROVEMENTS.md 2.27).
}

// ---------------------------------------------------------------------------
// The menu
// ---------------------------------------------------------------------------

// item is one menu row: its letter, its label, whether it can be chosen now, and
// what it does.
type item struct {
	key   platform.Key
	label string
	ok    bool
	do    func()
}

// menu is this shell's UpdateMenus (Menu.c:62-96, §3.5.2, and P8's advice to port
// it as one function): the single place that decides what is available.
//
// It is rebuilt on every draw rather than cached and patched, which is the one real
// improvement over the original's arrangement. UpdateMenus is called from more than
// twenty sites after any state change, and the bug that shape invites is a state
// change with no call after it -- an item that stays enabled because somebody forgot.
// Deriving the whole menu from the state at draw time cannot get out of step.
func (s *Shell) menu() []item {
	_, have := s.House()
	return []item{
		{key: platform.KeyN, label: "New Game", ok: have, do: func() { s.play(false) }},
		{key: platform.Key2, label: "Two Player Game", ok: have, do: func() { s.play(true) }},
		{key: platform.KeyL, label: "Load House...", ok: len(s.lib.Houses) > 0, do: s.openPicker},
		{key: platform.KeyA, label: "About...", ok: true, do: func() { s.mode = modeAbout }},
		{key: platform.KeyQ, label: "Quit", ok: true, do: func() { s.quit = true }},
	}
}

func (s *Shell) splashKey(k platform.Key) {
	m := s.menu()
	switch k {
	case platform.KeyUp:
		s.sel = (s.sel + len(m) - 1) % len(m)
		return
	case platform.KeyDown:
		s.sel = (s.sel + 1) % len(m)
		return
	case platform.KeyReturn, platform.KeySpace:
		s.choose(m, s.sel)
		return
	case platform.KeyEscape:
		s.quit = true
		return
	}
	for i := range m {
		if m[i].key == k {
			s.sel = i
			s.choose(m, i)
			return
		}
	}
}

// choose runs an item, or explains why it cannot run. The original's answer to the
// second case is a menu item drawn in grey and a keystroke that does nothing at
// all, which leaves a player with no idea what is wrong.
func (s *Shell) choose(m []item, i int) {
	if i < 0 || i >= len(m) {
		return
	}
	if !m[i].ok {
		s.msg = s.why(m[i].label)
		return
	}
	m[i].do()
}

func (s *Shell) why(label string) string {
	if len(s.lib.Houses) == 0 {
		return "no houses in " + s.libRoot() + " -- run `make assets`"
	}
	return label + " is not available"
}

func (s *Shell) libRoot() string {
	if s.lib.Root == "" {
		return "the houses directory"
	}
	return s.lib.Root
}

// play hands a game to the host and comes back when it is over.
//
// Everything about the game happens inside that call, including several minutes of
// somebody playing: the shell is not drawing, not polling and not presenting while
// it runs, because the game is doing all three with the same window. That is
// exactly the original's arrangement -- NewGame does not return until PlayGame does
// (Play.c:219-221) -- and it is why the shell has no state to save across it.
func (s *Shell) play(two bool) {
	h, ok := s.House()
	if !ok {
		s.msg = s.why("New Game")
		return
	}
	s.mode = modeSplash
	s.msg = "playing " + h.Name + "..."

	out, err := s.host.Play(Choice{House: h, TwoPlayer: two})
	if err != nil {
		s.msg = h.Name + ": " + err.Error()
		s.notify("glidergo: " + h.Name + ": " + err.Error())
		return
	}
	if out.Closed {
		s.quit = true
		return
	}
	s.msg = fmt.Sprintf("%s -- score %d, %d stars left", h.Name, out.Score, out.StarsLeft)
}

func (s *Shell) notify(line string) {
	if s.host.Notify != nil {
		s.host.Notify(line)
	}
}

func (s *Shell) setTitle() {
	if s.host.Title == nil {
		return
	}
	if h, ok := s.House(); ok {
		s.host.Title("gliderGo -- " + h.Name)
		return
	}
	s.host.Title("gliderGo")
}

// opening is the status line a fresh shell comes up with: the one place a player is
// told what is missing before they have pressed anything.
func (s *Shell) opening() string {
	switch {
	case len(s.lib.Houses) == 0:
		return "no houses in " + s.libRoot() + " -- run `make assets`"
	case len(s.lib.Skipped) > 0:
		return fmt.Sprintf("%d houses, %d files skipped -- press L to see them",
			len(s.lib.Houses), len(s.lib.Skipped))
	default:
		return fmt.Sprintf("%d houses", len(s.lib.Houses))
	}
}

// ---------------------------------------------------------------------------
// The house picker
// ---------------------------------------------------------------------------

func (s *Shell) openPicker() {
	if len(s.lib.Houses) == 0 {
		s.msg = s.why("Load House...")
		return
	}
	s.mode = modeHouses
	if s.cur >= 0 {
		s.pick = s.cur
	}
	s.clampPick()
}

func (s *Shell) clampPick() {
	if s.pick < 0 {
		s.pick = 0
	}
	if s.pick >= len(s.lib.Houses) {
		s.pick = len(s.lib.Houses) - 1
	}
}

// pickerKey is LoadFilter (SelectHouse.c:435-514, §7.8) with the mouse taken out:
// the arrows move, Return chooses, Escape leaves, and a letter jumps to the next
// house that starts with it -- which is the original's type-select, kept because it
// is the only thing that makes a list of forty houses navigable with a keyboard.
func (s *Shell) pickerKey(k platform.Key) {
	n := len(s.lib.Houses)
	if n == 0 {
		s.mode = modeSplash
		return
	}
	switch k {
	case platform.KeyUp:
		s.pick = (s.pick + n - 1) % n
	case platform.KeyDown:
		s.pick = (s.pick + 1) % n
	case platform.KeyLeft:
		s.pick -= pickerRows
		if s.pick < 0 {
			s.pick = 0
		}
	case platform.KeyRight:
		s.pick += pickerRows
		if s.pick >= n {
			s.pick = n - 1
		}
	case platform.KeyReturn:
		s.commitPick()
		s.play(false)
	case platform.KeySpace:
		s.commitPick()
		s.mode = modeSplash
	case platform.KeyEscape, platform.KeyTab:
		s.mode = modeSplash
	default:
		if c, ok := letterOf(k); ok {
			s.typeSelect(c)
		}
	}
}

func (s *Shell) commitPick() {
	s.clampPick()
	s.cur = s.pick
	s.mode = modeSplash
	s.sel = 0
	s.setTitle()
	if h, ok := s.House(); ok {
		s.msg = fmt.Sprintf("%s -- %d rooms", h.Name, h.Rooms)
	}
}

// typeSelect moves to the next house whose name starts with c, wrapping. The
// original keeps a fileFirstChar[12] table of the upper-cased first letters of the
// twelve visible names and searches only those (§7.8); searching the whole list is
// the same idea without the paging accident, so a letter finds a house that is not
// on this page.
func (s *Shell) typeSelect(c byte) {
	n := len(s.lib.Houses)
	want := upperASCII(c)
	for i := 1; i <= n; i++ {
		j := (s.pick + i) % n
		name := s.lib.Houses[j].Name
		if name != "" && upperASCII(name[0]) == want {
			s.pick = j
			return
		}
	}
}

// letterOf maps a key back to the character it would type, for type-select. Only
// the letters and digits, and no shift state: the original upper-cases both sides
// of the comparison too.
func letterOf(k platform.Key) (byte, bool) {
	switch {
	case k >= platform.KeyA && k <= platform.KeyZ:
		return byte('A' + int(k-platform.KeyA)), true
	case k >= platform.Key0 && k <= platform.Key9:
		return byte('0' + int(k-platform.Key0)), true
	}
	return 0, false
}
