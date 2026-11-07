package handler

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/e-spl/e-sp-line2/internal/message"
	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/internal/repository"
	"github.com/e-spl/e-sp-line2/internal/service"
	"github.com/e-spl/e-sp-line2/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func TestMain(m *testing.M) {
	logger.Init("error")
	gin.SetMode(gin.TestMode)
	m.Run()
}

// wsFixture is a running gin server with the real websocket handlers, a real
// hub and a real (in-memory) message service behind them.
type wsFixture struct {
	hub     *message.Hub
	msgSvc  *service.MessageService
	srv     *httptest.Server
	appURL  string
	adpURL  string
	cleanup []func()
}

func newWSFixture(t *testing.T) *wsFixture {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_busy_timeout=5000"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(&models.Platform{}, &models.AdapterInstance{}, &models.InboundEvent{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, table := range []string{"inbound_events", "adapter_instances", "platforms"} {
		if err := db.Exec("DELETE FROM " + table).Error; err != nil {
			t.Fatalf("clear %s: %v", table, err)
		}
	}

	repos := &repository.Repositories{
		DB:           db,
		InboundEvent: repository.NewInboundEventRepository(db),
		Instance:     repository.NewInstanceRepository(db),
	}

	hub := message.NewHub()
	msgSvc := service.NewMessageService(repos.InboundEvent, repos.Instance)
	msgSvc.SetHub(hub)

	router := gin.New()
	router.GET("/ws/adapter", AdapterWebSocket(hub, msgSvc, nil))
	router.GET("/ws/app", AppWebSocket(hub, msgSvc))

	srv := httptest.NewServer(router)
	wsBase := "ws" + strings.TrimPrefix(srv.URL, "http")

	f := &wsFixture{
		hub:    hub,
		msgSvc: msgSvc,
		srv:    srv,
		appURL: wsBase + "/ws/app?app_id=",
		adpURL: wsBase + "/ws/adapter?instance_id=",
	}
	f.cleanup = append(f.cleanup,
		func() { srv.Close() },
		func() { _ = hub.Close() },
		func() { _ = sqlDB.Close() },
	)
	t.Cleanup(func() {
		for i := len(f.cleanup) - 1; i >= 0; i-- {
			f.cleanup[i]()
		}
	})

	return f
}

// dial opens a client socket and consumes the server's welcome frame so later
// reads only see message traffic.
func dial(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", url, err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var welcome map[string]interface{}
	if err := conn.ReadJSON(&welcome); err != nil {
		t.Fatalf("welcome frame: %v", err)
	}
	if welcome["type"] != "connected" {
		t.Fatalf("welcome type = %v, want connected", welcome["type"])
	}
	return conn
}

// waitForConnections blocks until the hub reports the expected registration, so
// the test never races the handler goroutine.
func waitForConnections(t *testing.T, hub *message.Hub, connType string, want int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if hub.ConnectionCount(connType) == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s connections = %d, want %d", connType, hub.ConnectionCount(connType), want)
}

