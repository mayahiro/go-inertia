# Validation and Flash

Inertia validation usually redirects back and flashes validation errors instead
of returning `422` JSON responses. On the next request, the client reads those
validation errors from the `errors` prop.

## FlashStore

`go-inertia` provides the `FlashStore` interface.

```go
type FlashStore interface {
	Pull(req *http.Request) (FlashData, error)
	Put(w http.ResponseWriter, req *http.Request, data FlashData) error
	Reflash(w http.ResponseWriter, req *http.Request) error
}
```

Production applications should implement `FlashStore` with their session
library of choice.

## MemoryFlashStore

Use `NewMemoryFlashStore` for local development, tests, and single-process
examples. It keeps flash data in process memory and stores only a session id in
an HTTP-only cookie.

```go
renderer, err := inertia.New(inertia.Config{
	RootView:   rootView,
	FlashStore: inertia.NewMemoryFlashStore(),
})
```

`NewMemoryFlashStore` is not durable storage. Use a different `FlashStore` for
multi-process deployments, server restarts, or production session policies.

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

Inertia preserves component state after `post`, `put`, `patch`, and `delete`
requests, so applications usually do not need to send old input back through
server props. Use the Inertia form helpers on the React, Vue, or Svelte side to
handle redirect errors and form state naturally.

## Error Bags

Use `WithErrorBag` when multiple forms on the same page share field names and
need separate validation error scopes.

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
