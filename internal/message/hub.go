package message

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/e-spl/e-sp-line2/internal/models"
	v3 "github.com/e-spl/e-sp-line2/internal/protocol/v3"
	"github.com/e-spl/e-sp-line2/pkg/logger"
)

// Hub coordinates message routing and dispatching
type Hub struct {
	router     *Router
	dispatcher *Dispatcher
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.RWMutex
}

// NewHub creates a new message hub
func NewHub() *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		router:     NewRouter(),
		dispatcher: NewDispatcher(),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// GetRouter returns the router
func (h *Hub) GetRouter() *Router {
	return h.router
}

// GetDispatcher returns the dispatcher
func (h *Hub) GetDispatcher() *Dispatcher {
	return h.dispatcher
}

// ProcessMessage validates, routes and dispatches an incoming message.
//
// It returns the number of frames actually enqueued on live connections and
// whether routing rules matched at all. The caller uses those two numbers to
// decide the stored status: a matched-but-undeliverable message is not the same
// as an unrouted one, and neither is a delivery.
func (h *Hub) ProcessMessage(envelope *v3.MessageEnvelope) (delivered int, matched bool, err error) {
	// Validate envelope
	if err := envelope.Validate(); err != nil {
		logger.Warn("Invalid message envelope",
			logger.String("error", err.Error()))
		return 0, false, err
	}

	// Route message
	targets, err := h.router.Route(envelope)
	if err != nil {
		logger.Error("Message routing failed",
			logger.String("error", err.Error()))
		return 0, false, err
	}

	if len(targets) == 0 {
		logger.Debug("No routing targets found",
			logger.String("event_id", envelope.EventID))
		return 0, false, nil
	}

	// Dispatch message
	delivered, err = h.dispatcher.Dispatch(envelope, targets)
	if err != nil {
		logger.Error("Message dispatching failed",
			logger.String("error", err.Error()))
		return 0, true, err
	}

	logger.Info("Message processed",
		logger.String("event_id", envelope.EventID),
		logger.String("event_type", string(envelope.EventType)),
		logger.Int("target_count", len(targets)),
		logger.Int("delivered_count", delivered))

	return delivered, true, nil
}

// RegisterAdapter registers an adapter connection
func (h *Hub) RegisterAdapter(conn *Connection) {
	conn.Type = "adapter"
	h.dispatcher.RegisterConnection(conn)
}

// RegisterApp registers an app connection
func (h *Hub) RegisterApp(conn *Connection) {
	conn.Type = "app"
	h.dispatcher.RegisterConnection(conn)
}

// UnregisterConnection unregisters a connection
func (h *Hub) UnregisterConnection(connID string) {
	h.dispatcher.UnregisterConnection(connID)
}

// SyncRules replaces the in-memory routing table with the given rules.
//
// The rules are the database's rows, converted to the router's shape. Parsing
// failures are logged and the rule is skipped rather than aborting the whole
// sync: one malformed `conditions` column must not silently disable every other
// rule. Conditions are the JSON object form the router's matchConditions reads
// (`{"event_type": "..."}`, `{"adapter_id": "..."}`); a blank column matches
// everything.
func (h *Hub) SyncRules(rules []models.RouteRule) {
	router := h.router
	for _, existing := range router.GetRules() {
		router.RemoveRule(existing.ID)
	}

	for _, rule := range rules {
		conditions := map[string]interface{}{}
		if strings.TrimSpace(rule.Conditions) != "" {
			if err := json.Unmarshal([]byte(rule.Conditions), &conditions); err != nil {
				logger.Warn("Skipping route rule with unparsable conditions",
					logger.String("rule_id", rule.ID),
					logger.String("error", err.Error()))
				continue
			}
		}

		router.AddRule(&RouteRule{
			ID:         rule.ID,
			PlatformID: rule.PlatformID,
			Priority:   rule.Priority,
			Conditions: conditions,
			TargetType: rule.TargetType,
			TargetID:   rule.TargetID,
			Enabled:    rule.Enabled,
		})
	}
}

