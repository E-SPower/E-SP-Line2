//go:build windows

package main

// On Windows, startup failures are surfaced through a modal MessageBox and the
// process waits for acknowledgement. A double-clicked console application would
// otherwise close its window instantly, hiding the reason for the failure.

import (
	"syscall"
	"unsafe"
)

var (
	user32          = syscall.NewLazyDLL("user32.dll")
	procMessageBoxW = user32.NewProc("MessageBoxW")
)

const (
	mbOK              = 0x00000000
	mbIconError       = 0x00000010
	mbSetForeground   = 0x00010000
)

func utf16Ptr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

// showFatalDialog displays a modal error dialog on Windows.
func showFatalDialog(title, msg string) {
	procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(utf16Ptr(msg))),
		uintptr(unsafe.Pointer(utf16Ptr(title))),
		uintptr(mbOK|mbIconError|mbSetForeground),
	)
}

// waitForAck does nothing extra on Windows: MessageBox is already modal, so the
// user has seen and dismissed the message by the time it returns.
func waitForAck() {}

// Keep syscall import used even if the dialog call is optimized differently.
var _ = syscall.UTF16PtrFromString
