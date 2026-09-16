package game

// The RNG's verification, which is the one thing rand.go could not claim for itself.
//
// `Random()` is a Toolbox trap: its code is in a 68k ROM or in InterfaceLib, and it is not in the
// vendored tree at any line number. rand.go is therefore a *reconstruction* -- Park-Miller with
// Schrage decomposition, returning the low word signed -- and the honest question about it is not
// "is the Go right" but "is the reconstruction right". Nothing in this repository can answer the
// second question; there is no Mac here to ask.
//
// What can be answered is everything downstream of it, and this file answers all of it:
//
//   - the stream from seed 1 matches the 24 draws tabulated in
//     docs/analysis/toolbox-primitives.md §1.6, state by state, so a port and a doc that were
//     written from the same paper cannot drift apart silently;
//   - Schrage agrees with the arithmetic it is standing in for, over every seed a test can afford,
//     and the int32 multiply-then-mod a literal translation would have written does not (§1.8);
//   - `RandomInt`'s inclusive upper bound and its skew are exactly the distribution §1.7
//     enumerates, checked over all 65536 raw words rather than sampled;
//   - and the player package -- the whole of the physics -- never draws at all, which is step 2 of
//     R-RNG-2's proof that the shipped demo replay is RNG-independent (§1.12).
//
// So a future divergence has somewhere to be. If a real Mac ever contradicts §1.6, exactly one
// table changes and every other test here keeps its meaning. See docs/IMPROVEMENTS.md 2.18.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// randM is the Lehmer modulus, 2^31-1, spelled out because three tests need it as a number
// rather than as `0x7FFFFFFF` masquerading as a bit pattern.
const randM = 2147483647

// seed1Stream is docs/analysis/toolbox-primitives.md §1.6, transcribed.
//
// Each row is one draw from `randSeed == 1`: the state afterwards, `Random()`'s return as an
// int16, and what `RandomInt` would have returned from that same draw at the three ranges the
// game actually asks for -- 2 (pendulum direction, DynamicMaps.c:594), 3 (telephone rings,
// Play.c:736) and 6 (star phase, DynamicMaps.c:685; flower species, InterfaceInit.c:160).
//
// The ranges are columns rather than separate tables because a `RandomInt` column is a *fold* of
// the same draw: the nth call to RandomInt from seed 1 consumes the nth state. Four separate
// worlds walk the four columns in step, which is also the cheapest possible check that RandomInt
// advances the stream exactly once per call.
var seed1Stream = []struct {
	state            int32
	random           int16
	int2, int3, int6 int16
}{
	{16807, 16807, 1, 1, 3},
	{282475249, 15089, 0, 1, 2},
	{1622650073, -21287, 1, 1, 3},
	{984943658, 3114, 0, 0, 0},
	{1144108930, -18558, 1, 1, 3},
	{470211272, -9528, 0, 0, 1},
	{101027544, -28968, 1, 2, 5},
	{1457850878, 2558, 0, 0, 0},
	{1458777923, 12099, 0, 1, 2},
	{2007237709, 1101, 0, 0, 0},
	{823564440, -26472, 1, 2, 4},
	{1115438165, 15445, 0, 1, 2},
	{1784484492, 4748, 0, 0, 0},
	{74243042, -9246, 0, 0, 1},
	{114807987, -11085, 0, 1, 2},
	{1137522503, 14151, 0, 1, 2},
	{1441282327, 14615, 0, 1, 2},
	{16531729, 16657, 1, 1, 3},
	{823378840, -15464, 0, 1, 2},
	{143542612, 18772, 1, 1, 3},
	{896544303, 11823, 0, 1, 2},
	{1474833169, 11025, 0, 1, 2},
	{1264817709, -27091, 1, 2, 4},
	{1998097157, -29947, 1, 2, 5},
}

func TestRandomMatchesTheVerifiedSeed1Stream(t *testing.T) {
	w := &World{RandSeed: 1}
	r2 := &World{RandSeed: 1}
	r3 := &World{RandSeed: 1}
	r6 := &World{RandSeed: 1}
	for i, want := range seed1Stream {
		draw := i + 1
		if got := w.Random(); got != want.random {
			t.Errorf("draw %d: Random() = %d, want %d (§1.6)", draw, got, want.random)
		}
		if w.RandSeed != want.state {
			t.Errorf("draw %d: state = %d, want %d (§1.6)", draw, w.RandSeed, want.state)
		}
		if got := r2.RandomInt(2); got != want.int2 {
			t.Errorf("draw %d: RandomInt(2) = %d, want %d", draw, got, want.int2)
		}
		if got := r3.RandomInt(3); got != want.int3 {
			t.Errorf("draw %d: RandomInt(3) = %d, want %d", draw, got, want.int3)
		}
		if got := r6.RandomInt(6); got != want.int6 {
			t.Errorf("draw %d: RandomInt(6) = %d, want %d", draw, got, want.int6)
		}
		// RandomInt must consume exactly one draw, or the three columns would walk off the
		// table at different rates and every row after the first would be wrong for a
		// reason no assertion above would name.
		if r2.RandSeed != want.state || r3.RandSeed != want.state || r6.RandSeed != want.state {
			t.Fatalf("draw %d: RandomInt left state %d/%d/%d, want %d -- one call is one draw",
				draw, r2.RandSeed, r3.RandSeed, r6.RandSeed, want.state)
		}
	}
}

