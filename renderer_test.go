package inertia

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTMLResponseIncludesSafePageJSON(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/dashboard", nil)
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"message": "</script><script>alert(1)</script>",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := w.Body.String()
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	if got := w.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("unexpected content type: %s", got)
	}
	if !strings.Contains(body, `<script data-page="app" type="application/json">`) {
		t.Fatalf("missing inertia script: %s", body)
	}
	if strings.Contains(body, "</script><script>alert") {
		t.Fatalf("unsafe script content: %s", body)
	}
	if !strings.Contains(body, `\u003c/script\u003e`) {
		t.Fatalf("expected escaped script content: %s", body)
	}
	if w.Header().Get("Vary") != HeaderInertia {
		t.Fatalf("unexpected vary: %s", w.Header().Get("Vary"))
	}
}

func TestJSONResponseIncludesInertiaHeadersAndErrors(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{"message": "Hello"})
	if err != nil {
		t.Fatal(err)
	}

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	if got := w.Header().Get(HeaderInertia); got != "true" {
		t.Fatalf("unexpected inertia header: %s", got)
	}
	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("unexpected content type: %s", got)
	}
	if got := w.Header().Get("Vary"); got != HeaderInertia {
		t.Fatalf("unexpected vary: %s", got)
	}

	var page Page
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Component != "Dashboard" {
		t.Fatalf("unexpected component: %s", page.Component)
	}
	if _, ok := page.Props["errors"]; !ok {
		t.Fatalf("errors prop missing: %#v", page.Props)
	}
}

func TestNewRequiresRootView(t *testing.T) {
	_, err := New(Config{})
	if !errors.Is(err, ErrMissingRootView) {
		t.Fatalf("expected ErrMissingRootView, got %v", err)
	}
}

func TestRenderRejectsEmptyComponent(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/dashboard", nil)
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "", Props{})
	if !errors.Is(err, ErrInvalidComponent) {
		t.Fatalf("expected ErrInvalidComponent, got %v", err)
	}
	if w.Body.Len() != 0 {
		t.Fatalf("response should not be written: %s", w.Body.String())
	}
}

func TestDefaultRenderOptions(t *testing.T) {
	view := NewTemplateRootView(template.Must(template.New("app").Parse(`<!doctype html>{{ .ViteTags }}{{ .InertiaScript }}<div id="app"></div>`)), "app")
	renderer := newTestRenderer(t, Config{
		RootView: view,
		DefaultRenderOptions: []RenderOption{
			WithViteTags(template.HTML(`<script type="module" src="/build/app.js"></script>`)),
		},
	})
	req := httptest.NewRequest("GET", "/dashboard", nil)
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(w.Body.String(), `<script type="module" src="/build/app.js"></script>`) {
		t.Fatalf("default render option was not applied: %s", w.Body.String())
	}
}

func TestRenderOptionsOverrideDefaults(t *testing.T) {
	view := NewTemplateRootView(template.Must(template.New("app").Parse(`<!doctype html>{{ .ViteTags }}{{ .InertiaScript }}<div id="app"></div>`)), "app")
	renderer := newTestRenderer(t, Config{
		RootView: view,
		DefaultRenderOptions: []RenderOption{
			WithViteTags(template.HTML(`default-tags`)),
		},
	})
	req := httptest.NewRequest("GET", "/dashboard", nil)
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{}, WithViteTags(template.HTML(`request-tags`)))
	if err != nil {
		t.Fatal(err)
	}

	body := w.Body.String()
	if !strings.Contains(body, `request-tags`) || strings.Contains(body, `default-tags`) {
		t.Fatalf("request render option should override default: %s", body)
	}
}

func TestCustomURLResolver(t *testing.T) {
	renderer := newTestRenderer(t, Config{
		URLResolver: URLResolverFunc(func(req *http.Request) string {
			return "/resolved"
		}),
	})
	req := httptest.NewRequest("GET", "/original", nil)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if page.URL != "/resolved" {
		t.Fatalf("unexpected url: %s", page.URL)
	}
}

