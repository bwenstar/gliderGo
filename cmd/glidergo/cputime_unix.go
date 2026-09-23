//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package main

import (
	"syscall"
	"time"
)

// cpuTime is the processor time this process has used so far, user and system together, and
// whether the platform could say. A timed run reports the difference across its frame loop
// (docs/IMPROVEMENTS.md 2.76): a paced game's cost is its CPU, since its frame rate is the
// limiter's.
func cpuTime() (time.Duration, bool) {
	var ru syscall.Rusage
	if syscall.Getrusage(syscall.RUSAGE_SELF, &ru) != nil {
		return 0, false
	}
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano()), true
}
