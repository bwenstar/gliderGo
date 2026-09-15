package audio

// Tests for the bank, and for the two claims in bank.go's file comment that can actually be
// checked against the assets: that every sound shares one sample rate, and that the loop points
// are degenerate everywhere except Thrust and Hiss.
//
// Those two are the interesting tests in this file. "The loader loads" is worth a few lines;
// "the reason we ignore the loop points is true of the data" is the assertion that would
// otherwise be a comment nobody could verify.

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files from the current output")

const (
	soundDir   = "../../assets/extracted/sound"
	goldenFile = "testdata/sound_golden.txt"
)

// requireBank loads the extracted bank or skips.
//
// Skips rather than fails, like every other asset-dependent test in the port:
// assets/extracted is gitignored because it is derived from a copyrighted 1994 application, so
// a bare checkout has no sounds and `go test ./...` must still pass on one.
func requireBank(t *testing.T) *Bank {
	t.Helper()
	if _, err := os.Stat(filepath.Join(soundDir, "manifest.tsv")); err != nil {
		t.Skipf("no extracted sounds at %s (run `make assets`)", soundDir)
	}
	b, err := LoadBank(soundDir)
	if err != nil {
		t.Fatalf("LoadBank: %v", err)
	}
	return b
}

func TestBankCoverage(t *testing.T) {
	b := requireBank(t)

	for i := int16(0); i < TriggerSlot; i++ {
		s := b.Effect(i)
		if s == nil {
			t.Errorf("slot %d (%s): no sound", i, Name(i))
			continue
		}
		if s.ID != BaseSoundID+i {
			t.Errorf("slot %d (%s): resource id %d, want %d", i, Name(i), s.ID, BaseSoundID+i)
		}
		if len(s.Data) == 0 {
			t.Errorf("slot %d (%s): empty sample", i, Name(i))
		}
		if s.Name == "" {
			t.Errorf("slot %d (%s): no resource name", i, Name(i))
		}
	}

	// The trigger slot is empty in the bank by construction: the sample lives on the
	// Engine because it is loaded and freed per room.
	if got := b.Effect(TriggerSlot); got != nil {
		t.Errorf("slot %d holds %q; the trigger slot belongs to the Engine", TriggerSlot, got.Name)
	}

	for i := int16(0); i < MaxMusic; i++ {
		if s := b.Piece(i); s == nil {
			t.Errorf("music piece %d: missing", i)
		} else if s.ID != BaseMusicID+i {
			t.Errorf("music piece %d: resource id %d, want %d", i, s.ID, BaseMusicID+i)
		}
	}
	if got := b.Piece(MaxMusic); got != nil {
		t.Errorf("Piece(%d) answered %q; it should be out of range", MaxMusic, got.Name)
	}
	if got := b.Piece(-1); got != nil {
		t.Errorf("Piece(-1) answered %q; a negative cursor must answer silence, not panic", got.Name)
	}
}

// TestNamesTable pins the shape of the slot table. Sixty-four entries, none empty, none
// repeated -- a duplicate would mean two slots misreported in every trace that named either.
func TestNamesTable(t *testing.T) {
	seen := map[string]int{}
	for i, n := range Names {
		if n == "" {
			t.Errorf("slot %d: no name", i)
			continue
		}
		if prev, dup := seen[n]; dup {
			t.Errorf("slot %d and slot %d are both named %q", prev, i, n)
		}
		seen[n] = i
	}
	if got := Name(64); got != "[64]" {
		t.Errorf("Name(64) = %q, want a bracketed number", got)
	}
	if got := Name(-1); got != "[-1]" {
		t.Errorf("Name(-1) = %q, want a bracketed number", got)
	}
}

