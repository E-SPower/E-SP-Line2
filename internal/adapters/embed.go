// Package adapters embeds the bundled Python adapters into the binary so the
// shipped executable is self-contained: no external adapters/ directory is
// required next to it.
//
// Flow:
//
//	binary (go:embed adapters/**)  ->  EnsureExtracted()  ->  data/adapters/<platform>/
//	                                                            |
//	                                              AdapterCatalog / InstanceDirManager
//	                                                            |
//	                                                 data/instances/<id>/adapter/
//
// Extraction is idempotent and version-aware: the embedded tree is stamped with
// a content fingerprint, and files are (re)written only when the stamp changes,
// so normal startups do not touch the disk.
//
// An existing external adapters/ directory is still honoured as a developer
// override (see EffectiveDir).
package adapters

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// embedded holds the bundled adapter sources. The directory must exist at build
// time (a placeholder keeps it compilable before the sync step runs).
//
//go:embed all:adapters
var embedded embed.FS

// root is the prefix inside the embedded filesystem.
const root = "adapters"

// stampFileName records the fingerprint of the last extracted tree.
const stampFileName = ".embedded-stamp"

// DefaultDir is where the bundled adapters are extracted at runtime.
const DefaultDir = "data/adapters"

// Available reports whether any adapter is actually embedded (a build produced
// without the sync step only contains the placeholder and is treated as empty).
func Available() bool {
	dirs, err := embedded.ReadDir(root)
	if err != nil {
		return false
	}
	for _, d := range dirs {
		if d.IsDir() {
			return true
		}
	}
	return false
}

// List returns the platform codes (sub-directory names) of the embedded
// adapters, sorted.
func List() []string {
	dirs, err := embedded.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []string
	for _, d := range dirs {
		if d.IsDir() {
			out = append(out, d.Name())
		}
	}
	sort.Strings(out)
	return out
}

// fingerprint hashes every embedded file path and its content, producing a
// stable identifier for this build's adapter bundle.
func fingerprint() (string, error) {
	h := sha256.New()
	var files []string
	err := fs.WalkDir(embedded, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, p)
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(files)
	for _, p := range files {
		b, err := embedded.ReadFile(p)
		if err != nil {
			return "", err
		}
		h.Write([]byte(p))
		h.Write([]byte{0})
		h.Write(b)
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// EnsureExtracted writes the embedded adapters to destDir when they are missing
// or stale, and returns the number of files written.
//
// It is safe to call on every startup: when the on-disk stamp already matches
// the embedded fingerprint, nothing is written.
func EnsureExtracted(destDir string) (int, error) {
	if destDir == "" {
		destDir = DefaultDir
	}
	if !Available() {
		// Build without a synced bundle: nothing to do.
		return 0, nil
	}

	want, err := fingerprint()
	if err != nil {
		return 0, fmt.Errorf("failed to fingerprint embedded adapters: %w", err)
	}

	stampPath := filepath.Join(destDir, stampFileName)
	if cur, err := os.ReadFile(stampPath); err == nil && strings.TrimSpace(string(cur)) == want {
		return 0, nil // already up to date
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return 0, err
	}

	written := 0
	err = fs.WalkDir(embedded, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(p, root), "/")
		if rel == "" {
			return nil
		}
		target := filepath.Join(destDir, filepath.FromSlash(rel))

		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		// Skip generated caches that must never be shipped or extracted.
		base := path.Base(p)
		if base == "__pycache__" || strings.HasSuffix(base, ".pyc") {
			return nil
		}

		data, err := embedded.ReadFile(p)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
		written++
		return nil
	})
	if err != nil {
		return written, err
	}

	// Stamp only after a fully successful extraction.
	_ = os.WriteFile(stampPath, []byte(want+"\n"), 0o644)
	return written, nil
}

// EffectiveDir decides which adapters directory the server should scan.
//
// Priority:
//  1. external override — a real adapters/ directory next to the executable
//     (development convenience: edits there win without a rebuild);
//  2. otherwise data/adapters/ — the embedded bundle extracted at runtime.
//
// The returned directory always exists and contains the bundled adapters.
func EffectiveDir(externalDir string) (string, error) {
	if externalDir == "" {
		externalDir = "adapters"
	}

	// 1) Honour an external directory that actually holds adapters.
	if hasAdapterYAML(externalDir) {
		return externalDir, nil
	}

	// 2) Extract the embedded bundle and use it.
	if !Available() {
		// Nothing embedded: keep the configured directory so error messages
		// still point at the expected location.
		_ = os.MkdirAll(externalDir, 0o755)
		return externalDir, nil
	}

	dest := DefaultDir
	written, err := EnsureExtracted(dest)
	if err != nil {
		return "", fmt.Errorf("failed to extract embedded adapters to %s: %w", dest, err)
	}
	_ = written
	if !hasAdapterYAML(dest) {
		return "", fmt.Errorf("embedded adapters extracted to %s but none were found", dest)
	}
	return dest, nil
}

// hasAdapterYAML reports whether dir contains at least one <platform>/adapter.yaml.
func hasAdapterYAML(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, e.Name(), "adapter.yaml")); err == nil {
			return true
		}
	}
	return false
}

// ExtractedAt reports the modification time of the extraction stamp, or the
// zero time when the bundle has never been extracted.
func ExtractedAt(destDir string) time.Time {
	if destDir == "" {
		destDir = DefaultDir
	}
	fi, err := os.Stat(filepath.Join(destDir, stampFileName))
	if err != nil {
		return time.Time{}
	}
	return fi.ModTime()
}
