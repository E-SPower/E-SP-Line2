package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/e-spl/e-sp-line2/internal/message"
	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/internal/repository"
	"github.com/e-spl/e-sp-line2/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// commandFixture wires CreateCommand to a real sqlite-backed CommandService and
// the real hub, so a POST produces the same row/dispatch/status sequence as
// production.
type commandFixture struct {
	hub    *message.Hub
	cmdSvc *service.CommandService
	srv    *httptest.Server
}

func newCommandFixture(t *testing.T) *commandFixture {
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

	if err := db.AutoMigrate(&models.OutboundCommand{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("DELETE FROM outbound_commands").Error; err != nil {
		t.Fatalf("clear: %v", err)
	}

	hub := message.NewHub()
	cmdSvc := service.NewCommandService(repository.NewOutboundCommandRepository(db))

	router := gin.New()
	router.POST("/commands", CreateCommand(cmdSvc, hub))
	srv := httptest.NewServer(router)

	t.Cleanup(func() {
		srv.Close()
		_ = hub.Close()
		_ = sqlDB.Close()
	})

	return &commandFixture{hub: hub, cmdSvc: cmdSvc, srv: srv}
}

func (f *commandFixture) post(t *testing.T, body string) (*http.Response, models.OutboundCommand) {
	t.Helper()
	resp, err := f.srv.Client().Post(f.srv.URL+"/commands", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	var cmd models.OutboundCommand
	if err := json.NewDecoder(resp.Body).Decode(&cmd); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return resp, cmd
}

// B5: with a live adapter connection the created command is written to it and
// recorded as sent. Before the fix the row stayed "created" forever.
func TestCreateCommandDispatchesToAdapter(t *testing.T) {
	f := newCommandFixture(t)

	// The fixture server only mounts POST /commands, so register the adapter
	// side directly on the hub's dispatcher — same registration the websocket
	// handler performs.
	conn := &message.Connection{
		ID:         "c1",
		Type:       "adapter",
		InstanceID: "instance-1",
		Send:       make(chan []byte, 4),
	}
	f.hub.GetDispatcher().RegisterConnection(conn)
	t.Cleanup(func() { f.hub.UnregisterConnection(conn.ID) })

	resp, created := f.post(t, `{"instance_id":"instance-1","command_type":"send_text",`+
		`"payload":{"conversation_id":"conv-1","message_content":"hello"}}`)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	if created.Status != "sent" {
		t.Fatalf("status = %q, want sent", created.Status)
	}
	if created.SentAt == nil {
		t.Fatal("sent_at was not set")
	}

	select {
	case frame := <-conn.Send:
		var envelope map[string]interface{}
		if err := json.Unmarshal(frame, &envelope); err != nil {
			t.Fatalf("frame is not JSON: %v", err)
		}
		payload, _ := envelope["payload"].(map[string]interface{})
		if payload == nil || payload["command_type"] != "send_text" {
			t.Fatalf("dispatched frame = %#v", envelope)
		}
		if payload["target_id"] != "conv-1" {
			t.Fatalf("target_id = %v, want conv-1", payload["target_id"])
		}
	case <-time.After(3 * time.Second):
		t.Fatal("command was not written to the adapter connection")
	}

	// The stored row agrees with the response.
	stored, err := f.cmdSvc.GetByID(created.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.Status != "sent" || stored.SentAt == nil {
		t.Fatalf("stored row = %#v, want sent with sent_at", stored)
	}
}

// No adapter connection: the dispatch fails, MarkFailed bumps retry_count and
// parks the row on "retrying" so the scanner can have another go.
func TestCreateCommandWithoutAdapterMarksRetrying(t *testing.T) {
	f := newCommandFixture(t)

	resp, created := f.post(t, `{"instance_id":"instance-absent","command_type":"send_text",`+
		`"payload":{"conversation_id":"conv-1","message_content":"hello"}}`)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (the row is durable even when undeliverable)", resp.StatusCode)
	}
	if created.Status != "retrying" {
		t.Fatalf("status = %q, want retrying", created.Status)
	}
	if created.RetryCount != 1 {
		t.Fatalf("retry_count = %d, want 1", created.RetryCount)
	}

	pending, err := f.cmdSvc.ListPending(10)
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if len(pending) != 1 || pending[0].ID != created.ID {
		t.Fatalf("pending = %#v, want the created command", pending)
	}
}

// MarkFailed must settle on "failed" once MaxRetries is exhausted, so a command
// for a permanently absent instance stops being retried.
func TestCreateCommandExhaustsRetries(t *testing.T) {
	f := newCommandFixture(t)

	created, err := f.cmdSvc.Create(&service.CreateCommandRequest{
		InstanceID:  "instance-absent",
		CommandType: "send_text",
		MaxRetries:  2,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	for i := 0; i < 2; i++ {
		if derr := f.cmdSvc.DispatchCommand(f.hub, created); derr == nil {
			t.Fatalf("dispatch #%d unexpectedly succeeded", i+1)
		}
	}

	stored, err := f.cmdSvc.GetByID(created.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.Status != "failed" {
		t.Fatalf("status = %q, want failed after max retries", stored.Status)
	}
	if pending, _ := f.cmdSvc.ListPending(10); len(pending) != 0 {
		t.Fatalf("failed command still pending: %#v", pending)
	}
}