// B2: a bridge frame must land in the database as a message AND be delivered to
// the routed app peer. Before the fix both sockets only echoed the sender's own
// frame back and nothing was persisted.
func TestAdapterFrameIsPersistedAndDeliveredToApp(t *testing.T) {
	f := newWSFixture(t)

	// Rule: any message reaches app "app-1".
	f.hub.SyncRules([]*message.RouteRule{{
		ID:         "rule-1",
		TargetType: "app",
		TargetID:   "app-1",
		Enabled:    true,
	}})

	app := dial(t, f.appURL+"app-1")
	waitForConnections(t, f.hub, "app", 1)

	adapter := dial(t, f.adpURL+"instance-1")
	waitForConnections(t, f.hub, "adapter", 1)

	frame := map[string]interface{}{
		"protocol_version": "v3",
		"event_id":         "evt-integration",
		"event_type":       "message.received",
		"platform_id":      "platform-xianyu",
		"payload": map[string]interface{}{
			"conversation_id": "conv-1",
			"sender_id":       "buyer-1",
			"sender_name":     "Buyer",
			"message_type":    "text",
			"message_content": "还在吗",
			"idempotency_key": "xianyu-buyer-1-conv-1-1",
		},
	}
	if err := adapter.WriteJSON(frame); err != nil {
		t.Fatalf("adapter send: %v", err)
	}

	// 1) The app peer receives the routed envelope.
	_ = app.SetReadDeadline(time.Now().Add(3 * time.Second))
	var got map[string]interface{}
	if err := app.ReadJSON(&got); err != nil {
		t.Fatalf("app did not receive the routed frame: %v", err)
	}
	if got["event_type"] != "message.received" {
		t.Fatalf("event_type = %v, want message.received", got["event_type"])
	}
	payload, _ := got["payload"].(map[string]interface{})
	if payload == nil || payload["message_content"] != "还在吗" {
		t.Fatalf("delivered payload = %#v, want the message content", got["payload"])
	}
	if payload["instance_id"] != "instance-1" {
		t.Fatalf("delivered instance_id = %v, want instance-1", payload["instance_id"])
	}

	// 2) The frame is in the database, and the bridge was acknowledged.
	all, _, err := f.msgSvc.List(10, 0)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("stored messages = %d, want 1", len(all))
	}
	if all[0].MessageContent != "还在吗" || all[0].InstanceID != "instance-1" {
		t.Fatalf("stored row = %#v", all[0])
	}

	_ = adapter.SetReadDeadline(time.Now().Add(3 * time.Second))
	var ack map[string]interface{}
	if err := adapter.ReadJSON(&ack); err != nil {
		t.Fatalf("no ack for the bridge: %v", err)
	}
	if ack["type"] != "ack" || ack["event_id"] != all[0].ID {
		t.Fatalf("ack = %#v, want ack carrying the stored message id", ack)
	}
}

