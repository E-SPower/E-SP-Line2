// Package bootstrap performs cross-platform process startup preparation that
// must run before configuration is loaded and the database is opened.
//
// The problems it solves (notably on Windows, where the process is usually
// started by double-clicking the .exe):
//
//  1. Working directory. Configuration is discovered relative to the current
//     directory ("./config/config.yaml"), and the SQLite DSN defaults to the
//     relative path "data/e-sp-line2.db". When Windows launches an .exe the
//     working directory is NOT guaranteed to be the executable's folder (it
//     can be C:\Windows\System32), so config discovery and DB creation fail.
//     Prepare() chdirs to the executable's directory when sensible.
//
//  2. Missing data directory. SQLite cannot create the database file if its
//     parent directory does not exist, which produced a fatal error and an
//     immediate "flash close" on Windows. Prepare() creates data/.
package bootstrap

import (
	"os"
	"path/filepath"
)

// Prepare normalizes the working directory and creates runtime directories.
//
// It is intentionally tolerant: any failure is non-fatal and returned for
// logging only, because the process may legitimately run from a directory
// where the executable path cannot be determined (e.g. go run).
func Prepare() (notes []string) {
	if exe, err := os.Executable(); err == nil {
		if resolved, rerr := filepath.EvalSymlinks(exe); rerr == nil {
			exe = resolved
		}
		exeDir := filepath.Dir(exe)

		// Only chdir when it is safe and useful:
		//  - never for `go run` (binary lives in a temp dir)
		//  - never override an explicit cwd that already has a config file
		if !isGoRun(exeDir) && shouldChdir(exeDir) {
			if err := os.Chdir(exeDir); err == nil {
				notes = append(notes, "working directory -> "+exeDir)
			} else {
				notes = append(notes, "chdir failed: "+err.Error())
			}
		}
	}

	// Runtime directories. The default SQLite DSN is data/e-sp-line2.db, so the
	// data directory must exist before the database is opened. Errors are
	// reported rather than swallowed, because a failure here is precisely what
	// makes the process exit immediately after a double-click.
	for _, dir := range []string{"data", "data/instances", "data/logs", "data/deps"} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			notes = append(notes, "mkdir "+dir+" failed: "+err.Error())
		}
	}
	return notes
}

// isGoRun reports whether the executable lives in Go's temporary build dir
// produced by `go run` / `go test`.
//
// Detection is deliberately narrow: only the "go-build" component counts.
// Treating every path under the OS temp dir as a go run was wrong, because
// release builds are frequently unpacked into /tmp on Linux, and skipping the
// chdir there re-introduces the missing-database failure.
func isGoRun(dir string) bool {
	for d := dir; ; {
		if filepath.Base(d) == "go-build" {
			return true
		}
		parent := filepath.Dir(d)
		if parent == d {
			return false
		}
		d = parent
	}
}

// shouldChdir decides whether to switch to the executable directory. We avoid
// moving if the current directory already looks like a deployment root (it has
// a config directory or the binary itself), so that explicitly chosen working
// directories keep precedence.
func shouldChdir(exeDir string) bool {
	cwd, err := os.Getwd()
	if err != nil {
		return true
	}
	if cwd == exeDir {
		return false
	}
	// Current dir already looks usable: keep it.
	if st, err := os.Stat(filepath.Join(cwd, "config")); err == nil && st.IsDir() {
		return false
	}
	if _, err := os.Stat(filepath.Join(cwd, "config.yaml")); err == nil {
		return false
	}
	return true
}
