//go:build desktop && !windows

package main

import (
	"bufio"
	"fmt"
	"os"
)

// showFatalDialog prints the error; there is no portable GUI dialog.
func showFatalDialog(title, msg string) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", title, msg)
}

// waitForAck pauses only when an interactive terminal is attached, to avoid
// blocking scripts and CI.
func waitForAck() {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return
	}
	if fi.Mode()&os.ModeCharDevice == 0 {
		return
	}
	fmt.Fprint(os.Stderr, "按回车键退出...")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}
