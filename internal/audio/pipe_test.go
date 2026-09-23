package audio

// The Pipe, with this test binary as the player.
//
// sink_test.go used to explain why the Pipe had no tests: a real player is either missing from the
// build machine or makes noise on somebody's desktop. Neither is true of a player that is this
// binary run again with one test selected, which reads what it is sent and does nothing with it
// but what the test asks. That is the only part of the Pipe worth a test in any case -- the
// players' own behaviour is theirs -- and it is the part `make race` needs: Write runs on the
// game's goroutine, pump on its own, and Close is where they meet.
//
// This is the harness docs/IMPROVEMENTS.md 2.71 asks for, in three of its four modes. The fourth,
// a player that dies after some bytes, and the two-candidate test belong to 2.71's fall-through,
// which is what they test.

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

// playerEnv selects what TestPipeHelperPlayer does when this binary is started as a player.
const playerEnv = "GLIDERGO_TEST_PLAYER"

// playerReady is what the echo player says before it reads anything.
const playerReady = "ready\n"

// TestPipeHelperPlayer is not a test. Run as a child with playerEnv set it is a player, and run
// any other way it does nothing and passes.
func TestPipeHelperPlayer(t *testing.T) {
	switch os.Getenv(playerEnv) {
	case "":
		return
	case "echo":
		// Says it is running, then hands back every byte it is sent, so the test can see
		// what reached it.
		os.Stdout.WriteString(playerReady)
		io.Copy(os.Stdout, os.Stdin)
	case "gone":
		// Exits without reading, which is what a player that cannot open the device does.
	case "stall":
		// Stays alive and never reads, which is what aplay without -N does on a busy device.
		time.Sleep(time.Hour) // not select {}, which the runtime ends as a deadlock
	}
	os.Exit(0)
}

// lockedBuffer is the player's stdout, read by the test while os/exec's goroutine writes it.
type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// startPlayer starts this binary as a player that behaves as `mode` says, the way OpenPipe starts
// a real one, and returns the Pipe over it and whatever the player writes to stdout.
func startPlayer(t *testing.T, mode string) (*Pipe, *lockedBuffer) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestPipeHelperPlayer$")
	// Built with -race, the child is a race-detector binary too, and those wait a second before
	// they exit (GORACE's atexit_sleep_ms) -- a second per test, spent inside Close.
	cmd.Env = append(os.Environ(), playerEnv+"="+mode,
		"GORACE="+strings.TrimSpace(os.Getenv("GORACE")+" atexit_sleep_ms=0"))
	out := &lockedBuffer{}
	cmd.Stdout = out
	w, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	// A player a failing test left running would outlive it; one Close already reaped is not
	// there to kill, and Kill says so, which is ignored.
	t.Cleanup(func() { cmd.Process.Kill() })
	return newPipe("helper", cmd, w, nil), out
}

// Everything Write accepted reaches the player, in order and encoded, by the time Close returns:
// Close drains the queue before it closes the player's stdin. Fewer blocks than the queue holds,
// so that none of them is dropped and the comparison can be exact.
//
// The test waits for the player to say it is running before it sends anything. Close gives a
// player pipeDrain to finish, which is long for one that has been playing all session and short
// for a process that has not started yet -- a race-detector binary on a busy machine, or a new
// executable that Windows Defender is still scanning.
func TestPipeDeliversWhatItQueues(t *testing.T) {
	p, out := startPlayer(t, "echo")
	deadline := time.Now().Add(time.Minute)
	for out.String() != playerReady {
		if time.Now().After(deadline) {
			t.Fatalf("the player had not started after a minute; its stdout is %q", out.String())
		}
		time.Sleep(time.Millisecond)
	}

	var all []int16
	for i := 0; i < pipeDepth/2; i++ {
		block := make([]int16, 735) // one frame at Rate
		for j := range block {
			block[j] = int16(i*1000 - j)
		}
		all = append(all, block...)
		if err := p.Write(block); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	got, _ := strings.CutPrefix(out.String(), playerReady)
	if want := encode(nil, all); got != string(want) {
		t.Errorf("the player got %d bytes, want the %d Write was given", len(got), len(want))
	}
	if p.Samples() != int64(len(all)) || p.Dropped() != 0 || p.Err() != nil {
		t.Errorf("the pipe reports %d samples, %d dropped and %v; want %d, 0 and nil",
			p.Samples(), p.Dropped(), p.Err(), len(all))
	}
	if err := p.Close(); err != nil {
		t.Errorf("a second Close: %v", err)
	}
}

// closeWithin runs Close and fails the test if it has not returned in time.
func closeWithin(t *testing.T, p *Pipe, limit time.Duration) time.Duration {
	t.Helper()
	done := make(chan struct{})
	start := time.Now()
	go func() { p.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(limit):
		t.Fatalf("Close had not returned after %v", limit)
	}
	return time.Since(start)
}

// A player that goes away does not take the game with it: Write keeps returning nil at once,
// the pump records the refused write as the pipe's error, and Close still returns. This is the
// path a player that cannot open the device takes.
func TestPipeOutlivesItsPlayer(t *testing.T) {
	p, _ := startPlayer(t, "gone")
	block := make([]int16, 735)
	deadline := time.Now().Add(10 * time.Second)
	for p.Err() == nil {
		if time.Now().After(deadline) {
			t.Fatal("ten seconds of writes to a player that has exited, and none of them failed")
		}
		if err := p.Write(block); err != nil {
			t.Fatalf("Write returned %v; a failed player should be reported by Err only", err)
		}
		time.Sleep(time.Millisecond)
	}
	closeWithin(t, p, 10*time.Second)

	// And a Write after Close is dropped, as WaveOut's is, rather than panicking.
	before := p.Dropped()
	if err := p.Write(block); err != nil {
		t.Errorf("Write after the player went and the pipe closed: %v", err)
	}
	if p.Dropped() != before+int64(len(block)) {
		t.Errorf("a Write after Close was not counted as dropped: %d, then %d", before, p.Dropped())
	}
}

// A player that is alive and not reading cannot stop the game from quitting. The pipe to it
// fills and the pump blocks writing into it; Close waits pipeDrain for the pump and then kills
// the player, which fails the write. Before pipeDrain, Close waited on the pump for as long as
// the player sat there.
func TestPipeClosesOnAPlayerThatStopsReading(t *testing.T) {
	p, _ := startPlayer(t, "stall")
	block := make([]int16, 735)
	for i := 0; i < 200; i++ { // 294 KB, several times what a pipe holds
		p.Write(block)
		time.Sleep(time.Millisecond)
	}
	if p.Err() != nil {
		t.Fatalf("a player that is still running refused a write: %v", p.Err())
	}

	took := closeWithin(t, p, 10*time.Second)
	if took < pipeDrain/2 || took > pipeDrain+5*time.Second {
		t.Errorf("Close took %v; it should wait pipeDrain (%v) for the player and then stop it",
			took, pipeDrain)
	}
	if p.Err() == nil {
		t.Error("the player was stopped and the pipe does not say so; the end-of-run line reads Err")
	}
}
