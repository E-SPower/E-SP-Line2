package adapter

import (
	"encoding/json"

	"github.com/e-spl/e-sp-line2/internal/platformspec"
	v3 "github.com/e-spl/e-sp-line2/internal/protocol/v3"
)

// GetTaobaoAdapterInfo returns Taobao adapter info
func GetTaobaoAdapterInfo() *AdapterInfo {
	return getPlatformAdapterInfo("taobao")
}

// GetXianyuAdapterInfo returns Xianyu adapter info
func GetXianyuAdapterInfo() *AdapterInfo {
	return getPlatformAdapterInfo("xianyu")
}

// GetAdapterInfo returns a built-in adapter by platform id or alias.
//
// 平台清单来自 platformspec（迁移自 catbus 的平台注册表），因此新增平台不需要再写
// 一个 Get<Platform>AdapterInfo。
func GetAdapterInfo(platform string) (*AdapterInfo, bool) {
	p, ok := platformspec.Get(platform)
	if !ok {
		return nil, false
	}
	return BuiltinAdapterInfo(p), true
}

// GetAdaptersByStage returns built-in adapters whose platform is on the given stage.
func GetAdaptersByStage(stage platformspec.Stage) []*AdapterInfo {
	var platforms []*platformspec.Platform
	switch stage {
	case platformspec.StageActive:
		platforms = platformspec.Active()
	case platformspec.StageUpcoming:
		platforms = platformspec.Upcoming()
	}
	out := make([]*AdapterInfo, 0, len(platforms))
	for _, p := range platforms {
		out = append(out, BuiltinAdapterInfo(p))
	}
	return out
}

func getPlatformAdapterInfo(platform string) *AdapterInfo {
	info, ok := GetAdapterInfo(platform)
	if !ok {
		// 注册表里一定有这两个平台；丢了就让调用方在启动时立刻看到。
		panic("platform not registered in platformspec: " + platform)
	}
	return info
}

// RegisterBuiltinAdapters registers built-in adapters to registry
func RegisterBuiltinAdapters(registry *Registry) error {
	// 只注册已上线的平台；「即将上车」的平台留到启用时再注册。
	for _, p := range platformspec.Active() {
		if err := registry.Register(BuiltinAdapterInfo(p)); err != nil {
			return err
		}
	}
	return nil
}

// PlatformManifest returns the V3 adapter manifest of a platform.
func PlatformManifest(platform string) (string, error) {
	info, ok := GetAdapterInfo(platform)
	if !ok {
		return "", ErrAdapterNotFound
	}

	manifest := v3.AdapterManifest{
		ID:              info.ID,
		PlatformID:      info.PlatformID,
		Name:            info.Name,
		Version:         info.Version,
		RuntimeType:     info.RuntimeType,
		ProtocolVersion: info.ProtocolVersion,
		Capabilities:    info.Capabilities,
		ConfigSchema:    info.ConfigSchema,
		I18n:            info.I18n,
		Operations:      info.Operations,
		Security:        info.Security,
	}

	data, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// PlatformConfig is the per-platform instance config with common reconnect options.
type PlatformConfig struct {
	Cookie            string `json:"cookie,omitempty"`
	DeviceID          string `json:"device_id,omitempty"`
	Token             string `json:"token,omitempty"`
	APIKey            string `json:"api_key,omitempty"`
	UID               string `json:"uid,omitempty"`
	Box               string `json:"box,omitempty"`
	HeartbeatInterval int    `json:"heartbeat_interval,omitempty"`
	ReconnectDelay    int    `json:"reconnect_delay,omitempty"`
}

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
