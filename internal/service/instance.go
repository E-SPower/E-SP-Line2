package service

import (
	"errors"
	"time"

	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/internal/repository"
)

// InstanceService handles adapter instance operations
type InstanceService struct {
	repo        *repository.InstanceRepository
	sessionRepo *repository.AdapterSessionRepository

	// runner manages the Python adapter subprocess lifecycle (optional).
	runner *PythonRunner
}

// NewInstanceService creates a new instance service
func NewInstanceService(repo *repository.InstanceRepository, sessionRepo *repository.AdapterSessionRepository) *InstanceService {
	return &InstanceService{
		repo:        repo,
		sessionRepo: sessionRepo,
	}
}

// SetRunner attaches the Python process runner (used for WebUI start/stop).
func (s *InstanceService) SetRunner(runner *PythonRunner) {
	s.runner = runner
}

// Start starts the instance's Python adapter process (if a runner is attached).
func (s *InstanceService) Start(id string) error {
	if s.runner == nil {
		return errors.New("python runner is not initialized")
	}
	return s.runner.Start(id)
}

// Stop stops the instance's Python adapter process.
func (s *InstanceService) Stop(id string) error {
	if s.runner == nil {
		return errors.New("python runner is not initialized")
	}
	return s.runner.Stop(id)
}

// CreateInstanceRequest represents a create instance request
type CreateInstanceRequest struct {
	AdapterID  string `json:"adapter_id" binding:"required"`
	PlatformID string `json:"platform_id" binding:"required"`
	Name       string `json:"name" binding:"required"`
	Config     string `json:"config"`
	UserID     string `json:"user_id"`
}

// UpdateInstanceRequest represents an update instance request
type UpdateInstanceRequest struct {
	Name   string `json:"name"`
	Config string `json:"config"`
	Status string `json:"status"`
}

// Create creates a new instance
func (s *InstanceService) Create(req *CreateInstanceRequest) (*models.AdapterInstance, error) {
	instance := &models.AdapterInstance{
		ID:         models.GenerateID(),
		AdapterID:  req.AdapterID,
		PlatformID: req.PlatformID,
		Name:       req.Name,
		Config:     req.Config,
		Status:     "stopped",
		UserID:     req.UserID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.repo.Create(instance); err != nil {
		return nil, err
	}

	return instance, nil
}

// GetByID gets an instance by ID
func (s *InstanceService) GetByID(id string) (*models.AdapterInstance, error) {
	return s.repo.FindByID(id)
}

// GetByUserID gets instances by user ID
func (s *InstanceService) GetByUserID(userID string) ([]models.AdapterInstance, error) {
	return s.repo.FindByUserID(userID)
}

// Update updates an instance
func (s *InstanceService) Update(id string, req *UpdateInstanceRequest) (*models.AdapterInstance, error) {
	instance, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		instance.Name = req.Name
	}
	if req.Config != "" {
		instance.Config = req.Config
	}
	if req.Status != "" {
		instance.Status = req.Status
	}
	instance.UpdatedAt = time.Now()

	if err := s.repo.Update(instance); err != nil {
		return nil, err
	}

	return instance, nil
}

// Delete deletes an instance
func (s *InstanceService) Delete(id string) error {
	return s.repo.Delete(id)
}

// List lists all instances
func (s *InstanceService) List(limit, offset int) ([]models.AdapterInstance, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(limit, offset)
}

// GetSession gets the session for an instance
func (s *InstanceService) GetSession(instanceID string) (*models.AdapterSession, error) {
	return s.sessionRepo.FindByInstanceID(instanceID)
}

// UpdateSessionHeartbeat updates the heartbeat for a session
func (s *InstanceService) UpdateSessionHeartbeat(sessionID string) error {
	return s.sessionRepo.UpdateHeartbeat(sessionID)
}
