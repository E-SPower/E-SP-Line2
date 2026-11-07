package platformspec

import "sort"

// Stage 区分已上线平台与「即将上车」的平台，对应 catbus README 的两张清单。
type Stage string

const (
	// StageActive 是已上线平台。
	StageActive Stage = "active"
	// StageUpcoming 是 catbus 里还在写、尚未开放的平台。
	StageUpcoming Stage = "upcoming"
)

// Support 是某条命令在某个平台上的支持度，取值与 catbus 能力矩阵的符号一一对应。
type Support string

const (
	// SupportImplemented 上游已有（矩阵 ✓）。
	SupportImplemented Support = "implemented"
	// SupportPartial 上游部分支持，限制写在 Platform.Notes（矩阵 ◐）。
	SupportPartial Support = "partial"
	// SupportPlanned 平台有这个概念、上游没有，执行报 NOT_IMPLEMENTED（矩阵 ○）。
	SupportPlanned Support = "planned"
	// SupportUnsupported 平台没有这个概念，执行报 UNSUPPORTED（矩阵 —）。
	SupportUnsupported Support = "unsupported"
)

// LoginMethod 是登录方式。
type LoginMethod struct {
	Method LoginKind `json:"method"`
	// Notes 说明这种方式的额外要求，例如抖音要先扫码拿到私信签名的私钥。
	Notes string `json:"notes,omitempty"`
}

// LoginKind 是登录方式的取值。
type LoginKind string

// 登录方式。与 catbus 的 LOGIN_METHODS 一致。
const (
	LoginQrcode   LoginKind = "qrcode"
	LoginSms      LoginKind = "sms"
	LoginPassword LoginKind = "password"
	LoginCookie   LoginKind = "cookie"
	LoginOauth    LoginKind = "oauth"
	LoginApikey   LoginKind = "apikey"
)

// ConfigField 是适配器实例配置里的一项，用来生成 JSON Schema。
type ConfigField struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	Required      bool   `json:"required"`
	Description   string `json:"description"`
	DescriptionEN string `json:"description_en"`
	Default       int    `json:"default,omitempty"`
}

// Platform 是一个平台的注册信息。
type Platform struct {
	ID string
	// Ordinal 是能力矩阵的列序，All() 按它排序。
	Ordinal int
	Name    string
	NameEN  string
	Item    string
	ItemEN  string
	Aliases []string
	// Upstream 是上游实现的位置：catbus 的 UPSTREAM 记录的是仓库或官方站点。
	Upstream string
	// Site 是平台网页版；DM 是私信页。
	Site string
	DM   string

	LoginMethods []LoginMethod
	ConfigFields []ConfigField
	// Capabilities 是折算后的能力清单，既有 E-SP-Line2 的顶层能力词，也有 `<resource>.<action>` 细项。
	Capabilities []string
	// Notes 是能力矩阵下面那些行内说明，按「命令：说明」保存。
	Notes []string
	Stage Stage
}

// Command 是一条规范命令（资源 + 动作）在各平台上的支持度。
type Command struct {
	Resource string
	Action   string
	Section  string
	Support  map[string]Support
}

// For 返回这条命令在某个平台上的支持度，未列出即不支持。
func (c *Command) For(platform string) Support {
	if s, ok := c.Support[platform]; ok {
		return s
	}
	return SupportUnsupported
}

var (
	platforms = map[string]*Platform{}
	order     []string
	commands  []*Command
	byAction  = map[string]*Command{}
	aliases   = map[string]string{}
)

func register(p *Platform) {
	if _, exists := platforms[p.ID]; exists {
		return
	}
	platforms[p.ID] = p
	order = append(order, p.ID)
	for _, a := range p.Aliases {
		aliases[a] = p.ID
	}
}

func registerCommand(c *Command) {
	commands = append(commands, c)
	if _, exists := byAction[c.Resource+" "+c.Action]; !exists {
		byAction[c.Resource+" "+c.Action] = c
	}
}

// All 返回全部平台，按 catbus 能力矩阵的列顺序。
func All() []*Platform {
	out := make([]*Platform, 0, len(order))
	for _, id := range order {
		out = append(out, platforms[id])
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ordinal < out[j].Ordinal })
	return out
}

// Active 返回已上线平台。
func Active() []*Platform {
	out := make([]*Platform, 0, len(order))
	for _, id := range order {
		if platforms[id].Stage == StageActive {
			out = append(out, platforms[id])
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ordinal < out[j].Ordinal })
	return out
}

// Upcoming 返回「即将上车」的平台。
func Upcoming() []*Platform {
	out := make([]*Platform, 0, len(order))
	for _, id := range order {
		if platforms[id].Stage == StageUpcoming {
			out = append(out, platforms[id])
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ordinal < out[j].Ordinal })
	return out
}

// Get 按 id 或别名取平台。
func Get(id string) (*Platform, bool) {
	if id == "" {
		return nil, false
	}
	if p, ok := platforms[id]; ok {
		return p, true
	}
	if id, ok := aliases[id]; ok {
		return platforms[id], true
	}
	return nil, false
}

// Canonical 把别名归一成规范 id。
func Canonical(id string) string {
	if p, ok := Get(id); ok {
		return p.ID
	}
	return id
}

// Commands 返回全部命令，按资源、动作排序。
func Commands() []*Command {
	out := append([]*Command(nil), commands...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Resource != out[j].Resource {
			return out[i].Resource < out[j].Resource
		}
		return out[i].Action < out[j].Action
	})
	return out
}

// Lookup 按「资源 动作」查命令。
func Lookup(resource, action string) (*Command, bool) {
	c, ok := byAction[resource+" "+action]
	return c, ok
}

// Supports 判断平台是否有这条命令。
func Supports(platform, resource, action string) Support {
	c, ok := Lookup(resource, action)
	if !ok {
		return SupportUnsupported
	}
	return c.For(Canonical(platform))
}

// Sections 返回能力矩阵的分节顺序（去掉纯说明性的分节）。
func Sections() []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range commands {
		if c.Section == "" || seen[c.Section] {
			continue
		}
		seen[c.Section] = true
		out = append(out, c.Section)
	}
	return out
}
