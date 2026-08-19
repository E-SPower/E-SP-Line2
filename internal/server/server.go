package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/e-spl/e-sp-line2/internal/adaptergateway"
	"github.com/e-spl/e-sp-line2/internal/config"
	"github.com/e-spl/e-sp-line2/internal/handler"
	"github.com/e-spl/e-sp-line2/internal/middleware"
	"github.com/e-spl/e-sp-line2/internal/models"
	v3 "github.com/e-spl/e-sp-line2/internal/protocol/v3"
	"github.com/e-spl/e-sp-line2/internal/service"
	"github.com/e-spl/e-sp-line2/pkg/logger"
	"github.com/gin-gonic/gin"
)

// Server represents the HTTP server
type Server struct {
	cfg        *config.Config
	httpServer *http.Server
	router     *gin.Engine
	services   *service.Services
	gateway    *adaptergateway.Gateway
}

// New creates a new server instance
func New(cfg *config.Config) (*Server, error) {
	// Inject JWT secret into middleware for token validation
	middleware.SetJWTSecret(cfg.JWT.Secret)

	// Initialize services
	services, err := service.NewServices(cfg)
	if err != nil {
		return nil, err
	}

	// Set gin mode
	if cfg.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())

	s := &Server{
		cfg:      cfg,
		router:   router,
		services: services,
	}

	// Initialize the adapter gateway (接入器) WebSocket server.
	s.gateway = adaptergateway.NewGateway(
		adaptergateway.DefaultGatewayConfig(),
		services.AdapterGateway,
	)

	// Register the change callback: whenever an adapter is created, updated or
	// deleted, the gateway reloads client-mode outbound connections live.
	services.AdapterGateway.SetOnChangeCallback(func() {
		if adapters, _, err := services.AdapterGateway.List(1000, 0); err == nil {
			ptrs := make([]*models.Adapter, len(adapters))
			for i := range adapters {
				ptrs[i] = &adapters[i]
			}
			s.gateway.SyncAdapters(ptrs)
		}
	})

	// Start outbound connections for enabled client-mode adapters.
	if adapters, _, err := services.AdapterGateway.List(1000, 0); err == nil {
		ptrs := make([]*models.Adapter, len(adapters))
		for i := range adapters {
			ptrs[i] = &adapters[i]
		}
		s.gateway.SyncAdapters(ptrs)
	}

	s.setupRoutes()
	s.registerVersionCheck()

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
	}

	return s, nil
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", s.healthCheck)

	// API v1 group
	v1 := s.router.Group("/api/v1")
	{
		// Auth routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", handler.Register(s.services.Auth))
			auth.POST("/login", handler.Login(s.services.Auth))
			auth.GET("/me", middleware.AuthRequired(), handler.GetCurrentUser(s.services.Auth))
		}

		// User management routes (admin only)
		users := v1.Group("/users")
		users.Use(middleware.AuthRequired(), middleware.AdminRequired())
		{
			users.GET("", handler.ListUsers(s.services.User))
			users.POST("", handler.CreateUser(s.services.User))
			users.PUT("/:id/role", handler.UpdateUserRole(s.services.User))
			users.PUT("/:id/status", handler.UpdateUserStatus(s.services.User))
			users.DELETE("/:id", handler.DeleteUser(s.services.User))
		}

		// Registration toggle (admin only)
		settings := v1.Group("/settings")
		settings.Use(middleware.AuthRequired())
		{
			settings.GET("/registration", handler.GetRegistrationStatus(s.services.User))
			settings.PUT("/registration", middleware.AdminRequired(), handler.SetRegistrationStatus(s.services.User))
		}

		// Platform routes
		platforms := v1.Group("/platforms")
		platforms.Use(middleware.AuthRequired())
		{
			platforms.GET("", handler.ListPlatforms(s.services.Platform))
			platforms.POST("", handler.CreatePlatform(s.services.Platform))
			platforms.GET("/:id", handler.GetPlatform(s.services.Platform))
			platforms.PUT("/:id", handler.UpdatePlatform(s.services.Platform))
			platforms.DELETE("/:id", handler.DeletePlatform(s.services.Platform))
		}

		// Adapter routes
		adapters := v1.Group("/adapters")
		adapters.Use(middleware.AuthRequired())
		{
			adapters.GET("", handler.ListAdapters(s.services.Adapter))
			adapters.GET("/catalog", handler.ListAdapterCatalog(s.services.Catalog))
			adapters.POST("", handler.CreateAdapter(s.services.Adapter))
			adapters.GET("/:id", handler.GetAdapter(s.services.Adapter))
			adapters.PUT("/:id", handler.UpdateAdapter(s.services.Adapter))
			adapters.DELETE("/:id", handler.DeleteAdapter(s.services.Adapter))
			adapters.POST("/:id/start", handler.StartAdapter(s.services.Adapter))
			adapters.POST("/:id/stop", handler.StopAdapter(s.services.Adapter))
		}

		// Adapter instance routes
		instances := v1.Group("/instances")
		instances.Use(middleware.AuthRequired())
		{
			instances.GET("", handler.ListInstances(s.services.Instance))
			instances.POST("", handler.CreateInstance(s.services.Instance))
			instances.GET("/:id", handler.GetInstance(s.services.Instance))
			instances.PUT("/:id", handler.UpdateInstance(s.services.Instance))
			instances.DELETE("/:id", handler.DeleteInstance(s.services.Instance))
			instances.POST("/:id/start", handler.StartInstance(s.services.Instance))
			instances.POST("/:id/stop", handler.StopInstance(s.services.Instance))
			instances.GET("/:id/logs", handler.GetInstanceLogs(s.services.Instance))
			instances.GET("/:id/logs/heatmap", handler.GetInstanceLogHeatmap(s.services.Instance))
			instances.DELETE("/:id/logs", handler.ClearInstanceLogs(s.services.Instance))
			instances.GET("/:id/init", handler.GetInstanceInitStatus(s.services.Instance))
		}

		// Message routes
		messages := v1.Group("/messages")
		messages.Use(middleware.AuthRequired())
		{
			messages.GET("", handler.ListMessages(s.services.Message))
			messages.GET("/:id", handler.GetMessage(s.services.Message))
			messages.POST("/:id/ack", handler.AckMessage(s.services.Message))
		}

		// Adapter gateway (接入器) entity & connection routes
		adapterGateways := v1.Group("/adapter-gateways")
		adapterGateways.Use(middleware.AuthRequired())
		{
			adapterGateways.GET("", handler.ListGatewayAdapters(s.services.AdapterGateway))
			adapterGateways.POST("", handler.CreateGatewayAdapter(s.services.AdapterGateway))
			adapterGateways.GET("/:id", handler.GetGatewayAdapter(s.services.AdapterGateway))
			adapterGateways.PUT("/:id", handler.UpdateGatewayAdapter(s.services.AdapterGateway))
			adapterGateways.DELETE("/:id", handler.DeleteGatewayAdapter(s.services.AdapterGateway))
			adapterGateways.GET("/:id/connections", handler.ListAdapterConnectionsByAdapter(s.services.AdapterGateway))
		}

		adapterConnections := v1.Group("/adapter-connections")
		adapterConnections.Use(middleware.AuthRequired())
		{
			adapterConnections.GET("", handler.ListAdapterConnections(s.services.AdapterGateway))
		}

		// Command routes
		commands := v1.Group("/commands")
		commands.Use(middleware.AuthRequired())
		{
			commands.GET("", handler.ListCommands(s.services.Command))
			commands.POST("", handler.CreateCommand(s.services.Command))
			commands.GET("/:id", handler.GetCommand(s.services.Command))
		}

		// Route rules
		routes := v1.Group("/routes")
		routes.Use(middleware.AuthRequired())
		{
			routes.GET("", handler.ListRoutes(s.services.Route))
			routes.POST("", handler.CreateRoute(s.services.Route))
			routes.GET("/:id", handler.GetRoute(s.services.Route))
			routes.PUT("/:id", handler.UpdateRoute(s.services.Route))
			routes.DELETE("/:id", handler.DeleteRoute(s.services.Route))
		}

		// Documentation
		docs := v1.Group("/docs")
		docs.Use(middleware.AuthRequired())
		{
			docs.GET("", handler.ListDocs())
			docs.GET("/:key", handler.GetDoc())
		}

		// Form options (from YAML registry)
		options := v1.Group("/options")
		options.Use(middleware.AuthRequired())
		{
			options.GET("", handler.ListOptions(s.services.Options))
			options.GET("/:key", handler.GetOption(s.services.Options))
		}

		// System stats (storage usage, database size, etc.)
		stats := v1.Group("/stats")
		stats.Use(middleware.AuthRequired())
		{
			stats.GET("/storage", s.storageStats)
		}
	}

	// WebSocket routes
	ws := s.router.Group("/ws")
	{
		// Bridge (桥) reconnect endpoint. Inbound messages reported by bridges
		// are persisted AND fanned out to the adapter gateway (接入器).
		// The gateway is passed so outbound commands can be routed back to the
		// correct bridge instance.
		ws.GET("/adapter", handler.AdapterWebSocket(
			s.services.Adapter,
			s.services.Message,
			s.gateway,
			func(payload map[string]interface{}) {
				s.broadcastToAdapterGateway(payload)
			},
		))
		ws.GET("/app", handler.AppWebSocket(s.services.Message))

		// Adapter gateway (接入器) — external systems connect here with ?key=xxx
		// Default listen path is simply /ws (简洁).
		ws.GET("", gin.WrapH(s.gateway))

		// Backward-compatible alias: /ws/adapter-gateway?key=xxx
		ws.GET("/adapter-gateway", gin.WrapH(s.gateway))

		// Per-adapter listen path: /ws/adapter-gateway/:adapterID?key=xxx
		// The adapter ID + key are both verified by the gateway.
		ws.GET("/adapter-gateway/:adapterID", s.handleGatewayWithID())
	}

	// Register a custom listen path for each enabled server-mode adapter that
	// explicitly configured one (listen_path). Paths registered here are
	// unique per adapter, so no runtime route conflicts arise.
	if adapters, _, err := s.services.AdapterGateway.List(1000, 0); err == nil {
		for i := range adapters {
			a := &adapters[i]
			if a.Mode != "server" || !a.Enabled || a.Status != "active" {
				continue
			}
			path := a.ListenPath
			if path == "" {
				continue // default path /ws already handled
			}
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
			s.router.GET(path, gin.WrapH(s.gateway))
			logger.Info("Adapter gateway custom listen path registered",
				logger.String("adapter_id", a.ID),
				logger.String("path", path))
		}
	}
}

