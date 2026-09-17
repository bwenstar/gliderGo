package game

// The frame loop, end to end: NewGame composes a house's first room, PlayGame steps it,
// and the two clocks and the two dirty-rect lists advance the way the original's do.
//
// These are the first tests in the package that run the *whole* game rather than one
// function of it, so they are also the first that can fail for a reason no unit test can
// see -- a call left out of PlayGame, a hook the host has to supply and does not, a rect
// in the wrong coordinate system. That is what they are for.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/game/player"
	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/render"
)

// playTestWorld loads a house with its art and returns a world ready for NewGame, with a
// budget of `budget` simulated frames wired into Present.
//
// Present is the *sampling point* and Frame is the quantity, and those have to be two
// different things: **Present is not called once per frame.** A room transition presents
// once per wipe strip -- 116 or 160 times inside a single game frame (see screen.go's
// WipeScreenOn) -- and HideGlider presents once on its own. A budget that counted presents
// therefore bought a wildly variable number of frames: it read as "100 frames" and delivered
// 61 in Art Museum, where the unattended glider reaches a transporter on frame 46 and the
// wipe eats the remaining 38. That is what this signature used to say and it was wrong.
//
// A budget of 0 stops the game at NewGame's DumpScreenOn, before the loop runs at all,
// which is what the set-up tests want: they assert on the state NewGame leaves behind and
// a single simulated frame would already have moved the glider.
func playTestWorld(t *testing.T, houseName string, budget int) *World {
	t.Helper()
	houses := requireAssets(t, "houses")
	art := requireAssets(t, "art")

	h, err := house.LoadFile(filepath.Join(houses, houseName+".house"))
	if err != nil {
		t.Fatalf("load %s: %v", houseName, err)
	}
	fork := filepath.Join(assetRoot, "houseart", houseName)
	if _, err := os.Stat(fork); err != nil {
		fork = ""
	}

	w := newTestWorld(h, art, fork)
	w.DoBackground = true

	w.Present = func() {
		if w.Frame >= int64(budget) {
			w.Quitting = true
			w.SwitchedOut = false
		}
	}
	return w
}

// litPixels counts how much of a surface is not black.
//
// The port's black is palette index 255 (render.Black8) and every surface starts filled
// with index 0, which is white -- so "all black" is a state something has to have written,
// and it is the state a composition that silently drew nothing leaves behind after
// NewGame's two black fills. Counting is the cheapest assertion that separates "the room
// composed" from "the room is a hole".
func litPixels(s *render.Surface) int {
	n := 0
	for _, p := range s.Pix {
		if p != render.Black8 {
			n++
		}
	}
	return n
}

// TestRebuildComposesIntoWorkAndBack is the narrowest form of the whole-game test: a room
// change alone has to leave a picture in both offscreens.
//
// It is separate from the NewGame tests because it isolates the one link in the chain that
// has no game state in it at all. If this passes and the NewGame test does not, the fault
// is in NewGame's order; if this fails, the fault is in the renderer or the assets and no
// amount of reading Play.c will find it.
func TestRebuildComposesIntoWorkAndBack(t *testing.T) {
	w := playTestWorld(t, "Slumberland", 0)
	w.ForceThisRoom(w.GetFirstRoomNumber())
	w.Rebuild()

	back := litPixels(w.R.Back)
	work := litPixels(w.R.Work)
	total := w.R.Back.W * w.R.Back.H

	if back == 0 {
		t.Fatalf("Rebuild left the background entirely black (%d pixels)", total)
	}
	// RestoreWorkMap is DrawLocale's last statement, so the two are equal here and the
	// equality is the invariant worth pinning: every dirty rect the frame loop copies
	// back to work assumes it.
	if work != back {
		t.Errorf("work has %d lit pixels, back has %d; RestoreWorkMap should have made them equal", work, back)
	}
	if back < total/10 {
		t.Errorf("only %d of %d pixels lit; a composed room should cover most of the view", back, total)
	}
}

