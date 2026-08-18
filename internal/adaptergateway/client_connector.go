package adaptergateway

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/pkg/logger"
	"github.com/gorilla/websocket"
)

// ClientConnector manages "client" mode adapters: E-SP-Line2 actively
// connects to an external WebSocket URL, presenting the adapter's key for
// authentication. It maintains one outbound connection per client-mode
// adapter and reconnects automatically with backoff.
type ClientConnector struct {
	gateway *Gateway

	mu      sync.Mutex
	clients map[string]*outboundClient // key: adapter ID
	ctx     context.Context
	cancel  context.CancelFunc
}

// outboundClient represents a single outbound WebSocket connection.
type outboundClient struct {
	adapter    *models.Adapter
	conn       *websocket.Conn
	connection *models.AdapterConnection
	send       chan []byte
	stop       chan struct{}
	stopOnce   sync.Once
}

// NewClientConnector creates a new client connector.
func NewClientConnector(g *Gateway) *ClientConnector {
	ctx, cancel := context.WithCancel(context.Background())
	return &ClientConnector{
		gateway: g,
		clients: make(map[string]*outboundClient),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// StartAdapter starts an outbound connection for a client-mode adapter.
func (cc *ClientConnector) StartAdapter(adapter *models.Adapter) {
	if adapter == nil {
		return
	}
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.startAdapterLocked(adapter)
}

// startAdapterLocked starts an outbound client connection (caller holds lock).
func (cc *ClientConnector) startAdapterLocked(adapter *models.Adapter) {
	if adapter.Mode != "client" || adapter.WSURL == "" ||
		!adapter.Enabled || adapter.Status != "active" {
		return
	}
	if _, ok := cc.clients[adapter.ID]; ok {
		return // already running
	}

	oc := &outboundClient{
		adapter: adapter,
		send:    make(chan []byte, 256),
		stop:    make(chan struct{}),
	}
	cc.clients[adapter.ID] = oc

	go cc.runOutbound(oc)
	logger.Info("Adapter client connector started",
		logger.String("adapter_id", adapter.ID),
		logger.String("ws_url", adapter.WSURL))
}

// ReloadAdapters reconciles client-mode outbound connections against the
// current adapter set. Adapters that are no longer enabled (or removed) are
// stopped; new/enabled client adapters are started. This is called whenever
// adapters are created, updated or deleted so changes take effect live.
func (cc *ClientConnector) ReloadAdapters(adapters []*models.Adapter) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	// Build the set of adapter IDs that should be running.
	desired := make(map[string]*models.Adapter)
	for _, a := range adapters {
		if a.Mode == "client" && a.WSURL != "" && a.Enabled && a.Status == "active" {
			desired[a.ID] = a
		}
	}

	// Stop adapters that should not be running (or whose config changed
	// meaningfully — simplest: stop and let it restart).
	for id, oc := range cc.clients {
		want, ok := desired[id]
		if !ok || want.Key != oc.adapter.Key || want.WSURL != oc.adapter.WSURL ||
			want.Scope != oc.adapter.Scope || want.Platform != oc.adapter.Platform {
			delete(cc.clients, id)
			oc.stopOnce.Do(func() { close(oc.stop) })
			if oc.conn != nil {
				_ = oc.conn.Close()
			}
			logger.Info("Adapter client connector stopped",
				logger.String("adapter_id", id))
		}
	}

	// Start any new / desired adapters.
	for id, a := range desired {
		if _, ok := cc.clients[id]; !ok {
			cc.startAdapterLocked(a)
		}
	}
}

// StopAdapter stops the outbound connection for an adapter.
func (cc *ClientConnector) StopAdapter(adapterID string) {
	cc.mu.Lock()
	oc, ok := cc.clients[adapterID]
	if ok {
		delete(cc.clients, adapterID)
		oc.stopOnce.Do(func() { close(oc.stop) })
	}
	cc.mu.Unlock()

	if ok && oc.conn != nil {
		_ = oc.conn.Close()
	}
}

// runOutbound maintains the outbound connection with reconnect backoff.
func (cc *ClientConnector) runOutbound(oc *outboundClient) {
	backoff := 1 * time.Second
	for {
		select {
		case <-oc.stop:
			return
		case <-cc.ctx.Done():
			return
		default:
		}

		err := cc.connectOnce(oc)
		if err != nil {
			logger.Warn("Adapter client connector connect failed",
				logger.String("adapter_id", oc.adapter.ID),
				logger.String("error", err.Error()))
		}

		select {
		case <-oc.stop:
			return
		case <-cc.ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

// connectOnce establishes a single outbound connection and serves it until
// it drops.
func (cc *ClientConnector) connectOnce(oc *outboundClient) error {
	// Build the URL with the key as a query parameter.
	url := oc.adapter.WSURL
	sep := "?"
	if contains(url, "?") {
		sep = "&"
	}
	url = url + sep + "key=" + oc.adapter.Key

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}

	// Create a connection record.
	record, err := cc.gateway.service.CreateConnection(oc.adapter, "client:"+oc.adapter.WSURL)
	if err != nil {
		conn.Close()
		return err
	}

	oc.conn = conn
	oc.connection = record
	logger.Info("Adapter client connector connected",
		logger.String("adapter_id", oc.adapter.ID),
		logger.String("conn_id", record.ID),
		logger.String("ws_url", oc.adapter.WSURL))

	// Send the connected handshake.
	oc.sendConnected()

	// Serve the connection: read loop + write pump. The write pump exits as
	// soon as the read loop returns (via done) or the connection is stopped,
	// so no goroutine is left blocked on a dead socket.
	done := make(chan struct{})
	go cc.writePump(oc, done)
	cc.readLoop(oc)
	close(done)

	// Cleanup.
	_ = cc.gateway.service.MarkConnectionDisconnected(record.ID)
	_ = conn.Close()
	oc.conn = nil
	oc.connection = nil
	return nil
}

// readLoop reads messages from the outbound connection.
func (cc *ClientConnector) readLoop(oc *outboundClient) {
	oc.conn.SetReadLimit(cc.gateway.cfg.MaxMessageSize)
	_ = oc.conn.SetReadDeadline(time.Now().Add(time.Duration(cc.gateway.cfg.ReadTimeout) * time.Second))
	oc.conn.SetPongHandler(func(string) error {
		_ = oc.conn.SetReadDeadline(time.Now().Add(time.Duration(cc.gateway.cfg.ReadTimeout) * time.Second))
		if oc.connection != nil {
			_ = cc.gateway.service.TouchConnection(oc.connection.ID)
		}
		return nil
	})

	for {
		_, message, err := oc.conn.ReadMessage()
		if err != nil {
			return
		}
		if oc.connection != nil {
			_ = cc.gateway.service.TouchConnection(oc.connection.ID)
		}
		cc.handleInbound(oc, message)
	}
}

// writePump writes messages from the send channel to the outbound connection.
func (cc *ClientConnector) writePump(oc *outboundClient, done chan struct{}) {
	ticker := time.NewTicker(time.Duration(cc.gateway.cfg.HeartbeatInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-oc.send:
			_ = oc.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				return
			}
			if err := oc.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = oc.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := oc.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-done:
			return
		case <-oc.stop:
			return
		}
	}
}

// handleInbound handles a message received from the external system.
func (cc *ClientConnector) handleInbound(oc *outboundClient, message []byte) {
	var frame map[string]interface{}
	if err := json.Unmarshal(message, &frame); err != nil {
		return
	}

	msgType, _ := frame["type"].(string)
	switch msgType {
	case "ping":
		oc.sendPong()
	case "message":
		// Outbound message (reply) from the external system.
		cc.handleOutbound(oc, frame)
	case "ack":
		// Acknowledgment; nothing to do.
	default:
		// Ignore unknown frames.
	}
}

// handleOutbound handles an outbound message from the external system.
func (cc *ClientConnector) handleOutbound(oc *outboundClient, frame map[string]interface{}) {
	// Check write permission.
	if oc.adapter.Scope != "write" && oc.adapter.Scope != "read+write" {
		oc.sendError("40103", "adapter has no write permission")
		return
	}

	payload, ok := frame["payload"].(map[string]interface{})
	if !ok {
		oc.sendError("40001", "missing payload")
		return
	}

	instanceID, _ := payload["instance_id"].(string)
	if instanceID == "" {
		oc.sendError("40001", "missing instance_id in payload")
		return
	}

	commandType, _ := payload["command_type"].(string)
	if commandType == "" {
		commandType = "send_text"
	}

	// TODO: route to the bridge via the message hub / dispatcher.
	logger.Info("Adapter client connector outbound command",
		logger.String("adapter_id", oc.adapter.ID),
		logger.String("instance_id", instanceID),
		logger.String("command_type", commandType))

	oc.sendAck(frame)
}

// sendConnected sends the connected handshake.
func (oc *outboundClient) sendConnected() {
	msg := map[string]interface{}{
		"type":            "connected",
		"id":              oc.connection.ID,
		"timestamp":       time.Now().UnixMilli(),
		"adapter_id":      oc.adapter.ID,
		"gateway_version": "v3",
		"session_id":      oc.connection.ID,
		"adapter_name":    oc.adapter.Name,
		"platform":        oc.adapter.Platform,
	}
	data, _ := json.Marshal(msg)
	select {
	case oc.send <- data:
	default:
	}
}

// sendPong sends a pong response.
func (oc *outboundClient) sendPong() {
	msg := map[string]interface{}{
		"type":      "pong",
		"timestamp": time.Now().UnixMilli(),
	}
	data, _ := json.Marshal(msg)
	select {
	case oc.send <- data:
	default:
	}
}

// sendAck sends an ack for a client message.
func (oc *outboundClient) sendAck(frame map[string]interface{}) {
	msg := map[string]interface{}{
		"type":      "ack",
		"id":        frame["id"],
		"timestamp": time.Now().UnixMilli(),
	}
	data, _ := json.Marshal(msg)
	select {
	case oc.send <- data:
	default:
	}
}

// sendError sends an error notification.
func (oc *outboundClient) sendError(code, message string) {
	msg := map[string]interface{}{
		"type":      "error",
		"code":      code,
		"message":   message,
		"timestamp": time.Now().UnixMilli(),
	}
	data, _ := json.Marshal(msg)
	select {
	case oc.send <- data:
	default:
	}
}

// Close stops all outbound connections.
func (cc *ClientConnector) Close() {
	cc.cancel()
	cc.mu.Lock()
	defer cc.mu.Unlock()
	for id, oc := range cc.clients {
		oc.stopOnce.Do(func() { close(oc.stop) })
		if oc.conn != nil {
			_ = oc.conn.Close()
		}
		delete(cc.clients, id)
	}
}

// contains reports whether a string contains a substring.
func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && indexOf(s, substr) >= 0)
}

// indexOf returns the index of the first occurrence of substr in s, or -1.
func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
