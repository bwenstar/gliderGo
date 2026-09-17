package audio

// The shapes, the numbers and the error names the Windows sink is built out of, in a file with no
// build tag.
//
// Deliberately no tag, for the reason internal/platform/win32/keys.go carries none: the port is
// written on an airgapped Linux box that cannot run a line of waveout_windows.go, so everything
// about it that can be checked without Windows is put where a `go test ./internal/audio` on this
// machine will check it. Two struct layouts and a table of error codes is most of what there is to
// get wrong on paper -- a field at the wrong offset is a device that plays the buffer length as
// audio -- and waveout_test.go pins all of it.

import (
	"strconv"
	"time"
	"unsafe"
)

// The tuning, all of it, in one place.
//
// WHAT SETS THESE NUMBERS. The device consumes samples at exactly its own crystal rate and the
// game produces them at what the wall clock says is due (pump.go's ClockTick), so the two agree on
// average and disagree constantly in the small. Everything below is slack against that disagreement:
//
//   - waveBlocks and waveDepth are how much slack there is. Together they are how far the game may
//     run ahead of the device before samples start being thrown away, which is the same bargain
//     Pipe strikes with pipeDepth and is struck the same way round -- a sink that blocks stops the
//     simulation, and a stutter is a worse bug than a mix that skips a frame.
//   - wavePrefill is how far the device runs *behind*, on purpose. A real-time producer feeding a
//     device with no cushion starves on the first hiccup, so the sink writes a tenth of a second of
//     silence before the first mixed sample and stays that far behind for the rest of the session.
//     It is the whole difference between sound and sound with clicks in it.
//   - wavePoll is how often the writer goroutine asks the device whether a block has finished. A
//     block is 33 ms of audio, so 4 ms is fine grain, and Go's runtime on Windows 10 1803 and
//     later sleeps on a high-resolution timer (runtime/os_windows.go's initHighResTimer), so the
//     ask really does land in 4 ms rather than at the 15.6 ms tick an older Windows would round it
//     to. Being wrong about that costs latency, not correctness.
const (
	waveBlocks = 8

	// waveBlockSamples is the most one block carries: the pump's own catch-up cap, so that any
	// single Write fits in one block and Write never has to split except in a caller that
	// ignores maxCatchUp.
	waveBlockSamples = maxCatchUp
	waveBlockBytes   = waveBlockSamples * 2

	// waveDepth is how many filled blocks may wait for the writer goroutine. pipeDepth, for
	// pipeDepth's reasons.
	waveDepth = 8

	// wavePrefill blocks of silence, each one frame long, go in before the first mixed sample:
	// 3 x 33.25 ms, a tenth of a second of cushion. Two would be tight against a garbage
	// collection and eight would be a fifth of a second of lag on a rubber band.
	wavePrefill       = 3
	wavePrefillLength = SamplesPerFrame

	wavePoll = 4 * time.Millisecond

	// waveStall is how long the writer goroutine waits for the device to hand a block back
	// before deciding the device is gone. The device can hold waveBlocks blocks, each at most
	// waveBlockSamples long, which is 1.06 seconds of audio; three times that is unambiguous.
	// Reaching it fails the sink for good, and a failed sink is silence and not a crash.
	waveStall = 3 * time.Second

	// waveDrain is how long Close waits for the device to play out what it already has. It is
	// the difference between a clean last note and a click, and it is capped low because a game
	// that takes a second to quit reads as a game that has hung.
	waveDrain = 500 * time.Millisecond
)

// waveFormatEx is WAVEFORMATEX (mmreg.h): what the stream is, told to the device once at open.
//
// Mono signed 16-bit little-endian at Rate, which is what encode produces and what the WAV writer
// declares -- one description of the same bytes in three places, and the reason a `-wav` recording
// and what came out of the speakers are the same stream.
//
// One layout note. C packs this to 18 bytes and Go rounds it up to 20, because the struct's
// alignment is 4 and its last field is a uint16. The *offsets* are identical, which is what
// matters: cbSize is zero, so Windows is told there is no extra data after the 18th byte and has
// no reason to read the two Go added.
type waveFormatEx struct {
	formatTag      uint16
	channels       uint16
	samplesPerSec  uint32
	avgBytesPerSec uint32
	blockAlign     uint16
	bitsPerSample  uint16
	cbSize         uint16
}

