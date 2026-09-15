package render

// DynamicMaps.c's five animated families, registration half: the table entry, the saved-map
// slot it claims, the filmstrip baked into that slot, and the RandomInt draw that seeds the
// starting cel.
//
// The simulation half is internal/game/anim_test.go. The split is grease's: this file pins
// what a registration leaves behind and that one pins what the animator does with it, so
// neither has to reproduce the other's arithmetic.
//
// Most of it needs no art. bakeStrip composites the background and then masks a cel over it,
// and with no sheets loaded the mask step is skipped while the *claim* -- the slot's size, its
// (where, who) tag, and the seeded Src rect -- still happens. That is the half worth pinning
// here; where the pixels themselves matter it is TestComposeEveryRoom's golden hashes that
// say so.

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"glidergo/internal/house"
)

// animScene is a composed-enough Scene: a real view so the origin is non-zero, no art, and a
// back map big enough for bakeStrip to copy out of. It does not call DrawLocale, because the
// five add* functions are pure functions of their arguments plus the tables they append to,
// and driving them directly is what lets a test saturate 24 saved-map slots without building
// a house that has 24 objects in it.
func animScene(t *testing.T) *Scene {
	t.Helper()
	s := NewScene(DefaultView(), NewAssets(""), testHouse())
	if s.Back == nil || s.Back.Bounds().Wide() < 64 {
		t.Fatalf("fixture back map is %v; bakeStrip needs somewhere to copy from",
			s.Back.Bounds())
	}
	return s
}

// countingRandom installs a RandomInt hook that returns a fixed value and counts its calls,
// which is the only way to observe the fifth thing on this page: **whether the draw happened
// at all.** See TestASaturatedTableDoesNotConsumeARandomDraw.
func countingRandom(s *Scene, give int16) *int {
	n := 0
	s.RandomInt = func(int16) int16 {
		n++
		return give
	}
	return &n
}

// ---------------------------------------------------------------------------
// The five registrations
// ---------------------------------------------------------------------------

// TestTheFiveFamiliesClaimAFilmstripNotABackground is the single most misreadable thing about
// this subsystem, asserted five times.
//
// Every other saved map in the game is a swatch of wall the size of the thing in front of it.
// These five are films: one cel wide and *every cel* tall, holding N separately-composited
// pictures of the same patch of wall with a different animation frame on each. A candle's
// slot is 16x75, not 16x15.
//
// Getting it wrong is not a crash. Claim one cel instead of N and the strip holds only the
// first frame, so the flame animates by redrawing the same picture -- a candle that is lit and
// perfectly still, which is exactly what the original looks like on a machine too slow to
// keep up and therefore the last thing anyone would suspect.
func TestTheFiveFamiliesClaimAFilmstripNotABackground(t *testing.T) {
	cases := []struct {
		name       string
		add        func(s *Scene)
		table      func(s *Scene) []Anim
		celW, celH int16
		frames     int16
	}{
		{
			name:   "candle flame",
			add:    func(s *Scene) { s.addCandleFlame(0, 3, 96, 120) },
			table:  func(s *Scene) []Anim { return s.Flames },
			celW:   16,
			celH:   15,
			frames: NumCandleFrames,
		},
		{
			name:   "tiki flame",
			add:    func(s *Scene) { s.addTikiFlame(0, 3, 96, 120) },
			table:  func(s *Scene) []Anim { return s.TikiFlames },
			celW:   8,
			celH:   10,
			frames: NumTikiFrames,
		},
		{
			name:   "bbq coals",
			add:    func(s *Scene) { s.addBBQCoals(0, 3, 96, 120) },
			table:  func(s *Scene) []Anim { return s.Coals },
			celW:   32,
			celH:   9,
			frames: NumCoalFrames,
		},
		{
			name:   "pendulum",
			add:    func(s *Scene) { s.addPendulum(0, 3, 96, 120) },
			table:  func(s *Scene) []Anim { return s.Pendulums },
			celW:   32,
			celH:   28,
			frames: NumPendulumFrames,
		},
		{
			name:   "star",
			add:    func(s *Scene) { s.addStar(0, 3, 96, 120) },
			table:  func(s *Scene) []Anim { return s.Stars },
			celW:   32,
			celH:   31,
			frames: NumStarFrames,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := animScene(t)
			c.add(s)

			table := c.table(s)
			if len(table) != 1 {
				t.Fatalf("%d entries registered, want 1", len(table))
			}
			a := table[0]

			if len(s.SavedMaps) != 1 {
				t.Fatalf("%d saved-map slots claimed, want 1", len(s.SavedMaps))
			}
			slot := s.SavedMaps[a.SavedMap]
			if slot.Map == nil {
				t.Fatal("slot has no surface")
			}
			w, h := slot.Map.Bounds().Wide(), slot.Map.Bounds().Tall()
			if w != c.celW || h != c.celH*c.frames {
				t.Errorf("strip is %dx%d, want %dx%d: one cel wide and %d cels tall",
					w, h, c.celW, c.celH*c.frames, c.frames)
			}
			if slot.Where != 0 || slot.Who != 3 {
				t.Errorf("slot identifies (%d, %d), want (0, 3)", slot.Where, slot.Who)
			}

			// Dest is one cel, on screen. Src is one cel, inside the strip. The two
			// being the same size is what makes the animator's blit a straight copy.
			if got := a.Dest.Wide(); got != c.celW {
				t.Errorf("Dest is %d wide, want one cel (%d)", got, c.celW)
			}
			if got := a.Dest.Tall(); got != c.celH {
				t.Errorf("Dest is %d tall, want one cel (%d)", got, c.celH)
			}
			if got := a.Src.Wide(); got != c.celW {
				t.Errorf("Src is %d wide, want one cel (%d)", got, c.celW)
			}
			if got := a.Src.Tall(); got != c.celH {
				t.Errorf("Src is %d tall, want one cel (%d)", got, c.celH)
			}
			if a.Where != 0 || a.Who != 3 {
				t.Errorf("entry identifies (%d, %d), want (0, 3) -- the pair reBackUp "+
					"matches on", a.Where, a.Who)
			}
			if a.Stopped {
				t.Error("a fresh entry is Stopped; nothing would ever animate")
			}
		})
	}
}

