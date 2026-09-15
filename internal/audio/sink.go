package audio

// Where the mix goes: nowhere, a file (wav.go), or a command-line audio player's stdin.
//
// ---------------------------------------------------------------------------
// Why a subprocess and not a device
// ---------------------------------------------------------------------------
//
// Talking to ALSA, PulseAudio, PipeWire, WASAPI or CoreAudio means cgo or a hand-written
// syscall layer per platform, and the port is standard-library-only Go by necessity: the build
// host is airgapped, so there is no golang.org/x/sys, no oto, no beep and no ebiten. That
// constraint turns out to point at the right answer anyway. Every Linux desktop ships at least
// one of pw-play, paplay, aplay, ffplay or sox, they all read raw PCM on stdin, and a
// subprocess that dies takes nothing with it -- which is more than can be said for a cgo audio
// callback that segfaults inside the game's address space.
//
// The cost is honest and worth stating: one process, one pipe, and a latency the port does not
// control, because the player chooses its own buffer size. A sound is heard perhaps 50 to 200
// milliseconds after the frame that asked for it, depending on which player was found. For
// Glider PRO -- where the loudest thing in the game is a glider being shredded, and nothing
// depends on sub-frame audio timing -- that is fine, and it stays fine unless the port ever
// wants sample-accurate feedback, which it never will. If a platform's answer turns out to be
// worse than this (Windows has no such tool in the box), the Sink interface is two methods
// wide and a native driver can be dropped in behind it without the engine knowing. See
// docs/IMPROVEMENTS.md 2.48.

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// Sink is somewhere a mix can go. Write is called from the goroutine that owns the Engine and
// must not block on anything slower than a memory copy -- everything in this file that could
// block does it on another goroutine.
type Sink interface {
	Write(samples []int16) error
	Close() error
}

// Discard is the sink for a build with sound off, and for the tests: it counts and drops.
//
// It exists rather than a nil Sink because "no sink" and "a sink that goes nowhere" want
// different code at the call sites, and because Samples() is worth having in a headless run
// that wants to prove the mixer ran without keeping 26 MB of it.
type Discard struct{ samples int64 }

func (d *Discard) Write(s []int16) error { d.samples += int64(len(s)); return nil }
func (d *Discard) Close() error          { return nil }
func (d *Discard) Samples() int64        { return d.samples }

// Tee is several sinks at once: play and record in the same session.
//
// It exists so that `-wav` is not exclusive with `-audio`. Recording what a real play session
// actually sounded like is the only way to answer a bug report about audio from a machine the
// developer cannot reach, and asking the reporter to play it twice -- once to hear it, once to
// capture it -- would produce two different recordings, because the live path's timing comes
// from the wall clock.
//
// The first error wins and the rest of the sinks are still written to, because a full disk should
// not stop the sound coming out of the speakers. Close closes all of them for the same reason.
type Tee []Sink

