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
	Catalog  *AdapterCatalog

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

	// Initialize the adapter catalog by scanning adapters/*/adapter.yaml.
	catalog, err := NewAdapterCatalog(cfg.Adapter.AdaptersDir)
	if err != nil {
		return nil, err
	}
	services.Catalog = catalog
	// Let the runner resolve platform codes through the catalog, and mint a
	// service token (using the JWT secret) so Python adapters can call the API.
	services.Runner.SetCatalog(catalog)
	services.Runner.SetJWTSecret(cfg.JWT.Secret)

	// Initialize the instance sandbox manager (data/instances/<id>/adapter).
	dirs := NewInstanceDirManager(cfg.Adapter.AdaptersDir)

	// Initialize the dependency installer and attach it to the runner and the
	// instance service so new instances auto-install adapter dependencies.
	installer := NewDependencyInstaller(cfg.Adapter.PythonBin, cfg.Adapter.AdaptersDir)
	services.Runner.SetInstaller(installer)
	services.Instance.SetInstaller(installer)
	services.Runner.SetDirManager(dirs)
	services.Instance.SetDirManager(dirs)

	// Attach the runner to InstanceService for WebUI start/stop.
	services.Instance.SetRunner(services.Runner)
	// Attach the runner & instance repo to AdapterService so deleting an adapter
	// stops its running instances.
	services.Adapter.SetRunner(services.Runner)
	services.Adapter.SetInstanceRepository(repos.Instance)

	// Reconcile instances left in "initializing" state (e.g. server restarted
	// mid-install): re-run dependency installation for them.
	services.reconcileInitializing()

	logger.Info("Services initialized")
	return services, nil
}

// reconcileInitializing re-runs dependency installation for instances left in
// "initializing" state (e.g. server restarted mid-install), and ensures every
// existing instance has a sandboxed adapter copy under data/instances/<id>/
// (missing copies are created for pre-existing instances).
func (s *Services) reconcileInitializing() {
	if s.Instance == nil || s.Runner == nil {
		return
	}
	instances, _, err := s.Instance.List(1000, 0)
	if err != nil {
		logger.Warn("Failed to list instances for initializing reconcile", logger.String("error", err.Error()))
		return
	}
	count := 0
	for _, inst := range instances {
		// Resolve the source platform code (via the runner's catalog lookup).
		platformCode, perr := s.Runner.resolvePlatformCode(inst.ID)
		if perr != nil {
			logger.Warn("Failed to resolve platform code during reconcile",
				logger.String("instance_id", inst.ID),
				logger.String("error", perr.Error()))
			continue
		}

		// Ensure a sandbox copy exists; copy it if missing.
		created := false
		if s.Instance.dirs != nil {
			adapterDir, createdNow, cerr := s.Instance.dirs.EnsureSandbox(inst.ID, platformCode)
			created = createdNow
			if cerr != nil {
				logger.Warn("Failed to ensure instance sandbox",
					logger.String("instance_id", inst.ID),
					logger.String("error", cerr.Error()))
				continue
			}
			if created {
				logger.Info("Created missing instance sandbox",
					logger.String("instance_id", inst.ID),
					logger.String("adapter_dir", adapterDir))
			}
		}

		// Trigger dependency installation when:
		//   - the instance is still "initializing" (restart mid-install), or
		//   - the sandbox was freshly created for a pre-existing instance
		//     (it has never had its dependencies installed before).
		if s.Instance.installer != nil && s.Instance.dirs != nil {
			adapterDir := s.Instance.dirs.AdapterDir(inst.ID)
			_, stateExists := s.Instance.installer.GetState(inst.ID)
			if inst.Status == "initializing" || created || !stateExists {
				s.Instance.installer.InstallDependencies(inst.ID, adapterDir, s.Instance.updateInstanceStatus)
				count++
			}
		}
	}
	if count > 0 {
		logger.Info("Reconciled initializing instances", logger.Int("count", count))
	}
}

// Close closes all service resources
func (s *Services) Close() error {
	// Stop all managed Python adapter processes.
	if s.Runner != nil {
		s.Runner.StopAll()
	}
	return s.repos.Close()
}
