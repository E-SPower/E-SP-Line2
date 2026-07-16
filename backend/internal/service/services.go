package service

import (
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
	services.Adapter = NewAdapterService(repos.Adapter, repos.AdapterPackage)
	services.Instance = NewInstanceService(repos.Instance, repos.AdapterSession)
	services.Message = NewMessageService(repos.InboundEvent)
	services.Command = NewCommandService(repos.OutboundCommand)
	services.Route = NewRouteService(repos.RouteRule)

	logger.Info("Services initialized")
	return services, nil
}

// Close closes all service resources
func (s *Services) Close() error {
	return s.repos.Close()
}
