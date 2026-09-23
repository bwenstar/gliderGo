package platform

// What a frame changed, so that a backend sends only that (docs/IMPROVEMENTS.md 2.76).
//
// A magnified window multiplies every byte of a frame by the square of its scale, and sending
// the frame is most of what a frame costs: at 4x a whole frame is 19.7 MB, and sending it took
// two thirds of a 24 ms frame on the development host. Most frames change very little. Over a
// 300-frame Slumberland run the median frame changed 17 of the 480 rows, and a title screen or
// a pause changes none. So a backend keeps the frame it sent last, compares, and expands and
// sends only the runs of rows that differ, each cut to the columns that changed in it.
//
// The game is not asked. Its dirty rectangles would do the same job at a finer grain, but they
// are the 1994 renderer's, a present that trusted them would inherit any rect the original
// forgot, and the shell's screens have none at all. Comparing a frame costs tens of
// microseconds.

import "bytes"

// A Span is a block of the framebuffer: rows Y0 to Y1-1 and columns X0 to X1-1, in framebuffer
// pixels, before any magnification.
type Span struct{ X0, Y0, X1, Y1 int }

// maxSpans is how many separate blocks one frame may send before it is sent as one block that
// covers them all. A frame that changed every other row would otherwise be hundreds of requests.
const maxSpans = 32

// Changes remembers the last frame a window was sent. The zero value reports the whole of the
// first frame.
type Changes struct {
	last  []byte // the frame, W*4 bytes a row with no padding
	w, h  int
	stale bool // set by All: the next Diff reports everything
	spans []Span
}

// All makes the next Diff report the whole frame. A backend calls it when what the window
// showed was lost: an expose, a map, WM_PAINT, a frame it did not send.
func (c *Changes) All() { c.stale = true }

// Diff reports what fb changed since the frame before, and remembers fb as the frame before
// the next one.
//
// It reports the whole frame the first time, after All, when fb is a different size from the
// last one, and when fb is too short for its own size (so that Expand is what says what is
// wrong with it). A frame that changed nothing reports nothing. The slice is reused by the next
// call.
func (c *Changes) Diff(fb *Framebuffer) []Span {
	c.spans = c.spans[:0]
	if fb.W <= 0 || fb.H <= 0 {
		return c.spans
	}
	row := fb.W * 4
	whole := Span{0, 0, fb.W, fb.H}
	if fb.Stride < row || len(fb.Pix) < (fb.H-1)*fb.Stride+row {
		c.stale = true
		return append(c.spans, whole)
	}
	if c.stale || c.w != fb.W || c.h != fb.H || len(c.last) != row*fb.H {
		if len(c.last) != row*fb.H {
			c.last = make([]byte, row*fb.H)
		}
		for y := 0; y < fb.H; y++ {
			copy(c.last[y*row:][:row], fb.Pix[y*fb.Stride:][:row])
		}
		c.w, c.h, c.stale = fb.W, fb.H, false
		return append(c.spans, whole)
	}

	open := false // the last span ends at the row above this one
	for y := 0; y < fb.H; y++ {
		now := fb.Pix[y*fb.Stride:][:row]
		was := c.last[y*row:][:row]
		if bytes.Equal(now, was) {
			open = false
			continue
		}
		x0, x1 := changedColumns(was, now)
		copy(was, now)
		if open {
			s := &c.spans[len(c.spans)-1]
			s.Y1 = y + 1
			s.X0, s.X1 = min(s.X0, x0), max(s.X1, x1)
			continue
		}
		c.spans = append(c.spans, Span{x0, y, x1, y + 1})
		open = true
	}

	if len(c.spans) > maxSpans {
		all := c.spans[0]
		for _, s := range c.spans[1:] {
			all.X0, all.X1 = min(all.X0, s.X0), max(all.X1, s.X1)
			all.Y1 = s.Y1
		}
		c.spans = append(c.spans[:0], all)
	}
	return c.spans
}

// changedColumns is the first pixel at which two rows differ and one past the last. The rows
// must differ.
func changedColumns(was, now []byte) (x0, x1 int) {
	i := 0
	for was[i] == now[i] {
		i++
	}
	j := len(now) - 1
	for was[j] == now[j] {
		j--
	}
	return i / 4, j/4 + 1
}
