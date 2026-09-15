package game

// The three opt-in corrections, tested in the state nothing else tests: on.
//
// Off is the original's behaviour and is already pinned elsewhere, by the tests that were
// written when each defect was transcribed -- TestTheSpuriousSparkleLandsAtTheCorner for the
// sparkle, and the fidelity corpus for the two mirror lines. So each test here checks three
// things: that the flag changes what the code does, that the *unflagged* path is still the
// one the corpus expects, and that the correction is inert where the defect was harmless.
//
// The last of those is the one worth having. Both mirror flags are one term added to one
// expression, and the failure mode of a badly written fix is not "the bug remains" but "a
// room with no mirror now draws nothing".

import (
	"testing"

	"glidergo/internal/game/player"
	"glidergo/internal/house"
	"glidergo/internal/render"
)

// The reflected glider: a 48x15 sprite at a known place in the room, with two flat-coloured
// sheets so that "which sheet was blitted" is one pixel read.
const (
	fixSheetWide = 48
	fixSheetTall = 15

	fixGlidIndex  = 10 // GlidSrc: player one's own sheet
	fixGlid2Index = 20 // Glid2Src: the foil sheet in a one-player game, player two's in a two
)

// mirrorWorld is a world set up for one DrawReflection call.
//
// The sheets are synthetic rather than extracted, which is what makes this test run on a
// checkout with no assets -- and it costs nothing, because the question is which of two
// surfaces was read, not what was in it. A nil Mask is fully opaque (Surface.opaque), so
// render.Masked writes every pixel and the whole destination carries the answer.
func mirrorWorld(t *testing.T) (*World, Rect) {
	t.Helper()
	w := newTestWorld(oneRoomHouse(house.ObjectIsEmpty), "", "")

	w.GlidSrc = render.NewSurface(fixSheetWide, fixSheetTall)
	fillSurface(w.GlidSrc, fixGlidIndex)
	w.Glid2Src = render.NewSurface(fixSheetWide, fixSheetTall)
	fillSurface(w.Glid2Src, fixGlid2Index)

	w.P1.Which = player.Player1
	w.P1.Src = player.Rect(render.SetRect(0, 0, fixSheetWide, fixSheetTall))
	w.P1.Dest = player.Rect(render.SetRect(100, 100, 100+fixSheetWide, 100+fixSheetTall))
	w.P1.Whole = w.P1.Dest

	// Where the reflection lands: the C's fixed (-20,-16) parallax off the glider's own
	// position, in the work map's coordinates.
	dest := render.Offset(Rect(w.P1.Dest),
		w.R.V.OriginH+player.MirrorOffsetH, w.R.V.OriginV+player.MirrorOffsetV)
	return w, dest
}

// sheetAt reads the index at the centre of a rect, which for a flat-filled sheet is the
// sheet's identity.
func sheetAt(w *World, r Rect) uint8 {
	h, v := int(r.Left+r.Wide()/2), int(r.Top+r.Tall()/2)
	return w.R.Work.Pix[v*w.R.Work.W+h]
}

// ---------------------------------------------------------------------------
// MirrorFoil (2.20)
// ---------------------------------------------------------------------------

// TestMirrorFoilCorrectsTheSheetInATwoPlayerGame is the whole of the defect: in a two-player
// game with foil showing, DrawReflection's bare `showFoil` reaches for glid2SrcMap, which in
// a two-player game holds player *two's* artwork rather than the foil sheet the test was
// after. So player one's reflection is drawn as the other player's glider.
func TestMirrorFoilCorrectsTheSheetInATwoPlayerGame(t *testing.T) {
	for _, tc := range []struct {
		name string
		fix  bool
		want uint8
	}{
		{"the original", false, fixGlid2Index},
		{"corrected", true, fixGlidIndex},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, dest := mirrorWorld(t)
			w.TwoPlayer = true
			w.ShowFoil = true
			w.Fix.MirrorFoil = tc.fix
			w.R.MirrorRects = []Rect{dest}

			w.DrawReflection(&w.P1, true)

			if got := sheetAt(w, dest); got != tc.want {
				t.Errorf("the reflection was drawn from index %d, want %d", got, tc.want)
			}
		})
	}
}

// TestMirrorFoilChangesNothingInAOnePlayerGame: the guard it adds is `!twoPlayerGame`, so in
// the game almost everybody plays it must be inert -- and inert means the foil sheet, which
// in a one-player game is what the second slot holds. A fix that turned the foil reflection
// off would be a worse bug than the one it corrected, and it would look right in every test
// that only ever asserted "the two flags differ".
func TestMirrorFoilChangesNothingInAOnePlayerGame(t *testing.T) {
	for _, fix := range []bool{false, true} {
		w, dest := mirrorWorld(t)
		w.TwoPlayer = false
		w.ShowFoil = true
		w.Fix.MirrorFoil = fix
		w.R.MirrorRects = []Rect{dest}

		w.DrawReflection(&w.P1, true)

		if got := sheetAt(w, dest); got != fixGlid2Index {
			t.Errorf("MirrorFoil=%v: drawn from index %d, want the foil sheet's %d",
				fix, got, fixGlid2Index)
		}
	}
}

// ---------------------------------------------------------------------------
// MirrorFlame (2.19)
// ---------------------------------------------------------------------------

