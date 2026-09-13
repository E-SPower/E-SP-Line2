//go:build desktop

package main

import (
	"fmt"
	"net"
)

// pickFreePort returns a usable loopback TCP port, starting from preferred and
// walking forward until one is available.
//
// Returning whether the port changed lets the caller surface a helpful log
// line instead of silently behaving differently from the configured value.
//
// The check binds and immediately releases the port. There is an inherent race
// between this probe and the real listener, but for a desktop app the window is
// tiny and the alternative (starting the server and reacting to its error after
// having already opened a window) is far worse.
func pickFreePort(preferred int) (port int, changed bool, err error) {
	if preferred <= 0 {
		preferred = 8080
	}

	const attempts = 20
	for i := 0; i < attempts; i++ {
		p := preferred + i
		if p > 65535 {
			break
		}
		l, lerr := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if lerr == nil {
			_ = l.Close()
			return p, p != preferred, nil
		}
	}
	return 0, false, fmt.Errorf("no free port in range %d-%d", preferred, preferred+attempts-1)
}
