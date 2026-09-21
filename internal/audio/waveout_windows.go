//go:build windows

package audio

// The native Windows output: winmm's waveOut, so an unzipped .exe makes a noise on its own.
//
// HONEST CAVEAT, READ FIRST. This file was written on an airgapped Linux host with no Windows to
// test on, for the reason internal/platform/win32 gives at the same length, and it still cannot be
// exercised from there. `GOOS=windows go build` and `go vet` pass for amd64 and arm64, the two
// struct layouts and the error table are unit-tested on Linux (see waveout.go, which carries no
// build tag for exactly that reason), and every call below is one of seven with a documented
// MMRESULT that is checked.
//
// It has been run, once, on Windows Server 2025 (amd64) against a real output device: `-audio list`
// found waveout, the sink opened, and across the four runs that used the device rather than a WAV
// file the game asked it for 73 sounds and it played 73, refusing none. docs/windows-first-run.md
// is the write-up.
//
// Two things that run did not settle. It was one device on one machine, so a driver that refuses
// this format is still an open question -- `-audio ffplay` takes the external-player path (sink.go)
// and `-wav out.wav` writes a file, so there are two ways out. And the drop counter is not a fault
// when the game runs unpaced: `-bench` mixes far faster than 22 kHz, so a bench log reporting tens
// of thousands of samples dropped by waveout is the sink doing its job. A *paced* run that drops
// anything is the thing to look at.
//
// WHY THIS EXISTS. sink.go pipes the mix to a command-line player because that is the right answer
// on Linux and needs no driver. Windows ships none of the five, so a player who downloaded a
// release archive got silence and an installation instruction -- which for a game is the same thing
// as a bug. waveOut is what makes the answer "it just plays": it is in winmm.dll on every Windows
// since 3.1, it is reachable with pure syscall and no cgo, and on Windows 10 and 11 it is a shim
// over WASAPI in shared mode, so the game mixes with everything else on the desktop rather than
// seizing the device. docs/IMPROVEMENTS.md 2.48 is what this closes.
//
// WHY waveOut AND NOT WASAPI. WASAPI is COM: five interfaces, a class factory, IIDs to get right by
// hand, reference counting across a syscall boundary, and an event-driven render thread. Every one
// of those is a thing to get wrong in code nobody here can run, and the reward would be perhaps
// twenty milliseconds of latency on a game whose loudest event is a glider hitting a fan. waveOut
// is seven functions and a struct. If this port ever needs sample-accurate audio it will need a
// different sink and this comment will be the reason it did not start there.
//
// THE ONE THING THAT IS NOT ORDINARY GO. waveOutWrite hands the device a pointer and returns
// immediately; the device reads that memory afterwards, from its own thread, and writes a flag back
// into the header when it has finished. Go's rules for pointers handed to foreign code do not cover
// that case -- they assume the call is done with the memory by the time it returns -- so the blocks
// and their headers are package-level arrays rather than anything allocated per sink. A global is
// the one allocation whose address the language cannot ever change: the linker assigns it, so no
// collector can move it and none can free it, which is exactly the promise waveOutWrite needs and
// one the heap does not make. Two other spellings were tried first and are worse. runtime.Pinner
// (Go 1.21) pins heap memory, but a Pinner collected without Unpin panics the whole process by
// design (runtime/pinner.go), which turns a bookkeeping slip in a shipped game into a crash.
// VirtualAlloc'd memory is outside Go's world entirely and needs no pin, but reading it back needs a
// pointer made out of an integer, which is the one thing `go vet`'s unsafeptr check exists to object
// to -- and on the general case it is right to. Arrays need neither trick. The cost is that there is
// one set of blocks, so one WaveOut at a time, which waveInUse says out loud.

