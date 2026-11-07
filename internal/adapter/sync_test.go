package adapter

import (
	"testing"

	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func newTestRepos(t *testing.T) (*repository.PlatformRepository, *repository.AdapterPackageRepository, *repository.AdapterRepository) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.Platform{}, &models.AdapterPackage{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return repository.NewPlatformRepository(db), repository.NewAdapterPackageRepository(db), repository.NewAdapterRepository(db)
}

func TestSyncBuiltinsIsIdempotent(t *testing.T) {
	platforms, adapters, packages := newTestRepos(t)

	for i := 0; i < 2; i++ {
		if err := SyncBuiltins(platforms, adapters); err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
	}

	all, total, err := platforms.List(100, 0)
	if err != nil {
		t.Fatalf("list platforms: %v", err)
	}
	if total != 12 {
		t.Fatalf("platforms = %d, want 12 with no duplicates", total)
	}
	for _, p := range all {
		if p.Code == "" {
			t.Fatalf("platform %q has no code", p.ID)
		}
	}

	pkgs, total, err := packages.List(100, 0)
	if err != nil {
		t.Fatalf("list adapters: %v", err)
	}
	if total != 12 {
		t.Fatalf("adapters = %d, want 12", total)
	}
	for _, pkg := range pkgs {
		if pkg.Manifest == "" {
			t.Fatalf("adapter %q has no manifest", pkg.ID)
		}
		if pkg.Status != "active" {
			t.Fatalf("adapter %q status = %q", pkg.ID, pkg.Status)
		}
	}
}

func TestSyncBuiltinsSkipsUpcoming(t *testing.T) {
	platforms, adapters, _ := newTestRepos(t)
	if err := SyncBuiltins(platforms, adapters); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if _, err := platforms.FindByCode("jimeng"); err == nil {
		t.Fatal("jimeng is upcoming and should not be synced")
	}
}
