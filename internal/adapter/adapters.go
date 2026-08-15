package adapter

import v3 "github.com/e-spl/e-sp-line2/internal/protocol/v3"

// TaobaoAdapterConfig represents Taobao adapter configuration
type TaobaoAdapterConfig struct {
	Cookie            string `json:"cookie"`
	DeviceID          string `json:"device_id,omitempty"`
	HeartbeatInterval int    `json:"heartbeat_interval,omitempty"`
	ReconnectDelay    int    `json:"reconnect_delay,omitempty"`
}

// XianyuAdapterConfig represents Xianyu adapter configuration
type XianyuAdapterConfig struct {
	Cookie            string `json:"cookie"`
	DeviceID          string `json:"device_id,omitempty"`
	HeartbeatInterval int    `json:"heartbeat_interval,omitempty"`
	ReconnectDelay    int    `json:"reconnect_delay,omitempty"`
}

// GetTaobaoAdapterInfo returns Taobao adapter info
func GetTaobaoAdapterInfo() *AdapterInfo {
	return &AdapterInfo{
		ID:              "taobao-adapter-v1",
		PlatformID:      "taobao",
		Name:            "Taobao Message Adapter",
		Version:         "1.0.0",
		RuntimeType:     "python",
		ProtocolVersion: "v3",
		Capabilities: []string{
			"receive_message",
			"send_text",
			"send_image",
			"upload_media",
			"get_history",
			"create_conversation",
		},
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"cookie": map[string]interface{}{
					"type":        "string",
					"description": "Taobao login cookie",
					"format":      "password",
				},
				"device_id": map[string]interface{}{
					"type":        "string",
					"description": "Device ID (optional)",
				},
				"heartbeat_interval": map[string]interface{}{
					"type":        "integer",
					"description": "Heartbeat interval in seconds",
					"default":     15,
				},
				"reconnect_delay": map[string]interface{}{
					"type":        "integer",
					"description": "Reconnect delay in seconds",
					"default":     5,
				},
			},
			"required": []string{"cookie"},
		},
		I18n: map[string]v3.I18nResource{
			"zh-CN": {
				DisplayName:  "淘宝消息适配器",
				Description:  "支持淘宝平台消息收发，多开多账号，每个实例独立 Cookie 配置",
				InstallGuide: "1. 登录淘宝网页版\n2. 复制 Cookie\n3. 在实例管理中创建多个实例（多开）\n4. 每个实例填写独立 Cookie",
				ErrorMessages: map[string]string{
					"cookie_expired":    "Cookie 已过期，请重新登录",
					"connection_failed": "连接失败，请检查网络",
				},
			},
			"en-US": {
				DisplayName:  "Taobao Message Adapter",
				Description:  "Supports Taobao messaging with multiple accounts; each instance has its own Cookie",
				InstallGuide: "1. Login to Taobao web\n2. Copy Cookie\n3. Create multiple instances in Instance Management (multi-instance)\n4. Fill in a separate Cookie per instance",
				ErrorMessages: map[string]string{
					"cookie_expired":    "Cookie expired, please login again",
					"connection_failed": "Connection failed, please check network",
				},
			},
		},
		Operations: v3.OperationPolicy{
			HeartbeatInterval: 15,
			ReconnectDelay:    5,
			MaxRetries:        3,
			MaxQueueSize:      1000,
		},
		Security: v3.SecurityPolicy{
			SensitiveFields:  []string{"cookie"},
			EncryptedFields:  []string{"cookie"},
			PermissionScopes: []string{"message:read", "message:write"},
		},
		Status: "active",
	}
}

// GetXianyuAdapterInfo returns Xianyu adapter info
func GetXianyuAdapterInfo() *AdapterInfo {
	return &AdapterInfo{
		ID:              "xianyu-adapter-v1",
		PlatformID:      "xianyu",
		Name:            "Xianyu Message Adapter",
		Version:         "1.0.0",
		RuntimeType:     "python",
		ProtocolVersion: "v3",
		Capabilities: []string{
			"receive_message",
			"send_text",
			"send_image",
			"upload_media",
			"get_history",
			"create_conversation",
			"get_product_info",
		},
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"cookie": map[string]interface{}{
					"type":        "string",
					"description": "Xianyu login cookie",
					"format":      "password",
				},
				"device_id": map[string]interface{}{
					"type":        "string",
					"description": "Device ID (optional)",
				},
				"heartbeat_interval": map[string]interface{}{
					"type":        "integer",
					"description": "Heartbeat interval in seconds",
					"default":     15,
				},
				"reconnect_delay": map[string]interface{}{
					"type":        "integer",
					"description": "Reconnect delay in seconds",
					"default":     5,
				},
			},
			"required": []string{"cookie"},
		},
		I18n: map[string]v3.I18nResource{
			"zh-CN": {
				DisplayName:  "闲鱼消息适配器",
				Description:  "支持闲鱼平台消息收发，多开多账号，每个实例独立 Cookie 配置",
				InstallGuide: "1. 登录闲鱼网页版\n2. 复制 Cookie\n3. 在实例管理中创建多个实例（多开）\n4. 每个实例填写独立 Cookie",
				ErrorMessages: map[string]string{
					"cookie_expired":    "Cookie 已过期，请重新登录",
					"connection_failed": "连接失败，请检查网络",
				},
			},
			"en-US": {
				DisplayName:  "Xianyu Message Adapter",
				Description:  "Supports Xianyu messaging with multiple accounts; each instance has its own Cookie",
				InstallGuide: "1. Login to Xianyu web\n2. Copy Cookie\n3. Create multiple instances in Instance Management (multi-instance)\n4. Fill in a separate Cookie per instance",
				ErrorMessages: map[string]string{
					"cookie_expired":    "Cookie expired, please login again",
					"connection_failed": "Connection failed, please check network",
				},
			},
		},
		Operations: v3.OperationPolicy{
			HeartbeatInterval: 15,
			ReconnectDelay:    5,
			MaxRetries:        3,
			MaxQueueSize:      1000,
		},
		Security: v3.SecurityPolicy{
			SensitiveFields:  []string{"cookie"},
			EncryptedFields:  []string{"cookie"},
			PermissionScopes: []string{"message:read", "message:write", "product:read"},
		},
		Status: "active",
	}
}

// RegisterBuiltinAdapters registers built-in adapters to registry
func RegisterBuiltinAdapters(registry *Registry) error {
	// Register Taobao adapter
	if err := registry.Register(GetTaobaoAdapterInfo()); err != nil {
		return err
	}

	// Register Xianyu adapter
	if err := registry.Register(GetXianyuAdapterInfo()); err != nil {
		return err
	}

	return nil
}