// Without a matching rule the message is still persisted; it just reaches no
// app. Routing must not be a precondition for durability.
func TestAdapterFramePersistsWithoutRoutingRule(t *testing.T) {
	f := newWSFixture(t)
	adapter := dial(t, f.adpURL+"instance-2")
	waitForConnections(t, f.hub, "adapter", 1)

	if err := adapter.WriteJSON(map[string]interface{}{
		"platform_id":     "platform-xianyu",
		"conversation_id": "conv-2",
		"sender_id":       "buyer-2",
		"message_type":    "text",
		"message_content": "unrouted",
	}); err != nil {
		t.Fatalf("adapter send: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		all, _, err := f.msgSvc.List(10, 0)
		if err == nil && len(all) == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("frame was not persisted without a routing rule")
}

// App sockets are registered by app_id, which is the routing key (previously the
// query parameter was only logged).
func TestAppConnectionRegistersByAppID(t *testing.T) {
	f := newWSFixture(t)
	dial(t, f.appURL+"app-7")
	waitForConnections(t, f.hub, "app", 1)

	if !f.hub.HasConnection("app", "app-7") {
		t.Fatal("app_id did not become the registration key")
	}
	if f.hub.HasConnection("app", "app-other") {
		t.Fatal("unexpected registration for app-other")
	}
}

// Both entry points reject a socket without their identity parameter instead of
// registering an unroutable connection.
func TestWebSocketsRequireIdentityParam(t *testing.T) {
	f := newWSFixture(t)
	base := "ws" + strings.TrimPrefix(f.srv.URL, "http")

	for _, path := range []string{"/ws/adapter", "/ws/app"} {
		conn, _, err := websocket.DefaultDialer.Dial(base+path, nil)
		if err != nil {
			t.Fatalf("dial %s: %v", path, err)
		}
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		var frame map[string]interface{}
		if err := conn.ReadJSON(&frame); err != nil {
			t.Fatalf("%s error frame: %v", path, err)
		}
		if _, ok := frame["error"]; !ok {
			t.Fatalf("%s = %#v, want an error frame", path, frame)
		}
		_ = conn.Close()
	}

	if n := f.hub.ConnectionCount("app") + f.hub.ConnectionCount("adapter"); n != 0 {
		t.Fatalf("connections registered without an id: %d", n)
	}
}

// The inbound HTTP entry point must persist and route too — it is the non-bridge
// path and shares the service with the socket.
func TestCreateMessageEndpointPersistsAndRoutes(t *testing.T) {
	f := newWSFixture(t)

	f.hub.SyncRules([]*message.RouteRule{{
		ID:         "rule-http",
		TargetType: "app",
		TargetID:   "app-http",
		Enabled:    true,
	}})
	app := dial(t, f.appURL+"app-http")
	waitForConnections(t, f.hub, "app", 1)

	body := `{"platform_id":"platform-xianyu","instance_id":"instance-http",` +
		`"conversation_id":"c1","sender_id":"b1","message_type":"text",` +
		`"message_content":"via http"}`

	httpURL := f.srv.URL + "/api/v1/messages"
	// Mount the API route lazily on a second server so this test exercises the
	// real handler, not a stub.
	apiSrv := newMessageAPI(t, f)
	resp, err := apiSrv.Client().Post(apiSrv.URL+"/api/v1/messages", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 201 {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	_ = httpURL

	var created models.InboundEvent
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.ID == "" || created.IdempotencyKey == nil || *created.IdempotencyKey == "" {
		t.Fatalf("created row missing id or key: %#v", created)
	}

	_ = app.SetReadDeadline(time.Now().Add(3 * time.Second))
	var got map[string]interface{}
	if err := app.ReadJSON(&got); err != nil {
		t.Fatalf("routed frame not delivered: %v", err)
	}

	// Replaying the same idempotency key returns the original row with 200.
	replay := strings.Replace(body, `"message_content":"via http"`,
		`"message_content":"via http","idempotency_key":"`+*created.IdempotencyKey+`"`, 1)
	resp2, err := apiSrv.Client().Post(apiSrv.URL+"/api/v1/messages", "application/json", strings.NewReader(replay))
	if err != nil {
		t.Fatalf("replay POST: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != 200 {
		t.Fatalf("replay status = %d, want 200", resp2.StatusCode)
	}
	var replayed models.InboundEvent
	if err := json.NewDecoder(resp2.Body).Decode(&replayed); err != nil {
		t.Fatalf("decode replay: %v", err)
	}
	if replayed.ID != created.ID {
		t.Fatalf("replay returned %s, want %s", replayed.ID, created.ID)
	}
}

// newMessageAPI mounts POST /api/v1/messages on its own test server, reusing the
// fixture's service.
func newMessageAPI(t *testing.T, f *wsFixture) *httptest.Server {
	t.Helper()
	router := gin.New()
	router.POST("/api/v1/messages", CreateMessage(f.msgSvc))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv
}

// A message that matched no rule must stay "received" so the retry scanner picks
// it up once a rule is added — marking it "routed" would strand it.
func TestUnroutedMessageStaysPending(t *testing.T) {
	f := newWSFixture(t)
	adapter := dial(t, f.adpURL+"instance-3")
	waitForConnections(t, f.hub, "adapter", 1)

	if err := adapter.WriteJSON(map[string]interface{}{
		"platform_id":     "platform-xianyu",
		"instance_id":     "instance-3",
		"message_type":    "text",
		"message_content": "no rule",
	}); err != nil {
		t.Fatalf("adapter send: %v", err)
	}

	var stored models.InboundEvent
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		all, _, err := f.msgSvc.List(10, 0)
		if err == nil && len(all) == 1 {
			stored = all[0]
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if stored.ID == "" {
		t.Fatal("frame was not persisted")
	}
	if stored.Status != "received" {
		t.Fatalf("status = %q, want received (still pending)", stored.Status)
	}

	pending, err := f.msgSvc.ListPending(10)
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("pending = %d, want 1", len(pending))
	}
}
