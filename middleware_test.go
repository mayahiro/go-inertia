package inertia

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareAssetVersionMismatchReturns409(t *testing.T) {
	store := &testFlashStore{}
	renderer := newTestRenderer(t, Config{
		VersionProvider: StaticVersion("new"),
		FlashStore:      store,
	})
	req := httptest.NewRequest("GET", "/users?page=2", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaVersion, "old")
	w := httptest.NewRecorder()
	called := false

	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		called = true
	})).ServeHTTP(w, req)

	if called {
		t.Fatal("next handler should not be called")
	}
	if w.Code != http.StatusConflict {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	if got := w.Header().Get(HeaderInertiaLocation); got != "/users?page=2" {
		t.Fatalf("unexpected inertia location: %s", got)
	}
	if !store.reflashed {
		t.Fatal("expected reflash")
	}
}

func TestMiddlewareAssetVersionMatchCallsNext(t *testing.T) {
	renderer := newTestRenderer(t, Config{VersionProvider: StaticVersion("same")})
	req := httptest.NewRequest("GET", "/users", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaVersion, "same")
	w := httptest.NewRecorder()
	called := false

	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(w, req)

	if !called {
		t.Fatal("next handler should be called")
	}
	if w.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", w.Code)
	}
}

func TestMiddlewareAssetVersionMismatchIgnoresNonGET(t *testing.T) {
	renderer := newTestRenderer(t, Config{VersionProvider: StaticVersion("new")})
	req := httptest.NewRequest("POST", "/users", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaVersion, "old")
	w := httptest.NewRecorder()
	called := false

	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(w, req)

	if !called {
		t.Fatal("next handler should be called")
	}
}

func TestMiddlewareEmptyVersionCallsNext(t *testing.T) {
	renderer := newTestRenderer(t, Config{VersionProvider: StaticVersion("")})
	req := httptest.NewRequest("GET", "/users", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaVersion, "old")
	w := httptest.NewRecorder()
	called := false

	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(w, req)

	if !called {
		t.Fatal("next handler should be called")
	}
	if w.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", w.Code)
	}
}
