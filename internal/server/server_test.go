package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/e-spl/e-sp-line2/internal/config"
)

// TestNewAppliesServerTimeouts is the M15 regression test: the http.Server must
// carry the hardening timeouts, not just an Addr and a Handler.
//
// It reads the built *http.Server off the constructor so a later refactor that
// drops a field fails here instead of only under load.
func TestNewAppliesServerTimeouts(t *testing.T) {
	cfg := testConfig(t)
	cfg.Server.ReadHeaderTimeout = 3
	cfg.Server.ReadTimeout = 11
	cfg.Server.WriteTimeout = 13
	cfg.Server.IdleTimeout = 17

	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = srv.services.Close() }()

	cases := []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{"ReadHeaderTimeout", srv.httpServer.ReadHeaderTimeout, 3 * time.Second},
		{"ReadTimeout", srv.httpServer.ReadTimeout, 11 * time.Second},
		{"WriteTimeout", srv.httpServer.WriteTimeout, 13 * time.Second},
		{"IdleTimeout", srv.httpServer.IdleTimeout, 17 * time.Second},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s = %v, want %v", tc.name, tc.got, tc.want)
		}
	}
}

// TestNewTranslatesZeroTimeoutToDisabled keeps the "0 means off" contract: a
// zero in config must not become a zero-second deadline, which would expire
// every connection instantly if handed to net/http as-is.
func TestNewTranslatesZeroTimeoutToDisabled(t *testing.T) {
	cfg := testConfig(t)
	cfg.Server.ReadHeaderTimeout = 0
	cfg.Server.ReadTimeout = 0
	cfg.Server.WriteTimeout = 0
	cfg.Server.IdleTimeout = 0

	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = srv.services.Close() }()

	if srv.httpServer.ReadHeaderTimeout != 0 || srv.httpServer.ReadTimeout != 0 ||
		srv.httpServer.WriteTimeout != 0 || srv.httpServer.IdleTimeout != 0 {
		t.Fatalf("zero config produced non-zero timeouts: %+v", srv.httpServer)
	}
}

// TestNewBuildsSharedOriginPolicy proves CORS and the WS upgrader were handed
// the same policy object. Two independently-built policies drift, and the drift
// shows up as "REST works, socket 403s".
func TestNewBuildsSharedOriginPolicy(t *testing.T) {
	cfg := testConfig(t)
	cfg.Server.AllowedOrigins = []string{"http://trusted.example"}

	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = srv.services.Close() }()

	if srv.originPolicy == nil {
		t.Fatal("no origin policy on the server")
	}
	if !srv.originPolicy.Allows("http://trusted.example") {
		t.Error("configured origin not allowed by the server policy")
	}
	if srv.originPolicy.Allows("https://evil.example") {
		t.Error("unlisted origin allowed by the server policy")
	}
}

// chdir moves into dir for the duration of the test. testing.Chdir needs
// go1.24 while go.mod still declares 1.22, so this does it by hand.
func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(old); err != nil {
			t.Fatalf("restore wd: %v", err)
		}
	})
}

// testConfig is a minimal config pointing at a throwaway SQLite file so New can
// build the services stack without touching the developer's data/ directory.
//
// The form-options registry is a hard startup dependency, so the temp dir gets
// a stub copy before anything runs.
func testConfig(t *testing.T) *config.Config {
	t.Helper()

	dir := t.TempDir()
	chdir(t, dir)
	if err := os.MkdirAll(filepath.Join(dir, "config"), 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config", "form-options.yaml"), []byte("platform_codes: []\n"), 0o600); err != nil {
		t.Fatalf("write form options: %v", err)
	}

	return &config.Config{
		Host:     "127.0.0.1",
		Port:     0,
		Version:  "test",
		LogLevel: "error",
		Database: config.DatabaseConfig{Driver: "sqlite", DSN: "test.db"},
		JWT:      config.JWTConfig{Secret: "test-secret", Expiration: 1},
		Adapter:  config.AdapterConfig{AdaptersDir: "adapters", PythonBin: "python3"},
		Server: config.ServerConfig{
			AllowedOrigins:    []string{"http://localhost:3000"},
			ReadHeaderTimeout: 5,
			ReadTimeout:       60,
			WriteTimeout:      60,
			IdleTimeout:       120,
			MaxBodyBytes:      1 << 20,
		},
	}
}
