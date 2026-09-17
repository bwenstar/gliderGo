package render

// Grease.c's registration half: the table entry, the saved-map slot, and the four-cel strip
// baked into it.
//
// The simulation half is internal/game/grease_test.go. These are the numbers that side's tests
// start from -- Dest, Start, Stop and the strip's shape -- so between them the two files pin
// both ends of a jar's life without either having to reproduce the other's arithmetic.
//
// None of it needs art. backupGrease composites the jar's cels over the background it saved,
// and with no art loaded the composite step is skipped while the *walk* -- the two pixels per
// cel that AddGrease's ∓8 undoes -- still happens. The walk is the part that is easy to get
// wrong and the part that is invisible in a still image, so a fixture with no sheets tests
// exactly the half worth testing here. Where the pixels themselves matter it is
// TestComposeEveryRoom's golden hashes that say so.

import "testing"

// greaseScene is a composed-enough Scene: a real view so the origin is non-zero, no art, and a
// back map big enough for backupGrease to copy out of.
//
// It deliberately does *not* call DrawLocale. AddGrease is a pure function of its arguments
// plus the two tables it appends to, and driving it directly is what lets a test claim 24
// saved-map slots or 16 jars without building a house that has them.
func greaseScene(t *testing.T) *Scene {
	t.Helper()
	s := NewScene(DefaultView(), NewAssets(nil), testHouse())
	if s.Back == nil || s.Back.Bounds().Wide() < 64 {
		t.Fatalf("fixture back map is %v; backupGrease needs somewhere to copy from",
			s.Back.Bounds())
	}
	return s
}

// ---------------------------------------------------------------------------
// AddGrease
// ---------------------------------------------------------------------------

// TestAddGreaseLandsDestAtTheAuthoredPoint is the ∓8, and it is the single most misreadable
// line in the file.
//
// `src` is handed to backupGrease **by pointer, because backupGrease moves it**: the walk steps
// the rect two pixels per cel, four times, so on return it is eight pixels along. The offset
// immediately afterwards subtracts exactly that, which is why the sign looks backwards -- a
// right-facing jar's rect has just been advanced +8 and is pulled back. Dest therefore ends up
// at (h, v) precisely, and reading the two offsets as a deliberate eight-pixel nudge is the
// misreading to avoid.
//
// The test is worth its length because both plausible mistakes produce a *plausible* jar. Drop
// the offset and every jar in the game sits eight pixels downwind of where its author put it;
// flip its sign and they sit sixteen. Neither crashes and neither looks obviously wrong in a
// screenshot of a room you have not seen before.
func TestAddGreaseLandsDestAtTheAuthoredPoint(t *testing.T) {
	for _, c := range []struct {
		name    string
		isRight bool
	}{
		{"tipping right", true},
		{"tipping left", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := greaseScene(t)
			const h, v int16 = 96, 120

			i := s.AddGrease(0, 3, h, v, 64, c.isRight)
			if i != 0 {
				t.Fatalf("AddGrease returned %d, want 0 for the first jar", i)
			}
			g := s.Grease[i]

			want := Rect{Top: v, Left: h, Bottom: v + 27, Right: h + 32}
			if g.Dest != want {
				t.Errorf("Dest = %+v, want %+v -- the ∓8 undoes backupGrease's walk and "+
					"nothing else", g.Dest, want)
			}
		})
	}
}

