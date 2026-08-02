package inertia

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInvalidPropModifierCombinationsReturnPathError(t *testing.T) {
	load := func(req *http.Request) (any, error) { return []string{"value"}, nil }
	tests := map[string]Prop{
		"rescue without defer":   Merge([]string{"value"}).Rescue(),
		"match without merge":    Optional([]string{"value"}).MatchOn("id"),
		"wrapper without scroll": Computed(load).Wrapper("items"),
		"multiple defer groups":  Defer(load, "first", "second"),
		"multiple rescue values": Defer(load).Rescue(true, false),
		"multiple fresh values":  Once(load).Fresh(true, false),
	}

	for name, prop := range tests {
		t.Run(name, func(t *testing.T) {
			renderer := newTestRenderer(t, Config{})
			req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
			req.Header.Set(HeaderInertia, "true")
			w := httptest.NewRecorder()

			err := renderer.Render(w, req, "Dashboard", Props{
				"auth": Props{"value": prop},
			})
			if !errors.Is(err, ErrInvalidPropConfiguration) {
				t.Fatalf("expected ErrInvalidPropConfiguration, got %v", err)
			}
			var configurationError *PropConfigurationError
			if !errors.As(err, &configurationError) || configurationError.Path != "auth.value" {
				t.Fatalf("unexpected configuration error: %#v", err)
			}
			if w.Body.Len() != 0 {
				t.Fatalf("invalid prop should not write a response: %s", w.Body.String())
			}
		})
	}
}

func TestInvalidUnrequestedPropIsStillValidated(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaPartialComponent, "Dashboard")
	req.Header.Set(HeaderInertiaPartialData, "stats")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", Props{
		"stats": Props{"users": 1},
		"auth":  Props{"value": Merge([]string{"value"}).Rescue()},
	})
	if !errors.Is(err, ErrInvalidPropConfiguration) {
		t.Fatalf("unrequested invalid prop should be rejected, got %v", err)
	}
}

func TestNilPropPointerReturnsConfigurationError(t *testing.T) {
	renderer := newTestRenderer(t, Config{})
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	w := httptest.NewRecorder()

	var prop *Prop
	err := renderer.Render(w, req, "Dashboard", Props{"auth": Props{"value": prop}})
	if !errors.Is(err, ErrInvalidPropConfiguration) {
		t.Fatalf("expected ErrInvalidPropConfiguration, got %v", err)
	}
	var configurationError *PropConfigurationError
	if !errors.As(err, &configurationError) || configurationError.Path != "auth.value" {
		t.Fatalf("unexpected configuration error: %#v", err)
	}
}
