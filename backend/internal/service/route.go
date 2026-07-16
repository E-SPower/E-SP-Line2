package service

import (
	"time"

	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/internal/repository"
)

// RouteService handles route rule operations
type RouteService struct {
	repo *repository.RouteRuleRepository
}

// NewRouteService creates a new route service
func NewRouteService(repo *repository.RouteRuleRepository) *RouteService {
	return &RouteService{repo: repo}
}

// CreateRouteRequest represents a create route request
type CreateRouteRequest struct {
	Name       string `json:"name" binding:"required"`
	PlatformID string `json:"platform_id"`
	InstanceID string `json:"instance_id"`
	Priority   int    `json:"priority"`
	Conditions string `json:"conditions"`
	TargetType string `json:"target_type" binding:"required"`
	TargetID   string `json:"target_id" binding:"required"`
	Enabled    bool   `json:"enabled"`
}

// UpdateRouteRequest represents an update route request
type UpdateRouteRequest struct {
	Name       string `json:"name"`
	Priority   int    `json:"priority"`
	Conditions string `json:"conditions"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	Enabled    *bool  `json:"enabled"`
}

// Create creates a new route rule
func (s *RouteService) Create(req *CreateRouteRequest) (*models.RouteRule, error) {
	rule := &models.RouteRule{
		ID:         models.GenerateID(),
		Name:       req.Name,
		PlatformID: req.PlatformID,
		InstanceID: req.InstanceID,
		Priority:   req.Priority,
		Conditions: req.Conditions,
		TargetType: req.TargetType,
		TargetID:   req.TargetID,
		Enabled:    req.Enabled,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.repo.Create(rule); err != nil {
		return nil, err
	}

	return rule, nil
}

// GetByID gets a route rule by ID
func (s *RouteService) GetByID(id string) (*models.RouteRule, error) {
	return s.repo.FindByID(id)
}

// Update updates a route rule
func (s *RouteService) Update(id string, req *UpdateRouteRequest) (*models.RouteRule, error) {
	rule, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		rule.Name = req.Name
	}
	if req.Priority != 0 {
		rule.Priority = req.Priority
	}
	if req.Conditions != "" {
		rule.Conditions = req.Conditions
	}
	if req.TargetType != "" {
		rule.TargetType = req.TargetType
	}
	if req.TargetID != "" {
		rule.TargetID = req.TargetID
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	rule.UpdatedAt = time.Now()

	if err := s.repo.Update(rule); err != nil {
		return nil, err
	}

	return rule, nil
}

// Delete deletes a route rule
func (s *RouteService) Delete(id string) error {
	return s.repo.Delete(id)
}

// List lists all route rules
func (s *RouteService) List(limit, offset int) ([]models.RouteRule, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(limit, offset)
}

// ListEnabled lists all enabled route rules
func (s *RouteService) ListEnabled() ([]models.RouteRule, error) {
	return s.repo.ListEnabled()
}

// GetByPlatformID gets route rules by platform ID
func (s *RouteService) GetByPlatformID(platformID string) ([]models.RouteRule, error) {
	return s.repo.FindByPlatformID(platformID)
}
