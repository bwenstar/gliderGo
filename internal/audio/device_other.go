//go:build !windows

package audio

// Every platform except Windows: no native driver, so Open falls through to the external players.
//
// The exact complement of waveout_windows.go's tag, on internal/platform/backend/doc.go's
// principle -- every build gets one openDevice, no build gets two, and none gets none. A macOS or
// ALSA driver would narrow this tag rather than adding a runtime probe.

// deviceName is empty, and Open compares against it only after checking that the caller asked for
// something, so no `-audio ""` can reach openDevice by accident.
const deviceName = ""

func openDevice() (Stream, error) { return nil, errNoDevice }

func haveDevice() bool { return false }
