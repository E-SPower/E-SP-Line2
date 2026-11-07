package repository

import (
	"time"

	"github.com/e-spl/e-sp-line2/internal/models"
	"gorm.io/gorm"
)

// InstanceRepository handles adapter instance database operations
type InstanceRepository struct {
	db *gorm.DB
}

// NewInstanceRepository creates a new instance repository
func NewInstanceRepository(db *gorm.DB) *InstanceRepository {
	return &InstanceRepository{db: db}
}

// Create creates a new instance
func (r *InstanceRepository) Create(instance *models.AdapterInstance) error {
	return r.db.Create(instance).Error
}

// FindByID finds an instance by ID
func (r *InstanceRepository) FindByID(id string) (*models.AdapterInstance, error) {
	var instance models.AdapterInstance
	err := r.db.First(&instance, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

// FindByUserID finds instances by user ID
func (r *InstanceRepository) FindByUserID(userID string) ([]models.AdapterInstance, error) {
	var instances []models.AdapterInstance
	err := r.db.Where("user_id = ?", userID).Find(&instances).Error
	return instances, err
}

// FindByAdapterID finds instances belonging to the given adapter.
func (r *InstanceRepository) FindByAdapterID(adapterID string) ([]models.AdapterInstance, error) {
	var instances []models.AdapterInstance
	err := r.db.Where("adapter_id = ?", adapterID).Find(&instances).Error
	return instances, err
}

// Update updates an instance
func (r *InstanceRepository) Update(instance *models.AdapterInstance) error {
	return r.db.Save(instance).Error
}

// Delete deletes an instance. It returns gorm.ErrRecordNotFound when no row
// was deleted.
func (r *InstanceRepository) Delete(id string) error {
	res := r.db.Delete(&models.AdapterInstance{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// RecordHeartbeat stamps the instance's last heartbeat and marks it running in
// a single UPDATE, so a heartbeat is cheap and cannot interleave with a
// concurrent status write from the runner.
func (r *InstanceRepository) RecordHeartbeat(id string, at time.Time) error {
	res := r.db.Model(&models.AdapterInstance{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_heartbeat": at,
			"status":         "running",
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// IncrementMessageCount atomically bumps the instance's inbound message
// counter. A missing instance is not an error on the message hot path.
func (r *InstanceRepository) IncrementMessageCount(id string) error {
	return r.db.Model(&models.AdapterInstance{}).Where("id = ?", id).
		UpdateColumn("message_count", gorm.Expr("message_count + 1")).Error
}

// IncrementErrorCount atomically bumps the instance's failure counter.
func (r *InstanceRepository) IncrementErrorCount(id string) error {
	return r.db.Model(&models.AdapterInstance{}).Where("id = ?", id).
		UpdateColumn("error_count", gorm.Expr("error_count + 1")).Error
}

// List lists all instances
func (r *InstanceRepository) List(limit, offset int) ([]models.AdapterInstance, int64, error) {
	var instances []models.AdapterInstance
	var total int64

	r.db.Model(&models.AdapterInstance{}).Count(&total)
	err := r.db.Limit(limit).Offset(offset).Find(&instances).Error
	return instances, total, err
}

// AdapterSessionRepository handles adapter session database operations
type AdapterSessionRepository struct {
	db *gorm.DB
}

// NewAdapterSessionRepository creates a new adapter session repository
func NewAdapterSessionRepository(db *gorm.DB) *AdapterSessionRepository {
	return &AdapterSessionRepository{db: db}
}

// Create creates a new session
func (r *AdapterSessionRepository) Create(session *models.AdapterSession) error {
	return r.db.Create(session).Error
}

// FindByInstanceID finds a session by instance ID
func (r *AdapterSessionRepository) FindByInstanceID(instanceID string) (*models.AdapterSession, error) {
	var session models.AdapterSession
	err := r.db.Where("instance_id = ?", instanceID).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// Update updates a session
func (r *AdapterSessionRepository) Update(session *models.AdapterSession) error {
	return r.db.Save(session).Error
}

// Delete deletes a session. It returns gorm.ErrRecordNotFound when no row was
// deleted.
func (r *AdapterSessionRepository) Delete(id string) error {
	res := r.db.Delete(&models.AdapterSession{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateHeartbeat updates the last heartbeat time
func (r *AdapterSessionRepository) UpdateHeartbeat(id string) error {
	return r.db.Model(&models.AdapterSession{}).Where("id = ?", id).
		Update("last_heartbeat", gorm.Expr("CURRENT_TIMESTAMP")).Error
}
