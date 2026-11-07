package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

// do runs one request through a freshly built engine with the middleware under
// test and reports the response. The handler is only there to make "the request
// reached the application" observable.
func do(t *testing.T, mw gin.HandlerFunc, method, origin, body string) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	router.Use(mw)
	router.Any("/x", func(c *gin.Context) { c.String(http.StatusOK, "reached") })

	req := httptest.NewRequest(method, "/x", strings.NewReader(body))
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func allowOnly(allowed ...string) func(string) bool {
	return func(o string) bool {
		for _, a := range allowed {
			if a == o {
				return true
			}
		}
		return false
	}
}

func TestCORSEchoesAllowedOrigin(t *testing.T) {
	rec := do(t, CORS(allowOnly("https://app.example")), http.MethodGet, "https://app.example", "")

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example" {
		t.Fatalf("Allow-Origin = %q, want the echoed origin (not a wildcard)", got)
	}
	// With the Origin echoed, credentials are legal and must stay allowed.
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Allow-Credentials = %q, want true", got)
	}
	// A response that varies by Origin must say so, or a shared cache serves
	// one client's CORS headers to another.
	if got := rec.Header().Get("Vary"); !strings.Contains(got, "Origin") {
		t.Fatalf("Vary = %q, want it to contain Origin", got)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestCORSRejectsUnlistedOrigin(t *testing.T) {
	rec := do(t, CORS(allowOnly("https://app.example")), http.MethodGet, "https://evil.example", "")

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Allow-Origin = %q, want empty for an unlisted origin", got)
	}
	// The request still reaches the app; the browser is what blocks it.
	if rec.Code != http.StatusOK || rec.Body.String() != "reached" {
		t.Fatalf("unlisted origin should reach the handler, got %d %q", rec.Code, rec.Body.String())
	}
}

func TestCORSPreflight(t *testing.T) {
	rec := do(t, CORS(allowOnly("https://app.example")), http.MethodOptions, "https://app.example", "")

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, "POST") {
		t.Fatalf("Allow-Methods = %q, want it to include POST", got)
	}
	if rec.Body.String() != "" {
		t.Fatalf("preflight should not reach the handler, got body %q", rec.Body.String())
	}
}

// A request with no Origin at all (curl, a Python bridge) is not a browser
// request and must be left completely alone.
func TestCORSWithoutOrigin(t *testing.T) {
	rec := do(t, CORS(allowOnly("https://app.example")), http.MethodGet, "", "")

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Allow-Origin = %q, want empty when no Origin was sent", got)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestMaxBodyBytes(t *testing.T) {
	const limit = 8

	t.Run("under the limit passes", func(t *testing.T) {
		rec := do(t, MaxBodyBytes(limit), http.MethodPost, "", "12345678")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("declared oversize is rejected before the handler runs", func(t *testing.T) {
		rec := do(t, MaxBodyBytes(limit), http.MethodPost, "", "123456789")
		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, want 413 for a declared oversized body", rec.Code)
		}
		if rec.Body.String() == "reached" {
			t.Fatal("an oversized body should not reach the handler")
		}
	})

	t.Run("undeclared oversize is caught while reading", func(t *testing.T) {
		// A handler must read the body for the cap to trip; the engine here is
		// built by do(), so use a bespoke one that reads it fully.
		router := gin.New()
		router.Use(MaxBodyBytes(limit))
		router.POST("/x", func(c *gin.Context) {
			if _, err := io.ReadAll(c.Request.Body); err != nil {
				c.String(http.StatusRequestEntityTooLarge, "too large")
				return
			}
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("123456789"))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, want 413 for an oversized body", rec.Code)
		}
	})

	t.Run("zero limit disables the cap", func(t *testing.T) {
		rec := do(t, MaxBodyBytes(0), http.MethodPost, "", "123456789")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})
}
