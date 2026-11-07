package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeConfig drops a config.yaml into a temp dir laid out the way the loader
// searches for it, and points the working directory at it.
func writeConfig(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	chdir(t, dir)
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

// TestJWTSecretWarning is the acceptance test for M3: the shipped placeholder
// secret must be reported at startup, and a real secret must not be.
func TestJWTSecretWarning(t *testing.T) {
	tests := []struct {
		name     string
		secret   string
		wantWarn bool
	}{
		{"shipped placeholder warns", PlaceholderJWTSecret, true},
		{"empty secret warns", "", true},
		{"a changed secret is silent", "s3cr3t-from-the-operator", false},
		{"a secret that only looks similar is silent", "change-this-secret-in-production-2", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JWTSecretWarning(tt.secret)
			if tt.wantWarn && got == "" {
				t.Fatalf("JWTSecretWarning(%q) = \"\", want a warning", tt.secret)
			}
			if !tt.wantWarn && got != "" {
				t.Fatalf("JWTSecretWarning(%q) = %q, want no warning", tt.secret, got)
			}
		})
	}
}

// TestLoadKeepsPlaceholderSecret proves the loader really returns the shipped
// value when the config file leaves it alone — otherwise the warning above
// would never fire in a real deployment.
func TestLoadKeepsPlaceholderSecret(t *testing.T) {
	writeConfig(t, "port: 8080\n")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.JWT.Secret != PlaceholderJWTSecret {
		t.Fatalf("JWT.Secret = %q, want the placeholder %q", cfg.JWT.Secret, PlaceholderJWTSecret)
	}
	if warning := JWTSecretWarning(cfg.JWT.Secret); warning == "" {
		t.Fatal("default config produced no placeholder warning")
	}
}

// TestLoadChangedSecretIsSilent is the other half: setting a secret in the file
// must suppress the warning.
func TestLoadChangedSecretIsSilent(t *testing.T) {
	writeConfig(t, "jwt:\n  secret: \"a-real-deployment-secret\"\n")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.JWT.Secret != "a-real-deployment-secret" {
		t.Fatalf("JWT.Secret = %q, want the configured value", cfg.JWT.Secret)
	}
	if warning := JWTSecretWarning(cfg.JWT.Secret); warning != "" {
		t.Fatalf("configured secret still warned: %s", warning)
	}
}

// TestLoadServerDefaults pins the hardening defaults: an operator who never
// touches config/config.yaml must still get timeouts and a body cap rather than
// the pre-M15 zero server.
func TestLoadServerDefaults(t *testing.T) {
	writeConfig(t, "port: 8080\n")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.ReadHeaderTimeout <= 0 {
		t.Error("ReadHeaderTimeout default is not positive")
	}
	if cfg.Server.ReadTimeout <= 0 {
		t.Error("ReadTimeout default is not positive")
	}
	if cfg.Server.WriteTimeout <= 0 {
		t.Error("WriteTimeout default is not positive")
	}
	if cfg.Server.IdleTimeout <= 0 {
		t.Error("IdleTimeout default is not positive")
	}
	if cfg.Server.MaxBodyBytes <= 0 {
		t.Error("MaxBodyBytes default is not positive")
	}
	if len(cfg.Server.AllowedOrigins) == 0 {
		t.Error("AllowedOrigins default is empty; the Vite dev server would be rejected")
	}
}

// TestOriginAllowed checks the allow-list predicate the CORS middleware and the
// WebSocket upgrader share.
func TestOriginAllowed(t *testing.T) {
	cfg := &ServerConfig{AllowedOrigins: []string{"http://localhost:3000"}}

	if !cfg.OriginAllowed("http://localhost:3000") {
		t.Error("listed origin rejected")
	}
	// A prefix match would wrongly accept this one.
	if cfg.OriginAllowed("http://localhost:3000.evil.example") {
		t.Error("suffix-extended origin accepted")
	}
	if cfg.OriginAllowed("https://localhost:3000") {
		t.Error("different scheme accepted")
	}
	if cfg.OriginAllowed("http://localhost:3001") {
		t.Error("different port accepted")
	}

	wildcard := &ServerConfig{AllowedOrigins: []string{"*"}}
	if !wildcard.OriginAllowed("https://evil.example") {
		t.Error("wildcard did not allow an unlisted origin")
	}
}

// TestAllowedOriginsStringForm covers the `server.allowed_origins: "..."` form
// somebody will inevitably write (a single string rather than a list).
func TestAllowedOriginsStringForm(t *testing.T) {
	writeConfig(t, "server:\n  allowed_origins: \"https://ui.example\"\n")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.Server.OriginAllowed("https://ui.example") {
		t.Fatalf("single-string allowed_origins not loaded: %v", cfg.Server.AllowedOrigins)
	}
	if cfg.Server.OriginAllowed("http://localhost:3000") {
		t.Fatalf("file value did not replace the default list: %v", cfg.Server.AllowedOrigins)
	}
}

// TestWarningNamesTheFix keeps the message actionable; a bare "change me" sends
// the operator looking for the key.
func TestWarningNamesTheFix(t *testing.T) {
	warning := JWTSecretWarning(PlaceholderJWTSecret)
	for _, want := range []string{"jwt.secret", "config/config.yaml", "JWT.SECRET"} {
		if !strings.Contains(warning, want) {
			t.Errorf("warning does not mention %q: %s", want, warning)
		}
	}
}
