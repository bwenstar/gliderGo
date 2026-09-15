package demo

// Tests for the codec, the cursor and the recorder.
//
// Two of them are worth more than the rest. TestShippedResource pins the one file in the world
// this format was designed around -- every number in docs/analysis/input.md §14, re-derived from
// the bytes -- so that a change to the extractor, the endianness or the stride is caught by a
// test that names the file rather than by a demo that flies into a wall. And TestCursorRepeated
// Frame demonstrates the hazard the whole package is arranged around: two records on one frame
// do not lose one action, they lose the entire rest of the stream.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const assetRoot = "../../assets/extracted"

// The shipped `'demo'` 128, as extracted from Glider PRO 1.0.5's application resource fork.
//
// The hash is here so that this test can say *which* file it is looking at when the counts
// below stop matching: "the resource changed" and "the decoder changed" are different bugs and
// only one of them is this package's.
const shippedSHA256 = "5b103d5b7df504515ce0df366652e932bc38193d983f56eba14ee59bff2a593c"

// requireShipped loads `'demo'` 128 out of the extracted tree, or skips.
//
// Skips rather than fails: assets/extracted is derived from a copyrighted 1994 application and
// is gitignored, so `go test ./...` on a bare checkout must pass without it.
func requireShipped(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join(assetRoot, ShippedPath)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("no extracted demo at %s: run `make assets`", path)
	}
	return b
}

func TestShippedResource(t *testing.T) {
	raw := requireShipped(t)

	sum := sha256.Sum256(raw)
	if got := hex.EncodeToString(sum[:]); got != shippedSHA256 {
		t.Fatalf("sha256 %s, want %s -- this is not the resource the numbers below describe",
			got, shippedSHA256)
	}
	if len(raw) != ShippedLength {
		t.Fatalf("%d bytes, want kDemoLength = %d", len(raw), ShippedLength)
	}

	s, err := Decode(raw)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(s) != ShippedRecords {
		t.Fatalf("%d records, want %d", len(s), ShippedRecords)
	}

	// The elimination argument of docs/analysis/input.md §14.1 in one assertion: a 6-byte
	// stride read big-endian is the only reading of these bytes that yields a monotonic frame
	// sequence, so a monotonic sequence is the proof the stride and the byte order are right.
	if err := s.Validate(); err != nil {
		t.Fatalf("shipped stream does not validate: %v", err)
	}

	if got, want := s[0], (Record{Frame: 46, Key: KeyRight, Pad: 114}); got != want {
		t.Errorf("first record %+v, want %+v", got, want)
	}
	if got, want := s[len(s)-1], (Record{Frame: 3414, Key: KeyRight, Pad: 1}); got != want {
		t.Errorf("last record %+v, want %+v", got, want)
	}
	if got, want := s.LastFrame(), int64(3414); got != want {
		t.Errorf("LastFrame %d, want %d", got, want)
	}

	// The histogram. **No battery records at all**, which is why difference 4 of
	// docs/analysis/input.md §14.3 -- playback's missing `batteryTotal != 0` guard -- is
	// unreachable with this stream and reachable with any stream this port records.
	want := map[Key]int{KeyRight: 910, KeyLeft: 198, KeyBand: 9}
	got := s.Keys()
	for k, n := range want {
		if got[k] != n {
			t.Errorf("%d %s records, want %d", got[k], k, n)
		}
	}
	if n := got[KeyBatt]; n != 0 {
		t.Errorf("%d batt records; the shipped demo never uses the battery", n)
	}
	if len(got) != len(want) {
		t.Errorf("key codes %v, want exactly %v", got, want)
	}

	// The padding byte is heap residue and not data (Record.Pad). 109 distinct values is the
	// evidence: a field the recorder wrote would have one or two.
	pads := map[byte]bool{}
	for _, r := range s {
		pads[r.Pad] = true
	}
	if len(pads) != 109 {
		t.Errorf("%d distinct padding values, want 109", len(pads))
	}

	// And the round trip, on the bytes that matter most: the decoder is only trustworthy if
	// re-encoding reproduces the resource, padding included.
	if enc := s.Encode(); !bytes.Equal(enc, raw) {
		t.Errorf("Encode(Decode(shipped)) differs from the resource")
	}
}

