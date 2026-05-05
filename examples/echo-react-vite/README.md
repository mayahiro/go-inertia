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
- shared props
- form submission with redirect
- flash messages
- validation errors flashed through a small in-memory example store

The in-memory flash store is for demonstration only. Use a real session store in
production applications.

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
