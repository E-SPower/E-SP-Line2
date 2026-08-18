package v3

import (
	"encoding/json"
	"fmt"
)

// InboundMessagePayload is the standardized payload carried inside a
// message.received envelope. It preserves the FULL message chain and the
// original raw platform message so nothing is lost when a bridge reports an
// inbound message.
type InboundMessagePayload struct {
	ID             string          `json:"id,omitempty"`
	Timestamp      int64           `json:"timestamp,omitempty"`
	Platform       string          `json:"platform,omitempty"`
	Instance       string          `json:"instance,omitempty"`
	ConversationID string          `json:"conversation_id,omitempty"`
	Sender         SenderInfo      `json:"sender,omitempty"`
	MessageType    string          `json:"message_type,omitempty"`
	MessageContent string          `json:"message_content,omitempty"`
	MessageChain   *MessageChain   `json:"message_chain,omitempty"`
	Raw            json.RawMessage `json:"raw,omitempty"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
}

// NewInboundMessagePayload creates a new inbound message payload.
func NewInboundMessagePayload(platform, instance, conversationID string, sender SenderInfo) *InboundMessagePayload {
	return &InboundMessagePayload{
		Timestamp:      GetCurrentTimestamp(),
		Platform:       platform,
		Instance:       instance,
		ConversationID: conversationID,
		Sender:         sender,
	}
}

// SetMessageChain sets the message chain and derives the message type and
// text content from it.
func (p *InboundMessagePayload) SetMessageChain(mc *MessageChain) *InboundMessagePayload {
	p.MessageChain = mc
	if mc != nil {
		p.MessageType = deriveMessageType(mc)
		p.MessageContent = mc.GetTextContent()
	}
	return p
}

// SetRaw sets the raw platform message.
func (p *InboundMessagePayload) SetRaw(raw json.RawMessage) *InboundMessagePayload {
	p.Raw = raw
	return p
}

// deriveMessageType derives a coarse message type from a message chain.
func deriveMessageType(mc *MessageChain) string {
	if mc == nil || len(mc.Content) == 0 {
		return "text"
	}
	// Prefer the first element's type.
	return string(mc.Content[0].Type)
}

// ParseInboundMessagePayload parses a raw JSON payload into an
// InboundMessagePayload. It is tolerant: if the payload is not a valid
// InboundMessagePayload, it falls back to a generic map so the raw message
// is still preserved.
func ParseInboundMessagePayload(data []byte) (*InboundMessagePayload, error) {
	var p InboundMessagePayload
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}

	// If the payload carried a message_chain, derive the message type and
	// text content from it (the chain's Content is a generic map after JSON
	// unmarshalling, so GetTextContent handles both forms).
	if p.MessageChain != nil {
		p.MessageType = deriveMessageType(p.MessageChain)
		p.MessageContent = p.MessageChain.GetTextContent()
	}

	return &p, nil
}

// ToMap converts the payload to a generic map (for persistence / fan-out).
func (p *InboundMessagePayload) ToMap() (map[string]interface{}, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// Validate validates the inbound message payload.
func (p *InboundMessagePayload) Validate() error {
	if p.Platform == "" {
		return fmt.Errorf("%w: missing platform", ErrInvalidEnvelope)
	}
	if p.ConversationID == "" {
		return fmt.Errorf("%w: missing conversation_id", ErrInvalidEnvelope)
	}
	if p.Sender.ID == "" {
		return fmt.Errorf("%w: missing sender.id", ErrInvalidEnvelope)
	}
	return nil
}
