package service

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/e-spl/e-sp-line2/internal/message"
	"github.com/e-spl/e-sp-line2/internal/models"
	v3 "github.com/e-spl/e-sp-line2/internal/protocol/v3"
	"github.com/e-spl/e-sp-line2/internal/repository"
	"github.com/e-spl/e-sp-line2/pkg/logger"
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

// DispatchCommand pushes a persisted command to its adapter connection and
// records the outcome: MarkSent on delivery, MarkFailed otherwise (which retries
// up to MaxRetries before settling on "failed"). Both the HTTP handler and the
// retry scanner go through here so the transition lives in one place.
//
// The returned error is the dispatch error, not the status-write error; the
// former is what the caller reports, and a status write that fails after a
// successful dispatch only costs a duplicate retry.
func (s *CommandService) DispatchCommand(hub *message.Hub, cmd *models.OutboundCommand) error {
	if hub == nil {
		err := errors.New("no message hub")
		_ = s.MarkFailed(cmd.ID)
		return err
	}

	payload, targetID := decodeCommandPayload(cmd.Payload)
	proto := v3.NewOutboundCommand(cmd.CommandType, cmd.InstanceID, targetID, payload)
	proto.TraceID = cmd.TraceID

	if err := hub.SendCommand(cmd.InstanceID, proto); err != nil {
		if ferr := s.MarkFailed(cmd.ID); ferr != nil {
			logger.Warn("Failed to mark command as failed",
				logger.String("command_id", cmd.ID),
				logger.String("error", ferr.Error()))
		}
		return err
	}

	if err := s.MarkSent(cmd.ID); err != nil {
		logger.Warn("Failed to mark command as sent",
			logger.String("command_id", cmd.ID),
			logger.String("error", err.Error()))
	}
	return nil
}

// decodeCommandPayload decodes the stored JSON payload into a wire value and
// pulls the conversation/target id out of it so the v3 command envelope is
// addressable. An unparsable payload is passed through as the raw string rather
// than dropped.
func decodeCommandPayload(raw string) (interface{}, string) {
	if raw == "" {
		return nil, ""
	}
	var payload interface{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return raw, ""
	}

	targetID := ""
	if m, ok := payload.(map[string]interface{}); ok {
		for _, key := range []string{"conversation_id", "cid", "target_id", "toid"} {
			if v, ok := m[key].(string); ok && v != "" {
				targetID = v
				break
			}
		}
	}
	return payload, targetID
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

// StartRetryScanner periodically replays pending commands through dispatch until
// stop is closed. Rows left in created/queued/retrying (adapter was offline, or
// the process died mid-dispatch) get MaxRetries attempts before MarkFailed
// settles them on "failed".
func (s *CommandService) StartRetryScanner(interval time.Duration, stop <-chan struct{}, dispatch func(*models.OutboundCommand)) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.retryPending(dispatch)
			case <-stop:
				return
			}
		}
	}()
}

// retryPending dispatches one batch of pending commands.
func (s *CommandService) retryPending(dispatch func(*models.OutboundCommand)) {
	if dispatch == nil {
		return
	}
	pending, err := s.ListPending(100)
	if err != nil {
		logger.Warn("Command retry scan failed", logger.String("error", err.Error()))
		return
	}
	for i := range pending {
		dispatch(&pending[i])
	}
	if len(pending) > 0 {
		logger.Info("Retried pending commands", logger.Int("count", len(pending)))
	}
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
	// If already a string, return as-is
	if str, ok := payload.(string); ok {
		return str
	}
	// Otherwise, marshal to JSON
	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(data)
}