func TestShippedLoaders(t *testing.T) {
	raw := requireShipped(t)

	direct, err := Load(filepath.Join(assetRoot, ShippedPath))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	shipped, err := LoadShipped(assetRoot)
	if err != nil {
		t.Fatalf("LoadShipped: %v", err)
	}
	viaReader, err := Read(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(direct) != len(shipped) || len(direct) != len(viaReader) {
		t.Fatalf("three loaders, three lengths: %d, %d, %d", len(direct), len(shipped), len(viaReader))
	}
	for i := range direct {
		if direct[i] != shipped[i] || direct[i] != viaReader[i] {
			t.Fatalf("record %d differs between loaders: %+v %+v %+v",
				i, direct[i], shipped[i], viaReader[i])
		}
	}

	// A missing file comes back as the filesystem's own error, because the shell's attract
	// mode tests it with os.IsNotExist and carries on without a demo.
	if _, err := LoadShipped(filepath.Join(t.TempDir(), "nothing")); !os.IsNotExist(err) {
		t.Errorf("LoadShipped of an empty tree: %v, want a not-exist error", err)
	}
}

func TestDecodeRoundTrip(t *testing.T) {
	// Big-endian by hand, so this test would fail on a little-endian decoder even if
	// Encode were wrong the same way.
	raw := []byte{
		0x00, 0x00, 0x00, 0x2e, 0x00, 0x72, // frame 46, right, pad 'r'
		0x00, 0x00, 0x01, 0x00, 0x01, 0x00, // frame 256, left, pad 0
		0x00, 0x00, 0xff, 0xff, 0x03, 0xff, // frame 65535, band, pad 255
		0x00, 0x01, 0x00, 0x00, 0x02, 0x41, // frame 65536, batt, pad 'A'
	}
	s, err := Decode(raw)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	want := Stream{
		{Frame: 46, Key: KeyRight, Pad: 'r'},
		{Frame: 256, Key: KeyLeft, Pad: 0},
		{Frame: 65535, Key: KeyBand, Pad: 255},
		{Frame: 65536, Key: KeyBatt, Pad: 'A'},
	}
	if len(s) != len(want) {
		t.Fatalf("%d records, want %d", len(s), len(want))
	}
	for i := range want {
		if s[i] != want[i] {
			t.Errorf("record %d = %+v, want %+v", i, s[i], want[i])
		}
	}
	if got := s.Encode(); !bytes.Equal(got, raw) {
		t.Errorf("Encode = % x, want % x", got, raw)
	}

	var buf bytes.Buffer
	n, err := s.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	if n != int64(len(raw)) || !bytes.Equal(buf.Bytes(), raw) {
		t.Errorf("WriteTo wrote %d bytes % x, want %d % x", n, buf.Bytes(), len(raw), raw)
	}

	path := filepath.Join(t.TempDir(), "demo.bin")
	if err := s.WriteFile(path); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	back, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !bytes.Equal(back.Encode(), raw) {
		t.Errorf("file round trip differs")
	}
}

func TestDecodeEmpty(t *testing.T) {
	// An empty file is a legal stream of no records, not an error: a fresh recorder that
	// logged nothing writes one, and playback of it is a game with no input.
	s, err := Decode(nil)
	if err != nil {
		t.Fatalf("Decode(nil): %v", err)
	}
	if len(s) != 0 {
		t.Fatalf("%d records from no bytes", len(s))
	}
	if got := s.LastFrame(); got != -1 {
		t.Errorf("LastFrame of an empty stream = %d, want -1", got)
	}
	if err := s.Validate(); err != nil {
		t.Errorf("empty stream does not validate: %v", err)
	}
	if _, ok := s.Cursor().Key(0); ok {
		t.Error("an empty cursor produced a key")
	}
}

func TestDecodePartialRecord(t *testing.T) {
	for _, n := range []int{1, 5, 7, RecordSize*3 - 1} {
		if _, err := Decode(make([]byte, n)); err == nil {
			t.Errorf("Decode of %d bytes: no error", n)
		} else if !strings.Contains(err.Error(), "6-byte records") {
			t.Errorf("Decode of %d bytes: %v", n, err)
		}
	}

	// Load reports the path, because the tool that reads a stream is reading a file
	// somebody named on a command line.
	path := filepath.Join(t.TempDir(), "short.bin")
	if err := os.WriteFile(path, []byte{1, 2, 3}, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load of a 3-byte file: no error")
	}
	if !strings.Contains(err.Error(), "short.bin") {
		t.Errorf("error does not name the file: %v", err)
	}
}

func TestKeyNames(t *testing.T) {
	// The four codes and their spelling, pinned because they are the format: right is 0 and
	// left is 1, which is the opposite of GetDemoInput's own comments. See Key.
	for _, tc := range []struct {
		k    Key
		name string
		ok   bool
	}{
		{KeyRight, "right", true},
		{KeyLeft, "left", true},
		{KeyBatt, "batt", true},
		{KeyBand, "band", true},
		{Key(4), "key(4)", false},
		{Key(255), "key(255)", false},
	} {
		if got := tc.k.String(); got != tc.name {
			t.Errorf("Key(%d).String() = %q, want %q", byte(tc.k), got, tc.name)
		}
		if got := tc.k.Valid(); got != tc.ok {
			t.Errorf("Key(%d).Valid() = %v, want %v", byte(tc.k), got, tc.ok)
		}
	}
	if KeyRight != 0 || KeyLeft != 1 || KeyBatt != 2 || KeyBand != 3 {
		t.Fatalf("the wire codes moved: %d %d %d %d", KeyRight, KeyLeft, KeyBatt, KeyBand)
	}
}

func TestValidate(t *testing.T) {
	for _, tc := range []struct {
		name string
		s    Stream
		want string // a substring, or "" for valid
	}{
		{"empty", Stream{}, ""},
		{"one", Stream{{Frame: 0, Key: KeyRight}}, ""},
		{"increasing", Stream{{Frame: 1}, {Frame: 2}, {Frame: 100}}, ""},
		{"negative", Stream{{Frame: -1}}, "32-bit field"},
		{"too big", Stream{{Frame: 0x80000000}}, "32-bit field"},
		{"repeated", Stream{{Frame: 7}, {Frame: 7}}, "does not follow"},
		{"descending", Stream{{Frame: 9}, {Frame: 8}}, "does not follow"},
		{"bad key", Stream{{Frame: 1, Key: Key(9)}}, "not one of 0..3"},
		// Both problems: the ordering is the one reported, because a stream that stalls
		// has nothing to say about the keys after the stall.
		{"both", Stream{{Frame: 5, Key: Key(9)}, {Frame: 5}}, "does not follow"},
	} {
		err := tc.s.Validate()
		switch {
		case tc.want == "" && err != nil:
			t.Errorf("%s: %v, want valid", tc.name, err)
		case tc.want != "" && err == nil:
			t.Errorf("%s: valid, want %q", tc.name, tc.want)
		case tc.want != "" && err != nil && !strings.Contains(err.Error(), tc.want):
			t.Errorf("%s: %v, want %q", tc.name, err, tc.want)
		}
	}
}

func TestWriteRefusesInvalid(t *testing.T) {
	// The asymmetry WriteTo documents: a stream that would stall on playback is readable and
	// unwritable. This is the last moment anybody can be told, because the file itself has no
	// way to complain.
	bad := Stream{{Frame: 3}, {Frame: 3}}

	var buf bytes.Buffer
	if n, err := bad.WriteTo(&buf); err == nil {
		t.Error("WriteTo accepted a repeated frame")
	} else if n != 0 || buf.Len() != 0 {
		t.Errorf("WriteTo wrote %d bytes (%d buffered) before refusing", n, buf.Len())
	}

	path := filepath.Join(t.TempDir(), "bad.bin")
	if err := bad.WriteFile(path); err == nil {
		t.Error("WriteFile accepted a repeated frame")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("WriteFile created %s anyway", path)
	}

	// Encode itself does not check, and truncates a frame that does not fit. Documented
	// rather than guarded, and pinned here so that a future guard is a deliberate change.
	if got := (Stream{{Frame: 0x1_0000_002e}}).Encode(); !bytes.Equal(got, []byte{0, 0, 0, 0x2e, 0, 0}) {
		t.Errorf("Encode of an oversized frame = % x, want the low 32 bits", got)
	}
}

func TestCursor(t *testing.T) {
	s := Stream{
		{Frame: 10, Key: KeyRight},
		{Frame: 11, Key: KeyLeft},
		{Frame: 20, Key: KeyBand},
	}
	c := s.Cursor()
	if c.Len() != 3 || c.Index() != 0 || c.Done() || c.PastEnd() != 0 {
		t.Fatalf("fresh cursor: len %d index %d done %v past %d", c.Len(), c.Index(), c.Done(), c.PastEnd())
	}
	if len(c.Stream()) != len(s) {
		t.Errorf("Stream() has %d records, want %d", len(c.Stream()), len(s))
	}

	// Frames before the first record produce nothing and do not advance: the demo's first
	// 46 frames are the glider gliding.
	for f := int64(0); f < 10; f++ {
		if k, ok := c.Key(f); ok {
			t.Fatalf("frame %d produced %s", f, k)
		}
	}
	if c.Index() != 0 {
		t.Fatalf("index %d after nine keyless frames", c.Index())
	}

	if k, ok := c.Key(10); !ok || k != KeyRight {
		t.Fatalf("frame 10 = (%s, %v), want (right, true)", k, ok)
	}
	if c.Index() != 1 {
		t.Errorf("index %d after one match, want 1", c.Index())
	}
	// The same frame again produces nothing: one record per frame, and the cursor has
	// already moved on to frame 11.
	if k, ok := c.Key(10); ok {
		t.Errorf("frame 10 a second time = %s", k)
	}
	if k, ok := c.Key(11); !ok || k != KeyLeft {
		t.Fatalf("frame 11 = (%s, %v), want (left, true)", k, ok)
	}
	// Frames 12..19 are a gap. The record for 20 is still waiting and is not applied early,
	// which is the equality test doing its job.
	for f := int64(12); f < 20; f++ {
		if k, ok := c.Key(f); ok {
			t.Fatalf("frame %d produced %s early", f, k)
		}
	}
	if k, ok := c.Key(20); !ok || k != KeyBand {
		t.Fatalf("frame 20 = (%s, %v), want (band, true)", k, ok)
	}
	if !c.Done() {
		t.Error("cursor not done after the last record")
	}

	// Past the end: the original's out-of-bounds read, counted instead.
	for f := int64(21); f <= 25; f++ {
		if _, ok := c.Key(f); ok {
			t.Fatalf("frame %d produced a key past the end", f)
		}
	}
	if got := c.PastEnd(); got != 5 {
		t.Errorf("PastEnd = %d, want 5", got)
	}

	// Reset is `demoIndex = 0`, and deliberately does not clear the diagnostic.
	c.Reset()
	if c.Index() != 0 || c.Done() {
		t.Errorf("after Reset: index %d done %v", c.Index(), c.Done())
	}
	if got := c.PastEnd(); got != 5 {
		t.Errorf("Reset cleared PastEnd (%d); it is a fact about the run, not the game", got)
	}
	if k, ok := c.Key(10); !ok || k != KeyRight {
		t.Errorf("after Reset frame 10 = (%s, %v), want (right, true)", k, ok)
	}
}

func TestCursorRepeatedFrameStalls(t *testing.T) {
	// The hazard, demonstrated: two records on frame 10 do not cost one action, they cost
	// **every record after the first**. The cursor is left pointing at a frame the game has
	// already passed, and the equality test can never match again.
	s := Stream{
		{Frame: 10, Key: KeyRight},
		{Frame: 10, Key: KeyBand}, // the same frame: unreachable
		{Frame: 11, Key: KeyLeft}, // and so is this, and everything else
		{Frame: 12, Key: KeyLeft},
	}
	c := s.Cursor()
	if k, ok := c.Key(10); !ok || k != KeyRight {
		t.Fatalf("frame 10 = (%s, %v)", k, ok)
	}
	for f := int64(11); f < 200; f++ {
		if k, ok := c.Key(f); ok {
			t.Fatalf("frame %d produced %s; a stalled cursor must produce nothing", f, k)
		}
	}
	if c.Index() != 1 {
		t.Errorf("index %d, want 1: the cursor is stranded on the duplicate", c.Index())
	}
	if c.PastEnd() != 0 {
		t.Errorf("PastEnd = %d; a stall is not an overrun and must not be reported as one", c.PastEnd())
	}
	// Which is why Validate refuses it before anybody can write one.
	if err := s.Validate(); err == nil {
		t.Error("Validate accepted the stalling stream")
	}
}

func TestRecorder(t *testing.T) {
	r := NewRecorder()
	if len(r.Stream()) != 0 || r.Dropped() != 0 {
		t.Fatal("a fresh recorder is not empty")
	}

	r.Log(0, KeyRight) // frame 0 is a legal first frame
	r.Log(5, KeyLeft)
	r.Log(5, KeyBand)  // same frame: dropped, and the direction survives
	r.Log(5, KeyBatt)  // and again
	r.Log(4, KeyRight) // a frame in the past: dropped too
	r.Log(6, KeyBand)

	want := Stream{
		{Frame: 0, Key: KeyRight},
		{Frame: 5, Key: KeyLeft},
		{Frame: 6, Key: KeyBand},
	}
	got := r.Stream()
	if len(got) != len(want) {
		t.Fatalf("%d records, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("record %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	if r.Dropped() != 3 {
		t.Errorf("Dropped = %d, want 3", r.Dropped())
	}

	// The pad byte is zero in anything this port records: it was never data.
	for i, rec := range got {
		if rec.Pad != 0 {
			t.Errorf("record %d has pad %d; a recorder writes zero", i, rec.Pad)
		}
	}

	// And the whole point of the one-per-frame rule: what a recorder produces is always
	// writable, and therefore always replays to its end.
	if err := got.Validate(); err != nil {
		t.Errorf("recorded stream does not validate: %v", err)
	}
	c := got.Cursor()
	for _, rec := range want {
		if k, ok := c.Key(rec.Frame); !ok || k != rec.Key {
			t.Errorf("replay of frame %d = (%s, %v), want (%s, true)", rec.Frame, k, ok, rec.Key)
		}
	}
	if !c.Done() {
		t.Error("replaying a recorded stream did not consume it")
	}
}
