package middleware

import (
	"net/http"
	"net/url"
)

// OriginPolicy decides which browser origins an HTTP response and a WebSocket
// handshake accept.
//
// CORS and CheckOrigin answer the same question, so they read one policy built
// once at startup from the same config list. Two copies of that decision drift:
// a working REST call plus a rejected socket is the failure mode.
type OriginPolicy struct {
	// allowAny is set by a "*" entry and means "do not check".
	allowAny bool
	// allowed is matched exactly: scheme + host + port, no trailing slash. A
	// prefix match would let `http://localhost:3000.evil.example` pass for
	// `http://localhost:3000`.
	allowed map[string]struct{}
}

// NewOriginPolicy builds a policy from the configured allow-list.
//
// An empty list allows no *cross* origin. Same-origin still works for the
// WebSocket handshake — see CheckOrigin — and for CORS it never needed an entry
// because the browser sends no Origin on a same-origin request.
func NewOriginPolicy(origins []string) *OriginPolicy {
	p := &OriginPolicy{allowed: make(map[string]struct{}, len(origins))}
	for _, origin := range origins {
		if origin == "*" {
			p.allowAny = true
			continue
		}
		p.allowed[origin] = struct{}{}
	}
	return p
}

// Allows reports whether the literal origin string may talk to us. An empty
// origin (curl, a native client, a same-origin GET) is always allowed: no page
// is in a position to forge it, which is why it needs no entry.
func (p *OriginPolicy) Allows(origin string) bool {
	if origin == "" {
		return true
	}
	if p == nil {
		return false
	}
	if p.allowAny {
		return true
	}
	_, ok := p.allowed[origin]
	return ok
}

// AllowsRequest is the single decision CORS and the upgrader both consume.
//
// Beyond the list it accepts a same-origin request. Browsers omit Origin on a
// same-origin GET (so CORS never needed the entry) but always send it on a
// handshake, so the two surfaces cannot share only Allows() without one of
// them needing a URL for the deployment's own host.
func (p *OriginPolicy) AllowsRequest(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if p.Allows(origin) {
		return true
	}
	return sameOrigin(r, origin)
}

// AllowsAny reports whether the policy was configured with "*". CORS uses it
// to decide between echoing the origin and sending a literal "*".
func (p *OriginPolicy) AllowsAny() bool {
	return p != nil && p.allowAny
}

// CheckOrigin is the websocket.Upgrader hook.
//
// A rejected origin fails the handshake with 403 before any app-level bytes are
// exchanged. The decision itself is AllowsRequest, shared with CORS.
func (p *OriginPolicy) CheckOrigin(r *http.Request) bool {
	return p.AllowsRequest(r)
}

// sameOrigin reports whether origin names the host the request was addressed
// to. Only the host:port is compared, because the scheme the browser reports
// and the scheme this process sees differ whenever TLS is terminated upstream.
func sameOrigin(r *http.Request, origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return false
	}
	return parsed.Host == r.Host
}