// TestMirrorFlameClipsTheBackRect is the flame blink, in the only form it can be measured
// without a candle: the erase list.
//
// The C registers the unclipped destination, so the next frame's restore covers ground the
// draw never touched -- and a candle flame or a pendulum in that ground, which redraws itself
// opaquely rather than registering a back rect of its own, is wiped a frame early. The fix
// registers what was actually drawn.
func TestMirrorFlameClipsTheBackRect(t *testing.T) {
	w, dest := mirrorWorld(t)
	// A mirror over the left half of the reflection: the case that has something to clip.
	mirror := dest
	mirror.Right = dest.Left + dest.Wide()/2
	w.R.MirrorRects = []Rect{mirror}

	w.Fix.MirrorFlame = false
	w.DrawReflection(&w.P1, true)
	if len(w.Back2Work) != 1 || w.Back2Work[0] != dest {
		t.Fatalf("Back2Work = %+v, want the one unclipped rect %+v: the original's "+
			"behaviour has changed and the fidelity corpus depends on it", w.Back2Work, dest)
	}

	w.Back2Work = w.Back2Work[:0]
	w.Fix.MirrorFlame = true
	w.DrawReflection(&w.P1, true)
	if len(w.Back2Work) != 1 || w.Back2Work[0] != mirror {
		t.Errorf("Back2Work = %+v, want the clipped rect %+v", w.Back2Work, mirror)
	}
}

// TestMirrorFlameRegistersOneRectPerMirror: mirrorRgn is a union of one rect per mirror
// object, so a room with two mirrors has two places the reflection was really drawn. Both
// have to be erased, and this is where the fix costs more than the defect -- two entries in
// a 47-slot list instead of one, which is still fewer than the unclipped rect costs in
// erased flames.
func TestMirrorFlameRegistersOneRectPerMirror(t *testing.T) {
	w, dest := mirrorWorld(t)
	left, right := dest, dest
	left.Right = dest.Left + 8
	right.Left = dest.Right - 8
	w.R.MirrorRects = []Rect{left, right}
	w.Fix.MirrorFlame = true

	w.DrawReflection(&w.P1, true)

	if len(w.Back2Work) != 2 {
		t.Fatalf("Back2Work = %+v, want two rects, one per mirror", w.Back2Work)
	}
	if w.Back2Work[0] != left || w.Back2Work[1] != right {
		t.Errorf("Back2Work = %+v, want [%+v %+v]", w.Back2Work, left, right)
	}
}

// TestMirrorFlameRegistersNothingOutsideAMirror: a reflection entirely outside the mirrors
// draws nothing, so there is nothing to erase. The C would register the rect anyway, which
// is the defect in its purest form -- an erase with no draw behind it.
func TestMirrorFlameRegistersNothingOutsideAMirror(t *testing.T) {
	w, dest := mirrorWorld(t)
	away := render.Offset(dest, 0, dest.Tall()+40)
	w.R.MirrorRects = []Rect{away}
	w.Fix.MirrorFlame = true

	w.DrawReflection(&w.P1, true)

	if len(w.Back2Work) != 0 {
		t.Errorf("Back2Work = %+v, want none: nothing was drawn inside a mirror", w.Back2Work)
	}
	if got := sheetAt(w, dest); got == fixGlidIndex || got == fixGlid2Index {
		t.Error("the reflection was drawn outside every mirror; the clip is not being applied")
	}
}

// ---------------------------------------------------------------------------
// SwitchSparkle (2.39)
// ---------------------------------------------------------------------------

// TestSwitchSparkleDropsTheSpuriousPuffAndKeepsTheRealOne.
//
// The prize arm emits two sparkles: RestoreFromSavedMap's, which is placed on the prize and
// is the one the player is meant to see, and a second on the uninitialised `bounds`, which
// in this port is the zero rect and therefore the corner of the play area. The fix must drop
// exactly the second -- a switch that removed a prize with no puff of light at all would be
// a worse outcome than the stray puff, because the puff is how the player learns that
// something they were not looking at has just vanished.
func TestSwitchSparkleDropsTheSpuriousPuffAndKeepsTheRealOne(t *testing.T) {
	w, who := switchWorld(t, LightSwitch, prizeObj(RedClock), Toggle, 1, false)
	w.Fix.SwitchSparkle = true

	w.HandleSwitches(who)

	if w.NumSparkles != 1 {
		t.Fatalf("NumSparkles = %d, want 1: the restore's puff and no other", w.NumSparkles)
	}
	onPrize := w.Sparkles[0].Bounds
	prizeOnScreen := render.Offset(prizeAt, w.R.V.OriginH, w.R.V.OriginV)
	if onPrize.Left < prizeOnScreen.Left || onPrize.Left >= prizeOnScreen.Right {
		t.Errorf("Sparkles[0] at %+v is not over the prize at %+v: the fix has dropped the "+
			"wrong sparkle", onPrize, prizeOnScreen)
	}
}

// TestSwitchSparkleFreesTheSlotItWasWasting is the second half of the argument for the flag.
// The sparkle table has three slots and AddSparkle silently drops the fourth request, so the
// stray puff is not merely ugly -- it spends a third of the room's effect budget on a puff
// nobody asked for.
func TestSwitchSparkleFreesTheSlotItWasWasting(t *testing.T) {
	count := func(fix bool) int16 {
		w, who := switchWorld(t, LightSwitch, prizeObj(RedClock), Toggle, 1, false)
		w.Fix.SwitchSparkle = fix
		w.HandleSwitches(who)
		return w.NumSparkles
	}
	if with, without := count(true), count(false); with >= without {
		t.Errorf("NumSparkles = %d with the fix and %d without; the fix should use one "+
			"fewer slot", with, without)
	}
}
