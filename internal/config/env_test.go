package config

import "testing"

// The env override exists so a deployment can set its Origin without editing
// config.yaml. It must reach the []string field, which viper only does with a
// comma-separated value and weak typing.
func TestAllowedOriginsEnvOverride(t *testing.T) {
	t.Setenv("ESP_SECURITY_ALLOWED_ORIGINS", "https://a.example,https://b.example")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Security.AllowedOrigins) != 2 {
		t.Fatalf("allowed origins = %v, want the two env values", cfg.Security.AllowedOrigins)
	}
	if !cfg.IsAllowedOrigin("https://b.example") {
		t.Fatalf("env origin not honoured: %v", cfg.Security.AllowedOrigins)
	}
}

func TestJWTSecretEnvOverride(t *testing.T) {
	t.Setenv("ESP_JWT_SECRET", "from-the-environment")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.JWT.Secret != "from-the-environment" {
		t.Fatalf("jwt.secret = %q, want the env value", cfg.JWT.Secret)
	}
	if cfg.UsingPlaceholderJWTSecret() {
		t.Fatal("rotated env secret still reported as placeholder")
	}
}
