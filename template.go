package inertia

import (
	"html/template"
	"io"
	"io/fs"
	"os"
)

type templateJS = template.JS

// RootView renders the root HTML document for initial browser visits.
type RootView interface {
	// Render writes the root document using data.
	Render(w io.Writer, data RootViewData) error
}

// RootViewData contains the data passed to a RootView.
type RootViewData struct {
	// Page is the Inertia page object.
	Page Page
	// PageJSON is the safe JSON representation of Page.
	PageJSON template.JS
	// InertiaScript is the script tag containing PageJSON.
	InertiaScript template.HTML
	// InertiaHead is HTML rendered in the document head.
	InertiaHead template.HTML
	// ViteTags contains script and stylesheet tags for Vite assets.
	ViteTags template.HTML
	// Data contains additional template data.
	Data map[string]any
}

// TemplateRootView renders a RootView with html/template.
type TemplateRootView struct {
	template *template.Template
	name     string
}

// NewTemplateRootView creates a RootView from t and template name.
func NewTemplateRootView(t *template.Template, name string) RootView {
	return &TemplateRootView{template: t, name: name}
}

// NewTemplateRootViewFromFile parses path and returns a RootView.
func NewTemplateRootViewFromFile(path string, name string) (RootView, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	t, err := template.New(name).Parse(string(body))
	if err != nil {
		return nil, err
	}
	return NewTemplateRootView(t, name), nil
}

// NewTemplateRootViewFromFS parses path from fsys and returns a RootView.
func NewTemplateRootViewFromFS(fsys fs.FS, path string, name string) (RootView, error) {
	body, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, err
	}
	t, err := template.New(name).Parse(string(body))
	if err != nil {
		return nil, err
	}
	return NewTemplateRootView(t, name), nil
}

// Render executes the configured template.
func (v *TemplateRootView) Render(w io.Writer, data RootViewData) error {
	return v.template.ExecuteTemplate(w, v.name, data)
}

func inertiaScript(pageJSON template.JS) template.HTML {
	return template.HTML(`<script data-page="app" type="application/json">` + string(pageJSON) + `</script>`)
}
