package house

// The exported high-score codec: 292 bytes on their own, out of a house and back.
//
// The vendored houses are never written to by this port, so the board a player fills in
// has to live somewhere else -- a side-car file of exactly these 292 bytes
// (docs/analysis/scoring.md 7.14). That makes EncodeScores/DecodeScores the only writers
// of a 1994 structure outside the house codec, which is why they are checked against the
// shipped headers rather than only against themselves: a codec that round-trips its own
// output is consistent, not correct.

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
)

// A synthetic board, chosen so that every field is distinguishable from every other and
// so that the residue past each Pascal string's length byte is non-zero. Residue is the
// part a lazier codec loses, and losing it is invisible until a side-car written by this
// build is compared with the board it was seeded from.
func sampleScores() Scores {
	var s Scores
	s.Banner = PStr32{}
	copy(s.Banner[:], append([]byte{5}, []byte("Hello")...))
	for i := 6; i < len(s.Banner); i++ {
		s.Banner[i] = byte(0xA0 + i) // residue
	}
	for i := range s.Names {
		n := PStr16{}
		name := "Player " + string(rune('A'+i))
		copy(n[:], append([]byte{byte(len(name))}, name...))
		for j := 1 + len(name); j < len(n); j++ {
			n[j] = byte(0x40 + j + i)
		}
		s.Names[i] = n
		s.Scores[i] = int32(10000 - i*137)
		s.TimeStamps[i] = uint32(0x50000000 + i)
		s.Levels[i] = int16(40 - i)
	}
	return s
}

func TestScoresCodecRoundTrips(t *testing.T) {
	in := sampleScores()
	b := EncodeScores(&in)
	if len(b) != SizeofScores {
		t.Fatalf("EncodeScores produced %d bytes, want %d", len(b), SizeofScores)
	}
	out, err := DecodeScores(b)
	if err != nil {
		t.Fatalf("DecodeScores: %v", err)
	}
	if out != in {
		t.Errorf("round trip changed the board:\n have %+v\n want %+v", out, in)
	}
	// And the other way round, which is the property the side-car needs: two encodes of
	// the same board are the same bytes, so writing the file twice is not a diff.
	if again := EncodeScores(&out); !bytes.Equal(again, b) {
		t.Error("re-encoding the decoded board produced different bytes")
	}
}

// The board's bytes are big-endian in the order scoresType declares them: the banner,
// then all ten names, then all ten scores, then all ten timestamps, then all ten room
// counts. Field-by-field, because "it round-trips" would hold just as well for a codec
// that had two fields swapped.
func TestScoresCodecLayout(t *testing.T) {
	var s Scores
	s.Banner[0] = 1
	s.Banner[1] = 'X'
	s.Names[0][0] = 1
	s.Names[0][1] = 'a'
	s.Names[9][0] = 1
	s.Names[9][1] = 'z'
	s.Scores[0] = 0x01020304
	s.Scores[9] = -1
	s.TimeStamps[0] = 0xAABBCCDD
	s.Levels[0] = 0x0102
	s.Levels[9] = 0x7F7E

	b := EncodeScores(&s)
	for _, c := range []struct {
		at   int
		want []byte
		what string
	}{
		{0, []byte{1, 'X'}, "banner"},
		{32, []byte{1, 'a'}, "names[0]"},
		{32 + 9*16, []byte{1, 'z'}, "names[9]"},
		{192, []byte{0x01, 0x02, 0x03, 0x04}, "scores[0]"},
		{192 + 9*4, []byte{0xFF, 0xFF, 0xFF, 0xFF}, "scores[9]"},
		{232, []byte{0xAA, 0xBB, 0xCC, 0xDD}, "timeStamps[0]"},
		{272, []byte{0x01, 0x02}, "levels[0]"},
		{272 + 9*2, []byte{0x7F, 0x7E}, "levels[9]"},
	} {
		if got := b[c.at : c.at+len(c.want)]; !bytes.Equal(got, c.want) {
			t.Errorf("%s at byte %d is % X, want % X", c.what, c.at, got, c.want)
		}
	}
}

// The length is a requirement, not a hint. This is the one place the port deliberately
// differs from the original, which read its side-car with GetEOF and one unchecked FSRead
// into the middle of the house struct -- so a 400-byte file overwrote the saved game and
// the room count behind the board (scoring.md 7.14). Callers that want to survive a
// truncated file overlay it onto a known-good board instead, which is a decision they
// make rather than one the reader makes for them.
func TestScoresCodecRefusesTheWrongLength(t *testing.T) {
	for _, n := range []int{0, 1, SizeofScores - 1, SizeofScores + 1, 4096} {
		_, err := DecodeScores(make([]byte, n))
		if err == nil {
			t.Errorf("DecodeScores accepted %d bytes", n)
			continue
		}
		// Both numbers, because the player's side-car is a file they may have to look
		// at, and "wrong length" without the lengths is not actionable.
		msg := err.Error()
		for _, want := range []string{strconv.Itoa(n), strconv.Itoa(SizeofScores)} {
			if !strings.Contains(msg, want) {
				t.Errorf("DecodeScores(%d bytes): %q does not mention %s", n, msg, want)
			}
		}
	}
}

// The load-bearing one: for all 22 shipped houses, the exported codec agrees with the
// bytes in the file at offHighScores. Those are the boards a side-car is seeded from, and
// they carry real 1994 stack garbage in their residue -- SortHighScores sorted through an
// uninitialised local -- so this compares bytes rather than fields on purpose.
func TestScoresCodecMatchesEveryShippedHeader(t *testing.T) {
	for _, c := range loadCorpus(t) {
		want := c.raw[offHighScores : offHighScores+SizeofScores]

		got := EncodeScores(&c.house.HighScores)
		if !bytes.Equal(got, want) {
			at := 0
			for at < len(got) && got[at] == want[at] {
				at++
			}
			t.Errorf("%s: EncodeScores differs from the file at byte %d (%d of the board): "+
				"have %02X, want %02X", c.stem, offHighScores+at, at, got[at], want[at])
			continue
		}
		s, err := DecodeScores(want)
		if err != nil {
			t.Errorf("%s: DecodeScores of the file's own bytes: %v", c.stem, err)
			continue
		}
		if s != c.house.HighScores {
			t.Errorf("%s: DecodeScores disagrees with the board Load produced", c.stem)
		}
	}
}
