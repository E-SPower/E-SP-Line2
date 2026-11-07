package service

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// Per-instance egress proxy for the sandboxed adapter process.
//
// The adapter sandbox (data/instances/<id>/adapter/) runs a Python process that
// talks to the platform's web endpoints. Some networks can only reach those
// endpoints through a proxy, so an instance may opt in to one. The proxy is per
// instance — never global — so two instances of the same adapter can egress
// through different proxies, or none at all.
//
// The instance config accepts either shape:
//
//	{"proxy": "http://user:pass@host:8080", "no_proxy": "localhost,127.0.0.1"}
//	{"proxy": {"host": "host", "port": 8080, "username": "u", "password": "p"}}
//
// An absent or empty proxy means "inherit": the sandbox keeps whatever proxy the
// backend process itself was started with. `proxy: "none"` forces a direct
// connection and neutralises an ambient proxy.
const (
	// proxyNone is the sentinel that forces a direct connection.
	proxyNone = "none"

	// defaultNoProxy keeps the adapter's own loopback connection to the backend
	// WebSocket off the proxy.
	defaultNoProxy = "localhost,127.0.0.1,::1"
)

// ProxyConfig is the resolved egress proxy of one instance.
type ProxyConfig struct {
	// URL is a complete proxy URL, or empty for "inherit", or proxyNone for
	// "force direct".
	URL string
	// NoProxy is the comma-separated bypass list.
	NoProxy string
}

// Enabled reports whether an explicit proxy URL is configured.
func (p ProxyConfig) Enabled() bool {
	return p.URL != "" && p.URL != proxyNone
}

// Direct reports whether the instance forces a direct connection.
func (p ProxyConfig) Direct() bool {
	return p.URL == proxyNone
}

// String returns a loggable form with the password masked.
func (p ProxyConfig) String() string {
	switch {
	case p.Direct():
		return "direct (forced)"
	case p.URL == "":
		return "inherit"
	}
	return maskProxyPassword(p.URL)
}

// Env returns the proxy environment variables for the sandbox process, in both
// lower- and upper-case spellings: requests/aiohttp/websockets read the
// lower-case names, curl- and Go-style tooling read the upper-case ones.
//
// Nothing configured yields nil, so the caller just appends it to the inherited
// environment. A forced-direct instance writes empty values instead, because
// that is what overrides an ambient proxy.
func (p ProxyConfig) Env() []string {
	if p.URL == "" {
		return nil
	}

	keys := []string{
		"ALL_PROXY", "all_proxy",
		"HTTP_PROXY", "http_proxy",
		"HTTPS_PROXY", "https_proxy",
	}
	value := p.URL
	if p.Direct() {
		value = ""
	}

	env := make([]string, 0, len(keys)+2)
	for _, key := range keys {
		env = append(env, key+"="+value)
	}

	noProxy := p.NoProxy
	if noProxy == "" {
		noProxy = defaultNoProxy
	}
	env = append(env, "NO_PROXY="+noProxy, "no_proxy="+noProxy)
	return env
}

// proxyFromInstanceConfig resolves the proxy of an instance from its stored
// config object. An unusable shape is an error so the user hears about it when
// saving instead of running direct by accident.
func proxyFromInstanceConfig(cfg map[string]interface{}) (ProxyConfig, error) {
	out := ProxyConfig{}

	if raw, ok := cfg["no_proxy"]; ok {
		if s, ok := raw.(string); ok {
			out.NoProxy = strings.TrimSpace(s)
		}
	}

	raw, ok := cfg["proxy"]
	if !ok || raw == nil {
		return out, nil
	}

	switch v := raw.(type) {
	case string:
		out.URL = strings.TrimSpace(v)
	case map[string]interface{}:
		url, err := proxyURLFromObject(v)
		if err != nil {
			return out, err
		}
		out.URL = url
	default:
		return out, fmt.Errorf("proxy must be a URL string or an object, got %T", raw)
	}

	if out.URL == "" {
		return out, nil
	}
	if strings.EqualFold(out.URL, proxyNone) {
		out.URL = proxyNone
		return out, nil
	}
	if err := validateProxyURL(out.URL); err != nil {
		return out, err
	}
	return out, nil
}

// proxyURLFromObject builds a proxy URL out of the structured form. The
// username and password are optional and are escaped into the userinfo part.
func proxyURLFromObject(obj map[string]interface{}) (string, error) {
	host, _ := obj["host"].(string)
	host = strings.TrimSpace(host)
	if host == "" {
		return "", nil
	}

	scheme := "http"
	if raw, ok := obj["scheme"].(string); ok && strings.TrimSpace(raw) != "" {
		scheme = strings.ToLower(strings.TrimSpace(raw))
	}

	port := ""
	switch v := obj["port"].(type) {
	case float64:
		if v != 0 {
			port = strconv.Itoa(int(v))
		}
	case int:
		if v != 0 {
			port = strconv.Itoa(v)
		}
	case string:
		port = strings.TrimSpace(v)
	}
	if port != "" {
		if _, err := strconv.Atoi(port); err != nil {
			return "", fmt.Errorf("proxy port %q is not a number", port)
		}
	}

	authority := host
	if port != "" {
		authority = net.JoinHostPort(host, port)
	}

	userinfo := ""
	if username, _ := obj["username"].(string); username != "" {
		userinfo = escapeUserinfo(username)
		if password, _ := obj["password"].(string); password != "" {
			userinfo += ":" + escapeUserinfo(password)
		}
		userinfo += "@"
	}

	url := scheme + "://" + userinfo + authority
	if err := validateProxyURL(url); err != nil {
		return "", err
	}
	return url, nil
}

// escapeUserinfo percent-escapes the characters that would otherwise end the
// userinfo or the authority part.
func escapeUserinfo(s string) string {
	return strings.NewReplacer(
		"%", "%25",
		"@", "%40",
		":", "%3A",
		"/", "%2F",
		"#", "%23",
		"?", "%3F",
		" ", "%20",
	).Replace(s)
}

// validateProxyURL checks the scheme and host of a proxy URL. A missing port is
// allowed; the HTTP client then applies the scheme's default.
func validateProxyURL(raw string) error {
	scheme, rest, ok := strings.Cut(raw, "://")
	if !ok {
		return fmt.Errorf("proxy %q has no scheme; use http://, https:// or socks5://", maskProxyPassword(raw))
	}
	switch strings.ToLower(scheme) {
	case "http", "https", "socks5", "socks5h":
	default:
		return fmt.Errorf("proxy scheme %q is not supported; use http://, https:// or socks5://", scheme)
	}
	if strings.TrimSpace(rest) == "" {
		return fmt.Errorf("proxy %q has no host", maskProxyPassword(raw))
	}
	return nil
}

// maskProxyPassword replaces the password inside a proxy URL with "***" so the
// value can go into the instance log.
func maskProxyPassword(raw string) string {
	scheme, rest, ok := strings.Cut(raw, "://")
	if !ok {
		return raw
	}
	at := strings.LastIndex(rest, "@")
	if at < 0 {
		return raw
	}
	userinfo, authority := rest[:at], rest[at+1:]
	if user, _, found := strings.Cut(userinfo, ":"); found {
		userinfo = user + ":***"
	}
	return scheme + "://" + userinfo + "@" + authority
}
