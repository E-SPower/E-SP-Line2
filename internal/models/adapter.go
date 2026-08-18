package models

import "time"

// Adapter represents a configurable WebSocket adapter (接入器) entity.
//
// An adapter is a WebSocket connection endpoint that external systems use to
// exchange e-commerce messages with E-SP-Line2. It supports two modes:
//
//   - "server": E-SP-Line2 listens on a WebSocket endpoint; external systems
//     connect to it as clients, authenticating with the adapter's key.
//   - "client": E-SP-Line2 actively connects to an external WebSocket URL,
//     presenting the adapter's key for authentication.
//
// The key is user-defined and editable. The WS address is configurable.
type Adapter struct {
	ID         string     `json:"id" gorm:"primaryKey"`
	Name       string     `json:"name" gorm:"not null"`
	Mode       string     `json:"mode" gorm:"default:'server'"` // server / client
	ListenPath string     `json:"listen_path"`                  // server mode: listen path (default /ws/adapter-gateway/:id)
	WSURL      string     `json:"ws_url"`                       // client mode: target WebSocket URL to connect to
	Key        string     `json:"key"`                          // user-defined, editable access key
	Platform   string     `json:"platform"`                     // taobao / xianyu / "" (all)
	Scope      string     `json:"scope" gorm:"default:'read+write'"` // read / write / read+write
	Status     string     `json:"status" gorm:"default:'active'"`    // active / disabled
	Enabled    bool       `json:"enabled" gorm:"default:true"`
	LastConnectedAt *time.Time `json:"last_connected_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	CreatedBy  string     `json:"created_by"`
}

// TableName returns the table name for the Adapter model.
func (Adapter) TableName() string {
	return "adapters"
}

// AdapterConnection represents a live (or historical) WebSocket connection
// made through an adapter.
type AdapterConnection struct {
	ID             string     `json:"id" gorm:"primaryKey"`
	AdapterID      string     `json:"adapter_id" gorm:"index;not null"`
	AdapterName    string     `json:"adapter_name"`
	Mode           string     `json:"mode"` // server / client
	Platform       string     `json:"platform"`
	RemoteAddr     string     `json:"remote_addr"`
	Status         string     `json:"status" gorm:"default:'connected'"` // connected, disconnected
	ConnectedAt    time.Time  `json:"connected_at"`
	DisconnectedAt *time.Time `json:"disconnected_at"`
	LastHeartbeat  time.Time  `json:"last_heartbeat"`
	MessageCount   int64      `json:"message_count" gorm:"default:0"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// TableName returns the table name for the AdapterConnection model.
func (AdapterConnection) TableName() string {
	return "adapter_connections"
}
