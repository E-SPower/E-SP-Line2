package service

import (
	"fmt"
	"time"

	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/internal/repository"
)

// PlatformService handles platform operations
type PlatformService struct {
	repo *repository.PlatformRepository
}

// NewPlatformService creates a new platform service
func NewPlatformService(repo *repository.PlatformRepository) *PlatformService {
	return &PlatformService{repo: repo}
}

// CreatePlatformRequest represents a create platform request
type CreatePlatformRequest struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
}

// UpdatePlatformRequest represents an update platform request
type UpdatePlatformRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// Create creates a new platform
func (s *PlatformService) Create(req *CreatePlatformRequest) (*models.Platform, error) {
	// Check if code already exists
	existing, _ := s.repo.FindByCode(req.Code)
	if existing != nil {
		return nil, fmt.Errorf("%w: platform code %q already exists", ErrConflict, req.Code)
	}

	platform := &models.Platform{
		ID:          models.GenerateID(),
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.Create(platform); err != nil {
		return nil, err
	}

	return platform, nil
}

// GetByID gets a platform by ID
func (s *PlatformService) GetByID(id string) (*models.Platform, error) {
	platform, err := s.repo.FindByID(id)
	if err != nil {
		return nil, notFound("platform", id, err)
	}
	return platform, nil
}

// GetByCode gets a platform by code
func (s *PlatformService) GetByCode(code string) (*models.Platform, error) {
	return s.repo.FindByCode(code)
}

// Update updates a platform
func (s *PlatformService) Update(id string, req *UpdatePlatformRequest) (*models.Platform, error) {
	platform, err := s.repo.FindByID(id)
	if err != nil {
		return nil, notFound("platform", id, err)
	}

	if req.Name != "" {
		platform.Name = req.Name
	}
	if req.Description != "" {
		platform.Description = req.Description
	}
	if req.Status != "" {
		platform.Status = req.Status
	}
	platform.UpdatedAt = time.Now()

	if err := s.repo.Update(platform); err != nil {
		return nil, err
	}

	return platform, nil
}

// Delete deletes a platform. Deleting an unknown id is 404, not a silent 200.
func (s *PlatformService) Delete(id string) error {
	if err := s.repo.Delete(id); err != nil {
		return notFound("platform", id, err)
	}
	return nil
}

// List lists all platforms
func (s *PlatformService) List(limit, offset int) ([]models.Platform, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(limit, offset)
}

// ListActive lists all active platforms
func (s *PlatformService) ListActive() ([]models.Platform, error) {
	return s.repo.ListActive()
}
