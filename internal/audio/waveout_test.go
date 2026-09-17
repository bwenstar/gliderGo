package audio

// What can be checked about the Windows sink from a machine that is not Windows.
//
// waveout_windows.go cannot run here and neither can anything it calls, so this file tests the part
// of it that is arithmetic: two structure layouts that have to match a C header exactly, a table of
// error names, and the tuning numbers' relationships to each other. That is not the same as testing
// the sink -- see the caveat at the top of waveout_windows.go -- but it is the half of the work
// where a mistake is silent, and it runs on every platform in `go test ./...`.

import (
	"testing"
	"time"
	"unsafe"
)

// TestWaveHdrLayout pins WAVEHDR against the Microsoft definition.
//
// This is the structure the driver reads and writes, so a field at the wrong offset is not a
// compile error or a crash: it is the device playing the buffer length as if it were audio, or
// WHDR_DONE being read out of dwLoops and a block that never comes back. The numbers below were
// taken from mmeapi.h's declaration and the platform's alignment rules, and are asserted rather
// than assumed because nobody here can hear the failure.
func TestWaveHdrLayout(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skipf("layout asserted for 64-bit; this host has %d-bit pointers", unsafe.Sizeof(uintptr(0))*8)
	}
	var h waveHdr
	for _, c := range []struct {
		field string
		got   uintptr
		want  uintptr
	}{
		{"lpData", unsafe.Offsetof(h.data), 0},
		{"dwBufferLength", unsafe.Offsetof(h.bufferLength), 8},
		{"dwBytesRecorded", unsafe.Offsetof(h.bytesRecorded), 12},
		{"dwUser", unsafe.Offsetof(h.user), 16},
		{"dwFlags", unsafe.Offsetof(h.flags), 24},
		{"dwLoops", unsafe.Offsetof(h.loops), 28},
		{"lpNext", unsafe.Offsetof(h.next), 32},
		{"reserved", unsafe.Offsetof(h.reserved), 40},
	} {
		if c.got != c.want {
			t.Errorf("WAVEHDR.%s at offset %d, want %d", c.field, c.got, c.want)
		}
	}
	if got := unsafe.Sizeof(h); got != 48 {
		t.Errorf("sizeof(WAVEHDR) = %d, want 48", got)
	}

	// dwFlags is read with sync/atomic while the driver writes it, which on every architecture
	// Go supports requires the word to be 4-byte aligned. It is, at offset 24 of an 8-aligned
	// array element -- and this asserts it so that a future field cannot quietly move it.
	if unsafe.Offsetof(h.flags)%4 != 0 {
		t.Errorf("dwFlags at offset %d is not 4-byte aligned; the atomic load in reclaim needs it to be", unsafe.Offsetof(h.flags))
	}
}

// TestWaveFormatExLayout pins WAVEFORMATEX, where Go and C legitimately disagree about the size.
//
// C packs it to 18 bytes; Go rounds the struct up to 20 because its alignment is 4 and its last
// field is a uint16. That is harmless only because cbSize is zero, which tells Windows there is no
// extra data after the 18th byte -- so what has to hold is that every offset matches and that the
// declared fields end at 18. Both are checked here, on both word sizes.
func TestWaveFormatExLayout(t *testing.T) {
	var f waveFormatEx
	for _, c := range []struct {
		field string
		got   uintptr
		want  uintptr
	}{
		{"wFormatTag", unsafe.Offsetof(f.formatTag), 0},
		{"nChannels", unsafe.Offsetof(f.channels), 2},
		{"nSamplesPerSec", unsafe.Offsetof(f.samplesPerSec), 4},
		{"nAvgBytesPerSec", unsafe.Offsetof(f.avgBytesPerSec), 8},
		{"nBlockAlign", unsafe.Offsetof(f.blockAlign), 12},
		{"wBitsPerSample", unsafe.Offsetof(f.bitsPerSample), 14},
		{"cbSize", unsafe.Offsetof(f.cbSize), 16},
	} {
		if c.got != c.want {
			t.Errorf("WAVEFORMATEX.%s at offset %d, want %d", c.field, c.got, c.want)
		}
	}
	if end := unsafe.Offsetof(f.cbSize) + unsafe.Sizeof(f.cbSize); end != 18 {
		t.Errorf("declared fields end at %d, want the C structure's 18", end)
	}
	if f.cbSize != 0 {
		t.Errorf("cbSize = %d; the zero value has to mean no extra data, or Windows reads Go's padding", f.cbSize)
	}
}