// TestNewGamePutsTheFirstRoomOnScreen pins NewGame's whole set-up sequence by its one
// observable outcome: before a single frame is simulated, the player is looking at the
// house's first room.
//
// The budget is zero frames, so the game ends inside NewGame's DumpScreenOn -- this asserts
// on the state of the screen at the moment the game becomes playable and before
// HandleDynamics or the glider have touched anything.
func TestNewGamePutsTheFirstRoomOnScreen(t *testing.T) {
	w := playTestWorld(t, "Slumberland", 0)
	w.NewGame(NewGameMode)

	if got := litPixels(w.Main); got == 0 {
		t.Fatalf("Main is entirely black after NewGame; DumpScreenOn published nothing")
	}

	// The scoreboard band is the one part of Main the room path may not touch, and
	// NewGame's Play.c:141 black fill is what guarantees it. Rows 460..480 being black is
	// therefore not an accident of an empty surface -- index 0 is white -- it is that
	// statement having run.
	for y := int(w.R.V.Screen.Bottom - ScoreboardTall); y < int(w.R.V.Screen.Bottom); y++ {
		for x := 0; x < w.Main.W; x++ {
			if w.Main.Pix[y*w.Main.W+x] != render.Black8 {
				t.Fatalf("Main(%d,%d) is not black; the scoreboard band was not cleared", x, y)
			}
		}
	}

	if w.R.RoomNumber != w.H.FirstRoom {
		t.Errorf("room %d, want the house's first room %d", w.R.RoomNumber, w.H.FirstRoom)
	}
	// NewGame's teardown puts the mode back, so by the time it returns the world is a
	// shell again. PlayMode is only observable from inside the loop -- which is what the
	// mode is for: the menu code asks "are we in a game" and the answer has to be no here.
	if w.TheMode != SplashMode {
		t.Errorf("TheMode %d after NewGame returns, want SplashMode %d", w.TheMode, SplashMode)
	}
}

// TestNewGameSetsUpTheGliderAndTheGame pins InitGlider's split personality: the glider's
// own fields and the seven whole-game ones it also writes.
func TestNewGameSetsUpTheGliderAndTheGame(t *testing.T) {
	w := playTestWorld(t, "Slumberland", 0)

	// Dirt in every field InitGlider is responsible for, so that a field it fails to
	// write is a failure and not a coincidence.
	w.Score = 999
	w.Battery = 50
	w.Bands = 50
	w.Foil = 50
	w.ShowFoil = true
	w.P1.HVel = 7
	w.P1.VVel = 7
	w.P1.Tipped = true
	w.P1.Sliding = true
	w.P1.DontDraw = true
	w.P1.Facing = player.FaceLeft

	w.NewGame(NewGameMode)

	if w.Score != 0 {
		t.Errorf("Score %d, want 0", w.Score)
	}
	if w.Mortals != InitialGliders {
		t.Errorf("Mortals %d, want %d", w.Mortals, InitialGliders)
	}
	for _, c := range []struct {
		name string
		got  int16
	}{{"Battery", w.Battery}, {"Bands", w.Bands}, {"Foil", w.Foil}} {
		if c.got != 0 {
			t.Errorf("%s %d, want 0", c.name, c.got)
		}
	}
	if w.ShowFoil {
		t.Error("ShowFoil is set, want clear")
	}
	if w.P1.HVel != 0 || w.P1.VVel != 0 {
		t.Errorf("velocity (%d,%d), want (0,0)", w.P1.HVel, w.P1.VVel)
	}
	if w.P1.Tipped || w.P1.Sliding || w.P1.DontDraw {
		t.Errorf("tipped=%v sliding=%v dontDraw=%v, want all false",
			w.P1.Tipped, w.P1.Sliding, w.P1.DontDraw)
	}

	// The starting rect is the house's authored point with the glider's nominal size, and
	// the shadow takes only its horizontal edges: DestShadow's top is the fixed
	// ShadowTop whatever height the glider begins at (Play.c:358-360).
	want := player.Rect{Top: 0, Left: 0, Bottom: player.GliderHigh, Right: player.GliderWide}.
		Offset(w.H.Initial.H, w.H.Initial.V)
	if w.P1.Dest != want {
		t.Errorf("Dest %+v, want %+v", w.P1.Dest, want)
	}
	if w.P1.DestShadow.Top != player.ShadowTop || w.P1.DestShadow.Left != want.Left {
		t.Errorf("DestShadow %+v, want top %d and left %d",
			w.P1.DestShadow, player.ShadowTop, want.Left)
	}

	// The glider is mid-fade, not standing: NewGame's last act before PlayGame is
	// StartGliderFadingIn, and the zero-frame budget stops the game inside DumpScreenOn,
	// before any frame has run.
	if w.P1.Mode != player.GliderFadingIn {
		t.Errorf("Mode %d, want GliderFadingIn %d", w.P1.Mode, player.GliderFadingIn)
	}
	if w.P1.Facing != player.FaceRight {
		t.Error("Facing is left; a new game always starts facing right")
	}
}

