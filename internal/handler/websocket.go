package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/e-spl/e-sp-line2/internal/adaptergateway"
	"github.com/e-spl/e-sp-line2/internal/message"
	"github.com/e-spl/e-sp-line2/internal/middleware"
	v3 "github.com/e-spl/e-sp-line2/internal/protocol/v3"
	"github.com/e-spl/e-sp-line2/internal/service"
	"github.com/e-spl/e-sp-line2/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// allowedOrigin decides whether a browser Origin may open a WebSocket. The
// server sets it from the same config list the CORS middleware uses, so the two
// surfaces share one allowlist and cannot drift apart.
var allowedOrigin = func(string) bool { return true }

// SetAllowedOrigin installs the Origin allowlist used by the WebSocket
// upgrader. A nil func restores "allow everything".
func SetAllowedOrigin(fn func(string) bool) {
	if fn == nil {
		fn = func(string) bool { return true }
	}
	allowedOrigin = fn
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// An empty Origin is a non-browser client (the Python bridges,
		// websocket tooling); those are not subject to browser CORS.
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		return allowedOrigin(origin)
	},
}

// wsAuthError marks a failed WebSocket handshake authentication.
type wsAuthError struct{ reason string }

func (e *wsAuthError) Error() string { return e.reason }

// wsToken extracts a handshake token. Three transports are accepted because the
// callers differ:
//
//   - `?token=` — the Python bridges, which cannot set headers on connect;
//   - `Authorization: Bearer` — server-to-server clients and tests;
//   - `Sec-WebSocket-Protocol: espl, <token>` — the only header a browser
//     WebSocket can set (the `espl` sentinel must come first, per the same
//     convention the adapter gateway already uses).
func wsToken(r *http.Request) string {
	if q := r.URL.Query().Get("token"); q != "" {
		return q
	}
	if auth := r.Header.Get("Authorization"); auth != "" {
		if parts := strings.SplitN(auth, " ", 2); len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}
	protocols := websocket.Subprotocols(r)
	for _, p := range protocols {
		if p != "espl" && p != "" {
			return p
		}
	}
	return ""
}

// authenticateWS validates the handshake token. A missing token is refused as
// well as a bad one: before this, knowing an instance_id or app_id was enough to
// open a socket, because neither route sat behind any middleware.
func authenticateWS(r *http.Request) error {
	token := wsToken(r)
	if token == "" {
		return &wsAuthError{"missing token (use ?token=, Authorization, or the espl subprotocol)"}
	}
	if _, err := middleware.ParseToken(token); err != nil {
		return &wsAuthError{"invalid or expired token"}
	}
	return nil
}

// rejectWS answers a WebSocket handshake that was refused before upgrading, so
// the client gets a plain HTTP 401 instead of a socket that immediately closes.
func rejectWS(c *gin.Context, err error) {
	logger.Warn("WebSocket handshake rejected",
		logger.String("path", c.Request.URL.Path),
		logger.String("origin", c.Request.Header.Get("Origin")),
		logger.String("error", err.Error()),
	)
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
}

