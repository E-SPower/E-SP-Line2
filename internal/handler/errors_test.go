package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

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

// modelsUnderTest are the only tables these cases touch. Migrating the full
// schema would drag in the adapters/platforms foreign keys for no benefit.
var modelsUnderTest = []any{
	&models.RouteRule{},
	&models.Platform{},
	&models.AdapterInstance{},
	&models.AdapterPackage{},
	&models.InboundEvent{},
	&models.User{},
}

// TestMain initializes the nil-until-Init zap logger; every test in this
// package runs handlers that log, and a nil logger would panic on the first
// heartbeat or error branch.
func TestMain(m *testing.M) {
	logger.Init("error")
	gin.SetMode(gin.TestMode)
	code := m.Run()
	logger.Sync()
	os.Exit(code)
}

// newTestDB opens a throwaway file-backed SQLite database. ":memory:" is not
// used on purpose: the sqlite driver opens a new connection per pooled handle,
// and an in-memory DB is per-connection, so the schema would vanish.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	dsn := path + "?_busy_timeout=10000&_journal_mode=WAL&_foreign_keys=on"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("underlying sql.DB: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := db.AutoMigrate(modelsUnderTest...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// render builds a gin engine with the route under test, executes one request
// against it and returns the recorder.
func render(t *testing.T, method, path string, body any, register func(*gin.Engine)) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	register(router)

	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// errorBody decodes the {"error": ...} envelope every handler now emits.
func errorBody(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error body %q: %v", rec.Body.String(), err)
	}
	return payload.Error
}

// newServices wires the subset of services these cases need against a fresh
// database.
func newServices(t *testing.T) (*service.RouteService, *service.PlatformService, *service.InstanceService, *service.AdapterService) {
	t.Helper()
	db := newTestDB(t)

	routeService := service.NewRouteService(repository.NewRouteRuleRepository(db))
	platformService := service.NewPlatformService(repository.NewPlatformRepository(db))
	instanceService := service.NewInstanceService(
		repository.NewInstanceRepository(db),
		repository.NewAdapterSessionRepository(db),
	)
	adapterService := service.NewAdapterService(
		repository.NewAdapterRepository(db),
		repository.NewAdapterPackageRepository(db),
		repository.NewAdapterSessionRepository(db),
	)
	return routeService, platformService, instanceService, adapterService
}

// TestDeleteUnknownResourceIs404 is the M11 regression: deleting an id that
// does not exist used to answer 200 because the repository ignored
// RowsAffected. Each case failed with 200 before the fix.
func TestDeleteUnknownResourceIs404(t *testing.T) {
	routeService, platformService, instanceService, adapterService := newServices(t)

	cases := []struct {
		name     string
		path     string
		register func(*gin.Engine)
	}{
		{
			name: "route",
			path: "/routes/does-not-exist",
			register: func(r *gin.Engine) {
				r.DELETE("/routes/:id", DeleteRoute(routeService))
			},
		},
		{
			name: "platform",
			path: "/platforms/does-not-exist",
			register: func(r *gin.Engine) {
				r.DELETE("/platforms/:id", DeletePlatform(platformService))
			},
		},
		{
			name: "instance",
			path: "/instances/does-not-exist",
			register: func(r *gin.Engine) {
				r.DELETE("/instances/:id", DeleteInstance(instanceService))
			},
		},
		{
			name: "adapter",
			path: "/adapters/does-not-exist",
			register: func(r *gin.Engine) {
				r.DELETE("/adapters/:id", DeleteAdapter(adapterService))
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := render(t, http.MethodDelete, tc.path, nil, tc.register)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("DELETE %s = %d, want %d (body %q)",
					tc.path, rec.Code, http.StatusNotFound, rec.Body.String())
			}
			if got := errorBody(t, rec); got != "resource not found" {
				t.Fatalf("error body = %q, want %q", got, "resource not found")
			}
		})
	}
}

