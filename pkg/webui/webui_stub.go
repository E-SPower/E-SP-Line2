//go:build !desktop

// Package webui provides optional embedding of the built frontend.
//
// In the default (web/server) build the frontend is NOT embedded: web/dist is
// served by an external static server / reverse proxy. Only the `desktop`
// build tag pulls in the embedded bundle (see webui_desktop.go).
package webui

import (
	"io/fs"
	"net/http"
)

// Available is false in non-desktop builds.
const Available = false

// FS returns nil in non-desktop builds.
func FS() (fs.FS, bool) { return nil, false }

// Handler returns a 404 handler; the frontend is served externally.
func Handler() http.Handler { return http.NotFoundHandler() }