// TestTwoPlayerInitGliderDoesNotDoubleTheLives is the trap in InitGlider written down as a
// test: NewGame calls it twice for a two-player game and every whole-game field it writes is
// therefore written twice.
//
// `mortals = kInitialGliders` followed by a conditional `+=` gives 4 both times. An
// idiomatic `mortals += kInitialGliders` would give 8, silently, and only in two-player
// games -- which is exactly the kind of thing that ships.
func TestTwoPlayerInitGliderDoesNotDoubleTheLives(t *testing.T) {
	w := playTestWorld(t, "Slumberland", 0)
	w.TwoPlayer = true
	w.NewGame(NewGameMode)

	if want := 2 * InitialGliders; w.Mortals != want {
		t.Errorf("Mortals %d for a two-player game, want %d", w.Mortals, want)
	}
	// Player two fades in and is immediately idled and hidden (Play.c:198-203), so a
	// two-player game opens with one glider visible.
	if w.P2.Mode != player.GliderIdle {
		t.Errorf("P2 Mode %d, want GliderIdle %d", w.P2.Mode, player.GliderIdle)
	}
	if !w.P2.DontDraw {
		t.Error("P2 should start with DontDraw set")
	}
	// The identities are set once at launch, not per game, so a game must not disturb
	// them -- an unset Which makes both gliders Player2 and hands the Command key to
	// nobody.
	if w.P1.Which != player.Player1 || w.P2.Which != player.Player2 {
		t.Errorf("identities P1=%v P2=%v, want %v/%v",
			w.P1.Which, w.P2.Which, player.Player1, player.Player2)
	}
}

// TestPlayGameAdvancesTheTwoClocks pins the loop head (Play.c:434-435).
//
// EvenFrame is not Frame&1 and this is where that becomes observable: with no dynamic
// object able to steal a toggle, the two must stay in lock step for the whole run, and the
// test states the relationship rather than the parity so that a later stage adding
// Dynamics2.c:420 has a failing test to read.
func TestPlayGameAdvancesTheTwoClocks(t *testing.T) {
	const budget = 90
	w := playTestWorld(t, "Slumberland", budget)

	// One sample per *frame*, not per present: a transition presents once per wipe strip
	// and would otherwise contribute a hundred samples with the same Frame. Slumberland's
	// first room happens to be static for far longer than this budget, so no transition
	// runs here -- but a test that would break the day one did is a test that will break
	// for the wrong reason.
	var frames []int64
	var evens []bool
	prev := w.Present
	w.Present = func() {
		if len(frames) == 0 || frames[len(frames)-1] != w.Frame {
			frames = append(frames, w.Frame)
			evens = append(evens, w.EvenFrame)
		}
		prev()
	}

	w.NewGame(NewGameMode)

	// The first present is DumpScreenOn, before the loop has run: Frame is still 0.
	if len(frames) < 2 {
		t.Fatalf("only %d frames presented; the loop did not run", len(frames))
	}
	if frames[0] != 0 {
		t.Errorf("first present at Frame %d, want 0 (NewGame's DumpScreenOn)", frames[0])
	}
	for i := 1; i < len(frames); i++ {
		if frames[i] != int64(i) {
			t.Fatalf("presented frame %d out of order: got Frame %d, want %d", i, frames[i], i)
		}
		// PlayGame's loop head toggles EvenFrame beside Frame, so with nothing else
		// writing it the flag is the frame's parity. It starts false at launch and is
		// toggled before the first frame is drawn, so frame 1 is even-flagged.
		if want := i%2 == 1; evens[i] != want {
			t.Errorf("Frame %d: EvenFrame %v, want %v", frames[i], evens[i], want)
		}
	}
}

