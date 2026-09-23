package render

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/png"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
)

// pngHeader is a PNG that declares w x h RGB pixels and has none: the signature, an
// IHDR, an IDAT holding an empty zlib stream, and an IEND. It is what png.DecodeConfig
// reads, and all a picture needs in order to have png.Decode allocate for every pixel
// before it finds there are none.
func pngHeader(w, h uint32) []byte {
	var b bytes.Buffer
	b.WriteString("\x89PNG\r\n\x1a\n")
	chunk := func(kind string, data []byte) {
		binary.Write(&b, binary.BigEndian, uint32(len(data)))
		body := append([]byte(kind), data...)
		b.Write(body)
		binary.Write(&b, binary.BigEndian, crc32.ChecksumIEEE(body))
	}
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], w)
	binary.BigEndian.PutUint32(ihdr[4:], h)
	ihdr[8], ihdr[9] = 8, 2 // 8 bits a sample, RGB
	chunk("IHDR", ihdr)
	var idat bytes.Buffer
	zlib.NewWriter(&idat).Close()
	chunk("IDAT", idat.Bytes())
	chunk("IEND", nil)
	return b.Bytes()
}

// A picture that declares itself 16384x16384 is refused from its header, before a byte
// of it is decoded (docs/IMPROVEMENTS.md 4.36). Reproduced before the fix: a file of a
// few hundred kilobytes that said this cost 0.5 to 2.1 GB and a nine-second stall. Both
// loaders are checked, because -art reaches one and -levels and -houseart the other.
func TestAPictureTooLargeToBeArtIsRefusedFromItsHeader(t *testing.T) {
	fsys := fstest.MapFS{"huge.png": &fstest.MapFile{Data: pngHeader(16384, 16384)}}
	for _, load := range []struct {
		name string
		do   func(a *Assets) *Surface
	}{
		{"the art tree", func(a *Assets) *Surface { return a.load("huge.png") }},
		{"a house's pictures", func(a *Assets) *Surface { return a.loadHousePict(fsys, "test", "huge.png") }},
	} {
		a := NewAssets(fsys)
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		s := load.do(a)
		runtime.ReadMemStats(&after)
		if s != nil {
			t.Fatalf("%s: a 16384x16384 picture loaded", load.name)
		}
		if err := a.Err(); err == nil || !strings.Contains(err.Error(), "16384x16384") {
			t.Errorf("%s: err = %v, want one that names the size it refused", load.name, err)
		}
		if grew := after.TotalAlloc - before.TotalAlloc; grew > 1<<20 {
			t.Errorf("%s: refusing it allocated %d bytes, so it was decoded first", load.name, grew)
		}
	}
}

// Each limit, at its edge. The header alone is enough to tell a refusal from a picture
// that got past the check, because one that got past fails later, for having no pixels.
func TestThePictureLimitsAreWhereTheySay(t *testing.T) {
	for _, tc := range []struct {
		w, h    uint32
		refused bool
	}{
		{maxPictSide, 1, false},
		{maxPictSide + 1, 1, true},
		{1, maxPictSide + 1, true},
		{2048, 2048, false}, // exactly maxPictPixels
		{2049, 2048, true},
		{4096, 1025, true}, // within the side cap and over the pixel cap
	} {
		a := NewAssets(fstest.MapFS{"p.png": &fstest.MapFile{Data: pngHeader(tc.w, tc.h)}})
		a.load("p.png")
		err := a.Err()
		if err == nil {
			t.Fatalf("%dx%d: a picture with no pixels decoded", tc.w, tc.h)
		}
		if got := strings.Contains(err.Error(), "larger than a picture can be"); got != tc.refused {
			t.Errorf("%dx%d: refused = %v, want %v (%v)", tc.w, tc.h, got, tc.refused, err)
		}
	}
}

// And a picture inside the limits still comes through the second open whole: the
// header check reads from one open and the decode from another.
func TestAPictureInsideTheLimitsStillLoads(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, maxPictSide, 2))
	for i := range img.Pix {
		img.Pix[i] = 0xFF // white and opaque, which is palette index 0
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	a := NewAssets(fstest.MapFS{"wide.png": &fstest.MapFile{Data: b.Bytes()}})
	s := a.load("wide.png")
	if err := a.Err(); err != nil || s == nil || s.W != maxPictSide || s.H != 2 {
		t.Fatalf("a %dx2 picture: surface %v, err %v", maxPictSide, s, err)
	}
}
