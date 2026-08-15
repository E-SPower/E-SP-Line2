package service

import (
	"fmt"

	"github.com/e-spl/e-sp-line2/internal/config"
	"github.com/e-spl/e-sp-line2/internal/repository"
	"github.com/e-spl/e-sp-line2/pkg/logger"
)

// Services holds all service instances
type Services struct {
	Auth     *AuthService
	Platform *PlatformService
	Adapter  *AdapterService
	Instance *InstanceService
	Message  *MessageService
	Command  *CommandService
	Route    *RouteService
	Options  *OptionService
	Runner   *PythonRunner

	repos *repository.Repositories
}

// NewServices creates all service instances
func NewServices(cfg *config.Config) (*Services, error) {
	// Initialize repositories
	repos, err := repository.NewRepositories(cfg)
	if err != nil {
		return nil, err
	}

	services := &Services{
		repos: repos,
	}

	// Initialize services
	services.Auth = NewAuthService(cfg, repos.User)
	services.Platform = NewPlatformService(repos.Platform)
	services.Adapter = NewAdapterService(repos.Adapter, repos.AdapterPackage, repos.AdapterSession)
	services.Instance = NewInstanceService(repos.Instance, repos.AdapterSession)
	services.Message = NewMessageService(repos.InboundEvent, repos.Instance)
	services.Command = NewCommandService(repos.OutboundCommand)
	services.Route = NewRouteService(repos.RouteRule)

	// Initialize form options from the YAML registry file
	optionsPath := cfg.FormOptionsPath()
	options, err := NewOptionService(optionsPath)
	if err != nil {
		return nil, err
	}
	services.Options = options

	// Initialize the Python adapter process manager (WebUI start/stop control).
	backendURL := fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port)
	if cfg.Host == "0.0.0.0" {
		backendURL = fmt.Sprintf("http://127.0.0.1:%d", cfg.Port)
	}
	services.Runner = NewPythonRunner(
		cfg.Adapter.PythonBin,
		cfg.Adapter.AdaptersDir,
		backendURL,
		cfg.Adapter.AutoRestart,
		repos.Instance,
		repos.Adapter,
		repos.Platform,
	)
	// Attach the runner to InstanceService for WebUI start/stop.
	services.Instance.SetRunner(services.Runner)

	logger.Info("Services initialized")
	return services, nil
}

// Close closes all service resources
func (s *Services) Close() error {
	// Stop all managed Python adapter processes.
	if s.Runner != nil {
		s.Runner.StopAll()
	}
	return s.repos.Close()
}
