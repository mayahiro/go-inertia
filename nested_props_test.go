package inertia

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNestedPropsResolveDeferredOptionalAndMetadata(t *testing.T) {
	deferredCalls := 0
	optionalCalls := 0
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"auth": Props{
			"user": "hiro",
			"notifications": Defer(func(req *http.Request) (any, error) {
				deferredCalls++
				return []string{"one"}, nil
			}),
			"invoices": Optional(func(req *http.Request) (any, error) {
				optionalCalls++
				return []string{"invoice"}, nil
			}),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	auth := page.Props["auth"].(map[string]any)
	if auth["user"] != "hiro" {
		t.Fatalf("unexpected nested user: %#v", auth)
	}
	if _, ok := auth["notifications"]; ok {
		t.Fatalf("deferred nested prop should be omitted: %#v", auth)
	}
	if _, ok := auth["invoices"]; ok {
		t.Fatalf("optional nested prop should be omitted: %#v", auth)
	}
	if deferredCalls != 0 || optionalCalls != 0 {
		t.Fatalf("nested callbacks should not run on the initial response: %d %d", deferredCalls, optionalCalls)
	}
	if got := page.DeferredProps["default"]; len(got) != 1 || got[0] != "auth.notifications" {
		t.Fatalf("unexpected nested deferred metadata: %#v", page.DeferredProps)
	}
}

func TestNestedPartialReloadUsesDotNotation(t *testing.T) {
	calls := 0
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaPartialComponent, "Dashboard")
	req.Header.Set(HeaderInertiaPartialData, "auth.notifications")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"auth": Props{
			"user": "hiro",
			"notifications": Defer(func(req *http.Request) (any, error) {
				calls++
				return []string{"one"}, nil
			}),
			"invoices": Optional([]string{"invoice"}),
		},
		"stats": Props{"users": 1},
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	auth := page.Props["auth"].(map[string]any)
	if calls != 1 {
		t.Fatalf("nested deferred callback should run once: %d", calls)
	}
	if _, ok := auth["notifications"]; !ok {
		t.Fatalf("requested nested prop missing: %#v", auth)
	}
	if _, ok := auth["user"]; ok {
		t.Fatalf("unrequested nested prop should be omitted: %#v", auth)
	}
	if _, ok := auth["invoices"]; ok {
		t.Fatalf("unrequested optional prop should be omitted: %#v", auth)
	}
	if _, ok := page.Props["stats"]; ok {
		t.Fatalf("unrequested top-level prop should be omitted: %#v", page.Props)
	}
}

func TestNestedPartialReloadCanRequestParentPath(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaPartialComponent, "Dashboard")
	req.Header.Set(HeaderInertiaPartialData, "auth")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"auth": Props{
			"user":          "hiro",
			"notifications": Defer(func(req *http.Request) (any, error) { return []string{"one"}, nil }),
			"invoices":      Optional([]string{"invoice"}),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	auth := decodePage(t, w).Props["auth"].(map[string]any)
	for _, key := range []string{"user", "notifications", "invoices"} {
		if _, ok := auth[key]; !ok {
			t.Fatalf("parent path should include %s: %#v", key, auth)
		}
	}
}

func TestNestedPartialReloadAppliesOnlyAndExceptTogether(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaPartialComponent, "Dashboard")
	req.Header.Set(HeaderInertiaPartialData, "auth")
	req.Header.Set(HeaderInertiaPartialExcept, "auth.secret")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"auth": Props{
			"user":   "hiro",
			"secret": Props{"token": "private"},
		},
		"stats": Props{"users": 1},
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	auth := page.Props["auth"].(map[string]any)
	if auth["user"] != "hiro" {
		t.Fatalf("included nested prop missing: %#v", auth)
	}
	if _, ok := auth["secret"]; ok {
		t.Fatalf("excepted nested branch should be omitted: %#v", auth)
	}
	if _, ok := page.Props["stats"]; ok {
		t.Fatalf("prop outside partial data should be omitted: %#v", page.Props)
	}
}

