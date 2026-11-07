package platformspec

import "testing"

func TestAliasesResolveToCanonical(t *testing.T) {
	for _, p := range All() {
		for _, a := range p.Aliases {
			got, ok := Get(a)
			if !ok {
				t.Fatalf("alias %q of %q does not resolve", a, p.ID)
			}
			if got.ID != p.ID {
				t.Fatalf("alias %q resolves to %q, want %q", a, got.ID, p.ID)
			}
			if Canonical(a) != p.ID {
				t.Fatalf("Canonical(%q) = %q, want %q", a, Canonical(a), p.ID)
			}
		}
		if Canonical(p.ID) != p.ID {
			t.Fatalf("Canonical(%q) mangled the canonical id", p.ID)
		}
	}
}

func TestStagesArePartition(t *testing.T) {
	all := All()
	active := Active()
	upcoming := Upcoming()
	if len(active)+len(upcoming) != len(all) {
		t.Fatalf("stage partition lost platforms: %d active + %d upcoming != %d all", len(active), len(upcoming), len(all))
	}
	for _, p := range active {
		if p.Stage != StageActive {
			t.Fatalf("%q is in Active() but stage is %q", p.ID, p.Stage)
		}
	}
	for _, p := range upcoming {
		if p.Stage != StageUpcoming {
			t.Fatalf("%q is in Upcoming() but stage is %q", p.ID, p.Stage)
		}
	}
	// catbus README 的「即将上车」清单是 4 个平台之外的那几个，这里只固定数量以免漏改。
	if len(upcoming) != 3 {
		t.Fatalf("upcoming stage has %d platforms, want 3 (jimeng, maoyan, mteam)", len(upcoming))
	}
}

func TestEveryPlatformHasMeta(t *testing.T) {
	for _, p := range All() {
		if p.Name == "" || p.NameEN == "" || p.Item == "" || p.ItemEN == "" {
			t.Fatalf("%q misses display metadata: %+v", p.ID, p)
		}
	}
}

func TestSupportLookup(t *testing.T) {
	if got := Supports("xhs", "msg", "send"); got != SupportImplemented {
		t.Fatalf("xhs msg send = %q, want implemented", got)
	}
	if got := Supports("xhs", "item", "delete"); got != SupportPlanned {
		t.Fatalf("xhs item delete = %q, want planned", got)
	}
	if got := Supports("youtube", "msg", "send"); got != SupportUnsupported {
		t.Fatalf("youtube msg send = %q, want unsupported", got)
	}
	if got := Supports("bili", "user", "get"); got != SupportImplemented {
		t.Fatalf("alias lookup for bilibili user get = %q, want implemented", got)
	}
	if _, ok := Lookup("nope", "nope"); ok {
		t.Fatal("unknown command should not be found")
	}
}

// 能力清单必须与能力矩阵一致：矩阵里 implemented / partial 的命令一定在清单里。
func TestCapabilitiesTrackMatrix(t *testing.T) {
	for _, p := range Active() {
		for _, c := range Commands() {
			switch c.For(p.ID) {
			case SupportImplemented, SupportPartial:
				name := c.Resource + "." + c.Action
				if !contains(p.Capabilities, name) {
					t.Fatalf("%q has %q but capability %q missing", p.ID, c.For(p.ID), name)
				}
			}
		}
	}
}

func TestConfigFieldsAreKnown(t *testing.T) {
	allowed := map[string]bool{
		"cookie": true, "device_id": true, "token": true, "api_key": true,
		"uid": true, "box": true, "heartbeat_interval": true, "reconnect_delay": true,
	}
	for _, p := range All() {
		for _, f := range p.ConfigFields {
			if !allowed[f.Name] {
				t.Fatalf("%q has unknown config field %q", p.ID, f.Name)
			}
			if f.Description == "" || f.DescriptionEN == "" {
				t.Fatalf("%q config field %q misses a description", p.ID, f.Name)
			}
		}
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
