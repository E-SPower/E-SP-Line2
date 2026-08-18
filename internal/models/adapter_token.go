package models

import "time"

// AdapterToken represents an access token for the adapter gateway.
// The adapter gateway is the "接入器" (Adapter) — a WebSocket server endpoint
// that external chat frameworks connect to as clients.
// Each token grants one framework access to consume e-commerce messages and
// send replies back.
type AdapterToken struct {
	ID          string     `json:"id" gorm:"primaryKey"`
	Name        string     `json:"name" gorm:"not null"`
	TokenHash   string     `json:"-" gorm:"not null;uniqueIndex"` // SHA-256 hash only, plaintext never stored
	Platform    string     `json:"platform"`                      // taobao / xianyu / "" (all)
	Scope       string     `json:"scope" gorm:"default:'read+write'"` // read / write / read+write
	Status      string     `json:"status" gorm:"default:'active'"`    // active, revoked
	ExpiresAt   *time.Time `json:"expires_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CreatedBy   string     `json:"created_by"`
}

// TableName returns the table name for the AdapterToken model.
func (AdapterToken) TableName() string {
	return "adapter_tokens"
}
