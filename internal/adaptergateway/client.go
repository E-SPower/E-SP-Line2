package adaptergateway

import (
	"encoding/json"
	"time"

	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/pkg/logger"
	"github.com/gorilla/websocket"
)

// Client represents a connected adapter gateway client (external system).
type Client struct {
	gateway    *Gateway
	conn       *websocket.Conn
	adapter    *models.Adapter
	connection *models.AdapterConnection
	send       chan []byte
}

// sendConnected sends the connected handshake to the client.
func (c *Client) sendConnected() {
	msg := map[string]interface{}{
		"type":            "connected",
		"id":              c.connection.ID,
		"timestamp":       time.Now().UnixMilli(),
		"adapter_id":      c.adapter.ID,
		"gateway_version": "v3",
		"session_id":      c.connection.ID,
		"adapter_name":    c.adapter.Name,
		"platform":        c.adapter.Platform,
	}
	data, _ := json.Marshal(msg)
	select {
	case c.send <- data:
	default:
	}
}

// readPump reads messages from the client and handles them.
func (c *Client) readPump() {
	defer func() {
		c.gateway.unregister(c)
		_ = c.gateway.service.MarkConnectionDisconnected(c.connection.ID)
		_ = c.conn.Close()
		logger.Info("Adapter gateway client disconnected",
			logger.String("conn_id", c.connection.ID))
	}()

	c.conn.SetReadLimit(c.gateway.cfg.MaxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(time.Duration(c.gateway.cfg.ReadTimeout) * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(time.Duration(c.gateway.cfg.ReadTimeout) * time.Second))
		_ = c.gateway.service.TouchConnection(c.connection.ID)
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("Adapter gateway read error",
					logger.String("conn_id", c.connection.ID),
					logger.String("error", err.Error()))
			}
			break
		}

		// Touch heartbeat on any received frame.
		_ = c.gateway.service.TouchConnection(c.connection.ID)

		c.handleMessage(message)
	}
}

// writePump writes messages from the send channel to the client.
func (c *Client) writePump() {
	ticker := time.NewTicker(time.Duration(c.gateway.cfg.HeartbeatInterval) * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.Error("Adapter gateway write error",
					logger.String("conn_id", c.connection.ID),
					logger.String("error", err.Error()))
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Error("Adapter gateway ping error",
					logger.String("conn_id", c.connection.ID),
					logger.String("error", err.Error()))
				return
			}
		}
	}
}

// handleMessage handles an inbound message from the client.
func (c *Client) handleMessage(message []byte) {
	var frame map[string]interface{}
	if err := json.Unmarshal(message, &frame); err != nil {
		c.sendError("40001", "invalid JSON message")
		return
	}

	msgType, _ := frame["type"].(string)
	switch msgType {
	case "ping":
		c.sendPong()
	case "ack":
		// Client acknowledges a server message; nothing to do.
		logger.Debug("Adapter gateway client ack",
			logger.String("conn_id", c.connection.ID))
	case "message":
		c.handleOutboundMessage(frame)
	case "subscribe":
		// Optional: client subscribes to a specific platform/instance.
		c.sendAck(frame)
	default:
		c.sendError("40001", "unsupported message type")
	}
}

// handleOutboundMessage handles an outbound message (reply) from the client.
// It routes the message back to the target bridge instance.
func (c *Client) handleOutboundMessage(frame map[string]interface{}) {
	// Check write permission.
	if c.adapter.Scope != "write" && c.adapter.Scope != "read+write" {
		c.sendError("40103", "adapter has no write permission")
		return
	}

	payload, ok := frame["payload"].(map[string]interface{})
	if !ok {
		c.sendError("40001", "missing payload")
		return
	}

	// The outbound message must reference a target bridge instance.
	instanceID, _ := payload["instance_id"].(string)
	if instanceID == "" {
		c.sendError("40001", "missing instance_id in payload")
		return
	}

	// Build an outbound command and route it to the bridge.
	commandType, _ := payload["command_type"].(string)
	if commandType == "" {
		commandType = "send_text"
	}

	// TODO: route to the bridge via the message hub / dispatcher.
	// The bridge receives commands over its own /ws/adapter connection;
	// wiring the hub dispatch is the next step.
	logger.Info("Adapter gateway outbound command",
		logger.String("conn_id", c.connection.ID),
		logger.String("instance_id", instanceID),
		logger.String("command_type", commandType),
		logger.String("target_id", getString(payload["target_id"])))

	c.sendAck(frame)
}

// getString returns a string representation of a value.
func getString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// sendPong sends a pong response.
func (c *Client) sendPong() {
	msg := map[string]interface{}{
		"type":      "pong",
		"timestamp": time.Now().UnixMilli(),
	}
	data, _ := json.Marshal(msg)
	select {
	case c.send <- data:
	default:
	}
}

// sendAck sends an ack for a client message.
func (c *Client) sendAck(frame map[string]interface{}) {
	msg := map[string]interface{}{
		"type":      "ack",
		"id":        frame["id"],
		"timestamp": time.Now().UnixMilli(),
	}
	data, _ := json.Marshal(msg)
	select {
	case c.send <- data:
	default:
	}
}

// sendError sends an error notification to the client.
func (c *Client) sendError(code, message string) {
	msg := map[string]interface{}{
		"type":      "error",
		"code":      code,
		"message":   message,
		"timestamp": time.Now().UnixMilli(),
	}
	data, _ := json.Marshal(msg)
	select {
	case c.send <- data:
	default:
	}
}

// close closes the client connection.
func (c *Client) close() {
	c.gateway.unregister(c)
	_ = c.gateway.service.MarkConnectionDisconnected(c.connection.ID)
	_ = c.conn.Close()
}
