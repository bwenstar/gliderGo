package house

import (
	"testing"
	"time"
)

// TestMacTimeRoundTrip checks both encodings against each other.
func TestMacTimeRoundTrip(t *testing.T) {
	for _, want := range []string{
		"1995-07-19 14:03:11", "2000-05-11 19:52:08", "1904-01-01 00:00:00",
		"2026-09-11 00:00:00",
	} {
		tm, err := time.Parse("2006-01-02 15:04:05", want)
		if err != nil {
			t.Fatal(err)
		}
		if got := MacTime(MacTimeFrom(tm)); !got.Equal(tm) {
			t.Errorf("MacTime(MacTimeFrom(%s)) = %s", want, got)
		}
		if got := MacTimeMasked(MacTimeMaskedFrom(tm)); !got.Equal(tm) && tm.Year() > 1972 {
			t.Errorf("MacTimeMasked(MacTimeMaskedFrom(%s)) = %s", want, got)
		}
	}
	// The masked encoder must leave bit 31 clear, as the original's write path did.
	tm := time.Date(1995, 7, 19, 0, 0, 0, 0, time.UTC)
	if uint32(MacTimeMaskedFrom(tm))&MaskedTimeBit != 0 {
		t.Error("MacTimeMaskedFrom left bit 31 set")
	}
}

// TestHouseTimeStampIsMasked is the evidence for MacTimeMasked, checked against
// the corpus rather than asserted in a comment.
//
// Read as stored, every shipped house claims a date in the 1920s or 30s. With bit
// 31 restored they land in 1995 and 2000, and for eleven of the twenty-two the
// restored date equals -- to the day -- the newest high-score stamp in the same
// file. Those score stamps are unsigned and unmasked, so they are independent of
// the hypothesis being tested.
func TestHouseTimeStampIsMasked(t *testing.T) {
	corpus := loadCorpus(t)

	lo := time.Date(1995, 1, 1, 0, 0, 0, 0, time.UTC)
	hi := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	sameDay, comparable := 0, 0

	for _, c := range corpus {
		asStored := MacTime(c.house.TimeStamp)
		if asStored.Year() > 1970 {
			t.Errorf("%s: stored timestamp decodes to %s; the premise of the "+
				"bit-31 correction is that it is implausibly early",
				c.stem, asStored.Format("2006-01-02"))
		}
		restored := c.house.Saved()
		if restored.Before(lo) || !restored.Before(hi) {
			t.Errorf("%s: restored timestamp is %s, outside 1995..2000",
				c.stem, restored.Format("2006-01-02"))
		}

		var newest uint32
		for _, s := range c.house.HighScores.TimeStamps {
			if s > newest {
				newest = s
			}
		}
		if newest == 0 {
			continue // Empty House and Land of Illusion have no scores
		}
		comparable++
		if MacTime(int32(newest)).Format("2006-01-02") == restored.Format("2006-01-02") {
			sameDay++
		}
	}

	if comparable != 20 {
		t.Errorf("%d houses have high scores to compare against, expected 20", comparable)
	}
	if sameDay != 11 {
		t.Errorf("%d houses' restored save date matches their newest high score "+
			"to the day, expected 11", sameDay)
	}
}

// TestLockBit pins the other use of the same field: bit 0 is the house lock flag,
// 15 locked and 7 unlocked across the corpus (house-format.md 3.3).
func TestLockBit(t *testing.T) {
	locked, unlocked := 0, 0
	for _, c := range loadCorpus(t) {
		if c.house.Unlocked() {
			unlocked++
		} else {
			locked++
		}
	}
	if locked != 15 || unlocked != 7 {
		t.Errorf("%d locked / %d unlocked, house-format.md 3.3 says 15 / 7",
			locked, unlocked)
	}
}
