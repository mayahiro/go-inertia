# go-inertia

A small server-side Inertia.js adapter for Go.

The core package is built on `net/http` and has no runtime dependencies outside
the Go standard library. Echo v5 support lives in a separate adapter module.

## Status

This project is in the v0 release line. Check the Git tags for available
versions.

## Package Layout

- `github.com/mayahiro/go-inertia` is the framework-independent core package.
- `github.com/mayahiro/go-inertia/adapters/echo` adapts the core renderer to Echo v5.
- `examples/echo-react-vite` is a separate example application module.

## Requirements

- Core package: Go 1.25.0 or newer
- Echo adapter: Go 1.25.0 or newer, because Echo v5 requires it
- Example frontend: Node.js 24 or newer

## Installation

```sh
go get github.com/mayahiro/go-inertia
```

For Echo v5:

```sh
go get github.com/mayahiro/go-inertia/adapters/echo
```

## What This Package Provides

- HTML first-visit responses
- Inertia JSON responses
- `Vary: X-Inertia`
- asset version mismatch handling
- Inertia redirects, back redirects, and external locations
- shared props
- flash data and validation error interfaces
- single-process in-memory flash store
- top-level partial reload filtering
- Vite manifest and dev-server tag generation
- default render options
- Echo v5 adapter

## Integration Notes

- Register `Renderer.Middleware` or the framework adapter middleware before routes that render Inertia pages.
- Values in `Props`, shared props, flash data, and validation errors are sent to the browser. Do not put secrets in them.
- For larger pages, define page-specific Go structs and convert them to `inertia.Props` at the render boundary. This keeps the server/frontend contract easier to review.
- `NewMemoryFlashStore` is intended for local development, tests, and single-process examples. Production and clustered applications should implement `FlashStore` with their session store, Redis, a database, or another shared backend.
- `go build` builds Go code only. Templates and Vite assets are deployed as files unless your application embeds them.

## Core Example

```go
package main

import (
	"net/http"

	inertia "github.com/mayahiro/go-inertia"
)

func main() {
	rootView, err := inertia.NewTemplateRootViewFromFile("views/app.html", "app.html")
	if err != nil {
		panic(err)
	}

	renderer, err := inertia.New(inertia.Config{
		RootView: rootView,
		SharedProps: inertia.StaticSharedProps(inertia.Props{
			"app": map[string]any{"name": "Admin"},
		}),
	})
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		err := renderer.Render(w, req, "Dashboard", inertia.Props{
			"message": "Hello",
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	http.ListenAndServe(":8080", renderer.Middleware(mux))
}
```

## Root Template

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    {{ .ViteTags }}
    {{ .InertiaHead }}
  </head>
  <body>
    {{ .InertiaScript }}
    <div id="app"></div>
  </body>
</html>
```

## Echo Example

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

	if err := e.Start(":8080"); err != nil && err != http.ErrServerClosed {
		e.Logger.Error("server error", "error", err)
	}
}
```

## Vite

Use `NewVite` to generate tags for a Vite entrypoint.

Development mode:

```go
vite, err := inertia.NewVite(inertia.ViteConfig{
	DevServerURL: "http://127.0.0.1:5173",
	Entry:        "resources/js/app.tsx",
	ReactRefresh: true,
})
```

Production mode:

```go
vite, err := inertia.NewVite(inertia.ViteConfig{
	ManifestPath: "public/build/.vite/manifest.json",
	PublicPath:   "/build",
	Entry:        "resources/js/app.tsx",
})
```

Set generated tags as default render options when every page uses the same
root template assets.

```go
tags, err := vite.Tags()
if err != nil {
	return err
}

renderer, err := inertia.New(inertia.Config{
	RootView:        rootView,
	VersionProvider: vite.VersionProvider(),
	DefaultRenderOptions: []inertia.RenderOption{
		inertia.WithViteTags(tags),
	},
})
```

You can still pass `inertia.WithViteTags(tags)` to an individual `Render` call
when a request needs to override the default tags.

## Flash and Validation

Inertia validation usually redirects back and flashes validation errors instead
of returning `422` JSON responses. `go-inertia` provides the `FlashStore`
interface and a small in-memory implementation for development use.

```go
renderer, err := inertia.New(inertia.Config{
	RootView:   rootView,
	FlashStore: inertia.NewMemoryFlashStore(),
})
```

Use `Back` and `WithValidationErrors` after validation fails.

```go
return renderer.Back(w, req, inertia.WithValidationErrors(inertia.ValidationErrors{
	"name": "Name is required",
}))
```

Inertia preserves component state after non-GET requests, so applications
usually do not need to send old input back through server props.

## React + Vite Example

See [examples/echo-react-vite](examples/echo-react-vite) for a TypeScript
React + Vite + Echo example.

## Documentation

- [Getting started](docs/getting-started.md)
- [Protocol](docs/protocol.md)
- [Echo adapter](docs/echo.md)
- [Vite](docs/vite.md)
- [Validation and flash](docs/validation-and-flash.md)

## Not Yet Covered by Public Helpers

The core page object can serialize Inertia's advanced prop metadata fields, but
user-facing helpers are not available yet for these workflows.

- server-side rendering
- deferred props
- once props
- merge, prepend, and deep merge props
- infinite scroll protocol features
- history encryption
- Precognition validation
- production-ready session store
- Echo v4 adapter
- adapters for frameworks other than Echo v5
- CLI scaffolding

## Development

Go imports and formatting are handled by `goimports`. The tool dependency is
kept in a separate `tools` module so the public root module stays dependency
free.

```sh
cd tools
go tool goimports -w ..
```

Run the core checks:

```sh
go test ./...
go vet ./...
```

Run the Echo adapter checks:

```sh
cd adapters/echo
go test ./...
go vet ./...
```

Run the example checks:

```sh
cd examples/echo-react-vite
npm ci
npm run build
go test ./...
go vet ./...
```

## References

- Inertia protocol: https://inertiajs.com/the-protocol
- Inertia redirects: https://inertiajs.com/redirects
- Inertia validation: https://inertiajs.com/validation
- Inertia partial reloads: https://inertiajs.com/partial-reloads
- Inertia asset versioning: https://inertiajs.com/asset-versioning
- Vite backend integration: https://vite.dev/guide/backend-integration.html
- Echo: https://echo.labstack.com/

## License

MIT
