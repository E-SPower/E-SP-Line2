package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/e-spl/e-sp-line2/internal/message"
	"github.com/e-spl/e-sp-line2/internal/middleware"
	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/internal/repository"
	"github.com/e-spl/e-sp-line2/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

// These cases pin the two WebSocket routes to the hub. Before the fix both routes
// only echoed, `internal/message` had zero importers, and no frame ever reached
// the database or another connection.

type wsFixture struct {
	router *gin.Engine
	db     *gorm.DB
	hub    *message.Hub
	server *httptest.Server
}

func newWSFixture(t *testing.T) *wsFixture {
	t.Helper()

	r, db := newIngestRouter(t)
	repos := &repository.Repositories{
		DB:           db,
		InboundEvent: repository.NewInboundEventRepository(db),
		Instance:     repository.NewInstanceRepository(db),
	}
	svc := service.NewMessageService(repos.InboundEvent, repos.Instance)
	hub := message.NewHub()
	t.Cleanup(func() { hub.Close() })
	svc.SetHub(hub)

	cmdSvc := service.NewCommandService(repository.NewOutboundCommandRepository(db))
	cmdSvc.SetHub(hub)
	r.POST("/commands", CreateCommand(cmdSvc))

	r.GET("/ws/adapter", AdapterWebSocket(hub, nil, svc, nil))
	r.GET("/ws/app", AppWebSocket(hub, svc))

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	return &wsFixture{router: r, db: db, hub: hub, server: srv}
}

func (f *wsFixture) dial(t *testing.T, path string) *websocket.Conn {
	t.Helper()

	url := "ws" + strings.TrimPrefix(f.server.URL, "http") + path + "&token=" + signTestToken(t, time.Hour)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", path, err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func readFrame(t *testing.T, conn *websocket.Conn) map[string]interface{} {
	t.Helper()

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var frame map[string]interface{}
	if err := json.Unmarshal(raw, &frame); err != nil {
		t.Fatalf("decode frame %q: %v", raw, err)
	}
	return frame
}

