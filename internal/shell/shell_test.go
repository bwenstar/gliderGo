package shell

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/platform"
	"github.com/bwenstar/gliderGo/internal/prefs"
	"github.com/bwenstar/gliderGo/internal/render"
)

// The shell is driven entirely through its Host, which is what makes it testable
// without a window, a house, a sound card or a game. fake is that host: a real
// surface to draw on, a script of key events handed out one pass at a time, and a
// Play that records what it was asked for instead of playing anything.
type fake struct {
	scr    *render.Surface
	script [][]platform.Event
	pass   int

	presents int
	plays    []Choice
	out      Outcome
	err      error
	titles   []string
	notes    []string

	// The preferences the settings screen edits. host() fills this in with the
	// defaults if a test has not put anything there, because a shell with no Prefs is
	// a shell with the Settings item switched off -- see noPrefs.
	prefs   *prefs.Prefs
	noPrefs bool

	saves   int
	saveErr error
	noSave  bool
	applies int
	noApply bool
}

func (f *fake) host() Host {
	if f.prefs == nil && !f.noPrefs {
		f.prefs = prefs.Default()
	}
	h := Host{
		Screen:  f.scr,
		Present: func() { f.presents++ },
		Poll: func() []platform.Event {
			// Past the end of the script, close the window. Without this a shell
			// that ignores a key it should have acted on would hang the test run
			// instead of failing it.
			if f.pass >= len(f.script) {
				f.pass++
				return []platform.Event{{Kind: platform.EventQuit}}
			}
			evs := f.script[f.pass]
			f.pass++
			return evs
		},
		Play: func(c Choice) (Outcome, error) {
			f.plays = append(f.plays, c)
			return f.out, f.err
		},
		Title:   func(s string) { f.titles = append(f.titles, s) },
		Notify:  func(s string) { f.notes = append(f.notes, s) },
		Prefs:   f.prefs,
		Version: "test",
	}
	if !f.noSave {
		h.SavePrefs = func() error {
			f.saves++
			return f.saveErr
		}
	}
	if !f.noApply {
		h.ApplyPrefs = func() { f.applies++ }
	}
	return h
}

// key is one press; rep is one auto-repeat of a held key.
func key(k platform.Key) []platform.Event {
	return []platform.Event{{Kind: platform.EventKeyDown, Key: k}}
}

func rep(k platform.Key) []platform.Event {
	return []platform.Event{{Kind: platform.EventKeyDown, Key: k, Repeat: true}}
}

// shellOver builds a shell over houses with these names and no files behind them.
// Discover is tested separately; this is about what the shell does with a list.
func shellOver(t *testing.T, names []string, script ...[]platform.Event) (*Shell, *fake) {
	t.Helper()
	lib := &Library{Root: filepath.Join("test", "houses")}
	for i, n := range names {
		lib.Houses = append(lib.Houses, House{
			Name:  n,
			Rel:   n + ".house",
			Path:  filepath.Join(lib.Root, n+".house"),
			Rooms: int16(i + 1),
		})
	}
	lib.Sort()

	f := &fake{scr: render.NewSurface(screenWide, screenTall), script: script}
	s, err := New(f.host(), lib)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s, f
}

func TestNewChecksItsHost(t *testing.T) {
	full := func() Host {
		f := &fake{scr: render.NewSurface(screenWide, screenTall)}
		return f.host()
	}
	if _, err := New(Host{}, nil); err == nil {
		t.Error("a host with no screen should be rejected")
	}
	small := full()
	small.Screen = render.NewSurface(320, 200)
	if _, err := New(small, nil); err == nil {
		t.Error("a screen of the wrong size should be rejected")
	}
	noPlay := full()
	noPlay.Play = nil
	if _, err := New(noPlay, nil); err == nil {
		t.Error("a host with no Play hook should be rejected")
	}
	// A nil library is not a broken host: it is a first run.
	if _, err := New(full(), nil); err != nil {
		t.Errorf("a nil library should be accepted: %v", err)
	}
}

func TestRunPresentsEveryPassAndQuitsOnEscape(t *testing.T) {
	s, f := shellOver(t, []string{"Slumberland"},
		nil, nil, key(platform.KeyEscape), key(platform.KeyN))

	if err := s.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if s.Frames != 3 || f.presents != 3 {
		t.Errorf("ran %d passes and presented %d, want 3 and 3 -- Escape should end Run "+
			"before the fourth pass reads the N", s.Frames, f.presents)
	}
	if len(f.plays) != 0 {
		t.Errorf("nothing should have been played, got %v", f.plays)
	}
}

