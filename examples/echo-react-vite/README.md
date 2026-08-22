# Echo + React + Vite Example

This example shows how to use `github.com/mayahiro/go-inertia` with Echo v5,
React, TypeScript, and Vite.

It demonstrates:

- Echo server setup
- Inertia renderer setup
- Echo adapter middleware
- React page components written in TSX
- Vite dev-server mode
- Vite production manifest mode
- Vite tags configured through default render options
- shared props
- `Always` shared props
- typed Go page props converted to `inertia.Props`
- composed `Defer(...).Rescue().Once()` props with React fallback and rescue UI
- dynamic partial reload polling with React `usePoll`
- infinite scroll props with the React `InfiniteScroll` component
- form submission with Inertia `useForm`
- flash messages
- validation errors flashed through `NewMemoryFlashStore`
- top-level Inertia v3 flash typing
- structured server-provided title and meta elements
- a custom root element id shared by the Go renderer and React client
- React automatic bootstrap with StrictMode
- centralized 404 and 500 pages through the Echo adapter error handler
- Inertia DevTools discovery and request inspection in development

The Users page intentionally sends `reset: ["users"]` after a successful create
so the infinite scroll list is rebuilt and the newly created user appears on
the first page. In a production page where a form sits beside a long loaded
list, omit that `reset` option when the existing scroll state should remain in
place, or update the current list with Inertia client-side prop helpers.

`NewMemoryFlashStore` is intended for local development and single-process
examples. Production or clustered applications can adapt an existing durable
session through `NewSessionFlashStore`, or implement `FlashStore` directly for
a shared backend.

The Dashboard polls only its `stats` prop every 10 seconds with `mode: "rest"`.
Its deferred server time combines `Rescue` and `Once`, so the client can show a
dedicated rescue state if loading fails and avoid loading the value again after
it succeeds. Open `/?failServerTime=1` to exercise the rescue state.

Visit an undefined path to see the mapped 404 page, or `/demo/error` to exercise
the default 500 page. Both receive structured server-provided head elements.

The frontend pins `@inertiajs/core` and `@inertiajs/react` to 3.7.0. The example
uses client-side rendering and does not configure Inertia SSR.

The Go module declares `v0.5.0` for `go-inertia` and the Echo adapter. Local
`replace` directives make the example run against this checkout instead of a
published module tag.

## Requirements

- Go 1.25 or newer
- Node.js 24 or newer
- npm

## Development

Install frontend dependencies:

```sh
npm ci
```

Start the Vite dev server:

```sh
npm run dev
```

In another terminal, start the Go server, enable the backend DevTools recorder,
and point it at the Vite dev server:

```sh
INERTIA_DEVTOOLS_ENABLED=true VITE_DEV_SERVER=http://127.0.0.1:5173 go run .
```

Open `http://localhost:8080`.

The React client passes `dev: import.meta.env.DEV` to `createInertiaApp`, while
`INERTIA_DEVTOOLS_ENABLED=true` enables the matching server protocol. Install
the Inertia DevTools browser extension to inspect visits, props, route metadata,
and render source locations. The recorder keeps entries in process memory and,
by default, only records and serves entries for direct loopback requests. Keep
it disabled outside a trusted local development environment.

If port `8080` is already in use, set `PORT`:

```sh
PORT=8081 INERTIA_DEVTOOLS_ENABLED=true VITE_DEV_SERVER=http://127.0.0.1:5173 go run .
```

## Production Build

Build the frontend assets:

```sh
npm ci
npm run build
```

Start the Go server:

```sh
go run .
```

In production mode, the server reads `public/build/.vite/manifest.json` and
serves built assets from `public/build`.

## Type Checking

Run TypeScript type checking without building assets:

```sh
npm run typecheck
```

## Building the Go Binary

`go build` builds only the Go binary. This example does not embed the root
template or Vite build output.

If you deploy the binary, deploy these files alongside it:

```txt
views/app.html
public/build/.vite/manifest.json
public/build/assets/...
```

Run `npm run build` before `go build` if you want the binary and assets to be
produced from the same source revision.

## Useful Paths

- `main.go`: Echo server and Inertia renderer setup
- `views/app.html`: root HTML template
- `resources/js/app.tsx`: Inertia React client entry
- `resources/js/Pages`: React page components
- `vite.config.ts`: Vite configuration
- `public/build`: generated production assets
