package house

// Hostile bytes, on purpose (docs/IMPROVEMENTS.md 4.30).
//
// A house is the file a player is most likely to be handed by somebody else, and SECURITY.md says
// parsing other people's files is the attack surface. So everything here that reads one is a fuzz
// target: the binary house, its text form, the saved game and the high-score board. Plain `go test`
// runs each over its seeds. `make fuzz` runs the engine, and an input that fails is written to
// testdata/fuzz/<target>/ to be committed, which makes it a seed from then on.
//
// Each target checks the codec's own promise, not just that nothing panicked. Load's is that Save
// gives back the bytes it read, and the text format's is that its residue mode does the same.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedHouseLimit is the largest shipped house used as a seed. The engine mutates a seed a byte at a
// time and runs the whole round trip on every mutation, so a 180 KB house is mostly time spent
// copying 180 KB. The five houses under it are 1.5 to 17 KB each, and they include Sampler's
// PowerPC slack and a house with holes in its object slots. TestCorpusRoundTrip already holds every
// shipped house to the same round trip.
const seedHouseLimit = 20 << 10

// shippedSeeds is the shipped houses at or under seedHouseLimit, in name order. None at all when
// the tree has no extracted assets: the targets still have the seeds built in code below.
func shippedSeeds(f *testing.F) [][]byte {
	f.Helper()
	names, _ := filepath.Glob(filepath.Join("..", "..", "assets", "extracted", "houses", "*.house"))
	var out [][]byte
	for _, name := range names {
		b, err := os.ReadFile(name)
		if err != nil {
			f.Fatal(err)
		}
		if len(b) <= seedHouseLimit {
			out = append(out, b)
		}
	}
	return out
}

// minimalText is the smallest house the text format describes that still has a room and objects in
// it, as TestParseMinimalHouse types it.
const minimalText = `format 1

house
    version    0x0200
    firstroom  0
    banner     "Hello"
    trailer    "Bye"
    initial    100 200

room 0 "Start"
    bounds     0x001f
    background 2000
    tiles      0 1 2 3 4 5 6 7
    floor      0
    suite      60
    start      10 20
    object 0   kFloorVent  at 200 64  distance 5  initial 1 state 1 vector 0 tall 0
    object 4   kTable      rect 100 50 110 150  pict 0
`

// builtSeeds is a house with no rooms, the same with PowerPC slack, and minimalText saved.
func builtSeeds(f *testing.F) [][]byte {
	f.Helper()
	var out [][]byte
	minimal, err := ParseText(strings.NewReader(minimalText))
	if err != nil {
		f.Fatal(err)
	}
	for _, h := range []*House{{}, {Slack: []byte{1, 1}}, minimal} {
		b, err := h.Save()
		if err != nil {
			f.Fatal(err)
		}
		out = append(out, b)
	}
	return out
}

// FuzzLoad is a house file from anywhere.
//
// What must hold for any file Load accepts:
//   - Save gives back the file, byte for byte (Save's own comment);
//   - the text with residue parses back to the same bytes (TextOptions.Residue's);
//   - the text without it parses back to Canonical, field for field;
//   - Lint gets to the end, with no asset tree to check against.
//
// And nothing panics on a file it refuses.
func FuzzLoad(f *testing.F) {
	for _, b := range append(builtSeeds(f), shippedSeeds(f)...) {
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		h, err := Load(b)
		if err != nil {
			return
		}
		out, err := h.Save()
		if err != nil {
			t.Fatalf("Load took the file and Save refuses it: %v", err)
		}
		if d := firstDiff(b, out); len(out) != len(b) || d >= 0 {
			t.Fatalf("Save changed the file it loaded: %d bytes to %d, first at %d (%s)",
				len(b), len(out), d, locate(max(d, 0)))
		}

		txt, err := h.Text(TextOptions{Residue: true})
		if err != nil {
			t.Fatalf("a loaded house has no text: %v", err)
		}
		back, err := ParseText(strings.NewReader(txt))
		if err != nil {
			t.Fatalf("the text of a loaded house does not parse: %v", err)
		}
		if again, err := back.Save(); err != nil || !bytes.Equal(again, b) {
			t.Fatalf("the residue text of a loaded house does not save back to the file "+
				"(%v, %d bytes of %d)", err, len(again), len(b))
		}

		txt, err = h.Text(TextOptions{})
		if err != nil {
			t.Fatalf("a loaded house has no canonical text: %v", err)
		}
		back, err = ParseText(strings.NewReader(txt))
		if err != nil {
			t.Fatalf("the canonical text of a loaded house does not parse: %v", err)
		}
		if diff := diffHouses(h.Canonical(), back); diff != "" {
			t.Fatalf("the canonical text lost a field: %s", diff)
		}

		h.Lint(LintOptions{})
	})
}