// TestRandomAgreesWithTheWiderArithmetic is the check Schrage exists to pass.
//
// The recurrence is `state = 16807 * state mod 2^31-1`, and rand.go computes it in int32 without
// ever exceeding int32, which is the only reason the transcription can stay in the width the C
// used. This computes the same thing in int64 -- where it is one multiply and one remainder --
// and requires they agree for every seed in a dense low band, at the Schrage split itself, and
// at the top of the range.
func TestRandomAgreesWithTheWiderArithmetic(t *testing.T) {
	seeds := make([]int32, 0, 4200)
	for s := int32(1); s <= 4000; s++ {
		seeds = append(seeds, s)
	}
	// 127773 is the split (16807 * 127773 is 400k short of int32's ceiling), so its
	// neighbourhood is where an off-by-one in the decomposition would live.
	for _, s := range []int32{127771, 127772, 127773, 127774, 127775} {
		seeds = append(seeds, s)
	}
	// Large states, including the two the guard in Random rejects the neighbours of.
	for _, s := range []int32{1 << 20, 1 << 24, 1000000007, 2000000000, randM - 2, randM - 1} {
		seeds = append(seeds, s)
	}
	for _, seed := range seeds {
		want := int32(int64(16807) * int64(seed) % int64(randM))
		w := &World{RandSeed: seed}
		w.Random()
		if w.RandSeed != want {
			t.Fatalf("seed %d: Schrage gave state %d, int64 arithmetic gives %d",
				seed, w.RandSeed, want)
		}
	}
}

// TestNaiveInt32ModularWouldHaveDiverged pins §1.8's trap, which is the reason rand.go is written
// the way it is and not the obvious way.
//
// `state * 16807 % 2147483647` in int32 is what a literal translation of a 68k `long` would say,
// and it is right until the product overflows -- which happens above the Schrage split, 127773,
// and therefore within three draws of seed 1: draw 2 is 16807 squared, which just fits, and the
// multiply after that does not. This test is the only place the wrong implementation is written
// down, so that "why not the one-liner" has an answer that runs.
func TestNaiveInt32ModularWouldHaveDiverged(t *testing.T) {
	naive := func(s int32) int32 { return s * 16807 % randM } // deliberately wrong

	// Below the split the two agree, which is exactly what makes the bug hard to notice.
	for _, seed := range []int32{1, 2, 1000, 127772, 127773} {
		w := &World{RandSeed: seed}
		w.Random()
		if got := naive(seed); got != w.RandSeed {
			t.Fatalf("seed %d: expected the naive form to still agree, got %d want %d",
				seed, got, w.RandSeed)
		}
	}
	// 127774 is the first seed where it does not (§1.8).
	w := &World{RandSeed: 127774}
	w.Random()
	if got := naive(127774); got == w.RandSeed {
		t.Fatalf("seed 127774: the naive int32 form agreed (%d); §1.8 says this is the first "+
			"seed where it must not, so either the split moved or int32 stopped wrapping", got)
	}
	// And three draws from seed 1 is all it takes to leave the safe band for good: draw 2 is
	// 282475249 -- 16807 squared, which just fits -- and the multiply after that does not.
	s, ref := int32(1), &World{RandSeed: 1}
	for i := 0; i < 3; i++ {
		s = naive(s)
		ref.Random()
		if i < 2 && s != ref.RandSeed {
			t.Fatalf("draw %d: the naive form left the stream at %d, want %d -- it should "+
				"survive two draws", i+1, s, ref.RandSeed)
		}
	}
	if s == ref.RandSeed {
		t.Errorf("the naive form survived three draws from seed 1 (%d); §1.8 says the "+
			"sequence visits a state too large to multiply immediately", s)
	}
}

