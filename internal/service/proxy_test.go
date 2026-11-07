package service

import (
	"strings"
	"testing"
)

// TestProxyFromInstanceConfigPlainURL covers the plain-string form.
func TestProxyFromInstanceConfigPlainURL(t *testing.T) {
	got, err := proxyFromInstanceConfig(map[string]interface{}{
		"proxy": "http://127.0.0.1:7890",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Enabled() {
		t.Fatal("expected proxy to be enabled")
	}
	if got.URL != "http://127.0.0.1:7890" {
		t.Fatalf("unexpected url %q", got.URL)
	}
}

// TestProxyFromInstanceConfigObject covers the structured form, including
// credential escaping.
func TestProxyFromInstanceConfigObject(t *testing.T) {
	got, err := proxyFromInstanceConfig(map[string]interface{}{
		"proxy": map[string]interface{}{
			"scheme":   "socks5",
			"host":     "proxy.example.com",
			"port":     float64(1080),
			"username": "user@corp",
			"password": "p@ss:word",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "socks5://user%40corp:p%40ss%3Aword@proxy.example.com:1080"
	if got.URL != want {
		t.Fatalf("url = %q, want %q", got.URL, want)
	}
}

// TestProxyFromInstanceConfigAbsentAndEmpty keeps the inherit default.
func TestProxyFromInstanceConfigAbsentAndEmpty(t *testing.T) {
	for name, cfg := range map[string]map[string]interface{}{
		"absent": {},
		"empty":  {"proxy": ""},
		"null":   {"proxy": nil},
	} {
		got, err := proxyFromInstanceConfig(cfg)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", name, err)
		}
		if got.URL != "" || got.Enabled() || got.Direct() {
			t.Fatalf("%s: expected inherit, got %q", name, got.URL)
		}
		if env := got.Env(); env != nil {
			t.Fatalf("%s: expected no env for inherit, got %v", name, env)
		}
	}
}

// TestProxyFromInstanceConfigNone forces direct and neutralises ambient proxies.
func TestProxyFromInstanceConfigNone(t *testing.T) {
	got, err := proxyFromInstanceConfig(map[string]interface{}{"proxy": "none"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Direct() || got.Enabled() {
		t.Fatalf("expected forced direct, got %+v", got)
	}
	env := strings.Join(got.Env(), " ")
	for _, key := range []string{"HTTP_PROXY=", "http_proxy=", "HTTPS_PROXY=", "ALL_PROXY="} {
		if !strings.Contains(env, key) {
			t.Fatalf("expected %s to be cleared in %q", key, env)
		}
	}
}

// TestProxyEnvKeepsLoopbackDirect guards the backend WebSocket loop: loopback
// must be in NO_PROXY even when the user configured a proxy.
func TestProxyEnvKeepsLoopbackDirect(t *testing.T) {
	got, err := proxyFromInstanceConfig(map[string]interface{}{"proxy": "http://proxy:8080"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	env := strings.Join(got.Env(), " ")
	if !strings.Contains(env, "NO_PROXY="+defaultNoProxy) {
		t.Fatalf("expected default no_proxy in %q", env)
	}
	if !strings.Contains(env, "127.0.0.1") {
		t.Fatalf("loopback missing from no_proxy: %q", env)
	}
}

// TestProxyEnvHonoursExplicitNoProxy keeps a user override.
func TestProxyEnvHonoursExplicitNoProxy(t *testing.T) {
	got, err := proxyFromInstanceConfig(map[string]interface{}{
		"proxy":    "http://proxy:8080",
		"no_proxy": "corp.example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	env := strings.Join(got.Env(), " ")
	if !strings.Contains(env, "NO_PROXY=corp.example.com") {
		t.Fatalf("expected explicit no_proxy in %q", env)
	}
}

// TestProxyFromInstanceConfigRejectsBadInput keeps malformed values loud.
func TestProxyFromInstanceConfigRejectsBadInput(t *testing.T) {
	cases := map[string]map[string]interface{}{
		"no scheme":  {"proxy": "127.0.0.1:7890"},
		"bad scheme": {"proxy": "ftp://host:21"},
		"no host":    {"proxy": "http://"},
		"bad port":   {"proxy": map[string]interface{}{"host": "h", "port": "abc"}},
		"wrong type": {"proxy": float64(7890)},
	}
	for name, cfg := range cases {
		if _, err := proxyFromInstanceConfig(cfg); err == nil {
			t.Fatalf("%s: expected an error", name)
		}
	}
}

// TestMaskProxyPassword keeps credentials out of the log.
func TestMaskProxyPassword(t *testing.T) {
	got := maskProxyPassword("http://user:secret@proxy:8080")
	if strings.Contains(got, "secret") {
		t.Fatalf("password leaked: %q", got)
	}
	if got != "http://user:***@proxy:8080" {
		t.Fatalf("unexpected mask %q", got)
	}
}

// TestProxyStringForLog covers the three log forms.
func TestProxyStringForLog(t *testing.T) {
	if got := (ProxyConfig{}).String(); got != "inherit" {
		t.Fatalf("inherit form = %q", got)
	}
	if got := (ProxyConfig{URL: proxyNone}).String(); got != "direct (forced)" {
		t.Fatalf("direct form = %q", got)
	}
	got := ProxyConfig{URL: "http://u:p@h:1"}.String()
	if strings.Contains(got, ":p@") {
		t.Fatalf("password leaked: %q", got)
	}
}
