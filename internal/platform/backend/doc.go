// Package backend is the one place that decides which platform backend a build contains.
//
// It exists so that no other package has to care. Everything above it takes a platform.Window
// and never learns whether the pixels are going to an X server, to GDI, or to a PNG on disk:
// cmd/glidergo calls Open once (play.go's openWindow) and prints Name in its banner, and that is
// the whole of the port's contact with the choice.
//
// The choice is made at compile time by build tags and never by runtime probing. That is
// deliberate: a wrong tag is a build failure or a missing symbol, whereas a wrong probe is a
// game that comes up headless on a machine with a perfectly good display and says nothing about
// it. The three selectors are exact logical complements -- every build gets one Open and one
// Name, no build gets two, and none gets none:
//
//	backend_x11.go     linux && cgo && !nullbackend                              x11    Name = "x11"
//	backend_win32.go   windows && !nullbackend                                   win32  Name = "win32"
//	backend_null.go    nullbackend || (!linux && !windows) || (!cgo && !windows)  null   Name = "null"
//
// Read the last line as "asked for by name, or not one of the two platforms that has a backend,
// or Linux without the cgo that Xlib needs". Two details in it are load-bearing:
//
//   - `&& !windows` on the cgo clause. The win32 backend is pure syscall with no cgo at all, so a
//     CGO_ENABLED=0 Windows build -- which is what `make cross` produces and what every release
//     archive ships -- must still get a real window rather than falling through to null.
//   - `nullbackend` first. `make headless`, `go test` on a machine with no display, the fidelity
//     corpus and every frame-dumping tool ask for the null backend explicitly, on a host that
//     could perfectly well open a window. That has to win over both platform backends.
//
// So on Linux the x11 backend needs cgo and libX11, and on Windows nothing needs anything: which
// is the practical difference between the two, and the reason Windows has no equivalent of the
// `libx11-dev` line in the README's build instructions.
//
// macOS resolves to null today. That is Stage 6 (docs/PLAN.md), and an SDL2 or Cocoa selector
// would slot in here beside these three with `darwin` cut out of the null constraint the same way
// `windows` now is.
package backend
