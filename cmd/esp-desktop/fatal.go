//go:build desktop

package main

import (
	"fmt"
	"os"
)

// fatal reports a fatal startup error in a user-visible way and exits.
//
// The desktop build is normally started by double-clicking the executable, so
// a message written only to stderr would disappear with the console window.
// showFatalDialog / waitForAck are provided per-platform by fatal_windows.go
// and fatal_other.go.
func fatal(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, "FATAL:", msg)
	showFatalDialog("E-SP-Line2 启动失败", msg)
	waitForAck()
	os.Exit(1)
}
