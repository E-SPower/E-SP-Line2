package handler

import (
	"net/http"

	"github.com/e-spl/e-sp-line2/internal/messagefields"
	"github.com/e-spl/e-sp-line2/internal/platformspec"
	"github.com/gin-gonic/gin"
)

// ListPlatformCaps lists every platform with its command matrix.
//
// 这是 catbus `catbus platforms` 与 docs/capabilities.md 的合并面：同一个注册表同时喂给
// 命令行帮助、能力矩阵和这里的 JSON。
func ListPlatformCaps(c *gin.Context) {
	stage := c.DefaultQuery("stage", string(platformspec.StageActive))

	var platforms []*platformspec.Platform
	switch platformspec.Stage(stage) {
	case platformspec.StageUpcoming:
		platforms = platformspec.Upcoming()
	case "all":
		platforms = platformspec.All()
	default:
		platforms = platformspec.Active()
	}

	commands := platformspec.Commands()
	out := make([]gin.H, 0, len(platforms))
	for _, p := range platforms {
		support := make(map[string]platformspec.Support, len(commands))
		for _, cmd := range commands {
			support[cmd.Resource+" "+cmd.Action] = cmd.For(p.ID)
		}
		out = append(out, gin.H{
			"id":             p.ID,
			"name":           p.Name,
			"item":           p.Item,
			"aliases":        p.Aliases,
			"stage":          p.Stage,
			"upstream":       p.Upstream,
			"site":           p.Site,
			"login_methods":  loginMethods(p),
			"capabilities":   p.Capabilities,
			"notes":          p.Notes,
			"support_matrix": support,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     out,
		"commands": commands,
		"total":    len(out),
	})
}

// GetPlatformCap gets one platform with its support matrix.
func GetPlatformCap(c *gin.Context) {
	p, ok := platformspec.Get(c.Param("id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "platform not found"})
		return
	}

	commands := make([]gin.H, 0)
	for _, cmd := range platformspec.Commands() {
		state := cmd.For(p.ID)
		if state == platformspec.SupportUnsupported {
			continue
		}
		commands = append(commands, gin.H{
			"resource": cmd.Resource,
			"action":   cmd.Action,
			"section":  cmd.Section,
			"support":  state,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"id":            p.ID,
		"name":          p.Name,
		"item":          p.Item,
		"aliases":       p.Aliases,
		"stage":         p.Stage,
		"upstream":      p.Upstream,
		"site":          p.Site,
		"dm":            p.DM,
		"login_methods": loginMethods(p),
		"config_fields": p.ConfigFields,
		"capabilities":  p.Capabilities,
		"notes":         p.Notes,
		"commands":      commands,
	})
}

// ListMessageFields serves the message field contract.
//
// 文档正本是 docs/developer-guide/message-fields.md，这里输出同一份定义的结构化形式，
// 前端据此渲染字段表，测试检查两边一致。
func ListMessageFields(c *gin.Context) {
	kind := c.Query("kind")

	if kind != "" {
		o, ok := messagefields.ObjectOf(kind)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "unknown object kind"})
			return
		}
		c.JSON(http.StatusOK, o)
		return
	}

	objects := messagefields.Objects()
	out := make(map[string]*messagefields.Object, len(objects))
	for k, v := range objects {
		out[k] = v
	}
	c.JSON(http.StatusOK, gin.H{
		"data":     out,
		"elements": messagefields.Elements,
	})
}

// loginMethods flattens the login methods to plain strings for the API.
func loginMethods(p *platformspec.Platform) []string {
	out := make([]string, 0, len(p.LoginMethods))
	for _, m := range p.LoginMethods {
		out = append(out, string(m.Method))
	}
	return out
}