// TestFrameCountsMatchTheSrcTables ties the five exported cel counts to the five src-rect
// tables they index, which is the only thing keeping the two from drifting apart.
//
// They are separate declarations because the counts cross the package boundary (game/anim.go
// counts to them) and the tables do not. A count one too small animates all but the last cel
// and a count one too large blits from below the bottom of the strip.
func TestFrameCountsMatchTheSrcTables(t *testing.T) {
	cases := []struct {
		name  string
		count int16
		cels  int
	}{
		{"candle", NumCandleFrames, len(flameSrc)},
		{"tiki", NumTikiFrames, len(tikiFlameSrc)},
		{"coals", NumCoalFrames, len(coalsSrc)},
		{"pendulum", NumPendulumFrames, len(pendulumSrc)},
		{"star", NumStarFrames, len(StarSrc)},
	}
	for _, c := range cases {
		if int(c.count) != c.cels {
			t.Errorf("%s: constant says %d frames, src table holds %d",
				c.name, c.count, c.cels)
		}
	}
}

// TestTheCandleIsAnchoredBottomCentre is the one of the five whose Dest is not simply (h, v).
//
// A candle's flame sits on the wick, so the rect is placed h-8 (half its width to the left)
// and v-15 (its whole height up). The other four are top-left anchored and their callers have
// already done any offsetting -- the kTiki case passes itsRect.Top-9, for instance -- which is
// why only this one needs pinning.
func TestTheCandleIsAnchoredBottomCentre(t *testing.T) {
	s := animScene(t)
	const h, v int16 = 96, 120
	s.addCandleFlame(0, 3, h, v)

	want := Rect{Top: v - 15, Left: h - 8, Bottom: v, Right: h - 8 + 16}
	if got := s.Flames[0].Dest; got != want {
		t.Errorf("Dest = %+v, want %+v: h-8 and v-15, so the flame sits on the wick",
			got, want)
	}
}

