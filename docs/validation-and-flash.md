# Validation and Flash

Inertia validation flows usually redirect back and flash validation errors
instead of returning `422` JSON responses. On the next request, the client sees
the validation errors in the `errors` prop.

## FlashStore

`go-inertia` provides the `FlashStore` interface but does not include a
production session store.

```go
type FlashStore interface {
	Pull(req *http.Request) (FlashData, error)
	Put(w http.ResponseWriter, req *http.Request, data FlashData) error
	Reflash(w http.ResponseWriter, req *http.Request) error
}
```

Applications should implement `FlashStore` with their session library of
choice.

## Configure the Renderer

```go
renderer, err := inertia.New(inertia.Config{
	RootView:   rootView,
	FlashStore: flashStore,
})
```

## Flash Messages

Use `WithFlash` when redirecting after a successful action.

```go
return renderer.Redirect(w, req, "/users", inertia.WithFlash(inertia.Flash{
	"success": "User created",
}))
```

Flash data is sent in the `flash` prop on the next render.

## Validation Errors

Use `WithValidationErrors` when redirecting after validation fails.

```go
return renderer.Back(w, req, inertia.WithValidationErrors(inertia.ValidationErrors{
	"name": "Name is required",
}))
```

Validation errors are sent in the `errors` prop on the next render.

## Error Bags

Use `WithErrorBag` to store validation errors in a named error bag.

```go
return renderer.Back(w, req,
	inertia.WithErrorBag("createUser"),
	inertia.WithValidationErrors(inertia.ValidationErrors{
		"email": "Email is required",
	}),
)
```

## Missing FlashStore

If flash data or validation errors are passed to a redirect without a
configured `FlashStore`, the renderer returns `ErrMissingFlashStore`.