func TestSharedAndHandlerPropsMerge(t *testing.T) {
	renderer := newTestRenderer(t, Config{
		SharedProps: StaticSharedProps(Props{
			"app":     "admin",
			"message": "shared",
			"errors":  "ignored",
		}),
	})
	req := httptest.NewRequest("GET", "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"message": "handler",
		"errors":  "ignored",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if page.Props["app"] != "admin" {
		t.Fatalf("missing shared prop: %#v", page.Props)
	}
	if page.Props["message"] != "handler" {
		t.Fatalf("handler prop did not override shared prop: %#v", page.Props)
	}
	if _, ok := page.Props["errors"].(map[string]any); !ok {
		t.Fatalf("reserved errors prop was overridden: %#v", page.Props["errors"])
	}
}

func TestPropResolverAddsPageMetadata(t *testing.T) {
	items := &testPropResolver{
		result: propResult{
			Value: []map[string]any{{"id": 1}},
			Metadata: pageMetadata{
				MergeProps:   []string{"items"},
				MatchPropsOn: []string{"items.id"},
			},
		},
	}
	permissions := &testPropResolver{
		result: propResult{
			Omit: true,
			Metadata: pageMetadata{
				DeferredProps: map[string][]string{
					"default": {"permissions"},
				},
			},
		},
	}
	plans := &testPropResolver{
		result: propResult{
			Value: []string{"basic"},
			Metadata: pageMetadata{
				OnceProps: map[string]OncePropMetadata{
					"plans": {Prop: "plans"},
				},
			},
		},
	}
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"items":       items,
		"permissions": permissions,
		"plans":       plans,
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if !items.called || !permissions.called || !plans.called {
		t.Fatal("expected prop resolvers to be called")
	}
	if _, ok := page.Props["permissions"]; ok {
		t.Fatalf("omitted prop should not be rendered: %#v", page.Props)
	}
	if got := page.MergeProps; len(got) != 1 || got[0] != "items" {
		t.Fatalf("unexpected merge props: %#v", got)
	}
	if got := page.MatchPropsOn; len(got) != 1 || got[0] != "items.id" {
		t.Fatalf("unexpected match props: %#v", got)
	}
	if got := page.DeferredProps["default"]; len(got) != 1 || got[0] != "permissions" {
		t.Fatalf("unexpected deferred props: %#v", page.DeferredProps)
	}
	if got := page.OnceProps["plans"]; got.Prop != "plans" {
		t.Fatalf("unexpected once props: %#v", page.OnceProps)
	}
}

func TestPropResolverMetadataIsRemovedWhenPropIsOverridden(t *testing.T) {
	sharedItems := &testPropResolver{
		result: propResult{
			Value: []string{"shared"},
			Metadata: pageMetadata{
				MergeProps:   []string{"items"},
				MatchPropsOn: []string{"items.id"},
			},
		},
	}
	renderer := newTestRenderer(t, Config{
		SharedProps: StaticSharedProps(Props{
			"items": sharedItems,
		}),
	})
	req := httptest.NewRequest("GET", "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"items": []string{"handler"},
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if !sharedItems.called {
		t.Fatal("expected shared prop resolver to be called")
	}
	if page.Props["items"].([]any)[0] != "handler" {
		t.Fatalf("handler prop should override shared prop: %#v", page.Props["items"])
	}
	if len(page.MergeProps) > 0 || len(page.MatchPropsOn) > 0 {
		t.Fatalf("overridden prop metadata should be removed: %#v %#v", page.MergeProps, page.MatchPropsOn)
	}
}

func TestPropResolverMetadataFollowsPartialReload(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaPartialComponent, "Dashboard")
	req.Header.Set(HeaderInertiaPartialData, "items")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"items": &testPropResolver{
			result: propResult{
				Value: []map[string]any{{"id": 1}},
				Metadata: pageMetadata{
					MergeProps:   []string{"items"},
					MatchPropsOn: []string{"items.id"},
					ScrollProps: map[string]any{
						"items": Props{"pageName": "page"},
					},
				},
			},
		},
		"stats": &testPropResolver{
			result: propResult{
				Value: Props{"users": 1},
				Metadata: pageMetadata{
					MergeProps:   []string{"stats"},
					MatchPropsOn: []string{"stats.users.id"},
					ScrollProps: map[string]any{
						"stats": Props{"pageName": "statsPage"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if _, ok := page.Props["stats"]; ok {
		t.Fatalf("partial reload should exclude stats: %#v", page.Props)
	}
	if got := page.MergeProps; len(got) != 1 || got[0] != "items" {
		t.Fatalf("unexpected merge props: %#v", got)
	}
	if got := page.MatchPropsOn; len(got) != 1 || got[0] != "items.id" {
		t.Fatalf("unexpected match props: %#v", got)
	}
	if _, ok := page.ScrollProps["items"]; !ok {
		t.Fatalf("scroll metadata should keep included props: %#v", page.ScrollProps)
	}
	if _, ok := page.ScrollProps["stats"]; ok {
		t.Fatalf("scroll metadata should follow filtered props: %#v", page.ScrollProps)
	}
}

func TestPropResolverError(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/dashboard", nil)
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"items": &testPropResolver{err: errors.New("resolve failed")},
	})
	if err == nil || err.Error() != "resolve failed" {
		t.Fatalf("expected resolver error, got %v", err)
	}
	if w.Body.Len() != 0 {
		t.Fatalf("response should not be written: %s", w.Body.String())
	}
}

func TestDeferredPropIsOmittedUntilPartialReload(t *testing.T) {
	calls := 0
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"permissions": Defer(func(req *http.Request) (any, error) {
			calls++
			return []string{"read"}, nil
		}),
		"teams": Defer(func(req *http.Request) (any, error) {
			calls++
			return []string{"admin"}, nil
		}, "attributes"),
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if calls != 0 {
		t.Fatalf("deferred props should not resolve on initial response: %d", calls)
	}
	if _, ok := page.Props["permissions"]; ok {
		t.Fatalf("deferred prop should be omitted: %#v", page.Props)
	}
	if got := page.DeferredProps["default"]; len(got) != 1 || got[0] != "permissions" {
		t.Fatalf("unexpected default deferred props: %#v", page.DeferredProps)
	}
	if got := page.DeferredProps["attributes"]; len(got) != 1 || got[0] != "teams" {
		t.Fatalf("unexpected grouped deferred props: %#v", page.DeferredProps)
	}
}

func TestDeferredPropResolvesWhenRequestedByPartialReload(t *testing.T) {
	calls := 0
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaPartialComponent, "Dashboard")
	req.Header.Set(HeaderInertiaPartialData, "permissions")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"permissions": Defer(func(req *http.Request) (any, error) {
			calls++
			return []string{"read"}, nil
		}),
		"teams": Defer(func(req *http.Request) (any, error) {
			calls++
			return []string{"admin"}, nil
		}, "attributes"),
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if calls != 1 {
		t.Fatalf("only requested deferred prop should resolve: %d", calls)
	}
	if got := page.Props["permissions"].([]any); len(got) != 1 || got[0] != "read" {
		t.Fatalf("unexpected deferred prop value: %#v", page.Props["permissions"])
	}
	if _, ok := page.Props["teams"]; ok {
		t.Fatalf("unrequested deferred prop should be omitted: %#v", page.Props)
	}
	if len(page.DeferredProps) > 0 {
		t.Fatalf("partial response should not include deferred metadata: %#v", page.DeferredProps)
	}
}

func TestDeferredPropRespectsPartialReloadExcept(t *testing.T) {
	calls := 0
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaPartialComponent, "Dashboard")
	req.Header.Set(HeaderInertiaPartialExcept, "filters")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"permissions": Defer(func(req *http.Request) (any, error) {
			calls++
			return []string{"read"}, nil
		}),
		"filters": Props{"q": ""},
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if calls != 1 {
		t.Fatalf("deferred prop should resolve when not excluded: %d", calls)
	}
	if _, ok := page.Props["permissions"]; !ok {
		t.Fatalf("deferred prop should be included: %#v", page.Props)
	}
	if _, ok := page.Props["filters"]; ok {
		t.Fatalf("excluded prop should be omitted: %#v", page.Props)
	}
}

func TestDeferredPropIsSkippedWhenPartialReloadDoesNotRequestIt(t *testing.T) {
	calls := 0
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaPartialComponent, "Dashboard")
	req.Header.Set(HeaderInertiaPartialData, "users")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"users": []string{"alice"},
		"permissions": Defer(func(req *http.Request) (any, error) {
			calls++
			return []string{"read"}, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if calls != 0 {
		t.Fatalf("unrequested deferred prop should not resolve: %d", calls)
	}
	if _, ok := page.Props["permissions"]; ok {
		t.Fatalf("unrequested deferred prop should be omitted: %#v", page.Props)
	}
	if len(page.DeferredProps) > 0 {
		t.Fatalf("unrequested partial response should not include deferred metadata: %#v", page.DeferredProps)
	}
}

func TestDeferredPropError(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaPartialComponent, "Dashboard")
	req.Header.Set(HeaderInertiaPartialData, "permissions")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"permissions": Defer(func(req *http.Request) (any, error) {
			return nil, errors.New("load failed")
		}),
	})
	if err == nil || err.Error() != "load failed" {
		t.Fatalf("expected deferred error, got %v", err)
	}
	if w.Body.Len() != 0 {
		t.Fatalf("response should not be written: %s", w.Body.String())
	}
}

