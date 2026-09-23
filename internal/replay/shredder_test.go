package replay_test

// The paper shredder, end to end and with the art loaded.
//
// RenderShreds used to fetch its picture with Sheet("shred"), and `shred` is not a sheet. It
// is shredSrcMap, a 40x35 GWorld of its own loaded from PICT 4010 with its mask from 5010
// (StructuresInit.c:556-563), and the port files it with the strips -- render.stripBounds,
// beside the toast and the fish. Sheet does not panic on a name it has no row for: it records
// the sticky asset error and returns nil, and every blit in RenderShreds is guarded by `art !=
// nil`. So the cloud kept its dirty rects, its 35 shred sounds and its sparkle, which is
// everything the trace can see, and drew no confetti at all. The only symptom was
// `render: no such sheet "shred"` at the end of the game.
//
// Nothing that loads art had ever shredded a glider. The golden traces stay clear of
// shredders (TestSixHundredFramesMatchTheGoldenTrace says so), game's shreds_test.go runs
// with no art on purpose, and TestEveryHouseStartsAndRuns gives each house a hundred frames
// from its own start, none of which is spent in a shredder. This file is that missing case.

import (
	"os"
	"testing"

	"github.com/bwenstar/gliderGo/internal/game/player"
	"github.com/bwenstar/gliderGo/internal/render"
	"github.com/bwenstar/gliderGo/internal/replay"
)

// TestAShreddedGliderFallsAsConfetti drops a glider into Slumberland's room 43, Northern
// Exposure, whose kShredder sits at h 364, v 167 with clear air above it, and watches the
// picture until the sparkle.
//
// The run takes 110 frames and needs no keys. The glider is caught on frame 29, fed through
// the slot until frame 48, and the cloud grows on 49..83, falls on 84..101 and sparkles on 102.
// The replacement glider arrives on 117, so the budget ends before it and holds one cloud.
//
// # What is asserted
//
// That Run returns nil, which is the asset check the harness makes at the end of every run
// and the one that used to fail. And that the confetti is on the screen: every opaque pixel of
// the 40x35 picture is found in Work, exactly, on the last two frames of the growth and on each
// of the eighteen frames the fall draws, in one column and four pixels lower each frame of the
// fall -- and is never found in Back. The search is over the whole plane rather than at a
// computed rect, so a cloud drawn in the wrong place fails as a cloud that moved rather than
// passing by coincidence.
//
// The growth arm is searched for the whole picture only, which it shows on its last frame and
// in effect on the one before (see the comment at the check). The slices before that are
// TestTheGrowthArmEmergesBottomFirst's question; this one is whether there is a picture at all.
func TestAShreddedGliderFallsAsConfetti(t *testing.T) {
	s := replay.NewScript("Slumberland", 110)
	s.Seed = 1
	s.Room = 43
	// Straight above the shredder's slot and clear of the ceiling. With no keys the glider
	// sinks at the unpowered rate and the shredder's hot rect catches it on the way down.
	s.Where.H, s.Where.V = 370, 100
	s.Sound, s.Music = false, false
	s = localAssets(t, s)

	// The reference picture, from the same tree and through the same accessor the fix uses.
	// A failure here is the tree, not the game.
	ref := render.NewAssets(os.DirFS(s.ArtDir))
	art := ref.Strip("shred")
	if err := ref.Err(); err != nil {
		t.Fatalf("load the reference strip: %v", err)
	}
	if art.W != 40 || art.H != 35 {
		t.Fatalf("strip/shred.png is %dx%d, want shredSrcRect's 40x35", art.W, art.H)
	}
	sprite := opaquePixels(art)
	// A mostly empty picture would match almost anywhere, which would make every assertion
	// below say nothing. The extracted cloud is 850 opaque pixels of 1,400.
	if len(sprite) < 40*35/4 {
		t.Fatalf("the shred strip has %d opaque pixels of %d; too few to search for",
			len(sprite), 40*35)
	}

	type hit struct {
		frame int64
		h, v  int
	}
	var hits []hit
	var inBack []int64
	watch := func(sm replay.Sample, _, work, back *render.Surface) {
		if sm.Mode != int16(player.GliderShredding) {
			return
		}
		if h, v, ok := findSprite(work, sprite); ok {
			hits = append(hits, hit{sm.Frame, h, v})
		}
		if _, _, ok := findSprite(back, sprite); ok {
			inBack = append(inBack, sm.Frame)
		}
	}

	res, err := replay.RunWatching(s, nil, watch)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Frames != int64(s.Frames) {
		t.Fatalf("the run stopped after %d of %d frames", res.Frames, s.Frames)
	}

	if len(inBack) != 0 {
		t.Errorf("the confetti is in Back on frames %v: it is an animation and belongs in "+
			"Work only, or the next frame's restore cannot take it off", inBack)
	}
	// Twenty frames, and the first two are the growth arm's last. The cloud's top edge is pinned
	// while it grows and its rows come out bottom-first, so at 34 rows tall it shows rows 1..34
	// of the picture from the pinned top down. Row 0 of the extracted picture is empty, so that
	// is already every opaque pixel, one row above where the whole picture sits a frame later
	// at 35 rows tall. Then the fall, whose nineteenth step is the sparkle frame: it moves the
	// cloud and draws nothing (Render.c:593-605), so the fall is eighteen of the twenty.
	if len(hits) != 2+shredFallDraws {
		t.Fatalf("the whole sprite is in Work on %d frame(s), want %d (two of the growth "+
			"and the fall's %d); found at %v", len(hits), 2+shredFallDraws, shredFallDraws, hits)
	}
	top := hits[1].v
	for i, got := range hits {
		want := hit{hits[0].frame + int64(i), hits[0].h, top + 4*(i-1)}
		switch i {
		case 0:
			want.v = top - 1
		case 1:
			want.v = top
		}
		if got != want {
			t.Errorf("hit %d is frame %d at h %d v %d, want frame %d at h %d v %d: the "+
				"cloud grows from a pinned top and then falls four pixels a frame, straight "+
				"down (Render.c:570-575 and 589-590)",
				i, got.frame, got.h, got.v, want.frame, want.h, want.v)
		}
	}
	t.Logf("confetti on frames %d..%d at h %d, grown at v %d and falling to v %d",
		hits[0].frame, hits[len(hits)-1].frame, hits[0].h, top, hits[len(hits)-1].v)
}

