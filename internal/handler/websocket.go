package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	v3 "github.com/e-spl/e-sp-line2/internal/protocol/v3"
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

// AdapterWebSocket handles adapter (bridge) WebSocket connections.
// The bridge connects here over /ws/adapter?instance_id=xxx and reports
// inbound messages. Each reported message is persisted (with its full
// payload) and, if a broadcast callback is provided, fanned out to the
// adapter gateway (接入器) so external frameworks receive it.
func AdapterWebSocket(
	adapterService *service.AdapterService,
	messageService *service.MessageService,
	onInbound ...func(payload map[string]interface{}),
) gin.HandlerFunc {
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

				// Fan out the full message to the adapter gateway (接入器).
				if len(onInbound) > 0 && onInbound[0] != nil {
					onInbound[0](payload)
				}

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
// It preserves the FULL message payload — including the complete V3 message
// chain and the original raw platform message — so nothing is lost when a
// bridge reports an inbound message.
//
// The payload may be either:
//   - a flat bridge payload (platform_id / conversation_id / sender_id /
//     sender_name / message_type / message_content / idempotency_key), or
//   - a full V3 envelope (protocol_version / event_id / event_type / payload)
//     whose payload is an InboundMessagePayload with a message_chain.
//
// It returns the created event ID, or an empty string if persistence fails.
func persistInboundMessage(messageService *service.MessageService, instanceID string, payload map[string]interface{}) string {
	// If the payload is a full V3 envelope, unwrap its inner payload.
	inner := payload
	if protocolVersion, _ := payload["protocol_version"].(string); protocolVersion == v3.Version {
		if p, ok := payload["payload"].(map[string]interface{}); ok {
			inner = p
		}
	}

	// Extract the message chain (if present) and preserve it in the raw field.
	messageChain := inner["message_chain"]
	raw := inner
	if messageChain != nil {
		// Keep the full envelope + chain in the raw message for debugging.
		raw = map[string]interface{}{
			"payload":       inner,
			"message_chain": messageChain,
		}
	}

	req := &service.CreateMessageRequest{
		PlatformID:     getStringField(inner, "platform_id"),
		InstanceID:     instanceID,
		ConversationID: getStringField(inner, "conversation_id"),
		SenderID:       getStringField(inner, "sender_id"),
		SenderName:     getStringField(inner, "sender_name"),
		MessageType:    getStringField(inner, "message_type"),
		MessageContent: getStringField(inner, "message_content"),
		RawMessage:     raw,
		IdempotencyKey: getStringField(inner, "idempotency_key"),
	}

	// Fall back to the sender object if flat sender fields are absent.
	if req.SenderID == "" {
		if sender, ok := inner["sender"].(map[string]interface{}); ok {
			req.SenderID = getStringField(sender, "id")
			req.SenderName = getStringField(sender, "name")
		}
	}

	// If the message chain is present but no flat message_content was given,
	// derive the text content from the chain so the message list still shows
	// something readable. The full chain is always preserved in RawMessage.
	if req.MessageContent == "" && messageChain != nil {
		req.MessageContent = extractChainText(messageChain)
	}

	// If no message_type was given, derive it from the first chain element.
	if req.MessageType == "" && messageChain != nil {
		req.MessageType = extractChainType(messageChain)
	}

	event, err := messageService.Create(req)
	if err != nil {
		logger.Warn("Failed to persist inbound message",
			logger.String("error", err.Error()))
		return ""
	}
	return event.ID
}

// extractChainText extracts the concatenated text content from a V3 message
// chain. The chain is a list of elements, each with a "type" and "content".
// Text elements contribute their "text" field; other elements are skipped.
func extractChainText(messageChain interface{}) string {
	elements, ok := messageChain.([]interface{})
	if !ok {
		return ""
	}
	var parts []string
	for _, el := range elements {
		elem, ok := el.(map[string]interface{})
		if !ok {
			continue
		}
		if elemType, _ := elem["type"].(string); elemType == "text" {
			if content, ok := elem["content"].(map[string]interface{}); ok {
				if text, ok := content["text"].(string); ok && text != "" {
					parts = append(parts, text)
				}
			}
		}
	}
	return strings.Join(parts, " ")
}

// extractChainType derives a coarse message type from the first element of a
// V3 message chain.
func extractChainType(messageChain interface{}) string {
	elements, ok := messageChain.([]interface{})
	if !ok || len(elements) == 0 {
		return "text"
	}
	if elem, ok := elements[0].(map[string]interface{}); ok {
		if elemType, ok := elem["type"].(string); ok && elemType != "" {
			return elemType
		}
	}
	return "text"
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
