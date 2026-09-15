package fidelity

// The other half of the corpus: the screens a player meets before the game starts.
//
// No replay script can reach these. The shell is a separate program from the simulation by
// design -- internal/shell does not import internal/game (docs/PLAN.md, 1.7a) -- so the
// splash screen, the house picker, the settings rows and the high-score board are pixels
// that the frame corpus can never cover, and they are also the pixels a player sees first
// and judges the port by. A single misplaced row on the settings screen is the kind of thing
// that survives a whole stage unnoticed, because everybody developing here knows what the
// screen is supposed to say and reads it as intended rather than as drawn.
//
// This is the same composition `glidergo -shot` performs, which is not duplication for its
// own sake: -shot writes a PNG for a human to look at, scaled by whatever the flag said,
// and this hashes the index plane at 1:1. A corpus of PNG *bytes* would be a corpus of Go's
// image/png encoder, and a corpus of scaled images would compare the upscaler.

import (
	"errors"
	"fmt"
	"os"

	"glidergo/internal/platform"
	"glidergo/internal/prefs"
	"glidergo/internal/render"
	"glidergo/internal/shell"
)

// DefaultScreens is every screen Shell.Show knows, in the order a player meets them.
var DefaultScreens = []string{"splash", "houses", "settings", "about", "credits", "scores"}

// refVersion is the version string the About box draws in a recording. See Screen.
const refVersion = "fidelity"

// ScreensOpts is the shell's world for a recording: where the art and the houses are, and
// which house is selected.
//
// The house matters to three of the six screens -- the splash names it, the picker
// highlights it, the board is its -- so it is pinned rather than left to whatever
// Library.Discover happened to sort first.
type ScreensOpts struct {
	ArtDir   string
	HouseDir string
	House    string

	// Screens to record, in order. Nil means DefaultScreens.
	Screens []string
}

// RecordScreens composes each screen and hashes it.
func RecordScreens(o ScreensOpts) (*Reference, error) {
	if len(o.Screens) == 0 {
		o.Screens = DefaultScreens
	}
	ref := &Reference{Kind: "screens", Columns: []string{"screen"}}

	view := ""
	for _, name := range o.Screens {
		surf, err := Screen(o, name)
		if err != nil {
			return nil, err
		}
		if view == "" {
			view = fmt.Sprintf("%dx%d", surf.W, surf.H)
		}
		ref.Rows = append(ref.Rows, Row{Name: name, Hashes: []string{Hash(surf)}})
	}

	// The house count is in the header because the picker draws a list of them: extracting a
	// seventh house is a legitimate reason for the `houses` row to change, and without this
	// line that change looks like a layout regression.
	lib, err := shell.Discover(o.HouseDir)
	if err != nil {
		return nil, err
	}
	ref.Head = []KV{
		{"house", o.House},
		{"houses", fmt.Sprint(len(lib.Houses))},
		{"skipped", fmt.Sprint(len(lib.Skipped))},
		{"prefs", "defaults"},
		{"version", refVersion},
		{"view", view},
	}
	return ref, nil
}

// Screen composes one screen of the shell and returns the surface it drew on.
//
// Everything optional about the host is left out, and each omission is what makes the image
// reproducible rather than what makes it convenient:
//
//   - No SavePrefs and no ApplyPrefs: the settings screen is being photographed, not used.
//   - No Scores hook, so the board is the *house file's* own table rather than this
//     machine's side-car. A reference that changed the first time somebody on the build
//     machine got onto the board would be a test that fails for the best possible reason
//     and still fails.
//   - Defaults for the preferences, never a file on disk, for the same reason.
//   - Present does nothing and Poll returns no events: one Draw is the whole recording, so
//     there is no frame after this one for an event to affect.
func Screen(o ScreensOpts, name string) (*render.Surface, error) {
	lib, err := shell.Discover(o.HouseDir)
	if err != nil {
		return nil, err
	}
	if len(lib.Houses) == 0 {
		return nil, fmt.Errorf("fidelity: no houses in %s", o.HouseDir)
	}
	if _, err := os.Stat(o.ArtDir); err != nil {
		return nil, fmt.Errorf("fidelity: no art in %s: %w", o.ArtDir, err)
	}

	view := render.DefaultView()
	scr := render.NewSurface(int(view.Screen.Wide()), int(view.Screen.Tall()))
	host := shell.Host{
		Screen:  scr,
		Assets:  render.NewAssets(o.ArtDir),
		Present: func() {},
		Poll:    func() []platform.Event { return nil },
		Play: func(shell.Choice) (shell.Outcome, error) {
			return shell.Outcome{}, errors.New("fidelity: a reference does not play")
		},
		Prefs: prefs.Default(),

		// A constant and not the build's version, which the About box draws: tagging a
		// release must not invalidate the corpus. It is in the header so that the string
		// the reference was recorded with is on the file rather than in this comment.
		Version: refVersion,
	}
	sh, err := shell.New(host, lib)
	if err != nil {
		return nil, err
	}
	if o.House != "" && !sh.Select(o.House) {
		return nil, fmt.Errorf("fidelity: %s is not in %s", o.House, o.HouseDir)
	}
	if err := sh.Show(name); err != nil {
		return nil, err
	}
	sh.Draw()
	return scr, nil
}
