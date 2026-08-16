package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// DocsDirectory is the base directory containing markdown documentation.
// It resolves relative to the process working directory (project root when
// running via `go run main.go` / `make build`).
const DocsDirectory = "docs"

// docIndexEntry describes one documentation page exposed to the frontend.
type docIndexEntry struct {
	Key   string `json:"key"`   // unique identifier used by the frontend route
	Title string `json:"title"` // human-readable title
	Path  string `json:"path"`  // relative file path inside docs/
	Group string `json:"group"` // category used by the frontend to group docs
}

// docIndex is the whitelist of documents that can be served. Any file not in
// this list is unreachable, which also prevents path-traversal attempts.
var docIndex = []docIndexEntry{
	// ── 使用指南 ──
	{Key: "getting-started", Title: "快速开始", Path: "user-guide/getting-started.md", Group: "guide"},
	{Key: "adapter-management", Title: "接入器管理", Path: "user-guide/adapters.md", Group: "guide"},
	{Key: "message-routing", Title: "消息路由", Path: "user-guide/routing.md", Group: "guide"},
	{Key: "troubleshooting", Title: "故障排除", Path: "user-guide/troubleshooting.md", Group: "guide"},

	// ── 接入器文档 ──
	{Key: "adapter-xianyu", Title: "闲鱼接入器", Path: "user-guide/adapters-xianyu.md", Group: "adapters"},
	{Key: "adapter-taobao", Title: "淘宝接入器", Path: "user-guide/adapters-taobao.md", Group: "adapters"},

	// ── API 参考 ──
	{Key: "rest-api", Title: "REST API", Path: "developer-guide/api-reference.md", Group: "api"},
	{Key: "websocket-api", Title: "WebSocket API", Path: "developer-guide/api-reference.md", Group: "api"},

	// ── 开发者指南 ──
	{Key: "adapter-development", Title: "接入器开发", Path: "developer-guide/adapter-development.md", Group: "dev"},
	{Key: "adapter-yaml", Title: "adapter.yaml 字段规则", Path: "developer-guide/adapter-yaml.md", Group: "dev"},
	{Key: "protocol", Title: "协议规范", Path: "developer-guide/protocol-v3.md", Group: "dev"},
	{Key: "contributing", Title: "参与贡献", Path: "developer-guide/contributing.md", Group: "dev"},
}

// lookupDoc finds a docIndexEntry by key. Returns nil if not whitelisted.
func lookupDoc(key string) *docIndexEntry {
	for i := range docIndex {
		if docIndex[i].Key == key {
			return &docIndex[i]
		}
	}
	return nil
}

// ListDocs returns the whitelisted document index.
func ListDocs() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data":  docIndex,
			"total": len(docIndex),
		})
	}
}

// GetDoc reads and returns the markdown content of a whitelisted document.
func GetDoc() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Param("key")
		entry := lookupDoc(key)
		if entry == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
			return
		}

		// The path comes from our whitelist (never from user input directly),
		// but we still defend against traversal by resolving and re-checking.
		resolved := filepath.Clean(filepath.Join(DocsDirectory, entry.Path))
		if !strings.HasPrefix(resolved, DocsDirectory+string(os.PathSeparator)) &&
			resolved != DocsDirectory {
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid document path"})
			return
		}

		content, err := os.ReadFile(resolved)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read document"})
			return
		}

		// Return JSON so the frontend can safely consume it through its API client.
		// The client unwraps `{data, ...}` into the data payload.
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"key":     entry.Key,
				"title":   entry.Title,
				"path":    entry.Path,
				"content": string(content),
			},
		})
	}
}
