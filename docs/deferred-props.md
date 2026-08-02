# Deferred Props

Deferred props let the first page response render without waiting for expensive
data. The server sends metadata that tells the Inertia client which props to
request after the page has mounted.

## Server Usage

Wrap an expensive prop with `Defer`.

```go
err := renderer.Render(w, req, "Users/Index", inertia.Props{
	"users": users,
	"permissions": inertia.Defer(func(req *http.Request) (any, error) {
		return loadPermissions(req.Context())
	}),
})
```

The callback is not executed during the initial response. It runs when the
client requests `permissions` with a partial reload.

## Groups

Deferred props are grouped under `default` unless you pass a group name.

```go
err := renderer.Render(w, req, "Users/Index", inertia.Props{
	"permissions": inertia.Defer(loadPermissions),
	"teams":       inertia.Defer(loadTeams, "attributes"),
	"projects":    inertia.Defer(loadProjects, "attributes"),
})
```

The resulting page object includes:

```json
{
  "deferredProps": {
    "default": ["permissions"],
    "attributes": ["teams", "projects"]
  }
}
```

The client loads each group with a separate partial reload.

## Composing Modifiers

Deferred props can also be marked as mergeable or once props. Merge and once
metadata is sent with the initial response even though the prop value remains
deferred. This lets the client prepare both behaviors before loading the value.

```go
err := renderer.Render(w, req, "Users/Index", inertia.Props{
	"results": inertia.Defer(loadResults).DeepMerge().MatchOn("data.id"),
	"permissions": inertia.Defer(loadPermissions).Once(),
})
```

The first page response includes `deferredProps` plus the configured
`deepMergeProps`, `matchPropsOn`, and `onceProps` metadata. The prop values are
added when the client requests them.

## Partial Reload Behavior

For a matching partial reload, `go-inertia` resolves a deferred prop only when
its path satisfies `X-Inertia-Partial-Data`, when present, and is not excluded
by `X-Inertia-Partial-Except`.

Nested deferred props use full dot paths in both request and response metadata.

```go
"auth": inertia.Props{
	"notifications": inertia.Defer(loadNotifications),
}
```

The resulting deferred path is `auth.notifications`.

If a partial reload does not request a deferred prop, the callback is not
executed and the prop is omitted from the response.

## Client Usage

Use the Inertia client adapter's `Deferred` component to render fallback UI
until the prop is available.

```tsx
import { Deferred } from "@inertiajs/react"

export default function UsersIndex({ users, permissions }) {
  return (
    <Deferred data="permissions" fallback={<div>Loading...</div>}>
      <PermissionList permissions={permissions} />
    </Deferred>
  )
}
```

For multiple props, pass an array to `data`.

```tsx
<Deferred data={["teams", "projects"]} fallback={<div>Loading...</div>}>
  <ProjectAccess />
</Deferred>
```

## Rescue Mode

Use `Rescue` when the client `<Deferred>` component should render its rescue
slot after a deferred loader error.

```go
"permissions": inertia.Defer(loadPermissions).Rescue()
```

When a rescued deferred prop fails during a matching partial reload, the prop is
omitted from `props` and its key is included in `rescuedProps`.

Without `Rescue`, callback errors are returned by `Render` and no response body
is written.
