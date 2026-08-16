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
	"github.com/golang-jwt/jwt/v5"
)

// PythonRunner manages the lifecycle of Python adapter processes.
// 它让用户可以在 WebUI 中直接启动/停止闲鱼、淘宝等 Python 接入器，
// 而无需单独在终端启动。
type PythonRunner struct {
	// process mapping: instanceID -> *exec.Cmd
	mu      sync.Mutex
	procs   map[string]*exec.Cmd
	stopping map[string]bool // instances marked for intentional stop
	restarts map[string]int  // consecutive crash count per instance
	python  string
	dir     string
	autoRestart bool
	backendURL string
	jwtSecret  string

	// installer installs adapter Python dependencies on instance creation.
	installer *DependencyInstaller

	// dirs manages per-instance sandbox copies of adapter code.
	dirs *InstanceDirManager

	instances *repository.InstanceRepository
	adapters  *repository.AdapterRepository
	platforms *repository.PlatformRepository
	catalog   *AdapterCatalog
}

// maxAutoRestarts caps how many consecutive crashes are auto-restarted before
// giving up (prevents log spam when dependencies are missing, etc.).
const maxAutoRestarts = 5

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
		stopping:    make(map[string]bool),
		restarts:    make(map[string]int),
		python:      python,
		dir:         adaptersDir,
		autoRestart: autoRestart,
		backendURL:  backendURL,
		instances:   instances,
		adapters:    adapters,
		platforms:   platforms,
	}
}

// SetCatalog attaches the adapter catalog used to resolve platform codes.
func (r *PythonRunner) SetCatalog(catalog *AdapterCatalog) {
	r.catalog = catalog
}

// SetInstaller attaches the dependency installer used on instance creation.
func (r *PythonRunner) SetInstaller(installer *DependencyInstaller) {
	r.installer = installer
}

// SetDirManager attaches the instance sandbox manager used at start time.
func (r *PythonRunner) SetDirManager(dirs *InstanceDirManager) {
	r.dirs = dirs
}

// SetJWTSecret sets the JWT secret used to mint a service token for the
// Python adapter process so it can call protected backend APIs.
func (r *PythonRunner) SetJWTSecret(secret string) {
	r.jwtSecret = secret
}

// serviceToken mints a short-lived JWT that lets the Python adapter process
// authenticate against protected backend APIs (e.g. fetching instance config).
func (r *PythonRunner) serviceToken() (string, error) {
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "adapter-runner",
		"role": "service",
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	})
	return claims.SignedString([]byte(r.jwtSecret))
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

// logDir returns the directory where per-instance logs are stored.
func (r *PythonRunner) logDir() string {
	return filepath.Join(r.dir, "..", "data", "logs")
}

// logPath returns the log file path for an instance.
func (r *PythonRunner) logPath(instanceID string) string {
	return filepath.Join(r.logDir(), instanceID+".log")
}

