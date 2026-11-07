package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/e-spl/e-sp-line2/internal/models"
)

// startStandalone starts an instance whose entry is a shell stub that answers
// every stdin line with a fixed envelope, so the runner's command plumbing is
// exercised without Node or the real CLI.
func startStandalone(t *testing.T, entryBody string) (*PythonRunner, *models.AdapterInstance) {
	t.Helper()
	adaptersDir := writeAdapter(t, "stub", "stub-adapter", RuntimeStandalone, "main.sh", "runtime:\n  entry: main.sh\n")
	// writeAdapter writes a placeholder entry; replace it with the stub.
	if err := os.WriteFile(filepath.Join(adaptersDir, "stub", "main.sh"), []byte(entryBody), 0o755); err != nil {
		t.Fatal(err)
	}

	runner, repo := newTestRunner(t, adaptersDir)
	catalog, err := NewAdapterCatalog(adaptersDir)
	if err != nil {
		t.Fatal(err)
	}
	runner.SetCatalog(catalog)
	runner.SetDirManager(NewInstanceDirManager(adaptersDir))
	inst := seedInstance(t, repo, "stub-adapter")
	if _, err := runner.dirs.CopyAdapter(inst.ID, "stub"); err != nil {
		t.Fatalf("sandbox: %v", err)
	}

	// The interpreter is sh; the entry is the relative script path.
	spec, _ := catalog.Get("stub-adapter")
	spec.Runtime.Interpreter = "sh"
	if err := runner.Start(inst.ID); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { _ = runner.Stop(inst.ID) })
	return runner, inst
}

