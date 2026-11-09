package message

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	v3 "github.com/e-spl/e-sp-line2/internal/protocol/v3"
	"github.com/e-spl/e-sp-line2/pkg/logger"
	"github.com/gorilla/websocket"
)

// Dispatcher handles message dispatching to connected clients
type Dispatcher struct {
	connections map[string]*Connection
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc

	handlerMu sync.RWMutex
	onMessage func(conn *Connection, raw []byte) string
}

// Connection represents a WebSocket connection
type Connection struct {
	ID         string
	Type       string // "adapter", "app"
	Conn       *websocket.Conn
	PlatformID string
	InstanceID string
	AppID      string
	Send       chan []byte
	CreatedAt  time.Time
	LastPing   time.Time

	// closeOnce guards Send: a connection is closed from both readPump's defer
	// and Dispatcher.Close, and possibly from a third path later. A bare
	// close(chan) there panics on the loser of the race.
	closeOnce sync.Once

	// done is closed when the connection goes away, so the HTTP handler that
	// upgraded it can block until the pumps finish instead of returning from
	// the gin handler while the socket is still live.
	done     chan struct{}
	doneOnce sync.Once
}

// Done returns a channel closed when the connection is torn down.
func (c *Connection) Done() <-chan struct{} {
	if c.done == nil {
		return nil
	}
	return c.done
}

// markDone closes Done exactly once.
func (c *Connection) markDone() {
	c.doneOnce.Do(func() {
		if c.done != nil {
			close(c.done)
		}
	})
}

// closeSend closes the send channel exactly once, however many shutdown paths
// reach the connection. The websocket conn is closed separately — net.Conn.Close
// is itself idempotent enough and callers need not thread through here.
func (c *Connection) closeSend() {
	c.closeOnce.Do(func() { close(c.Send) })
}

// NewDispatcher creates a new message dispatcher
func NewDispatcher() *Dispatcher {
	ctx, cancel := context.WithCancel(context.Background())
	d := &Dispatcher{
		connections: make(map[string]*Connection),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Start heartbeat checker
	go d.heartbeatChecker()

	return d
}

// RegisterConnection registers a new connection and starts the hub-owned
// read/write pumps. Use this when the dispatcher may own the socket.
func (d *Dispatcher) RegisterConnection(conn *Connection) {
	d.register(conn, true)
}

// RegisterConnectionOnly registers a connection whose socket is already owned by
// the caller: the caller runs the read loop and is the single writer, and the
// dispatcher only routes frames into conn.Send. This is what the bridge route
// needs — it has its own frame handling (tool_result, tool_catalog, fan-out) and
// a bridge writer goroutine that must stay the only writer on that socket.
// Starting a second reader/writer here would race both.
func (d *Dispatcher) RegisterConnectionOnly(conn *Connection) {
	d.register(conn, false)
}

// SetMessageHandler installs a callback invoked for every frame read by a
// dispatcher-owned read pump, before the ACK is queued. It returns the ID of the
// event it persisted ("" when it stored nothing); the read pump echoes that back
// in the ACK, matching what the hand-written loops used to send. The server uses
// it to persist app-socket messages through the message service.
func (d *Dispatcher) SetMessageHandler(fn func(conn *Connection, raw []byte) string) {
	d.handlerMu.Lock()
	defer d.handlerMu.Unlock()
	d.onMessage = fn
}

func (d *Dispatcher) callMessageHandler(conn *Connection, raw []byte) string {
	d.handlerMu.RLock()
	fn := d.onMessage
	d.handlerMu.RUnlock()
	if fn == nil {
		return ""
	}
	return fn(conn, raw)
}

func (d *Dispatcher) register(conn *Connection, startPumps bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.connections[conn.ID] = conn
	logger.Info("Connection registered",
		logger.String("conn_id", conn.ID),
		logger.String("type", conn.Type))

	if !startPumps {
		return
	}
	// Start read/write pumps
	go d.readPump(conn)
	go d.writePump(conn)
}

// UnregisterConnection unregisters a connection
func (d *Dispatcher) UnregisterConnection(connID string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if conn, ok := d.connections[connID]; ok {
		conn.closeSend()
		conn.markDone()
		delete(d.connections, connID)
		logger.Info("Connection unregistered", logger.String("conn_id", connID))
	}
}

// GetConnection gets a connection by ID
func (d *Dispatcher) GetConnection(connID string) (*Connection, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	conn, ok := d.connections[connID]
	return conn, ok
}

// GetConnectionsByType gets connections by type
func (d *Dispatcher) GetConnectionsByType(connType string) []*Connection {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make([]*Connection, 0)
	for _, conn := range d.connections {
		if conn.Type == connType {
			result = append(result, conn)
		}
	}
	return result
}

// GetConnectionsByPlatform gets connections by platform
func (d *Dispatcher) GetConnectionsByPlatform(platformID string) []*Connection {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make([]*Connection, 0)
	for _, conn := range d.connections {
		if conn.PlatformID == platformID {
			result = append(result, conn)
		}
	}
	return result
}

// DispatchRaw enqueues an already-marshaled frame on the connections a single
// target resolves to, and reports how many accepted it. Used for frames that do
// not follow the v3 envelope shape (the bridge command protocol is flat).
func (d *Dispatcher) DispatchRaw(target *RouteTarget, data []byte) int {
	switch target.Type {
	case "app":
		return d.dispatchToApp(data, target.ID)
	case "adapter":
		return d.dispatchToAdapter(data, target.ID)
	case "broadcast":
		return d.broadcast(data)
	}
	return 0
}

// Dispatch dispatches a message to target connections and reports how many
// frames were actually enqueued. A target that matched a rule but has no live
// connection contributes 0, which is what lets the caller tell "routed" apart
// from "nobody was listening".
func (d *Dispatcher) Dispatch(envelope *v3.MessageEnvelope, targets []*RouteTarget) (int, error) {
	data, err := json.Marshal(envelope)
	if err != nil {
		return 0, err
	}

	delivered := 0
	for _, target := range targets {
		switch target.Type {
		case "app":
			delivered += d.dispatchToApp(data, target.ID)
		case "adapter":
			delivered += d.dispatchToAdapter(data, target.ID)
		case "broadcast":
			delivered += d.broadcast(data)
		}
	}

	return delivered, nil
}

// dispatchToApp dispatches message to a specific app
func (d *Dispatcher) dispatchToApp(data []byte, appID string) int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	delivered := 0
	for _, conn := range d.connections {
		if conn.Type == "app" && conn.AppID == appID {
			select {
			case conn.Send <- data:
				delivered++
			default:
				logger.Warn("App connection send buffer full",
					logger.String("app_id", appID))
			}
		}
	}
	return delivered
}

