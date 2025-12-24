package render

import (
	"html/template"
	"net/http"
)

type Renderer struct {
	t *template.Template
}

func NewRenderer(glob string) (*Renderer, error) {
	t, err := template.ParseGlob(glob)
	if err != nil {
		return nil, err
	}
	return &Renderer{t: t}, nil
}

func (r *Renderer) Render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = r.t.ExecuteTemplate(w, name, data)
}