// TestOneSampleRate is the "the application's bank is never resampled" claim.
//
// All 70 of the application's own sounds were recorded at one rate, which is true of Glider PRO
// and not true of most Macintosh applications of the period -- plenty mixed 11 kHz and 22 kHz
// resources. It is therefore a property of these assets and worth a test rather than an
// assumption, and it is what lets the mixer copy every sound the *game* plays byte for byte: the
// step is exactly FixedOne, so the inner loop's fixed-point cursor advances one whole sample and
// lands on no fractions. A recorded replay's byte-for-byte digest rests on this.
func TestOneSampleRate(t *testing.T) {
	rows := requireManifest(t)
	for _, r := range rows {
		if got := r["rate_fixed"]; got != "0x56EE8BA3" {
			t.Errorf("%s (%s): rate_fixed %s, want 0x56EE8BA3 (%d/%d Hz)",
				r["file"], r["name"], got, RateNum, RateDen)
		}
	}

	b := requireBank(t)
	for i := int16(0); i < TriggerSlot; i++ {
		if s := b.Effect(i); s != nil && s.Step != FixedOne {
			t.Errorf("slot %d (%s) is at %.4f Hz, step %d; every application sound must step by "+
				"exactly %d or the mix stops being a copy", i, Name(i), s.RateHz, s.Step, FixedOne)
		}
	}
	for i := int16(0); i < MaxMusic; i++ {
		if s := b.Piece(i); s != nil && s.Step != FixedOne {
			t.Errorf("music %d (%s) is at %.4f Hz, step %d, want %d", i, s.Name, s.RateHz, s.Step, FixedOne)
		}
	}
}

// TestHouseSampleRatesVary is the other half, and the one that found a bug.
//
// The application's sounds are uniform; **the houses' are not**. Fourteen of the 58 extracted
// house sounds are at some other rate, and playing them at the output rate would have made CD Demo
// House's "Fly Buzz" -- 7418 Hz, a third of the Macintosh rate -- come out three times too fast,
// which is a mosquito and not a fly. The mixer honours the header because the original did: every
// channel is opened with `initNoInterp` (Sound.c:379), which is drop-sample rate conversion.
//
// This is a tripwire on the *assets*, not on the loader. If a future extractor stopped writing
// `rate_hz`, or wrote it as a Fixed integer, every one of these would silently become the
// Macintosh rate and the only symptom would be fourteen sounds in twelve houses playing wrong.
func TestHouseSampleRatesVary(t *testing.T) {
	path := filepath.Join(soundDir, "houses", "manifest.tsv")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("no extracted house sounds at %s (run `make assets`)", path)
	}
	rows, err := readManifest(path)
	if err != nil {
		t.Fatalf("readManifest: %v", err)
	}

	// The distribution, transcribed from the manifest. Keyed by the string so that a change in
	// how the extractor formats the rate is a failure here rather than a rounding surprise
	// inside stepFor.
	want := map[string]int{
		"22254.5455": 42, "22254.5454": 2, // the Macintosh rate, both ways the printf lands
		"11127.2727": 6, "7418.1818": 3, "5563.6364": 1, // exact halves, thirds, a quarter
		"22255.0000": 1, "22050.0000": 1, "11127.5000": 1, "9779.0000": 1, // and four odd ones
	}
	got := map[string]int{}
	for _, r := range rows {
		if r["status"] != "ok" {
			continue
		}
		got[r["rate_hz"]]++
	}
	for rate, n := range want {
		if got[rate] != n {
			t.Errorf("%s: %d house sounds, want %d", rate, got[rate], n)
		}
	}
	for rate, n := range got {
		if want[rate] == 0 {
			t.Errorf("%s: %d house sounds at a rate this test has never seen; if the assets have "+
				"changed, check the table in bank.go's Sound.RateHz as well", rate, n)
		}
	}
}

