//go:build desktop

// Command esp-desktop builds the native desktop build of E-SP-Line2.
//
// Unlike the default web/server build (which serves the API only and expects
// an external static server for web/dist), this entrypoint:
//
//  1. embeds the built frontend (web/dist) into the binary via pkg/webui;
//  2. starts the same HTTP API server on a loopback port;
//  3. opens a native WebView window (WebView2 on Windows, WebKitGTK on Linux)
//     pointing at the embedded SPA, so end users see an application window
//     instead of a browser tab.
//
// Build requirements:
//
//	go build -tags desktop .          # native, needs WebKitGTK/GTK on Linux
//
// The frontend MUST be built first (web/dist), otherwise the embedded bundle
// is empty and the window shows a blank page.
package main

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/e-spl/e-sp-line2/internal/config"
	"github.com/e-spl/e-sp-line2/internal/server"
	"github.com/e-spl/e-sp-line2/pkg/bootstrap"
	"github.com/e-spl/e-sp-line2/pkg/logger"
	"github.com/e-spl/e-sp-line2/pkg/webui"
	webview "github.com/webview/webview_go"
)

const (
	windowTitle = "E-SP-Line2"
	windowW     = 1280
	windowH     = 800
)

func main() {
	// On Windows the desktop build is a GUI-subsystem binary (-H=windowsgui) so
	// double-clicking does not pop a console window. Mirror output to
	// data/desktop.log and re-attach to a parent console when one exists, so the
	// build stays diagnosable.
	attachConsoleLog()

	// Fail fast with a clear message when the binary was compiled without a
	// frontend bundle (e.g. `go build -tags desktop` before `pnpm build`).
	if !webui.Available {
		fatal("前端资源未内嵌。请先执行 make build-frontend 再重新编译。")
	}
	if sub, ok := webui.FS(); !ok || !hasIndex(sub) {
		fatal("内嵌前端为空（缺少 index.html）。请先执行 make build-frontend 再重新编译。")
	}

	// Normalize working directory and create runtime directories so that
	// config/ and the default SQLite path (data/e-sp-line2.db) resolve even
	// when the .exe was launched by double-click.
	for _, note := range bootstrap.Prepare() {
		fmt.Println("[INFO]", note)
	}

	cfg, err := config.Load()
	if err != nil {
		fatal("加载配置失败: %v", err)
	}
	logger.Init(cfg.LogLevel)
	defer logger.Sync()

	// The desktop build must bind a loopback port only; an externally exposed
	// 0.0.0.0 listener makes no sense for an application window.
	cfg.Host = "127.0.0.1"
	if cfg.Port == 0 {
		cfg.Port = 8080
	}

	// Pick a port that is actually free.
	//
	// A desktop app is routinely launched while another copy (or the web
	// server build) already holds the configured port. Previously the listener
	// failed, but the failure happened in a goroutine while the main goroutine
	// kept going: waitReady() then talked to the *other* instance and the window
	// opened against a server this process does not own, ending with a stray
	// FATAL. Resolving the port up front removes that whole class of confusion.
	port, portChanged, err := pickFreePort(cfg.Port)
	if err != nil {
		fatal("找不到可用端口（已尝试 %d）：%v", cfg.Port, err)
	}
	if portChanged {
		logger.Warn("Configured port is in use; using another port",
			logger.Int("configured_port", cfg.Port),
			logger.Int("using_port", port))
	}
	cfg.Port = port

	srv, err := server.New(cfg)
	if err != nil {
		fatal("创建服务失败: %v", err)
	}

	// Surface a listener failure through this channel instead of only calling
	// fatal() inside the goroutine, so the window is never opened for a server
	// this process does not actually own.
	startErr := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			select {
			case startErr <- err:
			default:
			}
		}
	}()

	// Wait until the API is reachable before pointing the WebView at it.
	url := fmt.Sprintf("http://127.0.0.1:%d", cfg.Port)
	if !waitReadyOrError(url+"/health", 15*time.Second, startErr) {
		fatal("服务未能在 %s 就绪", url)
	}
	logger.Infof("Desktop window opening at %s", url)

	// Open the native window. debug=true enables devtools (set ESPL_DEBUG=1).
	//
	// On Windows the WebView backend requires the WebView2 runtime. When it is
	// missing, webview_create returns a NULL handle instead of raising an error,
	// so the failure must be checked explicitly -- otherwise the process would
	// exit silently with no window (or crash on the first call).
	w := webview.New(os.Getenv("ESPL_DEBUG") == "1")
	if w == nil || w.Window() == nil {
		fatal("无法创建窗口：WebView 后端初始化失败。\n\n" +
			"Windows 需要 Microsoft Edge WebView2 Runtime（Win11 通常已内置）。\n" +
			"请安装后重试：\n" +
			"  https://developer.microsoft.com/microsoft-edge/webview2/\n\n" +
			"或使用 winget：\n" +
			"  winget install -e --id Microsoft.EdgeWebView2Runtime")
	}
	defer w.Destroy()

	w.SetTitle(windowTitle)
	w.SetSize(windowW, windowH, webview.HintNone)
	w.Navigate(url)

	// Allow Ctrl+C / SIGTERM to close the window.
	//
	// IMPORTANT: WebView/GTK calls (Destroy, Terminate, ...) must run on the
	// main thread. Calling them from a signal goroutine aborts the process
	// (SIGABRT during cgo execution). Dispatch marshals the closure back onto
	// the main UI thread before terminating.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		w.Dispatch(func() {
			w.Terminate()
		})
	}()

	// Blocks until the window is closed (or Terminate is dispatched above).
	// The deferred Destroy then runs on the main thread as required.
	w.Run()

	logger.Info("Window closed, shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Stop(ctx); err != nil {
		logger.Error("Server shutdown error", logger.ErrorField("error", err))
	}
	logger.Info("Server exited")
}

// waitReadyOrError polls an HTTP endpoint until it returns 200, the timeout
// elapses, or the server goroutine reports a startup error.
//
// Watching startErr matters: a listener failure would otherwise leave the
// caller waiting the full timeout and then reporting the vague "did not become
// ready", hiding the real reason (most often a port already in use).
func waitReadyOrError(url string, timeout time.Duration, startErr <-chan error) bool {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		select {
		case err := <-startErr:
			logger.Error("Server failed to start", logger.ErrorField("error", err))
			return false
		default:
		}

		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return true
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	return false
}

// hasIndex reports whether the embedded bundle contains an index.html entry.
func hasIndex(fsys fs.FS) bool {
	if _, err := fs.Stat(fsys, "index.html"); err != nil {
		return false
	}
	return true
}
