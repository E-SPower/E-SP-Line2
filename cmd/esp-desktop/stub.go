//go:build !desktop

// Command esp-desktop is the native desktop build of E-SP-Line2.
//
// This stub keeps the package buildable without the `desktop` tag so that
// `go build ./...` and `go vet ./...` succeed on the default configuration.
// The real entrypoint lives in main.go and requires:
//
//	go build -tags desktop ./cmd/esp-desktop
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "E-SP-Line2 desktop build requires the `desktop` build tag:")
	fmt.Fprintln(os.Stderr, "    go build -tags desktop ./cmd/esp-desktop")
	fmt.Fprintln(os.Stderr, "or use the build script:")
	fmt.Fprintln(os.Stderr, "    ./scripts/build.sh --desktop")
	os.Exit(1)
}
