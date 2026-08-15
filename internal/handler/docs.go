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
}

// docIndex is the whitelist of documents that can be served. Any file not in
// this list is unreachable, which also prevents path-traversal attempts.
var docIndex = []docIndexEntry{
	{Key: "getting-started", Title: "快速开始", Path: "user-guide/getting-started.md"},
	{Key: "platform-config", Title: "平台配置", Path: "user-guide/getting-started.md"},
	{Key: "adapter-management", Title: "接入器管理", Path: "user-guide/adapters.md"},
	{Key: "message-routing", Title: "消息路由", Path: "user-guide/routing.md"},
	{Key: "troubleshooting", Title: "故障排除", Path: "user-guide/troubleshooting.md"},

	{Key: "rest-api", Title: "REST API", Path: "developer-guide/api-reference.md"},
	{Key: "websocket-api", Title: "WebSocket API", Path: "developer-guide/api-reference.md"},
	{Key: "auth", Title: "认证授权", Path: "developer-guide/api-reference.md"},
	{Key: "error-codes", Title: "错误码说明", Path: "developer-guide/api-reference.md"},

	{Key: "adapter-development", Title: "接入器开发", Path: "developer-guide/adapter-development.md"},
	{Key: "protocol", Title: "协议规范", Path: "developer-guide/protocol-v3.md"},
	{Key: "best-practices", Title: "最佳实践", Path: "developer-guide/adapter-development.md"},
	{Key: "examples", Title: "示例代码", Path: "developer-guide/adapter-development.md"},
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
