package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestStartupWarnsOnPlaceholderSecret is the M3 acceptance test in its
// strongest form: it starts the real binary with the shipped secret and asserts
// the WARN is actually printed.
//
// The unit tests in internal/config cover the predicate; this one covers the
// wiring — main has to call it after logger.Init, and a refactor that drops
// that call must not stay green.
func TestStartupWarnsOnPlaceholderSecret(t *testing.T) {
	runStartup(t, "change-this-secret-in-production", "placeholder")
}

// TestStartupIsSilentWithRealSecret is the control. Without it, a build that
// warns unconditionally would pass the test above.
func TestStartupIsSilentWithRealSecret(t *testing.T) {
	runStartup(t, "a-real-deployment-secret", "")
}

func runStartup(t *testing.T, secret, wantInLog string) {
	t.Helper()
	if testing.Short() {
		t.Skip("builds and runs the binary; skipped under -short")
	}

	bin := filepath.Join(t.TempDir(), "espl")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Env = os.Environ()
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "config"), 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	cfg := "host: \"127.0.0.1\"\n" +
		"port: 0\n" +
		"database:\n  driver: sqlite\n  dsn: \"test.db\"\n" +
		"log_level: info\n" +
		"jwt:\n  secret: \"" + secret + "\"\n  expiration: 1\n" +
		"adapter:\n  adapters_dir: adapters\n  python_bin: python3\n"
	if err := os.WriteFile(filepath.Join(dir, "config", "config.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	// Startup has a hard dependency on the form-options registry.
	if err := os.WriteFile(filepath.Join(dir, "config", "form-options.yaml"), []byte("platform_codes: []\n"), 0o600); err != nil {
		t.Fatalf("write form options: %v", err)
	}

	cmd := exec.Command(bin)
	cmd.Dir = dir
	out := &strings.Builder{}
	cmd.Stdout = out
	cmd.Stderr = out

	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	// Give it long enough to get past logger.Init and either warn or bind.
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(out.String(), "Starting E-SP-Line2 server") {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	time.Sleep(500 * time.Millisecond) // let the warn line flush

	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()

	log := out.String()
	if !strings.Contains(log, "Starting E-SP-Line2 server") {
		t.Fatalf("server did not reach startup; log:\n%s", log)
	}

	if wantInLog == "placeholder" {
		if !strings.Contains(log, "WARN") || !strings.Contains(log, "placeholder") {
			t.Fatalf("no WARN about the placeholder secret; log:\n%s", log)
		}
		if !strings.Contains(log, "jwt.secret") {
			t.Fatalf("warning does not name jwt.secret; log:\n%s", log)
		}
		return
	}

	if strings.Contains(log, "placeholder") {
		t.Fatalf("unexpected placeholder warning with a real secret; log:\n%s", log)
	}
}
