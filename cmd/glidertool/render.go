package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwenstar/gliderGo/internal/assetfs"
	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/render"
)

// renderCmd composes rooms the way the game does and writes them out as PNGs.
//
// This is how the renderer is checked by eye. The game itself will draw the same
// composition to a window, but a PNG can be opened beside a screenshot of the
// original, diffed between two commits, and produced for every room of every
// shipped house in one command -- which is the only practical way to notice that,
// say, one house's floor supports have gone missing.
func renderCmd(args []string) error {
	fs := flag.NewFlagSet(prog+" render", flag.ContinueOnError)
	var (
		artDir    = fs.String("art", "", "extracted application art tree to use instead of the one built in")
		houseDir  = fs.String("houseart", "", "extracted per-house resource forks to use instead of the ones built in")
		roomNum   = fs.Int("room", -1, "room number to centre on (default: the house's first room)")
		at        = fs.String("at", "", "room to centre on as floor,suite (overrides -room)")
		neighbors = fs.Int("neighbors", 9, "how much of the surrounding house to compose: 1, 3 or 9")
		clockAt   = fs.String("clock", "1994-10-14T10:09:00Z", "the time the clock faces show, RFC3339")
		music     = fs.Bool("music", false, "isPlayMusicGame: whether a stereo draws as on")
		scale     = fs.Int("scale", 1, "integer nearest-neighbour upscale of the output")
		all       = fs.Bool("all", false, "compose every room; -o names a directory")
		quiet     = fs.Bool("quiet", false, "do not print the per-room summary")
		out       = fs.String("o", "room.png", "PNG to write, or the directory to fill with -all")
	)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: %s render [flags] <house>\n\n", prog)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("expected exactly one house file")
	}
	if *scale < 1 {
		return fmt.Errorf("-scale must be at least 1")
	}
	switch *neighbors {
	case 1, 3, 9:
	default:
		return fmt.Errorf("-neighbors must be 1, 3 or 9")
	}
	clock, err := time.Parse(time.RFC3339, *clockAt)
	if err != nil {
		return fmt.Errorf("-clock: %w", err)
	}

	path := fs.Arg(0)
	h, err := house.LoadFile(path)
	if err != nil {
		return err
	}

	artFS, _ := assetRoot(*artDir, "art")
	assets := render.NewAssets(artFS)
	// The house's own resource fork goes in front of the application's, for as
	// long as the house is open, exactly as HouseIO.c does it. Without this a
	// house's custom backgrounds all fall back to PICT 2000.
	//
	// The house is named by a path here rather than by a name in a library, so the fork is
	// looked up by that path's base name -- which is what the house is called, on disk and in
	// the built-in tree alike.
	houseArtFS, houseArtName := assetRoot(*houseDir, "houseart")
	name := strings.TrimSuffix(filepath.Base(path), ".house")
	fork := assetfs.Name(houseArtName, name)
	if assetfs.IsDir(houseArtFS, name) {
		assets.OpenHouseResFork(fork, assetfs.Sub(houseArtFS, name))
	} else if !*quiet {
		fmt.Fprintf(os.Stderr, "%s: no extracted resource fork at %s; custom art will fall back\n", prog, fork)
	}

	scene := render.NewScene(render.DefaultView(), assets, h)
	scene.NumNeighbors = *neighbors
	scene.PlayMusicGame = *music
	scene.Clock = clock

	// This tool composes a Scene with no game World behind it, so the four dinahs
	// hooks are nil and nothing registers. That is right for a static render -- the
	// dinahs table holds only per-frame animation state and contributes no pixel to
	// the background -- but the count is still worth reporting, because "this room
	// wants nineteen dynamic objects and the table holds eighteen" is exactly the kind
	// of thing a still image cannot show. So count the calls and answer -1 to all of
	// them, which is what a saturated table answers anyway.
	dynamics := 0
	scene.ZeroDinahs = func() { dynamics = 0 }
	scene.AddDynamicObject = func(int16, render.Rect, house.Object, int16, int16, bool) int16 {
		dynamics++
		return -1
	}

	rooms := []int{}
	switch {
	case *all:
		for i := range h.Rooms {
			rooms = append(rooms, i)
		}
	case *at != "":
		var floor, suite int
		if _, err := fmt.Sscanf(*at, "%d,%d", &floor, &suite); err != nil {
			return fmt.Errorf("-at: want floor,suite: %w", err)
		}
		n := scene.GetRoomNumber(int16(floor), int16(suite))
		if n < 0 {
			return fmt.Errorf("no room at floor %d suite %d", floor, suite)
		}
		rooms = append(rooms, int(n))
	case *roomNum >= 0:
		rooms = append(rooms, *roomNum)
	default:
		rooms = append(rooms, int(h.FirstRoom))
	}

	if *all {
		if err := os.MkdirAll(*out, 0o777); err != nil {
			return err
		}
	}

	for _, n := range rooms {
		if n < 0 || n >= len(h.Rooms) {
			return fmt.Errorf("room %d out of range (house has %d)", n, len(h.Rooms))
		}
		scene.RoomNumber = int16(n)
		scene.DrawLocale()

		dst := *out
		if *all {
			dst = filepath.Join(*out, fmt.Sprintf("%03d_%s.png", n, safeName(h.Rooms[n].Name.Text())))
		}
		if err := writePNG(dst, scene.Back, *scale); err != nil {
			return err
		}
		if !*quiet {
			printSceneSummary(os.Stdout, scene, n, dst, dynamics)
		}
	}

	// Errors are sticky rather than fatal, so that a partly extracted asset tree
	// still produces a picture; but they are reported, because a silently missing
	// PICT looks exactly like a room that is meant to be empty.
	if err := assets.Err(); err != nil {
		return err
	}
	if n := assets.Approximations(); n > 0 && !*quiet {
		fmt.Fprintf(os.Stderr, "%s: %d pixels colour-matched (house pictures with their own ColorTable)\n", prog, n)
	}
	return nil
}