// dispatchToAdapter dispatches message to a specific adapter
func (d *Dispatcher) dispatchToAdapter(data []byte, instanceID string) int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	delivered := 0
	for _, conn := range d.connections {
		if conn.Type == "adapter" && conn.InstanceID == instanceID {
			select {
			case conn.Send <- data:
				delivered++
			default:
				logger.Warn("Adapter connection send buffer full",
					logger.String("instance_id", instanceID))
			}
		}
	}
	return delivered
}

// broadcast broadcasts message to all connections
func (d *Dispatcher) broadcast(data []byte) int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	delivered := 0
	for _, conn := range d.connections {
		select {
		case conn.Send <- data:
			delivered++
		default:
			logger.Warn("Connection send buffer full",
				logger.String("conn_id", conn.ID))
		}
	}
	return delivered
}

// readPump pumps messages from WebSocket connection to dispatcher
func (d *Dispatcher) readPump(conn *Connection) {
	defer func() {
		d.UnregisterConnection(conn.ID)
		conn.Conn.Close()
	}()

	conn.Conn.SetReadLimit(1024 * 1024) // 1MB
	conn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.Conn.SetPongHandler(func(string) error {
		conn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := conn.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("WebSocket read error",
					logger.String("conn_id", conn.ID),
					logger.String("error", err.Error()))
			}
			break
		}

		// Hand the raw frame to whatever persistence the server installed. It
		// gets raw bytes because not every frame on these sockets is a v3
		// envelope — bridges also report flat payloads.
		eventID := d.callMessageHandler(conn, message)

		// Parse message
		var envelope v3.MessageEnvelope
		if err := json.Unmarshal(message, &envelope); err != nil {
			logger.Warn("Failed to parse message",
				logger.String("conn_id", conn.ID),
				logger.String("error", err.Error()))
			// Still ACK: the frame may have been persisted as a flat payload.
			d.ack(conn, eventID)
			continue
		}

		// A well-formed envelope carries its own event ID; prefer it.
		d.handleMessage(conn, &envelope, eventID)
	}
}

// writePump pumps messages from dispatcher to WebSocket connection
func (d *Dispatcher) writePump(conn *Connection) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		conn.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-conn.Send:
			conn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				conn.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := conn.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.Error("WebSocket write error",
					logger.String("conn_id", conn.ID),
					logger.String("error", err.Error()))
				return
			}

		case <-ticker.C:
			conn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Error("WebSocket ping error",
					logger.String("conn_id", conn.ID),
					logger.String("error", err.Error()))
				return
			}
			conn.LastPing = time.Now()
		}
	}
}

// handleMessage handles incoming message from connection
func (d *Dispatcher) handleMessage(conn *Connection, envelope *v3.MessageEnvelope, storedEventID string) {
	logger.Debug("Message received",
		logger.String("conn_id", conn.ID),
		logger.String("event_type", string(envelope.EventType)))

	if envelope.EventID != "" {
		storedEventID = envelope.EventID
	}
	d.ack(conn, storedEventID)
}

// ack queues an ACK frame on the connection.
func (d *Dispatcher) ack(conn *Connection, eventID string) {
	ack := map[string]interface{}{
		"type":      "ack",
		"event_id":  eventID,
		"timestamp": time.Now().UnixMilli(),
	}

	data, err := json.Marshal(ack)
	if err != nil {
		return
	}
	select {
	case conn.Send <- data:
	default:
	}
}

// heartbeatChecker checks connection heartbeats
func (d *Dispatcher) heartbeatChecker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.checkHeartbeats()
		case <-d.ctx.Done():
			return
		}
	}
}

// checkHeartbeats checks all connection heartbeats
func (d *Dispatcher) checkHeartbeats() {
	d.mu.RLock()
	defer d.mu.RUnlock()

	now := time.Now()
	for _, conn := range d.connections {
		if now.Sub(conn.LastPing) > 90*time.Second {
			logger.Warn("Connection heartbeat timeout",
				logger.String("conn_id", conn.ID))
			// Connection will be cleaned up by readPump
		}
	}
}

// Close closes the dispatcher
func (d *Dispatcher) Close() error {
	d.cancel()

	d.mu.Lock()
	defer d.mu.Unlock()

	for _, conn := range d.connections {
		conn.closeSend()
		conn.markDone()
		// Conn is nil for a connection whose socket the caller owns and already
		// closed (the adapter route), so it must not be dereferenced blindly.
		if conn.Conn != nil {
			conn.Conn.Close()
		}
	}
	// Drop the entries too: a running readPump still hits its deferred
	// UnregisterConnection. closeSend makes the double close safe; clearing the
	// map keeps a later heartbeat tick from writing to a closed connection.
	d.connections = make(map[string]*Connection)

	return nil
}
