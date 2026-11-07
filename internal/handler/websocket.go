package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/e-spl/e-sp-line2/internal/adaptergateway"
	"github.com/e-spl/e-sp-line2/internal/message"
	"github.com/e-spl/e-sp-line2/internal/models"
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
// inbound messages.
//
// The socket is registered with the message hub as an adapter connection, and
// the hub's readPump/writePump take over the read loop, the write serialization
// and the ping/pong heartbeat. Inbound frames are persisted and routed through
// the hub, so a live app connection actually receives them instead of the
// bridge only getting its own frame echoed back.
//
// The bridge is also registered with the adapter gateway (接入器) so outbound
// commands (replies from external frameworks like LangBot) can be routed back
// to this bridge.
func AdapterWebSocket(
	hub *message.Hub,
	messageService *service.MessageService,
	gateway *adaptergateway.Gateway,
	onInbound ...func(payload map[string]interface{}),
) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.Error("WebSocket upgrade failed", logger.String("error", err.Error()))
			return
		}

		// instance_id identifies the connection for routing; it is the
		// dispatcher key for adapter-bound frames (commands), no longer just a
		// log field.
		instanceID := c.Query("instance_id")
		if instanceID == "" {
			_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"instance_id required"}`))
			_ = conn.Close()
			return
		}

		logger.Info("Adapter connected", logger.String("instance_id", instanceID))

		// Register the bridge connection with the gateway so outbound commands
		// can be routed back to it.
		bc := adaptergateway.NewBridgeConn(instanceID)
		if gateway != nil {
			gateway.RegisterBridge(instanceID, bc)
		}

		wsConn := &message.Connection{
			ID:         models.GenerateID(),
			Conn:       conn,
			InstanceID: instanceID,
			PlatformID: c.Query("platform_id"),
			Send:       make(chan []byte, 256),
			CreatedAt:  time.Now(),
			LastPing:   time.Now(),
		}

		// Frames the gateway pushes for this bridge (outbound commands) are
		// copied onto the connection's send channel, which writePump owns — the
		// only writer to the socket.
		forwardDone := make(chan struct{})
		go func() {
			for {
				select {
				case data, ok := <-bc.Chan():
					if !ok {
						return
					}
					if !enqueue(wsConn, data) {
						logger.Warn("Adapter connection send buffer full",
							logger.String("instance_id", instanceID))
					}
				case <-forwardDone:
					return
				}
			}
		}()

		wsConn.OnFrame = func(_ *message.Connection, raw []byte) {
			handleAdapterFrame(wsConn, instanceID, messageService, gateway, onInbound, raw)
		}
		wsConn.OnClose = func(*message.Connection) {
			close(forwardDone)
			if gateway != nil {
				gateway.UnregisterBridge(instanceID, bc)
			}
			logger.Info("Adapter disconnected", logger.String("instance_id", instanceID))
		}

		hub.RegisterAdapter(wsConn)

		enqueueJSON(wsConn, map[string]interface{}{
			"type":      "connected",
			"timestamp": time.Now().Unix(),
			"message":   "Adapter WebSocket connected",
		})
	}
}

// handleAdapterFrame processes one frame reported by a bridge.
func handleAdapterFrame(
	wsConn *message.Connection,
	instanceID string,
	messageService *service.MessageService,
	gateway *adaptergateway.Gateway,
	onInbound []func(payload map[string]interface{}),
	raw []byte,
) {
	logger.Debug("Adapter message received", logger.String("message", string(raw)))

	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		// Raw text frame: acknowledge without persistence.
		enqueueJSON(wsConn, map[string]interface{}{
			"type":      "ack",
			"timestamp": time.Now().Unix(),
			"data":      string(raw),
		})
		return
	}

	// Tool (inactive field) result frames are NOT messages: they carry the
	// answer to an outstanding tool_call and must be handed back to the
	// gateway's pending-call registry.
	if frameType, _ := payload["type"].(string); frameType == "tool_result" {
		if gateway != nil {
			result := payload
			if p, ok := payload["payload"].(map[string]interface{}); ok {
				result = p
			}
			gateway.HandleToolResult(instanceID, result)
		}
		enqueueJSON(wsConn, map[string]interface{}{"type": "ack", "timestamp": time.Now().Unix()})
		return
	}

	// Tool catalog frames let a bridge declare/advertise its own inactive-field
	// tools. They are registered but not persisted.
	if frameType, _ := payload["type"].(string); frameType == "tool_catalog" {
		if gateway != nil {
			registerToolsFromFrame(gateway, payload)
		}
		enqueueJSON(wsConn, map[string]interface{}{"type": "ack", "timestamp": time.Now().Unix()})
		return
	}

	eventID := persistInboundMessage(messageService, instanceID, payload)

	// Inject instance_id into the payload for adapter gateway routing. The
	// bridge connects with ?instance_id=xxx in the query string, but the payload
	// it sends does not contain instance_id. The adapter gateway (接入器) and
	// downstream frameworks (e.g. LangBot's ESPL adapter) need instance_id on the
	// event so they can route outbound (reply) messages back to the correct
	// bridge instance.
	//
	// Both "instance_id" (canonical) and "instance" (legacy alias used by some
	// external adapters) are set so routing works regardless of which key the
	// downstream looks for.
	if instanceID != "" {
		if _, exists := payload["instance_id"]; !exists {
			payload["instance_id"] = instanceID
		}
		if _, exists := payload["instance"]; !exists {
			payload["instance"] = instanceID
		}
	}

	// Fan out the full message to the adapter gateway (接入器).
	if len(onInbound) > 0 && onInbound[0] != nil {
		onInbound[0](payload)
	}

	enqueueJSON(wsConn, map[string]interface{}{
		"type":      "ack",
		"timestamp": time.Now().Unix(),
		"event_id":  eventID,
	})
}

// AppWebSocket handles app WebSocket connections.
//
// Like the adapter socket it is registered with the hub (as an app connection)
// and the hub pumps own the socket. The app_id is the dispatcher key that
// routing rules target, so routed inbound messages are delivered here.
func AppWebSocket(hub *message.Hub, messageService *service.MessageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.Error("WebSocket upgrade failed", logger.String("error", err.Error()))
			return
		}

		appID := c.Query("app_id")
		if appID == "" {
			_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"app_id required"}`))
			_ = conn.Close()
			return
		}

		logger.Info("App connected", logger.String("app_id", appID))

		wsConn := &message.Connection{
			ID:        models.GenerateID(),
			Conn:      conn,
			AppID:     appID,
			Send:      make(chan []byte, 256),
			CreatedAt: time.Now(),
			LastPing:  time.Now(),
		}

		wsConn.OnFrame = func(_ *message.Connection, raw []byte) {
			var payload map[string]interface{}
			if err := json.Unmarshal(raw, &payload); err != nil {
				enqueueJSON(wsConn, map[string]interface{}{
					"type":      "ack",
					"timestamp": time.Now().Unix(),
					"data":      string(raw),
				})
				return
			}

			eventID := persistInboundMessage(messageService, "", payload)
			enqueueJSON(wsConn, map[string]interface{}{
				"type":      "ack",
				"timestamp": time.Now().Unix(),
				"event_id":  eventID,
			})
		}
		wsConn.OnClose = func(*message.Connection) {
			logger.Info("App disconnected", logger.String("app_id", appID))
		}

		hub.RegisterApp(wsConn)

		enqueueJSON(wsConn, map[string]interface{}{
			"type":      "connected",
			"timestamp": time.Now().Unix(),
			"message":   "App WebSocket connected",
		})
	}
}