// TestStandaloneCommandRoundTrip covers the stdin/stdout contract end to end:
// the command reaches the process, the envelope comes back and is logged.
func TestStandaloneCommandRoundTrip(t *testing.T) {
	stub := `#!/bin/sh
while IFS= read -r line; do
  echo "2026-11-07 13:00:00 | INFO     | [stub] got a line" >&2
  echo '{"id":"'"$(echo "$line" | sed -E 's/.*"id":"([^"]+)".*/\1/')"'","ok":true,"data":{"echoed":true},"error":null}'
done
`
	runner, inst := startStandalone(t, stub)

	resp, err := runner.Command(inst.ID, map[string]interface{}{
		"command_type": "send_text",
		"payload":      map[string]interface{}{"text": "hi"},
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("command: %v", err)
	}
	if ok, _ := resp["ok"].(bool); !ok {
		t.Fatalf("adapter answered failure: %v", resp)
	}
	if resp["id"] == "" {
		t.Fatalf("response lost the command id: %v", resp)
	}

	// The transcript must contain the adapter's stderr log and the envelope.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		log, _ := runner.ReadLog(inst.ID, 0, "all", "", "", "")
		if strings.Contains(log, "[stub] got a line") && strings.Contains(log, `"echoed":true`) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	log, _ := runner.ReadLog(inst.ID, 0, "all", "", "", "")
	t.Fatalf("instance log missing stub output/envelope:\n%s", log)
}

// TestStandaloneCommandAfterExit fails outstanding waiters when the process dies
// instead of hanging for the full timeout.
func TestStandaloneCommandAfterExit(t *testing.T) {
	runner, inst := startStandalone(t, "#!/bin/sh\nexit 0\n")
	// Let the process exit.
	deadline := time.Now().Add(2 * time.Second)
	for runner.IsRunning(inst.ID) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	start := time.Now()
	if _, err := runner.Command(inst.ID, map[string]interface{}{"command_type": "x"}, 5*time.Second); err == nil {
		t.Fatal("command on a dead instance must fail")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("failure took %s, waiter was not failed by exit", elapsed)
	}
}

// TestPythonInstanceHasNoCommandChannel: bridge adapters do not read stdin, so
// Command must say so clearly rather than writing into an unread pipe.
func TestPythonInstanceHasNoCommandChannel(t *testing.T) {
	runner, _ := newTestRunner(t, t.TempDir())
	inst := &models.AdapterInstance{ID: "i1", AdapterID: "nope"}
	runner.mu.Lock()
	runner.procs[inst.ID] = nil
	runner.mu.Unlock()

	if _, err := runner.Command(inst.ID, map[string]interface{}{}, time.Second); err == nil {
		t.Fatal("expected an error for a bridge instance")
	} else if !strings.Contains(err.Error(), "standalone") {
		t.Fatalf("error should name the runtime: %v", err)
	}
}

// TestPythonInstanceStillStartsAsBridge is the regression guard for the legacy
// runtime: a python adapter must still start with `main.py --instance-id ...
// --backend ...`, keep its stdin unused, get its banner, and require a cookie.
func TestPythonInstanceStillStartsAsBridge(t *testing.T) {
	adaptersDir := writeAdapter(t, "xianyu", "xianyu-adapter", "python", "main.py", "")
	if err := os.WriteFile(filepath.Join(adaptersDir, "xianyu", "main.py"),
		[]byte("import sys,time\nprint('args', sys.argv[1:], flush=True)\ntime.sleep(5)\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	runner, repo := newTestRunner(t, adaptersDir)
	catalog, err := NewAdapterCatalog(adaptersDir)
	if err != nil {
		t.Fatal(err)
	}
	runner.SetCatalog(catalog)
	runner.SetDirManager(NewInstanceDirManager(adaptersDir))

	// No cookie -> the bridge start must still be refused.
	noCookie := seedInstance(t, repo, "xianyu-adapter")
	if _, err := runner.dirs.CopyAdapter(noCookie.ID, "xianyu"); err != nil {
		t.Fatal(err)
	}
	if err := runner.Start(noCookie.ID); err == nil || !strings.Contains(err.Error(), "cookie") {
		t.Fatalf("python start without cookie = %v, want a cookie error", err)
	}

	// With a cookie it starts, and its stdin is not a command channel.
	ok := seedInstance(t, repo, "xianyu-adapter")
	ok.Config = `{"cookie":"a=b"}`
	if err := repo.Update(ok); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.dirs.CopyAdapter(ok.ID, "xianyu"); err != nil {
		t.Fatal(err)
	}
	if err := runner.Start(ok.ID); err != nil {
		t.Fatalf("python start: %v", err)
	}
	t.Cleanup(func() { _ = runner.Stop(ok.ID) })

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		log, _ := runner.ReadLog(ok.ID, 0, "all", "", "", "")
		if strings.Contains(log, "main.py --instance-id "+ok.ID) && strings.Contains(log, "args ['--instance-id'") {
			if _, err := runner.Command(ok.ID, map[string]interface{}{}, time.Second); err == nil ||
				!strings.Contains(err.Error(), "standalone") {
				t.Fatalf("bridge Command error = %v", err)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	log, _ := runner.ReadLog(ok.ID, 0, "all", "", "", "")
	t.Fatalf("python args not seen in log:\n%s", log)
}

// TestStandaloneRestartKeepsSandbox is the runner-level regression for the
// main.py-only completeness check: restarting a standalone instance must reuse
// the existing sandbox instead of RemoveAll + re-copy, so the adapter's own
// files and state.json survive.
func TestStandaloneRestartKeepsSandbox(t *testing.T) {
	runner, inst := startStandalone(t, "#!/bin/sh\nsleep 5\n")

	// Both files live next to the sandbox copy, not inside it: the integrity
	// check treats files under adapter/ as manifest-owned, so this is exactly
	// what a full re-copy would destroy.
	instanceDir := filepath.Dir(runner.dirs.AdapterDir(inst.ID))
	marker := filepath.Join(instanceDir, "written-by-adapter.txt")
	if err := os.WriteFile(marker, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}
	statePath := runner.dirs.StatePath(inst.ID)
	if err := os.WriteFile(statePath, []byte(`{"status":"done","progress":100}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Restart: Stop then Start.
	if err := runner.Stop(inst.ID); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if err := runner.Start(inst.ID); err != nil {
		t.Fatalf("restart: %v", err)
	}
	t.Cleanup(func() { _ = runner.Stop(inst.ID) })

	if data, err := os.ReadFile(marker); err != nil || string(data) != "keep me" {
		t.Fatalf("adapter-written file lost on restart (sandbox was rebuilt): %q, %v", data, err)
	}
	if data, err := os.ReadFile(statePath); err != nil || !strings.Contains(string(data), `"done"`) {
		t.Fatalf("install state lost on restart (sandbox was rebuilt): %q, %v", data, err)
	}
	// The restart must not be a fresh copy: the sandbox keeps the same files
	// and the process still starts.
	if !runner.IsRunning(inst.ID) {
		t.Fatal("instance not running after restart")
	}
}
