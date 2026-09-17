//go:build nullbackend || !cgo || !linux

package backend

import (
	"os"

	"github.com/bwenstar/gliderGo/internal/platform"
	"github.com/bwenstar/gliderGo/internal/platform/null"
)

// Name identifies the compiled-in backend.
const Name = "null"

// Open creates a headless window. GLIDERGO_FRAMEDUMP, if set, names a directory
// that every presented frame is written to as a PNG.
func Open(cfg platform.Config) (platform.Window, error) {
	return null.New(cfg, os.Getenv("GLIDERGO_FRAMEDUMP"))
}
