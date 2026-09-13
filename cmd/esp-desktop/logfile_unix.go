//go:build desktop && (linux || darwin || freebsd || netbsd || openbsd)

package main

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// attachConsoleLog redirects stdout/stderr to data/desktop.log when the process
// has no controlling terminal.
//
// Linux/macOS have no PE "GUI subsystem" concept: a windowed app is just a
// normal executable that opens an X11/Wayland window. Whether a terminal exists
// depends on how it was launched:
//
//   - from a file manager / app menu via the bundled .desktop file
//     (Terminal=false) -> no terminal, so output would otherwise be lost;
//   - from a shell -> a terminal exists and output should stay inline.
//
// Terminal detection uses the TCGETS ioctl, which is the authoritative test:
// unlike "is a character device", it correctly rejects /dev/null and other
// non-TTY character devices, and it works regardless of redirection.
func attachConsoleLog() {
	if isTerminal(os.Stdout) || isTerminal(os.Stderr) {
		return // launched from a terminal: keep output visible there
	}

	// Runs before bootstrap.Prepare(), so data/ may not exist yet.
	if err := os.MkdirAll("data", 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join("data", "desktop.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	os.Stdout = f
	os.Stderr = f
}

// isTerminal reports whether f is a real terminal, using the TCGETS ioctl.
func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	_, err := unix.IoctlGetTermios(int(f.Fd()), unix.TCGETS)
	return err == nil
}
