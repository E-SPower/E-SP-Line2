package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// InstanceDirManager manages per-instance sandbox copies of adapter code under
// data/instances/<instance_id>/.
//
// Layout:
//
//	data/instances/<id>/adapter/    - sandboxed copy of adapters/<platform>/ (no __pycache__)
//	data/instances/<id>/manifest.json - SHA-256 digest of every file in adapter/
//	data/instances/<id>/state.json  - persisted dependency-install state
//
// The adapter runs from the sandbox copy so that the original adapters/ source
// can never be modified at runtime. If a file in the sandbox copy no longer
// matches its digest, the instance is refused to start (integrity check).
type InstanceDirManager struct {
	root       string // e.g. data/instances
	adaptersDir string // e.g. adapters
}

// NewInstanceDirManager creates a manager rooted at data/instances. The base
// dir is derived from the adapters dir location so it stays next to the DB
// (data/...).
func NewInstanceDirManager(adaptersDir string) *InstanceDirManager {
	return &InstanceDirManager{
		root:        filepath.Join(adaptersDir, "..", "data", "instances"),
		adaptersDir: adaptersDir,
	}
}

// instanceDir returns the sandbox root for a single instance.
func (m *InstanceDirManager) instanceDir(instanceID string) string {
	return filepath.Join(m.root, instanceID)
}

// AdapterDir returns the sandboxed adapter copy directory for an instance.
func (m *InstanceDirManager) AdapterDir(instanceID string) string {
	return filepath.Join(m.instanceDir(instanceID), "adapter")
}

// ManifestPath returns the digest manifest path for an instance.
func (m *InstanceDirManager) ManifestPath(instanceID string) string {
	return filepath.Join(m.instanceDir(instanceID), "manifest.json")
}

// StatePath returns the persisted install-state path for an instance.
func (m *InstanceDirManager) StatePath(instanceID string) string {
	return filepath.Join(m.instanceDir(instanceID), "state.json")
}

// CopyAdapter copies the adapter source directory (adapters/<platformCode>)
// into the instance sandbox, excluding __pycache__ directories and *.pyc files,
// and writes a SHA-256 manifest. Returns the sandbox adapter dir.
func (m *InstanceDirManager) CopyAdapter(instanceID, platformCode string) (string, error) {
	src := filepath.Join(m.adaptersDir, platformCode)
	if fi, err := os.Stat(src); err != nil || !fi.IsDir() {
		return "", fmt.Errorf("adapter source dir not found: %s", src)
	}
	dest := m.AdapterDir(instanceID)

	if err := os.RemoveAll(m.instanceDir(instanceID)); err != nil {
		return "", fmt.Errorf("failed to clear old instance dir: %w", err)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return "", err
	}

	// Copy all files, skipping __pycache__ and .pyc.
	if err := copyDirFiltered(src, dest); err != nil {
		return "", err
	}

	// Generate manifest.
	if err := m.writeManifest(instanceID, dest); err != nil {
		return "", err
	}

	return dest, nil
}

// copyDirFiltered recursively copies srcDir into destDir, skipping any
// __pycache__ directory or *.pyc file.
func copyDirFiltered(srcDir, destDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		src := filepath.Join(srcDir, e.Name())
		dst := filepath.Join(destDir, e.Name())
		if e.IsDir() {
			if e.Name() == "__pycache__" {
				continue
			}
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return err
			}
			if err := copyDirFiltered(src, dst); err != nil {
				return err
			}
			continue
		}
		// Skip compiled bytecode.
		if strings.HasSuffix(e.Name(), ".pyc") {
			continue
		}
		if err := copyFile(src, dst); err != nil {
			return err
		}
	}
	return nil
}

// copyFile copies a single file preserving permissions.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

// manifestEntry maps a relative path to its SHA-256 digest.
type manifestEntry struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
}

// writeManifest computes SHA-256 for every file under dir and writes the
// manifest to the instance dir.
func (m *InstanceDirManager) writeManifest(instanceID, dir string) error {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(files)

	entries := make([]manifestEntry, 0, len(files))
	for _, f := range files {
		rel, err := filepath.Rel(dir, f)
		if err != nil {
			return err
		}
		h, err := hashFile(f)
		if err != nil {
			return err
		}
		entries = append(entries, manifestEntry{Path: filepath.ToSlash(rel), Hash: h})
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.ManifestPath(instanceID), data, 0o644)
}

// hashFile returns the SHA-256 hex digest of a file.
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// VerifyIntegrity checks that every file in the sandbox adapter dir still
// matches its manifest digest. Returns an error (with details) if any file was
// added, removed or modified.
func (m *InstanceDirManager) VerifyIntegrity(instanceID string) error {
	dir := m.AdapterDir(instanceID)
	manifestPath := m.ManifestPath(instanceID)

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("instance sandbox manifest missing; instance not initialized")
		}
		return err
	}
	var entries []manifestEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	// Build current file map.
	current := map[string]string{}
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(dir, path)
		if rerr != nil {
			return rerr
		}
		h, herr := hashFile(path)
		if herr != nil {
			return herr
		}
		current[filepath.ToSlash(rel)] = h
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to scan sandbox: %w", err)
	}

	// Check each manifest entry still matches.
	for _, e := range entries {
		h, ok := current[e.Path]
		if !ok {
			return fmt.Errorf("sandbox file missing: %s", e.Path)
		}
		if h != e.Hash {
			return fmt.Errorf("sandbox file modified: %s", e.Path)
		}
	}
	// Check for unexpected extra files.
	for p := range current {
		// Python bytecode caches (__pycache__/*.pyc) are generated at runtime
		// and are not part of the manifest; they are safe to ignore.
		if isPythonCacheFile(p) {
			continue
		}
		found := false
		for _, e := range entries {
			if e.Path == p {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unexpected file in sandbox: %s", p)
		}
	}
	return nil
}

// isPythonCacheFile reports whether a path is a Python bytecode cache file
// (e.g. "__pycache__/module.cpython-314.pyc" or "module.pyc").
func isPythonCacheFile(p string) bool {
	base := filepath.Base(p)
	if strings.HasSuffix(base, ".pyc") || strings.HasSuffix(base, ".pyo") {
		return true
	}
	// Also ignore any file inside a __pycache__ directory.
	return strings.Contains(filepath.ToSlash(p), "/__pycache__/")
}

// Remove deletes the entire sandbox for an instance.
func (m *InstanceDirManager) Remove(instanceID string) error {
	return os.RemoveAll(m.instanceDir(instanceID))
}

// EnsureSandbox copies the adapter source into the instance sandbox if it does
// not exist yet (used for pre-existing instances on startup reconcile).
func (m *InstanceDirManager) EnsureSandbox(instanceID, platformCode string) (string, bool, error) {
	dest := m.AdapterDir(instanceID)
	if _, err := os.Stat(filepath.Join(dest, "main.py")); err == nil {
		if _, mErr := os.Stat(m.ManifestPath(instanceID)); mErr == nil {
			return dest, false, nil
		}
	}
	// Re-copy (missing or incomplete sandbox).
	dest, err := m.CopyAdapter(instanceID, platformCode)
	if err != nil {
		return "", false, err
	}
	return dest, true, nil
}
