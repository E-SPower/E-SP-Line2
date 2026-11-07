package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e-spl/e-sp-line2/internal/config"
	"github.com/e-spl/e-sp-line2/internal/models"
	"github.com/e-spl/e-sp-line2/internal/repository"
	"github.com/e-spl/e-sp-line2/pkg/logger"
)

// newTestRunner builds a runner with a real (sqlite) instance repository and a
// sandbox manager rooted at a temp dir, so Start's plumbing (sandbox lookup,
// integrity check, status writes) runs for real.
func newTestRunner(t *testing.T, adaptersDir string) (*PythonRunner, *repository.InstanceRepository) {
	t.Helper()
	logger.Init("error")
	cfg := &config.Config{Database: config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "t.db")}}
	repos, err := repository.NewRepositories(cfg)
	if err != nil {
		t.Fatalf("repositories: %v", err)
	}
	t.Cleanup(func() { _ = repos.Close() })
	runner := NewPythonRunner("python3", adaptersDir, "http://127.0.0.1:1", false, repos.Instance, repos.Adapter, repos.Platform)
	return runner, repos.Instance
}

// seedInstance stores an instance row so Start can resolve it.
func seedInstance(t *testing.T, repo *repository.InstanceRepository, adapterID string) *models.AdapterInstance {
	t.Helper()
	inst := &models.AdapterInstance{
		ID:         models.GenerateID(),
		AdapterID:  adapterID,
		PlatformID: "xianyu",
		Name:       "t",
		Config:     `{}`,
		Status:     "stopped",
	}
	if err := repo.Create(inst); err != nil {
		t.Fatalf("seed instance: %v", err)
	}
	return inst
}

