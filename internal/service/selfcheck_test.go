package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/e-spl/e-sp-line2/pkg/logger"
)

// digestAdapterState snapshots an install state so a test can assert on it.
// TestStandaloneSelfCheckRunsAdapterEntry: for a standalone adapter the
// dependency path is the adapter's own --self-check, not pip.
func TestStandaloneSelfCheckRunsAdapterEntry(t *testing.T) {
	logger.Init("error")
	adaptersDir := t.TempDir()
	dir := filepath.Join(adaptersDir, "stub")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// The stub records its argv and exits 0.
	entry := "#!/bin/sh\necho \"argv=$*\"\nexit 0\n"
	if err := os.WriteFile(filepath.Join(dir, "main.sh"), []byte(entry), 0o755); err != nil {
		t.Fatal(err)
	}

	installer := NewDependencyInstaller("python3", adaptersDir)
	// sh, not node, so the stub is directly executable.
	installer.adoptRuntime("i1", RuntimeStandalone, RuntimeSpec{Interpreter: "sh", Entry: "main.sh"}, `{"catbus_bin":"/x"}`)
	installer.InstallDependencies("i1", dir, "stub", nil)

	deadline := 0
	for {
		state, ok := installer.GetState("i1")
		if ok && state.Status != "installing" {
			break
		}
		deadline++
		if deadline > 500 {
			t.Fatal("self-check never finished")
		}
		time.Sleep(10 * time.Millisecond)
	}
	state, _ := installer.GetState("i1")
	if state.Status != "done" {
		t.Fatalf("state = %+v", state)
	}
	log, _ := os.ReadFile(filepath.Join(dataRootDir(adaptersDir), "logs", "i1.log"))
	if !strings.Contains(string(log), "--self-check") {
		t.Fatalf("self-check was not invoked:\n%s", log)
	}
	// $0 is the script itself, so argv starts at the first flag.
	if !strings.Contains(string(log), "argv=--self-check --instance-id i1") {
		t.Fatalf("unexpected argv:\n%s", log)
	}
	if !strings.Contains(string(log), "--config /") {
		t.Fatalf("config path not absolute:\n%s", log)
	}
	// The adapter output itself is in the transcript.
	if !strings.Contains(string(log), "argv=") {
		t.Fatalf("adapter output missing:\n%s", log)
	}
}

// TestStandaloneSelfCheckFailureBlocks: a failing self-check must leave the
// instance in "error" with the adapter's message, so it is not startable.
func TestStandaloneSelfCheckFailure(t *testing.T) {
	logger.Init("error")
	adaptersDir := t.TempDir()
	dir := filepath.Join(adaptersDir, "stub")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.sh"),
		[]byte("#!/bin/sh\necho '2026-01-01 00:00:00 | ERROR    | [selfcheck] 缺少 catbus' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	installer := NewDependencyInstaller("python3", adaptersDir)
	installer.adoptRuntime("i2", RuntimeStandalone, RuntimeSpec{Interpreter: "sh", Entry: "main.sh"}, "")

	var got string
	installer.InstallDependencies("i2", dir, "stub", func(_ string, s string) error { got = s; return nil })
	for i := 0; i < 500; i++ {
		if s, ok := installer.GetState("i2"); ok && s.Status != "installing" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	state, _ := installer.GetState("i2")
	if state.Status != "failed" {
		t.Fatalf("state = %+v, want failed", state)
	}
	if got != "stopped" {
		t.Fatalf("updateStatus = %q, want stopped", got)
	}
	if !strings.Contains(state.Message, "缺少 catbus") {
		t.Fatalf("state message lost the adapter explanation: %q", state.Message)
	}
}

// TestStandaloneSelfCheckLogLinesAreNotDoublePrefixed pins the log contract:
// adapter lines already in `时间 | 级别 | [tag] 消息` stay as they are, and a
// JSON envelope is leveled by its ok flag (not by the literal `"error":null`).
func TestStandaloneSelfCheckLogLinesAreNotDoublePrefixed(t *testing.T) {
	logger.Init("error")
	adaptersDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(adaptersDir, "log"), 0o755); err != nil {
		t.Fatal(err)
	}
	d := NewDependencyInstaller("python3", adaptersDir)
	instanceID := "i3"
	d.appendLog(instanceID, "2026-11-07 13:00:00 | INFO     | [selfcheck] node: ok")
	d.appendLog(instanceID, `{"id":"self-check","ok":true,"data":{},"error":null}`)
	d.appendLog(instanceID, `{"id":"self-check","ok":false,"data":null,"error":{"code":"ERROR"}}`)
	d.appendLog(instanceID, "plain install line")

	data, err := os.ReadFile(filepath.Join(dataRootDir(adaptersDir), "logs", instanceID+".log"))
	if err != nil {
		t.Fatal(err)
	}
	log := string(data)
	if strings.Contains(log, "| INFO    | 2026-11-07 13:00:00") {
		t.Fatalf("adapter line was prefixed twice:\n%s", log)
	}
	if !strings.Contains(log, `| INFO    | {"id":"self-check","ok":true`) {
		t.Fatalf("successful envelope not leveled INFO (got the `error:null` trap?):\n%s", log)
	}
	if !strings.Contains(log, `| ERROR   | {"id":"self-check","ok":false`) {
		t.Fatalf("failed envelope not leveled ERROR:\n%s", log)
	}
	if !strings.Contains(log, "| INFO    | plain install line") {
		t.Fatalf("plain installer line lost its prefix:\n%s", log)
	}
}