// TestTheFourRandomFamiliesSeedTheirStartingCel is why a room of candles does not flicker in
// lockstep.
//
// Four of the five draw their opening cel from RandomInt and the Src rect follows it -- cel k
// is at k*celH down the strip. The pendulum is the exception and is covered by
// TestThePendulumSeedsCelOneAndDrawsForDirection.
func TestTheFourRandomFamiliesSeedTheirStartingCel(t *testing.T) {
	cases := []struct {
		name  string
		add   func(s *Scene)
		table func(s *Scene) []Anim
		celH  int16
		seed  int16
	}{
		{"candle", func(s *Scene) { s.addCandleFlame(0, 3, 96, 120) },
			func(s *Scene) []Anim { return s.Flames }, 15, 3},
		{"tiki", func(s *Scene) { s.addTikiFlame(0, 3, 96, 120) },
			func(s *Scene) []Anim { return s.TikiFlames }, 10, 4},
		{"coals", func(s *Scene) { s.addBBQCoals(0, 3, 96, 120) },
			func(s *Scene) []Anim { return s.Coals }, 9, 2},
		{"star", func(s *Scene) { s.addStar(0, 3, 96, 120) },
			func(s *Scene) []Anim { return s.Stars }, 31, 5},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := animScene(t)
			draws := countingRandom(s, c.seed)
			c.add(s)

			if *draws != 1 {
				t.Fatalf("RandomInt called %d times, want exactly 1", *draws)
			}
			a := c.table(s)[0]
			if a.Mode != c.seed {
				t.Errorf("Mode = %d, want the drawn %d", a.Mode, c.seed)
			}
			if a.Src.Top != c.seed*c.celH {
				t.Errorf("Src.Top = %d, want %d (cel %d of a %d-pixel stride)",
					a.Src.Top, c.seed*c.celH, c.seed, c.celH)
			}
		})
	}
}

// TestASeedOnePastTheEndIsLeftUnclamped documents a deliberate out-of-range value.
//
// RandomInt(n) can return n -- see World.RandomInt, which is `(|rand| * n) / 32768` and
// therefore hits n when rand is 32767. So a five-cel flame can be seeded at Mode 5 with a Src
// rect entirely below the bottom of its own strip.
//
// It is not clamped, and clamping it would be a divergence rather than a fix. The animator
// pre-increments and wraps before its first blit, so Mode 5 becomes 6, fails the same
// `>= frames` test that 4 would have failed, and the wrap resets Src absolutely. Cel 0 is
// drawn either way. Clamping here would consume the same draw and leave Mode at 4, which
// draws cel 0 as well on the first frame -- and cel 1 on the second where the original draws
// cel 1 too. So the *visible* difference is nil and the divergence would be silent, which is
// exactly the kind of change that makes a replay stop matching for reasons nobody can find.
// Pinned as-is so that clamping has to be a decision.
func TestASeedOnePastTheEndIsLeftUnclamped(t *testing.T) {
	s := animScene(t)
	countingRandom(s, NumCandleFrames) // one past the last valid cel index
	s.addCandleFlame(0, 3, 96, 120)

	a := s.Flames[0]
	if a.Mode != NumCandleFrames {
		t.Errorf("Mode = %d, want the unclamped %d", a.Mode, NumCandleFrames)
	}
	strip := s.SavedMaps[a.SavedMap].Map
	if a.Src.Top < strip.Bounds().Tall() {
		t.Errorf("Src.Top = %d is still inside a %d-tall strip; the point of this test is "+
			"that it is not", a.Src.Top, strip.Bounds().Tall())
	}
}

