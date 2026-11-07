package adapter

import (
	"encoding/json"
	"testing"

	"github.com/e-spl/e-sp-line2/internal/platformspec"
	v3 "github.com/e-spl/e-sp-line2/internal/protocol/v3"
)

func TestBuiltinAdaptersCoverActivePlatforms(t *testing.T) {
	registry := NewRegistry()
	if err := RegisterBuiltinAdapters(registry); err != nil {
		t.Fatalf("register builtin adapters: %v", err)
	}

	all := registry.GetAll()
	if len(all) != 12 {
		t.Fatalf("registered %d adapters, want the 12 active platforms", len(all))
	}

	for _, a := range all {
		if a.Status != "active" {
			t.Fatalf("%q registered with status %q", a.ID, a.Status)
		}
		if a.RuntimeType != "node" {
			t.Fatalf("%q runtime type = %q, want node", a.ID, a.RuntimeType)
		}
		if len(a.I18n) != 2 {
			t.Fatalf("%q has %d locales, want zh-CN and en-US", a.ID, len(a.I18n))
		}
		for _, locale := range []string{"zh-CN", "en-US"} {
			resource, ok := a.I18n[locale]
			if !ok {
				t.Fatalf("%q misses locale %q", a.ID, locale)
			}
			if resource.DisplayName == "" || resource.Description == "" || resource.InstallGuide == "" {
				t.Fatalf("%q locale %q is incomplete", a.ID, locale)
			}
		}
	}
}

func TestUpcomingAdaptersAreInactive(t *testing.T) {
	upcoming := GetAdaptersByStage(platformspec.StageUpcoming)
	if len(upcoming) != 3 {
		t.Fatalf("upcoming adapters = %d, want 3", len(upcoming))
	}
	for _, a := range upcoming {
		if a.Status != "inactive" {
			t.Fatalf("%q status = %q, want inactive", a.ID, a.Status)
		}
	}
}

func TestAdapterConfigSchema(t *testing.T) {
	schema := GetTaobaoAdapterInfo().ConfigSchema

	if schema["type"] != "object" {
		t.Fatalf("schema type = %v, want object", schema["type"])
	}
	required, ok := schema["required"].([]string)
	if !ok || len(required) != 1 || required[0] != "cookie" {
		t.Fatalf("required = %v, want [cookie]", schema["required"])
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties missing")
	}
	heartbeat, ok := props["heartbeat_interval"].(map[string]interface{})
	if !ok {
		t.Fatal("heartbeat_interval missing")
	}
	if heartbeat["default"] != 15 {
		t.Fatalf("heartbeat default = %v, want 15", heartbeat["default"])
	}
}

func TestAdapterSecurityMarksCredentials(t *testing.T) {
	info := GetXianyuAdapterInfo()
	for _, field := range info.Security.SensitiveFields {
		if field == "heartbeat_interval" {
			t.Fatal("heartbeat_interval should not be a sensitive field")
		}
	}
	if len(info.Security.SensitiveFields) != len(info.Security.EncryptedFields) {
		t.Fatal("sensitive and encrypted fields diverged")
	}
	hasWrite := false
	for _, scope := range info.Security.PermissionScopes {
		if scope == "message:write" {
			hasWrite = true
		}
	}
	if !hasWrite {
		t.Fatalf("xianyu sends messages, want message:write scope: %v", info.Security.PermissionScopes)
	}
}

func TestReadOnlyPlatformHasNoWriteScope(t *testing.T) {
	info, ok := GetAdapterInfo("12306")
	if !ok {
		t.Fatal("12306 not registered")
	}
	for _, scope := range info.Security.PermissionScopes {
		if scope == "message:write" {
			t.Fatalf("12306 has no msg commands, got scope %q", scope)
		}
	}
}

func TestAdapterAliases(t *testing.T) {
	for alias, want := range map[string]string{"tb": "taobao", "xy": "xianyu", "ks": "kuaishou", "yt": "youtube"} {
		info, ok := GetAdapterInfo(alias)
		if !ok {
			t.Fatalf("alias %q does not resolve", alias)
		}
		if info.PlatformID != want {
			t.Fatalf("alias %q resolved to %q, want %q", alias, info.PlatformID, want)
		}
	}
}

func TestUnknownPlatform(t *testing.T) {
	if _, ok := GetAdapterInfo("nope"); ok {
		t.Fatal("unknown platform resolved")
	}
	if _, err := PlatformManifest("nope"); err != ErrAdapterNotFound {
		t.Fatalf("manifest error = %v, want ErrAdapterNotFound", err)
	}
}

func TestPlatformManifestIsValidV3(t *testing.T) {
	raw, err := PlatformManifest("taobao")
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}

	var manifest v3.AdapterManifest
	if err := json.Unmarshal([]byte(raw), &manifest); err != nil {
		t.Fatalf("manifest is not valid JSON: %v", err)
	}
	if manifest.ID != "taobao-adapter-v1" || manifest.PlatformID != "taobao" {
		t.Fatalf("manifest ids = %q / %q", manifest.ID, manifest.PlatformID)
	}
	if len(manifest.Capabilities) == 0 {
		t.Fatal("manifest has no capabilities")
	}
}

func TestRegistryHasCapability(t *testing.T) {
	registry := NewRegistry()
	if err := RegisterBuiltinAdapters(registry); err != nil {
		t.Fatalf("register: %v", err)
	}
	if !registry.HasCapability("taobao-adapter-v1", "msg.send") {
		t.Fatal("taobao should have msg.send")
	}
	// 淘宝没有关键词热搜，也没有关注 / 粉丝。
	if registry.HasCapability("taobao-adapter-v1", "keyword.hot") {
		t.Fatal("taobao should not have keyword.hot")
	}
	// 规划中的能力也在清单里，另外再带一条 .planned 标记。
	if !registry.HasCapability("taobao-adapter-v1", "item.search") {
		t.Fatal("taobao item.search should be listed")
	}
	if !registry.HasCapability("taobao-adapter-v1", "item.search.planned") {
		t.Fatal("taobao item.search is planned, want item.search.planned")
	}
}

func TestNoDuplicateCapabilities(t *testing.T) {
	for _, a := range GetAdaptersByStage(platformspec.StageActive) {
		seen := map[string]bool{}
		for _, c := range a.Capabilities {
			if seen[c] {
				t.Fatalf("%q lists capability %q twice", a.ID, c)
			}
			seen[c] = true
		}
	}
}
