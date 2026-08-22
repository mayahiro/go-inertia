# DevTools

`go-inertia` implements the backend-independent
[Inertia DevTools protocol](https://inertiajs.com/docs/v3/advanced/devtools-protocol).
It records request, response, page, prop, route, and source metadata for the
Inertia DevTools Chrome extension without adding a runtime dependency.

DevTools is disabled by default and must be enabled explicitly.

## Setup

Enable the recorder when constructing the renderer.

```go
renderer, err := inertia.New(inertia.Config{
	RootView: rootView,
	DevTools: inertia.DevToolsConfig{
		Enabled: os.Getenv("INERTIA_DEVTOOLS_ENABLED") == "true",
		ComponentPathResolver: func(component string) string {
			return "resources/js/Pages/" + component + ".tsx"
		},
	},
})
```

The library does not read `INERTIA_DEVTOOLS_ENABLED` itself. The environment
variable above is an application-level convention that matches the official
adapter and the included Echo example.

Register `Renderer.Middleware`, or the corresponding framework adapter
middleware, so the recorder can observe requests and serve its read API.

Enable the client hooks in `createInertiaApp`.

```ts
createInertiaApp({
  // ...
  dev: import.meta.env.DEV,
})
```

The client-side adapter must support Inertia DevTools. The official Inertia
documentation requires a client adapter at version 3.6 or newer. Install the
[Inertia DevTools Chrome extension](https://inertiajs.com/docs/v3/advanced/devtools)
and open its Inertia panel while using the application.

## Configuration

| Option | Default | Behavior |
| --- | --- | --- |
| `Enabled` | `false` | Enables recording, discovery headers, the initial HTML discovery tag, and the read API |
| `Authorize` | direct loopback requests only | Controls both recording and read API access |
| `Limit` | `100` | Keeps at most this many entries per browser tab |
| `TTL` | `24h` | Removes older entries during later store operations |
| `MaxBodyBytes` | `256000` | Limits captured request bodies |
| `RedactKeys` | empty | Adds case-insensitive JSON and form keys to the built-in secret list |
| `RedactHeaders` | empty | Adds case-insensitive headers to the built-in secret list |
| `ComponentPathResolver` | `nil` | Maps a component name to a frontend source path |

Values below one for `Limit`, `TTL`, and `MaxBodyBytes` select their defaults.

The in-memory store belongs to one `Renderer` instance. Restarting the process
clears it. Multi-process deployments do not share entries, so the read request
must reach the same process that recorded the response.

## Security

DevTools entries may contain page props, headers, and request data. Enable the
recorder only in a trusted development environment.

When `Authorize` is nil, both recording and reads require the direct peer in
`Request.RemoteAddr` to be a loopback address. Forwarded headers are
intentionally not trusted. A reverse proxy can make an external request appear
to come from its own loopback connection, so do not rely on the default gate
when an enabled application is reachable through a proxy. Supply an
`Authorize` callback backed by your application's developer authentication in
that situation.

The recorder always redacts these key families before storage: passwords,
tokens, secrets, client secrets, and API keys. It also always redacts cookies,
authorization headers, proxy authorization headers, XSRF tokens, and CSRF
tokens. `RedactKeys` and `RedactHeaders` add entries to those defaults; they do
not remove the built-in protections. Matching query parameters in recorded page,
request, and redirect URLs are redacted as well.

Read API responses use `Cache-Control: no-store` and
`X-Content-Type-Options: nosniff`. Recorder and resolver failures are isolated
from the application response.

## Recorded Data and Limits

Each authorized request receives `X-Inertia-Devtools-Id` and
`X-Inertia-Devtools-Parent-Out`. Initial Inertia HTML also receives the required
`script[data-inertia-devtools-id]` discovery element. Correlation request
headers from the client are preserved in entry metadata.

Inertia page responses include the redacted page object, resolved prop values,
prop modifier metadata, and component source path when configured. The Echo
adapter also records its matched route URI and the application call site of
`Adapter.Render` or `Adapter.RenderError`.

Raw non-Inertia response bodies are deliberately not buffered and are reported
as omitted. Their status and headers are still recorded. Request bodies are
captured when downstream application code reads them completely; partially
consumed bodies are reported as omitted. JSON and form values are redacted,
multipart files are reduced to name, size, and media type, and non-textual or
over-limit bodies are omitted.

Framework adapters can add metadata through these no-op-safe hooks.

```go
renderer.RecordDevToolsRoute(req, name, uri, action)
renderer.RecordDevToolsRenderSource(req, file, line)
```

Both methods do nothing when the request was not accepted by the recorder.

## Read API

The protocol paths are fixed.

- `GET /_inertia/devtools/entries/{id}` returns one entry or `404`
- `GET /_inertia/devtools/entries` returns entries newest first

While DevTools is enabled, `Renderer.Middleware` reserves these paths and
handles them before the downstream application handler.

The list endpoint supports `component`, `type`, `exclude`, `offset`, and `limit`
query filters. Any other method returns `405`.
