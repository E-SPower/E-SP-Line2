package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/e-spl/e-sp-line2/internal/models"
)

// Adapter runtime types.
//
// `runtime_type` is not a free-form label in adapter.yaml: it selects HOW the
// runner launches the adapter process. The default, "python", is the historical
// bridge model (the process dials back to /ws/adapter). "standalone" launches a
// self-contained process that is driven by commands over stdin and answers with
// a JSON envelope on stdout — see docs/developer-guide/adapter-runtime.md.
const (
	// RuntimePython is the legacy bridge model: `python main.py --instance-id ...`.
	RuntimePython = "python"
	// RuntimeStandalone launches `runtime.entry` with `runtime.interpreter` and
	// talks to it over stdin/stdout instead of the bridge WebSocket.
	RuntimeStandalone = "standalone"
)

// RuntimeSpec is the `runtime:` block of an adapter.yaml for the standalone
// runtime. It lives in the adapter catalog, next to runtime_type, so the
// launcher, the dependency self-check and the catalog endpoint all read the
// same declaration.
type RuntimeSpec struct {
	// Interpreter is the executable used to run Entry (default: the configured
	// node binary, i.e. "node").
	Interpreter string `json:"interpreter,omitempty" yaml:"interpreter,omitempty"`
	// Entry is the script inside the adapter directory (e.g. "main.mjs").
	Entry string `json:"entry,omitempty" yaml:"entry,omitempty"`
	// Args are extra literal arguments appended after the runner's own flags.
	Args []string `json:"args,omitempty" yaml:"args,omitempty"`
}

// ValidRuntimeType reports whether rt selects a launcher the runner implements.
// node/go stay accepted for adapters that only carry the label, but only
// python and standalone have a dispatch path.
func ValidRuntimeType(rt string) bool {
	switch strings.ToLower(strings.TrimSpace(rt)) {
	case RuntimePython, RuntimeStandalone, "node", "go":
		return true
	}
	return false
}

// validateRuntimeSpec checks a standalone runtime declaration at catalog load
// time, so a broken adapter.yaml fails the scan instead of the first start.
func validateRuntimeSpec(name, rt string, spec RuntimeSpec) error {
	if strings.ToLower(strings.TrimSpace(rt)) != RuntimeStandalone {
		return nil
	}
	if strings.TrimSpace(spec.Entry) == "" {
		return fmt.Errorf("adapter.yaml %s: runtime_type standalone requires runtime.entry", name)
	}
	entry := filepath.Clean(spec.Entry)
	if filepath.IsAbs(entry) || strings.HasPrefix(entry, "..") {
		return fmt.Errorf("adapter.yaml %s: runtime.entry must stay inside the adapter directory: %s", name, spec.Entry)
	}
	for _, a := range spec.Args {
		if strings.ContainsRune(a, '\x00') {
			return fmt.Errorf("adapter.yaml %s: runtime.args contains a NUL byte", name)
		}
	}
	return nil
}

// adapterRuntime resolves the runtime type and spec of an instance's adapter.
// The catalog (adapter.yaml) is authoritative; a DB adapter record is only the
// backwards-compatible fallback for adapters created through the API.
func (r *PythonRunner) adapterRuntime(instance *models.AdapterInstance) (string, RuntimeSpec, error) {
	if r.catalog != nil {
		if a, ok := r.catalog.Get(instance.AdapterID); ok {
			rt := strings.ToLower(strings.TrimSpace(a.RuntimeType))
			if rt == "" {
				rt = RuntimePython
			}
			return rt, a.Runtime, nil
		}
	}
	adapter, err := r.adapters.FindByID(instance.AdapterID)
	if err != nil {
		return "", RuntimeSpec{}, fmt.Errorf("adapter not found for instance %s: %w", instance.ID, err)
	}
	rt := strings.ToLower(strings.TrimSpace(adapter.RuntimeType))
	if rt == "" {
		rt = RuntimePython
	}
	return rt, RuntimeSpec{}, nil
}

