package main

// How big the window is (docs/IMPROVEMENTS.md 2.1).
//
// A 640x480 window is a postage stamp on a 1080p monitor and a quarter of one on a 4K one, and a
// stranger's first minute with the game should not start with finding the settings screen. So
// the default magnification is auto: the largest integer scale at which the whole window,
// title bar and all, fits the monitor it opens on. A number in the settings is kept as a number,
// but a window it would make too big for this monitor is opened at the largest that fits, for
// this launch only -- the setting is still there for the monitor it was chosen on.

import (
	"fmt"

	"github.com/bwenstar/gliderGo/internal/platform"
	"github.com/bwenstar/gliderGo/internal/prefs"
)

// autoMax is the most auto chooses. 4x fits a 4K monitor, but it is a 19.7 MB frame, and a
// window that size is only offered once 2.76's bench row has it inside its budget on both
// backends -- on X11 that is `make bench`, and on Windows it is unmeasured. Until then a 4K
// player gets 3x unasked and 4x by asking for it.
const autoMax = 3

// windowScale is the magnification the window opens at, and a line to print about it when it
// is not what was asked for.
//
// asked is the setting, prefs.ScaleAuto or a number; given says the number came from -scale on
// this command line rather than from the settings. room is the space the display has, or nil
// when it was not asked -- a hermetic run must not depend on the machine it runs on (2.53) --
// or could not say. w and h are the game's size.
//
// A -scale that does not fit is honoured anyway, with a warning, because it was typed for this
// run: `make bench` asks for 4x on whatever display it is given, and a bench row that quietly
// measured 2x would be worse than a window that runs off the screen.
func windowScale(asked int, given bool, room *platform.Room, w, h int) (scale int, note string) {
	if asked == prefs.ScaleAuto {
		if room == nil {
			return 1, ""
		}
		return room.Fit(w, h, autoMax), ""
	}
	if room == nil {
		return asked, ""
	}
	fit := room.Fit(w, h, prefs.MaxScale)
	if asked <= fit {
		return asked, ""
	}
	where := fmt.Sprintf("%s (%dx%d)", room.From, room.W, room.H)
	if given {
		return asked, fmt.Sprintf("-scale %d is a %dx%d window, which is larger than %s; "+
			"the largest that fits is %d", asked, w*asked, h*asked, where, fit)
	}
	return fit, fmt.Sprintf("magnification %dx does not fit %s, so this window is %dx; "+
		"the setting is unchanged", asked, where, fit)
}

// scaleWord is the banner's account of the magnification: the number, and where it came from
// when that was not the number itself.
func scaleWord(scale, asked int) string {
	switch {
	case asked == prefs.ScaleAuto:
		return fmt.Sprintf("%d (auto)", scale)
	case scale != asked:
		return fmt.Sprintf("%d (fitted from %d)", scale, asked)
	}
	return fmt.Sprint(scale)
}
