package webui

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// NewSPAHandler wraps a static filesystem (typically the built frontend) with
// single-page-application history fallback: any path that does not resolve to
// a real file is served as index.html so client-side routing keeps working on
// deep links and refreshes.
//
// Static assets under /assets/... are served verbatim; API routes must be
// registered before this handler so they are never shadowed.
//
// Note: the fallback writes index.html content directly instead of rewriting
// r.URL.Path to "/index.html". http.FileServer canonicalizes a request for
// "/index.html" into a 301 redirect to "./", which would turn a naive rewrite
// into an infinite redirect loop.
func NewSPAHandler(root fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if clean == "" || clean == "." {
			clean = "index.html"
		}

		// Serve real files (and directories) as-is.
		if info, err := fs.Stat(root, clean); err == nil {
			if !info.IsDir() {
				fileServer.ServeHTTP(w, r)
				return
			}
			// Directory request: fall through to the SPA entrypoint.
		}

		// Unknown/deep path: serve the SPA entrypoint directly.
		data, err := fs.ReadFile(root, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})
}
