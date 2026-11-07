package config

import "testing"

func TestUsingPlaceholderJWTSecret(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		want   bool
	}{
		{"shipped placeholder", PlaceholderJWTSecret, true},
		{"empty secret", "", true},
		{"rotated secret", "s3cr3t-from-the-deployment", false},
		// A secret that merely contains the placeholder text is still a real
		// rotation and must not warn.
		{"placeholder with suffix", PlaceholderJWTSecret + "-prod", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{JWT: JWTConfig{Secret: tt.secret}}
			if got := cfg.UsingPlaceholderJWTSecret(); got != tt.want {
				t.Fatalf("UsingPlaceholderJWTSecret() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAllowedOrigin(t *testing.T) {
	tests := []struct {
		name    string
		allowed []string
		origin  string
		want    bool
	}{
		{"listed origin", []string{"https://app.example"}, "https://app.example", true},
		{"unlisted origin", []string{"https://app.example"}, "https://evil.example", false},
		{"empty origin is a non-browser client", []string{"https://app.example"}, "", true},
		{"empty allowlist falls back to allow-all", nil, "https://anything.example", true},
		{"prefix does not match", []string{"https://app.example"}, "https://app.example.evil", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Security: SecurityConfig{AllowedOrigins: tt.allowed}}
			if got := cfg.IsAllowedOrigin(tt.origin); got != tt.want {
				t.Fatalf("IsAllowedOrigin(%q) = %v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}

// TestDefaultsAreSafe pins the shipped defaults: without an explicit config the
// server must still have every timeout set and a non-empty Origin allowlist.
func TestDefaultsAreSafe(t *testing.T) {
	cfg := &Config{Security: SecurityConfig{AllowedOrigins: DefaultAllowedOrigins()}}
	if len(cfg.Security.AllowedOrigins) == 0 {
		t.Fatal("default Origin allowlist is empty")
	}
	if !cfg.IsAllowedOrigin("http://localhost:3000") {
		t.Fatal("the Vite dev server origin is not allowed by default")
	}
	if cfg.IsAllowedOrigin("https://evil.example") {
		t.Fatal("an arbitrary origin is allowed by default")
	}
}