import (
	"fmt"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

// The blocks the device plays out of, and the headers that describe them. See the note above for why
// these are package-level and not fields: it is the only way to hand a pointer to a driver that
// keeps it. 47 KB of zeroed BSS in a Windows build, and nothing at all in any other.
var (
	waveHdrs [waveBlocks]waveHdr
	waveBufs [waveBlocks][waveBlockBytes]byte

	// waveInUse is held for the lifetime of an open WaveOut, because the arrays above cannot be
	// shared. The game opens one sink and closes it at exit, so this guards against a future
	// mistake rather than anything that happens today -- and an error at the second open is a
	// great deal easier to understand than two sinks writing into the same eight blocks.
	waveInUse atomic.Bool
)

// The one library loaded by bare name, which is safe because kernel32 is on Windows' KnownDLLs list
// and so is resolved before any path search -- the same reasoning internal/platform/win32 sets out
// for user32, gdi32 and kernel32, and the same reasoning says winmm cannot be loaded that way.
var (
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procGetSystemDirectoryW = kernel32.NewProc("GetSystemDirectoryW")

	winmmOnce sync.Once
	winmmErr  error

	procWaveOutGetNumDevs      *syscall.Proc
	procWaveOutOpen            *syscall.Proc
	procWaveOutClose           *syscall.Proc
	procWaveOutPrepareHeader   *syscall.Proc
	procWaveOutUnprepareHeader *syscall.Proc
	procWaveOutWrite           *syscall.Proc
	procWaveOutReset           *syscall.Proc
)

// hdrSize is the cbwh argument every prepare, write and unprepare takes. Windows uses it to tell
// WAVEHDR versions apart, so it has to be the real size of the structure and not a literal.
var hdrSize = unsafe.Sizeof(waveHdr{})

// loadWinmm resolves the library and the seven entry points, once, on first use.
//
// **winmm is not a KnownDLL**, and it is not in Go's own system-DLL set either (verified in
// internal/syscall/windows/sysdll: syscall.LoadDLL pins a name to System32 only if the standard
// library itself uses it, and the standard library does not use winmm outside the runtime's timer
// code, which loads it by handle). Loading it by bare name would search the directory the
// executable was started from first, and a release archive tells the player to run the binary from
// the directory they unpacked it into -- which is exactly the arrangement a planted winmm.dll needs.
// So the system directory is asked for and the library is loaded by absolute path.
//
// Nothing is loaded at all by a build that never opens audio -- `-sound=false`, `-wav` on its own,
// every test in this package -- which is the property syscall.NewLazyDLL gives the display backend,
// arrived at here with a sync.Once because the path has to be computed before the load.
func loadWinmm() error {
	winmmOnce.Do(func() {
		dir, err := systemDirectory()
		if err != nil {
			winmmErr = err
			return
		}
		dll, err := syscall.LoadDLL(dir + `\winmm.dll`)
		if err != nil {
			winmmErr = fmt.Errorf("audio: %w", err)
			return
		}
		for _, p := range []struct {
			name string
			dst  **syscall.Proc
		}{
			{"waveOutGetNumDevs", &procWaveOutGetNumDevs},
			{"waveOutOpen", &procWaveOutOpen},
			{"waveOutClose", &procWaveOutClose},
			{"waveOutPrepareHeader", &procWaveOutPrepareHeader},
			{"waveOutUnprepareHeader", &procWaveOutUnprepareHeader},
			{"waveOutWrite", &procWaveOutWrite},
			{"waveOutReset", &procWaveOutReset},
		} {
			proc, err := dll.FindProc(p.name)
			if err != nil {
				winmmErr = fmt.Errorf("audio: %w", err)
				return
			}
			*p.dst = proc
		}
	})
	return winmmErr
}

// systemDirectory is C:\Windows\System32, or whatever this installation calls it -- and on an arm64
// Windows it is still System32, holding arm64 binaries, which is why the answer is asked for rather
// than assembled out of an environment variable.
func systemDirectory() (string, error) {
	if err := procGetSystemDirectoryW.Find(); err != nil {
		return "", fmt.Errorf("audio: %w", err)
	}
	var buf [syscall.MAX_PATH]uint16
	// On success the return is the length in UTF-16 units without the terminator; on failure it
	// is zero, and on too small a buffer it is the size needed *with* the terminator, so one
	// comparison covers both.
	n, _, err := procGetSystemDirectoryW.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 || n >= uintptr(len(buf)) {
		return "", fmt.Errorf("audio: GetSystemDirectoryW failed: %v", err)
	}
	return syscall.UTF16ToString(buf[:n]), nil
}

// deviceName is what `-audio list` calls this output and what `-audio waveout` insists on. It names
// the API rather than the platform, so that a WASAPI or XAudio2 sink could be offered beside it
// instead of replacing it.
const deviceName = "waveout"

// haveDevice reports whether this machine has an output device at all, for `-audio list`.
//
// A Windows box with no sound card -- a server, a virtual machine, most CI runners -- has winmm and
// zero devices, and offering "waveout" to somebody there would be a lie they would spend an evening
// on.
func haveDevice() bool {
	if loadWinmm() != nil {
		return false
	}
	n, _, _ := procWaveOutGetNumDevs.Call()
	return n > 0
}

func openDevice() (Stream, error) {
	w, err := openWaveOut()
	if err != nil {
		return nil, err
	}
	return w, nil
}

// WaveOut is the native Windows sink.
//
// Its shape is Pipe's on purpose, down to the two channels and the one goroutine: Write encodes and
// enqueues and never blocks, a full queue is dropped and counted rather than waited on, and all the
// blocking lives on a goroutine that touches nothing but bytes. What differs is the far end. An
// external player has a buffer of its own and hides the device from this process; here there is no
// player, so this file owns the two things the player was doing -- a rotation of blocks the device
// reads directly, and a cushion of silence ahead of the first sample so that the device does not
// starve on the first hiccup.
type WaveOut struct {
	dev uintptr // HWAVEOUT

	// busy[i] says the device has waveHdrs[i] and has not finished with it. Only the writer
	// goroutine reads or writes this (and Close, after that goroutine has stopped), so it needs
	// no lock; the *device's* half of the same fact is the WHDR_DONE bit in waveHdrs[i].flags,
	// which is written from a driver thread and is read with sync/atomic for that reason.
	busy    [waveBlocks]bool
	playing bool // the prefill is in, so an idle device now means the player heard a gap

	free  chan []byte
	queue chan []byte

	dropped   atomic.Int64
	written   atomic.Int64
	underruns atomic.Int64
	failed    atomic.Pointer[error]

	closed    atomic.Bool
	closeOnce sync.Once
	done      chan struct{}
}

// openWaveOut opens the default output device and primes it.
//
// The failure paths matter more here than the success one, because two of them are ordinary: a
// machine with no sound card at all, and a device some other program holds exclusively
// (MMSYSERR_ALLOCATED). Both come back as an error and Open (device.go) goes on to the external
// players, so a Windows box with FFmpeg installed and a busy device still makes a noise.
func openWaveOut() (*WaveOut, error) {
	if err := loadWinmm(); err != nil {
		// Not a device failure: winmm is missing or unreadable, which no fallback can fix
		// and which the caller should read as "there is nothing native here".
		return nil, fmt.Errorf("%w: %v", errNoDevice, err)
	}
	if n, _, _ := procWaveOutGetNumDevs.Call(); n == 0 {
		return nil, fmt.Errorf("%w: Windows reports no audio output device", errNoDevice)
	}
	if !waveInUse.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("audio: the %s device is already open in this process", deviceName)
	}

	// waveOutOpen copies the format and fills in the handle before it returns, so both of these
	// may be locals. The memory the device *keeps* is the two arrays at the top of the file.
	format := waveFormatEx{
		formatTag:      waveFormatPCM,
		channels:       1,
		samplesPerSec:  Rate,
		avgBytesPerSec: Rate * 2,
		blockAlign:     2,
		bitsPerSample:  16,
	}
	var dev uintptr
	r, _, _ := procWaveOutOpen.Call(
		uintptr(unsafe.Pointer(&dev)),
		waveMapper,
		uintptr(unsafe.Pointer(&format)),
		0, 0, callbackNull)
	if r != mmsyserrNoError {
		waveInUse.Store(false)
		return nil, fmt.Errorf("audio: waveOutOpen: %s", mmName(uint32(r)))
	}

	w := &WaveOut{
		dev:   dev,
		free:  make(chan []byte, waveDepth),
		queue: make(chan []byte, waveDepth),
		done:  make(chan struct{}),
	}
	// The one field of a header that is set once and never again: a block's samples do not move.
	for i := range waveHdrs {
		waveHdrs[i] = waveHdr{data: unsafe.Pointer(&waveBufs[i][0])}
	}

	// The cushion: wavePrefill blocks of silence before the first mixed sample. The blocks are
	// cleared rather than assumed empty, because although they are zero at process start, a
	// second open in the same process would otherwise replay a tenth of a second of whatever the
	// last session ended on.
	for i := 0; i < wavePrefill; i++ {
		clear(waveBufs[i][:wavePrefillLength*2])
		if err := w.send(i, wavePrefillLength*2); err != nil {
			w.shutdown(nil)
			return nil, err
		}
	}
	w.playing = true

	go w.pump()
	return w, nil
}

