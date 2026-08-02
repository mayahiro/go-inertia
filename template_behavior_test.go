package inertia

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCustomRootElementIDPopulatesAllTemplateHelpers(t *testing.T) {
	rootView := rootViewFunc(func(w io.Writer, data RootViewData) error {
		_, err := io.WriteString(w, string(data.InertiaApp))
		if data.RootElementID != "custom-app" {
			t.Fatalf("unexpected root element id: %s", data.RootElementID)
		}
		return err
	})
	renderer, err := New(Config{RootView: rootView, RootElementID: "custom-app"})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	w := httptest.NewRecorder()

	if err := renderer.Render(w, req, "Dashboard", Props{}); err != nil {
		t.Fatal(err)
	}
	body := w.Body.String()
	if !strings.Contains(body, `data-page="custom-app"`) || !strings.Contains(body, `id="custom-app"`) {
		t.Fatalf("custom root helpers are not synchronized: %s", body)
	}
}

func TestNewRejectsUnsafeRootElementID(t *testing.T) {
	_, err := New(Config{RootView: rootViewFunc(func(w io.Writer, data RootViewData) error { return nil }), RootElementID: `app"]`})
	if !errors.Is(err, ErrInvalidRootElementID) {
		t.Fatalf("expected ErrInvalidRootElementID, got %v", err)
	}
}

func TestInitialTemplateErrorDoesNotCommitPartialResponse(t *testing.T) {
	templateErr := errors.New("template failed")
	rootView := rootViewFunc(func(w io.Writer, data RootViewData) error {
		_, _ = io.WriteString(w, "partial")
		return templateErr
	})
	renderer, err := New(Config{RootView: rootView})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	w := httptest.NewRecorder()

	err = renderer.Render(w, req, "Dashboard", Props{})
	if !errors.Is(err, templateErr) {
		t.Fatalf("expected template error, got %v", err)
	}
	if w.Body.Len() != 0 {
		t.Fatalf("partial template output should stay buffered: %s", w.Body.String())
	}
	if got := w.Header().Get("Content-Type"); got != "" {
		t.Fatalf("headers should not be committed after template failure: %s", got)
	}
}

func TestServerHeadIsRenderedIntoInitialHeadFallback(t *testing.T) {
	head, err := NewServerHead(HeadTitle("Users"), HeadMeta("description", "All users"))
	if err != nil {
		t.Fatal(err)
	}
	rootView := rootViewFunc(func(w io.Writer, data RootViewData) error {
		_, err := io.WriteString(w, string(data.InertiaHead))
		return err
	})
	renderer, err := New(Config{RootView: rootView})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()

	if err := renderer.Render(w, req, "Users/Index", Props{}, WithServerHead(head)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(w.Body.String(), `<title data-inertia="title">Users</title>`) || !strings.Contains(w.Body.String(), `data-inertia="meta:name:description"`) {
		t.Fatalf("missing initial server head fallback: %s", w.Body.String())
	}
}

type rootViewFunc func(w io.Writer, data RootViewData) error

func (f rootViewFunc) Render(w io.Writer, data RootViewData) error {
	return f(w, data)
}
