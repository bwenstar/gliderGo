//go:build !windows

package main

// holdConsole is console_windows.go's. Nowhere else does a console close with the program that
// was started in it.
func holdConsole() {}
