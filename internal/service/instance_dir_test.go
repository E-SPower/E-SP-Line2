package service

import (
	"path/filepath"
	"testing"
)

// TestDataRootDir pins the data-root derivation for every adapters directory
// shape the app can produce. A regression here silently breaks the instance
// sandbox layout (data/instances/...), which surfaces to users as
// "instance sandbox is not initialized; please recreate the instance".
func TestDataRootDir(t *testing.T) {
	cases := []struct {
		name     string
		adapters string
		want     string
	}{
		// External adapters (developer override / pre-embed layout).
		{"external", "adapters", filepath.Join("adapters", "..", "data")},
		{"external abs", "/opt/app/adapters", filepath.Join("/opt/app/adapters", "..", "data")},

		// Embedded bundle extracted into data/adapters.
		{"embedded relative", "data/adapters", "data"},
		{"embedded trailing slash", "data/adapters/", "data"},
		{"embedded abs", "/opt/app/data/adapters", "/opt/app/data"},
		{"data itself", "data", "data"},
		{"nested under data", "data/adapters/nested", "data"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := dataRootDir(tc.adapters)
			want := filepath.Clean(tc.want)
			if filepath.Clean(got) != want {
				t.Fatalf("dataRootDir(%q) = %q, want %q", tc.adapters, got, want)
			}
		})
	}
}

// TestDataInstancesDir ensures the sandbox root is always <data>/instances and
// never the buggy <adapters>/../data/instances form when adapters live in data.
func TestDataInstancesDir(t *testing.T) {
	cases := []struct {
		adapters string
		want     string
	}{
		{"adapters", filepath.Join("data", "instances")},
		{"data/adapters", filepath.Join("data", "instances")},
		{"/opt/app/data/adapters", "/opt/app/data/instances"},
	}

	for _, tc := range cases {
		got := dataInstancesDir(tc.adapters)
		if filepath.Clean(got) != filepath.Clean(tc.want) {
			t.Fatalf("dataInstancesDir(%q) = %q, want %q", tc.adapters, got, tc.want)
		}
	}
}

// TestInstanceDirManagerPaths verifies the manager's public path helpers stay
// consistent with the data root (the manifest and sandbox must live together).
func TestInstanceDirManagerPaths(t *testing.T) {
	for _, adaptersDir := range []string{"adapters", "data/adapters"} {
		m := NewInstanceDirManager(adaptersDir)

		wantRoot := filepath.Join("data", "instances")
		if filepath.Clean(m.root) != filepath.Clean(wantRoot) {
			t.Fatalf("adaptersDir=%q root=%q, want %q", adaptersDir, m.root, wantRoot)
		}

		id := "abc123"
		wantAdapter := filepath.Join(wantRoot, id, "adapter")
		if got := m.AdapterDir(id); filepath.Clean(got) != filepath.Clean(wantAdapter) {
			t.Fatalf("adaptersDir=%q AdapterDir=%q, want %q", adaptersDir, got, wantAdapter)
		}
		if got := m.ManifestPath(id); filepath.Clean(got) != filepath.Clean(filepath.Join(wantRoot, id, "manifest.json")) {
			t.Fatalf("adaptersDir=%q ManifestPath=%q", adaptersDir, got)
		}
		if got := m.StatePath(id); filepath.Clean(got) != filepath.Clean(filepath.Join(wantRoot, id, "state.json")) {
			t.Fatalf("adaptersDir=%q StatePath=%q", adaptersDir, got)
		}

		// Dependency markers and logs must also live under the data root.
		d := NewDependencyInstaller("python3", adaptersDir)
		if got := d.dependencyMarkerDir(); filepath.Clean(got) != filepath.Clean(filepath.Join("data", "deps")) {
			t.Fatalf("adaptersDir=%q markerDir=%q, want data/deps", adaptersDir, got)
		}
		if got := d.logDir(); filepath.Clean(got) != filepath.Clean(filepath.Join("data", "logs")) {
			t.Fatalf("adaptersDir=%q logDir=%q, want data/logs", adaptersDir, got)
		}
	}
}

// TestPythonRunnerLogDir mirrors the runner's log path derivation, which must
// follow the same data root as everything else.
func TestPythonRunnerLogDir(t *testing.T) {
	for _, adaptersDir := range []string{"adapters", "data/adapters"} {
		r := &PythonRunner{dir: adaptersDir}
		if got := r.logDir(); filepath.Clean(got) != filepath.Clean(filepath.Join("data", "logs")) {
			t.Fatalf("adaptersDir=%q logDir=%q, want data/logs", adaptersDir, got)
		}
	}
}
