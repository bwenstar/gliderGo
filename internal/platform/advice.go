package platform

// What to tell somebody whose window did not open.
//
// This lives here, beside Expand, for the reason Expand's own comment gives: the x11 backend needs
// libX11 and a display and the machine this port is written on can be made to have neither, so a
// plain function over two strings is the only part of the diagnosis that can be tested anywhere.
// The advice is also the half most worth testing, because it is the half that is *guessed*.
//
// The old message was one sentence -- "cannot open display (is DISPLAY set?)" -- and it is right
// about one of the three ways to get here and actively misleading about the other two. Three
// different people read it:
//
//   - DISPLAY is unset because there is no graphical session at all: a container, a cron job, an
//     `ssh` without -X. "Is DISPLAY set?" is the right question and the answer is a headless run.
//   - DISPLAY is set and the server refused anyway: the value is wrong, or there is no xauth
//     cookie for it, which is what `ssh` without -X looks like once something has exported DISPLAY.
//     Being asked whether DISPLAY is set, when it visibly is, reads as the program not knowing.
//   - WAYLAND_DISPLAY is set and DISPLAY is not, which on a 2026 desktop is the likeliest of the
//     three and the one the old message sent furthest wrong. There is a session, it is graphical,
//     and it is running; what is missing is XWayland, and nothing about "is DISPLAY set?" says so.
//
// Guessing between them from two environment variables is not certainty and the wording does not
// claim any: each branch says what was observed before what to do about it, so a reader who is in a
// fourth situation can see which of the three they were mistaken for.

import "fmt"

// DisplayAdvice returns the sentence to print when an X11 connection fails, given the two
// environment variables that distinguish the reasons. Both are taken as parameters rather than read
// here, so that every branch is reachable from a test on a host with a working display.
func DisplayAdvice(display, wayland string) string {
	switch {
	case display == "" && wayland != "":
		return fmt.Sprintf("WAYLAND_DISPLAY is %q and DISPLAY is unset, so this looks like a "+
			"Wayland session with no XWayland: install it (xwayland, or your distribution's "+
			"name for it) and DISPLAY will be set for you", wayland)
	case display == "":
		return "DISPLAY is unset, so there is no X session to open a window in. " +
			"For a machine with no display, -shot FILE draws one screen to a PNG and " +
			"-frames N plays without one"
	default:
		return fmt.Sprintf("DISPLAY is %q and the X server refused the connection. Usually the "+
			"value is wrong, or there is no authorisation for it -- over ssh that means -X or -Y",
			display)
	}
}