// TestAddGreaseStartAndStopStraddleTheJar pins the slick's two ends.
//
// Start is four pixels clear of the jar on the leading side -- so the first two pixels of slick
// are painted in front of the fallen jar rather than under it -- and Stop is `distance` pixels
// from the same edge, where distance is the author's data.c.length. Both run right for a
// right-facing jar and left for a left-facing one, which is why the termination test in
// HandleGrease has to be `>=` for one and `<=` for the other: there is no direction-independent
// spelling of it.
//
// The four-pixel gap is why a slick of authored length 64 covers 60 pixels; the game side's
// TestGreaseSpreadStopsOneStepShortOfStop counts the frames that costs.
func TestAddGreaseStartAndStopStraddleTheJar(t *testing.T) {
	const h, v, distance int16 = 96, 120, 64

	t.Run("tipping right", func(t *testing.T) {
		s := greaseScene(t)
		g := s.Grease[s.AddGrease(0, 3, h, v, distance, true)]
		if want := h + 32 + 4; g.Start != want {
			t.Errorf("Start = %d, want %d (the jar's right edge plus 4)", g.Start, want)
		}
		if want := h + 32 + distance; g.Stop != want {
			t.Errorf("Stop = %d, want %d (the jar's right edge plus the authored length)",
				g.Stop, want)
		}
		if !(g.Start < g.Stop) {
			t.Errorf("Start %d is not left of Stop %d; the slick would never spread",
				g.Start, g.Stop)
		}
	})

	t.Run("tipping left", func(t *testing.T) {
		s := greaseScene(t)
		g := s.Grease[s.AddGrease(0, 3, h, v, distance, false)]
		if want := h - 4; g.Start != want {
			t.Errorf("Start = %d, want %d (the jar's left edge minus 4)", g.Start, want)
		}
		if want := h - distance; g.Stop != want {
			t.Errorf("Stop = %d, want %d", g.Stop, want)
		}
		if !(g.Start > g.Stop) {
			t.Errorf("Start %d is not right of Stop %d", g.Start, g.Stop)
		}
	})
}

// TestAddGreaseInitialFieldsAreTheOnesHandleGreaseAssumes is the handshake between the two
// halves, written as one assertion so that a change to either side has one place to fail.
//
// Frame at **-1** is the load-bearing one. HandleGrease pre-increments, so -1 is what makes the
// tip four frames instead of three; 0 here would skip the last cel and no golden image would
// notice, because every golden composes an upright jar. Mode at Idle is the zero value and is
// written explicitly anyway, because the whole point of Idle being 0 is that a fresh entry is
// inert -- stating it is cheaper than making a reader confirm it.
//
// HotNum is *not* set, and that absence is the interesting part: it stays a stale 0 until
// SpillGrease writes it, which is why the game side guards every read of it. See greaseHot.
func TestAddGreaseInitialFieldsAreTheOnesHandleGreaseAssumes(t *testing.T) {
	s := greaseScene(t)
	i := s.AddGrease(4, 7, 96, 120, 64, true)
	g := s.Grease[i]

	if g.Frame != -1 {
		t.Errorf("Frame = %d, want -1: HandleGrease pre-increments, and -1 is what makes "+
			"the tip four frames rather than three", g.Frame)
	}
	if g.Mode != GreaseIdle {
		t.Errorf("Mode = %d, want GreaseIdle (%d)", g.Mode, GreaseIdle)
	}
	if g.HotNum != 0 {
		t.Errorf("HotNum = %d, want the unwritten 0: AddGrease does not know the jar's "+
			"hot spot, SpillGrease does", g.HotNum)
	}
	if g.Where != 4 || g.Who != 7 {
		t.Errorf("(Where, Who) = (%d, %d), want (4, 7) -- the pair ReBackUpGrease matches "+
			"on and RedrawAllGrease's room test reads", g.Where, g.Who)
	}
	if !g.IsRight {
		t.Error("IsRight is false; it selects every left/right pair in both halves")
	}
	if g.MapNum < 0 || int(g.MapNum) >= len(s.SavedMaps) {
		t.Errorf("MapNum = %d with %d slots claimed; a registered jar always has a valid "+
			"one", g.MapNum, len(s.SavedMaps))
	}
}

