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
	if sandboxUsable(adapterDir, "main.py") {
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
	if sandboxUsable(adapterDir, "main.py") {
		t.Fatal("sandbox without manifest must not be usable")
	}

	// Both present -> usable.
	if err := os.WriteFile(filepath.Join(base, "instances", "i1", "manifest.json"), []byte("[]"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !sandboxUsable(adapterDir, "main.py") {
		t.Fatal("complete sandbox must be usable")
	}
}

// TestSandboxUsableStandalone is the regression for the standalone runtime: its
// sandbox has no main.py, so the old main.py-only check declared a perfectly
// good sandbox broken and every start rebuilt it (RemoveAll), wiping the
// adapter's files and state.json.
func TestSandboxUsableStandalone(t *testing.T) {
	base := t.TempDir()
	adapterDir := filepath.Join(base, "instances", "i1", "adapter")
	if err := os.MkdirAll(adapterDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Only the standalone entry, deliberately no main.py.
	if err := os.WriteFile(filepath.Join(adapterDir, "main.mjs"), []byte("// stub\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "instances", "i1", "manifest.json"), []byte("[]"), 0o644); err != nil {
		t.Fatal(err)
	}

	spec := RuntimeSpec{Entry: "main.mjs"}
	entry := requiredEntry(RuntimeStandalone, spec)
	if entry != "main.mjs" {
		t.Fatalf("requiredEntry(standalone) = %q, want main.mjs", entry)
	}
	if !sandboxUsable(adapterDir, entry) {
		t.Fatal("standalone sandbox with its entry + manifest must be usable")
	}
	// The bridge entry must still be main.py and still be required.
	if got := requiredEntry(RuntimePython, RuntimeSpec{Entry: "main.mjs"}); got != "main.py" {
		t.Fatalf("requiredEntry(python) = %q, want main.py", got)
	}
	if sandboxUsable(adapterDir, requiredEntry(RuntimePython, spec)) {
		t.Fatal("a standalone sandbox must not pass as a bridge sandbox")
	}

	// Missing the declared entry -> not usable, even though the manifest exists.
	if err := os.Remove(filepath.Join(adapterDir, "main.mjs")); err != nil {
		t.Fatal(err)
	}
	if sandboxUsable(adapterDir, "main.mjs") {
		t.Fatal("sandbox without its entry must not be usable")
	}
	// A standalone spec without an entry cannot be judged usable.
	if sandboxUsable(adapterDir, requiredEntry(RuntimeStandalone, RuntimeSpec{})) {
		t.Fatal("standalone without runtime.entry must not be usable")
	}
}

// TestEnsureSandboxStandaloneKeepsInstanceState pins the actual damage: a
// second EnsureSandbox for a standalone instance must report "not recreated"
// and leave adapter-written files and state.json in place.
func TestEnsureSandboxStandaloneKeepsInstanceState(t *testing.T) {
	root := t.TempDir()
	adaptersDir := filepath.Join(root, "data", "adapters")
	dir := filepath.Join(adaptersDir, "catbus")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"main.mjs":     "// stub\n",
		"adapter.yaml": "id: catbus-adapter\nplatform_code: xianyu\nruntime_type: standalone\nruntime:\n  entry: main.mjs\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	oldWD, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	m := NewInstanceDirManager("data/adapters")
	entry := requiredEntry(RuntimeStandalone, RuntimeSpec{Entry: "main.mjs"})
	dest, created, err := m.EnsureSandbox("i1", "catbus", entry)
	if err != nil || !created {
		t.Fatalf("first EnsureSandbox = (%q, %v, %v), want created", dest, created, err)
	}

	// Files the adapter and the installer wrote into the instance dir.
	written := filepath.Join(dest, "adapter-state.json")
	if err := os.WriteFile(written, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(m.StatePath("i1"), []byte(`{"status":"done"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	again, recreated, err := m.EnsureSandbox("i1", "catbus", entry)
	if err != nil {
		t.Fatal(err)
	}
	if recreated {
		t.Fatal("an intact standalone sandbox must not be re-copied")
	}
	if again != dest {
		t.Fatalf("EnsureSandbox moved the sandbox: %q != %q", again, dest)
	}
	if data, err := os.ReadFile(written); err != nil || string(data) != "keep me" {
		t.Fatalf("adapter-written file lost on the second pass: %q, %v", data, err)
	}
	if data, err := os.ReadFile(m.StatePath("i1")); err != nil || string(data) != `{"status":"done"}` {
		t.Fatalf("state.json lost on the second pass: %q, %v", data, err)
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

	if !sandboxUsable(dest, "main.py") {
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
