package service

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/e-spl/e-sp-line2/pkg/logger"
)

// DependencyInstaller installs Python dependencies (requirements.txt) for
// adapter instances. When a new instance is created, its sandboxed adapter
// copy (data/instances/<id>/adapter) is used as the working directory, so
// dependencies are installed relative to the instance copy. The status/progress
// is persisted to data/instances/<id>/state.json and exposed to the WebUI so
// the instance shows "初始化中" until installation completes.
type DependencyInstaller struct {
	mu       sync.Mutex
	python   string
	dirs     *InstanceDirManager

	// states: instanceID -> installState (in-memory cache)
	states map[string]*installState
}

// installState tracks a single instance's dependency installation.
type installState struct {
	Status    string    `json:"status"` // installing, done, failed
	Progress  int       `json:"progress"` // 0-100
	Message   string    `json:"message"`
	UpdatedAt time.Time `json:"updated_at"`
	Error     string    `json:"error,omitempty"`
	done      bool      `json:"-"`
}

// NewDependencyInstaller creates a dependency installer.
func NewDependencyInstaller(python, adaptersDir string) *DependencyInstaller {
	return &DependencyInstaller{
		python: python,
		dirs:   NewInstanceDirManager(adaptersDir),
		states: make(map[string]*installState),
	}
}

// GetState returns a copy of the install state for an instance, falling back
// to the persisted state file if the in-memory entry is absent.
func (d *DependencyInstaller) GetState(instanceID string) (*installState, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if s, ok := d.states[instanceID]; ok {
		cp := *s
		return &cp, true
	}
	// Try persisted state.
	if s, err := d.loadState(instanceID); err == nil && s != nil {
		return s, true
	}
	return nil, false
}

// loadState reads the persisted state.json for an instance.
func (d *DependencyInstaller) loadState(instanceID string) (*installState, error) {
	data, err := os.ReadFile(d.dirs.StatePath(instanceID))
	if err != nil {
		return nil, err
	}
	var s installState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.Progress == 0 && s.Status == "done" {
		s.Progress = 100
	}
	s.done = true
	return &s, nil
}