// TestThePendulumSeedsCelOneAndDrawsForDirection is the odd one of the five, and it is odd
// four ways.
//
// It claims its slot *before* computing Dest, the opposite order from the flames. It does not
// draw for its cel: the swing always starts at cel 1, the middle of three. It draws for the
// *direction* instead, which is the only use of ToOrFro in the game. And it writes
// ClockFrame, a phase counter shared by every pendulum in the locale.
//
// ClockFrame = 10 is the one to check against the C rather than to reason about: it is not
// "start at frame 10", it is "make the first swing happen in five frames rather than ten",
// because RenderPendulums fires at 10 and at 15. See game/anim.go.
func TestThePendulumSeedsCelOneAndDrawsForDirection(t *testing.T) {
	for _, c := range []struct {
		name    string
		draw    int16
		toOrFro bool
	}{
		{"heads swings one way", 0, true},
		{"tails the other", 1, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := animScene(t)
			s.ClockFrame = -1 // so that 10 is observably written and not merely absent
			draws := countingRandom(s, c.draw)
			s.addPendulum(0, 3, 96, 120)

			if *draws != 1 {
				t.Fatalf("RandomInt called %d times, want exactly 1 (the direction, "+
					"not the cel)", *draws)
			}
			p := s.Pendulums[0]
			if p.Mode != 1 {
				t.Errorf("Mode = %d, want 1: the swing starts at the centre cel and is "+
					"never seeded randomly", p.Mode)
			}
			if p.Src.Top != 28 {
				t.Errorf("Src.Top = %d, want 28 -- one 28-pixel stride, matching Mode 1",
					p.Src.Top)
			}
			if p.ToOrFro != c.toOrFro {
				t.Errorf("ToOrFro = %v with a draw of %d, want %v",
					p.ToOrFro, c.draw, c.toOrFro)
			}
			if s.ClockFrame != 10 {
				t.Errorf("ClockFrame = %d, want 10 (DynamicMaps.c:574)", s.ClockFrame)
			}
		})
	}
}

// TestClockFrameSurvivesADrawLocaleReset is the one line missing from
// ZeroFlamesAndTheLike, and it is missing in the original too.
//
// The C's reset clears eight counters and clockFrame is not one of them, so it carries across
// a room change. That is unobservable in the shipped game for two reasons that both have to
// hold: a locale with no pendulum never reads it (RenderPendulums returns before the
// increment), and a locale with one reseeds it to 10 during the compose. The test states both
// halves so that adding the "missing" line is a decision rather than a tidy-up.
func TestClockFrameSurvivesADrawLocaleReset(t *testing.T) {
	s := animScene(t)
	s.ClockFrame = 7
	s.DrawLocale()

	if s.ClockFrame != 7 {
		t.Errorf("ClockFrame = %d after DrawLocale, want the carried-over 7: it is not one "+
			"of ZeroFlamesAndTheLike's eight assignments", s.ClockFrame)
	}
	// And the reseed, which is the other half of why it does not matter.
	s.addPendulum(0, 3, 96, 120)
	if s.ClockFrame != 10 {
		t.Errorf("ClockFrame = %d after a registration, want 10", s.ClockFrame)
	}
}

// ---------------------------------------------------------------------------
// The caps
// ---------------------------------------------------------------------------

// TestEachFamilyIsRefusedAtItsOwnCap covers the first `return` of all five, which are five
// different numbers over one shared 24-slot budget.
//
//	candles  20 -- but 20 candles want 20 of the 24 saved-map slots, so the table cap is
//	           unreachable in practice and the saved-map cap is what bites
//	tikis     8
//	coals     8
//	pendulums 8
//	stars     4
//
// A refused registration must not be half-done: no entry appended and no slot claimed.
func TestEachFamilyIsRefusedAtItsOwnCap(t *testing.T) {
	cases := []struct {
		name  string
		cap   int
		fill  func(s *Scene)
		add   func(s *Scene)
		table func(s *Scene) []Anim
	}{
		{"candles", kMaxCandles,
			func(s *Scene) { s.Flames = append(s.Flames, Anim{SavedMap: -1}) },
			func(s *Scene) { s.addCandleFlame(0, 3, 96, 120) },
			func(s *Scene) []Anim { return s.Flames }},
		{"tikis", kMaxTikis,
			func(s *Scene) { s.TikiFlames = append(s.TikiFlames, Anim{SavedMap: -1}) },
			func(s *Scene) { s.addTikiFlame(0, 3, 96, 120) },
			func(s *Scene) []Anim { return s.TikiFlames }},
		{"coals", kMaxCoals,
			func(s *Scene) { s.Coals = append(s.Coals, Anim{SavedMap: -1}) },
			func(s *Scene) { s.addBBQCoals(0, 3, 96, 120) },
			func(s *Scene) []Anim { return s.Coals }},
		{"pendulums", kMaxPendulums,
			func(s *Scene) { s.Pendulums = append(s.Pendulums, Anim{SavedMap: -1}) },
			func(s *Scene) { s.addPendulum(0, 3, 96, 120) },
			func(s *Scene) []Anim { return s.Pendulums }},
		{"stars", kMaxStars,
			func(s *Scene) { s.Stars = append(s.Stars, Anim{SavedMap: -1}) },
			func(s *Scene) { s.addStar(0, 3, 96, 120) },
			func(s *Scene) []Anim { return s.Stars }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := animScene(t)
			draws := countingRandom(s, 0)
			for len(c.table(s)) < c.cap {
				c.fill(s)
			}
			c.add(s)

			if got := len(c.table(s)); got != c.cap {
				t.Errorf("the refused entry was appended anyway: %d entries, cap %d",
					got, c.cap)
			}
			if len(s.SavedMaps) != 0 {
				t.Errorf("the refused entry claimed %d slots; the cap test is above the "+
					"claim", len(s.SavedMaps))
			}
			if *draws != 0 {
				t.Errorf("RandomInt called %d times on a refusal, want 0", *draws)
			}
		})
	}
}

