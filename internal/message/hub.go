package message

import (
	"context"
	"fmt"
	"sync"
	"time"

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

// ProcessMessage processes an incoming message.
func (h *Hub) ProcessMessage(envelope *v3.MessageEnvelope) error {
	_, err := h.Process(envelope)
	return err
}

// Process is ProcessMessage with the number of matched route targets. Callers
// use it to tell "routed somewhere" from "no rule matched": the latter must stay
// replayable for the retry scanner instead of being marked routed forever.
func (h *Hub) Process(envelope *v3.MessageEnvelope) (int, error) {
	// Validate envelope
	if err := envelope.Validate(); err != nil {
		logger.Warn("Invalid message envelope",
			logger.String("error", err.Error()))
		return 0, err
	}

	// Route message
	targets, err := h.router.Route(envelope)
	if err != nil {
		logger.Error("Message routing failed",
			logger.String("error", err.Error()))
		return 0, err
	}

	if len(targets) == 0 {
		logger.Debug("No routing targets found",
			logger.String("event_id", envelope.EventID))
		return 0, nil
	}

	// Dispatch message
	delivered, err := h.dispatcher.Dispatch(envelope, targets)
	if err != nil {
		logger.Error("Message dispatching failed",
			logger.String("error", err.Error()))
		return len(targets), err
	}

	logger.Info("Message processed",
		logger.String("event_id", envelope.EventID),
		logger.String("event_type", string(envelope.EventType)),
		logger.Int("target_count", len(targets)),
		logger.Int("delivered", delivered))

	return len(targets), nil
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

// SendCommand sends a command to an adapter connection. No live connection for
// the instance is an error, not a no-op: the caller uses it to decide between
// MarkSent and MarkFailed, and a command that was never written to a socket must
// not be recorded as sent.
func (h *Hub) SendCommand(instanceID string, command *v3.OutboundCommandProto) error {
	envelope := v3.NewMessageEnvelope(
		command.InstanceID,
		instanceID,
		v3.EventCommandCreated,
		command,
	)

	delivered, err := h.dispatcher.Dispatch(envelope, []*RouteTarget{
		{
			Type: "adapter",
			ID:   instanceID,
		},
	})
	if err != nil {
		return err
	}
	if delivered == 0 {
		return fmt.Errorf("no adapter connection for instance %q", instanceID)
	}
	return nil
}

// ConnectionCount reports live connections of a type; used by tests and stats.
func (h *Hub) ConnectionCount(connType string) int {
	return len(h.dispatcher.GetConnectionsByType(connType))
}

// HasConnection reports whether any connection of the given type is registered
// for the given instance/app id.
func (h *Hub) HasConnection(connType, id string) bool {
	for _, conn := range h.dispatcher.GetConnectionsByType(connType) {
		if (connType == "adapter" && conn.InstanceID == id) || (connType == "app" && conn.AppID == id) {
			return true
		}
	}
	return false
}

// SyncRules replaces the in-memory rule table with the given rules. Route
// changes take effect on the next message instead of on the next restart.
func (h *Hub) SyncRules(rules []*RouteRule) {
	router := h.router
	router.mu.Lock()
	router.rules = rules
	router.mu.Unlock()
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
