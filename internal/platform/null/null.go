// Package null is the headless backend: it presents frames to memory (and
// optionally to PNG files) and mixes audio into a WAV file.
//
// This is not a stub for completeness' sake, it is load-bearing. The
// development host has no audio device at all, and its X server is a remote
// desktop, so the only way to verify rendering and sound automatically is to run
// the game with no host at all and diff the output against reference bytes. The
// fidelity test suite for the port is built on this backend.
package null

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"glidergo/internal/platform"
)

// Window records presented frames instead of displaying them.
type Window struct {
	w, h    int
	last    *platform.Framebuffer
	frames  int
	dumpDir string // if non-empty, every Present writes a PNG here

	// Script drives deterministic input for tests: frame number -> events to
	// deliver at the start of that frame.
	Script map[int][]platform.Event
	held   map[platform.Key]bool
	closed bool
}

// New creates a headless window. dumpDir may be empty.
func New(cfg platform.Config, dumpDir string) (*Window, error) {
	w, h := cfg.Width, cfg.Height
	if w == 0 {
		w = platform.ScreenWidth
	}
	if h == 0 {
		h = platform.ScreenHeight
	}
	if dumpDir != "" {
		if err := os.MkdirAll(dumpDir, 0o755); err != nil {
			return nil, err
		}
	}
	return &Window{w: w, h: h, dumpDir: dumpDir, held: map[platform.Key]bool{}}, nil
}

// Present copies the frame and, if configured, writes it as a PNG.
func (w *Window) Present(fb *platform.Framebuffer) error {
	if fb.W != w.w || fb.H != w.h {
		return fmt.Errorf("null: framebuffer is %dx%d, window expects %dx%d", fb.W, fb.H, w.w, w.h)
	}
	if w.last == nil {
		w.last = platform.NewFramebuffer(w.w, w.h)
	}
	for y := 0; y < fb.H; y++ {
		copy(w.last.Pix[y*w.last.Stride:], fb.Pix[y*fb.Stride:y*fb.Stride+fb.W*4])
	}
	if w.dumpDir != "" {
		if err := WritePNG(filepath.Join(w.dumpDir, fmt.Sprintf("frame-%06d.png", w.frames)), fb); err != nil {
			return err
		}
	}
	w.frames++
	return nil
}

// PollEvents returns the scripted events for the current frame.
func (w *Window) PollEvents() []platform.Event {
	evs := w.Script[w.frames]
	for _, e := range evs {
		switch e.Kind {
		case platform.EventKeyDown:
			w.held[e.Key] = true
		case platform.EventKeyUp:
			delete(w.held, e.Key)
		case platform.EventQuit:
			w.closed = true
		}
	}
	return evs
}

// KeyDown reports scripted key state.
func (w *Window) KeyDown(k platform.Key) bool { return w.held[k] }

// SetTitle is a no-op.
func (w *Window) SetTitle(string) error { return nil }

// Close is a no-op.
func (w *Window) Close() error { return nil }

// Closed reports whether a scripted quit has been delivered.
func (w *Window) Closed() bool { return w.closed }

// Frames counts presented frames.
func (w *Window) Frames() int { return w.frames }

// LastFrame returns the most recently presented frame, or nil.
func (w *Window) LastFrame() *platform.Framebuffer { return w.last }

// WritePNG saves a framebuffer as a PNG, converting BGRX to RGBA.
func WritePNG(path string, fb *platform.Framebuffer) error {
	img := image.NewRGBA(image.Rect(0, 0, fb.W, fb.H))
	for y := 0; y < fb.H; y++ {
		for x := 0; x < fb.W; x++ {
			o := y*fb.Stride + x*4
			img.SetRGBA(x, y, color.RGBA{R: fb.Pix[o+2], G: fb.Pix[o+1], B: fb.Pix[o+0], A: 0xff})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// WAVSink writes mixed audio to a RIFF/WAVE file. It is the reference audio
// backend on hosts without sound hardware.
type WAVSink struct {
	f        *os.File
	rate     int
	channels int
	samples  int
}

// NewWAVSink creates path and reserves space for the header.
func NewWAVSink(path string, rate, channels int) (*WAVSink, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	s := &WAVSink{f: f, rate: rate, channels: channels}
	if _, err := f.Write(make([]byte, 44)); err != nil { // header written on Close
		f.Close()
		return nil, err
	}
	return s, nil
}

// SampleRate reports the configured rate.
func (s *WAVSink) SampleRate() int { return s.rate }

// Channels reports the configured channel count.
func (s *WAVSink) Channels() int { return s.channels }

// Write appends samples.
func (s *WAVSink) Write(samples []int16) error {
	buf := make([]byte, len(samples)*2)
	for i, v := range samples {
		binary.LittleEndian.PutUint16(buf[i*2:], uint16(v))
	}
	if _, err := s.f.Write(buf); err != nil {
		return err
	}
	s.samples += len(samples)
	return nil
}

// Close back-fills the RIFF header and closes the file.
func (s *WAVSink) Close() error {
	dataLen := s.samples * 2
	h := make([]byte, 44)
	copy(h[0:], "RIFF")
	binary.LittleEndian.PutUint32(h[4:], uint32(36+dataLen))
	copy(h[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(h[16:], 16)
	binary.LittleEndian.PutUint16(h[20:], 1) // PCM
	binary.LittleEndian.PutUint16(h[22:], uint16(s.channels))
	binary.LittleEndian.PutUint32(h[24:], uint32(s.rate))
	binary.LittleEndian.PutUint32(h[28:], uint32(s.rate*s.channels*2))
	binary.LittleEndian.PutUint16(h[32:], uint16(s.channels*2))
	binary.LittleEndian.PutUint16(h[34:], 16)
	copy(h[36:], "data")
	binary.LittleEndian.PutUint32(h[40:], uint32(dataLen))
	if _, err := s.f.WriteAt(h, 0); err != nil {
		s.f.Close()
		return err
	}
	return s.f.Close()
}
