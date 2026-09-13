package adapters

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFingerprintDeterministic guards the stability of the extraction stamp:
// a non-deterministic fingerprint would re-extract on every startup.
func TestFingerprintDeterministic(t *testing.T) {
	a, err := fingerprint()
	if err != nil {
		t.Fatalf("fingerprint failed: %v", err)
	}
	b, err := fingerprint()
	if err != nil {
		t.Fatalf("fingerprint failed: %v", err)
	}
	if a != b {
		t.Fatalf("fingerprint not deterministic: %s != %s", a, b)
	}
}

// TestEnsureExtractedIdempotent verifies that extraction happens once and that a
// second call is a no-op (no files rewritten), which is what keeps normal
// startups cheap.
func TestEnsureExtractedIdempotent(t *testing.T) {
	if !Available() {
		t.Skip("no adapters embedded in this build (sync step not run)")
	}

	dir := filepath.Join(t.TempDir(), "adapters")

	n1, err := EnsureExtracted(dir)
	if err != nil {
		t.Fatalf("first extract failed: %v", err)
	}
	if n1 == 0 {
		t.Fatal("expected files to be written on first extract")
	}
	if _, err := os.Stat(filepath.Join(dir, stampFileName)); err != nil {
		t.Fatalf("stamp not written: %v", err)
	}

	n2, err := EnsureExtracted(dir)
	if err != nil {
		t.Fatalf("second extract failed: %v", err)
	}
	if n2 != 0 {
		t.Fatalf("expected no writes on second extract, got %d", n2)
	}
}

// TestEffectiveDirPrefersExternal confirms the developer override: a real
// external adapters/ directory wins over the embedded copy.
func TestEffectiveDirPrefersExternal(t *testing.T) {
	ext := t.TempDir()
	pf := filepath.Join(ext, "demo")
	if err := os.MkdirAll(pf, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pf, "adapter.yaml"), []byte("id: demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := EffectiveDir(ext)
	if err != nil {
		t.Fatalf("EffectiveDir failed: %v", err)
	}
	if got != ext {
		t.Fatalf("expected external dir %s, got %s", ext, got)
	}
}

// TestEffectiveDirFallsBackToEmbedded confirms that with no external adapters
// the embedded bundle is extracted and used.
func TestEffectiveDirFallsBackToEmbedded(t *testing.T) {
	if !Available() {
		t.Skip("no adapters embedded in this build")
	}

	// Run from a directory with no adapters/ so the external check fails.
	base := t.TempDir()
	oldWD, _ := os.Getwd()
	if err := os.Chdir(base); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	got, err := EffectiveDir("adapters")
	if err != nil {
		t.Fatalf("EffectiveDir failed: %v", err)
	}
	if got != DefaultDir {
		t.Fatalf("expected %s, got %s", DefaultDir, got)
	}
	if !hasAdapterYAML(DefaultDir) {
		t.Fatalf("expected extracted adapters under %s", DefaultDir)
	}
}
