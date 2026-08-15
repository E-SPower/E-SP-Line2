package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/e-spl/e-sp-line2/internal/service"
	"github.com/e-spl/e-sp-line2/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// AdapterWebSocket handles adapter WebSocket connections
func AdapterWebSocket(adapterService *service.AdapterService, messageService *service.MessageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.Error("WebSocket upgrade failed", logger.String("error", err.Error()))
			return
		}
		defer conn.Close()

		// Get adapter instance ID from query params
		instanceID := c.Query("instance_id")
		if instanceID == "" {
			conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"instance_id required"}`))
			return
		}

		logger.Info("Adapter connected", logger.String("instance_id", instanceID))

		// Send welcome message
		welcome := map[string]interface{}{
			"type":      "connected",
			"timestamp": time.Now().Unix(),
			"message":   "Adapter WebSocket connected",
		}
		conn.WriteJSON(welcome)

		// Message handling loop
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				logger.Error("Adapter read error", logger.String("error", err.Error()))
				break
			}

			logger.Debug("Adapter message received",
				logger.Int("type", messageType),
				logger.String("message", string(message)))

			// Parse inbound message and persist it
			var payload map[string]interface{}
			if err := json.Unmarshal(message, &payload); err == nil {
				eventID := persistInboundMessage(messageService, instanceID, payload)
				response := map[string]interface{}{
					"type":      "ack",
					"timestamp": time.Now().Unix(),
					"event_id":  eventID,
				}
				conn.WriteJSON(response)
				continue
			}

			// Raw text message, acknowledge without persistence
			response := map[string]interface{}{
				"type":      "ack",
				"timestamp": time.Now().Unix(),
				"data":      string(message),
			}
			conn.WriteJSON(response)
		}

		logger.Info("Adapter disconnected", logger.String("instance_id", instanceID))
	}
}

// AppWebSocket handles app WebSocket connections
func AppWebSocket(messageService *service.MessageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.Error("WebSocket upgrade failed", logger.String("error", err.Error()))
			return
		}
		defer conn.Close()

		// Get app ID from query params
		appID := c.Query("app_id")
		if appID == "" {
			conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"app_id required"}`))
			return
		}

		logger.Info("App connected", logger.String("app_id", appID))

		// Send welcome message
		welcome := map[string]interface{}{
			"type":      "connected",
			"timestamp": time.Now().Unix(),
			"message":   "App WebSocket connected",
		}
		conn.WriteJSON(welcome)

		// Message handling loop
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				logger.Error("App read error", logger.String("error", err.Error()))
				break
			}

			logger.Debug("App message received",
				logger.Int("type", messageType),
				logger.String("message", string(message)))

			// Parse inbound message and persist it
			var payload map[string]interface{}
			if err := json.Unmarshal(message, &payload); err == nil {
				eventID := persistInboundMessage(messageService, "", payload)
				response := map[string]interface{}{
					"type":      "ack",
					"timestamp": time.Now().Unix(),
					"event_id":  eventID,
				}
				conn.WriteJSON(response)
				continue
			}

			// Raw text message, acknowledge without persistence
			response := map[string]interface{}{
				"type":      "ack",
				"timestamp": time.Now().Unix(),
				"data":      string(message),
			}
			conn.WriteJSON(response)
		}

		logger.Info("App disconnected", logger.String("app_id", appID))
	}
}

// persistInboundMessage persists an inbound message through the message service.
// It returns the created event ID, or an empty string if persistence fails.
func persistInboundMessage(messageService *service.MessageService, instanceID string, payload map[string]interface{}) string {
	req := &service.CreateMessageRequest{
		PlatformID:     getStringField(payload, "platform_id"),
		InstanceID:     instanceID,
		ConversationID: getStringField(payload, "conversation_id"),
		SenderID:       getStringField(payload, "sender_id"),
		SenderName:     getStringField(payload, "sender_name"),
		MessageType:    getStringField(payload, "message_type"),
		MessageContent: getStringField(payload, "message_content"),
		RawMessage:     payload,
		IdempotencyKey: getStringField(payload, "idempotency_key"),
	}

	event, err := messageService.Create(req)
	if err != nil {
		logger.Warn("Failed to persist inbound message",
			logger.String("error", err.Error()))
		return ""
	}
	return event.ID
}

// getStringField extracts a string field from a payload map
func getStringField(payload map[string]interface{}, key string) string {
	if payload == nil {
		return ""
	}
	if value, ok := payload[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}
