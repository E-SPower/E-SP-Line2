package repository

import (
	"github.com/e-spl/e-sp-line2/internal/models"
	"gorm.io/gorm"
)

// AdapterRepository handles adapter database operations
type AdapterRepository struct {
	db *gorm.DB
}

// NewAdapterRepository creates a new adapter repository
func NewAdapterRepository(db *gorm.DB) *AdapterRepository {
	return &AdapterRepository{db: db}
}

// Create creates a new adapter
func (r *AdapterRepository) Create(adapter *models.AdapterPackage) error {
	return r.db.Create(adapter).Error
}

// FindByID finds an adapter by ID
func (r *AdapterRepository) FindByID(id string) (*models.AdapterPackage, error) {
	var adapter models.AdapterPackage
	err := r.db.First(&adapter, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &adapter, nil
}

// FindByPlatformID finds adapters by platform ID
func (r *AdapterRepository) FindByPlatformID(platformID string) ([]models.AdapterPackage, error) {
	var adapters []models.AdapterPackage
	err := r.db.Where("platform_id = ?", platformID).Find(&adapters).Error
	return adapters, err
}

// Update updates an adapter
func (r *AdapterRepository) Update(adapter *models.AdapterPackage) error {
	return r.db.Save(adapter).Error
}

// Delete deletes an adapter
func (r *AdapterRepository) Delete(id string) error {
	return r.db.Delete(&models.AdapterPackage{}, "id = ?", id).Error
}

// List lists all adapters
func (r *AdapterRepository) List(limit, offset int) ([]models.AdapterPackage, int64, error) {
	var adapters []models.AdapterPackage
	var total int64

	r.db.Model(&models.AdapterPackage{}).Count(&total)
	err := r.db.Limit(limit).Offset(offset).Find(&adapters).Error
	return adapters, total, err
}

// AdapterPackageRepository handles adapter package database operations
type AdapterPackageRepository struct {
	db *gorm.DB
}

// NewAdapterPackageRepository creates a new adapter package repository
func NewAdapterPackageRepository(db *gorm.DB) *AdapterPackageRepository {
	return &AdapterPackageRepository{db: db}
}

// Create creates a new adapter package
func (r *AdapterPackageRepository) Create(pkg *models.AdapterPackage) error {
	return r.db.Create(pkg).Error
}

// FindByID finds an adapter package by ID
func (r *AdapterPackageRepository) FindByID(id string) (*models.AdapterPackage, error) {
	var pkg models.AdapterPackage
	err := r.db.First(&pkg, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &pkg, nil
}

// Update updates an adapter package
func (r *AdapterPackageRepository) Update(pkg *models.AdapterPackage) error {
	return r.db.Save(pkg).Error
}

// Delete deletes an adapter package
func (r *AdapterPackageRepository) Delete(id string) error {
	return r.db.Delete(&models.AdapterPackage{}, "id = ?", id).Error
}
