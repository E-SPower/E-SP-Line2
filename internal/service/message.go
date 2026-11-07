package service

import (
	"encoding/json"
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

	// hub routes a stored event to its targets. Nil means "store only", which
	// keeps the service usable in tests and keeps Create from routing twice.
	hub *message.Hub
}

// NewMessageService creates a new message service
func NewMessageService(repo *repository.InboundEventRepository, instanceRepo *repository.InstanceRepository) *MessageService {
	return &MessageService{
		repo:         repo,
		instanceRepo: instanceRepo,
	}
}

// SetHub attaches the routing hub. Called once by NewServices.
func (s *MessageService) SetHub(h *message.Hub) { s.hub = h }

// CreateMessageRequest represents a create message request
type CreateMessageRequest struct {
	PlatformID     string      `json:"platform_id" binding:"required"`
	InstanceID     string      `json:"instance_id" binding:"required"`
	ConversationID string      `json:"conversation_id"`
	SenderID       string      `json:"sender_id"`
	SenderName     string      `json:"sender_name"`
	MessageType    string      `json:"message_type" binding:"required"`
	MessageContent string      `json:"message_content"`
	RawMessage     interface{} `json:"raw_message"`
	IdempotencyKey string      `json:"idempotency_key"`
}

// Create creates a new inbound message.
//
// Idempotency: when the caller supplies a key, an existing event with that key
// is returned instead of inserting a second row. When the caller supplies none,
// a key is generated here rather than storing "" — the column is under a UNIQUE
// index, and empty strings collide while NULLs do not. Every row therefore has a
// real key, so the deduplication query can always be trusted.
func (s *MessageService) Create(req *CreateMessageRequest) (*models.InboundEvent, error) {
	event, _, err := s.create(req)
	return event, err
}

// create is Create plus a "this key already existed" bit, so the ingest handler
// can answer 200 for a replay and 201 for a new row.
func (s *MessageService) create(req *CreateMessageRequest) (*models.InboundEvent, bool, error) {
	// Resolve platform ID if not provided: prefer explicit value,
	// otherwise derive it from the adapter instance.
	platformID := req.PlatformID
	if platformID == "" && req.InstanceID != "" && s.instanceRepo != nil {
		if instance, err := s.instanceRepo.FindByID(req.InstanceID); err == nil {
			platformID = instance.PlatformID
		}
	}

	key := req.IdempotencyKey
	if key == "" {
		key = models.GenerateIdempotencyKey(platformID, req.InstanceID, models.GenerateID())
	} else if existing, err := s.repo.FindByIdempotencyKey(key); err == nil && existing != nil {
		return existing, true, nil // Return existing message for idempotency
	}

	event := &models.InboundEvent{
		ID:             models.GenerateID(),
		PlatformID:     platformID,
		InstanceID:     req.InstanceID,
		ConversationID: req.ConversationID,
		SenderID:       req.SenderID,
		SenderName:     req.SenderName,
		MessageType:    req.MessageType,
		MessageContent: req.MessageContent,
		RawMessage:     serializeRawMessage(req.RawMessage),
		IdempotencyKey: &key,
		TraceID:        models.GenerateTraceID(),
		Status:         "received",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.repo.Create(event); err != nil {
		// Two concurrent deliveries of the same key: the pre-insert lookup can
		// both miss. Re-read so the loser of the race still answers 200 with the
		// winner's row instead of surfacing a UNIQUE constraint error.
		if existing, lookupErr := s.repo.FindByIdempotencyKey(key); lookupErr == nil && existing != nil {
			return existing, true, nil
		}
		return nil, false, err
	}

	return event, false, nil
}

// GetByID gets a message by ID
func (s *MessageService) GetByID(id string) (*models.InboundEvent, error) {
	return s.repo.FindByID(id)
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

// ListPending lists pending messages
func (s *MessageService) ListPending(limit int) ([]models.InboundEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListPending(limit)
}

// UpdateStatus updates message status
func (s *MessageService) UpdateStatus(id, status string) error {
	return s.repo.UpdateStatus(id, status)
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

// CreateAndRoute creates an inbound message and then routes it through the hub.
//
// Routing is best-effort and never fails the write: the row is already the
// durable record, and the status says how far it got. `delivered` means at least
// one live connection received it, `routed` means a rule matched but nobody was
// listening, `received` means no rule matched. An unroutable event stays
// `received` on purpose — ListPending picks it up again if a rule appears.
func (s *MessageService) CreateAndRoute(req *CreateMessageRequest) (*models.InboundEvent, error) {
	event, _, err := s.CreateAndRouteIdempotent(req)
	return event, err
}

// CreateAndRouteIdempotent is CreateAndRoute plus the "key already existed" bit
// for the HTTP layer.
func (s *MessageService) CreateAndRouteIdempotent(req *CreateMessageRequest) (*models.InboundEvent, bool, error) {
	event, existed, err := s.create(req)
	if err != nil {
		return nil, false, err
	}
	// A replay returns the earlier row untouched: it was routed once already.
	if existed {
		return event, true, nil
	}
	s.Route(event)
	return event, false, nil
}

// Route hands a stored event to the hub and records the outcome on the row.
func (s *MessageService) Route(event *models.InboundEvent) {
	if s.hub == nil || event == nil {
		return
	}

	envelope := v3.NewMessageEnvelope(
		event.PlatformID,
		event.InstanceID,
		v3.EventMessageReceived,
		map[string]interface{}{
			"id":              event.ID,
			"platform_id":     event.PlatformID,
			"instance_id":     event.InstanceID,
			"conversation_id": event.ConversationID,
			"sender_id":       event.SenderID,
			"sender_name":     event.SenderName,
			"message_type":    event.MessageType,
			"message_content": event.MessageContent,
			"idempotency_key": event.IdempotencyKey,
			"trace_id":        event.TraceID,
			"raw_message":     event.RawMessage,
		},
	)

	delivered, matched, err := s.hub.ProcessMessage(envelope)
	if err != nil {
		logger.Warn("Failed to route inbound message",
			logger.String("event_id", event.ID),
			logger.String("error", err.Error()))
		return
	}

	status := "received"
	switch {
	case delivered > 0:
		status = "delivered"
	case matched:
		status = "routed"
	}
	if status == event.Status {
		return
	}
	if err := s.repo.UpdateStatus(event.ID, status); err != nil {
		logger.Warn("Failed to record message routing status",
			logger.String("event_id", event.ID),
			logger.String("error", err.Error()))
		return
	}
	event.Status = status
}