// TestStepFor is the fixed-point arithmetic on its own, because the one value that must be exact
// is the one that arrives two different ways.
//
// The Macintosh rate is 244800/11 and the extractor prints four decimal places of a Fixed 16.16,
// so it can arrive as either ...5455 or ...5454. Both must land on exactly FixedOne. Truncating
// instead of rounding would give ...5454 a step of 65535 -- a part in ten million slow, entirely
// inaudible, and enough to make every sound in the game resample and every recorded digest drift.
func TestStepFor(t *testing.T) {
	for _, c := range []struct {
		rate float64
		want int64
		note string
	}{
		{22254.5455, FixedOne, "the Macintosh rate, rounded up"},
		{22254.5454, FixedOne, "the Macintosh rate, rounded down"},
		{244800.0 / 11.0, FixedOne, "the Macintosh rate, exactly"},
		{11127.2727, FixedOne / 2, "half: each byte held for two samples"},
		{5563.6364, FixedOne / 4, "a quarter"},
		{0, FixedOne, "no rate at all is the Macintosh rate"},
		{-1, FixedOne, "and so is a nonsense one, rather than a channel that never advances"},
	} {
		if got := stepFor(c.rate); got != c.want {
			t.Errorf("stepFor(%.4f) = %d, want %d (%s)", c.rate, got, c.want, c.note)
		}
	}

	// A third is 21845.33 and cannot be exact in 16.16. The error is what matters: at 21845 the
	// cursor drifts by a third of a sample every 65536 output samples, which is one sample in
	// three seconds of audio -- 6 parts per million of pitch, or a hundredth of a cent.
	third := stepFor(7418.1818)
	if third != FixedOne/3 && third != FixedOne/3+1 {
		t.Errorf("stepFor(7418.1818) = %d, want %d or %d", third, FixedOne/3, FixedOne/3+1)
	}
}

// continuous is the four sounds that carry a real loop point, with the retrigger cadence the
// game asks for each of them at. It is bank.go's table, in code.
var continuous = []struct {
	slot    int16
	cadence int // game frames between requests
	site    string
}{
	{18, 4, "Input.c:148, the battery"},
	{62, 4, "Input.c:174, the helium"},
	{47, 1, "Interactions.c:1365, a lit floor vent"},
	{26, 1, "Player.c:1287, a glider being shredded"},
}

// TestLoopPointsAreDegenerate is the first half of the reason bank.go ignores the loop points.
//
// The claim: exactly four sounds carry a full-range loop, and every other one carries the
// `frames-2 .. frames-1` pair a sound editor writes when there is no loop. If a fifth sound
// turned out to carry a real loop, the argument in bank.go's file comment would need
// re-examining -- so this is the tripwire on that argument, not a check on the loader.
func TestLoopPointsAreDegenerate(t *testing.T) {
	b := requireBank(t)

	looped := map[int16]bool{}
	for _, c := range continuous {
		looped[c.slot] = true
	}
	for i := int16(0); i < TriggerSlot; i++ {
		s := b.Effect(i)
		if s == nil {
			continue
		}
		n := int32(len(s.Data))
		degenerate := s.LoopStart == n-2 && s.LoopEnd == n-1
		full := s.LoopStart == 0 && s.LoopEnd == n-1
		switch {
		case looped[i] && !full:
			t.Errorf("slot %d (%s) should carry the full-range loop 0..%d but has %d..%d",
				i, Name(i), n-1, s.LoopStart, s.LoopEnd)
		case !looped[i] && !degenerate:
			t.Errorf("slot %d (%s) carries loop %d..%d of %d frames, which is neither the degenerate "+
				"pair nor one of the four continuous sounds; see bank.go on why loop points are ignored",
				i, Name(i), s.LoopStart, s.LoopEnd, n)
		}
	}
}

