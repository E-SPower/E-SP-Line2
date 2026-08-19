// Package adaptergateway implements the "接入器" (Adapter) WebSocket gateway.
//
// An adapter is a configurable WebSocket endpoint that external systems use
// to exchange e-commerce messages with E-SP-Line2. It supports two modes:
//
//   - "server": E-SP-Line2 listens on a WebSocket endpoint; external systems
//     connect to it as clients, authenticating with the adapter's key.
//   - "client": E-SP-Line2 actively connects to an external WebSocket URL,
//     presenting the adapter's key for authentication.
//
// The gateway speaks the ESPL v3 protocol (message envelopes + message
// chains), fans out inbound messages (from bridges) to connected adapters,
// and accepts outbound messages (replies) to route back to bridges.
package adaptergateway

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/internal/service"
	"github.com/e-spl/e-sp-line2/pkg/logger"
	"github.com/gorilla/websocket"
)

// GatewayConfig holds the adapter gateway configuration.
type GatewayConfig struct {
	// HeartbeatInterval is how often the server pings clients (seconds).
	HeartbeatInterval int
	// ReadTimeout is how long the server waits for a frame before
	// considering the connection dead (seconds).
	ReadTimeout int
	// MaxMessageSize is the maximum inbound frame size in bytes.
	MaxMessageSize int64
}

// DefaultGatewayConfig returns the default gateway configuration.
func DefaultGatewayConfig() GatewayConfig {
	return GatewayConfig{
		HeartbeatInterval: 30,
		ReadTimeout:       90,
		MaxMessageSize:    1024 * 1024,
	}
}

// Gateway is the adapter gateway WebSocket server.
type Gateway struct {
	cfg     GatewayConfig
	service *service.AdapterGatewayService

	upgrader websocket.Upgrader

	mu          sync.RWMutex
	connections map[string]*Client

	// clientConnector manages "client" mode adapters (outbound connections).
	clientConnector *ClientConnector

	// counterMu guards the async message-count flush loop.
	counterMu   sync.Mutex
	counterDirty map[string]int64
	counterStop  chan struct{}
	counterOnce  sync.Once
}

// NewGateway creates a new adapter gateway.
func NewGateway(cfg GatewayConfig, svc *service.AdapterGatewayService) *Gateway {
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = 30
	}
	if cfg.ReadTimeout <= 0 {
		cfg.ReadTimeout = 90
	}
	if cfg.MaxMessageSize <= 0 {
		cfg.MaxMessageSize = 1024 * 1024
	}

	g := &Gateway{
		cfg:          cfg,
		service:      svc,
		connections:  make(map[string]*Client),
		counterDirty: make(map[string]int64),
		counterStop:  make(chan struct{}),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
	}
	g.clientConnector = NewClientConnector(g)
	g.startCounterFlusher()
	return g
}

// SyncAdapters reconciles all client-mode outbound connections against the
// current adapter list. It is called at server startup and whenever adapters
// are created, updated or deleted so changes take effect live.
func (g *Gateway) SyncAdapters(adapters []*models.Adapter) {
	if g.clientConnector == nil {
		return
	}
	g.clientConnector.ReloadAdapters(adapters)
}

// ServeHTTP upgrades an HTTP request to a WebSocket connection and handles
// the adapter gateway protocol (server mode). The access key must be provided
// either as a query parameter (?key=xxx) or via the Sec-WebSocket-Protocol
// header.
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.serve(w, r, "")
}

// ServeHTTPWithID upgrades an HTTP request and authenticates against a
// specific adapter ID (used for per-adapter listen paths like
// /ws/adapter-gateway/:adapterID). Both the adapter ID and the key must match.
func (g *Gateway) ServeHTTPWithID(w http.ResponseWriter, r *http.Request, adapterID string) {
	g.serve(w, r, adapterID)
}

// serve performs the common server-mode connection handling.
func (g *Gateway) serve(w http.ResponseWriter, r *http.Request, adapterID string) {
	// 1. Authenticate the access key BEFORE upgrading.
	key := r.URL.Query().Get("key")
	if key == "" {
		// Fallback: Sec-WebSocket-Protocol header (e.g. "espl, <key>").
		protocols := websocket.Subprotocols(r)
		for _, p := range protocols {
			if p != "espl" && p != "" {
				key = p
				break
			}
		}
	}

	var adapter *models.Adapter
	var err error
	if adapterID != "" {
		adapter, err = g.service.ValidateAdapterIDAndKey(adapterID, key)
	} else {
		adapter, err = g.service.ValidateKey(key)
	}
	if err != nil {
		logger.Warn("Adapter gateway auth failed",
			logger.String("remote", r.RemoteAddr),
			logger.String("error", err.Error()))
		http.Error(w, `{"error":"unauthorized","code":"40101"}`, http.StatusUnauthorized)
		return
	}

	// 2. Upgrade to WebSocket.
	conn, err := g.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Adapter gateway upgrade failed",
			logger.String("error", err.Error()))
		return
	}

	// 3. Create a connection record.
	record, err := g.service.CreateConnection(adapter, r.RemoteAddr)
	if err != nil {
		logger.Error("Adapter gateway create connection failed",
			logger.String("error", err.Error()))
		conn.Close()
		return
	}

	client := &Client{
		gateway:    g,
		conn:       conn,
		adapter:    adapter,
		connection: record,
		send:       make(chan []byte, 256),
	}

	g.register(client)
	logger.Info("Adapter gateway client connected",
		logger.String("conn_id", record.ID),
		logger.String("adapter_id", adapter.ID),
		logger.String("adapter_name", adapter.Name),
		logger.String("platform", adapter.Platform),
		logger.String("remote", r.RemoteAddr))

	// 4. Send the connected handshake.
	client.sendConnected()

	// 5. Run read/write pumps.
	go client.writePump()
	client.readPump()
}