// openLogFile creates (append) the per-instance log file.
func (r *PythonRunner) openLogFile(instanceID string) (*os.File, error) {
	dir := r.logDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return os.OpenFile(r.logPath(instanceID), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
}

// ReadLog returns the (possibly filtered) log of an instance.
//
//   - maxLines <= 0  : return the FULL log (no tail truncation)
//   - maxLines  > 0  : return only the last maxLines lines
//   - level          : filter lines by log level. Allowed values are
//                      "debug", "info", "warning", "error" (case-insensitive).
//                      Empty or "all" returns every line. Only the chosen level
//                      and higher-severity lines are kept (e.g. "warning"
//                      keeps warning + error).
//
// If the file does not exist, it returns an empty log (not an error).
func (r *PythonRunner) ReadLog(instanceID string, maxLines int, level string) (string, error) {
	path := r.logPath(instanceID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	lines := strings.Split(string(data), "\n")
	// Trim trailing empty line.
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	// Filter by minimum log level.
	level = strings.ToLower(strings.TrimSpace(level))
	if level != "" && level != "all" {
		keep := severityAtLeast(level)
		filtered := make([]string, 0, len(lines))
		for _, ln := range lines {
			if logLineMatches(ln, keep) {
				filtered = append(filtered, ln)
			}
		}
		lines = filtered
	}

	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return strings.Join(lines, "\n"), nil
}

// severityAtLeast returns the set of level names that satisfy "at least level".
func severityAtLeast(level string) map[string]bool {
	order := []string{"debug", "info", "warning", "error"}
	start := -1
	for i, l := range order {
		if l == level {
			start = i
			break
		}
	}
	if start < 0 {
		return nil
	}
	set := make(map[string]bool)
	for _, l := range order[start:] {
		set[l] = true
	}
	return set
}

// logLineMatches reports whether a log line should be kept given the allowed
// levels. Lines without a recognized level marker are always kept (e.g.
// multi-line tracebacks).
func logLineMatches(line string, keep map[string]bool) bool {
	if keep == nil {
		return true
	}
	// Matches patterns like " | INFO     | ..." (our loguru format) and
	// "INFO:" / "WARNING:" / "ERROR:" / "DEBUG:" (standard).
	lower := strings.ToLower(line)
	for lvl := range keep {
		if strings.Contains(lower, " | "+lvl+" ") ||
			strings.HasPrefix(lower, lvl+":") ||
			strings.Contains(lower, " "+lvl+":") {
			return true
		}
	}
	// Traceback / continuation lines have no level marker; keep them so we
	// don't break stack traces.
	return !hasLevelMarker(lower)
}

// hasLevelMarker reports whether the line contains any known level marker.
func hasLevelMarker(line string) bool {
	for _, lvl := range []string{"debug", "info", "warning", "error"} {
		if strings.Contains(line, " | "+lvl+" ") ||
			strings.HasPrefix(line, lvl+":") ||
			strings.Contains(line, " "+lvl+":") {
			return true
		}
	}
	return false
}

// resolvePlatformCode resolves the platform code (directory name) for an instance.
// chain: instance -> adapter -> platform code.
// Since adapters are created directly with a message platform code
// (e.g. "xianyu", "taobao"), the adapter's PlatformID usually IS the code.
// If it doesn't correspond to an adapters/<code>/main.py directory, we fall
// back to looking up the platform record by ID (backwards compatibility).
func (r *PythonRunner) resolvePlatformCode(instanceID string) (string, error) {
	instance, err := r.instances.FindByID(instanceID)
	if err != nil {
		return "", fmt.Errorf("instance not found: %w", err)
	}

	// 1) Try the adapter catalog (adapters/*/adapter.yaml). The instance's
	//    adapter_id is the YAML adapter id (e.g. "xianyu-adapter").
	if r.catalog != nil {
		if a, ok := r.catalog.Get(instance.AdapterID); ok {
			code := strings.ToLower(strings.TrimSpace(a.PlatformCode))
			if code != "" {
				return code, nil
			}
		}
	}

	// 2) Fallback: resolve through the DB adapter record.
	adapter, err := r.adapters.FindByID(instance.AdapterID)
	if err != nil {
		return "", fmt.Errorf("adapter not found: %w", err)
	}

	// Prefer direct use: if the adapter's platform id is a code with a script.
	code := strings.ToLower(strings.TrimSpace(adapter.PlatformID))
	if code != "" {
		script := filepath.Join(r.dir, code, "main.py")
		if _, statErr := os.Stat(script); statErr == nil {
			return code, nil
		}
	}

	// Fallback: resolve platform record by ID to get its code.
	if r.platforms != nil {
		if platform, pErr := r.platforms.FindByID(adapter.PlatformID); pErr == nil {
			pCode := strings.ToLower(strings.TrimSpace(platform.Code))
			if pCode != "" {
				return pCode, nil
			}
		}
	}

	return "", errors.New("platform code not resolvable for adapter")
}

// sandboxDir returns the instance's sandboxed adapter copy directory, or ""
// if the sandbox manager is not attached or the directory does not exist.
func (r *PythonRunner) sandboxDir(instanceID, platformCode string) string {
	if r.dirs == nil {
		return ""
	}
	dir := r.dirs.AdapterDir(instanceID)
	if _, err := os.Stat(dir); err != nil {
		return ""
	}
	return dir
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
	// Clear any intentional-stop marker; we are (re)starting now.
	delete(r.stopping, instanceID)
	r.mu.Unlock()

	// Resolve the Python script for this instance's platform.
	platformCode, err := r.resolvePlatformCode(instanceID)
	if err != nil {
		return err
	}

	// Validate the instance has a cookie configured.
	instance, err := r.instances.FindByID(instanceID)
	if err != nil {
		return err
	}

	// Reject starting while dependencies are still being installed.
	if instance.Status == "initializing" {
		return errors.New("instance is initializing; please wait for dependency installation to finish")
	}

	var cfg map[string]interface{}
	if instance.Config != "" {
		_ = json.Unmarshal([]byte(instance.Config), &cfg)
	}
	if cookie, _ := cfg["cookie"].(string); strings.TrimSpace(cookie) == "" {
		return errors.New("instance has no cookie configured; please fill it in the WebUI")
	}

	// Run from the instance's sandboxed adapter copy, never from the source
	// adapters/ directory. Verify the copy is unmodified before starting.
	workDir := r.sandboxDir(instanceID, platformCode)
	if workDir == "" {
		return errors.New("instance sandbox is not initialized; please recreate the instance")
	}
	if r.dirs != nil {
		if err := r.dirs.VerifyIntegrity(instanceID); err != nil {
			return fmt.Errorf("instance sandbox integrity check failed: %w", err)
		}
	}
	script := filepath.Join(workDir, "main.py")
	if _, err := os.Stat(script); err != nil {
		return fmt.Errorf("python adapter script not found in sandbox: %s (%v)", script, err)
	}

	// Build the command: python main.py --instance-id X --backend URL
	// Run inside the sandboxed adapter directory, so the script path is
	// relative.
	args := []string{"main.py", "--instance-id", instanceID, "--backend", r.backendURL}
	if r.jwtSecret != "" {
		if token, tErr := r.serviceToken(); tErr == nil {
			args = append(args, "--token", token)
		}
	}
	cmd := exec.Command(r.python, args...)
	cmd.Dir = workDir

	// Redirect the adapter process stdout/stderr to a per-instance log file so
	// its logs can be viewed from the WebUI.
	logFile, logErr := r.openLogFile(instanceID)
	if logErr == nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	} else {
		logger.Warn("Failed to open instance log file, falling back to stdout",
			logger.String("instance_id", instanceID),
			logger.String("error", logErr.Error()))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start python adapter: %w", err)
	}

	r.mu.Lock()
	r.procs[instanceID] = cmd
	r.mu.Unlock()

	// Mark the instance as running in the database.
	if err := r.updateInstanceStatus(instanceID, "running"); err != nil {
		logger.Warn("Failed to update instance status to running",
			logger.String("instance_id", instanceID),
			logger.String("error", err.Error()))
	}

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
	// Mark this as an intentional stop so the monitor does not auto-restart it.
	r.stopping[instanceID] = true
	r.mu.Unlock()
	if !ok || cmd == nil || cmd.Process == nil {
		return errors.New("instance is not running")
	}

	// Try graceful termination, then force kill.
	_ = cmd.Process.Signal(syscall.SIGTERM)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cmd.Process.Signal(syscall.Signal(0)) != nil {
			break // process exited
		}
		time.Sleep(200 * time.Millisecond)
	}
	// Force kill if still alive.
	if cmd.Process.Signal(syscall.Signal(0)) == nil {
		_ = cmd.Process.Kill()
		// Give the monitor's Wait a moment to reap the process.
		time.Sleep(500 * time.Millisecond)
	}

	r.mu.Lock()
	delete(r.procs, instanceID)
	delete(r.stopping, instanceID)
	delete(r.restarts, instanceID)
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

	// Do not auto-restart if this was an intentional stop.
	r.mu.Lock()
	stopping := r.stopping[instanceID]
	delete(r.stopping, instanceID)
	r.mu.Unlock()
	if stopping {
		return
	}

	// Auto-restart crashed processes, but cap consecutive crashes to avoid
	// infinite restart loops (e.g. when Python dependencies are missing).
	r.mu.Lock()
	restartCount := r.restarts[instanceID]
	r.mu.Unlock()
	if err != nil {
		r.mu.Lock()
		r.restarts[instanceID] = restartCount + 1
		r.mu.Unlock()
	}

	if r.autoRestart && err != nil && restartCount < maxAutoRestarts {
		logger.Info("Restarting python adapter process",
			logger.String("instance_id", instanceID),
			logger.Int("attempt", restartCount+1),
			logger.Int("max", maxAutoRestarts))
		if serr := r.Start(instanceID); serr != nil {
			logger.Error("Failed to restart python adapter",
				logger.String("instance_id", instanceID),
				logger.String("error", serr.Error()))
		}
	} else if err != nil && restartCount >= maxAutoRestarts {
		r.mu.Lock()
		delete(r.restarts, instanceID)
		r.mu.Unlock()
		_ = r.updateInstanceStatus(instanceID, "error")
		logger.Error("Python adapter keeps crashing; auto-restart stopped",
			logger.String("instance_id", instanceID),
			logger.String("platform", platformCode),
			logger.Int("attempts", restartCount+1),
			logger.String("hint", "check the instance log; Python dependencies may be missing"))
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