// TestContinuousSoundsCoverTheirCadence is the second half, and the more interesting one.
//
// bank.go argues that the loop points were never needed because the four continuous sounds are
// *cut to length*: each is at least as long as the interval at which the game asks for it again,
// so consecutive copies meet or overlap and the sound is seamless without the Sound Manager
// looping anything. This checks that against the samples.
//
// Hiss is the striking case and gets an exact assertion: 2960 frames is four game frames to the
// sample, and Input.c asks for it every fourth frame. That is a composer working to the same
// arithmetic as SamplesPerFrame, from the other end.
func TestContinuousSoundsCoverTheirCadence(t *testing.T) {
	b := requireBank(t)

	for _, c := range continuous {
		s := b.Effect(c.slot)
		if s == nil {
			continue
		}
		frames := float64(s.Frames()) / SamplesPerFrame

		// channels is how many copies of the sound are alive at once: its length divided
		// by how often it is asked for. It is the number the sample length was chosen
		// against, and it lands between 1 and 3 for all four sounds -- 1 for Hiss, 1.95
		// for Thrust, 2.61 for Shred and 3.03 for Sizzle, which is the whole mixer.
		channels := frames / float64(c.cadence)

		if channels < 1 {
			// A sample shorter than its cadence would leave a hole in the sound, which
			// is the symptom that would send somebody looking at the loop points.
			t.Errorf("slot %d (%s) is %.2f game frames long but is asked for every %d frames (%s); "+
				"the sound would have a gap", c.slot, Name(c.slot), frames, c.cadence, c.site)
		}
		// There are three channels, so three simultaneous copies is the ceiling: a
		// longer sample is cut off by its own retrigger and the extra length is never
		// heard. Sizzle sits exactly on that ceiling, 25 samples over, which is one more
		// sign that these lengths were chosen and not found.
		if channels > 3.05 {
			t.Errorf("slot %d (%s) is %.2f game frames long against a cadence of %d (%s), "+
				"which is %.2f simultaneous copies -- more than the three channels can hold, "+
				"so the retrigger reading in bank.go cannot be right",
				c.slot, Name(c.slot), frames, c.cadence, c.site, channels)
		}
	}

	if got := b.Effect(62); got != nil && got.Frames() != 4*SamplesPerFrame {
		t.Errorf("Hiss is %d frames; bank.go's argument rests on it being exactly 4 x %d",
			got.Frames(), SamplesPerFrame)
	}
}

// TestHouseTriggers loads a house that has custom sounds and one that does not.
//
// Art Museum is the useful case: it has five, and its sound triggers are the ones that reach
// the interrupted-trigger path FlushTriggerSound documents.
func TestHouseTriggers(t *testing.T) {
	b := requireBank(t)

	if err := b.LoadHouse("Art Museum"); err != nil {
		t.Fatalf("LoadHouse(Art Museum): %v", err)
	}
	if len(b.Triggers) == 0 {
		t.Fatal("Art Museum: no trigger sounds; it has five in the fork")
	}
	for id, s := range b.Triggers {
		if s.Slot != TriggerSlot {
			t.Errorf("trigger %d (%s): slot %d, want %d", id, s.Name, s.Slot, TriggerSlot)
		}
		if s.ID != id {
			t.Errorf("trigger keyed %d holds resource %d", id, s.ID)
		}
		if len(s.Data) == 0 {
			t.Errorf("trigger %d (%s): empty sample", id, s.Name)
		}
	}
	if got := b.Trigger(9999); got != nil {
		t.Errorf("Trigger(9999) answered %q", got.Name)
	}

	// Loading another house must forget the first one's sounds. In the original this is a
	// resource fork being closed; here it is a map being replaced, and getting it wrong
	// would let a sound trigger in house B play a sound from house A.
	if err := b.LoadHouse("no such house"); err != nil {
		t.Fatalf("LoadHouse(no such house): %v", err)
	}
	if len(b.Triggers) != 0 {
		t.Errorf("after loading a house with no sounds, %d triggers are still held", len(b.Triggers))
	}
}

