package inertia

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRedirectAfterNonGETInertiaRequestUses303(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("POST", "/users", nil)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Redirect(w, req, "/users")
	if err != nil {
		t.Fatal(err)
	}

	if w.Code != http.StatusSeeOther {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	if got := w.Header().Get("Location"); got != "/users" {
		t.Fatalf("unexpected location: %s", got)
	}
}

func TestRedirectNormalGETUses302(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()

	err := renderer.Redirect(w, req, "/dashboard")
	if err != nil {
		t.Fatal(err)
	}

	if w.Code != http.StatusFound {
		t.Fatalf("unexpected status: %d", w.Code)
	}
}

func TestRedirectStoresFlashAndErrors(t *testing.T) {
	store := &testFlashStore{}
	renderer := newTestRenderer(t, Config{FlashStore: store})
	req := httptest.NewRequest("POST", "/users", nil)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Redirect(w, req, "/users", WithFlash(Flash{"success": "created"}), WithValidationErrors(ValidationErrors{"name": "required"}))
	if err != nil {
		t.Fatal(err)
	}

	if store.putData.Flash["success"] != "created" {
		t.Fatalf("flash not stored: %#v", store.putData)
	}
	if store.putData.Errors["name"] != "required" {
		t.Fatalf("errors not stored: %#v", store.putData)
	}
}

func TestRedirectStoresValidationErrorsInBag(t *testing.T) {
	store := &testFlashStore{}
	renderer := newTestRenderer(t, Config{FlashStore: store})
	req := httptest.NewRequest("POST", "/users", nil)
	w := httptest.NewRecorder()

	err := renderer.Redirect(w, req, "/users", WithValidationErrors(ValidationErrors{"name": "required"}), WithErrorBag("createUser"))
	if err != nil {
		t.Fatal(err)
	}

	if len(store.putData.Errors) != 0 {
		t.Fatalf("top-level errors should be empty for bagged errors: %#v", store.putData)
	}
	if store.putData.Bags["createUser"]["name"] != "required" {
		t.Fatalf("bagged errors not stored: %#v", store.putData)
	}
}

func TestRedirectMissingFlashStoreReturnsError(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("POST", "/users", nil)
	w := httptest.NewRecorder()

	err := renderer.Redirect(w, req, "/users", WithFlash(Flash{"success": "created"}))
	if !errors.Is(err, ErrMissingFlashStore) {
		t.Fatalf("expected ErrMissingFlashStore, got %v", err)
	}
}

func TestRedirectStatusOverride(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()

	err := renderer.Redirect(w, req, "/users", WithStatus(http.StatusTemporaryRedirect))
	if err != nil {
		t.Fatal(err)
	}

	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("unexpected status: %d", w.Code)
	}
}

func TestLocationInertiaRequestUses409(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/external", nil)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Location(w, req, "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	if w.Code != http.StatusConflict {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	if got := w.Header().Get(HeaderInertiaLocation); got != "https://example.com" {
		t.Fatalf("unexpected location: %s", got)
	}
}

func TestLocationNonInertiaRequestUses302(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/external", nil)
	w := httptest.NewRecorder()

	err := renderer.Location(w, req, "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	if w.Code != http.StatusFound {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	if got := w.Header().Get("Location"); got != "https://example.com" {
		t.Fatalf("unexpected location: %s", got)
	}
}

func TestRedirectWithPreserveFragmentUsesInertiaRedirect(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/articles/old", nil)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Redirect(w, req, "/articles/new#section", WithPreserveFragment())
	if err != nil {
		t.Fatal(err)
	}

	if w.Code != http.StatusConflict {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	if got := w.Header().Get(HeaderInertiaRedirect); got != "/articles/new#section" {
		t.Fatalf("unexpected inertia redirect: %s", got)
	}
}

func TestBackRedirectsToRootWithoutReferer(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()

	err := renderer.Back(w, req)
	if err != nil {
		t.Fatal(err)
	}

	if got := w.Header().Get("Location"); got != "/" {
		t.Fatalf("unexpected location: %s", got)
	}
}

func TestBackRedirectsToReferer(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest("GET", "/users", nil)
	req.Header.Set("Referer", "/dashboard")
	w := httptest.NewRecorder()

	err := renderer.Back(w, req)
	if err != nil {
		t.Fatal(err)
	}

	if got := w.Header().Get("Location"); got != "/dashboard" {
		t.Fatalf("unexpected location: %s", got)
	}
}
