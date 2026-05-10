# Echo Adapter

The Echo v5 adapter lives in a separate module:

```txt
github.com/mayahiro/go-inertia/adapters/echo
```

The core package does not import Echo.

## Requirements

- Go 1.25.0 or newer
- Echo v5

## Installation

```sh
go get github.com/mayahiro/go-inertia/adapters/echo
```

## Setup

Create a core renderer, wrap it with the Echo adapter, and register the adapter
middleware. `Renderer.Middleware` is a `net/http` middleware, but the Echo
adapter exposes `app.Middleware` directly for `e.Use`. Applications do not need
to call `echo.WrapMiddleware`.

```go
package main

import (
	"net/http"

	echo "github.com/labstack/echo/v5"
	inertia "github.com/mayahiro/go-inertia"
	inertiaecho "github.com/mayahiro/go-inertia/adapters/echo"
)

func main() {
	rootView, err := inertia.NewTemplateRootViewFromFile("views/app.html", "app.html")
	if err != nil {
		panic(err)
	}

	renderer, err := inertia.New(inertia.Config{
		RootView:   rootView,
		FlashStore: inertia.NewMemoryFlashStore(),
	})
	if err != nil {
		panic(err)
	}

	app := inertiaecho.New(renderer)
	e := echo.New()
	e.Use(app.Middleware)

	e.GET("/", func(c *echo.Context) error {
		return app.Render(c, "Dashboard", inertia.Props{
			"message": "Hello",
		})
	})

	e.POST("/users", func(c *echo.Context) error {
		return app.Redirect(c, "/users", inertia.WithFlash(inertia.Flash{
			"success": "User created",
		}))
	})

	e.RouteNotFound("/*", func(c *echo.Context) error {
		return app.RenderError(c, "Errors/NotFound", inertia.Props{}, http.StatusNotFound)
	})

	if err := e.Start(":8080"); err != nil && err != http.ErrServerClosed {
		e.Logger.Error("server error", "error", err)
	}
}
```

## Handler Helpers

The adapter exposes Echo-friendly methods:

- `Render`
- `RenderError`
- `Redirect`
- `Back`
- `Location`

These methods delegate protocol behavior to the underlying core `Renderer`.
Use `RenderError` for status-specific error pages, or pass render options such
as `inertia.WithRenderStatus` through `Render` when a page needs non-default
response behavior.

## 404 Fallback

Routing is an application concern. Use Echo's `RouteNotFound` when an undefined
path should render an Inertia 404 page.

```go
e.RouteNotFound("/*", func(c *echo.Context) error {
	return app.RenderError(c, "Errors/NotFound", inertia.Props{}, http.StatusNotFound)
})
```

The adapter does not automatically convert every Echo 404 into an Inertia
response. API routes, static files, and browser pages often need different
error behavior.

## Echo v4

The published Echo adapter targets Echo v5. If Echo v4 support is added later,
it should live in a separate adapter module.