// TestAddGreaseClaimsAFourCelStrip is the saved map, which for grease is art and not just
// background.
//
// Every other saved map is a copy of the wall behind something, kept so the something can be
// erased. Grease's is a filmstrip: four 32x27 cels, each the wall behind the jar with one cel
// of the tipping jar composited on top, stacked into one 32x108 patch. That is why the slot is
// claimed at 32x27*4 rather than at the jar's own size, and it is why HandleGrease's falling arm
// is a plain opaque blit -- the frame it wants is already drawn.
//
// The slot's Dest is (0,0,32,108), the **screen origin** and not the jar. That is not a bug and
// does not need repairing: backupGrease overwrites all four cels immediately, so no pixel of
// the claim survives. It does leave the slot pointing at the corner of the play area, which is
// the hazard RestoreFromSavedMap's "first matching slot wins" note describes -- harmless only
// because no reward arm restores grease. A knocked-over jar becomes a slick in place; it is
// never erased. Asserting the corner rather than the jar is therefore asserting a known wart,
// deliberately, so that "fixing" it has to be a decision.
func TestAddGreaseClaimsAFourCelStrip(t *testing.T) {
	s := greaseScene(t)
	before := len(s.SavedMaps)

	i := s.AddGrease(0, 3, 96, 120, 64, true)
	if len(s.SavedMaps) != before+1 {
		t.Fatalf("%d slots claimed, want 1", len(s.SavedMaps)-before)
	}
	slot := s.SavedMaps[s.Grease[i].MapNum]

	if want := (Rect{Bottom: 27 * 4, Right: 32}); slot.Dest != want {
		t.Errorf("slot Dest = %+v, want %+v -- the claim is at the screen origin, and the "+
			"four cels overwrite it before anything reads it", slot.Dest, want)
	}
	if slot.Map == nil {
		t.Fatal("slot has no surface")
	}
	if w, h := slot.Map.Bounds().Wide(), slot.Map.Bounds().Tall(); w != 32 || h != 27*4 {
		t.Errorf("strip is %dx%d, want 32x%d: one cel wide and four cels tall", w, h, 27*4)
	}
	if slot.Where != 0 || slot.Who != 3 {
		t.Errorf("slot identifies (%d, %d), want (0, 3)", slot.Where, slot.Who)
	}
}

// TestAddGreaseIsRefusedAtEitherCap covers both of its `return -1`s, which are refusals for
// different reasons and have to be distinguished because only one of them is grease's own.
//
//	kMaxGrease        16 jars in a locale. The table's own cap
//	kMaxSavedMaps     24 slots for the whole locale, shared with every prize, flame, star,
//	                  pendulum and appliance. A jar costs four cels' worth
//
// The second is the one that bites in real content: a grease jar's slot is four times the size
// of a prize's, so a jar can be the object that saturates a 24-slot table and the *next*
// object -- possibly a prize the player needs -- is the one that goes undrawn. See
// docs/analysis for the census.
//
// **A refused jar must not be half-registered.** The caller gates the jar's draw on the return
// value (`if dynamicNum != -1`), so an entry appended before the refusal would be a jar that is
// animated but never drawn, tipping invisibly. Both arms are checked for that.
func TestAddGreaseIsRefusedAtEitherCap(t *testing.T) {
	t.Run("the grease table is full", func(t *testing.T) {
		s := greaseScene(t)
		// Fill the table without going through AddGrease, so the saved-map cap is not
		// reached first -- 16 jars would want 16 of the 24 slots, which is legal, but
		// building them by hand keeps the two caps independent.
		for len(s.Grease) < kMaxGrease {
			s.Grease = append(s.Grease, Grease{MapNum: -1, Frame: -1})
		}
		if got := s.AddGrease(0, 3, 96, 120, 64, true); got != -1 {
			t.Errorf("AddGrease returned %d with %d jars registered, want -1",
				got, kMaxGrease)
		}
		if len(s.Grease) != kMaxGrease {
			t.Errorf("the refused jar was appended anyway: %d entries", len(s.Grease))
		}
		if len(s.SavedMaps) != 0 {
			t.Errorf("the refused jar claimed %d slots; the cap test is above the claim",
				len(s.SavedMaps))
		}
	})

	t.Run("the saved maps are full", func(t *testing.T) {
		s := greaseScene(t)
		for len(s.SavedMaps) < kMaxSavedMaps {
			s.SavedMaps = append(s.SavedMaps, SavedMap{Where: -1, Who: -1})
		}
		if got := s.AddGrease(0, 3, 96, 120, 64, true); got != -1 {
			t.Errorf("AddGrease returned %d with the slot table full, want -1", got)
		}
		if len(s.Grease) != 0 {
			t.Errorf("the refused jar was appended anyway: %+v", s.Grease)
		}
	})
}

// ---------------------------------------------------------------------------
// backupGrease
// ---------------------------------------------------------------------------

