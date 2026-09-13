//go:build desktop && windows

package main

import (
	"os"
	"path/filepath"
)

// attachConsoleLog mirrors stdout/stderr into data/desktop.log on Windows.
//
// The desktop build is linked with -H=windowsgui so double-clicking it does not
// pop a console window. The downside is that anything printed to stdout/stderr
// (banner, logger output, panic traces) would be lost entirely. Mirroring it to
// a file keeps the build diagnosable without a console.
//
// It also attaches the process to the parent console when one exists (i.e. when
// started from a terminal), so running the exe from cmd/PowerShell still shows
// output inline.
func attachConsoleLog() {
	// Attach to the parent console if present. Failure is expected (and fine)
	// when launched by double-click.
	if attachParentConsole() {
		// Console attached: stdout/stderr already go somewhere visible.
	}

	// This runs before bootstrap.Prepare(), so data/ may not exist yet.
	if err := os.MkdirAll("data", 0o755); err != nil {
		return
	}

	f, err := os.OpenFile(filepath.Join("data", "desktop.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	// Mirror standard streams into the log file.
	os.Stdout = f
	os.Stderr = f
}
