package inertia

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPrecognitionSuccess(t *testing.T) {
	w := httptest.NewRecorder()

	PrecognitionSuccess(w)

	if w.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	if got := w.Header().Get(HeaderPrecognition); got != "true" {
		t.Fatalf("unexpected precognition header: %s", got)
	}
	if got := w.Header().Get(HeaderPrecognitionSuccess); got != "true" {
		t.Fatalf("unexpected precognition success header: %s", got)
	}
	if got := w.Header().Get("Vary"); got != HeaderPrecognition {
		t.Fatalf("unexpected vary: %s", got)
	}
}

func TestPrecognitionErrors(t *testing.T) {
	w := httptest.NewRecorder()

	err := PrecognitionErrors(w, ValidationErrors{"email": []string{"required"}})
	if err != nil {
		t.Fatal(err)
	}

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	if got := w.Header().Get(HeaderPrecognition); got != "true" {
		t.Fatalf("unexpected precognition header: %s", got)
	}
	if got := w.Header().Get("Vary"); got != HeaderPrecognition {
		t.Fatalf("unexpected vary: %s", got)
	}

	var body map[string]map[string][]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["errors"]["email"][0] != "required" {
		t.Fatalf("unexpected errors: %#v", body)
	}
}