// handleGatewayWithID wires the adapter ID path parameter into the gateway.
func (s *Server) handleGatewayWithID() gin.HandlerFunc {
	return func(c *gin.Context) {
		adapterID := c.Param("adapterID")
		s.gateway.ServeHTTPWithID(c.Writer, c.Request, adapterID)
	}
}

// registerVersionCheck registers the version check endpoint.
func (s *Server) registerVersionCheck() {
	s.router.GET("/api/version", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"version": "1.0.0",
			"backend": "1.0.0",
			"frontend": "1.0.0",
		})
	})
}

// broadcastToAdapterGateway converts a bridge-reported inbound payload into a
// full ESPL v3 message envelope and fans it out to all connected adapter
// gateway (接入器) clients that match the message platform.
//
// The bridge may report either a flat payload (platform_id / conversation_id /
// sender_id / ... / message_chain / raw) or a full V3 envelope whose inner
// "payload" holds the actual message. In the latter case the inner payload is
// unwrapped so gateway clients receive a flat, consistent message payload.
func (s *Server) broadcastToAdapterGateway(payload map[string]interface{}) {
	if s.gateway == nil || payload == nil {
		return
	}

	// If the bridge reported a full V3 envelope, unwrap its inner payload so
	// the fan-out carries the flat message fields (message_content,
	// message_chain, raw, ...) directly.
	inner := payload
	if protocolVersion, _ := payload["protocol_version"].(string); protocolVersion == v3.Version {
		if p, ok := payload["payload"].(map[string]interface{}); ok {
			inner = p
		}
	}

	platform := getString(inner, "platform_id")
	if platform == "" {
		platform = getString(inner, "platform")
	}

	// Ensure instance_id is present on the inner payload for adapter gateway
	// routing.  If the bridge's websocket handler already injected it (see
	// AdapterWebSocket in handler/websocket.go), this is a no-op.
	//
	// The adapter gateway (接入器) and downstream frameworks (e.g. LangBot's
	// ESPL adapter) rely on instance_id to route outbound (reply) messages
	// back to the correct bridge instance.
	if _, exists := inner["instance_id"]; !exists {
		inner["instance_id"] = ""
	}
	if _, exists := inner["instance"]; !exists {
		inner["instance"] = inner["instance_id"]
	}

	envelope := map[string]interface{}{
		"protocol_version": "v3",
		"event_id":         models.GenerateID(),
		"trace_id":         models.GenerateTraceID(),
		"timestamp":        time.Now().UnixMilli(),
		"platform":         platform,
		"event_type":       "message.received",
		"payload":          inner,
	}

	s.gateway.BroadcastInbound(envelope)
}

