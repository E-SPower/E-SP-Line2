package v3

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
