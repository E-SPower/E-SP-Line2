package adapter

import (
	"fmt"
	"strings"

	"github.com/e-spl/e-sp-line2/internal/platformspec"
	v3 "github.com/e-spl/e-sp-line2/internal/protocol/v3"
)

// 从 catbus 迁移过来的平台适配器。
//
// catbus 的约定是「一个平台，一组文件」：client（会话与签名）、api（一一对应上游方法）、
// normalize（把原始数据转成统一类型）、resolve（ID、链接、分享短链、me 都能当参数）。
// E-SP-Line2 这边把同一组约定放平：平台上的一切元数据来自 platformspec，适配器包只负责
// 把注册表翻译成 AdapterInfo / manifest，运行时侧（client、api、normalize、resolve）按
// 平台目录放置，见 docs/developer-guide/adapter-development.md。

// AdapterIDFor 返回平台内置适配器的 id。适配器包按平台一份，版本号固定 v1。
func AdapterIDFor(platform string) string {
	return platform + "-adapter-v1"
}

// BuiltinAdapterInfo 由平台注册表生成内置适配器。
func BuiltinAdapterInfo(p *platformspec.Platform) *AdapterInfo {
	info := &AdapterInfo{
		ID:              AdapterIDFor(p.ID),
		PlatformID:      p.ID,
		Name:            fmt.Sprintf("%s Message Adapter", p.NameEN),
		Version:         "1.0.0",
		RuntimeType:     "node",
		ProtocolVersion: "v3",
		Capabilities:    p.Capabilities,
		ConfigSchema:    configSchema(p),
		I18n:            i18n(p),
		Operations:      operations(p),
		Security:        security(p),
		Status:          adapterStatus(p),
	}
	return info
}

// adapterStatus 把「即将上车」的平台标成 inactive，新装的环境不会自动启用它们。
func adapterStatus(p *platformspec.Platform) string {
	if p.Stage == platformspec.StageUpcoming {
		return "inactive"
	}
	return "active"
}

// operations 里只有平台自己的重连参数才覆盖默认值，其余沿用协议默认。
func operations(p *platformspec.Platform) v3.OperationPolicy {
	policy := v3.OperationPolicy{
		HeartbeatInterval: v3.DefaultHeartbeatInterval,
		ReconnectDelay:    v3.DefaultReconnectDelay,
		MaxRetries:        v3.DefaultMaxRetries,
		MaxQueueSize:      v3.DefaultMaxQueueSize,
	}
	for _, f := range p.ConfigFields {
		if f.Default == 0 {
			continue
		}
		switch f.Name {
		case "heartbeat_interval":
			policy.HeartbeatInterval = f.Default
		case "reconnect_delay":
			policy.ReconnectDelay = f.Default
		}
	}
	return policy
}

// security 把凭证字段全部标成敏感并加密。
func security(p *platformspec.Platform) v3.SecurityPolicy {
	sensitive := make([]string, 0, len(p.ConfigFields))
	for _, f := range p.ConfigFields {
		if f.Type != "integer" {
			sensitive = append(sensitive, f.Name)
		}
	}
	scopes := []string{"message:read"}
	if hasCapability(p, "send_text") {
		scopes = append(scopes, "message:write")
	}
	if hasCapability(p, "get_product_info") {
		scopes = append(scopes, "product:read")
	}
	if hasCapability(p, "get_order_info") {
		scopes = append(scopes, "order:read")
	}
	return v3.SecurityPolicy{
		SensitiveFields:  sensitive,
		EncryptedFields:  sensitive,
		PermissionScopes: scopes,
	}
}

func hasCapability(p *platformspec.Platform, capability string) bool {
	for _, c := range p.Capabilities {
		if c == capability {
			return true
		}
	}
	return false
}

