package service

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/e-spl/e-sp-line2/internal/message"
	"github.com/e-spl/e-sp-line2/internal/models"
	v3 "github.com/e-spl/e-sp-line2/internal/protocol/v3"
	"github.com/e-spl/e-sp-line2/internal/repository"
	"github.com/e-spl/e-sp-line2/pkg/logger"
)

// MessageService handles inbound message operations
type MessageService struct {
	repo         *repository.InboundEventRepository
	instanceRepo *repository.InstanceRepository
	hub          *message.Hub
}

// NewMessageService creates a new message service
func NewMessageService(repo *repository.InboundEventRepository, instanceRepo *repository.InstanceRepository) *MessageService {
	return &MessageService{
		repo:         repo,
		instanceRepo: instanceRepo,
	}
}

// SetHub attaches the message hub. Once set, every successfully persisted
// inbound message is handed to the hub for routing. The hub is optional so the
// service stays constructible in unit tests.
func (s *MessageService) SetHub(hub *message.Hub) {
	s.hub = hub
}

// CreateMessageRequest represents a create message request
type CreateMessageRequest struct {
	PlatformID     string      `json:"platform_id"`
	InstanceID     string      `json:"instance_id" binding:"required"`
	ConversationID string      `json:"conversation_id"`
	SenderID       string      `json:"sender_id"`
	SenderName     string      `json:"sender_name"`
	MessageType    string      `json:"message_type" binding:"required"`
	MessageContent string      `json:"message_content"`
	RawMessage     interface{} `json:"raw_message"`
	IdempotencyKey string      `json:"idempotency_key"`

	// EventType is the v3 event type the message is routed as. It defaults to
	// message.received; callers that report a lifecycle event (order.shipped,
	// logistics.updated, ...) set it explicitly.
	EventType string `json:"event_type"`
}

