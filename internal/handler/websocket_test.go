package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/e-spl/e-sp-line2/internal/middleware"
	"github.com/e-spl/e-sp-line2/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

// The handlers log through pkg/logger, which is a nil zap logger until Init.
func TestMain(m *testing.M) {
	logger.Init("error")
	code := m.Run()
	logger.Sync()
	os.Exit(code)
}

const testSecret = "test-secret-for-websocket-handshakes"

func init() {
	gin.SetMode(gin.TestMode)
	middleware.SetJWTSecret(testSecret)
}

func signTestToken(t *testing.T, expires time.Duration) string {
	t.Helper()
	claims := &middleware.Claims{
		UserID:   "user-1",
		Username: "tester",
		Role:     "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expires)),
		},
	}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}
	return s
}

// newWSServer starts a real HTTP server around the app socket so the handshake
// (including the upgrader's Origin check) is exercised end to end.
func newWSServer(t *testing.T) *httptest.Server {
	t.Helper()
	router := gin.New()
	router.GET("/ws/app", AppWebSocket(nil))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv
}

// wsURL builds an app-socket URL. app_id is always present so the handler does
// not reject the connection after a successful upgrade for an unrelated reason.
func wsURL(srv *httptest.Server, query string) string {
	u := strings.Replace(srv.URL, "http://", "ws://", 1) + "/ws/app?app_id=app-1"
	if query != "" {
		u += "&" + query
	}
	return u
}

// dial opens a socket and returns the first frame, so a successful handshake is
// proven by the welcome message rather than merely by no error.
func dial(t *testing.T, rawURL string, header http.Header) (*websocket.Conn, map[string]interface{}) {
	t.Helper()
	conn, resp, err := websocket.DefaultDialer.Dial(rawURL, header)
	if err != nil {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("dial %s failed (%d): %v", rawURL, status, err)
	}
	t.Cleanup(func() { conn.Close() })
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var frame map[string]interface{}
	if err := conn.ReadJSON(&frame); err != nil {
		t.Fatalf("reading welcome frame: %v", err)
	}
	return conn, frame
}

func expectDialFails(t *testing.T, rawURL string, header http.Header, wantStatus int) {
	t.Helper()
	conn, resp, err := websocket.DefaultDialer.Dial(rawURL, header)
	if err == nil {
		conn.Close()
		t.Fatalf("dial %s succeeded, want failure with %d", rawURL, wantStatus)
	}
	if resp == nil || resp.StatusCode != wantStatus {
		got := 0
		if resp != nil {
			got = resp.StatusCode
		}
		t.Fatalf("handshake status = %d, want %d", got, wantStatus)
	}
}

func TestAppWebSocketRequiresValidToken(t *testing.T) {
	srv := newWSServer(t)
	token := signTestToken(t, time.Hour)

	t.Run("missing token is rejected before upgrade", func(t *testing.T) {
		expectDialFails(t, wsURL(srv, ""), nil, http.StatusUnauthorized)
	})

	t.Run("garbage token is rejected", func(t *testing.T) {
		expectDialFails(t, wsURL(srv, "token=total-garbage"), nil, http.StatusUnauthorized)
	})

	t.Run("expired token is rejected", func(t *testing.T) {
		expired := signTestToken(t, -time.Hour)
		expectDialFails(t, wsURL(srv, "token="+url.QueryEscape(expired)), nil, http.StatusUnauthorized)
	})

	t.Run("valid token via query param", func(t *testing.T) {
		_, frame := dial(t, wsURL(srv, "token="+url.QueryEscape(token)), nil)
		if frame["type"] != "connected" {
			t.Fatalf("welcome frame = %v, want type=connected", frame)
		}
	})

	t.Run("valid token via Authorization header", func(t *testing.T) {
		h := http.Header{}
		h.Set("Authorization", "Bearer "+token)
		if _, frame := dial(t, wsURL(srv, ""), h); frame["type"] != "connected" {
			t.Fatalf("welcome frame = %v", frame)
		}
	})

	t.Run("valid token via the espl subprotocol", func(t *testing.T) {
		h := http.Header{}
		h.Set("Sec-WebSocket-Protocol", "espl, "+token)
		if _, frame := dial(t, wsURL(srv, ""), h); frame["type"] != "connected" {
			t.Fatalf("welcome frame = %v", frame)
		}
	})
}

// A fabricated token must never be treated the same as no token: that was the
// original hole (CheckOrigin always true, no middleware at all).
func TestAppWebSocketOriginAllowlist(t *testing.T) {
	t.Cleanup(func() { SetAllowedOrigin(nil) })
	SetAllowedOrigin(func(o string) bool { return o == "https://app.example" })

	srv := newWSServer(t)
	token := signTestToken(t, time.Hour)

	t.Run("listed origin is accepted", func(t *testing.T) {
		h := http.Header{}
		h.Set("Origin", "https://app.example")
		if _, frame := dial(t, wsURL(srv, "token="+url.QueryEscape(token)), h); frame["type"] != "connected" {
			t.Fatalf("welcome frame = %v", frame)
		}
	})

	t.Run("unlisted origin is refused", func(t *testing.T) {
		h := http.Header{}
		h.Set("Origin", "https://evil.example")
		expectDialFails(t, wsURL(srv, "token="+url.QueryEscape(token)), h, http.StatusForbidden)
	})

	t.Run("no Origin header is a non-browser client", func(t *testing.T) {
		if _, frame := dial(t, wsURL(srv, "token="+url.QueryEscape(token)), nil); frame["type"] != "connected" {
			t.Fatalf("welcome frame = %v", frame)
		}
	})
}