// TestASaturatedTableDoesNotConsumeARandomDraw is the sharpest thing on this page, because it
// is a statement about *every subsequent random number in the game*.
//
// The C puts each family's RandomInt call inside `if (savedNum != -1)` -- inside the
// saturation guard. So when the 24-slot saved-map table fills, the draw does not happen, and
// everything downstream in the stream shifts one place earlier.
//
// Whether a registration succeeds depends on how many objects were visible, which depends on
// SectRect against the screen, which depends on the window's dimensions. **The random stream
// is therefore a function of the resolution.** That is transcribed rather than repaired, and
// it is why a recorded replay has to state its window size and why Stage 3's networked race
// has to agree one. docs/IMPROVEMENTS.md 2.42.
//
// Hoisting the four draws above their guards would tidy four functions and change every
// random number in the game from the first saturated room onward. This test is what makes
// that fail loudly.
func TestASaturatedTableDoesNotConsumeARandomDraw(t *testing.T) {
	for _, c := range []struct {
		name string
		add  func(s *Scene)
	}{
		{"candle", func(s *Scene) { s.addCandleFlame(0, 3, 96, 120) }},
		{"tiki", func(s *Scene) { s.addTikiFlame(0, 3, 96, 120) }},
		{"coals", func(s *Scene) { s.addBBQCoals(0, 3, 96, 120) }},
		{"pendulum", func(s *Scene) { s.addPendulum(0, 3, 96, 120) }},
		{"star", func(s *Scene) { s.addStar(0, 3, 96, 120) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := animScene(t)
			draws := countingRandom(s, 0)
			for len(s.SavedMaps) < kMaxSavedMaps {
				s.SavedMaps = append(s.SavedMaps, SavedMap{Where: -1, Who: -1})
			}

			c.add(s)

			if *draws != 0 {
				t.Errorf("RandomInt called %d times with the saved-map table full, want 0 "+
					"-- the draw is inside the guard, and hoisting it would shift the "+
					"whole stream", *draws)
			}
		})
	}
}

// TestTheGuardedFamiliesRefuseANegativeOrigin covers the second half of four of the five
// entry tests, `h < 16 || v < 15` and its three siblings.
//
// It is not a bounds check on the room: it refuses a rect that would start left of or above
// the screen origin, which for an object in a neighbouring room is reachable. The star has no
// such guard at all -- see TestTheStarHasNoOriginGuard.
func TestTheGuardedFamiliesRefuseANegativeOrigin(t *testing.T) {
	cases := []struct {
		name  string
		add   func(s *Scene, h, v int16)
		table func(s *Scene) []Anim
		h, v  int16 // the smallest legal pair
	}{
		{"candle", func(s *Scene, h, v int16) { s.addCandleFlame(0, 3, h, v) },
			func(s *Scene) []Anim { return s.Flames }, 16, 15},
		{"tiki", func(s *Scene, h, v int16) { s.addTikiFlame(0, 3, h, v) },
			func(s *Scene) []Anim { return s.TikiFlames }, 8, 10},
		{"coals", func(s *Scene, h, v int16) { s.addBBQCoals(0, 3, h, v) },
			func(s *Scene) []Anim { return s.Coals }, 32, 9},
		{"pendulum", func(s *Scene, h, v int16) { s.addPendulum(0, 3, h, v) },
			func(s *Scene) []Anim { return s.Pendulums }, 32, 28},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// One under on either axis is refused.
			for _, bad := range [][2]int16{{c.h - 1, c.v}, {c.h, c.v - 1}} {
				s := animScene(t)
				c.add(s, bad[0], bad[1])
				if got := len(c.table(s)); got != 0 {
					t.Errorf("(%d, %d) registered %d entries, want 0 (the guard is "+
						"h < %d || v < %d)", bad[0], bad[1], got, c.h, c.v)
				}
			}
			// Exactly on the boundary is accepted, which is what makes the two above
			// a statement about `<` rather than about large numbers.
			s := animScene(t)
			c.add(s, c.h, c.v)
			if got := len(c.table(s)); got != 1 {
				t.Errorf("(%d, %d) registered %d entries, want 1: the guard is strict",
					c.h, c.v, got)
			}
		})
	}
}

