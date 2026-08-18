package service

import (
	"errors"
	"time"

	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/internal/repository"
	"github.com/e-spl/e-sp-line2/pkg/logger"
)

// InstanceService handles adapter instance operations
type InstanceService struct {
	repo        *repository.InstanceRepository
	sessionRepo *repository.AdapterSessionRepository

	// runner manages the Python adapter subprocess lifecycle (optional).
	runner *PythonRunner

	// installer installs adapter Python dependencies on instance creation.
	installer *DependencyInstaller

	// dirs manages per-instance sandbox copies of adapter code.
	dirs *InstanceDirManager
}

// NewInstanceService creates a new instance service
func NewInstanceService(repo *repository.InstanceRepository, sessionRepo *repository.AdapterSessionRepository) *InstanceService {
	return &InstanceService{
		repo:        repo,
		sessionRepo: sessionRepo,
	}
}

// SetInstaller attaches the dependency installer used on instance creation.
func (s *InstanceService) SetInstaller(installer *DependencyInstaller) {
	s.installer = installer
}

// SetDirManager attaches the instance sandbox manager.
func (s *InstanceService) SetDirManager(dirs *InstanceDirManager) {
	s.dirs = dirs
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

// ReadLog returns the instance's adapter process log, optionally filtered by
// level (debug/info/warning/error), keyword (case-insensitive substring,
// space-separated words are ANDed) and time range (from/to). maxLines <= 0
// returns the full log.
func (s *InstanceService) ReadLog(id string, maxLines int, level, keyword, from, to string) (string, error) {
	if s.runner == nil {
		return "", errors.New("python runner is not initialized")
	}
	return s.runner.ReadLog(id, maxLines, level, keyword, from, to)
}

// LogHeatmap returns per-hour log level counts for an instance.
func (s *InstanceService) LogHeatmap(id, level string) ([]LogHeatmapEntry, error) {
	if s.runner == nil {
		return nil, errors.New("python runner is not initialized")
	}
	return s.runner.LogHeatmap(id, level)
}

// ClearLog truncates the instance's log file.
func (s *InstanceService) ClearLog(id string) (int64, error) {
	if s.runner == nil {
		return 0, errors.New("python runner is not initialized")
	}
	return s.runner.ClearLog(id)
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

// Create creates a new instance. The instance is placed into "initializing"
// state: a sandboxed copy of the adapter code is created under
// data/instances/<id>/adapter (excluding __pycache__), and its Python
// dependencies are installed automatically in the background. Once
// installation finishes the status becomes "stopped" (ready to start) or
// "error" if installation failed.
func (s *InstanceService) Create(req *CreateInstanceRequest) (*models.AdapterInstance, error) {
	instance := &models.AdapterInstance{
		ID:         models.GenerateID(),
		AdapterID:  req.AdapterID,
		PlatformID: req.PlatformID,
		Name:       req.Name,
		Config:     req.Config,
		Status:     "initializing",
		UserID:     req.UserID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.repo.Create(instance); err != nil {
		return nil, err
	}

	// Locate the source adapter directory (platform code) so we can copy it
	// into the instance sandbox. Reuse the runner's resolver if available.
	platformCode, err := s.resolvePlatformCode(instance.ID)
	if err != nil {
		instance.Status = "error"
		_ = s.repo.Update(instance)
		logger.Warn("Failed to resolve platform code for instance",
			logger.String("instance_id", instance.ID),
			logger.String("error", err.Error()))
		return instance, nil
	}

	// Create the sandboxed copy of the adapter code (no __pycache__).
	adapterDir, err := s.dirs.CopyAdapter(instance.ID, platformCode)
	if err != nil {
		instance.Status = "error"
		_ = s.repo.Update(instance)
		logger.Warn("Failed to create instance sandbox",
			logger.String("instance_id", instance.ID),
			logger.String("error", err.Error()))
		return instance, nil
	}

	// Automatically install the adapter's Python dependencies inside the
	// sandbox copy. The WebUI polls the init status while it is
	// "initializing".
	if s.installer != nil {
		s.installer.InstallDependencies(instance.ID, adapterDir, platformCode, s.updateInstanceStatus)
	} else {
		// No installer attached (e.g. tests): move straight to "stopped".
		instance.Status = "stopped"
		if err := s.repo.Update(instance); err != nil {
			logger.Warn("Failed to update instance status after create",
				logger.String("instance_id", instance.ID),
				logger.String("error", err.Error()))
		}
	}

	return instance, nil
}

// resolvePlatformCode resolves the source adapter directory name (platform
// code) for an instance, delegating to the PythonRunner resolver when
// available.
func (s *InstanceService) resolvePlatformCode(instanceID string) (string, error) {
	if s.runner != nil {
		return s.runner.resolvePlatformCode(instanceID)
	}
	// Fallback: use the platform id directly.
	instance, err := s.repo.FindByID(instanceID)
	if err != nil {
		return "", err
	}
	return instance.PlatformID, nil
}

// updateInstanceStatus updates an instance's status field.
func (s *InstanceService) updateInstanceStatus(instanceID, status string) error {
	instance, err := s.repo.FindByID(instanceID)
	if err != nil {
		return err
	}
	instance.Status = status
	instance.UpdatedAt = time.Now()
	return s.repo.Update(instance)
}

// InitStatus returns the dependency installation status for an instance.
// If the installer is not attached, it reports the instance as ready.
func (s *InstanceService) InitStatus(instanceID string) map[string]interface{} {
	instance, err := s.repo.FindByID(instanceID)
	if err != nil {
		return map[string]interface{}{"status": "unknown", "message": "instance not found"}
	}

	if s.installer == nil {
		return map[string]interface{}{
			"instance_id": instanceID,
			"status":      "done",
			"progress":    100,
			"message":     "instances 未启用依赖安装",
			"instance_status": instance.Status,
		}
	}

	state, ok := s.installer.GetState(instanceID)
	if !ok {
		// No install state tracked — either it never needed install (old
		// instance) or install already finished long ago. Fall back to the
		// instance status.
		st := instance.Status
		if st == "initializing" {
			st = "installing"
		}
		return map[string]interface{}{
			"instance_id":     instanceID,
			"status":          st,
			"progress":        progressForStatus(st),
			"message":         messageForStatus(st),
			"instance_status": instance.Status,
		}
	}

	return map[string]interface{}{
		"instance_id":     instanceID,
		"status":          state.Status,
		"progress":        state.Progress,
		"message":         state.Message,
		"error":           state.Error,
		"instance_status": instance.Status,
	}
}

// progressForStatus returns a coarse progress value for common statuses.
func progressForStatus(status string) int {
	switch status {
	case "running":
		return 100
	case "error":
		return 100
	case "installing", "initializing":
		return 50
	default:
		return 100
	}
}

// messageForStatus returns a human-readable message for a status.
func messageForStatus(status string) string {
	switch status {
	case "running":
		return "实例运行中"
	case "error":
		return "实例异常"
	case "installing", "initializing":
		return "正在初始化..."
	default:
		return "实例已就绪"
	}
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

// Delete deletes an instance. If the instance's Python adapter process is
// running, it is stopped first to avoid orphan processes. The instance's
// sandboxed adapter copy under data/instances/<id> is removed as well.
func (s *InstanceService) Delete(id string) error {
	if s.runner != nil && s.runner.IsRunning(id) {
		if err := s.runner.Stop(id); err != nil {
			return err
		}
	}
	if s.dirs != nil {
		if err := s.dirs.Remove(id); err != nil {
			logger.Warn("Failed to remove instance sandbox",
				logger.String("instance_id", id),
				logger.String("error", err.Error()))
		}
	}
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