// AdapterWebSocket handles adapter (bridge) WebSocket connections.
// The bridge connects here over /ws/adapter?instance_id=xxx and reports
// inbound messages. Each reported message is persisted (with its full
// payload) and, if a broadcast callback is provided, fanned out to the
// adapter gateway (接入器) so external frameworks receive it.
//
// The bridge connection is also registered with the adapter gateway so that
// outbound commands (replies from external frameworks like LangBot) can be
// routed back to the correct bridge.
func AdapterWebSocket(
	hub *message.Hub,
	adapterService *service.AdapterService,
	messageService *service.MessageService,
	gateway *adaptergateway.Gateway,
	onInbound ...func(payload map[string]interface{}),
) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Authenticate before upgrading: a refused handshake then arrives as a
		// readable 401 instead of a socket that closes straight away.
		if err := authenticateWS(c.Request); err != nil {
			rejectWS(c, err)
			return
		}

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
		messageService.RecordHeartbeat(instanceID)

		// Register the bridge connection with the gateway so outbound commands
		// can be routed back to it. A single writer goroutine pumps ALL frames
		// (welcome, acks and outbound commands) onto this WebSocket so we never
		// have concurrent writes to the same connection.
		bc := adaptergateway.NewBridgeConn(instanceID)
		if gateway != nil {
			gateway.RegisterBridge(instanceID, bc)
		}

		// Register with the hub too, so routed messages can reach this instance.
		// The socket is already owned here (this loop reads, the goroutine
		// writes), hence the "only" variant: no second reader/writer.
		hubConn := &message.Connection{
			ID:         "adapter:" + instanceID,
			Conn:       conn,
			InstanceID: instanceID,
			Send:       make(chan []byte, 64),
			CreatedAt:  time.Now(),
		}
		hub.RegisterAdapterConn(hubConn)
		hubWriteDone := make(chan struct{})
		go func() {
			for {
				select {
				case data, ok := <-hubConn.Send:
					if !ok {
						return
					}
					if !bc.Send(data) {
						logger.Warn("Adapter bridge send buffer full",
							logger.String("instance_id", instanceID))
					}
				case <-hubWriteDone:
					return
				}
			}
		}()
		writeDone := make(chan struct{})
		go func() {
			for {
				select {
				case data, ok := <-bc.Chan():
					if !ok {
						return
					}
					_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
					if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
						return
					}
				case <-writeDone:
					return
				}
			}
		}()
		defer func() {
			close(writeDone)
			close(hubWriteDone)
			hub.UnregisterConnection("adapter:" + instanceID)
			if gateway != nil {
				gateway.UnregisterBridge(instanceID, bc)
			}
		}()

		// Helper to marshal a JSON frame and push it through the single writer.
		writeJSON := func(v interface{}) {
			data, err := json.Marshal(v)
			if err != nil {
				return
			}
			if !bc.Send(data) {
				logger.Warn("Adapter bridge send buffer full",
					logger.String("instance_id", instanceID))
			}
		}

		// Send welcome message
		writeJSON(map[string]interface{}{
			"type":      "connected",
			"timestamp": time.Now().Unix(),
			"message":   "Adapter WebSocket connected",
		})

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
				// Tool (inactive field) result frames are NOT messages: they
				// carry the answer to an outstanding tool_call and must be
				// handed back to the gateway's pending-call registry.
				if frameType, _ := payload["type"].(string); frameType == "tool_result" {
					if gateway != nil {
						result := payload
						if p, ok := payload["payload"].(map[string]interface{}); ok {
							result = p
						}
						gateway.HandleToolResult(instanceID, result)
					}
					writeJSON(map[string]interface{}{
						"type":      "ack",
						"timestamp": time.Now().Unix(),
					})
					continue
				}

				// Tool catalog frames let a bridge declare/advertise its own
				// inactive-field tools. They are registered but not persisted.
				if frameType, _ := payload["type"].(string); frameType == "tool_catalog" {
					if gateway != nil {
						registerToolsFromFrame(gateway, payload)
					}
					writeJSON(map[string]interface{}{
						"type":      "ack",
						"timestamp": time.Now().Unix(),
					})
					continue
				}

				// Inject instance_id into the payload for adapter gateway routing.
				// The bridge connects with ?instance_id=xxx in the query string, but
				// the payload it sends does not contain instance_id.  The adapter
				// gateway (接入器) and downstream frameworks (e.g. LangBot's ESPL
				// adapter) need instance_id on the event so they can route outbound
				// (reply) messages back to the correct bridge instance.
				//
				// We set both "instance_id" (canonical) and "instance" (legacy
				// alias used by some external adapters) so routing works regardless
				// of which key the downstream looks for.
				if instanceID != "" {
					if _, exists := payload["instance_id"]; !exists {
						payload["instance_id"] = instanceID
					}
					if _, exists := payload["instance"]; !exists {
						payload["instance"] = instanceID
					}
				}

				// Persist first: the stored row is the durable record and its ID
				// is what the ACK and the routing status refer to.
				eventID := persistInboundMessage(messageService, instanceID, payload)

				// Fan out the full message to the adapter gateway (接入器).
				if len(onInbound) > 0 && onInbound[0] != nil {
					onInbound[0](payload)
				}

				writeJSON(map[string]interface{}{
					"type":      "ack",
					"timestamp": time.Now().Unix(),
					"event_id":  eventID,
				})
				continue
			}

			// Raw text message, acknowledge without persistence
			writeJSON(map[string]interface{}{
				"type":      "ack",
				"timestamp": time.Now().Unix(),
				"data":      string(message),
			})
		}

		logger.Info("Adapter disconnected", logger.String("instance_id", instanceID))
	}
}

// AppWebSocket handles downstream app WebSocket connections.
//
// Unlike the bridge route, this socket is handed to the hub: the dispatcher's
// readPump/writePump own it end to end, so routed messages, ACKs and the welcome
// frame all leave through a single writer. Every inbound frame is persisted and
// then routed back through the hub, which is what closes the loop — before this,
// the app socket also only echoed.
func AppWebSocket(hub *message.Hub, messageService *service.MessageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Same handshake rule as the adapter socket: a valid token is required
		// before the upgrade, so an unknown client cannot subscribe silently.
		if err := authenticateWS(c.Request); err != nil {
			rejectWS(c, err)
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.Error("WebSocket upgrade failed", logger.String("error", err.Error()))
			return
		}

		// Get app ID from query params
		appID := c.Query("app_id")
		if appID == "" {
			_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"app_id required"}`))
			conn.Close()
			return
		}

		logger.Info("App connected", logger.String("app_id", appID))

		// Persistence hook: store the frame, then let the read pump ACK with the
		// event ID. Not every frame is a v3 envelope, so it is handed the raw
		// bytes and simply ignores anything that is not a JSON object.
		hub.SetMessageHandler(func(c *message.Connection, raw []byte) string {
			var payload map[string]interface{}
			if err := json.Unmarshal(raw, &payload); err != nil {
				return ""
			}
			return persistInboundMessage(messageService, c.InstanceID, payload)
		})

		hubConn := &message.Connection{
			ID:        "app:" + appID,
			AppID:     appID,
			Conn:      conn,
			Send:      make(chan []byte, 64),
			CreatedAt: time.Now(),
		}

		// Registering starts the pumps; queue the welcome behind the channel so
		// it goes out through the same writer as everything else.
		hub.RegisterAppConn(hubConn)
		if welcome, err := json.Marshal(map[string]interface{}{
			"type":      "connected",
			"timestamp": time.Now().Unix(),
			"message":   "App WebSocket connected",
		}); err == nil {
			select {
			case hubConn.Send <- welcome:
			default:
			}
		}

		// Block until the read pump tears the socket down; it unregisters itself.
		<-hubConn.Done()

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

	event, err := messageService.CreateAndRoute(req)
	if err != nil {
		logger.Warn("Failed to persist inbound message",
			logger.String("error", err.Error()))
		messageService.RecordError(instanceID)
		return ""
	}
	// Keep the instance-level counters the Monitoring page reads in step with
	// what actually got stored.
	messageService.RecordInbound(instanceID)
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
