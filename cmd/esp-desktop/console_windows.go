//go:build desktop && windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	procAttachConsole   = kernel32.NewProc("AttachConsole")
	procAllocConsole    = kernel32.NewProc("AllocConsole")
)

const attachParentProcess = ^uint32(0) // (DWORD)-1

// attachParentConsole attaches this process to the console of its parent
// (i.e. the terminal it was launched from). It returns false when there is no
// parent console, which is the normal case for a double-clicked executable.
//
// This lets a GUI-subsystem binary still print to the terminal when the user
// runs it from cmd.exe or PowerShell, while staying silent when double-clicked.
func attachParentConsole() bool {
	r, _, _ := procAttachConsole.Call(uintptr(attachParentProcess))
	return r != 0
}

// ensureConsole allocates a console when none is attached. Used only if output
// must be forced visible; the desktop build normally relies on data/desktop.log.
func ensureConsole() bool {
	r, _, _ := procAllocConsole.Call()
	return r != 0
}

var _ = unsafe.Pointer(nil)