func TestRunQuitsOnWindowClose(t *testing.T) {
	s, _ := shellOver(t, []string{"Slumberland"},
		[]platform.Event{{Kind: platform.EventQuit}})
	if err := s.Run(); err != nil {
		t.Fatal(err)
	}
	if s.Frames != 1 {
		t.Errorf("ran %d passes, want 1", s.Frames)
	}
}

// The arrows move the menu cursor and Return chooses, which is the one deliberate
// departure from the original's arcade key map (see the package comment).
func TestArrowsMoveTheMenuAndReturnChooses(t *testing.T) {
	// Counted rather than hard-coded, so that adding a menu item is not a test failure
	// in a test about the arrow keys.
	s, f := shellOver(t, []string{"Slumberland"})
	at := -1
	for i, it := range s.menu() {
		if it.label == "About..." {
			at = i
		}
	}
	if at < 1 {
		t.Fatalf("About is at menu index %d; this test needs it below the first item", at)
	}
	for i := 0; i < at; i++ {
		s.event(key(platform.KeyDown)[0])
	}
	s.event(key(platform.KeyReturn)[0])
	if s.mode != modeAbout {
		t.Errorf("mode is %v after %d Downs and a Return, want the About box", s.mode, at)
	}
	if len(f.plays) != 0 {
		t.Error("About should not start a game")
	}
}

func TestMenuCursorWrapsBothWays(t *testing.T) {
	s, _ := shellOver(t, []string{"Slumberland"}, key(platform.KeyUp))
	s.event(key(platform.KeyUp)[0])
	if want := len(s.menu()) - 1; s.sel != want {
		t.Errorf("Up from the first item put the cursor on %d, want %d (it should wrap)", s.sel, want)
	}
	s.event(key(platform.KeyDown)[0])
	if s.sel != 0 {
		t.Errorf("Down from the last item put the cursor on %d, want 0", s.sel)
	}
}

func TestAcceleratorsStartGames(t *testing.T) {
	s, f := shellOver(t, []string{"Slumberland"}, key(platform.KeyN))
	if err := s.Run(); err != nil {
		t.Fatal(err)
	}
	if len(f.plays) != 1 {
		t.Fatalf("N played %d games, want 1", len(f.plays))
	}
	if got := f.plays[0].House.Name; got != "Slumberland" {
		t.Errorf("played %q, want Slumberland", got)
	}
	if f.plays[0].TwoPlayer {
		t.Error("N should start a one-player game")
	}

	s2, f2 := shellOver(t, []string{"Slumberland"}, key(platform.Key2))
	if err := s2.Run(); err != nil {
		t.Fatal(err)
	}
	if len(f2.plays) != 1 || !f2.plays[0].TwoPlayer {
		t.Errorf("2 should start a two-player game, got %v", f2.plays)
	}
}

func TestQuitAccelerator(t *testing.T) {
	s, _ := shellOver(t, []string{"Slumberland"}, key(platform.KeyQ), key(platform.KeyN))
	if err := s.Run(); err != nil {
		t.Fatal(err)
	}
	if s.Frames != 1 {
		t.Errorf("Q ended the shell after %d passes, want 1", s.Frames)
	}
}

// A held key must not start a game over and over. Held arrows must still move.
func TestKeyRepeatsAreIgnoredForActionsAndHonouredForArrows(t *testing.T) {
	s, f := shellOver(t, []string{"Slumberland"},
		rep(platform.KeyReturn), rep(platform.KeyReturn), rep(platform.KeyDown))
	if err := s.Run(); err != nil {
		t.Fatal(err)
	}
	if len(f.plays) != 0 {
		t.Errorf("a repeated Return played %d games, want 0", len(f.plays))
	}
	if s.sel != 1 {
		t.Errorf("a repeated Down left the cursor on %d, want 1", s.sel)
	}
}

func TestAboutBoxIsDismissedByAnyKey(t *testing.T) {
	s, _ := shellOver(t, []string{"Slumberland"}, key(platform.KeyA), key(platform.KeyX))
	if err := s.Run(); err != nil {
		t.Fatal(err)
	}
	if s.mode != modeSplash {
		t.Errorf("mode is %v after a key in the About box, want the splash screen", s.mode)
	}
}