// TestPlayGameKeepsPublishingFrames is the loop's liveness assertion: the frame limiter
// reseeds and the dirty-rect lists are cleared, so a run of any length keeps drawing.
//
// It is worth having as its own test because the two ways it can fail are both silent.
// A limiter that accumulated its deadline would stall on the first overrun -- see
// awaitFrame's note on there being no catch-up -- and a dirty-rect list that was never
// truncated would grow without bound and start dropping rects at the 47-entry cap.
func TestPlayGameKeepsPublishingFrames(t *testing.T) {
	const budget = 200
	w := playTestWorld(t, "Slumberland", budget)

	maxWork, maxBack := 0, 0
	prev := w.Present
	w.Present = func() {
		if n := len(w.Work2Main); n > maxWork {
			maxWork = n
		}
		if n := len(w.Back2Work); n > maxBack {
			maxBack = n
		}
		prev()
	}

	w.NewGame(NewGameMode)

	// The budget quits at the first present of frame `budget`, so the count is exact: a
	// lower number means the loop stalled or the game ended, and both are failures here --
	// Slumberland's first room has nothing in it that can kill an idle glider.
	if w.Frame != int64(budget) {
		t.Errorf("stopped at Frame %d, want %d (gameOver=%v playing=%v)",
			w.Frame, budget, w.GameOver, w.Playing)
	}
	// MaxGarbageRects is the cap the C silently drops past. A static room with one glider
	// nowhere approaches it, so a count anywhere near it means rects are accumulating.
	if maxWork >= MaxGarbageRects-1 {
		t.Errorf("Work2Main reached %d rects, at or past the %d cap", maxWork, MaxGarbageRects-1)
	}
	if maxBack >= MaxGarbageRects-1 {
		t.Errorf("Back2Work reached %d rects, at or past the %d cap", maxBack, MaxGarbageRects-1)
	}
}

// TestEveryHouseStartsAndRuns is the coverage assertion for stage 1.5b: every house that
// shipped with the game can be opened, composed, and stepped for a hundred frames without
// panicking.
//
// A hundred frames is chosen rather than one because the first frame exercises almost
// nothing: the glider is mid-fade for sixteen of them, the first room-lighting redraw is on
// frame two, and the phone's first ring is scheduled ninety frames out. It is not enough to
// find everything, and it is enough to find a nil surface, an out-of-range object code or a
// rect in the wrong coordinate system -- which are the three ways a house full of
// unfamiliar art breaks a renderer.
func TestEveryHouseStartsAndRuns(t *testing.T) {
	dir := requireAssets(t, "houses")
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range ents {
		if filepath.Ext(e.Name()) == ".house" {
			names = append(names, e.Name()[:len(e.Name())-len(".house")])
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		t.Skip("no extracted houses")
	}

	const budget = 100
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			w := playTestWorld(t, name, budget)

			// The Empty House is exactly what it says, and GetFirstRoomNumber's -1 path is
			// the one this exercises: no rooms, so no composition, and the loop still has
			// to run rather than crash.
			w.NewGame(NewGameMode)

			if len(w.H.Rooms) == 0 {
				if !w.NoRoomAtAll {
					t.Error("a house with no rooms should have set NoRoomAtAll")
				}
				return
			}
			// A short run is only a failure if the game did not end. An unattended glider
			// drifts in the direction it faces and several of these houses open somewhere
			// that kills it -- Titanic's engine room and Metropolis's rooftop both do --
			// so "ran out of gliders in under a hundred frames" is the house behaving,
			// not the port misbehaving.
			if w.Frame < int64(budget) && !w.GameOver {
				t.Errorf("only %d of %d frames ran and the game is not over (playing=%v)",
					w.Frame, budget, w.Playing)
			}
			if litPixels(w.Main) == 0 {
				t.Error("Main is entirely black; nothing composed")
			}
			if err := w.R.A.Err(); err != nil {
				t.Errorf("asset error: %v", err)
			}
		})
	}
}

