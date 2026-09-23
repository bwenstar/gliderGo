package main

import (
	"bufio"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var getConsoleProcessList = syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleProcessList")

// holdConsole keeps a console that is about to close open until Enter is pressed, so that the
// error above it can be read (docs/IMPROVEMENTS.md 4.35).
//
// Only a console this process has to itself, which is what double-clicking the .exe in Explorer
// gives it: GetConsoleProcessList counts the processes attached, and one means nobody else is
// there to keep the window open. Started from cmd, PowerShell or a terminal, the shell is attached
// too and stays after the game has gone, so there is nothing to wait for -- and a CI runner, which
// has no console at all, gets 0 and is not held either.
func holdConsole() {
	if getConsoleProcessList.Find() != nil {
		return
	}
	var ids [2]uint32
	if n, _, _ := getConsoleProcessList.Call(uintptr(unsafe.Pointer(&ids[0])), uintptr(len(ids))); n != 1 {
		return
	}
	fmt.Fprintf(os.Stderr, "%spress Enter to close this window\n", errorPrefix)
	bufio.NewReader(os.Stdin).ReadString('\n')
}