func (t Tee) Write(samples []int16) error {
	var first error
	for _, s := range t {
		if err := s.Write(samples); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (t Tee) Close() error {
	var first error
	for _, s := range t {
		if err := s.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// Digest is a Sink that hashes the mix instead of keeping it.
//
// It is what makes "the audio is deterministic" a checked claim rather than a design intention.
// The recorded path mixes exactly SamplesPerFrame per game frame from an engine with no clock and
// no goroutine, so two runs of one replay script must produce the same bytes -- and 26 MB of
// samples is far too much to keep as a fixture, while sixteen hex characters is a line in a bug
// report. `glidertool replay` prints it beside the sample digest, and the pair covers the two
// halves of a frame: what the simulation did and what it sounded like.
//
// It hashes the encoded little-endian bytes rather than the int16s, so the value is the checksum
// of the WAV payload a `-wav` run would write and can be checked against that file from outside
// the program.
type Digest struct {
	h       hash.Hash
	buf     []byte
	samples int64
}

func NewDigest() *Digest { return &Digest{h: sha256.New()} }

func (d *Digest) Write(samples []int16) error {
	d.buf = encode(d.buf, samples)
	d.h.Write(d.buf)
	d.samples += int64(len(samples))
	return nil
}

// Close is a no-op: there is nothing to flush and Sum stays readable afterwards, which is what
// lets a Pump close its sinks before its owner asks for the result.
func (d *Digest) Close() error { return nil }

func (d *Digest) Samples() int64 { return d.samples }

// Sum is the hash so far, truncated to 16 hex characters for the same reason
// replay.Result.Digest is: it is quoted by people, and 64 characters of hex is not quotable.
// Sixty-four bits is ample for detecting a change nobody is trying to hide.
func (d *Digest) Sum() string { return hex.EncodeToString(d.h.Sum(nil))[:16] }

// player is one candidate external player: the executable and the arguments that make it read
// signed 16-bit little-endian mono at a given rate from stdin.
type player struct {
	name string
	args []string // %r is replaced with the sample rate
	note string   // what it belongs to, for the error message when nothing is found
}

// players is the search order, best first.
//
// The order is by *how few layers the samples cross*: pw-play and paplay talk to the sound
// server the desktop is already using, aplay talks to ALSA directly (which is right on a
// server or a minimal window manager and wrong on a desktop where the server holds the
// device), and the last two are transcoders that happen to be able to play. ffplay needs
// -nodisp or it opens a window for a file with no video, and -autoexit or it hangs on end of
// input.
var players = []player{
	{"pw-play", []string{"--rate=%r", "--channels=1", "--format=s16", "-"}, "PipeWire"},
	{"paplay", []string{"--raw", "--rate=%r", "--channels=1", "--format=s16le", "--client-name=gliderGo"}, "PulseAudio"},
	{"aplay", []string{"-q", "-t", "raw", "-f", "S16_LE", "-r", "%r", "-c", "1", "-"}, "ALSA"},
	{"ffplay", []string{"-hide_banner", "-loglevel", "quiet", "-nodisp", "-autoexit", "-f", "s16le", "-ar", "%r", "-ac", "1", "-i", "-"}, "FFmpeg"},
	{"play", []string{"-q", "-t", "raw", "-e", "signed", "-b", "16", "-c", "1", "-r", "%r", "-"}, "SoX"},
}

// Players lists the external players available on this machine, best first, for `-audio list`
// and for the error a failed Pipe returns.
func Players() []string {
	var found []string
	for _, p := range players {
		if _, err := exec.LookPath(p.name); err == nil {
			found = append(found, p.name)
		}
	}
	return found
}

// Pipe is a Sink that feeds an external player's stdin.
//
// The queue is bounded and overflow is dropped rather than waited on. That is the whole
// contract, and it is the right way round for a game: an audio sink that blocks stops the
// simulation, and a simulation that stutters is a worse bug than a mix that skips 33
// milliseconds. Drops are counted, and a nonzero count after a normal session is a real
// finding -- it means either the pump is running ahead of the player or the player has stopped
// reading.
type Pipe struct {
	name string
	cmd  *exec.Cmd
	w    io.WriteCloser

	// free and queue are a two-channel buffer pool: queue holds the blocks waiting to be
	// written and free holds the blocks waiting to be filled, so a steady stream allocates
	// nothing after the first few frames. A `chan []byte` of fixed capacity is a bounded
	// ring with a blocking reader already written for us; hand-rolling one with a mutex and
	// two indices would be more code doing the same thing less clearly.
	free  chan []byte
	queue chan []byte

	dropped atomic.Int64
	written atomic.Int64
	failed  atomic.Pointer[error]

	closeOnce sync.Once
	done      chan struct{}
}

// pipeDepth is how many blocks may be in flight. One block is one Write, which on the live
// path is one frame's worth or less, so eight blocks is about a quarter of a second of slack
// against a game that stalls -- enough to ride out a garbage collection or a slow room wipe,
// short enough that a player who quits does not hear a quarter-second tail.
const pipeDepth = 8

// OpenPipe starts the best available player.
//
// `prefer` names one to insist on, or is empty to take the first that exists. An unavailable
// preference is an error rather than a silent fallback, because the flag exists for somebody
// diagnosing a specific player and quietly using a different one would waste their afternoon.
func OpenPipe(prefer string) (*Pipe, error) {
	list := players
	if prefer != "" {
		list = nil
		for _, p := range players {
			if p.name == prefer {
				list = append(list, p)
			}
		}
		if list == nil {
			return nil, fmt.Errorf("audio: unknown player %q; known players are %s", prefer, strings.Join(playerNames(), ", "))
		}
	}

	for _, p := range list {
		path, err := exec.LookPath(p.name)
		if err != nil {
			continue
		}
		args := make([]string, len(p.args))
		for i, a := range p.args {
			args[i] = strings.ReplaceAll(a, "%r", strconv.Itoa(Rate))
		}
		cmd := exec.Command(path, args...)
		// The player's own diagnostics go to the terminal: if aplay cannot open the
		// device it says so, and swallowing that would leave "there is no sound" with
		// no explanation anywhere. Its stdout is left alone for the same reason.
		cmd.Stderr = os.Stderr

		w, err := cmd.StdinPipe()
		if err != nil {
			return nil, err
		}
		if err := cmd.Start(); err != nil {
			continue
		}
		return newPipe(p.name, cmd, w), nil
	}

	if prefer != "" {
		return nil, fmt.Errorf("audio: %s is not installed", prefer)
	}
	return nil, fmt.Errorf("audio: no audio player found; install one of %s (%s) or use -wav to write a file instead",
		strings.Join(playerNames(), ", "), "PipeWire, PulseAudio, ALSA, FFmpeg or SoX")
}

func playerNames() []string {
	out := make([]string, len(players))
	for i, p := range players {
		out[i] = p.name
	}
	return out
}

func newPipe(name string, cmd *exec.Cmd, w io.WriteCloser) *Pipe {
	p := &Pipe{
		name:  name,
		cmd:   cmd,
		w:     w,
		free:  make(chan []byte, pipeDepth),
		queue: make(chan []byte, pipeDepth),
		done:  make(chan struct{}),
	}
	go p.pump()
	return p
}

// pump is the only goroutine in the package, and it owns exactly one thing: the pipe.
//
// It never touches the Engine, a channel, a Sound or a Stat -- it moves byte slices from a
// channel into a file descriptor. That is what keeps the engine single-threaded and free of
// locks: the concurrency in this package is one goroutine wide and it is here.
func (p *Pipe) pump() {
	defer close(p.done)
	for b := range p.queue {
		if p.failed.Load() == nil {
			if _, err := p.w.Write(b); err != nil {
				// Almost always EPIPE: the player exited, because the device was
				// busy or the user killed it. Recorded and then ignored -- the
				// queue keeps draining so that Write never blocks, and the game
				// plays on in silence. This is failedSound, arrived at from the
				// other direction.
				e := err
				p.failed.Store(&e)
			} else {
				p.written.Add(int64(len(b)) / 2)
			}
		}
		select {
		case p.free <- b[:0]:
		default:
		}
	}
}

// Write encodes and enqueues, dropping the block if the queue is full.
func (p *Pipe) Write(samples []int16) error {
	if len(samples) == 0 {
		return nil
	}
	var b []byte
	select {
	case b = <-p.free:
	default:
		b = make([]byte, 0, len(samples)*2)
	}
	b = encode(b, samples)

	select {
	case p.queue <- b:
	default:
		p.dropped.Add(int64(len(samples)))
	}
	return nil
}

// Close stops the player.
//
// Closing stdin rather than killing the process is what lets the player finish what it has
// buffered, which is the difference between a clean last note and a click. The pump goroutine
// is drained first so nothing is written after the descriptor closes.
func (p *Pipe) Close() error {
	var err error
	p.closeOnce.Do(func() {
		close(p.queue)
		<-p.done
		err = p.w.Close()
		// The player is expected to exit when its input ends; ffplay needs -autoexit
		// for that and has it. A nonzero status here is the player's business and not
		// the game's, so it is not reported: the samples were delivered either way.
		_ = p.cmd.Wait()
	})
	return err
}

// Name is which player this is, for the report a session prints on exit.
func (p *Pipe) Name() string { return p.name }

// Samples is how many frames reached the player, and Dropped is how many did not. Both are
// safe to read while the pump is running.
func (p *Pipe) Samples() int64 { return p.written.Load() }
func (p *Pipe) Dropped() int64 { return p.dropped.Load() }

// Err is the write error that stopped the pipe, or nil. A pipe that has failed still accepts
// writes and still drops them, so this is the only way to find out.
func (p *Pipe) Err() error {
	if e := p.failed.Load(); e != nil {
		return *e
	}
	return nil
}

// Decode is the inverse of encode, for the tests and for `glidertool` reading back a WAV it
// wrote. It reads little-endian 16-bit samples out of b.
func Decode(b []byte) []int16 {
	out := make([]int16, len(b)/2)
	for i := range out {
		out[i] = int16(binary.LittleEndian.Uint16(b[i*2:]))
	}
	return out
}