// An empty library must not be a dead end with no explanation: every item that needs a
// house is unavailable, and pressing one says why.
func TestEmptyLibraryExplainsItself(t *testing.T) {
	s, f := shellOver(t, nil, key(platform.KeyN), key(platform.KeyL))
	if err := s.Run(); err != nil {
		t.Fatal(err)
	}
	if len(f.plays) != 0 {
		t.Errorf("played %d games with no houses, want 0", len(f.plays))
	}
	if !strings.Contains(s.msg, "make assets") {
		t.Errorf("status line is %q; it should say how to get some houses", s.msg)
	}
	if s.mode != modeSplash {
		t.Errorf("the picker should not open with nothing in it (mode %v)", s.mode)
	}
	for _, it := range s.menu() {
		switch it.label {
		case "New Game", "Two Player Game", "Open Saved Game...", "Load House...", "High Scores...":
			if it.ok {
				t.Errorf("%q should be unavailable with no houses", it.label)
			}
		default:
			if !it.ok {
				t.Errorf("%q should always be available", it.label)
			}
		}
	}
}

func TestPickerNavigatesSelectsAndCancels(t *testing.T) {
	names := []string{"Amy", "Beth", "Cass", "Dot"}

	// L opens the picker, Down moves to Beth, Space selects and closes it.
	s, f := shellOver(t, names, key(platform.KeyL), key(platform.KeyDown), key(platform.KeySpace))
	if err := s.Run(); err != nil {
		t.Fatal(err)
	}
	if s.mode != modeSplash {
		t.Errorf("Space should close the picker, mode is %v", s.mode)
	}
	if h, _ := s.House(); h.Name != "Beth" {
		t.Errorf("selected %q, want Beth", h.Name)
	}
	if len(f.plays) != 0 {
		t.Error("Space selects a house; it does not start a game")
	}
	if got := f.titles[len(f.titles)-1]; !strings.Contains(got, "Beth") {
		t.Errorf("window title is %q; it should name the selected house", got)
	}

	// Escape leaves the selection alone.
	s2, _ := shellOver(t, names, key(platform.KeyL), key(platform.KeyDown), key(platform.KeyEscape))
	if err := s2.Run(); err != nil {
		t.Fatal(err)
	}
	if h, _ := s2.House(); h.Name != "Amy" {
		t.Errorf("Escape changed the selection to %q; it should still be Amy", h.Name)
	}
	if s2.mode != modeSplash {
		t.Errorf("Escape should close the picker, mode is %v", s2.mode)
	}
}

func TestPickerReturnPlaysTheHighlightedHouse(t *testing.T) {
	s, f := shellOver(t, []string{"Amy", "Beth", "Cass"},
		key(platform.KeyL), key(platform.KeyDown), key(platform.KeyDown), key(platform.KeyReturn))
	if err := s.Run(); err != nil {
		t.Fatal(err)
	}
	if len(f.plays) != 1 {
		t.Fatalf("played %d games, want 1", len(f.plays))
	}
	if got := f.plays[0].House.Name; got != "Cass" {
		t.Errorf("played %q, want Cass", got)
	}
	if s.mode != modeSplash {
		t.Errorf("mode is %v after a game, want the splash screen", s.mode)
	}
}

func TestPickerTypeSelectAndPaging(t *testing.T) {
	names := make([]string, 0, 25)
	for i := 0; i < 12; i++ {
		names = append(names, string(rune('A'+i))+"nnex")
	}
	names = append(names, "Zoo", "zebra")

	s, _ := shellOver(t, names)
	s.openPicker()

	s.typeSelect('z')
	if got := s.lib.Houses[s.pick].Name; got != "zebra" && got != "Zoo" {
		t.Errorf("typing z landed on %q, want Zoo or zebra -- the fold is case-insensitive", got)
	}
	first := s.lib.Houses[s.pick].Name
	s.typeSelect('z')
	if second := s.lib.Houses[s.pick].Name; second == first {
		t.Errorf("typing z twice stayed on %q; it should advance to the other z", first)
	}

	// A letter must find a house that is not on the current page: that is the whole
	// reason for searching the list rather than the twelve visible names.
	s.pick = 0
	s.typeSelect('L')
	if got := s.lib.Houses[s.pick].Name; got != "Lnnex" {
		t.Errorf("typing L from the first page landed on %q, want Lnnex", got)
	}
	if s.pick/pickerRows == 0 {
		t.Errorf("Lnnex is at index %d, which should not be on the first page", s.pick)
	}

	// Left and Right are the page keys, and they clamp rather than wrap.
	s.pick = 0
	s.pickerKey(platform.KeyLeft)
	if s.pick != 0 {
		t.Errorf("Left from the first row moved to %d, want 0", s.pick)
	}
	s.pick = len(names) - 1
	s.pickerKey(platform.KeyRight)
	if s.pick != len(names)-1 {
		t.Errorf("Right from the last row moved to %d, want %d", s.pick, len(names)-1)
	}
	s.pick = 0
	s.pickerKey(platform.KeyRight)
	if s.pick != pickerRows {
		t.Errorf("Right moved to %d, want a whole page down (%d)", s.pick, pickerRows)
	}
}

