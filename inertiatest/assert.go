package inertiatest

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"

	inertia "github.com/mayahiro/go-inertia"
)

// ResponseAssertion provides assertions for an Inertia HTTP response.
type ResponseAssertion struct {
	t testing.TB
	w *httptest.ResponseRecorder
}

// PageAssertion provides assertions for an Inertia page object.
type PageAssertion struct {
	t    testing.TB
	page inertia.Page
}

// AssertResponse starts assertions for a recorded HTTP response.
func AssertResponse(t testing.TB, w *httptest.ResponseRecorder) *ResponseAssertion {
	t.Helper()
	return &ResponseAssertion{t: t, w: w}
}

// AssertPage starts assertions for a JSON encoded page object.
func AssertPage(t testing.TB, body []byte) *PageAssertion {
	t.Helper()
	return &PageAssertion{t: t, page: DecodePage(t, body)}
}

// DecodePage decodes a JSON encoded Inertia page object.
func DecodePage(t testing.TB, body []byte) inertia.Page {
	t.Helper()
	var page inertia.Page
	if err := json.Unmarshal(body, &page); err != nil {
		t.Fatalf("decode inertia page: %v", err)
	}
	return page
}

// Status asserts the response status code.
func (a *ResponseAssertion) Status(code int) *ResponseAssertion {
	a.t.Helper()
	if a.w.Code != code {
		a.t.Fatalf("unexpected status: got %d, want %d", a.w.Code, code)
	}
	return a
}

// Header asserts a response header value.
func (a *ResponseAssertion) Header(name string, value string) *ResponseAssertion {
	a.t.Helper()
	if got := a.w.Header().Get(name); got != value {
		a.t.Fatalf("unexpected %s header: got %q, want %q", name, got, value)
	}
	return a
}

// IsInertia asserts that the response is an Inertia JSON response.
func (a *ResponseAssertion) IsInertia() *ResponseAssertion {
	a.t.Helper()
	return a.Header(inertia.HeaderInertia, "true")
}

// Page decodes the response body and starts page assertions.
func (a *ResponseAssertion) Page() *PageAssertion {
	a.t.Helper()
	return AssertPage(a.t, a.w.Body.Bytes())
}

// Component asserts the page component name.
func (a *PageAssertion) Component(component string) *PageAssertion {
	a.t.Helper()
	if a.page.Component != component {
		a.t.Fatalf("unexpected component: got %q, want %q", a.page.Component, component)
	}
	return a
}

// URL asserts the page URL.
func (a *PageAssertion) URL(url string) *PageAssertion {
	a.t.Helper()
	if a.page.URL != url {
		a.t.Fatalf("unexpected url: got %q, want %q", a.page.URL, url)
	}
	return a
}

// HasProp asserts that a page prop exists.
func (a *PageAssertion) HasProp(path string) *PageAssertion {
	a.t.Helper()
	if _, ok := lookupPath(a.page.Props, path); !ok {
		a.t.Fatalf("missing prop %q", path)
	}
	return a
}

// MissingProp asserts that a page prop does not exist.
func (a *PageAssertion) MissingProp(path string) *PageAssertion {
	a.t.Helper()
	if value, ok := lookupPath(a.page.Props, path); ok {
		a.t.Fatalf("unexpected prop %q: %#v", path, value)
	}
	return a
}

// PropEqual asserts that a page prop equals want.
func (a *PageAssertion) PropEqual(path string, want any) *PageAssertion {
	a.t.Helper()
	got, ok := lookupPath(a.page.Props, path)
	if !ok {
		a.t.Fatalf("missing prop %q", path)
	}
	if !reflect.DeepEqual(got, want) {
		a.t.Fatalf("unexpected prop %q: got %#v, want %#v", path, got, want)
	}
	return a
}

// HasDeferredProp asserts that a deferred prop is listed in a group.
func (a *PageAssertion) HasDeferredProp(group string, prop string) *PageAssertion {
	a.t.Helper()
	if !contains(a.page.DeferredProps[group], prop) {
		a.t.Fatalf("missing deferred prop %q in group %q: %#v", prop, group, a.page.DeferredProps)
	}
	return a
}

// HasRescuedProp asserts that a rescued prop is listed.
func (a *PageAssertion) HasRescuedProp(prop string) *PageAssertion {
	a.t.Helper()
	if !contains(a.page.RescuedProps, prop) {
		a.t.Fatalf("missing rescued prop %q: %#v", prop, a.page.RescuedProps)
	}
	return a
}

