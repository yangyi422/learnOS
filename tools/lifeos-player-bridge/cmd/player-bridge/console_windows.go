//go:build windows

package main

import (
	"os"
	"syscall"
)

const attachParentProcess = ^uintptr(0) // (DWORD)-1

var procAttachConsole = syscall.NewLazyDLL("kernel32.dll").NewProc("AttachConsole")

// attachParentConsole re-connects stdout/stderr when the binary is built as a GUI
// app: the protocol launch must not flash a console window, while `-verify` and
// `-play` run from a terminal still need visible output.
func attachParentConsole() {
	if ok, _, _ := procAttachConsole.Call(attachParentProcess); ok == 0 {
		return
	}
	if out, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0); err == nil {
		os.Stdout = out
		os.Stderr = out
	}
}