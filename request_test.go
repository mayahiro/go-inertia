package inertia

import (
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestRequestHelpers(t *testing.T) {
	req := httptest.NewRequest("GET", "/users", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaPartialComponent, "Users/Index")
	req.Header.Set(HeaderInertiaPartialData, "users, filters")
	req.Header.Set(HeaderInertiaPartialExcept, "stats")
	req.Header.Set(HeaderInertiaReset, "results")
	req.Header.Set(HeaderInertiaErrorBag, "createUser")
	req.Header.Set(HeaderInertiaInfiniteScrollMergeIntent, "prepend")
	req.Header.Set(HeaderInertiaExceptOnceProps, "plans, roles")
	req.Header.Set(HeaderPurpose, "prefetch")
	req.Header.Set(HeaderPrecognition, "true")
	req.Header.Set(HeaderPrecognitionValidateOnly, "name, email")

	if !IsInertiaRequest(req) {
		t.Fatal("expected inertia request")
	}
	if !IsPartialReload(req) {
		t.Fatal("expected partial reload")
	}
	if PartialComponent(req) != "Users/Index" {
		t.Fatalf("unexpected partial component: %s", PartialComponent(req))
	}
	if !reflect.DeepEqual(PartialData(req), []string{"users", "filters"}) {
		t.Fatalf("unexpected partial data: %#v", PartialData(req))
	}
	if !reflect.DeepEqual(PartialExcept(req), []string{"stats"}) {
		t.Fatalf("unexpected partial except: %#v", PartialExcept(req))
	}
	if !reflect.DeepEqual(ResetProps(req), []string{"results"}) {
		t.Fatalf("unexpected reset props: %#v", ResetProps(req))
	}
	if ErrorBag(req) != "createUser" {
		t.Fatalf("unexpected error bag: %s", ErrorBag(req))
	}
	if InfiniteScrollMergeIntent(req) != "prepend" {
		t.Fatalf("unexpected infinite scroll merge intent: %s", InfiniteScrollMergeIntent(req))
	}
	if !reflect.DeepEqual(ExceptOnceProps(req), []string{"plans", "roles"}) {
		t.Fatalf("unexpected except once props: %#v", ExceptOnceProps(req))
	}
	if !IsPrefetch(req) {
		t.Fatal("expected prefetch request")
	}
	if !IsPrecognition(req) {
		t.Fatal("expected precognition request")
	}
	if !reflect.DeepEqual(PrecognitionValidateOnly(req), []string{"name", "email"}) {
		t.Fatalf("unexpected precognition fields: %#v", PrecognitionValidateOnly(req))
	}
}

func TestRequestHelpersDefaultValues(t *testing.T) {
	req := httptest.NewRequest("GET", "/users", nil)

	if IsInertiaRequest(req) {
		t.Fatal("expected normal request")
	}
	if IsPartialReload(req) {
		t.Fatal("expected no partial reload")
	}
	if PartialComponent(req) != "" {
		t.Fatalf("unexpected component: %s", PartialComponent(req))
	}
	if PartialData(req) != nil {
		t.Fatalf("unexpected partial data: %#v", PartialData(req))
	}
	if PartialExcept(req) != nil {
		t.Fatalf("unexpected partial except: %#v", PartialExcept(req))
	}
	if ResetProps(req) != nil {
		t.Fatalf("unexpected reset props: %#v", ResetProps(req))
	}
	if ErrorBag(req) != "" {
		t.Fatalf("unexpected error bag: %s", ErrorBag(req))
	}
	if InfiniteScrollMergeIntent(req) != "" {
		t.Fatalf("unexpected infinite scroll merge intent: %s", InfiniteScrollMergeIntent(req))
	}
	if ExceptOnceProps(req) != nil {
		t.Fatalf("unexpected except once props: %#v", ExceptOnceProps(req))
	}
	if IsPrefetch(req) {
		t.Fatal("unexpected prefetch request")
	}
	if IsPrecognition(req) {
		t.Fatal("unexpected precognition request")
	}
	if PrecognitionValidateOnly(req) != nil {
		t.Fatalf("unexpected precognition fields: %#v", PrecognitionValidateOnly(req))
	}
}

func TestAppendVary(t *testing.T) {
	header := httptest.NewRecorder().Header()
	header.Set("Vary", "Accept-Encoding, x-inertia")

	AppendVary(header, HeaderInertia)

	values := header.Values("Vary")
	if len(values) != 1 {
		t.Fatalf("expected no duplicate vary, got %#v", values)
	}

	AppendVary(header, "Accept-Language")
	if got := header.Values("Vary"); len(got) != 2 {
		t.Fatalf("expected appended vary, got %#v", got)
	}
}

func TestAppendVaryPreservesWildcard(t *testing.T) {
	header := httptest.NewRecorder().Header()
	header.Set("Vary", "*")

	AppendVary(header, HeaderInertia)

	if got := header.Values("Vary"); len(got) != 1 || got[0] != "*" {
		t.Fatalf("unexpected vary: %#v", got)
	}
}