// Write encodes and enqueues, dropping the block if the queue is full. Pipe.Write, and the same
// bargain: an audio sink that blocks stops the simulation.
//
// The split is for a caller that hands over more than maxCatchUp samples at once, which the pump
// never does. Without it a long Write would be silently truncated to one block.
func (w *WaveOut) Write(samples []int16) error {
	for len(samples) > 0 {
		n := len(samples)
		if n > waveBlockSamples {
			n = waveBlockSamples
		}
		w.enqueue(samples[:n])
		samples = samples[n:]
	}
	return nil
}

func (w *WaveOut) enqueue(samples []int16) {
	if len(samples) == 0 {
		return
	}
	// Write after Close is a mistake rather than a race -- both come from the goroutine that
	// owns the Engine -- and this turns it into a dropped block instead of a send on a closed
	// channel, which would panic in the middle of somebody's game.
	if w.closed.Load() {
		w.dropped.Add(int64(len(samples)))
		return
	}
	var b []byte
	select {
	case b = <-w.free:
	default:
		b = make([]byte, 0, len(samples)*2)
	}
	b = encode(b, samples)

	select {
	case w.queue <- b:
	default:
		w.dropped.Add(int64(len(samples)))
	}
}

// pump is the only goroutine here, and it owns the device.
func (w *WaveOut) pump() {
	defer close(w.done)
	for b := range w.queue {
		w.play(b)
		select {
		case w.free <- b[:0]:
		default:
		}
	}
}

