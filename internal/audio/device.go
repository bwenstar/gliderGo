package audio

// Choosing an output, and the seam a native driver arrives through.
//
// sink.go's answer -- encode the mix as raw PCM and pipe it to whichever command-line player the
// machine has -- is a good one on Linux, where every desktop ships at least one of five, and a
// poor one on Windows, which ships none of them. This file is the decision that lets a platform
// have a real driver instead: Open tries the native device first and the external players second,
// and nothing above it learns which it got. internal/platform/backend does the same job for the
// display, and for the same reason.
//
// Windows has such a driver, in waveout_windows.go. Linux deliberately does not: ALSA,
// PulseAudio and PipeWire are C libraries, this port has no cgo on the audio path and no way to
// dlopen one without it, and the subprocess it uses instead is both simpler and safer than a
// hand-declared snd_pcm_* ABI would be -- a player that dies takes nothing with it. macOS is
// Stage 6 and will want one, because it ships none of the five players either.

import (
	"errors"
	"fmt"
)

// Stream is a Sink that plays somewhere outside this process -- a sound device or an external
// player -- and can account for itself afterwards.
//
// It exists so that the shutdown report does not have to know which one it got. The four methods
// are the four things a bug report about bad sound needs: what was opened, how much of the mix
// reached it, how much was thrown away on the way, and whether it stopped accepting samples
// altogether. Both *Pipe and the native device satisfy it.
type Stream interface {
	Sink
	Name() string
	Samples() int64
	Dropped() int64
	Err() error
}

// errNoDevice means "there is no native driver to try on this machine", which is not a failure:
// it is every Linux build, and it is a Windows box with no sound card. Open treats it as a reason
// to go on to the external players rather than as something to report, which is why it is a
// sentinel and not a message.
var errNoDevice = errors.New("no native audio device")

// Open returns the best output available, or the one named exactly.
//
// An empty `prefer` takes the best: the native device if this platform has one and it opens, then
// the first external player that exists. A name is insisted on -- an unavailable one is an error
// rather than a silent fallback, because the flag exists for somebody diagnosing a specific
// output and quietly using a different one would waste their afternoon.
func Open(prefer string) (Stream, error) {
	switch {
	case prefer != "" && prefer == deviceName:
		return openDevice()
	case prefer != "":
		return OpenPipe(prefer)
	}

	dev, devErr := openDevice()
	if devErr == nil {
		return dev, nil
	}
	pipe, pipeErr := OpenPipe("")
	if pipeErr == nil {
		return pipe, nil
	}
	if errors.Is(devErr, errNoDevice) {
		// Nothing was tried but the players, so their error is the whole story.
		return nil, pipeErr
	}
	// The device failure is the more informative one -- it means this machine has an output
	// and something is wrong with it -- but the player search failed as well, and an error
	// that named only one of the two would send somebody after the wrong half.
	return nil, fmt.Errorf("%v (and %v)", devErr, pipeErr)
}

// outputNames is every name `-audio` accepts on this platform, installed or not, for the message
// that answers a mistyped one.
//
// Deliberately not the same list as Outputs: this one includes the native device whether or not the
// machine has a sound card, because somebody who typed `-audio wavout` wants to be told the spelling
// and not to be told the name does not exist.
func outputNames() []string {
	if deviceName == "" {
		return playerNames()
	}
	return append([]string{deviceName}, playerNames()...)
}

// Outputs lists what this machine can play through, best first, for `-audio list`.
//
// The native device is named first when there is one, because that is the order Open tries them
// in and because a Windows player who has just been told there is no sound needs to see that the
// answer needs no download.
func Outputs() []string {
	var out []string
	if haveDevice() {
		out = append(out, deviceName)
	}
	return append(out, Players()...)
}
