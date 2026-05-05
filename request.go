package inertia

import (
	"net/http"
	"strings"
)

const (
	// HeaderInertia is the request and response header that marks an Inertia request or response.
	HeaderInertia = "X-Inertia"
	// HeaderInertiaVersion carries the client asset version.
	HeaderInertiaVersion = "X-Inertia-Version"
	// HeaderInertiaLocation carries the target URL for an Inertia location visit.
	HeaderInertiaLocation = "X-Inertia-Location"
	// HeaderInertiaRedirect carries a redirect target that should preserve the URL fragment.
	HeaderInertiaRedirect = "X-Inertia-Redirect"
	// HeaderInertiaPartialComponent carries the component name for a partial reload.
	HeaderInertiaPartialComponent = "X-Inertia-Partial-Component"
	// HeaderInertiaPartialData carries the prop names to include in a partial reload.
	HeaderInertiaPartialData = "X-Inertia-Partial-Data"
	// HeaderInertiaPartialExcept carries the prop names to exclude in a partial reload.
	HeaderInertiaPartialExcept = "X-Inertia-Partial-Except"
	// HeaderInertiaErrorBag carries the requested validation error bag name.
	HeaderInertiaErrorBag = "X-Inertia-Error-Bag"
)

// IsInertiaRequest reports whether req is an Inertia request.
func IsInertiaRequest(req *http.Request) bool {
	return strings.EqualFold(req.Header.Get(HeaderInertia), "true")
}

// IsPartialReload reports whether req asks for a partial reload.
func IsPartialReload(req *http.Request) bool {
	return IsInertiaRequest(req) && PartialComponent(req) != ""
}

// PartialComponent returns the partial reload component name from req.
func PartialComponent(req *http.Request) string {
	return req.Header.Get(HeaderInertiaPartialComponent)
}

// PartialData returns the requested partial reload prop names from req.
func PartialData(req *http.Request) []string {
	return splitHeaderList(req.Header.Get(HeaderInertiaPartialData))
}

// PartialExcept returns the excluded partial reload prop names from req.
func PartialExcept(req *http.Request) []string {
	return splitHeaderList(req.Header.Get(HeaderInertiaPartialExcept))
}

// ErrorBag returns the requested validation error bag name from req.
func ErrorBag(req *http.Request) string {
	return req.Header.Get(HeaderInertiaErrorBag)
}

func splitHeaderList(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}
