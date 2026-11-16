// Package backend is the one place that decides which platform backend a build contains.
//
// It exists so that no other package has to care. Everything above it takes a platform.Window
// and never learns whether the pixels are going to an X server, to GDI, or to a PNG on disk:
// cmd/glidergo calls Room and Open once each (play.go's openWindow) and prints Name in its
// banner, and that is the whole of the port's contact with the choice.
//
// The choice is made at compile time by build tags and never by runtime probing. That is
// deliberate: a wrong tag is a build failure or a missing symbol, whereas a wrong probe is a
// game that comes up headless on a machine with a perfectly good display and says nothing about
// it. The four selectors are exact logical complements -- every build gets one Open, one Room
// and one Name, no build gets two, and none gets none:
//
//	backend_x11.go     linux && cgo && !nullbackend      x11    Name = "x11"
//	backend_win32.go   windows && !nullbackend           win32  Name = "win32"
//	backend_web.go     js && wasm && !nullbackend        web    Name = "web"
//	backend_null.go    nullbackend, and everything else  null   Name = "null"
//
// The last line's constraint is the long way of saying so:
//
//	nullbackend || (!linux && !windows && !(js && wasm)) || (!cgo && !windows && !(js && wasm))
//
// Read it as "asked for by name, or not one of the three platforms that has a backend, or Linux
// without the cgo that Xlib needs". Three details in it are load-bearing:
//
//   - `&& !windows` on the cgo clause. The win32 backend is pure syscall with no cgo at all, so a
//     CGO_ENABLED=0 Windows build -- which is what `make cross` produces and what every release
//     archive ships -- must still get a real window rather than falling through to null.
//   - `&& !(js && wasm)` on both clauses, for the same reason. A browser build never has cgo, so
//     without it the second clause would take every one of them for null, and the page would
//     load a game that drew nothing.
//   - `nullbackend` first. `make headless`, `go test` on a machine with no display, the fidelity
//     corpus and every frame-dumping tool ask for the null backend explicitly, on a host that
//     could perfectly well open a window. That has to win over every platform backend. A js/wasm
//     build asks for it too: tools/webcheck runs one in the browser, so that the frames it dumps
//     into the page's storage can be compared with the native build's byte for byte.
//
// So on Linux the x11 backend needs cgo and libX11, and on Windows nothing needs anything: which
// is the practical difference between the two, and the reason Windows has no equivalent of the
// `libx11-dev` line in the README's build instructions. A browser needs nothing either, but it
// needs a page: `make web` builds one around the binary.
//
// macOS resolves to null today. That is Stage 6 (docs/PLAN.md), and an SDL2 or Cocoa selector
// would slot in here beside these four with `darwin` cut out of the null constraint the same way
// `windows` and `js && wasm` now are.
package backend
