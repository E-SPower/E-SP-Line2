package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

// testServer builds a real *Server and returns its router behind an httptest
// server, so the WS tests exercise the registered routes and their middleware
// chain rather than the handler functions in isolation.
func testServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()

	cfg := testConfig(t)
	cfg.JWT.Secret = "ws-test-secret"
	cfg.Server.AllowedOrigins = []string{"http://trusted.example"}

	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = srv.services.Close() })

	ts := httptest.NewServer(srv.router)
	t.Cleanup(ts.Close)

	// A token minted with the same secret the middleware was given.
	claims := jwt.MapClaims{
		"user_id":  "user-1",
		"username": "tester",
		"role":     "user",
		"exp":      time.Now().Add(time.Hour).Unix(),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(cfg.JWT.Secret))
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	return ts, token
}

// dialWS attempts a handshake and returns the error, if any.
func dialWS(t *testing.T, url, origin string) error {
	t.Helper()
	header := http.Header{}
	if origin != "" {
		header.Set("Origin", origin)
	}
	conn, resp, err := websocket.DefaultDialer.Dial(url, header)
	if conn != nil {
		_ = conn.Close()
	}
	if err != nil {
		return err
	}
	_ = resp
	return nil
}

// TestWSHandshakeRejectsMissingToken is the M4 end-to-end regression test: the
// socket route is now behind WSTokenRequired, so a bare handshake gets 401.
func TestWSHandshakeRejectsMissingToken(t *testing.T) {
	ts, _ := testServer(t)
	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/adapter?instance_id=x"

	_, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err == nil {
		t.Fatal("handshake without a token was accepted")
	}
	if resp == nil {
		t.Fatalf("no HTTP response to inspect: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

// TestWSHandshakeRejectsGarbageToken separates "some string was sent" from
// "a valid JWT was sent".
func TestWSHandshakeRejectsGarbageToken(t *testing.T) {
	ts, _ := testServer(t)
	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/adapter?instance_id=x&token=nope"

	_, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err == nil {
		t.Fatal("handshake with a garbage token was accepted")
	}
	if resp == nil {
		t.Fatalf("no HTTP response to inspect: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

// TestWSHandshakeRejectsUntrustedOrigin is the other M4 half: even a correctly
// authenticated handshake is refused from an origin that is not allow-listed.
func TestWSHandshakeRejectsUntrustedOrigin(t *testing.T) {
	ts, token := testServer(t)
	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/adapter?instance_id=x&token=" + token

	if err := dialWS(t, url, "https://evil.example"); err == nil {
		t.Fatal("cross-origin handshake was accepted")
	}
}

// TestWSHandshakeAcceptsValidTokenAndOrigin is the positive control; without it
// the three rejection tests could pass on a route that never works.
func TestWSHandshakeAcceptsValidTokenAndOrigin(t *testing.T) {
	ts, token := testServer(t)
	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/adapter?instance_id=x&token=" + token

	if err := dialWS(t, url, "http://trusted.example"); err != nil {
		t.Fatalf("allow-listed handshake rejected: %v", err)
	}
}

// TestWSAppRouteAlsoAuthenticated guards the second WS route. It used to share
// the permissive upgrader and no middleware at all.
func TestWSAppRouteAlsoAuthenticated(t *testing.T) {
	ts, token := testServer(t)
	base := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/app?app_id=app-1"

	if _, resp, err := websocket.DefaultDialer.Dial(base, nil); err == nil || resp == nil ||
		resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("app route without a token: err=%v status=%v, want 401", err, resp)
	}
	if err := dialWS(t, base+"&token="+token, "http://trusted.example"); err != nil {
		t.Fatalf("authenticated app handshake rejected: %v", err)
	}
}

// TestCORSPreflightOnRealRouter runs the M2 check through the assembled router,
// including its middleware order, not just the CORS function.
func TestCORSPreflightOnRealRouter(t *testing.T) {
	ts, _ := testServer(t)

	req, err := http.NewRequest(http.MethodOptions, ts.URL+"/api/v1/adapters", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Origin", "http://trusted.example")
	req.Header.Set("Access-Control-Request-Method", "GET")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://trusted.example" {
		t.Fatalf("Allow-Origin = %q, want the trusted origin", got)
	}

	untrusted, err := http.NewRequest(http.MethodOptions, ts.URL+"/api/v1/adapters", nil)
	if err != nil {
		t.Fatalf("new untrusted request: %v", err)
	}
	untrusted.Header.Set("Origin", "https://evil.example")
	resp2, err := http.DefaultClient.Do(untrusted)
	if err != nil {
		t.Fatalf("untrusted preflight: %v", err)
	}
	defer resp2.Body.Close()

	if got := resp2.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Allow-Origin = %q for an untrusted origin, want empty", got)
	}
}

// TestOversizeBodyGets413 runs the body cap through the real router so the
// middleware is proven to be registered, not merely to work standalone.
func TestOversizeBodyGets413(t *testing.T) {
	ts, _ := testServer(t)

	big := `{"username":"u","password":"` + strings.Repeat("x", 3<<20) + `"}`
	resp, err := http.Post(ts.URL+"/api/v1/auth/login", "application/json", strings.NewReader(big))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", resp.StatusCode)
	}
}

// TestOversizeBodyGets413OnIngestRoutes is the same guarantee for the two
// handlers that were written after BodyTooLarge became the convention and
// skipped it: message ingest and the instance command runner answered 400 with
// the raw MaxBytesReader text, so a client could not tell "too big" from
// "malformed". Both are 413 again.
func TestOversizeBodyGets413OnIngestRoutes(t *testing.T) {
	ts, token := testServer(t)

	big := `{"pad":"` + strings.Repeat("x", 3<<20) + `"}`
	for _, path := range []string{"/api/v1/messages", "/api/v1/instances/i1/command"} {
		req, err := http.NewRequest(http.MethodPost, ts.URL+path, strings.NewReader(big))
		if err != nil {
			t.Fatalf("%s: new request: %v", path, err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: post: %v", path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusRequestEntityTooLarge {
			t.Fatalf("%s: status = %d, want 413 (body %q)", path, resp.StatusCode, body)
		}
	}
}
