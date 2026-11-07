package service

import (
	"os"
	"testing"

	"github.com/e-spl/e-sp-line2/internal/message"
	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/internal/repository"
	"github.com/e-spl/e-sp-line2/pkg/logger"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// The service tests touch the real SQLite driver, so cgo must be on — the same
// requirement as `make test` (internal/adapter has the same shape).
func TestMain(m *testing.M) {
	logger.Init("error")
	code := m.Run()
	logger.Sync()
	os.Exit(code)
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	// A file per test, not `file::memory:`: the pool is capped at one connection
	// by NewRepositories and tests want the same behaviour.
	db, err := gorm.Open(
		sqlite.Open(t.TempDir()+"/test.db?cache=shared"),
		&gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)},
	)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.InboundEvent{},
		&models.AdapterInstance{},
		&models.RouteRule{},
		&models.OutboundCommand{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newMessageService(t *testing.T) (*MessageService, *repository.Repositories, *gorm.DB) {
	t.Helper()
	db := newTestDB(t)
	repos := &repository.Repositories{
		DB:              db,
		InboundEvent:    repository.NewInboundEventRepository(db),
		Instance:        repository.NewInstanceRepository(db),
		RouteRule:       repository.NewRouteRuleRepository(db),
		OutboundCommand: repository.NewOutboundCommandRepository(db),
	}
	return NewMessageService(repos.InboundEvent, repos.Instance), repos, db
}

func baseRequest() *CreateMessageRequest {
	return &CreateMessageRequest{
		PlatformID:     "xianyu",
		InstanceID:     "inst-1",
		ConversationID: "conv-1",
		SenderID:       "buyer-1",
		SenderName:     "buyer",
		MessageType:    "text",
		MessageContent: "hello",
	}
}

// B4: the unique index over an empty idempotency key. Two messages without a key
// in a row used to fail with `UNIQUE constraint failed: inbound_events
// .idempotency_key`, which turned into a 500 for every message after the first
// once an ingest route existed. Both must now be stored.
func TestCreateWithoutIdempotencyKeyTwice(t *testing.T) {
	svc, _, db := newMessageService(t)

	first, err := svc.Create(baseRequest())
	if err != nil {
		t.Fatalf("first keyless create: %v", err)
	}
	second, err := svc.Create(baseRequest())
	if err != nil {
		t.Fatalf("second keyless create must not hit UNIQUE: %v", err)
	}

	if first.ID == second.ID {
		t.Fatal("keyless creates must be distinct events, got the same row twice")
	}
	for _, ev := range []*models.InboundEvent{first, second} {
		if ev.IdempotencyKey == nil || *ev.IdempotencyKey == "" {
			t.Fatalf("event %s has no generated idempotency key", ev.ID)
		}
	}
	if first.IdempotencyKey == nil || second.IdempotencyKey == nil ||
		*first.IdempotencyKey == *second.IdempotencyKey {
		t.Fatal("generated keys must differ, otherwise the unique index dedupes them")
	}

	var count int64
	if err := db.Model(&models.InboundEvent{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("stored %d rows, want 2", count)
	}
}

// A repeated explicit key is a replay: the caller gets the first row back and no
// second row is written.
func TestCreateWithDuplicateKeyReturnsExisting(t *testing.T) {
	svc, _, db := newMessageService(t)

	req := baseRequest()
	req.IdempotencyKey = "key-1"

	first, err := svc.Create(req)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	second, err := svc.Create(req)
	if err != nil {
		t.Fatalf("replayed create: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("replay returned %s, want the original %s", second.ID, first.ID)
	}

	var count int64
	if err := db.Model(&models.InboundEvent{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("stored %d rows, want 1", count)
	}
}

// CreateAndRouteIdempotent reports the replay through its own return value so the
// handler can answer 200 instead of 201 without re-querying.
func TestCreateAndRouteIdempotentFlagsReplay(t *testing.T) {
	svc, _, _ := newMessageService(t)

	req := baseRequest()
	req.IdempotencyKey = "key-2"

	if _, existed, err := svc.CreateAndRouteIdempotent(req); err != nil || existed {
		t.Fatalf("first call: existed=%v err=%v, want false/nil", existed, err)
	}
	if _, existed, err := svc.CreateAndRouteIdempotent(req); err != nil || !existed {
		t.Fatalf("replay: existed=%v err=%v, want true/nil", existed, err)
	}
}

// A stored event with a matching rule and a live adapter connection is delivered
// and recorded as such; the status is the observable proof that routing ran.
func TestRouteMarksDeliveredForLiveTarget(t *testing.T) {
	svc, repos, _ := newMessageService(t)

	hub := message.NewHub()
	defer hub.Close()
	svc.SetHub(hub)
	hub.SyncRules([]models.RouteRule{{
		ID:         "rule-1",
		PlatformID: "xianyu",
		TargetType: "adapter",
		TargetID:   "inst-1",
		Enabled:    true,
	}})

	conn := &message.Connection{ID: "adapter:inst-1", InstanceID: "inst-1", Send: make(chan []byte, 4)}
	hub.RegisterAdapterConn(conn)

	event, err := svc.CreateAndRoute(baseRequest())
	if err != nil {
		t.Fatalf("create and route: %v", err)
	}
	if event.Status != "delivered" {
		t.Fatalf("status = %q, want delivered", event.Status)
	}

	select {
	case frame := <-conn.Send:
		if len(frame) == 0 {
			t.Fatal("empty frame delivered")
		}
	default:
		t.Fatal("no frame was queued on the adapter connection")
	}

	// The persisted row must agree with the returned struct, otherwise a rerun
	// sees a different status than the API reported.
	stored, err := repos.InboundEvent.FindByID(event.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.Status != "delivered" {
		t.Fatalf("stored status = %q, want delivered", stored.Status)
	}
}

// A rule that matches with nobody connected records `routed`, not `delivered`,
// and not the initial `received` — the distinction is what tells an operator the
// rule works but the bridge is down.
func TestRouteMarksRoutedWithoutConnection(t *testing.T) {
	svc, _, _ := newMessageService(t)

	hub := message.NewHub()
	defer hub.Close()
	svc.SetHub(hub)
	hub.SyncRules([]models.RouteRule{{
		ID:         "rule-1",
		PlatformID: "xianyu",
		TargetType: "adapter",
		TargetID:   "inst-1",
		Enabled:    true,
	}})

	event, err := svc.CreateAndRoute(baseRequest())
	if err != nil {
		t.Fatalf("create and route: %v", err)
	}
	if event.Status != "routed" {
		t.Fatalf("status = %q, want routed", event.Status)
	}
}

// No matching rule leaves the event `received` so ListPending retries it once a
// rule appears.
func TestRouteLeavesUnmatchedReceived(t *testing.T) {
	svc, _, _ := newMessageService(t)

	hub := message.NewHub()
	defer hub.Close()
	svc.SetHub(hub)

	event, err := svc.CreateAndRoute(baseRequest())
	if err != nil {
		t.Fatalf("create and route: %v", err)
	}
	if event.Status != "received" {
		t.Fatalf("status = %q, want received", event.Status)
	}
}

// B5: a command with no live bridge is not silently `created` forever — it gets
// parked in retrying so the sweep picks it up, and the retry counter advances.
func TestCreateAndDispatchWithoutConnectionParksRetrying(t *testing.T) {
	db := newTestDB(t)
	cmdSvc := NewCommandService(repository.NewOutboundCommandRepository(db))

	hub := message.NewHub()
	defer hub.Close()
	cmdSvc.SetHub(hub)

	cmd, err := cmdSvc.CreateAndDispatch(&CreateCommandRequest{
		InstanceID:  "inst-1",
		CommandType: "send_text",
		Payload:     map[string]interface{}{"conversation_id": "c", "target_id": "u", "text": "hi"},
	})
	if err != nil {
		t.Fatalf("create and dispatch: %v", err)
	}
	if cmd.Status != "retrying" {
		t.Fatalf("status = %q, want retrying", cmd.Status)
	}
	if cmd.RetryCount != 1 {
		t.Fatalf("retry count = %d, want 1", cmd.RetryCount)
	}
}

// The delivered path: a live adapter connection makes the command `sent` and
// stamps SentAt, so the table stops being a mere log.
func TestCreateAndDispatchMarksSent(t *testing.T) {
	db := newTestDB(t)
	cmdSvc := NewCommandService(repository.NewOutboundCommandRepository(db))

	hub := message.NewHub()
	defer hub.Close()
	cmdSvc.SetHub(hub)

	conn := &message.Connection{ID: "adapter:inst-1", InstanceID: "inst-1", Send: make(chan []byte, 4)}
	hub.RegisterAdapterConn(conn)

	cmd, err := cmdSvc.CreateAndDispatch(&CreateCommandRequest{
		InstanceID:  "inst-1",
		CommandType: "send_text",
		Payload:     map[string]interface{}{"conversation_id": "c", "target_id": "u", "text": "hi"},
	})
	if err != nil {
		t.Fatalf("create and dispatch: %v", err)
	}
	if cmd.Status != "sent" {
		t.Fatalf("status = %q, want sent", cmd.Status)
	}
	if cmd.SentAt == nil {
		t.Fatal("sent_at not set on a delivered command")
	}

	select {
	case frame := <-conn.Send:
		if len(frame) == 0 {
			t.Fatal("empty command frame")
		}
	default:
		t.Fatal("no command frame was queued")
	}
}

// The retry sweep re-sends what the failed first attempt parked, and a later
// successful attempt must flip the row to sent.
func TestRetryPendingDeliversAfterReconnect(t *testing.T) {
	db := newTestDB(t)
	cmdSvc := NewCommandService(repository.NewOutboundCommandRepository(db))

	hub := message.NewHub()
	defer hub.Close()
	cmdSvc.SetHub(hub)

	parked := &models.OutboundCommand{
		ID:          "cmd-1",
		InstanceID:  "inst-1",
		CommandType: "send_text",
		Payload:     `{"text":"hi","target_id":"u"}`,
		Status:      "retrying",
		RetryCount:  1,
		MaxRetries:  3,
		TraceID:     "trace-1",
	}
	if err := db.Create(parked).Error; err != nil {
		t.Fatalf("seed command: %v", err)
	}

	// Nobody connected yet: the sweep must leave it parked, not mark it sent.
	cmdSvc.retryPending()
	var afterMiss models.OutboundCommand
	if err := db.First(&afterMiss, "id = ?", "cmd-1").Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if afterMiss.Status != "retrying" {
		t.Fatalf("status after a failed retry = %q, want retrying", afterMiss.Status)
	}

	// The bridge reconnects.
	conn := &message.Connection{ID: "adapter:inst-1", InstanceID: "inst-1", Send: make(chan []byte, 4)}
	hub.RegisterAdapterConn(conn)
	cmdSvc.retryPending()

	var afterHit models.OutboundCommand
	if err := db.First(&afterHit, "id = ?", "cmd-1").Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if afterHit.Status != "sent" {
		t.Fatalf("status after a delivered retry = %q, want sent", afterHit.Status)
	}
}