func TestNestedPropsResolveInsideComputedContainer(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaPartialComponent, "Dashboard")
	req.Header.Set(HeaderInertiaPartialData, "auth.notifications")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"auth": Computed(func(req *http.Request) (any, error) {
			return Props{
				"user":          "hiro",
				"notifications": Defer(func(req *http.Request) (any, error) { return []string{"one"}, nil }),
			}, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	auth := decodePage(t, w).Props["auth"].(map[string]any)
	if _, ok := auth["notifications"]; !ok {
		t.Fatalf("nested deferred prop returned by callback should resolve: %#v", auth)
	}
	if auth["user"] != "hiro" {
		t.Fatalf("resolved parent callback should return its complete container: %#v", auth)
	}
}

func TestNestedMergeMetadataUsesDotPaths(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"auth": Props{
			"feed": Merge(Props{"data": []int{1}}).MatchOn("data.id"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if got := page.MergeProps; len(got) != 1 || got[0] != "auth.feed" {
		t.Fatalf("unexpected nested merge metadata: %#v", got)
	}
	if got := page.MatchPropsOn; len(got) != 1 || got[0] != "auth.feed.data.id" {
		t.Fatalf("unexpected nested match metadata: %#v", got)
	}
}

func TestNestedDeferredCompositionPublishesInitialMetadata(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"auth": Props{
			"feed": Defer(func(req *http.Request) (any, error) {
				return Props{"data": []int{1}}, nil
			}).DeepMerge().MatchOn("data.id").Once(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if got := page.DeferredProps["default"]; len(got) != 1 || got[0] != "auth.feed" {
		t.Fatalf("unexpected deferred metadata: %#v", page.DeferredProps)
	}
	if got := page.DeepMergeProps; len(got) != 1 || got[0] != "auth.feed" {
		t.Fatalf("unexpected deep merge metadata: %#v", got)
	}
	if got := page.MatchPropsOn; len(got) != 1 || got[0] != "auth.feed.data.id" {
		t.Fatalf("unexpected match metadata: %#v", got)
	}
	if got := page.OnceProps["auth.feed"]; got.Prop != "auth.feed" {
		t.Fatalf("unexpected once metadata: %#v", page.OnceProps)
	}
}

func TestOptionalOncePublishesInitialMetadata(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"companies": Optional([]string{"ACME"}).Once(),
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if _, ok := page.Props["companies"]; ok {
		t.Fatalf("optional prop should be omitted initially: %#v", page.Props)
	}
	if got := page.OnceProps["companies"]; got.Prop != "companies" {
		t.Fatalf("optional once metadata should be available initially: %#v", page.OnceProps)
	}
}

func TestNestedPropsSupportDotNotationInputKeys(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"auth.user":          "hiro",
		"auth.notifications": Defer(func(req *http.Request) (any, error) { return []string{"one"}, nil }),
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	auth := page.Props["auth"].(map[string]any)
	if auth["user"] != "hiro" {
		t.Fatalf("dot-notation input key was not unpacked: %#v", auth)
	}
	if got := page.DeferredProps["default"]; len(got) != 1 || got[0] != "auth.notifications" {
		t.Fatalf("unexpected deferred path from dot input: %#v", page.DeferredProps)
	}
}

func TestDotNotationMergesSharedAndHandlerBranches(t *testing.T) {
	renderer := newTestRenderer(t, Config{
		SharedProps: StaticSharedProps(Props{
			"auth": Props{
				"user": Props{"name": "Hiro", "role": "member"},
			},
		}),
	})
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"auth.user.role": "admin",
		"auth.team":      "platform",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	auth := page.Props["auth"].(map[string]any)
	user := auth["user"].(map[string]any)
	if user["name"] != "Hiro" || user["role"] != "admin" || auth["team"] != "platform" {
		t.Fatalf("dot branches were not merged: %#v", auth)
	}
	if got := page.SharedProps; len(got) != 1 || got[0] != "auth" {
		t.Fatalf("unexpected shared prop metadata: %#v", got)
	}
}

func TestDotNotationResolvesTraversedCallback(t *testing.T) {
	calls := 0
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"auth": func(req *http.Request) (any, error) {
			calls++
			return Props{"user": Props{"name": "Hiro"}}, nil
		},
		"auth.user.role": "admin",
	})
	if err != nil {
		t.Fatal(err)
	}

	auth := decodePage(t, w).Props["auth"].(map[string]any)
	user := auth["user"].(map[string]any)
	if calls != 1 || user["name"] != "Hiro" || user["role"] != "admin" {
		t.Fatalf("traversed callback was not merged once: calls=%d auth=%#v", calls, auth)
	}
}

func TestDotNotationBuildsSequentialArrays(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"users.0.name": "Hiro",
		"users.1.name": "Ada",
	})
	if err != nil {
		t.Fatal(err)
	}

	users := decodePage(t, w).Props["users"].([]any)
	if len(users) != 2 || users[0].(map[string]any)["name"] != "Hiro" || users[1].(map[string]any)["name"] != "Ada" {
		t.Fatalf("dot notation did not build an array: %#v", users)
	}
}

func TestNestedPropsResolveInsideSlices(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"groups": []any{
			Props{"members": Defer(func(req *http.Request) (any, error) { return []string{"hiro"}, nil })},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	if got := page.DeferredProps["default"]; len(got) != 1 || got[0] != "groups.0.members" {
		t.Fatalf("unexpected slice metadata: %#v", page.DeferredProps)
	}
}

func TestNestedPartialReloadPreservesSparseSequenceIndexes(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaPartialComponent, "Dashboard")
	req.Header.Set(HeaderInertiaPartialData, "groups.1.members")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"groups": []any{
			Props{"name": "first", "members": []string{"hiro"}},
			Props{"name": "second", "members": Merge([]string{"ada"})},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, w)
	groups := page.Props["groups"].(map[string]any)
	if _, ok := groups["0"]; ok {
		t.Fatalf("unrequested sequence index should be omitted: %#v", groups)
	}
	second := groups["1"].(map[string]any)
	if _, ok := second["name"]; ok {
		t.Fatalf("unrequested nested prop should be omitted: %#v", second)
	}
	members := second["members"].([]any)
	if len(members) != 1 || members[0] != "ada" {
		t.Fatalf("requested sequence index was not preserved: %#v", groups)
	}
	if got := page.MergeProps; len(got) != 1 || got[0] != "groups.1.members" {
		t.Fatalf("sequence metadata index was not preserved: %#v", got)
	}
}
