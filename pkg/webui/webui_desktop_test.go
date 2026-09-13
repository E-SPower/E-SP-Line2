//go:build desktop

package webui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestEmbeddedFrontendAvailable 验证 desktop 构建确实内嵌了前端产物。
func TestEmbeddedFrontendAvailable(t *testing.T) {
	if !Available {
		t.Fatal("desktop 构建下 Available 应为 true")
	}
	sub, ok := FS()
	if !ok {
		t.Fatal("FS() 返回失败")
	}
	f, err := sub.Open("index.html")
	if err != nil {
		t.Fatalf("内嵌产物缺少 index.html: %v", err)
	}
	defer f.Close()
	b, _ := io.ReadAll(f)
	if !strings.Contains(strings.ToLower(string(b)), "<!doctype html") {
		t.Fatalf("index.html 内容异常: %q", string(b[:min(len(b), 80)]))
	}
}

// TestSPAHandlerFallback 验证未知路径回退到 index.html（客户端路由）。
func TestSPAHandlerFallback(t *testing.T) {
	srv := httptest.NewServer(Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/some/deep/client/route")
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(strings.ToLower(string(body)), "<!doctype html") {
		t.Fatal("深链未回退到 index.html")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
