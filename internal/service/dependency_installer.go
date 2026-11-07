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
	"time"

	"github.com/e-spl/e-sp-line2/pkg/logger"
)

// errStorePythonStub marks the Microsoft Store "python" placeholder as
// unusable. It is not a real failure of the install itself: the cascade simply
// moves on to the next installer (pip3/pip/pipx).
var errStorePythonStub = errors.New("python on PATH is the Microsoft Store stub")

// isStorePythonStub reports whether bin behaves like the Windows Store
// placeholder for Python.
//
// Windows ships an App Execution Alias named python.exe that, when no real
// interpreter is installed, prints
//
//	Python was not found but can be installed from the Microsoft Store: https://...
//
// and exits non-zero. Running `python -m pip` against it produces a confusing
// error, so it is detected and skipped.
func isStorePythonStub(bin string) bool {
	if bin == "" {
		return false
	}
	out, err := exec.Command(bin, "-c", "print(1)").CombinedOutput()
	if err == nil {
		return false // a working interpreter
	}
	lower := strings.ToLower(string(out))
	return strings.Contains(lower, "microsoft store") ||
		strings.Contains(lower, "was not found") ||
		strings.Contains(lower, "app execution alias")
}

// DependencyInstaller installs Python dependencies (requirements.txt) for
// adapter instances. When a new instance is created, its sandboxed adapter
// copy (data/instances/<id>/adapter) is used as the working directory, so
// dependencies are installed relative to the instance copy. The status/progress
// is persisted to data/instances/<id>/state.json and exposed to the WebUI so
// the instance shows "初始化中" until installation completes.
type DependencyInstaller struct {
	mu     sync.Mutex
	python string
	node   string
	dirs   *InstanceDirManager

	// runtimes maps instanceID -> the runtime used for its dependency check.
	// A standalone instance skips pip entirely and runs the adapter self-check.
	runtimes map[string]dependencyRuntime

	// states: instanceID -> installState (in-memory cache)
	states map[string]*installState
}

// installState tracks a single instance's dependency installation.
type installState struct {
	Status    string    `json:"status"`   // installing, done, failed
	Progress  int       `json:"progress"` // 0-100
	Message   string    `json:"message"`
	UpdatedAt time.Time `json:"updated_at"`
	Error     string    `json:"error,omitempty"`
	done      bool      `json:"-"`
}

