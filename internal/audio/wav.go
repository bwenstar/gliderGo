package audio

// A WAV writer, because the machine this port is being developed on has no sound card.
//
// That is not a footnote, it is the reason this file is in the package rather than in a tool:
// the only way to check the audio path on an airgapped build box with no output device is to
// write the mix to a file and listen to it somewhere else. So the WAV is the *primary* sink
// during development and the test harness's only way to assert on real samples --
// `glidertool replay -wav` produces one, and its length, its checksum and the sound events
// beside it are all checkable without a speaker.
//
// Mono, 16-bit, little-endian, at Rate. No metadata, no LIST chunk, nothing a player has to
// tolerate: this file is read by ffplay, by aplay, by Audacity and by whatever the person
// reading a bug report has, and the way to be sure of that is to write the plainest
// 44-byte-header WAV that exists.

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// wavHeader is the canonical 44-byte RIFF/WAVE header: 12 bytes of RIFF, 24 of 'fmt ', 8 of
// 'data'. The two length fields are written as zero and patched on Close.
const wavHeader = 44

// WAV is a Sink that writes a RIFF file.
type WAV struct {
	f       *os.File
	w       *bufio.Writer
	buf     []byte
	samples int64
	closed  bool
}

// CreateWAV opens path and writes a placeholder header.
//
// The two sizes in a RIFF header come before the data they describe, so a streaming writer has
// exactly two choices: buffer the whole stream in memory, or write zeros and seek back at the
// end. This seeks back -- a ten-minute session is 26 MB of samples and holding that in memory
// to avoid one Seek would be a strange trade.
func CreateWAV(path string) (*WAV, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	w := &WAV{f: f, w: bufio.NewWriterSize(f, 64*1024)}
	if err := w.writeHeader(0); err != nil {
		f.Close()
		return nil, err
	}
	return w, nil
}

// writeHeader writes the 44 bytes with a given payload length.
func (w *WAV) writeHeader(dataBytes uint32) error {
	const (
		pcmFormat  = 1
		channels   = 1
		bitsPerSam = 16
	)
	blockAlign := uint16(channels * bitsPerSam / 8)

	var h [wavHeader]byte
	copy(h[0:], "RIFF")
	// 36 is the header's own size after the first eight bytes: the whole file minus
	// "RIFF" and this field.
	binary.LittleEndian.PutUint32(h[4:], 36+dataBytes)
	copy(h[8:], "WAVE")

	copy(h[12:], "fmt ")
	binary.LittleEndian.PutUint32(h[16:], 16) // length of the PCM fmt payload
	binary.LittleEndian.PutUint16(h[20:], pcmFormat)
	binary.LittleEndian.PutUint16(h[22:], channels)
	binary.LittleEndian.PutUint32(h[24:], Rate)
	binary.LittleEndian.PutUint32(h[28:], Rate*uint32(blockAlign)) // bytes per second
	binary.LittleEndian.PutUint16(h[32:], blockAlign)
	binary.LittleEndian.PutUint16(h[34:], bitsPerSam)

	copy(h[36:], "data")
	binary.LittleEndian.PutUint32(h[40:], dataBytes)

	_, err := w.w.Write(h[:])
	return err
}

// Write appends samples.
func (w *WAV) Write(samples []int16) error {
	if w.closed {
		return fmt.Errorf("audio: write to closed WAV %s", w.f.Name())
	}
	w.buf = encode(w.buf, samples)
	if _, err := w.w.Write(w.buf); err != nil {
		return err
	}
	w.samples += int64(len(samples))
	return nil
}

// Samples is how many frames have been written, for a caller that wants to check the stream's
// length against the frame count that produced it.
func (w *WAV) Samples() int64 { return w.samples }

// Close flushes, patches the two length fields and closes the file.
//
// A stream longer than 4 GiB -- 27 hours -- cannot be described by a 32-bit RIFF length. The
// port refuses to write a header it knows is wrong rather than truncating the number and
// producing a file that plays for four minutes: the samples are all on disk and the error says
// where the problem is.
func (w *WAV) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true

	if err := w.w.Flush(); err != nil {
		w.f.Close()
		return err
	}
	n := w.samples * 2
	if n > int64(^uint32(0))-36 {
		w.f.Close()
		return fmt.Errorf("audio: %s: %d samples is too long for a 32-bit RIFF length; the samples are written but the header is not", w.f.Name(), w.samples)
	}
	if _, err := w.f.Seek(0, io.SeekStart); err != nil {
		w.f.Close()
		return err
	}
	w.w.Reset(w.f)
	if err := w.writeHeader(uint32(n)); err != nil {
		w.f.Close()
		return err
	}
	if err := w.w.Flush(); err != nil {
		w.f.Close()
		return err
	}
	return w.f.Close()
}

// encode appends samples to dst as little-endian 16-bit, reusing dst's capacity.
//
// Little-endian regardless of the host, because that is what a WAV is and what every raw-PCM
// player on Linux defaults to. The port runs on nothing big-endian today, but a byte order
// that depends on the build host is the kind of bug that surfaces as "the audio is white
// noise" years later, so the encoding is explicit.
func encode(dst []byte, samples []int16) []byte {
	need := len(samples) * 2
	if cap(dst) < need {
		dst = make([]byte, need)
	}
	dst = dst[:need]
	for i, s := range samples {
		u := uint16(s)
		dst[i*2] = byte(u)
		dst[i*2+1] = byte(u >> 8)
	}
	return dst
}