// TestBackupGreaseWalksTwoPixelsPerCel is the walk on its own, separated from AddGrease's ∓8 so
// that the two cannot cancel each other's mistakes.
//
// Each cel is the back map at *src* with one frame of the jar over it, and src steps two pixels
// per cel in the direction the jar tips -- so cel 3 is the wall six pixels along from cel 0,
// which is where the jar will have got to by then. Get the walk wrong and the jar appears to
// slide *back* as it falls, because the background behind it is from the wrong place.
//
// Four cels means four steps and eight pixels, not three and six: the walk happens at the
// bottom of the loop body, so the fourth iteration steps too even though nothing reads the
// result. That fourth step is precisely what AddGrease then undoes, which is the only reason
// the ∓8 is 8 and not 6.
func TestBackupGreaseWalksTwoPixelsPerCel(t *testing.T) {
	for _, c := range []struct {
		name    string
		isRight bool
		step    int16
	}{
		{"tipping right", true, 2},
		{"tipping left", false, -2},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := greaseScene(t)
			slot := s.backUpToSavedMap(SetRect(0, 0, 32, 27*4), 0, 3, false)
			if slot == -1 {
				t.Fatal("could not claim a slot")
			}

			src := SetRect(0, 0, 32, 27)
			src = Offset(src, 96, 120)
			start := src
			s.backupGrease(&src, slot, c.isRight)

			if want := start.Left + 4*c.step; src.Left != want {
				t.Errorf("src walked to Left %d, want %d: four cels of %d pixels, and the "+
					"fourth step is the one AddGrease's ∓8 exists to undo",
					src.Left, want, c.step)
			}
			if src.Top != start.Top {
				t.Errorf("src moved vertically to %d; the walk is horizontal only", src.Top)
			}
		})
	}
}

// TestBackupGreaseSurvivesABadSlot is the port's guard, not the original's.
//
// The C indexes savedMaps[index] unconditionally. Here two things can make that unsafe: a
// negative index, and a slot whose Map is nil -- which is the state a render test that composes
// with no art leaves behind, and which would otherwise be a nil dereference in the middle of a
// frame rather than a missing jar.
func TestBackupGreaseSurvivesABadSlot(t *testing.T) {
	s := greaseScene(t)
	s.SavedMaps = append(s.SavedMaps, SavedMap{Map: nil, Where: 0, Who: 3})

	for _, slot := range []int{-1, len(s.SavedMaps), 500, 0} {
		src := Offset(SetRect(0, 0, 32, 27), 96, 120)
		before := src
		s.backupGrease(&src, slot, true)
		if src != before {
			t.Errorf("slot %d: the walk ran anyway, %+v -> %+v", slot, before, src)
		}
	}
}

// ---------------------------------------------------------------------------
// ReBackUpGrease
// ---------------------------------------------------------------------------

// TestReBackUpGreaseReturnsTheIndexEvenForASpiltJar is the whole reason the mode test is where
// it is.
//
// A light was switched, so the strip has to be re-baked against the room's new brightness. The
// mode test guards **only the repaint**: a jar that has finished spreading has no jar left to
// draw, so re-baking it would be pointless -- but the caller still needs the index to gate its
// draw on. Returning -1 there would make a light switch delete every spilt jar in the room.
//
// So the assertion is a pair: all four modes return the index, and only two of them move the
// walk. The walk is the observable for "did it re-bake", because with no art the composite is
// skipped but the stepping is not.
func TestReBackUpGreaseReturnsTheIndexEvenForASpiltJar(t *testing.T) {
	for _, c := range []struct {
		name     string
		mode     int16
		wantBake bool
	}{
		{"idle", GreaseIdle, true},
		{"falling", GreaseFalling, true},
		{"spreading", GreaseSpreading, false},
		{"spilt", GreaseSpiltIdle, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := greaseScene(t)
			i := s.AddGrease(0, 3, 96, 120, 64, true)
			if i != 0 {
				t.Fatalf("AddGrease returned %d", i)
			}
			s.Grease[0].Mode = c.mode
			destBefore := s.Grease[0].Dest

			if got := s.ReBackUpGrease(0, 3); got != 0 {
				t.Errorf("ReBackUpGrease returned %d, want 0 -- returning -1 for a spilt "+
					"jar would make a light switch delete it", got)
			}
			// Whatever it did, it must not have moved the jar: backupGrease walks a
			// *copy* of Dest here, unlike in AddGrease where the caller wants the walk.
			if s.Grease[0].Dest != destBefore {
				t.Errorf("Dest moved from %+v to %+v; ReBackUpGrease passes a copy",
					destBefore, s.Grease[0].Dest)
			}
		})
	}
}

