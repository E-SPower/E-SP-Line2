package server

import (
	"net/http"
	"testing"
	"time"

	"github.com/e-spl/e-sp-line2/internal/config"
)

// TestNewHTTPServerWiresTimeouts is M15's failing case: the struct used to be
// `{Addr, Handler}` only, so every deadline stayed at net/http's "none" default
// and one slow client could hold a connection open forever.
func TestNewHTTPServerWiresTimeouts(t *testing.T) {
	cfg := &config.Config{Port: 8080}
	cfg.Server.ReadHeaderTimeout = 10
	cfg.Server.ReadTimeout = 60
	cfg.Server.WriteTimeout = 120
	cfg.Server.IdleTimeout = 120

	srv := newHTTPServer(cfg, http.NewServeMux())

	if got, want := srv.ReadHeaderTimeout, 10*time.Second; got != want {
		t.Fatalf("ReadHeaderTimeout = %v, want %v (Slowloris guard)", got, want)
	}
	if got, want := srv.ReadTimeout, 60*time.Second; got != want {
		t.Fatalf("ReadTimeout = %v, want %v", got, want)
	}
	if got, want := srv.WriteTimeout, 120*time.Second; got != want {
		t.Fatalf("WriteTimeout = %v, want %v", got, want)
	}
	if got, want := srv.IdleTimeout, 120*time.Second; got != want {
		t.Fatalf("IdleTimeout = %v, want %v", got, want)
	}
	if srv.Addr != ":8080" {
		t.Fatalf("Addr = %q, want :8080", srv.Addr)
	}
}

// A configured 0 means "no timeout", which net/http expresses as a zero
// duration — the value must not be turned into something else.
func TestNewHTTPServerZeroMeansNoTimeout(t *testing.T) {
	srv := newHTTPServer(&config.Config{}, http.NewServeMux())
	if srv.ReadHeaderTimeout != 0 || srv.ReadTimeout != 0 || srv.WriteTimeout != 0 || srv.IdleTimeout != 0 {
		t.Fatalf("zero config produced timeouts: %+v", srv)
	}
}
