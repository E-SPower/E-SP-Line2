package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/e-spl/e-sp-line2/internal/repository"
	"github.com/e-spl/e-sp-line2/pkg/logger"
)

// PythonRunner manages the lifecycle of Python adapter processes.
// 它让用户可以在 WebUI 中直接启动/停止闲鱼、淘宝等 Python 接入器，
// 而无需单独在终端启动。
type PythonRunner struct {
	// process mapping: instanceID -> *exec.Cmd
	mu      sync.Mutex
	procs   map[string]*exec.Cmd
	python  string
	dir     string
	autoRestart bool
	backendURL string

	instances *repository.InstanceRepository
	adapters  *repository.AdapterRepository
	platforms *repository.PlatformRepository
}

// NewPythonRunner creates a Python process manager.
//
// python: Python executable (e.g. python3)
// adaptersDir: root dir containing adapters/<platform>/main.py
// backendURL: the backend base URL passed to Python adapters (e.g. http://localhost:8080)
// autoRestart: restart crashed processes automatically
func NewPythonRunner(
	python, adaptersDir, backendURL string,
	autoRestart bool,
	instances *repository.InstanceRepository,
	adapters *repository.AdapterRepository,
	platforms *repository.PlatformRepository,
) *PythonRunner {
	return &PythonRunner{
		procs:       make(map[string]*exec.Cmd),
		python:      python,
		dir:         adaptersDir,
		autoRestart: autoRestart,
		backendURL:  backendURL,
		instances:   instances,
		adapters:    adapters,
		platforms:   platforms,
	}
}

// IsRunning reports whether the given instance has a live Python process.
func (r *PythonRunner) IsRunning(instanceID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	cmd, ok := r.procs[instanceID]
	if !ok || cmd == nil || cmd.Process == nil {
		return false
	}
	// On Linux, signal 0 tests whether the process exists.
	return cmd.Process.Signal(syscall.Signal(0)) == nil
}

// resolvePlatformCode resolves the platform code (directory name) for an instance.
// chain: instance -> adapter -> platform -> code
func (r *PythonRunner) resolvePlatformCode(instanceID string) (string, error) {
	instance, err := r.instances.FindByID(instanceID)
	if err != nil {
		return "", fmt.Errorf("instance not found: %w", err)
	}
	adapter, err := r.adapters.FindByID(instance.AdapterID)
	if err != nil {
		return "", fmt.Errorf("adapter not found: %w", err)
	}
	platform, err := r.platforms.FindByID(adapter.PlatformID)
	if err != nil {
		return "", fmt.Errorf("platform not found: %w", err)
	}
	code := strings.ToLower(strings.TrimSpace(platform.Code))
	if code == "" {
		return "", errors.New("platform code is empty")
	}
	return code, nil
}

// Start launches the Python adapter process for the given instance.
func (r *PythonRunner) Start(instanceID string) error {
	r.mu.Lock()
	if cmd, ok := r.procs[instanceID]; ok && cmd != nil && cmd.Process != nil {
		if cmd.Process.Signal(syscall.Signal(0)) == nil {
			r.mu.Unlock()
			return errors.New("instance already running")
		}
	}
	r.mu.Unlock()

	// Resolve the Python script for this instance's platform.
	platformCode, err := r.resolvePlatformCode(instanceID)
	if err != nil {
		return err
	}
	workDir := filepath.Join(r.dir, platformCode)
	script := filepath.Join(workDir, "main.py")
	if _, err := os.Stat(script); err != nil {
		return fmt.Errorf("python adapter script not found: %s (%v)", script, err)
	}
	_ = script // script exists check only; run relative to cmd.Dir

	// Validate the instance has a cookie configured.
	instance, err := r.instances.FindByID(instanceID)
	if err != nil {
		return err
	}
	var cfg map[string]interface{}
	if instance.Config != "" {
		_ = json.Unmarshal([]byte(instance.Config), &cfg)
	}
	if cookie, _ := cfg["cookie"].(string); strings.TrimSpace(cookie) == "" {
		return errors.New("instance has no cookie configured; please fill it in the WebUI")
	}

	// Build the command: python main.py --instance-id X --backend URL
	// Run inside the platform adapter directory, so the script path is relative.
	args := []string{"main.py", "--instance-id", instanceID, "--backend", r.backendURL}
	cmd := exec.Command(r.python, args...)
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start python adapter: %w", err)
	}

	r.mu.Lock()
	r.procs[instanceID] = cmd
	r.mu.Unlock()

	logger.Info("Python adapter process started",
		logger.String("instance_id", instanceID),
		logger.String("platform", platformCode),
		logger.Int("pid", cmd.Process.Pid))

	// Monitor process exit; optionally auto-restart.
	go r.monitor(instanceID, cmd, platformCode)

	return nil
}

// Stop terminates the Python adapter process for the given instance.
func (r *PythonRunner) Stop(instanceID string) error {
	r.mu.Lock()
	cmd, ok := r.procs[instanceID]
	r.mu.Unlock()
	if !ok || cmd == nil || cmd.Process == nil {
		return errors.New("instance is not running")
	}

	// Try graceful termination, then force kill.
	_ = cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		_, _ = cmd.Process.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		<-done
	}

	r.mu.Lock()
	delete(r.procs, instanceID)
	r.mu.Unlock()

	// Update instance status to stopped.
	if err := r.updateInstanceStatus(instanceID, "stopped"); err != nil {
		logger.Warn("Failed to update instance status", logger.String("error", err.Error()))
	}

	logger.Info("Python adapter process stopped", logger.String("instance_id", instanceID))
	return nil
}

// StopAll terminates all managed Python processes (called on server shutdown).
func (r *PythonRunner) StopAll() {
	r.mu.Lock()
	ids := make([]string, 0, len(r.procs))
	for id := range r.procs {
		ids = append(ids, id)
	}
	r.mu.Unlock()

	for _, id := range ids {
		_ = r.Stop(id)
	}
}

// monitor watches a Python process and updates status when it exits.
func (r *PythonRunner) monitor(instanceID string, cmd *exec.Cmd, platformCode string) {
	err := cmd.Wait()

	r.mu.Lock()
	// Only clear if it's still the same command.
	if cur, ok := r.procs[instanceID]; ok && cur == cmd {
		delete(r.procs, instanceID)
	}
	r.mu.Unlock()

	if err != nil {
		logger.Warn("Python adapter process exited",
			logger.String("instance_id", instanceID),
			logger.String("platform", platformCode),
			logger.String("error", err.Error()))
	}

	_ = r.updateInstanceStatus(instanceID, "stopped")

	// Auto-restart crashed processes.
	if r.autoRestart && err != nil {
		logger.Info("Restarting python adapter process",
			logger.String("instance_id", instanceID))
		if serr := r.Start(instanceID); serr != nil {
			logger.Error("Failed to restart python adapter",
				logger.String("instance_id", instanceID),
				logger.String("error", serr.Error()))
		}
	}
}

// updateInstanceStatus updates the status field of an instance.
func (r *PythonRunner) updateInstanceStatus(instanceID, status string) error {
	instance, err := r.instances.FindByID(instanceID)
	if err != nil {
		return err
	}
	instance.Status = status
	return r.instances.Update(instance)
}
