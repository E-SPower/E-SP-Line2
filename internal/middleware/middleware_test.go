package middleware

import (
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

// TestCORSAllowedOriginEchoed is the M2 regression test: a listed origin gets
// its own value echoed back, not "*", which is what makes the paired
// Allow-Credentials: true actually mean something.
func TestCORSAllowedOriginEchoed(t *testing.T) {
	policy := NewOriginPolicy([]string{"http://localhost:3000"})

	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/adapters", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	router := gin.New()
	router.Use(CORS(policy))
	router.GET("/api/v1/adapters", func(c *gin.Context) { c.Status(nethttp.StatusOK) })
	router.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Allow-Origin = %q, want the echoed origin", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Allow-Credentials = %q, want true on an echoed origin", got)
	}
	if got := rec.Header().Get("Vary"); !strings.Contains(got, "Origin") {
		t.Fatalf("Vary = %q, want it to include Origin", got)
	}
}

// TestCORSUnlistedOriginHasNoAllowOrigin is the other half of M2: an origin
// nobody listed must not receive an allow header at all.
func TestCORSUnlistedOriginHasNoAllowOrigin(t *testing.T) {
	policy := NewOriginPolicy([]string{"http://localhost:3000"})

	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/adapters", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()

	router := gin.New()
	router.Use(CORS(policy))
	router.GET("/api/v1/adapters", func(c *gin.Context) { c.Status(nethttp.StatusOK) })
	router.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Allow-Origin = %q, want empty for an unlisted origin", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("Allow-Credentials = %q, want empty without an allowed origin", got)
	}
}

// TestCORSWildcard covers the escape hatch. Even under "*" the origin is
// echoed rather than sent as a literal star, because the echoed form keeps
// Allow-Credentials meaningful; the wildcard only skips the membership check.
func TestCORSWildcard(t *testing.T) {
	policy := NewOriginPolicy([]string{"*"})

	req := httptest.NewRequest(nethttp.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://anything.example")
	rec := httptest.NewRecorder()

	router := gin.New()
	router.Use(CORS(policy))
	router.GET("/x", func(c *gin.Context) { c.Status(nethttp.StatusOK) })
	router.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://anything.example" {
		t.Fatalf("Allow-Origin = %q, want the echoed origin under a wildcard policy", got)
	}
	// With no Origin at all there is nothing to echo, so the literal is used.
	plain := httptest.NewRequest(nethttp.MethodGet, "/x", nil)
	plainRec := httptest.NewRecorder()
	router.ServeHTTP(plainRec, plain)
	if got := plainRec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Allow-Origin without an Origin = %q, want *", got)
	}
}