// predecessorOf returns the state whose successor is want, so that a test can arrange for a
// specific `Random()` return without searching for it.
//
// The recurrence is a bijection on [1, 2^31-2], so every state has exactly one predecessor:
// multiply by 16807's modular inverse. The inverse is computed here rather than written down,
// because a constant would be one more number to trust.
func predecessorOf(t *testing.T, want int32) int32 {
	t.Helper()
	// Fermat: 16807^(m-2) mod m, m prime.
	inv, base, exp := int64(1), int64(16807), int64(randM-2)
	for exp > 0 {
		if exp&1 == 1 {
			inv = inv * base % randM
		}
		base = base * base % randM
		exp >>= 1
	}
	if inv*16807%randM != 1 {
		t.Fatalf("16807 * %d mod %d = %d, want 1 -- the inverse is wrong",
			inv, randM, inv*16807%randM)
	}
	return int32(inv * int64(want) % randM)
}

// TestRandomIntUpperBoundIsInclusive is the original's off-by-one, reproduced on purpose.
//
// `rawResult = Random()` can be -32768; negating that as a *long* gives 32768, not the -32768 a
// short would give back; and 32768 * range / 32768 is range exactly. So RandomInt(n) returns n
// once per 65536 draws, one past the end of whatever array the caller is indexing. The port keeps
// it because removing it would shift every later draw, and clamps at the call sites that cannot
// take it.
func TestRandomIntUpperBoundIsInclusive(t *testing.T) {
	// The draw whose low word is 0x8000 -- i.e. Random() == -32768.
	pred := predecessorOf(t, 0x8000)

	w := &World{RandSeed: pred}
	if got := w.Random(); got != -32768 {
		t.Fatalf("arranged draw returned %d, want -32768", got)
	}
	for _, rng := range []int16{2, 3, 4, 5, 6, 8, 10, 16, 25000} {
		w := &World{RandSeed: pred}
		if got := w.RandomInt(rng); got != rng {
			t.Errorf("RandomInt(%d) on the -32768 draw = %d, want %d (the inclusive bound)",
				rng, got, rng)
		}
		// A modular reduction -- the reimplementation everyone reaches for -- would have
		// returned 0 here, and would have been wrong in the other 65535 cases too (§1.8's
		// sibling trap, §1.7's skew).
		if 32768%int32(rng) != 0 {
			continue
		}
		if got := int16(32768 % int32(rng)); got == rng {
			t.Errorf("the modular form also returned %d for range %d; the test proves nothing", got, rng)
		}
	}
}

// TestRandomIntSkewOverEveryRawWord pins §1.7's distribution table exactly rather than sampling
// it, because the interesting entries are a count of *one*.
//
// Every one of the 65536 possible 16-bit draws is visited by arranging its predecessor, so this
// is the whole domain of the fold and not a sample of the stream: 0 is always slightly
// under-represented, and `range` occurs exactly once.
func TestRandomIntSkewOverEveryRawWord(t *testing.T) {
	want := map[int16][]int{
		2: {32767, 32768, 1},
		3: {21845, 21846, 21844, 1},
		6: {10923, 10922, 10922, 10924, 10922, 10922, 1},
	}
	// Arranging 65536 predecessors costs one modular inverse each if done naively, so the
	// inverse is taken once: state(w) = w or 65536 for w == 0, and its predecessor follows.
	invOfOne := predecessorOf(t, 1) // == 16807^-1
	pred := func(state int32) int32 {
		return int32(int64(invOfOne) * int64(state) % randM)
	}
	for rng, counts := range want {
		got := make([]int, len(counts))
		for word := 0; word < 65536; word++ {
			state := int32(word)
			if state == 0 {
				// Low word 0 needs a state with more than 16 bits; 65536 is the
				// smallest, and 0 itself is not a legal state anyway.
				state = 65536
			}
			w := &World{RandSeed: pred(state)}
			v := w.RandomInt(rng)
			if v < 0 || int(v) >= len(got) {
				t.Fatalf("RandomInt(%d) returned %d, outside [0, %d]", rng, v, rng)
			}
			got[v]++
		}
		for v := range counts {
			if got[v] != counts[v] {
				t.Errorf("RandomInt(%d): value %d occurred %d times, want %d (§1.7)",
					rng, v, got[v], counts[v])
			}
		}
	}
}

// TestRandomIntNegativeRange records what nothing in the game does.
//
// A negative range makes the third statement divide a positive by 32768 and multiply by a
// negative, so the result is in [range, 0]. No call site passes one; the behaviour is pinned so
// that a future one gets an answer rather than a surprise.
func TestRandomIntNegativeRange(t *testing.T) {
	w := &World{RandSeed: 1}
	for i := 0; i < 200; i++ {
		if got := w.RandomInt(-6); got > 0 || got < -6 {
			t.Fatalf("RandomInt(-6) = %d, want [-6, 0]", got)
		}
	}
}