// FuzzParseText is a house somebody typed, or a text somebody else's program wrote.
//
// What must hold for any text ParseText accepts, when the house it makes fits a file:
//   - the file loads, and saves back to itself;
//   - the text with residue parses back to the same file;
//   - the text without it is a fixed point: parsed and written again, it is the same text,
//     which is TestTextIdempotent's property and the reason a house in git does not churn;
//   - Lint gets to the end.
func FuzzParseText(f *testing.F) {
	f.Add(minimalText)
	f.Add("format 1\n")
	f.Add("")
	for _, b := range append(builtSeeds(f), shippedSeeds(f)...) {
		// Sampler and California or Bust!. The text of a house is five to ten times its
		// file, so the bigger seeds are FuzzLoad's.
		if len(b) > 8<<10 {
			continue
		}
		h, err := Load(b)
		if err != nil {
			f.Fatal(err)
		}
		for _, opt := range []TextOptions{{}, {Residue: true}} {
			txt, err := h.Text(opt)
			if err != nil {
				f.Fatal(err)
			}
			f.Add(txt)
		}
	}
	f.Fuzz(func(t *testing.T, src string) {
		h, err := ParseText(strings.NewReader(src))
		if err != nil {
			return
		}
		h.Lint(LintOptions{})
		file, err := h.Save()
		if err != nil {
			return // a parsed house can be one no file can hold; Save says so
		}
		loaded, err := Load(file)
		if err != nil {
			t.Fatalf("a parsed house saves to a file Load refuses: %v", err)
		}
		if again, err := loaded.Save(); err != nil || !bytes.Equal(again, file) {
			t.Fatalf("a parsed house's file does not load and save to itself (%v)", err)
		}

		txt, err := h.Text(TextOptions{Residue: true})
		if err != nil {
			t.Fatalf("a parsed house has no text: %v", err)
		}
		back, err := ParseText(strings.NewReader(txt))
		if err != nil {
			t.Fatalf("the text written for a parsed house does not parse: %v", err)
		}
		if again, err := back.Save(); err != nil || !bytes.Equal(again, file) {
			t.Fatalf("the residue text of a parsed house does not save to the same file (%v)", err)
		}

		first, err := h.Text(TextOptions{})
		if err != nil {
			t.Fatalf("a parsed house has no canonical text: %v", err)
		}
		back, err = ParseText(strings.NewReader(first))
		if err != nil {
			t.Fatalf("the canonical text of a parsed house does not parse: %v", err)
		}
		second, err := back.Text(TextOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if first != second {
			t.Fatalf("the writer is not a fixed point: %s", firstLineDiff(first, second))
		}
	})
}

// FuzzDecodeSavedGame is a saved game from anywhere. A save that decodes encodes back to itself,
// which is the fixed point TestSavedGameRoundTrips states: a player who resumes and saves again
// must not lose something a byte at a time.
func FuzzDecodeSavedGame(f *testing.F) {
	for _, rooms := range []int{0, 1, 3} {
		b, err := sampleSavedGame(rooms).Encode()
		if err != nil {
			f.Fatal(err)
		}
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		sg, err := DecodeSavedGame(b)
		if err != nil {
			return
		}
		out, err := sg.Encode()
		if err != nil {
			t.Fatalf("a decoded save does not encode: %v", err)
		}
		if !bytes.Equal(out, b) {
			t.Fatalf("a decoded save encodes to other bytes: %d to %d", len(b), len(out))
		}
	})
}

// FuzzDecodeScores is a high-score side-car from anywhere. Every byte of a board is a field, so a
// board that decodes encodes back to the same bytes, residue and all (EncodeScores).
func FuzzDecodeScores(f *testing.F) {
	f.Add(make([]byte, SizeofScores))
	for _, b := range shippedSeeds(f) {
		h, err := Load(b)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(EncodeScores(&h.HighScores))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		s, err := DecodeScores(b)
		if err != nil {
			return
		}
		if out := EncodeScores(&s); !bytes.Equal(out, b) {
			t.Fatal("a decoded board encodes to other bytes")
		}
	})
}
