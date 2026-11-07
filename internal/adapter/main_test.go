package adapter

import (
	"os"
	"testing"

	"github.com/e-spl/e-sp-line2/pkg/logger"
)

// registry.Register 会打日志，测试里先把 logger 初始化掉。
func TestMain(m *testing.M) {
	logger.Init("error")
	code := m.Run()
	logger.Sync()
	os.Exit(code)
}