// waveHdr is WAVEHDR (mmeapi.h): one block of samples handed to the device.
//
// This is the structure Windows writes to from another thread -- it sets WHDR_DONE in flags when it
// has finished with the block -- so it is the one piece of memory in the port with two writers, and
// the reason acquire reads flags with sync/atomic rather than plainly. Everything else here is
// written by this side only, and only while the device does not hold the block.
//
// Field for field the Microsoft definition, and the offsets on both 64-bit Windows targets are
// 0, 8, 12, 16, 24, 28, 32, 40 for a total of 48. Go arrives at the same numbers on its own, which
// waveout_test.go checks rather than trusts.
//
// lpData is an unsafe.Pointer and lpNext is a uintptr, and the difference is deliberate: the first
// is a pointer this side sets and the garbage collector may as well know it, while the second is
// filled in by the driver with a pointer into the driver's own world, which the collector must not
// be told to follow. Both are one word wide either way.
type waveHdr struct {
	data          unsafe.Pointer // LPSTR lpData -- the samples
	bufferLength  uint32
	bytesRecorded uint32 // recording only; zero here
	user          uintptr
	flags         uint32
	loops         uint32
	next          uintptr // the driver's own queue pointer; not ours to touch
	reserved      uintptr
}

// The constants these six calls need, from mmeapi.h and mmsyscom.h.
const (
	waveFormatPCM = 1

	// waveMapper is WAVE_MAPPER, (UINT)-1, and it is the whole reason this sink can declare an
	// odd sample rate. The mapper picks the default output device *and* converts the format to
	// whatever the hardware wants, so 22255 Hz mono -- 244800/11 rounded, see bank.go -- is
	// resampled by Windows instead of being refused with WAVERR_BADFORMAT the way a directly
	// opened device could refuse it.
	waveMapper = 0xFFFFFFFF

	// No callback and no event: the writer goroutine polls WHDR_DONE. A callback would run Go
	// code on a driver thread through syscall.NewCallback with the usual don't-block-in-here
	// rules, and an event needs a reset-then-recheck dance to avoid losing a wakeup and an
	// unblocking trick to avoid hanging Close. Polling a flag every 4 ms can do neither.
	callbackNull = 0x00000000

	whdrDone = 0x00000001

	mmsyserrNoError = 0
)

// mmErrors is the MMSYSERR/WAVERR names, for the one thing this file can do for somebody on a
// machine nobody here can reach: put a greppable name in the message rather than a bare number.
//
// The general errors and the four wave ones. The gap between 20 and 32 is real -- those codes
// belong to the MMIO and joystick calls, which nothing here makes -- and an unnamed code still
// prints, as a number.
var mmErrors = map[uint32]string{
	0:  "MMSYSERR_NOERROR",
	1:  "MMSYSERR_ERROR",
	2:  "MMSYSERR_BADDEVICEID",
	3:  "MMSYSERR_NOTENABLED",
	4:  "MMSYSERR_ALLOCATED",
	5:  "MMSYSERR_INVALHANDLE",
	6:  "MMSYSERR_NODRIVER",
	7:  "MMSYSERR_NOMEM",
	8:  "MMSYSERR_NOTSUPPORTED",
	9:  "MMSYSERR_BADERRNUM",
	10: "MMSYSERR_INVALFLAG",
	11: "MMSYSERR_INVALPARAM",
	12: "MMSYSERR_HANDLEBUSY",
	32: "WAVERR_BADFORMAT",
	33: "WAVERR_STILLPLAYING",
	34: "WAVERR_UNPREPARED",
	35: "WAVERR_SYNC",
}

// mmName is how an MMRESULT reads in an error message.
func mmName(code uint32) string {
	if name, ok := mmErrors[code]; ok {
		return name
	}
	return "MMRESULT " + strconv.Itoa(int(code))
}
