//go:build linux && cgo && !nullbackend

package backend

import (
	"glidergo/internal/platform"
	"glidergo/internal/platform/x11"
)

// Name identifies the compiled-in backend.
const Name = "x11"

// Open creates the host window for this build.
func Open(cfg platform.Config) (platform.Window, error) { return x11.New(cfg) }