func writeFrame(t *testing.T, conn *websocket.Conn, frame map[string]interface{}) {
	t.Helper()

	raw, err := json.Marshal(frame)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func (f *wsFixture) postCommand(t *testing.T, body interface{}) *httptest.ResponseRecorder {
	t.Helper()

	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/commands", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	return w
}

func (f *wsFixture) allow(instanceID, targetType, targetID string) {
	f.hub.SyncRules([]models.RouteRule{{
		ID:         "rule-1",
		PlatformID: "xianyu",
		TargetType: targetType,
		TargetID:   targetID,
		Enabled:    true,
	}})
	_ = instanceID
}

// B2 on the bridge side: a frame reported by a bridge is persisted, and the
// generic ACK still comes back with the stored event ID.
func TestAdapterWebSocketPersistsFrame(t *testing.T) {
	f := newWSFixture(t)

	conn := f.dial(t, "/ws/adapter?instance_id=inst-1")
	if welcome := readFrame(t, conn); welcome["type"] != "connected" {
		t.Fatalf("first frame = %v, want the connected handshake", welcome["type"])
	}

	writeFrame(t, conn, map[string]interface{}{
		"platform_id":     "xianyu",
		"conversation_id": "conv-1",
		"sender_id":       "buyer-1",
		"sender_name":     "buyer",
		"message_type":    "text",
		"message_content": "在吗",
	})

	ack := readFrame(t, conn)
	if ack["type"] != "ack" {
		t.Fatalf("frame = %v, want ack", ack)
	}
	eventID, _ := ack["event_id"].(string)
	if eventID == "" {
		t.Fatal("ack carries no event_id, so nothing was persisted")
	}

	var stored models.InboundEvent
	if err := f.db.First(&stored, "id = ?", eventID).Error; err != nil {
		t.Fatalf("stored row %s not found: %v", eventID, err)
	}
	if stored.MessageContent != "在吗" {
		t.Fatalf("stored content = %q, want 在吗", stored.MessageContent)
	}
	if stored.InstanceID != "inst-1" {
		t.Fatalf("stored instance = %q, want the query parameter", stored.InstanceID)
	}
}

// B2 across connections: the receiving peer does not have to be the sender. A
// bridge frame routed to a registered app arrives on that app's socket, and the
// stored row records the delivery. This is the end-to-end case the review asked
// for ("a frame in, an extra row, the peer receives it").
func TestAdapterFrameRoutedToAppSocket(t *testing.T) {
	f := newWSFixture(t)
	f.allow("inst-1", "app", "app-1")

	app := f.dial(t, "/ws/app?app_id=app-1")
	if welcome := readFrame(t, app); welcome["type"] != "connected" {
		t.Fatalf("app handshake = %v", welcome)
	}

	// An unreachable rule would silently "route" without this: the app is
	// registered before the bridge sends anything.
	bridge := f.dial(t, "/ws/adapter?instance_id=inst-1")
	readFrame(t, bridge) // handshake

	writeFrame(t, bridge, map[string]interface{}{
		"platform_id":     "xianyu",
		"conversation_id": "conv-1",
		"sender_id":       "buyer-1",
		"message_type":    "text",
		"message_content": "转发给我",
	})

	// The bridge itself still gets its own ACK.
	if ack := readFrame(t, bridge); ack["type"] != "ack" {
		t.Fatalf("bridge frame = %v, want ack", ack)
	}

	// And the app receives the routed envelope — not the bridge's own echo.
	routed := readFrame(t, app)
	if got, _ := routed["event_type"].(string); got != "message.received" {
		t.Fatalf("app received frame %v, want a routed message.received", routed)
	}

	var count int64
	if err := f.db.Model(&models.InboundEvent{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("stored %d rows, want 1", count)
	}

	// The row must say delivered: that status is what proves the hub reached a
	// connection rather than only matching a rule.
	var stored models.InboundEvent
	if err := f.db.First(&stored).Error; err != nil {
		t.Fatalf("load stored: %v", err)
	}
	if stored.Status != "delivered" {
		t.Fatalf("stored status = %q, want delivered", stored.Status)
	}
}

// B5 end to end: a command created over HTTP reaches the connected bridge as a
// flat bridge-protocol frame, and the row is recorded as sent.
//
// The frame shape matters as much as the delivery: bridges read `command_type`
// and `payload.cid/toid/text` off the top level (adapters/xianyu/main.py:197),
// so a nested v3 envelope arrives and is dropped as "指令缺少 cid/toid".
func TestCreateCommandDispatchesToBridge(t *testing.T) {
	f := newWSFixture(t)

	bridge := f.dial(t, "/ws/adapter?instance_id=inst-1")
	readFrame(t, bridge) // handshake

	f.postCommand(t, map[string]interface{}{
		"instance_id":  "inst-1",
		"command_type": "send_text",
		"payload": map[string]interface{}{
			"conversation_id": "conv-1",
			"target_id":       "buyer-1",
			"text":            "您好，有货",
		},
	})

	_ = bridge.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, raw, err := bridge.ReadMessage()
	if err != nil {
		t.Fatalf("bridge read: %v", err)
	}

	var frame map[string]interface{}
	if err := json.Unmarshal(raw, &frame); err != nil {
		t.Fatalf("decode command frame: %v", err)
	}
	if got, _ := frame["command_type"].(string); got != "send_text" {
		t.Fatalf("command_type = %q (frame %s), bridges drop anything else", got, raw)
	}
	payload, _ := frame["payload"].(map[string]interface{})
	if payload == nil {
		t.Fatalf("frame has no payload: %s", raw)
	}
	if payload["cid"] != "conv-1" || payload["toid"] != "buyer-1" {
		t.Fatalf("payload missing cid/toid: %s", raw)
	}
	if payload["text"] != "您好，有货" {
		t.Fatalf("payload text = %v", payload["text"])
	}

	var stored models.OutboundCommand
	if err := f.db.First(&stored).Error; err != nil {
		t.Fatalf("load command: %v", err)
	}
	if stored.Status != "sent" {
		t.Fatalf("status = %q, want sent", stored.Status)
	}
	if stored.SentAt == nil {
		t.Fatal("sent_at not stamped")
	}
}

// A command with no live bridge must be parked, not silently left `created`.
func TestCreateCommandWithoutBridgeParksRetrying(t *testing.T) {
	f := newWSFixture(t)

	w := f.postCommand(t, map[string]interface{}{
		"instance_id":  "inst-nobody",
		"command_type": "send_text",
		"payload":      map[string]interface{}{"conversation_id": "c", "target_id": "t", "text": "x"},
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}

	var stored models.OutboundCommand
	if err := f.db.First(&stored).Error; err != nil {
		t.Fatalf("load command: %v", err)
	}
	if stored.Status != "retrying" {
		t.Fatalf("status = %q, want retrying", stored.Status)
	}
	if stored.RetryCount != 1 {
		t.Fatalf("retry count = %d, want 1", stored.RetryCount)
	}
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
	r, db := newIngestRouter(t)
	hub := message.NewHub()
	t.Cleanup(func() { _ = hub.Close() })
	svc := service.NewMessageService(repository.NewInboundEventRepository(db), repository.NewInstanceRepository(db))
	svc.SetHub(hub)
	router := r
	router.GET("/ws/app", AppWebSocket(hub, svc))
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
func dialPlain(t *testing.T, rawURL string, header http.Header) (*websocket.Conn, map[string]interface{}) {
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
		_, frame := dialPlain(t, wsURL(srv, "token="+url.QueryEscape(token)), nil)
		if frame["type"] != "connected" {
			t.Fatalf("welcome frame = %v, want type=connected", frame)
		}
	})

	t.Run("valid token via Authorization header", func(t *testing.T) {
		h := http.Header{}
		h.Set("Authorization", "Bearer "+token)
		if _, frame := dialPlain(t, wsURL(srv, ""), h); frame["type"] != "connected" {
			t.Fatalf("welcome frame = %v", frame)
		}
	})

	t.Run("valid token via the espl subprotocol", func(t *testing.T) {
		h := http.Header{}
		h.Set("Sec-WebSocket-Protocol", "espl, "+token)
		if _, frame := dialPlain(t, wsURL(srv, ""), h); frame["type"] != "connected" {
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
		if _, frame := dialPlain(t, wsURL(srv, "token="+url.QueryEscape(token)), h); frame["type"] != "connected" {
			t.Fatalf("welcome frame = %v", frame)
		}
	})

	t.Run("unlisted origin is refused", func(t *testing.T) {
		h := http.Header{}
		h.Set("Origin", "https://evil.example")
		expectDialFails(t, wsURL(srv, "token="+url.QueryEscape(token)), h, http.StatusForbidden)
	})

	t.Run("no Origin header is a non-browser client", func(t *testing.T) {
		if _, frame := dialPlain(t, wsURL(srv, "token="+url.QueryEscape(token)), nil); frame["type"] != "connected" {
			t.Fatalf("welcome frame = %v", frame)
		}
	})
}
