//go:build desktop

package main

import (
	"fmt"
	"net"
	"testing"
)

// TestPickFreePortReturnsPreferredWhenFree verifies the common case: the
// configured port is available, so the app keeps using it.
func TestPickFreePortReturnsPreferredWhenFree(t *testing.T) {
	// Grab a free port, release it, then ask for it.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	free := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()

	port, changed, err := pickFreePort(free)
	if err != nil {
		t.Fatalf("pickFreePort failed: %v", err)
	}
	if port != free {
		t.Fatalf("expected preferred port %d, got %d", free, port)
	}
	if changed {
		t.Fatal("changed should be false when the preferred port is free")
	}
}

// TestPickFreePortSkipsBusyPort is the regression test for the reported bug:
// launching a second instance previously failed with
// "bind: Only one usage of each socket address ... permitted".
func TestPickFreePortSkipsBusyPort(t *testing.T) {
	// Occupy a port for the duration of the test.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	busy := l.Addr().(*net.TCPAddr).Port

	port, changed, err := pickFreePort(busy)
	if err != nil {
		t.Fatalf("pickFreePort failed: %v", err)
	}
	if port == busy {
		t.Fatalf("pickFreePort returned the busy port %d", busy)
	}
	if !changed {
		t.Fatal("changed should be true when the preferred port is busy")
	}
	if port <= busy {
		t.Fatalf("expected a port above %d, got %d", busy, port)
	}

	// The chosen port must actually be bindable.
	nl, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("returned port %d is not free: %v", port, err)
	}
	_ = nl.Close()
}

// TestPickFreePortInvalidInput ensures a zero/negative preference falls back to
// the documented default instead of failing.
func TestPickFreePortInvalidInput(t *testing.T) {
	port, _, err := pickFreePort(0)
	if err != nil {
		t.Fatalf("pickFreePort(0) failed: %v", err)
	}
	if port <= 0 {
		t.Fatalf("expected a valid port, got %d", port)
	}
}