// TestRenderFrameOrder pins the order of RenderFrame's calls, because that order is the
// z-order.
//
// This is the test grease_test.go and bands_test.go point at when they say they own a
// *consequence* of the sequence and not the position in it, and it is deliberately a
// reading of the source rather than a measurement of behaviour. Registration order is
// z-order (render_frame.go): each renderer appends to Work2Main and CopyRectsQD replays
// that list in order, so what has to be pinned is the sequence of thirteen calls in one
// function body. A behavioural equivalent would need a single room holding a mirror,
// grease, a pendulum, a flame, a dynamic, a flying point, a sparkle, a glider, a shred
// and a band at once, with all ten overlapping so the covering is visible, and it would
// then be pinning that room's art as much as the order. Parsing the function is exact,
// needs no assets, and fails with the two sequences side by side.
//
// The list is Render.c:639-671 transcribed. If this test fails the question is not "what
// is the new order" but "was Render.c wrong": reordering two lines here changes what the
// player sees and produces no other symptom, which is the whole reason the nine renderers
// went in as named no-ops in their real positions rather than being added as they landed.
func TestRenderFrameOrder(t *testing.T) {
	const src = "render_frame.go"

	f, err := parser.ParseFile(token.NewFileSet(), src, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", src, err)
	}
	decl, recv := methodBody(t, f, "RenderFrame")

	// The two-player calls are distinguished by their first argument, so that the test
	// also pins player 1 going down before player 2 -- which is z-order too, and is what
	// decides who is on top when two gliders overlap.
	want := []string{
		"DrawReflection(P1)", "DrawReflection(P2)",
		"HandleGrease",
		"RenderPendulums",
		"RenderFlames", "RenderStars",
		"RenderDynamics",
		"RenderFlyingPoints",
		"RenderSparkles",
		"RenderGlider(P1)", "RenderGlider(P2)",
		"RenderShreds",
		"RenderBands",
		"awaitFrame",
		"CopyRectsQD",
	}
	got := receiverCalls(decl.Body, recv)
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("RenderFrame's call order changed:\n  got  %s\n  want %s",
			strings.Join(got, " "), strings.Join(want, " "))
	}

	// Flames and stars are the one either/or in the sequence: they share a position and
	// alternate on frame parity, so a flat reading of the source cannot tell "both, in
	// this order" from "one or the other". Pin the branch as well, or the list above
	// would still pass with the two calls made unconditionally one after the other --
	// which is a room with its candles and its stars both animating at 30fps.
	var parity *ast.IfStmt
	ast.Inspect(decl.Body, func(n ast.Node) bool {
		if is, ok := n.(*ast.IfStmt); ok && hasCall(receiverCalls(is.Body, recv), "RenderFlames") {
			parity = is
		}
		return true
	})
	if parity == nil {
		t.Fatal("RenderFlames is no longer inside an if; the flame/star alternation is gone")
	}
	if cond, ok := parity.Cond.(*ast.SelectorExpr); !ok || !isReceiverField(cond, recv, "EvenFrame") {
		t.Error("the flame/star branch no longer tests EvenFrame; it must not be gameFrame&1 -- " +
			"a stalled ball or fish writes EvenFrame mid-frame (see World.EvenFrame)")
	}
	els, ok := parity.Else.(*ast.BlockStmt)
	if !ok || !hasCall(receiverCalls(els, recv), "RenderStars") {
		t.Error("RenderStars is no longer the else arm of the parity if; " +
			"flames and stars must never both draw in one frame")
	}
}