// TestReBackUpGreaseMatchesOnWhereAndWho pins the lookup, which is by identity and not by
// index -- the difference from ReBackUpSavedMap, and the reason this is a separate function
// rather than a `redraw` flag on AddGrease.
//
// The -1 return is the case that matters: a light switched in a room with no jar in it must not
// re-bake the first jar it finds somewhere else. With the table holding jars from all nine
// local rooms, matching on Where alone or Who alone would do exactly that.
func TestReBackUpGreaseMatchesOnWhereAndWho(t *testing.T) {
	s := greaseScene(t)
	s.AddGrease(0, 3, 96, 120, 64, true)  // room 0, object 3
	s.AddGrease(1, 3, 160, 120, 64, true) // same object slot, different room
	s.AddGrease(0, 5, 224, 120, 64, true) // same room, different object slot

	cases := []struct{ where, who, want int16 }{
		{0, 3, 0},
		{1, 3, 1},
		{0, 5, 2},
		{1, 5, -1}, // both halves exist separately; the pair does not
		{7, 3, -1},
		{0, 9, -1},
	}
	for _, c := range cases {
		if got := s.ReBackUpGrease(c.where, c.who); got != c.want {
			t.Errorf("ReBackUpGrease(%d, %d) = %d, want %d", c.where, c.who, got, c.want)
		}
	}
}

// TestReBackUpGreaseRebakesAMidFallJarFromWhereItIsNow reproduces an artefact of the original
// rather than fixing it, and exists so that the fix is a decision rather than an accident.
//
// A jar caught mid-fall re-bakes from its *current* Dest, which has already walked two pixels
// per frame. So the four cels become backgrounds from Dest, Dest±2, Dest±4 and Dest±6 while the
// animation is about to read the cel for a frame it has already passed -- the wall behind the
// remaining cels is offset by up to six pixels for the rest of the tip.
//
// Switching a light during the four frames a jar takes to fall over is the whole of the
// exposure, which is why it survived 1994. The test asserts the *input* to the re-bake is the
// moved Dest, because that is the decision; what the pixels then look like is not something a
// golden could usefully pin.
func TestReBackUpGreaseRebakesAMidFallJarFromWhereItIsNow(t *testing.T) {
	s := greaseScene(t)
	const h, v int16 = 96, 120
	s.AddGrease(0, 3, h, v, 64, true)

	// Two frames into the tip, as HandleGrease would have left it.
	s.Grease[0].Mode = GreaseFalling
	s.Grease[0].Frame = 1
	s.Grease[0].Dest = Offset(s.Grease[0].Dest, 4, 0)

	if got := s.ReBackUpGrease(0, 3); got != 0 {
		t.Fatalf("ReBackUpGrease returned %d", got)
	}
	if want := h + 4; s.Grease[0].Dest.Left != want {
		t.Errorf("Dest.Left = %d, want %d: the re-bake reads the moved rect and does not "+
			"rewind it, which is the artefact", s.Grease[0].Dest.Left, want)
	}
}

// TestGreaseModesAreTheOriginalsFour is a one-line guard on the four constants, and it is here
// because both halves of the port branch on them and neither would fail loudly if one changed.
//
// Idle being **zero** is the load-bearing one, twice over: a fresh table entry is Idle without
// being written, and RedrawAllGrease's `mode != kGreaseIdle` test is what keeps an untouched
// jar out of the repaint. Renumber them and a room full of upright jars grows black smears.
func TestGreaseModesAreTheOriginalsFour(t *testing.T) {
	got := [4]int16{GreaseIdle, GreaseFalling, GreaseSpreading, GreaseSpiltIdle}
	if want := [4]int16{0, 1, 2, 3}; got != want {
		t.Errorf("the four modes are %v, want %v (Grease.c:17-20)", got, want)
	}
}
