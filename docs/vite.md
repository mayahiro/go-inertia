# Vite

The `Vite` helper generates asset tags for Vite dev-server mode and production
manifest mode.

## Development Mode

Set `DevServerURL` to generate tags that load assets from the Vite dev server.
Enable `ReactRefresh` when using React.

```go
vite, err := inertia.NewVite(inertia.ViteConfig{
	DevServerURL: "http://127.0.0.1:5173",
	Entry:        "resources/js/app.tsx",
	ReactRefresh: true,
})
```

`Tags` returns the Vite client script, the entry script, and the React refresh
preamble when enabled.

## Production Manifest Mode

Build the frontend with Vite manifest output enabled, then point `NewVite` at
the generated manifest.

```go
vite, err := inertia.NewVite(inertia.ViteConfig{
	ManifestPath: "public/build/.vite/manifest.json",
	PublicPath:   "/build",
	Entry:        "resources/js/app.tsx",
})
```

In production mode, `Tags` reads the manifest entry and emits:

- a module script tag for the entry JavaScript file
- stylesheet tags for CSS files listed on the entry
- stylesheet tags for CSS files listed on imported chunks
- `modulepreload` tags for imported chunks

Imported chunks are collected recursively from the manifest `imports` graph.

## Asset Versioning

`vite.VersionProvider()` uses the manifest file hash as the Inertia asset
version. Pass it to `Config.VersionProvider` to enable asset version mismatch
handling.

```go
renderer, err := inertia.New(inertia.Config{
	RootView:        rootView,
	VersionProvider: vite.VersionProvider(),
})
```

## Default Render Options

Set `Tags` as a default render option when every page uses the same Vite
entrypoint. This keeps `inertia.WithViteTags(tags)` out of individual handlers.

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

When a request needs different tags, pass `inertia.WithViteTags(tags)` to
`Render`. Per-call options are applied after default options.

## Deployment

`go build` builds only the Go binary. This package does not embed templates or
Vite assets automatically.

If you do not embed assets, deploy these files with the binary:

```txt
views/app.html
public/build/.vite/manifest.json
public/build/assets/...
```

If assets are served from a CDN, set `PublicPath` to the public URL prefix.