// shredFallDraws is how many frames of the fall arm draw the whole sprite: frames 1..19 of the
// counter, less the nineteenth, which sparkles instead. See game/shreds.go.
const shredFallDraws = 18

// spritePixel is one opaque pixel of a picture, at its offset from the picture's top left.
type spritePixel struct {
	x, y int
	v    uint8
}

// opaquePixels lists the pixels a Masked copy of s would write -- the same test Surface.opaque
// makes, a nil mask being opaque everywhere.
func opaquePixels(s *render.Surface) []spritePixel {
	var out []spritePixel
	for y := 0; y < s.H; y++ {
		for x := 0; x < s.W; x++ {
			i := y*s.W + x
			if s.Mask == nil || s.Mask[i] != 0 {
				out = append(out, spritePixel{x, y, s.Pix[i]})
			}
		}
	}
	return out
}

// findSprite returns the top left of the first place in s where every opaque pixel of the
// sprite is present exactly. The bounding box is taken from the pixels themselves, so a
// sprite whose outer rows are transparent can still be found against an edge.
func findSprite(s *render.Surface, sprite []spritePixel) (h, v int, ok bool) {
	w, ht := 0, 0
	for _, p := range sprite {
		w, ht = max(w, p.x+1), max(ht, p.y+1)
	}
	for y := 0; y+ht <= s.H; y++ {
		for x := 0; x+w <= s.W; x++ {
			match := true
			for _, p := range sprite {
				if s.Pix[(y+p.y)*s.W+x+p.x] != p.v {
					match = false
					break
				}
			}
			if match {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}