// printSceneSummary reports what the composition decided, which is most of what
// goes wrong: the wrong neighbour, the wrong light count, a saved-map table that
// filled up and silently dropped a clock.
func printSceneSummary(w *os.File, s *render.Scene, n int, dst string, dynamics int) {
	rm := &s.H.Rooms[n]
	fmt.Fprintf(w, "%s\n", dst)
	fmt.Fprintf(w, "  room %d %q  floor %d suite %d  background %d  lights %d  objects %d\n",
		n, rm.Name.Text(), rm.Floor, rm.Suite, rm.Background, s.NumLights, rm.LiveObjects())
	fmt.Fprintf(w, "  neighbours %v\n", s.LocalNumbers)
	fmt.Fprintf(w, "  structures %v\n", s.IsStructure)
	fmt.Fprintf(w, "  savedMaps %d/%d  flames %d  tikis %d  coals %d  pendulums %d  stars %d  dynamics %d  manholes %d\n",
		len(s.SavedMaps), 24, len(s.Flames), len(s.TikiFlames), len(s.Coals),
		len(s.Pendulums), len(s.Stars), dynamics, len(s.TempManholes))
}

// writePNG saves a surface, optionally upscaled by an integer factor. The upscale
// is nearest-neighbour and exists only so that 640x460 of 1994 pixels can be
// looked at on a modern display without a viewer's own smoothing in the way.
func writePNG(path string, s *render.Surface, scale int) error {
	img := s.ToRGBA()
	if scale > 1 {
		b := img.Bounds()
		big := image.NewRGBA(image.Rect(0, 0, b.Dx()*scale, b.Dy()*scale))
		for y := 0; y < big.Rect.Dy(); y++ {
			for x := 0; x < big.Rect.Dx(); x++ {
				big.Set(x, y, img.At(x/scale, y/scale))
			}
		}
		img = big
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// safeName makes a room name usable as a file name. Room names are MacRoman and
// may hold anything, including slashes and colons.
func safeName(s string) string {
	if s == "" {
		return "unnamed"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