// A game that cannot be started is not a reason to take the title screen away. The
// player's next move is to pick a different house, so the shell has to still be
// there to pick it with.
func TestPlayErrorIsShownAndTheShellStaysUp(t *testing.T) {
	s, f := shellOver(t, []string{"Slumberland"}, key(platform.KeyN), key(platform.KeyL))
	f.err = errors.New("2 bytes past the last of 40 rooms")

	if err := s.Run(); err != nil {
		t.Fatalf("a Play error must not come out of Run: %v", err)
	}
	if !strings.Contains(s.msg, "past the last") || !strings.Contains(s.msg, "Slumberland") {
		t.Errorf("status line is %q; it should name the house and the reason", s.msg)
	}
	if len(f.notes) != 1 {
		t.Errorf("the failure should also reach the log, got %v", f.notes)
	}
	// And the next keypress still works: the player can go straight to choosing a
	// different house, which is the reason for keeping the shell up at all.
	if s.mode != modeHouses {
		t.Errorf("L after the failure left the mode at %v, want the picker open", s.mode)
	}
	if !messageIsOnScreen(f.scr) {
		t.Error("the status line was not actually drawn")
	}
}

// The end-to-end version of the same thing, with a real file: PeekFile accepts it,
// so the picker lists it, and Load refuses it, so the shell has to report that
// rather than pretend the house is not there.
func TestHouseThatSniffsButWillNotLoad(t *testing.T) {
	root := t.TempDir()
	b := houseBytes(t, 1)
	b = append(b, 1, 2, 3) // three trailing bytes: not 0 and not the PowerPC slack
	if err := os.WriteFile(filepath.Join(root, "Wonky.house"), b, 0o644); err != nil {
		t.Fatal(err)
	}

	lib, err := Discover(os.DirFS(root), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(lib.Houses) != 1 {
		t.Fatalf("the picker should list a house that sniffs as one; found %d", len(lib.Houses))
	}

	f := &fake{scr: render.NewSurface(screenWide, screenTall), script: [][]platform.Event{key(platform.KeyN)}}
	h := f.host()
	h.Play = func(c Choice) (Outcome, error) {
		_, err := lib.Open(c.House)
		return Outcome{}, err
	}
	s, err := New(h, lib)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(s.msg, "Wonky") || !strings.Contains(s.msg, "3 bytes past") {
		t.Errorf("status line is %q; it should name the house and quote the loader", s.msg)
	}
}

func TestOutcomeIsReportedAndAClosedWindowEndsTheShell(t *testing.T) {
	s, f := shellOver(t, []string{"Slumberland"}, key(platform.KeyN), key(platform.KeyN))
	f.out = Outcome{Score: 12345, StarsLeft: 7, Frames: 9000}
	if err := s.Run(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s.msg, "12345") || !strings.Contains(s.msg, "7") {
		t.Errorf("status line is %q; it should show the score and the stars left", s.msg)
	}

	s2, f2 := shellOver(t, []string{"Slumberland"}, key(platform.KeyN), key(platform.KeyN))
	f2.out = Outcome{Closed: true}
	if err := s2.Run(); err != nil {
		t.Fatal(err)
	}
	if s2.Frames != 1 {
		t.Errorf("a window closed during the game left the shell running for %d passes, want 1",
			s2.Frames)
	}
	if len(f2.plays) != 1 {
		t.Errorf("played %d games, want 1", len(f2.plays))
	}
}

func TestSelectByName(t *testing.T) {
	s, _ := shellOver(t, []string{"Amy", "Beth"})
	if !s.Select("beth") {
		t.Fatal("Select should match a name case-insensitively")
	}
	if h, _ := s.House(); h.Name != "Beth" {
		t.Errorf("selected %q, want Beth", h.Name)
	}
	if s.Select("Nowhere") {
		t.Error("Select should report an unknown name")
	}
	if h, _ := s.House(); h.Name != "Beth" {
		t.Errorf("a failed Select changed the selection to %q", h.Name)
	}
}

// ---------------------------------------------------------------------------
// The settings screen
// ---------------------------------------------------------------------------

// onSettings opens the settings screen the way a player does and hands back the shell,
// the host and the live preferences the screen is editing.
func onSettings(t *testing.T, script ...[]platform.Event) (*Shell, *fake, *prefs.Prefs) {
	t.Helper()
	s, f := shellOver(t, []string{"Slumberland"}, script...)
	s.openSettings()
	if s.mode != modeSettings {
		t.Fatalf("the settings screen did not open: %s", s.msg)
	}
	return s, f, f.prefs
}

// rowOf finds a row by its group and label, so the tests do not carry indices that move
// every time a setting is added.
func rowOf(t *testing.T, group, label string) int {
	t.Helper()
	in := ""
	for i, r := range settings {
		if r.group != "" {
			in = r.group
		}
		if in == group && r.label == label {
			return i
		}
	}
	t.Fatalf("no %q row in the %s group", label, group)
	return -1
}

func TestSettingsOpensFromTheMenuAndClosesOnEscape(t *testing.T) {
	s, f := shellOver(t, []string{"Slumberland"}, key(platform.KeyS), key(platform.KeyEscape))
	if err := s.Run(); err != nil {
		t.Fatal(err)
	}
	if s.mode != modeSplash {
		t.Errorf("mode is %v after Escape, want the splash screen", s.mode)
	}
	// Nothing was changed, so nothing was written: opening the screen to look at it
	// must not rewrite the file.
	if f.saves != 0 {
		t.Errorf("the file was saved %d times by a visit that changed nothing", f.saves)
	}
}

// A build with no preferences file says so instead of offering a screen that could not
// keep anything. -shot and the fidelity replays are exactly that build.
func TestSettingsUnavailableWithoutPrefs(t *testing.T) {
	f := &fake{
		scr:     render.NewSurface(screenWide, screenTall),
		script:  [][]platform.Event{key(platform.KeyS)},
		noPrefs: true,
	}
	s, err := New(f.host(), &Library{Houses: []House{{Name: "Slumberland"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Run(); err != nil {
		t.Fatal(err)
	}
	if s.mode != modeSplash {
		t.Errorf("mode is %v; the settings screen must not open with no preferences", s.mode)
	}
	if !strings.Contains(s.msg, "preferences") {
		t.Errorf("status line is %q; it should say why the item did nothing", s.msg)
	}
	for _, it := range s.menu() {
		if it.label == "Settings..." && it.ok {
			t.Error("Settings should be unavailable with no preferences file")
		}
	}
	if err := s.Show("settings"); err == nil {
		t.Error("Show(settings) should fail rather than draw an empty screen")
	}
}

func TestSettingsRebindsAKey(t *testing.T) {
	s, f, p := onSettings(t)
	s.set = rowOf(t, "Player One", "steer left")

	s.settingsKey(platform.KeyReturn)
	if s.capture != s.set {
		t.Fatalf("Return left capture at %d, want %d", s.capture, s.set)
	}
	if !strings.Contains(s.msg, "press a key") {
		t.Errorf("status line is %q; it should ask for a key", s.msg)
	}

	s.settingsKey(platform.KeyZ)
	if s.capture != -1 {
		t.Error("the capture should end with the keystroke that answered it")
	}
	if p.Player1.Left != "z" {
		t.Errorf("player one's left key is %q, want z", p.Player1.Left)
	}
	if !s.setDirty {
		t.Error("a rebind is a change and should be saved on the way out")
	}
	if f.applies == 0 {
		t.Error("every change should be handed to ApplyPrefs")
	}
}

// The three keys the screen keeps for itself are the three Validate reserves, which is
// what lets it capture anything else without asking what the key is for.
func TestSettingsCaptureCancelsAndClears(t *testing.T) {
	s, _, p := onSettings(t)
	s.set = rowOf(t, "Player One", "steer left")

	s.settingsKey(platform.KeyReturn)
	s.settingsKey(platform.KeyEscape)
	if s.capture != -1 {
		t.Error("Escape should end the capture")
	}
	if s.mode != modeSettings {
		t.Errorf("mode is %v; Escape during a capture cancels the capture, not the screen", s.mode)
	}
	if p.Player1.Left != "left" {
		t.Errorf("a cancelled capture changed the binding to %q", p.Player1.Left)
	}
	if s.setDirty {
		t.Error("a cancelled capture is not a change")
	}

	s.settingsKey(platform.KeyReturn)
	s.settingsKey(platform.KeyDelete)
	if p.Player1.Left != prefs.Unbound {
		t.Errorf("Delete left the binding at %q, want %q", p.Player1.Left, prefs.Unbound)
	}
	if !strings.Contains(s.msg, "before playing") {
		t.Errorf("status line is %q; an unbound control should say it needs setting", s.msg)
	}
	// An unbound binding survives a revalidation silently: it is where an unresolvable
	// collision ends up, so complaining about it on every pass would be noise.
	s.changed(p, "")
	if p.Player1.Left != prefs.Unbound {
		t.Errorf("revalidating moved the unbound binding to %q", p.Player1.Left)
	}
}

// A collision is reported by the same function that repairs the file, on the status line,
// next to the keystroke that caused it.
func TestSettingsCollisionIsReportedByValidate(t *testing.T) {
	s, _, p := onSettings(t)
	s.set = rowOf(t, "Player One", "steer left")

	// A is player two's left thruster at the defaults. Player one is checked first, so
	// player one keeps it and player two -- whose own default is the key just taken --
	// is left unbound rather than moved onto some third key nobody chose.
	s.settingsKey(platform.KeyReturn)
	s.settingsKey(platform.KeyA)
	if p.Player1.Left != "a" {
		t.Fatalf("player one's left key is %q, want a", p.Player1.Left)
	}
	if p.Player2.Left != prefs.Unbound {
		t.Errorf("player two's left key is %q, want %q", p.Player2.Left, prefs.Unbound)
	}
	if !strings.Contains(s.msg, "player two") {
		t.Errorf("status line is %q; it should say what the collision cost", s.msg)
	}
	// The note belongs to this screen, not to whoever prints Notes next: a session of
	// rebinding would otherwise leave a hundred of them on the Prefs.
	if len(p.Notes) != 0 {
		t.Errorf("Validate's notes were left on the Prefs: %v", p.Notes)
	}
}

// The way back from a keyboard somebody has locked themselves out of.
func TestSettingsResetRestoresTheDefaults(t *testing.T) {
	s, _, p := onSettings(t)
	p.Player1.Left = "z"
	p.Player2 = prefs.Controls{}
	p.PauseKey = "escape"
	p.Volume, p.Neighbors, p.Scale = 0, 1, 4
	p.MusicInGame = false

	s.settingsKey(platform.KeyR)

	d := prefs.Default()
	if p.Player1 != d.Player1 || p.Player2 != d.Player2 {
		t.Errorf("R left the bindings at %+v %+v", p.Player1, p.Player2)
	}
	if p.PauseKey != d.PauseKey || p.Neighbors != d.Neighbors || p.Scale != d.Scale {
		t.Errorf("R left pause %q, neighbors %d, scale %d", p.PauseKey, p.Neighbors, p.Scale)
	}
	if p.Volume != d.Volume || !p.MusicInGame {
		t.Errorf("R left volume %d, music %v", p.Volume, p.MusicInGame)
	}
	if !p.Sound {
		t.Error("the volume came back up and the sound did not: isSoundOn is derived from it")
	}
	// The house and the high-score name are not settings this screen shows, so R must
	// not take them: a player fixing their key bindings has not asked to lose either.
	if p.House != d.House {
		t.Errorf("R changed the house to %q", p.House)
	}
}

func TestSettingsValuesStepAndClamp(t *testing.T) {
	s, _, p := onSettings(t)

	// The volume is a range and clamps at both ends rather than wrapping: an arrow held
	// down to reach silence must not come back round at full blast.
	s.set = rowOf(t, "Sound", "volume")
	for i := 0; i < prefs.MaxVolume+3; i++ {
		s.settingsKey(platform.KeyLeft)
	}
	if p.Volume != 0 {
		t.Errorf("volume is %d after ten Lefts, want 0", p.Volume)
	}
	if p.Sound {
		t.Error("volume 0 with the sound on: one of them is lying")
	}
	s.settingsKey(platform.KeyRight)
	if p.Volume != 1 || !p.Sound {
		t.Errorf("volume %d sound %v after one Right, want 1 and true", p.Volume, p.Sound)
	}

	// Rooms in view is a set of three and wraps, because there is no "between" to clamp
	// against -- numNeighbors picks one of three composition paths, not a radius.
	s.set = rowOf(t, "General", "rooms in view")
	seen := map[int]bool{}
	for i := 0; i < 4; i++ {
		seen[p.Neighbors] = true
		s.settingsKey(platform.KeyRight)
	}
	for _, want := range []int{1, 3, 9} {
		if !seen[want] {
			t.Errorf("stepping four times never showed %d rooms (saw %v)", want, seen)
		}
	}

	// The pause key is the original's own binary and Return toggles it, because a row
	// with two states has nothing for left and right to mean separately.
	s.set = rowOf(t, "General", "pause key")
	s.settingsKey(platform.KeyReturn)
	if p.PauseKey != "escape" {
		t.Errorf("pause key is %q after a Return, want escape", p.PauseKey)
	}
	if s.capture != -1 {
		t.Error("a value row must not start a rebind")
	}
	s.settingsKey(platform.KeyReturn)
	if p.PauseKey != "tab" {
		t.Errorf("pause key is %q after a second Return, want tab", p.PauseKey)
	}

	// Escape is reserved either way round, so switching the pause key onto Tab must not
	// leave a binding sitting on the key that now pauses.
	s.set = rowOf(t, "Player Two", "rubber band")
	s.settingsKey(platform.KeyReturn)
	s.settingsKey(platform.KeyTab)
	if p.Player2.Band == "tab" {
		t.Error("Tab was bound to a control while Tab is the pause key")
	}
}

func TestSettingsCursorWrapsAndCoversEveryRow(t *testing.T) {
	s, _, p := onSettings(t)
	s.settingsKey(platform.KeyUp)
	if s.set != len(settings)-1 {
		t.Errorf("Up from the first row went to %d, want %d", s.set, len(settings)-1)
	}
	s.settingsKey(platform.KeyDown)
	if s.set != 0 {
		t.Errorf("Down from the last row went to %d, want 0", s.set)
	}
	// Every row is one thing or the other, and every row can say what it holds. A row
	// that was neither would draw an empty value and do nothing when chosen.
	for i, r := range settings {
		if (r.bind == nil) == (r.step == nil) {
			t.Errorf("row %d (%q) is both a binding and a value, or neither", i, r.label)
			continue
		}
		if r.bind == nil && r.show == nil {
			t.Errorf("row %d (%q) has no way to show its value", i, r.label)
		}
		if got := s.rowValue(p, i); got == "" {
			t.Errorf("row %d (%q) shows nothing", i, r.label)
		}
	}
}

// The original wrote its preferences once, at quit, and lost them all on a crash. This
// writes on the way out of the screen, which is the last moment the player is looking.
func TestSettingsSaveOnClose(t *testing.T) {
	s, f, _ := onSettings(t)
	s.set = rowOf(t, "Sound", "volume")
	s.settingsKey(platform.KeyLeft)
	s.settingsKey(platform.KeyEscape)

	if s.mode != modeSplash {
		t.Errorf("mode is %v after Escape, want the splash screen", s.mode)
	}
	if f.saves != 1 {
		t.Errorf("the file was saved %d times, want 1", f.saves)
	}
	if s.setDirty {
		t.Error("the screen is still dirty after a successful save")
	}
	if !strings.Contains(s.msg, "saved") {
		t.Errorf("status line is %q; it should confirm the save", s.msg)
	}

	// A second visit that changes nothing does not write again.
	s.openSettings()
	s.settingsKey(platform.KeyEscape)
	if f.saves != 1 {
		t.Errorf("a visit that changed nothing saved again (%d saves)", f.saves)
	}

	// A failed save is reported on the status line *and* in the log, because a setting
	// that did not persist looks exactly like one that was never made.
	f.saveErr = errors.New("read-only file system")
	s.openSettings()
	s.settingsKey(platform.KeyLeft)
	s.settingsKey(platform.KeyEscape)
	if !strings.Contains(s.msg, "read-only") {
		t.Errorf("status line is %q; it should quote the error", s.msg)
	}
	if len(f.notes) == 0 {
		t.Error("a failed save should also reach the log")
	}
}

// A session with settings and nowhere to keep them -- `-prefs none`, or a read-only
// configuration directory -- says so rather than pretending it saved. This host also has
// no ApplyPrefs, which is the other optional hook: both absences are ordinary.
func TestSettingsWithNowhereToSave(t *testing.T) {
	f := &fake{scr: render.NewSurface(screenWide, screenTall), noSave: true, noApply: true}
	s, err := New(f.host(), &Library{Houses: []House{{Name: "Slumberland"}}})
	if err != nil {
		t.Fatal(err)
	}
	s.openSettings()
	if s.mode != modeSettings {
		t.Fatalf("the screen should still open with no way to save: %s", s.msg)
	}
	s.set = rowOf(t, "Sound", "volume")
	s.settingsKey(platform.KeyLeft)
	s.settingsKey(platform.KeyEscape)
	if !strings.Contains(s.msg, "session") {
		t.Errorf("status line is %q; it should say the change is not being kept", s.msg)
	}
}

// Every screen has to draw, with no artwork and no assets at all, without panicking
// and without leaving the screen blank. This is what `make headless` exercises for
// real; here it is the cheap version of the same check.
func TestEveryScreenDrawsWithoutArt(t *testing.T) {
	s, f := shellOver(t, []string{"Amy", "Beth", "Cass"})
	for _, tc := range []struct {
		name string
		mode mode
	}{
		{"splash", modeSplash},
		{"picker", modeHouses},
		{"settings", modeSettings},
		{"about", modeAbout},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for i := range f.scr.Pix {
				f.scr.Pix[i] = 0
			}
			s.mode = tc.mode
			s.Draw()

			if !messageIsOnScreen(f.scr) {
				t.Error("nothing was drawn in the status band")
			}
			if cream := count(f.scr, cream); cream < 200 {
				t.Errorf("only %d cream pixels on screen; the panel and its text are missing", cream)
			}
			if black := count(f.scr, render.Black8); black < 10000 {
				t.Errorf("only %d black pixels; the fallback title screen is missing", black)
			}
		})
	}
}

// The house label goes exactly where DrawOnSplash puts it, and a name too long for
// the screen slides left instead of running off the edge.
func TestHouseLabelPlacementAndClamping(t *testing.T) {
	s, f := shellOver(t, []string{"Amy"})
	s.Draw()
	if !inkNear(f.scr, houseLabelH, houseLabelV, label) {
		t.Errorf("no label-coloured ink near (%d,%d), where DrawOnSplash writes the house name",
			houseLabelH, houseLabelV)
	}

	long, _ := shellOver(t, []string{strings.Repeat("Wide", 30)})
	long.Draw()
	for v := houseLabelV - 8; v <= houseLabelV; v++ {
		for x := screenWide - 3; x < screenWide; x++ {
			if long.host.Screen.Pix[v*screenWide+x] == label {
				t.Fatalf("the label reaches column %d; it should have been slid left", x)
			}
		}
	}
	if !inkNear(long.host.Screen, 8, houseLabelV, label) {
		t.Error("a long name should have been slid left, leaving ink near the left edge")
	}
}

func TestFitShortensText(t *testing.T) {
	if got := fit("Slumberland", 4000, 1); got != "Slumberland" {
		t.Errorf("fit of a string that fits returned %q", got)
	}
	got := fit("Slumberland", render.StringWidth("Slumb..."), 1)
	if !strings.HasSuffix(got, "...") {
		t.Errorf("fit returned %q, want an ellipsis on the end", got)
	}
	if w := render.StringWidth(got); w > render.StringWidth("Slumb...") {
		t.Errorf("fit returned %q, which is %d wide -- wider than it was asked for", got, w)
	}
	if got := fit("Slumberland", 0, 1); got != "" {
		t.Errorf("fit into no space returned %q", got)
	}
	// Runes, not bytes: a truncated UTF-8 sequence would draw a replacement glyph.
	if got := fit("École de danse", 40, 1); !utf8Valid(got) {
		t.Errorf("fit produced invalid UTF-8: %q", got)
	}
}

func TestBestScoreReadsTheHousesOwnBoard(t *testing.T) {
	var h House
	if _, _, ok := h.Best(); ok {
		t.Error("an empty board should have no best score")
	}
	h.Scores.Scores[3] = 4200
	h.Scores.Names[3].SetText("Ada")
	h.Scores.Scores[0] = 900
	h.Scores.Names[0].SetText("Bob")
	who, score, ok := h.Best()
	if !ok || score != 4200 || who != "Ada" {
		t.Errorf("Best() = %q %d %v, want Ada 4200 true -- it should scan the whole board",
			who, score, ok)
	}

	h.Scores.Names[3] = house.PStr16{}
	if who, _, _ := h.Best(); who != "(nameless)" {
		t.Errorf("a scoring row with no name reads as %q, want (nameless)", who)
	}
}

// ---------------------------------------------------------------------------

// messageIsOnScreen reports whether anything was drawn in the status band. The band
// is filled black and the message is cream, so cream in those rows means text.
func messageIsOnScreen(s *render.Surface) bool {
	for v := splashTall; v < screenTall; v++ {
		for x := 0; x < screenWide; x++ {
			if s.Pix[v*s.W+x] == cream {
				return true
			}
		}
	}
	return false
}

func count(s *render.Surface, idx uint8) int {
	n := 0
	for _, p := range s.Pix {
		if p == idx {
			n++
		}
	}
	return n
}

// inkNear looks for a pixel of this colour within a glyph's reach of (h, v).
func inkNear(s *render.Surface, h, v int, idx uint8) bool {
	for y := v - render.FontTall; y <= v+2; y++ {
		for x := h - 2; x < h+int(render.FontWide)*8; x++ {
			if x < 0 || y < 0 || x >= s.W || y >= s.H {
				continue
			}
			if s.Pix[y*s.W+x] == idx {
				return true
			}
		}
	}
	return false
}

func utf8Valid(s string) bool {
	for _, r := range s {
		if r == 0xFFFD {
			return false
		}
	}
	return true
}