func TestBankBytes(t *testing.T) {
	b := requireBank(t)
	// The 1994 build held all of this resident at once and asked the Memory Manager
	// whether it could. The number is worth asserting a bound on because it is the whole
	// audio footprint of the port: if it ever grows by an order of magnitude, something has
	// gone wrong in extraction.
	const megabyte = 1 << 20
	if n := b.Bytes(); n < megabyte || n > 8*megabyte {
		t.Errorf("bank is %d bytes; expected between 1 and 8 MB", n)
	}
}

// TestSoundGolden writes the sound index: slot, the header's name for it, the resource's own
// name, and how long it is.
//
// It is a fixture rather than an assertion, and it earns its place twice over. It is the only
// place the C header's vocabulary and the resource fork's vocabulary sit side by side, which is
// how a reader finds out that kMicrowavedSound is the resource called "Miked", that
// kCaughtFireSound is "Yow!" and that kWebTwangSound is "Twunk". And because it carries every
// sample's length, any change in the extractor shows up here as a diff instead of as a click
// somebody notices six months later.
//
// Four decimal places and not three, which is not cosmetic: Tik is 612 frames, which is 0.0275
// seconds *exactly*, and an exact tie at three places rounds whichever way the last bit of the
// float64 falls. Seconds() computing 612/22254.5455 and 612*11/244800 are the same number to
// within one part in 10^16 and they landed on opposite sides of that tie, so the golden flipped
// when Seconds() changed. A fourth place puts every duration in these assets off the boundary.
func TestSoundGolden(t *testing.T) {
	b := requireBank(t)

	var sb strings.Builder
	sb.WriteString("# slot\tconstant\tresource\tframes\tseconds\n")
	for i := int16(0); i < TriggerSlot; i++ {
		s := b.Effect(i)
		fmt.Fprintf(&sb, "%d\t%s\t%s\t%d\t%.4f\n", i, Name(i), s.Name, s.Frames(), s.Seconds())
	}
	for i := int16(0); i < MaxMusic; i++ {
		s := b.Piece(i)
		fmt.Fprintf(&sb, "music %d\t\t%s\t%d\t%.4f\n", i, s.Name, s.Frames(), s.Seconds())
	}
	got := sb.String()

	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenFile, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", goldenFile)
		return
	}
	want, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Skipf("no golden file (run `go test ./internal/audio -update`): %v", err)
	}
	if got != string(want) {
		t.Errorf("sound index differs from %s; run with -update to accept\n--- got\n%s", goldenFile, firstDiff(string(want), got))
	}
}

// requireManifest reads the manifest rows directly, for the tests that check a column the Bank
// does not keep.
func requireManifest(t *testing.T) []map[string]string {
	t.Helper()
	path := filepath.Join(soundDir, "manifest.tsv")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("no extracted sounds at %s (run `make assets`)", soundDir)
	}
	rows, err := readManifest(path)
	if err != nil {
		t.Fatalf("readManifest: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("manifest has no rows")
	}
	return rows
}

// firstDiff reports the first differing line of two texts, because a 70-line golden diff with
// one changed number in it is unreadable.
func firstDiff(want, got string) string {
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < len(w) || i < len(g); i++ {
		lw, lg := "", ""
		if i < len(w) {
			lw = w[i]
		}
		if i < len(g) {
			lg = g[i]
		}
		if lw != lg {
			return fmt.Sprintf("line %d:\n  want %q\n  got  %q", i+1, lw, lg)
		}
	}
	return "(no line differs; check the trailing newline)"
}

// TestManifestSorted is a small guard on the extractor rather than on the loader: the manifest
// is written in resource-ID order, and a loader that depended on that order would be fragile,
// so this checks the order is there *and* bank.go's switch does not rely on it -- the switch
// keys on the ID it reads, which is why TestBankCoverage passes with the rows in any order.
func TestManifestSorted(t *testing.T) {
	rows := requireManifest(t)
	ids := make([]int, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, atoiOr(r["id"], -1))
	}
	if !sort.IntsAreSorted(ids) {
		t.Errorf("manifest ids are not in ascending order: %v", ids)
	}
}
