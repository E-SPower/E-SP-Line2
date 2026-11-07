package server

import (
	"os"
	"testing"

	"github.com/e-spl/e-sp-line2/pkg/logger"
)

// repository.NewRepositories logs during construction, so the logger has to be
// live before any test builds a server. Same reason internal/adapter has this.
func TestMain(m *testing.M) {
	logger.Init("error")
	code := m.Run()
	_ = logger.Sync()
	os.Exit(code)
}
