package repository

import (
	"github.com/e-spl/e-sp-line2/internal/models"
	"gorm.io/gorm"
)

// InboundEventRepository handles inbound event database operations
type InboundEventRepository struct {
	db *gorm.DB
}

// NewInboundEventRepository creates a new inbound event repository
func NewInboundEventRepository(db *gorm.DB) *InboundEventRepository {
	return &InboundEventRepository{db: db}
}

// Create creates a new inbound event
func (r *InboundEventRepository) Create(event *models.InboundEvent) error {
	return r.db.Create(event).Error
}

// FindByID finds an event by ID
func (r *InboundEventRepository) FindByID(id string) (*models.InboundEvent, error) {
	var event models.InboundEvent
	err := r.db.First(&event, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// FindByIdempotencyKey finds an event by idempotency key
func (r *InboundEventRepository) FindByIdempotencyKey(key string) (*models.InboundEvent, error) {
	var event models.InboundEvent
	err := r.db.Where("idempotency_key = ?", key).First(&event).Error
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// Update updates an event
func (r *InboundEventRepository) Update(event *models.InboundEvent) error {
	return r.db.Save(event).Error
}

// UpdateStatus updates event status
func (r *InboundEventRepository) UpdateStatus(id, status string) error {
	return r.db.Model(&models.InboundEvent{}).Where("id = ?", id).
		Update("status", status).Error
}

// List lists all events
func (r *InboundEventRepository) List(limit, offset int) ([]models.InboundEvent, int64, error) {
	var events []models.InboundEvent
	var total int64

	r.db.Model(&models.InboundEvent{}).Count(&total)
	err := r.db.Limit(limit).Offset(offset).Order("created_at DESC").Find(&events).Error
	return events, total, err
}

// ListByInstanceID lists events by instance ID
func (r *InboundEventRepository) ListByInstanceID(instanceID string, limit, offset int) ([]models.InboundEvent, error) {
	var events []models.InboundEvent
	err := r.db.Where("instance_id = ?", instanceID).
		Limit(limit).Offset(offset).Order("created_at DESC").Find(&events).Error
	return events, err
}

// ListPending lists pending events for delivery
func (r *InboundEventRepository) ListPending(limit int) ([]models.InboundEvent, error) {
	var events []models.InboundEvent
	err := r.db.Where("status IN ?", []string{"received", "routed"}).
		Limit(limit).Order("created_at ASC").Find(&events).Error
	return events, err
}

// OutboundCommandRepository handles outbound command database operations
type OutboundCommandRepository struct {
	db *gorm.DB
}

// NewOutboundCommandRepository creates a new outbound command repository
func NewOutboundCommandRepository(db *gorm.DB) *OutboundCommandRepository {
	return &OutboundCommandRepository{db: db}
}

// Create creates a new command
func (r *OutboundCommandRepository) Create(cmd *models.OutboundCommand) error {
	return r.db.Create(cmd).Error
}

// FindByID finds a command by ID
func (r *OutboundCommandRepository) FindByID(id string) (*models.OutboundCommand, error) {
	var cmd models.OutboundCommand
	err := r.db.First(&cmd, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &cmd, nil
}

// Update updates a command
func (r *OutboundCommandRepository) Update(cmd *models.OutboundCommand) error {
	return r.db.Save(cmd).Error
}

// UpdateStatus updates command status
func (r *OutboundCommandRepository) UpdateStatus(id, status string) error {
	return r.db.Model(&models.OutboundCommand{}).Where("id = ?", id).
		Update("status", status).Error
}

// List lists all commands
func (r *OutboundCommandRepository) List(limit, offset int) ([]models.OutboundCommand, int64, error) {
	var commands []models.OutboundCommand
	var total int64

	r.db.Model(&models.OutboundCommand{}).Count(&total)
	err := r.db.Limit(limit).Offset(offset).Order("created_at DESC").Find(&commands).Error
	return commands, total, err
}

// ListPending lists pending commands for sending
func (r *OutboundCommandRepository) ListPending(limit int) ([]models.OutboundCommand, error) {
	var commands []models.OutboundCommand
	err := r.db.Where("status IN ?", []string{"created", "queued", "retrying"}).
		Limit(limit).Order("created_at ASC").Find(&commands).Error
	return commands, err
}

// RouteRuleRepository handles route rule database operations
type RouteRuleRepository struct {
	db *gorm.DB
}

// NewRouteRuleRepository creates a new route rule repository
func NewRouteRuleRepository(db *gorm.DB) *RouteRuleRepository {
	return &RouteRuleRepository{db: db}
}

// Create creates a new route rule
func (r *RouteRuleRepository) Create(rule *models.RouteRule) error {
	return r.db.Create(rule).Error
}

// FindByID finds a route rule by ID
func (r *RouteRuleRepository) FindByID(id string) (*models.RouteRule, error) {
	var rule models.RouteRule
	err := r.db.First(&rule, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// Update updates a route rule
func (r *RouteRuleRepository) Update(rule *models.RouteRule) error {
	return r.db.Save(rule).Error
}

// Delete deletes a route rule
func (r *RouteRuleRepository) Delete(id string) error {
	return r.db.Delete(&models.RouteRule{}, "id = ?", id).Error
}

// List lists all route rules
func (r *RouteRuleRepository) List(limit, offset int) ([]models.RouteRule, int64, error) {
	var rules []models.RouteRule
	var total int64

	r.db.Model(&models.RouteRule{}).Count(&total)
	err := r.db.Limit(limit).Offset(offset).Order("priority DESC").Find(&rules).Error
	return rules, total, err
}

// ListEnabled lists all enabled route rules
func (r *RouteRuleRepository) ListEnabled() ([]models.RouteRule, error) {
	var rules []models.RouteRule
	err := r.db.Where("enabled = ?", true).Order("priority DESC").Find(&rules).Error
	return rules, err
}

// FindByPlatformID finds route rules by platform ID
func (r *RouteRuleRepository) FindByPlatformID(platformID string) ([]models.RouteRule, error) {
	var rules []models.RouteRule
	err := r.db.Where("platform_id = ? AND enabled = ?", platformID, true).
		Order("priority DESC").Find(&rules).Error
	return rules, err
}