// TestTheStarHasNoOriginGuard is the absence, stated so it cannot be filled in by analogy.
//
// AddStar checks the table cap and nothing else -- no h/v test at all, unlike its four
// siblings. A star registered at a negative origin claims a slot and bakes a strip from
// whatever the back map has at that rect, which the Surface clips.
func TestTheStarHasNoOriginGuard(t *testing.T) {
	s := animScene(t)
	s.addStar(0, 3, 0, 0)
	if len(s.Stars) != 1 {
		t.Errorf("a star at the origin registered %d entries, want 1: AddStar has no h/v "+
			"guard and adding one would be a divergence", len(s.Stars))
	}
}

// ---------------------------------------------------------------------------
// Re-baking after a lighting change
// ---------------------------------------------------------------------------

// TestReBackUpMatchesOnWhereAndWho pins the lookup, which is by identity and not by index.
//
// The C reaches the same entry the long way round -- it searches savedMaps for (where, who)
// and then searches the animation table for the entry pointing at *that slot index*, because
// its `flames[].who` is a saved-map index rather than an object index. The port's Anim carries
// the object identity too, so this matches directly.
//
// The equivalence is worth stating because of what the C's version depends on: its outer loop
// looks like it should break on a match and must not. The star and the cuckoo are the only
// objects that claim **two** slots under the same (where, who) tag -- one for the object's own
// rect and one for the filmstrip -- and the object-rect slot is found first and matches no
// animation. Add the missing break and stars and pendulums stop being re-baked by a light
// switch. Nothing to transcribe here; the note is in anim.go.
func TestReBackUpMatchesOnWhereAndWho(t *testing.T) {
	s := animScene(t)
	s.addCandleFlame(0, 3, 96, 120)  // room 0, object 3
	s.addCandleFlame(1, 3, 160, 120) // same object slot, different room
	s.addCandleFlame(0, 5, 224, 120) // same room, different object slot

	if len(s.Flames) != 3 {
		t.Fatalf("%d flames registered, want 3", len(s.Flames))
	}
	// A re-bake writes into an existing slot and claims nothing new. That is the only
	// externally visible thing about it with no art loaded, and it is the thing that
	// would break if reBackUp ever fell through to a registration.
	before := len(s.SavedMaps)
	for _, c := range [][2]int16{{0, 3}, {1, 3}, {0, 5}, {1, 5}, {7, 3}, {0, 9}} {
		s.ReBackUpFlames(c[0], c[1])
	}
	if len(s.SavedMaps) != before {
		t.Errorf("re-baking claimed %d new slots; it repaints in place",
			len(s.SavedMaps)-before)
	}
}

// TestReBackUpRepaintsFromTheEntrysCurrentDest is the difference from grease, and the reason
// there is no equivalent of ReBackUpGrease's mid-fall artefact here.
//
// A jar caught mid-tip re-bakes from a Dest that has already walked, so its remaining cels
// hold backgrounds from up to six pixels away. A flame does not move: Dest is the rect the
// registration computed and stays that rect for the life of the locale, so a light switched at
// any moment re-bakes from exactly where the first bake read.
func TestReBackUpRepaintsFromTheEntrysCurrentDest(t *testing.T) {
	s := animScene(t)
	s.addCandleFlame(0, 3, 96, 120)
	before := s.Flames[0].Dest

	// Advance the animation the way RenderFlames would, then re-bake.
	s.Flames[0].Mode = 3
	s.Flames[0].Src = Offset(SetRect(0, 0, 16, 15), 0, 3*15)
	s.ReBackUpFlames(0, 3)

	if s.Flames[0].Dest != before {
		t.Errorf("Dest moved from %+v to %+v; a re-bake reads Dest and never writes it",
			before, s.Flames[0].Dest)
	}
	if s.Flames[0].Mode != 3 {
		t.Errorf("Mode = %d after a re-bake, want the preserved 3: a light switch "+
			"repaints the cels and must not restart the animation", s.Flames[0].Mode)
	}
}

