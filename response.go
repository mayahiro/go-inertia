package inertia

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

// JSONEncoder encodes values for Inertia JSON responses.
type JSONEncoder interface {
	// Encode returns the JSON representation of v.
	Encode(v any) ([]byte, error)
}

// StandardJSONEncoder encodes values with encoding/json.
type StandardJSONEncoder struct{}

// Encode returns the JSON representation of v.
func (StandardJSONEncoder) Encode(v any) ([]byte, error) {
	return json.Marshal(v)
}

// Render renders an Inertia page.
//
// For Inertia requests it writes a JSON page response.
// For normal browser visits it renders the configured RootView.
func (r *Renderer) Render(w http.ResponseWriter, req *http.Request, component string, props Props, opts ...RenderOption) error {
	if component == "" {
		return ErrInvalidComponent
	}

	options := r.renderOptionsFrom(opts)
	page, err := r.page(req, component, props, options)
	if err != nil {
		return err
	}
	statusCode := options.statusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	AppendVary(w.Header(), HeaderInertia)

	if IsInertiaRequest(req) {
		body, err := r.jsonEncoder.Encode(page)
		if err != nil {
			return err
		}
		w.Header().Set(HeaderInertia, "true")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, err = w.Write(body)
		return err
	}

	pageJSON, err := safePageJSON(page, r.jsonEncoder)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	return r.rootView.Render(w, RootViewData{
		Page:          page,
		PageJSON:      pageJSON,
		InertiaScript: inertiaScript(pageJSON),
		Data:          options.data,
		ViteTags:      options.viteTags,
		InertiaHead:   options.inertiaHead,
	})
}

// RenderError renders an Inertia error page with status.
func (r *Renderer) RenderError(w http.ResponseWriter, req *http.Request, component string, props Props, status int, opts ...RenderOption) error {
	renderOpts := make([]RenderOption, 0, len(opts)+1)
	renderOpts = append(renderOpts, WithRenderStatus(status))
	renderOpts = append(renderOpts, opts...)
	return r.Render(w, req, component, props, renderOpts...)
}

func (r *Renderer) renderOptionsFrom(opts []RenderOption) renderOptions {
	options := renderOptions{}
	for _, opt := range r.renderOptions {
		opt(&options)
	}
	for _, opt := range opts {
		opt(&options)
	}
	return options
}

// AppendVary appends value to the Vary header without duplicating existing values.
func AppendVary(h http.Header, value string) {
	if value == "" {
		return
	}
	values := h.Values("Vary")
	for _, current := range values {
		for _, part := range strings.Split(current, ",") {
			part = strings.TrimSpace(part)
			if part == "*" || strings.EqualFold(part, value) {
				return
			}
		}
	}
	h.Add("Vary", value)
}

func safePageJSON(page Page, encoder JSONEncoder) (templateJS, error) {
	body, err := encoder.Encode(page)
	if err != nil {
		return "", err
	}
	body = bytes.ReplaceAll(body, []byte("<"), []byte("\\u003c"))
	body = bytes.ReplaceAll(body, []byte(">"), []byte("\\u003e"))
	body = bytes.ReplaceAll(body, []byte("&"), []byte("\\u0026"))
	body = bytes.ReplaceAll(body, []byte("\u2028"), []byte("\\u2028"))
	body = bytes.ReplaceAll(body, []byte("\u2029"), []byte("\\u2029"))
	return templateJS(body), nil
}
