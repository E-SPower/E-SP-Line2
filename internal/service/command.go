package service

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/e-spl/e-sp-line2/internal/message"
	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/internal/repository"
	"github.com/e-spl/e-sp-line2/pkg/logger"
)

// CommandService handles outbound command operations
type CommandService struct {
	repo *repository.OutboundCommandRepository

	// hub delivers a command to a connected adapter. Nil means "create only".
	hub *message.Hub

	// retryLoop state.
	retryMu   sync.Mutex
	retryStop chan struct{}
}

// NewCommandService creates a new command service
func NewCommandService(repo *repository.OutboundCommandRepository) *CommandService {
	return &CommandService{repo: repo}
}

// SetHub attaches the delivery hub. Called once by NewServices.
func (s *CommandService) SetHub(h *message.Hub) { s.hub = h }

// CreateCommandRequest represents a create command request
type CreateCommandRequest struct {
	InstanceID  string      `json:"instance_id" binding:"required"`
	CommandType string      `json:"command_type" binding:"required"`
	Payload     interface{} `json:"payload"`
	MaxRetries  int         `json:"max_retries"`
}

// CreateAndDispatch persists a new outbound command, delivers it to the target
// adapter, and records the outcome.
//
// Delivery is what makes the row mean something: `sent` only when a live
// connection accepted the frame, otherwise MarkFailed bumps the retry counter
// and parks the row in `retrying` for the background sweep. The dispatch itself
// never fails the write — the command is already durable and retryable.
func (s *CommandService) CreateAndDispatch(req *CreateCommandRequest) (*models.OutboundCommand, error) {
	cmd, err := s.Create(req)
	if err != nil {
		return nil, err
	}

	if err := s.dispatch(cmd); err != nil {
		logger.Warn("Command not delivered, parked for retry",
			logger.String("command_id", cmd.ID),
			logger.String("instance_id", cmd.InstanceID),
			logger.String("error", err.Error()))
	}

	// dispatch already moved the row to sent/retrying/failed; re-read so the
	// response carries the real status instead of the stale "created".
	if stored, err := s.repo.FindByID(cmd.ID); err == nil {
		return stored, nil
	}

	return cmd, nil
}

// Create creates a new outbound command without dispatching it.
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

// dispatch delivers one stored command and updates its status accordingly.
func (s *CommandService) dispatch(cmd *models.OutboundCommand) error {
	if s.hub == nil {
		return nil
	}

	// Claim the row first: the retry sweep scans `created`, which is the status
	// every command starts with, so without a claim it can grab this very row
	// while this call is still in flight and the two overwrite each other.
	claimed, err := s.repo.ClaimForDispatch(cmd.ID)
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}

	payload := decodePayload(cmd.Payload)

	delivered, err := s.hub.SendCommand(cmd.InstanceID, cmd.CommandType, payload, cmd.TraceID)
	if err != nil {
		return s.MarkFailed(cmd.ID)
	}
	if delivered == 0 {
		// No live connection for that instance: same handling as a send error.
		return s.MarkFailed(cmd.ID)
	}

	return s.MarkSent(cmd.ID)
}

// decodePayload parses a stored payload back into a map. A payload that is not
// a JSON object (a bare string, or empty) is passed through under "text" so a
// bridge still gets something to send.
func decodePayload(raw string) map[string]interface{} {
	payload := map[string]interface{}{}
	if raw == "" {
		return payload
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return map[string]interface{}{"text": raw}
	}
	return payload
}

// StartRetryLoop starts a background sweep that re-dispatches commands parked in
// `created`/`queued`/`retrying` — a bridge that reconnects after a failed send
// gets its backlog without anyone re-POSTing. It is idempotent and stopped by
// StopRetryLoop / Close.
func (s *CommandService) StartRetryLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}

	s.retryMu.Lock()
	defer s.retryMu.Unlock()
	if s.retryStop != nil {
		return
	}
	stop := make(chan struct{})
	s.retryStop = stop

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.retryPending()
			case <-stop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// StopRetryLoop stops the background sweep if it is running.
func (s *CommandService) StopRetryLoop() {
	s.retryMu.Lock()
	defer s.retryMu.Unlock()
	if s.retryStop != nil {
		close(s.retryStop)
		s.retryStop = nil
	}
}

// retryPending re-dispatches one batch of parked commands.
func (s *CommandService) retryPending() {
	cmds, err := s.ListPending(100)
	if err != nil {
		logger.Warn("Failed to list pending commands", logger.String("error", err.Error()))
		return
	}
	for i := range cmds {
		cmd := &cmds[i]
		if s.hub == nil {
			return
		}
		if err := s.dispatch(cmd); err != nil {
			logger.Debug("Retry did not deliver command",
				logger.String("command_id", cmd.ID))
		}
	}
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