// Create persists a new inbound message and, when it is genuinely new, hands it
// to the hub for routing.
//
// Idempotency: a non-empty caller-supplied key is looked up first and the
// existing row is returned, so a bridge redelivering after a reconnect does not
// duplicate a message. An empty key gets a generated fallback so the unique
// index never sees the same value twice (see models.InboundEvent.IdempotencyKey).
func (s *MessageService) Create(req *CreateMessageRequest) (*models.InboundEvent, error) {
	if req.IdempotencyKey != "" {
		existing, err := s.repo.FindByIdempotencyKey(req.IdempotencyKey)
		if err == nil && existing != nil {
			return existing, nil // Return existing message for idempotency
		}
	}

	// Resolve the platform ID. The instance is authoritative when it exists:
	// it belongs to exactly one platform, and bridges report the platform as its
	// v3 code ("xianyu") while platform_id columns hold the platform row ID.
	// Only when the instance is unknown is the caller's value kept (and even
	// then it is only used for attribution and the fallback key).
	platformID := req.PlatformID
	if req.InstanceID != "" && s.instanceRepo != nil {
		if instance, err := s.instanceRepo.FindByID(req.InstanceID); err == nil && instance.PlatformID != "" {
			platformID = instance.PlatformID
		}
	}

	idempotencyKey := req.IdempotencyKey
	if idempotencyKey == "" {
		idempotencyKey = models.GenerateIdempotencyKey(platformID, req.InstanceID, models.GenerateID())
	}

	messageType := req.MessageType
	if messageType == "" {
		messageType = "text"
	}

	event := &models.InboundEvent{
		ID:             models.GenerateID(),
		PlatformID:     platformID,
		InstanceID:     req.InstanceID,
		ConversationID: req.ConversationID,
		SenderID:       req.SenderID,
		SenderName:     req.SenderName,
		MessageType:    messageType,
		MessageContent: req.MessageContent,
		RawMessage:     serializeRawMessage(req.RawMessage),
		IdempotencyKey: &idempotencyKey,
		TraceID:        models.GenerateTraceID(),
		Status:         "received",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.repo.Create(event); err != nil {
		return nil, err
	}

	s.dispatchToHub(event, req)
	return event, nil
}

// dispatchToHub converts a persisted event into a v3 envelope and routes it
// through the hub. Failures are logged rather than returned: the message is
// already durable, and a routing miss must not turn an accepted message into a
// 5xx that makes the bridge redeliver it.
func (s *MessageService) dispatchToHub(event *models.InboundEvent, req *CreateMessageRequest) {
	if s.hub == nil {
		return
	}

	eventType := v3.EventMessageReceived
	if req != nil && req.EventType != "" {
		eventType = v3.EventType(req.EventType)
	}

	payload := map[string]interface{}{
		"id":              event.ID,
		"platform_id":     event.PlatformID,
		"instance_id":     event.InstanceID,
		"conversation_id": event.ConversationID,
		"sender_id":       event.SenderID,
		"sender_name":     event.SenderName,
		"message_type":    event.MessageType,
		"message_content": event.MessageContent,
		"idempotency_key": idempotencyKeyValue(event.IdempotencyKey),
		"trace_id":        event.TraceID,
		"timestamp":       event.CreatedAt.UnixMilli(),
	}
	if req != nil && req.RawMessage != nil {
		payload["raw"] = req.RawMessage
	}

	envelope := v3.NewMessageEnvelope(event.PlatformID, event.InstanceID, eventType, payload)
	envelope.EventID = event.ID
	envelope.TraceID = event.TraceID

	matched, err := s.hub.Process(envelope)
	if err != nil {
		logger.Warn("Failed to route inbound message",
			logger.String("message_id", event.ID),
			logger.String("error", err.Error()))
		return
	}
	// "routed" is the state the retry scanner (ListPending) stops on, so only
	// flip it once a rule actually matched. A message that arrived before any
	// rule existed stays "received" and the scanner replays it.
	if matched == 0 {
		logger.Debug("No route matched, message left pending",
			logger.String("message_id", event.ID))
		return
	}
	if err := s.repo.UpdateStatus(event.ID, "routed"); err != nil {
		logger.Warn("Failed to mark message as routed",
			logger.String("message_id", event.ID),
			logger.String("error", err.Error()))
	}
	event.Status = "routed"
}

// GetByID gets a message by ID
func (s *MessageService) GetByID(id string) (*models.InboundEvent, error) {
	return s.repo.FindByID(id)
}

// FindByIdempotencyKey returns the message with the given idempotency key.
func (s *MessageService) FindByIdempotencyKey(key string) (*models.InboundEvent, error) {
	if key == "" {
		return nil, nil
	}
	return s.repo.FindByIdempotencyKey(key)
}

// Ack acknowledges a message
func (s *MessageService) Ack(id string) error {
	return s.repo.UpdateStatus(id, "acked")
}

// List lists all messages
func (s *MessageService) List(limit, offset int) ([]models.InboundEvent, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(limit, offset)
}

// ListByInstanceID lists messages by instance ID
func (s *MessageService) ListByInstanceID(instanceID string, limit, offset int) ([]models.InboundEvent, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListByInstanceID(instanceID, limit, offset)
}

// ListPending lists messages that have not reached a terminal state yet.
func (s *MessageService) ListPending(limit int) ([]models.InboundEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListPending(limit)
}

// RetryPending re-routes messages stuck in received/routed, so a hub that was
// down or a routing rule created after reception still gets a chance. It
// returns the number of messages replayed.
func (s *MessageService) RetryPending(limit int) (int, error) {
	if s.hub == nil {
		return 0, nil
	}
	pending, err := s.ListPending(limit)
	if err != nil {
		return 0, err
	}

	replayed := 0
	for i := range pending {
		event := &pending[i]
		eventType := v3.EventMessageReceived
		payload := map[string]interface{}{
			"id":              event.ID,
			"platform_id":     event.PlatformID,
			"instance_id":     event.InstanceID,
			"conversation_id": event.ConversationID,
			"sender_id":       event.SenderID,
			"sender_name":     event.SenderName,
			"message_type":    event.MessageType,
			"message_content": event.MessageContent,
			"idempotency_key": idempotencyKeyValue(event.IdempotencyKey),
			"trace_id":        event.TraceID,
			"timestamp":       event.CreatedAt.UnixMilli(),
		}
		if raw := strings.TrimSpace(event.RawMessage); raw != "" {
			payload["raw"] = json.RawMessage(raw)
		}

		envelope := v3.NewMessageEnvelope(event.PlatformID, event.InstanceID, eventType, payload)
		envelope.EventID = event.ID
		envelope.TraceID = event.TraceID

		matched, err := s.hub.Process(envelope)
		if err != nil {
			logger.Warn("Failed to replay pending message",
				logger.String("message_id", event.ID),
				logger.String("error", err.Error()))
			continue
		}
		if matched == 0 {
			continue // still no rule; stays pending for the next scan
		}
		if err := s.repo.UpdateStatus(event.ID, "routed"); err == nil {
			replayed++
		}
	}
	return replayed, nil
}

// StartRetryScanner runs RetryPending on a fixed interval until stop is closed.
// It is meant to be started once at boot, not per request.
func (s *MessageService) StartRetryScanner(interval time.Duration, stop <-chan struct{}) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if n, err := s.RetryPending(100); err != nil {
					logger.Warn("Message retry scan failed", logger.String("error", err.Error()))
				} else if n > 0 {
					logger.Info("Replayed pending messages", logger.Int("count", n))
				}
			case <-stop:
				return
			}
		}
	}()
}

// UpdateStatus updates message status
func (s *MessageService) UpdateStatus(id, status string) error {
	return s.repo.UpdateStatus(id, status)
}

// idempotencyKeyValue dereferences a nullable key for payload marshalling.
func idempotencyKeyValue(key *string) string {
	if key == nil {
		return ""
	}
	return *key
}

// serializeRawMessage serializes raw message to JSON string
func serializeRawMessage(raw interface{}) string {
	if raw == nil {
		return ""
	}
	// If already a string, return as-is
	if str, ok := raw.(string); ok {
		return str
	}
	// Otherwise, marshal to JSON
	data, err := json.Marshal(raw)
	if err != nil {
		return ""
	}
	return string(data)
}
