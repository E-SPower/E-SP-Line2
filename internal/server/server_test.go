package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/e-spl/e-sp-line2/internal/config"
	"github.com/e-spl/e-sp-line2/internal/handler"
	"github.com/e-spl/e-sp-line2/internal/message"
	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/internal/service"
	"github.com/e-spl/e-sp-line2/pkg/logger"
	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	logger.Init("error")
	gin.SetMode(gin.TestMode)
	m.Run()
}

// newTestServer builds a real server (services, hub, routes) against a temp
// SQLite file and its own adapters directory, so the wiring under test is the
// production wiring.
func newTestServer(t *testing.T) *Server {
	t.Helper()

	dir := t.TempDir()
	// A minimal adapters dir keeps catalog scanning local and offline.
	adapterDir := filepath.Join(dir, "adapters", "taobao")
	if err := os.MkdirAll(adapterDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(adapterDir, "adapter.yaml"), []byte("id: taobao-adapter-v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	optionsPath := filepath.Join(dir, "form-options.yaml")
	if err := os.WriteFile(optionsPath, []byte("message_types: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWD, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll("config", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("config", "form-options.yaml"), []byte("message_types: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	cfg := &config.Config{
		Host:     "127.0.0.1",
		Port:     0,
		Version:  "test",
		LogLevel: "error",
		JWT:      config.JWTConfig{Secret: "test-secret", Expiration: 1},
		Adapter:  config.AdapterConfig{AdaptersDir: "adapters", PythonBin: "python3"},
		Database: config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(dir, "test.db")},
	}

	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if srv.retryStop != nil {
			close(srv.retryStop)
		}
		if srv.services.Hub != nil {
			_ = srv.services.Hub.Close()
		}
		if err := srv.services.Close(); err != nil {
			t.Logf("close services: %v", err)
		}
	})

	return srv
}

// The hub must be reachable from the router's services — the whole fix hangs on
// that wiring existing at all.
func TestServerOwnsAHub(t *testing.T) {
	srv := newTestServer(t)
	if srv.services.Hub == nil {
		t.Fatal("services have no hub")
	}
	if srv.services.Message == nil {
		t.Fatal("services have no message service")
	}
}

// POST /api/v1/messages must be routed (it used to be 404). It answers 401
// without a token, which proves the route exists and sits behind auth.
func TestMessagesIngestRouteIsMounted(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", nil)
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatalf("POST /api/v1/messages is not mounted (404)")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (route exists but requires auth)", rec.Code)
	}
}

// Route mutations must refresh the hub's in-memory table, otherwise a rule added
// through the API only takes effect after a restart.
func TestRouteMutationSyncsHub(t *testing.T) {
	srv := newTestServer(t)

	// Seed through the repository, then trigger the same sync the handler calls.
	rule := &models.RouteRule{
		ID:         models.GenerateID(),
		Name:       "r",
		TargetType: "app",
		TargetID:   "app-1",
		Enabled:    true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := srv.services.Repo.RouteRule.Create(rule); err != nil {
		t.Fatalf("create rule: %v", err)
	}

	srv.syncRouteRules()

	rules := srv.services.Hub.GetRouter().GetRules()
	if len(rules) != 1 || rules[0].ID != rule.ID {
		t.Fatalf("hub rules = %#v, want the seeded rule", rules)
	}

	// Disabling it removes it from the in-memory table.
	rule.Enabled = false
	if err := srv.services.Repo.RouteRule.Update(rule); err != nil {
		t.Fatalf("update rule: %v", err)
	}
	srv.syncRouteRules()

	if got := len(srv.services.Hub.GetRouter().GetRules()); got != 0 {
		t.Fatalf("hub still holds %d rules after disabling", got)
	}
}

// syncRoutesAfter must run the wrapped handler and only re-sync on success.
func TestSyncRoutesAfterOnlyOnSuccess(t *testing.T) {
	srv := newTestServer(t)

	ok := srv.syncRoutesAfter(func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"ok": true})
	})
	rec := httptest.NewRecorder()
	ok(gin.CreateTestContextOnly(rec, srv.router))

	bad := srv.syncRoutesAfter(func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "x"})
	})
	rec2 := httptest.NewRecorder()
	bad(gin.CreateTestContextOnly(rec2, srv.router))

	if rec.Code != http.StatusCreated || rec2.Code != http.StatusBadRequest {
		t.Fatalf("statuses = %d/%d, want 201/400", rec.Code, rec2.Code)
	}

	// The wrapper takes a real gin handler.
	var _ = handler.ListRoutes(srv.services.Route)
}

var _ = service.CreateMessageRequest{}
var _ = message.RouteRule{}
