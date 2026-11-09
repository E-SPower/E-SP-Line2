package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/e-spl/e-sp-line2/internal/config"
	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/internal/repository"
	"github.com/e-spl/e-sp-line2/internal/service"
	"github.com/e-spl/e-sp-line2/pkg/logger"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// The handler package had no tests at all (review item M13), so every status
// code regression — including the "deleted something that does not exist → 200"
// one — went unnoticed. These cases pin the not-found/conflict contract on a
// real sqlite database: sqlite is cgo-backed, hence the CGO_ENABLED=1 test lane.

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	logger.Init("error")
	code := m.Run()
	logger.Sync()
	os.Exit(code)
}

type testEnv struct {
	db      *gorm.DB
	router  *gin.Engine
	user    *service.UserService
	auth    *service.AuthService
	route   *service.RouteService
	message *service.MessageService
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	// One connection keeps the shared in-memory database from being torn down
	// between statements of a case.
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(
		&models.User{},
		&models.Platform{},
		&models.AdapterPackage{},
		&models.AdapterInstance{},
		&models.InboundEvent{},
		&models.RouteRule{},
		&models.SystemSetting{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	env := &testEnv{
		db:      db,
		user:    service.NewUserService(repository.NewUserRepository(db), repository.NewSystemSettingRepository(db)),
		route:   service.NewRouteService(repository.NewRouteRuleRepository(db)),
		message: service.NewMessageService(repository.NewInboundEventRepository(db), repository.NewInstanceRepository(db)),
	}
	env.auth = service.NewAuthService(&config.Config{JWT: config.JWTConfig{Secret: "test-secret", Expiration: 1}},
		repository.NewUserRepository(db), repository.NewSystemSettingRepository(db))

	// Mount the real handlers behind a stub middleware that supplies userID,
	// so the cases exercise the handler+service+repository stack, not a mock.
	withUser := func(c *gin.Context) {
		c.Set("userID", "test-user")
		c.Next()
	}
	r := gin.New()
	v1 := r.Group("/api/v1", withUser)
	v1.DELETE("/routes/:id", DeleteRoute(env.route))
	v1.POST("/messages/:id/ack", AckMessage(env.message))
	instances := service.NewInstanceService(repository.NewInstanceRepository(db), nil)
	v1.GET("/instances/:id", GetInstance(instances))
	v1.PUT("/instances/:id", UpdateInstance(instances))
	v1.POST("/auth/register", Register(env.auth))
	v1.POST("/users", CreateUser(env.user))
	env.router = r
	return env
}

func (e *testEnv) do(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

// errBody decodes {"error": "..."} and doubles as a "is this JSON" assertion.
func errBody(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response body is not JSON: %q", w.Body.String())
	}
	return payload.Error
}

func TestDeleteMissingResourcesAre404(t *testing.T) {
	env := newTestEnv(t)

	created, err := env.route.Create(&service.CreateRouteRequest{Name: "r", TargetType: "app", TargetID: "t"})
	if err != nil {
		t.Fatalf("seed route: %v", err)
	}

	cases := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{"delete existing route", http.MethodDelete, "/api/v1/routes/" + created.ID, "", http.StatusOK},
		{"delete missing route", http.MethodDelete, "/api/v1/routes/does-not-exist", "", http.StatusNotFound},
		{"ack missing message", http.MethodPost, "/api/v1/messages/does-not-exist/ack", "", http.StatusNotFound},
		{"update missing instance", http.MethodPut, "/api/v1/instances/does-not-exist", `{"name":"x"}`, http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := env.do(t, tc.method, tc.path, tc.body)
			if w.Code != tc.want {
				t.Fatalf("%s %s = %d, want %d (body %q)", tc.method, tc.path, w.Code, tc.want, w.Body.String())
			}
			if tc.want == http.StatusNotFound {
				if body := errBody(t, w); body != msgNotFound {
					t.Fatalf("404 body = %q, want %q", body, msgNotFound)
				}
			}
		})
	}
}

func TestDuplicateNamesAre409(t *testing.T) {
	env := newTestEnv(t)

	// Self-registration first: the second call must collide, and the body must
	// keep the actionable reason rather than just the sentinel name.
	if w := env.do(t, http.MethodPost, "/api/v1/auth/register", `{"username":"alice","password":"secret123"}`); w.Code != http.StatusCreated {
		t.Fatalf("first register = %d, want 201 (body %q)", w.Code, w.Body.String())
	}
	w := env.do(t, http.MethodPost, "/api/v1/auth/register", `{"username":"alice","password":"secret123"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("duplicate register = %d, want 409 (body %q)", w.Code, w.Body.String())
	}
	if body := errBody(t, w); body != "username already exists" {
		t.Fatalf("409 body = %q, want the reason without the sentinel prefix", body)
	}

	// Admin create path uses the same mapping.
	if w := env.do(t, http.MethodPost, "/api/v1/users", `{"username":"bob","password":"secret123"}`); w.Code != http.StatusCreated {
		t.Fatalf("first admin create = %d, want 201 (body %q)", w.Code, w.Body.String())
	}
	if w := env.do(t, http.MethodPost, "/api/v1/users", `{"username":"bob","password":"secret123"}`); w.Code != http.StatusConflict {
		t.Fatalf("duplicate admin create = %d, want 409 (body %q)", w.Code, w.Body.String())
	}
}

func TestValidationErrorsStay400(t *testing.T) {
	env := newTestEnv(t)

	w := env.do(t, http.MethodPost, "/api/v1/users", `{"username":"short","password":"x"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("short password = %d, want 400 (body %q)", w.Code, w.Body.String())
	}
	if body := errBody(t, w); body != "password must be at least 6 characters" {
		t.Fatalf("400 body = %q, want the validation reason", body)
	}
}

// TestInstanceMetricsRoundTrip covers M10: the Monitoring page reads
// last_heartbeat / message_count / error_count off the instance JSON, so the
// columns must exist and the counters must move.
func TestInstanceMetricsRoundTrip(t *testing.T) {
	env := newTestEnv(t)

	instance := &models.AdapterInstance{
		ID:         "inst-1",
		AdapterID:  "adapter-1",
		PlatformID: "platform-1",
		Name:       "shop",
		Status:     "stopped",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := env.db.Create(instance).Error; err != nil {
		t.Fatalf("seed instance: %v", err)
	}

	// The JSON contract must always carry the three numbers, even at rest.
	w := env.do(t, http.MethodGet, "/api/v1/instances/inst-1", "")
	if w.Code != http.StatusOK {
		t.Fatalf("instance detail = %d (body %q)", w.Code, w.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("instance JSON: %v", err)
	}
	for _, key := range []string{"last_heartbeat", "message_count", "error_count"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("instance JSON missing %q: %s", key, w.Body.String())
		}
	}

	instances := service.NewInstanceService(repository.NewInstanceRepository(env.db), nil)
	instances.RecordInbound("inst-1")
	instances.RecordInbound("inst-1")
	instances.RecordError("inst-1")

	got, err := instances.GetByID("inst-1")
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.MessageCount != 2 {
		t.Fatalf("message_count = %d, want 2", got.MessageCount)
	}
	if got.ErrorCount != 1 {
		t.Fatalf("error_count = %d, want 1", got.ErrorCount)
	}
	if got.LastHeartbeat == nil {
		t.Fatal("last_heartbeat = nil, want a timestamp after RecordInbound")
	}

	// A heartbeat for an instance that vanished must not fail the caller.
	instances.RecordHeartbeat("inst-1")
	instances.RecordInbound("gone")
}