// TestPlayGameOrder pins the frame loop's call order and, for each call, which of the
// loop's two guards it is inside.
//
// This is item 6 of the fidelity contract (docs/ORIGINAL_GAME.md §19) and it is the item
// with the widest blast radius: every enemy in the game moves against *last* frame's glider
// position because HandleDynamics runs before the input is read, and every contested pickup
// in a two-player game goes to player 1 because player 1's input, hot-spot pass and
// interaction all happen first. Reorder two lines and every trajectory in every house
// changes, with no other symptom -- so, like TestRenderFrameOrder above, this reads the
// source rather than measuring behaviour, because a behavioural equivalent would be pinning
// one house's geometry rather than the sequence.
//
// The guard membership is asserted as well as the order, because the three ungated calls are
// ungated *on purpose* and it is the kind of thing a later stage tidies up: HandleDynamics
// before the player, and HandleTriggers/HandleBands outside `!gameOver` so that a fuse lit on
// the frame the last glider died still burns through the sixteen-frame countdown
// (Play.c:445-497).
func TestPlayGameOrder(t *testing.T) {
	const src = "play.go"

	f, err := parser.ParseFile(token.NewFileSet(), src, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", src, err)
	}
	decl, recv := methodBody(t, f, "PlayGame")

	// Play.c:445-497 and :499-546 transcribed, one body instead of the C's two arms. The
	// demo branch is player one's alone; the two P2 calls are the whole of the C's
	// two-player arm.
	want := []string{
		"pumpWhileSwitchedOut",
		"HandleTelephone",
		"HandleDynamics",
		"GetDemoInput(P1)",
		"GetInput(P1)", "GetInput(P2)",
		"HandleInteraction",
		"HandleTriggers", "HandleBands",
		"HandleGlider(P1)", "HandleGlider(P2)",
		"RenderFrame", "HandleDynamicScoreboard",
		"HideGlider(P1)", "RefreshScoreboard", "arcadeBlackenBoard",
		"DoDiedGameOver", "DoGameOver",
		"arcadeBlackenBoard",
	}
	if got := receiverCalls(decl.Body, recv); strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("PlayGame's call order changed:\n  got  %s\n  want %s",
			strings.Join(got, " "), strings.Join(want, " "))
	}

	// Which guards each call sits inside, innermost last. `!w.GameOver` and `w.Playing`
	// are the two that matter; `w.TwoPlayer` is here because the P2 calls must stay
	// under it and the P1 calls must not.
	guards := receiverCallGuards(decl.Body, recv)
	for _, c := range []struct {
		call  string
		under []string
		why   string
	}{
		{"HandleTelephone", nil,
			"the phone rings through the death countdown"},
		{"HandleDynamics", nil,
			"enemies move before the input is read, and keep moving after the last death"},
		{"GetDemoInput(P1)", []string{"!w.GameOver", "!w.TwoPlayer && w.DemoGoing"},
			"the attract mode drives player one only, and only in a one-player game"},
		{"GetInput(P1)", []string{"!w.GameOver", "!(!w.TwoPlayer && w.DemoGoing)"},
			"a dead player's keys are not read, and a demo's keys come from the resource"},
		{"HandleInteraction", []string{"!w.GameOver"},
			"one call serves both players, inside the same guard as the input"},
		{"HandleTriggers", nil,
			"a lit fuse burns through the countdown (Play.c:481-482 is outside the guard)"},
		{"HandleBands", nil,
			"a band in flight still lands after the last glider dies"},
		{"HandleGlider(P1)", []string{"!w.GameOver"},
			"the only caller of MoveGlider, gated with the input"},
		{"GetInput(P2)", []string{"!w.GameOver", "!(!w.TwoPlayer && w.DemoGoing)", "w.TwoPlayer"},
			"player two exists only in a two-player game"},
		{"HandleGlider(P2)", []string{"!w.GameOver", "w.TwoPlayer"},
			"as above"},
		{"RenderFrame", []string{"w.Playing"},
			"the second test of the loop condition: the last simulated frame is never drawn"},
		{"HandleDynamicScoreboard", []string{"w.Playing"},
			"drawn with the frame it belongs to"},
	} {
		got, ok := guards[c.call]
		if !ok {
			t.Errorf("%s is no longer called in PlayGame", c.call)
			continue
		}
		if strings.Join(got, " ") != strings.Join(c.under, " ") {
			t.Errorf("%s is guarded by [%s], want [%s] -- %s",
				c.call, strings.Join(got, ", "), strings.Join(c.under, ", "), c.why)
		}
	}
}