// TestDeleteExistingResourceIs200 keeps the happy path honest: the 404 work
// must not turn a real delete into a failure.
func TestDeleteExistingResourceIs200(t *testing.T) {
	routeService, _, _, _ := newServices(t)

	created, err := routeService.Create(&service.CreateRouteRequest{
		Name:       "r",
		TargetType: "app",
		TargetID:   "app-1",
	})
	if err != nil {
		t.Fatalf("create route: %v", err)
	}

	rec := render(t, http.MethodDelete, "/routes/"+created.ID, nil, func(r *gin.Engine) {
		r.DELETE("/routes/:id", DeleteRoute(routeService))
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE existing route = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	if _, err := routeService.GetByID(created.ID); err == nil {
		t.Fatal("route still present after delete")
	}
}

// TestAckUnknownMessageIs404 covers the other M11 case: POST /messages/:id/ack
// reported success for an unknown id because UpdateStatus ignored RowsAffected.
func TestAckUnknownMessageIs404(t *testing.T) {
	db := newTestDB(t)
	messageService := service.NewMessageService(
		repository.NewInboundEventRepository(db),
		repository.NewInstanceRepository(db),
	)

	rec := render(t, http.MethodPost, "/messages/does-not-exist/ack", nil, func(r *gin.Engine) {
		r.POST("/messages/:id/ack", AckMessage(messageService))
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("ack unknown message = %d, want 404 (body %q)", rec.Code, rec.Body.String())
	}
}

// TestNotFoundMapsTo404 is the M12 regression table: every "id is unknown"
// branch returned 400 (update/start) or a blanket 500 (delete), and service
// text like "record not found" leaked into the body.
func TestNotFoundMapsTo404(t *testing.T) {
	routeService, platformService, instanceService, adapterService := newServices(t)

	cases := []struct {
		name     string
		method   string
		path     string
		body     any
		register func(*gin.Engine)
	}{
		{
			name:   "update instance",
			method: http.MethodPut,
			path:   "/instances/nope",
			body:   map[string]any{"name": "x"},
			register: func(r *gin.Engine) {
				r.PUT("/instances/:id", UpdateInstance(instanceService))
			},
		},
		{
			name:   "update route",
			method: http.MethodPut,
			path:   "/routes/nope",
			body:   map[string]any{"name": "x"},
			register: func(r *gin.Engine) {
				r.PUT("/routes/:id", UpdateRoute(routeService))
			},
		},
		{
			name:   "update platform",
			method: http.MethodPut,
			path:   "/platforms/nope",
			body:   map[string]any{"name": "x"},
			register: func(r *gin.Engine) {
				r.PUT("/platforms/:id", UpdatePlatform(platformService))
			},
		},
		{
			name:   "get instance",
			method: http.MethodGet,
			path:   "/instances/nope",
			register: func(r *gin.Engine) {
				r.GET("/instances/:id", GetInstance(instanceService))
			},
		},
		{
			name:   "get route",
			method: http.MethodGet,
			path:   "/routes/nope",
			register: func(r *gin.Engine) {
				r.GET("/routes/:id", GetRoute(routeService))
			},
		},
		{
			name:   "start unknown adapter",
			method: http.MethodPost,
			path:   "/adapters/nope/start",
			register: func(r *gin.Engine) {
				r.POST("/adapters/:id/start", StartAdapter(adapterService))
			},
		},
		{
			name:   "update adapter",
			method: http.MethodPut,
			path:   "/adapters/nope",
			body:   map[string]any{"name": "x"},
			register: func(r *gin.Engine) {
				r.PUT("/adapters/:id", UpdateAdapter(adapterService))
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := render(t, tc.method, tc.path, tc.body, tc.register)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("%s %s = %d, want 404 (body %q)",
					tc.method, tc.path, rec.Code, rec.Body.String())
			}
			body := errorBody(t, rec)
			if body == "record not found" {
				t.Fatalf("raw gorm text leaked into the body: %q", body)
			}
		})
	}
}

// TestDuplicateIs409 covers the M12 conflict case: the API documented 409 but
// emitted no 409 anywhere. A duplicate platform code and a duplicate username
// both answered 400.
func TestDuplicateIs409(t *testing.T) {
	_, platformService, _, _ := newServices(t)
	db := newTestDB(t)
	authService := service.NewAuthService(
		&config.Config{JWT: config.JWTConfig{Secret: "test", Expiration: 1}},
		repository.NewUserRepository(db),
		repository.NewSystemSettingRepository(db),
	)

	if rec := render(t, http.MethodPost, "/platforms",
		map[string]any{"name": "Taobao", "code": "taobao"},
		func(r *gin.Engine) { r.POST("/platforms", CreatePlatform(platformService)) },
	); rec.Code != http.StatusCreated {
		t.Fatalf("seeding platform = %d, want 201 (body %q)", rec.Code, rec.Body.String())
	}

	cases := []struct {
		name     string
		path     string
		body     map[string]any
		register func(*gin.Engine)
	}{
		{
			name: "duplicate platform code",
			path: "/platforms",
			body: map[string]any{"name": "Taobao again", "code": "taobao"},
			register: func(r *gin.Engine) {
				r.POST("/platforms", CreatePlatform(platformService))
			},
		},
		{
			name: "duplicate username",
			path: "/auth/register",
			body: map[string]any{"username": "dup", "password": "secret1"},
			register: func(r *gin.Engine) {
				r.POST("/auth/register", Register(authService))
			},
		},
	}

	// Seed the username once, outside the subtest so both cases share state.
	if rec := render(t, http.MethodPost, "/auth/register",
		map[string]any{"username": "dup", "password": "secret1"},
		func(r *gin.Engine) { r.POST("/auth/register", Register(authService)) },
	); rec.Code != http.StatusCreated {
		t.Fatalf("seeding user = %d, want 201 (body %q)", rec.Code, rec.Body.String())
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := render(t, http.MethodPost, tc.path, tc.body, tc.register)
			if rec.Code != http.StatusConflict {
				t.Fatalf("POST %s = %d, want 409 (body %q)",
					tc.path, rec.Code, rec.Body.String())
			}
			if msg := errorBody(t, rec); msg == "" {
				t.Fatal("409 body carried no message")
			}
		})
	}
}

// TestBadBodyIs400 guards the boundary: a malformed body is still a 400 and
// must not be reclassified as a 404/409 by the new mapping.
func TestBadBodyIs400(t *testing.T) {
	routeService, _, _, _ := newServices(t)

	router := gin.New()
	router.POST("/routes", CreateRoute(routeService))
	req := httptest.NewRequest(http.MethodPost, "/routes", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed body = %d, want 400 (body %q)", rec.Code, rec.Body.String())
	}
}

// TestInternalErrorIs500 pins the remaining branch: a real (non not-found)
// database error must not be reported as 400 any more.
func TestInternalErrorIs500(t *testing.T) {
	// Dropping the table makes FindByID fail with something other than
	// ErrRecordNotFound.
	db := newTestDB(t)
	broken := service.NewRouteService(repository.NewRouteRuleRepository(db))
	if err := db.Migrator().DropTable(&models.RouteRule{}); err != nil {
		t.Fatalf("drop table: %v", err)
	}

	rec := render(t, http.MethodGet, "/routes/x", nil, func(r *gin.Engine) {
		r.GET("/routes/:id", GetRoute(broken))
	})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("broken query = %d, want 500 (body %q)", rec.Code, rec.Body.String())
	}
}
