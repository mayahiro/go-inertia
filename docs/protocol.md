# Protocol

`go-inertia` implements the server-side pieces needed for the basic Inertia
protocol: HTML first visits, JSON Inertia visits, asset version mismatches,
redirects, shared props, flash data, validation errors, top-level partial
reload filtering, and deferred props. The page object can also serialize
advanced prop metadata fields used by current Inertia clients, although public
helpers for merge, once, and infinite scroll prop workflows are not available
yet.

## HTML First Visits

Normal browser visits receive an HTML document rendered by the configured
`RootView`. The document includes a safe JSON page payload and the client app
mount element.

```html
<script data-page="app" type="application/json">...</script>
<div id="app"></div>
```

The response includes `Vary: X-Inertia`.

## Inertia JSON Visits

Requests with `X-Inertia: true` receive a JSON page object. The response sets:

- `X-Inertia: true`
- `Content-Type: application/json`
- `Vary: X-Inertia`

## Page Object

The v0.1 page object supports these core fields:

- `component`
- `props`
- `url`
- `version`
- `encryptHistory`
- `clearHistory`
- `preserveFragment`
- `sharedProps`

It also has JSON fields for advanced prop metadata:

- `mergeProps`
- `prependProps`
- `deepMergeProps`
- `matchPropsOn`
- `scrollProps`
- `deferredProps`
- `onceProps`

These metadata fields are present so deferred props and future once, merge, and
infinite scroll helpers can use the protocol shape expected by Inertia clients.

`props.errors` is always present. When there are no validation errors, it is an
empty object.

## Asset Version Mismatches

For GET Inertia requests, middleware compares `X-Inertia-Version` with the
current asset version. If they differ, it returns `409 Conflict` and sets
`X-Inertia-Location` to the current URL.

Non-GET requests do not return an asset mismatch response directly.

## Redirects

Non-GET Inertia redirects use `303 See Other`. External locations use
`409 Conflict` with `X-Inertia-Location`.

`WithPreserveFragment` returns `409 Conflict` with `X-Inertia-Redirect` for
Inertia requests.

## Partial Reloads

v0.1 supports top-level prop filtering only.

- Filtering applies only when `X-Inertia-Partial-Component` matches the rendered component.
- `X-Inertia-Partial-Except` excludes listed top-level props.
- `X-Inertia-Partial-Data` includes only listed top-level props when `Partial-Except` is not set.
- `errors` is always included.
- `flash` is included when flash data exists.

## Deferred Props

`Defer` omits the prop from the initial page object and adds the prop name to
`deferredProps`.

```json
{
  "component": "Users/Index",
  "props": {
    "errors": {},
    "users": []
  },
  "url": "/users",
  "deferredProps": {
    "default": ["permissions"]
  }
}
```

When the client requests the deferred prop with a matching partial reload, the
callback is evaluated and the resolved prop is included in `props`.
