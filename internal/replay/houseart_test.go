package replay_test

// The end-to-end test for a house carrying pictures of its own.
//
// docs/IMPROVEMENTS.md 4.15 is the note that a new house could not have art. -houseart was the
// only way to give one any, and it worked by *replacing* the root half the twenty-two shipped
// houses read from, so a house written for this port could have pictures or the originals could and
// never both. The fix is that the lookup searches a list instead of substituting one root
// (internal/assetfs.ArtRoots), and that the levels tree -- which ships inside every executable
// beside the houses themselves -- is one of the places searched.
//
// # What this asserts, and why it is a digest comparison
//
// Three runs of one house, whose room 0 has been given background PICT 3000: with the picture in
// the levels tree, with the picture nowhere, and with the picture named by HouseArtDir. The first
// and the third must draw the same pixels and the second must differ.
//
// That shape is chosen rather than "is this pixel red", because a house picture is colour-matched
// into the 1994 palette on the way in (render.Assets.Approximations) and an assertion on a literal
// colour would be an assertion about the matcher. What these two comparisons pin is the whole of
// the claim without touching the palette at all: the new root delivers art (run 1 differs from run
// 2), and it delivers *the same* art as the flag that already worked (run 1 equals run 3). A search
// order that quietly fell through to the built-in forks would fail the first; a levels root wired
// to the wrong subdirectory would fail both.
//
// The house is built here from levels/Open House.house.txt, for the reason openhouse_test.go gives
// at length: the text is what is under version control, and assets/levels is a build product a
// stale copy of which could pass.

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/replay"
)

// artHouseDir builds Open House with room 0's background moved into the user range and returns the
// directory holding it, plus the house's name.
//
// Room 0 and not a room on the route, because this test flies nothing: the script below starts in
// room 0 and the composition is of room 0 and its eight neighbours, so room 0's background is on
// screen in the first frame whatever the glider then does. FirstUserBackground is the id because it
// is the definition of "a picture only the house could supply" -- the same test
// house.WantsOwnArt makes.
func artHouseDir(t *testing.T) (dir, name string) {
	t.Helper()
	h, err := house.ParseTextFile(openHouseSource)
	if err != nil {
		t.Fatalf("parse %s: %v", openHouseSource, err)
	}
	h.Rooms[0].Background = house.FirstUserBackground
	if !h.WantsOwnArt() {
		t.Fatal("a room with a background in the user range wants the house's own art; " +
			"WantsOwnArt disagrees, so one of the two is wrong")
	}
	raw, err := h.Save()
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	dir, name = t.TempDir(), "Open House"
	if err := os.WriteFile(filepath.Join(dir, name+".house"), raw, 0o666); err != nil {
		t.Fatalf("write house: %v", err)
	}
	return dir, name
}

// artTree writes one house's fork into a new directory and returns the directory that *contains*
// the houseart/ level -- which is what a levels root is, and what HouseArtDir names one level
// further in. Both callers below need one of the two, so returning the outer one keeps the
// difference between them visible at the call site.
//
// The picture is 640x460, a room background's size, and it is a gradient rather than a flat colour
// so that a composition which drew it at the wrong offset would still change the digest. Written
// here rather than committed as a fixture because it has no content worth reading: what is under
// test is where the file was found, not what is in it.
func artTree(t *testing.T, house string) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 640, 460))
	for y := 0; y < 460; y++ {
		for x := 0; x < 640; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x / 3), G: uint8(y / 2), B: 0x40, A: 0xff})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	root := t.TempDir()
	pict := filepath.Join(root, "houseart", house, "pict")
	if err := os.MkdirAll(pict, 0o777); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pict, "3000.png"), buf.Bytes(), 0o666); err != nil {
		t.Fatalf("write picture: %v", err)
	}
	return root
}

