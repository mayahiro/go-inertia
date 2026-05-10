package inertiaecho

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	echo "github.com/labstack/echo/v5"
	inertia "github.com/mayahiro/go-inertia"
)

func TestAdapterMiddlewareCallsCoreMiddleware(t *testing.T) {
	renderer := newRenderer(t, inertia.Config{VersionProvider: inertia.StaticVersion("new")})
	adapter := New(renderer)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req.Header.Set(inertia.HeaderInertia, "true")
	req.Header.Set(inertia.HeaderInertiaVersion, "old")
	w := httptest.NewRecorder()
	c := e.NewContext(req, w)
	called := false

	err := adapter.Middleware(func(c *echo.Context) error {
		called = true
		return c.NoContent(http.StatusNoContent)
	})(c)
	if err != nil {
		t.Fatal(err)
	}

	if called {
		t.Fatal("next handler should not be called")
	}
	if w.Code != http.StatusConflict {
		t.Fatalf("unexpected status: %d", w.Code)
	}
}

func TestAdapterRenderJSONForInertiaRequest(t *testing.T) {
	adapter := New(newRenderer(t, inertia.Config{}))
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set(inertia.HeaderInertia, "true")
	w := httptest.NewRecorder()
	c := e.NewContext(req, w)

	err := adapter.Render(c, "Dashboard", inertia.Props{"message": "Hello"})
	if err != nil {
		t.Fatal(err)
	}

	if got := w.Header().Get(inertia.HeaderInertia); got != "true" {
		t.Fatalf("unexpected inertia header: %s", got)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
}

func TestAdapterRenderHTMLForNormalRequest(t *testing.T) {
	adapter := New(newRenderer(t, inertia.Config{}))
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	w := httptest.NewRecorder()
	c := e.NewContext(req, w)

	err := adapter.Render(c, "Dashboard", inertia.Props{"message": "Hello"})
	if err != nil {
		t.Fatal(err)
	}

	if got := w.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("unexpected content type: %s", got)
	}
}

func TestAdapterRenderErrorSetsStatus(t *testing.T) {
	adapter := New(newRenderer(t, inertia.Config{}))
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/not-found", nil)
	req.Header.Set(inertia.HeaderInertia, "true")
	w := httptest.NewRecorder()
	c := e.NewContext(req, w)

	err := adapter.RenderError(c, "Errors/NotFound", inertia.Props{}, http.StatusNotFound)
	if err != nil {
		t.Fatal(err)
	}

	if w.Code != http.StatusNotFound {
		t.Fatalf("unexpected status: %d", w.Code)
	}
}

func TestAdapterRedirectReturns303ForNonGETInertiaRequest(t *testing.T) {
	adapter := New(newRenderer(t, inertia.Config{}))
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/users", nil)
	req.Header.Set(inertia.HeaderInertia, "true")
	w := httptest.NewRecorder()
	c := e.NewContext(req, w)

	err := adapter.Redirect(c, "/users")
	if err != nil {
		t.Fatal(err)
	}

	if w.Code != http.StatusSeeOther {
		t.Fatalf("unexpected status: %d", w.Code)
	}
}

func TestAdapterLocationReturns409ForInertiaRequest(t *testing.T) {
	adapter := New(newRenderer(t, inertia.Config{}))
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/external", nil)
	req.Header.Set(inertia.HeaderInertia, "true")
	w := httptest.NewRecorder()
	c := e.NewContext(req, w)

	err := adapter.Location(c, "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	if w.Code != http.StatusConflict {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	if got := w.Header().Get(inertia.HeaderInertiaLocation); got != "https://example.com" {
		t.Fatalf("unexpected location: %s", got)
	}
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