// RegisterAdapterConn registers an adapter connection whose socket is owned by
// the caller, so routed frames are queued onto it without a second read/write
// pump racing the caller's own loop.
func (h *Hub) RegisterAdapterConn(conn *Connection) {
	conn.Type = "adapter"
	h.dispatcher.RegisterConnectionOnly(conn)
}

// RegisterAppConn registers an app connection and starts the hub-owned pumps.
func (h *Hub) RegisterAppConn(conn *Connection) {
	conn.Type = "app"
	h.dispatcher.RegisterConnection(conn)
}

// SetMessageHandler installs the persistence callback the app read pump calls
// for every inbound frame. It returns the stored event ID to echo in the ACK.
func (h *Hub) SetMessageHandler(fn func(conn *Connection, raw []byte) string) {
	h.dispatcher.SetMessageHandler(fn)
}

// SendCommand sends a command to the bridge that owns instanceID and returns how
// many live connections it reached (0 means nothing is connected).
//
// The frame is the flat bridge command documented in
// docs/developer-guide/bridge-protocol.md §3.2, not a v3 envelope: the bridges
// read `command_type`/`type` and `payload.cid`/`payload.toid`/`payload.text` off
// the top level (adapters/xianyu/main.py:197-220), so a nested envelope reaches
// them and is thrown away as "指令缺少 cid/toid".
func (h *Hub) SendCommand(instanceID, commandType string, payload map[string]interface{}, traceID string) (int, error) {
	cid := firstString(payload, "conversation_id", "cid", "target_id", "toid")
	toid := firstString(payload, "target_id", "toid", "sender_id", "conversation_id", "cid")
	text := firstString(payload, "text", "message_content")

	bridgePayload := map[string]interface{}{
		"cid":             cid,
		"conversation_id": cid,
		"toid":            toid,
		"target_id":       toid,
		"text":            text,
		"message_content": text,
		"command_type":    commandType,
		"instance_id":     instanceID,
	}
	if chain, ok := payload["message_chain"]; ok {
		bridgePayload["message_chain"] = chain
	}
	for key, value := range payload {
		if _, exists := bridgePayload[key]; !exists {
			bridgePayload[key] = value
		}
	}

	frame := map[string]interface{}{
		"type":         commandType,
		"command_type": commandType,
		"instance_id":  instanceID,
		"trace_id":     traceID,
		"payload":      bridgePayload,
	}

	data, err := json.Marshal(frame)
	if err != nil {
		return 0, err
	}

	return h.dispatcher.DispatchRaw(&RouteTarget{Type: "adapter", ID: instanceID}, data), nil
}

// firstString returns the first non-empty string among the given payload keys.
func firstString(payload map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v, ok := payload[key].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// BroadcastEvent broadcasts an event to all connections
func (h *Hub) BroadcastEvent(envelope *v3.MessageEnvelope) error {
	_, err := h.dispatcher.Dispatch(envelope, []*RouteTarget{
		{
			Type: "broadcast",
		},
	})
	return err
}

// GetStats returns hub statistics
func (h *Hub) GetStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	adapterConns := h.dispatcher.GetConnectionsByType("adapter")
	appConns := h.dispatcher.GetConnectionsByType("app")

	return map[string]interface{}{
		"adapter_connections": len(adapterConns),
		"app_connections":     len(appConns),
		"route_rules":         len(h.router.GetRules()),
		"timestamp":           time.Now().UnixMilli(),
	}
}

// Close closes the hub
func (h *Hub) Close() error {
	h.cancel()

	if err := h.router.Close(); err != nil {
		logger.Error("Failed to close router", logger.String("error", err.Error()))
	}

	if err := h.dispatcher.Close(); err != nil {
		logger.Error("Failed to close dispatcher", logger.String("error", err.Error()))
	}

	return nil
}
