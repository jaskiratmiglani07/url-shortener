package api

import (
	"net/http"
	"strings"
)

// NewRouter registers all API endpoints and static assets onto a standard ServeMux.
func NewRouter(h *Handler, staticHandler http.Handler) *http.ServeMux {
	mux := http.NewServeMux()

	// Health probes
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /healthz", h.Health)

	// API Endpoints
	mux.HandleFunc("POST /api/v1/shorten", h.Shorten)
	mux.HandleFunc("POST /shorten", h.Shorten) // backwards compatible with reference repo

	mux.HandleFunc("GET /api/v1/preview/{shortCode}", h.Preview)
	mux.HandleFunc("GET /preview/{shortCode}", h.Preview)

	mux.HandleFunc("GET /api/v1/analytics/{shortCode}", h.Analytics)
	mux.HandleFunc("GET /analytics/{shortCode}", h.Analytics)

	// Static frontend assets (if provided)
	if staticHandler != nil {
		mux.Handle("GET /static/", staticHandler)
	}

	// Root and Short URL redirection handler
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		path := strings.Trim(r.URL.Path, "/")
		if path == "" || path == "index.html" {
			if staticHandler != nil {
				staticHandler.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("URL Shortener API is running."))
			return
		}

		// Otherwise, treat as short code redirect
		h.Redirect(w, r)
	})

	return mux
}
