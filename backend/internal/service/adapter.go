package service

import (
	"errors"
	"time"

	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/internal/repository"
)

// AdapterService handles adapter operations
type AdapterService struct {
	repo    *repository.AdapterRepository
	pkgRepo *repository.AdapterPackageRepository
}

// NewAdapterService creates a new adapter service
func NewAdapterService(repo *repository.AdapterRepository, pkgRepo *repository.AdapterPackageRepository) *AdapterService {
	return &AdapterService{
		repo:    repo,
		pkgRepo: pkgRepo,
	}
}

// CreateAdapterRequest represents a create adapter request
type CreateAdapterRequest struct {
	PlatformID      string `json:"platform_id" binding:"required"`
	Name            string `json:"name" binding:"required"`
	Version         string `json:"version" binding:"required"`
	RuntimeType     string `json:"runtime_type"`
	ProtocolVersion string `json:"protocol_version"`
	Manifest        string `json:"manifest"`
}

// UpdateAdapterRequest represents an update adapter request
type UpdateAdapterRequest struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Status   string `json:"status"`
	Manifest string `json:"manifest"`
}

// Create creates a new adapter package
func (s *AdapterService) Create(req *CreateAdapterRequest) (*models.AdapterPackage, error) {
	adapter := &models.AdapterPackage{
		ID:              models.GenerateID(),
		PlatformID:      req.PlatformID,
		Name:            req.Name,
		Version:         req.Version,
		RuntimeType:     req.RuntimeType,
		ProtocolVersion: req.ProtocolVersion,
		Status:          "active",
		Manifest:        req.Manifest,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.repo.Create(adapter); err != nil {
		return nil, err
	}

	return adapter, nil
}

// GetByID gets an adapter by ID
func (s *AdapterService) GetByID(id string) (*models.AdapterPackage, error) {
	return s.repo.FindByID(id)
}

// GetByPlatformID gets adapters by platform ID
func (s *AdapterService) GetByPlatformID(platformID string) ([]models.AdapterPackage, error) {
	return s.repo.FindByPlatformID(platformID)
}

// Update updates an adapter
func (s *AdapterService) Update(id string, req *UpdateAdapterRequest) (*models.AdapterPackage, error) {
	adapter, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		adapter.Name = req.Name
	}
	if req.Version != "" {
		adapter.Version = req.Version
	}
	if req.Status != "" {
		adapter.Status = req.Status
	}
	if req.Manifest != "" {
		adapter.Manifest = req.Manifest
	}
	adapter.UpdatedAt = time.Now()

	if err := s.repo.Update(adapter); err != nil {
		return nil, err
	}

	return adapter, nil
}

// Delete deletes an adapter
func (s *AdapterService) Delete(id string) error {
	return s.repo.Delete(id)
}

// List lists all adapters
func (s *AdapterService) List(limit, offset int) ([]models.AdapterPackage, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(limit, offset)
}

// StartAdapter starts an adapter (placeholder for actual implementation)
func (s *AdapterService) StartAdapter(id string) error {
	adapter, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	if adapter.Status != "active" {
		return errors.New("adapter is not active")
	}

	// TODO: Implement actual adapter start logic
	return nil
}

// StopAdapter stops an adapter (placeholder for actual implementation)
func (s *AdapterService) StopAdapter(id string) error {
	// TODO: Implement actual adapter stop logic
	return nil
}
