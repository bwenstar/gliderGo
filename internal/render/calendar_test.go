package render

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bwenstar/gliderGo/internal/house"
)

// The calendar is the one object in a static room whose appearance depends on
// something outside the house file: the date. These two tests cover the two halves
// of that -- the strings are the resource's, and the one drawn is the clock's.

// TestMonthNamesMatchTheResource checks the transcription in objectdraw2.go against
// STR# 1005, which is what makes transcribing it legitimate rather than a guess.
//
// It is the same argument TestPaletteMatchesClut makes for the palette: the data is
// generated or copied into Go source so the game does not need a second asset tree
// at runtime, and a test reads the shipped resource so the copy cannot drift from it.
// The decoder is inline and small for the same reason the ColorTable decoder in that
// test is -- STR# is two lines of format, and 1.7 will want the general version when
// it needs STR# 150's fifty-one strings, not before.
func TestMonthNamesMatchTheResource(t *testing.T) {
	dir := requireAssets(t, "res/STR#")
	b, err := os.ReadFile(filepath.Join(dir, "1005.bin"))
	if err != nil {
		t.Fatal(err)
	}
	got := decodeStringList(t, b)

	if len(got) != len(monthNames) {
		t.Fatalf("STR# 1005 has %d strings, want %d", len(got), len(monthNames))
	}
	for i, want := range monthNames {
		if got[i] != want {
			t.Errorf("month %d is %q in the resource, %q in monthNames", i+1, got[i], want)
		}
	}
}

// TestMonthNamesFitTheCalendar is the layout half of the transcription: this font is
// not the nine-point one the original centres, so the fit has to be re-established
// rather than inherited. The picture is 64 wide and the pen is centred in it, so a
// month wider than 64 would hang off both edges.
func TestMonthNamesFitTheCalendar(t *testing.T) {
	for i, m := range monthNames {
		if bad := UnrenderableRunes(m); len(bad) != 0 {
			t.Errorf("month %d %q: unrenderable %q", i+1, m, string(bad))
		}
		if w := StringWidth(m); w > 64 {
			t.Errorf("month %d %q is %d pixels, past the calendar's 64", i+1, m, w)
		}
	}
	// SEPTEMBER is the worst case and is worth stating as a number, because it is
	// the margin a wider font would eat: nine cells of six pixels.
	if w := StringWidth("SEPTEMBER"); w != 54 {
		t.Errorf("SEPTEMBER is %d pixels, want 54", w)
	}
}