func TestAHouseCanCarryArtOfItsOwnInTheLevelsTree(t *testing.T) {
	houses, name := artHouseDir(t)
	tree := artTree(t, name)

	// One script, run three times with one field moved. Ten frames because nothing here is
	// about the flight model: the planes are hashed as the last frame left them, and the first
	// frame already has room 0's background on it.
	run := func(what string, set func(*replay.Script)) replay.Planes {
		t.Helper()
		s := localAssets(t, replay.NewScript(name, 10))
		s.HouseDir = houses // after localAssets, which points it at the extracted tree
		s.HouseArtDir = ""  // ditto, and this is the field under test
		s.Room = 0
		s.Sound, s.Music = false, false
		set(s)
		res, err := replay.Run(s)
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		return res.Planes
	}

	levels := run("art in the levels tree", func(s *replay.Script) { s.LevelDir = tree })
	none := run("art nowhere", func(s *replay.Script) {})
	flag := run("art named by -houseart", func(s *replay.Script) {
		s.HouseArtDir = filepath.Join(tree, "houseart")
	})

	if levels.Back == none.Back {
		t.Errorf("the levels tree's houseart/%s/pict/3000.png changed nothing: back plane is %s "+
			"with it and without it. That is 4.15 unfixed -- the house is drawing PICT 2000 "+
			"instead of its own background.", name, levels.Back)
	}
	if levels.Back != flag.Back {
		t.Errorf("back plane is %s with the picture in the levels tree and %s with the same "+
			"picture named by -houseart; the two roots must deliver the same pixels",
			levels.Back, flag.Back)
	}
	// The other two planes are asserted together with Back rather than instead of it: Back is
	// the composed room, and Main is what reached the screen. A change that showed up in one and
	// not the other would mean the background was composed and then covered, which is a finding
	// and not a pass.
	if levels.Main == none.Main {
		t.Errorf("the main plane is %s either way, so whatever was composed never reached the "+
			"screen", levels.Main)
	}
}

// The named root wins a house it holds, which is the property -houseart was added for -- somebody
// testing an extraction wants the tree they typed and not the copy inside the binary -- and the
// property that made it a substitute in the first place. Searching a list must not have cost it.
func TestANamedArtRootStillWinsTheHouseItHolds(t *testing.T) {
	houses, name := artHouseDir(t)
	mine, theirs := artTree(t, name), artTree(t, name)
	// Two trees of the same shape and different pictures, so that "the named one won" is a
	// visible fact rather than a coincidence of both being identical.
	if err := os.WriteFile(filepath.Join(theirs, "houseart", name, "pict", "3000.png"),
		flatPNG(t, color.RGBA{R: 0x20, G: 0x90, B: 0x20, A: 0xff}), 0o666); err != nil {
		t.Fatalf("overwrite: %v", err)
	}

	plane := func(art, levels string) string {
		t.Helper()
		s := localAssets(t, replay.NewScript(name, 10))
		s.HouseDir, s.HouseArtDir, s.LevelDir = houses, art, levels
		s.Room = 0
		s.Sound, s.Music = false, false
		res, err := replay.Run(s)
		if err != nil {
			t.Fatalf("run: %v", err)
		}
		return res.Planes.Back
	}

	mineAlone := plane(filepath.Join(mine, "houseart"), "")
	mineOverTheirs := plane(filepath.Join(mine, "houseart"), theirs)
	theirsAlone := plane("", theirs)

	if mineAlone == theirsAlone {
		t.Fatal("the two art trees hold the same picture, so this test proves nothing")
	}
	if mineOverTheirs != mineAlone {
		t.Errorf("with -houseart naming a tree that has %s, the levels tree's copy won "+
			"(%s, want %s); a named root is first in the search order for a reason",
			name, mineOverTheirs, mineAlone)
	}
}

// flatPNG is a room-sized picture of one colour, for the second of the two trees above.
func flatPNG(t *testing.T, c color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 640, 460))
	for y := 0; y < 460; y++ {
		for x := 0; x < 640; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}
