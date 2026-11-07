package service

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	mu          sync.Mutex
	procs       map[string]*exec.Cmd
	stdins      map[string]io.WriteCloser                             // standalone: command channel to the process
	readers     map[string]*envelopeReader                            // standalone: stdout envelope demux
	toolSink    func(instanceID string, frame map[string]interface{}) // standalone tool_catalog
	stopping    map[string]bool                                       // instances marked for intentional stop
	restarts    map[string]int                                        // consecutive crash count per instance
	python      string
	node        string
	dir         string
	autoRestart bool
	backendURL  string
	jwtSecret   string

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

// adapterBanner is the ASCII art echoed into every adapter instance log when
// its process starts, so the WebUI instance log always shows the same banner
// regardless of the Python script's own startup output.
const adapterBanner = `
  ███████╗   ███████╗██████╗     ██╗     ██╗███╗   ██╗███████╗██████╗
  ██╔════╝   ██╔════╝██╔══██╗    ██║     ██║████╗  ██║██╔════╝╚════██╗
  █████╗     ███████╗██████╔╝    ██║     ██║██╔██╗ ██║█████╗    ▄███╔╝
  ██╔══╝     ╚════██║██╔═══╝     ██║     ██║██║╚██╗██║██╔══╝  ▄▀══╝
  ███████╗   ███████║██║         ███████╗██║██║ ╚████║███████╗███████╗
  ╚══════╝   ╚══════╝╚═╝         ╚══════╝╚═╝╚═╝  ╚═══╝╚══════╝╚══════╝

  Power By LangBot-community-team

  --------------------------------------------------------------------
`

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
		stdins:      make(map[string]io.WriteCloser),
		readers:     make(map[string]*envelopeReader),
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

// SetToolCatalogSink installs the callback that receives a standalone adapter's
// "tool_catalog" envelope at startup. The bridge model sends the same frame over
// the WebSocket; standalone adapters have no socket, so the stdout pump hands it
// here to keep the inactive-field tool registry populated for both runtimes.
func (r *PythonRunner) SetToolCatalogSink(fn func(instanceID string, frame map[string]interface{})) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.toolSink = fn
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
	// Derived from the data root so logs stay in data/logs whether the adapters
	// are external (adapters/) or extracted from the embedded bundle
	// (data/adapters/).
	return filepath.Join(dataRootDir(r.dir), "logs")
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
//     "debug", "info", "warning", "error" (case-insensitive).
//     Empty or "all" returns every line. Only the chosen level
//     and higher-severity lines are kept (e.g. "warning"
//     keeps warning + error). Continuation lines (indented
//     tracebacks / stack traces) are kept with their parent.
//   - keyword        : optional substring filter (case-insensitive). Multiple
//     space-separated words are ANDed; lines must contain all
//     of them. If empty, no keyword filtering.
//   - from / to      : optional time-range filters (RFC3339). Lines whose
//     timestamp falls outside [from, to] are dropped. Lines
//     without a parseable timestamp are always kept.
//
// If the file does not exist, it returns an empty log (not an error).
func (r *PythonRunner) ReadLog(instanceID string, maxLines int, level, keyword, from, to string) (string, error) {
	lines, err := r.readLogLines(instanceID, maxLines)
	if err != nil {
		return "", err
	}

	// Time-range filter (applied first).
	if from != "" || to != "" {
		var fromT, toT *time.Time
		if from != "" {
			if t, perr := parseLogTime(from); perr == nil {
				fromT = &t
			}
		}
		if to != "" {
			if t, perr := parseLogTime(to); perr == nil {
				toT = &t
			}
		}
		if fromT != nil || toT != nil {
			filtered := make([]string, 0, len(lines))
			for _, ln := range lines {
				ts, ok := logLineTime(ln)
				if !ok {
					// No timestamp -> keep (continuation / raw output).
					filtered = append(filtered, ln)
					continue
				}
				if fromT != nil && ts.Before(*fromT) {
					continue
				}
				if toT != nil && ts.After(*toT) {
					continue
				}
				filtered = append(filtered, ln)
			}
			lines = filtered
		}
	}

	level = strings.ToLower(strings.TrimSpace(level))
	if level != "" && level != "all" {
		filtered := make([]string, 0, len(lines))
		for i, ln := range lines {
			if logLineMatches(ln, level) {
				filtered = append(filtered, ln)
				continue
			}
			// Keep indented continuation lines (traceback / stack frames) that
			// immediately follow a kept line, so stack traces stay complete.
			if isContinuation(ln) && len(filtered) > 0 && i > 0 {
				if len(filtered) > 0 && filtered[len(filtered)-1] == lines[i-1] {
					filtered = append(filtered, ln)
				}
			}
		}
		lines = filtered
	}

	// Keyword filter (applied after level filtering).
	if keyword != "" {
		words := strings.Fields(strings.ToLower(keyword))
		kwFiltered := make([]string, 0, len(lines))
		for _, ln := range lines {
			lower := strings.ToLower(ln)
			all := true
			for _, w := range words {
				if !strings.Contains(lower, w) {
					all = false
					break
				}
			}
			if all {
				kwFiltered = append(kwFiltered, ln)
			}
		}
		lines = kwFiltered
	}

	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return strings.Join(lines, "\n"), nil
}

