# Testing

Use `github.com/mayahiro/go-inertia/inertiatest` for endpoint tests that
inspect Inertia responses.

```go
func TestDashboard(t *testing.T) {
	req := httptest.NewRequest("GET", "/dashboard", nil)
	req.Header.Set(inertia.HeaderInertia, "true")
	w := httptest.NewRecorder()

	err := renderer.Render(w, req, "Dashboard", inertia.Props{
		"message": "Hello",
	})
	if err != nil {
		t.Fatal(err)
	}

	inertiatest.AssertResponse(t, w).
		Status(http.StatusOK).
		IsInertia().
		Page().
		Component("Dashboard").
		PropEqual("message", "Hello")
}
```

## Page Assertions

`Page()` decodes the JSON page object. Assertions support component name, URL,
props, shared props, deferred props, rescued props, once props, merge metadata,
scroll metadata, flash data, and validation errors.

```go
inertiatest.AssertResponse(t, w).
	Page().
	HasSharedProp("auth").
	HasDeferredProp("default", "permissions").
	HasMergeProp("posts.data").
	HasScrollProp("posts").
	HasFlash("notice").
	HasError("email")
```

Nested prop paths use dot notation.

```go
inertiatest.AssertResponse(t, w).
	Page().
	PropEqual("user.name", "Ada")
```

## Redirects and Headers

Use `Status` and `Header` for redirect and middleware responses.

```go
inertiatest.AssertResponse(t, w).
	Status(http.StatusConflict).
	Header(inertia.HeaderInertiaLocation, "https://example.com")
```
