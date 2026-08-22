# Echo Adapter

[日本語](echo_ja.md)

The Echo v5 adapter lives in a separate module:

```txt
github.com/mayahiro/go-inertia/adapters/echo
```

The core package does not import Echo.

## Requirements

- Go 1.25.0 or newer
- Echo v5.3.1 or newer
- go-inertia v0.5.0

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
	e.HTTPErrorHandler = app.ErrorHandler(inertiaecho.ErrorHandlerConfig{
		Pages: map[int]string{
			http.StatusNotFound: "Errors/NotFound",
		},
		DefaultComponent: "Errors/Error",
		Props: func(c *echo.Context, err error, status int) (inertia.Props, error) {
			return inertia.Props{"status": status}, nil
		},
	})
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

## Error Handler

`Adapter.ErrorHandler` maps Echo errors to Inertia components and can be
installed as Echo's centralized handler.

```go
e.HTTPErrorHandler = app.ErrorHandler(inertiaecho.ErrorHandlerConfig{
	Pages: map[int]string{
		http.StatusNotFound: "Errors/NotFound",
		http.StatusForbidden: "Errors/Forbidden",
	},
	DefaultComponent: "Errors/Error",
	Props: func(c *echo.Context, err error, status int) (inertia.Props, error) {
		return inertia.Props{"status": status}, nil
	},
})
```

The default predicate renders Inertia requests and browser requests whose
`Accept` header explicitly includes `text/html` or `application/xhtml+xml`.
`HEAD`, wildcard-only, and non-HTML requests use the fallback handler, which
defaults to Echo's JSON error handler. This keeps API and static requests from
receiving an Inertia page unless they explicitly accept HTML. Responses using
the error handler vary on `Accept` and `X-Inertia`.

Override `ShouldRender` when route groups need a different policy, or set a
custom `Fallback`. `Props` can derive safe public values from the original
error. Shared props configured on the core renderer are still applied.
Custom policies can call `DefaultErrorRenderPredicate` to retain the default
content-negotiation behavior while adding application-specific conditions.
Echo's default fallback does not log errors, so use logging middleware or a
custom fallback when centralized logging is required.

Use `RenderOptions` for per-error render settings such as server-provided head
elements.

```go
RenderOptions: func(c *echo.Context, err error, status int) ([]inertia.RenderOption, error) {
	head, buildErr := inertia.NewServerHead(
		inertia.HeadTitle(http.StatusText(status)),
	)
	return []inertia.RenderOption{inertia.WithServerHead(head)}, buildErr
},
```

If prop creation, render-option creation, or rendering fails, the configured
fallback receives the original and generated errors. Responses already
committed by an Echo handler are left unchanged.

## Echo v4

The published Echo adapter targets Echo v5. If Echo v4 support is added later,
it should live in a separate adapter module.