// writeAdapter lays down adapters/<dir>/adapter.yaml (+ entry) and returns the
// adapters dir. The dir name intentionally differs from platform_code in the
// catbus test to pin the source-dir resolution.
func writeAdapter(t *testing.T, dir, adapterID, runtimeType, entry, yamlExtra string) string {
	t.Helper()
	adaptersDir := t.TempDir()
	a := filepath.Join(adaptersDir, dir)
	if err := os.MkdirAll(a, 0o755); err != nil {
		t.Fatal(err)
	}
	if entry != "" {
		if err := os.WriteFile(filepath.Join(a, entry), []byte("// stub\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	yaml := "id: " + adapterID + "\nplatform_code: xianyu\nname: t\nversion: 1\nruntime_type: " + runtimeType + "\n" + yamlExtra
	if err := os.WriteFile(filepath.Join(a, "adapter.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	return adaptersDir
}

// TestPythonLaunchUnchanged pins the legacy bridge command line: adding the
// runtime dispatch must not change a single byte for python adapters.
func TestPythonLaunchUnchanged(t *testing.T) {
	adaptersDir := writeAdapter(t, "xianyu", "xianyu-adapter", "python", "main.py", "")
	runner, _ := newTestRunner(t, adaptersDir)
	inst := &models.AdapterInstance{ID: "i1", AdapterID: "xianyu-adapter", Config: `{}`}

	plan, err := runner.buildLaunch(inst, RuntimePython, RuntimeSpec{}, adaptersDir, map[string]interface{}{},
		[]string{"main.py", "--instance-id", "i1", "--backend", "http://b"})
	if err != nil {
		t.Fatalf("buildLaunch: %v", err)
	}
	if plan.bin != "python3" {
		t.Fatalf("python adapter must launch %q, got %q", "python3", plan.bin)
	}
	want := "main.py --instance-id i1 --backend http://b"
	if got := strings.Join(plan.args, " "); got != want {
		t.Fatalf("python args = %q, want %q", got, want)
	}
	if len(plan.env) != 0 {
		t.Fatalf("python launch must not invent env, got %v", plan.env)
	}
}

// TestStandaloneLaunch pins the standalone command line: entry, instance flags,
// config file (0600 next to, not inside, the sandbox) and ESP_* env.
func TestStandaloneLaunch(t *testing.T) {
	adaptersDir := writeAdapter(t, "catbus", "catbus-adapter", RuntimeStandalone, "main.mjs",
		"runtime:\n  entry: main.mjs\n  interpreter: node\n")
	runner, repo := newTestRunner(t, adaptersDir)
	runner.SetDirManager(NewInstanceDirManager(adaptersDir))
	inst := seedInstance(t, repo, "catbus-adapter")
	if _, err := runner.dirs.CopyAdapter(inst.ID, "catbus"); err != nil {
		t.Fatalf("sandbox: %v", err)
	}

	cfg := map[string]interface{}{"catbus_bin": "/opt/catbus", "cookie": "a=b"}
	plan, err := runner.buildLaunch(inst, RuntimeStandalone, RuntimeSpec{Interpreter: "node", Entry: "main.mjs"},
		runner.dirs.AdapterDir(inst.ID), cfg,
		[]string{"--instance-id", inst.ID, "--backend", "http://b"})
	if err != nil {
		t.Fatalf("buildLaunch: %v", err)
	}
	if plan.bin != "node" {
		t.Fatalf("standalone must use the declared interpreter, got %q", plan.bin)
	}
	joined := strings.Join(plan.args, " ")
	if !strings.Contains(joined, "main.mjs --instance-id "+inst.ID) {
		t.Fatalf("args missing entry/instance: %q", joined)
	}
	if !strings.Contains(joined, "--config ") {
		t.Fatalf("standalone must receive --config, got %q", joined)
	}
	if strings.Contains(joined, "a=b") {
		t.Fatalf("config must go through the file, not argv: %q", joined)
	}

	// The config file lives next to the sandbox copy and is 0600.
	cfgPath := filepath.Join(filepath.Dir(runner.dirs.AdapterDir(inst.ID)), "config.runtime.json")
	info, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatalf("config file not written: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("config file mode = %o, want 600", perm)
	}
	data, _ := os.ReadFile(cfgPath)
	var round map[string]interface{}
	if err := json.Unmarshal(data, &round); err != nil {
		t.Fatalf("config file is not the instance config: %v", err)
	}
	if round["catbus_bin"] != "/opt/catbus" {
		t.Fatalf("config file lost catbus_bin: %v", round)
	}

	env := strings.Join(plan.env, "\n")
	for _, want := range []string{"ESP_ADAPTER_RUNTIME=standalone", "ESP_INSTANCE_ID=" + inst.ID, "CATBUS_BIN=/opt/catbus"} {
		if !strings.Contains(env, want) {
			t.Fatalf("env missing %q: %v", want, plan.env)
		}
	}
}

// TestStandaloneRequiresEntry covers the adapter.yaml validation that keeps a
// half-declared standalone adapter from reaching the first start.
func TestStandaloneRequiresEntry(t *testing.T) {
	if err := validateRuntimeSpec("a.yaml", RuntimeStandalone, RuntimeSpec{}); err == nil {
		t.Fatal("standalone without runtime.entry must be rejected")
	}
	if err := validateRuntimeSpec("a.yaml", RuntimeStandalone, RuntimeSpec{Entry: "../x.mjs"}); err == nil {
		t.Fatal("runtime.entry escaping the adapter dir must be rejected")
	}
	if err := validateRuntimeSpec("a.yaml", RuntimeStandalone, RuntimeSpec{Entry: "main.mjs"}); err != nil {
		t.Fatalf("valid spec rejected: %v", err)
	}
	if err := validateRuntimeSpec("a.yaml", RuntimePython, RuntimeSpec{}); err != nil {
		t.Fatalf("python adapter must not need runtime.entry: %v", err)
	}
}

// TestCatalogRejectsUnknownRuntime makes a typo'd runtime_type a scan error
// instead of a silent fallback to the python launcher.
func TestCatalogRejectsUnknownRuntime(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := "id: x\nplatform_code: x\nname: x\nversion: 1\nruntime_type: python3\n"
	if err := os.WriteFile(filepath.Join(dir, "x", "adapter.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewAdapterCatalog(dir); err == nil {
		t.Fatal("unknown runtime_type must fail the catalog scan")
	}
}

// TestSourceDirPrefersCatalogDir: an adapter directory that differs from the
// platform code must be the sandbox source (adapters/catbus serving xianyu).
func TestSourceDirPrefersCatalogDir(t *testing.T) {
	adaptersDir := writeAdapter(t, "catbus", "catbus-adapter", "standalone", "main.mjs", "runtime:\n  entry: main.mjs\n")
	runner, repo := newTestRunner(t, adaptersDir)
	catalog, err := NewAdapterCatalog(adaptersDir)
	if err != nil {
		t.Fatal(err)
	}
	runner.SetCatalog(catalog)
	inst := seedInstance(t, repo, "catbus-adapter")

	dir, err := runner.resolveSourceDir(inst.ID)
	if err != nil {
		t.Fatal(err)
	}
	if dir != "catbus" {
		t.Fatalf("source dir = %q, want catbus", dir)
	}

	// The legacy resolver still answers the platform code for python adapters.
	py := writeAdapter(t, "xianyu", "xianyu-adapter", "python", "main.py", "")
	pyRunner, pyRepo := newTestRunner(t, py)
	pyCatalog, _ := NewAdapterCatalog(py)
	pyRunner.SetCatalog(pyCatalog)
	pyInst := seedInstance(t, pyRepo, "xianyu-adapter")
	dir, err = pyRunner.resolveSourceDir(pyInst.ID)
	if err != nil {
		t.Fatal(err)
	}
	if dir != "xianyu" {
		t.Fatalf("python source dir = %q, want xianyu", dir)
	}
}
