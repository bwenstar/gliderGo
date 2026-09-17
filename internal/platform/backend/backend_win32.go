//go:build windows && !nullbackend

package backend

import (
	"github.com/bwenstar/gliderGo/internal/platform"
	"github.com/bwenstar/gliderGo/internal/platform/win32"
)

// Name identifies the compiled-in backend.
const Name = "win32"

// Open creates the host window for this build. No cgo: win32 is pure syscall, so this is what a
// CGO_ENABLED=0 cross-build from Linux produces, and `-tags nullbackend` is still the way to ask
// for a headless Windows build.
func Open(cfg platform.Config) (platform.Window, error) { return win32.New(cfg) }
