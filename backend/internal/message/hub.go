package message

import (
	"context"
	"sync"
	"time"

	"github.com/e-spl/e-sp-line2/internal/protocol/v3"
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

// ProcessMessage processes an incoming message
func (h *Hub) ProcessMessage(envelope *v3.MessageEnvelope) error {
	// Validate envelope
	if err := envelope.Validate(); err != nil {
		logger.Warn("Invalid message envelope",
			logger.String("error", err.Error()))
		return err
	}

	// Route message
	targets, err := h.router.Route(envelope)
	if err != nil {
		logger.Error("Message routing failed",
			logger.String("error", err.Error()))
		return err
	}

	if len(targets) == 0 {
		logger.Debug("No routing targets found",
			logger.String("event_id", envelope.EventID))
		return nil
	}

	// Dispatch message
	if err := h.dispatcher.Dispatch(envelope, targets); err != nil {
		logger.Error("Message dispatching failed",
			logger.String("error", err.Error()))
		return err
	}

	logger.Info("Message processed",
		logger.String("event_id", envelope.EventID),
		logger.String("event_type", string(envelope.EventType)),
		logger.Int("target_count", len(targets)))

	return nil
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

// SendCommand sends a command to an adapter
func (h *Hub) SendCommand(instanceID string, command *v3.OutboundCommandProto) error {
	envelope := v3.NewMessageEnvelope(
		command.InstanceID,
		instanceID,
		v3.EventCommandCreated,
		command,
	)

	return h.dispatcher.Dispatch(envelope, []*RouteTarget{
		{
			Type: "adapter",
			ID:   instanceID,
		},
	})
}

// BroadcastEvent broadcasts an event to all connections
func (h *Hub) BroadcastEvent(envelope *v3.MessageEnvelope) error {
	return h.dispatcher.Dispatch(envelope, []*RouteTarget{
		{
			Type: "broadcast",
		},
	})
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