// TestCORSPreflight: an OPTIONS preflight for an allowed origin is answered
// 204 with the allow headers, which is what the browser checks before sending
// the real request.
func TestCORSPreflight(t *testing.T) {
	policy := NewOriginPolicy([]string{"http://localhost:3000"})

	req := httptest.NewRequest(nethttp.MethodOptions, "/api/v1/adapters", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()

	router := gin.New()
	router.Use(CORS(policy))
	router.GET("/api/v1/adapters", func(c *gin.Context) { t.Fatal("preflight reached the handler") })
	router.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("preflight Allow-Origin = %q, want the echoed origin", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, "PUT") {
		t.Fatalf("Allow-Methods = %q, want the method list", got)
	}
}

// TestOriginPolicyAllowsEmptyOrigin: same-origin navigation, curl and native
// clients send no Origin and must keep working without an allow-list entry.
func TestOriginPolicyAllowsEmptyOrigin(t *testing.T) {
	policy := NewOriginPolicy([]string{"http://localhost:3000"})
	if !policy.Allows("") {
		t.Fatal("empty origin rejected; same-origin clients would break")
	}
}

// TestOriginPolicyCheckOrigin drives the upgrader hook the WS handlers use.
func TestOriginPolicyCheckOrigin(t *testing.T) {
	policy := NewOriginPolicy([]string{"http://localhost:3000"})

	req := httptest.NewRequest(nethttp.MethodGet, "/ws/adapter", nil)
	req.Header.Set("Origin", "https://evil.example")
	if policy.CheckOrigin(req) {
		t.Error("cross-origin handshake accepted")
	}

	req.Header.Set("Origin", "http://localhost:3000")
	if !policy.CheckOrigin(req) {
		t.Error("allow-listed origin rejected")
	}
}

// TestOriginPolicyCheckOriginSameHost covers the case that made a bare
// allow-list unusable: a browser sends Origin on a *same-origin* handshake too,
// and a deployment serving the UI from the API's own port must not have to list
// its own URL.
func TestOriginPolicyCheckOriginSameHost(t *testing.T) {
	policy := NewOriginPolicy(nil) // nothing configured at all

	req := httptest.NewRequest(nethttp.MethodGet, "http://example.test:8080/ws/adapter", nil)
	req.Header.Set("Origin", "http://example.test:8080")
	if !policy.CheckOrigin(req) {
		t.Fatal("same-origin handshake rejected; browsers always send Origin")
	}

	// Port still distinguishes, so a different port is cross-origin.
	req.Header.Set("Origin", "http://example.test:9999")
	if policy.CheckOrigin(req) {
		t.Fatal("different port treated as same-origin")
	}

	// A host that merely ends in the right name is not the same origin.
	req.Header.Set("Origin", "http://example.test:8080.evil.example")
	if policy.CheckOrigin(req) {
		t.Fatal("suffix-extended host treated as same-origin")
	}
}

// mintToken produces a real signed token through the same code path the server
// uses, so the WS test is not passing on a string shape.
func mintToken(t *testing.T, userID string) string {
	t.Helper()
	SetJWTSecret("test-secret")
	token, err := signForTest(userID)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	return token
}

// TestWSTokenRequired is the M4 acceptance test: the handshake is refused
// without a valid token and accepted with one.
func TestWSTokenRequired(t *testing.T) {
	valid := mintToken(t, "user-1")

	tests := []struct {
		name     string
		query    string
		header   string
		wantCode int
		wantPass bool
	}{
		{"no token at all", "", "", nethttp.StatusUnauthorized, false},
		{"garbage token", "?token=not-a-jwt", "", nethttp.StatusUnauthorized, false},
		{"valid query token", "?token=" + valid, "", nethttp.StatusOK, true},
		{"valid bearer header", "", "Bearer " + valid, nethttp.StatusOK, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(nethttp.MethodGet, "/ws/adapter"+tt.query, nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()

			router := gin.New()
			var passed bool
			var gotUser string
			router.GET("/ws/adapter", WSTokenRequired(), func(c *gin.Context) {
				passed = true
				if value, ok := c.Get("userID"); ok {
					gotUser, _ = value.(string)
				}
				c.Status(nethttp.StatusOK)
			})
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantCode)
			}
			if passed != tt.wantPass {
				t.Fatalf("handler reached = %v, want %v", passed, tt.wantPass)
			}
			if tt.wantPass && gotUser != "user-1" {
				t.Fatalf("context userID = %q, want the claim value", gotUser)
			}
		})
	}
}

// TestMaxBodyBytesRejectsOversizeBody is the M15 test for the body cap: a body
// past the limit must not be read whole.
func TestMaxBodyBytesRejectsOversizeBody(t *testing.T) {
	router := gin.New()
	router.Use(MaxBodyBytes(64))
	router.POST("/login", func(c *gin.Context) {
		var body map[string]any
		if err := c.ShouldBindJSON(&body); err != nil {
			if !BodyTooLarge(c, err) {
				c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
			}
			return
		}
		c.Status(nethttp.StatusOK)
	})

	big := `{"pad":"` + strings.Repeat("x", 200) + `"}`
	req := httptest.NewRequest(nethttp.MethodPost, "/login", strings.NewReader(big))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", rec.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if resp["error"] == "" {
		t.Fatal("413 carried no error message")
	}
}

// TestMaxBodyBytesAllowsSmallBody guards against the cap breaking normal input.
func TestMaxBodyBytesAllowsSmallBody(t *testing.T) {
	router := gin.New()
	router.Use(MaxBodyBytes(1 << 20))
	router.POST("/login", func(c *gin.Context) { c.Status(nethttp.StatusOK) })

	req := httptest.NewRequest(nethttp.MethodPost, "/login", strings.NewReader(`{"a":1}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

// TestMaxBodyBytesSkipsGET keeps the read-only path untouched.
func TestMaxBodyBytesSkipsGET(t *testing.T) {
	router := gin.New()
	router.Use(MaxBodyBytes(1))
	router.GET("/health", func(c *gin.Context) { c.Status(nethttp.StatusOK) })

	req := httptest.NewRequest(nethttp.MethodGet, "/health", strings.NewReader(strings.Repeat("x", 100)))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want 200 — GET body should not be capped", rec.Code)
	}
}