// register adds a client to the connection registry.
func (g *Gateway) register(c *Client) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.connections[c.connection.ID] = c
}

// unregister removes a client from the connection registry.
func (g *Gateway) unregister(c *Client) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.connections[c.connection.ID]; ok {
		delete(g.connections, c.connection.ID)
		close(c.send)
	}
}

// BroadcastInbound fans out an inbound message (from a bridge) to all
// connected adapter gateway clients that match the message platform.
// The payload is the full ESPL v3 message envelope.
//
// It delivers to BOTH:
//   - server-mode clients (connected to E-SP-Line2, held in g.connections)
//   - client-mode outbound connections (E-SP-Line2 connects to an external
//     system such as LangBot's ESPL adapter, held in cc.clients)
func (g *Gateway) BroadcastInbound(envelope map[string]interface{}) {
	platform, _ := envelope["platform"].(string)

	data, err := json.Marshal(envelope)
	if err != nil {
		logger.Error("Adapter gateway marshal envelope failed",
			logger.String("error", err.Error()))
		return
	}

	// 1. Server-mode clients.
	g.mu.RLock()
	clients := make([]*Client, 0, len(g.connections))
	for _, c := range g.connections {
		// Only deliver to clients that match the platform (or all platforms).
		if c.adapter.Platform == "" || c.adapter.Platform == platform {
			clients = append(clients, c)
		}
	}
	g.mu.RUnlock()

	for _, c := range clients {
		select {
		case c.send <- data:
			// Increment the message counter asynchronously (batched) so the
			// hot broadcast path never blocks on a database write.
			g.bumpMessageCount(c.connection.ID)
		default:
			logger.Warn("Adapter gateway client send buffer full",
				logger.String("conn_id", c.connection.ID))
		}
	}

	// 2. Client-mode outbound connections (e.g. LangBot ESPL adapter).
	if g.clientConnector != nil {
		g.clientConnector.BroadcastInbound(envelope, platform)
	}
}

// bumpMessageCount records a message-count increment for a connection. The
// actual database write is deferred to the background flusher so the
// broadcast path stays lock-free.
func (g *Gateway) bumpMessageCount(connID string) {
	g.counterMu.Lock()
	g.counterDirty[connID]++
	g.counterMu.Unlock()
}

// startCounterFlusher launches the background loop that periodically flushes
// accumulated message counts to the database in a single batched pass.
func (g *Gateway) startCounterFlusher() {
	g.counterOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					g.flushMessageCounts()
				case <-g.counterStop:
					g.flushMessageCounts()
					return
				}
			}
		}()
	})
}

// flushMessageCounts writes all accumulated message-count increments to the
// database. It swaps the dirty map so concurrent broadcasts never block.
func (g *Gateway) flushMessageCounts() {
	g.counterMu.Lock()
	if len(g.counterDirty) == 0 {
		g.counterMu.Unlock()
		return
	}
	dirty := g.counterDirty
	g.counterDirty = make(map[string]int64)
	g.counterMu.Unlock()

	for connID, n := range dirty {
		for i := int64(0); i < n; i++ {
			if err := g.service.IncrementConnectionMessageCount(connID); err != nil {
				logger.Warn("Adapter gateway increment message count failed",
					logger.String("conn_id", connID),
					logger.String("error", err.Error()))
				break
			}
		}
	}
}

// ClientCount returns the number of currently connected clients.
func (g *Gateway) ClientCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.connections)
}

// Close closes all connected clients and stops all outbound connections.
func (g *Gateway) Close() {
	g.mu.RLock()
	clients := make([]*Client, 0, len(g.connections))
	for _, c := range g.connections {
		clients = append(clients, c)
	}
	g.mu.RUnlock()

	for _, c := range clients {
		c.close()
	}

	if g.clientConnector != nil {
		g.clientConnector.Close()
	}

	// Stop the counter flusher and flush any remaining counts.
	select {
	case <-g.counterStop:
	default:
		close(g.counterStop)
	}
	g.flushMessageCounts()
}
