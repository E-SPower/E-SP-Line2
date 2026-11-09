package adapter

import (
	"time"

	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/internal/platformspec"
	"github.com/e-spl/e-sp-line2/internal/repository"
	"github.com/e-spl/e-sp-line2/pkg/logger"
)

// SyncBuiltins puts the built-in platform adapters into the database.
//
// 迁移过来的平台来自注册表，靠这一段在启动时落到 platforms / adapter_packages 两张表，
// 面板和现有的实例、路由接口才能按平台工作。重复执行是安全的：按 code / id 判断，已存在就只更新。
func SyncBuiltins(platforms *repository.PlatformRepository, adapters *repository.AdapterPackageRepository) error {
	for _, p := range platformspec.Active() {
		platform, err := ensurePlatform(platforms, p)
		if err != nil {
			return err
		}
		if err := ensureAdapter(adapters, platform.ID, p); err != nil {
			return err
		}
	}
	return nil
}

func ensurePlatform(repo *repository.PlatformRepository, p *platformspec.Platform) (*models.Platform, error) {
	existing, err := repo.FindByCode(p.ID)
	if err == nil && existing != nil {
		if existing.Name != p.Name || existing.Description != description(p) {
			existing.Name = p.Name
			existing.Description = description(p)
			existing.UpdatedAt = time.Now()
			if err := repo.Update(existing); err != nil {
				return nil, err
			}
		}
		return existing, nil
	}

	platform := &models.Platform{
		ID:          models.GenerateID(),
		Name:        p.Name,
		Code:        p.ID,
		Description: description(p),
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := repo.Create(platform); err != nil {
		return nil, err
	}
	return platform, nil
}

func ensureAdapter(repo *repository.AdapterPackageRepository, platformID string, p *platformspec.Platform) error {
	info := BuiltinAdapterInfo(p)

	existing, err := repo.FindByID(info.ID)
	if err == nil && existing != nil {
		existing.Name = info.Name
		existing.Version = info.Version
		existing.Manifest = syncManifest(info)
		existing.UpdatedAt = time.Now()
		return repo.Update(existing)
	}

	manifest, err := PlatformManifest(p.ID)
	if err != nil {
		return err
	}

	pkg := &models.AdapterPackage{
		ID:              info.ID,
		PlatformID:      platformID,
		Name:            info.Name,
		Version:         info.Version,
		RuntimeType:     info.RuntimeType,
		ProtocolVersion: info.ProtocolVersion,
		Status:          info.Status,
		Manifest:        manifest,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if err := repo.Create(pkg); err != nil {
		return err
	}

	logger.Info("Built-in adapter synced", logger.String("id", info.ID), logger.String("platform", p.ID))
	return nil
}

func syncManifest(info *AdapterInfo) string {
	manifest, err := PlatformManifest(info.PlatformID)
	if err != nil {
		return ""
	}
	return manifest
}

func description(p *platformspec.Platform) string {
	if p.DM != "" {
		return p.Name + " adapter (" + p.DM + ")"
	}
	return p.Name + " adapter"
}
