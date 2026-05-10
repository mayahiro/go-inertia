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
- composed `Defer(...).Once()` props
- infinite scroll props with the React `InfiniteScroll` component
- form submission with Inertia `useForm`
- flash messages
- validation errors flashed through `NewMemoryFlashStore`
- a 404 fallback route rendered with the Echo adapter `RenderError` helper

The Users page intentionally sends `reset: ["users"]` after a successful create
so the infinite scroll list is rebuilt and the newly created user appears on
the first page. In a production page where a form sits beside a long loaded
list, omit that `reset` option when the existing scroll state should remain in
place, or update the current list with Inertia client-side prop helpers.

`NewMemoryFlashStore` is intended for local development and single-process
examples. Production or clustered applications should implement `FlashStore`
with a real session library, Redis, a database, or another shared backend.

The Go module uses local `replace` directives for `go-inertia` and the Echo
adapter, so the example runs against this checkout instead of a published
module tag.

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

In another terminal, start the Go server and point it at the Vite dev server:

```sh
VITE_DEV_SERVER=http://127.0.0.1:5173 go run .
```

Open `http://localhost:8080`.

If port `8080` is already in use, set `PORT`:

```sh
PORT=8081 VITE_DEV_SERVER=http://127.0.0.1:5173 go run .
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
