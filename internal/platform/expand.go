package platform

// Nearest-neighbour magnification, shared by every backend that owns a scaled upload buffer.
//
// It lives here rather than in a backend because there are now two of them doing the identical
// thing -- x11 into a malloc'd XImage, win32 into a Go DIB -- and because of where the two can
// be tested. The x11 backend needs libX11 and a display, the win32 backend needs Windows, and
// the machine this port is written on has neither of the latter. A plain function over two byte
// slices needs no window at all, so the arithmetic that decides where every pixel lands is
// exercised by `go test ./internal/platform` on any host, including the one that cannot run the
// backend it belongs to.
//
// Nearest-neighbour is not a default chosen for speed. It is the only resampling that keeps
// 1994 pixel art honest: every source pixel becomes an exact scale x scale block of the same
// colour, so a doubled window is the original image and not a blurred guess at it.

import "fmt"

// Expand copies fb into dst, magnifying by an integer scale.
//
// dst is row-major with dstStride bytes per row and the same BGRX pixel layout as fb, which is
// why this is a copy and never a conversion. dstStride may exceed the pixels a row needs: a
// backend whose surface is padded for alignment keeps its padding, because only the
// fb.W*scale*4 bytes at the start of each row are written.
//
// Every size relationship is checked before anything is written, and a bad one is returned
// rather than panicked on. The reason is the call site: this runs inside Present, once a frame,
// from a game loop that treats a Present error as a dead window and shuts down cleanly. A
// panic out of a backend would instead take the process down mid-frame with a stack trace in
// place of a message, and the sizes involved come from a window manager and from cfg, not from
// this package.
func Expand(dst []byte, dstStride int, fb *Framebuffer, scale int) error {
	if scale < 1 {
		scale = 1
	}
	if fb.W < 0 || fb.H < 0 {
		return fmt.Errorf("platform: framebuffer size %dx%d is negative", fb.W, fb.H)
	}
	if fb.Stride < fb.W*4 {
		return fmt.Errorf("platform: framebuffer stride is %d bytes, too short for %d pixels",
			fb.Stride, fb.W)
	}
	rowBytes := fb.W * scale * 4
	if dstStride < rowBytes {
		return fmt.Errorf("platform: destination stride is %d bytes, too short for %d pixels at scale %d",
			dstStride, fb.W, scale)
	}
	if fb.W == 0 || fb.H == 0 {
		return nil
	}
	if need := (fb.H-1)*fb.Stride + fb.W*4; len(fb.Pix) < need {
		return fmt.Errorf("platform: framebuffer holds %d bytes, needs %d for %dx%d",
			len(fb.Pix), need, fb.W, fb.H)
	}
	if need := (fb.H*scale-1)*dstStride + rowBytes; len(dst) < need {
		return fmt.Errorf("platform: destination holds %d bytes, needs %d for %dx%d at scale %d",
			len(dst), need, fb.W, fb.H, scale)
	}

	for y := 0; y < fb.H; y++ {
		src := fb.Pix[y*fb.Stride : y*fb.Stride+fb.W*4]
		row := dst[y*scale*dstStride:][:rowBytes]
		if scale == 1 {
			copy(row, src)
			continue
		}
		for x := 0; x < fb.W; x++ {
			px := src[x*4 : x*4+4]
			for s := 0; s < scale; s++ {
				copy(row[(x*scale+s)*4:], px)
			}
		}
		// The row is already expanded, so the remaining scale-1 lines of the block are a
		// straight copy of it rather than the same per-pixel loop run again.
		for s := 1; s < scale; s++ {
			copy(dst[(y*scale+s)*dstStride:][:rowBytes], row)
		}
	}
	return nil
}
