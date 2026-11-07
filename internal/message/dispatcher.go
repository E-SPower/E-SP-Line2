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

	// OnFrame, when set, receives every raw frame the read pump pulls off this
	// connection. It is the per-connection hook the server uses to persist and
	// route bridge traffic; without it the pump only acknowledges.
	OnFrame func(conn *Connection, raw []byte)
	// OnClose, when set, runs once after the read pump exits (and the
	// connection is unregistered). Used to drop the adapter-gateway bridge
	// registration that belongs to this socket.
	OnClose func(conn *Connection)

	// closeOnce guards Send: a connection is closed from both readPump's defer
	// and Dispatcher.Close, and possibly from a third path later. A bare
	// close(chan) there panics on the loser of the race.
	closeOnce sync.Once
	// sendMu serializes sends against the close, so producers that do not hold
	// the dispatcher lock (the bridge fan-out goroutine) cannot send on a
	// closed channel.
	sendMu sync.RWMutex
	closed bool
}

// closeSend closes the send channel exactly once, however many shutdown paths
// reach the connection. The websocket conn is closed separately — net.Conn.Close
// is itself idempotent enough and callers need not thread through here.
func (c *Connection) closeSend() {
	c.closeOnce.Do(func() {
		c.sendMu.Lock()
		c.closed = true
		close(c.Send)
		c.sendMu.Unlock()
	})
}

// Enqueue buffers a frame for this connection without blocking, reporting
// whether it fit. It is safe to call concurrently with closeSend.
func (c *Connection) Enqueue(data []byte) bool {
	c.sendMu.RLock()
	defer c.sendMu.RUnlock()
	if c.closed {
		return false
	}
	select {
	case c.Send <- data:
		return true
	default:
		return false
	}
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

// RegisterConnection registers a new connection
func (d *Dispatcher) RegisterConnection(conn *Connection) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.connections[conn.ID] = conn
	logger.Info("Connection registered",
		logger.String("conn_id", conn.ID),
		logger.String("type", conn.Type))

	// Start read/write pumps. A connection without a socket (tests, or a
	// connection whose socket is owned elsewhere) is registered but not pumped.
	if conn.Conn != nil {
		go d.readPump(conn)
		go d.writePump(conn)
	}
}

// UnregisterConnection unregisters a connection
func (d *Dispatcher) UnregisterConnection(connID string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if conn, ok := d.connections[connID]; ok {
		conn.closeSend()
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

// Dispatch dispatches a message to target connections and reports how many
// sockets the frame was buffered on. Callers that need "delivered" semantics
// (command dispatch) treat zero as a failure rather than a silent success.
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

// dispatchToApp dispatches message to a specific app and returns the number of
// connections the frame was buffered on.
func (d *Dispatcher) dispatchToApp(data []byte, appID string) int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	n := 0
	for _, conn := range d.connections {
		if conn.Type == "app" && conn.AppID == appID {
			select {
			case conn.Send <- data:
				n++
			default:
				logger.Warn("App connection send buffer full",
					logger.String("app_id", appID))
			}
		}
	}
	return n
}

// dispatchToAdapter dispatches message to a specific adapter and returns the
// number of connections the frame was buffered on.
func (d *Dispatcher) dispatchToAdapter(data []byte, instanceID string) int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	n := 0
	for _, conn := range d.connections {
		if conn.Type == "adapter" && conn.InstanceID == instanceID {
			select {
			case conn.Send <- data:
				n++
			default:
				logger.Warn("Adapter connection send buffer full",
					logger.String("instance_id", instanceID))
			}
		}
	}
	return n
}

// broadcast broadcasts message to all connections and returns the number of
// connections the frame was buffered on.
func (d *Dispatcher) broadcast(data []byte) int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	n := 0
	for _, conn := range d.connections {
		select {
		case conn.Send <- data:
			n++
		default:
			logger.Warn("Connection send buffer full",
				logger.String("conn_id", conn.ID))
		}
	}
	return n
}

// readPump pumps messages from WebSocket connection to dispatcher
func (d *Dispatcher) readPump(conn *Connection) {
	defer func() {
		d.UnregisterConnection(conn.ID)
		conn.Conn.Close()
		if conn.OnClose != nil {
			conn.OnClose(conn)
		}
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

		// A per-connection hook (the bridge/server wiring) owns the frame.
		// Frames are handed over raw: not every bridge frame is a v3
		// envelope (tool_result / tool_catalog are flat maps).
		if conn.OnFrame != nil {
			conn.OnFrame(conn, message)
			continue
		}

		var envelope v3.MessageEnvelope
		if err := json.Unmarshal(message, &envelope); err != nil {
			logger.Warn("Failed to parse message",
				logger.String("conn_id", conn.ID),
				logger.String("error", err.Error()))
			continue
		}
		d.handleMessage(conn, &envelope)
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
func (d *Dispatcher) handleMessage(conn *Connection, envelope *v3.MessageEnvelope) {
	logger.Debug("Message received",
		logger.String("conn_id", conn.ID),
		logger.String("event_type", string(envelope.EventType)))

	// Send ACK
	ack := map[string]interface{}{
		"type":      "ack",
		"event_id":  envelope.EventID,
		"timestamp": time.Now().UnixMilli(),
	}

	data, _ := json.Marshal(ack)
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
