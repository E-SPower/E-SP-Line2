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

	// Start read/write pumps
	go d.readPump(conn)
	go d.writePump(conn)
}

// UnregisterConnection unregisters a connection
func (d *Dispatcher) UnregisterConnection(connID string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if conn, ok := d.connections[connID]; ok {
		close(conn.Send)
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

// Dispatch dispatches a message to target connections
func (d *Dispatcher) Dispatch(envelope *v3.MessageEnvelope, targets []*RouteTarget) error {
	data, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	for _, target := range targets {
		switch target.Type {
		case "app":
			d.dispatchToApp(data, target.ID)
		case "adapter":
			d.dispatchToAdapter(data, target.ID)
		case "broadcast":
			d.broadcast(data)
		}
	}

	return nil
}

// dispatchToApp dispatches message to a specific app
func (d *Dispatcher) dispatchToApp(data []byte, appID string) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, conn := range d.connections {
		if conn.Type == "app" && conn.AppID == appID {
			select {
			case conn.Send <- data:
			default:
				logger.Warn("App connection send buffer full",
					logger.String("app_id", appID))
			}
		}
	}
}

// dispatchToAdapter dispatches message to a specific adapter
func (d *Dispatcher) dispatchToAdapter(data []byte, instanceID string) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, conn := range d.connections {
		if conn.Type == "adapter" && conn.InstanceID == instanceID {
			select {
			case conn.Send <- data:
			default:
				logger.Warn("Adapter connection send buffer full",
					logger.String("instance_id", instanceID))
			}
		}
	}
}

// broadcast broadcasts message to all connections
func (d *Dispatcher) broadcast(data []byte) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, conn := range d.connections {
		select {
		case conn.Send <- data:
		default:
			logger.Warn("Connection send buffer full",
				logger.String("conn_id", conn.ID))
		}
	}
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

		// Parse message
		var envelope v3.MessageEnvelope
		if err := json.Unmarshal(message, &envelope); err != nil {
			logger.Warn("Failed to parse message",
				logger.String("conn_id", conn.ID),
				logger.String("error", err.Error()))
			continue
		}

		// Handle message based on type
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
		close(conn.Send)
		conn.Conn.Close()
	}

	return nil
}