// play copies one block into the device's memory and queues it.
func (w *WaveOut) play(b []byte) {
	if w.failed.Load() != nil {
		return // a failed sink is silence, not a crash: keep draining the queue
	}
	if w.playing && w.inFlight() == 0 {
		// The device has nothing left to play and is idle, so what the player heard was a
		// gap: the frame loop stalled for longer than the cushion. Counted and reported at
		// exit, and deliberately not made up for -- writing silence to rebuild the cushion
		// would add latency that never comes back, and the pump's clock has already decided
		// what is due next (pump.go's Skipped is the same stall seen from the other end).
		w.underruns.Add(1)
	}
	i := w.acquire()
	if i < 0 {
		return
	}
	n := copy(waveBufs[i][:], b)
	if err := w.send(i, n); err != nil {
		w.fail(err)
		return
	}
	w.written.Add(int64(n) / 2)
}

// send prepares and writes block i, whose first n bytes are the samples.
func (w *WaveOut) send(i, n int) error {
	h := &waveHdrs[i]
	h.bufferLength = uint32(n)
	h.flags = 0
	h.loops = 0

	// Prepared before every write and unprepared after every completion, rather than prepared
	// once: a header may not be modified while it is prepared, and dwBufferLength changes from
	// block to block because the live path mixes what the wall clock says is due.
	if r, _, _ := procWaveOutPrepareHeader.Call(w.dev, uintptr(unsafe.Pointer(h)), hdrSize); r != mmsyserrNoError {
		return fmt.Errorf("audio: waveOutPrepareHeader: %s", mmName(uint32(r)))
	}
	if r, _, _ := procWaveOutWrite.Call(w.dev, uintptr(unsafe.Pointer(h)), hdrSize); r != mmsyserrNoError {
		procWaveOutUnprepareHeader.Call(w.dev, uintptr(unsafe.Pointer(h)), hdrSize)
		return fmt.Errorf("audio: waveOutWrite: %s", mmName(uint32(r)))
	}
	w.busy[i] = true
	return nil
}

// acquire returns a block the sink may fill, waiting for the device to give one back if it holds
// them all, or -1 if the device has stopped returning them.
//
// Waiting here is what means Write does not have to: the queue is the slack, and this is the one
// place that blocks. It polls rather than taking a callback or an event, for the reason callbackNull
// records in waveout.go.
func (w *WaveOut) acquire() int {
	deadline := time.Now().Add(waveStall)
	for {
		if i := w.idle(); i >= 0 {
			return i
		}
		if i := w.reclaim(); i >= 0 {
			return i
		}
		if w.failed.Load() != nil {
			return -1
		}
		if time.Now().After(deadline) {
			w.fail(fmt.Errorf("audio: the device has not returned a block in %v", waveStall))
			return -1
		}
		time.Sleep(wavePoll)
	}
}

// idle is a block the device does not have, or -1.
func (w *WaveOut) idle() int {
	for i, busy := range w.busy {
		if !busy {
			return i
		}
	}
	return -1
}

