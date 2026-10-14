package web

import (
	"html/template"
	"net/http"
)

var templates = template.Must(template.ParseFS(staticFS, "templates/*.html"))

// render executes the named template against the given data and writes it to
// w. On error it falls back to a plain-text 500 response.
func render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "failed to render page: "+err.Error(), http.StatusInternalServerError)
	}
}
