package main

import (
	"syscall"
	"time"
)

// cpuTime is cputime_unix.go's, from GetProcessTimes.
func cpuTime() (time.Duration, bool) {
	h, err := syscall.GetCurrentProcess()
	if err != nil {
		return 0, false
	}
	var created, exited, kernel, user syscall.Filetime
	if syscall.GetProcessTimes(h, &created, &exited, &kernel, &user) != nil {
		return 0, false
	}
	// Both are counts of 100 ns, and not dates, so Filetime.Nanoseconds -- which counts from
	// 1601 to 1970 first -- is the wrong conversion.
	ticks := func(f syscall.Filetime) int64 { return int64(f.HighDateTime)<<32 | int64(f.LowDateTime) }
	return time.Duration(ticks(kernel)+ticks(user)) * 100, true
}
