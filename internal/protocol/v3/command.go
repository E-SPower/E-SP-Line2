package v3

// OutboundCommandProto represents a standardized outbound command protocol
type OutboundCommandProto struct {
	CommandType string      `json:"command_type"` // send_text, send_image, upload_media, create_conversation
	InstanceID  string      `json:"instance_id"`
	TargetID    string      `json:"target_id"` // conversation ID or user ID
	Payload     interface{} `json:"payload"`
	TraceID     string      `json:"trace_id"`
	Timestamp   int64       `json:"timestamp"`
}

// CommandType constants
const (
	CommandTypeSendText           = "send_text"
	CommandTypeSendImage          = "send_image"
	CommandTypeUploadMedia        = "upload_media"
	CommandTypeCreateConversation = "create_conversation"
	CommandTypeGetHistory         = "get_history"
	CommandTypeRefreshToken       = "refresh_token"
)

// NewOutboundCommand creates a new outbound command
func NewOutboundCommand(commandType, instanceID, targetID string, payload interface{}) *OutboundCommandProto {
	return &OutboundCommandProto{
		CommandType: commandType,
		InstanceID:  instanceID,
		TargetID:    targetID,
		Payload:     payload,
		TraceID:     GenerateTraceID(),
		Timestamp:   GetCurrentTimestamp(),
	}
}

// Validate validates the outbound command
func (c *OutboundCommandProto) Validate() error {
	if c.CommandType == "" {
		return ErrInvalidEnvelope
	}
	if c.InstanceID == "" {
		return ErrInvalidEnvelope
	}
	if c.TargetID == "" {
		return ErrInvalidEnvelope
	}
	return nil
}