// inFlight is how many blocks the device is holding: the queue depth at the far end, and zero means
// it has run dry.
func (w *WaveOut) inFlight() int {
	n := 0
	for _, busy := range w.busy {
		if busy {
			n++
		}
	}
	return n
}

// reclaim takes back one block the device has finished with, or returns -1.
//
// The WHDR_DONE read is atomic because the driver writes that word from its own thread. Nothing
// else in this port reads memory two threads share, and the alternative -- a plain load the
// compiler is free to hoist out of the loop above -- is a poll that never sees the flag change.
func (w *WaveOut) reclaim() int {
	for i := range waveHdrs {
		if !w.busy[i] || atomic.LoadUint32(&waveHdrs[i].flags)&whdrDone == 0 {
			continue
		}
		if r, _, _ := procWaveOutUnprepareHeader.Call(w.dev, uintptr(unsafe.Pointer(&waveHdrs[i])), hdrSize); r != mmsyserrNoError {
			// WAVERR_STILLPLAYING here would mean the driver set WHDR_DONE and then
			// said the block is in use, which cannot both be true. Whatever it is, the
			// block is not reusable and neither is the device.
			w.fail(fmt.Errorf("audio: waveOutUnprepareHeader: %s", mmName(uint32(r))))
			return -1
		}
		w.busy[i] = false
		return i
	}
	return -1
}

// fail latches the first error. Only the writer goroutine calls it, and Close only after that
// goroutine has stopped, so the read and the store need not be one operation.
func (w *WaveOut) fail(err error) {
	if w.failed.Load() == nil {
		w.failed.Store(&err)
	}
}

// Close stops the device and gives the blocks back. Safe to call twice.
func (w *WaveOut) Close() error {
	var err error
	w.closeOnce.Do(func() {
		w.closed.Store(true)
		close(w.queue)
		<-w.done
		w.shutdown(&err)
	})
	return err
}

// shutdown is the teardown, in the one order that works: play out, stop, unprepare, close.
//
// The brief wait at the top is the difference between a clean last note and a click, and it is short
// because a game that takes a second to quit reads as a game that has hung. waveOutReset then marks
// whatever is left as done, which is what makes unpreparing those headers legal -- without it
// waveOutClose answers WAVERR_STILLPLAYING and the device stays open for the life of the process.
//
// errp collects the first failure, or is nil when the caller has nowhere to put one (openWaveOut's
// own failure path, which already has a better error to report).
func (w *WaveOut) shutdown(errp *error) {
	fail := func(err error) {
		if errp != nil && *errp == nil {
			*errp = err
		}
	}

	deadline := time.Now().Add(waveDrain)
	for w.inFlight() > 0 && time.Now().Before(deadline) {
		if w.reclaim() < 0 {
			time.Sleep(wavePoll)
		}
	}

	if w.dev != 0 {
		if r, _, _ := procWaveOutReset.Call(w.dev); r != mmsyserrNoError {
			fail(fmt.Errorf("audio: waveOutReset: %s", mmName(uint32(r))))
		}
		for i := range w.busy {
			if w.busy[i] {
				procWaveOutUnprepareHeader.Call(w.dev, uintptr(unsafe.Pointer(&waveHdrs[i])), hdrSize)
				w.busy[i] = false
			}
		}
		if r, _, _ := procWaveOutClose.Call(w.dev); r != mmsyserrNoError {
			fail(fmt.Errorf("audio: waveOutClose: %s", mmName(uint32(r))))
		}
		w.dev = 0
	}
	// Last, and only now: the device is shut, so nothing outside this process is still reading
	// the blocks and another sink may have them.
	waveInUse.Store(false)
}

// Name is which output this is, for the startup line and the report on exit.
func (w *WaveOut) Name() string { return deviceName }

// Samples is how many reached the device and Dropped is how many did not, both safe to read while
// the writer goroutine runs.
func (w *WaveOut) Samples() int64 { return w.written.Load() }
func (w *WaveOut) Dropped() int64 { return w.dropped.Load() }

// Underruns is how many times the device ran dry: the count of audible gaps, which is the one number
// this sink has that an external player's could not report.
func (w *WaveOut) Underruns() int64 { return w.underruns.Load() }

// Err is the error that stopped the device, or nil. A failed sink still accepts writes and still
// drops them, so this is the only way to find out.
func (w *WaveOut) Err() error {
	if e := w.failed.Load(); e != nil {
		return *e
	}
	return nil
}