// maxLogReadLines caps how many lines are read from a log file in one pass.
// Logs grow unboundedly over long-running deployments; reading the whole file
// on every request would exhaust memory and stall the server. When a caller
// asks for the "full" log (maxLines <= 0) we still cap the read so the
// request stays bounded.
const maxLogReadLines = 20000

// readLogLines reads and splits the instance log file into lines. When
// maxLines > 0 only the trailing maxLines lines are read (tail semantics);
// otherwise the whole file is read but capped at maxLogReadLines lines so a
// huge log never OOMs the server.
func (r *PythonRunner) readLogLines(instanceID string, maxLines int) ([]string, error) {
	path := r.logPath(instanceID)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	// If the caller wants a bounded tail, read only the last maxLines lines
	// from the end of the file instead of the whole file.
	if maxLines > 0 {
		return readTailLines(f, maxLines)
	}

	// Full read, but bounded to maxLogReadLines lines.
	return readBoundedLines(f, maxLogReadLines)
}

// readTailLines reads the last n lines of an open file by scanning backwards
// from the end. This avoids loading a huge log into memory.
func readTailLines(f *os.File, n int) ([]string, error) {
	const chunkSize = 64 * 1024

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := info.Size()
	if size == 0 {
		return nil, nil
	}

	// Read backwards in chunks, collecting lines until we have n of them or
	// reach the start of the file.
	var buf []byte
	pos := size
	lines := make([]string, 0, n)
	// leftover holds a partial line carried across chunk boundaries.
	leftover := ""

	for pos > 0 && len(lines) < n {
		readSize := int64(chunkSize)
		if readSize > pos {
			readSize = pos
		}
		pos -= readSize

		chunk := make([]byte, readSize)
		if _, err := f.ReadAt(chunk, pos); err != nil && err != io.EOF {
			return nil, err
		}

		// Prepend the chunk to the buffer (we are scanning backwards).
		buf = append(chunk, buf...)

		// Split the buffer into lines. The buffer may contain more than n
		// lines; we only keep the last n.
		parts := strings.Split(string(buf), "\n")
		if len(parts) > 1 {
			// The last element is either a partial line (no trailing \n) or
			// empty (trailing \n). Combine it with leftover from the next
			// (earlier) chunk.
			last := parts[len(parts)-1]
			if last != "" {
				leftover = last + leftover
			}
			// Take the last (n - len(lines)) complete lines from parts.
			complete := parts[:len(parts)-1]
			need := n - len(lines)
			if len(complete) > need {
				complete = complete[len(complete)-need:]
			}
			// Prepend: earlier lines come first.
			lines = append(append([]string{}, complete...), lines...)
			// Keep only the unprocessed prefix of the buffer for the next
			// (earlier) iteration.
			buf = []byte(strings.Join(parts[:len(parts)-1], "\n"))
		}
	}

	// If we reached the start of the file, prepend the leftover (the first
	// line of the file).
	if pos == 0 && leftover != "" {
		lines = append([]string{leftover}, lines...)
	}

	// Trim trailing empty lines.
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines, nil
}

// readBoundedLines reads at most n lines from the start of an open file.
func readBoundedLines(f *os.File, n int) ([]string, error) {
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines, nil
}

// LogHeatmapEntry is one hour bucket of log counts.
type LogHeatmapEntry struct {
	Hour     string `json:"hour"` // RFC3339 at the top of the hour
	Debug    int    `json:"debug"`
	Info     int    `json:"info"`
	Warning  int    `json:"warning"`
	Error    int    `json:"error"`
	Unmarked int    `json:"unmarked"`
}

