package netplay

import "syscall"

// Winsock's numbers for the three dial failures joinFailure sorts by errno. The syscall package
// has names for them on Windows, and they are the wrong ones: syscall.ECONNREFUSED there is a
// number Go invented for its own use, and Winsock reports WSAECONNREFUSED, 10061. Kept to these
// three lines, because nothing this repository runs can exercise them (docs/IMPROVEMENTS.md
// 4.33); dial_test.go checks the sort over them by building the error a dial would return.
var (
	errRefused         error = syscall.Errno(10061) // WSAECONNREFUSED
	errNetUnreachable  error = syscall.Errno(10051) // WSAENETUNREACH
	errHostUnreachable error = syscall.Errno(10065) // WSAEHOSTUNREACH
)
