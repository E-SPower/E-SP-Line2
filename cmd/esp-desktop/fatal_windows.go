//go:build desktop && windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	user32          = syscall.NewLazyDLL("user32.dll")
	procMessageBoxW = user32.NewProc("MessageBoxW")
)

const (
	mbOK            = 0x00000000
	mbIconError     = 0x00000010
	mbSetForeground = 0x00010000
)

func utf16Ptr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

// showFatalDialog displays a modal error dialog so the user can read the reason
// for a failed start even though the console window closes immediately.
func showFatalDialog(title, msg string) {
	procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(utf16Ptr(msg))),
		uintptr(unsafe.Pointer(utf16Ptr(title))),
		uintptr(mbOK|mbIconError|mbSetForeground),
	)
}

// waitForAck is a no-op on Windows: MessageBox is modal and already awaited.
func waitForAck() {}

var _ = syscall.UTF16PtrFromString