func TestOncePropResolvesAndAddsMetadata(t *testing.T) {
	calls := 0
	expiresAt := time.UnixMilli(4102444800000)
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/plans", nil)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Plans/Index", Props{
		"plans": Once(func(req *http.Request) (any, error) {
			calls++
			return []string{"basic"}, nil
		}).Until(expiresAt),
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if calls != 1 {
		t.Fatalf("once prop should resolve: %d", calls)
	}
	if got := page.Props["plans"].([]any); len(got) != 1 || got[0] != "basic" {
		t.Fatalf("unexpected once prop value: %#v", page.Props["plans"])
	}
	metadata := page.OnceProps["plans"]
	if metadata.Prop != "plans" {
		t.Fatalf("unexpected once metadata: %#v", page.OnceProps)
	}
	if metadata.ExpiresAt == nil || *metadata.ExpiresAt != expiresAt.UnixMilli() {
		t.Fatalf("unexpected once expiration: %#v", metadata.ExpiresAt)
	}
}

func TestOncePropIsOmittedWhenClientAlreadyLoadedIt(t *testing.T) {
	calls := 0
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/plans", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaExceptOnceProps, "plans")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Plans/Index", Props{
		"plans": Once(func(req *http.Request) (any, error) {
			calls++
			return []string{"basic"}, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if calls != 0 {
		t.Fatalf("loaded once prop should not resolve: %d", calls)
	}
	if _, ok := page.Props["plans"]; ok {
		t.Fatalf("loaded once prop should be omitted: %#v", page.Props)
	}
	if got := page.OnceProps["plans"]; got.Prop != "plans" {
		t.Fatalf("once metadata should remain when omitted: %#v", page.OnceProps)
	}
}

func TestOncePropUsesCustomKey(t *testing.T) {
	calls := 0
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/plans", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaExceptOnceProps, "roles")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Plans/Index", Props{
		"availableRoles": Once(func(req *http.Request) (any, error) {
			calls++
			return []string{"admin"}, nil
		}).As("roles"),
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if calls != 0 {
		t.Fatalf("loaded custom once prop should not resolve: %d", calls)
	}
	if _, ok := page.Props["availableRoles"]; ok {
		t.Fatalf("loaded custom once prop should be omitted: %#v", page.Props)
	}
	if got := page.OnceProps["roles"]; got.Prop != "availableRoles" {
		t.Fatalf("unexpected custom once metadata: %#v", page.OnceProps)
	}
}

func TestOncePropFreshIgnoresLoadedHeader(t *testing.T) {
	calls := 0
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/plans", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaExceptOnceProps, "plans")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Plans/Index", Props{
		"plans": Once(func(req *http.Request) (any, error) {
			calls++
			return []string{"basic"}, nil
		}).Fresh(),
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if calls != 1 {
		t.Fatalf("fresh once prop should resolve: %d", calls)
	}
	if _, ok := page.Props["plans"]; !ok {
		t.Fatalf("fresh once prop should be included: %#v", page.Props)
	}
}

func TestOncePropResolvesWhenRequestedByPartialReload(t *testing.T) {
	calls := 0
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/plans", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaExceptOnceProps, "plans")
	req.Header.Set(HeaderInertiaPartialComponent, "Plans/Index")
	req.Header.Set(HeaderInertiaPartialData, "plans")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Plans/Index", Props{
		"plans": Once(func(req *http.Request) (any, error) {
			calls++
			return []string{"basic"}, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if calls != 1 {
		t.Fatalf("requested once prop should resolve during partial reload: %d", calls)
	}
	if _, ok := page.Props["plans"]; !ok {
		t.Fatalf("requested once prop should be included: %#v", page.Props)
	}
	if got := page.OnceProps["plans"]; got.Prop != "plans" {
		t.Fatalf("once metadata should be included: %#v", page.OnceProps)
	}
}

func TestOncePropIsSkippedWhenPartialReloadDoesNotRequestIt(t *testing.T) {
	calls := 0
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/plans", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaExceptOnceProps, "plans")
	req.Header.Set(HeaderInertiaPartialComponent, "Plans/Index")
	req.Header.Set(HeaderInertiaPartialData, "users")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Plans/Index", Props{
		"users": []string{"alice"},
		"plans": Once(func(req *http.Request) (any, error) {
			calls++
			return []string{"basic"}, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if calls != 0 {
		t.Fatalf("unrequested once prop should not resolve: %d", calls)
	}
	if _, ok := page.Props["plans"]; ok {
		t.Fatalf("unrequested once prop should be omitted: %#v", page.Props)
	}
	if len(page.OnceProps) > 0 {
		t.Fatalf("unrequested partial response should not include once metadata: %#v", page.OnceProps)
	}
}

func TestOncePropError(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/plans", nil)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Plans/Index", Props{
		"plans": Once(func(req *http.Request) (any, error) {
			return nil, errors.New("load once failed")
		}),
	})
	if err == nil || err.Error() != "load once failed" {
		t.Fatalf("expected once error, got %v", err)
	}
	if w.Body.Len() != 0 {
		t.Fatalf("response should not be written: %s", w.Body.String())
	}
}

func TestContextPropsFlashAndValidationErrorsMerge(t *testing.T) {
	renderer := newTestRenderer(t, Config{
		SharedProps: StaticSharedProps(Props{
			"global":  "shared",
			"message": "global",
		}),
	})
	req := httptest.NewRequest("GET", "/dashboard", nil)
	ctx := req.Context()
	ctx = WithSharedProps(ctx, Props{
		"contextShared": "shared",
		"message":       "context-shared",
	})
	ctx = WithProps(ctx, Props{
		"contextProp": "prop",
		"errors":      "ignored",
		"flash":       "ignored",
	})
	ctx = WithFlashContext(ctx, Flash{"notice": "saved"})
	ctx = WithValidationErrorsContext(ctx, ValidationErrors{"title": "required"})
	req = req.WithContext(ctx)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{"message": "handler"})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if page.Props["global"] != "shared" {
		t.Fatalf("missing global shared prop: %#v", page.Props)
	}
	if page.Props["contextShared"] != "shared" {
		t.Fatalf("missing context shared prop: %#v", page.Props)
	}
	if page.Props["message"] != "handler" {
		t.Fatalf("handler prop should override shared props: %#v", page.Props)
	}
	if page.Props["contextProp"] != "prop" {
		t.Fatalf("missing context prop: %#v", page.Props)
	}
	flash, ok := page.Props["flash"].(map[string]any)
	if !ok || flash["notice"] != "saved" {
		t.Fatalf("unexpected flash: %#v", page.Props["flash"])
	}
	renderedErrors, ok := page.Props["errors"].(map[string]any)
	if !ok || renderedErrors["title"] != "required" {
		t.Fatalf("unexpected errors: %#v", page.Props["errors"])
	}
}

func TestFlashAndValidationErrorsMerge(t *testing.T) {
	store := &testFlashStore{data: FlashData{
		Flash:  Flash{"success": "created"},
		Errors: ValidationErrors{"name": "required"},
	}}
	renderer := newTestRenderer(t, Config{FlashStore: store})
	req := httptest.NewRequest("GET", "/users", nil)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Users/Index", Props{})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	flash, ok := page.Props["flash"].(map[string]any)
	if !ok || flash["success"] != "created" {
		t.Fatalf("unexpected flash: %#v", page.Props["flash"])
	}
	errors, ok := page.Props["errors"].(map[string]any)
	if !ok || errors["name"] != "required" {
		t.Fatalf("unexpected errors: %#v", page.Props["errors"])
	}
	if !store.pulled {
		t.Fatal("expected flash store pull")
	}
}

func TestValidationErrorBagShape(t *testing.T) {
	store := &testFlashStore{data: FlashData{
		Bags: map[string]ValidationErrors{
			"createUser": {"name": "required"},
		},
	}}
	renderer := newTestRenderer(t, Config{FlashStore: store})
	req := httptest.NewRequest("GET", "/users", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaErrorBag, "createUser")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Users/Index", Props{})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	errors, ok := page.Props["errors"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected errors: %#v", page.Props["errors"])
	}
	bag, ok := errors["createUser"].(map[string]any)
	if !ok || bag["name"] != "required" {
		t.Fatalf("unexpected bag errors: %#v", errors)
	}
}

func TestPartialReloadData(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/users", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaPartialComponent, "Users/Index")
	req.Header.Set(HeaderInertiaPartialData, "users")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Users/Index", Props{
		"users":   []string{"a"},
		"filters": map[string]any{"q": ""},
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if _, ok := page.Props["users"]; !ok {
		t.Fatalf("users should be included: %#v", page.Props)
	}
	if _, ok := page.Props["filters"]; ok {
		t.Fatalf("filters should be excluded: %#v", page.Props)
	}
	if _, ok := page.Props["errors"]; !ok {
		t.Fatalf("errors should be included: %#v", page.Props)
	}
}

func TestPartialReloadDataIncludesExistingFlash(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/users", nil)
	req = req.WithContext(WithFlashContext(req.Context(), Flash{"success": "created"}))
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaPartialComponent, "Users/Index")
	req.Header.Set(HeaderInertiaPartialData, "users")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Users/Index", Props{
		"users":   []string{"a"},
		"filters": map[string]any{"q": ""},
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if _, ok := page.Props["flash"]; !ok {
		t.Fatalf("flash should be included in partial reload: %#v", page.Props)
	}
}

func TestPartialReloadExceptTakesPrecedenceOverData(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/users", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaPartialComponent, "Users/Index")
	req.Header.Set(HeaderInertiaPartialData, "users")
	req.Header.Set(HeaderInertiaPartialExcept, "filters")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Users/Index", Props{
		"users":   []string{"a"},
		"filters": map[string]any{"q": ""},
		"stats":   map[string]any{"count": 1},
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if _, ok := page.Props["users"]; !ok {
		t.Fatalf("users should be included: %#v", page.Props)
	}
	if _, ok := page.Props["stats"]; !ok {
		t.Fatalf("stats should be included because except takes precedence: %#v", page.Props)
	}
	if _, ok := page.Props["filters"]; ok {
		t.Fatalf("filters should be excluded: %#v", page.Props)
	}
}

func TestPartialReloadExcept(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/users", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaPartialComponent, "Users/Index")
	req.Header.Set(HeaderInertiaPartialExcept, "filters")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Users/Index", Props{
		"users":   []string{"a"},
		"filters": map[string]any{"q": ""},
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if _, ok := page.Props["users"]; !ok {
		t.Fatalf("users should be included: %#v", page.Props)
	}
	if _, ok := page.Props["filters"]; ok {
		t.Fatalf("filters should be excluded: %#v", page.Props)
	}
}

func TestPartialReloadIgnoredWhenComponentDiffers(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/users", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaPartialComponent, "Dashboard")
	req.Header.Set(HeaderInertiaPartialData, "users")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Users/Index", Props{
		"users":   []string{"a"},
		"filters": map[string]any{"q": ""},
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if _, ok := page.Props["filters"]; !ok {
		t.Fatalf("partial reload should be ignored: %#v", page.Props)
	}
}

func TestTemplateRootViewMissingTemplateReturnsError(t *testing.T) {
	view := NewTemplateRootView(template.Must(template.New("app").Parse(`ok`)), "missing")
	err := view.Render(httptest.NewRecorder(), RootViewData{})
	if err == nil {
		t.Fatal("expected missing template error")
	}
}

func newTestRenderer(t *testing.T, config Config) *Renderer {
	t.Helper()
	if config.RootView == nil {
		config.RootView = NewTemplateRootView(template.Must(template.New("app").Parse(`<!doctype html>{{ .InertiaScript }}<div id="app"></div>`)), "app")
	}
	renderer, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	return renderer
}

func decodePage(t *testing.T, w *httptest.ResponseRecorder) Page {
	t.Helper()
	var page Page
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	return page
}

type testFlashStore struct {
	data      FlashData
	pulled    bool
	putData   FlashData
	reflashed bool
}

func (s *testFlashStore) Pull(req *http.Request) (FlashData, error) {
	s.pulled = true
	return s.data, nil
}

func (s *testFlashStore) Put(w http.ResponseWriter, req *http.Request, data FlashData) error {
	s.putData = data
	return nil
}

func (s *testFlashStore) Reflash(w http.ResponseWriter, req *http.Request) error {
	s.reflashed = true
	return nil
}

type testPropResolver struct {
	result propResult
	err    error
	called bool
}

func (r *testPropResolver) resolveProp(req *http.Request, component string, key string) (propResult, error) {
	r.called = true
	return r.result, r.err
}