// LogHeatmap returns per-hour counts of log lines by level. When `level` is
// non-empty (and not "all"), only lines of that level or higher are counted.
func (r *PythonRunner) LogHeatmap(instanceID, level string) ([]LogHeatmapEntry, error) {
	lines, err := r.readLogLines(instanceID, 0)
	if err != nil || len(lines) == 0 {
		return []LogHeatmapEntry{}, err
	}

	level = strings.ToLower(strings.TrimSpace(level))
	keep := severityAtLeast(level)

	buckets := map[string]*LogHeatmapEntry{}
	var order []string
	for _, ln := range lines {
		ts, ok := logLineTime(ln)
		if !ok {
			continue // no timestamp, skip from heatmap
		}
		lvl := logLineLevel(ln)
		if keep != nil && lvl != "" && !keep[lvl] {
			continue
		}
		hour := ts.Truncate(time.Hour)
		key := hour.Format(time.RFC3339)
		b, exists := buckets[key]
		if !exists {
			b = &LogHeatmapEntry{Hour: key}
			buckets[key] = b
			order = append(order, key)
		}
		switch lvl {
		case "debug":
			b.Debug++
		case "info":
			b.Info++
		case "warning":
			b.Warning++
		case "error":
			b.Error++
		default:
			b.Unmarked++
		}
	}

	result := make([]LogHeatmapEntry, 0, len(order))
	for _, k := range order {
		result = append(result, *buckets[k])
	}
	return result, nil
}

// ClearLog truncates the instance's log file. If the adapter process is
// currently running, its open file handle keeps writing at the new offset, so
// subsequent logs are preserved. Returns the number of bytes removed.
func (r *PythonRunner) ClearLog(instanceID string) (int64, error) {
	path := r.logPath(instanceID)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // nothing to clear
		}
		return 0, err
	}
	if err := os.Truncate(path, 0); err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// logLineTime parses the leading timestamp of a log line (e.g.
// "2026-08-16 17:27:29 | INFO | ..."). Returns the time and true if parseable.
func logLineTime(line string) (time.Time, bool) {
	// Support "2006-01-02 15:04:05" prefix (our loguru/installer format).
	if len(line) >= 19 {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", line[:19], time.Local); err == nil {
			return t, true
		}
	}
	// ISO8601 with T (from other sources).
	if len(line) >= 19 {
		if t, err := time.Parse(time.RFC3339, line[:19]+"Z"); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// parseLogTime parses a user-supplied time filter (RFC3339 or "2006-01-02 15:04:05").
func parseLogTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid time: %s", s)
}

