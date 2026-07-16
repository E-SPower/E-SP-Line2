package service

import (
	"time"

	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/internal/repository"
)

// CommandService handles outbound command operations
type CommandService struct {
	repo *repository.OutboundCommandRepository
}

// NewCommandService creates a new command service
func NewCommandService(repo *repository.OutboundCommandRepository) *CommandService {
	return &CommandService{repo: repo}
}

// CreateCommandRequest represents a create command request
type CreateCommandRequest struct {
	InstanceID  string      `json:"instance_id" binding:"required"`
	CommandType string      `json:"command_type" binding:"required"`
	Payload     interface{} `json:"payload"`
	MaxRetries  int         `json:"max_retries"`
}

// Create creates a new outbound command
func (s *CommandService) Create(req *CreateCommandRequest) (*models.OutboundCommand, error) {
	maxRetries := req.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	cmd := &models.OutboundCommand{
		ID:          models.GenerateID(),
		InstanceID:  req.InstanceID,
		CommandType: req.CommandType,
		Payload:     serializePayload(req.Payload),
		Status:      "created",
		RetryCount:  0,
		MaxRetries:  maxRetries,
		TraceID:     models.GenerateTraceID(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.Create(cmd); err != nil {
		return nil, err
	}

	return cmd, nil
}

// GetByID gets a command by ID
func (s *CommandService) GetByID(id string) (*models.OutboundCommand, error) {
	return s.repo.FindByID(id)
}

// UpdateStatus updates command status
func (s *CommandService) UpdateStatus(id, status string) error {
	return s.repo.UpdateStatus(id, status)
}

// List lists all commands
func (s *CommandService) List(limit, offset int) ([]models.OutboundCommand, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(limit, offset)
}

// ListPending lists pending commands
func (s *CommandService) ListPending(limit int) ([]models.OutboundCommand, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListPending(limit)
}

// MarkSent marks a command as sent
func (s *CommandService) MarkSent(id string) error {
	now := time.Now()
	cmd, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	cmd.Status = "sent"
	cmd.SentAt = &now
	cmd.UpdatedAt = now
	return s.repo.Update(cmd)
}

// MarkFailed marks a command as failed
func (s *CommandService) MarkFailed(id string) error {
	cmd, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	cmd.RetryCount++
	if cmd.RetryCount >= cmd.MaxRetries {
		cmd.Status = "failed"
	} else {
		cmd.Status = "retrying"
	}
	cmd.UpdatedAt = time.Now()
	return s.repo.Update(cmd)
}

// serializePayload serializes payload to JSON string
func serializePayload(payload interface{}) string {
	if payload == nil {
		return ""
	}
	// TODO: Implement proper JSON serialization
	return ""
}