// TestRandomGuardsTheStatesTheRecurrenceCannotLeave documents the one place rand.go is not a
// transcription.
//
// 0 is a fixed point (0 * 16807 mod m is 0) and 2^31-1 maps to 0, so both would freeze the
// stream. The Toolbox cannot reach either -- a classic Mac's InitGraf sets randSeed to 1, and
// Random never produces 0 -- but a Go zero value can, so both are nudged to 1.
//
// The consequence is load-bearing for the harness: `seed 0` in a replay script is seed **1**,
// which is the seed the shipped Carbon build starts from every launch (Prefix.h sets
// TARGET_CARBON, so Utilities.c:61's GetDateTime is compiled out). Scripts that say nothing about
// the seed are therefore already at the historically correct one.
func TestRandomGuardsTheStatesTheRecurrenceCannotLeave(t *testing.T) {
	for _, seed := range []int32{0, 1, -1, -2147483648, randM} {
		w := &World{RandSeed: seed}
		if got := w.Random(); got != seed1Stream[0].random {
			t.Errorf("seed %d: first draw %d, want %d -- the guard should have made this "+
				"the seed-1 stream", seed, got, seed1Stream[0].random)
		}
	}
	// A guarded seed produces the whole seed-1 stream, not just its first draw.
	zero, one := &World{RandSeed: 0}, &World{RandSeed: 1}
	for i := range seed1Stream {
		if a, b := zero.Random(), one.Random(); a != b {
			t.Fatalf("draw %d: seed 0 gave %d, seed 1 gave %d", i+1, a, b)
		}
	}
}

// TestThePhysicsNeverDrawsFromTheRNG is step 2 of R-RNG-2's proof, kept true in Go.
//
// docs/analysis/toolbox-primitives.md §1.12 disproves the claim that the shipped demo replay
// depends on the random stream, and its second step is that `Player.c` contains no Random call at
// all -- so a glider's trajectory is a function of (initial state, input, geometry) and nothing
// else. internal/game/player is that file's port, and this walks its syntax trees for any
// identifier naming the generator.
//
// An AST walk rather than a grep because a comment mentioning RandomInt is fine and a field named
// RandomInt is not; and it is worth having as a test because the day somebody adds a jitter to a
// bounce, the demo replay stops being an oracle and this is the only thing that would say so.
func TestThePhysicsNeverDrawsFromTheRNG(t *testing.T) {
	dir := filepath.Join("player")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	fset := token.NewFileSet()
	files := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}
		files++
		ast.Inspect(f, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if ok && strings.Contains(id.Name, "Random") {
				t.Errorf("%s: identifier %q -- the physics must not draw from the RNG, "+
					"or the demo replay stops being a fidelity oracle "+
					"(toolbox-primitives.md §1.12 step 2)",
					fset.Position(id.Pos()), id.Name)
			}
			return true
		})
	}
	if files == 0 {
		t.Fatalf("no non-test Go files found in %s, so this test proved nothing", dir)
	}
}

// TestAdvanceRandSeedIsOneDraw pins the launch-time accounting cmd/glidergo does.
//
// One draw, not two and not zero: the original's VariableInit consumes exactly one
// (InterfaceInit.c:160), so a session's first game begins on §1.6's first state rather than on
// the seed itself. The test is here rather than in cmd/glidergo because what it is really
// asserting is that the helper does not have its own copy of the recurrence.
func TestAdvanceRandSeedIsOneDraw(t *testing.T) {
	if got := AdvanceRandSeed(1); got != seed1Stream[0].state {
		t.Errorf("AdvanceRandSeed(1) = %d, want %d -- the state after one draw from seed 1",
			got, seed1Stream[0].state)
	}
	// Walking it repeatedly must trace the same stream a World does, or the two ways of
	// advancing the seed would drift and a game handed the app's seed would start somewhere
	// the table does not describe.
	s, w := int32(1), &World{RandSeed: 1}
	for i, want := range seed1Stream {
		s = AdvanceRandSeed(s)
		w.Random()
		if s != w.RandSeed || s != want.state {
			t.Fatalf("draw %d: AdvanceRandSeed gave %d, World gave %d, §1.6 says %d",
				i+1, s, w.RandSeed, want.state)
		}
	}
	// The guarded seeds go through it too: `-seed 0` means seed 1, so a clock reading that
	// folded onto a dead state cannot make the first game start from nowhere.
	if got := AdvanceRandSeed(0); got != seed1Stream[0].state {
		t.Errorf("AdvanceRandSeed(0) = %d, want %d (the guard in Random)", got, seed1Stream[0].state)
	}
}
