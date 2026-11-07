package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/e-spl/e-sp-line2/internal/message"
	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/internal/repository"
	"github.com/e-spl/e-sp-line2/internal/service"
	"github.com/e-spl/e-sp-line2/pkg/logger"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	logger.Init("error")
	code := m.Run()
	logger.Sync()
	os.Exit(code)
}

func newIngestRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open(t.TempDir()+"/test.db"),
		&gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)},
	)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.InboundEvent{},
		&models.RouteRule{},
		&models.OutboundCommand{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	svc := service.NewMessageService(
		repository.NewInboundEventRepository(db),
		repository.NewInstanceRepository(db),
	)
	hub := message.NewHub()
	t.Cleanup(func() { hub.Close() })
	svc.SetHub(hub)

	r := gin.New()
	r.POST("/messages", CreateMessage(svc))
	return r, db
}

func postJSON(t *testing.T, r *gin.Engine, body interface{}) *httptest.ResponseRecorder {
	t.Helper()

	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func validBody() map[string]interface{} {
	return map[string]interface{}{
		"platform_id":     "xianyu",
		"instance_id":     "inst-1",
		"conversation_id": "conv-1",
		"sender_id":       "buyer-1",
		"message_type":    "text",
		"message_content": "你好",
	}
}

// B3: the route exists and stores a message. Before this, POST /messages was 404
// and GET /messages could only ever be empty.
func TestCreateMessageIngest(t *testing.T) {
	r, db := newIngestRouter(t)

	w := postJSON(t, r, validBody())
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}

	var count int64
	if err := db.Model(&models.InboundEvent{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("stored %d rows, want 1", count)
	}
}

// B4 exercised through the API: the second message without an idempotency_key
// must be 201, not 500. This is the exact case the review warned would start as
// soon as the route was opened.
func TestCreateMessageTwiceWithoutKey(t *testing.T) {
	r, db := newIngestRouter(t)

	if w := postJSON(t, r, validBody()); w.Code != http.StatusCreated {
		t.Fatalf("first status = %d, want 201", w.Code)
	}
	w := postJSON(t, r, validBody())
	if w.Code != http.StatusCreated {
		t.Fatalf("second keyless status = %d, want 201; body=%s", w.Code, w.Body.String())
	}

	var count int64
	if err := db.Model(&models.InboundEvent{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("stored %d rows, want 2", count)
	}
}

// A replayed key answers 200 with the original event and stores nothing new, so a
// retrying producer cannot duplicate the row.
func TestCreateMessageDuplicateKeyIs200(t *testing.T) {
	r, db := newIngestRouter(t)

	body := validBody()
	body["idempotency_key"] = "dup-1"

	first := postJSON(t, r, body)
	if first.Code != http.StatusCreated {
		t.Fatalf("first status = %d, want 201", first.Code)
	}
	second := postJSON(t, r, body)
	if second.Code != http.StatusOK {
		t.Fatalf("replay status = %d, want 200; body=%s", second.Code, second.Body.String())
	}

	var firstEvent, secondEvent models.InboundEvent
	if err := json.Unmarshal(first.Body.Bytes(), &firstEvent); err != nil {
		t.Fatalf("decode first: %v", err)
	}
	if err := json.Unmarshal(second.Body.Bytes(), &secondEvent); err != nil {
		t.Fatalf("decode second: %v", err)
	}
	if firstEvent.ID != secondEvent.ID {
		t.Fatalf("replay returned %s, want original %s", secondEvent.ID, firstEvent.ID)
	}

	var count int64
	if err := db.Model(&models.InboundEvent{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("stored %d rows, want 1", count)
	}
}

// Missing required fields stay a 400 rather than a 500 from the database layer.
func TestCreateMessageValidation(t *testing.T) {
	r, _ := newIngestRouter(t)

	w := postJSON(t, r, map[string]interface{}{"message_content": "x"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
}