// NewDependencyInstaller creates a dependency installer.
func NewDependencyInstaller(python, adaptersDir string) *DependencyInstaller {
	return &DependencyInstaller{
		python:   python,
		node:     "node",
		dirs:     NewInstanceDirManager(adaptersDir),
		states:   make(map[string]*installState),
		runtimes: make(map[string]dependencyRuntime),
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

// dependencyMarkerDir returns the directory holding per-platform "deps
// installed" markers (data/deps). A marker means the platform's requirements
// were already installed once system-wide, so new instances of the same
// platform skip re-installation.
func (d *DependencyInstaller) dependencyMarkerDir() string {
	// Derived from the data root so it stays data/deps whether the adapters are
	// external (adapters/) or extracted from the embedded bundle (data/adapters/).
	return filepath.Join(dataRootDir(d.dirs.adaptersDir), "deps")
}

// markerPath returns the marker path for a platform code.
func (d *DependencyInstaller) markerPath(platformCode string) string {
	return filepath.Join(d.dependencyMarkerDir(), platformCode+".installed")
}

// depsInstalled reports whether the platform's dependencies were already
// installed (marker file present).
func (d *DependencyInstaller) depsInstalled(platformCode string) bool {
	if platformCode == "" {
		return false
	}
	_, err := os.Stat(d.markerPath(platformCode))
	return err == nil
}

// markDepsInstalled writes the marker for a platform after a successful
// install.
func (d *DependencyInstaller) markDepsInstalled(platformCode string) {
	if platformCode == "" {
		return
	}
	dir := d.dependencyMarkerDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	_ = os.WriteFile(d.markerPath(platformCode), []byte(time.Now().Format(time.RFC3339)), 0o644)
}

// InstallDependencies installs the requirements.txt of the instance's sandbox
// adapter copy in the background. adapterDir is the sandboxed copy path
// (data/instances/<id>/adapter); platformCode identifies the adapter family so
// that repeated installs are de-duplicated (the first instance of a platform
// installs once; later instances of the same platform skip).
//
// If the shared marker exists but a dependency is actually missing at runtime,
// the adapter start will fail with a clear traceback; deleting the marker (or
// removing data/deps/<platform>.installed) forces re-installation.
func (d *DependencyInstaller) InstallDependencies(instanceID, adapterDir, platformCode string, updateStatus func(string, string) error) {
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

	// InstallDependencies returns immediately: pip can take tens of seconds
	// (downloading/building wheels), and callers invoke it straight from an
	// HTTP handler. Running it inline blocked POST /instances long enough for
	// the browser to time out, and left the UI stuck on "initializing" with no
	// visible progress. The WebUI polls the init status, so the work is done in
	// the background and reported through installState.
	go d.install(instanceID, adapterDir, platformCode, updateStatus)
}

// install performs the actual dependency installation in the background.
func (d *DependencyInstaller) install(instanceID, adapterDir, platformCode string, updateStatus func(string, string) error) {
	defer func() {
		if r := recover(); r != nil {
			d.finish(instanceID, "failed", 100, "依赖安装异常终止", fmt.Sprintf("%v", r))
			logger.Error("dependency install panicked",
				logger.String("instance_id", instanceID),
				logger.String("error", fmt.Sprintf("%v", r)))
		}
	}()

	// Write an initial log line so the WebUI can show install progress.
	d.appendLog(instanceID, "开始安装适配器依赖...")

	// Standalone adapters do not use pip: their dependencies are the Node
	// runtime and the external CLI, which the adapter's own --self-check
	// verifies. Branch before any requirements.txt handling.
	if rt := d.runtimeOf(instanceID); rt.Type == RuntimeStandalone {
		d.runStandaloneSelfCheck(instanceID, adapterDir, rt)
		if updateStatus != nil {
			_ = updateStatus(instanceID, "stopped")
		}
		return
	}

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

	// De-duplicate: if this platform already had its dependencies installed,
	// skip the pip step for this instance.
	if d.depsInstalled(platformCode) {
		d.finish(instanceID, "done", 100, "依赖已安装（复用同一平台共享依赖）", "")
		d.appendLog(instanceID, "该平台依赖已安装过，跳过重复安装")
		if updateStatus != nil {
			_ = updateStatus(instanceID, "stopped")
		}
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
			// Success. Mark the platform as having dependencies installed so
			// future instances of the same platform skip re-installation.
			d.markDepsInstalled(platformCode)
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

	// All whole-file install attempts failed (e.g. one optional dependency is
	// unavailable on this Python version, like execjs on 3.14). Fall back to
	// installing each requirement individually, skipping packages that cannot
	// be resolved, so the adapter can still start with the available deps
	// (loguru, requests, websockets, ...).
	d.appendLog(instanceID, "整体安装失败，尝试逐个安装依赖（跳过不可用的包）...")
	installed, failedPkgs := d.installIndividually(adapterDir)
	if installed > 0 {
		d.markDepsInstalled(platformCode)
		msg := fmt.Sprintf("依赖部分安装完成（%d 个包已装）", installed)
		if len(failedPkgs) > 0 {
			msg += "；跳过不可用: " + strings.Join(failedPkgs, ", ")
		}
		d.finish(instanceID, "done", 100, msg, "")
		d.appendLog(instanceID, msg)
		if updateStatus != nil {
			_ = updateStatus(instanceID, "stopped")
		}
		return
	}

	// Everything failed.
	d.finish(instanceID, "failed", 100, "依赖安装失败", lastErr.Error())
	d.appendLog(instanceID, "依赖安装失败: "+lastErr.Error())
	if lastOutput != "" {
		d.appendLog(instanceID, lastOutput)
	}
	if updateStatus != nil {
		_ = updateStatus(instanceID, "error")
	}
}

// installIndividually parses requirements.txt and attempts to install each
// requirement on its own using the first working pip installer. Packages that
// fail to resolve are skipped. Returns the number installed and the names of
// the failed ones.
func (d *DependencyInstaller) installIndividually(adapterDir string) (int, []string) {
	reqPath := filepath.Join(adapterDir, "requirements.txt")
	data, err := os.ReadFile(reqPath)
	if err != nil {
		return 0, nil
	}
	var pkgs []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Skip lines that are options or editable installs.
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "http") {
			continue
		}
		pkgs = append(pkgs, line)
	}

	// Pick the installer. Prefer `python -m pip` (the interpreter that the
	// adapter runs under), so packages land in the system/user site-packages
	// that `python` can actually import. Plain `pip3`/`pip` on PATH may be
	// pipx shims that install into an isolated venv, so they are only used as
	// a fallback when `python -m pip` itself is unavailable.
	var base []string
	bin := ""
	if probe := exec.Command(d.python, "-m", "pip", "--version"); probe.Run() == nil {
		bin = d.python
		base = []string{"-m", "pip", "install"}
	} else {
		for _, name := range []string{"pip3", "pip"} {
			if probe := exec.Command(name, "--version"); probe.Run() == nil {
				bin = name
				base = []string{"install"}
				break
			}
		}
	}
	if bin == "" {
		return 0, nil
	}

	installed := 0
	var failed []string
	for _, pkg := range pkgs {
		cmd := exec.Command(bin, append(append([]string{}, base...), pkg)...)
		cmd.Dir = adapterDir
		cmd.Env = append(os.Environ(), "PIP_DISABLE_PIP_VERSION_CHECK=1")
		out, cerr := cmd.CombinedOutput()
		lower := strings.ToLower(string(out))
		if cerr != nil && strings.Contains(lower, "externally-managed-environment") {
			// PEP 668 retry for this package.
			retryArgs := append([]string{}, base...)
			retryArgs = append(retryArgs, "--break-system-packages", pkg)
			retry := exec.Command(bin, retryArgs...)
			retry.Dir = adapterDir
			retry.Env = append(os.Environ(), "PIP_DISABLE_PIP_VERSION_CHECK=1")
			out, cerr = retry.CombinedOutput()
		}
		if cerr == nil {
			installed++
			logger.Info("[init] individual dep installed", logger.String("pkg", pkg))
		} else {
			failed = append(failed, pkg)
			// Log the tail of the real output to diagnose failures.
			outStr := strings.TrimSpace(string(out))
			if len(outStr) > 400 {
				outStr = outStr[len(outStr)-400:]
			}
			logger.Warn("[init] individual dep skipped",
				logger.String("pkg", pkg),
				logger.String("error", cerr.Error()),
				logger.String("output", outStr))
		}
	}
	return installed, failed
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
			name:  "python -m pip",
			args:  func() []string { return append([]string{"-m", "pip"}, reqInstall("-r", "requirements.txt")...) },
			env:   []string{"PIP_DISABLE_PIP_VERSION_CHECK=1"},
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

	// A "python" found on PATH on Windows is frequently the Microsoft Store
	// stub, which is not a real interpreter: it prints an advertisement and
	// exits non-zero. Detect it up front so the cascade moves on to a working
	// pip instead of reporting a confusing pip error.
	if !at.isPipx && strings.HasPrefix(at.name, "python") && isStorePythonStub(d.python) {
		return "python on PATH is the Microsoft Store stub, not a real interpreter; " +
			"skipping this installer. Install Python from https://www.python.org/downloads/ " +
			"and ensure it is on PATH.", errStorePythonStub
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

	// PEP 668 retry for pip-style installers: insert --break-system-packages
	// right after the "install" subcommand.
	//   python -m pip install --break-system-packages -r requirements.txt
	//   pip3 install --break-system-packages -r requirements.txt
	if err != nil && at.isPip && !at.isPipx &&
		strings.Contains(strings.ToLower(output), "externally-managed-environment") {
		d.appendLogTo(adapterDir, "检测到 PEP 668 外部管理环境，使用 --break-system-packages 重试...")
		args := at.args()
		// Find the "install" token and insert after it.
		insertAt := -1
		for i, a := range args {
			if a == "install" {
				insertAt = i + 1
				break
			}
		}
		retryArgs := args
		if insertAt > 0 {
			retryArgs = append([]string{}, args[:insertAt]...)
			retryArgs = append(retryArgs, "--break-system-packages")
			retryArgs = append(retryArgs, args[insertAt:]...)
		}
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
	return filepath.Join(dataRootDir(d.dirs.adaptersDir), "logs")
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
	//
	// Standalone adapters already emit `时间 | 级别 | [tag] 消息` themselves and
	// answer with a JSON envelope: those lines go through verbatim instead of
	// being prefixed twice and re-classified by keyword (an envelope contains
	// the literal `"error":null`, which would otherwise mark it ERROR).
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if hasLevelColumn(line) {
			_, _ = f.WriteString(line + "\n")
			continue
		}
		marker := "INFO"
		lower := strings.ToLower(line)
		switch {
		case strings.HasPrefix(strings.TrimSpace(line), "{"):
			marker = envelopeLevel(line, marker)
		case strings.Contains(lower, "error") || strings.Contains(lower, "failed") || strings.Contains(lower, "exception"):
			marker = "ERROR"
		case strings.Contains(lower, "warning"):
			marker = "WARNING"
		}
		ts := time.Now().Format("2006-01-02 15:04:05")
		_, _ = fmt.Fprintf(f, "%s | %-7s | %s\n", ts, marker, line)
	}
}

// hasLevelColumn reports whether a line already starts with our log record
// format `YYYY-MM-DD HH:MM:SS | LEVEL`.
func hasLevelColumn(line string) bool {
	if len(line) < 21 || line[4] != '-' || line[10] != ' ' || line[13] != ':' {
		return false
	}
	return strings.Contains(line[19:], "|")
}

// envelopeLevel derives the log level of a JSON envelope from its ok flag.
func envelopeLevel(line, fallback string) string {
	var env struct {
		Ok *bool `json:"ok"`
	}
	if err := json.Unmarshal([]byte(line), &env); err != nil || env.Ok == nil {
		return fallback
	}
	if *env.Ok {
		return "INFO"
	}
	return "ERROR"
}

// ---------------------------------------------------------------- runtime ---

// dependencyRuntime is how a dependency check runs for one instance. The
// default checks the adapter family's requirements.txt with pip; the standalone
// runtime additionally runs the adapter's own `--self-check` so a missing Node
// or catbus CLI is reported at creation time, not on the first start.
type dependencyRuntime struct {
	// Type is the adapter runtime_type.
	Type string
	// Spec is the standalone runtime declaration (interpreter/entry/args).
	Spec RuntimeSpec
	// Config is the instance config, handed to the self-check via --config so
	// the adapter sees the same values a real start would give it.
	Config string
}

// SetNodeBin sets the interpreter used for standalone adapters (default node).
func (d *DependencyInstaller) SetNodeBin(bin string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.node = bin
}

// adoptRuntime must be called before InstallDependencies to switch the
// dependency path away from pip. It is a no-op for python adapters, keeping
// their behaviour byte-for-byte identical.
func (d *DependencyInstaller) adoptRuntime(instanceID, runtimeType string, spec RuntimeSpec, cfg string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.runtimes == nil {
		d.runtimes = make(map[string]dependencyRuntime)
	}
	if runtimeType == RuntimeStandalone {
		d.runtimes[instanceID] = dependencyRuntime{Type: runtimeType, Spec: spec, Config: cfg}
		return
	}
	delete(d.runtimes, instanceID)
}

// runtimeOf returns the registered runtime for an instance (python default).
func (d *DependencyInstaller) runtimeOf(instanceID string) dependencyRuntime {
	d.mu.Lock()
	defer d.mu.Unlock()
	rt, ok := d.runtimes[instanceID]
	if !ok {
		return dependencyRuntime{Type: RuntimePython}
	}
	return rt
}

// runStandaloneSelfCheck executes `<interpreter> <entry> --self-check` in the
// sandbox and records the result in the instance log and install state.
//
// The check is deliberately the adapter's own: only the adapter knows which
// CLI and credentials it needs. Exit code 0 with a JSON envelope means ready;
// anything else leaves the instance in "error" with the adapter's explanation.
func (d *DependencyInstaller) runStandaloneSelfCheck(instanceID, adapterDir string, rt dependencyRuntime) {
	interpreter := rt.Spec.Interpreter
	if interpreter == "" {
		interpreter = d.node
	}
	if interpreter == "" {
		interpreter = "node"
	}
	entry := rt.Spec.Entry
	args := []string{entry, "--self-check", "--instance-id", instanceID}
	args = append(args, rt.Spec.Args...)

	// Hand the instance config over exactly like a real start does, so the
	// self-check validates the configured CLI instead of a guessed default.
	var cfg map[string]interface{}
	if strings.TrimSpace(rt.Config) != "" {
		_ = json.Unmarshal([]byte(rt.Config), &cfg)
	}
	if path, err := d.writeSelfCheckConfig(instanceID, cfg); err == nil {
		args = append(args, "--config", path)
	}

	d.setProgress(instanceID, 40, "正在检查独立式运行时依赖 ...")
	d.appendLog(instanceID, "[standalone] 依赖自检: "+interpreter+" "+strings.Join(args, " "))

	cmd := exec.Command(interpreter, args...)
	cmd.Dir = adapterDir
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if text != "" {
		d.appendLog(instanceID, text)
	}
	if err != nil {
		msg := "独立式运行时自检失败"
		if text != "" {
			msg += ": " + lastLine(text)
		}
		d.appendLog(instanceID, "[standalone] "+msg+"; 安装 Node（或补齐 catbus CLI）后重试")
		d.finish(instanceID, "failed", 100, msg, err.Error())
		return
	}
	d.appendLog(instanceID, "[standalone] 自检通过")
	d.finish(instanceID, "done", 100, "独立式运行时依赖自检通过", "")
}

// writeSelfCheckConfig writes the instance config next to the sandbox copy for
// the self-check run.
func (d *DependencyInstaller) writeSelfCheckConfig(instanceID string, cfg map[string]interface{}) (string, error) {
	if cfg == nil {
		cfg = map[string]interface{}{}
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(d.dirs.AdapterDir(instanceID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "config.runtime.json")
	// The command runs with cmd.Dir set to the sandbox, so this must be
	// absolute or the adapter resolves it against the wrong directory.
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return abs, os.WriteFile(abs, data, 0o600)
}

// lastLine returns the last non-empty line of text (trimmed), for short errors.
func lastLine(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if s := strings.TrimSpace(lines[i]); s != "" {
			if len(s) > 300 {
				return s[:300]
			}
			return s
		}
	}
	return ""
}