// configSchema 把配置字段转成 JSON Schema。required 只有平台真的需要的那几个字段。
func configSchema(p *platformspec.Platform) map[string]interface{} {
	properties := map[string]interface{}{}
	required := make([]string, 0, len(p.ConfigFields))

	for _, f := range p.ConfigFields {
		prop := map[string]interface{}{
			"type":        f.Type,
			"description": f.Description,
		}
		if f.Default != 0 {
			prop["default"] = f.Default
		}
		if f.Name == "box" {
			prop["enum"] = []string{"inbox", "system", "sent"}
		}
		properties[f.Name] = prop
		if f.Required {
			required = append(required, f.Name)
		}
	}

	return map[string]interface{}{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

// i18n 生成中英两份文案。安装引导按平台支持的登录方式拼。
func i18n(p *platformspec.Platform) map[string]v3.I18nResource {
	return map[string]v3.I18nResource{
		"zh-CN": {
			DisplayName:   p.Name + "适配器",
			Description:   fmt.Sprintf("接收和发送 %s 的私信与通知", p.Name),
			InstallGuide:  installGuide(p, true),
			ErrorMessages: errorMessages(true),
		},
		"en-US": {
			DisplayName:   p.NameEN + " Adapter",
			Description:   fmt.Sprintf("Receive and send %s messages and notifications", p.NameEN),
			InstallGuide:  installGuide(p, false),
			ErrorMessages: errorMessages(false),
		},
	}
}

func installGuide(p *platformspec.Platform, zh bool) string {
	steps := make([]string, 0, 4)
	if zh {
		switch {
		case len(p.LoginMethods) == 0:
			steps = append(steps, "1. 免登录平台，无需配置凭证")
			return strings.Join(steps, "\n")
		default:
			for i, m := range p.LoginMethods {
				steps = append(steps, fmt.Sprintf("%d. 可用的登录方式：%s", i+1, loginLabel(m.Method, true)))
			}
		}
		steps = append(steps, fmt.Sprintf("%d. 在 %s 完成登录后把凭证填入配置", len(steps)+1, p.Site))
		return strings.Join(steps, "\n")
	}
	if len(p.LoginMethods) == 0 {
		return "1. No credentials required"
	}
	for i, m := range p.LoginMethods {
		steps = append(steps, fmt.Sprintf("%d. Supported login method: %s", i+1, loginLabel(m.Method, false)))
	}
	steps = append(steps, fmt.Sprintf("%d. Log in at %s, then put the credential into the config", len(steps)+1, p.Site))
	return strings.Join(steps, "\n")
}

func loginLabel(k platformspec.LoginKind, zh bool) string {
	switch k {
	case platformspec.LoginQrcode:
		if zh {
			return "扫码"
		}
		return "QR code"
	case platformspec.LoginSms:
		if zh {
			return "短信验证码"
		}
		return "SMS code"
	case platformspec.LoginPassword:
		if zh {
			return "账号密码"
		}
		return "password"
	case platformspec.LoginCookie:
		if zh {
			return "Cookie"
		}
		return "cookie"
	case platformspec.LoginOauth:
		if zh {
			return "OAuth 授权"
		}
		return "OAuth"
	case platformspec.LoginApikey:
		if zh {
			return "API Key"
		}
		return "API key"
	default:
		return string(k)
	}
}

func errorMessages(zh bool) map[string]string {
	if zh {
		return map[string]string{
			"cookie_expired":    "登录态已过期，请重新登录",
			"connection_failed": "连接失败，请检查网络和代理",
			"risk_control":      "被风控拦截，放慢节奏稍后重试",
			"not_implemented":   "该能力还在规划中",
			"unsupported":       "该平台没有这个能力",
		}
	}
	return map[string]string{
		"cookie_expired":    "Credential expired, please log in again",
		"connection_failed": "Connection failed, check network and proxy",
		"risk_control":      "Blocked by risk control, slow down and retry later",
		"not_implemented":   "This capability is still planned",
		"unsupported":       "This platform does not have the capability",
	}
}
