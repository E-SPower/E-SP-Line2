//go:build !windows

package main

// On non-Windows platforms a fatal startup error is simply printed to stderr,
// which stays visible in the terminal that launched the process.

import (
	"bufio"
	"fmt"
	"os"
)

// showFatalDialog has no GUI dialog outside Windows.
func showFatalDialog(title, msg string) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", title, msg)
}

// waitForAck only pauses when an interactive stdin is attached, so scripts and
// CI runs are never blocked waiting for input.
func waitForAck() {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return
	}
	if fi.Mode()&os.ModeCharDevice == 0 {
		return // stdin is a pipe/file: do not block
	}
	fmt.Fprint(os.Stderr, "按回车键退出...")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}
