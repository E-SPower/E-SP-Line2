package repository

import (
	"github.com/e-spl/e-sp-line2/internal/models"
	"gorm.io/gorm"
)

// PlatformRepository handles platform database operations
type PlatformRepository struct {
	db *gorm.DB
}

// NewPlatformRepository creates a new platform repository
func NewPlatformRepository(db *gorm.DB) *PlatformRepository {
	return &PlatformRepository{db: db}
}

// Create creates a new platform
func (r *PlatformRepository) Create(platform *models.Platform) error {
	return r.db.Create(platform).Error
}

// FindByID finds a platform by ID
func (r *PlatformRepository) FindByID(id string) (*models.Platform, error) {
	var platform models.Platform
	err := r.db.First(&platform, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &platform, nil
}

// FindByCode finds a platform by code
func (r *PlatformRepository) FindByCode(code string) (*models.Platform, error) {
	var platform models.Platform
	err := r.db.Where("code = ?", code).First(&platform).Error
	if err != nil {
		return nil, err
	}
	return &platform, nil
}

// Update updates a platform
func (r *PlatformRepository) Update(platform *models.Platform) error {
	return r.db.Save(platform).Error
}

// Delete deletes a platform
func (r *PlatformRepository) Delete(id string) error {
	return r.db.Delete(&models.Platform{}, "id = ?", id).Error
}

// List lists all platforms
func (r *PlatformRepository) List(limit, offset int) ([]models.Platform, int64, error) {
	var platforms []models.Platform
	var total int64

	r.db.Model(&models.Platform{}).Count(&total)
	err := r.db.Limit(limit).Offset(offset).Find(&platforms).Error
	return platforms, total, err
}

// ListActive lists all active platforms
func (r *PlatformRepository) ListActive() ([]models.Platform, error) {
	var platforms []models.Platform
	err := r.db.Where("status = ?", "active").Find(&platforms).Error
	return platforms, err
}
