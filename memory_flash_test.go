package inertia

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMemoryFlashStoreStoresAndPullsOnce(t *testing.T) {
	store := NewMemoryFlashStore()
	renderer := newTestRenderer(t, Config{FlashStore: store})
	postReq := httptest.NewRequest(http.MethodPost, "/users", nil)
	postReq.Header.Set(HeaderInertia, "true")
	postW := httptest.NewRecorder()

	err := renderer.Redirect(postW, postReq, "/users",
		WithFlash(Flash{"success": "created"}),
		WithValidationErrors(ValidationErrors{"name": "required"}),
	)
	if err != nil {
		t.Fatal(err)
	}

	cookies := postW.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one flash cookie, got %d", len(cookies))
	}
	if cookies[0].Name != defaultMemoryFlashCookieName {
		t.Fatalf("unexpected cookie name: %s", cookies[0].Name)
	}
	if !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected cookie attributes: %#v", cookies[0])
	}

	firstReq := httptest.NewRequest(http.MethodGet, "/users", nil)
	firstReq.Header.Set(HeaderInertia, "true")
	firstReq.AddCookie(cookies[0])
	firstW := httptest.NewRecorder()

	if err := renderer.Render(firstW, firstReq, "Users/Index", Props{}); err != nil {
		t.Fatal(err)
	}

	page := decodePage(t, firstW)
	flash, ok := page.Props["flash"].(map[string]any)
	if !ok || flash["success"] != "created" {
		t.Fatalf("unexpected flash: %#v", page.Props["flash"])
	}
	errors, ok := page.Props["errors"].(map[string]any)
	if !ok || errors["name"] != "required" {
		t.Fatalf("unexpected errors: %#v", page.Props["errors"])
	}

	secondReq := httptest.NewRequest(http.MethodGet, "/users", nil)
	secondReq.Header.Set(HeaderInertia, "true")
	secondReq.AddCookie(cookies[0])
	secondW := httptest.NewRecorder()

	if err := renderer.Render(secondW, secondReq, "Users/Index", Props{}); err != nil {
		t.Fatal(err)
	}

	page = decodePage(t, secondW)
	if _, ok := page.Props["flash"]; ok {
		t.Fatalf("flash should only be pulled once: %#v", page.Props["flash"])
	}
	errors, ok = page.Props["errors"].(map[string]any)
	if !ok || len(errors) != 0 {
		t.Fatalf("errors should only be pulled once: %#v", page.Props["errors"])
	}
}

func TestMemoryFlashStoreZeroValue(t *testing.T) {
	store := &MemoryFlashStore{CookieName: "custom_flash", CookiePath: "/app"}
	req := httptest.NewRequest(http.MethodPost, "/users", nil)
	w := httptest.NewRecorder()

	err := store.Put(w, req, FlashData{Flash: Flash{"success": "created"}})
	if err != nil {
		t.Fatal(err)
	}

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one flash cookie, got %d", len(cookies))
	}
	if cookies[0].Name != "custom_flash" || cookies[0].Path != "/app" {
		t.Fatalf("unexpected cookie: %#v", cookies[0])
	}
}
