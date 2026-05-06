# Once Props

Once props are props that the Inertia client can reuse after receiving them
once. They are useful for data that is expensive to load and changes
infrequently, such as billing plans, feature flags, or role lists.

## Server Usage

Wrap the prop with `Once`.

```go
err := renderer.Render(w, req, "Dashboard", inertia.Props{
	"plans": inertia.Once(func(req *http.Request) (any, error) {
		return loadPlans(req.Context())
	}),
})
```

The callback runs until the client reports that it already has the once key.
After that, `go-inertia` omits the prop and keeps `onceProps` metadata in the
page object.

## Custom Keys

By default, the once key is the prop name. Use `As` when a prop should share a
stable key across pages or prop names.

```go
err := renderer.Render(w, req, "Users/Index", inertia.Props{
	"availableRoles": inertia.Once(loadRoles).As("roles"),
})
```

The page object includes:

```json
{
  "onceProps": {
    "roles": {
      "prop": "availableRoles"
    }
  }
}
```

## Fresh Values

Use `Fresh` to force the prop to resolve even when the client reports that it
already has the once key.

```go
"plans": inertia.Once(loadPlans).Fresh()
```

Pass `false` when the value is only conditionally fresh.

```go
"plans": inertia.Once(loadPlans).Fresh(forceRefresh)
```

## Expiration

Use `Until` to send an expiration timestamp to the client.

```go
"plans": inertia.Once(loadPlans).Until(time.Now().Add(30 * 24 * time.Hour))
```

The timestamp is serialized in `onceProps` as `expiresAt`.

## Composing Modifiers

Once props can be combined with deferred, merge, optional, and lazy props.

```go
"permissions": inertia.Defer(loadPermissions).Once()
"activity": inertia.Merge(loadActivity).Once()
"companies": inertia.Optional(loadCompanies).Once()
```

For deferred props, `onceProps` metadata is sent when the deferred prop is
loaded. For optional props, a standard visit omits the prop until the client
explicitly requests it with `only`.

## Partial Reload Behavior

A matching partial reload resolves a once prop when the prop is requested by
`X-Inertia-Partial-Data`, or when it is not excluded by
`X-Inertia-Partial-Except`. This lets the client refresh a once prop when it
explicitly asks for it.

If a matching partial reload does not request the once prop, the callback is not
executed and the once metadata is omitted from that partial response.
