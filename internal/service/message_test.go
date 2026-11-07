package service

import (
	"fmt"
	"strings"
	"testing"

	"github.com/e-spl/e-sp-line2/internal/message"
	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/internal/repository"
	"github.com/e-spl/e-sp-line2/pkg/logger"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func TestMain(m *testing.M) {
	logger.Init("error")
	m.Run()
}

// newTestMessageService builds a MessageService over a throwaway in-memory
// SQLite database with only the tables the message path touches. AutoMigrate is
// deliberate: it is exactly how production creates the schema, so the unique
// index under test is the real one.
func newTestMessageService(t *testing.T) *MessageService {
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
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := db.AutoMigrate(&models.Platform{}, &models.AdapterInstance{}, &models.InboundEvent{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Keep tests independent even though the DSN shares one in-memory database.
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
	return NewMessageService(repos.InboundEvent, repos.Instance)
}

func listMessages(t *testing.T, svc *MessageService) []models.InboundEvent {
	t.Helper()
	all, _, err := svc.List(100, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	return all
}

// B4 regression: two messages without an idempotency key must both persist.
//
// Before the fix the second insert died with
// "UNIQUE constraint failed: inbound_events.idempotency_key", because the field
// was a `string` whose zero value "" is itself a value — unlike NULL, SQLite's
// UNIQUE allows only one of those.
func TestCreateMessageWithoutIdempotencyKeyTwice(t *testing.T) {
	svc := newTestMessageService(t)

	for i := 1; i <= 2; i++ {
		msg, err := svc.Create(&CreateMessageRequest{
			PlatformID:     "platform-taobao",
			InstanceID:     "instance-1",
			ConversationID: "conv-1",
			SenderID:       "buyer-1",
			MessageType:    "text",
			MessageContent: fmt.Sprintf("message %d", i),
		})
		if err != nil {
			t.Fatalf("insert #%d failed: %v", i, err)
		}
		if msg.IdempotencyKey == nil || *msg.IdempotencyKey == "" {
			t.Fatalf("insert #%d got an empty per-row key: %#v", i, msg.IdempotencyKey)
		}
	}

	if got := len(listMessages(t, svc)); got != 2 {
		t.Fatalf("stored messages = %d, want 2", got)
	}
}

// A caller-supplied key stays authoritative: the duplicate returns the original
// row instead of a second one, even though the rows are otherwise identical.
func TestCreateMessageDeduplicatesSuppliedKey(t *testing.T) {
	svc := newTestMessageService(t)

	req := &CreateMessageRequest{
		PlatformID:     "platform-taobao",
		InstanceID:     "instance-1",
		MessageType:    "text",
		MessageContent: "hello",
		IdempotencyKey: "taobao-buyer-1-conv-1-1700000000000",
	}

	first, err := svc.Create(req)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	second, err := svc.Create(req)
	if err != nil {
		t.Fatalf("duplicate insert: %v", err)
	}

	if first.ID != second.ID {
		t.Fatalf("duplicate returned a new row: %s != %s", first.ID, second.ID)
	}
	if got := len(listMessages(t, svc)); got != 1 {
		t.Fatalf("stored messages = %d, want 1", got)
	}
}

// Distinct keys must not deduplicate each other.
func TestCreateMessageDistinctKeysBothPersist(t *testing.T) {
	svc := newTestMessageService(t)

	for i := 1; i <= 2; i++ {
		if _, err := svc.Create(&CreateMessageRequest{
			PlatformID:     "platform-taobao",
			InstanceID:     "instance-1",
			MessageType:    "text",
			MessageContent: "hello",
			IdempotencyKey: fmt.Sprintf("key-%d", i),
		}); err != nil {
			t.Fatalf("insert #%d: %v", i, err)
		}
	}

	if got := len(listMessages(t, svc)); got != 2 {
		t.Fatalf("stored messages = %d, want 2", got)
	}
}

// The platform is derived from the instance when the caller omits it, and the
// derived value is what the generated fallback key is built from.
func TestCreateMessageDerivesPlatformFromInstance(t *testing.T) {
	svc := newTestMessageService(t)

	instance := &models.AdapterInstance{
		ID:         "instance-9",
		AdapterID:  "adapter-1",
		PlatformID: "platform-xianyu",
		Name:       "shop",
		Status:     "stopped",
	}
	if err := svc.instanceRepo.Create(instance); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	msg, err := svc.Create(&CreateMessageRequest{
		InstanceID:     "instance-9",
		MessageType:    "text",
		MessageContent: "hi",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if msg.PlatformID != "platform-xianyu" {
		t.Fatalf("platform_id = %q, want platform-xianyu", msg.PlatformID)
	}
}

// The instance wins over the caller's platform value: bridges send the platform
// code ("xianyu") in the v3 envelope while platform_id columns hold the row ID.
func TestCreateMessagePrefersInstancePlatform(t *testing.T) {
	svc := newTestMessageService(t)

	instance := &models.AdapterInstance{
		ID:         "instance-9",
		AdapterID:  "adapter-1",
		PlatformID: "platform-xianyu",
		Name:       "shop",
		Status:     "stopped",
	}
	if err := svc.instanceRepo.Create(instance); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	msg, err := svc.Create(&CreateMessageRequest{
		PlatformID:     "xianyu", // platform code, not the row ID
		InstanceID:     "instance-9",
		MessageType:    "text",
		MessageContent: "hi",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if msg.PlatformID != "platform-xianyu" {
		t.Fatalf("platform_id = %q, want the instance platform platform-xianyu", msg.PlatformID)
	}
}

// A message that could not be routed on arrival is replayed by the retry
// scanner once a rule exists, and the replay is idempotent.
func TestRetryPendingReplaysUnroutedMessages(t *testing.T) {
	svc := newTestMessageService(t)

	// No hub yet: Create persists without routing.
	msg, err := svc.Create(&CreateMessageRequest{
		PlatformID:     "platform-taobao",
		InstanceID:     "instance-1",
		MessageType:    "text",
		MessageContent: "stuck",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if msg.Status != "received" {
		t.Fatalf("status = %q, want received", msg.Status)
	}

	// Attaching a hub makes the message replayable.
	hub := message.NewHub()
	t.Cleanup(func() { _ = hub.Close() })
	svc.SetHub(hub)

	conn := &message.Connection{ID: "app-1", Type: "app", AppID: "app-1", Send: make(chan []byte, 4)}
	hub.GetDispatcher().RegisterConnection(conn)
	t.Cleanup(func() { hub.UnregisterConnection(conn.ID) })
	hub.SyncRules([]*message.RouteRule{{ID: "r1", TargetType: "app", TargetID: "app-1", Enabled: true}})

	n, err := svc.RetryPending(10)
	if err != nil {
		t.Fatalf("RetryPending: %v", err)
	}
	if n != 1 {
		t.Fatalf("replayed = %d, want 1", n)
	}

	select {
	case frame := <-conn.Send:
		if !strings.Contains(string(frame), "stuck") {
			t.Fatalf("replayed frame lost the content: %s", frame)
		}
	default:
		t.Fatal("replayed frame was not delivered")
	}

	stored, err := svc.GetByID(msg.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.Status != "routed" {
		t.Fatalf("status = %q, want routed", stored.Status)
	}

	// A second scan has nothing left to do.
	if n, err := svc.RetryPending(10); err != nil || n != 0 {
		t.Fatalf("second scan = (%d, %v), want (0, nil)", n, err)
	}
}