// TestWaveFormatMatchesTheMix is the one test here that is about the sound rather than the ABI: the
// format handed to the device has to describe the bytes encode produces, or the game plays at the
// wrong speed or as noise. Three files declare this stream (bank.go's rate, wav.go's header, and
// the sink) and this is the assertion that they agree.
func TestWaveFormatMatchesTheMix(t *testing.T) {
	f := waveFormatEx{
		formatTag:      waveFormatPCM,
		channels:       1,
		samplesPerSec:  Rate,
		avgBytesPerSec: Rate * 2,
		blockAlign:     2,
		bitsPerSample:  16,
	}
	if want := uint16(f.channels * f.bitsPerSample / 8); f.blockAlign != want {
		t.Errorf("nBlockAlign = %d, want %d for %d channel(s) of %d bits", f.blockAlign, want, f.channels, f.bitsPerSample)
	}
	if want := f.samplesPerSec * uint32(f.blockAlign); f.avgBytesPerSec != want {
		t.Errorf("nAvgBytesPerSec = %d, want %d", f.avgBytesPerSec, want)
	}
	if got := len(encode(nil, make([]int16, 100))); got != 100*int(f.blockAlign) {
		t.Errorf("encode produced %d bytes for 100 samples; the format says %d", got, 100*int(f.blockAlign))
	}
}

// TestMMName checks that an MMRESULT reads as something greppable, named or not. It is the only
// diagnostic anybody on a machine nobody here can reach will have.
func TestMMName(t *testing.T) {
	for _, c := range []struct {
		code uint32
		want string
	}{
		{0, "MMSYSERR_NOERROR"},
		{4, "MMSYSERR_ALLOCATED"},   // the device is somebody else's: the interesting one
		{33, "WAVERR_STILLPLAYING"}, // what a botched teardown produces
		{999, "MMRESULT 999"},
	} {
		if got := mmName(c.code); got != c.want {
			t.Errorf("mmName(%d) = %q, want %q", c.code, got, c.want)
		}
	}
}

// TestWaveTuning checks the numbers against each other rather than against a reference, because
// there is no reference: they are a compromise between latency and underruns that only a listener
// can settle. What can be checked is that they are still consistent after somebody edits one, which
// is where the damage would be -- a prefill as deep as the block rotation would deadlock the first
// write, and a poll interval coarser than a frame would turn every block into a stall.
func TestWaveTuning(t *testing.T) {
	// Integer arithmetic on purpose: these are exact constant expressions, and a frame is
	// 33.25 ms because the game runs at 30.075 fps.
	const frame = SamplesPerFrame * time.Second / Rate

	if waveBlockSamples != maxCatchUp {
		t.Errorf("waveBlockSamples = %d, want maxCatchUp (%d) so that one Write is one block", waveBlockSamples, maxCatchUp)
	}
	if waveBlockBytes != waveBlockSamples*2 {
		t.Errorf("waveBlockBytes = %d, want %d for 16-bit samples", waveBlockBytes, waveBlockSamples*2)
	}
	if wavePrefill < 1 || wavePrefill >= waveBlocks {
		t.Errorf("wavePrefill = %d, want between 1 and waveBlocks-1 (%d): no cushion starves, a full one leaves nothing to fill", wavePrefill, waveBlocks-1)
	}
	if wavePrefillLength > waveBlockSamples {
		t.Errorf("wavePrefillLength = %d samples, which does not fit in a %d-sample block", wavePrefillLength, waveBlockSamples)
	}
	if waveDepth < 1 {
		t.Errorf("waveDepth = %d, want at least one queued block", waveDepth)
	}

	if 4*wavePoll >= frame {
		t.Errorf("wavePoll = %v, want well under a frame (%v): the writer asks about completions this often", wavePoll, frame)
	}
	// The device can be holding every block at once, so the stall deadline has to be longer
	// than the time it takes to play them all -- otherwise a full, healthy queue looks like a
	// dead driver.
	const full = waveBlocks * waveBlockSamples * time.Second / Rate
	if waveStall <= full {
		t.Errorf("waveStall = %v, want more than the %v a full device holds", waveStall, full)
	}
	if waveDrain < frame || waveDrain > time.Second {
		t.Errorf("waveDrain = %v, want between a frame (%v) and a second: below that it clicks, above it reads as a hang", waveDrain, frame)
	}
}
