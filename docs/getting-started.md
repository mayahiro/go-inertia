# Getting Started

`go-inertia` is a small server-side Inertia.js adapter for Go. The core package
is built on `net/http` and has no runtime dependencies outside the Go standard
library.

## Requirements

- Go 1.25.0 or newer
- A root HTML template
- An Inertia client application

## Installation

```sh
go get github.com/mayahiro/go-inertia
```

For Echo v5:

```sh
go get github.com/mayahiro/go-inertia/adapters/echo
```

## Package Layout

- `github.com/mayahiro/go-inertia` is the framework-independent core package.
- `github.com/mayahiro/go-inertia/adapters/echo` adapts the core renderer to Echo v5.
- The core package does not import Echo.

## Root Template

The root template is rendered for normal browser visits. It should include
`InertiaScript` and the element used by the client app.

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

## Renderer

Create a `Renderer` with a root view, register its middleware, and call
`Render` from handlers.

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

	renderer, err := inertia.New(inertia.Config{RootView: rootView})
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

## Notes

Values in `Props`, shared props, flash data, and validation errors are sent to
the browser. Do not put secrets or server-only values in them.
