package inertia

import "net/http"

// URLResolver returns the URL stored in the Inertia page object.
type URLResolver interface {
	// URL returns the URL for req.
	URL(req *http.Request) string
}

// URLResolverFunc adapts a function to URLResolver.
type URLResolverFunc func(req *http.Request) string

// URL calls f(req).
func (f URLResolverFunc) URL(req *http.Request) string {
	return f(req)
}