// enqueue pushes a frame onto a connection's send channel without blocking.
func enqueue(conn *message.Connection, data []byte) bool {
	return conn.Enqueue(data)
}

// enqueueJSON marshals v and enqueues it.
func enqueueJSON(conn *message.Connection, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	enqueue(conn, data)
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
// Persisting through the service also hands the message to the hub, which is
// what makes it reach routed app connections.
//
// It returns the created event ID, or an empty string if persistence fails.
func persistInboundMessage(messageService *service.MessageService, instanceID string, payload map[string]interface{}) string {
	if messageService == nil {
		return ""
	}

	// If the payload is a full V3 envelope, unwrap its inner payload. The
	// envelope carries fields the inner payload usually omits (platform,
	// event_type) at its top level; they act as defaults so the persisted row
	// is still attributable and routable.
	inner := payload
	envelope := map[string]interface{}(nil)
	if protocolVersion, _ := payload["protocol_version"].(string); protocolVersion == v3.Version {
		if p, ok := payload["payload"].(map[string]interface{}); ok {
			envelope = payload
			inner = p
		}
	}

	// field reads a key from the inner payload, falling back to the envelope.
	field := func(key string) string {
		if v := getStringField(inner, key); v != "" {
			return v
		}
		return getStringField(envelope, key)
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
		PlatformID:     field("platform_id"),
		InstanceID:     instanceID,
		ConversationID: field("conversation_id"),
		SenderID:       field("sender_id"),
		SenderName:     field("sender_name"),
		MessageType:    field("message_type"),
		MessageContent: field("message_content"),
		RawMessage:     raw,
		IdempotencyKey: field("idempotency_key"),
		EventType:      field("event_type"),
	}
	// The envelope names the platform as "platform" (v3 envelope field), not
	// "platform_id"; accept both spellings.
	if req.PlatformID == "" {
		req.PlatformID = field("platform")
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

// registerToolsFromFrame parses a bridge "tool_catalog" frame and registers the
// declared tools with the gateway. The frame may carry the tool list either at
// the top level ("tools") or inside "payload" ("tools").
//
// Accepted tool shape (JSON):
//
//	{
//	  "name": "get_product",
//	  "semantic_id": "get_product",
//	  "category": "query_only",
//	  "description": "...",
//	  "parameters": [{"name":"item_id","type":"string","required":true}],
//	  "enabled": true
//	}
func registerToolsFromFrame(gateway *adaptergateway.Gateway, frame map[string]interface{}) {
	if gateway == nil {
		return
	}
	if n := gateway.RegisterToolsFromCatalogFrame(frame); n > 0 {
		logger.Info("Bridge registered tools",
			logger.Int("count", n))
	}
}
