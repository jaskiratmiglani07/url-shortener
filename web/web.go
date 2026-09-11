package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/* index.html
var WebFS embed.FS

// StaticHandler returns an HTTP handler serving files under the /static/ route.
func StaticHandler() (http.Handler, error) {
	sub, err := fs.Sub(WebFS, "static")
	if err != nil {
		return nil, err
	}
	return http.StripPrefix("/static/", http.FileServer(http.FS(sub))), nil
}

// IndexHandler serves the embedded index.html.
func IndexHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		content, err := WebFS.ReadFile("index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}
}
