package service

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFakeAdapter creates adapters/<code>/main.py plus adapter.yaml so the
// sandbox manager can copy it.
func writeFakeAdapter(t *testing.T, adaptersDir, code string) {
	t.Helper()
	dir := filepath.Join(adaptersDir, code)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"main.py":      "# fake adapter\nprint('ok')\n",
		"adapter.yaml": "id: " + code + "-adapter\nplatform_code: " + code + "\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestSandboxUsable covers the completeness check used before self-healing.
func TestSandboxUsable(t *testing.T) {
	// Missing sandbox.
	base := t.TempDir()
	adapterDir := filepath.Join(base, "instances", "i1", "adapter")
	if sandboxUsable(adapterDir) {
		t.Fatal("non-existent sandbox must not be usable")
	}

	// main.py present but manifest missing -> not usable (integrity check
	// would fail right after).
	if err := os.MkdirAll(adapterDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(adapterDir, "main.py"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if sandboxUsable(adapterDir) {
		t.Fatal("sandbox without manifest must not be usable")
	}

	// Both present -> usable.
	if err := os.WriteFile(filepath.Join(base, "instances", "i1", "manifest.json"), []byte("[]"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !sandboxUsable(adapterDir) {
		t.Fatal("complete sandbox must be usable")
	}
}

// TestCopyAdapterCreatesUsableSandbox is the core of the self-healing path: a
// sandbox built from the adapter source must pass the usability check, i.e.
// main.py and manifest.json both exist.
func TestCopyAdapterCreatesUsableSandbox(t *testing.T) {
	root := t.TempDir()
	adaptersDir := filepath.Join(root, "data", "adapters")
	writeFakeAdapter(t, adaptersDir, "taobao")

	// Run from root so the relative data dir lands inside the temp tree.
	oldWD, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	m := NewInstanceDirManager("data/adapters")
	dest, err := m.CopyAdapter("i1", "taobao")
	if err != nil {
		t.Fatalf("CopyAdapter failed: %v", err)
	}

	if !sandboxUsable(dest) {
		t.Fatalf("sandbox created by CopyAdapter is not usable: %s", dest)
	}
	if _, err := os.Stat(filepath.Join(dest, "main.py")); err != nil {
		t.Fatalf("main.py missing in sandbox: %v", err)
	}
	if _, err := os.Stat(m.ManifestPath("i1")); err != nil {
		t.Fatalf("manifest missing: %v", err)
	}

	// Integrity must pass on a freshly created sandbox.
	if err := m.VerifyIntegrity("i1"); err != nil {
		t.Fatalf("fresh sandbox failed integrity check: %v", err)
	}
}

// TestCopyAdapterRejectsMissingSource documents the error the runner surfaces
// when self-healing is impossible (adapter source absent).
func TestCopyAdapterRejectsMissingSource(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	m := NewInstanceDirManager("data/adapters")
	if _, err := m.CopyAdapter("i1", "does-not-exist"); err == nil {
		t.Fatal("expected error for missing adapter source")
	}
}
