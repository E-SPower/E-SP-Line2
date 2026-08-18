package repository

import (
	"time"

	"github.com/e-spl/e-sp-line2/internal/models"
	"gorm.io/gorm"
)

// AdapterGatewayRepository handles adapter (接入器) entity database operations.
type AdapterGatewayRepository struct {
	db *gorm.DB
}

// NewAdapterGatewayRepository creates a new adapter gateway repository.
func NewAdapterGatewayRepository(db *gorm.DB) *AdapterGatewayRepository {
	return &AdapterGatewayRepository{db: db}
}

// Create creates a new adapter.
func (r *AdapterGatewayRepository) Create(adapter *models.Adapter) error {
	return r.db.Create(adapter).Error
}

// FindByID finds an adapter by ID.
func (r *AdapterGatewayRepository) FindByID(id string) (*models.Adapter, error) {
	var adapter models.Adapter
	err := r.db.First(&adapter, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &adapter, nil
}

// FindByKey finds an adapter by its access key.
func (r *AdapterGatewayRepository) FindByKey(key string) (*models.Adapter, error) {
	var adapter models.Adapter
	err := r.db.Where("key = ?", key).First(&adapter).Error
	if err != nil {
		return nil, err
	}
	return &adapter, nil
}

// Update updates an adapter.
func (r *AdapterGatewayRepository) Update(adapter *models.Adapter) error {
	return r.db.Save(adapter).Error
}

// UpdateStatus updates an adapter's status.
func (r *AdapterGatewayRepository) UpdateStatus(id, status string) error {
	return r.db.Model(&models.Adapter{}).Where("id = ?", id).
		Update("status", status).Error
}

// UpdateEnabled updates an adapter's enabled flag.
func (r *AdapterGatewayRepository) UpdateEnabled(id string, enabled bool) error {
	return r.db.Model(&models.Adapter{}).Where("id = ?", id).
		Update("enabled", enabled).Error
}

// UpdateLastConnected updates the last connected timestamp of an adapter.
func (r *AdapterGatewayRepository) UpdateLastConnected(id string, t interface{}) error {
	return r.db.Model(&models.Adapter{}).Where("id = ?", id).
		Update("last_connected_at", t).Error
}

// Delete deletes an adapter.
func (r *AdapterGatewayRepository) Delete(id string) error {
	return r.db.Delete(&models.Adapter{}, "id = ?", id).Error
}

// List lists all adapters.
func (r *AdapterGatewayRepository) List(limit, offset int) ([]models.Adapter, int64, error) {
	var adapters []models.Adapter
	var total int64

	r.db.Model(&models.Adapter{}).Count(&total)
	err := r.db.Limit(limit).Offset(offset).Order("created_at DESC").Find(&adapters).Error
	return adapters, total, err
}

// ListEnabled lists all enabled adapters.
func (r *AdapterGatewayRepository) ListEnabled() ([]models.Adapter, error) {
	var adapters []models.Adapter
	err := r.db.Where("enabled = ?", true).Order("created_at DESC").Find(&adapters).Error
	return adapters, err
}

// AdapterConnectionRepository handles adapter connection database operations.
type AdapterConnectionRepository struct {
	db *gorm.DB
}

// NewAdapterConnectionRepository creates a new adapter connection repository.
func NewAdapterConnectionRepository(db *gorm.DB) *AdapterConnectionRepository {
	return &AdapterConnectionRepository{db: db}
}

// Create creates a new connection record.
func (r *AdapterConnectionRepository) Create(conn *models.AdapterConnection) error {
	return r.db.Create(conn).Error
}

// FindByID finds a connection by ID.
func (r *AdapterConnectionRepository) FindByID(id string) (*models.AdapterConnection, error) {
	var conn models.AdapterConnection
	err := r.db.First(&conn, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &conn, nil
}

// Update updates a connection record.
func (r *AdapterConnectionRepository) Update(conn *models.AdapterConnection) error {
	return r.db.Save(conn).Error
}

// TouchHeartbeat updates only the heartbeat columns of a connection. This is
// a single UPDATE (no read-then-write) so it stays cheap on the hot path
// (every received frame / pong touches the connection).
func (r *AdapterConnectionRepository) TouchHeartbeat(id string) error {
	return r.db.Model(&models.AdapterConnection{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_heartbeat": time.Now(),
			"updated_at":     time.Now(),
		}).Error
}

// MarkDisconnected marks a connection as disconnected.
func (r *AdapterConnectionRepository) MarkDisconnected(id string, t interface{}) error {
	return r.db.Model(&models.AdapterConnection{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":          "disconnected",
			"disconnected_at": t,
		}).Error
}

// IncrementMessageCount increments the message count of a connection.
func (r *AdapterConnectionRepository) IncrementMessageCount(id string) error {
	return r.db.Model(&models.AdapterConnection{}).Where("id = ?", id).
		UpdateColumn("message_count", gorm.Expr("message_count + 1")).Error
}

// List lists all connection records.
func (r *AdapterConnectionRepository) List(limit, offset int) ([]models.AdapterConnection, int64, error) {
	var conns []models.AdapterConnection
	var total int64

	r.db.Model(&models.AdapterConnection{}).Count(&total)
	err := r.db.Limit(limit).Offset(offset).Order("created_at DESC").Find(&conns).Error
	return conns, total, err
}

// ListByAdapterID lists connection records for an adapter.
func (r *AdapterConnectionRepository) ListByAdapterID(adapterID string, limit, offset int) ([]models.AdapterConnection, error) {
	var conns []models.AdapterConnection
	err := r.db.Where("adapter_id = ?", adapterID).
		Limit(limit).Offset(offset).Order("created_at DESC").Find(&conns).Error
	return conns, err
}

// ListConnected lists all currently connected connections.
func (r *AdapterConnectionRepository) ListConnected() ([]models.AdapterConnection, error) {
	var conns []models.AdapterConnection
	err := r.db.Where("status = ?", "connected").Order("created_at DESC").Find(&conns).Error
	return conns, err
}