// isContinuation reports whether a line is a continuation (indented stack
// frame / traceback) line rather than a top-level log record.
func isContinuation(line string) bool {
	return strings.HasPrefix(line, " ") && strings.TrimSpace(line) != ""
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

// logLineMatches reports whether a log line's level is within the allowed
// set. It extracts the line's level via logLineLevel; lines without any
// recognized level marker are NOT kept by the level filter (so choosing ERROR
// does not leak unrelated continuation/noise lines). Continuation lines are
// re-attached by ReadLog based on adjacency instead.
func logLineMatches(line string, level string) bool {
	// "debug" shows everything (most verbose).
	if level == "debug" {
		return true
	}
	lvl := logLineLevel(line)
	if lvl == "" {
		// ASCII banner / decorative lines (block-drawing chars, "Power By")
		// show under the "info" filter regardless of leading whitespace.
		if isBannerLine(line) {
			return level == "info"
		}
		// Indented continuation lines (traceback frames) are not judged on
		// their own; the caller re-attaches them to their parent line.
		if isContinuation(line) {
			return false
		}
		// Other unmarked lines (raw adapter output) show under "info".
		return level == "info"
	}
	// Exact-level matching: "info" shows only info lines, "error" only
	// error lines, etc. This keeps Python error blocks out of the info view.
	return lvl == level
}

// isBannerLine reports whether a line is part of the ASCII startup banner
// (block-drawing characters or the "Power By" footer). Banner lines carry no
// log level but should still appear under the info/debug filters.
func isBannerLine(line string) bool {
	lower := strings.ToLower(line)
	return strings.ContainsAny(line, "█╗╝╔╚═║") || strings.Contains(lower, "power by")
}

// logLineLevel extracts the log level from a line. Recognizes:
//   - loguru style:   "... | INFO     | ..." (also "| ERROR |")
//   - installer log:  "2026-08-16 17:27:33 | ERROR | ..."
//   - standard:       "INFO:" / "WARNING:" / "ERROR:" / "DEBUG:" / "ERROR -"
//   - python errors:  "Traceback (most recent call last):", "KeyError: ...",
//     "TypeError: ...", "<...>Exception: ...", "ModuleNotFoundError: ..."
//
// Returns "" if no level marker is found.
func logLineLevel(line string) string {
	lower := strings.ToLower(line)
	levels := []string{"debug", "info", "warning", "error"}
	// 1) loguru / installer format: " | <LEVEL> |" or " | <LEVEL>  " (padded)
	for _, lvl := range levels {
		if strings.Contains(lower, " | "+lvl+" ") || strings.Contains(lower, " | "+lvl+"|") {
			return lvl
		}
	}
	// 2) standard "LEVEL:" prefix or " LEVEL:" mid-line, and "LEVEL - " form.
	for _, lvl := range levels {
		if strings.HasPrefix(lower, lvl+":") || strings.Contains(lower, " "+lvl+":") {
			return lvl
		}
		if strings.HasPrefix(lower, lvl+" - ") {
			return lvl
		}
	}
	// 3) Python exception lines count as errors.
	if isPythonErrorLine(lower) {
		return "error"
	}
	return ""
}

// isPythonErrorLine reports whether a line looks like a Python exception or
// traceback marker. Uses common patterns; the level filter runs on the
// lowercased line.
func isPythonErrorLine(lower string) bool {
	// Traceback header.
	if strings.HasPrefix(lower, "traceback (most recent call last):") {
		return true
	}
	// "Exception:" / "Error:" / "<class>Error: ..." / "ModuleNotFoundError: ..."
	// e.g. "KeyError: ...", "TypeError: ...", "NameError: ...", "ModuleNotFoundError: ..."
	for _, suffix := range []string{"error:", "exception:", "traceback (most recent call last)"} {
		if strings.HasPrefix(lower, suffix) {
			return true
		}
	}
	// "xxxError: ..." / "xxxException: ..." mid-line start (common python raises).
	// Match patterns like "keyerror:", "modulenotfounderror:", "valueerror:".
	for _, marker := range []string{"error:", "exception:"} {
		idx := strings.Index(lower, marker)
		if idx > 0 && idx <= 40 {
			// Ensure the token before the marker is a plausible exception name
			// (letters/underscores only), e.g. "KeyError:".
			prev := lower[:idx]
			lastSpace := strings.LastIndex(prev, " ")
			lastColon := strings.LastIndex(prev, ":")
			start := lastSpace
			if lastColon > start {
				start = lastColon
			}
			if start < 0 {
				start = 0
			}
			name := strings.TrimSpace(prev[start:])
			if name != "" && isAlnumOrUnderscore(name) {
				return true
			}
		}
	}
	// Common Python exception class names appearing anywhere in the line
	// (e.g. "raise ValueError", "FileNotFoundError", "json.JSONDecodeError").
	// This ensures every Python error is classified as error level.
	for _, exc := range pythonExceptionNames {
		if strings.Contains(lower, exc) {
			return true
		}
	}
	return false
}

// pythonExceptionNames lists common Python exception class names (lowercased)
// used to classify error lines that do not follow the "xxxError:" pattern.
var pythonExceptionNames = []string{
	"keyerror", "valueerror", "typeerror", "nameerror", "indexerror",
	"attributeerror", "importerror", "modulenotfounderror", "syntaxerror",
	"runtimeerror", "zerodivisionerror", "oserror", "ioerror",
	"filenotfounderror", "connectionerror", "timeouterror", "jsondecodeerror",
	"stopiteration", "assertionerror", "notimplementederror", "overflowerror",
	"recursionerror", "unboundlocalerror", "indentationerror", "taberror",
	"systemerror", "memoryerror", "arithmeticerror", "lookuperror",
	"environmenterror", "exception", "error",
}

// isAlnumOrUnderscore reports whether s consists only of letters, digits and
// underscores (used to validate exception class names).
func isAlnumOrUnderscore(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return false
	}
	return true
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

// resolveSourceDir resolves the adapters/<dir> source directory an instance is
// sandboxed from.
//
// It is normally the platform code, but an adapter whose directory name differs
// from its platform_code (e.g. adapters/catbus serving platform xianyu) must be
// copied from its own directory. The catalog records the real directory, so it
// wins whenever the adapter is declared in adapter.yaml.
func (r *PythonRunner) resolveSourceDir(instanceID string) (string, error) {
	if r.catalog != nil {
		if instance, err := r.instances.FindByID(instanceID); err == nil {
			if a, ok := r.catalog.Get(instance.AdapterID); ok {
				if dir := strings.TrimSpace(a.Dir); dir != "" {
					return dir, nil
				}
			}
		}
	}
	return r.resolvePlatformCode(instanceID)
}

// sandboxDir returns the instance's sandboxed adapter copy directory.
//
// The sandbox is normally created when the instance is created, but it can be
// missing for legitimate reasons that have nothing to do with the instance
// being invalid:
//
//   - the data/ directory was deleted or moved (e.g. the user copied the
//     executable elsewhere, or ran from a different working directory);
//   - the database was kept while data/instances was cleaned up;
//   - the server was upgraded across a layout change.
//
// Because the adapter sources are embedded in the binary (or available in the
// external adapters/ directory), the sandbox can always be rebuilt. Previously
// this returned "" and the user was told to "recreate the instance", which
// throws away their configuration for no good reason.
//
// It now self-heals: if the sandbox copy (or its manifest) is absent, it is
// restored from the adapter source.
func (r *PythonRunner) sandboxDir(instanceID, platformCode, entry string) string {
	if r.dirs == nil {
		return ""
	}
	dir := r.dirs.AdapterDir(instanceID)
	if sandboxUsable(dir, entry) {
		return dir
	}

	// Attempt to restore the sandbox from the adapter source.
	if platformCode == "" {
		return ""
	}
	restored, err := r.dirs.CopyAdapter(instanceID, platformCode)
	if err != nil {
		logger.Warn("Failed to rebuild instance sandbox",
			logger.String("instance_id", instanceID),
			logger.String("platform_code", platformCode),
			logger.String("error", err.Error()))
		return ""
	}
	logger.Info("Rebuilt missing instance sandbox",
		logger.String("instance_id", instanceID),
		logger.String("adapter_dir", restored))
	return restored
}

// requiredEntry returns the launcher file a sandbox of the given runtime must
// contain, relative to the sandbox dir.
//
// The bridge model always launches main.py. A standalone adapter launches its
// declared runtime.entry and has no main.py at all — checking for main.py there
// made every standalone sandbox look broken, so every start did a full
// RemoveAll + re-copy and wiped the adapter's own files plus state.json.
func requiredEntry(runtimeType string, spec RuntimeSpec) string {
	if runtimeType != RuntimeStandalone {
		return "main.py"
	}
	entry := filepath.Clean(strings.TrimSpace(spec.Entry))
	if entry == "" || entry == "." {
		return ""
	}
	return entry
}

// sandboxUsable reports whether dir holds a complete sandbox copy: the
// runtime's launcher entry present and a manifest written (the manifest is
// required by the integrity check performed before start).
func sandboxUsable(dir, entry string) bool {
	if dir == "" || entry == "" {
		return false
	}
	if _, err := os.Stat(filepath.Join(dir, entry)); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "manifest.json")); err != nil {
		return false
	}
	return true
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

	// Resolve the platform code (for the start record and catalogue paths).
	platformCode, err := r.resolvePlatformCode(instanceID)
	if err != nil {
		return err
	}

	// Validate the instance has a cookie configured.
	instance, err := r.instances.FindByID(instanceID)
	if err != nil {
		return err
	}

	// Which launcher this instance uses. Resolved before the cookie check
	// because a standalone adapter owns its own login state (e.g. catbus
	// keeps credentials in CATBUS_HOME), so a browser cookie is not required.
	runtimeType, runtimeSpec, err := r.adapterRuntime(instance)
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
	if cookie, _ := cfg["cookie"].(string); runtimeType == RuntimePython && strings.TrimSpace(cookie) == "" {
		return errors.New("instance has no cookie configured; please fill it in the WebUI")
	}

	// Resolve the optional per-instance egress proxy before touching the
	// sandbox: a malformed proxy should fail the start, not the connection.
	proxy, err := proxyFromInstanceConfig(cfg)
	if err != nil {
		return fmt.Errorf("invalid proxy configuration: %w", err)
	}

	// Run from the instance's sandboxed adapter copy, never from the source
	// adapters/ directory. Verify the copy is unmodified before starting.
	sourceDir, err := r.resolveSourceDir(instanceID)
	if err != nil {
		return err
	}
	workDir := r.sandboxDir(instanceID, sourceDir, requiredEntry(runtimeType, runtimeSpec))
	if workDir == "" {
		// Self-healing was attempted but failed. Report something actionable
		// instead of asking the user to recreate the instance (which would
		// discard their configuration).
		return fmt.Errorf("instance sandbox unavailable for platform %q and could not "+
			"be rebuilt: the adapter source was not found. Ensure the built-in adapter "+
			"is available (it is embedded in the binary and extracted to data/adapters), "+
			"then restart the application", platformCode)
	}
	if r.dirs != nil {
		if err := r.dirs.VerifyIntegrity(instanceID); err != nil {
			return fmt.Errorf("instance sandbox integrity check failed: %w", err)
		}
	}
	// The sandbox must hold the launcher: main.py for bridges, runtime.entry
	// for standalone adapters (buildLaunch checks the latter too).
	if entry := requiredEntry(runtimeType, runtimeSpec); runtimeType == RuntimePython {
		script := filepath.Join(workDir, entry)
		if _, err := os.Stat(script); err != nil {
			return fmt.Errorf("python adapter script not found in sandbox: %s (%v)", script, err)
		}
	}

	// Bridge flags (--instance-id/--backend/--token) are shared by both
	// launchers: a standalone adapter uses --config for its payload but still
	// gets the instance identity, and both are harmless extra flags for python.
	args := []string{"main.py", "--instance-id", instanceID, "--backend", r.backendURL}
	if runtimeType == RuntimeStandalone {
		args = args[1:] // the standalone entry replaces main.py
	}
	if r.jwtSecret != "" {
		if token, tErr := r.serviceToken(); tErr == nil {
			args = append(args, "--token", token)
		}
	}
	plan, err := r.buildLaunch(instance, runtimeType, runtimeSpec, workDir, cfg, args)
	if err != nil {
		return err
	}
	args = plan.args
	cmd := exec.Command(plan.bin, args...)
	cmd.Dir = workDir
	// Wire the opt-in egress proxy into the sandbox's environment. Appending to
	// the inherited environment keeps PATH and the adapter's own ESP_* settings
	// intact; the proxy variables are written last so they win.
	cmd.Env = os.Environ()
	if len(plan.env) > 0 {
		cmd.Env = append(cmd.Env, plan.env...)
	}
	if proxyEnv := proxy.Env(); len(proxyEnv) > 0 {
		cmd.Env = append(cmd.Env, proxyEnv...)
	}

	// Redirect the adapter process stdout/stderr to a per-instance log file so
	// its logs can be viewed from the WebUI.
	var (
		stdoutR *os.File
		stdoutW *os.File
		reader  *envelopeReader
	)
	logFile, logErr := r.openLogFile(instanceID)
	if logErr != nil {
		logger.Warn("Failed to open instance log file, falling back to stdout",
			logger.String("instance_id", instanceID),
			logger.String("error", logErr.Error()))
	}
	switch {
	case logFile != nil:
		cmd.Stderr = logFile
		if runtimeType == RuntimeStandalone {
			// Standalone stdout carries the JSON envelopes: tee it into the log
			// so the WebUI still shows the adapter's own log lines, while the
			// reader demultiplexes envelopes back to Command callers.
			reader = newEnvelopeReader()
			var pipeErr error
			stdoutR, stdoutW, pipeErr = os.Pipe()
			if pipeErr != nil {
				return fmt.Errorf("failed to create adapter stdout pipe: %w", pipeErr)
			}
			cmd.Stdout = stdoutW
		} else {
			cmd.Stdout = logFile
		}
	case runtimeType == RuntimeStandalone:
		return errors.New("standalone adapter needs a log file to capture its JSON envelopes")
	default:
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	// A standalone adapter is driven over stdin: keep the pipe open and store
	// the writing end. Bridge (python) adapters dial back over WebSocket, so
	// their stdin stays unused.
	var stdin io.WriteCloser
	if runtimeType == RuntimeStandalone {
		stdin, err = cmd.StdinPipe()
		if err != nil {
			return fmt.Errorf("failed to open adapter stdin: %w", err)
		}
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start adapter process: %w", err)
	}
	if stdoutW != nil {
		// The parent's copy of the write end goes away now; the reader closes
		// the read end when the process exits.
		_ = stdoutW.Close()
		go r.pumpStandaloneOutput(instanceID, stdoutR, logFile, reader)
	}

	// Echo the startup banner and a detailed start record into the instance
	// log file. This guarantees every adapter instance shows the banner in
	// the WebUI log regardless of what the Python script itself prints, and
	// gives the bridge log a full startup context (instance, platform, PID,
	// backend URL, command).
	if logFile != nil {
		ts := time.Now().Format("2006-01-02 15:04:05")
		var sb strings.Builder
		sb.WriteString(adapterBanner)
		fmt.Fprintf(&sb, "%s | INFO    | ============================================================\n", ts)
		fmt.Fprintf(&sb, "%s | INFO    | [bridge] 适配器进程启动 (Adapter process started)\n", ts)
		fmt.Fprintf(&sb, "%s | INFO    | [bridge]   instance_id : %s\n", ts, instanceID)
		fmt.Fprintf(&sb, "%s | INFO    | [bridge]   platform    : %s\n", ts, platformCode)
		fmt.Fprintf(&sb, "%s | INFO    | [bridge]   runtime     : %s\n", ts, runtimeType)
		fmt.Fprintf(&sb, "%s | INFO    | [bridge]   pid         : %d\n", ts, cmd.Process.Pid)
		fmt.Fprintf(&sb, "%s | INFO    | [bridge]   backend     : %s\n", ts, r.backendURL)
		fmt.Fprintf(&sb, "%s | INFO    | [bridge]   workdir     : %s\n", ts, workDir)
		fmt.Fprintf(&sb, "%s | INFO    | [bridge]   proxy       : %s\n", ts, proxy.String())
		fmt.Fprintf(&sb, "%s | INFO    | [bridge]   command     : %s %s\n", ts, plan.bin, strings.Join(args, " "))
		if runtimeType == RuntimeStandalone {
			fmt.Fprintf(&sb, "%s | INFO    | [standalone] 独立式接入器：进程自起，命令经 stdin，结果走 stdout JSON 信封\n", ts)
			fmt.Fprintf(&sb, "%s | INFO    | [standalone] 不等待 WebSocket 回连；把命令写成一行 JSON 从 stdin 送进进程\n", ts)
		} else {
			fmt.Fprintf(&sb, "%s | INFO    | [bridge] 等待适配器连接后端 WebSocket...\n", ts)
		}
		fmt.Fprintf(&sb, "%s | INFO    | ============================================================\n", ts)
		if _, werr := logFile.WriteString(sb.String()); werr != nil {
			logger.Warn("Failed to write startup banner to instance log",
				logger.String("instance_id", instanceID),
				logger.String("error", werr.Error()))
		}
	}

	r.mu.Lock()
	r.procs[instanceID] = cmd
	if stdin != nil {
		r.stdins[instanceID] = stdin
	}
	if reader != nil {
		r.readers[instanceID] = reader
	}
	r.mu.Unlock()

	// Mark the instance as running in the database.
	if err := r.updateInstanceStatus(instanceID, "running"); err != nil {
		logger.Warn("Failed to update instance status to running",
			logger.String("instance_id", instanceID),
			logger.String("error", err.Error()))
	}

	logger.Info("Adapter process started",
		logger.String("instance_id", instanceID),
		logger.String("platform", platformCode),
		logger.String("runtime", runtimeType),
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
		// The process already exited (e.g. it crashed or was killed by a
		// non-running condition such as an invalid cookie). Treat this as an
		// already-stopped instance instead of an error, and make sure the
		// database status reflects "stopped".
		_ = r.updateInstanceStatus(instanceID, "stopped")
		return nil
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
	if in, ok := r.stdins[instanceID]; ok {
		_ = in.Close()
		delete(r.stdins, instanceID)
	}
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

// Command sends one command line to a running standalone adapter and returns
// the adapter's JSON envelope ({"ok", "data", "error"}).
//
// The protocol is deliberately line-oriented: one compact JSON object per line
// on stdin, exactly one JSON envelope line on stdout. The adapter's stdout is
// the instance log, so a reader goroutine demultiplexes envelope lines back
// here while still appending everything (including the adapter's own log
// lines) to the per-instance log.
//
// Bridge (python) adapters do not read stdin, so this reports a clear error
// instead of writing into a pipe nobody drains.
func (r *PythonRunner) Command(instanceID string, command map[string]interface{}, timeout time.Duration) (map[string]interface{}, error) {
	if timeout <= 0 {
		timeout = standaloneCommandTimeout
	}

	r.mu.Lock()
	cmd, running := r.procs[instanceID]
	stdin, hasStdin := r.stdins[instanceID]
	reader := r.readers[instanceID]
	r.mu.Unlock()
	if !running || cmd == nil || cmd.Process == nil || !hasStdin || stdin == nil {
		return nil, errors.New("instance is not running as a standalone adapter")
	}

	id := fmt.Sprintf("%d", time.Now().UnixNano())
	req := map[string]interface{}{}
	for k, v := range command {
		req[k] = v
	}
	req["id"] = id
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')

	waiter := reader.expect(id)

	if _, err := stdin.Write(data); err != nil {
		reader.cancel(id)
		return nil, fmt.Errorf("failed to write command to adapter: %w", err)
	}

	select {
	case resp := <-waiter:
		if resp == nil {
			return nil, errors.New("adapter exited before answering the command")
		}
		return resp, nil
	case <-time.After(timeout):
		reader.cancel(id)
		return nil, fmt.Errorf("adapter did not answer the command within %s", timeout)
	}
}

// standaloneCommandTimeout bounds how long Command waits for an answer.
// A standalone adapter may shell out to a CLI whose cold start (npx fetching
// the package) took ~45s here, so the bound is generous rather than tight.
const standaloneCommandTimeout = 90 * time.Second

// envelopeReader demultiplexes command envelopes out of an adapter's stdout.
type envelopeReader struct {
	mu      sync.Mutex
	pending map[string]chan map[string]interface{}
}

func newEnvelopeReader() *envelopeReader {
	return &envelopeReader{pending: make(map[string]chan map[string]interface{})}
}

// expect registers interest in the answer to one command id.
func (e *envelopeReader) expect(id string) <-chan map[string]interface{} {
	ch := make(chan map[string]interface{}, 1)
	e.mu.Lock()
	e.pending[id] = ch
	e.mu.Unlock()
	return ch
}

func (e *envelopeReader) cancel(id string) {
	e.mu.Lock()
	delete(e.pending, id)
	e.mu.Unlock()
}

// failAll resolves every outstanding waiter with nil (process exited).
func (e *envelopeReader) failAll() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for id, ch := range e.pending {
		ch <- nil
		delete(e.pending, id)
	}
}

// dispatch hands an envelope line to the waiter registered for its id.
func (e *envelopeReader) dispatch(env map[string]interface{}) {
	id, _ := env["id"].(string)
	if id == "" {
		return
	}
	e.mu.Lock()
	ch, ok := e.pending[id]
	if ok {
		delete(e.pending, id)
	}
	e.mu.Unlock()
	if !ok {
		return
	}
	select {
	case ch <- env:
	default:
	}
}

// DeliverStandalone hands one outbound bridge command (send_text / send_image,
// as normalized by the adapter gateway) to a running standalone instance.
//
// It runs asynchronously: the gateway route happens on the WebSocket read loop
// and must not block on a subprocess round-trip. The answer is written to the
// instance log by the adapter itself.
func (r *PythonRunner) DeliverStandalone(instanceID string, command map[string]interface{}) bool {
	r.mu.Lock()
	_, running := r.stdins[instanceID]
	r.mu.Unlock()
	if !running {
		return false
	}
	go func() {
		if _, err := r.Command(instanceID, command, standaloneCommandTimeout); err != nil {
			logger.Warn("Standalone adapter command failed",
				logger.String("instance_id", instanceID),
				logger.String("command_type", fmt.Sprint(command["command_type"])),
				logger.String("error", err.Error()))
		}
	}()
	return true
}

// pumpStandaloneOutput reads the standalone adapter's stdout, appends every line
// to the instance log and hands JSON envelopes to their waiting callers.
//
// The envelope itself is logged too: it is the adapter's answer and belongs in
// the transcript users inspect from the WebUI.
func (r *PythonRunner) pumpStandaloneOutput(instanceID string, stdout *os.File, logFile *os.File, reader *envelopeReader) {
	defer stdout.Close()
	defer func() {
		r.mu.Lock()
		if cur, ok := r.readers[instanceID]; ok && cur == reader {
			delete(r.readers, instanceID)
		}
		r.mu.Unlock()
		reader.failAll()
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if logFile != nil {
			_, _ = logFile.WriteString(line + "\n")
		}
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "{") {
			continue
		}
		var env map[string]interface{}
		if err := json.Unmarshal([]byte(trimmed), &env); err != nil {
			continue
		}
		if kind, _ := env["type"].(string); kind == "tool_catalog" {
			r.mu.Lock()
			sink := r.toolSink
			r.mu.Unlock()
			if sink != nil {
				sink(instanceID, env)
			}
		}
		reader.dispatch(env)
	}
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
	if rd, ok := r.readers[instanceID]; ok {
		rd.failAll()
		delete(r.readers, instanceID)
	}
	if in, ok := r.stdins[instanceID]; ok {
		_ = in.Close()
		delete(r.stdins, instanceID)
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
	// "running" means the process is (re)starting: stamp the first heartbeat so
	// the Monitoring page does not show an em dash for a live adapter.
	if status == "running" {
		now := time.Now()
		instance.LastHeartbeat = &now
	}
	return r.instances.Update(instance)
}
