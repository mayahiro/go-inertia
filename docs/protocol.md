# Protocol

`go-inertia` implements the server-side pieces needed for the basic Inertia
protocol: HTML first visits, JSON Inertia visits, asset version mismatches,
redirects, shared props, flash data, validation errors, top-level partial
reload filtering, deferred props, once props, and merge props. The page object
can also serialize advanced prop metadata fields used by current Inertia
clients, although public helpers for infinite scroll prop workflows are not
available yet.

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

These metadata fields are present so deferred props, once props, merge props,
and future infinite scroll helpers can use the protocol shape expected by
Inertia clients.

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
- `X-Inertia-Reset` removes merge metadata for listed top-level props.
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

## Once Props

`Once` resolves the prop normally until the client reports that it already has
the once key. The client reports loaded once keys with
`X-Inertia-Except-Once-Props`.

```json
{
  "component": "Dashboard",
  "props": {
    "errors": {},
    "plans": []
  },
  "url": "/dashboard",
  "onceProps": {
    "plans": {
      "prop": "plans"
    }
  }
}
```

When a later Inertia request includes `X-Inertia-Except-Once-Props: plans`, the
prop is omitted and the `onceProps` metadata remains in the page object. A
matching partial reload still resolves the prop when it requests the prop.

## Merge Props

`Merge` includes the prop value and adds merge metadata to the page object. A
plain merge prop appends at the root prop path.

```json
{
  "component": "Items/Index",
  "props": {
    "errors": {},
    "items": []
  },
  "url": "/items",
  "mergeProps": ["items"]
}
```

Nested append/prepend paths are serialized as full page prop paths.

```json
{
  "mergeProps": ["results.data"],
  "prependProps": ["results.pinned"],
  "matchPropsOn": ["results.data.id"]
}
```

When the client sends `X-Inertia-Reset`, matching merge metadata is omitted so
the client replaces the prop value instead of merging it.