// getString extracts a string field from a map.
func getString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// Start starts the HTTP server
func (s *Server) Start() error {
	logger.Info("Starting HTTP server")
	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	logger.Info("Stopping HTTP server")
	if s.gateway != nil {
		s.gateway.Close()
	}
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return err
	}
	return s.services.Close()
}

// healthCheck handles health check requests
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"version": s.cfg.Version,
	})
}

// storageStats returns database storage usage: DB file size, per-table row
// counts and the total log/data directory footprint.
func (s *Server) storageStats(c *gin.Context) {
	result := gin.H{
		"tables": gin.H{},
	}

	// 1) Database file size (SQLite) and per-table row counts.
	db := s.services.Repo.DB
	if db != nil {
		// Resolve the SQLite file path (DSN) to measure its size.
		var dbFile string
		if s.cfg.Database.Driver == "sqlite" {
			dbFile = s.cfg.Database.DSN
			if dbFile != "" {
				if fi, serr := os.Stat(dbFile); serr == nil {
					result["db_file"] = dbFile
					result["db_size_bytes"] = fi.Size()
				}
			}
		}

		// Per-table row counts.
		tables := map[string]interface{}{
			"users":               &models.User{},
			"platforms":           &models.Platform{},
			"adapter_packages":    &models.AdapterPackage{},
			"adapter_capabilities": &models.AdapterCapability{},
			"adapter_instances":   &models.AdapterInstance{},
			"adapter_sessions":    &models.AdapterSession{},
			"inbound_events":      &models.InboundEvent{},
			"outbound_commands":   &models.OutboundCommand{},
			"route_rules":         &models.RouteRule{},
			"audit_logs":          &models.AuditLog{},
			"adapters":            &models.Adapter{},
			"adapter_connections": &models.AdapterConnection{},
		}
		counts := gin.H{}
		for name, model := range tables {
			var cnt int64
			if db.Model(model).Count(&cnt).Error == nil {
				counts[name] = cnt
			}
		}
		result["tables"] = counts
	}

	// 2) Total data directory footprint (DB + logs + instances + deps).
	base := "data"
	total := int64(0)
	_ = filepath.Walk(base, func(_ string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !fi.IsDir() {
			total += fi.Size()
		}
		return nil
	})
	result["data_dir_bytes"] = total

	c.JSON(http.StatusOK, gin.H{"data": result})
}
