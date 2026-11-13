package web

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed templates static
var assets embed.FS

func Page(name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := name
		if !strings.HasPrefix(page, "templates/") {
			if strings.Contains(page, "/") {
				page = "templates/" + page + ".html"
			} else {
				page = "templates/auth/" + page + ".html"
			}
		}
		page = path.Clean(page)
		if !strings.HasPrefix(page, "templates/") || strings.Contains(page, "..") {
			http.NotFound(w, r)
			return
		}
		view, err := template.ParseFS(assets, "templates/components/*.html", page)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if err := view.ExecuteTemplate(w, path.Base(page), nil); err != nil {
			http.Error(w, "Page could not be rendered.", http.StatusInternalServerError)
		}
	})
}

func Static() http.Handler {
	static, err := fs.Sub(assets, "static")
	if err != nil {
		panic(err)
	}
	return http.StripPrefix("/static/", http.FileServer(http.FS(static)))
}
