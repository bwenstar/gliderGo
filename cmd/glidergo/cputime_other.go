//go:build !(linux || darwin || freebsd || netbsd || openbsd || dragonfly || windows)

package main

import "time"

// cpuTime is cputime_unix.go's, on a platform this port has no way to ask.
func cpuTime() (time.Duration, bool) { return 0, false }
