//go:build windows

package audio

// The two tests that need a Windows to mean anything, and the only place in the port where a test
// touches sound hardware.
//
// These exist because waveout_windows.go was written on a machine that cannot run it. waveout_test.go
// pins everything checkable on paper -- struct offsets, the format against the mix, the error table --
// and stops exactly where the syscalls begin. This file starts there, and the payoff is that the CI
// job which runs `go test ./...` on windows-latest (see .github/workflows/ci.yml) executes the DLL
// path and the entry-point names for real, on every push, without anybody owning a Windows machine.
//
// Neither test needs a sound card to be useful and neither makes a noise if it finds one: the second
// writes zeroes. A hosted runner has winmm and no output device, so the first test runs there and the
// second skips; a developer's desktop runs both.

import (
	"os"
	"strings"
	"testing"
)

// TestWinmmLoadsFromTheSystemDirectory is the load path: ask Windows where System32 is, load
// winmm.dll out of it by absolute path, resolve the seven entry points, and call the harmless one.
//
// What it can catch is most of what a never-run loader gets wrong: a mishandled
// GetSystemDirectoryW length, a misspelt export, a bad `\` in the path, a Proc.Call signature that
// faults. What it cannot catch is anything about playback, which is the second test's job.
func TestWinmmLoadsFromTheSystemDirectory(t *testing.T) {
	dir, err := systemDirectory()
	if err != nil {
		t.Fatalf("systemDirectory: %v", err)
	}
	// Absolute, and not truncated at a NUL -- the two ways a UTF-16 out-parameter goes wrong.
	// Deliberately not asserted to end in "System32": a 386 build of this game running under
	// WOW64 is told SysWOW64, and that is the correct answer for it, because that is where the
	// 32-bit winmm.dll lives.
	if !strings.Contains(dir, `:\`) && !strings.HasPrefix(dir, `\\`) {
		t.Errorf("systemDirectory = %q, which is not an absolute path", dir)
	}
	if strings.ContainsRune(dir, 0) {
		t.Errorf("systemDirectory = %q, which still has its terminator in it", dir)
	}
	t.Logf("system directory: %s", dir)

	// A Windows without winmm.dll is a Windows without audio -- Nano Server, a stripped
	// container image -- and that is an environment fact rather than a defect in the loader, so
	// it skips. Anything else the load can fail with is a defect and fails the test.
	path := dir + `\winmm.dll`
	if _, err := os.Stat(path); err != nil {
		t.Skipf("no %s on this installation, so there is nothing to load: %v", path, err)
	}
	if err := loadWinmm(); err != nil {
		t.Fatalf("loadWinmm: %v", err)
	}

	// The call itself, not just the resolution: waveOutGetNumDevs takes no arguments and cannot
	// change anything, so it is the one entry point a test may make freely. Zero is a legitimate
	// answer and is logged rather than failed, because it is the answer on every CI runner.
	n, _, _ := procWaveOutGetNumDevs.Call()
	t.Logf("waveOutGetNumDevs: %d output device(s)", n)
	if (n > 0) != haveDevice() {
		t.Errorf("waveOutGetNumDevs says %d devices but haveDevice says %v", n, haveDevice())
	}
}

// TestWaveOutPlaysSilence is the device path end to end: open, prefill, prepare, write, poll for
// WHDR_DONE, reclaim the block, reset, unprepare, close. On a machine with an output device this is
// the first and only proof that the sink works; everywhere else it skips.
//
// It writes silence, so running it is inaudible, and it writes as fast as it can rather than at frame
// rate on purpose -- filling all eight blocks in a few milliseconds is what forces acquire to wait
// for the device to give one back, which is the part of the sink a leisurely test would never reach.
func TestWaveOutPlaysSilence(t *testing.T) {
	if !haveDevice() {
		t.Skip("no waveOut output device on this machine")
	}
	w, err := openWaveOut()
	if err != nil {
		// A device another program holds exclusively is an ordinary state of a desktop, not a
		// bug in this file, and it is worth telling apart from the failures that are: a
		// refused format, a missing driver. The test only has the message to tell them apart
		// by, which is one of the reasons the message carries the MMRESULT name.
		if strings.Contains(err.Error(), mmErrors[4]) {
			t.Skipf("another program holds the output device: %v", err)
		}
		t.Fatalf("openWaveOut: %v", err)
	}

	// One Write of one frame is one queue entry is one device block -- SamplesPerFrame is well
	// under waveBlockSamples, so Write never splits it -- and waveDepth of them may be waiting
	// for the pump at once. Writing exactly that many is the longest burst whose outcome does
	// not depend on whether the pump goroutine gets scheduled during the loop: they all fit the
	// channel even if it does not run at all, so nothing can be dropped. Writing more (this said
	// twelve, and claimed in a comment that the queue was deeper than the test was long, which
	// it never was) makes the assertions below a bet on the scheduler.
	//
	// Eight still does the job the burst is here for. Open leaves only waveBlocks - wavePrefill
	// idle blocks, so the last three of these cannot go out until the device has finished with
	// an earlier one and acquire has taken it back -- which is the part of the sink a test that
	// wrote at frame rate would never reach, and now reaches every time instead of usually.
	const frames = waveDepth
	silence := make([]int16, SamplesPerFrame)
	for i := 0; i < frames; i++ {
		if err := w.Write(silence); err != nil {
			t.Fatalf("Write %d: %v", i, err)
		}
	}

	// Close drains the queue before it stops the device, so every accepted block has been sent
	// by the time this returns and the counters below are final.
	if err := w.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	if err := w.Err(); err != nil {
		t.Errorf("the device stopped taking samples: %v", err)
	}
	if got, want := w.Samples(), int64(frames*SamplesPerFrame); got != want {
		t.Errorf("Samples = %d, want %d", got, want)
	}
	if got := w.Dropped(); got != 0 {
		t.Errorf("Dropped = %d, want 0: the burst above is exactly as long as the queue is "+
			"deep, so the only way to drop a block is to have stopped accepting them", got)
	}
	// Underruns are the machine's business, not the sink's: a runner that loses the goroutine to
	// a scheduler hiccup mid-test really did leave the device idle. Logged, never failed.
	if got := w.Underruns(); got != 0 {
		t.Logf("%d underruns, which on a loaded machine is not a fault", got)
	}

	// The blocks are shared package state, so the two things that must hold about the guard are
	// that Close gives them back and that a second sink cannot take them while a first one has
	// them. Both are cheap to check here and impossible to check anywhere else.
	if waveInUse.Load() {
		t.Error("Close left the device marked in use")
	}
	again, err := openWaveOut()
	if err != nil {
		t.Fatalf("reopen after Close: %v", err)
	}
	// Reaching this would mean the compare-and-swap in openWaveOut is broken, and a leaked
	// handle is then the least of it, so nothing is closed on this path.
	if _, err := openWaveOut(); err == nil {
		t.Error("two sinks opened at once, which the shared blocks cannot support")
	}
	if err := again.Close(); err != nil {
		t.Errorf("Close after reopen: %v", err)
	}
}