// HasSharedProp asserts that a shared prop is listed.
func (a *PageAssertion) HasSharedProp(prop string) *PageAssertion {
	a.t.Helper()
	if !contains(a.page.SharedProps, prop) {
		a.t.Fatalf("missing shared prop %q: %#v", prop, a.page.SharedProps)
	}
	return a
}

// HasMergeProp asserts that a merge prop path is listed.
func (a *PageAssertion) HasMergeProp(prop string) *PageAssertion {
	a.t.Helper()
	return a.hasPath("merge prop", prop, a.page.MergeProps)
}

// HasPrependProp asserts that a prepend prop path is listed.
func (a *PageAssertion) HasPrependProp(prop string) *PageAssertion {
	a.t.Helper()
	return a.hasPath("prepend prop", prop, a.page.PrependProps)
}

// HasDeepMergeProp asserts that a deep merge prop path is listed.
func (a *PageAssertion) HasDeepMergeProp(prop string) *PageAssertion {
	a.t.Helper()
	return a.hasPath("deep merge prop", prop, a.page.DeepMergeProps)
}

// HasMatchProp asserts that a match prop path is listed.
func (a *PageAssertion) HasMatchProp(prop string) *PageAssertion {
	a.t.Helper()
	return a.hasPath("match prop", prop, a.page.MatchPropsOn)
}

// HasScrollProp asserts that scroll metadata exists for a prop.
func (a *PageAssertion) HasScrollProp(prop string) *PageAssertion {
	a.t.Helper()
	if _, ok := a.page.ScrollProps[prop]; !ok {
		a.t.Fatalf("missing scroll prop %q: %#v", prop, a.page.ScrollProps)
	}
	return a
}

// HasOnceProp asserts that once metadata exists for a key.
func (a *PageAssertion) HasOnceProp(key string) *PageAssertion {
	a.t.Helper()
	if _, ok := a.page.OnceProps[key]; !ok {
		a.t.Fatalf("missing once prop %q: %#v", key, a.page.OnceProps)
	}
	return a
}

// HasFlash asserts that a flash prop exists.
func (a *PageAssertion) HasFlash(path string) *PageAssertion {
	a.t.Helper()
	if _, ok := lookupPath(a.page.Props, joinPath("flash", path)); !ok {
		a.t.Fatalf("missing flash %q", path)
	}
	return a
}

// MissingFlash asserts that a flash prop does not exist.
func (a *PageAssertion) MissingFlash(path string) *PageAssertion {
	a.t.Helper()
	if value, ok := lookupPath(a.page.Props, joinPath("flash", path)); ok {
		a.t.Fatalf("unexpected flash %q: %#v", path, value)
	}
	return a
}

// HasError asserts that a validation error prop exists.
func (a *PageAssertion) HasError(path string) *PageAssertion {
	a.t.Helper()
	if _, ok := lookupPath(a.page.Props, joinPath("errors", path)); !ok {
		a.t.Fatalf("missing error %q", path)
	}
	return a
}

// MissingError asserts that a validation error prop does not exist.
func (a *PageAssertion) MissingError(path string) *PageAssertion {
	a.t.Helper()
	if value, ok := lookupPath(a.page.Props, joinPath("errors", path)); ok {
		a.t.Fatalf("unexpected error %q: %#v", path, value)
	}
	return a
}

func (a *PageAssertion) hasPath(kind string, prop string, values []string) *PageAssertion {
	a.t.Helper()
	if !contains(values, prop) {
		a.t.Fatalf("missing %s %q: %#v", kind, prop, values)
	}
	return a
}

func contains(values []string, value string) bool {
	for _, current := range values {
		if current == value {
			return true
		}
	}
	return false
}

func lookupPath(value any, path string) (any, bool) {
	current := value
	for _, part := range strings.Split(path, ".") {
		next, ok := lookupPart(current, part)
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func lookupPart(value any, part string) (any, bool) {
	switch typed := value.(type) {
	case inertia.Props:
		next, ok := typed[part]
		return next, ok
	case map[string]any:
		next, ok := typed[part]
		return next, ok
	case []any:
		index, err := strconv.Atoi(part)
		if err != nil || index < 0 || index >= len(typed) {
			return nil, false
		}
		return typed[index], true
	default:
		return nil, false
	}
}

func joinPath(prefix string, path string) string {
	if path == "" {
		return prefix
	}
	return prefix + "." + path
}
