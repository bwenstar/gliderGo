package replay_test

// Hostile bytes, on purpose (docs/IMPROVEMENTS.md 4.30).
//
// A replay script is the file a bug report carries, so it is read from somebody else as a matter
// of course. Plain `go test` runs FuzzParse over its seeds; `make fuzz` runs the engine, and an
// input that fails is written to testdata/fuzz/FuzzParse/ to be committed.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/game/player"
	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/replay"
)

// FuzzParse is a script from anywhere.
//
// What must hold for any script Parse accepts that names a house:
//   - Write takes it, and Parse reads what Write wrote;
//   - that gives back every field Run reads (Write's comment says which it leaves out, and why);
//   - written again, it is the same text, so a script passed through `glidertool replay -script`
//     twice does not drift.
//
// And nothing panics on a script it refuses. That is not a formality here: a bare `artdir` line
// used to crash the tool meant to read it (Parse's word).
func FuzzParse(f *testing.F) {
	names, _ := filepath.Glob(filepath.Join("testdata", "*.script"))
	for _, name := range names {
		b, err := os.ReadFile(name)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(string(b))
	}
	var tmpl strings.Builder
	if err := replay.NewScript("CD Demo House", 600).Write(&tmpl); err != nil {
		f.Fatal(err)
	}
	f.Add(tmpl.String())
	f.Add("house Demo House\nroom 4\nwhere 420 20\nfacing left\ngliders 3\nstars 5\n" +
		"players 2\nat 0 right\nat 45 left,batt band\nat 90 - none\n")
	f.Add("house a\ndemo res/demo/my recording.bin\nclock 1995-03-02T04:05:06.5+09:30\n")
	f.Add("house a\nartdir\n")
	f.Add("house a\nroom 65540\n")
	f.Add("house a\nframes 600 1200\n")
	f.Fuzz(func(t *testing.T, src string) {
		s, err := replay.Parse(strings.NewReader(src))
		if err != nil || s.House == "" {
			return
		}
		var first strings.Builder
		if err := s.Write(&first); err != nil {
			t.Fatalf("Write refuses a script Parse read: %v", err)
		}
		back, err := replay.Parse(strings.NewReader(first.String()))
		if err != nil {
			t.Fatalf("Parse refuses what Write wrote: %v\n%s", err, first.String())
		}
		if d := scriptDiff(s, back); d != "" {
			t.Fatalf("a script written and read again has %s\n%s", d, first.String())
		}
		var second strings.Builder
		if err := back.Write(&second); err != nil {
			t.Fatal(err)
		}
		if first.String() != second.String() {
			t.Fatalf("Write is not a fixed point:\n%s\n---\n%s", first.String(), second.String())
		}
	})
}

// scriptDiff is the first field Run reads that differs between a and b, or "".
func scriptDiff(a, b *replay.Script) string {
	resume := a.Room >= 0 && a.Where != (house.Point{})
	switch {
	case a.House != b.House:
		return fmt.Sprintf("house %q, was %q", b.House, a.House)
	case a.Demo != b.Demo:
		return fmt.Sprintf("demo %q, was %q", b.Demo, a.Demo)
	case a.Seed != b.Seed || a.Frames != b.Frames || a.Neighbors != b.Neighbors ||
		a.TwoPlayer != b.TwoPlayer || a.Sound != b.Sound || a.Music != b.Music:
		return fmt.Sprintf("settings %d/%d/%d/%v/%v/%v, were %d/%d/%d/%v/%v/%v",
			b.Seed, b.Frames, b.Neighbors, b.TwoPlayer, b.Sound, b.Music,
			a.Seed, a.Frames, a.Neighbors, a.TwoPlayer, a.Sound, a.Music)
	case !a.Clock.Equal(b.Clock):
		return fmt.Sprintf("clock %v, was %v", b.Clock, a.Clock)
	case a.Room != b.Room && (a.Room >= 0 || b.Room >= 0):
		// Every negative room is the house's own start, and Write writes none of them.
		return fmt.Sprintf("room %d, was %d", b.Room, a.Room)
	case a.Room >= 0 && a.Where != b.Where:
		return fmt.Sprintf("where %v, was %v", b.Where, a.Where)
	case resume && (a.Facing != b.Facing || a.Gliders != b.Gliders || a.Stars != b.Stars):
		return fmt.Sprintf("facing/gliders/stars %d/%d/%d, were %d/%d/%d",
			b.Facing, b.Gliders, b.Stars, a.Facing, a.Gliders, a.Stars)
	case len(a.Input) != len(b.Input):
		return fmt.Sprintf("%d holds, was %d", len(b.Input), len(a.Input))
	}
	for i := range a.Input {
		x, y := a.Input[i], b.Input[i]
		if !a.TwoPlayer {
			// A one-player run reads only player one's keys, and Write writes only those.
			x.P2, y.P2 = player.Keys{}, player.Keys{}
		}
		if x != y {
			return fmt.Sprintf("hold %d %+v, was %+v", i, y, x)
		}
	}
	return ""
}
