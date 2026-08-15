// Package v3 implements the V3 protocol for E-SP-Line2
//
// The V3 protocol provides a standardized message format for platform adapters,
// including message envelopes, message chains, and e-commerce specific extensions.
//
// Key features:
// - Message envelope with signature verification
// - Message chain with multiple element types
// - E-commerce extensions (product cards, order info, inquiries)
// - HMAC signing for API requests
// - Idempotency keys for message deduplication
package v3

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Protocol errors
var (
	ErrInvalidSignature    = errors.New("invalid signature")
	ErrExpiredRequest      = errors.New("request expired")
	ErrInvalidAPIKey       = errors.New("invalid API key")
	ErrMissingSignature    = errors.New("missing signature")
	ErrInvalidEnvelope     = errors.New("invalid message envelope")
	ErrInvalidMessageChain = errors.New("invalid message chain")
	ErrUnsupportedType     = errors.New("unsupported message type")
)

// Protocol constants
const (
	// Protocol version
	Version = "v3"

	// Maximum message size (1MB)
	MaxMessageSize = 1024 * 1024

	// Maximum elements in message chain
	MaxElements = 100

	// Signature expiration (5 minutes)
	SignatureExpiration = 300

	// Default heartbeat interval (30 seconds)
	DefaultHeartbeatInterval = 30

	// Default reconnect delay (5 seconds)
	DefaultReconnectDelay = 5

	// Default max retries
	DefaultMaxRetries = 3

	// Default max queue size
	DefaultMaxQueueSize = 1000
)

// Validate validates the message envelope
func (e *MessageEnvelope) Validate() error {
	if e.ProtocolVersion != Version {
		return fmt.Errorf("%w: expected %s, got %s", ErrInvalidEnvelope, Version, e.ProtocolVersion)
	}

	if e.EventID == "" {
		return fmt.Errorf("%w: missing event_id", ErrInvalidEnvelope)
	}

	if e.Platform == "" {
		return fmt.Errorf("%w: missing platform", ErrInvalidEnvelope)
	}

	if e.EventType == "" {
		return fmt.Errorf("%w: missing event_type", ErrInvalidEnvelope)
	}

	if e.Timestamp <= 0 {
		return fmt.Errorf("%w: invalid timestamp", ErrInvalidEnvelope)
	}

	return nil
}

// Validate validates the message chain
func (mc *MessageChain) Validate() error {
	if mc.ID == "" {
		return fmt.Errorf("%w: missing id", ErrInvalidMessageChain)
	}

	if mc.Platform == "" {
		return fmt.Errorf("%w: missing platform", ErrInvalidMessageChain)
	}

	if len(mc.Content) == 0 {
		return fmt.Errorf("%w: empty content", ErrInvalidMessageChain)
	}

	if len(mc.Content) > MaxElements {
		return fmt.Errorf("%w: too many elements (max %d)", ErrInvalidMessageChain, MaxElements)
	}

	for i, elem := range mc.Content {
		if elem.Type == "" {
			return fmt.Errorf("%w: element %d missing type", ErrInvalidMessageChain, i)
		}
	}

	return nil
}

// ParseMessageElement parses a message element content based on type
func ParseMessageElement(elemType ElementType, content json.RawMessage) (interface{}, error) {
	switch elemType {
	case ElementTypeText:
		var c TextContent
		if err := json.Unmarshal(content, &c); err != nil {
			return nil, err
		}
		return c, nil

	case ElementTypeImage:
		var c ImageContent
		if err := json.Unmarshal(content, &c); err != nil {
			return nil, err
		}
		return c, nil

	case ElementTypeAudio:
		var c AudioContent
		if err := json.Unmarshal(content, &c); err != nil {
			return nil, err
		}
		return c, nil

	case ElementTypeVideo:
		var c VideoContent
		if err := json.Unmarshal(content, &c); err != nil {
			return nil, err
		}
		return c, nil

	case ElementTypeFile:
		var c FileContent
		if err := json.Unmarshal(content, &c); err != nil {
			return nil, err
		}
		return c, nil

	case ElementTypeProductCard:
		var c ProductCard
		if err := json.Unmarshal(content, &c); err != nil {
			return nil, err
		}
		return c, nil

	case ElementTypeOrderInfo:
		var c OrderInfo
		if err := json.Unmarshal(content, &c); err != nil {
			return nil, err
		}
		return c, nil

	case ElementTypeInquiry:
		var c Inquiry
		if err := json.Unmarshal(content, &c); err != nil {
			return nil, err
		}
		return c, nil

	case ElementTypeLocation:
		var c LocationContent
		if err := json.Unmarshal(content, &c); err != nil {
			return nil, err
		}
		return c, nil

	case ElementTypeEmoji:
		var c EmojiContent
		if err := json.Unmarshal(content, &c); err != nil {
			return nil, err
		}
		return c, nil

	case ElementTypeMention:
		var c MentionContent
		if err := json.Unmarshal(content, &c); err != nil {
			return nil, err
		}
		return c, nil

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedType, elemType)
	}
}

// MessageElementToJSON converts a message element to JSON
func MessageElementToJSON(elem MessageElement) (json.RawMessage, error) {
	return json.Marshal(elem.Content)
}

// IsMessageEvent checks if the event type is a message event
func IsMessageEvent(eventType EventType) bool {
	switch eventType {
	case EventMessageReceived, EventMessageSent, EventMessageDelivered,
		EventMessageAcked, EventMessageFailed:
		return true
	default:
		return false
	}
}

// IsAdapterEvent checks if the event type is an adapter event
func IsAdapterEvent(eventType EventType) bool {
	switch eventType {
	case EventAdapterConnected, EventAdapterDisconnected,
		EventAdapterStarted, EventAdapterStopped, EventAdapterError:
		return true
	default:
		return false
	}
}

// IsCommandEvent checks if the event type is a command event
func IsCommandEvent(eventType EventType) bool {
	switch eventType {
	case EventCommandCreated, EventCommandQueued, EventCommandSending,
		EventCommandSent, EventCommandFailed, EventCommandRetrying, EventCommandExpired:
		return true
	default:
		return false
	}
}

// IsSystemEvent checks if the event type is a system event
func IsSystemEvent(eventType EventType) bool {
	switch eventType {
	case EventSystemError, EventSystemHealthCheck:
		return true
	default:
		return false
	}
}
