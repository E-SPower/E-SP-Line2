package v3

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"
)

// ProtocolVersion represents the V3 protocol version
const ProtocolVersion = "v3"

// MessageEnvelope represents the V3 protocol message envelope
type MessageEnvelope struct {
	ProtocolVersion string      `json:"protocol_version"` // "v3"
	EventID         string      `json:"event_id"`         // UUID
	TraceID         string      `json:"trace_id"`         // Trace ID for request tracking
	Timestamp       int64       `json:"timestamp"`        // Unix timestamp in milliseconds
	Platform        string      `json:"platform"`         // Platform identifier
	AdapterID       string      `json:"adapter_id"`       // Adapter instance ID
	EventType       EventType   `json:"event_type"`       // Event type
	Payload         interface{} `json:"payload"`          // Message content
	Signature       string      `json:"signature"`        // HMAC signature
}

// EventType represents the type of event
type EventType string

const (
	// Message events
	EventMessageReceived  EventType = "message.received"
	EventMessageSent      EventType = "message.sent"
	EventMessageDelivered EventType = "message.delivered"
	EventMessageAcked     EventType = "message.acked"
	EventMessageFailed    EventType = "message.failed"

	// Adapter events
	EventAdapterConnected    EventType = "adapter.connected"
	EventAdapterDisconnected EventType = "adapter.disconnected"
	EventAdapterStarted      EventType = "adapter.started"
	EventAdapterStopped      EventType = "adapter.stopped"
	EventAdapterError        EventType = "adapter.error"

	// Command events
	EventCommandCreated  EventType = "command.created"
	EventCommandQueued   EventType = "command.queued"
	EventCommandSending  EventType = "command.sending"
	EventCommandSent     EventType = "command.sent"
	EventCommandFailed   EventType = "command.failed"
	EventCommandRetrying EventType = "command.retrying"
	EventCommandExpired  EventType = "command.expired"

	// System events
	EventSystemError       EventType = "system.error"
	EventSystemHealthCheck EventType = "system.health_check"
)

// NewMessageEnvelope creates a new message envelope
func NewMessageEnvelope(platform, adapterID string, eventType EventType, payload interface{}) *MessageEnvelope {
	return &MessageEnvelope{
		ProtocolVersion: ProtocolVersion,
		EventID:         GenerateEventID(),
		TraceID:         GenerateTraceID(),
		Timestamp:       time.Now().UnixMilli(),
		Platform:        platform,
		AdapterID:       adapterID,
		EventType:       eventType,
		Payload:         payload,
	}
}

// GenerateEventID generates a unique event ID using crypto/rand
func GenerateEventID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GenerateTraceID generates a trace ID for request tracking
func GenerateTraceID() string {
	bytes := make([]byte, 8)
	_, _ = rand.Read(bytes)
	return "trace-" + hex.EncodeToString(bytes)
}

// Sign signs the envelope with the given secret using MD5
func (e *MessageEnvelope) Sign(secret string) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}

	hash := md5.Sum(append(data, []byte(secret)...))
	e.Signature = hex.EncodeToString(hash[:])
	return nil
}

// Verify verifies the envelope signature
func (e *MessageEnvelope) Verify(secret string) bool {
	if e.Signature == "" {
		return false
	}

	// Create a copy without signature
	envelopeCopy := *e
	envelopeCopy.Signature = ""

	data, err := json.Marshal(envelopeCopy)
	if err != nil {
		return false
	}

	hash := md5.Sum(append(data, []byte(secret)...))
	expectedSignature := hex.EncodeToString(hash[:])

	return e.Signature == expectedSignature
}

// ToJSON converts the envelope to JSON
func (e *MessageEnvelope) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// FromJSON parses JSON into envelope
func FromJSON(data []byte) (*MessageEnvelope, error) {
	var envelope MessageEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	return &envelope, nil
}