// receiverCallGuards maps each call on the receiver to the conditions it is nested under,
// outermost first, rendered as source. An `else` arm contributes `!(cond)`, because "inside
// the else of `if demo`" is a different guard from "inside `if demo`" and the whole point of
// the map is which calls a guard protects.
//
// The walk is written out rather than done with a plain ast.Inspect stack because Inspect
// cannot tell a body from an else: it hands out both as children of the same IfStmt. Only
// ifs contribute -- the loop is the frame itself, and the switches are all inside the callees.
//
// A call appearing twice keeps its *first* entry, which is the innermost-guarded one in
// source order (arcadeBlackenBoard is called in the countdown and again after the loop), so
// an assertion about it gets the tighter answer rather than an arbitrary one.
func receiverCallGuards(n ast.Node, recv string) map[string][]string {
	out := map[string][]string{}
	var walk func(ast.Node, []string)
	walk = func(nd ast.Node, conds []string) {
		if nd == nil {
			return
		}
		if is, ok := nd.(*ast.IfStmt); ok {
			cond := types.ExprString(is.Cond)
			walk(is.Init, conds)
			walk(is.Cond, conds)
			walk(is.Body, append(conds, cond))
			walk(is.Else, append(conds, "!("+cond+")"))
			return
		}
		if call, ok := nd.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok && isReceiver(sel.X, recv) {
				if names := receiverCalls(call, recv); len(names) > 0 {
					if _, seen := out[names[0]]; !seen {
						out[names[0]] = append([]string(nil), conds...)
					}
				}
			}
		}
		// Children only: Inspect visits nd first, and every deeper node is handed to
		// walk instead so that a nested if pushes its condition.
		ast.Inspect(nd, func(c ast.Node) bool {
			if c == nil || c == nd {
				return true
			}
			walk(c, conds)
			return false
		})
	}
	walk(n, nil)
	return out
}

// methodBody finds a method by name and returns it with its receiver's identifier, which
// the two helpers below need to tell `w.RenderBands()` from any other call.
func methodBody(t *testing.T, f *ast.File, name string) (*ast.FuncDecl, string) {
	t.Helper()
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Recv == nil || fd.Name.Name != name || fd.Body == nil {
			continue
		}
		if len(fd.Recv.List) == 0 || len(fd.Recv.List[0].Names) == 0 {
			t.Fatalf("%s has an unnamed receiver", name)
		}
		return fd, fd.Recv.List[0].Names[0].Name
	}
	t.Fatalf("no method named %s", name)
	return nil, ""
}

// receiverCalls lists the calls made on the receiver, in source order.
//
// ast.Inspect walks depth-first and pre-order, which for a straight-line body is source
// order, and for an if is cond, then body, then else -- so the flame/star pair comes out
// in the order the two arms are written. Nothing else in RenderFrame is a call on the
// receiver: the frame counter is an increment and the two list resets are slice
// truncations, so neither appears here.
//
// A call whose first argument is &recv.P1 or &recv.P2 gets that noted, because the two
// per-player calls are otherwise indistinguishable.
func receiverCalls(n ast.Node, recv string) []string {
	var out []string
	ast.Inspect(n, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !isReceiver(sel.X, recv) {
			return true
		}
		name := sel.Sel.Name
		if len(call.Args) > 0 {
			if u, ok := call.Args[0].(*ast.UnaryExpr); ok && u.Op == token.AND {
				if f, ok := u.X.(*ast.SelectorExpr); ok && isReceiver(f.X, recv) {
					name += "(" + f.Sel.Name + ")"
				}
			}
		}
		out = append(out, name)
		return true
	})
	return out
}

func isReceiver(e ast.Expr, recv string) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == recv
}

func isReceiverField(sel *ast.SelectorExpr, recv, field string) bool {
	return isReceiver(sel.X, recv) && sel.Sel.Name == field
}

func hasCall(calls []string, name string) bool {
	for _, c := range calls {
		if c == name || strings.HasPrefix(c, name+"(") {
			return true
		}
	}
	return false
}
