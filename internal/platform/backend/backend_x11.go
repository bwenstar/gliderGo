//go:build linux && cgo && !nullbackend

package backend

import (
	"github.com/bwenstar/gliderGo/internal/platform"
	"github.com/bwenstar/gliderGo/internal/platform/x11"
)

// Name identifies the compiled-in backend.
const Name = "x11"

// Open creates the host window for this build.
func Open(cfg platform.Config) (platform.Window, error) { return x11.New(cfg) }

// Room is the space a window may have on this display (platform.Room).
func Room() (platform.Room, error) { return x11.Room() }
