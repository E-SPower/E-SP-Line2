package platformspec

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// 改 matrix.json 后忘了重跑生成脚本时，这里会失败。
func TestGeneratedFilesAreCurrent(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	out, err := exec.Command("python3", filepath.Join("..", "..", "scripts", "check_generated.py"),
		filepath.Join("..", "..", "matrix.json")).CombinedOutput()
	if err != nil {
		t.Fatalf("generated files are stale:\n%s", out)
	}
}
