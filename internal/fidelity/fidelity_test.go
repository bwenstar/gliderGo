package fidelity_test

// The corpus, and the tests that keep it honest.
//
// Regenerate with:
//
//	go test ./internal/fidelity -update
//
// and read the diff before committing it. Every changed line is a pixel that moved, so the
// diff is either the change you meant to make -- in which case commit it and say so in the
// message -- or it is a bug you have just been handed for free.
//
// The references and the assets are both checked in, so a fresh clone can check its own pixels
// with no extraction step. This package reads the asset *tree* rather than the copy compiled into
// the executables, and deliberately: it names files by path, `-update` writes references beside
// them, and the thing being pinned is what the extractors produce. The tests still skip when the
// tree is absent -- `make clean-assets` and an interrupted extraction both leave one -- for
// internal/replay's reason, that a checkout must be able to run its own suite; `make fidelity` is
// the step that refuses to skip on a machine where the assets *are* present.

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/fidelity"
	"github.com/bwenstar/gliderGo/internal/render"
	"github.com/bwenstar/gliderGo/internal/replay"
)

var update = flag.Bool("update", false, "rewrite the references in testdata/")

const assetRoot = "../../assets/extracted"

// referenceHouse is the house the screens corpus is recorded against. The shipped demo
// house, because it is the one every copy of the original came with and the one the port's
// default selection lands on.
const referenceHouse = "CD Demo House"

// requireAssets skips unless the extracted tree has the subdirectories named.
func requireAssets(t *testing.T, subs ...string) {
	t.Helper()
	for _, sub := range subs {
		dir := filepath.Join(assetRoot, sub)
		if _, err := os.Stat(dir); err != nil {
			t.Skipf("no extracted assets at %s: run `make assets`", dir)
		}
	}
}

// frameScript is the script the frame corpus is recorded from.
//
// It is internal/replay's own 600-frame determinism script, read from that package's testdata
// rather than copied here. Sharing it is deliberate: the trace and the picture are two views
// of one run, so when a change shows up in both, the two failures name the same frame and can
// be read together. A copy would drift, and a corpus recorded from a drifted copy of a script
// is a corpus of nothing in particular.
//
// Sound is left as the script has it -- on -- because switching it off changes the
// *composition*: a trigger whose sample cannot be loaded gets no hot spot (replay.Script.Sound).
// A pixel reference has to be recorded with the audio settings it will be compared under.
func frameScript(t *testing.T) *replay.Script {
	t.Helper()
	requireAssets(t, "art", "houses", "sound")

	f, err := os.Open(filepath.Join("..", "replay", "testdata", "duct.script"))
	if err != nil {
		t.Fatalf("open script: %v", err)
	}
	defer f.Close()
	s, err := replay.Parse(f)
	if err != nil {
		t.Fatalf("parse script: %v", err)
	}
	s.ArtDir = filepath.Join(assetRoot, "art")
	s.HouseDir = filepath.Join(assetRoot, "houses")
	s.HouseArtDir = filepath.Join(assetRoot, "houseart")
	s.SoundDir = filepath.Join(assetRoot, "sound")
	return s
}

func screensOpts() fidelity.ScreensOpts {
	return fidelity.ScreensOpts{
		ArtDir:   filepath.Join(assetRoot, "art"),
		HouseDir: filepath.Join(assetRoot, "houses"),
		House:    referenceHouse,
	}
}

// check compares a fresh recording against the checked-in reference, or writes it under
// -update. The shape is the same for both corpora, so it lives here once.
func check(t *testing.T, name string, got *fidelity.Reference) *fidelity.Diff {
	t.Helper()
	path := filepath.Join("testdata", name)

	if *update {
		if err := got.WriteFile(path); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		t.Logf("wrote %s (%d rows)", path, len(got.Rows))
		return nil
	}

	want, err := fidelity.Load(path)
	if err != nil {
		t.Fatalf("%v (run `go test ./internal/fidelity -update`)", err)
	}
	return want.Compare(got)
}

// TestFrames is the frame corpus: 600 frames of CD Demo House, hashed three planes deep.
func TestFrames(t *testing.T) {
	s := frameScript(t)
	got, err := fidelity.RecordFrames(s)
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	// One row per recorded frame, no more and no less -- Present fires up to 161 times on a
	// transition frame and the corpus must still have exactly one line for it. The count is
	// frames+1 for internal/replay's reason: frame 0 is the picture the first render left
	// before the loop ran, so a 600-frame script samples 0 through 600, and its trace has 601
	// lines too.
	if n, want := len(got.Rows), int(s.Frames)+1; n != want {
		t.Errorf("recorded %d rows for a %d-frame script, want %d", n, s.Frames, want)
	}

	d := check(t, "duct.frames", got)
	if d == nil {
		return
	}

	// The failure, and then the evidence. A hash tells you frame 217 changed and nothing
	// else, so the frame is re-run and written out as three PNGs -- which is the whole
	// reason Snapshot exists and the reason this is a package and not a golden-file
	// one-liner.
	msg := d.String()
	if d.First != nil && len(d.Header) == 0 {
		var frame int64
		if _, err := fmt.Sscanf(d.First.Name, "%d", &frame); err == nil {
			dir := filepath.Join(os.TempDir(), "glidergo-fidelity")
			if paths, err := shot(s, frame, dir); err == nil {
				msg += "\nthe frame as this build draws it:\n  " + strings.Join(paths, "\n  ")
			} else {
				msg += fmt.Sprintf("\n(could not write the frame out: %v)", err)
			}
		}
	}
	t.Errorf("the picture changed at 600 frames of %s:\n%s\n\n"+
		"If the change was intended, `go test ./internal/fidelity -update` and say so in the "+
		"commit message. If it was not, the frame above is the first one that moved.",
		s.House, msg)
}

