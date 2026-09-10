package house

import "time"

// Classic Mac OS timestamps: seconds since 1904-01-01 00:00:00, written by
// GetDateTime, which returns *local* time with no zone recorded anywhere. A 1995
// date is about 2.9e9 seconds, which does not fit a signed 32-bit integer, so
// these values must be read as unsigned even though the C declared `long`.
//
// A house file contains three of them under two different conventions, which is
// why this file exists rather than one call to time.Unix:
//
//   - scoresType.timeStamps[] is `unsigned long`, stored as written. Read with
//     MacTime.
//   - gameType.timeStamp is a signed `long` straight from GetDateTime, so it
//     appears negative in Go. uint32 conversion recovers it; MacTime does that.
//   - houseType.timeStamp is masked. Read with MacTimeMasked, and see below.
//
// The port never uses any of them for logic -- high-score ordering is by score,
// and the house lock flag is bit 0 of the stamp rather than the date -- so all
// that is needed is a legible rendering that does not lie about its assumptions.

// MacEpochOffset is 1904-01-01 to 1970-01-01 in seconds.
const MacEpochOffset = 2082844800

// MacTime converts a Mac timestamp to a Go time, interpreted as UTC because the
// original recorded no zone. Treat the result as "the wall clock the author saw",
// not as an instant.
func MacTime(mac int32) time.Time {
	return time.Unix(int64(uint32(mac))-MacEpochOffset, 0).UTC()
}

// MacTimeFrom converts the other way, for houses this port writes. The result is
// suitable for the score and saved-game stamps; use MacTimeMaskedFrom for the
// house stamp.
func MacTimeFrom(t time.Time) int32 {
	return int32(uint32(t.Unix() + MacEpochOffset))
}

// MaskedTimeBit is the bit that houseType.timeStamp loses on the way to disk.
// Declared unsigned so the arithmetic below reads as the bit operation it is;
// as an int32 the same pattern is the sign bit.
const MaskedTimeBit uint32 = 0x80000000

// MacTimeMasked decodes houseType.timeStamp, which is not a plain Mac timestamp:
// WriteHouse does `timeStamp &= 0x7FFFFFFF` (GliderPRO/Sources/HouseIO.c:478)
// before storing it, discarding bit 31. Since Mac seconds passed 2^31 in May
// 1972, every real house date has that bit set, so the stored value reads as a
// date about 68 years too early and the fix is to put the bit back.
//
// The corpus settles it. Read as stored, all 22 shipped houses claim to have
// been written in 1927 or 1932, before either the Macintosh or the authors. With
// bit 31 restored they fall in 1995-06 to 1995-12, except Sampler in 2000-05 --
// and for eleven of them the restored date equals, to the day, the newest
// high-score timestamp in the same file. Those score stamps are `unsigned long`
// and are *not* masked, so they are independent evidence rather than the same
// assumption twice.
//
// The mask was presumably meant to keep the value non-negative in a signed long;
// its real effect is that no house ever recorded the year it was made. The bit is
// restored on read and re-cleared on write, so a round trip is unaffected.
func MacTimeMasked(mac int32) time.Time {
	return MacTime(int32(uint32(mac) | MaskedTimeBit))
}

// MacTimeMaskedFrom encodes a house timestamp the way the original did, bit 31
// cleared and bit 0 left for the caller's lock flag.
func MacTimeMaskedFrom(t time.Time) int32 {
	return int32(uint32(MacTimeFrom(t)) &^ MaskedTimeBit)
}

// Saved is when the house was last written, as far as the file records it.
func (h *House) Saved() time.Time { return MacTimeMasked(h.TimeStamp) }

// macTimeString renders an unmasked Mac timestamp for a text-format comment.
func macTimeString(mac int32) string {
	if mac == 0 {
		return ""
	}
	return MacTime(mac).Format("2006-01-02 15:04:05") + " local"
}

// macMaskedTimeString renders houseType.timeStamp: the restored date, plus the
// lock state that bit 0 of the same field encodes.
func macMaskedTimeString(mac int32) string {
	lock := "unlocked"
	if mac&1 != 0 {
		lock = "locked"
	}
	if mac == 0 {
		return lock
	}
	return MacTimeMasked(mac).Format("2006-01-02 15:04:05") + " local, " + lock +
		" (bit 31 restored, see house-format.md 3.3)"
}
