package service

import (
	"encoding/json"
	"time"

	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/internal/repository"
)

// MessageService handles inbound message operations
type MessageService struct {
	repo         *repository.InboundEventRepository
	instanceRepo *repository.InstanceRepository
}

// NewMessageService creates a new message service
func NewMessageService(repo *repository.InboundEventRepository, instanceRepo *repository.InstanceRepository) *MessageService {
	return &MessageService{
		repo:         repo,
		instanceRepo: instanceRepo,
	}
}

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

// Create creates a new inbound message
func (s *MessageService) Create(req *CreateMessageRequest) (*models.InboundEvent, error) {
	// Check for duplicate message
	if req.IdempotencyKey != "" {
		existing, _ := s.repo.FindByIdempotencyKey(req.IdempotencyKey)
		if existing != nil {
			return existing, nil // Return existing message for idempotency
		}
	}

	// Resolve platform ID if not provided: prefer explicit value,
	// otherwise derive it from the adapter instance.
	platformID := req.PlatformID
	if platformID == "" && req.InstanceID != "" && s.instanceRepo != nil {
		if instance, err := s.instanceRepo.FindByID(req.InstanceID); err == nil {
			platformID = instance.PlatformID
		}
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
		IdempotencyKey: req.IdempotencyKey,
		TraceID:        models.GenerateTraceID(),
		Status:         "received",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.repo.Create(event); err != nil {
		return nil, err
	}

	return event, nil
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
	data, err := json.Marshal(raw)
	if err != nil {
		return ""
	}
	return string(data)
}