// ---------------------------------------------------------------------------
// The saved-map economy against shipped content
// ---------------------------------------------------------------------------

// TestTheSavedMapBudgetSaturatesInShippedContent is 1.5f's census, and it reports the
// opposite of what the plan expected: **no shipped locale comes close to the 24-slot cap.**
//
// Every animated object in view competes for the same 24 slots. Five families claim a
// filmstrip each, grease claims one, and seven kinds of prize claim one for their own rect so
// they can be erased when collected. The plan assumed some room in 22 houses would be over the
// line, the way the eighteen-slot dinahs cap is over the line in several. It is not: the
// busiest locale in the corpus claims 16, and the corpus is logged so the number is on the
// record rather than in a commit message.
//
// What that changes is how the -1 path has to be justified. It is not dead code -- a house
// authored in Stage 5 can trivially exceed it, four candles and a clock and eight prizes in one
// room will do it -- but nothing in the shipped corpus exercises it, so the synthetic tests
// above are the only coverage it will ever have and the drops are enumerated here from a room
// built for the purpose. If a future change to the drawing order, the window size or the
// neighbour count pushes a shipped room over, this test says which one and what it lost.
//
// The window size is part of the claim. Registration is gated on the object's rect
// intersecting the screen, so the headroom below is the headroom *at 640x480 with nine
// neighbours*; a taller window would see more objects at once. See
// TestASaturatedTableDoesNotConsumeARandomDraw for why that also makes the random stream
// resolution-dependent.
func TestTheSavedMapBudgetSaturatesInShippedContent(t *testing.T) {
	artDir := requireAssets(t, "art")
	houseDir := requireAssets(t, "houses")
	forkRoot := filepath.Join(assetRoot, "houseart")

	paths, err := filepath.Glob(filepath.Join(houseDir, "*.house"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Skipf("no houses in %s", houseDir)
	}
	sort.Strings(paths)

	type census struct {
		house string
		room  int16
		name  string
		used  int
	}
	var busiest census
	rooms, saturated, dropped := 0, 0, 0

	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), ".house")
		h, err := house.LoadFile(path)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		assets := NewAssets(artDir)
		if fork := filepath.Join(forkRoot, name); dirExists(fork) {
			assets.OpenHouseResFork(fork)
		}
		s := NewScene(DefaultView(), assets, h)

		for n := range h.Rooms {
			s.RoomNumber = int16(n)
			s.DrawLocale()
			rooms++

			if len(s.SavedMaps) > busiest.used {
				busiest = census{name, int16(n), h.Rooms[n].Name.Text(), len(s.SavedMaps)}
			}
			if len(s.SavedMaps) >= kMaxSavedMaps {
				saturated++
			}
			if len(s.SavedMapDrops) > 0 {
				dropped++
				t.Logf("%s room %d %q dropped %d registrations: %v",
					name, n, h.Rooms[n].Name.Text(), len(s.SavedMapDrops),
					s.SavedMapDrops)
			}
		}
	}

	t.Logf("busiest locale over %d rooms: %s room %d %q, %d of %d saved-map slots",
		rooms, busiest.house, busiest.room, busiest.name, busiest.used, kMaxSavedMaps)
	t.Logf("%d locales fill the table, %d drop a registration", saturated, dropped)

	if busiest.used == 0 {
		t.Fatal("no shipped locale claims a saved map; the census found nothing and the " +
			"corpus or the composition is broken")
	}
	// Stated as an upper bound rather than an equality, so that content changes and a
	// wider window can move the number without failing. What would fail is a shipped room
	// crossing the line, which is the event worth being told about.
	if dropped != 0 {
		t.Errorf("%d shipped locales now lose an object to the 24-slot cap; they used not "+
			"to. The dropped registrations are logged above -- each one is an object the "+
			"player cannot see", dropped)
	}
	if busiest.used > kMaxSavedMaps {
		t.Errorf("busiest locale claims %d slots against a cap of %d, which is impossible "+
			"and means the cap is not being enforced", busiest.used, kMaxSavedMaps)
	}
}

