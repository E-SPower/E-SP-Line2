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

	// Tool (inactive field) invocation commands. tool_call is sent to a bridge
	// to actively query/act on e-commerce data; tool_result carries the answer
	// back to the requesting adapter gateway client.
	CommandTypeToolCall   = "tool_call"
	CommandTypeToolResult = "tool_result"

	// E-commerce action commands (bridges that support write operations).
	CommandTypeShipOrder   = "ship_order"
	CommandTypeUpdateOrder = "update_order"

	// ── XianYuApis 原版兼容命令（HTTP 接口 + lwp WebSocket 命令）──────────
	// HTTP 接口命令（goofish_apis.py）。
	CommandTypePublishItem          = "publish_item"           // mtop.idle.pc.idleitem.publish
	CommandTypeUpdateItem           = "update_item"            // mtop.taobao.idle.item.update
	CommandTypeUpdateItemPrice      = "update_item_price"      // 商品改价
	CommandTypeGetCategoryRecommend = "get_category_recommend" // kgraph.property.recommend
	CommandTypeGetDefaultAddress    = "get_default_address"    // local.poi.get
	CommandTypeGetToken             = "get_token"              // login.token
	CommandTypeLoginQRCode          = "login_qrcode"           // 扫码登录

	// lwp WebSocket 命令（goofish_live.py）。
	CommandTypeListUserMessages    = "list_user_messages"     // /r/MessageManager/listUserMessages
	CommandTypeCreateSingleChat    = "create_single_chat"     // /r/SingleChatConversation/create
	CommandTypeSendByReceiverScope = "send_by_receiver_scope" // /r/MessageSend/sendByReceiverScope
	CommandTypeAckDiff             = "ack_diff"               // /r/SyncStatus/ackDiff
	CommandTypeRegister            = "reg"                    // /reg 注册连接
	CommandTypeHeartbeat           = "heartbeat"              // /! 心跳
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
