//go:build !windows

package netplay

import "syscall"

// The errnos joinFailure sorts a dial by. Windows has its own (dial_windows.go).
var (
	errRefused         error = syscall.ECONNREFUSED
	errNetUnreachable  error = syscall.ENETUNREACH
	errHostUnreachable error = syscall.EHOSTUNREACH
)