// TestDrawCalendarMonth draws a calendar and checks that the month lands where
// ObjectDraw2.c:1162-1163 puts it, in the colour it puts it in.
//
// The expectation is built by drawing the picture and then the string at pen
// coordinates spelled out here as numbers -- 105 and 155 for a calendar at (100,100)
// in September -- rather than by re-running DrawCalendar's own arithmetic. So the
// test asserts the result of the centring, not the shape of the expression.
func TestDrawCalendarMonth(t *testing.T) {
	artDir := requireAssets(t, "art")
	at := Rect{Top: 100, Left: 100, Bottom: 192, Right: 163}

	draw := func(clock time.Time) *Surface {
		s := NewScene(DefaultView(), NewAssets(artDir), testHouse())
		s.Clock = clock
		s.DrawCalendar(at)
		if err := s.A.Err(); err != nil {
			t.Fatal(err)
		}
		return s.Back
	}

	// September: (64 - 54) / 2 = 5 past the left edge, 55 below the top.
	got := draw(time.Date(1994, time.September, 14, 10, 9, 0, 0, time.UTC))

	want := NewSurface(got.W, got.H)
	want.Fill(want.Bounds(), White8)
	art := NewAssets(artDir).Pict(kCalendarPictID)
	if art == nil {
		t.Fatal("no calendar picture")
	}
	want.Copy(art, art.Bounds(), Offset(art.Bounds(), at.Left, at.Top), SrcCopy)
	want.DrawString(105, 155, "SEPTEMBER", DarkFlesh)

	diff := 0
	for i := range got.Pix {
		if got.Pix[i] != want.Pix[i] {
			diff++
		}
	}
	if diff != 0 {
		t.Errorf("%d pixels differ from a calendar drawn with SEPTEMBER at pen (105,155) in colour %d",
			diff, DarkFlesh)
	}

	// The colour is worth its own assertion, because the text would still land in
	// the right place if it were drawn in black.
	found := 0
	for _, v := range got.Pix {
		if v == DarkFlesh {
			found++
		}
	}
	if found == 0 {
		t.Errorf("no pixel is kDarkFleshColor (%d)", DarkFlesh)
	}

	// And the month follows the clock rather than being drawn once and cached.
	march := draw(time.Date(1994, time.March, 14, 10, 9, 0, 0, time.UTC))
	same := true
	for i := range march.Pix {
		if march.Pix[i] != got.Pix[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("March and September draw the same calendar")
	}

	// A zero Clock draws JANUARY. That is not a designed fallback, it is what
	// time.Time's zero value means, and DrawRedClock has the same property -- it
	// reads midnight. Both are only reachable if a caller forgets to set Clock,
	// which cmd/glidergo does at startup; the test is here so the behaviour is
	// written down rather than discovered.
	zero := draw(time.Time{})
	january := draw(time.Date(1994, time.January, 1, 0, 0, 0, 0, time.UTC))
	for i := range zero.Pix {
		if zero.Pix[i] != january.Pix[i] {
			t.Fatal("a zero Clock does not draw JANUARY")
		}
	}
}

// decodeStringList decodes a Mac 'STR#': a big-endian uint16 count, then that many
// Pascal strings back to back. Mac Roman throughout, so the text goes through the
// house loader's decoder -- STR# 150 has strings ending in the single-byte ellipsis
// 0xC9, and this one would too if a month were abbreviated.
func decodeStringList(t *testing.T, b []byte) []string {
	t.Helper()
	if len(b) < 2 {
		t.Fatalf("STR# is %d bytes, too short for a count", len(b))
	}
	n := int(binary.BigEndian.Uint16(b[:2]))
	out := make([]string, 0, n)
	off := 2
	for i := 0; i < n; i++ {
		if off >= len(b) {
			t.Fatalf("STR# ran out after %d of %d strings", i, n)
		}
		length := int(b[off])
		off++
		if off+length > len(b) {
			t.Fatalf("STR# string %d claims %d bytes, %d remain", i+1, length, len(b)-off)
		}
		out = append(out, house.MacRomanToUTF8(b[off:off+length]))
		off += length
	}
	if off != len(b) {
		t.Errorf("STR# has %d bytes past its %d strings", len(b)-off, n)
	}
	return out
}

// TestCalendarIsOnePixelNarrowerThanItsCentring pins the discrepancy the pen
// arithmetic above rests on.
//
// ObjectDraw2.c centres the month in **64** pixels and the calendar picture is
// **63** wide, so the original's own text sits half a pixel -- one pixel, after
// integer division -- right of centre. SEPTEMBER is the case where it shows: (64-54)/2
// is 5 and (63-54)/2 is 4.
//
// The port keeps the 64. It is the original's literal and reproducing it is the point
// of the stage; this test exists so that the next person to notice the off-by-one
// finds it already accounted for rather than "fixing" it and moving every calendar's
// text by a pixel.
func TestCalendarIsOnePixelNarrowerThanItsCentring(t *testing.T) {
	artDir := requireAssets(t, "art")
	art := NewAssets(artDir).Pict(kCalendarPictID)
	if art == nil {
		t.Fatal("no calendar picture")
	}
	if art.W != 63 {
		t.Errorf("PICT %d is %d wide, want 63 -- the width DrawCalendar's 64 is one more than",
			kCalendarPictID, art.W)
	}
	// srcRects says the object is 63 wide and 92 tall, one narrower than the
	// picture. That is the original's table and not a bug -- the rect places the
	// object, the picture's own frame sizes the blit -- but it is the kind of
	// difference that looks like one, so it is pinned here.
	if r := srcRects[kCalendar]; r.Wide() != 63 || r.Tall() != 92 {
		t.Errorf("srcRects[kCalendar] is %dx%d, want 63x92", r.Wide(), r.Tall())
	}
	if art.H != 92 {
		t.Errorf("PICT %d is %d tall, want 92", kCalendarPictID, art.H)
	}
}
