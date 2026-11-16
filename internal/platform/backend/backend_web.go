//go:build js && wasm && !nullbackend

package backend

import (
	"github.com/bwenstar/gliderGo/internal/platform"
	"github.com/bwenstar/gliderGo/internal/platform/web"
)

// Name identifies the compiled-in backend.
const Name = "web"

// Open binds to the canvas on the page the program was loaded into (web.CanvasID).
func Open(cfg platform.Config) (platform.Window, error) { return web.New(cfg) }

// Room is the screen the browser is on (platform.Room).
func Room() (platform.Room, error) { return web.Room() }