// saveState persists the install state to state.json.
func (d *DependencyInstaller) saveState(instanceID string, s *installState) {
	data, err := json.Marshal(s)
	if err != nil {
		return
	}
	dir := filepath.Dir(d.dirs.StatePath(instanceID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	_ = os.WriteFile(d.dirs.StatePath(instanceID), data, 0o644)
}

// InstallDependencies installs the requirements.txt of the instance's sandbox
// adapter copy in the background. adapterDir is the sandboxed copy path
// (data/instances/<id>/adapter). The instance status is updated to reflect
// progress.
func (d *DependencyInstaller) InstallDependencies(instanceID, adapterDir string, updateStatus func(string, string) error) {
	d.mu.Lock()
	if s, ok := d.states[instanceID]; ok && s.Status == "installing" && !s.done {
		d.mu.Unlock()
		return // already installing
	}
	st := &installState{
		Status:    "installing",
		Progress:  0,
		Message:   "正在检查依赖...",
		UpdatedAt: time.Now(),
	}
	d.states[instanceID] = st
	d.saveState(instanceID, st)
	d.mu.Unlock()

	// Write an initial log line so the WebUI can show install progress.
	d.appendLog(instanceID, "开始安装适配器依赖...")

	// Locate requirements.txt inside the sandbox copy.
	reqPath := filepath.Join(adapterDir, "requirements.txt")
	if _, err := os.Stat(reqPath); err != nil {
		// No requirements file: nothing to install.
		d.finish(instanceID, "done", 100, "适配器无依赖文件(requirements.txt)，无需安装", "")
		if updateStatus != nil {
			_ = updateStatus(instanceID, "stopped")
		}
		d.appendLog(instanceID, "适配器无 requirements.txt，跳过依赖安装")
		return
	}

	d.setProgress(instanceID, 5, "正在执行依赖安装 ...")

	// Try a cascade of installers until one succeeds:
	//   python -m pip  ->  pip3  ->  pip  ->  pipx
	// Each falls back on command-not-found / no-module errors; pip/pip3 get a
	// PEP 668 retry with --break-system-packages when needed.
	attempts := d.installAttempts()
	var lastErr error
	var lastOutput string
	for _, at := range attempts {
		out, runErr := d.runInstall(at, adapterDir)
		if len(out) > 0 {
			d.appendLog(instanceID, out)
		}
		if runErr == nil {
			// Success.
			d.finish(instanceID, "done", 100, "依赖安装完成", "")
			d.appendLog(instanceID, "依赖安装完成（"+at.name+"）")
			if updateStatus != nil {
				_ = updateStatus(instanceID, "stopped")
			}
			return
		}
		lastErr = runErr
		lastOutput = out
		d.appendLog(instanceID, fmt.Sprintf("%s 安装失败，尝试下一方式...", at.name))
	}

	// All attempts failed.
	d.finish(instanceID, "failed", 100, "依赖安装失败", lastErr.Error())
	d.appendLog(instanceID, "依赖安装失败: "+lastErr.Error())
	if lastOutput != "" {
		d.appendLog(instanceID, lastOutput)
	}
	if updateStatus != nil {
		_ = updateStatus(instanceID, "error")
	}
}

// installAttempt describes one command shape used to install dependencies.
type installAttempt struct {
	name   string
	args   func() []string // build the full args (needs d.python at call time)
	env    []string
	isPip  bool // supports the PEP 668 --break-system-packages retry
	isPipx bool // pipx semantics: needs a dedicated pipx venv
}

// installAttempts returns the ordered cascade of installers to try.
//  1. python -m pip (respecting configured python binary)
//  2. pip3
//  3. pip
//  4. pipx runpip pip (isolated pipx venv; deps land inside that venv)
func (d *DependencyInstaller) installAttempts() []installAttempt {
	reqInstall := func(args ...string) []string {
		return append([]string{"install"}, args...)
	}
	return []installAttempt{
		{
			name: "python -m pip",
			args: func() []string { return append([]string{"-m", "pip"}, reqInstall("-r", "requirements.txt")...) },
			env:  []string{"PIP_DISABLE_PIP_VERSION_CHECK=1"},
			isPip: true,
		},
		{
			name:  "pip3",
			args:  func() []string { return reqInstall("-r", "requirements.txt") },
			env:   []string{"PIP_DISABLE_PIP_VERSION_CHECK=1"},
			isPip: true,
		},
		{
			name:  "pip",
			args:  func() []string { return reqInstall("-r", "requirements.txt") },
			env:   []string{"PIP_DISABLE_PIP_VERSION_CHECK=1"},
			isPip: true,
		},
		{
			name:   "pipx",
			args:   func() []string { return append([]string{"runpip", "pip", "install"}, "-r", "requirements.txt") },
			env:    nil,
			isPipx: true,
		},
	}
}

// runInstall executes a single install attempt. Returns combined output and
// error. On PEP 668 ("externally-managed-environment") pip-style attempts are
// retried once with --break-system-packages. For pipx, the "pip" venv is
// ensured to exist first (pipx install pip creates it if missing).
func (d *DependencyInstaller) runInstall(at installAttempt, adapterDir string) (string, error) {
	run := func(bin string, args []string) (string, error) {
		cmd := exec.Command(bin, args...)
		cmd.Dir = adapterDir
		if len(at.env) > 0 {
			cmd.Env = append(os.Environ(), at.env...)
		} else {
			cmd.Env = os.Environ()
		}
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	// For pipx, make sure the dedicated "pip" venv exists before running.
	if at.isPipx {
		// pipx install pip — if already present it prints "already installed"
		// and exits non-zero, which is fine; runpip below is what matters.
		ensure := exec.Command("pipx", "install", "pip")
		ensure.Dir = adapterDir
		_ = ensure.Run()
	}

	// First plain attempt.
	var bin string
	if at.isPipx {
		bin = "pipx"
	} else if strings.HasPrefix(at.name, "python") {
		bin = d.python
	} else {
		bin = at.name
	}
	output, err := run(bin, at.args())

	// PEP 668 retry for pip-style installers.
	if err != nil && at.isPip && !at.isPipx &&
		strings.Contains(strings.ToLower(output), "externally-managed-environment") {
		d.appendLogTo(adapterDir, "检测到 PEP 668 外部管理环境，使用 --break-system-packages 重试...")
		retryArgs := append([]string{"--break-system-packages"}, at.args()[1:]...)
		output2, err2 := run(bin, retryArgs)
		return output2, err2
	}

	return output, err
}

// appendLogTo writes a server-log-only note during install fallback. The full
// per-instance log is written by the caller via appendLog.
func (d *DependencyInstaller) appendLogTo(_ string, msg string) {
	logger.Info("[init] " + msg)
}

// setProgress updates the state progress/message.
func (d *DependencyInstaller) setProgress(instanceID string, progress int, message string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if s, ok := d.states[instanceID]; ok {
		s.Progress = progress
		s.Message = message
		s.UpdatedAt = time.Now()
		d.saveState(instanceID, s)
	}
}

// finish marks the install state as done/failed.
func (d *DependencyInstaller) finish(instanceID, status string, progress int, message, errText string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.states[instanceID] = &installState{
		Status:    status,
		Progress:  progress,
		Message:   message,
		Error:     errText,
		UpdatedAt: time.Now(),
		done:      true,
	}
	d.saveState(instanceID, d.states[instanceID])
}

// logDir returns the directory where per-instance logs are stored (kept in
// sync with PythonRunner.logDir).
func (d *DependencyInstaller) logDir() string {
	return filepath.Join(d.dirs.root, "..", "logs")
}

// appendLog appends a line to the instance's log file (so install progress is
// visible from the WebUI log viewer) and to the server log.
func (d *DependencyInstaller) appendLog(instanceID, text string) {
	logger.Info("[init] instance dependency install",
		logger.String("instance_id", instanceID),
		logger.String("detail", text))

	dir := d.logDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	path := filepath.Join(dir, instanceID+".log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	// Prefix each line with a level marker so the log viewer's level filter
	// (info/warning/error) works for install output.
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		marker := "INFO"
		lower := strings.ToLower(line)
		if strings.Contains(lower, "error") || strings.Contains(lower, "failed") || strings.Contains(lower, "exception") {
			marker = "ERROR"
		} else if strings.Contains(lower, "warning") {
			marker = "WARNING"
		}
		ts := time.Now().Format("2006-01-02 15:04:05")
		_, _ = f.WriteString(fmt.Sprintf("%s | %-7s | %s\n", ts, marker, line))
	}
}
