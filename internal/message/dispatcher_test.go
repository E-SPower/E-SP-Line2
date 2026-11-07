package message

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/e-spl/e-sp-line2/pkg/logger"
	"github.com/gorilla/websocket"
)

// UnregisterConnection and friends log through pkg/logger, which is a nil
// zap logger until Init runs. Same TestMain shape as internal/adapter.
func TestMain(m *testing.M) {
	logger.Init("error")
	code := m.Run()
	logger.Sync()
	os.Exit(code)
}

// A Dispatcher is torn down from two different goroutines: Dispatcher.Close on
// server shutdown and readPump's deferred UnregisterConnection when the socket
// drops. Both used to call close(conn.Send) directly, so whichever lost the race
// panicked with "close of closed channel" — in a goroutine, so it took the
// process down instead of surfacing in a test. These cases pin both orders.

func newTestConnection(t *testing.T) (*Dispatcher, *Connection) {
	t.Helper()

	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { ws.Close() })

	d := NewDispatcher()
	conn := &Connection{ID: "c1", Type: "app", AppID: "a", Send: make(chan []byte, 4), Conn: ws}
	d.mu.Lock()
	d.connections[conn.ID] = conn
	d.mu.Unlock()
	return d, conn
}

func TestDispatcherClose(t *testing.T) {
	d, _ := newTestConnection(t)
	if err := d.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// Close first, then the unregister the read pump performs on the way out.
func TestDispatcherCloseThenUnregister(t *testing.T) {
	d, conn := newTestConnection(t)
	if err := d.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	d.UnregisterConnection(conn.ID) // must not panic
}

// The other order: the connection drops, then shutdown runs.
func TestDispatcherUnregisterThenClose(t *testing.T) {
	d, conn := newTestConnection(t)
	d.UnregisterConnection(conn.ID)
	if err := d.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// Unregistering twice must stay a no-op rather than closing a closed channel.
func TestDispatcherDoubleUnregister(t *testing.T) {
	d, conn := newTestConnection(t)
	d.UnregisterConnection(conn.ID)
	d.UnregisterConnection(conn.ID)
}

// Close leaves no connection behind, so a later heartbeat tick cannot write to a
// closed channel.
func TestDispatcherCloseDropsConnections(t *testing.T) {
	d, _ := newTestConnection(t)
	if err := d.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if n := len(d.GetConnectionsByType("app")); n != 0 {
		t.Fatalf("connections left after Close: %d", n)
	}
}
