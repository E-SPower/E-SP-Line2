//go:build desktop

// Package webui embeds the built frontend (web/dist) into the binary so the
// desktop build can serve the WebUI without any external static files.
//
// This file is only compiled with the `desktop` build tag:
//
//	go build -tags desktop .
//
// Web views on Linux and Windows need a frontend bundle available at build
// time, so `make build-frontend` must run before compiling a desktop binary.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:dist
var embeddedFS embed.FS

// Available reports whether an embedded frontend bundle is compiled in.
const Available = true

// FS returns the embedded frontend bundle rooted at web/dist.
func FS() (fs.FS, bool) {
	sub, err := fs.Sub(embeddedFS, "dist")
	if err != nil {
		return nil, false
	}
	return sub, true
}

// Handler returns an http.Handler serving the embedded SPA.
func Handler() http.Handler {
	sub, ok := FS()
	if !ok {
		return http.NotFoundHandler()
	}
	return NewSPAHandler(sub)
}
