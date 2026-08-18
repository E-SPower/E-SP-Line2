package repository

import (
	"time"

	"github.com/e-spl/e-sp-line2/internal/models"
	"gorm.io/gorm"
)

// SystemSettingRepository handles key/value system settings.
type SystemSettingRepository struct {
	db *gorm.DB
}

// NewSystemSettingRepository creates a new system setting repository.
func NewSystemSettingRepository(db *gorm.DB) *SystemSettingRepository {
	return &SystemSettingRepository{db: db}
}

// Get returns the value for a key, or "" if not set.
func (r *SystemSettingRepository) Get(key string) (string, error) {
	var s models.SystemSetting
	err := r.db.First(&s, "key = ?", key).Error
	if err == gorm.ErrRecordNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return s.Value, nil
}

// Set upserts a key/value pair.
func (r *SystemSettingRepository) Set(key, value string) error {
	var s models.SystemSetting
	err := r.db.First(&s, "key = ?", key).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(&models.SystemSetting{
			Key:       key,
			Value:     value,
			UpdatedAt: time.Now(),
		}).Error
	}
	if err != nil {
		return err
	}
	s.Value = value
	s.UpdatedAt = time.Now()
	return r.db.Save(&s).Error
}
