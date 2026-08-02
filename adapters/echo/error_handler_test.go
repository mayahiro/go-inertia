package inertiaecho

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	echo "github.com/labstack/echo/v5"
	inertia "github.com/mayahiro/go-inertia"
)

func TestAdapterErrorHandlerRendersMappedInertiaPage(t *testing.T) {
	adapter := New(newRenderer(t, inertia.Config{
		SharedProps: inertia.StaticSharedProps(inertia.Props{"appName": "Example"}),
	}))
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	req.Header.Set(inertia.HeaderInertia, "true")
	w := httptest.NewRecorder()
	c := e.NewContext(req, w)
	originalErr := echo.ErrNotFound

	handler := adapter.ErrorHandler(ErrorHandlerConfig{
		Pages: map[int]string{http.StatusNotFound: "Errors/NotFound"},
		Props: func(c *echo.Context, err error, status int) (inertia.Props, error) {
			if !errors.Is(err, originalErr) || status != http.StatusNotFound {
				t.Fatalf("unexpected error context: %v %d", err, status)
			}
			return inertia.Props{"status": status}, nil
		},
		RenderOptions: func(c *echo.Context, err error, status int) ([]inertia.RenderOption, error) {
			return []inertia.RenderOption{inertia.WithRenderPreserveFragment()}, nil
		},
	})
	handler(c, originalErr)

	if w.Code != http.StatusNotFound {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	var page inertia.Page
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Component != "Errors/NotFound" || page.Props["status"] != float64(http.StatusNotFound) {
		t.Fatalf("unexpected error page: %#v", page)
	}
	if page.Props["appName"] != "Example" {
		t.Fatalf("shared props should be preserved: %#v", page.Props)
	}
	if !page.PreserveFragment {
		t.Fatal("render options should be applied")
	}
}

func TestAdapterErrorHandlerIntegratesWithEchoRouting(t *testing.T) {
	adapter := New(newRenderer(t, inertia.Config{}))
	e := echo.New()
	e.HTTPErrorHandler = adapter.ErrorHandler(ErrorHandlerConfig{
		Pages: map[int]string{http.StatusNotFound: "Errors/NotFound"},
	})
	e.Use(adapter.Middleware)
	group := e.Group("/admin")
	group.GET("/dashboard", func(c *echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/admin/missing", nil)
	req.Header.Set(inertia.HeaderInertia, "true")
	w := httptest.NewRecorder()

	e.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	if got := w.Header().Get(inertia.HeaderInertia); got != "true" {
		t.Fatalf("unexpected inertia header: %s", got)
	}
	var page inertia.Page
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Component != "Errors/NotFound" {
		t.Fatalf("unexpected error component: %s", page.Component)
	}
}

func TestAdapterErrorHandlerUsesDefaultComponentForBrowserHTML(t *testing.T) {
	adapter := New(newRenderer(t, inertia.Config{}))
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/broken", nil)
	req.Header.Set(echo.HeaderAccept, "application/json, text/html;q=0.8")
	w := httptest.NewRecorder()
	c := e.NewContext(req, w)

	adapter.ErrorHandler(ErrorHandlerConfig{DefaultComponent: "Errors/ServerError"})(c, errors.New("broken"))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	if contentType := w.Header().Get(echo.HeaderContentType); contentType != "text/html; charset=utf-8" {
		t.Fatalf("unexpected content type: %s", contentType)
	}
	if body := w.Body.String(); !containsAll(body, `"component":"Errors/ServerError"`, `id="app"`) {
		t.Fatalf("unexpected error page body: %s", body)
	}
	vary := strings.Join(w.Header().Values(echo.HeaderVary), ",")
	if !containsAll(vary, echo.HeaderAccept, inertia.HeaderInertia) {
		t.Fatalf("unexpected vary header: %s", vary)
	}
}

func TestAdapterErrorHandlerFallsBackForNonHTMLRequest(t *testing.T) {
	adapter := New(newRenderer(t, inertia.Config{}))
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	req.Header.Set(echo.HeaderAccept, "application/json, text/html;q=0")
	w := httptest.NewRecorder()
	c := e.NewContext(req, w)
	fallbackCalled := false

	handler := adapter.ErrorHandler(ErrorHandlerConfig{
		DefaultComponent: "Errors/Default",
		Fallback: func(c *echo.Context, err error) {
			fallbackCalled = true
			_ = c.NoContent(http.StatusTeapot)
		},
	})
	handler(c, echo.ErrNotFound)

	if !fallbackCalled || w.Code != http.StatusTeapot {
		t.Fatalf("fallback was not used: called=%t status=%d", fallbackCalled, w.Code)
	}
	vary := strings.Join(w.Header().Values(echo.HeaderVary), ",")
	if !containsAll(vary, echo.HeaderAccept, inertia.HeaderInertia) {
		t.Fatalf("unexpected vary header: %s", vary)
	}
}

func TestDefaultErrorRenderPredicateRequiresExplicitAcceptableHTML(t *testing.T) {
	e := echo.New()
	tests := []struct {
		name   string
		method string
		accept string
	}{
		{name: "missing accept", method: http.MethodGet},
		{name: "wildcard only", method: http.MethodGet, accept: "*/*"},
		{name: "zero quality", method: http.MethodGet, accept: "text/html;q=0"},
		{name: "not a number quality", method: http.MethodGet, accept: "text/html;q=NaN"},
		{name: "quality above one", method: http.MethodGet, accept: "text/html;q=2"},
		{name: "invalid quality", method: http.MethodGet, accept: "text/html;q=invalid"},
		{name: "head request", method: http.MethodHead, accept: echo.MIMETextHTML},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(test.method, "/broken", nil)
			req.Header.Set(echo.HeaderAccept, test.accept)
			c := e.NewContext(req, httptest.NewRecorder())
			if DefaultErrorRenderPredicate(c, errors.New("broken"), http.StatusInternalServerError) {
				t.Fatalf("request should not render: method=%s accept=%s", test.method, test.accept)
			}
		})
	}
}

func TestAdapterErrorHandlerFallsBackWhenPropsFail(t *testing.T) {
	adapter := New(newRenderer(t, inertia.Config{}))
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/broken", nil)
	req.Header.Set(inertia.HeaderInertia, "true")
	w := httptest.NewRecorder()
	c := e.NewContext(req, w)
	propsErr := errors.New("props failed")
	var fallbackErr error

	handler := adapter.ErrorHandler(ErrorHandlerConfig{
		DefaultComponent: "Errors/Default",
		Props: func(c *echo.Context, err error, status int) (inertia.Props, error) {
			return nil, propsErr
		},
		Fallback: func(c *echo.Context, err error) {
			fallbackErr = err
		},
	})
	handler(c, echo.ErrInternalServerError)

	if !errors.Is(fallbackErr, echo.ErrInternalServerError) || !errors.Is(fallbackErr, propsErr) {
		t.Fatalf("fallback should receive both errors: %v", fallbackErr)
	}
}

func TestAdapterErrorHandlerFallsBackWhenRenderOptionsFail(t *testing.T) {
	adapter := New(newRenderer(t, inertia.Config{}))
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/forbidden", nil)
	req.Header.Set(inertia.HeaderInertia, "true")
	w := httptest.NewRecorder()
	c := e.NewContext(req, w)
	optionsErr := errors.New("render options failed")
	var fallbackErr error

	handler := adapter.ErrorHandler(ErrorHandlerConfig{
		DefaultComponent: "Errors/Default",
		RenderOptions: func(c *echo.Context, err error, status int) ([]inertia.RenderOption, error) {
			return nil, optionsErr
		},
		Fallback: func(c *echo.Context, err error) {
			fallbackErr = err
		},
	})
	handler(c, echo.ErrForbidden)

	if !errors.Is(fallbackErr, echo.ErrForbidden) || !errors.Is(fallbackErr, optionsErr) {
		t.Fatalf("fallback should receive both errors: %v", fallbackErr)
	}
}

func TestAdapterErrorHandlerFallsBackWhenRenderFails(t *testing.T) {
	renderErr := errors.New("template failed")
	renderer, err := inertia.New(inertia.Config{
		RootView: failingRootView{err: renderErr},
	})
	if err != nil {
		t.Fatal(err)
	}
	adapter := New(renderer)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/broken", nil)
	req.Header.Set(echo.HeaderAccept, echo.MIMETextHTML)
	w := httptest.NewRecorder()
	c := e.NewContext(req, w)
	originalErr := errors.New("broken")
	var fallbackErr error

	adapter.ErrorHandler(ErrorHandlerConfig{
		DefaultComponent: "Errors/Default",
		Fallback: func(c *echo.Context, err error) {
			fallbackErr = err
			_ = c.NoContent(http.StatusTeapot)
		},
	})(c, originalErr)

	if !errors.Is(fallbackErr, originalErr) || !errors.Is(fallbackErr, renderErr) {
		t.Fatalf("fallback should receive original and render errors: %v", fallbackErr)
	}
	if w.Code != http.StatusTeapot || w.Body.Len() != 0 {
		t.Fatalf("fallback should replace buffered render: status=%d body=%q", w.Code, w.Body.String())
	}
}

func TestAdapterErrorHandlerIgnoresCommittedResponse(t *testing.T) {
	adapter := New(newRenderer(t, inertia.Config{}))
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/already-written", nil)
	w := httptest.NewRecorder()
	c := e.NewContext(req, w)
	if err := c.String(http.StatusAccepted, "accepted"); err != nil {
		t.Fatal(err)
	}
	fallbackCalled := false

	adapter.ErrorHandler(ErrorHandlerConfig{
		DefaultComponent: "Errors/Default",
		Fallback: func(c *echo.Context, err error) {
			fallbackCalled = true
		},
	})(c, errors.New("late failure"))

	if fallbackCalled || w.Code != http.StatusAccepted || w.Body.String() != "accepted" {
		t.Fatalf("committed response changed: called=%t status=%d body=%q", fallbackCalled, w.Code, w.Body.String())
	}
}

func TestAdapterErrorHandlerDoesNotFallBackAfterWriteFailure(t *testing.T) {
	adapter := New(newRenderer(t, inertia.Config{}))
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/broken", nil)
	req.Header.Set(inertia.HeaderInertia, "true")
	writeErr := errors.New("write failed")
	w := &failingResponseWriter{
		header: make(http.Header),
		err:    writeErr,
	}
	c := e.NewContext(req, w)
	fallbackCalled := false

	adapter.ErrorHandler(ErrorHandlerConfig{
		DefaultComponent: "Errors/Default",
		Fallback: func(c *echo.Context, err error) {
			fallbackCalled = true
		},
	})(c, errors.New("broken"))

	if !w.writeCalled {
		t.Fatal("response write failure was not exercised")
	}
	if fallbackCalled {
		t.Fatal("fallback should not run after the response is committed")
	}
	if !responseCommitted(c) {
		t.Fatal("response should be committed")
	}
}

func containsAll(value string, fragments ...string) bool {
	for _, fragment := range fragments {
		if !strings.Contains(value, fragment) {
			return false
		}
	}
	return true
}

type failingRootView struct {
	err error
}

func (v failingRootView) Render(w io.Writer, data inertia.RootViewData) error {
	_, _ = io.WriteString(w, "partial")
	return v.err
}

type failingResponseWriter struct {
	header      http.Header
	err         error
	writeCalled bool
}

func (w *failingResponseWriter) Header() http.Header {
	return w.header
}

func (w *failingResponseWriter) WriteHeader(status int) {}

func (w *failingResponseWriter) Write(body []byte) (int, error) {
	w.writeCalled = true
	return 0, w.err
}