// TestDroppedRegistrationsNameTheObjectThatVanished builds the room the corpus does not
// contain, and enumerates what it loses.
//
// This is the other half of the census. Nothing shipped saturates, so the only way to state
// what saturation *does* is to cause it: fill the table to 23, then ask for a candle, a star
// and a prize slot. Every one of the three is refused, and the report says which object asked
// and how big a patch it wanted -- which names the family, since no two request the same size.
//
// The consequence to keep in view is the one in backUpToSavedMap's own comment: a refused
// object is not merely un-animated, it is **invisible**, because every caller gates the draw on
// the slot. A house author whose room is one object over budget loses a prize with no warning
// at all in the original. That is what SavedMapDrops is for.
func TestDroppedRegistrationsNameTheObjectThatVanished(t *testing.T) {
	s := animScene(t)
	for len(s.SavedMaps) < kMaxSavedMaps-1 {
		s.SavedMaps = append(s.SavedMaps, SavedMap{Where: -1, Who: -1})
	}

	// One slot left. The candle takes it.
	s.addCandleFlame(0, 1, 96, 120)
	if len(s.SavedMapDrops) != 0 {
		t.Fatalf("the last free slot was refused: %v", s.SavedMapDrops)
	}

	// Everything after it is refused, and each refusal is recorded once.
	s.addStar(0, 2, 96, 200)
	s.addPendulum(0, 3, 96, 280)
	s.backUpToSavedMap(SetRect(0, 0, 64, 43), 0, 4, false) // a prize's own rect

	want := []SavedMapDrop{
		{Where: 0, Who: 2, Rect: SetRect(0, 0, 32, 31*NumStarFrames)},
		{Where: 0, Who: 3, Rect: SetRect(0, 0, 32, 28*NumPendulumFrames)},
		{Where: 0, Who: 4, Rect: SetRect(0, 0, 64, 43)},
	}
	if len(s.SavedMapDrops) != len(want) {
		t.Fatalf("dropped %v, want %v", s.SavedMapDrops, want)
	}
	for i := range want {
		if s.SavedMapDrops[i] != want[i] {
			t.Errorf("drop %d = %+v, want %+v", i, s.SavedMapDrops[i], want[i])
		}
	}

	// And the refusals really did leave the objects out of their tables, which is the
	// difference between "not animated" and "not there".
	if len(s.Stars) != 0 || len(s.Pendulums) != 0 {
		t.Errorf("a refused registration was appended anyway: %d stars, %d pendulums",
			len(s.Stars), len(s.Pendulums))
	}

	// A recompose clears the report, so it always describes the current locale.
	s.DrawLocale()
	if len(s.SavedMapDrops) != 0 {
		t.Errorf("%d drops survived a recompose", len(s.SavedMapDrops))
	}
}

// dirExists is TestComposeEveryRoom's fork test, which the census needs too.
func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

// TestBakeStripSurvivesABadSlot is the port's guard, not the original's.
//
// The C indexes savedMaps[index] unconditionally. Here two things can make that unsafe: an
// out-of-range index, and a slot whose Map is nil -- which is the state a render test that
// composes with no art leaves behind, and which would otherwise be a nil dereference in the
// middle of a frame rather than a missing flame.
func TestBakeStripSurvivesABadSlot(t *testing.T) {
	s := animScene(t)
	s.SavedMaps = append(s.SavedMaps, SavedMap{Map: nil, Where: 0, Who: 3})

	for _, slot := range []int{-1, len(s.SavedMaps), 500, 0} {
		s.backUpFlames(SetRect(96, 120, 112, 135), slot)
	}
	// Reaching here without a panic is the assertion; an empty cel table is the other
	// early return and is checked the same way.
	s.bakeStrip(SetRect(0, 0, 16, 15), 0, "blower", nil)
}
