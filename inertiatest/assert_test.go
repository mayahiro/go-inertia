package inertiatest_test

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	inertia "github.com/mayahiro/go-inertia"
	"github.com/mayahiro/go-inertia/inertiatest"
)

func TestAssertResponseAndPage(t *testing.T) {
	renderer := newRenderer(t, inertia.Config{
		SharedProps: inertia.StaticSharedProps(inertia.Props{"app": "admin"}),
	})
	req := httptest.NewRequest("GET", "/dashboard", nil)
	ctx := inertia.WithFlashContext(req.Context(), inertia.Flash{"notice": "saved"})
	ctx = inertia.WithValidationErrorsContext(ctx, inertia.ValidationErrors{"name": "required"})
	req = req.WithContext(ctx)
	req.Header.Set(inertia.HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", inertia.Props{
		"message":     "Hello",
		"items":       inertia.Merge([]string{"a"}),
		"permissions": inertia.Defer(func(req *http.Request) (any, error) { return []string{"read"}, nil }),
		"plans": inertia.Once(func(req *http.Request) (any, error) {
			return []string{"basic"}, nil
		}),
		"posts": inertia.Scroll(inertia.Props{
			"data": []string{"first"},
		}, inertia.ScrollMetadata{CurrentPage: 1}),
	})
	if err != nil {
		t.Fatal(err)
	}

	inertiatest.AssertResponse(t, w).
		Status(http.StatusOK).
		IsInertia().
		Page().
		Component("Dashboard").
		URL("/dashboard").
		PropEqual("message", "Hello").
		HasSharedProp("app").
		HasDeferredProp("default", "permissions").
		HasOnceProp("plans").
		HasMergeProp("items").
		HasMergeProp("posts.data").
		HasScrollProp("posts").
		HasFlash("notice").
		HasError("name")
}

func TestAssertPageRescuedProp(t *testing.T) {
	renderer := newRenderer(t, inertia.Config{})
	req := httptest.NewRequest("GET", "/dashboard", nil)
	req.Header.Set(inertia.HeaderInertia, "true")
	req.Header.Set(inertia.HeaderInertiaPartialComponent, "Dashboard")
	req.Header.Set(inertia.HeaderInertiaPartialData, "permissions")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", inertia.Props{
		"permissions": inertia.Defer(func(req *http.Request) (any, error) {
			return nil, errors.New("failed")
		}).Rescue(),
	})
	if err != nil {
		t.Fatal(err)
	}

	inertiatest.AssertResponse(t, w).
		IsInertia().
		Page().
		MissingProp("permissions").
		HasRescuedProp("permissions")
}

func newRenderer(t *testing.T, config inertia.Config) *inertia.Renderer {
	t.Helper()
	if config.RootView == nil {
		config.RootView = inertia.NewTemplateRootView(template.Must(template.New("app").Parse(`<!doctype html>{{ .InertiaScript }}<div id="app"></div>`)), "app")
	}
	renderer, err := inertia.New(config)
	if err != nil {
		t.Fatal(err)
	}
	return renderer
}