// shot writes one frame's three planes into dir.
func shot(s *replay.Script, frame int64, dir string) ([]string, error) {
	sh, err := fidelity.Snapshot(s, frame)
	if err != nil {
		return nil, err
	}
	return sh.WritePNGs(dir, "got")
}

// TestScreens is the shell corpus: the six screens a player meets before a game starts.
func TestScreens(t *testing.T) {
	requireAssets(t, "art", "houses")
	got, err := fidelity.RecordScreens(screensOpts())
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if len(got.Rows) != len(fidelity.DefaultScreens) {
		t.Fatalf("recorded %d screens, want %d", len(got.Rows), len(fidelity.DefaultScreens))
	}

	if d := check(t, "screens.hashes", got); d != nil {
		t.Errorf("a shell screen changed:\n%s\n\n"+
			"Look at it with `glidergo -shot /tmp/screen.png -shot-screen <name>`. If the "+
			"change was intended, `go test ./internal/fidelity -update`.", d)
	}
}

// TestFramesAreStableWithinARun is determinism the corpus cannot state: two recordings, one
// process, same answer. It is the cheap half of the acceptance criterion and it runs with no
// reference file at all, so it is what catches a divergence introduced *and* blessed in the
// same commit.
func TestFramesAreStableWithinARun(t *testing.T) {
	s := frameScript(t)
	// Short, because this is checking that the recorder is a function of the script and not
	// that the game is right. 60 frames covers the duct firing and the room change.
	s.Frames = 60

	first, err := fidelity.RecordFrames(s)
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	second, err := fidelity.RecordFrames(s)
	if err != nil {
		t.Fatalf("record again: %v", err)
	}
	if d := first.Compare(second); d != nil {
		t.Fatalf("two recordings of one script disagree:\n%s", d)
	}
}

// TestSnapshotIsTheFrameTheCorpusHashed pins the two halves of the failure path together: the
// PNG a failing test writes has to be the frame whose hash failed, or the evidence sends the
// reader to the wrong frame.
func TestSnapshotIsTheFrameTheCorpusHashed(t *testing.T) {
	s := frameScript(t)
	s.Frames = 30

	ref, err := fidelity.RecordFrames(s)
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	const frame = 12
	sh, err := fidelity.Snapshot(s, frame)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	want := ref.Rows[frame]
	got := []string{fidelity.Hash(sh.Main), fidelity.Hash(sh.Work), fidelity.Hash(sh.Back)}
	if want.Name != fmt.Sprint(frame) {
		t.Fatalf("row %d is named %q: the rows are not frame-ordered", frame, want.Name)
	}
	if strings.Join(got, " ") != strings.Join(want.Hashes, " ") {
		t.Errorf("the snapshot of frame %d hashes %v, the corpus says %v", frame, got, want.Hashes)
	}

	// And it writes files a human can open.
	paths, err := sh.WritePNGs(t.TempDir(), "test")
	if err != nil {
		t.Fatalf("write PNGs: %v", err)
	}
	if len(paths) != 3 {
		t.Fatalf("wrote %d PNGs, want 3", len(paths))
	}
	for _, p := range paths {
		st, err := os.Stat(p)
		if err != nil || st.Size() == 0 {
			t.Errorf("%s: %v (size %v)", p, err, st)
		}
	}

	if _, err := fidelity.Snapshot(s, 999); err == nil {
		t.Error("a snapshot of a frame the run never reached succeeded")
	}
}

// ---------------------------------------------------------------------------
// The file format, which needs no assets
// ---------------------------------------------------------------------------

func sample() *fidelity.Reference {
	return &fidelity.Reference{
		Kind:    "frames",
		Head:    []fidelity.KV{{"house", "CD Demo House"}, {"seed", "1"}, {"deviation", "one"}},
		Columns: []string{"main", "work", "back"},
		Rows: []fidelity.Row{
			{Name: "0", Hashes: []string{"aaaa", "bbbb", "cccc"}},
			{Name: "1", Hashes: []string{"dddd", "eeee", "cccc"}},
		},
	}
}

func TestRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	if err := sample().Write(&buf); err != nil {
		t.Fatalf("write: %v", err)
	}
	first := buf.String() // kept, because Parse drains the buffer
	back, err := fidelity.Parse(&buf)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d := sample().Compare(back); d != nil {
		t.Errorf("a reference did not survive a round trip:\n%s", d)
	}
	// And writing what was parsed reproduces the bytes, which is what makes -update a
	// no-op on an unchanged corpus rather than a churning diff.
	var again bytes.Buffer
	if err := back.Write(&again); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	if again.String() != first {
		t.Errorf("write(parse(write(r))) differs:\n--- first\n%s\n--- second\n%s", first, &again)
	}
}

func TestCompareFindsTheFirstFrame(t *testing.T) {
	want := sample()
	got := sample()
	got.Rows[1].Hashes[1] = "ffff"

	d := want.Compare(got)
	if d == nil {
		t.Fatal("a changed hash compared equal")
	}
	if d.First == nil || d.First.Name != "1" || d.First.Column != "work" {
		t.Fatalf("first difference reported as %+v, want frame 1 column work", d.First)
	}
	if d.Rows != 1 || d.Total != 2 {
		t.Errorf("%d of %d rows differ, want 1 of 2", d.Rows, d.Total)
	}
	if len(d.Header) != 0 {
		t.Errorf("a row change reported header differences: %v", d.Header)
	}
	if s := d.String(); !strings.Contains(s, "work want eeee, got ffff") {
		t.Errorf("the message does not name the change:\n%s", s)
	}
}

func TestCompareReportsTheHeaderFirst(t *testing.T) {
	// A header change explains every row under it, so it has to be said before the rows are
	// -- a reader who does not know the seed changed will go looking for a physics bug.
	want := sample()
	got := sample()
	got.Head[1].Value = "2"
	got.Rows[0].Hashes[0] = "9999"

	d := want.Compare(got)
	if d == nil {
		t.Fatal("a changed seed compared equal")
	}
	if len(d.Header) != 1 || !strings.Contains(d.Header[0], "seed") {
		t.Fatalf("header differences are %v, want one about the seed", d.Header)
	}
	if lines := strings.SplitN(d.String(), "\n", 2); !strings.HasPrefix(lines[0], "header ") {
		t.Errorf("the message leads with %q, want the header line", lines[0])
	}
}

func TestCompareReportsMissingAndExtraRows(t *testing.T) {
	want := sample()
	short := sample()
	short.Rows = short.Rows[:1]
	d := want.Compare(short)
	if d == nil || len(d.Missing) != 1 || d.Missing[0] != "1" {
		t.Fatalf("a run that stopped early: %+v", d)
	}

	long := sample()
	long.Rows = append(long.Rows, fidelity.Row{Name: "2", Hashes: []string{"1", "2", "3"}})
	d = want.Compare(long)
	if d == nil || len(d.Extra) != 1 || d.Extra[0] != "2" {
		t.Fatalf("a run that went on longer: %+v", d)
	}
}

func TestCompareIgnoresHeaderOrderButNotContent(t *testing.T) {
	// Ordered in the file so that a regeneration is not a reshuffle; compared as a set so
	// that adding a key in the middle of the header does not report the two lines after it
	// as changed as well.
	want := sample()
	got := sample()
	got.Head = []fidelity.KV{got.Head[2], got.Head[0], got.Head[1]}
	if d := want.Compare(got); d != nil {
		t.Errorf("a reordered header compared unequal:\n%s", d)
	}

	got.Head = append(got.Head, fidelity.KV{Key: "deviation", Value: "two"})
	d := want.Compare(got)
	if d == nil || len(d.Header) != 1 || !strings.Contains(d.Header[0], "deviation") {
		t.Fatalf("a second deviation was not reported: %+v", d)
	}
}

func TestParseRejectsSomethingElse(t *testing.T) {
	if _, err := fidelity.Parse(strings.NewReader("# just a comment\nhouse x\n")); err == nil {
		t.Error("a file with no columns line parsed as a reference")
	}
}

func TestHashSeparatesSizeFromPixels(t *testing.T) {
	// Two surfaces with the same bytes and different shapes must not collide: a view that
	// came back transposed would otherwise pass.
	a := newSurface(t, 4, 2)
	b := newSurface(t, 2, 4)
	if fidelity.Hash(a) == fidelity.Hash(b) {
		t.Error("4x2 and 2x4 of identical pixels hash the same")
	}
	if fidelity.Hash(nil) != "-" {
		t.Error("a missing surface should hash to `-`, not to a hash of nothing")
	}
}

// newSurface is a surface with a distinguishable index in every pixel.
func newSurface(t *testing.T, w, h int) *render.Surface {
	t.Helper()
	s := render.NewSurface(w, h)
	for i := range s.Pix {
		s.Pix[i] = uint8(i)
	}
	return s
}
