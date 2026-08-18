package repository

import (
	"github.com/e-spl/e-sp-line2/internal/models"
	"gorm.io/gorm"
)

// AdapterTokenRepository handles adapter gateway token database operations.
type AdapterTokenRepository struct {
	db *gorm.DB
}

// NewAdapterTokenRepository creates a new adapter token repository.
func NewAdapterTokenRepository(db *gorm.DB) *AdapterTokenRepository {
	return &AdapterTokenRepository{db: db}
}

// Create creates a new token.
func (r *AdapterTokenRepository) Create(token *models.AdapterToken) error {
	return r.db.Create(token).Error
}

// FindByID finds a token by ID.
func (r *AdapterTokenRepository) FindByID(id string) (*models.AdapterToken, error) {
	var token models.AdapterToken
	err := r.db.First(&token, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// FindByTokenHash finds a token by its SHA-256 hash.
func (r *AdapterTokenRepository) FindByTokenHash(hash string) (*models.AdapterToken, error) {
	var token models.AdapterToken
	err := r.db.Where("token_hash = ?", hash).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// Update updates a token.
func (r *AdapterTokenRepository) Update(token *models.AdapterToken) error {
	return r.db.Save(token).Error
}

// UpdateStatus updates a token's status.
func (r *AdapterTokenRepository) UpdateStatus(id, status string) error {
	return r.db.Model(&models.AdapterToken{}).Where("id = ?", id).
		Update("status", status).Error
}

// UpdateLastUsed updates the last used timestamp of a token.
func (r *AdapterTokenRepository) UpdateLastUsed(id string, t interface{}) error {
	return r.db.Model(&models.AdapterToken{}).Where("id = ?", id).
		Update("last_used_at", t).Error
}

// Delete deletes a token.
func (r *AdapterTokenRepository) Delete(id string) error {
	return r.db.Delete(&models.AdapterToken{}, "id = ?", id).Error
}

// List lists all tokens.
func (r *AdapterTokenRepository) List(limit, offset int) ([]models.AdapterToken, int64, error) {
	var tokens []models.AdapterToken
	var total int64

	r.db.Model(&models.AdapterToken{}).Count(&total)
	err := r.db.Limit(limit).Offset(offset).Order("created_at DESC").Find(&tokens).Error
	return tokens, total, err
}

// ListActive lists all active tokens.
func (r *AdapterTokenRepository) ListActive() ([]models.AdapterToken, error) {
	var tokens []models.AdapterToken
	err := r.db.Where("status = ?", "active").Order("created_at DESC").Find(&tokens).Error
	return tokens, err
}
