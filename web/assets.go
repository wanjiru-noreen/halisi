package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed templates/auth/*.html static/js/auth.js static/js/dashboard.js static/js/vendor.js static/css/auth.css
var assets embed.FS

func Page(name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content, err := assets.ReadFile("templates/auth/" + name + ".html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(content)
	})
}

func Static() http.Handler {
	static, err := fs.Sub(assets, "static")
	if err != nil {
		panic(err)
	}
	return http.StripPrefix("/static/", http.FileServer(http.FS(static)))
}
