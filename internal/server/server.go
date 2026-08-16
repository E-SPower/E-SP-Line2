package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/e-spl/e-sp-line2/internal/config"
	"github.com/e-spl/e-sp-line2/internal/handler"
	"github.com/e-spl/e-sp-line2/internal/middleware"
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

	s.setupRoutes()

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
	}

	// WebSocket routes
	ws := s.router.Group("/ws")
	{
		ws.GET("/adapter", handler.AdapterWebSocket(s.services.Adapter, s.services.Message))
		ws.GET("/app", handler.AppWebSocket(s.services.Message))
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	logger.Info("Starting HTTP server")
	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	logger.Info("Stopping HTTP server")
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
