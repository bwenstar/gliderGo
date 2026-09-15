package render

// DrawStringScaled exists because the shell has to be read from across a desk and this
// port has one authored bitmap font where the original had two Toolbox faces. The three
// properties below are the ones internal/shell's layout depends on, and each of them is
// a way the magnified path can quietly disagree with the unmagnified one.

import "testing"

// Scale 1 must be DrawString and not merely resemble it: the shell mixes the two in a
// single panel -- a scale-2 house name beside a scale-1 room count -- so any difference
// at all would show as two fonts on one line.
func TestScaleOneIsDrawStringExactly(t *testing.T) {
	const text = "Slumberland 42 (locked)"
	plain := NewSurface(320, 40)
	scaled := NewSurface(320, 40)
	plain.DrawString(4, 20, text, Black8)
	scaled.DrawStringScaled(4, 20, text, Black8, 1)

	for i := range plain.Pix {
		if plain.Pix[i] != scaled.Pix[i] {
			t.Fatalf("scale 1 differs from DrawString at pixel %d (%d,%d): %d vs %d",
				i, i%plain.W, i/plain.W, plain.Pix[i], scaled.Pix[i])
		}
	}

	// And a scale of zero or less draws nothing rather than dividing by it or looping
	// forever. The shell never asks, but fit() and centerIn() do arithmetic on the
	// scale and a zero reaching here should be inert.
	for _, s := range []int{0, -1, -8} {
		blank := NewSurface(64, 20)
		blank.DrawStringScaled(0, 10, text, Black8, s)
		if n := inked(blank); n != 0 {
			t.Errorf("scale %d painted %d pixels", s, n)
		}
		if w := StringWidthScaled(text, s); w != 0 {
			t.Errorf("StringWidthScaled at scale %d = %d, want 0", s, w)
		}
	}
}

// Every font pixel becomes a scale x scale block, so the ink count is exactly the
// square of the scale times the unmagnified count. A magnifier that rounded a row or
// dropped the last column would pass a "looks bigger" test and fail this one.
func TestScaledInkIsTheSquareOfTheScale(t *testing.T) {
	const text = "gW,é"         // a cap, a wide letter, a comma and a fold
	base := NewSurface(200, 40) // room for the descender and the accent
	base.DrawString(4, 24, text, Black8)
	want := inked(base)
	if want == 0 {
		t.Fatal("the unmagnified string painted nothing")
	}

	for _, scale := range []int{2, 3, 5} {
		big := NewSurface(200*scale, 40*scale)
		big.DrawStringScaled(4, 24*int16(scale), text, Black8, scale)
		if got := inked(big); got != want*scale*scale {
			t.Errorf("scale %d painted %d pixels, want %d (%d x %d^2)",
				scale, got, want*scale*scale, want, scale)
		}
	}
}

// The baseline stays put, which is what lets the shell align a magnified label with
// unmagnified text on the same row. The ascent scales *upwards* from it: a scale-2
// capital is fourteen rows tall and all fourteen are above the baseline.
func TestScaledBaselineStaysPut(t *testing.T) {
	for _, scale := range []int{1, 2, 3} {
		s := NewSurface(int(FontWide)*scale+4, 80)
		s.Fill(s.Bounds(), White8)
		s.DrawStringScaled(0, 40, "A", Black8, scale)
		top, bot := rowExtent(s)
		// 'A' occupies the seven rows at and above the baseline unmagnified
		// (TestBaselinePlacement), so magnified it occupies 7*scale of them and the
		// last is still the row above the baseline.
		wantBot := 40 - 1
		wantTop := wantBot - 7*scale + 1
		if top != wantTop || bot != wantBot {
			t.Errorf("scale %d 'A' at v=40 fills rows %d..%d, want %d..%d",
				scale, top, bot, wantTop, wantBot)
		}
	}
}

// StringWidthScaled must agree with what DrawStringScaled advances, or every centred
// and right-aligned string in the shell is off by the difference. Measuring the painted
// extent rather than trusting the arithmetic is the point: the two are separate
// functions and a fold makes one rune three cells.
func TestScaledWidthMatchesWhatIsPainted(t *testing.T) {
	for _, text := range []string{"A", "Load House...", "no artwork found", "École", "22 houses"} {
		for _, scale := range []int{1, 2, 4} {
			w := StringWidthScaled(text, scale)
			if want := StringWidth(text) * int16(scale); w != want {
				t.Fatalf("StringWidthScaled(%q, %d) = %d, want %d", text, scale, w, want)
			}
			s := NewSurface(int(w)+40, 40*scale)
			s.DrawStringScaled(8, 30*int16(scale), text, Black8, scale)
			right := lastInkedColumn(s)
			// The advance includes the cell's blank trailing column(s), so the last
			// painted column is inside the advance and within one cell of its end.
			if right >= 8+int(w) {
				t.Errorf("%q at scale %d paints out to column %d, past its %d-pixel advance",
					text, scale, right, w)
			}
			if right < 8+int(w)-int(FontWide)*scale {
				t.Errorf("%q at scale %d ends at column %d, more than a cell short of its %d-pixel advance",
					text, scale, right, w)
			}
		}
	}
}

// FillPatOrGray is the shell's dim. Two claims: it touches exactly the checkerboard's
// half of the rect, and it ORs rather than assigns -- which is what makes it a *dim* of
// the artwork underneath rather than a fill over it.
func TestFillPatOrGrayIsHalfTheRectAndORs(t *testing.T) {
	s := NewSurface(40, 30)
	s.Fill(s.Bounds(), 0x0F)
	r := SetRect(4, 6, 20, 18)
	s.FillPatOrGray(r, 0x30)

	var hit, miss int
	for y := 0; y < s.H; y++ {
		for x := 0; x < s.W; x++ {
			got := s.Pix[y*s.W+x]
			inside := int16(x) >= r.Left && int16(x) < r.Right &&
				int16(y) >= r.Top && int16(y) < r.Bottom
			switch {
			case inside && grayPatternSet(x, y):
				hit++
				if got != 0x3F { // 0x0F | 0x30
					t.Fatalf("(%d,%d) is %#02x, want %#02x -- the pattern must OR, not assign",
						x, y, got, 0x3F)
				}
			default:
				miss++
				if got != 0x0F {
					t.Fatalf("(%d,%d) is %#02x, want the untouched %#02x", x, y, got, 0x0F)
				}
			}
		}
	}
	if want := int(r.Right-r.Left) * int(r.Bottom-r.Top) / 2; hit != want {
		t.Errorf("dimmed %d pixels in a %dx%d rect, want %d -- half of it",
			hit, r.Right-r.Left, r.Bottom-r.Top, want)
	}
	if miss == 0 {
		t.Fatal("nothing was left alone")
	}

	// A rect off the edge clips rather than panicking, which every other filler here
	// promises too.
	s.FillPatOrGray(SetRect(-10, -10, 1000, 1000), 0x30)
	s.FillPatOrGray(SetRect(100, 100, 200, 200), 0x30)
}

// ---------------------------------------------------------------------------

func inked(s *Surface) int {
	n := 0
	for _, p := range s.Pix {
		if p == Black8 {
			n++
		}
	}
	return n
}

// lastInkedColumn is the rightmost column holding ink, or -1.
func lastInkedColumn(s *Surface) int {
	for x := s.W - 1; x >= 0; x-- {
		for y := 0; y < s.H; y++ {
			if s.Pix[y*s.W+x] == Black8 {
				return x
			}
		}
	}
	return -1
}
