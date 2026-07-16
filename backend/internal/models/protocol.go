package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// GenerateID generates a unique ID
func GenerateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GenerateTraceID generates a trace ID for request tracking
func GenerateTraceID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "trace-" + hex.EncodeToString(bytes)
}

// GenerateIdempotencyKey generates an idempotency key
func GenerateIdempotencyKey(platformID, instanceID, messageID string) string {
	return platformID + "-" + instanceID + "-" + messageID
}

// Timestamp represents a timestamp in milliseconds
type Timestamp int64

// NewTimestamp creates a new timestamp from time.Time
func NewTimestamp(t time.Time) Timestamp {
	return Timestamp(t.UnixMilli())
}

// Time converts Timestamp to time.Time
func (t Timestamp) Time() time.Time {
	return time.UnixMilli(int64(t))
}

// MessageChain represents a message chain for V3_Pro protocol
type MessageChain struct {
	ID        string    `json:"id"`
	Timestamp Timestamp `json:"timestamp"`
	Platform  string    `json:"platform"`
	Instance  string    `json:"instance"`
	Content   string    `json:"content"`
	Type      string    `json:"type"` // text, image, audio, product_card
	Hash      string    `json:"hash"` // MD5 or Hash of message chain
}

// InboundMessage represents a standardized inbound message
type InboundMessage struct {
	PlatformID     string        `json:"platform_id"`
	InstanceID     string        `json:"instance_id"`
	ConversationID string        `json:"conversation_id"`
	SenderID       string        `json:"sender_id"`
	SenderName     string        `json:"sender_name"`
	MessageType    string        `json:"message_type"`
	MessageContent string        `json:"message_content"`
	RawMessage     interface{}   `json:"raw_message"`
	IdempotencyKey string        `json:"idempotency_key"`
	TraceID        string        `json:"trace_id"`
	Timestamp      Timestamp     `json:"timestamp"`
	MessageChain   *MessageChain `json:"message_chain,omitempty"`
}

// OutboundCommandProto represents a standardized outbound command protocol
type OutboundCommandProto struct {
	CommandType string      `json:"command_type"` // send_text, send_image, upload_media, create_conversation
	InstanceID  string      `json:"instance_id"`
	TargetID    string      `json:"target_id"` // conversation ID or user ID
	Payload     interface{} `json:"payload"`
	TraceID     string      `json:"trace_id"`
	Timestamp   Timestamp   `json:"timestamp"`
}

// AdapterManifest represents adapter package manifest
type AdapterManifest struct {
	ID              string                  `json:"id"`
	PlatformID      string                  `json:"platform_id"`
	Name            string                  `json:"name"`
	Version         string                  `json:"version"`
	RuntimeType     string                  `json:"runtime_type"`
	ProtocolVersion string                  `json:"protocol_version"`
	Capabilities    []string                `json:"capabilities"`
	ConfigSchema    map[string]interface{}  `json:"config_schema"`
	I18n            map[string]I18nResource `json:"i18n"`
	Operations      OperationPolicy         `json:"operations"`
	Security        SecurityPolicy          `json:"security"`
}

// I18nResource represents internationalization resources
type I18nResource struct {
	DisplayName   string            `json:"display_name"`
	Description   string            `json:"description"`
	InstallGuide  string            `json:"install_guide"`
	ErrorMessages map[string]string `json:"error_messages"`
}

// OperationPolicy represents operation policies
type OperationPolicy struct {
	HeartbeatInterval int `json:"heartbeat_interval"`
	ReconnectDelay    int `json:"reconnect_delay"`
	MaxRetries        int `json:"max_retries"`
	MaxQueueSize      int `json:"max_queue_size"`
}

// SecurityPolicy represents security policies
type SecurityPolicy struct {
	SensitiveFields  []string `json:"sensitive_fields"`
	EncryptedFields  []string `json:"encrypted_fields"`
	PermissionScopes []string `json:"permission_scopes"`
}
