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
metadata is sent when the deferred prop is actually loaded.

```go
err := renderer.Render(w, req, "Users/Index", inertia.Props{
	"results": inertia.Defer(loadResults).DeepMerge().MatchOn("data.id"),
	"permissions": inertia.Defer(loadPermissions).Once(),
})
```

The first page response only includes `deferredProps`. When the client loads
`results`, the response includes `deepMergeProps` and `matchPropsOn`. When the
client loads `permissions`, the response includes `onceProps`.

## Partial Reload Behavior

For a matching partial reload, `go-inertia` resolves a deferred prop only when
the prop is requested by `X-Inertia-Partial-Data`, or when it is not excluded by
`X-Inertia-Partial-Except`.

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
