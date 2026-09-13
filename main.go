package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/e-spl/e-sp-line2/internal/config"
	"github.com/e-spl/e-sp-line2/internal/server"
	"github.com/e-spl/e-sp-line2/pkg/bootstrap"
	"github.com/e-spl/e-sp-line2/pkg/logger"
)

// Build-time metadata. These variables are injected at compile time by the
// build scripts via:
//
//	-ldflags "-X main.version=<ver> -X main.buildTime=<utc>"
//
// They default to sane values so `go run main.go` still works.
var (
	version   = "dev"
	buildTime = ""
)

// fatal reports a fatal startup error in a way the user can actually see.
//
// On Windows a console application launched by double-clicking flashes its
// window and disappears before the message can be read, so the error is shown
// in a modal dialog and the process waits for the user to acknowledge it.
// On other platforms it behaves like log.Fatalf.
func fatal(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, "FATAL:", msg)
	showFatalDialog("E-SP-Line2 启动失败", msg)
	waitForAck()
	os.Exit(1)
}

// banner is the ASCII art shown at startup (also echoed by each adapter
// instance when it starts, so the WebUI instance log shows the same art).
const banner = `
  ███████╗   ███████╗██████╗     ██╗     ██╗███╗   ██╗███████╗██████╗
  ██╔════╝   ██╔════╝██╔══██╗    ██║     ██║████╗  ██║██╔════╝╚════██╗
  █████╗     ███████╗██████╔╝    ██║     ██║██╔██╗ ██║█████╗    ▄███╔╝
  ██╔══╝     ╚════██║██╔═══╝     ██║     ██║██║╚██╗██║██╔══╝  ▄▀══╝
  ███████╗   ███████║██║         ███████╗██║██║ ╚████║███████╗███████╗
  ╚══════╝   ╚══════╝╚═╝         ╚══════╝╚═╝╚═╝  ╚═══╝╚══════╝╚══════╝

  Power By LangBot-community-team

  --------------------------------------------------------------------
`

func main() {
	// Normalize the working directory and make sure runtime directories exist.
	// This is what prevents the "double-click and it instantly closes" failure
	// on Windows, where the initial working directory may not be the folder
	// holding config/ and data/.
	for _, note := range bootstrap.Prepare() {
		fmt.Println("[INFO]", note)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fatal("Failed to load config: %v", err)
	}

	// Initialize logger
	logger.Init(cfg.LogLevel)
	defer logger.Sync()

	fmt.Print(banner)
	logger.Info("Starting E-SP-Line2 server...")
	logger.Infof("Server version: %s (build=%s, compiled=%s)", cfg.Version, version, buildTime)

	// Create server
	srv, err := server.New(cfg)
	if err != nil {
		fatal("Failed to create server: %v", err)
	}

	// Start server in goroutine
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
		logger.Infof("Server listening on %s", addr)
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			fatal("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		logger.Error("Server forced to shutdown", logger.ErrorField("error", err))
	}

	logger.Info("Server exited")
}
