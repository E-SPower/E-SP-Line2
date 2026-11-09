package models

import (
	"time"
)

// Platform represents a business platform (e.g., Xianyu, Taobao)
type Platform struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"not null"`
	Code        string    `json:"code" gorm:"uniqueIndex;not null"`
	Description string    `json:"description"`
	Status      string    `json:"status" gorm:"default:'active'"` // active, inactive
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AdapterPackage represents an adapter package with version and capabilities
type AdapterPackage struct {
	ID              string    `json:"id" gorm:"primaryKey"`
	PlatformID      string    `json:"platform_id" gorm:"index;not null"`
	Name            string    `json:"name" gorm:"not null"`
	Version         string    `json:"version" gorm:"not null"`
	RuntimeType     string    `json:"runtime_type"` // python, node, go
	ProtocolVersion string    `json:"protocol_version"`
	Status          string    `json:"status" gorm:"default:'active'"` // active, deprecated
	Manifest        string    `json:"manifest" gorm:"type:text"`      // JSON manifest
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// AdapterCapability represents adapter capabilities
type AdapterCapability struct {
	ID         string `json:"id" gorm:"primaryKey"`
	AdapterID  string `json:"adapter_id" gorm:"index;not null"`
	Capability string `json:"capability" gorm:"not null"` // receive_message, send_text, send_image, etc.
	Enabled    bool   `json:"enabled" gorm:"default:true"`
}

// AdapterInstance represents a specific account/shop instance.
//
// LastHeartbeat / MessageCount / ErrorCount are the instance-level counters the
// Monitoring page reads. They are maintained on the instance row (not derived
// from adapter_sessions) so they survive a bridge reconnect and stay available
// while the adapter is stopped.
type AdapterInstance struct {
	ID            string     `json:"id" gorm:"primaryKey"`
	AdapterID     string     `json:"adapter_id" gorm:"index;not null"`
	PlatformID    string     `json:"platform_id" gorm:"index;not null"`
	Name          string     `json:"name" gorm:"not null"`
	Config        string     `json:"config" gorm:"type:text"`         // JSON config
	Status        string     `json:"status" gorm:"default:'stopped'"` // stopped, running, error
	UserID        string     `json:"user_id" gorm:"index"`
	LastHeartbeat *time.Time `json:"last_heartbeat"`                 // last bridge/process heartbeat
	MessageCount  int64      `json:"message_count" gorm:"default:0"` // inbound messages received
	ErrorCount    int64      `json:"error_count" gorm:"default:0"`   // recorded adapter errors
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// AdapterSession represents current running session
type AdapterSession struct {
	ID            string    `json:"id" gorm:"primaryKey"`
	InstanceID    string    `json:"instance_id" gorm:"uniqueIndex;not null"`
	WorkerID      string    `json:"worker_id"`
	LeaseExpiry   time.Time `json:"lease_expiry"`
	ConnectedAt   time.Time `json:"connected_at"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	Status        string    `json:"status"` // connected, disconnected
}

// InboundEvent represents an inbound message event
type InboundEvent struct {
	ID             string `json:"id" gorm:"primaryKey"`
	PlatformID     string `json:"platform_id" gorm:"index;not null"`
	InstanceID     string `json:"instance_id" gorm:"index;not null"`
	ConversationID string `json:"conversation_id" gorm:"index"`
	SenderID       string `json:"sender_id"`
	SenderName     string `json:"sender_name"`
	MessageType    string `json:"message_type"` // text, image, audio, product_card
	MessageContent string `json:"message_content" gorm:"type:text"`
	RawMessage     string `json:"raw_message" gorm:"type:text"`
	// IdempotencyKey is a pointer so "no key supplied" can be stored as NULL
	// instead of "". SQLite's UNIQUE treats NULLs as distinct but allows only a
	// single "", so with a plain string the second keyless message collided with
	// the first (UNIQUE constraint failed) the moment an ingest route existed.
	// MessageService.Create still generates a fallback key for every writer, so
	// the unique index below stays meaningful rather than becoming dead weight.
	IdempotencyKey *string   `json:"idempotency_key" gorm:"uniqueIndex"`
	TraceID        string    `json:"trace_id" gorm:"index"`
	Status         string    `json:"status" gorm:"default:'received'"` // received, routed, delivered, acked, retry_pending, dead_letter
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// OutboundCommand represents an outbound command
type OutboundCommand struct {
	ID          string     `json:"id" gorm:"primaryKey"`
	InstanceID  string     `json:"instance_id" gorm:"index;not null"`
	CommandType string     `json:"command_type"` // send_text, send_image, upload_media, create_conversation
	Payload     string     `json:"payload" gorm:"type:text"`
	Status      string     `json:"status" gorm:"default:'created'"` // created, queued, sending, sent, failed, retrying, expired
	RetryCount  int        `json:"retry_count" gorm:"default:0"`
	MaxRetries  int        `json:"max_retries" gorm:"default:3"`
	TraceID     string     `json:"trace_id" gorm:"index"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	SentAt      *time.Time `json:"sent_at"`
}

// RouteRule represents routing rules
type RouteRule struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	Name       string    `json:"name" gorm:"not null"`
	PlatformID string    `json:"platform_id" gorm:"index"`
	InstanceID string    `json:"instance_id" gorm:"index"`
	Priority   int       `json:"priority" gorm:"default:0"`
	Conditions string    `json:"conditions" gorm:"type:text"` // JSON conditions
	TargetType string    `json:"target_type"`                 // app, webhook
	TargetID   string    `json:"target_id"`
	Enabled    bool      `json:"enabled" gorm:"default:true"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// User represents a system user
type User struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	Username     string    `json:"username" gorm:"uniqueIndex;not null"`
	PasswordHash string    `json:"-" gorm:"not null"`
	Email        string    `json:"email"`
	Role         string    `json:"role" gorm:"default:'user'"`     // admin, user
	Status       string    `json:"status" gorm:"default:'active'"` // active, disabled
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AuditLog represents operation audit logs
type AuditLog struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	UserID     string    `json:"user_id" gorm:"index"`
	Action     string    `json:"action" gorm:"not null"`
	Resource   string    `json:"resource" gorm:"not null"`
	ResourceID string    `json:"resource_id"`
	Details    string    `json:"details" gorm:"type:text"`
	IPAddress  string    `json:"ip_address"`
	CreatedAt  time.Time `json:"created_at"`
}

// SystemSetting represents a key/value system setting (e.g. registration toggle).
type SystemSetting struct {
	Key       string    `json:"key" gorm:"primaryKey"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}