// launchPlan is one concrete process invocation.
type launchPlan struct {
	bin  string
	args []string
	env  []string
}

// buildLaunch turns a resolved runtime into the command line for an instance.
//
// workDir is the instance sandbox. For the standalone runtime the instance
// config is written to <instance_dir>/config.runtime.json (0600, outside the
// sandboxed adapter copy so the integrity manifest is unaffected) and handed to
// the process with --config, keeping secrets out of argv.
func (r *PythonRunner) buildLaunch(
	instance *models.AdapterInstance,
	runtimeType string,
	spec RuntimeSpec,
	workDir string,
	cfg map[string]interface{},
	baseArgs []string,
) (launchPlan, error) {
	switch runtimeType {
	case RuntimePython:
		return launchPlan{bin: r.python, args: baseArgs}, nil

	case RuntimeStandalone:
		entry := filepath.Clean(strings.TrimSpace(spec.Entry))
		if entry == "" || entry == "." {
			return launchPlan{}, errors.New("standalone adapter declares no runtime.entry in adapter.yaml")
		}
		script := filepath.Join(workDir, entry)
		if fi, err := os.Stat(script); err != nil || fi.IsDir() {
			return launchPlan{}, fmt.Errorf("standalone adapter entry not found in sandbox: %s", script)
		}
		interpreter := strings.TrimSpace(spec.Interpreter)
		if interpreter == "" {
			interpreter = r.nodeBin()
		}
		if interpreter == "" {
			return launchPlan{}, errors.New("no interpreter for standalone adapter (set adapter.node_bin or runtime.interpreter)")
		}

		configPath, err := r.writeRuntimeConfig(instance.ID, cfg)
		if err != nil {
			return launchPlan{}, err
		}

		args := []string{entry}
		args = append(args, baseArgs...)
		args = append(args, "--config", configPath)
		args = append(args, spec.Args...)

		// Announce the runtime to the process so an adapter can branch on it
		// (and so a shell-invoked CLI can be found without guessing).
		env := []string{
			"ESP_ADAPTER_RUNTIME=" + RuntimeStandalone,
			"ESP_INSTANCE_ID=" + instance.ID,
			"ESP_BACKEND_URL=" + r.backendURL,
		}
		if v, ok := cfg["catbus_bin"].(string); ok && strings.TrimSpace(v) != "" {
			env = append(env, "CATBUS_BIN="+strings.TrimSpace(v))
		}
		if v, ok := cfg["catbus_home"].(string); ok && strings.TrimSpace(v) != "" {
			env = append(env, "CATBUS_HOME="+strings.TrimSpace(v))
		}
		return launchPlan{bin: interpreter, args: args, env: env}, nil

	default:
		return launchPlan{}, fmt.Errorf("runtime_type %q is declared but not implemented by the runner", runtimeType)
	}
}

// nodeBin returns the configured interpreter for standalone adapters.
func (r *PythonRunner) nodeBin() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.node != "" {
		return r.node
	}
	return "node"
}

// SetNodeBin sets the interpreter used for standalone adapters (default node).
func (r *PythonRunner) SetNodeBin(bin string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.node = bin
}

// writeRuntimeConfig persists the instance config for the standalone process.
// It is rewritten on every start so a stale config never leaks into a later run.
func (r *PythonRunner) writeRuntimeConfig(instanceID string, cfg map[string]interface{}) (string, error) {
	if r.dirs == nil {
		return "", errors.New("instance sandbox manager is not attached")
	}
	dir := filepath.Dir(r.dirs.AdapterDir(instanceID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if cfg == nil {
		cfg = map[string]interface{}{}
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "config.runtime.json")
	// The process runs with its working directory set to the sandbox, so the
	// path must be absolute or the adapter resolves it against the wrong dir.
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(abs, data, 0o600); err != nil {
		return "", err
	}
	return abs, nil
}
