package repository

import (
	"fmt"

	"github.com/e-spl/e-sp-line2/internal/config"
	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Repositories holds all repository instances
type Repositories struct {
	DB              *gorm.DB
	User            *UserRepository
	Platform        *PlatformRepository
	Adapter         *AdapterRepository
	AdapterPackage  *AdapterPackageRepository
	Instance        *InstanceRepository
	AdapterSession  *AdapterSessionRepository
	InboundEvent    *InboundEventRepository
	OutboundCommand *OutboundCommandRepository
	RouteRule       *RouteRuleRepository
}

// NewRepositories creates all repository instances
func NewRepositories(cfg *config.Config) (*Repositories, error) {
	db, err := initDB(cfg)
	if err != nil {
		return nil, err
	}

	// Auto migrate models
	if err := db.AutoMigrate(
		&models.User{},
		&models.Platform{},
		&models.AdapterPackage{},
		&models.AdapterCapability{},
		&models.AdapterInstance{},
		&models.AdapterSession{},
		&models.InboundEvent{},
		&models.OutboundCommand{},
		&models.RouteRule{},
		&models.AuditLog{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	repos := &Repositories{
		DB:              db,
		User:            NewUserRepository(db),
		Platform:        NewPlatformRepository(db),
		Adapter:         NewAdapterRepository(db),
		AdapterPackage:  NewAdapterPackageRepository(db),
		Instance:        NewInstanceRepository(db),
		AdapterSession:  NewAdapterSessionRepository(db),
		InboundEvent:    NewInboundEventRepository(db),
		OutboundCommand: NewOutboundCommandRepository(db),
		RouteRule:       NewRouteRuleRepository(db),
	}

	logger.Info("Repositories initialized")
	return repos, nil
}

// Close closes the database connection
func (r *Repositories) Close() error {
	sqlDB, err := r.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// initDB initializes the database connection
func initDB(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.Database.Driver {
	case "postgres":
		dialector = postgres.Open(cfg.Database.GetDSN())
	case "sqlite":
		dialector = sqlite.Open(cfg.Database.DSN)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Database.Driver)
	}

	logLevel := gormlogger.Silent
	if cfg.LogLevel == "debug" {
		logLevel = gormlogger.Info
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: gormlogger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}
