//go:build desktop && !windows && !linux && !darwin && !freebsd && !netbsd && !openbsd

package main

import (
	"os"
	"path/filepath"
)

// attachConsoleLog redirects stdout/stderr to data/desktop.log.
//
// Fallback for platforms without a TCGETS-based terminal check. It always
// redirects, which keeps the log available; platforms in this list are not
// primary targets for the desktop build.
func attachConsoleLog() {
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
